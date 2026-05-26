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

// ChatModelConfig is the chat LLM section of the one-shot reconfig payload.
// All fields are required if `chat` is present in the request.
type ChatModelConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// EmbedderModelConfig is the embedder section. Mirrors the OpenAI-compatible
// embedder config persisted in space_embedding.config.openai_config.
type EmbedderModelConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	Dims    int    `json:"dims"`
}

// RerankModelConfig is the rerank section. Mirrors space_rerank.config.openai_config.
type RerankModelConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// ConfigureModelsRequest is the body for POST /api/space/configure_models.
//
// Any of `chat` / `embedder` / `rerank` is optional — if omitted, the
// corresponding UPDATE is skipped.
type ConfigureModelsRequest struct {
	SpaceID  int64                `thrift:"space_id,1,required" form:"space_id" json:"space_id,string"`
	Chat     *ChatModelConfig     `thrift:"chat,2,optional" json:"chat,omitempty"`
	Embedder *EmbedderModelConfig `thrift:"embedder,3,optional" json:"embedder,omitempty"`
	Rerank   *RerankModelConfig   `thrift:"rerank,4,optional" json:"rerank,omitempty"`
	Base     *base.Base           `thrift:"Base,255,optional" json:"Base,omitempty"`
}

// ConfigureModelsCounts is the diagnostic payload returned to the operator.
// Each *Updated count is the number of rows GORM reports affected by the
// corresponding UPDATE; RedisKeysDeleted is the count of cache keys actually
// removed (0 is fine — means there was nothing cached for this space).
//
// Warnings holds non-fatal issues (e.g. Redis nil, partial cache cleanup) —
// the API still returns code=0 in those cases.
type ConfigureModelsCounts struct {
	ModelMetaUpdated      int64    `json:"model_meta_updated"`
	SpaceEmbeddingUpdated int64    `json:"space_embedding_updated"`
	SpaceRerankUpdated    int64    `json:"space_rerank_updated"`
	RedisKeysDeleted      int64    `json:"redis_keys_deleted"`
	Warnings              []string `json:"warnings"`
}

// ConfigureModelsResponse wraps the diagnostic counts. `code != 0` ⇒ inspect
// `msg`; `code == 0` ⇒ operation succeeded, but check `data.warnings` for any
// non-fatal degradation (e.g. Redis layer was unreachable).
type ConfigureModelsResponse struct {
	Code int64                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *ConfigureModelsCounts `json:"data"`
}
