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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/application/space/release"
)

func resetReleaseTable(t *testing.T) {
	t.Helper()
	_, err := testRawDB.Exec("TRUNCATE TABLE space_release")
	require.NoError(t, err)
}

func newE2ERelease(spaceID int64, version, status string) *release.SpaceRelease {
	return &release.SpaceRelease{
		SpaceID:     spaceID,
		Version:     version,
		Status:      status,
		SyncType:    "full",
		Manifest:    json.RawMessage(`{"version":"2.0.0"}`),
		Statistics:  json.RawMessage(`{"agents":1}`),
		PackageKey:  "p/" + version,
		PackageSize: 100,
		ContentHash: "deadbeef",
		CreatedBy:   1,
		CreatedAt:   1234567890,
		UpdatedAt:   1234567890,
	}
}

func TestE2E_ReleaseRepo_CreateAndFetch(t *testing.T) {
	resetReleaseTable(t)
	repo := release.NewReleaseRepo(testGormDB)
	ctx := context.Background()

	rec := newE2ERelease(100, "v1.0.0", release.StatusDraft)
	require.NoError(t, repo.Create(ctx, rec))
	assert.NotZero(t, rec.ID, "auto-incremented ID should be set")

	got, err := repo.GetByVersion(ctx, 100, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v1.0.0", got.Version)
	assert.JSONEq(t, `{"version":"2.0.0"}`, string(got.Manifest), "JSON manifest should round-trip via real MySQL JSON column")
}

func TestE2E_ReleaseRepo_VersionUniquenessPerSpace(t *testing.T) {
	resetReleaseTable(t)
	repo := release.NewReleaseRepo(testGormDB)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newE2ERelease(100, "v1.0.0", release.StatusDraft)))
	// Same version in a different space — should be allowed because (space_id, version) is the unique key.
	require.NoError(t, repo.Create(ctx, newE2ERelease(200, "v1.0.0", release.StatusDraft)))

	// Re-using same (space_id, version) should fail.
	err := repo.Create(ctx, newE2ERelease(100, "v1.0.0", release.StatusDraft))
	require.Error(t, err, "duplicate (space_id, version) should violate unique constraint")
}

func TestE2E_ReleaseRepo_GetLatestPublished_OrderingByCreatedAt(t *testing.T) {
	resetReleaseTable(t)
	repo := release.NewReleaseRepo(testGormDB)
	ctx := context.Background()

	r1 := newE2ERelease(100, "v1.0.0", release.StatusPublished)
	r1.CreatedAt = 100
	r2 := newE2ERelease(100, "v2.0.0", release.StatusDraft) // skipped — draft
	r2.CreatedAt = 200
	r3 := newE2ERelease(100, "v1.5.0", release.StatusPublished)
	r3.CreatedAt = 150
	require.NoError(t, repo.Create(ctx, r1))
	require.NoError(t, repo.Create(ctx, r2))
	require.NoError(t, repo.Create(ctx, r3))

	got, err := repo.GetLatestPublished(ctx, 100)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v1.5.0", got.Version, "should pick the latest *published* (by created_at), skipping draft v2.0.0")
}

func TestE2E_ReleaseRepo_UpdateStatusPersists(t *testing.T) {
	resetReleaseTable(t)
	repo := release.NewReleaseRepo(testGormDB)
	ctx := context.Background()

	rec := newE2ERelease(100, "v1.0.0", release.StatusDraft)
	require.NoError(t, repo.Create(ctx, rec))

	publishedAt := int64(99999)
	require.NoError(t, repo.UpdateStatus(ctx, rec.ID, release.StatusPublished, &publishedAt))

	// Verify directly from raw SQL
	var status string
	var pubAt *int64
	row := testRawDB.QueryRowContext(ctx,
		"SELECT status, published_at FROM space_release WHERE id = ?", rec.ID)
	require.NoError(t, row.Scan(&status, &pubAt))
	assert.Equal(t, release.StatusPublished, status)
	require.NotNil(t, pubAt)
	assert.Equal(t, publishedAt, *pubAt)
}

func TestE2E_ReleaseRepo_ListBySpace_PaginationAcrossPages(t *testing.T) {
	resetReleaseTable(t)
	repo := release.NewReleaseRepo(testGormDB)
	ctx := context.Background()

	// Insert 7 releases
	for i := 0; i < 7; i++ {
		r := newE2ERelease(100, fmt.Sprintf("v0.0.%d", i+1), release.StatusDraft)
		r.CreatedAt = int64(i)
		require.NoError(t, repo.Create(ctx, r))
	}

	page1, total, err := repo.ListBySpace(ctx, 100, "", 3, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 7, total)
	assert.Len(t, page1, 3)

	page2, _, err := repo.ListBySpace(ctx, 100, "", 3, 3)
	require.NoError(t, err)
	assert.Len(t, page2, 3)

	page3, _, err := repo.ListBySpace(ctx, 100, "", 3, 6)
	require.NoError(t, err)
	assert.Len(t, page3, 1)

	// Pages should not overlap
	seen := map[string]bool{}
	for _, r := range page1 {
		seen[r.Version] = true
	}
	for _, r := range page2 {
		assert.False(t, seen[r.Version], "page2 should not include page1 versions")
		seen[r.Version] = true
	}
	for _, r := range page3 {
		assert.False(t, seen[r.Version], "page3 should not overlap")
	}
}

// TestE2E_ReleaseRepo_VersionConflictDetection simulates the version-conflict path:
// the system should detect that a release_version is already used.
func TestE2E_ReleaseRepo_VersionConflictDetection(t *testing.T) {
	resetReleaseTable(t)
	repo := release.NewReleaseRepo(testGormDB)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newE2ERelease(100, "v1.0.0", release.StatusPublished)))

	exists, err := repo.VersionExists(ctx, 100, "v1.0.0")
	require.NoError(t, err)
	assert.True(t, exists, "VersionExists should detect conflict for a re-used version")

	exists2, err := repo.VersionExists(ctx, 100, "v1.0.1")
	require.NoError(t, err)
	assert.False(t, exists2, "fresh version should not conflict")
}
