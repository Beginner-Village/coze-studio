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

describe('DeveloperApi super-agent workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (axiosInstance.request as any).mockResolvedValue({ code: 0 });
  });

  it('passes agent_id through workspace list, read, download, write, move, mkdir, stat, search, edit and patch calls', async () => {
    await (DeveloperApi as any).SuperAgentListWorkspaceFiles({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/workspace',
      recursive: true,
    });

    await (DeveloperApi as any).SuperAgentReadWorkspaceFile({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/skills/pdf/SKILL.md',
    });

    await (DeveloperApi as any).SuperAgentDownloadWorkspaceFile({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/outputs/report.html',
    });

    await (DeveloperApi as any).SuperAgentWriteWorkspaceFile({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/workspace/app.ts',
      content: 'console.log("ok");',
      is_base64: false,
      encoding: 'base64',
    });

    await (DeveloperApi as any).SuperAgentMoveWorkspaceFile({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/workspace/draft.txt',
      target_path: '/uploads/final.txt',
    });

    await (DeveloperApi as any).SuperAgentMoveWorkspaceFile({
      space_id: 'space-1',
      agent_id: 'agent-1',
      from_path: '/workspace/manifest-draft.txt',
      to_path: '/outputs/manifest-final.txt',
    });

    await (DeveloperApi as any).SuperAgentCreateWorkspaceDirectory({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/workspace/reports',
    });

    await (DeveloperApi as any).SuperAgentStatWorkspacePath({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/skills/pdf/SKILL.md',
    });

    await (DeveloperApi as any).SuperAgentGrepWorkspace({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/skills',
      pattern: 'PDF',
      case_sensitive: false,
      include: ['*.md'],
      exclude: ['draft*'],
    });

    await (DeveloperApi as any).SuperAgentGlobWorkspace({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/skills',
      pattern: 'SKILL.md',
      limit: 10,
    });

    await (DeveloperApi as any).SuperAgentEditWorkspaceFile({
      space_id: 'space-1',
      agent_id: 'agent-1',
      path: '/workspace/app.ts',
      old_string: 'oldCall()',
      new_string: 'newCall()',
      replace_all: true,
    });

    await (DeveloperApi as any).SuperAgentApplyWorkspacePatch({
      space_id: 'space-1',
      agent_id: 'agent-1',
      workdir: '/workspace',
      work_dir: '/outputs',
      patch: '*** Begin Patch\n*** Add File: app.ts\n+ok\n*** End Patch\n',
    });

    expect(axiosInstance.request).toHaveBeenNthCalledWith(1, {
      url: '/api/super-agent/workspace/list',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/workspace',
        recursive: true,
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(2, {
      url: '/api/super-agent/workspace/read',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/skills/pdf/SKILL.md',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(3, {
      url: '/api/super-agent/workspace/download',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/outputs/report.html',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(4, {
      url: '/api/super-agent/workspace/write',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/workspace/app.ts',
        content: 'console.log("ok");',
        is_base64: false,
        encoding: 'base64',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(5, {
      url: '/api/super-agent/workspace/move',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/workspace/draft.txt',
        from_path: undefined,
        target_path: '/uploads/final.txt',
        to_path: undefined,
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(6, {
      url: '/api/super-agent/workspace/move',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: undefined,
        from_path: '/workspace/manifest-draft.txt',
        target_path: undefined,
        to_path: '/outputs/manifest-final.txt',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(7, {
      url: '/api/super-agent/workspace/mkdir',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/workspace/reports',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(8, {
      url: '/api/super-agent/workspace/stat',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/skills/pdf/SKILL.md',
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(9, {
      url: '/api/super-agent/workspace/grep',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/skills',
        pattern: 'PDF',
        case_sensitive: false,
        include: ['*.md'],
        exclude: ['draft*'],
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(10, {
      url: '/api/super-agent/workspace/glob',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/skills',
        pattern: 'SKILL.md',
        limit: 10,
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(11, {
      url: '/api/super-agent/workspace/edit',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        path: '/workspace/app.ts',
        old_string: 'oldCall()',
        new_string: 'newCall()',
        replace_all: true,
        connector_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(12, {
      url: '/api/super-agent/workspace/patch',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        workdir: '/workspace',
        work_dir: '/outputs',
        patch: '*** Begin Patch\n*** Add File: app.ts\n+ok\n*** End Patch\n',
        connector_id: undefined,
      },
    });
  });

  it('routes harness state calls to the super-agent App Server endpoint', async () => {
    await (DeveloperApi as any).SuperAgentGetHarnessState({
      space_id: 'space-1',
      agent_id: 'agent-1',
      connector_id: 'connector-1',
    });

    expect(axiosInstance.request).toHaveBeenCalledWith({
      url: '/api/super-agent/harness/state',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        connector_id: 'connector-1',
      },
    });
  });

  it('routes session create, list, rename and delete calls to the super-agent App Server endpoints', async () => {
    await (DeveloperApi as any).SuperAgentCreateSession({
      space_id: 'space-1',
      bot_id: 'agent-1',
      title: '新会话',
    });

    await (DeveloperApi as any).SuperAgentListSessions({
      space_id: 'space-1',
      bot_id: 'agent-1',
      page: 1,
      page_size: 20,
    });

    await (DeveloperApi as any).SuperAgentRenameSession({
      conversation_id: 'conv-1',
      title: '竞品 Codex 对比',
    });

    await (DeveloperApi as any).SuperAgentDeleteSession({
      conversation_id: 'conv-1',
    });

    expect(axiosInstance.request).toHaveBeenNthCalledWith(1, {
      url: '/api/super-agent/sessions/create',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: undefined,
        bot_id: 'agent-1',
        connector_id: undefined,
        title: '新会话',
        user_id: undefined,
        client_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(2, {
      url: '/api/super-agent/sessions/list',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: undefined,
        bot_id: 'agent-1',
        connector_id: undefined,
        user_id: undefined,
        client_id: undefined,
        page: 1,
        page_size: 20,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(3, {
      url: '/api/super-agent/sessions/rename',
      method: 'POST',
      data: {
        conversation_id: 'conv-1',
        title: '竞品 Codex 对比',
        user_id: undefined,
        client_id: undefined,
      },
    });
    expect(axiosInstance.request).toHaveBeenNthCalledWith(4, {
      url: '/api/super-agent/sessions/delete',
      method: 'POST',
      data: {
        conversation_id: 'conv-1',
        user_id: undefined,
        client_id: undefined,
      },
    });
  });

  it('routes harness plan updates to the super-agent App Server endpoint', async () => {
    await (DeveloperApi as any).SuperAgentUpdateHarnessPlan({
      space_id: 'space-1',
      agent_id: 'agent-1',
      plan: [
        { content: '梳理需求', status: 'completed' },
        { content: '实现接口', status: 'in_progress' },
      ],
    });

    expect(axiosInstance.request).toHaveBeenCalledWith({
      url: '/api/super-agent/harness/plan',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        plan: [
          { content: '梳理需求', status: 'completed' },
          { content: '实现接口', status: 'in_progress' },
        ],
        connector_id: undefined,
      },
    });
  });

  it('preserves harness plan items alias for App Server clients', async () => {
    await (DeveloperApi as any).SuperAgentUpdateHarnessPlan({
      space_id: 'space-1',
      agent_id: 'agent-1',
      items: [{ content: '按 manifest 字段更新计划', status: 'completed' }],
    });

    expect(axiosInstance.request).toHaveBeenCalledWith({
      url: '/api/super-agent/harness/plan',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        plan: undefined,
        items: [{ content: '按 manifest 字段更新计划', status: 'completed' }],
        connector_id: undefined,
      },
    });
  });

  it('routes sandbox exec calls to the super-agent App Server endpoint', async () => {
    await (DeveloperApi as any).SuperAgentExecSandbox({
      space_id: 'space-1',
      agent_id: 'agent-1',
      command: 'npm test',
      workdir: '/workspace/project',
      timeout_sec: 12,
    });

    expect(axiosInstance.request).toHaveBeenCalledWith({
      url: '/api/super-agent/sandbox/exec',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        command: 'npm test',
        workdir: '/workspace/project',
        timeout_sec: 12,
        connector_id: undefined,
      },
    });
  });

  it('passes both workdir and work_dir through sandbox exec calls', async () => {
    await (DeveloperApi as any).SuperAgentExecSandbox({
      space_id: 'space-1',
      agent_id: 'agent-1',
      command: 'pwd',
      workdir: '/workspace',
      work_dir: '/outputs',
      timeout_sec: 10,
    });

    expect(axiosInstance.request).toHaveBeenCalledWith({
      url: '/api/super-agent/sandbox/exec',
      method: 'POST',
      data: {
        space_id: 'space-1',
        agent_id: 'agent-1',
        bot_id: undefined,
        command: 'pwd',
        workdir: '/workspace',
        work_dir: '/outputs',
        timeout_sec: 10,
        connector_id: undefined,
      },
    });
  });
});
