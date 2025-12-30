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

export interface CardContentItem {
  displayResponseType?: string;
  templateId?: string;
  templateName?: string;
  kvMap?: Record<string, unknown>;
  dataResponse?: Record<string, unknown>;
}

interface ContentListData {
  contentList?: unknown[];
  output?: string | ContentListData;
}

/**
 * 安全的 JSON 解析
 */
function safeJsonParse(str: string): unknown {
  try {
    return JSON.parse(str);
  } catch {
    // JSON 解析失败，返回 null
    return null;
  }
}

/**
 * 检查单个 item 是否是卡片内容
 */
function isCardContentItem(item: unknown): item is CardContentItem {
  return (
    item !== null && typeof item === 'object' && 'displayResponseType' in item
  );
}

/**
 * 从 contentList 中提取卡片内容
 */
function extractFromContentList(contentList: unknown[]): CardContentItem[] {
  return contentList.filter(isCardContentItem);
}

/**
 * 检测输出数据是否包含卡片内容
 */
export function isCardOutput(data: unknown): boolean {
  if (!data || typeof data !== 'object') {
    return false;
  }

  const typedData = data as ContentListData;

  // 检查 contentList 中是否有 displayResponseType
  if (Array.isArray(typedData.contentList)) {
    const hasCardItem = typedData.contentList.some(isCardContentItem);
    if (hasCardItem) {
      return true;
    }
  }

  // 检查 output 字段（某些情况下可能嵌套在 output 中）
  if (typedData.output) {
    const outputData =
      typeof typedData.output === 'string'
        ? (safeJsonParse(typedData.output) as ContentListData | null)
        : typedData.output;

    if (outputData && Array.isArray(outputData.contentList)) {
      return outputData.contentList.some(isCardContentItem);
    }
  }

  return false;
}

/**
 * 从输出数据中提取卡片内容列表
 */
export function extractCardContentList(
  data: unknown,
): CardContentItem[] | undefined {
  if (!data || typeof data !== 'object') {
    return undefined;
  }

  const typedData = data as ContentListData;

  // 直接从 contentList 提取
  if (Array.isArray(typedData.contentList)) {
    const cardItems = extractFromContentList(typedData.contentList);
    if (cardItems.length > 0) {
      return cardItems;
    }
  }

  // 从 output 字段提取
  if (typedData.output) {
    const outputData =
      typeof typedData.output === 'string'
        ? (safeJsonParse(typedData.output) as ContentListData | null)
        : typedData.output;

    if (outputData && Array.isArray(outputData.contentList)) {
      const cardItems = extractFromContentList(outputData.contentList);
      if (cardItems.length > 0) {
        return cardItems;
      }
    }
  }

  return undefined;
}
