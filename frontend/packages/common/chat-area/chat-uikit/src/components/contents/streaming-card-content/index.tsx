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
import { Spin } from '@coze-arch/coze-design';

import {
  type StreamingCardState,
  StreamingCardStatus,
  getStreamingCardStateManager,
  buildCardDataForIframe,
} from '@coze-common/chat-core';

import './index.less';

export interface StreamingCardContentProps {
  cardId: string;
  messageId: string;
  onCardComplete?: (cardId: string, data: Record<string, string>) => void;
}

declare global {
  interface Window {
    APP_CONFIG?: {
      CARD_URL?: string;
    };
  }
}

// eventId generator: timestamp + auto-increment integer
let eventIdCounter = 0;
const generateEventId = (): string => {
  const timestamp = Date.now();
  const counter = ++eventIdCounter;
  return `${timestamp}_${counter}`;
};

/**
 * Streaming Card Content Component
 *
 * Renders a card that updates in real-time as streaming data arrives.
 * Uses iframe for card display and postMessage for data communication.
 */
export const StreamingCardContent: FC<StreamingCardContentProps> = props => {
  const { cardId, messageId, onCardComplete } = props;

  const [cardState, setCardState] = useState<StreamingCardState | null>(null);
  const [iframeHeight, setIframeHeight] = useState<number>(120); // Lower default height
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const lastSentDataRef = useRef<string>('');

  // Get initial card state
  useEffect(() => {
    const manager = getStreamingCardStateManager();
    const initialState = manager.getCard(cardId);
    if (initialState) {
      setCardState(initialState);
    }
  }, [cardId]);

  // Subscribe to card state updates
  useEffect(() => {
    const manager = getStreamingCardStateManager();

    const unsubscribe = manager.subscribe((updatedCardId, state) => {
      if (updatedCardId === cardId) {
        setCardState({ ...state });

        // Notify completion
        if (
          state.status === StreamingCardStatus.COMPLETE &&
          onCardComplete
        ) {
          onCardComplete(cardId, state.fields);
        }
      }
    });

    return unsubscribe;
  }, [cardId, onCardComplete]);

  // Calculate target origin for postMessage
  const getTargetOrigin = useCallback(() => {
    const cardUrl =
      window.APP_CONFIG?.CARD_URL ||
      'https://agent.finmall.com/agent-h5-web/card/index.html';

    try {
      if (cardUrl.startsWith('http')) {
        return new URL(cardUrl).origin;
      }
    } catch {
      console.warn('[StreamingCard] Failed to parse card URL');
    }

    return window.location.origin;
  }, []);

  // Send card data to iframe
  const sendCardDataToIframe = useCallback(() => {
    const iframe = iframeRef.current;
    if (!iframe || !iframe.contentWindow || !cardState || !iframeLoaded) {
      console.log('[StreamingCard] Skipping send - not ready:', {
        hasIframe: !!iframe,
        hasContentWindow: !!iframe?.contentWindow,
        hasCardState: !!cardState,
        iframeLoaded,
      });
      return;
    }

    // Use same format as special-answer-content for compatibility
    const messagePayload = {
      channel: 'agent',
      eventId: generateEventId(),
      event: 'card',
      data: {
        code: cardState.templateId || '',
        data: cardState.fields || {},
      },
    };

    const dataString = JSON.stringify(messagePayload);

    // Skip if data hasn't changed
    if (dataString === lastSentDataRef.current) {
      return;
    }
    lastSentDataRef.current = dataString;

    const targetOrigin = getTargetOrigin();

    try {
      iframe.contentWindow.postMessage(dataString, targetOrigin);
      console.log('[StreamingCard] 📤 Sent data to iframe:', {
        cardId: cardState.cardId,
        templateId: cardState.templateId,
        fields: cardState.fields,
        fieldKeys: Object.keys(cardState.fields),
        status: cardState.status,
        targetOrigin,
      });
    } catch (error) {
      console.error('[StreamingCard] ❌ Failed to send data to iframe:', error);
    }
  }, [cardState, iframeLoaded, getTargetOrigin]);

  // Send data when card state changes
  useEffect(() => {
    if (cardState && iframeLoaded) {
      sendCardDataToIframe();
    }
  }, [cardState, iframeLoaded, sendCardDataToIframe]);

  // Handle iframe load event
  const handleIframeLoad = useCallback(() => {
    console.log('[StreamingCard] Iframe loaded');
    setIframeLoaded(true);

    // Send data immediately and retry a few times to ensure delivery
    const iframe = iframeRef.current;
    if (iframe && iframe.contentWindow && cardState) {
      const sendData = () => {
        const messagePayload = {
          channel: 'agent',
          eventId: generateEventId(),
          event: 'card',
          data: {
            code: cardState.templateId || '',
            data: cardState.fields || {},
          },
        };
        const targetOrigin = getTargetOrigin();
        iframe.contentWindow?.postMessage(JSON.stringify(messagePayload), targetOrigin);
        console.log('[StreamingCard] 📤 Initial data sent on load:', {
          templateId: cardState.templateId,
          fieldKeys: Object.keys(cardState.fields || {}),
        });
      };

      // Send immediately
      sendData();
      // Retry after short delays to handle race conditions
      setTimeout(sendData, 100);
      setTimeout(sendData, 500);
    }
  }, [cardState, getTargetOrigin]);

  // Handle messages from iframe (for height adjustment)
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      const targetOrigin = getTargetOrigin();

      // Security check for origin - allow same origin and card URL origin
      if (
        event.origin !== targetOrigin &&
        event.origin !== window.location.origin
      ) {
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

    window.addEventListener('message', handleMessage);
    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [getTargetOrigin]);

  // Generate iframe URL
  const generateIframeUrl = useCallback(() => {
    const baseUrl =
      window.APP_CONFIG?.CARD_URL ||
      'https://agent.finmall.com/agent-h5-web/card/index.html';

    // Extract spaceId from current URL
    let spaceId = '';
    try {
      const pathMatch = window.location.pathname.match(/\/space\/([^/]+)/);
      if (pathMatch && pathMatch[1]) {
        spaceId = pathMatch[1];
      }

      if (!spaceId) {
        const urlParams = new URLSearchParams(window.location.search);
        spaceId = urlParams.get('space_id') || urlParams.get('spaceId') || '';
      }
    } catch {
      console.warn('[StreamingCard] Failed to extract spaceId from URL');
    }

    if (spaceId) {
      return `${baseUrl}?spaceId=${spaceId}&streaming=true`;
    }

    return `${baseUrl}?streaming=true`;
  }, []);

  // Render loading state
  if (!cardState) {
    return (
      <div className="streaming-card-content streaming-card-loading">
        <Spin size="small" />
        <span className="loading-text">Loading card...</span>
      </div>
    );
  }

  const isStreaming = cardState.status === StreamingCardStatus.STREAMING;
  const isComplete = cardState.status === StreamingCardStatus.COMPLETE;

  return (
    <div
      className={`streaming-card-content ${isStreaming ? 'streaming' : ''} ${isComplete ? 'complete' : ''}`}
    >
      {/* Card header with status indicator */}
      <div className="streaming-card-header">
        <div className="card-info">
          <span className="card-name">{cardState.templateName || 'Card'}</span>
          {cardState.currentField && isStreaming && (
            <span className="current-field">
              Updating: {cardState.currentField}
            </span>
          )}
        </div>
        <div className="card-status">
          {isStreaming && (
            <span className="status-indicator streaming">
              <Spin size="small" />
              <span>Streaming...</span>
            </span>
          )}
          {isComplete && (
            <span className="status-indicator complete">Complete</span>
          )}
        </div>
      </div>

      {/* Card iframe */}
      <div className="streaming-card-iframe-container">
        <iframe
          ref={iframeRef}
          src={generateIframeUrl()}
          width="100%"
          height={`${iframeHeight}px`}
          frameBorder="0"
          title={`Streaming Card: ${cardState.templateName}`}
          sandbox="allow-scripts allow-same-origin allow-forms"
          onLoad={handleIframeLoad}
        />

        {/* Loading overlay while iframe loads */}
        {!iframeLoaded && (
          <div className="iframe-loading-overlay">
            <Spin />
          </div>
        )}
      </div>

      {/* Debug info in development */}
      {process.env.NODE_ENV === 'development' && (
        <details className="streaming-card-debug">
          <summary>Debug Info</summary>
          <pre>
            {JSON.stringify(
              {
                cardId: cardState.cardId,
                templateId: cardState.templateId,
                status: cardState.status,
                fields: cardState.fields,
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

StreamingCardContent.displayName = 'StreamingCardContent';

export default StreamingCardContent;
