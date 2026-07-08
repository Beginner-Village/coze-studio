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

import { axiosInstance } from './axios';

let cache: boolean | undefined;
let inflight: Promise<boolean> | undefined;

/**
 * 拉取后端「超级体(FinMallClaw)入口是否启用」的运行时配置。
 *
 * 值由后端 `GET /api/super-agent/ui-config` 决定，联动 `SANDBOX_ENABLED`：
 * 运维只改后端一个 env，前端无需重新构建即可显隐超级体相关入口。
 *
 * 结果进程内缓存，多处调用只发一次请求；接口不可用时降级为 false（保守隐藏入口）。
 */
export async function fetchSuperAgentEnabled(): Promise<boolean> {
  if (cache !== undefined) {
    return cache;
  }
  if (!inflight) {
    const requestOptions: Parameters<typeof axiosInstance.request>[0] = {
      url: '/api/super-agent/ui-config',
      method: 'GET',
      __disableErrorToast: true,
    };
    inflight = axiosInstance
      .request(requestOptions)
      .then((body: unknown) => {
        const enabled = Boolean(
          (body as { data?: { super_agent_enabled?: boolean } })?.data
            ?.super_agent_enabled,
        );
        cache = enabled;
        return enabled;
      })
      .catch(() => false)
      .finally(() => {
        inflight = undefined;
      });
  }
  return inflight;
}

/**
 * 同步读取已缓存的开关值（未拉取过时为 undefined）。
 * 用于组件初始化时避免「先显示后隐藏」的闪烁。
 */
export function getSuperAgentEnabledCache(): boolean | undefined {
  return cache;
}
