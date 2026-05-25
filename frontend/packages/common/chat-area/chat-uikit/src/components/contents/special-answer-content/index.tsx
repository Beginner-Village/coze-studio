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

import { useState, useRef, useEffect, useCallback, useMemo, type FC } from 'react';
import { type IBaseContentProps } from '@coze-common/chat-uikit-shared';

import { TextContent } from '../text-content';
import './index.less';

export interface SpecialContentItem {
  displayResponseType?: string;
  templateId?: string;
  kvMap?: Record<string, any>;
  dataResponse?: Record<string, any>;
  cardId?: string; // Card ID for group association
}

// Card group information for layout
export interface CardGroupInfo {
  groupId: string;
  layout: 'horizontal' | 'vertical' | 'waterfall';
  columns: number;
  cardIds: string[];
}

export interface SpecialAnswerContentProps extends IBaseContentProps {
  contentList?: SpecialContentItem[];
  groups?: CardGroupInfo[]; // Group layout information
  rawContent?: string; // Original text content (non-card content)
}

declare global {
  interface Window {
    APP_CONFIG?: {
      CARD_URL?: string;
    };
  }
}

// eventId 生成器：时间戳 + 自增整数
let eventIdCounter = 0;
const generateEventId = (): string => {
  const timestamp = Date.now();
  const counter = ++eventIdCounter;
  return `${timestamp}_${counter}`;
};

// 自动高度参数
const AUTO_HEIGHT_PADDING = 16;
const AUTO_HEIGHT_MIN_THRESHOLD = 50;
const AUTO_HEIGHT_POLL_MS = 300;
const AUTO_HEIGHT_POLL_COUNT = 10; // 共 3s 兜底
const MAX_AUTO_HEIGHT = 4000;

/**
 * 给 iframe 装上"自动高度"能力：
 * 1) ResizeObserver 监听同域 body（首选，实时）
 * 2) 兜底短期轮询（3s 内反复测量，处理图片/字体异步加载完后的高度变化）
 * 3) 跨域情况下两者都不工作，依赖 iframe 内部 postMessage resize（已有）
 */
const setupAutoHeight = (
  iframe: HTMLIFrameElement,
  applyHeight: (h: number) => void,
): (() => void) => {
  let observer: ResizeObserver | null = null;
  let pollTimer: number | null = null;

  const measure = () => {
    try {
      const doc = iframe.contentDocument || iframe.contentWindow?.document;
      if (!doc) return;
      const body = doc.body;
      const html = doc.documentElement;
      const h = Math.max(
        body?.scrollHeight || 0,
        body?.offsetHeight || 0,
        html?.clientHeight || 0,
        html?.scrollHeight || 0,
      );
      if (h > AUTO_HEIGHT_MIN_THRESHOLD) {
        applyHeight(Math.min(h + AUTO_HEIGHT_PADDING, MAX_AUTO_HEIGHT));
      }
    } catch (err) {
      // 跨域，依赖 postMessage 兜底
      console.debug('[Card] measure skipped (cross-origin):', err);
    }
  };

  measure();

  try {
    const doc = iframe.contentDocument || iframe.contentWindow?.document;
    if (doc?.body && typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(() => measure());
      observer.observe(doc.body);
    }
  } catch (err) {
    console.debug('[Card] ResizeObserver setup skipped:', err);
  }

  let pollCount = 0;
  pollTimer = window.setInterval(() => {
    measure();
    pollCount += 1;
    if (pollCount >= AUTO_HEIGHT_POLL_COUNT && pollTimer !== null) {
      window.clearInterval(pollTimer);
      pollTimer = null;
    }
  }, AUTO_HEIGHT_POLL_MS);

  return () => {
    observer?.disconnect();
    if (pollTimer !== null) {
      window.clearInterval(pollTimer);
    }
  };
};

/**
 * Parse JSON string values in fields to actual objects/arrays
 * Handles multiple formats:
 * 1. Single JSON object: "{...}"
 * 2. Single JSON array: "[...]"
 * 3. Newline-separated JSON objects: "{...}\n{...}\n" -> [{...}, {...}]
 *
 * This ensures iframe receives data in the same format as streaming-card-content
 * @param fields Record from kvMap or dataResponse
 * @returns Record with parsed JSON values
 */
