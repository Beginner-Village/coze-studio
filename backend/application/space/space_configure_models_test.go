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
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// --- fake cache.Cmdable ---------------------------------------------------
// The application service only calls .Del(). We don't need a full Redis fake;
// a tiny stub that records call args and returns canned (n, err) is enough.

type fakeIntCmd struct {
	n   int64
	err error
}

func (c *fakeIntCmd) Err() error            { return c.err }
func (c *fakeIntCmd) Result() (int64, error) { return c.n, c.err }

type fakeRedis struct {
	cache.Cmdable
	mu       sync.Mutex
	delKeys  []string
	delErr   error            // global error to inject for every Del call
	delPerN  map[string]int64 // per-key deleted count, defaults to 1 if not set
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{delPerN: map[string]int64{}}
}

func (r *fakeRedis) Del(_ context.Context, keys ...string) cache.IntCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.delErr != nil {
		return &fakeIntCmd{n: 0, err: r.delErr}
	}
	var n int64
	for _, k := range keys {
		r.delKeys = append(r.delKeys, k)
		if v, ok := r.delPerN[k]; ok {
			n += v
		} else {
			n++ // assume the key existed
		}
	}
	return &fakeIntCmd{n: n}
}

func (r *fakeRedis) deletedKeys() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.delKeys))
	copy(out, r.delKeys)
	return out
}

// --- helpers --------------------------------------------------------------

func newConfigureSvc(t *testing.T, user *fakeUserSVC, redis cache.Cmdable) (*SpaceConfigureModelsService, sqlmock.Sqlmock, func()) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      rawDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	svc := &SpaceConfigureModelsService{
		userSVC: user,
		db:      gormDB,
		redis:   redis,
	}
	cleanup := func() { _ = rawDB.Close() }
	return svc, mock, cleanup
}

func expectCacheLookup(mock sqlmock.Sqlmock, spaceID int64, modelEntityIDs ...uint64) {
	rows := sqlmock.NewRows([]string{"id"})
	for _, id := range modelEntityIDs {
		rows.AddRow(id)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM model_entity")).
		WithArgs(spaceID).
		WillReturnRows(rows)
}

// --- tests ----------------------------------------------------------------

func TestConfigureModels_NotLoggedIn(t *testing.T) {
	svc, mock, cleanup := newConfigureSvc(t, &fakeUserSVC{}, newFakeRedis())
	defer cleanup()

	_, err := svc.ConfigureModels(context.Background(), &spacemodel.ConfigureModelsRequest{SpaceID: 1})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrUserSessionInvalidateCode), statusErr.Code())
	assert.NoError(t, mock.ExpectationsWereMet(), "no SQL should have been issued")
}

func TestConfigureModels_PermissionDenied_NotOwner(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 1}}
	svc, mock, cleanup := newConfigureSvc(t, user, newFakeRedis())
	defer cleanup()

	_, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID: 100,
		Chat:    &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
	})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpacePermissionCode), statusErr.Code())
	assert.NoError(t, mock.ExpectationsWereMet(), "no SQL should have been issued for non-owner")
}

func TestConfigureModels_SpaceNotFound_NilSpace(t *testing.T) {
	user := &fakeUserSVC{space: nil}
	svc, _, cleanup := newConfigureSvc(t, user, newFakeRedis())
	defer cleanup()

	_, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpaceNotFoundCode), statusErr.Code())
}

func TestConfigureModels_SpaceNotFound_GetError(t *testing.T) {
	user := &fakeUserSVC{getErr: errors.New("db error")}
	svc, _, cleanup := newConfigureSvc(t, user, newFakeRedis())
	defer cleanup()

	_, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpaceNotFoundCode), statusErr.Code())
}

