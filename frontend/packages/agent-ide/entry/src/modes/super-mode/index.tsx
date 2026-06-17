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
import { AbilityAreaContainer } from '@coze-agent-ide/tool';
import { useBotPageStore } from '@coze-agent-ide/space-bot/store';
import {
  ContentView,
  BotDebugPanel,
  SingleSheet,
} from '@coze-agent-ide/space-bot/component';

import s from '../../index.module.less';
import {
  AgentConfigArea,
  type AgentConfigAreaProps,
} from '../single-mode/section-area/agent-config-area/index';
import {
  AgentChatArea,
  type AgentChatAreaProps,
} from '../single-mode/section-area/agent-chat-area';
import { SandboxWorkspace } from './sandbox-workspace';

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

  return (
    <div
      className={classNames(
        s.container,
        historyVisible && s['playground-neat'],
        pageFrom === BotPageFromEnum.Store && s.store,
      )}
    >
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
        <ContentView
          mode={BotMode.SingleMode}
          style={{ gridTemplateColumns: '28fr 16fr 17fr' }}
        >
          {/* 左:角色人设 + 技能/MCP（精简) */}
          <AgentConfigArea
            isAllToolHidden={isAllToolHidden}
            {...agentConfigAreaProps}
          />

          {/* 中:沙箱空间 —— 超级智能体独有，做成独立主角列 */}
          <SingleSheet
            title="沙箱空间"
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
                  沙箱空间
                </span>
                <span className="text-[11px] px-[6px] py-[1px] rounded-[4px] coz-mg-hglt coz-fg-hglt">
                  隔离环境
                </span>
              </div>
            }
          >
            <div className="h-full coz-bg-plus pt-[12px]">
              <SandboxWorkspace />
            </div>
          </SingleSheet>

          {/* 右:预览与调试 */}
          <AgentChatArea
            renderChatTitleNode={renderChatTitleNode}
            chatSlot={chatSlot}
            chatHeaderClassName={chatHeaderClassName}
            chatAreaReadOnly={chatAreaReadOnly}
          />
        </ContentView>

        <BotDebugPanel />
        {rightSheetSlot}
      </AbilityAreaContainer>
    </div>
  );
};
