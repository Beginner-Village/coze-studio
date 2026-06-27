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

package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/internal/dal"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/service"
)

// fakeIDGen is a simple counter-based ID generator for tests.
type fakeIDGen struct{ next int64 }

func (f *fakeIDGen) GenID(_ context.Context) (int64, error) {
	f.next++
	return f.next, nil
}

func (f *fakeIDGen) GenMultiIDs(_ context.Context, counts int) ([]int64, error) {
	ids := make([]int64, counts)
	for i := 0; i < counts; i++ {
		f.next++
		ids[i] = f.next
	}
	return ids, nil
}

// newTestService spins up an in-memory sqlite DAO and wraps it in the service.
func newTestService(t *testing.T) service.Strategy {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&model.Strategy{}, &model.StrategyScenario{}, &model.StrategyCapability{})
	require.NoError(t, err)
	dao := dal.NewStrategyDAO(db, &fakeIDGen{})
	return service.NewStrategyService(dao)
}

// ---- GetDetail tree assembly ----

func TestStrategyService_GetDetail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Create strategy
	createResp, err := svc.CreateStrategy(ctx, &service.CreateStrategyRequest{
		Strategy: &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "银行策略"},
	})
	require.NoError(t, err)
	stratID := createResp.ID

	// Create two scenarios
	sc1Resp, err := svc.CreateScenario(ctx, &service.CreateScenarioRequest{
		Scenario: &entity.Scenario{StrategyID: stratID, Name: "场景一", SortOrder: 1},
	})
	require.NoError(t, err)

	sc2Resp, err := svc.CreateScenario(ctx, &service.CreateScenarioRequest{
		Scenario: &entity.Scenario{StrategyID: stratID, Name: "场景二", SortOrder: 2},
	})
	require.NoError(t, err)

	// Attach 2 capabilities to scenario 1, 1 to scenario 2
	_, err = svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{
			StrategyID: stratID, ScenarioID: sc1Resp.ID,
			Type: entity.CapabilityTypeWorkflow, AliasName: "wf1",
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{
			StrategyID: stratID, ScenarioID: sc1Resp.ID,
			Type: entity.CapabilityTypePlugin, AliasName: "plug1",
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{
			StrategyID: stratID, ScenarioID: sc2Resp.ID,
			Type: entity.CapabilityTypeKnowledge, AliasName: "kb1",
		},
	})
	require.NoError(t, err)

	// GetDetail should assemble the full tree
	detail, err := svc.GetDetail(ctx, stratID)
	require.NoError(t, err)
	require.Equal(t, stratID, detail.ID)
	require.Equal(t, "银行策略", detail.Name)
	require.Len(t, detail.Scenarios, 2)

	// Scenarios are ordered by SortOrder — scenario 1 first
	s1 := detail.Scenarios[0]
	require.Equal(t, "场景一", s1.Name)
	require.Len(t, s1.Capabilities, 2)

	s2 := detail.Scenarios[1]
	require.Equal(t, "场景二", s2.Name)
	require.Len(t, s2.Capabilities, 1)
}

// ---- Publish flips status + version ----

func TestStrategyService_Publish(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createResp, err := svc.CreateStrategy(ctx, &service.CreateStrategyRequest{
		Strategy: &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "待发布"},
	})
	require.NoError(t, err)

	// Verify draft status initially
	s, err := svc.GetStrategy(ctx, createResp.ID)
	require.NoError(t, err)
	require.Equal(t, entity.StatusDraft, s.Status)

	// Publish
	err = svc.Publish(ctx, createResp.ID, "v2.0")
	require.NoError(t, err)

	// Verify published
	s, err = svc.GetStrategy(ctx, createResp.ID)
	require.NoError(t, err)
	require.Equal(t, entity.StatusPublished, s.Status)
	require.Equal(t, "v2.0", s.Version)
}

// ---- ResolveCapability returns the correct capability ----

func TestStrategyService_ResolveCapability(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createResp, err := svc.CreateStrategy(ctx, &service.CreateStrategyRequest{
		Strategy: &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s"},
	})
	require.NoError(t, err)
	scResp, err := svc.CreateScenario(ctx, &service.CreateScenarioRequest{
		Scenario: &entity.Scenario{StrategyID: createResp.ID, Name: "sc"},
	})
	require.NoError(t, err)

	capResp, err := svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{
			StrategyID: createResp.ID, ScenarioID: scResp.ID,
			Type: entity.CapabilityTypePrompt, AliasName: "my-prompt",
		},
	})
	require.NoError(t, err)

	cap, err := svc.ResolveCapability(ctx, capResp.ID)
	require.NoError(t, err)
	require.Equal(t, capResp.ID, cap.ID)
	require.Equal(t, entity.CapabilityTypePrompt, cap.Type)
	require.Equal(t, "my-prompt", cap.AliasName)
}

// ---- ListCapabilityIDsByAgentStrategies delegates correctly ----

func TestStrategyService_ListCapabilityIDsByAgentStrategies(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	sid1, _ := svc.CreateStrategy(ctx, &service.CreateStrategyRequest{
		Strategy: &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s1"},
	})
	sid2, _ := svc.CreateStrategy(ctx, &service.CreateStrategyRequest{
		Strategy: &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s2"},
	})

	sc1, _ := svc.CreateScenario(ctx, &service.CreateScenarioRequest{
		Scenario: &entity.Scenario{StrategyID: sid1.ID, Name: "sc1"},
	})
	sc2, _ := svc.CreateScenario(ctx, &service.CreateScenarioRequest{
		Scenario: &entity.Scenario{StrategyID: sid2.ID, Name: "sc2"},
	})

	cap1, _ := svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{StrategyID: sid1.ID, ScenarioID: sc1.ID, Type: entity.CapabilityTypeWorkflow},
	})
	cap2, _ := svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{StrategyID: sid2.ID, ScenarioID: sc2.ID, Type: entity.CapabilityTypePlugin},
	})

	ids, err := svc.ListCapabilityIDsByAgentStrategies(ctx, []int64{sid1.ID, sid2.ID})
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{cap1.ID, cap2.ID}, ids)
}

// ---- Validation: name must be non-empty ----

func TestStrategyService_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateStrategy(ctx, &service.CreateStrategyRequest{
		Strategy: &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: ""},
	})
	require.Error(t, err, "empty name should be rejected")

	_, err = svc.CreateScenario(ctx, &service.CreateScenarioRequest{
		Scenario: &entity.Scenario{StrategyID: 1, Name: ""},
	})
	require.Error(t, err, "empty scenario name should be rejected")

	_, err = svc.CreateCapability(ctx, &service.CreateCapabilityRequest{
		Capability: &entity.Capability{StrategyID: 1, ScenarioID: 1, Type: ""},
	})
	require.Error(t, err, "empty capability type should be rejected")
}