// Happy path with all 3 sections present + 2 cache keys to clean.
func TestConfigureModels_HappyPath_AllSections(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ? WHERE deleted_at IS NULL")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_embedding SET config = ? WHERE space_id = ? AND deleted_at IS NULL")).
		WithArgs(sqlmock.AnyArg(), spaceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_rerank SET config = ? WHERE space_id = ? AND deleted_at IS NULL")).
		WithArgs(sqlmock.AnyArg(), spaceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID, 10, 20)

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID:  spaceID,
		Chat:     &spacemodel.ChatModelConfig{BaseURL: "http://llm/v1", APIKey: "secret", Model: "qwen"},
		Embedder: &spacemodel.EmbedderModelConfig{BaseURL: "http://emb", APIKey: "ek", Model: "bge", Dims: 1024},
		Rerank:   &spacemodel.RerankModelConfig{BaseURL: "http://re", APIKey: "rk", Model: "rerank-large"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(0), resp.Code)
	require.NotNil(t, resp.Data)
	assert.Equal(t, int64(3), resp.Data.ModelMetaUpdated)
	assert.Equal(t, int64(1), resp.Data.SpaceEmbeddingUpdated)
	assert.Equal(t, int64(1), resp.Data.SpaceRerankUpdated)
	assert.Equal(t, int64(2), resp.Data.RedisKeysDeleted)
	assert.Empty(t, resp.Data.Warnings, "no warnings on happy path")

	assert.NoError(t, mock.ExpectationsWereMet())
	assert.ElementsMatch(t,
		[]string{"space:100:model:10", "space:100:model:20"},
		redis.deletedKeys(),
	)
}

// Partial update: only chat is supplied. The other two UPDATEs must NOT run.
func TestConfigureModels_OnlyChat(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID, 10)

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID: spaceID,
		Chat:    &spacemodel.ChatModelConfig{BaseURL: "http://llm", APIKey: "k", Model: "qwen"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), resp.Data.ModelMetaUpdated)
	assert.Equal(t, int64(0), resp.Data.SpaceEmbeddingUpdated)
	assert.Equal(t, int64(0), resp.Data.SpaceRerankUpdated)
	assert.Equal(t, int64(1), resp.Data.RedisKeysDeleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Partial update: only embedder.
func TestConfigureModels_OnlyEmbedder(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_embedding SET config = ?")).
		WithArgs(sqlmock.AnyArg(), spaceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID)

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID:  spaceID,
		Embedder: &spacemodel.EmbedderModelConfig{BaseURL: "http://emb", APIKey: "k", Model: "bge", Dims: 1024},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Data.ModelMetaUpdated)
	assert.Equal(t, int64(1), resp.Data.SpaceEmbeddingUpdated)
	assert.Equal(t, int64(0), resp.Data.SpaceRerankUpdated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Partial update: only rerank.
func TestConfigureModels_OnlyRerank(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_rerank SET config = ?")).
		WithArgs(sqlmock.AnyArg(), spaceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID)

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID: spaceID,
		Rerank:  &spacemodel.RerankModelConfig{BaseURL: "http://re", APIKey: "k", Model: "rerank"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Data.SpaceEmbeddingUpdated)
	assert.Equal(t, int64(1), resp.Data.SpaceRerankUpdated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// All-nil request: still ran permission check + cache cleanup, no SQL UPDATEs.
func TestConfigureModels_NoSections_StillClearsCache(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	// No begin/commit because the txn body is empty — but gorm still opens
	// the transaction. Actually GORM's Transaction() opens BEGIN even if the
	// closure is a no-op. So we still need to mock it.
	mock.ExpectBegin()
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID, 7)

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{SpaceID: spaceID})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Data.ModelMetaUpdated)
	assert.Equal(t, int64(1), resp.Data.RedisKeysDeleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// DB write fails → entire txn rolls back AND cache is NOT touched.
func TestConfigureModels_DBError_RollsBack_NoCacheTouch(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	// model_meta succeeds, then space_embedding fails — rollback expected.
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_embedding SET config = ?")).
		WithArgs(sqlmock.AnyArg(), spaceID).
		WillReturnError(errors.New("disk full"))
	mock.ExpectRollback()
	// NOTE: no cache lookup expected — invalidateSpaceModelCache must not run.

	_, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID:  spaceID,
		Chat:     &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
		Embedder: &spacemodel.EmbedderModelConfig{BaseURL: "u", APIKey: "k", Model: "m", Dims: 8},
	})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpaceConfigureModelsCode), statusErr.Code())
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Empty(t, redis.deletedKeys(), "cache must NOT be touched when DB rolls back")
}

