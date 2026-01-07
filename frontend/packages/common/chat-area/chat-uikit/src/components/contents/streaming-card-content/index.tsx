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
  type CardStreamPayload,
  StreamingCardStatus,
  getStreamingCardStateManager,
  buildCardDataForIframe,
} from '@coze-common/chat-core';

// Message type for cardstream communication with iframe
interface CardStreamMessage {
  channel: 'agent';
  eventId: string;
  event: 'cardstream';
  data: CardStreamPayload;
}

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
 * Parse JSON string values in fields to actual objects/arrays
 * Handles multiple formats:
 * 1. Single JSON object: "{...}"
 * 2. Single JSON array: "[...]"
 * 3. Newline-separated JSON objects: "{...}\n{...}\n" -> [{...}, {...}]
 *
 * This ensures iframe receives data in the same format as special-answer-content
 * @param fields Record<string, string> from StreamingCardState
 * @returns Record<string, unknown> with parsed JSON values
 */
const parseFieldValues = (fields: Record<string, string>): Record<string, unknown> => {
  const parsed: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(fields)) {
    if (!value) {
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
 * Streaming Card Content Component
 *
 * Renders a card that updates in real-time as streaming data arrives.
 * Uses iframe for card display and postMessage for data communication.
 */
export const StreamingCardContent: FC<StreamingCardContentProps> = props => {
  const { cardId, messageId, onCardComplete } = props;

  const [cardState, setCardState] = useState<StreamingCardState | null>(null);
  const [iframeHeight, setIframeHeight] = useState<number>(320); // Default height for card display
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

    // Parse JSON string values to actual objects/arrays for iframe compatibility
    // This matches the format used by special-answer-content
    const parsedFields = parseFieldValues(cardState.fields || {});

    // Use same format as special-answer-content for compatibility
    const messagePayload = {
      channel: 'agent',
      eventId: generateEventId(),
      event: 'card',
      data: {
        code: cardState.templateId || '',
        data: parsedFields,
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
        rawFields: cardState.fields,
        parsedFields: parsedFields,
        fieldKeys: Object.keys(cardState.fields),
        parsedFieldTypes: Object.entries(parsedFields).map(([k, v]) => [k, typeof v, Array.isArray(v)]),
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

  // Send cardstream message to iframe
  // Format: sendMessage('cardstream', {groups: []})
  const sendCardStreamToIframe = useCallback((payload: CardStreamPayload) => {
    const iframe = iframeRef.current;
    if (!iframe || !iframe.contentWindow || !iframeLoaded) {
      console.log('[StreamingCard] Skipping cardstream send - not ready');
      return;
    }

    const messagePayload: CardStreamMessage = {
      channel: 'agent',
      eventId: generateEventId(),
      event: 'cardstream',
      data: payload,
    };

    const targetOrigin = getTargetOrigin();

    try {
      iframe.contentWindow.postMessage(JSON.stringify(messagePayload), targetOrigin);
      console.log('[StreamingCard] 📤 Sent cardstream to iframe:', {
        messageId,
        groupCount: payload.groups.length,
        groups: payload.groups.map(g => ({
          id: g.id,
          code: g.code,
          columns: g.data.columns,
          childCount: g.children.length,
          children: g.children.map(c => ({
            id: c.id,
            code: c.code,
            dataKeys: Object.keys(c.data),
          })),
        })),
        targetOrigin,
      });
    } catch (error) {
      console.error('[StreamingCard] ❌ Failed to send cardstream to iframe:', error);
    }
  }, [iframeLoaded, getTargetOrigin, messageId]);

  // Subscribe to cardstream updates and send to iframe
  useEffect(() => {
    if (!iframeLoaded) {
      return;
    }

    const manager = getStreamingCardStateManager();

    // Subscribe to cardstream payload updates
    const unsubscribe = manager.subscribeToCardStream((updatedMessageId, payload) => {
      // Only send updates for this message
      if (updatedMessageId === messageId) {
        console.log('[StreamingCard] 🔄 Received cardstream update for message:', {
          messageId: updatedMessageId,
          groupCount: payload.groups.length,
        });
        sendCardStreamToIframe(payload);
      }
    });

    // Send initial cardstream payload if there are groups
    const initialPayload = manager.buildGroupsPayload(messageId);
    if (initialPayload.groups.length > 0) {
      console.log('[StreamingCard] 📤 Sending initial cardstream payload');
      sendCardStreamToIframe(initialPayload);
    }

    return unsubscribe;
  }, [messageId, iframeLoaded, sendCardStreamToIframe]);

  // Handle iframe load event
  const handleIframeLoad = useCallback(() => {
    console.log('[StreamingCard] Iframe loaded');
    setIframeLoaded(true);

    // Send data immediately and retry a few times to ensure delivery
    const iframe = iframeRef.current;
    if (iframe && iframe.contentWindow && cardState) {
      const sendData = () => {
        // Parse JSON string values to actual objects/arrays for iframe compatibility
        const parsedFields = parseFieldValues(cardState.fields || {});

        const messagePayload = {
          channel: 'agent',
          eventId: generateEventId(),
          event: 'card',
          data: {
            code: cardState.templateId || '',
            data: parsedFields,
          },
        };
        const targetOrigin = getTargetOrigin();
        iframe.contentWindow?.postMessage(JSON.stringify(messagePayload), targetOrigin);
        console.log('[StreamingCard] 📤 Initial data sent on load:', {
          templateId: cardState.templateId,
          fieldKeys: Object.keys(cardState.fields || {}),
          parsedFieldTypes: Object.entries(parsedFields).map(([k, v]) => [k, typeof v, Array.isArray(v)]),
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

      // IMPORTANT: Check if message comes from THIS iframe, not other iframes
      // This prevents height messages from other cards affecting this card
      const iframe = iframeRef.current;
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

    const iframeUrl = spaceId
      ? `${baseUrl}?spaceId=${spaceId}&streaming=true`
      : `${baseUrl}?streaming=true`;

    console.log('[StreamingCard] 🔗 Generated iframe URL:', {
      cardId,
      templateId: cardState?.templateId,
      baseUrl,
      spaceId,
      iframeUrl,
    });

    return iframeUrl;
  }, [cardId, cardState?.templateId]);

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
      {/* Card header - only show during streaming, hide when complete */}
      {isStreaming && (
        <div className="streaming-card-header streaming-only">
          <Spin size="small" />
          <span className="streaming-text">Streaming...</span>
        </div>
      )}

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
