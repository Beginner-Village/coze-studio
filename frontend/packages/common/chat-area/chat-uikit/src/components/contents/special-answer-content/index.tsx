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

import { useState, useRef, useEffect, useCallback, type FC } from 'react';
import { type IBaseContentProps } from '@coze-common/chat-uikit-shared';

import { TextContent } from '../text-content';
import './index.less';

export interface SpecialContentItem {
  displayResponseType?: string;
  templateId?: string;
  kvMap?: Record<string, any>;
  dataResponse?: Record<string, any>;
}

export interface SpecialAnswerContentProps extends IBaseContentProps {
  contentList?: SpecialContentItem[];
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
  const [iframeHeight, setIframeHeight] = useState<number>(600);
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
      sendCardData();
    };

    const handleMessage = (event: MessageEvent) => {
      if (event.origin !== targetOrigin && event.origin !== window.location.origin) {
        return;
      }
      if (event.data?.type === 'resize' && typeof event.data.height === 'number') {
        setIframeHeight(event.data.height + 20);
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
    <div className="single-card-iframe" style={{ marginBottom: '12px' }}>
      <iframe
        ref={iframeRef}
        src={generateIframeUrl()}
        width="100%"
        height={`${iframeHeight}px`}
        frameBorder="0"
        title={`Card Content ${index}`}
        sandbox="allow-scripts allow-same-origin allow-forms"
      />
    </div>
  );
};

/**
 * 特殊answer消息组件，用于处理包含displayResponseType的消息
 * 支持原生显示和iframe嵌套显示两种模式
 * 支持多张卡片渲染
 */
export const SpecialAnswerContent: FC<SpecialAnswerContentProps> = props => {
  const { message, contentList, ...restProps } = props;
  const [viewMode, setViewMode] = useState<'iframe' | 'native'>('iframe');

  // 获取所有有 displayResponseType 的内容（支持多张卡片）
  const specialContents = contentList?.filter(item => item.displayResponseType) || [];

  console.log('🃏 SpecialAnswerContent 渲染:', {
    totalItems: contentList?.length || 0,
    specialCardsCount: specialContents.length,
    templateIds: specialContents.map(c => c.templateId),
  });

  if (specialContents.length === 0) {
    // 如果没有特殊内容，回退到普通文本组件
    return <TextContent message={message} {...restProps} />;
  }

  return (
    <div className="special-answer-content">
      {/* 内容区域 */}
      <div className="answer-content">
        {viewMode === 'native' ? (
          <div className="special-answer-native">
            {/* 显示原始消息内容 */}
            <TextContent message={message} {...restProps} />

            {/* 显示所有特殊内容的JSON数据（调试用） */}
            <div className="special-answer-data">
              <details>
                <summary>原始数据 ({specialContents.length} 张卡片)</summary>
                <pre>{JSON.stringify(specialContents, null, 2)}</pre>
              </details>
            </div>
          </div>
        ) : (
          <div className="special-answer-iframe">
            {/* 渲染所有卡片 */}
            {specialContents.map((content, index) => (
              <SingleCardContent
                key={`${content.templateId}-${index}`}
                content={content}
                index={index}
              />
            ))}
          </div>
        )}
      </div>

      {/* 底部控制区域 */}
      <div className="answer-footer">
        <div className="view-mode-toggle">
          <div
            className={`toggle-option left ${viewMode === 'iframe' ? 'active' : ''}`}
            onClick={() => setViewMode('iframe')}
            title="卡片显示"
          >
            卡片 ({specialContents.length})
          </div>
          <div className="toggle-divider"></div>
          <div
            className={`toggle-option right ${viewMode === 'native' ? 'active' : ''}`}
            onClick={() => setViewMode('native')}
            title="原生显示"
          >
            原生
          </div>
        </div>
      </div>
    </div>
  );
};

SpecialAnswerContent.displayName = 'SpecialAnswerContent';
