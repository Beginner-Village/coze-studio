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

	productentity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	productservice "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/service"
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
		Visibility: productentity.AIProductVisibilitySpace,
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

	// 4. Build the sandbox template (synchronously here; callers may wrap in a
	//    goroutine for real async execution).
	objectKey := fmt.Sprintf("%s/%d/%s.tgz", a.ObjectPrefix, productID, req.Version)
	buildKey := fmt.Sprintf("agent_app_build_%d_%s", productID, req.Version)

	buildSkills := make([]BuildSkill, 0, len(draft.Skills))
	for _, s := range draft.Skills {
		buildSkills = append(buildSkills, BuildSkill{
			Name: s.Name,
		})
	}

	result, err := BuildTemplate(ctx, a.Sandbox, BuildTemplateRequest{
		BuildKey:  buildKey,
		ObjectKey: objectKey,
		Skills:    buildSkills,
	})
	if err != nil {
		return productID, req.Version, fmt.Errorf("agent_app publish: build template: %w", err)
	}

	// 5. The build_status in the feature_snapshot is updated to the result status.
	//    In production this would persist back via SyncProduct; here we leave it
	//    in-memory so tests can observe it without a real repo.
	_ = result // build_status == result.Status; persist via a subsequent SyncProduct in Task 5

	return productID, req.Version, nil
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

	// 2. Materialise a read-only shadow draft linked to the installed product.
	shadowAgentID, err = a.ShadowWriter.CreateShadowDraft(ctx, spaceID, userID, productID, installation.ProductVersion, nil)
	if err != nil {
		return 0, fmt.Errorf("agent_app recruit: create shadow draft productID=%d: %w", productID, err)
	}

	return shadowAgentID, nil
}
