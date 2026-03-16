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

import type {
  ListSpansRequest,
  ListSpansResponse,
  GetTraceRequest,
  GetTraceResponse,
} from '@coze-arch/idl/stone_cozeloop_observability_api';

const BASE = '/loop/api/observability/v1';

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const resp = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    },
    ...options,
  });
  if (!resp.ok) {
    throw new Error(`API error: ${resp.status} ${resp.statusText}`);
  }
  const json = await resp.json();
  // CozeLoop API wraps response in { code, msg, data }
  if (json.code !== undefined && json.code !== 0) {
    throw new Error(json.msg || `API error code: ${json.code}`);
  }
  return json.data ?? json;
}

function toQueryString(params: Record<string, unknown>): string {
  const parts: string[] = [];
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      parts.push(
        `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`,
      );
    }
  }
  return parts.length > 0 ? `?${parts.join('&')}` : '';
}

export async function listSpans(
  req: ListSpansRequest,
): Promise<ListSpansResponse> {
  return request<ListSpansResponse>(`${BASE}/spans/list`, {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export async function getTrace(
  req: GetTraceRequest,
): Promise<GetTraceResponse> {
  const { trace_id, ...params } = req;
  const qs = toQueryString(params);
  return request<GetTraceResponse>(`${BASE}/traces/${trace_id}${qs}`, {
    method: 'GET',
  });
}
