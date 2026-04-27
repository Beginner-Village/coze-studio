// Copyright 2025 ynet-dev Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package space_sync_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spacesync "github.com/ynet-dev/ynet-studio/backend/application/space/sync"
)

// resetMappingTable wipes the sync mapping table between subtests so they don't bleed.
func resetMappingTable(t *testing.T) {
	t.Helper()
	_, err := testRawDB.Exec("TRUNCATE TABLE space_sync_mapping")
	require.NoError(t, err)
}

func TestE2E_SyncMapping_UpsertAgainstRealMySQL(t *testing.T) {
	resetMappingTable(t)
	store := spacesync.NewSyncMappingStore(testGormDB, 1, 2)
	ctx := context.Background()

	// Initial insert
	require.NoError(t, store.UpsertMapping(ctx, nil, "agent", 100, 1001, 1234567890, "hash1"))

	got, ok := store.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), got)
}

func TestE2E_SyncMapping_OnConflictAgainstRealMySQL(t *testing.T) {
	// This test specifically validates that GORM's OnConflict clause produces
	// correct SQL for MySQL (uses INSERT … ON DUPLICATE KEY UPDATE), which is
	// different from SQLite's INSERT OR REPLACE.
	resetMappingTable(t)
	store := spacesync.NewSyncMappingStore(testGormDB, 1, 2)
	ctx := context.Background()

	require.NoError(t, store.UpsertMapping(ctx, nil, "agent", 100, 1001, 100, "hash-v1"))
	require.NoError(t, store.UpsertMapping(ctx, nil, "agent", 100, 9999, 200, "hash-v2"))

	got, ok := store.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(9999), got)
	assert.Equal(t, "hash-v2", store.GetContentHash("agent", 100))

	// Verify only one row exists in the DB (conflict updated, didn't create a duplicate).
	var count int64
	row := testRawDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM space_sync_mapping WHERE source_space_id = ? AND resource_type = ? AND source_resource_id = ?",
		1, "agent", 100)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(1), count, "OnConflict should update, not insert duplicate")
}

func TestE2E_SyncMapping_ReloadFromRealMySQL(t *testing.T) {
	resetMappingTable(t)
	ctx := context.Background()

	store1 := spacesync.NewSyncMappingStore(testGormDB, 1, 2)
	require.NoError(t, store1.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))
	require.NoError(t, store1.UpsertMapping(ctx, nil, "workflow", 200, 2002, 0))
	require.NoError(t, store1.UpsertMapping(ctx, nil, "plugin", 300, 3003, 0, "h"))

	// Fresh store reads from DB.
	store2 := spacesync.NewSyncMappingStore(testGormDB, 1, 2)
	require.NoError(t, store2.LoadAll(ctx))

	v, ok := store2.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.Equal(t, int64(1001), v)

	v, ok = store2.GetTargetID("workflow", 200)
	require.True(t, ok)
	assert.Equal(t, int64(2002), v)

	v, ok = store2.GetTargetID("plugin", 300)
	require.True(t, ok)
	assert.Equal(t, int64(3003), v)
	assert.Equal(t, "h", store2.GetContentHash("plugin", 300))
}

func TestE2E_SyncMapping_RemoveAgainstRealMySQL(t *testing.T) {
	resetMappingTable(t)
	store := spacesync.NewSyncMappingStore(testGormDB, 1, 2)
	ctx := context.Background()

	require.NoError(t, store.UpsertMapping(ctx, nil, "agent", 100, 1001, 0))
	require.NoError(t, store.RemoveMapping(ctx, "agent", 100))

	// Verify in DB
	var count int64
	row := testRawDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM space_sync_mapping WHERE resource_type = 'agent' AND source_resource_id = 100")
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(0), count)

	// Verify in cache
	_, ok := store.GetTargetID("agent", 100)
	assert.False(t, ok)
}

func TestE2E_SyncMapping_ConcurrentUpserts(t *testing.T) {
	// Race condition coverage: many concurrent UpsertMapping calls on the same key.
	// With OnConflict + unique index, all should converge.
	resetMappingTable(t)
	store := spacesync.NewSyncMappingStore(testGormDB, 1, 2)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Each goroutine writes a distinct target_id but the same source_id.
			err := store.UpsertMapping(ctx, nil, "agent", 100, int64(2000+i), int64(i))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Should still be exactly one row in the DB
	var count int64
	row := testRawDB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM space_sync_mapping WHERE resource_type = 'agent' AND source_resource_id = 100")
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, int64(1), count, "concurrent upserts should not create duplicates")

	// And we should be able to read some valid target ID.
	got, ok := store.GetTargetID("agent", 100)
	require.True(t, ok)
	assert.GreaterOrEqual(t, got, int64(2000))
	assert.Less(t, got, int64(2020))
}
