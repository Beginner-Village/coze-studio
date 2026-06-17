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

import { useShallow } from 'zustand/react/shallow';
import classNames from 'classnames';
import { SingleSheet } from '@coze-agent-ide/space-bot/component';
import { usePageRuntimeStore } from '@coze-studio/bot-detail-store/page-runtime';
import { I18n } from '@coze-arch/i18n';
import { LayoutContext, PlacementEnum } from '@coze-arch/bot-hooks';
import { PromptView } from '@coze-agent-ide/prompt-adapter';
import { BotConfigArea } from '@coze-agent-ide/bot-config-area-adapter';

import {
  ToolArea,
  type ToolAreaProps,
} from '../single-mode/section-area/agent-config-area/tool-area';

import cs from './super-config-area.module.less';

export type SuperConfigAreaProps = ToolAreaProps & {
  modelListExtraHeaderSlot?: React.ReactNode;
};

/**
 * 超级体左列:人设(上,紧凑)+ 技能/MCP 设置(下),上下排。
 * 不再用 single-mode 的左右并排,人设区压到一小块,只放提示词。
 */
export const SuperConfigArea: React.FC<SuperConfigAreaProps> = props => {
  const { editable, pageFrom } = usePageRuntimeStore(
    useShallow(state => ({
      editable: state.editable,
      pageFrom: state.pageFrom,
    })),
  );
  return (
    <SingleSheet
      headerClassName={classNames([
        'coz-bg-plus',
        'coz-fg-secondary',
        '!h-12',
        '!px-4',
        '!py-0',
      ])}
      title={I18n.t('bot_build_title')}
      titleClassName="!text-[16px]"
      titleNode={
        <div className={cs.titleNode}>
          <BotConfigArea
            pageFrom={pageFrom}
            editable={editable}
            modelListExtraHeaderSlot={props.modelListExtraHeaderSlot}
          />
        </div>
      }
    >
      <div className={cs.stack}>
        {/* 人设:紧凑,只放提示词 */}
        <div className={cs.persona}>
          <LayoutContext value={{ placement: PlacementEnum.LEFT }}>
            <PromptView />
          </LayoutContext>
        </div>
        {/* 技能 / MCP / 设置 */}
        <div className={cs.tools}>
          <ToolArea {...props} />
        </div>
      </div>
    </SingleSheet>
  );
};
