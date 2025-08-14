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

// import { nanoid } from 'nanoid';
// import { ViewVariableType } from '@coze-workflow/variable';

// path
export const INPUT_PATH = 'inputParameters';
export const CARD_SELECTOR_PATH = 'cardSelectorParams';
export const OUTPUT_PATH = 'outputs';
export const CARD_SELECTOR_OUTPUT_PATH = 'cardSelectorOutputs';

// default value
export const DEFAULT_OUTPUTS: Array<unknown> = [];

export const DEFAULT_INPUTS: Array<{ name: string; type?: string }> = [];

// 默认的卡片选择器输出结构
export const DEFAULT_CARD_SELECTOR_OUTPUT = {
  displayResponseType: 'TEMPLATE',
  rawContent: {},
  templateId: '{card_code}',
  templateName: '{card_name}',
  kvMap: {},
  dataResponse: {
    payeeList: '{payeeList}',
  },
};