const parseFieldValues = (
  fields: Record<string, any>,
): Record<string, unknown> => {
  const parsed: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(fields)) {
    // Handle null/undefined
    if (value === null || value === undefined) {
      parsed[key] = value;
      continue;
    }

    // If not a string, keep as is
    if (typeof value !== 'string') {
      parsed[key] = value;
      continue;
    }

    const trimmed = value.trim();
    if (!trimmed) {
      parsed[key] = value;
      continue;
    }

    // Case 1: Single JSON array "[...]"
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      try {
        parsed[key] = JSON.parse(trimmed);
        continue;
      } catch {
        // Fall through to other cases
      }
    }

    // Case 2: Newline-separated JSON objects -> parse as array
    // Format: "{...}\n{...}\n" from card_delta add operations
    if (trimmed.includes('\n') && trimmed.startsWith('{')) {
      try {
        const lines = trimmed.split('\n').filter(line => line.trim());
        const items = lines.map(line => JSON.parse(line.trim()));
        parsed[key] = items;
        continue;
      } catch {
        // Fall through to single object parsing
      }
    }

    // Case 3: Single JSON object "{...}"
    if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
      try {
        parsed[key] = JSON.parse(trimmed);
        continue;
      } catch {
        // Keep original value
      }
    }

    // Default: keep original string value
    parsed[key] = value;
  }

  return parsed;
};

/**
 * 单个卡片组件 - 用于渲染单张卡片的 iframe
 */
