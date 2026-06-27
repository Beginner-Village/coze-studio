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

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

// StrategyService is the cross-domain contract for strategy queries consumed by
// the agent runtime. It exposes only the read/list methods needed by the 3
// progressive-disclosure tools; the full domain service interface lives in
// domain/strategy/service.
type StrategyService interface {
	GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error)
	ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error)
	ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error)
	ResolveCapability(ctx context.Context, capID int64) (*entity.Capability, error)
	ListCapabilityIDsByStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error)
}

var defaultSVC StrategyService

func DefaultSVC() StrategyService {
	return defaultSVC
}

func SetDefaultSVC(svc StrategyService) {
	defaultSVC = svc
}
