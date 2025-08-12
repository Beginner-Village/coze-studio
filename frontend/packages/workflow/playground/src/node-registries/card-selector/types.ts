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

import type { OutputTreeMeta, Parameter } from '@coze-workflow/base';

export interface FalconCard {
  id: string;
  name: string;
  description: string;
  category?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CardSelectorParams {
  selectedCardId?: string;
  searchKeyword?: string;
  apiEndpoint?: string;
  apiKey?: string;
}

export interface FormData {
  inputParameters: Parameter[];
  cardSelectorParams: CardSelectorParams;
  outputs: OutputTreeMeta[];
}
