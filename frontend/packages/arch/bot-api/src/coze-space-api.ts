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

import CozeSpaceApiService from '@coze-arch/idl/stone_coze_space';

import { axiosInstance, type BotAPIRequestConfig } from './axios';

export const cozeSpaceApi = new CozeSpaceApiService<BotAPIRequestConfig>({
  request: (params, config = {}) => {
    const reqHeaders = {
      ...config.headers,
      ...params.headers,
      'Agw-Js-Conv': 'str',
    };
    return axiosInstance.request({ ...params, ...config, headers: reqHeaders });
  },
});

// Space 数据维护接口（非 IDL 生成，走 /api/space/*）

export interface ResyncESRequest {
  space_id: string; // 使用字符串避免 JS bigint 精度丢失
}

export interface ResyncESCounts {
  project_draft: number;
  coze_resource: number;
  kb_entries: number;
  slice_reindex_jobs: number;
}

export interface ResyncESResponse {
  code: number;
  msg: string;
  counts: ResyncESCounts | null;
}

// One-shot per-space model + embedder + rerank reconfig.
// Body keys are snake_case to match Go thrift tag conventions on the backend.
export interface ConfigureModelsChat {
  base_url: string;
  api_key: string;
  model: string;
}

export interface ConfigureModelsEmbedder {
  base_url: string;
  api_key: string;
  model: string;
  dims: number;
}

export interface ConfigureModelsRerank {
  base_url: string;
  api_key: string;
  model: string;
}

export interface ConfigureModelsRequest {
  space_id: string; // string to dodge JS bigint precision loss
  chat?: ConfigureModelsChat;
  embedder?: ConfigureModelsEmbedder;
  rerank?: ConfigureModelsRerank;
}

export interface ConfigureModelsCounts {
  model_meta_updated: number;
  space_embedding_updated: number;
  space_rerank_updated: number;
  redis_keys_deleted: number;
  warnings: string[];
}

export interface ConfigureModelsResponse {
  code: number;
  msg: string;
  data: ConfigureModelsCounts | null;
}

class SpaceApiService {
  /**
   * Re-sync all ES indices for the given space from MySQL.
   *
   * Drops + rebuilds project_draft / coze_resource / kb_entries; for each KB
   * drops the openynet_<kb_id> index and queues all slices for re-embedding
   * via the existing IndexSliceEvent pipeline.
   *
   * Only the space owner can call this — backend returns code != 0 otherwise.
   */
  async resyncES(
    data: ResyncESRequest,
    config?: BotAPIRequestConfig,
  ): Promise<ResyncESResponse> {
    return await axiosInstance.post('/api/space/resync_es', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  /**
   * One-shot per-space model reconfig.
   *
   * Backend rewrites model_meta.conn_config (global chat LLM),
   * space_embedding.config, and space_rerank.config in a single
   * transaction, then DELs every space:<space_id>:model:<entity_id>
   * Redis key so the new config takes effect on the next request.
   *
   * Any of `chat` / `embedder` / `rerank` can be omitted — that
   * section is then skipped server-side.
   *
   * Owner-only.
   */
  async configureModels(
    data: ConfigureModelsRequest,
    config?: BotAPIRequestConfig,
  ): Promise<ConfigureModelsResponse> {
    return await axiosInstance.post('/api/space/configure_models', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }
}

// eslint-disable-next-line @typescript-eslint/naming-convention
export const SpaceApi = new SpaceApiService();
