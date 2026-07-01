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
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// @coze-arch/bot-api 是巨型 barrel（传递加载 lottie-web 等 UI 依赖，测试环境无 canvas 会崩），
// loop-eval-api 仅用其 getLocalizedErrorMessage，这里 mock 掉以隔离。
vi.mock('@coze-arch/bot-api', () => ({
  getLocalizedErrorMessage: (msg: string) => msg,
}));

import {
  createExperiment,
  listExperimentResults,
  listExperiments,
} from '../loop-eval-api';

function okJson(data: unknown) {
  return { ok: true, json: () => Promise.resolve({ code: 0, data }) };
}

describe('loop-eval-api experiment requests', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('createExperiment 发送 Offline 实验 + 真实可执行 target 参数', async () => {
    fetchMock.mockResolvedValue(okJson({ experiment_id: 'e1' }));

    await createExperiment({
      workspace_id: 'ws1',
      name: 'exp',
      description: 'd',
      eval_set_id: 's1',
      eval_set_version_id: 'v1',
      evaluator_version_ids: [],
      create_eval_target_param: {
        eval_target_type: 4,
        source_target_id: '7412',
      },
      expt_type: 1,
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, opts] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/experiments/submit');
    const body = JSON.parse((opts as RequestInit).body as string);
    // 关键：Offline(1) 才会真实执行；不是 record-only 的 Online(2)
    expect(body.expt_type).toBe(1);
    expect(body.create_eval_target_param.eval_target_type).toBe(4);
    expect(body.create_eval_target_param.source_target_id).toBe('7412');
    expect(body.desc).toBe('d');
  });

  it('listExperimentResults 打逐行结果端点并传 experiment_ids 数组', async () => {
    fetchMock.mockResolvedValue(okJson({ item_results: [], total: 0 }));

    await listExperimentResults({
      workspace_id: 'ws1',
      experiment_id: 'e1',
      page_number: 2,
      page_size: 10,
    });

    const [url, opts] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/experiments/results/batch_get');
    const body = JSON.parse((opts as RequestInit).body as string);
    expect(body.experiment_ids).toEqual(['e1']);
    expect(body.page_number).toBe(2);
    expect(body.page_size).toBe(10);
  });

  it('listExperiments 透传分页参数', async () => {
    fetchMock.mockResolvedValue(okJson({ experiments: [], total: 3 }));

    await listExperiments({
      workspace_id: 'ws1',
      page_number: 2,
      page_size: 20,
    });

    const [url, opts] = fetchMock.mock.calls[0];
    expect(String(url)).toContain('/experiments/list');
    const body = JSON.parse((opts as RequestInit).body as string);
    expect(body.page_number).toBe(2);
  });

  it('接口返回非 0 code 时抛错', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ code: 500, msg: 'boom' }),
    });

    await expect(listExperiments({ workspace_id: 'ws1' })).rejects.toThrow();
  });
});
