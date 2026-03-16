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

/**
 * 检测是否为特殊的answer消息（包含displayResponseType）
 */
export function isSpecialAnswerMessage(message: any): boolean {
  // 必须是answer类型的消息
  if (message?.type !== 'answer') {
    return false;
  }

  if (!message.content || typeof message.content !== 'string') {
    return false;
  }

  try {
    // 尝试解析JSON内容
    const contentData = JSON.parse(message.content);
    
    // 检查是否有contentList且包含displayResponseType
    if (Array.isArray(contentData?.contentList)) {
      const hasSpecialType = contentData.contentList.some((item: any) => 
        item && typeof item === 'object' && 'displayResponseType' in item
      );
      
      // 添加调试日志
      if (hasSpecialType) {
        console.log('🎯 检测到特殊answer消息:', message.message_id, contentData);
      }
      
      return hasSpecialType;
    }

    return false;
  } catch (error) {
    // 如果JSON解析失败，但包含特殊关键字，也认为是特殊消息
    const isSpecial = message.content.includes('displayResponseType') && 
                      message.content.includes('contentList');
    
    if (isSpecial) {
      console.log('🎯 检测到特殊answer消息(fallback):', message.message_id, error);
    }
    
    return isSpecial;
  }
}

/**
 * 从消息中提取contentList数据
 */
export function extractContentList(message: any): Array<{
  displayResponseType?: string;
  templateId?: string;
  cardId?: string;
  kvMap?: Record<string, any>;
  dataResponse?: Record<string, any>;
}> | undefined {
  if (!message?.content || typeof message.content !== 'string') {
    return undefined;
  }

  try {
    const contentData = JSON.parse(message.content);

    if (Array.isArray(contentData?.contentList)) {
      return contentData.contentList;
    }

    return undefined;
  } catch {
    return undefined;
  }
}

/**
 * Card group information for layout
 */
export interface CardGroupInfo {
  groupId: string;
  layout: 'horizontal' | 'vertical' | 'waterfall';
  columns: number;
  cardIds: string[];
}

/**
 * 从消息中提取groups布局数据
 */
export function extractGroups(message: any): CardGroupInfo[] | undefined {
  if (!message?.content || typeof message.content !== 'string') {
    return undefined;
  }

  try {
    const contentData = JSON.parse(message.content);

    if (Array.isArray(contentData?.groups) && contentData.groups.length > 0) {
      console.log('📐 提取到组布局数据:', contentData.groups);
      return contentData.groups;
    }

    return undefined;
  } catch {
    return undefined;
  }
}

/**
 * 从消息中提取rawContent原始文本内容
 * 这是非卡片部分的文本内容，在最终JSON中保存
 */
export function extractRawContent(message: any): string | undefined {
  if (!message?.content || typeof message.content !== 'string') {
    return undefined;
  }

  try {
    const contentData = JSON.parse(message.content);

    if (contentData?.rawContent && typeof contentData.rawContent === 'string') {
      console.log('📝 提取到原始文本内容:', contentData.rawContent.substring(0, 100) + '...');
      return contentData.rawContent;
    }

    return undefined;
  } catch {
    return undefined;
  }
}

/**
 * 清理文本中的卡片标签，用于在有流式卡片时只显示普通文本
 * 移除以下标签：
 * - <<CARD:template_id:template_name>>
 * - <<field_name>>field_value (字段标签和其后的内容直到下一个标签)
 * - <</CARD>>
 * - <<GROUP:layout:columns>>
 * - <</GROUP>>
 * - 三个反引号包裹的代码块（如果只包含卡片内容）
 */
export function cleanCardTagsFromContent(content: string): string {
  if (!content || typeof content !== 'string') {
    return content;
  }

  let result = content;

  // Remove GROUP tags
  result = result.replace(/<<GROUP:[^>]*>>/g, '');
  result = result.replace(/<<\/GROUP>>/g, '');

  // Remove CARD open tags: <<CARD:template_id:template_name>>
  result = result.replace(/<<CARD:[^>]*>>/g, '');

  // Remove CARD close tags: <</CARD>>
  result = result.replace(/<<\/CARD>>/g, '');

  // Remove field tags and their values: <<field_name>>value
  // This regex matches <<field_name>> followed by any content until the next << or end of line
  result = result.replace(/<<([^/][^>]*)>>[^\n<]*/g, '');

  // Remove empty code blocks (``` followed by only whitespace/newlines and then ```)
  result = result.replace(/```\s*\n?\s*```/g, '');

  // Clean up excessive newlines (more than 2 consecutive)
  result = result.replace(/\n{3,}/g, '\n\n');

  // Trim whitespace
  result = result.trim();

  return result;
}