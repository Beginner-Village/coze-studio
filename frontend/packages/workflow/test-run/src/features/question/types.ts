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

import type { ContentType, MessageType } from './constants';

export interface Option {
  name: string;
}

export interface OptionMessageContent {
  question: string;
  options: Array<Option>;
}

export interface CardData {
  templateId?: string;
  templateName?: string;
  kvMap?: Record<string, unknown>;
  dataResponse?: Record<string, unknown>;
}

export interface ReceivedMessage {
  type: MessageType;
  content_type: ContentType;
  content: string;
  id: string;
  answered?: boolean;
  /** 卡片数据，当 content_type 为 Card 时使用 */
  cardData?: CardData;
}
