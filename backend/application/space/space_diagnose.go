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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// DiagnoseSVC is the global per-space read-only health-check service.
// Wired in initComplexServices alongside ResyncSVC / ConfigureModelsSVC.
var DiagnoseSVC *SpaceDiagnoseService

// probeTimeout caps every outbound HTTP probe so a hung model endpoint can't
// stall the diagnose call. 5s matches the spec: short enough to keep the
// total wall-clock bounded (probes run in parallel ⇒ wall ≤ 5s), long enough
// to give a struggling-but-alive endpoint a chance.
const probeTimeout = 5 * time.Second


// SpaceDiagnoseService runs the read-only health-check for a single space.
//
// READ-ONLY guarantee: none of the probe paths issue MySQL writes, Redis
// writes, ES writes, or queue events. The chat/embedder/rerank probes are
// outbound HTTP POSTs to the configured model endpoints — those endpoints
// may log, bill, etc. but they don't mutate Coze's own state.
//
// Sequencing:
//  1. permission check (owner only, same helper Resync/ConfigureModels uses)
//  2. four sections run in parallel via WaitGroup so the total wall time is
//     max(probe, mysql, es, milvus), not the sum.
//  3. EVERY section returns its results even on partial failure — the UI
//     renders per-section ok/error so operators can debug across components.
type SpaceDiagnoseService struct {
	userSVC      usersvc.User
	db           *gorm.DB
	esClient     es.Client
	ssManagers   []searchstore.Manager
	httpClient   *http.Client // shared, with probeTimeout — never nil
}

// InitDiagnoseService wires the read-only diagnose service. Must be called
// after the user/search/knowledge application services have init'd because
// it shares the same userSVC and the infra-level es client + searchstore
// managers.
func InitDiagnoseService(userSVC usersvc.User, db *gorm.DB, esClient es.Client, ssManagers []searchstore.Manager) {
	DiagnoseSVC = &SpaceDiagnoseService{
		userSVC:    userSVC,
		db:         db,
		esClient:   esClient,
		ssManagers: ssManagers,
		httpClient: &http.Client{Timeout: probeTimeout},
	}
}

// Diagnose runs all 4 health-check sections in parallel and returns the union.
// Per-section errors are surfaced inside the response (never as a top-level
// error) so the UI can render partial results — only auth / permission
// failures produce a top-level error here.
func (s *SpaceDiagnoseService) Diagnose(ctx context.Context, req *spacemodel.DiagnoseRequest) (*spacemodel.DiagnoseResponse, error) {
	// 1. permission: only the space owner can pull diagnostics.
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	spaceInfo, err := s.userSVC.GetSpaceByID(ctx, req.SpaceID)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceNotFoundCode, errorx.KV("msg", fmt.Sprintf("get space %d failed", req.SpaceID)))
	}
	if spaceInfo == nil {
		return nil, errorx.New(errno.ErrSpaceNotFoundCode, errorx.KV("msg", fmt.Sprintf("space %d not found", req.SpaceID)))
	}
	if spaceInfo.OwnerID != userID {
		return nil, errorx.New(errno.ErrSpacePermissionCode, errorx.KV("msg", "only the space owner can run diagnose"))
	}

	// 2. KB rows up-front — both ES (openynet_<kb_id>) and Milvus (same name)
	// sections need them, so fetch once and pass into both. A failure here
	// degrades those two sections to a single zero-row report; MySQL and
	// model probes still run.
	kbs, kbErr := s.listSpaceKBs(ctx, req.SpaceID)
	if kbErr != nil {
		logs.CtxWarnf(ctx, "[Diagnose] space=%d list kbs failed: %v", req.SpaceID, kbErr)
	}

	// 3. four sections in parallel — each writes into its slot of `data`.
	data := &spacemodel.DiagnoseData{}
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		data.ModelProbes = s.runModelProbes(ctx, req.SpaceID)
	}()
	go func() {
		defer wg.Done()
		data.MySQLTables = s.runMySQLCounts(ctx, req.SpaceID)
	}()
	go func() {
		defer wg.Done()
		data.ESIndices = s.runESCounts(ctx, kbs)
	}()
	go func() {
		defer wg.Done()
		data.MilvusCollections = s.runMilvusCollections(ctx, kbs)
	}()
	wg.Wait()

	logs.CtxInfof(ctx, "[Diagnose] space=%d done: chat_reachable=%v embedder_reachable=%v rerank_reachable=%v tables=%d es_indices=%d milvus_collections=%d",
		req.SpaceID,
		data.ModelProbes.Chat.Reachable,
		data.ModelProbes.Embedder.Reachable,
		data.ModelProbes.Rerank.Reachable,
		len(data.MySQLTables), len(data.ESIndices), len(data.MilvusCollections))

	return &spacemodel.DiagnoseResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	}, nil
}

