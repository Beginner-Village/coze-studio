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
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// ─────────────────────────────────────────────────────────────────────────────
// Batch 2: concurrency + runtime stress
// ─────────────────────────────────────────────────────────────────────────────

// concurrentUserSVC is a goroutine-safe variant of fakeUserSVC. It only
// returns a pre-set space and never mutates shared state, so N callers can
// hit it under -race.
type concurrentUserSVC struct {
	usersvc.User
	space *userentity.Space
}

func (f *concurrentUserSVC) GetSpaceByID(_ context.Context, _ int64) (*userentity.Space, error) {
	return f.space, nil
}

// concurrentSearchSVC is a goroutine-safe variant of fakeSearchSVC. It uses
// atomics for counters so the -race detector stays clean under N concurrent
// callers. Each ResyncSpace call returns a FRESH *ResyncESCounts copy so
// downstream mutations (e.g. counts.SliceReindexJobs = queued in the app
// service) don't race across goroutines — production behavior already does
// this because the search domain allocates a new struct per call.
type concurrentSearchSVC struct {
	searchsvc.Search
	counts    *spacemodel.ResyncESCounts
	resyncErr error
	calls     int64 // atomic
	// onCall, if non-nil, is invoked synchronously per call. Useful to
	// inject delays / ctx checks in flight.
	onCall func(ctx context.Context) error
}

func (f *concurrentSearchSVC) ResyncSpace(ctx context.Context, _ int64) (*spacemodel.ResyncESCounts, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.onCall != nil {
		if err := f.onCall(ctx); err != nil {
			return nil, err
		}
	}
	if f.resyncErr != nil {
		return nil, f.resyncErr
	}
	if f.counts == nil {
		return nil, nil
	}
	// Deep-copy: the app service mutates the returned counts (sets
	// SliceReindexJobs), so concurrent callers must not share storage.
	cp := *f.counts
	return &cp, nil
}

func (f *concurrentSearchSVC) SetResyncDeps(_ searchsvc.AgentLister, _ searchsvc.AppLister, _ searchsvc.KbLister) {
}

func (f *concurrentSearchSVC) SearchProjects(_ context.Context, _ *searchEntity.SearchProjectsRequest) (*searchEntity.SearchProjectsResponse, error) {
	return nil, nil
}

func (f *concurrentSearchSVC) SearchResources(_ context.Context, _ *searchEntity.SearchResourcesRequest) (*searchEntity.SearchResourcesResponse, error) {
	return nil, nil
}

// concurrentKnowledgeSVC is a goroutine-safe variant of fakeKnowledgeSVC.
type concurrentKnowledgeSVC struct {
	knowledgesvc.Knowledge
	queued int
	err    error
	calls  int64 // atomic
	onCall func(ctx context.Context) error
}

func (f *concurrentKnowledgeSVC) ResyncSpaceSlices(ctx context.Context, _ int64) (int, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.onCall != nil {
		if err := f.onCall(ctx); err != nil {
			return 0, err
		}
	}
	if f.err != nil {
		return f.queued, f.err
	}
	return f.queued, nil
}

// TestResyncES_ConcurrentSameSpace_NoPanic fires 5 goroutines at the same
// space simultaneously. The current impl has no in-flight de-dup lock (delete
// + write is idempotent — repeating it is fine), so all 5 callers should
// succeed. The test asserts: no panic / deadlock / data race / partial state.
//
// Run with `go test -race -run TestResyncES_Concurrent` to validate the race
// detector stays clean — that's the load-bearing part of this case.
func TestResyncES_ConcurrentSameSpace_NoPanic(t *testing.T) {
	const callers = 5
	const spaceID int64 = 100
	const ownerID int64 = 42

	user := &concurrentUserSVC{space: &userentity.Space{ID: spaceID, OwnerID: ownerID}}
	search := &concurrentSearchSVC{counts: &spacemodel.ResyncESCounts{ProjectDraft: 1, CozeResource: 2, KbEntries: 3}}
	kb := &concurrentKnowledgeSVC{queued: 7}
	svc := &SpaceResyncService{userSVC: user, searchSVC: search, knowledgeSVC: kb}

	// Build one shared base context (so they all see the same user session)
	// and gate all goroutines at the same starting line.
	base := ctxWithUser(ownerID)
	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		errs    = make([]error, callers)
		resps   = make([]*spacemodel.ResyncESResponse, callers)
		panics  = make([]any, callers)
		errsMu  sync.Mutex
		respsMu sync.Mutex
	)

	for i := 0; i < callers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errsMu.Lock()
					panics[i] = r
					errsMu.Unlock()
				}
			}()
			<-start
			resp, err := svc.ResyncES(base, &spacemodel.ResyncESRequest{SpaceID: spaceID})
			respsMu.Lock()
			errs[i], resps[i] = err, resp
			respsMu.Unlock()
		}()
	}

	close(start)
	wg.Wait()

	// 1. nobody panicked
	for i, p := range panics {
		if p != nil {
			t.Fatalf("caller #%d panicked: %v", i, p)
		}
	}

	// 2. all callers got a consistent result. Without de-dup all should
	// succeed (idempotent delete-then-write). If a future impl adds a
	// per-space lock, this assertion should be relaxed to allow some callers
	// returning a typed "sync already in progress" status — but they should
	// NEVER return a half-broken response or a random error.
	for i, err := range errs {
		require.NoErrorf(t, err, "caller #%d failed: %v", i, err)
		require.NotNilf(t, resps[i], "caller #%d got nil response", i)
		assert.Equal(t, int64(0), resps[i].Code, "caller #%d wrong code", i)
		require.NotNilf(t, resps[i].Counts, "caller #%d nil counts", i)
		// Counts come from the shared fake so they should match every time.
		assert.Equal(t, 1, resps[i].Counts.ProjectDraft, "caller #%d ProjectDraft", i)
		assert.Equal(t, 2, resps[i].Counts.CozeResource, "caller #%d CozeResource", i)
		assert.Equal(t, 3, resps[i].Counts.KbEntries, "caller #%d KbEntries", i)
		assert.Equal(t, 7, resps[i].Counts.SliceReindexJobs, "caller #%d SliceReindexJobs", i)
	}

	// 3. each step ran exactly once per caller.
	assert.Equal(t, int64(callers), atomic.LoadInt64(&search.calls), "search.ResyncSpace call count")
	assert.Equal(t, int64(callers), atomic.LoadInt64(&kb.calls), "knowledge.ResyncSpaceSlices call count")
}

