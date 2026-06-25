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

import { useSearchParams } from 'react-router-dom';
import { type ReactNode, useState } from 'react';

import { useShallow } from 'zustand/react/shallow';
import classNames from 'classnames';
import { usePageRuntimeStore } from '@coze-studio/bot-detail-store/page-runtime';
import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { useBotDetailIsReadonly } from '@coze-studio/bot-detail-store';
import { Button, Modal } from '@coze-arch/coze-design';
import { BotPageFromEnum } from '@coze-arch/bot-typings/common';
import { BotMode } from '@coze-arch/bot-api/developer_api';
import { AbilityAreaContainer } from '@coze-agent-ide/tool';
import { useBotPageStore } from '@coze-agent-ide/space-bot/store';
import {
  ContentView,
  BotDebugPanel,
} from '@coze-agent-ide/space-bot/component';

import { type AgentConfigAreaProps } from '../single-mode/section-area/agent-config-area/index';
import { type AgentChatAreaProps } from '../single-mode/section-area/agent-chat-area';
import s from '../../index.module.less';
import { SuperSessionSidebar } from './super-session-sidebar';
import { SuperAgentHero } from './super-hero';
import { SuperConfigArea } from './super-config-area';
import { SuperChatArea } from './super-chat-area';
import { SandboxWorkspace } from './sandbox-workspace';
import { EmployeeRoster } from './employee-roster';

import configStyles from './super-config-area.module.less';

export interface SuperModeProps
  extends Omit<AgentConfigAreaProps, 'isAllToolHidden'>,
    AgentChatAreaProps {
  rightSheetSlot?: ReactNode;
  chatAreaReadOnly?: boolean;
}

type SuperContentLayoutProps = Pick<
  AgentChatAreaProps,
  'renderChatTitleNode' | 'chatSlot' | 'chatHeaderClassName'
> & {
  employeeChat: boolean;
  currentBotId?: string;
  chatAreaReadOnly?: boolean;
};

/**
 * 超级体工作面三栏布局。
 * - 员工聊天模式(employeeChat):左员工列表 | 中聊天 | 右产物/文件。
 * - 普通模式:左沙箱文件 | 中编辑器/产物 | 右聊天。
 */
const SuperContentLayout: React.FC<SuperContentLayoutProps> = ({
  employeeChat,
  currentBotId,
  renderChatTitleNode,
  chatSlot,
  chatHeaderClassName,
  chatAreaReadOnly,
}) => {
  // 员工聊天:以对话为主，产物面板默认折叠，按需用右侧抽屉把手展开。
  const [artifactsOpen, setArtifactsOpen] = useState(false);
  const chatPanel = (
    <div className={configStyles.chatPanel}>
      <SuperChatArea
        renderChatTitleNode={renderChatTitleNode}
        chatSlot={chatSlot}
        chatHeaderClassName={chatHeaderClassName}
        chatAreaReadOnly={chatAreaReadOnly}
      />
    </div>
  );

  if (employeeChat) {
    return (
      <ContentView
        mode={BotMode.SingleMode}
        style={{
          gridTemplateColumns: artifactsOpen
            ? '248px minmax(420px, 1fr) 380px'
            : '248px minmax(480px, 1fr)',
          gap: 12,
          padding: 12,
          background: 'rgb(244, 246, 251)',
          position: 'relative',
        }}
      >
        <EmployeeRoster currentBotId={currentBotId} />
        {chatPanel}
        {artifactsOpen ? <SandboxWorkspace /> : null}
        {/* 产物抽屉把手：默认折叠，以对话为主；点开右侧看员工产出的文件 */}
        <button
          type="button"
          onClick={() => setArtifactsOpen(o => !o)}
          title={artifactsOpen ? '收起产物' : '查看产物文件'}
          style={{
            position: 'absolute',
            top: '50%',
            right: 0,
            transform: 'translateY(-50%)',
            zIndex: 20,
            width: 30,
            padding: '16px 0',
            borderRadius: '10px 0 0 10px',
            border: '1px solid rgba(28, 31, 35, 0.1)',
            borderRight: 'none',
            background: artifactsOpen ? 'rgb(76, 139, 255)' : '#fff',
            color: artifactsOpen ? '#fff' : '#4e5969',
            cursor: 'pointer',
            fontSize: 12,
            letterSpacing: 2,
            writingMode: 'vertical-rl',
            boxShadow: '-2px 0 10px rgba(28, 31, 35, 0.08)',
          }}
        >
          {artifactsOpen ? '收起 ▸' : '◂ 产物'}
        </button>
      </ContentView>
    );
  }

  return (
    <ContentView
      mode={BotMode.SingleMode}
      style={{
        gridTemplateColumns: '232px minmax(560px, 2.35fr) minmax(390px, 1fr)',
        gap: 12,
        padding: 12,
        background: 'rgb(244, 246, 251)',
      }}
    >
      <SuperSessionSidebar />
      <SandboxWorkspace />
      {chatPanel}
    </ContentView>
  );
};

