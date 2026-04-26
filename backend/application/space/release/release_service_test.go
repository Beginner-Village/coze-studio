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

package release

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newReleaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&SpaceRelease{}))
	return db
}

func mustJSONBytes(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func newRelease(spaceID int64, version, status string) *SpaceRelease {
	return &SpaceRelease{
		SpaceID:     spaceID,
		Version:     version,
		Status:      status,
		SyncType:    "full",
		Manifest:    json.RawMessage(`{}`),
		Statistics:  json.RawMessage(`{}`),
		PackageKey:  "p/" + version,
		PackageSize: 100,
		ContentHash: "deadbeef",
		CreatedAt:   1234567890,
		UpdatedAt:   1234567890,
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input                       string
		major, minor, patch int
		ok                          bool
	}{
		{"v1.0.0", 1, 0, 0, true},
		{"v2.3.4", 2, 3, 4, true},
		{"v0.0.1", 0, 0, 1, true},
		{"v10.20.30", 10, 20, 30, true},
		{"1.0.0", 0, 0, 0, false},      // missing v prefix
		{"v1.0", 0, 0, 0, false},        // missing patch
		{"v1.0.0-beta", 0, 0, 0, false}, // pre-release not supported
		{"", 0, 0, 0, false},
		{"vx.y.z", 0, 0, 0, false},
	}

	for _, tt := range tests {
		major, minor, patch, ok := parseSemver(tt.input)
		if ok != tt.ok {
			t.Errorf("parseSemver(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			continue
		}
		if ok {
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("parseSemver(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		}
	}
}

func TestDiffIDList(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	fromIDs := []int64{1, 2, 3}
	toIDs := []int64{2, 3, 4}
	fromHashes := map[string]string{
		"agent:1": "hash_a1",
		"agent:2": "hash_a2",
		"agent:3": "hash_a3",
	}
	toHashes := map[string]string{
		"agent:2": "hash_a2",     // same
		"agent:3": "hash_a3_new", // changed
		"agent:4": "hash_a4",
	}

	diffIDList(diff, "agents", fromIDs, toIDs, fromHashes, toHashes, "agent")

	// Added: 4
	if len(diff.Added["agents"]) != 1 || diff.Added["agents"][0].ID != 4 {
		t.Errorf("Added = %v, want [{ID:4}]", diff.Added["agents"])
	}

	// Removed: 1
	if len(diff.Removed["agents"]) != 1 || diff.Removed["agents"][0].ID != 1 {
		t.Errorf("Removed = %v, want [{ID:1}]", diff.Removed["agents"])
	}

	// Modified: 3 (hash changed)
	if len(diff.Modified["agents"]) != 1 || diff.Modified["agents"][0].ID != 3 {
		t.Errorf("Modified = %v, want [{ID:3}]", diff.Modified["agents"])
	}
}

func TestDiffIDListNoHashes(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	diffIDList(diff, "agents", []int64{1, 2}, []int64{2, 3}, nil, nil, "agent")

	if len(diff.Added["agents"]) != 1 || diff.Added["agents"][0].ID != 3 {
		t.Errorf("Added = %v, want [{ID:3}]", diff.Added["agents"])
	}
	if len(diff.Removed["agents"]) != 1 || diff.Removed["agents"][0].ID != 1 {
		t.Errorf("Removed = %v, want [{ID:1}]", diff.Removed["agents"])
	}
	// No hashes → no modified detection
	if len(diff.Modified["agents"]) != 0 {
		t.Errorf("Modified = %v, want empty (no hashes)", diff.Modified["agents"])
	}
}

func TestDiffIDListEmpty(t *testing.T) {
	diff := &VersionDiff{
		Added:    make(map[string][]ResourceSummary),
		Modified: make(map[string][]ResourceSummary),
		Removed:  make(map[string][]ResourceSummary),
	}

	diffIDList(diff, "agents", []int64{}, []int64{}, nil, nil, "agent")

	if len(diff.Added["agents"]) != 0 || len(diff.Removed["agents"]) != 0 || len(diff.Modified["agents"]) != 0 {
		t.Error("expected empty diff for empty inputs")
	}
}

// ---- New tests below: ReleaseRepo CRUD and ReleaseService logic ----

func TestReleaseRepo_Create_AndGetByVersion(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	rec := newRelease(100, "v1.0.0", StatusDraft)
	require.NoError(t, repo.Create(ctx, rec))
	require.NotZero(t, rec.ID)

	got, err := repo.GetByVersion(ctx, 100, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, rec.ID, got.ID)
	assert.Equal(t, "v1.0.0", got.Version)
}

func TestReleaseRepo_GetByVersion_NotFound(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	got, err := repo.GetByVersion(context.Background(), 100, "v9.9.9")
	require.NoError(t, err)
	assert.Nil(t, got, "missing version should return nil, nil")
}

func TestReleaseRepo_GetLatest(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	r1 := newRelease(100, "v1.0.0", StatusDraft)
	r1.CreatedAt = 100
	r2 := newRelease(100, "v1.1.0", StatusDraft)
	r2.CreatedAt = 200
	require.NoError(t, repo.Create(ctx, r1))
	require.NoError(t, repo.Create(ctx, r2))

	got, err := repo.GetLatest(ctx, 100)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v1.1.0", got.Version)
}

func TestReleaseRepo_GetLatest_Empty(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	got, err := repo.GetLatest(context.Background(), 999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestReleaseRepo_GetLatestPublished(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	r1 := newRelease(100, "v1.0.0", StatusPublished)
	r1.CreatedAt = 100
	r2 := newRelease(100, "v1.1.0", StatusDraft)
	r2.CreatedAt = 200
	r3 := newRelease(100, "v1.2.0", StatusPublished)
	r3.CreatedAt = 300
	require.NoError(t, repo.Create(ctx, r1))
	require.NoError(t, repo.Create(ctx, r2))
	require.NoError(t, repo.Create(ctx, r3))

	got, err := repo.GetLatestPublished(ctx, 100)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v1.2.0", got.Version, "should return the latest *published* one (v1.1.0 is draft)")
}

func TestReleaseRepo_VersionExists(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	exists, err := repo.VersionExists(ctx, 100, "v1.0.0")
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, repo.Create(ctx, newRelease(100, "v1.0.0", StatusDraft)))
	exists, err = repo.VersionExists(ctx, 100, "v1.0.0")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestReleaseRepo_UpdateStatus(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	r := newRelease(100, "v1.0.0", StatusDraft)
	require.NoError(t, repo.Create(ctx, r))

	publishedAt := int64(99999)
	require.NoError(t, repo.UpdateStatus(ctx, r.ID, StatusPublished, &publishedAt))

	got, err := repo.GetByVersion(ctx, 100, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, StatusPublished, got.Status)
	require.NotNil(t, got.PublishedAt)
	assert.Equal(t, publishedAt, *got.PublishedAt)
}

func TestReleaseRepo_ListBySpace_FilterByStatus(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	for i, version := range []string{"v1.0.0", "v1.1.0", "v1.2.0"} {
		r := newRelease(100, version, StatusDraft)
		if i == 1 {
			r.Status = StatusPublished
		}
		r.CreatedAt = int64(i)
		require.NoError(t, repo.Create(ctx, r))
	}
	// Other space, should be ignored
	require.NoError(t, repo.Create(ctx, newRelease(200, "v1.0.0", StatusDraft)))

	all, total, err := repo.ListBySpace(ctx, 100, "", 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, all, 3)

	published, total2, err := repo.ListBySpace(ctx, 100, StatusPublished, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total2)
	require.Len(t, published, 1)
	assert.Equal(t, "v1.1.0", published[0].Version)
}

func TestReleaseRepo_ListBySpace_Pagination(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		r := newRelease(100, fmt.Sprintf("v0.0.%d", i+1), StatusDraft)
		r.CreatedAt = int64(i)
		require.NoError(t, repo.Create(ctx, r))
	}

	page1, total, err := repo.ListBySpace(ctx, 100, "", 2, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 5, total)
	assert.Len(t, page1, 2)

	page2, _, err := repo.ListBySpace(ctx, 100, "", 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Pages should not overlap
	assert.NotEqual(t, page1[0].Version, page2[0].Version)
}

func TestReleaseService_NextVersion_FirstRelease(t *testing.T) {
	db := newReleaseTestDB(t)
	svc := &ReleaseService{repo: NewReleaseRepo(db)}

	got, err := svc.NextVersion(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", got)
}

func TestReleaseService_NextVersion_BumpsPatch(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRelease(100, "v1.2.3", StatusDraft)))

	svc := &ReleaseService{repo: repo}
	got, err := svc.NextVersion(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.4", got)
}

func TestReleaseService_NextVersion_InvalidLatestFallsBackTov1(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := NewReleaseRepo(db)
	ctx := context.Background()

	// Latest version doesn't match semver pattern → service should fall back to v1.0.0
	require.NoError(t, repo.Create(ctx, newRelease(100, "garbage", StatusDraft)))

	svc := &ReleaseService{repo: repo}
	got, err := svc.NextVersion(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", got)
}

func TestReleaseService_GetRepo(t *testing.T) {
	db := newReleaseTestDB(t)
	svc := NewReleaseService(db, nil, nil)
	assert.NotNil(t, svc.GetRepo())
}

func TestParseSemver_AdditionalEdgeCases(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"v1.0.0 ", false},  // trailing space
		{" v1.0.0", false},  // leading space
		{"V1.0.0", false},   // uppercase V
		{"v1.0.0.4", false}, // 4-part
	}
	for _, c := range cases {
		_, _, _, ok := parseSemver(c.in)
		assert.Equal(t, c.ok, ok, "input: %q", c.in)
	}
}

func TestDiffIDList_AddedAndRemovedDisjoint(t *testing.T) {
	diff := &VersionDiff{
		Added:    map[string][]ResourceSummary{},
		Modified: map[string][]ResourceSummary{},
		Removed:  map[string][]ResourceSummary{},
	}
	diffIDList(diff, "agents", []int64{1, 2}, []int64{3, 4}, nil, nil, "agent")
	assert.Len(t, diff.Added["agents"], 2)
	assert.Len(t, diff.Removed["agents"], 2)
	assert.Empty(t, diff.Modified["agents"])
}

func TestDiffIDList_AllUnchanged(t *testing.T) {
	diff := &VersionDiff{
		Added:    map[string][]ResourceSummary{},
		Modified: map[string][]ResourceSummary{},
		Removed:  map[string][]ResourceSummary{},
	}
	hashes := map[string]string{"agent:1": "h1", "agent:2": "h2"}
	diffIDList(diff, "agents", []int64{1, 2}, []int64{1, 2}, hashes, hashes, "agent")
	assert.Empty(t, diff.Added["agents"])
	assert.Empty(t, diff.Removed["agents"])
	assert.Empty(t, diff.Modified["agents"], "identical hashes → no modifications")
}

func TestSpaceRelease_TableName(t *testing.T) {
	assert.Equal(t, "space_release", SpaceRelease{}.TableName())
}

// _ is only used to silence unused-import "fmt" if any test paths drop fmt usage.
var _ = mustJSONBytes
