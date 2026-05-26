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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// ----- fakes --------------------------------------------------------------

// fakeESClient implements just enough of es.Client for the diagnose probe.
// Exists()/Search() are dispatched per-index via maps so a single test can
// model a happy index and a missing one simultaneously.
type fakeESClient struct {
	es.Client
	exists      map[string]bool   // name -> exists
	existsErr   map[string]error  // name -> error to return
	counts      map[string]int64  // name -> doc count (used by Search())
	countErr    map[string]error  // name -> Search() error
	existsCalls []string          // recorded for ordering checks
	searchCalls []string          // recorded for ordering checks
	mu          sync.Mutex
}

func newFakeES() *fakeESClient {
	return &fakeESClient{
		exists:    map[string]bool{},
		existsErr: map[string]error{},
		counts:    map[string]int64{},
		countErr:  map[string]error{},
	}
}

func (f *fakeESClient) Exists(_ context.Context, index string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.existsCalls = append(f.existsCalls, index)
	if err, ok := f.existsErr[index]; ok {
		return false, err
	}
	return f.exists[index], nil
}

func (f *fakeESClient) Search(_ context.Context, index string, _ *es.Request) (*es.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searchCalls = append(f.searchCalls, index)
	if err, ok := f.countErr[index]; ok {
		return nil, err
	}
	n := f.counts[index]
	return &es.Response{Hits: es.HitsMetadata{Total: &es.TotalHits{Value: n}}}, nil
}

// fakeSearchStore + fakeVectorManager — the diagnose flow only calls
// Manager.GetType and Manager.GetSearchStore, so we stub the bare minimum.
type fakeSearchStore struct {
	searchstore.SearchStore
}

type fakeVectorManager struct {
	searchstore.Manager
	storeType searchstore.SearchStoreType
	// per-collection error from GetSearchStore (e.g. "collection not found")
	getErr  map[string]error
	gotName []string
	mu      sync.Mutex
}

func newFakeVectorMgr() *fakeVectorManager {
	return &fakeVectorManager{
		storeType: searchstore.TypeVectorStore,
		getErr:    map[string]error{},
	}
}

func (m *fakeVectorManager) GetType() searchstore.SearchStoreType { return m.storeType }
func (m *fakeVectorManager) GetSearchStore(_ context.Context, name string) (searchstore.SearchStore, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gotName = append(m.gotName, name)
	if err, ok := m.getErr[name]; ok {
		return nil, err
	}
	return &fakeSearchStore{}, nil
}

// ----- helpers -----------------------------------------------------------

// newDiagSvc builds a SpaceDiagnoseService backed by go-sqlmock so we can
// assert on raw SQL — mirrors newConfigureSvc.
func newDiagSvc(t *testing.T, user *fakeUserSVC, esCli es.Client, mgrs []searchstore.Manager) (*SpaceDiagnoseService, sqlmock.Sqlmock, func()) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      rawDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	svc := &SpaceDiagnoseService{
		userSVC:    user,
		db:         gormDB,
		esClient:   esCli,
		ssManagers: mgrs,
		httpClient: &http.Client{Timeout: probeTimeout},
	}
	cleanup := func() { _ = rawDB.Close() }
	return svc, mock, cleanup
}

// expectMySQLCounts stubs the 9 fixed COUNT queries to return their input
// counts in fixedTables order.
func expectMySQLCounts(mock sqlmock.Sqlmock, spaceID int64, perTable map[string]int64) {
	for _, t := range fixedTables {
		n, ok := perTable[t.name]
		if !ok {
			n = 0
		}
		rows := sqlmock.NewRows([]string{"count"}).AddRow(n)
		eq := mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM " + t.name))
		if t.spaceScoped {
			eq.WithArgs(spaceID).WillReturnRows(rows)
		} else {
			eq.WillReturnRows(rows)
		}
	}
}

