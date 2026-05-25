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

package space

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	knowledgesvc "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/service"
	searchEntity "github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
	searchsvc "github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// --- fakes ---------------------------------------------------------------

type fakeUserSVC struct {
	usersvc.User
	space   *userentity.Space
	getErr  error
	gotCall struct {
		spaceID int64
	}
}

func (f *fakeUserSVC) GetSpaceByID(_ context.Context, spaceID int64) (*userentity.Space, error) {
	f.gotCall.spaceID = spaceID
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.space, nil
}

type fakeSearchSVC struct {
	searchsvc.Search
	counts    *spacemodel.ResyncESCounts
	resyncErr error
	called    bool
}

func (f *fakeSearchSVC) ResyncSpace(_ context.Context, _ int64) (*spacemodel.ResyncESCounts, error) {
	f.called = true
	if f.resyncErr != nil {
		return nil, f.resyncErr
	}
	return f.counts, nil
}

func (f *fakeSearchSVC) SetResyncDeps(_ searchsvc.AgentLister, _ searchsvc.AppLister, _ searchsvc.KbLister) {
}

func (f *fakeSearchSVC) SearchProjects(_ context.Context, _ *searchEntity.SearchProjectsRequest) (*searchEntity.SearchProjectsResponse, error) {
	return nil, nil
}

func (f *fakeSearchSVC) SearchResources(_ context.Context, _ *searchEntity.SearchResourcesRequest) (*searchEntity.SearchResourcesResponse, error) {
	return nil, nil
}

type fakeKnowledgeSVC struct {
	knowledgesvc.Knowledge
	queued int
	err    error
	called bool
}

func (f *fakeKnowledgeSVC) ResyncSpaceSlices(_ context.Context, _ int64) (int, error) {
	f.called = true
	if f.err != nil {
		return f.queued, f.err
	}
	return f.queued, nil
}

// --- helpers -------------------------------------------------------------

func ctxWithUser(userID int64) context.Context {
	ctx := ctxcache.Init(context.Background())
	if userID > 0 {
		ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: userID})
	}
	return ctx
}

func newTestSvc(user *fakeUserSVC, search *fakeSearchSVC, kb *fakeKnowledgeSVC) *SpaceResyncService {
	return &SpaceResyncService{
		userSVC:      user,
		searchSVC:    search,
		knowledgeSVC: kb,
	}
}

// --- tests ---------------------------------------------------------------

func TestResyncES_NotLoggedIn(t *testing.T) {
	svc := newTestSvc(&fakeUserSVC{}, &fakeSearchSVC{}, &fakeKnowledgeSVC{})

	_, err := svc.ResyncES(context.Background(), &spacemodel.ResyncESRequest{SpaceID: 1})
	require.Error(t, err)
	var statusErr errorx.StatusError
	ok := errors.As(err, &statusErr)
	require.True(t, ok, "expected typed status err, got %T: %v", err, err)
	assert.Equal(t, int32(errno.ErrUserSessionInvalidateCode), statusErr.Code())
}

func TestResyncES_SpaceNotFound_NilSpace(t *testing.T) {
	user := &fakeUserSVC{space: nil}
	svc := newTestSvc(user, &fakeSearchSVC{}, &fakeKnowledgeSVC{})

	_, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	ok := errors.As(err, &statusErr)
	require.True(t, ok, "expected typed status err, got %T: %v", err, err)
	assert.Equal(t, int32(errno.ErrSpaceNotFoundCode), statusErr.Code())
	assert.Equal(t, int64(100), user.gotCall.spaceID)
}

func TestResyncES_SpaceNotFound_GetError(t *testing.T) {
	user := &fakeUserSVC{getErr: errors.New("db error")}
	svc := newTestSvc(user, &fakeSearchSVC{}, &fakeKnowledgeSVC{})

	_, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	ok := errors.As(err, &statusErr)
	require.True(t, ok, "expected typed status err, got %T: %v", err, err)
	assert.Equal(t, int32(errno.ErrSpaceNotFoundCode), statusErr.Code())
}

func TestResyncES_PermissionDenied_NotOwner(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 1}}
	search := &fakeSearchSVC{}
	kb := &fakeKnowledgeSVC{}
	svc := newTestSvc(user, search, kb)

	_, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	ok := errors.As(err, &statusErr)
	require.True(t, ok, "expected typed status err, got %T: %v", err, err)
	assert.Equal(t, int32(errno.ErrSpacePermissionCode), statusErr.Code())
	assert.False(t, search.called, "search.ResyncSpace must not be called when permission denied")
	assert.False(t, kb.called, "knowledge.ResyncSpaceSlices must not be called when permission denied")
}

func TestResyncES_HappyPath(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	search := &fakeSearchSVC{counts: &spacemodel.ResyncESCounts{ProjectDraft: 3, CozeResource: 5, KbEntries: 2}}
	kb := &fakeKnowledgeSVC{queued: 17}
	svc := newTestSvc(user, search, kb)

	resp, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(0), resp.Code)
	require.NotNil(t, resp.Counts)
	assert.Equal(t, 3, resp.Counts.ProjectDraft)
	assert.Equal(t, 5, resp.Counts.CozeResource)
	assert.Equal(t, 2, resp.Counts.KbEntries)
	assert.Equal(t, 17, resp.Counts.SliceReindexJobs)
	assert.True(t, search.called)
	assert.True(t, kb.called)
}

func TestResyncES_SearchFails_AbortsCall(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	search := &fakeSearchSVC{resyncErr: errors.New("es down")}
	kb := &fakeKnowledgeSVC{}
	svc := newTestSvc(user, search, kb)

	_, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	ok := errors.As(err, &statusErr)
	require.True(t, ok, "expected typed status err, got %T: %v", err, err)
	assert.Equal(t, int32(errno.ErrSpaceResyncESCode), statusErr.Code())
	assert.False(t, kb.called, "knowledge step must be skipped when search aborts")
}

