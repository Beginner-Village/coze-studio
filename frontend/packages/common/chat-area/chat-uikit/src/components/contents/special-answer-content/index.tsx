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

/**
 * 单个卡片组件 - 用于渲染单张卡片的 iframe
 */
const SingleCardContent: FC<{
  content: SpecialContentItem;
  index: number;
}> = ({ content, index }) => {
  const [iframeHeight, setIframeHeight] = useState<number>(120); // Reduced from 600 to match streaming-card
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  // 计算目标 Origin
  const cardUrl =
    window.APP_CONFIG?.CARD_URL ||
    'https://agent.finmall.com/agent-h5-web/card/index.html';
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
      'https://agent.finmall.com/agent-h5-web/card/index.html';

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
        const cardData =
          kvMap && Object.keys(kvMap).length > 0 ? kvMap : dataResponse;

        console.log(`📦 [Card ${index}] 准备发送卡片数据:`, {
          templateId,
          cardDataKeys: Object.keys(cardData || {}),
        });

        const messagePayload = {
          channel: 'agent',
          eventId: generateEventId(),
          event: 'card',
          data: {
            code: templateId || '',
            data: cardData || {},
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

    const handleIframeLoad = () => {
      console.log(`✅ [Card ${index}] iframe load 事件触发`);
      setIframeLoaded(true);
      // Send data immediately and retry a few times
      sendCardData();
      setTimeout(sendCardData, 100);
      setTimeout(sendCardData, 500);
    };

    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== targetOrigin && event.origin !== window.location.origin) {
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
    };
  }, [content, targetOrigin, index]);

  return (
    <div className="single-card-wrapper">
      {/* Card header with template name */}
      <div className="single-card-header">
        <span className="card-name">{content.templateId || 'Card'}</span>
      </div>
      {/* Card iframe container */}
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
