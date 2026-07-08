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
import { type ReactNode, useState, useEffect } from 'react';

import { useShallow } from 'zustand/react/shallow';
import classNames from 'classnames';
import { usePageRuntimeStore } from '@coze-studio/bot-detail-store/page-runtime';
import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { useBotDetailIsReadonly } from '@coze-studio/bot-detail-store';
import { Button, Modal } from '@coze-arch/coze-design';
import { BotPageFromEnum } from '@coze-arch/bot-typings/common';
import { BotMode } from '@coze-arch/bot-api/developer_api';
import {
  fetchSuperAgentEnabled,
  getSuperAgentEnabledCache,
} from '@coze-arch/bot-api';
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
    // 员工聊天里 chatPanel 是 flex 列的子项，必须 flex:1 撑满，否则空会话时
    // 聊天面板按内容缩成一小块、输入框浮在顶部不沉底（普通布局是 grid 子项，flex 无影响）。
    <div
      className={configStyles.chatPanel}
      style={employeeChat ? { flex: '1 1 0%', minHeight: 0 } : undefined}
    >
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
          // 产物展开时占满主区（员工列表→最右），聊天用 display:none 暂藏（保留
          // 会话状态），不挤压聊天、不覆盖、不遮挡输入框；产物拿到全宽完整显示。
          gridTemplateColumns: '248px minmax(0, 1fr)',
          // 行高必须约束成视口高度（minmax(0,1fr)），否则网格行按聊天内容撑高、
          // 溢出被 overflow:hidden 裁掉，导致输入框被顶到视口外、消息区不能内部滚动。
          gridTemplateRows: 'minmax(0, 1fr)',
          gap: 12,
          padding: 12,
          background: 'rgb(244, 246, 251)',
          position: 'relative',
        }}
      >
        <EmployeeRoster currentBotId={currentBotId} />
        <div style={{ position: 'relative', minWidth: 0, height: '100%' }}>
          <div
            style={{
              height: '100%',
              minWidth: 0,
              display: artifactsOpen ? 'none' : 'flex',
              flexDirection: 'column',
            }}
          >
            {chatPanel}
          </div>
          {artifactsOpen ? (
            <div style={{ position: 'absolute', inset: 0, display: 'flex' }}>
              <SandboxWorkspace />
            </div>
          ) : null}
        </div>
        {/* 把手：右缘常驻，开/合产物面板 */}
        <button
          type="button"
          onClick={() => setArtifactsOpen(o => !o)}
          title={artifactsOpen ? '返回对话' : '查看产物文件'}
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

const SuperModeInner: React.FC<SuperModeProps> = ({
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

/**
 * 超级体编辑器入口。是否渲染改为运行时驱动：向后端 `/api/super-agent/ui-config` 拉取
 * （联动 SANDBOX_ENABLED），运维改后端一个 env 即可，前端无需重新构建。
 *
 * 这里用「外层 gate + 内层 SuperModeInner」而非在组件顶部早返回：外层只有恒定的
 * useState/useEffect 两个 hook，内层组件仅在启用时挂载，二者都不违反 rules-of-hooks。
 * 构建期常量 IS_DISABLE_SUPER_AGENT 保留为硬兜底（置 true 永远隐藏）；降级默认隐藏。
 */
export const SuperMode: React.FC<SuperModeProps> = props => {
  const [enabled, setEnabled] = useState<boolean>(
    getSuperAgentEnabledCache() ?? false,
  );
  useEffect(() => {
    fetchSuperAgentEnabled().then(setEnabled);
  }, []);
  if (IS_DISABLE_SUPER_AGENT || !enabled) {
    return null;
  }
  return <SuperModeInner {...props} />;
};
