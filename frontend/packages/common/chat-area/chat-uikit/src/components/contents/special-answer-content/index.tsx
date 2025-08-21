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

/**
 * 特殊answer消息组件，用于处理包含displayResponseType的消息
 * 支持原生显示和iframe嵌套显示两种模式
 */
export const SpecialAnswerContent: FC<SpecialAnswerContentProps> = props => {
  const { message, contentList, ...restProps } = props;
  const [viewMode, setViewMode] = useState<'iframe' | 'native'>('iframe'); // 默认显示卡片
  const [iframeHeight, setIframeHeight] = useState<number>(400); // 降低默认高度
  const iframeRef = useRef<HTMLIFrameElement>(null);

  // 检查是否有displayResponseType内容
  const specialContent = contentList?.find(item => item.displayResponseType);

  // 监听iframe加载完成，自动调整高度
  useEffect(() => {
    const iframe = iframeRef.current;
    if (!iframe) return;

    const handleIframeLoad = () => {
      // 延迟一小段时间等待iframe内容完全渲染
      setTimeout(() => {
        try {
          // 尝试获取iframe内容的高度
          const iframeDocument = iframe.contentDocument || iframe.contentWindow?.document;
          if (iframeDocument) {
            const body = iframeDocument.body;
            const html = iframeDocument.documentElement;
            
            // 获取实际内容高度
            const height = Math.max(
              body?.scrollHeight || 0,
              body?.offsetHeight || 0,
              html?.clientHeight || 0,
              html?.scrollHeight || 0,
              html?.offsetHeight || 0
            );
            
            // 确保有合理的高度，但设置更严格的范围
            if (height > 50 && height < 2000) {
              const adjustedHeight = Math.max(100, Math.min(height + 40, 800)); // 限制在100-800px之间
              setIframeHeight(adjustedHeight);
              console.log('🔗 自动调整iframe高度:', `${height}px -> ${adjustedHeight}px`);
            }
          }
        } catch (error) {
          // 跨域问题，尝试通过postMessage获取高度
          console.log('无法直接获取iframe内容高度，等待postMessage通信');
        }
      }, 500); // 等待500ms让内容渲染完成
    };

    // 监听来自iframe的消息（用于跨域高度获取）
    const handleMessage = (event: MessageEvent) => {
      // 验证消息来源（安全考虑）
      if (event.origin !== 'https://agent.finmall.com') return;
      
      if (event.data && typeof event.data === 'object' && event.data.type === 'resize') {
        const newHeight = event.data.height;
        if (typeof newHeight === 'number' && newHeight > 50 && newHeight < 2000) {
          const adjustedHeight = Math.max(100, Math.min(newHeight + 40, 800)); // 限制在100-800px之间
          setIframeHeight(adjustedHeight);
          console.log('🔗 通过postMessage调整iframe高度:', `${newHeight}px -> ${adjustedHeight}px`);
        }
      }
    };

    iframe.addEventListener('load', handleIframeLoad);
    window.addEventListener('message', handleMessage);
    
    return () => {
      iframe.removeEventListener('load', handleIframeLoad);
      window.removeEventListener('message', handleMessage);
    };
  }, [specialContent]);

  if (!specialContent) {
    // 如果没有特殊内容，回退到普通文本组件
    return <TextContent message={message} {...restProps} />;
  }

  // 生成iframe URL
  const generateIframeUrl = () => {
    const { templateId, kvMap, dataResponse } = specialContent;
    const baseUrl = 'https://agent.finmall.com/agent-h5-web/card/index.html';
    
    // 优先使用kvMap，否则使用dataResponse
    const data = kvMap && Object.keys(kvMap).length > 0 ? kvMap : dataResponse;
    
    if (!templateId || !data) {
      return baseUrl;
    }

    // 编码JSON数据
    const encodedData = encodeURIComponent(JSON.stringify(data));
    const iframeUrl = `${baseUrl}?code=${templateId}&data=${encodedData}`;
    
    // 打印调试信息
    console.log('🔗 iframe链接:', iframeUrl);
    console.log('📊 使用的数据:', data);
    console.log('🏷️ templateId:', templateId);
    
    return iframeUrl;
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