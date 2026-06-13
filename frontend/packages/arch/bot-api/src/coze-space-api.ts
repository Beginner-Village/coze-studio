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

// Read-only per-space health-check / diagnose.
//
// Reflects backend/api/model/data/space/diagnose.go — the shape is stable
// because the UI renders per-section partial results, so an empty array
// or `error` non-empty inside a section is the expected "this part is
// degraded" signal, NOT a top-level failure.
export interface DiagnoseRequest {
  space_id: string; // string to dodge JS bigint precision loss
}

export interface DiagnoseModelProbe {
  configured: boolean;
  endpoint?: string;
  model?: string;
  reachable: boolean;
  latency_ms: number;
  http_status: number;
  error?: string;
}

export interface DiagnoseMySQLTable {
  name: string;
  rows: number;
  error?: string;
}

export interface DiagnoseESIndex {
  name: string;
  exists: boolean;
  doc_count: number;
  error?: string;
}

export interface DiagnoseMilvusCollection {
  kb_id: string; // sent as JSON-string from Go for bigint safety
  kb_name: string;
  collection_name: string;
  exists: boolean;
  error?: string;
}

export interface DiagnoseData {
  model_probes: {
    chat: DiagnoseModelProbe;
    embedder: DiagnoseModelProbe;
    rerank: DiagnoseModelProbe;
  };
  mysql_tables: DiagnoseMySQLTable[];
  es_indices: DiagnoseESIndex[];
  milvus_collections: DiagnoseMilvusCollection[];
}

export interface DiagnoseResponse {
  code: number;
  msg: string;
  data: DiagnoseData | null;
}

// 空间级操作审计日志查询接口（非 IDL 生成，走 /api/operation_log/*）。
//
// 后端 backend/api/model/data/operationlog/operation_log.go 的 id 字段
// 均以 JSON 字符串形式传输（json:"...,string"），避免 JS bigint 精度丢失，
// 因此这里全部声明为 string。
export interface OperationLogListRequest {
  space_id: string; // 数字字符串
  operator_id?: string;
  resource_type?: number;
  action?: string;
  start_time?: number; // 毫秒
  end_time?: number; // 毫秒
  keyword?: string;
  page: number;
  page_size: number;
}

export interface OperationLogItem {
  id: string;
  operator_id: string;
  operator_name: string;
  module: string;
  resource_type: number;
  resource_id: string;
  resource_name: string;
  action: string;
  description: string;
  status: number; // 1=成功 2=失败
  client_ip: string;
  created_at: number; // 毫秒
}

export interface OperationLogListResponse {
  code: number;
  msg: string;
  logs: OperationLogItem[] | null;
  total: number;
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

  /**
   * Read-only per-space health-check.
   *
   * Backend probes chat / embedder / rerank endpoints (5s timeout each,
   * run in parallel server-side), counts rows in 9 fixed MySQL tables,
   * counts docs in every ES index reachable for the space (including
   * per-KB openynet_<kb_id>), and checks Milvus collection existence
   * per KB. All sections return regardless of partial failure — the UI
   * inspects per-section `error` to render green/red status.
   *
   * Owner-only. Read-only: never writes Coze state.
   */
  async diagnose(
    data: DiagnoseRequest,
    config?: BotAPIRequestConfig,
  ): Promise<DiagnoseResponse> {
    return await axiosInstance.post('/api/space/diagnose', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  /**
   * Query the space-level operation audit log.
   *
   * Owner/Admin only — backend returns a permission business error
   * (code 112100001) for regular members. Supports filtering by
   * operator / resource_type / action / time-range / keyword, paged.
   */
  async listOperationLog(
    data: OperationLogListRequest,
    config?: BotAPIRequestConfig,
  ): Promise<OperationLogListResponse> {
    return await axiosInstance.post('/api/operation_log/list', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }
}

// eslint-disable-next-line @typescript-eslint/naming-convention
export const SpaceApi = new SpaceApiService();
