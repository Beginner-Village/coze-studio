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

import "github.com/ynet-dev/ynet-studio/backend/api/model/base"

// DiagnoseRequest is the body for POST /api/space/diagnose. Read-only.
type DiagnoseRequest struct {
	SpaceID int64      `thrift:"space_id,1,required" form:"space_id" json:"space_id,string"`
	Base    *base.Base `thrift:"Base,255,optional" json:"Base,omitempty"`
}

// ModelProbeResult is the per-model live-probe outcome. `Configured=false`
// means the corresponding DB row was missing (or its JSON empty); the probe
// was then skipped and only `Error` is populated. `Reachable=true` ⇒ the
// endpoint returned 2xx within the 5s timeout AND, for embedder/rerank,
// the response body had the expected shape.
type ModelProbeResult struct {
	Configured bool   `json:"configured"`
	Endpoint   string `json:"endpoint,omitempty"`
	Model      string `json:"model,omitempty"`
	Reachable  bool   `json:"reachable"`
	LatencyMs  int64  `json:"latency_ms"`
	HTTPStatus int    `json:"http_status"`
	Error      string `json:"error,omitempty"`
}

// DiagnoseModelProbes packs the three live probes that run in parallel.
type DiagnoseModelProbes struct {
	Chat     ModelProbeResult `json:"chat"`
	Embedder ModelProbeResult `json:"embedder"`
	Rerank   ModelProbeResult `json:"rerank"`
}

// DiagnoseMySQLTable is a (table, row_count) pair. `Error` is set when the
// COUNT query failed and the row count is meaningless.
type DiagnoseMySQLTable struct {
	Name  string `json:"name"`
	Rows  int64  `json:"rows"`
	Error string `json:"error,omitempty"`
}

// DiagnoseESIndex is the existence + doc-count outcome for one ES index.
// `Exists=false` ⇒ either the index is genuinely missing or the count call
// reported "not found"; `DocCount=0` in either case.
type DiagnoseESIndex struct {
	Name     string `json:"name"`
	Exists   bool   `json:"exists"`
	DocCount int64  `json:"doc_count"`
	Error    string `json:"error,omitempty"`
}

// DiagnoseMilvusCollection is the per-KB vector-store sanity check. We don't
// try to read row counts (HasCollection / GetSearchStore isn't required to
// expose them across all backends — ES is the source of truth for slices).
type DiagnoseMilvusCollection struct {
	KbID           int64  `json:"kb_id,string"`
	KbName         string `json:"kb_name"`
	CollectionName string `json:"collection_name"`
	Exists         bool   `json:"exists"`
	Error          string `json:"error,omitempty"`
}

// DiagnoseData is the full response payload. ALL sections are always present
// (even if empty / errored) so the UI can render a partial result rather than
// guessing why a section is missing.
type DiagnoseData struct {
	ModelProbes       DiagnoseModelProbes        `json:"model_probes"`
	MySQLTables       []DiagnoseMySQLTable       `json:"mysql_tables"`
	ESIndices         []DiagnoseESIndex          `json:"es_indices"`
	MilvusCollections []DiagnoseMilvusCollection `json:"milvus_collections"`
}

// DiagnoseResponse wraps the full health-check payload.
type DiagnoseResponse struct {
	Code int64         `json:"code"`
	Msg  string        `json:"msg"`
	Data *DiagnoseData `json:"data"`
}
