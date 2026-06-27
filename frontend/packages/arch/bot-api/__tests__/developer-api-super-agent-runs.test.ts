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

import { vi } from 'vitest';

import { DeveloperApi } from '../src/developer-api';
import { axiosInstance } from '../src/axios';
import type { developer_api } from '../src/idl/developer_api';

vi.mock('../src/axios', () => ({
  axiosInstance: {
    request: vi.fn(),
  },
}));

describe('DeveloperApi super-agent runs', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (axiosInstance.request as any).mockResolvedValue({ code: 0 });
  });

  it('routes run status and list calls to the super-agent App Server endpoints', async () => {
    await (DeveloperApi as any).SuperAgentGetRun({
      space_id: 'space-1',
      agent_id: 'agent-1',
      run_id: 'run-1',
      user_id: 'external-user',
      client_id: 'client-1',
    });

    await (DeveloperApi as any).SuperAgentListRuns({
      space_id: 'space-1',
      agent_id: 'agent-1',
      conversation_id: 'conversation-1',
      limit: 20,
      order_by: 'desc',
      before_id: '100',
      user_id: 'external-user',
      client_id: 'client-1',
    });

    expect(axiosInstance.request).toHaveBeenNthCalledWith(1, {
      url: '/api/super-agent/runs/get',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        run_id: 'run-1',
        user_id: 'external-user',
        client_id: 'client-1',
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(2, {
      url: '/api/super-agent/runs/list',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        conversation_id: 'conversation-1',
        limit: 20,
        order_by: 'desc',
        before_id: '100',
        after_id: undefined,
        user_id: 'external-user',
        client_id: 'client-1',
      },
    });
  });

  it('routes trace replay calls to the super-agent App Server endpoint', async () => {
    await (DeveloperApi as any).SuperAgentGetTrace({
      conversation_id: 'conversation-1',
      run_id: 'run-1',
      limit: 50,
    });

    expect(axiosInstance.request).toHaveBeenCalledWith({
      url: '/api/super-agent/traces/get',
      method: 'POST',
      data: {
        conversation_id: 'conversation-1',
        run_id: 'run-1',
        limit: 50,
      },
    });
  });

  it('types persisted run metadata returned by super-agent run status', () => {
    const state: developer_api.SuperAgentRunState = {
      run_id: 'run-1',
      conversation_id: 'conversation-1',
      agent_id: 'agent-1',
      status: 'completed',
      active: false,
      error: {
        code: '500',
        msg: 'tool failed',
      },
      created_at: '1000',
      updated_at: '2000',
      completed_at: '2500',
      failed_at: '0',
    };

    expect(state.conversation_id).toBe('conversation-1');
    expect(state.completed_at).toBe('2500');
  });
});