// ----- model probes -------------------------------------------------------

// runModelProbes fans out the 3 probes (chat, embedder, rerank) concurrently.
// Each runs its own DB lookup + HTTP POST. A failure in one MUST NOT affect
// the others.
func (s *SpaceDiagnoseService) runModelProbes(ctx context.Context, spaceID int64) spacemodel.DiagnoseModelProbes {
	var out spacemodel.DiagnoseModelProbes
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		out.Chat = s.probeChat(ctx)
	}()
	go func() {
		defer wg.Done()
		out.Embedder = s.probeEmbedder(ctx, spaceID)
	}()
	go func() {
		defer wg.Done()
		out.Rerank = s.probeRerank(ctx, spaceID)
	}()
	wg.Wait()
	return out
}

// chatConnConfig models the subset of model_meta.conn_config we care about.
type chatConnConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// probeChat reads the first active model_meta row, parses conn_config, and
// POSTs a 1-token /chat/completions request. The 1-token cap keeps the
// upstream cost minimal even if the endpoint is happy and replies in full.
func (s *SpaceDiagnoseService) probeChat(ctx context.Context) spacemodel.ModelProbeResult {
	var row struct {
		ConnConfig []byte `gorm:"column:conn_config"`
	}
	err := s.db.WithContext(ctx).Raw(
		"SELECT conn_config FROM model_meta WHERE deleted_at IS NULL ORDER BY id ASC LIMIT 1",
	).Scan(&row).Error
	if err != nil {
		return spacemodel.ModelProbeResult{Configured: false, Error: fmt.Sprintf("read model_meta failed: %v", err)}
	}
	if len(row.ConnConfig) == 0 {
		return spacemodel.ModelProbeResult{Configured: false, Error: "no active model_meta row"}
	}
	var cfg chatConnConfig
	if err := json.Unmarshal(row.ConnConfig, &cfg); err != nil {
		return spacemodel.ModelProbeResult{Configured: false, Error: fmt.Sprintf("decode conn_config failed: %v", err)}
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		return spacemodel.ModelProbeResult{Configured: false, Endpoint: cfg.BaseURL, Model: cfg.Model, Error: "conn_config missing base_url or model"}
	}

	body, _ := json.Marshal(map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 1,
	})
	status, _, latency, herr := s.httpPostJSON(ctx, joinPath(cfg.BaseURL, "chat/completions"), cfg.APIKey, body)
	out := spacemodel.ModelProbeResult{
		Configured: true,
		Endpoint:   cfg.BaseURL,
		Model:      cfg.Model,
		LatencyMs:  latency,
		HTTPStatus: status,
	}
	switch {
	case herr != nil:
		out.Reachable = false
		out.Error = herr.Error()
	case status < 200 || status >= 300:
		out.Reachable = false
		out.Error = fmt.Sprintf("non-2xx response: %d", status)
	default:
		out.Reachable = true
	}
	return out
}

// embRerankConfig is the flattened view of space_embedding.config /
// space_rerank.config that the probes consume. The persisted JSON has a
// vendor-specific subtree (openai_config, ark_config, ...), but the three
// fields that matter for the probe — base_url, api_key, model — appear under
// `openai_config` for OpenAI-compatible endpoints (the customer's case) and
// directly at the top level for ark/ollama/http configs. readEmbRerank
// merges both into this flat shape.
type embRerankConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

// probeEmbedder pulls space_embedding for the space, POSTs a 1-token
// embedding request, and verifies the response includes a non-empty
// data[0].embedding (so an endpoint that returns 200 with garbage still
// fails the probe).
func (s *SpaceDiagnoseService) probeEmbedder(ctx context.Context, spaceID int64) spacemodel.ModelProbeResult {
	cfg, ok, err := s.readEmbRerank(ctx, "space_embedding", spaceID)
	if err != nil {
		return spacemodel.ModelProbeResult{Configured: false, Error: err.Error()}
	}
	if !ok {
		return spacemodel.ModelProbeResult{Configured: false, Error: "no space_embedding row"}
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		return spacemodel.ModelProbeResult{Configured: false, Endpoint: cfg.BaseURL, Model: cfg.Model, Error: "space_embedding config missing base_url or model"}
	}
	body, _ := json.Marshal(map[string]any{"model": cfg.Model, "input": "ping"})
	status, respBody, latency, herr := s.httpPostJSON(ctx, joinPath(cfg.BaseURL, "embeddings"), cfg.APIKey, body)
	out := spacemodel.ModelProbeResult{
		Configured: true,
		Endpoint:   cfg.BaseURL,
		Model:      cfg.Model,
		LatencyMs:  latency,
		HTTPStatus: status,
	}
	switch {
	case herr != nil:
		out.Reachable = false
		out.Error = herr.Error()
	case status < 200 || status >= 300:
		out.Reachable = false
		out.Error = fmt.Sprintf("non-2xx response: %d", status)
	case !embeddingResponseLooksValid(respBody):
		out.Reachable = false
		out.Error = "response missing data[].embedding"
	default:
		out.Reachable = true
	}
	return out
}

