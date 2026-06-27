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

package strategy

import (
	"context"
	"encoding/json"
	"fmt"

	resCommon "github.com/ynet-dev/ynet-studio/backend/api/model/resource/common"
	apiModel "github.com/ynet-dev/ynet-studio/backend/api/model/data/strategy"
	pluginModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/plugin"
	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/application/search"
	crossplugin "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/plugin"
	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	strategyEntity "github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	strategyDomain "github.com/ynet-dev/ynet-studio/backend/domain/strategy/service"
	searchEntity "github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// StrategyApplicationService is the application-layer orchestrator for strategy management.
type StrategyApplicationService struct {
	DomainSVC strategyDomain.Strategy
	Eventbus  search.ResourceEventBus
}

// StrategyApplicationSVC is the package-level singleton.
var StrategyApplicationSVC = &StrategyApplicationService{}

// ---------- auth helper ----------

func (s *StrategyApplicationService) requireUID(ctx context.Context) (int64, error) {
	uid := ctxutil.GetUIDFromCtx(ctx)
	if uid == nil {
		return 0, errorx.New(errno.ErrMemoryPermissionCode, errorx.KV("msg", "session required"))
	}
	return *uid, nil
}

func (s *StrategyApplicationService) checkSpaceAccess(ctx context.Context, uid, spaceID int64) error {
	spaces, err := crossuser.DefaultSVC().GetUserSpaceList(ctx, uid)
	if err != nil {
		return err
	}
	for _, sp := range spaces {
		if sp.ID == spaceID {
			return nil
		}
	}
	return errorx.New(errno.ErrMemoryPermissionCode, errorx.KV("msg", "space id is invalid"))
}

// ---------- schema helpers (mirrors node_tool_strategy.go logic) ----------

// capabilitySchemaForMgmt derives the input JSON schema for a capability so the
// strategy editor can display "what params the model passes" for each capability.
// Logic mirrors capabilityInputSchema in node_tool_strategy.go. Any failure is
// best-effort: logged at debug level, nil returned (schema field omitted).
func capabilitySchemaForMgmt(ctx context.Context, c *strategyEntity.Capability) map[string]any {
	switch c.Type {
	case strategyEntity.CapabilityTypePrompt:
		return nil // prompt takes no inputs; omit schema entirely

	case strategyEntity.CapabilityTypeKnowledge:
		return map[string]any{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Search query"},
				"top_k": map[string]any{"type": "integer", "description": "Number of results to return"},
			},
		}

	case strategyEntity.CapabilityTypeWorkflow:
		wfSVC := crossworkflow.DefaultSVC()
		if wfSVC == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: workflow service unavailable for cap %d", c.ID)
			return nil
		}
		policy := &vo.GetPolicy{
			ID:    c.RefID,
			QType: workflowModel.FromLatestVersion,
		}
		if c.RefVersion != "" {
			policy.QType = workflowModel.FromSpecificVersion
			policy.Version = c.RefVersion
		}
		wfTools, err := wfSVC.WorkflowAsModelTool(ctx, []*vo.GetPolicy{policy})
		if err != nil || len(wfTools) == 0 {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: load workflow tool for cap %d failed: %v", c.ID, err)
			return nil
		}
		info, err := wfTools[0].Info(ctx)
		if err != nil || info == nil || info.ParamsOneOf == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: workflow tool Info for cap %d failed: %v", c.ID, err)
			return nil
		}
		js, err := info.ParamsOneOf.ToJSONSchema()
		if err != nil || js == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: ToJSONSchema for cap %d failed: %v", c.ID, err)
			return nil
		}
		b, err := json.Marshal(js)
		if err != nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: marshal schema for cap %d failed: %v", c.ID, err)
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil || m == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: unmarshal schema for cap %d failed: %v", c.ID, err)
			return nil
		}
		if _, ok := m["type"]; !ok {
			m["type"] = "object"
		}
		if _, ok := m["properties"]; !ok {
			m["properties"] = map[string]any{}
		}
		return m

	case strategyEntity.CapabilityTypePlugin:
		pluginSVC := crossplugin.DefaultSVC()
		if pluginSVC == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: plugin service unavailable for cap %d", c.ID)
			return nil
		}
		req := &pluginModel.ToolsInvokableRequest{
			PluginEntity: pluginModel.PluginEntity{
				PluginID:      c.RefSubID,
				PluginVersion: ptr.Of(c.RefVersion),
			},
			ToolsInvokableInfo: map[int64]*pluginModel.ToolsInvokableInfo{
				c.RefID: {ToolID: c.RefID},
			},
			IsDraft: false,
		}
		toolMap, err := pluginSVC.GetPluginInvokableTools(ctx, req)
		if err != nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: load plugin tool for cap %d failed: %v", c.ID, err)
			return nil
		}
		pt, ok := toolMap[c.RefID]
		if !ok || pt == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: plugin tool %d not found for cap %d", c.RefID, c.ID)
			return nil
		}
		info, err := pt.Info(ctx)
		if err != nil || info == nil || info.ParamsOneOf == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: plugin tool Info for cap %d failed: %v", c.ID, err)
			return nil
		}
		js, err := info.ParamsOneOf.ToJSONSchema()
		if err != nil || js == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: plugin ToJSONSchema for cap %d failed: %v", c.ID, err)
			return nil
		}
		b, err := json.Marshal(js)
		if err != nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: plugin marshal schema for cap %d failed: %v", c.ID, err)
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil || m == nil {
			logs.CtxDebugf(ctx, "capabilitySchemaForMgmt: plugin unmarshal schema for cap %d failed: %v", c.ID, err)
			return nil
		}
		if _, ok := m["type"]; !ok {
			m["type"] = "object"
		}
		if _, ok := m["properties"]; !ok {
			m["properties"] = map[string]any{}
		}
		return m

	default:
		return nil
	}
}