// TestResyncES_ContextCanceled_PartwayThrough exercises ctx propagation: the
// caller cancels mid-flight, between the search ResyncSpace step and the kb
// ResyncSpaceSlices step. The kb fake observes the cancel and returns
// ctx.Err(); the application service treats kb failures as non-fatal so
// ResyncES returns success but SliceReindexJobs reflects whatever the kb
// fake claims to have queued (0 here).
func TestResyncES_ContextCanceled_PartwayThrough(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	search := &concurrentSearchSVC{counts: &spacemodel.ResyncESCounts{ProjectDraft: 1, CozeResource: 2, KbEntries: 3}}

	// kb fake selects on ctx.Done — simulates a real worker noticing cancel.
	kb := &concurrentKnowledgeSVC{
		queued: 0,
		onCall: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return nil
			}
		},
	}
	svc := &SpaceResyncService{userSVC: user, searchSVC: search, knowledgeSVC: kb}

	ctx, cancel := context.WithCancel(ctxWithUser(42))
	// Cancel only AFTER the search step finishes — onCall in concurrentSearchSVC
	// hook is nil, so it returns immediately. Then kb.onCall sleeps 50ms, and
	// we cancel within 10ms.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	resp, err := svc.ResyncES(ctx, &spacemodel.ResyncESRequest{SpaceID: 100})
	// Per current impl contract: knowledge step failure is logged but does
	// NOT fail the overall call. The user still sees success with whatever
	// counts the search step produced.
	require.NoError(t, err, "ctx cancel during kb step must NOT fail the call (knowledge is best-effort)")
	require.NotNil(t, resp)
	require.NotNil(t, resp.Counts)
	assert.Equal(t, 1, resp.Counts.ProjectDraft, "search counts must survive kb cancel")
	assert.Equal(t, 2, resp.Counts.CozeResource)
	assert.Equal(t, 3, resp.Counts.KbEntries)
	assert.Equal(t, 0, resp.Counts.SliceReindexJobs, "kb returned 0 queued on cancel")
	// kb was invoked + did observe the cancel.
	assert.Equal(t, int64(1), atomic.LoadInt64(&kb.calls), "kb must have been called")
}

// TestResyncES_HandlerTimeout_BoundaryDoc is a documentation test: the Hertz
// handler enforces a default 30s request timeout. If a real space has 1000+
// agents/resources/KBs the synchronous ResyncSpace step could exceed that
// boundary and the user will see a 500 even though the server keeps grinding.
//
// We don't unit-test the live boundary (it's a system-level concern), but we
// DO assert that the application service itself does not introduce its own
// timeout — it should faithfully propagate whatever ctx the handler gives
// it. If this assumption changes, this test will fire and force a re-think.
//
// See spec doc 2026-05-25-es-resync-design.md → "Section: scaling
// considerations" — moving to an async task pattern is the recommended
// follow-up if real-world space sizes start hitting this boundary.
func TestResyncES_HandlerTimeout_BoundaryDoc(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	search := &concurrentSearchSVC{counts: &spacemodel.ResyncESCounts{}}
	kb := &concurrentKnowledgeSVC{queued: 0}
	svc := &SpaceResyncService{userSVC: user, searchSVC: search, knowledgeSVC: kb}

	// Caller-supplied context has a 50ms deadline. If the app service ever
	// adds its own (longer) context.WithTimeout it would mask the caller's
	// deadline and this test would notice — both fakes' onCall would NOT
	// see Done.
	ctx, cancel := context.WithTimeout(ctxWithUser(42), 50*time.Millisecond)
	defer cancel()

	// Have the search fake check the deadline propagation.
	var sawDeadline bool
	search.onCall = func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		sawDeadline = ok
		return nil
	}

	_, err := svc.ResyncES(ctx, &spacemodel.ResyncESRequest{SpaceID: 100})
	require.NoError(t, err)
	assert.True(t, sawDeadline, "caller's ctx deadline must propagate into search.ResyncSpace — do NOT wrap with WithTimeout in the service")
}
