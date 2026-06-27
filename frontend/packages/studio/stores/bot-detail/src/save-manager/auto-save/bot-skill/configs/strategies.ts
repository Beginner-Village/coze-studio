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

import type { StrategyBindItem } from '@/types/skill';
import { type BotSkillStore, useBotSkillStore } from '@/store/bot-skill';
import { ItemTypeExtra } from '@/save-manager/types';

type RegisterStrategies = HostedObserverConfig<
  BotSkillStore,
  ItemTypeExtra,
  StrategyBindItem[]
>;

export const strategiesConfig: RegisterStrategies = {
  key: ItemTypeExtra.Strategies,
  selector: store => store.strategies,
  debounce: DebounceTime.Immediate,
  middleware: {
    onBeforeSave: (dataSource: StrategyBindItem[]) => ({
      strategy_id_list: useBotSkillStore
        .getState()
        .transformVo2Dto.strategies(cloneDeep(dataSource)),
    }),
  },
};
