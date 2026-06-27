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

vi.mock('../src/axios', () => ({
  axiosInstance: {
    request: vi.fn(),
  },
}));

describe('DeveloperApi super-agent artifacts', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (axiosInstance.request as any).mockResolvedValue({ code: 0 });
  });

  it('routes artifact list, download, delete and move calls to the super-agent App Server endpoints', async () => {
    await (DeveloperApi as any).SuperAgentListArtifacts({
      space_id: 'space-1',
      bot_id: 'bot-1',
      path: '/outputs',
      limit: 50,
    });

    await (DeveloperApi as any).SuperAgentDownloadArtifact({
      space_id: 'space-1',
      bot_id: 'bot-1',
      artifact_id: '/outputs/report.html',
    });

    await (DeveloperApi as any).SuperAgentMoveArtifact({
      space_id: 'space-1',
      bot_id: 'bot-1',
      artifact_id: '/outputs/report.html',
      target_path: '/outputs/final/report.html',
    });

    await (DeveloperApi as any).SuperAgentDeleteArtifact({
      space_id: 'space-1',
      bot_id: 'bot-1',
      artifact_id: '/outputs/report.html',
    });

    expect(axiosInstance.request).toHaveBeenNthCalledWith(1, {
      url: '/api/super-agent/artifacts/list',
      method: 'POST',
      data: {
        space_id: 'space-1',
        bot_id: 'bot-1',
        path: '/outputs',
        limit: 50,
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(2, {
      url: '/api/super-agent/artifacts/download',
      method: 'POST',
      data: {
        space_id: 'space-1',
        bot_id: 'bot-1',
        artifact_id: '/outputs/report.html',
        path: undefined,
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(3, {
      url: '/api/super-agent/artifacts/move',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: undefined,
        bot_id: 'bot-1',
        artifact_id: '/outputs/report.html',
        path: undefined,
        target_path: '/outputs/final/report.html',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(4, {
      url: '/api/super-agent/artifacts/delete',
      method: 'POST',
      data: {
        space_id: 'space-1',
        bot_id: 'bot-1',
        artifact_id: '/outputs/report.html',
        path: undefined,
        connector_id: undefined,
      },
    });
  });
});
