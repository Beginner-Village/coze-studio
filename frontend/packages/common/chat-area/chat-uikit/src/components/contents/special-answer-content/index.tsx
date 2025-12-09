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

import { useState, useRef, useEffect, type FC } from 'react';
import { Button } from '@coze-arch/coze-design';
import { type IBaseContentProps } from '@coze-common/chat-uikit-shared';

import { TextContent } from '../text-content';
import './index.less';

export interface SpecialAnswerContentProps extends IBaseContentProps {
  contentList?: Array<{
    displayResponseType?: string;
    templateId?: string;
    kvMap?: Record<string, any>;
    dataResponse?: Record<string, any>;
  }>;
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
 * 特殊answer消息组件，用于处理包含displayResponseType的消息
 * 支持原生显示和iframe嵌套显示两种模式
 */
export const SpecialAnswerContent: FC<SpecialAnswerContentProps> = props => {
  const { message, contentList, ...restProps } = props;
  const [viewMode, setViewMode] = useState<'iframe' | 'native'>('iframe'); // 默认显示卡片
  const [iframeHeight, setIframeHeight] = useState<number>(600); // 默认高度，使用手机比例
  const iframeRef = useRef<HTMLIFrameElement>(null);

  // 检查是否有displayResponseType内容
  const specialContent = contentList?.find(item => item.displayResponseType);

  // 监听iframe加载完成，发送卡片数据并处理高度调整
  useEffect(() => {
    const iframe = iframeRef.current;
    // 只在 iframe 模式下且有特殊内容时才处理
    if (!iframe || !specialContent || viewMode !== 'iframe') {
      console.log('⏭️ 跳过 iframe 事件设置:', {
        hasIframe: !!iframe,
        hasContent: !!specialContent,
        viewMode,
      });
      return;
    }

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
      console.warn('⚠️ 解析 Card URL 失败，使用当前 Origin:', e);
      targetOrigin = window.location.origin;
    }

    console.log('🔧 设置 iframe 事件监听器...', {
      src: iframe.src,
      targetOrigin,
    });

    const sendCardData = () => {
      try {
        // 准备卡片数据
        const { templateId, kvMap, dataResponse } = specialContent;
        const cardData =
          kvMap && Object.keys(kvMap).length > 0 ? kvMap : dataResponse;

        console.log('📦 准备发送卡片数据:', {
          templateId,
          hasKvMap: !!kvMap,
          hasDataResponse: !!dataResponse,
          cardDataKeys: Object.keys(cardData || {}),
        });

        // 构建 postMessage 消息结构
        const messagePayload = {
          channel: 'agent', // 固定标识
          eventId: generateEventId(), // 时间戳 + 自增整数
          event: 'card', // 渲染卡片消息
          data: {
            code: templateId || '', // 卡片模板ID
            data: cardData || {}, // 卡片数据
          },
        };

        // 序列化为 JSON 字符串后发送
        const messageString = JSON.stringify(messagePayload);

        // 通过 postMessage 发送卡片数据到 iframe
        if (iframe.contentWindow) {
          iframe.contentWindow.postMessage(messageString, targetOrigin);

          console.log('📤 发送卡片数据到 iframe:', {
            eventId: messagePayload.eventId,
            templateId,
            targetOrigin,
            dataSize: messageString.length,
          });
          console.log('📋 完整消息:', messagePayload);
        } else {
          console.warn('⚠️ iframe.contentWindow 不可用');
        }

        // 尝试获取iframe内容的高度（仅限同域情况）
        try {
          const iframeDocument =
            iframe.contentDocument || iframe.contentWindow?.document;
          if (iframeDocument) {
            const body = iframeDocument.body;
            const html = iframeDocument.documentElement;
            const height = Math.max(
              body?.scrollHeight || 0,
              body?.offsetHeight || 0,
              html?.clientHeight || 0,
              html?.scrollHeight || 0,
              html?.offsetHeight || 0,
            );

            if (height > 100) {
              setIframeHeight(height + 20);
              console.log('📐 同域iframe，自动调整高度:', height + 20);
            }
          } else {
            console.log('🔒 跨域iframe，等待通过 postMessage 调整高度');
          }
        } catch (crossOriginError) {
          // 跨域访问被阻止，这是正常的
          console.log('🔒 跨域iframe，无法直接获取高度（正常现象）');
        }
      } catch (error) {
        console.error('❌ 发送卡片数据或调整高度失败:', error);
      }
    };

    const handleIframeLoad = () => {
      console.log('✅ iframe load 事件触发', { src: iframe.src });
      sendCardData();
    };

    // 监听来自iframe的消息（用于跨域高度获取）
    const handleMessage = (event: MessageEvent) => {
      // 验证消息来源（安全考虑）
      if (event.origin !== targetOrigin) {
        // 如果是同域，origin 可能是 null (本地文件) 或 与 window.location.origin 相同
        // 这里主要防止恶意站点的消息
        // 对于相对路径（同域），我们允许 event.origin === window.location.origin
        if (
          targetOrigin === window.location.origin &&
          event.origin === window.location.origin
        ) {
          // pass
        } else {
          return;
        }
      }

      if (
        event.data &&
        typeof event.data === 'object' &&
        event.data.type === 'resize'
      ) {
        const newHeight = event.data.height;
        if (typeof newHeight === 'number' && newHeight > 100) {
          setIframeHeight(newHeight + 20);
          console.log('📐 通过postMessage调整iframe高度:', newHeight + 20);
        }
      }
    };

    // 绑定事件监听器
    iframe.addEventListener('load', handleIframeLoad);
    window.addEventListener('message', handleMessage);

    return () => {
      console.log('🧹 清理 iframe 事件监听器');
      iframe.removeEventListener('load', handleIframeLoad);
      window.removeEventListener('message', handleMessage);
    };
  }, [specialContent, viewMode]);