const SingleCardContent: FC<{
  content: SpecialContentItem;
  index: number;
}> = ({ content, index }) => {
  const [iframeHeight, setIframeHeight] = useState<number>(320); // Match streaming-card default height
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  // 计算目标 Origin
  const cardUrl =
    window.APP_CONFIG?.CARD_URL ||
    '/agent-h5-web/card/index.html';
  let targetOrigin = '';
  try {
    if (cardUrl.startsWith('http')) {
      targetOrigin = new URL(cardUrl).origin;
    } else {
      targetOrigin = window.location.origin;
    }
  } catch (e) {
    targetOrigin = window.location.origin;
  }

  // 生成iframe URL
  const generateIframeUrl = useCallback(() => {
    const baseUrl =
      window.APP_CONFIG?.CARD_URL ||
      '/agent-h5-web/card/index.html';

    let spaceId = '';
    try {
      const pathMatch = window.location.pathname.match(/\/space\/([^\/]+)/);
      if (pathMatch && pathMatch[1]) {
        spaceId = pathMatch[1];
      }
      if (!spaceId) {
        const urlParams = new URLSearchParams(window.location.search);
        spaceId = urlParams.get('space_id') || urlParams.get('spaceId') || '';
      }
    } catch (error) {
      console.warn('⚠️ 无法从 URL 获取 spaceId:', error);
    }

    return spaceId ? `${baseUrl}?spaceId=${spaceId}` : baseUrl;
  }, []);

  useEffect(() => {
    const iframe = iframeRef.current;
    if (!iframe || !content) {
      return;
    }

    const sendCardData = () => {
      try {
        const { templateId, kvMap, dataResponse } = content;
        const rawCardData =
          kvMap && Object.keys(kvMap).length > 0 ? kvMap : dataResponse;

        // Parse JSON string values to actual objects/arrays
        // This handles array fields stored as newline-separated JSON strings
        const cardData = rawCardData ? parseFieldValues(rawCardData) : {};

        console.log(`📦 [Card ${index}] 准备发送卡片数据:`, {
          templateId,
          rawCardDataKeys: Object.keys(rawCardData || {}),
          parsedCardDataKeys: Object.keys(cardData),
          // Log types of parsed values for debugging
          parsedFieldTypes: Object.entries(cardData).map(([k, v]) => [
            k,
            typeof v,
            Array.isArray(v),
          ]),
        });

        const messagePayload = {
          channel: 'agent',
          eventId: generateEventId(),
          event: 'card',
          data: {
            code: templateId || '',
            data: cardData,
          },
        };

        const messageString = JSON.stringify(messagePayload);

        if (iframe.contentWindow) {
          iframe.contentWindow.postMessage(messageString, targetOrigin);
          console.log(`📤 [Card ${index}] 发送卡片数据到 iframe:`, {
            templateId,
            targetOrigin,
          });
        }
      } catch (error) {
        console.error(`❌ [Card ${index}] 发送卡片数据失败:`, error);
      }
    };

    let teardownAutoHeight: (() => void) | null = null;
    const handleIframeLoad = () => {
      console.log(`✅ [Card ${index}] iframe load 事件触发`);
      setIframeLoaded(true);
      // Send data immediately and retry a few times
      sendCardData();
      setTimeout(sendCardData, 100);
      setTimeout(sendCardData, 500);
      // 启动自动高度（ResizeObserver + 轮询兜底）
      teardownAutoHeight?.();
      teardownAutoHeight = setupAutoHeight(iframe, h => setIframeHeight(h));
    };

    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== targetOrigin && event.origin !== window.location.origin) {
        return;
      }

      // IMPORTANT: Check if message comes from THIS iframe, not other iframes
      // This prevents height messages from other cards affecting this card
      if (!iframe || event.source !== iframe.contentWindow) {
        return;
      }

      // Parse message data if it's a string
      let messageData = event.data;
      if (typeof messageData === 'string') {
        try {
          messageData = JSON.parse(messageData);
        } catch {
          return;
        }
      }

      // Handle resize messages - support multiple formats
      if (messageData && typeof messageData === 'object') {
        // Format 1: { type: 'resize', height: number }
        if (messageData.type === 'resize' && typeof messageData.height === 'number') {
          const newHeight = messageData.height;
          if (newHeight > 50) {
            setIframeHeight(newHeight + 16);
          }
        }
        // Format 2: { event: 'resize', data: { height: number } }
        else if (messageData.event === 'resize' && messageData.data?.height) {
          const newHeight = messageData.data.height;
          if (typeof newHeight === 'number' && newHeight > 50) {
            setIframeHeight(newHeight + 16);
          }
        }
        // Format 3: { channel: 'agent', event: 'cardHeight', data: { height: number } }
        else if (messageData.channel === 'agent' && messageData.event === 'cardHeight') {
          const newHeight = messageData.data?.height;
          if (typeof newHeight === 'number' && newHeight > 50) {
            setIframeHeight(newHeight + 16);
          }
        }
      }
    };

    iframe.addEventListener('load', handleIframeLoad);
    window.addEventListener('message', handleMessage);

    return () => {
      iframe.removeEventListener('load', handleIframeLoad);
      window.removeEventListener('message', handleMessage);
      teardownAutoHeight?.();
    };
  }, [content, targetOrigin, index]);

  return (
    <div className="single-card-wrapper">
      {/* Card iframe container - no header */}
      <div className="single-card-iframe-container">
        <iframe
          ref={iframeRef}
          src={generateIframeUrl()}
          width="100%"
          height={`${iframeHeight}px`}
          frameBorder="0"
          title={`Card Content ${index}`}
          sandbox="allow-scripts allow-same-origin allow-forms"
        />
        {/* Loading overlay while iframe loads */}
        {!iframeLoaded && (
          <div className="iframe-loading-overlay">
            <span>Loading...</span>
          </div>
        )}
      </div>
    </div>
  );
};

/**
 * Card group component - renders cards with specified layout
 */
const CardGroupContent: FC<{
  group: CardGroupInfo;
  contentMap: Map<string, SpecialContentItem>;
}> = ({ group, contentMap }) => {
  const { groupId, layout, columns, cardIds } = group;

  // Generate layout class names
  const layoutClass = `layout-${layout}`;
  const columnsClass = `columns-${columns}`;

  // Get cards in this group
  const groupCards = cardIds
    .map(cardId => contentMap.get(cardId))
    .filter((item): item is SpecialContentItem => item !== undefined);

  if (groupCards.length === 0) {
    return null;
  }

  return (
    <div
      className={`card-group ${layoutClass} ${columnsClass}`}
      data-group-id={groupId}
      data-layout={layout}
      data-columns={columns}
    >
      <div className="card-group-cards">
        {groupCards.map((content, index) => (
          <div
            key={content.cardId || `${content.templateId}-${index}`}
            className="card-group-item"
            data-index={index}
          >
            <SingleCardContent content={content} index={index} />
          </div>
        ))}
      </div>
    </div>
  );
};

