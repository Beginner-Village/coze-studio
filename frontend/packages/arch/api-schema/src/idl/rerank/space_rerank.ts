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

import { createAPI } from './../../api/config';

/** Rerank类型 */
export enum RerankType {
  OpenAI = 'openai',
}

/** Rerank状态 */
export enum RerankStatus {
  Enabled = 1,
  Disabled = 2,
}

/** OpenAI Rerank配置 */
export interface OpenAIRerankConfig {
  base_url: string;
  api_key: string;
  model: string;
}

/** 统一Rerank配置 */
export interface RerankConfig {
  type: RerankType;
  openai_config?: OpenAIRerankConfig;
}

/** 空间Rerank配置 */
export interface SpaceRerankConfig {
  id: string;
  space_id: string;
  name: string;
  description?: string;
  rerank_type: RerankType;
  config?: RerankConfig;
  status: RerankStatus;
  is_default: boolean;
  created_at: number;
  updated_at: number;
}

/** 创建空间Rerank配置请求 */
export interface CreateSpaceRerankRequest {
  space_id: string;
  name: string;
  description?: string;
  type: RerankType;
  config: Record<string, any>;
  set_as_default?: boolean;
}

export interface CreateSpaceRerankResponse {
  code: number;
  msg: string;
  data?: SpaceRerankConfig;
}

/** 获取空间Rerank列表请求 */
export interface ListSpaceReranksRequest {
  space_id: string;
}

export interface ListSpaceReranksResponse {
  code: number;
  msg: string;
  data?: SpaceRerankConfig[];
}

/** 获取空间默认Rerank请求 */
export interface GetSpaceDefaultRerankRequest {
  space_id: string;
}

export interface GetSpaceDefaultRerankResponse {
  code: number;
  msg: string;
  data?: SpaceRerankConfig;
}

/** 更新空间Rerank配置请求 */
export interface UpdateSpaceRerankRequest {
  space_id: string;
  rerank_id: string;
  name?: string;
  description?: string;
  type?: RerankType;
  config?: Record<string, any>;
}

export interface UpdateSpaceRerankResponse {
  code: number;
  msg: string;
  data?: SpaceRerankConfig;
}

/** 删除空间Rerank配置请求 */
export interface DeleteSpaceRerankRequest {
  space_id: string;
  rerank_id: string;
}

export interface DeleteSpaceRerankResponse {
  code: number;
  msg: string;
}

/** 设置默认Rerank请求 */
export interface SetDefaultSpaceRerankRequest {
  space_id: string;
  rerank_id: string;
}

export interface SetDefaultSpaceRerankResponse {
  code: number;
  msg: string;
}

/** 启用/禁用Rerank请求 */
export interface EnableSpaceRerankRequest {
  space_id: string;
  rerank_id: string;
}

export interface EnableSpaceRerankResponse {
  code: number;
  msg: string;
}

export interface DisableSpaceRerankRequest {
  space_id: string;
  rerank_id: string;
}

export interface DisableSpaceRerankResponse {
  code: number;
  msg: string;
}

// API 函数定义

export const CreateSpaceRerank = /*#__PURE__*/createAPI<CreateSpaceRerankRequest, CreateSpaceRerankResponse>({
  url: '/api/rerank/space/create',
  method: 'POST',
  name: 'CreateSpaceRerank',
  reqType: 'CreateSpaceRerankRequest',
  reqMapping: {
    body: ['space_id', 'name', 'description', 'type', 'config', 'set_as_default'],
  },
  resType: 'CreateSpaceRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const ListSpaceReranks = /*#__PURE__*/createAPI<ListSpaceReranksRequest, ListSpaceReranksResponse>({
  url: '/api/rerank/space/list',
  method: 'POST',
  name: 'ListSpaceReranks',
  reqType: 'ListSpaceReranksRequest',
  reqMapping: {
    body: ['space_id'],
  },
  resType: 'ListSpaceReranksResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const GetSpaceDefaultRerank = /*#__PURE__*/createAPI<GetSpaceDefaultRerankRequest, GetSpaceDefaultRerankResponse>({
  url: '/api/rerank/space/default',
  method: 'POST',
  name: 'GetSpaceDefaultRerank',
  reqType: 'GetSpaceDefaultRerankRequest',
  reqMapping: {
    body: ['space_id'],
  },
  resType: 'GetSpaceDefaultRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const UpdateSpaceRerank = /*#__PURE__*/createAPI<UpdateSpaceRerankRequest, UpdateSpaceRerankResponse>({
  url: '/api/rerank/space/update',
  method: 'POST',
  name: 'UpdateSpaceRerank',
  reqType: 'UpdateSpaceRerankRequest',
  reqMapping: {
    body: ['space_id', 'rerank_id', 'name', 'description', 'type', 'config'],
  },
  resType: 'UpdateSpaceRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const DeleteSpaceRerank = /*#__PURE__*/createAPI<DeleteSpaceRerankRequest, DeleteSpaceRerankResponse>({
  url: '/api/rerank/space/delete',
  method: 'POST',
  name: 'DeleteSpaceRerank',
  reqType: 'DeleteSpaceRerankRequest',
  reqMapping: {
    body: ['space_id', 'rerank_id'],
  },
  resType: 'DeleteSpaceRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const SetDefaultSpaceRerank = /*#__PURE__*/createAPI<SetDefaultSpaceRerankRequest, SetDefaultSpaceRerankResponse>({
  url: '/api/rerank/space/set-default',
  method: 'POST',
  name: 'SetDefaultSpaceRerank',
  reqType: 'SetDefaultSpaceRerankRequest',
  reqMapping: {
    body: ['space_id', 'rerank_id'],
  },
  resType: 'SetDefaultSpaceRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const EnableSpaceRerank = /*#__PURE__*/createAPI<EnableSpaceRerankRequest, EnableSpaceRerankResponse>({
  url: '/api/rerank/space/enable',
  method: 'POST',
  name: 'EnableSpaceRerank',
  reqType: 'EnableSpaceRerankRequest',
  reqMapping: {
    body: ['space_id', 'rerank_id'],
  },
  resType: 'EnableSpaceRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});

export const DisableSpaceRerank = /*#__PURE__*/createAPI<DisableSpaceRerankRequest, DisableSpaceRerankResponse>({
  url: '/api/rerank/space/disable',
  method: 'POST',
  name: 'DisableSpaceRerank',
  reqType: 'DisableSpaceRerankRequest',
  reqMapping: {
    body: ['space_id', 'rerank_id'],
  },
  resType: 'DisableSpaceRerankResponse',
  schemaRoot: 'api://schemas/idl_rerank_space_rerank',
  service: 'space_rerank',
});
