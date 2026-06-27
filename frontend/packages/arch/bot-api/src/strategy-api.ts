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

import { axiosInstance, type BotAPIRequestConfig } from './axios';

// Strategy API — hand-written (no IDL codegen), mirrors folder-api.ts pattern.
// JSON field names match backend/api/model/data/strategy/strategy.go exactly.

// ---------- shared sub-models ----------
// NOTE: id fields are int64 on the backend (>2^53), sent as JSON strings via
// `json:"...,string"` tags. All id-type fields MUST be string here.

export interface CapabilitySchema {
  type?: string;
  properties?: Record<string, { type?: string; description?: string }>;
  required?: string[];
}

export interface CapabilityInfo {
  id: string;
  strategy_id: string;
  scenario_id: string;
  type: string;
  ref_id: string;
  ref_sub_id?: string;
  ref_version?: string;
  prompt_content?: string;
  retrieve_config?: string;
  alias_name?: string;
  alias_description?: string;
  sort_order: number;
  schema?: CapabilitySchema;
}

export interface ScenarioInfo {
  id: string;
  strategy_id: string;
  name: string;
  description?: string;
  sort_order: number;
  capabilities?: CapabilityInfo[];
}

export interface StrategyInfo {
  id: string;
  space_id: string;
  app_id?: string;
  creator_id: string;
  name: string;
  description?: string;
  icon_uri?: string;
  status: number;
  version?: string;
  scenarios?: ScenarioInfo[];
}

// ---------- Strategy ----------

export interface CreateStrategyRequest {
  space_id: string;
  name: string;
  description?: string;
  icon_uri?: string;
}

export interface CreateStrategyResponse {
  code: number;
  msg: string;
  data?: StrategyInfo;
}

export interface GetStrategyDetailRequest {
  id: string;
}

export interface GetStrategyDetailResponse {
  code: number;
  msg: string;
  data?: StrategyInfo;
}

export interface UpdateStrategyRequest {
  id: string;
  name?: string;
  description?: string;
  icon_uri?: string;
}

export interface UpdateStrategyResponse {
  code: number;
  msg: string;
}

export interface DeleteStrategyRequest {
  id: string;
}

export interface DeleteStrategyResponse {
  code: number;
  msg: string;
}

export interface PublishStrategyRequest {
  id: string;
  version?: string;
}

export interface PublishStrategyResponse {
  code: number;
  msg: string;
}

// ---------- Scenario ----------

export interface CreateScenarioRequest {
  strategy_id: string;
  name: string;
  description?: string;
  sort_order?: number;
}

export interface CreateScenarioResponse {
  code: number;
  msg: string;
  data?: ScenarioInfo;
}

export interface UpdateScenarioRequest {
  id: string;
  name?: string;
  description?: string;
  sort_order?: number;
}

export interface UpdateScenarioResponse {
  code: number;
  msg: string;
}

export interface DeleteScenarioRequest {
  id: string;
}

export interface DeleteScenarioResponse {
  code: number;
  msg: string;
}

// ---------- Capability ----------

export interface AddCapabilityRequest {
  scenario_id: string;
  strategy_id: string;
  type: string;
  ref_id: string;
  ref_sub_id?: string;
  ref_version?: string;
  prompt_content?: string;
  retrieve_config?: string;
  alias_name?: string;
  alias_description?: string;
  sort_order?: number;
}

export interface AddCapabilityResponse {
  code: number;
  msg: string;
  data?: CapabilityInfo;
}

export interface UpdateCapabilityRequest {
  id: string;
  type?: string;
  ref_id?: string;
  ref_sub_id?: string;
  ref_version?: string;
  prompt_content?: string;
  retrieve_config?: string;
  alias_name?: string;
  alias_description?: string;
  sort_order?: number;
}

export interface PreviewCapabilitySchemaRequest {
  space_id: string;
  type: string;
  ref_id: string;
  ref_sub_id?: string;
  ref_version?: string;
}

export interface PreviewCapabilitySchemaResponse {
  code: number;
  msg: string;
  schema?: CapabilitySchema;
}

export interface UpdateCapabilityResponse {
  code: number;
  msg: string;
}

export interface DeleteCapabilityRequest {
  id: string;
}

export interface DeleteCapabilityResponse {
  code: number;
  msg: string;
}

// ---------- Service class ----------

class StrategyApiService {
  // ---- Strategy ----

  async createStrategy(
    data: CreateStrategyRequest,
    config?: BotAPIRequestConfig,
  ): Promise<CreateStrategyResponse> {
    return await axiosInstance.post('/api/strategy/create', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async getStrategyDetail(
    data: GetStrategyDetailRequest,
    config?: BotAPIRequestConfig,
  ): Promise<GetStrategyDetailResponse> {
    return await axiosInstance.post('/api/strategy/get_detail', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async updateStrategy(
    data: UpdateStrategyRequest,
    config?: BotAPIRequestConfig,
  ): Promise<UpdateStrategyResponse> {
    return await axiosInstance.post('/api/strategy/update', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async deleteStrategy(
    data: DeleteStrategyRequest,
    config?: BotAPIRequestConfig,
  ): Promise<DeleteStrategyResponse> {
    return await axiosInstance.post('/api/strategy/delete', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async publishStrategy(
    data: PublishStrategyRequest,
    config?: BotAPIRequestConfig,
  ): Promise<PublishStrategyResponse> {
    return await axiosInstance.post('/api/strategy/publish', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  // ---- Scenario ----

  async createScenario(
    data: CreateScenarioRequest,
    config?: BotAPIRequestConfig,
  ): Promise<CreateScenarioResponse> {
    return await axiosInstance.post('/api/strategy/scenario/create', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async updateScenario(
    data: UpdateScenarioRequest,
    config?: BotAPIRequestConfig,
  ): Promise<UpdateScenarioResponse> {
    return await axiosInstance.post('/api/strategy/scenario/update', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async deleteScenario(
    data: DeleteScenarioRequest,
    config?: BotAPIRequestConfig,
  ): Promise<DeleteScenarioResponse> {
    return await axiosInstance.post('/api/strategy/scenario/delete', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  // ---- Capability ----

  async addCapability(
    data: AddCapabilityRequest,
    config?: BotAPIRequestConfig,
  ): Promise<AddCapabilityResponse> {
    return await axiosInstance.post('/api/strategy/capability/add', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async updateCapability(
    data: UpdateCapabilityRequest,
    config?: BotAPIRequestConfig,
  ): Promise<UpdateCapabilityResponse> {
    return await axiosInstance.post('/api/strategy/capability/update', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async deleteCapability(
    data: DeleteCapabilityRequest,
    config?: BotAPIRequestConfig,
  ): Promise<DeleteCapabilityResponse> {
    return await axiosInstance.post('/api/strategy/capability/delete', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  async previewCapabilitySchema(
    data: PreviewCapabilitySchemaRequest,
    config?: BotAPIRequestConfig,
  ): Promise<PreviewCapabilitySchemaResponse> {
    return await axiosInstance.post(
      '/api/strategy/capability/preview_schema',
      data,
      {
        headers: { 'Agw-Js-Conv': 'str' },
        ...config,
      },
    );
  }
}

export const strategyApi = new StrategyApiService();