// Redis nil-safe: when the service was constructed without a Redis client we
// still complete the DB write and surface a warning.
func TestConfigureModels_RedisNil_WarnsButSucceeds(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	svc, mock, cleanup := newConfigureSvc(t, user, nil) // no redis
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID: spaceID,
		Chat:    &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Data.ModelMetaUpdated)
	assert.Equal(t, int64(0), resp.Data.RedisKeysDeleted)
	require.Len(t, resp.Data.Warnings, 1)
	assert.Contains(t, resp.Data.Warnings[0], "redis not configured")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Redis DEL fails mid-stream → cache cleanup partial, surfaced as warning,
// but DB write is NOT rolled back.
func TestConfigureModels_RedisDelFails_WarnsButSucceeds(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	redis.delErr = errors.New("redis down")
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID, 99)

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID: spaceID,
		Chat:    &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	assert.Equal(t, int64(1), resp.Data.ModelMetaUpdated, "DB write must NOT be rolled back when cache cleanup fails")
	assert.Equal(t, int64(0), resp.Data.RedisKeysDeleted)
	require.Len(t, resp.Data.Warnings, 1)
	assert.Contains(t, resp.Data.Warnings[0], "redis down")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// invalidateSpaceModelCache works when the union query returns zero rows
// (e.g. brand-new space, no models attached). No warning, no DEL.
func TestConfigureModels_EmptyModelSet_NoCacheKeys(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID) // zero rows

	resp, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID: spaceID,
		Chat:    &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Data.RedisKeysDeleted)
	assert.Empty(t, resp.Data.Warnings)
	assert.Empty(t, redis.deletedKeys())
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Concurrency: two callers on the same space — both succeed (idempotent),
// each runs its own DB writes and cache cleanup. Demonstrates -race safety.
func TestConfigureModels_Concurrent_SameSpace(t *testing.T) {
	const spaceID int64 = 100
	const callers = 3

	user := &concurrentUserSVC{space: &userentity.Space{ID: spaceID, OwnerID: 42}}
	redis := newFakeRedis()

	// Each caller gets its own sqlmock — sqlmock is single-threaded so sharing
	// one across goroutines is a recipe for flake. Build N services that share
	// the user fake + redis fake only.
	type plan struct {
		svc     *SpaceConfigureModelsService
		mock    sqlmock.Sqlmock
		cleanup func()
	}
	plans := make([]plan, callers)
	for i := 0; i < callers; i++ {
		rawDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		gormDB, err := gorm.Open(mysql.New(mysql.Config{
			Conn:                      rawDB,
			SkipInitializeWithVersion: true,
		}), &gorm.Config{})
		require.NoError(t, err)
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
			WithArgs(sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		expectCacheLookup(mock, spaceID, uint64(100+i))
		plans[i] = plan{
			svc: &SpaceConfigureModelsService{
				userSVC: user,
				db:      gormDB,
				redis:   redis,
			},
			mock:    mock,
			cleanup: func() { _ = rawDB.Close() },
		}
	}
	defer func() {
		for _, p := range plans {
			p.cleanup()
		}
	}()

	var (
		wg    sync.WaitGroup
		start = make(chan struct{})
		errs  = make([]error, callers)
	)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			<-start
			_, err := plans[i].svc.ConfigureModels(
				ctxWithUser(42),
				&spacemodel.ConfigureModelsRequest{
					SpaceID: spaceID,
					Chat:    &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
				},
			)
			errs[i] = err
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		assert.NoErrorf(t, err, "caller %d failed", i)
	}
	for i, p := range plans {
		assert.NoErrorf(t, p.mock.ExpectationsWereMet(), "sqlmock expectations not met for caller %d", i)
	}
	// Every caller deleted exactly 1 key — total 3 across the shared redis.
	assert.Len(t, redis.deletedKeys(), callers)
}