// probeRerank pulls space_rerank for the space and POSTs a 2-doc rerank
// query. A 2xx is treated as healthy — rerank response shapes vary too
// much across vendors to validate the body cheaply.
func (s *SpaceDiagnoseService) probeRerank(ctx context.Context, spaceID int64) spacemodel.ModelProbeResult {
	cfg, ok, err := s.readEmbRerank(ctx, "space_rerank", spaceID)
	if err != nil {
		return spacemodel.ModelProbeResult{Configured: false, Error: err.Error()}
	}
	if !ok {
		return spacemodel.ModelProbeResult{Configured: false, Error: "no space_rerank row"}
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		return spacemodel.ModelProbeResult{Configured: false, Endpoint: cfg.BaseURL, Model: cfg.Model, Error: "space_rerank config missing base_url or model"}
	}
	body, _ := json.Marshal(map[string]any{
		"model":     cfg.Model,
		"query":     "ping",
		"documents": []string{"doc1", "doc2"},
	})
	status, _, latency, herr := s.httpPostJSON(ctx, joinPath(cfg.BaseURL, "rerank"), cfg.APIKey, body)
	out := spacemodel.ModelProbeResult{
		Configured: true,
		Endpoint:   cfg.BaseURL,
		Model:      cfg.Model,
		LatencyMs:  latency,
		HTTPStatus: status,
	}
	switch {
	case herr != nil:
		out.Reachable = false
		out.Error = herr.Error()
	case status < 200 || status >= 300:
		out.Reachable = false
		out.Error = fmt.Sprintf("non-2xx response: %d", status)
	default:
		out.Reachable = true
	}
	return out
}

// readEmbRerank pulls the single active config row for either space_embedding
// or space_rerank, decodes its JSON, and flattens the vendor-specific subtree
// (openai_config, ark_config, ...) into a single (base_url, api_key, model)
// triple. Returns (cfg, found, error).
func (s *SpaceDiagnoseService) readEmbRerank(ctx context.Context, table string, spaceID int64) (embRerankConfig, bool, error) {
	if table != "space_embedding" && table != "space_rerank" {
		return embRerankConfig{}, false, fmt.Errorf("unexpected table %q", table)
	}
	var row struct {
		Config []byte `gorm:"column:config"`
	}
	// table name is whitelisted just above ⇒ safe to interpolate.
	err := s.db.WithContext(ctx).Raw(
		fmt.Sprintf("SELECT config FROM %s WHERE space_id = ? AND deleted_at IS NULL ORDER BY id ASC LIMIT 1", table),
		spaceID,
	).Scan(&row).Error
	if err != nil {
		return embRerankConfig{}, false, fmt.Errorf("read %s failed: %w", table, err)
	}
	if len(row.Config) == 0 {
		return embRerankConfig{}, false, nil
	}
	return flattenEmbRerankConfig(row.Config)
}

// flattenEmbRerankConfig pulls base_url / api_key / model out of either
// `config.openai_config.*` (OpenAI-compatible — the customer's deployment)
// or the top-level `config.*` (ark / ollama / http variants). Whichever
// path has them wins; openai_config takes precedence so the canonical
// shape is honoured even if both happen to be set.
func flattenEmbRerankConfig(raw []byte) (embRerankConfig, bool, error) {
	var nested struct {
		BaseURL      string `json:"base_url"`
		APIKey       string `json:"api_key"`
		Model        string `json:"model"`
		OpenAIConfig struct {
			BaseURL string `json:"base_url"`
			APIKey  string `json:"api_key"`
			Model   string `json:"model"`
		} `json:"openai_config"`
	}
	if err := json.Unmarshal(raw, &nested); err != nil {
		return embRerankConfig{}, false, fmt.Errorf("decode config failed: %w", err)
	}
	out := embRerankConfig{
		BaseURL: nested.BaseURL,
		APIKey:  nested.APIKey,
		Model:   nested.Model,
	}
	if nested.OpenAIConfig.BaseURL != "" {
		out.BaseURL = nested.OpenAIConfig.BaseURL
	}
	if nested.OpenAIConfig.APIKey != "" {
		out.APIKey = nested.OpenAIConfig.APIKey
	}
	if nested.OpenAIConfig.Model != "" {
		out.Model = nested.OpenAIConfig.Model
	}
	return out, true, nil
}

