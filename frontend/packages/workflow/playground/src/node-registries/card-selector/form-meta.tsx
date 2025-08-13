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

import {
  ValidateTrigger,
  type FormMetaV2,
} from '@flowgram-adapter/free-layout-editor';

import { nodeMetaValidate } from '@/nodes-v2/materials/node-meta-validate';
import {
  fireNodeTitleChange,
  provideNodeOutputVariablesEffect,
} from '@/node-registries/common/effects';

import { outputTreeMetaValidator } from '../common/fields/outputs';
import { type FormData } from './types';
import { FormRender } from './form';
import { transformOnInit, transformOnSubmit } from './data-transformer';
import { CARD_SELECTOR_PATH, OUTPUT_PATH } from './constants';

// Card selector validator
const cardSelectorValidator = (value: unknown) => {
  if (!value) {
    console.log('🔍 CardSelector Validator: No value provided');
    return;
  }

  const cardSelectorValue = value as {
    selectedCardId?: string;
    searchKeyword?: string;
    apiEndpoint?: string;
  };

  console.log('🔍 CardSelector Validator:', {
    selectedCardId: cardSelectorValue.selectedCardId,
    searchKeyword: cardSelectorValue.searchKeyword,
    apiEndpoint: cardSelectorValue.apiEndpoint,
  });

  // 如果已经选择了卡片，则验证通过
  if (
    cardSelectorValue.selectedCardId &&
    cardSelectorValue.selectedCardId.trim() !== ''
  ) {
    console.log('✅ CardSelector Validation passed: Card selected');
    return;
  }

  // 如果有有效的搜索关键词，也认为是有效的
  if (
    cardSelectorValue.searchKeyword &&
    cardSelectorValue.searchKeyword.trim() !== ''
  ) {
    console.log('✅ CardSelector Validation passed: Search keyword provided');
    return;
  }

  // 只有在用户已经配置了API端点但没有选择卡片也没有搜索关键词时，才显示必填错误
  // 这样可以避免在初始状态就显示错误
  if (
    cardSelectorValue.apiEndpoint &&
    cardSelectorValue.apiEndpoint.trim() !== ''
  ) {
    console.warn(
      '❌ CardSelector Validation failed: API endpoint configured but no card selected',
    );
    return 'card_selector_required';
  }

  console.log('✅ CardSelector Validation passed: Initial state');
  return;
};

export const CARD_SELECTOR_FORM_META: FormMetaV2<FormData> = {
  // Node form rendering
  render: () => <FormRender />,

  // Validation trigger timing
  validateTrigger: ValidateTrigger.onChange,

  // Validation rules
  validate: {
    nodeMeta: nodeMetaValidate,
    [CARD_SELECTOR_PATH]: cardSelectorValidator,
    [OUTPUT_PATH]: outputTreeMetaValidator,
  },

  // Form effects
  effect: {
    nodeMeta: fireNodeTitleChange,
    [OUTPUT_PATH]: provideNodeOutputVariablesEffect,
  },

  // Data transformation
  formatOnInit: transformOnInit,
  formatOnSubmit: transformOnSubmit,
};
