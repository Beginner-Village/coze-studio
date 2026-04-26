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

package spacesync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Each test gets its own in-memory DB to avoid cross-test bleed.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&SyncMapping{}))
	// Add the unique index that production schema would have, so OnConflict works.
	require.NoError(t, db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS uniq_source_resource ON space_sync_mapping(source_space_id, resource_type, source_resource_id)",
	).Error)
	return db
}

func TestSyncMappingStore_LoadAll_Empty(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	require.NoError(t, s.LoadAll(context.Background()))
	_, ok := s.GetTargetID("agent", 100)
	assert.False(t, ok)
}

func TestSyncMappingStore_UpsertMapping_NewRecord(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	ctx := context.Background()

	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 100, 1001, 12345, "hash1"))

	got, ok := s.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), got)
	assert.Equal(t, "hash1", s.GetContentHash("agent", 100))
}

func TestSyncMappingStore_UpsertMapping_ConflictUpdates(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	ctx := context.Background()

	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 100, 1001, 100, "hash-old"))
	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 100, 9999, 200, "hash-new"))

	got, ok := s.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(9999), got, "second upsert should overwrite target_resource_id")
	assert.Equal(t, "hash-new", s.GetContentHash("agent", 100))
}

func TestSyncMappingStore_UpsertMapping_DifferentResourceTypes(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	ctx := context.Background()

	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))
	require.NoError(t, s.UpsertMapping(ctx, nil, "workflow", 100, 2001, 0))

	a, ok := s.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), a)

	w, ok := s.GetTargetID("workflow", 100)
	require.True(t, ok)
	assert.Equal(t, int64(2001), w)
}

func TestSyncMappingStore_GetTargetID_Miss(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)

	_, ok := s.GetTargetID("nonexistent", 99999)
	assert.False(t, ok)
}

func TestSyncMappingStore_GetContentHash_Miss(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	got := s.GetContentHash("agent", 100)
	assert.Equal(t, "", got)
}

func TestSyncMappingStore_RemoveMapping(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	ctx := context.Background()

	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))
	require.NoError(t, s.RemoveMapping(ctx, "agent", 100))

	_, ok := s.GetTargetID("agent", 100)
	assert.False(t, ok, "removed mapping should not be findable in cache")

	// Reload from DB to confirm it's gone there too.
	s2 := NewSyncMappingStore(db, 1, 2)
	require.NoError(t, s2.LoadAll(ctx))
	_, ok = s2.GetTargetID("agent", 100)
	assert.False(t, ok, "removed mapping should not be in DB")
}

func TestSyncMappingStore_LoadAll_FetchesPersistedRows(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Insert via one store
	s1 := NewSyncMappingStore(db, 1, 2)
	require.NoError(t, s1.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))
	require.NoError(t, s1.UpsertMapping(ctx, nil, "agent", 200, 2002, 0))

	// Read via fresh store
	s2 := NewSyncMappingStore(db, 1, 2)
	require.NoError(t, s2.LoadAll(ctx))

	v1, ok := s2.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), v1)

	v2, ok := s2.GetTargetID("agent", 200)
	require.True(t, ok)
	assert.Equal(t, int64(2002), v2)
}

func TestSyncMappingStore_LoadAll_OnlyForRequestedSpacePair(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Two different source-target pairs
	s1 := NewSyncMappingStore(db, 1, 2)
	require.NoError(t, s1.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))

	s2 := NewSyncMappingStore(db, 3, 4)
	require.NoError(t, s2.UpsertMapping(ctx, nil, "agent", 100, 9999, 0))

	// Reload s1 — should only have its own mapping
	s1Reload := NewSyncMappingStore(db, 1, 2)
	require.NoError(t, s1Reload.LoadAll(ctx))
	v, ok := s1Reload.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), v, "should not see other space pair's mapping")
}

func TestSyncMappingStore_UpsertMapping_OptionalContentHash(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	ctx := context.Background()

	// Without content hash
	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))
	assert.Equal(t, "", s.GetContentHash("agent", 100), "no hash supplied → empty")

	// With content hash
	require.NoError(t, s.UpsertMapping(ctx, nil, "agent", 200, 2002, 0, "deadbeef"))
	assert.Equal(t, "deadbeef", s.GetContentHash("agent", 200))
}

func TestSyncMappingStore_UpsertMapping_AcceptsExternalTransaction(t *testing.T) {
	db := newTestDB(t)
	s := NewSyncMappingStore(db, 1, 2)
	ctx := context.Background()

	// Run upsert inside an external transaction; should work.
	err := db.Transaction(func(tx *gorm.DB) error {
		return s.UpsertMapping(ctx, tx, "agent", 100, 1001, 0)
	})
	require.NoError(t, err)

	got, ok := s.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), got)
}

func TestSyncMapping_TableName(t *testing.T) {
	assert.Equal(t, "space_sync_mapping", SyncMapping{}.TableName())
}

func TestMappingKey_Format(t *testing.T) {
	got := mappingKey("agent", 100)
	assert.Equal(t, "agent:100", got)
}
