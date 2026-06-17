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

import { type ReactNode, useState } from 'react';

import { useShallow } from 'zustand/react/shallow';
import classNames from 'classnames';
import { usePageRuntimeStore } from '@coze-studio/bot-detail-store/page-runtime';
import { useBotDetailIsReadonly } from '@coze-studio/bot-detail-store';
import { BotPageFromEnum } from '@coze-arch/bot-typings/common';
import { BotMode, TabStatus } from '@coze-arch/bot-api/developer_api';
import { Modal } from '@coze-arch/coze-design';
import { AbilityAreaContainer } from '@coze-agent-ide/tool';
import { useBotPageStore } from '@coze-agent-ide/space-bot/store';
import {
  ContentView,
  BotDebugPanel,
  SingleSheet,
} from '@coze-agent-ide/space-bot/component';

import s from '../../index.module.less';
import { type AgentConfigAreaProps } from '../single-mode/section-area/agent-config-area/index';
import {
  AgentChatArea,
  type AgentChatAreaProps,
} from '../single-mode/section-area/agent-chat-area';
import { SuperConfigArea } from './super-config-area';
import { SandboxWorkspace } from './sandbox-workspace';
import { SuperAgentHero } from './super-hero';

export interface SuperModeProps
  extends Omit<AgentConfigAreaProps, 'isAllToolHidden'>,
    AgentChatAreaProps {
  rightSheetSlot?: ReactNode;
  chatAreaReadOnly?: boolean;
}

export const SuperMode: React.FC<SuperModeProps> = ({
  rightSheetSlot,
  renderChatTitleNode,
  chatSlot,
  chatHeaderClassName,
  chatAreaReadOnly,
  ...agentConfigAreaProps
}) => {
  const { isInit, historyVisible, pageFrom } = usePageRuntimeStore(
    useShallow(state => ({
      isInit: state.init,
      historyVisible: state.historyVisible,
      pageFrom: state.pageFrom,
    })),
  );

  const modeSwitching = useBotPageStore(state => state.bot.modeSwitching);
  const isReadonly = useBotDetailIsReadonly();
  const [isAllToolHidden, setIsAllToolHidden] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  return (
    <div
      className={classNames(
        s.container,
        historyVisible && s['playground-neat'],
        pageFrom === BotPageFromEnum.Store && s.store,
      )}
      style={{ display: 'flex', flexDirection: 'column', height: '100%' }}
    >
      <SuperAgentHero onOpenSettings={() => setSettingsOpen(true)} />
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
          {/* 两栏:左工作区(IDE)| 右聊天。人设/技能收进右上角设置弹框。 */}
          <ContentView
            mode={BotMode.SingleMode}
            style={{ gridTemplateColumns: '2fr 1fr' }}
          >
            {/* 左:工作区(IDE 文件树 + 查看器) */}
            <SingleSheet
              title="工作区"
              headerClassName={classNames([
                'coz-bg-plus',
                'coz-fg-secondary',
                '!h-12',
                '!px-4',
                '!py-0',
              ])}
              titleClassName="!text-[16px]"
              titleNode={
                <div className="flex items-center gap-[8px] h-full px-[4px]">
                  <span className="text-[16px] font-medium coz-fg-plus">
                    工作区
                  </span>
                  <span className="text-[11px] px-[6px] py-[1px] rounded-[4px] coz-mg-hglt coz-fg-hglt">
                    沙箱 · 隔离环境
                  </span>
                </div>
              }
            >
              <div className="h-full coz-bg-plus pt-[12px]">
                <SandboxWorkspace />
              </div>
            </SingleSheet>

            {/* 右:预览与调试(原生聊天) */}
            <AgentChatArea
              renderChatTitleNode={renderChatTitleNode}
              chatSlot={chatSlot}
              chatHeaderClassName={chatHeaderClassName}
              chatAreaReadOnly={chatAreaReadOnly}
            />
          </ContentView>

          {/* 人设 · 技能 · MCP 设置弹框(渲染在 AbilityAreaContainer 内,保证技能区 context 正常) */}
          <Modal
            visible={settingsOpen}
            onCancel={() => setSettingsOpen(false)}
            title="人设 · 技能 · MCP 设置"
            footer={null}
            width={760}
            height={620}
            bodyStyle={{ padding: 0, height: 560, overflow: 'hidden' }}
          >
            <div className="h-full">
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
