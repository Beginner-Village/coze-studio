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

import { type NodeData } from '@coze-workflow/base';

import { type FormData } from './types';
import { DEFAULT_INPUTS, DEFAULT_OUTPUTS } from './constants';

/**
 * Convert canvas data to form data
 */
export function transformOnInit(data: NodeData): FormData {
  return {
    inputParameters: data?.inputParameters || DEFAULT_INPUTS,
    cardSelectorParams: {
      selectedCardId: data?.cardSelectorParams?.selectedCardId || '',
      searchKeyword: data?.cardSelectorParams?.searchKeyword || '',
      apiEndpoint: data?.cardSelectorParams?.apiEndpoint || '',
      apiKey: data?.cardSelectorParams?.apiKey || '',
    },
    outputs: data?.outputs || DEFAULT_OUTPUTS,
  };
}

/**
 * Convert form data to canvas data
 */
export function transformOnSubmit(data: FormData): NodeData {
  return {
    inputParameters: data.inputParameters,
    cardSelectorParams: {
      selectedCardId: data.cardSelectorParams?.selectedCardId || '',
      searchKeyword: data.cardSelectorParams?.searchKeyword || '',
      apiEndpoint: data.cardSelectorParams?.apiEndpoint || '',
      apiKey: data.cardSelectorParams?.apiKey || '',
    },
    outputs:
      !data.outputs || data.outputs.length === 0
        ? DEFAULT_OUTPUTS
        : data.outputs,
  };
}
