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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';

/* eslint-disable @typescript-eslint/naming-convention --
 * Mocked third-party component exports keep their production names.
 */
const developerApiMock = {
  ListSandboxFiles: vi.fn(),
  ReadSandboxFile: vi.fn(),
  UploadSandboxFile: vi.fn(),
  DeleteSandboxFile: vi.fn(),
  SuperAgentGetHarnessState: vi.fn(),
  SuperAgentListWorkspaceFiles: vi.fn(),
  SuperAgentReadWorkspaceFile: vi.fn(),
  SuperAgentUploadWorkspaceFile: vi.fn(),
  SuperAgentDeleteWorkspaceFile: vi.fn(),
  SuperAgentWriteWorkspaceFile: vi.fn(),
};

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: (selector: (state: unknown) => unknown) =>
    selector({ botId: 'agent-1', space_id: 'space-1' }),
}));

vi.mock('@coze-common/chat-uikit', () => ({
  LazyCozeMdBox: ({ markDown }: { markDown: string }) => <div>{markDown}</div>,
}));

vi.mock('@coze-arch/bot-api', () => ({
  DeveloperApi: developerApiMock,
}));

vi.mock('@coze-arch/coze-design', () => ({
  Button: ({
    children,
    onClick,
  }: {
    children?: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  Spin: () => <div data-testid="spin" />,
  Input: ({
    value,
    onChange,
    placeholder,
  }: {
    value?: string;
    onChange?: (v: string) => void;
    placeholder?: string;
  }) => (
    <input
      aria-label="filename"
      placeholder={placeholder}
      value={value}
      onChange={e => onChange?.(e.target.value)}
    />
  ),
  Modal: ({
    visible,
    children,
    onOk,
  }: {
    visible?: boolean;
    children?: React.ReactNode;
    onOk?: () => void;
  }) =>
    visible ? (
      <div>
        {children}
        <button type="button" onClick={onOk}>
          创建
        </button>
      </div>
    ) : null,
  Toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => (
      <span>{children}</span>
    ),
  },
}));

const workspaceFiles = [
  {
    name: '.agent',
    path: '/workspace/.agent',
    is_dir: true,
    size: 0,
  },
  {
    name: 'hello.md',
    path: '/workspace/hello.md',
    is_dir: false,
    size: 18,
  },
];

describe('SandboxWorkspace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    developerApiMock.ListSandboxFiles.mockResolvedValue({
      code: 0,
      data: { files: workspaceFiles },
    });
    developerApiMock.SuperAgentListWorkspaceFiles.mockResolvedValue({
      code: 0,
      data: { files: workspaceFiles },
    });
    developerApiMock.ReadSandboxFile.mockResolvedValue({
      code: 0,
      data: {
        content: '# hello',
        is_binary: false,
      },
    });
    developerApiMock.SuperAgentReadWorkspaceFile.mockResolvedValue({
      code: 0,
      data: {
        content: '# hello',
        is_binary: false,
      },
    });
    developerApiMock.DeleteSandboxFile.mockResolvedValue({ code: 0 });
    developerApiMock.UploadSandboxFile.mockResolvedValue({ code: 0 });
    developerApiMock.SuperAgentDeleteWorkspaceFile.mockResolvedValue({
      code: 0,
    });
    developerApiMock.SuperAgentUploadWorkspaceFile.mockResolvedValue({
      code: 0,
    });
    developerApiMock.SuperAgentWriteWorkspaceFile.mockResolvedValue({
      code: 0,
    });
    developerApiMock.SuperAgentGetHarnessState.mockResolvedValue({
      code: 0,
      data: {
        plan: {
          exists: true,
          content: JSON.stringify([
            { content: 'inspect', status: 'completed' },
            { content: 'verify', status: 'in_progress' },
          ]),
        },
        tool_outputs: {
          root: '/workspace/.agent/tooloutputs',
          files: [
            {
              name: 'call-1.json',
              path: '/workspace/.agent/tooloutputs/call-1.json',
              is_dir: false,
              size: 12,
            },
            {
              name: 'call-2.json',
              path: '/workspace/.agent/tooloutputs/call-2.json',
              is_dir: false,
              size: 18,
            },
          ],
        },
        runtime_skills: {
          root: '/skills',
          skills: [
            {
              name: 'report-kit',
              path: '/skills/report-kit',
              entry_path: '/skills/report-kit/SKILL.md',
              standard: true,
              file_paths: ['SKILL.md', 'scripts/run.sh'],
            },
          ],
        },
      },
    });
  });

  it('loads the super-agent workspace tree with harness state and skill root', async () => {
    const { SandboxWorkspace } = await import('../sandbox-workspace');

    render(<SandboxWorkspace />);

    await waitFor(() => {
      expect(
        developerApiMock.SuperAgentListWorkspaceFiles,
      ).toHaveBeenCalledWith({
        space_id: 'space-1',
        bot_id: 'agent-1',
        path: '/workspace',
      });
    });
    expect(developerApiMock.SuperAgentGetHarnessState).toHaveBeenCalledWith({
      space_id: 'space-1',
      bot_id: 'agent-1',
    });

    expect(await screen.findByText('hello.md')).toBeTruthy();
    expect(screen.getByText('沙箱工作区')).toBeTruthy();
    expect(screen.getByText('文件')).toBeTruthy();
    expect(screen.getByText('产物')).toBeTruthy();
    expect(screen.getByText('上传区')).toBeTruthy();
    expect(screen.queryByText('技能')).toBeNull();
    expect(screen.getByText('计划 1/2 完成 · 1 进行中')).toBeTruthy();
    expect(screen.getByText('工具输出')).toBeTruthy();
    // real per-root counts (workspace tree has 2 entries), no hardcoded sample tabs
    await waitFor(() =>
      expect(screen.getAllByText('2').length).toBeGreaterThan(0),
    );
    expect(screen.queryByText('primes.py')).toBeNull();
    expect(screen.queryByText('glm5_model_comparison.md')).toBeNull();
    // empty editor state until a real file is opened
    expect(screen.getByText('选择左侧文件查看')).toBeTruthy();
  });

  it('creates a new empty file via SuperAgentWriteWorkspaceFile', async () => {
    const { SandboxWorkspace } = await import('../sandbox-workspace');

    render(<SandboxWorkspace />);

    fireEvent.click(await screen.findByTitle('新建'));
    fireEvent.change(screen.getByLabelText('filename'), {
      target: { value: 'notes.md' },
    });
    fireEvent.click(screen.getByText('创建'));

    await waitFor(() => {
      expect(
        developerApiMock.SuperAgentWriteWorkspaceFile,
      ).toHaveBeenCalledWith({
        space_id: 'space-1',
        bot_id: 'agent-1',
        path: '/workspace/notes.md',
        content: '',
        is_base64: false,
      });
    });
  });

  it('switches roots through the redesigned segment control', async () => {
    const { SandboxWorkspace } = await import('../sandbox-workspace');

    render(<SandboxWorkspace />);

    fireEvent.click(screen.getByText('产物'));

    await waitFor(() => {
      expect(
        developerApiMock.SuperAgentListWorkspaceFiles,
      ).toHaveBeenCalledWith({
        space_id: 'space-1',
        bot_id: 'agent-1',
        path: '/outputs',
      });
    });
  });

  it('opens a selected file in the inline preview area', async () => {
    const { SandboxWorkspace } = await import('../sandbox-workspace');

    render(<SandboxWorkspace />);

    fireEvent.click(await screen.findByText('hello.md'));

    await waitFor(() => {
      expect(developerApiMock.SuperAgentReadWorkspaceFile).toHaveBeenCalledWith(
        {
          space_id: 'space-1',
          bot_id: 'agent-1',
          path: '/workspace/hello.md',
        },
      );
    });
    expect(await screen.findByText('# hello')).toBeTruthy();
  });

  it('deletes files through the original sandbox endpoint', async () => {
    const { SandboxWorkspace } = await import('../sandbox-workspace');

    render(<SandboxWorkspace />);

    expect(await screen.findByText('hello.md')).toBeTruthy();
    fireEvent.click(screen.getAllByTitle('删除')[1]);

    await waitFor(() => {
      expect(
        developerApiMock.SuperAgentDeleteWorkspaceFile,
      ).toHaveBeenCalledWith({
        space_id: 'space-1',
        bot_id: 'agent-1',
        path: '/workspace/hello.md',
      });
    });
  });
});