// embeddingResponseLooksValid checks the body has data[].embedding with at
// least one float — enough to catch a 200 with an HTML error page or empty
// JSON without trying to validate every vendor variant.
func embeddingResponseLooksValid(body []byte) bool {
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return false
	}
	return len(parsed.Data) > 0 && len(parsed.Data[0].Embedding) > 0
}

// httpPostJSON issues a POST with JSON body and optional Bearer auth, returning
// (http status, body, latency_ms, error). Body is read fully so callers can
// validate response shape. The shared client carries the 5s timeout.
func (s *SpaceDiagnoseService) httpPostJSON(ctx context.Context, url, apiKey string, body []byte) (int, []byte, int64, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := s.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return 0, nil, latency, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, latency, nil
}

// joinPath concatenates a base URL and a path segment, collapsing any
// duplicate slash at the join. Avoids importing net/url for a one-liner.
func joinPath(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

// ----- mysql counts -------------------------------------------------------

// diagnoseTables is the fixed (table, scope) list of MySQL row-count probes.
// `spaceScoped=true` ⇒ count is restricted to the requested space_id;
// `false` ⇒ global count (model_meta is global, workflow_meta has no space
// column here either... wait — workflow_meta does. We keep it space-scoped
// for the tables that have space_id and global for the rest).
type diagnoseTable struct {
	name        string
	spaceScoped bool
}

// fixedTables matches the spec's 9 tables in order. Order is stable so the
// UI table render is deterministic.
var fixedTables = []diagnoseTable{
	{"single_agent_draft", true},
	{"knowledge", true},
	{"knowledge_document", false}, // joins via knowledge_id; cheaper to count globally for the diagnose
	{"knowledge_document_slice", false},
	{"model_meta", false},
	{"space_embedding", true},
	{"space_rerank", true},
	{"workflow_meta", true},
	{"plugin_draft", true},
}

// runMySQLCounts issues a COUNT(*) per fixed table, scoped to the space when
// applicable. Each query has its own error slot so one failing table doesn't
// blank the rest. All counts run sequentially against the shared *gorm.DB —
// cheap enough that goroutine fan-out is overhead.
func (s *SpaceDiagnoseService) runMySQLCounts(ctx context.Context, spaceID int64) []spacemodel.DiagnoseMySQLTable {
	out := make([]spacemodel.DiagnoseMySQLTable, 0, len(fixedTables))
	for _, t := range fixedTables {
		row := spacemodel.DiagnoseMySQLTable{Name: t.name}
		var count int64
		var err error
		if t.spaceScoped {
			err = s.db.WithContext(ctx).Raw(
				// All space-scoped tables we list here have space_id + deleted_at.
				fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE space_id = ? AND deleted_at IS NULL", t.name),
				spaceID,
			).Scan(&count).Error
		} else {
			err = s.db.WithContext(ctx).Raw(
				fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted_at IS NULL", t.name),
			).Scan(&count).Error
		}
		if err != nil {
			row.Error = err.Error()
		} else {
			row.Rows = count
		}
		out = append(out, row)
	}
	return out
}

// ----- es counts ----------------------------------------------------------

// fixedESIndices is the global list of always-present indices. Per-KB
// indices (openynet_<kb_id>) are appended at runtime.
var fixedESIndices = []string{"project_draft", "coze_resource"}

// kbRow is a (id, name) tuple — used for both ES and Milvus sections.
type kbRow struct {
	ID   int64
	Name string
}

// listSpaceKBs reads (id, name) for every non-deleted KB of the space.
// Pulled once and shared between the ES + Milvus sections.
func (s *SpaceDiagnoseService) listSpaceKBs(ctx context.Context, spaceID int64) ([]kbRow, error) {
	type row struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(
		"SELECT id, name FROM knowledge WHERE space_id = ? AND deleted_at IS NULL ORDER BY id ASC",
		spaceID,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]kbRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, kbRow{ID: r.ID, Name: r.Name})
	}
	return out, nil
}

