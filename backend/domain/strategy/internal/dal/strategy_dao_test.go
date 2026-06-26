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

package dal

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/internal/dal/model"
)

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

func newTestStrategyDAO(t *testing.T) *StrategyDAO {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Strategy{},
		&model.StrategyScenario{},
		&model.StrategyCapability{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return NewStrategyDAO(db, &fakeIDGen{})
}

func TestStrategyDAO_CreateAndGet(t *testing.T) {
	dao := newTestStrategyDAO(t)
	id, err := dao.CreateStrategy(context.Background(), &entity.Strategy{
		SpaceID: 1, CreatorID: 2, Name: "对公运营", Description: "银行对公",
	})
	require.NoError(t, err)
	got, err := dao.GetStrategy(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "对公运营", got.Name)
	require.Equal(t, int64(1), got.SpaceID)
	require.Equal(t, int64(2), got.CreatorID)
	require.Equal(t, "银行对公", got.Description)
}

func TestStrategyDAO_UpdateAndDelete(t *testing.T) {
	dao := newTestStrategyDAO(t)
	ctx := context.Background()

	id, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "orig"})
	require.NoError(t, err)

	err = dao.UpdateStrategy(ctx, &entity.Strategy{ID: id, SpaceID: 1, CreatorID: 1, Name: "updated"})
	require.NoError(t, err)

	got, err := dao.GetStrategy(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "updated", got.Name)

	err = dao.DeleteStrategy(ctx, id)
	require.NoError(t, err)

	_, err = dao.GetStrategy(ctx, id)
	require.Error(t, err)
}

func TestStrategyDAO_ListStrategy(t *testing.T) {
	dao := newTestStrategyDAO(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 10, CreatorID: 1, Name: "s"})
		require.NoError(t, err)
	}
	_, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 99, CreatorID: 1, Name: "other"})
	require.NoError(t, err)

	list, total, err := dao.ListStrategy(ctx, 10, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, list, 3)
}

func TestStrategyDAO_PublishStrategy(t *testing.T) {
	dao := newTestStrategyDAO(t)
	ctx := context.Background()

	id, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "pub"})
	require.NoError(t, err)

	err = dao.PublishStrategy(ctx, id, "v1.0")
	require.NoError(t, err)

	got, err := dao.GetStrategy(ctx, id)
	require.NoError(t, err)
	require.Equal(t, entity.StatusPublished, got.Status)
	require.Equal(t, "v1.0", got.Version)
}

func TestStrategyDAO_ScenarioCRUD(t *testing.T) {
	dao := newTestStrategyDAO(t)
	ctx := context.Background()

	sid, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s"})
	require.NoError(t, err)

	scID, err := dao.CreateScenario(ctx, &entity.Scenario{
		StrategyID: sid, Name: "sc1", Description: "desc", SortOrder: 1,
	})
	require.NoError(t, err)

	scID2, err := dao.CreateScenario(ctx, &entity.Scenario{
		StrategyID: sid, Name: "sc2", SortOrder: 2,
	})
	require.NoError(t, err)

	list, err := dao.ListScenarios(ctx, sid)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "sc1", list[0].Name)

	err = dao.UpdateScenario(ctx, &entity.Scenario{ID: scID, StrategyID: sid, Name: "sc1-updated"})
	require.NoError(t, err)

	list, err = dao.ListScenarios(ctx, sid)
	require.NoError(t, err)
	require.Equal(t, "sc1-updated", list[0].Name)

	err = dao.DeleteScenario(ctx, scID2)
	require.NoError(t, err)

	list, err = dao.ListScenarios(ctx, sid)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestStrategyDAO_CapabilityCRUD(t *testing.T) {
	dao := newTestStrategyDAO(t)
	ctx := context.Background()

	sid, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s"})
	require.NoError(t, err)

	scID, err := dao.CreateScenario(ctx, &entity.Scenario{StrategyID: sid, Name: "sc"})
	require.NoError(t, err)

	cID, err := dao.CreateCapability(ctx, &entity.Capability{
		StrategyID: sid, ScenarioID: scID,
		Type: entity.CapabilityTypeWorkflow, RefID: 99,
		AliasName: "wf-alias",
	})
	require.NoError(t, err)

	list, err := dao.ListCapabilities(ctx, scID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, entity.CapabilityTypeWorkflow, list[0].Type)
	require.Equal(t, "wf-alias", list[0].AliasName)

	err = dao.UpdateCapability(ctx, &entity.Capability{
		ID: cID, StrategyID: sid, ScenarioID: scID,
		Type: entity.CapabilityTypePlugin, AliasName: "plugin-alias",
	})
	require.NoError(t, err)

	list, err = dao.ListCapabilities(ctx, scID)
	require.NoError(t, err)
	require.Equal(t, entity.CapabilityTypePlugin, list[0].Type)
	require.Equal(t, "plugin-alias", list[0].AliasName)

	mget, err := dao.MGetCapabilities(ctx, []int64{cID})
	require.NoError(t, err)
	require.Len(t, mget, 1)
	require.Equal(t, cID, mget[0].ID)

	err = dao.DeleteCapability(ctx, cID)
	require.NoError(t, err)

	list, err = dao.ListCapabilities(ctx, scID)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestStrategyDAO_ListCapabilityIDsByStrategies(t *testing.T) {
	dao := newTestStrategyDAO(t)
	ctx := context.Background()

	sid1, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s1"})
	require.NoError(t, err)
	sid2, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s2"})
	require.NoError(t, err)
	sid3, err := dao.CreateStrategy(ctx, &entity.Strategy{SpaceID: 1, CreatorID: 1, Name: "s3"})
	require.NoError(t, err)

	scID1, err := dao.CreateScenario(ctx, &entity.Scenario{StrategyID: sid1, Name: "sc1"})
	require.NoError(t, err)
	scID2, err := dao.CreateScenario(ctx, &entity.Scenario{StrategyID: sid2, Name: "sc2"})
	require.NoError(t, err)

	cID1, err := dao.CreateCapability(ctx, &entity.Capability{StrategyID: sid1, ScenarioID: scID1, Type: entity.CapabilityTypeWorkflow})
	require.NoError(t, err)
	cID2, err := dao.CreateCapability(ctx, &entity.Capability{StrategyID: sid2, ScenarioID: scID2, Type: entity.CapabilityTypePlugin})
	require.NoError(t, err)

	ids, err := dao.ListCapabilityIDsByStrategies(ctx, []int64{sid1, sid2})
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{cID1, cID2}, ids)

	// sid3 has no capabilities
	ids, err = dao.ListCapabilityIDsByStrategies(ctx, []int64{sid3})
	require.NoError(t, err)
	require.Empty(t, ids)

	// Soft-delete one capability and confirm it is excluded from the auth id list.
	// This is the authorization-critical assertion: a deleted capability must NOT
	// remain in the set that gates agent invocation permissions.
	err = dao.DeleteCapability(ctx, cID1)
	require.NoError(t, err)

	ids, err = dao.ListCapabilityIDsByStrategies(ctx, []int64{sid1, sid2})
	require.NoError(t, err)
	require.NotContains(t, ids, cID1, "soft-deleted capability must not appear in auth id list")
	require.ElementsMatch(t, []int64{cID2}, ids)
}
