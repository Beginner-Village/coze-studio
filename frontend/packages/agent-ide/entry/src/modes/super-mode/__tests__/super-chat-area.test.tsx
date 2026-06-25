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

import { resolve } from 'node:path';
import { readFileSync } from 'node:fs';

import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

import { SuperChatArea } from '../super-chat-area';

const mocks = vi.hoisted(() => ({
  providerSpy: vi.fn(),
}));

/* eslint-disable @typescript-eslint/naming-convention --
 * Mocked third-party component exports keep their production names.
 */
vi.mock('@coze-agent-ide/chat-debug-area', () => ({
  BotDebugChatAreaComponentProvider: ({
    children,
    value,
  }: {
    children?: React.ReactNode;
    value?: unknown;
  }) => {
    mocks.providerSpy(value);
    return <div>{children}</div>;
  },
  BotDebugChatArea: ({ headerNode }: { headerNode?: React.ReactNode }) => (
    <div data-testid="bot-debug-chat-area">{headerNode}</div>
  ),
}));

vi.mock('../codex-trace/codex-trace-panel', () => ({
  CodexTracePanel: () => <div data-testid="codex-trace-panel" />,
}));

vi.mock('../codex-trace/trace-bridge', () => ({
  TraceBridge: () => <div data-testid="trace-bridge" />,
}));

describe('SuperChatArea', () => {
  beforeEach(() => {
    mocks.providerSpy.mockClear();
  });

  it('renders the redesigned chat header outside debug composer chrome', () => {
    render(<SuperChatArea />);

    expect(screen.getByText('预览与调试')).toBeTruthy();
    expect(screen.queryByText('布局')).toBeNull();
    expect(screen.getByTestId('codex-trace-panel')).toBeTruthy();
    expect(screen.getByTestId('bot-debug-chat-area').textContent).not.toContain(
      '执行台',
    );
  });

  it('keeps the composer at the bottom without sticky positioning', () => {
    const css = readFileSync(
      resolve(
        process.cwd(),
        'src/modes/super-mode/super-chat-area.module.less',
      ),
      'utf8',
    );

    expect(css).toContain('grid-template-rows: minmax(0, 1fr) auto;');
    expect(css).toContain('align-items: flex-end;');
    expect(css).toContain("[class*='chat-area-message-group-list']");
    expect(css).toContain("[class*='safe-area-']");
    expect(css).toContain('height: 0 !important;');
    expect(css).not.toContain('position: sticky;');
    expect(css).not.toContain('bottom: 0;');
  });

  it('can render the native chat stream for embedded workflow agents', () => {
    render(
      <SuperChatArea
        messageView="native"
        title="finmallclaw"
        chatSlot={<div data-testid="workflow-command-bridge" />}
      />,
    );

    expect(screen.getByText('finmallclaw')).toBeTruthy();
    expect(screen.getByTestId('bot-debug-chat-area')).toBeTruthy();
    expect(screen.getByTestId('workflow-command-bridge')).toBeTruthy();
    expect(screen.queryByTestId('codex-trace-panel')).toBeNull();
    expect(mocks.providerSpy).toHaveBeenCalledWith({});
  });

  it('keeps the workflow command bridge mounted in trace composer mode', () => {
    render(
      <SuperChatArea
        messageView="trace"
        title="finmallclaw"
        chatInputTopSlot={<div data-testid="workflow-command-bridge" />}
      />,
    );

    const lastProviderCall =
      mocks.providerSpy.mock.calls[mocks.providerSpy.mock.calls.length - 1];
    const injected = lastProviderCall?.[0] as {
      chatInputIntegration?: {
        renderChatInputTopSlot?: () => React.ReactNode;
      };
    };

    render(<>{injected.chatInputIntegration?.renderChatInputTopSlot?.()}</>);

    expect(screen.getByTestId('trace-bridge')).toBeTruthy();
    expect(screen.getByTestId('workflow-command-bridge')).toBeTruthy();
  });

  it('keeps native chat header from taking message-list height', () => {
    const css = readFileSync(
      resolve(
        process.cwd(),
        'src/modes/super-mode/super-chat-area.module.less',
      ),
      'utf8',
    );

    expect(css).toContain("[class*='header-node']");
    expect(css).toContain('flex: 0 0 0 !important;');
    expect(css).toContain("[class*='chat-area-message-group-list']");
    expect(css).toContain('flex: 1 1 auto !important;');
  });
});
