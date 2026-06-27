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

import { useState } from 'react';

import classNames from 'classnames';
import { LayoutContext, PlacementEnum } from '@coze-arch/bot-hooks';
import { PromptView } from '@coze-agent-ide/prompt-adapter';
import { SingleAgentModelView } from '@coze-agent-ide/bot-config-area-adapter';

import {
  ToolArea,
  type ToolAreaProps,
} from '../single-mode/section-area/agent-config-area/tool-area';
import { SuperCapabilitiesSection } from './super-capabilities-section';
import { PublishVirtualEmployeeModal } from './publish-virtual-employee-modal';
import { IcChevronDown, IcSettings } from './icons';

import cs from './super-config-area.module.less';

export type SuperConfigAreaProps = ToolAreaProps & {
  modelListExtraHeaderSlot?: React.ReactNode;
};

export const SuperConfigArea: React.FC<SuperConfigAreaProps> = props => {
  const [publishModalVisible, setPublishModalVisible] = useState(false);

  return (
    <div className={cs.settingsShell}>
      <div className={cs.settingsToolbar}>
        <div className={cs.toolbarLabel}>
          <IcSettings size={15} />
          模型与能力
        </div>
        <div className={cs.toolbarRight}>
          <button
            type="button"
            className={cs.publishVirtualEmployeeBtn}
            onClick={() => setPublishModalVisible(true)}
            data-testid="publish-virtual-employee-btn"
          >
            发布为虚拟员工
          </button>
          <div className={cs.modelSelectSlot}>
            <SingleAgentModelView
              modelListExtraHeaderSlot={props.modelListExtraHeaderSlot}
              popoverPosition="bottomRight"
              popoverClassName={cs.modelPopover}
              zIndex={1300}
              clickToHide
              triggerRender={(model, popoverVisible) => {
                const name = model?.name || '选择模型';
                const initial = name.slice(0, 1).toUpperCase();
                return (
                  <button
                    type="button"
                    className={classNames(
                      cs.modelTrigger,
                      popoverVisible && cs.modelTriggerOpen,
                    )}
                  >
                    <span className={cs.modelIcon}>{initial}</span>
                    <span className={cs.modelName}>{name}</span>
                    <IcChevronDown size={14} className={cs.modelChevron} />
                  </button>
                );
              }}
            />
          </div>
        </div>
      </div>

      <div className={cs.settingsScroll}>
        <section className={cs.personaSection}>
          <div className={cs.personaHead}>
            <div className={cs.personaLabel}>
              <span className={cs.required}>*</span>
              人设与回复逻辑
            </div>
          </div>
          <div className={cs.personaBox}>
            <LayoutContext value={{ placement: PlacementEnum.LEFT }}>
              <PromptView />
            </LayoutContext>
          </div>
        </section>

        <SuperCapabilitiesSection />

        <section className={cs.toolsSection}>
          <div className={cs.toolsHeader}>
            <div className={cs.toolsTitle}>
              <IcSettings size={16} />
              技能与 MCP
            </div>
          </div>
          <ToolArea {...props} />
        </section>
      </div>
      <PublishVirtualEmployeeModal
        visible={publishModalVisible}
        onClose={() => setPublishModalVisible(false)}
      />
    </div>
  );
};
