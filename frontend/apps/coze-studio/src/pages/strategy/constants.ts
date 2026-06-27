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

import { I18n } from '@coze-arch/i18n';

export const CAPABILITY_TYPES = [
  {
    label: () => I18n.t('strategy_capability_type_workflow'),
    value: 'workflow',
  },
  { label: () => I18n.t('strategy_capability_type_plugin'), value: 'plugin' },
  {
    label: () => I18n.t('strategy_capability_type_knowledge'),
    value: 'knowledge',
  },
  { label: () => I18n.t('strategy_capability_type_prompt'), value: 'prompt' },
];

export const TYPE_COLORS: Record<string, string> = {
  workflow: 'blue',
  plugin: 'green',
  knowledge: 'orange',
  prompt: 'purple',
};

export const CAPABILITY_TYPE_LABELS: Record<string, () => string> = {
  workflow: () => I18n.t('strategy_capability_type_workflow'),
  plugin: () => I18n.t('strategy_capability_type_plugin'),
  knowledge: () => I18n.t('strategy_capability_type_knowledge'),
  prompt: () => I18n.t('strategy_capability_type_prompt'),
};

export const DEFAULT_ADD_CAP_FORM = {
  type: 'workflow',
  ref_id: '',
  ref_sub_id: '',
  prompt_content: '',
  alias_name: '',
  alias_description: '',
};