// ---------- converters ----------

func toCapabilityInfo(ctx context.Context, c *strategyEntity.Capability) *apiModel.CapabilityInfo {
	if c == nil {
		return nil
	}
	return &apiModel.CapabilityInfo{
		ID:               c.ID,
		StrategyID:       c.StrategyID,
		ScenarioID:       c.ScenarioID,
		Type:             c.Type,
		RefID:            c.RefID,
		RefSubID:         c.RefSubID,
		RefVersion:       c.RefVersion,
		PromptContent:    c.PromptContent,
		RetrieveConfig:   c.RetrieveConfig,
		AliasName:        c.AliasName,
		AliasDescription: c.AliasDescription,
		SortOrder:        c.SortOrder,
		Schema:           capabilitySchemaForMgmt(ctx, c),
	}
}

func toScenarioInfo(ctx context.Context, sc *strategyEntity.Scenario) *apiModel.ScenarioInfo {
	if sc == nil {
		return nil
	}
	info := &apiModel.ScenarioInfo{
		ID:          sc.ID,
		StrategyID:  sc.StrategyID,
		Name:        sc.Name,
		Description: sc.Description,
		SortOrder:   sc.SortOrder,
	}
	for _, cap := range sc.Capabilities {
		info.Capabilities = append(info.Capabilities, toCapabilityInfo(ctx, cap))
	}
	return info
}

func toStrategyInfo(ctx context.Context, st *strategyEntity.Strategy) *apiModel.StrategyInfo {
	if st == nil {
		return nil
	}
	info := &apiModel.StrategyInfo{
		ID:          st.ID,
		SpaceID:     st.SpaceID,
		AppID:       st.AppID,
		CreatorID:   st.CreatorID,
		Name:        st.Name,
		Description: st.Description,
		IconURI:     st.IconURI,
		Status:      st.Status,
		Version:     st.Version,
	}
	for _, sc := range st.Scenarios {
		info.Scenarios = append(info.Scenarios, toScenarioInfo(ctx, sc))
	}
	return info
}

// ---------- Strategy operations ----------

