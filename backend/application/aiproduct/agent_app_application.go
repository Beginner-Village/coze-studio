/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package aiproduct

import (
	"context"
	"fmt"
	"time"

	productentity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	productservice "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/service"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// AgentDraftView is the minimal view of a super-agent's draft that the publish
// flow needs to build an agent snapshot. Credential values MUST be carried as
// *_ref strings so they are never resolved to plaintext during publish.
type AgentDraftView struct {
	ModelID      string
	ModelParams  map[string]any
	Prompt       string
	Capabilities map[string]any
	MCPServers   []map[string]any
	Skills       []SnapshotSkill
}

// AgentReader is a local port for reading a super-agent's draft. The real
// implementation (wired in Task 5) will delegate to
// backend/application/singleagent.
type AgentReader interface {
	GetDraft(ctx context.Context, agentID int64) (*AgentDraftView, error)
}

// ShadowAgentWriter is a local port for creating a read-only shadow agent that
// is seeded from a published agent_app product. The real implementation (wired
// in Task 5) will delegate to the singleagent domain service.
type ShadowAgentWriter interface {
	// CreateShadowDraft creates a shadow draft agent linked to productID/version
	// and pre-populated from snapshot. It returns the new shadow agentID.
	CreateShadowDraft(ctx context.Context, spaceID, userID, productID int64, version string, snapshot map[string]any) (int64, error)

	// FindUserInstance returns the agentID of the instance the user already
	// materialised from the given product, or 0 if none exists. It makes
	// recruitment idempotent so re-recruiting reuses the existing instance
	// instead of leaking a fresh sandbox-backed draft.
	FindUserInstance(ctx context.Context, userID, productID int64) (int64, error)
}

// PublishReq carries the parameters needed to publish a super-agent as an
// agent_app AI product.
type PublishReq struct {
	AgentID int64
	SpaceID int64
	UserID  int64
	Name    string
	Version string
}

// AgentAppSVC is the package-level singleton wired in application.Init.
var AgentAppSVC *AgentAppApplication

// AgentAppApplication orchestrates the publish and recruit flows for the
// virtual-employee agent_app product type.
type AgentAppApplication struct {
	DomainSVC    productservice.Service
	AgentReader  AgentReader
	ShadowWriter ShadowAgentWriter
	Sandbox      TemplateSandbox
	ObjectPrefix string // e.g. "templates/agent_app"
}

// PublishAgentApp freezes the super-agent's draft as an agent_snapshot, upserts
// the ai_product record (type=agent_app, status=building), and then runs
// BuildTemplate to produce the sandbox template checkpoint. The product version
// feature_snapshot is updated with the resulting build_status.
//
// Credential refs (fields ending in _ref) are copied verbatim from the draft
// snapshot; they are NEVER resolved to plaintext here.
func (a *AgentAppApplication) PublishAgentApp(ctx context.Context, req PublishReq) (productID int64, version string, err error) {
	// 1. Read the super-agent draft.
	draft, err := a.AgentReader.GetDraft(ctx, req.AgentID)
	if err != nil {
		return 0, "", fmt.Errorf("agent_app publish: read draft agentID=%d: %w", req.AgentID, err)
	}

	// 2. Build the immutable agent snapshot (credential *_ref strings are copied
	//    verbatim from draft; never resolve to plaintext).
	agentSnapshot := BuildAgentSnapshot(AgentSnapshotInput{
		ModelID:      draft.ModelID,
		ModelParams:  draft.ModelParams,
		Prompt:       draft.Prompt,
		Capabilities: draft.Capabilities,
		MCPServers:   draft.MCPServers,
		Skills:       draft.Skills,
	})

	// 3. Upsert the ai_product row (type=agent_app, status=building).
	featureSnapshot := map[string]any{
		"agent_snapshot": agentSnapshot,
		"template": map[string]any{
			"build_status": "building",
		},
	}
	product := &productentity.Product{
		SpaceID:    req.SpaceID,
		CreatorID:  req.UserID,
		Name:       req.Name,
		Type:       productentity.AIProductTypeAgentApp,
		Status:     productentity.AIProductStatusDraft,
		Visibility: productentity.AIProductVisibilityGlobal,
		Feature:    featureSnapshot,
	}
	pv := &productentity.ProductVersion{
		Version:         req.Version,
		Status:          productentity.AIProductStatusDraft,
		FeatureSnapshot: featureSnapshot,
	}

	synced, err := a.DomainSVC.SyncProduct(ctx, product, pv)
	if err != nil {
		return 0, "", fmt.Errorf("agent_app publish: sync product: %w", err)
	}
	productID = synced.ProductID

	// 4. Build the sandbox template asynchronously. The publish request returns
	//    immediately with build_status=building; a detached goroutine runs the
	//    (potentially slow) skill-dependency install + checkpoint, then persists
	//    the terminal build_status back to the product/version feature snapshot
	//    and flips the product to published on success.
	objectKey := fmt.Sprintf("%s/%d/%s.tgz", a.ObjectPrefix, productID, req.Version)
	buildKey := fmt.Sprintf("agent_app_build_%d_%s", productID, req.Version)

	buildSkills := make([]BuildSkill, 0, len(draft.Skills))
	for _, s := range draft.Skills {
		buildSkills = append(buildSkills, BuildSkill{
			Name: s.Name,
		})
	}

	go a.runTemplateBuild(buildJob{
		ProductID: productID,
		SpaceID:   req.SpaceID,
		UserID:    req.UserID,
		Name:      req.Name,
		Version:   req.Version,
		Snapshot:  agentSnapshot,
		ObjectKey: objectKey,
		BuildKey:  buildKey,
		Skills:    buildSkills,
	})

	return productID, req.Version, nil
}

// buildJob carries the inputs needed to run a template build and persist its
// result outside the publish request's lifecycle.
type buildJob struct {
	ProductID int64
	SpaceID   int64
	UserID    int64
	Name      string
	Version   string
	Snapshot  any
	ObjectKey string
	BuildKey  string
	Skills    []BuildSkill
}

// runTemplateBuild runs BuildTemplate on a detached context (the publish HTTP
// request has already returned) and persists the terminal build status back to
// the product/version feature snapshot. On a successful build the product is
// flipped to published so it becomes visible in the marketplace; on failure it
// stays draft with build_status=failed and a diagnostic detail.
func (a *AgentAppApplication) runTemplateBuild(job buildJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	buildStatus := "ready"
	detail := ""
	contentHash := ""
	result, err := BuildTemplate(ctx, a.Sandbox, BuildTemplateRequest{
		BuildKey:  job.BuildKey,
		ObjectKey: job.ObjectKey,
		Skills:    job.Skills,
	})
	switch {
	case err != nil:
		buildStatus = "failed"
		detail = err.Error()
		logs.CtxErrorf(ctx, "agent_app build productID=%d: %v", job.ProductID, err)
	default:
		buildStatus = result.Status
		detail = result.Detail
		contentHash = result.ContentHash
	}

	productStatus := productentity.AIProductStatusDraft
	publishedVersion := ""
	if buildStatus == "ready" {
		productStatus = productentity.AIProductStatusPublished
		// Recruit → Install requires a non-empty PublishedVersion (see
		// isInstallable); without it the published agent_app is not installable.
		publishedVersion = job.Version
	}

	featureSnapshot := map[string]any{
		"agent_snapshot": job.Snapshot,
		"template": map[string]any{
			"build_status": buildStatus,
			"detail":       detail,
			"object_key":   job.ObjectKey,
			"content_hash": contentHash,
		},
	}
	product := &productentity.Product{
		ProductID:        job.ProductID,
		SpaceID:          job.SpaceID,
		CreatorID:        job.UserID,
		Name:             job.Name,
		Type:             productentity.AIProductTypeAgentApp,
		Status:           productStatus,
		Visibility:       productentity.AIProductVisibilityGlobal,
		LatestVersion:    job.Version,
		PublishedVersion: publishedVersion,
		Feature:          featureSnapshot,
	}
	pv := &productentity.ProductVersion{
		Version:         job.Version,
		Status:          productStatus,
		FeatureSnapshot: featureSnapshot,
	}
	if _, err := a.DomainSVC.SyncProduct(ctx, product, pv); err != nil {
		logs.CtxErrorf(ctx, "agent_app persist build result productID=%d: %v", job.ProductID, err)
	}
}

// RecruitAgentApp installs the published agent_app product into a space and
// creates a read-only shadow agent draft that is pre-populated from the
// product's latest published snapshot.
func (a *AgentAppApplication) RecruitAgentApp(ctx context.Context, productID, spaceID, userID int64) (shadowAgentID int64, err error) {
	// 1. Install the product into the target space.
	installation, err := a.DomainSVC.Install(ctx, productID, spaceID, userID, "")
	if err != nil {
		return 0, fmt.Errorf("agent_app recruit: install productID=%d: %w", productID, err)
	}

	// 1.5 Idempotency: if this user already has an instance materialised from this
	//     product, reuse it instead of creating another shadow draft (which would
	//     leak a fresh sandbox + waste memory). Install above is idempotent.
	if existing, _ := a.ShadowWriter.FindUserInstance(ctx, userID, productID); existing > 0 {
		return existing, nil
	}

	// 2. Load the product's frozen agent_snapshot so the materialised instance is
	//    seeded with the real model/prompt/skills/capabilities. Without it the
	//    instance has no model bound and cannot run ("无法查看智能体").
	var snapshot map[string]any
	if product, perr := a.DomainSVC.GetVisibleProduct(ctx, productID, spaceID, userID); perr == nil && product != nil && product.Feature != nil {
		if s, ok := product.Feature["agent_snapshot"].(map[string]any); ok {
			snapshot = s
		}
	}

	// 3. Materialise a read-only shadow instance draft linked to the installed
	//    product, seeded from the snapshot.
	shadowAgentID, err = a.ShadowWriter.CreateShadowDraft(ctx, spaceID, userID, productID, installation.ProductVersion, snapshot)
	if err != nil {
		return 0, fmt.Errorf("agent_app recruit: create shadow draft productID=%d: %w", productID, err)
	}

	return shadowAgentID, nil
}
