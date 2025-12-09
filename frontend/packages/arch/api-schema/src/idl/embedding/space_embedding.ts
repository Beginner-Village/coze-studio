/*
 * Copyright 2025 coze-dev Authors
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

import { createAPI } from './../../api/config';

/** Embedding类型 */
export enum EmbeddingType {
  OpenAI = 'openai',
  Ark = 'ark',
  Ollama = 'ollama',
  HTTP = 'http',
}

/** Embedding状态 */
export enum EmbeddingStatus {
  Enabled = 1,
  Disabled = 2,
}

/** OpenAI Embedding配置 */
export interface OpenAIEmbeddingConfig {
  base_url: string;
  api_key: string;
  model: string;
  by_azure?: boolean;
  api_version?: string;
  dims: number;
  request_dims?: number;
}

/** ARK Embedding配置 */
export interface ArkEmbeddingConfig {
  base_url?: string;
  api_key: string;
  model: string;
  dims: number;
  api_type?: string;
}

/** Ollama Embedding配置 */
export interface OllamaEmbeddingConfig {
  base_url: string;
  model: string;
  dims: number;
}

/** HTTP Embedding配置 */
export interface HttpEmbeddingConfig {
  addr: string;
  dims: number;
}

/** 统一Embedding配置 */
export interface EmbeddingConfig {
  type: EmbeddingType;
  max_batch_size?: number;
  openai_config?: OpenAIEmbeddingConfig;
  ark_config?: ArkEmbeddingConfig;
  ollama_config?: OllamaEmbeddingConfig;
  http_config?: HttpEmbeddingConfig;
}

/** 空间Embedding配置 */
export interface SpaceEmbeddingConfig {
  id: string;
  space_id: string;
  name: string;
  description?: string;
  embedding_type: EmbeddingType;
  config?: EmbeddingConfig;
  status: EmbeddingStatus;
  is_default: boolean;
  dimensions: number;
  created_at: number;
  updated_at: number;
}

/** 创建空间Embedding配置请求 */
export interface CreateSpaceEmbeddingRequest {
  space_id: string;
  name: string;
  description?: string;
  type: EmbeddingType;
  config: Record<string, any>;
  max_batch_size?: number;
  set_as_default?: boolean;
}

export interface CreateSpaceEmbeddingResponse {
  code: number;
  msg: string;
  data?: SpaceEmbeddingConfig;
}

/** 获取空间Embedding列表请求 */
export interface ListSpaceEmbeddingsRequest {
  space_id: string;
}

export interface ListSpaceEmbeddingsResponse {
  code: number;
  msg: string;
  data?: SpaceEmbeddingConfig[];
}

/** 获取空间默认Embedding请求 */
export interface GetSpaceDefaultEmbeddingRequest {
  space_id: string;
}

export interface GetSpaceDefaultEmbeddingResponse {
  code: number;
  msg: string;
  data?: SpaceEmbeddingConfig;
}

/** 更新空间Embedding配置请求 */
export interface UpdateSpaceEmbeddingRequest {
  space_id: string;
  embedding_id: string;
  name?: string;
  description?: string;
  type?: EmbeddingType;
  config?: Record<string, any>;
  max_batch_size?: number;
}

export interface UpdateSpaceEmbeddingResponse {
  code: number;
  msg: string;
  data?: SpaceEmbeddingConfig;
}

/** 删除空间Embedding配置请求 */
export interface DeleteSpaceEmbeddingRequest {
  space_id: string;
  embedding_id: string;
}

export interface DeleteSpaceEmbeddingResponse {
  code: number;
  msg: string;
}

/** 设置默认Embedding请求 */
export interface SetDefaultSpaceEmbeddingRequest {
  space_id: string;
  embedding_id: string;
}

export interface SetDefaultSpaceEmbeddingResponse {
  code: number;
  msg: string;
}

/** 启用/禁用Embedding请求 */
export interface EnableSpaceEmbeddingRequest {
  space_id: string;
  embedding_id: string;
}

export interface EnableSpaceEmbeddingResponse {
  code: number;
  msg: string;
}

export interface DisableSpaceEmbeddingRequest {
  space_id: string;
  embedding_id: string;
}