func (s *StrategyApplicationService) CreateStrategy(ctx context.Context, req *apiModel.CreateStrategyRequest) (*apiModel.CreateStrategyResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, req.SpaceID); err != nil {
		return nil, err
	}

	res, err := s.DomainSVC.CreateStrategy(ctx, &strategyDomain.CreateStrategyRequest{
		Strategy: &strategyEntity.Strategy{
			SpaceID:     req.SpaceID,
			CreatorID:   uid,
			Name:        req.Name,
			Description: req.Description,
			IconURI:     req.IconURI,
		},
	})
	if err != nil {
		return nil, err
	}

	// I-2: Fetch the newly created strategy first so a re-fetch failure does not
	// leave us in a state where we've published an event but returned an error.
	// The event publish is the last step — success is authoritative.
	created, err := s.DomainSVC.GetStrategy(ctx, res.ID)
	if err != nil {
		return nil, err
	}

	// Publish resource event so the search index stays current.
	if pubErr := s.Eventbus.PublishResources(ctx, &searchEntity.ResourceDomainEvent{
		OpType: searchEntity.Created,
		Resource: &searchEntity.ResourceDocument{
			ResType:       resCommon.ResType_Strategy,
			ResID:         res.ID,
			Name:          &req.Name,
			SpaceID:       &req.SpaceID,
			OwnerID:       &uid,
			PublishStatus: ptr.Of(resCommon.PublishStatus_UnPublished),
		},
	}); pubErr != nil {
		return nil, fmt.Errorf("publish resource failed: %w", pubErr)
	}

	return &apiModel.CreateStrategyResponse{
		Code: 0,
		Msg:  "success",
		Data: toStrategyInfo(ctx, created),
	}, nil
}

