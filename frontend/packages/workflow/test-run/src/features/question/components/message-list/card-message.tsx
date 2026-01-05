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

import React, { useState, useRef, useEffect } from 'react';

import { I18n } from '@coze-arch/i18n';
import { IconCozWarningCircle } from '@coze-arch/coze-design/icons';

import { type ReceivedMessage } from '../../types';

import styles from './card-message.module.less';

// 常量定义
const DEFAULT_IFRAME_HEIGHT = 300;
const MIN_HEIGHT_THRESHOLD = 50;
const HEIGHT_PADDING = 10;
const DEFAULT_CARD_URL =
  'https://agent.finmall.com/agent-h5-web/card/index.html';

declare global {
  interface Window {
    APP_CONFIG?: {
      CARD_URL?: string;
    };
  }
}

interface CardMessageProps {
  message: ReceivedMessage;
}

// eventId 生成器
let eventIdCounter = 0;
const generateEventId = (): string => {
  const timestamp = Date.now();
  const counter = ++eventIdCounter;
  return `${timestamp}_${counter}`;
};

/**
 * 获取卡片 URL 配置
 */
const getCardUrl = (): string =>
  window.APP_CONFIG?.CARD_URL || DEFAULT_CARD_URL;

/**
 * 计算目标 Origin
 */
const getTargetOrigin = (cardUrl: string): string => {
  try {
    if (cardUrl.startsWith('http')) {
      return new URL(cardUrl).origin;
    }
  } catch (error) {
    console.warn('[CardMessage] URL parse error:', error);
  }
  return window.location.origin;
};

/**
 * 从 URL 中提取 spaceId
 */
const extractSpaceId = (): string => {
  try {
    const pathMatch = window.location.pathname.match(/\/space\/([^/]+)/);
    if (pathMatch?.[1]) {
      return pathMatch[1];
    }
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get('space_id') || urlParams.get('spaceId') || '';
  } catch {
    return '';
  }
};

/**
 * 生成 iframe URL
 */
const generateIframeUrl = (): string => {
  const baseUrl = getCardUrl();
  const spaceId = extractSpaceId();
  return spaceId ? `${baseUrl}?spaceId=${spaceId}` : baseUrl;
};

/**
 * 卡片消息组件 - 用于 chatflow 聊天界面渲染卡片
 */
export const CardMessage: React.FC<CardMessageProps> = ({ message }) => {
  const [iframeHeight, setIframeHeight] = useState<number>(DEFAULT_IFRAME_HEIGHT);
  const [renderError, setRenderError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const { cardData } = message;

  useEffect(() => {
    const iframe = iframeRef.current;
    if (!iframe || !cardData) {
      return undefined;
    }

    const cardUrl = getCardUrl();
    const targetOrigin = getTargetOrigin(cardUrl);

    const sendCardData = () => {
      const { templateId, kvMap, dataResponse } = cardData;
      const data = kvMap && Object.keys(kvMap).length > 0 ? kvMap : dataResponse;

      const messagePayload = {
        channel: 'agent',
        eventId: generateEventId(),
        event: 'card',
        data: {
          code: templateId || '',
          data: data || {},
        },
      };

      if (iframe.contentWindow) {
        iframe.contentWindow.postMessage(
          JSON.stringify(messagePayload),
          targetOrigin,
        );
        console.log('[CardMessage] Sent card data:', { templateId, targetOrigin });
      }
    };

    const handleIframeLoad = () => {
      setIsLoading(false);
      sendCardData();
    };

    // 监听来自 iframe 的消息
    const handleMessage = (event: MessageEvent) => {
      const isValidOrigin =
        event.origin === targetOrigin ||
        (targetOrigin === window.location.origin &&
          event.origin === window.location.origin);

      if (!isValidOrigin) {
        return;
      }

      let messageData = event.data;
      if (typeof messageData === 'string') {
        try {
          messageData = JSON.parse(messageData);
        } catch {
          return;
        }
      }

      if (!messageData || typeof messageData !== 'object') {
        return;
      }

      // 处理高度调整消息
      if (messageData.type === 'resize' || messageData.event === 'resize') {
        const newHeight = messageData.height || messageData.data?.height;
        if (typeof newHeight === 'number' && newHeight > MIN_HEIGHT_THRESHOLD) {
          setIframeHeight(newHeight + HEIGHT_PADDING);
        }
      }

      // 处理错误消息
      if (messageData.type === 'error' || messageData.event === 'error') {
        const errorMsg =
          messageData.message ||
          messageData.error ||
          messageData.data?.message ||
          messageData.data?.error ||
          I18n.t('workflow_chatflow_card_render_error', {}, 'Card render error');
        setRenderError(errorMsg);
        console.error('[CardMessage] Render error from iframe:', errorMsg);
      }

      // 处理渲染成功消息（清除之前的错误）
      if (messageData.type === 'rendered' || messageData.event === 'rendered') {
        setRenderError(null);
      }
    };

    iframe.addEventListener('load', handleIframeLoad);
    window.addEventListener('message', handleMessage);

    return () => {
      iframe.removeEventListener('load', handleIframeLoad);
      window.removeEventListener('message', handleMessage);
    };
  }, [cardData]);

  if (!cardData) {
    return null;
  }

  // 显示错误信息
  if (renderError) {
    return (
      <div className={styles['card-error']}>
        <div className={styles['card-error-header']}>
          <IconCozWarningCircle className={styles['card-error-icon']} />
          <span className={styles['card-error-title']}>
            {I18n.t('workflow_chatflow_card_error_title', {}, 'Card Error')}
          </span>
        </div>
        <div className={styles['card-error-content']}>
          {renderError}
        </div>
        {cardData.templateId && (
          <div className={styles['card-error-code']}>
            Template: {cardData.templateId}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className={styles['card-message']}>
      <div className={styles['card-iframe-container']}>
        {isLoading && (
          <div className={styles['card-loading']}>
            {I18n.t('workflow_chatflow_card_loading', {}, 'Loading card...')}
          </div>
        )}
        <iframe
          ref={iframeRef}
          src={generateIframeUrl()}
          width="100%"
          height={`${iframeHeight}px`}
          title="Card Message"
          sandbox="allow-scripts allow-same-origin allow-forms"
          style={{ display: isLoading ? 'none' : 'block' }}
        />
      </div>
    </div>
  );
};

CardMessage.displayName = 'CardMessage';