// expectListKBs stubs the (id, name) lookup used by both ES and Milvus
// sections. Use empty kbs to skip per-kb work.
func expectListKBs(mock sqlmock.Sqlmock, spaceID int64, kbs []kbRow) {
	rows := sqlmock.NewRows([]string{"id", "name"})
	for _, k := range kbs {
		rows.AddRow(k.ID, k.Name)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name FROM knowledge")).
		WithArgs(spaceID).
		WillReturnRows(rows)
}

// ─────────────────────────────────────────────────────────────────────────────
// Permission / auth boundaries
// ─────────────────────────────────────────────────────────────────────────────

func TestDiagnose_NotLoggedIn(t *testing.T) {
	svc, mock, cleanup := newDiagSvc(t, &fakeUserSVC{}, newFakeES(), nil)
	defer cleanup()

	_, err := svc.Diagnose(context.Background(), &spacemodel.DiagnoseRequest{SpaceID: 1})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrUserSessionInvalidateCode), statusErr.Code())
	assert.NoError(t, mock.ExpectationsWereMet(), "no SQL should run when session check fails")
}

func TestDiagnose_PermissionDenied_NotOwner(t *testing.T) {
	user := &fakeUserSVC{space: &userentity.Space{ID: 100, OwnerID: 1}}
	svc, mock, cleanup := newDiagSvc(t, user, newFakeES(), nil)
	defer cleanup()

	_, err := svc.Diagnose(ctxWithUser(42), &spacemodel.DiagnoseRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpacePermissionCode), statusErr.Code())
	assert.NoError(t, mock.ExpectationsWereMet(), "no SQL should run for non-owner")
}

func TestDiagnose_SpaceNotFound_NilSpace(t *testing.T) {
	svc, _, cleanup := newDiagSvc(t, &fakeUserSVC{space: nil}, newFakeES(), nil)
	defer cleanup()

	_, err := svc.Diagnose(ctxWithUser(42), &spacemodel.DiagnoseRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpaceNotFoundCode), statusErr.Code())
}

func TestDiagnose_SpaceNotFound_GetError(t *testing.T) {
	svc, _, cleanup := newDiagSvc(t, &fakeUserSVC{getErr: errors.New("db error")}, newFakeES(), nil)
	defer cleanup()

	_, err := svc.Diagnose(ctxWithUser(42), &spacemodel.DiagnoseRequest{SpaceID: 100})
	require.Error(t, err)
	var statusErr errorx.StatusError
	require.True(t, errors.As(err, &statusErr))
	assert.Equal(t, int32(errno.ErrSpaceNotFoundCode), statusErr.Code())
}

// ─────────────────────────────────────────────────────────────────────────────
// Probe helpers — direct calls to probeChat / probeEmbedder / probeRerank
// ─────────────────────────────────────────────────────────────────────────────

// stubHTTP spins up a httptest.NewServer that returns a canned (status, body)
// on every request and records the requests it saw. Use this to drive a
// probe without depending on the real network.
type stubHTTP struct {
	status  int
	body    string
	delay   time.Duration
	reqs    []*http.Request
	mu      sync.Mutex
	srv     *httptest.Server
}

func newStubHTTP(t *testing.T, status int, body string) *stubHTTP {
	t.Helper()
	s := &stubHTTP{status: status, body: body}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.reqs = append(s.reqs, r)
		s.mu.Unlock()
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
		w.WriteHeader(s.status)
		_, _ = w.Write([]byte(s.body))
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func mustNewSqlmock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = rawDB.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      rawDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, mock
}

