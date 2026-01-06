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

/**
 * Streaming Card State Manager
 *
 * Manages the state of cards being rendered during streaming responses.
 * Handles card_create, card_delta, and card_done events from the backend.
 */

import { type MessageExtraInfo } from './types';

// Card event types from backend
export enum StreamingCardEventType {
  CREATE = 'card_create',
  DELTA = 'card_delta',
  DONE = 'card_done',
}

// Card status during streaming
export enum StreamingCardStatus {
  CREATING = 'creating', // Initial creation
  STREAMING = 'streaming', // Receiving field updates
  COMPLETE = 'complete', // Done streaming
  ERROR = 'error', // Error occurred
}

// Single card state
export interface StreamingCardState {
  cardId: string;
  templateId: string;
  templateName: string;
  status: StreamingCardStatus;
  fields: Record<string, string>; // Accumulated field values
  currentField?: string; // Current field being updated
  createdAt: number;
  updatedAt: number;
  error?: string;
}

// Event listener callback
export type StreamingCardEventCallback = (
  cardId: string,
  state: StreamingCardState,
) => void;

/**
 * StreamingCardStateManager
 *
 * Tracks and manages streaming card states during SSE message flow.
 * Provides event subscription for UI components to react to card updates.
 */
export class StreamingCardStateManager {
  private cards: Map<string, StreamingCardState> = new Map();
  private messageCardMap: Map<string, Set<string>> = new Map(); // message_id -> card_ids
  private listeners: Set<StreamingCardEventCallback> = new Set();

  /**
   * Process a message with card metadata
   * @param messageId The message ID
   * @param extraInfo The message extra_info containing card metadata
   * @param content The message content (may contain field data for deltas)
   */
  processCardEvent(
    messageId: string,
    extraInfo: MessageExtraInfo,
    content: string,
  ): StreamingCardState | null {
    const { ynet_type, card_id, template_id, template_name, card_field } =
      extraInfo;

    // Skip if not a card event
    if (!ynet_type || !card_id) {
      return null;
    }

    let state: StreamingCardState | null = null;

    switch (ynet_type) {
      case StreamingCardEventType.CREATE:
        state = this.handleCardCreate(
          messageId,
          card_id,
          template_id || '',
          template_name || '',
        );
        break;

      case StreamingCardEventType.DELTA:
        state = this.handleCardDelta(card_id, card_field || '', content);
        break;

      case StreamingCardEventType.DONE:
        state = this.handleCardDone(card_id, content);
        break;

      default:
        console.warn(
          `[StreamingCardStateManager] Unknown ynet_type: ${ynet_type}`,
        );
    }

    return state;
  }

  /**
   * Handle card_create event
   */
  private handleCardCreate(
    messageId: string,
    cardId: string,
    templateId: string,
    templateName: string,
  ): StreamingCardState {
    const now = Date.now();

    const state: StreamingCardState = {
      cardId,
      templateId,
      templateName,
      status: StreamingCardStatus.CREATING,
      fields: {},
      createdAt: now,
      updatedAt: now,
    };

    this.cards.set(cardId, state);

    // Track message -> card mapping
    if (!this.messageCardMap.has(messageId)) {
      this.messageCardMap.set(messageId, new Set());
    }
    this.messageCardMap.get(messageId)!.add(cardId);

    console.log(`[StreamingCard] Created card: ${cardId}`, {
      templateId,
      templateName,
    });

    this.notifyListeners(cardId, state);
    return state;
  }

  /**
   * Handle card_delta event - accumulate field data
   */
  private handleCardDelta(
    cardId: string,
    field: string,
    content: string,
  ): StreamingCardState | null {
    const state = this.cards.get(cardId);

    if (!state) {
      console.warn(
        `[StreamingCard] Received delta for unknown card: ${cardId}`,
      );
      return null;
    }

    // Update status to streaming
    state.status = StreamingCardStatus.STREAMING;
    state.currentField = field;
    state.updatedAt = Date.now();

    // Accumulate field content
    if (field) {
      if (!state.fields[field]) {
        state.fields[field] = '';
      }
      state.fields[field] += content;
    }

    this.notifyListeners(cardId, state);
    return state;
  }

