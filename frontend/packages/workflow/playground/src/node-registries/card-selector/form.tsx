/*
 * Copyright 2025 coze-dev Authors
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

import React from 'react';

import { I18n } from '@coze-arch/i18n';

import { NodeConfigForm } from '@/node-registries/common/components';

import { InputsParametersField, OutputsField } from '../common/fields';
import { INPUT_PATH, CARD_SELECTOR_PATH, OUTPUT_PATH } from './constants';
import { CardSelectorField } from './components/card-selector-field';

export function FormRender() {
  return (
    <NodeConfigForm>
      <CardSelectorField
        name={CARD_SELECTOR_PATH}
        title={I18n.t('card_selector_config')}
        tooltip={I18n.t('card_selector_config_desc')}
        id="card-selector-node-config"
      />

      <InputsParametersField
        name={INPUT_PATH}
        title={I18n.t('workflow_detail_node_input')}
        tooltip={I18n.t('card_selector_inputs_desc')}
      />

      <OutputsField
        title={I18n.t('workflow_detail_node_output')}
        tooltip={I18n.t('card_selector_outputs_desc')}
        id="card-selector-node-outputs"
        name={OUTPUT_PATH}
        topLevelReadonly={true}
        customReadonly
      />
    </NodeConfigForm>
  );
}
