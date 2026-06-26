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
	"fmt"

	resCommon "github.com/ynet-dev/ynet-studio/backend/api/model/resource/common"
	apiModel "github.com/ynet-dev/ynet-studio/backend/api/model/data/strategy"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/application/search"
	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	strategyEntity "github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	strategyDomain "github.com/ynet-dev/ynet-studio/backend/domain/strategy/service"
	searchEntity "github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
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

// ---------- converters ----------

func toCapabilityInfo(c *strategyEntity.Capability) *apiModel.CapabilityInfo {
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
	}
}

func toScenarioInfo(sc *strategyEntity.Scenario) *apiModel.ScenarioInfo {
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
		info.Capabilities = append(info.Capabilities, toCapabilityInfo(cap))
	}
	return info
}

func toStrategyInfo(st *strategyEntity.Strategy) *apiModel.StrategyInfo {
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
		info.Scenarios = append(info.Scenarios, toScenarioInfo(sc))
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

	// Publish resource event so the search index stays current.
	if pubErr := s.Eventbus.PublishResources(ctx, &searchEntity.ResourceDomainEvent{
		OpType: searchEntity.Created,
		Resource: &searchEntity.ResourceDocument{
			ResType: resCommon.ResType_Strategy,
			ResID:   res.ID,
			Name:    &req.Name,
			SpaceID: &req.SpaceID,
			OwnerID: &uid,
			PublishStatus: ptr.Of(resCommon.PublishStatus_UnPublished),
		},
	}); pubErr != nil {
		return nil, fmt.Errorf("publish resource failed: %w", pubErr)
	}

	// Fetch the newly created strategy to return full info.
	created, err := s.DomainSVC.GetStrategy(ctx, res.ID)
	if err != nil {
		return nil, err
	}
	return &apiModel.CreateStrategyResponse{
		Code: 0,
		Msg:  "success",
		Data: toStrategyInfo(created),
	}, nil
}

func (s *StrategyApplicationService) GetStrategyDetail(ctx context.Context, req *apiModel.GetStrategyDetailRequest) (*apiModel.GetStrategyDetailResponse, error) {
	if _, err := s.requireUID(ctx); err != nil {
		return nil, err
	}

	detail, err := s.DomainSVC.GetDetail(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &apiModel.GetStrategyDetailResponse{
		Code: 0,
		Msg:  "success",
		Data: toStrategyInfo(detail),
	}, nil
}

func (s *StrategyApplicationService) UpdateStrategy(ctx context.Context, req *apiModel.UpdateStrategyRequest) (*apiModel.UpdateStrategyResponse, error) {
	uid, err := s.requireUID(ctx)
	if err != nil {
		return nil, err
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
	if _, err := s.requireUID(ctx); err != nil {
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
	if _, err := s.requireUID(ctx); err != nil {
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
	if _, err := s.requireUID(ctx); err != nil {
		return nil, err
	}
	if err := s.DomainSVC.DeleteScenario(ctx, req.ID); err != nil {
		return nil, err
	}
	return &apiModel.DeleteScenarioResponse{Code: 0, Msg: "success"}, nil
}

// ---------- Capability operations ----------

func (s *StrategyApplicationService) AddCapability(ctx context.Context, req *apiModel.AddCapabilityRequest) (*apiModel.AddCapabilityResponse, error) {
	if _, err := s.requireUID(ctx); err != nil {
		return nil, err
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
	if _, err := s.requireUID(ctx); err != nil {
		return nil, err
	}

	existing := &strategyEntity.Capability{
		ID:               req.ID,
		RefVersion:       req.RefVersion,
		PromptContent:    req.PromptContent,
		RetrieveConfig:   req.RetrieveConfig,
		AliasName:        req.AliasName,
		AliasDescription: req.AliasDescription,
		SortOrder:        req.SortOrder,
	}
	if err := s.DomainSVC.UpdateCapability(ctx, &strategyDomain.UpdateCapabilityRequest{Capability: existing}); err != nil {
		return nil, err
	}
	return &apiModel.UpdateCapabilityResponse{Code: 0, Msg: "success"}, nil
}

func (s *StrategyApplicationService) DeleteCapability(ctx context.Context, req *apiModel.DeleteCapabilityRequest) (*apiModel.DeleteCapabilityResponse, error) {
	if _, err := s.requireUID(ctx); err != nil {
		return nil, err
	}
	if err := s.DomainSVC.DeleteCapability(ctx, req.ID); err != nil {
		return nil, err
	}
	return &apiModel.DeleteCapabilityResponse{Code: 0, Msg: "success"}, nil
}