  if (!specialContent) {
    // 如果没有特殊内容，回退到普通文本组件
    return <TextContent message={message} {...restProps} />;
  }

  // 生成iframe URL（仅包含 spaceId 参数，卡片数据通过 postMessage 传递）
  const generateIframeUrl = () => {
    const baseUrl =
      window.APP_CONFIG?.CARD_URL ||
      'https://agent.finmall.com/agent-h5-web/card/index.html';

    // 从 URL 中提取 spaceId 参数
    // 支持两种格式：/space/{space_id}/... 或 ?space_id=xxx
    let spaceId = '';

    try {
      // 方式1：从路径参数中提取（如 /space/123456/agent）
      const pathMatch = window.location.pathname.match(/\/space\/([^\/]+)/);
      if (pathMatch && pathMatch[1]) {
        spaceId = pathMatch[1];
      }

      // 方式2：从查询参数中提取（作为备选）
      if (!spaceId) {
        const urlParams = new URLSearchParams(window.location.search);
        spaceId = urlParams.get('space_id') || urlParams.get('spaceId') || '';
      }
    } catch (error) {
      console.warn('⚠️ 无法从 URL 获取 spaceId:', error);
    }

    if (spaceId) {
      const iframeUrl = `${baseUrl}?spaceId=${spaceId}`;
      console.log(
        '🔗 iframe链接（含spaceId）:',
        iframeUrl,
        '| spaceId:',
        spaceId,
      );
      return iframeUrl;
    }

    console.log('🔗 iframe链接（无spaceId）:', baseUrl);
    return baseUrl;
  };

  return (
    <div className="special-answer-content">
      {/* 内容区域 */}
      <div className="answer-content">
        {viewMode === 'native' ? (
          <div className="special-answer-native">
            {/* 显示原始消息内容 */}
            <TextContent message={message} {...restProps} />

            {/* 显示特殊内容的JSON数据（调试用） */}
            <div className="special-answer-data">
              <details>
                <summary>原始数据</summary>
                <pre>{JSON.stringify(specialContent, null, 2)}</pre>
              </details>
            </div>
          </div>
        ) : (
          <div className="special-answer-iframe">
            <iframe
              ref={iframeRef}
              src={generateIframeUrl()}
              width="100%"
              height={`${iframeHeight}px`}
              frameBorder="0"
              title="Special Answer Content"
              sandbox="allow-scripts allow-same-origin allow-forms"
            />
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
            卡片
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