export const SuperMode: React.FC<SuperModeProps> = ({
  rightSheetSlot,
  renderChatTitleNode,
  chatSlot,
  chatHeaderClassName,
  chatAreaReadOnly,
  ...agentConfigAreaProps
}) => {
  const { isInit, historyVisible, pageFrom } = usePageRuntimeStore(
    useShallow(
      (state: {
        init: boolean;
        historyVisible: boolean;
        pageFrom?: BotPageFromEnum;
      }) => ({
        isInit: state.init,
        historyVisible: state.historyVisible,
        pageFrom: state.pageFrom,
      }),
    ),
  );

  const modeSwitching = useBotPageStore(
    (state: { bot: { modeSwitching: boolean } }) => state.bot.modeSwitching,
  );
  const isReadonly = useBotDetailIsReadonly();
  const [isAllToolHidden, setIsAllToolHidden] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  // 虚拟员工「对话」入口带 employeeChat=1：隐藏编辑/发布工具栏，做成纯对话界面
  // （只读实例，本就不可编辑），保留会话/沙箱产物/聊天三栏工作面。
  const [searchParams] = useSearchParams();
  const employeeChat = searchParams.get('employeeChat') === '1';
  const currentBotId = useBotInfoStore(
    (state: { botId: string }) => state.botId,
  );

  return (
    <div
      className={classNames(
        s.container,
        historyVisible && s['playground-neat'],
        pageFrom === BotPageFromEnum.Store && s.store,
      )}
      style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
    >
      {employeeChat ? null : (
        <SuperAgentHero onOpenSettings={() => setSettingsOpen(true)} />
      )}
      <div style={{ flex: 1, minHeight: 0, display: 'flex' }}>
        <AbilityAreaContainer
          enableToolHiddenMode
          eventCallbacks={{
            onAllToolHiddenStatusChange: (v: boolean) => setIsAllToolHidden(v),
          }}
          isReadonly={isReadonly}
          mode={BotMode.SingleMode}
          modeSwitching={modeSwitching}
          isInit={isInit}
        >
          <SuperContentLayout
            employeeChat={employeeChat}
            currentBotId={currentBotId}
            renderChatTitleNode={renderChatTitleNode}
            chatSlot={chatSlot}
            chatHeaderClassName={chatHeaderClassName}
            chatAreaReadOnly={chatAreaReadOnly}
          />

          {/* 人设 · 技能 · MCP 设置弹窗(渲染在 AbilityAreaContainer 内,保证技能区 context 正常) */}
          <Modal
            visible={settingsOpen}
            onCancel={() => setSettingsOpen(false)}
            title={
              <span className={configStyles.modalTitle}>
                人设
                <span className={configStyles.modalTitleDot} />
                技能
                <span className={configStyles.modalTitleDot} />
                MCP 设置
              </span>
            }
            footer={
              <div className={configStyles.settingsModalFooter}>
                <div className={configStyles.saveHint}>修改自动保存</div>
                <div className={configStyles.footerActions}>
                  <Button
                    color="secondary"
                    onClick={() => setSettingsOpen(false)}
                  >
                    取消
                  </Button>
                  <Button
                    color="primary"
                    onClick={() => setSettingsOpen(false)}
                  >
                    完成
                  </Button>
                </div>
              </div>
            }
            width={880}
            // Semi 默认 Modal=1000、Popover/Dropdown=1030,后者本就该盖在 Modal 上。
            // 之前显式设 1100 反超了浮层,导致「添加技能」弹层被遮挡,故回落到 1000。
            zIndex={1000}
            bodyStyle={{ padding: 0, overflow: 'hidden' }}
            maskStyle={{
              background: 'rgba(8, 12, 24, 0.42)',
              backdropFilter: 'blur(2px)',
            }}
            className={configStyles.settingsModal}
          >
            <div className={configStyles.settingsModalBody}>
              <SuperConfigArea
                isAllToolHidden={isAllToolHidden}
                {...agentConfigAreaProps}
              />
            </div>
          </Modal>

          <BotDebugPanel />
          {rightSheetSlot}
        </AbilityAreaContainer>
      </div>
    </div>
  );
};