/**
 * 特殊answer消息组件，用于处理包含displayResponseType的消息
 * 支持原生显示和iframe嵌套显示两种模式
 * 支持多张卡片渲染
 * 支持卡片组布局（horizontal/vertical/waterfall）
 */
export const SpecialAnswerContent: FC<SpecialAnswerContentProps> = props => {
  const { message, contentList, groups, rawContent, ...restProps } = props;

  // Create a message with rawContent for TextContent rendering
  const rawContentMessage = useMemo(() => {
    if (!rawContent) {
      return null;
    }
    return {
      ...message,
      content: rawContent,
    };
  }, [message, rawContent]);

  // 获取所有有 displayResponseType 的内容（支持多张卡片）
  const specialContents = contentList?.filter(item => item.displayResponseType) || [];

  // Build a map of cardId -> content for group rendering
  const contentMap = useMemo(() => {
    const map = new Map<string, SpecialContentItem>();
    specialContents.forEach((item, index) => {
      // Use cardId if available, otherwise generate a fallback key
      const key = item.cardId || `${item.templateId}-${index}`;
      map.set(key, item);
    });
    return map;
  }, [specialContents]);

  // Identify cards that are in groups vs ungrouped cards
  const { groupedCardIds, ungroupedCards } = useMemo(() => {
    const groupedIds = new Set<string>();
    if (groups && groups.length > 0) {
      groups.forEach(group => {
        group.cardIds.forEach(cardId => groupedIds.add(cardId));
      });
    }

    // Find cards not in any group
    const ungrouped = specialContents.filter(item => {
      const key = item.cardId || `${item.templateId}-${specialContents.indexOf(item)}`;
      return !groupedIds.has(key);
    });

    return { groupedCardIds: groupedIds, ungroupedCards: ungrouped };
  }, [groups, specialContents]);

  console.log('🃏 SpecialAnswerContent 渲染:', {
    totalItems: contentList?.length || 0,
    specialCardsCount: specialContents.length,
    groupsCount: groups?.length || 0,
    ungroupedCount: ungroupedCards.length,
    templateIds: specialContents.map(c => c.templateId),
    hasRawContent: !!rawContent,
    rawContentLength: rawContent?.length || 0,
  });

  if (specialContents.length === 0) {
    // 如果没有特殊内容，回退到普通文本组件
    return <TextContent message={message} {...restProps} />;
  }

  // Render cards with group layout support
  const renderCards = () => {
    const elements: JSX.Element[] = [];

    // Render grouped cards first (in group order)
    if (groups && groups.length > 0) {
      groups.forEach((group, groupIndex) => {
        elements.push(
          <CardGroupContent
            key={`group-${group.groupId || groupIndex}`}
            group={group}
            contentMap={contentMap}
          />
        );
      });
    }

    // Render ungrouped cards (if any)
    if (ungroupedCards.length > 0) {
      ungroupedCards.forEach((content, index) => {
        elements.push(
          <SingleCardContent
            key={content.cardId || `ungrouped-${content.templateId}-${index}`}
            content={content}
            index={index}
          />
        );
      });
    }

    return elements;
  };

  return (
    <div className="special-answer-content">
      {/* 先渲染原始文本内容（非卡片部分） */}
      {rawContentMessage && (
        <div className="raw-content-section">
          <TextContent message={rawContentMessage} {...restProps} />
        </div>
      )}
      {/* 渲染所有卡片（支持组布局） */}
      {renderCards()}

      {/* Debug info in development */}
      {process.env.NODE_ENV === 'development' && (
        <details className="special-answer-debug">
          <summary>Debug Info</summary>
          <pre>
            {JSON.stringify(
              {
                cardsCount: specialContents.length,
                groupsCount: groups?.length || 0,
                templateIds: specialContents.map(c => c.templateId),
              },
              null,
              2,
            )}
          </pre>
        </details>
      )}
    </div>
  );
};

SpecialAnswerContent.displayName = 'SpecialAnswerContent';
