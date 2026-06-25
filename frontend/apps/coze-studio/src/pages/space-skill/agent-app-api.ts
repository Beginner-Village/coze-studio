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

import { axiosInstance } from '@coze-arch/bot-api';

export interface PublishParams {
  bot_id: string;
  space_id: string;
  name: string;
  version: string;
}

export interface PublishResult {
  code: number;
  msg: string;
  data: {
    product_id: string;
    version: string;
  };
}

export interface BuildStatusParams {
  product_id: string;
  space_id?: string;
  version?: string;
}

export interface BuildStatusResult {
  code: number;
  msg: string;
  data: {
    product_id: string;
    version: string;
    build_status: string;
  };
}

export interface RecruitParams {
  product_id: string;
  space_id: string;
}

export interface RecruitResult {
  code: number;
  msg: string;
  data: {
    shadow_agent_id: string;
  };
}

const post = <T>(url: string, data: Record<string, unknown>): Promise<T> =>
  axiosInstance.request({
    url,
    method: 'POST',
    data,
    withCredentials: true,
  }) as unknown as Promise<T>;

export const agentAppApi = {
  publish: (params: PublishParams): Promise<PublishResult> =>
    post<PublishResult>('/api/super-agent/agent-app/publish', {
      agent_id: params.bot_id,
      space_id: params.space_id,
      name: params.name,
      version: params.version,
    }),

  buildStatus: (params: BuildStatusParams): Promise<BuildStatusResult> =>
    post<BuildStatusResult>('/api/super-agent/agent-app/build-status', {
      product_id: params.product_id,
      space_id: params.space_id,
      version: params.version,
    }),

  recruit: (params: RecruitParams): Promise<RecruitResult> =>
    post<RecruitResult>('/api/super-agent/agent-app/recruit', {
      product_id: params.product_id,
      space_id: params.space_id,
    }),
};
