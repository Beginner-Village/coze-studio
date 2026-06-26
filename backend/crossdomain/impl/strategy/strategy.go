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

	crossstrategy "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/strategy"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	strategyservice "github.com/ynet-dev/ynet-studio/backend/domain/strategy/service"
)

type impl struct {
	domainSVC strategyservice.Strategy
}

// InitDomainService wraps the strategy domain service as a crossstrategy.StrategyService.
func InitDomainService(domainSVC strategyservice.Strategy) crossstrategy.StrategyService {
	return &impl{domainSVC: domainSVC}
}

func (s *impl) GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error) {
	return s.domainSVC.GetStrategy(ctx, id)
}

func (s *impl) ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error) {
	return s.domainSVC.ListScenarios(ctx, strategyID)
}

func (s *impl) ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error) {
	return s.domainSVC.ListCapabilities(ctx, scenarioID)
}

func (s *impl) ResolveCapability(ctx context.Context, capID int64) (*entity.Capability, error) {
	return s.domainSVC.ResolveCapability(ctx, capID)
}

// ListCapabilityIDsByStrategies maps the cross-domain contract method to the
// domain service's ListCapabilityIDsByAgentStrategies (name differs by convention).
func (s *impl) ListCapabilityIDsByStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error) {
	return s.domainSVC.ListCapabilityIDsByAgentStrategies(ctx, strategyIDs)
}