func TestResyncES_KnowledgeFails_DoesNotAbort(t *testing.T) {
	// Knowledge step is best-effort: failure should be logged but the
	// overall call returns success with whatever counts we already got.
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	search := &fakeSearchSVC{counts: &spacemodel.ResyncESCounts{ProjectDraft: 1, CozeResource: 2, KbEntries: 3}}
	kb := &fakeKnowledgeSVC{queued: 7, err: errors.New("mq down")}
	svc := newTestSvc(user, search, kb)

	resp, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(0), resp.Code)
	assert.Equal(t, 1, resp.Counts.ProjectDraft)
	assert.Equal(t, 2, resp.Counts.CozeResource)
	assert.Equal(t, 3, resp.Counts.KbEntries)
	// The fake returns its `queued` value even on error.
	assert.Equal(t, 7, resp.Counts.SliceReindexJobs)
}

func TestResyncES_NilCounts_StillPopulatesSliceJobs(t *testing.T) {
	// Defensive: if search domain returns nil counts (shouldn't happen
	// in practice, but we guard against it), we still surface slice jobs.
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	search := &fakeSearchSVC{counts: nil}
	kb := &fakeKnowledgeSVC{queued: 9}
	svc := newTestSvc(user, search, kb)

	resp, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.NoError(t, err)
	require.NotNil(t, resp.Counts)
	assert.Equal(t, 9, resp.Counts.SliceReindexJobs)
}

// TestResyncES_PermissionDenied_CrossSpace ensures the error returned to a
// non-owner caller does NOT leak the real owner's ID. The handler maps the
// status err code to a generic "permission denied" message — but the inner
// `errorx.KV("msg", ...)` payload itself must not contain owner identifiers.
func TestResyncES_PermissionDenied_CrossSpace(t *testing.T) {
	const realOwnerID int64 = 999
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: realOwnerID}}
	search := &fakeSearchSVC{}
	kb := &fakeKnowledgeSVC{}
	svc := newTestSvc(user, search, kb)

	_, err := svc.ResyncES(ctxWithUser(111), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr), "expected typed status err, got %T: %v", err, err)
	assert.Equal(t, int32(errno.ErrSpacePermissionCode), statusErr.Code())

	// Drill into the full error chain to confirm no owner-id leakage. The
	// caller user id (111) is OK to surface to itself, but neither the real
	// owner id (999) nor a hint like "owner is …" should appear anywhere.
	msg := err.Error()
	assert.NotContains(t, msg, "999", "error must not leak the owning user id: %s", msg)
	assert.NotContains(t, msg, "owner is", "error must not name the owner: %s", msg)
	// Both search + knowledge work must stay un-invoked.
	assert.False(t, search.called, "search.ResyncSpace must not be called on permission denied")
	assert.False(t, kb.called, "knowledge.ResyncSpaceSlices must not be called on permission denied")
}

// TestResyncES_OwnerOfDifferentSpace_AllowedOnly_OwnSpace verifies that being
// the owner of *some* space doesn't grant cross-space access. Caller=42 owns
// space A (100) but not space B (200).
func TestResyncES_OwnerOfDifferentSpace_AllowedOnly_OwnSpace(t *testing.T) {
	// case 1: caller owns the space → success.
	{
		user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
		search := &fakeSearchSVC{counts: &spacemodel.ResyncESCounts{}}
		kb := &fakeKnowledgeSVC{}
		svc := newTestSvc(user, search, kb)

		_, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 100})
		require.NoError(t, err, "owner of space 100 should be allowed")
		assert.True(t, search.called)
	}
	// case 2: same caller asks for a space they don't own → denied.
	{
		user := &fakeUserSVC{space: &userentity.Space{ID: 200, OwnerID: 99}}
		search := &fakeSearchSVC{}
		kb := &fakeKnowledgeSVC{}
		svc := newTestSvc(user, search, kb)

		_, err := svc.ResyncES(ctxWithUser(42), &spacemodel.ResyncESRequest{SpaceID: 200})
		require.Error(t, err)
		var statusErr errorx.StatusError
		require.True(t, errors.As(err, &statusErr))
		assert.Equal(t, int32(errno.ErrSpacePermissionCode), statusErr.Code())
		assert.False(t, search.called, "search must not be called for a non-owned space")
	}
}

// TestResyncES_OwnerID_Zero_StillRejects is a defense-in-depth check. If the
// DB ever has space.OwnerID=0 (corrupt row) AND somehow user_id=0 sneaks past
// the session check (it shouldn't — TestResyncES_NotLoggedIn covers that),
// the permission gate would still wrongly equate them. We assert the session
// guard *catches it first* so a 0==0 owner match never executes.
func TestResyncES_OwnerID_Zero_StillRejects(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 0}}
	search := &fakeSearchSVC{}
	kb := &fakeKnowledgeSVC{}
	svc := newTestSvc(user, search, kb)

	// ctxWithUser(0) is intentionally an empty session.
	_, err := svc.ResyncES(ctxWithUser(0), &spacemodel.ResyncESRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr), "expected typed status err, got %T: %v", err, err)
	// The session layer must reject first — NOT a permission-code match.
	assert.Equal(t, int32(errno.ErrUserSessionInvalidateCode), statusErr.Code(),
		"empty session must be rejected by the session guard, not silently allowed via 0==0 owner match")
	assert.False(t, search.called, "search must not be called when session check rejects")
}
