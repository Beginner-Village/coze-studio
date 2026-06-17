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

import { cloneDeep } from 'lodash-es';
import { DebounceTime, type HostedObserverConfig } from '@coze-studio/autosave';

import type { AgentSkillItem } from '@/types/skill';
import { type BotSkillStore, useBotSkillStore } from '@/store/bot-skill';
import { ItemTypeExtra } from '@/save-manager/types';

type RegisterAgentSkills = HostedObserverConfig<
  BotSkillStore,
  ItemTypeExtra,
  AgentSkillItem[]
>;

// 绑定/移除技能后,把 agentSkills 持久化为 skill_info_list。
// 之前缺这个配置,导致 UI 绑定技能不落库(skill_info_list 一直为空)。
export const agentSkillsConfig: RegisterAgentSkills = {
  key: ItemTypeExtra.AgentSkills,
  selector: store => store.agentSkills,
  debounce: DebounceTime.Immediate,
  middleware: {
    onBeforeSave: (dataSource: AgentSkillItem[]) => ({
      skill_info_list: useBotSkillStore
        .getState()
        .transformVo2Dto.agentSkills(cloneDeep(dataSource)),
    }),
  },
};
