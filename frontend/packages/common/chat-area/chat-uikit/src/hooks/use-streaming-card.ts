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

import { useState, useEffect, useCallback } from 'react';

import {
  type StreamingCardState,
  StreamingCardStatus,
  getStreamingCardStateManager,
} from '@coze-common/chat-core';

/**
 * Hook to get streaming card state for a specific card
 * @param cardId The card ID to track
 * @returns The card state or null if not found
 */
export function useStreamingCard(
  cardId: string | undefined,
): StreamingCardState | null {
  const [cardState, setCardState] = useState<StreamingCardState | null>(null);

  useEffect(() => {
    if (!cardId) {
      setCardState(null);
      return;
    }

    const manager = getStreamingCardStateManager();

    // Get initial state
    const initialState = manager.getCard(cardId);
    if (initialState) {
      setCardState(initialState);
    }

    // Subscribe to updates
    const unsubscribe = manager.subscribe((updatedCardId, state) => {
      if (updatedCardId === cardId) {
        setCardState({ ...state });
      }
    });

    return unsubscribe;
  }, [cardId]);

  return cardState;
}

/**
 * Hook to get all streaming cards for a message
 * @param messageId The message ID
 * @returns Array of card states
 */
export function useStreamingCardsForMessage(
  messageId: string | undefined,
): StreamingCardState[] {
  const [cards, setCards] = useState<StreamingCardState[]>([]);

  useEffect(() => {
    if (!messageId) {
      setCards([]);
      return;
    }

    const manager = getStreamingCardStateManager();

    // Get initial cards
    const initialCards = manager.getCardsForMessage(messageId);
    setCards(initialCards);

    // Subscribe to updates
    const unsubscribe = manager.subscribe((cardId, state) => {
      setCards(prevCards => {
        // Find and update existing card, or add new card
        const existingIndex = prevCards.findIndex(c => c.cardId === cardId);

        // Check if this card belongs to our message
        const messageCards = manager.getCardsForMessage(messageId);
        const belongsToMessage = messageCards.some(c => c.cardId === cardId);

        if (!belongsToMessage) {
          // Remove card if it doesn't belong anymore
          if (existingIndex >= 0) {
            return prevCards.filter(c => c.cardId !== cardId);
          }
          return prevCards;
        }

        if (existingIndex >= 0) {
          // Update existing card
          const newCards = [...prevCards];
          newCards[existingIndex] = { ...state };
          return newCards;
        } else {
          // Add new card
          return [...prevCards, { ...state }];
        }
      });
    });

    return unsubscribe;
  }, [messageId]);

  return cards;
}

/**
 * Hook to check if a message has any streaming cards
 * @param messageId The message ID
 * @returns boolean indicating if message has streaming cards
 */
export function useHasStreamingCards(messageId: string | undefined): boolean {
  const cards = useStreamingCardsForMessage(messageId);
  return cards.length > 0;
}

/**
 * Hook to check if any cards are currently streaming
 * @param messageId The message ID (optional, checks all if not provided)
 * @returns boolean indicating if any cards are streaming
 */
export function useIsStreaming(messageId?: string): boolean {
  const [isStreaming, setIsStreaming] = useState(false);

  useEffect(() => {
    const manager = getStreamingCardStateManager();

    const checkStreaming = () => {
      if (messageId) {
        const cards = manager.getCardsForMessage(messageId);
        setIsStreaming(
          cards.some(c => c.status === StreamingCardStatus.STREAMING),
        );
      } else {
        // Check all cards
        setIsStreaming(manager.getCardCount() > 0);
      }
    };

    checkStreaming();

    const unsubscribe = manager.subscribe(() => {
      checkStreaming();
    });

    return unsubscribe;
  }, [messageId]);

  return isStreaming;
}

/**
 * Hook to get streaming card manager actions
 * @returns Object with manager actions
 */
export function useStreamingCardActions() {
  const clearCardsForMessage = useCallback((messageId: string) => {
    const manager = getStreamingCardStateManager();
    manager.clearCardsForMessage(messageId);
  }, []);

  const clearAllCards = useCallback(() => {
    const manager = getStreamingCardStateManager();
    manager.clearAll();
  }, []);

  const getCard = useCallback((cardId: string) => {
    const manager = getStreamingCardStateManager();
    return manager.getCard(cardId);
  }, []);

  return {
    clearCardsForMessage,
    clearAllCards,
    getCard,
  };
}