  /**
   * Handle card_done event - finalize card
   */
  private handleCardDone(
    cardId: string,
    content: string,
  ): StreamingCardState | null {
    const state = this.cards.get(cardId);

    if (!state) {
      console.warn(`[StreamingCard] Received done for unknown card: ${cardId}`);
      return null;
    }

    state.status = StreamingCardStatus.COMPLETE;
    state.currentField = undefined;
    state.updatedAt = Date.now();

    // If content contains final JSON, try to parse and merge
    if (content) {
      try {
        const finalData = JSON.parse(content);
        if (finalData && typeof finalData === 'object') {
          // Merge final data into fields
          Object.entries(finalData).forEach(([key, value]) => {
            if (typeof value === 'string') {
              state.fields[key] = value;
            } else if (value !== null && value !== undefined) {
              state.fields[key] = JSON.stringify(value);
            }
          });
        }
      } catch {
        // Content is not JSON, ignore
      }
    }

    console.log(`[StreamingCard] Card complete: ${cardId}`, {
      fields: Object.keys(state.fields),
    });

    this.notifyListeners(cardId, state);
    return state;
  }

  /**
   * Get card state by ID
   */
  getCard(cardId: string): StreamingCardState | undefined {
    return this.cards.get(cardId);
  }

  /**
   * Get all cards for a message
   */
  getCardsForMessage(messageId: string): StreamingCardState[] {
    const cardIds = this.messageCardMap.get(messageId);
    if (!cardIds) {
      return [];
    }

    return Array.from(cardIds)
      .map(id => this.cards.get(id))
      .filter((state): state is StreamingCardState => state !== undefined);
  }

  /**
   * Check if a message has streaming cards
   */
  hasStreamingCards(messageId: string): boolean {
    return (
      this.messageCardMap.has(messageId) &&
      this.messageCardMap.get(messageId)!.size > 0
    );
  }

  /**
   * Subscribe to card state changes
   */
  subscribe(callback: StreamingCardEventCallback): () => void {
    this.listeners.add(callback);
    return () => {
      this.listeners.delete(callback);
    };
  }

  /**
   * Notify all listeners of card state change
   */
  private notifyListeners(cardId: string, state: StreamingCardState): void {
    this.listeners.forEach(listener => {
      try {
        listener(cardId, state);
      } catch (error) {
        console.error('[StreamingCard] Listener error:', error);
      }
    });
  }

  /**
   * Clear cards for a specific message
   */
  clearCardsForMessage(messageId: string): void {
    const cardIds = this.messageCardMap.get(messageId);
    if (cardIds) {
      cardIds.forEach(cardId => {
        this.cards.delete(cardId);
      });
      this.messageCardMap.delete(messageId);
    }
  }

  /**
   * Clear all card states
   */
  clearAll(): void {
    this.cards.clear();
    this.messageCardMap.clear();
  }

  /**
   * Get card count
   */
  getCardCount(): number {
    return this.cards.size;
  }
}

// Singleton instance for global use
let globalInstance: StreamingCardStateManager | null = null;

export function getStreamingCardStateManager(): StreamingCardStateManager {
  if (!globalInstance) {
    globalInstance = new StreamingCardStateManager();
  }
  return globalInstance;
}

/**
 * Check if extra_info indicates a streaming card event
 */
export function isStreamingCardEvent(extraInfo: MessageExtraInfo): boolean {
  return (
    !!extraInfo.ynet_type &&
    !!extraInfo.card_id &&
    [
      StreamingCardEventType.CREATE,
      StreamingCardEventType.DELTA,
      StreamingCardEventType.DONE,
    ].includes(extraInfo.ynet_type as StreamingCardEventType)
  );
}

/**
 * Build card data object for iframe rendering
 */
export function buildCardDataForIframe(
  state: StreamingCardState,
): Record<string, unknown> {
  return {
    code: state.templateId,
    name: state.templateName,
    data: state.fields,
    status: state.status,
    isStreaming: state.status === StreamingCardStatus.STREAMING,
    isComplete: state.status === StreamingCardStatus.COMPLETE,
  };
}
