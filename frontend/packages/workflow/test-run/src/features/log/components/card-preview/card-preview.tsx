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

import React, { useState, useRef, useEffect, type FC } from 'react';

import { I18n } from '@coze-arch/i18n';
import { IconCozWarningCircle } from '@coze-arch/coze-design/icons';

import { extractCardContentList, type CardContentItem } from './utils';

import styles from './card-preview.module.less';

// 常量定义
const DEFAULT_IFRAME_HEIGHT = 400;
const MIN_HEIGHT_THRESHOLD = 100;
const HEIGHT_PADDING = 20;
const JSON_INDENT = 2;
const DEFAULT_CARD_URL =
  'https://agent.finmall.com/agent-h5-web/card/index.html';

declare global {
  interface Window {
    APP_CONFIG?: {
      CARD_URL?: string;
    };
  }
}

export interface CardPreviewProps {
  /** 输出数据，包含 contentList */
  data: Record<string, unknown>;
  /** 自定义类名 */
  className?: string;
}

// eventId 生成器：时间戳 + 自增整数
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
    console.warn('⚠️ 解析 Card URL 失败，使用当前 Origin:', error);
  }
  return window.location.origin;
};

/**
 * 从 URL 中提取 spaceId
 */
const extractSpaceId = (): string => {
  try {
    // 从路径参数中提取（如 /space/123456/agent）
    const pathMatch = window.location.pathname.match(/\/space\/([^/]+)/);
    if (pathMatch?.[1]) {
      return pathMatch[1];
    }
    // 从查询参数中提取
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get('space_id') || urlParams.get('spaceId') || '';
  } catch (error) {
    console.warn('⚠️ 无法从 URL 获取 spaceId:', error);
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
 * 发送卡片数据到 iframe
 */
const sendCardDataToIframe = (
  iframe: HTMLIFrameElement,
  content: CardContentItem,
  targetOrigin: string,
): void => {
  const { templateId, kvMap, dataResponse } = content;
  const cardData =
    kvMap && Object.keys(kvMap).length > 0 ? kvMap : dataResponse;

  const messagePayload = {
    channel: 'agent',
    eventId: generateEventId(),
    event: 'card',
    data: {
      code: templateId || '',
      data: cardData || {},
    },
  };

  if (iframe.contentWindow) {
    iframe.contentWindow.postMessage(
      JSON.stringify(messagePayload),
      targetOrigin,
    );
    console.log('📤 [CardPreview] 发送卡片数据:', { templateId, targetOrigin });
  }
};

/**
 * 尝试获取 iframe 内容高度（仅限同域）
 */
const tryGetIframeHeight = (iframe: HTMLIFrameElement): number | null => {
  try {
    const iframeDocument =
      iframe.contentDocument || iframe.contentWindow?.document;
    if (!iframeDocument) {
      return null;
    }

    const { body } = iframeDocument;
    const { documentElement: html } = iframeDocument;
    const height = Math.max(
      body?.scrollHeight || 0,
      body?.offsetHeight || 0,
      html?.clientHeight || 0,
      html?.scrollHeight || 0,
      html?.offsetHeight || 0,
    );

    return height > MIN_HEIGHT_THRESHOLD ? height + HEIGHT_PADDING : null;
  } catch {
    // 跨域访问被阻止，这是正常的，静默处理
    return null;
  }
};

/**
 * 卡片预览组件 - 用于工作流试运行时渲染卡片输出
 * 复用智能体测试界面的 iframe 渲染逻辑
 */
export const CardPreview: FC<CardPreviewProps> = ({ data, className }) => {
  const [viewMode, setViewMode] = useState<'iframe' | 'json'>('iframe');
  const [iframeHeight, setIframeHeight] = useState<number>(
    DEFAULT_IFRAME_HEIGHT,
  );
  const [renderError, setRenderError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  // 提取卡片内容
  const contentList = extractCardContentList(data);
  const specialContent = contentList?.[0];

  // 监听 iframe 加载完成，发送卡片数据
  useEffect(() => {
    const iframe = iframeRef.current;
    if (!iframe || !specialContent || viewMode !== 'iframe') {
      return undefined;
    }

    // 重置状态
    setRenderError(null);
    setIsLoading(true);

    const cardUrl = getCardUrl();
    const targetOrigin = getTargetOrigin(cardUrl);

    const handleIframeLoad = () => {
      setIsLoading(false);
      sendCardDataToIframe(iframe, specialContent, targetOrigin);

      // 尝试获取高度
      const height = tryGetIframeHeight(iframe);
      if (height) {
        setIframeHeight(height);
      }
    };

    // 监听来自 iframe 的消息（用于跨域高度获取和错误处理）
    const handleMessage = (event: MessageEvent) => {
      // 验证消息来源
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

      // 处理 resize 消息
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
          I18n.t('workflow_testrun_card_render_error', {}, 'Card render failed');
        setRenderError(errorMsg);
        console.error('[CardPreview] Render error from iframe:', errorMsg);
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
  }, [specialContent, viewMode]);

  if (!specialContent) {
    return null;
  }

  return (
    <div className={`${styles['card-preview']} ${className || ''}`}>
      {/* 标题栏 */}
      <div className={styles['card-preview-header']}>
        <span className={styles['card-preview-title']}>
          {I18n.t('workflow_testrun_card_preview', undefined, '卡片预览')}
        </span>
        <div className={styles['view-mode-toggle']}>
          <div
            className={`${styles['toggle-option']} ${viewMode === 'iframe' ? styles.active : ''}`}
            onClick={() => setViewMode('iframe')}
          >
            {I18n.t('workflow_testrun_card_mode', undefined, '卡片')}
          </div>
          <div className={styles['toggle-divider']} />
          <div
            className={`${styles['toggle-option']} ${viewMode === 'json' ? styles.active : ''}`}
            onClick={() => setViewMode('json')}
          >
            {I18n.t('workflow_testrun_json_mode', undefined, 'JSON')}
          </div>
        </div>
      </div>

      {/* 内容区域 */}
      <div className={styles['card-preview-content']}>
        {viewMode === 'iframe' ? (
          <>
            {/* 错误显示 */}
            {renderError && (
              <div className={styles['card-error-container']}>
                <div className={styles['card-error-header']}>
                  <IconCozWarningCircle className={styles['card-error-icon']} />
                  <span className={styles['card-error-title']}>
                    {I18n.t('workflow_testrun_card_render_error_title', undefined, 'Card Render Error')}
                  </span>
                </div>
                <div className={styles['card-error-message']}>
                  {renderError}
                </div>
                {specialContent?.templateId && (
                  <div className={styles['card-error-details']}>
                    Template ID: {specialContent.templateId}
                    {specialContent?.templateName && `\nTemplate Name: ${specialContent.templateName}`}
                  </div>
                )}
              </div>
            )}
            {/* iframe 容器 */}
            {!renderError && (
              <div className={styles['card-iframe-container']}>
                {isLoading && (
                  <div className={styles['card-loading']}>
                    {I18n.t('workflow_testrun_card_loading', undefined, 'Loading card...')}
                  </div>
                )}
                <iframe
                  ref={iframeRef}
                  src={generateIframeUrl()}
                  width="100%"
                  height={`${iframeHeight}px`}
                  frameBorder="0"
                  title="Card Preview"
                  sandbox="allow-scripts allow-same-origin allow-forms"
                  style={{ display: isLoading ? 'none' : 'block' }}
                />
              </div>
            )}
          </>
        ) : (
          <div className={styles['card-json-container']}>
            <pre>{JSON.stringify(specialContent, null, JSON_INDENT)}</pre>
          </div>
        )}
      </div>
    </div>
  );
};

CardPreview.displayName = 'CardPreview';