func TestProbeChat_200OK(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 200, `{"choices":[{"text":"pong"}]}`)
	connCfg, _ := json.Marshal(map[string]any{"base_url": stub.srv.URL, "api_key": "sk", "model": "qwen"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeChat(context.Background())
	assert.True(t, out.Configured)
	assert.True(t, out.Reachable, "200 ⇒ reachable=true; got error=%q", out.Error)
	assert.Equal(t, 200, out.HTTPStatus)
	assert.Equal(t, stub.srv.URL, out.Endpoint)
	assert.Equal(t, "qwen", out.Model)
	assert.GreaterOrEqual(t, out.LatencyMs, int64(0))
	assert.NoError(t, mock.ExpectationsWereMet())

	// Verify the upstream got an OpenAI chat-completions body with the
	// expected fields — we don't want to silently regress to the wrong API.
	stub.mu.Lock()
	defer stub.mu.Unlock()
	require.Len(t, stub.reqs, 1)
	require.Equal(t, "/chat/completions", stub.reqs[0].URL.Path)
	require.Equal(t, "Bearer sk", stub.reqs[0].Header.Get("Authorization"))
}

func TestProbeChat_500_MarkedUnreachable(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 500, `boom`)
	connCfg, _ := json.Marshal(map[string]any{"base_url": stub.srv.URL, "api_key": "sk", "model": "qwen"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeChat(context.Background())
	assert.True(t, out.Configured)
	assert.False(t, out.Reachable)
	assert.Equal(t, 500, out.HTTPStatus)
	assert.Contains(t, out.Error, "non-2xx")
}

func TestProbeChat_ConnectionRefused(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	// 127.0.0.1:1 is reserved + closed on any reasonable test box — connect
	// returns ECONNREFUSED before any HTTP exchange.
	connCfg, _ := json.Marshal(map[string]any{"base_url": "http://127.0.0.1:1/v1", "api_key": "sk", "model": "qwen"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeChat(context.Background())
	assert.True(t, out.Configured)
	assert.False(t, out.Reachable)
	assert.NotEmpty(t, out.Error)
}

func TestProbeChat_Unconfigured_NoRow(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeChat(context.Background())
	assert.False(t, out.Configured)
	assert.False(t, out.Reachable)
	assert.Equal(t, 0, out.HTTPStatus)
	assert.Contains(t, out.Error, "no active model_meta")
}

func TestProbeChat_Unconfigured_BadJSON(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow([]byte(`not-json`)))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeChat(context.Background())
	assert.False(t, out.Configured)
	assert.Contains(t, out.Error, "decode")
}

func TestProbeChat_Unconfigured_MissingBaseURL(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	// Valid JSON but base_url missing — probe must NOT try to POST.
	cc, _ := json.Marshal(map[string]any{"api_key": "sk", "model": "qwen"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(cc))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeChat(context.Background())
	assert.False(t, out.Configured)
	assert.Contains(t, out.Error, "missing")
}

func TestProbeEmbedder_200_WithValidShape(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 200, `{"data":[{"embedding":[0.1,0.2,0.3]}]}`)
	embCfg, _ := json.Marshal(map[string]any{
		"type": "openai",
		"openai_config": map[string]any{
			"base_url": stub.srv.URL,
			"api_key":  "ek",
			"model":    "bge",
		},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(embCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeEmbedder(context.Background(), 100)
	assert.True(t, out.Configured)
	assert.True(t, out.Reachable, "200 + valid shape ⇒ reachable; got error=%q", out.Error)
	assert.Equal(t, 200, out.HTTPStatus)
	assert.Equal(t, "bge", out.Model)
	require.Len(t, stub.reqs, 1)
	require.Equal(t, "/embeddings", stub.reqs[0].URL.Path)
}

// Embedder responding 200 but with an empty body fails the shape check —
// catches the "endpoint up but model misconfigured" failure mode.
func TestProbeEmbedder_200_EmptyData_MarkedUnreachable(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 200, `{"data":[]}`)
	embCfg, _ := json.Marshal(map[string]any{
		"type":          "openai",
		"openai_config": map[string]any{"base_url": stub.srv.URL, "api_key": "ek", "model": "bge"},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(embCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeEmbedder(context.Background(), 100)
	assert.True(t, out.Configured)
	assert.False(t, out.Reachable, "empty data[] ⇒ unreachable")
	assert.Equal(t, 200, out.HTTPStatus)
	assert.Contains(t, out.Error, "embedding")
}

func TestProbeEmbedder_404(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 404, `{"error":"model not found"}`)
	embCfg, _ := json.Marshal(map[string]any{
		"type":          "openai",
		"openai_config": map[string]any{"base_url": stub.srv.URL, "api_key": "ek", "model": "bge"},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(embCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeEmbedder(context.Background(), 100)
	assert.False(t, out.Reachable)
	assert.Equal(t, 404, out.HTTPStatus)
}

func TestProbeEmbedder_Unconfigured_NoRow(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"config"}))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeEmbedder(context.Background(), 100)
	assert.False(t, out.Configured)
	assert.Contains(t, out.Error, "no space_embedding row")
}

func TestProbeRerank_200(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 200, `{"results":[{"index":0},{"index":1}]}`)
	rrCfg, _ := json.Marshal(map[string]any{
		"type":          "openai",
		"openai_config": map[string]any{"base_url": stub.srv.URL, "api_key": "rk", "model": "rr"},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_rerank")).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(rrCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeRerank(context.Background(), 100)
	assert.True(t, out.Reachable)
	assert.Equal(t, 200, out.HTTPStatus)
	require.Len(t, stub.reqs, 1)
	require.Equal(t, "/rerank", stub.reqs[0].URL.Path)
}

func TestProbeRerank_500(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 500, `boom`)
	rrCfg, _ := json.Marshal(map[string]any{
		"type":          "openai",
		"openai_config": map[string]any{"base_url": stub.srv.URL, "api_key": "rk", "model": "rr"},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_rerank")).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(rrCfg))

	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: probeTimeout}}
	out := svc.probeRerank(context.Background(), 100)
	assert.False(t, out.Reachable)
	assert.Equal(t, 500, out.HTTPStatus)
}

// 5-second timeout cap: even if the upstream sleeps longer than the
// client timeout, the probe must return inside its own bound. We use a
// stub that delays > the client timeout and verify wall time.
//
// The delay is short (1.5s) so httptest.Server.Close — which waits for
// in-flight handlers — doesn't slow down the test suite. Override the
// client timeout to a tighter value than the delay so the probe times
// out before the stub returns.
func TestProbeChat_TimeoutCap(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	stub := newStubHTTP(t, 200, `{}`)
	stub.delay = 1500 * time.Millisecond // > probe timeout
	connCfg, _ := json.Marshal(map[string]any{"base_url": stub.srv.URL, "api_key": "sk", "model": "qwen"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))

	// override client to 200ms timeout for fast test
	svc := &SpaceDiagnoseService{db: db, httpClient: &http.Client{Timeout: 200 * time.Millisecond}}
	start := time.Now()
	out := svc.probeChat(context.Background())
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 1*time.Second, "timeout cap must short-circuit hung probe (took %v)", elapsed)
	assert.False(t, out.Reachable)
	assert.NotEmpty(t, out.Error)
}

// ─────────────────────────────────────────────────────────────────────────────
// MySQL count probe
// ─────────────────────────────────────────────────────────────────────────────

func TestRunMySQLCounts_HappyPath(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	const spaceID int64 = 100
	expectMySQLCounts(mock, spaceID, map[string]int64{
		"single_agent_draft":       6,
		"knowledge":                7,
		"knowledge_document":       7,
		"knowledge_document_slice": 2231,
		"model_meta":               1,
		"space_embedding":          1,
		"space_rerank":             1,
		"workflow_meta":            0,
		"plugin_draft":             0,
	})

	svc := &SpaceDiagnoseService{db: db}
	rows := svc.runMySQLCounts(context.Background(), spaceID)
	require.Len(t, rows, len(fixedTables))

	byName := map[string]spacemodel.DiagnoseMySQLTable{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	assert.Equal(t, int64(6), byName["single_agent_draft"].Rows)
	assert.Equal(t, int64(2231), byName["knowledge_document_slice"].Rows)
	assert.Equal(t, int64(1), byName["model_meta"].Rows)
	assert.Equal(t, int64(0), byName["workflow_meta"].Rows)
	for _, r := range rows {
		assert.Empty(t, r.Error, "no errors expected on happy path; got error on %s", r.Name)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

// One failing table must NOT zero out the others — each row is per-table.
func TestRunMySQLCounts_OneTableFails_OthersSurvive(t *testing.T) {
	db, mock := mustNewSqlmock(t)
	const spaceID int64 = 100
	for _, tbl := range fixedTables {
		eq := mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM " + tbl.name))
		if tbl.name == "knowledge_document_slice" {
			// Failing this one validates the per-row error isolation.
			eq.WillReturnError(errors.New("table not found"))
			continue
		}
		if tbl.spaceScoped {
			eq.WithArgs(spaceID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		} else {
			eq.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		}
	}

	svc := &SpaceDiagnoseService{db: db}
	rows := svc.runMySQLCounts(context.Background(), spaceID)
	byName := map[string]spacemodel.DiagnoseMySQLTable{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	assert.NotEmpty(t, byName["knowledge_document_slice"].Error, "failed table must report Error")
	assert.Empty(t, byName["model_meta"].Error)
	assert.Equal(t, int64(1), byName["model_meta"].Rows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ─────────────────────────────────────────────────────────────────────────────
// ES section
// ─────────────────────────────────────────────────────────────────────────────

func TestRunESCounts_MixedExistence(t *testing.T) {
	es := newFakeES()
	es.exists["project_draft"] = true
	es.counts["project_draft"] = 4
	es.exists["coze_resource"] = true
	es.counts["coze_resource"] = 70
	es.exists["openynet_111"] = true
	es.counts["openynet_111"] = 104
	es.exists["openynet_222"] = false // missing index
	es.counts["openynet_222"] = 0

	svc := &SpaceDiagnoseService{esClient: es}
	rows := svc.runESCounts(context.Background(), []kbRow{
		{ID: 111, Name: "Bank Q&A"},
		{ID: 222, Name: "Stale KB"},
	})
	require.Len(t, rows, 4)

	byName := map[string]spacemodel.DiagnoseESIndex{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	assert.True(t, byName["project_draft"].Exists)
	assert.Equal(t, int64(4), byName["project_draft"].DocCount)
	assert.True(t, byName["coze_resource"].Exists)
	assert.Equal(t, int64(70), byName["coze_resource"].DocCount)
	assert.True(t, byName["openynet_111"].Exists)
	assert.Equal(t, int64(104), byName["openynet_111"].DocCount)
	assert.False(t, byName["openynet_222"].Exists, "missing index ⇒ exists=false")
	assert.Empty(t, byName["openynet_222"].Error, "missing index is not an error condition")
}

func TestRunESCounts_ExistsError_RowGetsError(t *testing.T) {
	es := newFakeES()
	es.exists["project_draft"] = true
	es.counts["project_draft"] = 1
	es.exists["coze_resource"] = true
	es.counts["coze_resource"] = 1
	es.existsErr["openynet_333"] = errors.New("es down")

	svc := &SpaceDiagnoseService{esClient: es}
	rows := svc.runESCounts(context.Background(), []kbRow{{ID: 333, Name: "kb"}})
	require.Len(t, rows, 3)
	byName := map[string]spacemodel.DiagnoseESIndex{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	assert.False(t, byName["openynet_333"].Exists)
	assert.Contains(t, byName["openynet_333"].Error, "es down")
	// The two fixed indices still get counted — section is best-effort per-index.
	assert.True(t, byName["project_draft"].Exists)
	assert.Equal(t, int64(1), byName["project_draft"].DocCount)
}

func TestRunESCounts_NilClient_AllRowsErrored(t *testing.T) {
	svc := &SpaceDiagnoseService{esClient: nil}
	rows := svc.runESCounts(context.Background(), []kbRow{{ID: 1, Name: "kb"}})
	require.Len(t, rows, 3) // project_draft + coze_resource + openynet_1
	for _, r := range rows {
		assert.False(t, r.Exists)
		assert.Contains(t, r.Error, "es client")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Milvus section
// ─────────────────────────────────────────────────────────────────────────────

func TestRunMilvusCollections_MixedExistence(t *testing.T) {
	mgr := newFakeVectorMgr()
	mgr.getErr["openynet_222"] = errors.New("collection not found")

	svc := &SpaceDiagnoseService{ssManagers: []searchstore.Manager{mgr}}
	rows := svc.runMilvusCollections(context.Background(), []kbRow{
		{ID: 111, Name: "AI Banking"},
		{ID: 222, Name: "Stale KB"},
	})
	require.Len(t, rows, 2)
	byID := map[int64]spacemodel.DiagnoseMilvusCollection{}
	for _, r := range rows {
		byID[r.KbID] = r
	}
	assert.True(t, byID[111].Exists)
	assert.Equal(t, "openynet_111", byID[111].CollectionName)
	assert.Equal(t, "AI Banking", byID[111].KbName)
	assert.False(t, byID[222].Exists)
	assert.Contains(t, byID[222].Error, "collection not found")
}

func TestRunMilvusCollections_NoManager(t *testing.T) {
	svc := &SpaceDiagnoseService{ssManagers: nil}
	rows := svc.runMilvusCollections(context.Background(), []kbRow{{ID: 1, Name: "kb"}})
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Exists)
	assert.Contains(t, rows[0].Error, "no vector-store")
}

// Only text-store managers? We still skip them — vector section reports
// "no vector-store manager".
func TestRunMilvusCollections_OnlyTextManager_SkipsTextStore(t *testing.T) {
	textOnly := newFakeVectorMgr()
	textOnly.storeType = searchstore.TypeTextStore
	svc := &SpaceDiagnoseService{ssManagers: []searchstore.Manager{textOnly}}
	rows := svc.runMilvusCollections(context.Background(), []kbRow{{ID: 1, Name: "kb"}})
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Exists)
	assert.Contains(t, rows[0].Error, "no vector-store")
	assert.Empty(t, textOnly.gotName, "text-store manager must NOT be touched")
}

// ─────────────────────────────────────────────────────────────────────────────
// End-to-end Diagnose (full flow)
// ─────────────────────────────────────────────────────────────────────────────

// E2E test: owner check passes → all 4 sections produce results → no error.
func TestDiagnose_E2E_HappyPath(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: spaceID, OwnerID: 42}}

	// HTTP stubs for the 3 probes.
	chatSrv := newStubHTTP(t, 200, `{"choices":[{"text":"pong"}]}`)
	embSrv := newStubHTTP(t, 200, `{"data":[{"embedding":[0.1,0.2]}]}`)
	rrSrv := newStubHTTP(t, 200, `{"results":[]}`)

	// SQL mock: chat read, embedder read, rerank read, list KBs, 9 counts.
	es := newFakeES()
	es.exists["project_draft"] = true
	es.counts["project_draft"] = 4
	es.exists["coze_resource"] = true
	es.counts["coze_resource"] = 70
	es.exists["openynet_55"] = true
	es.counts["openynet_55"] = 33
	mgr := newFakeVectorMgr()

	svc, mock, cleanup := newDiagSvc(t, user, es, []searchstore.Manager{mgr})
	defer cleanup()

	// Probe SQL rows
	connCfg, _ := json.Marshal(map[string]any{"base_url": chatSrv.srv.URL, "api_key": "sk", "model": "qwen"})
	mock.MatchExpectationsInOrder(false) // 4 sections run in parallel
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))
	embCfg, _ := json.Marshal(map[string]any{
		"type":          "openai",
		"openai_config": map[string]any{"base_url": embSrv.srv.URL, "api_key": "ek", "model": "bge"},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
		WithArgs(spaceID).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(embCfg))
	rrCfg, _ := json.Marshal(map[string]any{
		"type":          "openai",
		"openai_config": map[string]any{"base_url": rrSrv.srv.URL, "api_key": "rk", "model": "rr"},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_rerank")).
		WithArgs(spaceID).
		WillReturnRows(sqlmock.NewRows([]string{"config"}).AddRow(rrCfg))
	// listKBs runs in main goroutine BEFORE the parallel fan-out.
	expectListKBs(mock, spaceID, []kbRow{{ID: 55, Name: "Bank Q&A"}})
	// 9 fixed counts
	expectMySQLCounts(mock, spaceID, map[string]int64{
		"single_agent_draft":       6,
		"knowledge":                7,
		"knowledge_document":       7,
		"knowledge_document_slice": 2231,
		"model_meta":               1,
		"space_embedding":          1,
		"space_rerank":             1,
		"workflow_meta":            0,
		"plugin_draft":             0,
	})

	resp, err := svc.Diagnose(ctxWithUser(42), &spacemodel.DiagnoseRequest{SpaceID: spaceID})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int64(0), resp.Code)
	require.NotNil(t, resp.Data)

	assert.True(t, resp.Data.ModelProbes.Chat.Reachable, "chat probe failed: %q", resp.Data.ModelProbes.Chat.Error)
	assert.True(t, resp.Data.ModelProbes.Embedder.Reachable, "embedder probe failed: %q", resp.Data.ModelProbes.Embedder.Error)
	assert.True(t, resp.Data.ModelProbes.Rerank.Reachable, "rerank probe failed: %q", resp.Data.ModelProbes.Rerank.Error)

	require.Len(t, resp.Data.MySQLTables, len(fixedTables))
	require.Len(t, resp.Data.ESIndices, 3) // 2 fixed + 1 kb
	require.Len(t, resp.Data.MilvusCollections, 1)
	assert.True(t, resp.Data.MilvusCollections[0].Exists)
	assert.Equal(t, int64(55), resp.Data.MilvusCollections[0].KbID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// E2E partial failure: ES is down, MySQL is OK, model endpoints return 500.
// Verify caller still gets a response with per-section errors.
func TestDiagnose_E2E_PartialFailure_StillReturns(t *testing.T) {
	const spaceID int64 = 100
	user := &fakeUserSVC{space: &userentity.Space{ID: spaceID, OwnerID: 42}}

	chatSrv := newStubHTTP(t, 500, `down`)
	es := newFakeES()
	es.exists["project_draft"] = true
	es.counts["project_draft"] = 0
	es.exists["coze_resource"] = true
	es.counts["coze_resource"] = 0

	svc, mock, cleanup := newDiagSvc(t, user, es, nil) // no vector mgr
	defer cleanup()

	connCfg, _ := json.Marshal(map[string]any{"base_url": chatSrv.srv.URL, "api_key": "sk", "model": "qwen"})
	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
		WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))
	// embedder + rerank unconfigured
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
		WithArgs(spaceID).WillReturnRows(sqlmock.NewRows([]string{"config"}))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_rerank")).
		WithArgs(spaceID).WillReturnRows(sqlmock.NewRows([]string{"config"}))
	expectListKBs(mock, spaceID, []kbRow{{ID: 77, Name: "kb"}})
	expectMySQLCounts(mock, spaceID, map[string]int64{"knowledge": 1, "model_meta": 1})

	resp, err := svc.Diagnose(ctxWithUser(42), &spacemodel.DiagnoseRequest{SpaceID: spaceID})
	require.NoError(t, err, "partial failure must not bubble as top-level error")
	require.NotNil(t, resp)
	assert.False(t, resp.Data.ModelProbes.Chat.Reachable)
	assert.False(t, resp.Data.ModelProbes.Embedder.Configured)
	assert.False(t, resp.Data.ModelProbes.Rerank.Configured)
	assert.NotEmpty(t, resp.Data.MySQLTables)
	require.Len(t, resp.Data.ESIndices, 3)            // project_draft + coze_resource + openynet_77
	require.Len(t, resp.Data.MilvusCollections, 1)    // kb 77, no manager → error row
	assert.False(t, resp.Data.MilvusCollections[0].Exists)
	assert.Contains(t, resp.Data.MilvusCollections[0].Error, "no vector-store")
}

// Concurrent callers on the same service — checks the goroutine fan-out is
// race-free under -race. Each caller gets its own sqlmock (sqlmock is
// single-threaded) but shares the ES + manager fakes.
func TestDiagnose_Concurrent_SameService(t *testing.T) {
	const spaceID int64 = 100
	const callers = 3
	user := &concurrentUserSVC{space: &userentity.Space{ID: spaceID, OwnerID: 42}}

	chatSrv := newStubHTTP(t, 200, `{}`)
	es := newFakeES()
	es.exists["project_draft"] = true
	es.exists["coze_resource"] = true
	mgr := newFakeVectorMgr()

	type plan struct {
		svc     *SpaceDiagnoseService
		cleanup func()
	}
	plans := make([]plan, callers)
	for i := 0; i < callers; i++ {
		rawDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: rawDB, SkipInitializeWithVersion: true}), &gorm.Config{})
		require.NoError(t, err)

		mock.MatchExpectationsInOrder(false)
		connCfg, _ := json.Marshal(map[string]any{"base_url": chatSrv.srv.URL, "api_key": "sk", "model": "m"})
		mock.ExpectQuery(regexp.QuoteMeta("SELECT conn_config FROM model_meta")).
			WillReturnRows(sqlmock.NewRows([]string{"conn_config"}).AddRow(connCfg))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_embedding")).
			WithArgs(spaceID).WillReturnRows(sqlmock.NewRows([]string{"config"}))
		mock.ExpectQuery(regexp.QuoteMeta("SELECT config FROM space_rerank")).
			WithArgs(spaceID).WillReturnRows(sqlmock.NewRows([]string{"config"}))
		expectListKBs(mock, spaceID, nil)
		expectMySQLCounts(mock, spaceID, nil)

		plans[i] = plan{
			svc: &SpaceDiagnoseService{
				userSVC:    user,
				db:         gormDB,
				esClient:   es,
				ssManagers: []searchstore.Manager{mgr},
				httpClient: &http.Client{Timeout: probeTimeout},
			},
			cleanup: func() { _ = rawDB.Close() },
		}
	}
	defer func() {
		for _, p := range plans {
			p.cleanup()
		}
	}()

	var (
		wg         sync.WaitGroup
		start      = make(chan struct{})
		errs       = make([]error, callers)
		respCounts = int64(0)
	)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			<-start
			resp, err := plans[i].svc.Diagnose(ctxWithUser(42), &spacemodel.DiagnoseRequest{SpaceID: spaceID})
			errs[i] = err
			if resp != nil {
				atomic.AddInt64(&respCounts, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		assert.NoErrorf(t, err, "caller %d failed", i)
	}
	assert.Equal(t, int64(callers), respCounts)
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestJoinPath(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		{"http://x/v1", "chat/completions", "http://x/v1/chat/completions"},
		{"http://x/v1/", "/chat/completions", "http://x/v1/chat/completions"},
		{"http://x", "embeddings", "http://x/embeddings"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, joinPath(c.base, c.path))
	}
}

func TestFlattenEmbRerankConfig(t *testing.T) {
	// nested openai_config wins
	cfg, ok, err := flattenEmbRerankConfig([]byte(`{"type":"openai","openai_config":{"base_url":"a","api_key":"b","model":"c"}}`))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "a", cfg.BaseURL)
	assert.Equal(t, "b", cfg.APIKey)
	assert.Equal(t, "c", cfg.Model)

	// top-level fields used when openai_config is missing
	cfg, ok, err = flattenEmbRerankConfig([]byte(`{"base_url":"x","api_key":"y","model":"z"}`))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "x", cfg.BaseURL)
	assert.Equal(t, "y", cfg.APIKey)
	assert.Equal(t, "z", cfg.Model)

	// openai_config overrides top-level
	cfg, ok, err = flattenEmbRerankConfig([]byte(`{"base_url":"top","openai_config":{"base_url":"nested"}}`))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "nested", cfg.BaseURL)

	// bad JSON
	_, _, err = flattenEmbRerankConfig([]byte(`not-json`))
	assert.Error(t, err)
}

func TestKbCollectionName(t *testing.T) {
	assert.Equal(t, "openynet_42", kbCollectionName(42))
	assert.Equal(t, "openynet_7643406904894423040", kbCollectionName(7643406904894423040))
}

func TestEmbeddingResponseLooksValid(t *testing.T) {
	assert.True(t, embeddingResponseLooksValid([]byte(`{"data":[{"embedding":[0.1]}]}`)))
	assert.False(t, embeddingResponseLooksValid([]byte(`{"data":[]}`)))
	assert.False(t, embeddingResponseLooksValid([]byte(`{"data":[{"embedding":[]}]}`)))
	assert.False(t, embeddingResponseLooksValid([]byte(`not-json`)))
}
