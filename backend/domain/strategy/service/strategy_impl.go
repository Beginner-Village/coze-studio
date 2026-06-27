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

package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/internal/dal"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/repository"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

type strategyService struct {
	dao repository.StrategyDAO
}

// NewStrategyService creates a Strategy domain service backed by the given DAO.
func NewStrategyService(dao repository.StrategyDAO) Strategy {
	return &strategyService{dao: dao}
}

// NewStrategyServiceWithDB creates a Strategy domain service using the given
// database and id-generator, constructing the DAO internally. This is the
// preferred entry-point for callers outside the domain/strategy tree that
// cannot import the internal/dal package directly.
func NewStrategyServiceWithDB(db *gorm.DB, gen idgen.IDGenerator) Strategy {
	return NewStrategyService(dal.NewStrategyDAO(db, gen))
}

// ---- Strategy CRUD ----

func (s *strategyService) CreateStrategy(ctx context.Context, req *CreateStrategyRequest) (*CreateStrategyResponse, error) {
	if req.Strategy.Name == "" {
		return nil, errors.New("strategy name must not be empty")
	}
	id, err := s.dao.CreateStrategy(ctx, req.Strategy)
	if err != nil {
		return nil, err
	}
	return &CreateStrategyResponse{ID: id}, nil
}

func (s *strategyService) UpdateStrategy(ctx context.Context, req *UpdateStrategyRequest) error {
	if req.Strategy.Name == "" {
		return errors.New("strategy name must not be empty")
	}
	return s.dao.UpdateStrategy(ctx, req.Strategy)
}

func (s *strategyService) DeleteStrategy(ctx context.Context, id int64) error {
	return s.dao.DeleteStrategy(ctx, id)
}

func (s *strategyService) GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error) {
	return s.dao.GetStrategy(ctx, id)
}

func (s *strategyService) ListStrategy(ctx context.Context, req *ListStrategyRequest) (*ListStrategyResponse, error) {
	list, total, err := s.dao.ListStrategy(ctx, req.SpaceID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	return &ListStrategyResponse{Strategies: list, Total: total}, nil
}

// GetDetail assembles the full tree: strategy → scenarios → capabilities.
func (s *strategyService) GetDetail(ctx context.Context, id int64) (*entity.Strategy, error) {
	strat, err := s.dao.GetStrategy(ctx, id)
	if err != nil {
		return nil, err
	}

	scenarios, err := s.dao.ListScenarios(ctx, id)
	if err != nil {
		return nil, err
	}

	for _, sc := range scenarios {
		caps, err := s.dao.ListCapabilities(ctx, sc.ID)
		if err != nil {
			return nil, err
		}
		sc.Capabilities = caps
	}

	strat.Scenarios = scenarios
	return strat, nil
}

// Publish sets status=Published and stamps version on the strategy.
func (s *strategyService) Publish(ctx context.Context, id int64, version string) error {
	return s.dao.PublishStrategy(ctx, id, version)
}

// ---- Scenario CRUD ----

func (s *strategyService) CreateScenario(ctx context.Context, req *CreateScenarioRequest) (*CreateScenarioResponse, error) {
	if req.Scenario.Name == "" {
		return nil, errors.New("scenario name must not be empty")
	}
	id, err := s.dao.CreateScenario(ctx, req.Scenario)
	if err != nil {
		return nil, err
	}
	return &CreateScenarioResponse{ID: id}, nil
}

func (s *strategyService) UpdateScenario(ctx context.Context, req *UpdateScenarioRequest) error {
	if req.Scenario.Name == "" {
		return errors.New("scenario name must not be empty")
	}
	return s.dao.UpdateScenario(ctx, req.Scenario)
}

func (s *strategyService) DeleteScenario(ctx context.Context, id int64) error {
	return s.dao.DeleteScenario(ctx, id)
}

func (s *strategyService) GetScenario(ctx context.Context, id int64) (*entity.Scenario, error) {
	return s.dao.GetScenario(ctx, id)
}

func (s *strategyService) ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error) {
	return s.dao.ListScenarios(ctx, strategyID)
}

// ---- Capability CRUD ----

func (s *strategyService) CreateCapability(ctx context.Context, req *CreateCapabilityRequest) (*CreateCapabilityResponse, error) {
	if req.Capability.Type == "" {
		return nil, errors.New("capability type must not be empty")
	}
	id, err := s.dao.CreateCapability(ctx, req.Capability)
	if err != nil {
		return nil, err
	}
	return &CreateCapabilityResponse{ID: id}, nil
}

func (s *strategyService) UpdateCapability(ctx context.Context, req *UpdateCapabilityRequest) error {
	return s.dao.UpdateCapability(ctx, req.Capability)
}

func (s *strategyService) DeleteCapability(ctx context.Context, id int64) error {
	return s.dao.DeleteCapability(ctx, id)
}

func (s *strategyService) ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error) {
	return s.dao.ListCapabilities(ctx, scenarioID)
}

// ResolveCapability returns the capability with the given ID.
func (s *strategyService) ResolveCapability(ctx context.Context, capID int64) (*entity.Capability, error) {
	caps, err := s.dao.MGetCapabilities(ctx, []int64{capID})
	if err != nil {
		return nil, err
	}
	if len(caps) == 0 {
		return nil, errors.New("capability not found")
	}
	return caps[0], nil
}

// ListCapabilityIDsByAgentStrategies returns all capability IDs for the given strategy IDs.
func (s *strategyService) ListCapabilityIDsByAgentStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error) {
	return s.dao.ListCapabilityIDsByStrategies(ctx, strategyIDs)
}
