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

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

// Strategy is the domain service interface for strategy management.
// It wraps the StrategyDAO with validation and tree-assembly logic.
type Strategy interface {
	// --- Strategy CRUD ---
	CreateStrategy(ctx context.Context, req *CreateStrategyRequest) (*CreateStrategyResponse, error)
	UpdateStrategy(ctx context.Context, req *UpdateStrategyRequest) error
	DeleteStrategy(ctx context.Context, id int64) error
	GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error)
	ListStrategy(ctx context.Context, req *ListStrategyRequest) (*ListStrategyResponse, error)

	// GetDetail returns the full tree: strategy + scenarios + capabilities.
	GetDetail(ctx context.Context, id int64) (*entity.Strategy, error)

	// Publish sets status=Published and stamps version.
	Publish(ctx context.Context, id int64, version string) error

	// --- Scenario CRUD ---
	CreateScenario(ctx context.Context, req *CreateScenarioRequest) (*CreateScenarioResponse, error)
	UpdateScenario(ctx context.Context, req *UpdateScenarioRequest) error
	DeleteScenario(ctx context.Context, id int64) error
	GetScenario(ctx context.Context, id int64) (*entity.Scenario, error)
	ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error)

	// --- Capability CRUD ---
	CreateCapability(ctx context.Context, req *CreateCapabilityRequest) (*CreateCapabilityResponse, error)
	UpdateCapability(ctx context.Context, req *UpdateCapabilityRequest) error
	DeleteCapability(ctx context.Context, id int64) error
	ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error)

	// ResolveCapability returns a single capability by ID.
	ResolveCapability(ctx context.Context, capID int64) (*entity.Capability, error)

	// ListCapabilityIDsByAgentStrategies returns all capability IDs belonging to
	// the given strategy IDs (used by the agent authorization gate).
	ListCapabilityIDsByAgentStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error)
}

// --- Strategy request / response types ---

type CreateStrategyRequest struct {
	Strategy *entity.Strategy
}

type CreateStrategyResponse struct {
	ID int64
}

type UpdateStrategyRequest struct {
	Strategy *entity.Strategy
}

type ListStrategyRequest struct {
	SpaceID int64
	Page    int
	Size    int
}

type ListStrategyResponse struct {
	Strategies []*entity.Strategy
	Total      int64
}

// --- Scenario request / response types ---

type CreateScenarioRequest struct {
	Scenario *entity.Scenario
}

type CreateScenarioResponse struct {
	ID int64
}

type UpdateScenarioRequest struct {
	Scenario *entity.Scenario
}

// --- Capability request / response types ---

type CreateCapabilityRequest struct {
	Capability *entity.Capability
}

type CreateCapabilityResponse struct {
	ID int64
}

type UpdateCapabilityRequest struct {
	Capability *entity.Capability
}