// runESCounts queries each ES index for its doc count via Search(size=0).
// Indices that don't exist surface as exists=false with no error — Exists
// returns (false, nil) on a missing index. A connection error becomes
// exists=false + error so the UI shows red.
//
// We don't fan out across goroutines: ES queries are cheap (size=0), the
// index count is bounded by KB count, and the es.Client is HTTP-keep-alive
// shared.
func (s *SpaceDiagnoseService) runESCounts(ctx context.Context, kbs []kbRow) []spacemodel.DiagnoseESIndex {
	all := make([]string, 0, len(fixedESIndices)+len(kbs))
	all = append(all, fixedESIndices...)
	for _, k := range kbs {
		all = append(all, kbCollectionName(k.ID))
	}

	out := make([]spacemodel.DiagnoseESIndex, 0, len(all))
	for _, name := range all {
		row := spacemodel.DiagnoseESIndex{Name: name}
		if s.esClient == nil {
			row.Error = "es client not configured"
			out = append(out, row)
			continue
		}
		exists, err := s.esClient.Exists(ctx, name)
		if err != nil {
			row.Error = fmt.Sprintf("exists check failed: %v", err)
			out = append(out, row)
			continue
		}
		if !exists {
			out = append(out, row) // exists=false, doc_count=0
			continue
		}
		count, cerr := s.esDocCount(ctx, name)
		if cerr != nil {
			row.Exists = true
			row.Error = fmt.Sprintf("count failed: %v", cerr)
			out = append(out, row)
			continue
		}
		row.Exists = true
		row.DocCount = count
		out = append(out, row)
	}
	return out
}

// esDocCount runs a size=0 search and reads hits.total.value. Avoids
// adding a Count() method to the es.Client contract.
func (s *SpaceDiagnoseService) esDocCount(ctx context.Context, index string) (int64, error) {
	zero := 0
	resp, err := s.esClient.Search(ctx, index, &es.Request{Size: &zero})
	if err != nil {
		return 0, err
	}
	if resp == nil || resp.Hits.Total == nil {
		return 0, nil
	}
	return resp.Hits.Total.Value, nil
}

// kbCollectionName mirrors the private knowledge.service.getCollectionName.
// Kept in sync with that helper — change in one place ⇒ change both. The
// alternative (exporting it) leaks an internal detail of the knowledge
// domain, which the resync flow has also avoided so far.
func kbCollectionName(kbID int64) string {
	return fmt.Sprintf("openynet_%d", kbID)
}

// ----- milvus collections -------------------------------------------------

// runMilvusCollections walks the vector-store managers and reports per-KB
// existence. We iterate `ssManagers` filtered to TypeVectorStore — the text
// store is already covered by the ES section above.
//
// Existence is probed via GetSearchStore: it succeeds only if the backing
// collection exists. Different vector backends (Milvus, OB, VikingDB) treat
// "missing" differently — some return a typed error, some return
// (nil, err). We treat any error as exists=false and surface the message.
func (s *SpaceDiagnoseService) runMilvusCollections(ctx context.Context, kbs []kbRow) []spacemodel.DiagnoseMilvusCollection {
	out := make([]spacemodel.DiagnoseMilvusCollection, 0, len(kbs))
	vector := s.vectorManager()
	for _, kb := range kbs {
		name := kbCollectionName(kb.ID)
		row := spacemodel.DiagnoseMilvusCollection{
			KbID:           kb.ID,
			KbName:         kb.Name,
			CollectionName: name,
		}
		if vector == nil {
			row.Error = "no vector-store manager configured"
			out = append(out, row)
			continue
		}
		_, err := vector.GetSearchStore(ctx, name)
		if err != nil {
			row.Error = collectionMissingMessage(err)
			out = append(out, row)
			continue
		}
		row.Exists = true
		out = append(out, row)
	}
	return out
}

// vectorManager returns the first searchstore.Manager whose GetType is
// TypeVectorStore. Studio ships exactly one vector backend at runtime, but
// the slice is plural for forward-compat, so we iterate and pick the first.
func (s *SpaceDiagnoseService) vectorManager() searchstore.Manager {
	for _, m := range s.ssManagers {
		if m != nil && m.GetType() == searchstore.TypeVectorStore {
			return m
		}
	}
	return nil
}

// collectionMissingMessage normalises the vendor-specific "collection not
// found" string. Currently we just return err.Error() unwrapped — the UI
// only cares that the error exists.
func collectionMissingMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
