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

import { type ReactNode } from 'react';

import { act, render, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { DeveloperApi } from '@coze-arch/bot-api';

import { BotDebugChatAreaProviderAdapter } from '../index';

vi.mock('@coze-studio/bot-detail-store/page-runtime', () => ({
  usePageRuntimeStore: (selector: (state: unknown) => unknown) =>
    selector({ setPageRuntimeBotInfo: vi.fn() }),
}));

vi.mock('@coze-studio/bot-detail-store/bot-skill', () => ({
  useBotSkillStore: Object.assign(vi.fn(), {
    getState: vi.fn(() => ({
      onboardingContent: { prologue: 'hello' },
      backgroundImageInfoList: [],
    })),
  }),
}));

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: Object.assign(vi.fn(), {
    getState: vi.fn(() => ({
      name: 'Super Bot',
      icon_url: 'https://example.com/icon.png',
    })),
  }),
}));

vi.mock('@coze-common/chat-hooks', () => ({
  useEventCallback: (fn: unknown) => fn,
}));

vi.mock('@coze-common/chat-core', () => ({
  Scene: { Playground: 'playground' },
}));

vi.mock('@coze-common/chat-area-plugin-resume', () => ({
  ResumePluginRegistry: {},
}));

vi.mock('@coze-common/chat-area-plugin-reasoning', () => ({
  ReasoningPluginRegistry: {},
}));

vi.mock('@coze-common/chat-area-plugin-message-grab', () => ({
  useCreateGrabPlugin: () => ({
    grabEnableUpload: true,
    GrabPlugin: {},
    grabPluginId: 'grab-plugin',
  }),
}));

vi.mock('@coze-agent-ide/chat-area-plugin-debug-common', () => ({
  getDebugCommonPluginRegistry: vi.fn(() => ({})),
}));

vi.mock('@coze-agent-ide/chat-area-provider', () => ({
  BotDebugChatAreaProvider: ({
    children,
    requestToInit,
  }: {
    children: ReactNode;
    requestToInit: () => Promise<unknown>;
  }) => {
    requestToInit();
    return <div data-testid="base-provider">{children}</div>;
  },
  useBotEditorChatBackground: () => ({
    ChatBackgroundPlugin: {},
    showBackground: false,
  }),
}));

vi.mock('@coze-arch/bot-hooks', () => ({
  useMessageReportEvent: vi.fn(),
}));

vi.mock('@coze-arch/bot-api', () => ({
  DeveloperApi: {
    GetMessageList: vi.fn(req =>
      Promise.resolve({
        conversation_id: req.conversation_id || 'latest-conversation',
        cursor: '0',
        hasmore: false,
        message_list: [],
        last_section_id: 'section-1',
      }),
    ),
  },
}));

describe('BotDebugChatAreaProviderAdapter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('can initialize directly with a scoped conversation', async () => {
    render(
      <BotDebugChatAreaProviderAdapter
        botId="bot-1"
        userId="user-1"
        initialConversationId="workflow-session"
        initialScene={4}
      >
        <div>children</div>
      </BotDebugChatAreaProviderAdapter>,
    );

    await waitFor(() =>
      expect(DeveloperApi.GetMessageList).toHaveBeenCalledWith(
        expect.objectContaining({
          bot_id: 'bot-1',
          conversation_id: 'workflow-session',
          scene: 4,
        }),
      ),
    );
    expect(DeveloperApi.GetMessageList).not.toHaveBeenCalledWith(
      expect.objectContaining({
        bot_id: 'bot-1',
        conversation_id: undefined,
      }),
    );
  });

  it('reloads messages for the selected super-agent session', async () => {
    render(
      <BotDebugChatAreaProviderAdapter botId="bot-1" userId="user-1">
        <div>children</div>
      </BotDebugChatAreaProviderAdapter>,
    );

    await waitFor(() =>
      expect(DeveloperApi.GetMessageList).toHaveBeenCalledWith(
        expect.objectContaining({
          bot_id: 'bot-1',
          conversation_id: undefined,
        }),
      ),
    );

    act(() => {
      window.dispatchEvent(
        new CustomEvent('coze:super-agent-session-select', {
        detail: {
          botId: 'bot-1',
          conversationId: 'session-1',
          scene: 9,
        },
      }),
      );
    });

    await waitFor(() =>
      expect(DeveloperApi.GetMessageList).toHaveBeenCalledWith(
        expect.objectContaining({
          bot_id: 'bot-1',
          conversation_id: 'session-1',
          scene: 9,
        }),
      ),
    );
  });
});