// ctx-cancel propagation: caller cancels before the txn runs. GORM's Tx
// receives the cancelled ctx and returns context.Canceled, which we wrap.
func TestConfigureModels_CtxCanceled_NoSQLExecuted(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	redis := newFakeRedis()
	svc, mock, cleanup := newConfigureSvc(t, user, redis)
	defer cleanup()

	ctx, cancel := context.WithCancel(ctxWithUser(42))
	cancel() // pre-cancel

	// GORM short-circuits on a cancelled ctx before reaching the driver,
	// so we do NOT set ExpectBegin — the driver is never called.

	_, err := svc.ConfigureModels(ctx, &spacemodel.ConfigureModelsRequest{
		SpaceID: 100,
		Chat:    &spacemodel.ChatModelConfig{BaseURL: "u", APIKey: "k", Model: "m"},
	})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpaceConfigureModelsCode), statusErr.Code())
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Empty(t, redis.deletedKeys(), "cache must not be touched after a failed DB phase")
}

// Sanity check: the JSON we marshal into the SQL bind matches the shape the
// downstream code (chatmodel.Config / EmbeddingConfig / RerankConfig) expects.
// We capture the literal SQL arg and re-parse it.
func TestConfigureModels_MarshalShape_MatchesDownstream(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 42}}
	svc, mock, cleanup := newConfigureSvc(t, user, newFakeRedis())
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE model_meta SET conn_config = ?")).
		WithArgs(sqlmockMatcher{check: func(v any) bool {
			s, ok := v.(string)
			if !ok {
				return false
			}
			return strings.Contains(s, `"base_url":"http://llm"`) &&
				strings.Contains(s, `"api_key":"sk"`) &&
				strings.Contains(s, `"model":"qwen"`) &&
				strings.Contains(s, `"enable_thinking":false`)
		}}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_embedding SET config = ?")).
		WithArgs(sqlmockMatcher{check: func(v any) bool {
			s, ok := v.(string)
			if !ok {
				return false
			}
			return strings.Contains(s, `"type":"openai"`) &&
				strings.Contains(s, `"openai_config":`) &&
				strings.Contains(s, `"dims":1024`) &&
				strings.Contains(s, `"max_batch_size":100`)
		}}, spaceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE space_rerank SET config = ?")).
		WithArgs(sqlmockMatcher{check: func(v any) bool {
			s, ok := v.(string)
			if !ok {
				return false
			}
			return strings.Contains(s, `"type":"openai"`) &&
				strings.Contains(s, `"model":"rerank-large"`) &&
				!strings.Contains(s, `"dims":`)
		}}, spaceID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectCacheLookup(mock, spaceID)

	_, err := svc.ConfigureModels(ctxWithUser(42), &spacemodel.ConfigureModelsRequest{
		SpaceID:  spaceID,
		Chat:     &spacemodel.ChatModelConfig{BaseURL: "http://llm", APIKey: "sk", Model: "qwen"},
		Embedder: &spacemodel.EmbedderModelConfig{BaseURL: "http://emb", APIKey: "ek", Model: "bge", Dims: 1024},
		Rerank:   &spacemodel.RerankModelConfig{BaseURL: "http://re", APIKey: "rk", Model: "rerank-large"},
	})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// sqlmockMatcher is a tiny custom Argument matcher — sqlmock ships sqlmock.AnyArg
// but we need to assert specific JSON shape, not just type identity. It must
// implement sqlmock.Argument exactly: Match(driver.Value) bool.
type sqlmockMatcher struct {
	check func(any) bool
}

// Match satisfies the sqlmock.Argument interface.
func (m sqlmockMatcher) Match(v driver.Value) bool {
	return m.check(v)
}

// Make sure we link in time so unused-import doesn't bite (used by ctxWithUser hop).
var _ = time.Second
