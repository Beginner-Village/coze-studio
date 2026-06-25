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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { DeveloperApi } from '@coze-arch/bot-api';

import { SuperSessionSidebar } from '../super-session-sidebar';

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: (selector: (state: unknown) => unknown) =>
    selector({ botId: 'bot-1', space_id: 'space-1' }),
}));

vi.mock('@coze-arch/bot-api', () => ({
  DeveloperApi: {
    SuperAgentCreateSession: vi.fn(),
    SuperAgentDeleteSession: vi.fn(),
    SuperAgentListSessions: vi.fn(),
    SuperAgentRenameSession: vi.fn(),
  },
}));

vi.mock('@coze-arch/coze-design', () => ({
  Modal: {
    confirm: vi.fn(({ onOk }) => onOk?.()),
  },
  Toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

describe('SuperSessionSidebar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    try {
      window.sessionStorage.clear();
    } catch {
      /* noop */
    }
    (DeveloperApi.SuperAgentListSessions as any).mockResolvedValue({
      data: {
        sessions: [
          {
            session_id: 'session-1',
            title: 'Deep research',
            renamable: true,
            updated_at: 1760954400,
            scene: 9,
          },
        ],
      },
    });
    (DeveloperApi.SuperAgentRenameSession as any).mockResolvedValue({
      data: {
        session: {
          session_id: 'session-1',
          title: '竞品 Codex 对比',
        },
      },
    });
    (DeveloperApi.SuperAgentCreateSession as any).mockResolvedValue({
      data: {
        session: {
          session_id: 'session-new',
          title: '新会话',
          renamable: true,
          scene: 9,
        },
      },
    });
    (DeveloperApi as any).SuperAgentDeleteSession.mockResolvedValue({
      code: 0,
      data: {
        session_id: 'session-1',
        conversation_id: 'session-1',
      },
    });
  });

  it('loads super-agent sessions for the current bot', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');
    render(<SuperSessionSidebar />);

    await waitFor(() =>
      expect(DeveloperApi.SuperAgentListSessions).toHaveBeenCalledWith({
        space_id: 'space-1',
        bot_id: 'bot-1',
        page: 1,
        page_size: 30,
      }),
    );
    await waitFor(() =>
      expect(screen.getAllByText('Deep research').length).toBeGreaterThan(0),
    );
    await waitFor(() =>
      expect(dispatchSpy).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'coze:super-agent-session-select',
          detail: expect.objectContaining({
            botId: 'bot-1',
            conversationId: 'session-1',
            scene: 9,
          }),
        }),
      ),
    );
    expect(screen.getByText('最近会话')).toBeTruthy();
  });

  it('renames a renamable session inline', async () => {
    render(<SuperSessionSidebar />);

    await screen.findByText('Deep research');
    fireEvent.click(screen.getByLabelText('重命名会话'));
    fireEvent.change(screen.getByRole('textbox'), {
      target: { value: '竞品 Codex 对比' },
    });
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' });

    await waitFor(() =>
      expect(DeveloperApi.SuperAgentRenameSession).toHaveBeenCalledWith({
        conversation_id: 'session-1',
        title: '竞品 Codex 对比',
      }),
    );
    await waitFor(() =>
      expect(screen.getAllByText('竞品 Codex 对比').length).toBeGreaterThan(0),
    );
  });

  it('creates a durable session before selecting a new session', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');
    render(<SuperSessionSidebar />);

    await screen.findByText('Deep research');
    fireEvent.click(screen.getByLabelText('新会话'));

    await waitFor(() =>
      expect(DeveloperApi.SuperAgentCreateSession).toHaveBeenCalledWith({
        space_id: 'space-1',
        bot_id: 'bot-1',
        title: '新会话',
      }),
    );
    await waitFor(() =>
      expect(screen.getAllByText('新会话').length).toBeGreaterThan(0),
    );
    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'coze:super-agent-session-select',
        detail: expect.objectContaining({
          botId: 'bot-1',
          conversationId: 'session-new',
          scene: 9,
        }),
      }),
    );
  });

  it('dispatches session selection changes to the chat provider', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent');
    render(<SuperSessionSidebar />);

    await screen.findByText('Deep research');
    const sessionTitle = screen
      .getAllByText('Deep research')
      .find(node => node.closest('[role="button"]'));
    fireEvent.click(sessionTitle!);

    expect(dispatchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'coze:super-agent-session-select',
        detail: expect.objectContaining({
          botId: 'bot-1',
          conversationId: 'session-1',
          scene: 9,
        }),
      }),
    );

    expect(DeveloperApi.SuperAgentCreateSession).not.toHaveBeenCalled();
  });

  it('deletes a renamable session from the session list', async () => {
    render(<SuperSessionSidebar />);

    await screen.findByText('Deep research');
    fireEvent.click(screen.getByLabelText('删除会话'));

    await waitFor(() =>
      expect(
        (DeveloperApi as any).SuperAgentDeleteSession,
      ).toHaveBeenCalledWith({
        conversation_id: 'session-1',
      }),
    );
    await waitFor(() =>
      expect(screen.queryByText('Deep research')).toBeNull(),
    );
  });

  it('batch-deletes multiple selected sessions', async () => {
    (DeveloperApi.SuperAgentListSessions as any).mockResolvedValue({
      data: {
        sessions: [
          {
            session_id: 'session-1',
            title: 'Deep research',
            renamable: true,
            updated_at: 1760954400,
            scene: 9,
          },
          {
            session_id: 'session-2',
            title: 'Code refactor',
            renamable: true,
            updated_at: 1760954500,
            scene: 9,
          },
        ],
      },
    });
    render(<SuperSessionSidebar />);

    await screen.findByText('Deep research');
    // Enter multi-select mode.
    fireEvent.click(screen.getByLabelText('多选'));
    // Select all, then batch delete.
    fireEvent.click(screen.getByText('全选'));
    fireEvent.click(screen.getByText('删除'));

    await waitFor(() =>
      expect(
        (DeveloperApi as any).SuperAgentDeleteSession,
      ).toHaveBeenCalledWith({ conversation_id: 'session-1' }),
    );
    expect(
      (DeveloperApi as any).SuperAgentDeleteSession,
    ).toHaveBeenCalledWith({ conversation_id: 'session-2' });
    await waitFor(() =>
      expect(screen.queryByText('Code refactor')).toBeNull(),
    );
  });
});
