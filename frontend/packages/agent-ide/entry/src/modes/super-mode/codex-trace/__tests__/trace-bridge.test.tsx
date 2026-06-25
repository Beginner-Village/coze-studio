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

import { act, render, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { DeveloperApi } from '@coze-arch/bot-api';

import { TraceBridge } from '../trace-bridge';
import { useTraceStore } from '../trace-store';

const bridgeMocks = vi.hoisted(() => ({
  liveMessages: [] as unknown[],
  waitingState: {
    sending: null as unknown,
    waiting: null as unknown,
    responding: null as unknown,
  },
}));

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: (selector: (state: unknown) => unknown) =>
    selector({ botId: 'bot-1', space_id: 'space-1' }),
}));

vi.mock('@coze-arch/bot-api', () => ({
  DeveloperApi: {
    SuperAgentGetTrace: vi.fn(),
  },
}));

vi.mock('@coze-common/chat-area', () => ({
  useChatAreaStoreSet: () => ({
    useMessagesStore: (selector: (state: { messages: unknown[] }) => unknown) =>
      selector({ messages: bridgeMocks.liveMessages }),
    useWaitingStore: (
      selector: (state: typeof bridgeMocks.waitingState) => unknown,
    ) => selector(bridgeMocks.waitingState),
  }),
}));

describe('TraceBridge', () => {
  beforeEach(() => {
    bridgeMocks.liveMessages = [];
    bridgeMocks.waitingState = {
      sending: null,
      waiting: null,
      responding: null,
    };
    useTraceStore.getState().setMessages([]);
    useTraceStore.getState().setPendingReply(false);
    (DeveloperApi.SuperAgentGetTrace as any).mockResolvedValue({
      code: 0,
      data: {
        events: [],
      },
    });
  });

  it('mirrors only live chat messages so session cleanup can clear the panel', async () => {
    bridgeMocks.liveMessages = [
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: 'hello',
        extra_info: {},
      },
    ];

    render(<TraceBridge />);

    await waitFor(() =>
      expect(useTraceStore.getState().messages).toEqual(
        bridgeMocks.liveMessages,
      ),
    );
  });

  it('hides workflow canvas directives from mirrored user messages', async () => {
    bridgeMocks.liveMessages = [
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: [
          '你正在工作流编辑器右侧聊天中,必须真实调用 workflow_canvas_* 工具来操作左侧画布。',
          '当前可用节点类型: 5:开始, 2:大模型',
          '用户需求: 搭一个开始到大模型再到结束的工作流',
        ].join('\n'),
        extra_info: {},
      },
    ];

    render(<TraceBridge />);

    await waitFor(() =>
      expect(useTraceStore.getState().messages).toEqual([
        expect.objectContaining({
          message_id: 'user-1',
          content: '搭一个开始到大模型再到结束的工作流',
        }),
      ]),
    );
  });

  it('mirrors sending and waiting state for immediate assistant feedback', async () => {
    bridgeMocks.waitingState = {
      sending: { message_id: 'local-1' },
      waiting: null,
      responding: null,
    };

    render(<TraceBridge />);

    await waitFor(() =>
      expect(useTraceStore.getState().pendingReply).toBe(true),
    );
  });

  it('merges context compaction trace events from the selected session', async () => {
    bridgeMocks.liveMessages = [
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '继续做 harness',
        extra_info: {},
      },
    ];
    (DeveloperApi.SuperAgentGetTrace as any).mockResolvedValue({
      code: 0,
      data: {
        events: [
          {
            id: 'context:compacted:/workspace/.agent/context-summary.json',
            event: 'context.compacted',
            kind: 'context',
            content: 'Older context was summarized.',
            created_at: 1781900400,
            metadata: {
              trigger: 'history_bytes_exceeded',
              original_messages: '42',
              compacted_messages: '26',
              retained_messages: '16',
              original_bytes: '262144',
              max_bytes: '163840',
              summary_path: '/workspace/.agent/context-summary.json',
            },
          },
        ],
      },
    });

    render(<TraceBridge />);
    act(() => {
      window.dispatchEvent(
        new CustomEvent('coze:super-agent-session-select', {
          detail: {
            botId: 'bot-1',
            conversationId: 'session-1',
          },
        }),
      );
    });

    await waitFor(() =>
      expect(DeveloperApi.SuperAgentGetTrace).toHaveBeenCalledWith({
        conversation_id: 'session-1',
        limit: 100,
      }),
    );
    await waitFor(() =>
      expect(useTraceStore.getState().messages).toEqual(
        expect.arrayContaining([
          expect.objectContaining({
            message_id: 'context:compacted:/workspace/.agent/context-summary.json',
            type: 'context_compacted',
            extra_info: expect.objectContaining({
              event: 'context.compacted',
              original_messages: '42',
              summary_path: '/workspace/.agent/context-summary.json',
            }),
          }),
        ]),
      ),
    );
  });
});
