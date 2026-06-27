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

package repository

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

type StrategyDAO interface {
	CreateStrategy(ctx context.Context, s *entity.Strategy) (int64, error)
	UpdateStrategy(ctx context.Context, s *entity.Strategy) error
	DeleteStrategy(ctx context.Context, id int64) error
	GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error)
	ListStrategy(ctx context.Context, spaceID int64, page, size int) ([]*entity.Strategy, int64, error)
	PublishStrategy(ctx context.Context, id int64, version string) error

	CreateScenario(ctx context.Context, sc *entity.Scenario) (int64, error)
	UpdateScenario(ctx context.Context, sc *entity.Scenario) error
	DeleteScenario(ctx context.Context, id int64) error
	GetScenario(ctx context.Context, id int64) (*entity.Scenario, error)
	ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error)

	CreateCapability(ctx context.Context, c *entity.Capability) (int64, error)
	UpdateCapability(ctx context.Context, c *entity.Capability) error
	DeleteCapability(ctx context.Context, id int64) error
	ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error)
	MGetCapabilities(ctx context.Context, ids []int64) ([]*entity.Capability, error)
	ListCapabilityIDsByStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error)
}
