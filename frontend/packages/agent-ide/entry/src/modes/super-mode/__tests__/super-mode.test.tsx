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

import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

const superChatAreaSpy = vi.fn();

vi.mock('zustand/react/shallow', () => ({
  useShallow: (selector: unknown) => selector,
}));

vi.mock('@coze-studio/bot-detail-store/page-runtime', () => ({
  usePageRuntimeStore: (selector: (state: unknown) => unknown) =>
    selector({ init: true, historyVisible: false, pageFrom: undefined }),
}));

vi.mock('@coze-studio/bot-detail-store', () => ({
  useBotDetailIsReadonly: () => false,
}));

vi.mock('@coze-arch/bot-typings/common', () => ({
  BotPageFromEnum: { Store: 'store' },
}));

vi.mock('@coze-arch/bot-api/developer_api', () => ({
  BotMode: { SingleMode: 1 },
  TabStatus: {},
}));

vi.mock('@coze-agent-ide/tool', () => ({
  AbilityAreaContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="ability-area">{children}</div>
  ),
}));

vi.mock('@coze-agent-ide/space-bot/store', () => ({
  useBotPageStore: (selector: (state: unknown) => unknown) =>
    selector({ bot: { modeSwitching: false } }),
}));

vi.mock('@coze-agent-ide/space-bot/component', () => ({
  ContentView: ({
    children,
    style,
  }: {
    children: React.ReactNode;
    style?: React.CSSProperties;
  }) => (
    <div data-testid="content-view" style={style}>
      {children}
    </div>
  ),
  BotDebugPanel: () => <div data-testid="debug-panel" />,
  SingleSheet: ({
    title,
    titleNode,
    children,
    renderContent,
  }: {
    title?: string;
    titleNode?: React.ReactNode;
    children?: React.ReactNode;
    renderContent?: (headerNode: React.ReactNode) => React.ReactNode;
  }) => (
    <section data-title={title}>
      {titleNode}
      {renderContent
        ? renderContent(<div data-testid="header-node" />)
        : children}
    </section>
  ),
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
  Modal: ({
    visible,
    children,
  }: {
    visible?: boolean;
    children?: React.ReactNode;
  }) => (visible ? <aside>{children}</aside> : null),
}));

vi.mock('../sandbox-workspace', () => ({
  SandboxWorkspace: () => <div data-testid="sandbox-workspace" />,
}));

vi.mock('../super-session-sidebar', () => ({
  SuperSessionSidebar: () => <div data-testid="super-session-sidebar" />,
}));

vi.mock('../super-config-area', () => ({
  SuperConfigArea: () => <div data-testid="super-config-area" />,
}));

vi.mock('../super-hero', () => ({
  SuperAgentHero: ({ onOpenSettings }: { onOpenSettings?: () => void }) => (
    <button type="button" onClick={onOpenSettings}>
      hero
    </button>
  ),
}));

vi.mock('../super-chat-area', () => ({
  SuperChatArea: (props: unknown) => {
    superChatAreaSpy(props);
    return <div data-testid="super-chat-area">调试聊天</div>;
  },
}));

describe('SuperMode', () => {
  it('renders the super chat area while preserving chat props', async () => {
    const { SuperMode } = await import('../index');
    const chatSlot = <div data-testid="chat-slot">slot</div>;
    const renderChatTitleNode = () => <span>trace title</span>;

    render(
      <SuperMode
        renderChatTitleNode={renderChatTitleNode}
        chatSlot={chatSlot}
        chatAreaReadOnly
      />,
    );

    expect(screen.getByTestId('super-chat-area').textContent).toContain(
      '调试聊天',
    );
    expect(screen.getByTestId('super-session-sidebar')).toBeTruthy();
    expect(superChatAreaSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        renderChatTitleNode,
        chatSlot,
        chatAreaReadOnly: true,
      }),
    );
  });

  it('uses the redesigned workbench/chat split', async () => {
    const { SuperMode } = await import('../index');

    render(<SuperMode />);

    expect(screen.getByTestId('content-view').style.gridTemplateColumns).toBe(
      '232px minmax(560px, 2.35fr) minmax(390px, 1fr)',
    );
  });
});
