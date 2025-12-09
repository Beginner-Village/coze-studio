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

import AopApiService from '@coze-arch/idl/aop_api';

import { axiosInstance, type BotAPIRequestConfig } from './axios';

// 转换 sassWorkspaceId：当值为特定ID时转换为 'dev'
function transformSassWorkspaceId(data: Record<string, unknown>): Record<string, unknown> {
  if (data && typeof data === 'object') {
    const transformed = { ...data };
    if (transformed.sassWorkspaceId === '7533521629687578624') {
      transformed.sassWorkspaceId = 'dev';
    }
    return transformed;
  }
  return data;
}

export const aopApi = new AopApiService<BotAPIRequestConfig>({
  request: (params, config = {}) => {
    // 转换请求体中的 sassWorkspaceId
    const transformedData = transformSassWorkspaceId(params.data);
    params.data = {
      header: {
        version: '1.0.0',
        commType: '',
        transCode: params.url.split('/').pop(),
        srcIP: '',
        channelType: '',
        channelNo: '',
        srcSystemId: '',
        srcSystemDevId: '',
        charset: 'json',
        respFormat: 'UTF-8',
        reqNo: '',
        securityFlag: '',
        macCode: '',
        macValue: '',
        transTime: Date.now(),
        globalFlowNo: '',
      },
      body: transformedData,
    };
    return axiosInstance.request({
      ...params,
      ...config,
      headers: {
        ...params.headers,
        ...config?.headers,
      },
    });
  },
});