func (s *StrategyApplicationService) GetStrategyDetail(ctx context.Context, req *apiModel.GetStrategyDetailRequest) (*apiModel.GetStrategyDetailResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	detail, err := s.DomainSVC.GetDetail(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	// C-1: verify caller belongs to the strategy's space.
	if err := s.checkSpaceAccess(ctx, uid, detail.SpaceID); err != nil {
		return nil, err
	}
	return &apiModel.GetStrategyDetailResponse{
		Code: 0,
		Msg:  "success",
		Data: toStrategyInfo(ctx, detail),
	}, nil
}

func (s *StrategyApplicationService) UpdateStrategy(ctx context.Context, req *apiModel.UpdateStrategyRequest) (*apiModel.UpdateStrategyResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// I-1: reject no-op updates — at least one field must be provided.
	if req.Name == "" && req.Description == "" && req.IconURI == "" {
		return nil, errorx.New(errno.ErrMemoryInvalidParamCode, errorx.KV("msg", "at least one of name, description, icon_uri must be provided"))
	}

	// Fetch existing to get SpaceID for auth check.
	existing, err := s.DomainSVC.GetStrategy(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, existing.SpaceID); err != nil {
		return nil, err
	}

	// Apply patch — only overwrite non-zero fields.
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.IconURI != "" {
		existing.IconURI = req.IconURI
	}

	// I-3: UpdateStrategy DAO does a column-scoped map[string]any update (not a
	// full-row replace), so passing the loaded entity with an empty Scenarios
	// slice does NOT clear child associations — safe as-is.
	if err := s.DomainSVC.UpdateStrategy(ctx, &strategyDomain.UpdateStrategyRequest{Strategy: existing}); err != nil {
		return nil, err
	}
	return &apiModel.UpdateStrategyResponse{Code: 0, Msg: "success"}, nil
}

func (s *StrategyApplicationService) DeleteStrategy(ctx context.Context, req *apiModel.DeleteStrategyRequest) (*apiModel.DeleteStrategyResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.DomainSVC.GetStrategy(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, existing.SpaceID); err != nil {
		return nil, err
	}

	if err := s.DomainSVC.DeleteStrategy(ctx, req.ID); err != nil {
		return nil, err
	}

	if pubErr := s.Eventbus.PublishResources(ctx, &searchEntity.ResourceDomainEvent{
		OpType: searchEntity.Deleted,
		Resource: &searchEntity.ResourceDocument{
			ResType: resCommon.ResType_Strategy,
			ResID:   req.ID,
		},
	}); pubErr != nil {
		return nil, pubErr
	}
	return &apiModel.DeleteStrategyResponse{Code: 0, Msg: "success"}, nil
}

func (s *StrategyApplicationService) PublishStrategy(ctx context.Context, req *apiModel.PublishStrategyRequest) (*apiModel.PublishStrategyResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.DomainSVC.GetStrategy(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, existing.SpaceID); err != nil {
		return nil, err
	}

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	if err := s.DomainSVC.Publish(ctx, req.ID, version); err != nil {
		return nil, err
	}

	name := existing.Name
	if pubErr := s.Eventbus.PublishResources(ctx, &searchEntity.ResourceDomainEvent{
		OpType: searchEntity.Updated,
		Resource: &searchEntity.ResourceDocument{
			ResType:       resCommon.ResType_Strategy,
			ResID:         req.ID,
			Name:          &name,
			SpaceID:       &existing.SpaceID,
			PublishStatus: ptr.Of(resCommon.PublishStatus_Published),
		},
	}); pubErr != nil {
		return nil, fmt.Errorf("publish resource event failed: %w", pubErr)
	}
	return &apiModel.PublishStrategyResponse{Code: 0, Msg: "success"}, nil
}

// ---------- Scenario operations ----------

func (s *StrategyApplicationService) CreateScenario(ctx context.Context, req *apiModel.CreateScenarioRequest) (*apiModel.CreateScenarioResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// C-2: verify caller belongs to the parent strategy's space.
	parent, err := s.DomainSVC.GetStrategy(ctx, req.StrategyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, parent.SpaceID); err != nil {
		return nil, err
	}

	res, err := s.DomainSVC.CreateScenario(ctx, &strategyDomain.CreateScenarioRequest{
		Scenario: &strategyEntity.Scenario{
			StrategyID:  req.StrategyID,
			Name:        req.Name,
			Description: req.Description,
			SortOrder:   req.SortOrder,
		},
	})
	if err != nil {
		return nil, err
	}
	return &apiModel.CreateScenarioResponse{
		Code: 0,
		Msg:  "success",
		Data: &apiModel.ScenarioInfo{
			ID:         res.ID,
			StrategyID: req.StrategyID,
			Name:       req.Name,
		},
	}, nil
}

func (s *StrategyApplicationService) UpdateScenario(ctx context.Context, req *apiModel.UpdateScenarioRequest) (*apiModel.UpdateScenarioResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// C-3: resolve owning strategy and verify space access.
	sc, err := s.DomainSVC.GetScenario(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	parent, err := s.DomainSVC.GetStrategy(ctx, sc.StrategyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, parent.SpaceID); err != nil {
		return nil, err
	}

	// Build a patch entity — domain service applies non-zero fields.
	patch := &strategyEntity.Scenario{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
	}
	if err := s.DomainSVC.UpdateScenario(ctx, &strategyDomain.UpdateScenarioRequest{Scenario: patch}); err != nil {
		return nil, err
	}
	return &apiModel.UpdateScenarioResponse{Code: 0, Msg: "success"}, nil
}

func (s *StrategyApplicationService) DeleteScenario(ctx context.Context, req *apiModel.DeleteScenarioRequest) (*apiModel.DeleteScenarioResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// C-4: resolve owning strategy and verify space access before deletion.
	sc, err := s.DomainSVC.GetScenario(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	parent, err := s.DomainSVC.GetStrategy(ctx, sc.StrategyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, parent.SpaceID); err != nil {
		return nil, err
	}

	if err := s.DomainSVC.DeleteScenario(ctx, req.ID); err != nil {
		return nil, err
	}
	return &apiModel.DeleteScenarioResponse{Code: 0, Msg: "success"}, nil
}

// ---------- Capability operations ----------

func (s *StrategyApplicationService) AddCapability(ctx context.Context, req *apiModel.AddCapabilityRequest) (*apiModel.AddCapabilityResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// C-5: verify caller belongs to the parent strategy's space.
	parent, err := s.DomainSVC.GetStrategy(ctx, req.StrategyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, parent.SpaceID); err != nil {
		return nil, err
	}

	// C-8: verify the scenario actually belongs to the stated strategy (parent-child consistency).
	sc, err := s.DomainSVC.GetScenario(ctx, req.ScenarioID)
	if err != nil {
		return nil, err
	}
	if sc.StrategyID != req.StrategyID {
		return nil, errorx.New(errno.ErrMemoryInvalidParamCode, errorx.KV("msg", "scenario does not belong to the specified strategy"))
	}

	res, err := s.DomainSVC.CreateCapability(ctx, &strategyDomain.CreateCapabilityRequest{
		Capability: &strategyEntity.Capability{
			StrategyID:       req.StrategyID,
			ScenarioID:       req.ScenarioID,
			Type:             req.Type,
			RefID:            req.RefID,
			RefSubID:         req.RefSubID,
			RefVersion:       req.RefVersion,
			PromptContent:    req.PromptContent,
			RetrieveConfig:   req.RetrieveConfig,
			AliasName:        req.AliasName,
			AliasDescription: req.AliasDescription,
			SortOrder:        req.SortOrder,
		},
	})
	if err != nil {
		return nil, err
	}
	return &apiModel.AddCapabilityResponse{
		Code: 0,
		Msg:  "success",
		Data: &apiModel.CapabilityInfo{
			ID:               res.ID,
			StrategyID:       req.StrategyID,
			ScenarioID:       req.ScenarioID,
			Type:             req.Type,
			RefID:            req.RefID,
			RefSubID:         req.RefSubID,
			RefVersion:       req.RefVersion,
			PromptContent:    req.PromptContent,
			RetrieveConfig:   req.RetrieveConfig,
			AliasName:        req.AliasName,
			AliasDescription: req.AliasDescription,
			SortOrder:        req.SortOrder,
		},
	}, nil
}

func (s *StrategyApplicationService) UpdateCapability(ctx context.Context, req *apiModel.UpdateCapabilityRequest) (*apiModel.UpdateCapabilityResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// C-6: resolve owning strategy via capability and verify space access.
	cap, err := s.DomainSVC.ResolveCapability(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	parent, err := s.DomainSVC.GetStrategy(ctx, cap.StrategyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, parent.SpaceID); err != nil {
		return nil, err
	}

	patch := &strategyEntity.Capability{
		ID:               req.ID,
		RefVersion:       req.RefVersion,
		PromptContent:    req.PromptContent,
		RetrieveConfig:   req.RetrieveConfig,
		AliasName:        req.AliasName,
		AliasDescription: req.AliasDescription,
		SortOrder:        req.SortOrder,
	}
	if err := s.DomainSVC.UpdateCapability(ctx, &strategyDomain.UpdateCapabilityRequest{Capability: patch}); err != nil {
		return nil, err
	}
	return &apiModel.UpdateCapabilityResponse{Code: 0, Msg: "success"}, nil
}

func (s *StrategyApplicationService) DeleteCapability(ctx context.Context, req *apiModel.DeleteCapabilityRequest) (*apiModel.DeleteCapabilityResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
	}

	// C-7: resolve owning strategy via capability and verify space access before deletion.
	cap, err := s.DomainSVC.ResolveCapability(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	parent, err := s.DomainSVC.GetStrategy(ctx, cap.StrategyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkSpaceAccess(ctx, uid, parent.SpaceID); err != nil {
		return nil, err
	}

	if err := s.DomainSVC.DeleteCapability(ctx, req.ID); err != nil {
		return nil, err
	}
	return &apiModel.DeleteCapabilityResponse{Code: 0, Msg: "success"}, nil
}
