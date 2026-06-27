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

package usermemorydal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	usermemory "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/usermemory"
	"github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/orm"
)

func newTestDAO(t *testing.T) *UserMemoryDAO {
	t.Helper()
	mockDBGen := orm.NewMockDB()
	mockDBGen.AddTable(&userMemoryPO{})
	db, err := mockDBGen.DB()
	require.NoError(t, err)
	return NewUserMemoryDAO(db)
}

func TestUserMemoryDAO_SaveByKeySupersedes(t *testing.T) {
	ctx := context.Background()
	dao := newTestDAO(t)

	_, err := dao.Save(ctx, &usermemory.UserMemory{UserID: 42, Kind: usermemory.KindPreference, MemKey: "tone", Content: "concise"})
	require.NoError(t, err)
	updated, err := dao.Save(ctx, &usermemory.UserMemory{UserID: 42, Kind: usermemory.KindPreference, MemKey: "tone", Content: "very concise, no fluff"})
	require.NoError(t, err)
	assert.Equal(t, "very concise, no fluff", updated.Content)

	got, err := dao.Recall(ctx, 42, usermemory.RecallQuery{})
	require.NoError(t, err)
	toneCount := 0
	for _, m := range got {
		if m.MemKey == "tone" {
			toneCount++
			assert.Equal(t, "very concise, no fluff", m.Content)
		}
	}
	assert.Equal(t, 1, toneCount, "keyed save must supersede, not duplicate")
}

func TestUserMemoryDAO_KeylessAlwaysInserts(t *testing.T) {
	ctx := context.Background()
	dao := newTestDAO(t)

	_, err := dao.Save(ctx, &usermemory.UserMemory{UserID: 7, Kind: usermemory.KindFact, Content: "uses Go and TypeScript"})
	require.NoError(t, err)
	_, err = dao.Save(ctx, &usermemory.UserMemory{UserID: 7, Kind: usermemory.KindFact, Content: "deploys to server 224"})
	require.NoError(t, err)

	got, err := dao.Recall(ctx, 7, usermemory.RecallQuery{})
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestUserMemoryDAO_RecallProfileAlwaysAndQueryFilters(t *testing.T) {
	ctx := context.Background()
	dao := newTestDAO(t)

	_, err := dao.Save(ctx, &usermemory.UserMemory{UserID: 1, Kind: usermemory.KindProfile, MemKey: "identity", Content: "Senior backend engineer at a bank"})
	require.NoError(t, err)
	_, err = dao.Save(ctx, &usermemory.UserMemory{UserID: 1, Kind: usermemory.KindFact, Content: "prefers Postgres for analytics"})
	require.NoError(t, err)
	_, err = dao.Save(ctx, &usermemory.UserMemory{UserID: 1, Kind: usermemory.KindFact, Content: "likes hiking on weekends"})
	require.NoError(t, err)

	got, err := dao.Recall(ctx, 1, usermemory.RecallQuery{Query: "Postgres"})
	require.NoError(t, err)
	var hasProfile, hasPostgres, hasHiking bool
	for _, m := range got {
		switch {
		case m.Kind == usermemory.KindProfile:
			hasProfile = true
		case m.Content == "prefers Postgres for analytics":
			hasPostgres = true
		case m.Content == "likes hiking on weekends":
			hasHiking = true
		}
	}
	assert.True(t, hasProfile, "profile must always be recalled regardless of query")
	assert.True(t, hasPostgres, "query should surface the matching fact")
	assert.False(t, hasHiking, "non-matching, non-profile facts filtered out by query")
}

func TestUserMemoryDAO_PerUserIsolation(t *testing.T) {
	ctx := context.Background()
	dao := newTestDAO(t)

	_, err := dao.Save(ctx, &usermemory.UserMemory{UserID: 100, Kind: usermemory.KindFact, Content: "user-100 secret"})
	require.NoError(t, err)
	_, err = dao.Save(ctx, &usermemory.UserMemory{UserID: 200, Kind: usermemory.KindFact, Content: "user-200 secret"})
	require.NoError(t, err)

	got, err := dao.Recall(ctx, 100, usermemory.RecallQuery{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "user-100 secret", got[0].Content)
	assert.Equal(t, int64(100), got[0].UserID)
}