export interface DisableSpaceEmbeddingResponse {
  code: number;
  msg: string;
}

// API 函数定义

export const CreateSpaceEmbedding = /*#__PURE__*/createAPI<CreateSpaceEmbeddingRequest, CreateSpaceEmbeddingResponse>({
  url: '/api/embedding/space/create',
  method: 'POST',
  name: 'CreateSpaceEmbedding',
  reqType: 'CreateSpaceEmbeddingRequest',
  reqMapping: {
    body: ['space_id', 'name', 'description', 'type', 'config', 'max_batch_size', 'set_as_default'],
  },
  resType: 'CreateSpaceEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const ListSpaceEmbeddings = /*#__PURE__*/createAPI<ListSpaceEmbeddingsRequest, ListSpaceEmbeddingsResponse>({
  url: '/api/embedding/space/list',
  method: 'POST',
  name: 'ListSpaceEmbeddings',
  reqType: 'ListSpaceEmbeddingsRequest',
  reqMapping: {
    body: ['space_id'],
  },
  resType: 'ListSpaceEmbeddingsResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const GetSpaceDefaultEmbedding = /*#__PURE__*/createAPI<GetSpaceDefaultEmbeddingRequest, GetSpaceDefaultEmbeddingResponse>({
  url: '/api/embedding/space/default',
  method: 'POST',
  name: 'GetSpaceDefaultEmbedding',
  reqType: 'GetSpaceDefaultEmbeddingRequest',
  reqMapping: {
    body: ['space_id'],
  },
  resType: 'GetSpaceDefaultEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const UpdateSpaceEmbedding = /*#__PURE__*/createAPI<UpdateSpaceEmbeddingRequest, UpdateSpaceEmbeddingResponse>({
  url: '/api/embedding/space/update',
  method: 'POST',
  name: 'UpdateSpaceEmbedding',
  reqType: 'UpdateSpaceEmbeddingRequest',
  reqMapping: {
    body: ['space_id', 'embedding_id', 'name', 'description', 'type', 'config', 'max_batch_size'],
  },
  resType: 'UpdateSpaceEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const DeleteSpaceEmbedding = /*#__PURE__*/createAPI<DeleteSpaceEmbeddingRequest, DeleteSpaceEmbeddingResponse>({
  url: '/api/embedding/space/delete',
  method: 'POST',
  name: 'DeleteSpaceEmbedding',
  reqType: 'DeleteSpaceEmbeddingRequest',
  reqMapping: {
    body: ['space_id', 'embedding_id'],
  },
  resType: 'DeleteSpaceEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const SetDefaultSpaceEmbedding = /*#__PURE__*/createAPI<SetDefaultSpaceEmbeddingRequest, SetDefaultSpaceEmbeddingResponse>({
  url: '/api/embedding/space/set-default',
  method: 'POST',
  name: 'SetDefaultSpaceEmbedding',
  reqType: 'SetDefaultSpaceEmbeddingRequest',
  reqMapping: {
    body: ['space_id', 'embedding_id'],
  },
  resType: 'SetDefaultSpaceEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const EnableSpaceEmbedding = /*#__PURE__*/createAPI<EnableSpaceEmbeddingRequest, EnableSpaceEmbeddingResponse>({
  url: '/api/embedding/space/enable',
  method: 'POST',
  name: 'EnableSpaceEmbedding',
  reqType: 'EnableSpaceEmbeddingRequest',
  reqMapping: {
    body: ['space_id', 'embedding_id'],
  },
  resType: 'EnableSpaceEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});

export const DisableSpaceEmbedding = /*#__PURE__*/createAPI<DisableSpaceEmbeddingRequest, DisableSpaceEmbeddingResponse>({
  url: '/api/embedding/space/disable',
  method: 'POST',
  name: 'DisableSpaceEmbedding',
  reqType: 'DisableSpaceEmbeddingRequest',
  reqMapping: {
    body: ['space_id', 'embedding_id'],
  },
  resType: 'DisableSpaceEmbeddingResponse',
  schemaRoot: 'api://schemas/idl_embedding_space_embedding',
  service: 'space_embedding',
});
