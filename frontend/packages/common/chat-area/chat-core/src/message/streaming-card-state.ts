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
  GROUP_START = 'card_group_start',
  GROUP_END = 'card_group_end',
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

// Group status during streaming
export enum StreamingGroupStatus {
  ACTIVE = 'active', // Group is active, receiving cards
  COMPLETE = 'complete', // Group is complete
}

// Layout types for card groups
export enum CardGroupLayout {
  HORIZONTAL = 'horizontal',
  VERTICAL = 'vertical',
  WATERFALL = 'waterfall',
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
  groupId?: string; // ID of the group this card belongs to (if any)
}

// Card group state
export interface StreamingGroupState {
  groupId: string;
  layout: CardGroupLayout;
  columns: number;
  status: StreamingGroupStatus;
  cardIds: string[]; // IDs of cards in this group
  createdAt: number;
  updatedAt: number;
}

// Event listener callback for card events
export type StreamingCardEventCallback = (
  cardId: string,
  state: StreamingCardState,
) => void;

// Event listener callback for group events
export type StreamingGroupEventCallback = (
  groupId: string,
  state: StreamingGroupState,
) => void;

// ===== CardStream Payload Types (for sendMessage('cardstream', payload)) =====

/**
 * Single card item in the cardstream payload
 * Used for rendering cards within groups
 */
export interface CardStreamItem {
  id: string;                    // Card ID
  code: string;                  // Card template code (templateId)
  data: Record<string, unknown>; // Card data (accumulated fields)
}

/**
 * Card group in the cardstream payload
 * Contains layout configuration and child cards
 */
export interface CardStreamGroup {
  id: string;                    // Group ID
  code: string;                  // Layout type: 'Vertical' | 'Horizontal' | 'Waterfall'
  data: {
    columns?: number;            // Number of columns (for grid layouts)
    [key: string]: unknown;      // Additional layout configuration
  };
  children: CardStreamItem[];    // Child cards in this group
}

/**
 * CardStream payload structure
 * Sent via sendMessage('cardstream', payload)
 */
export interface CardStreamPayload {
  groups: CardStreamGroup[];
}

// Event listener callback for cardstream payload updates (full state)
export type CardStreamPayloadCallback = (
  messageId: string,
  payload: CardStreamPayload,
) => void;

// ===== Incremental CardStream Event Types =====

/**
 * Incremental event types for streaming updates
 */
export type CardStreamEventType =
  | 'group_start'
  | 'group_end'
  | 'card_create'
  | 'card_delta'
  | 'card_done';

/**
 * Base interface for incremental events
 */
interface CardStreamEventBase {
  type: CardStreamEventType;
}

/**
 * Group start event
 */
export interface CardStreamGroupStartEvent extends CardStreamEventBase {
  type: 'group_start';
  group: {
    id: string;
    code: string;
    data: { columns?: number; [key: string]: unknown };
  };
}

/**
 * Group end event
 */
export interface CardStreamGroupEndEvent extends CardStreamEventBase {
  type: 'group_end';
  groupId: string;
}

/**
 * Card create event
 */
export interface CardStreamCardCreateEvent extends CardStreamEventBase {
  type: 'card_create';
  groupId: string | null;
  card: CardStreamItem;
}

/**
 * Card delta event - incremental field update
 */
export interface CardStreamCardDeltaEvent extends CardStreamEventBase {
  type: 'card_delta';
  cardId: string;
  field: string;
  value: unknown;
  op: 'set' | 'add';
}

/**
 * Card done event
 */
export interface CardStreamCardDoneEvent extends CardStreamEventBase {
  type: 'card_done';
  cardId: string;
  data: Record<string, unknown>;
}

/**
 * Union type for all incremental events
 */
export type CardStreamIncrementalEvent =
  | CardStreamGroupStartEvent
  | CardStreamGroupEndEvent
  | CardStreamCardCreateEvent
  | CardStreamCardDeltaEvent
  | CardStreamCardDoneEvent;

/**
 * Callback for incremental cardstream events
 */
export type CardStreamIncrementalCallback = (
  messageId: string,
  event: CardStreamIncrementalEvent,
) => void;

/**
 * StreamingCardStateManager
 *
 * Tracks and manages streaming card states during SSE message flow.
 * Provides event subscription for UI components to react to card updates.
 */
export class StreamingCardStateManager {
  private cards: Map<string, StreamingCardState> = new Map();
  private groups: Map<string, StreamingGroupState> = new Map();
  private messageCardMap: Map<string, Set<string>> = new Map(); // message_id -> card_ids
  private messageGroupMap: Map<string, Set<string>> = new Map(); // message_id -> group_ids
  private currentGroupId: string | null = null; // Track current active group
  private listeners: Set<StreamingCardEventCallback> = new Set();
  private groupListeners: Set<StreamingGroupEventCallback> = new Set();
  private cardstreamListeners: Set<CardStreamPayloadCallback> = new Set();
  private incrementalListeners: Set<CardStreamIncrementalCallback> = new Set();

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
  ): StreamingCardState | StreamingGroupState | null {
    const {
      ynet_type,
      card_id,
      template_id,
      template_name,
      card_field,
      card_value,
      card_op,
      group_id,
      card_layout,
      card_columns,
    } = extraInfo;

    // Skip if not a card or group event
    if (!ynet_type) {
      return null;
    }

    // Handle group events
    if (ynet_type === StreamingCardEventType.GROUP_START && group_id) {
      return this.handleGroupStart(
        messageId,
        group_id,
        (card_layout as CardGroupLayout) || CardGroupLayout.VERTICAL,
        parseInt(card_columns || '1', 10),
      );
    }

    if (ynet_type === StreamingCardEventType.GROUP_END && group_id) {
      return this.handleGroupEnd(messageId, group_id);
    }

    // Handle card events - require card_id
    if (!card_id) {
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
          group_id, // Pass group_id from card_create event metadata
        );
        break;

      case StreamingCardEventType.DELTA:
        // card_value is now in meta_data instead of content/answer
        // card_op determines if we should append to array (add) or set value (set)
        state = this.handleCardDelta(card_id, card_field || '', card_value || '', card_op || 'set');
        break;

      case StreamingCardEventType.DONE:
        state = this.handleCardDone(card_id, content);
        break;

      default:
        console.warn(
          `[StreamingCardStateManager] Unknown ynet_type: ${ynet_type}`,
        );
    }

    // Notify cardstream listeners after any card/group event
    if (state) {
      this.notifyCardStreamListeners(messageId);
    }

    return state;
  }

  /**
   * Handle card_group_start event
   */
  private handleGroupStart(
    messageId: string,
    groupId: string,
    layout: CardGroupLayout,
    columns: number,
  ): StreamingGroupState {
    const now = Date.now();

    const state: StreamingGroupState = {
      groupId,
      layout,
      columns: columns > 0 ? columns : 1,
      status: StreamingGroupStatus.ACTIVE,
      cardIds: [],
      createdAt: now,
      updatedAt: now,
    };

    this.groups.set(groupId, state);
    this.currentGroupId = groupId;

    // Track message -> group mapping
    if (!this.messageGroupMap.has(messageId)) {
      this.messageGroupMap.set(messageId, new Set());
    }
    this.messageGroupMap.get(messageId)!.add(groupId);

    console.log(`[StreamingCard] Group started: ${groupId}`, {
      layout,
      columns,
    });

    this.notifyGroupListeners(groupId, state);

    // Send incremental event
    this.notifyIncrementalListeners(messageId, {
      type: 'group_start',
      group: {
        id: groupId,
        code: this.layoutToCode(layout),
        data: { columns: state.columns },
      },
    });

    return state;
  }

  /**
   * Handle card_group_end event
   */
  private handleGroupEnd(messageId: string, groupId: string): StreamingGroupState | null {
    const state = this.groups.get(groupId);

    if (!state) {
      console.warn(
        `[StreamingCard] Received group_end for unknown group: ${groupId}`,
      );
      return null;
    }

    state.status = StreamingGroupStatus.COMPLETE;
    state.updatedAt = Date.now();

    // Clear current group if it matches
    if (this.currentGroupId === groupId) {
      this.currentGroupId = null;
    }

    console.log(`[StreamingCard] Group complete: ${groupId}`, {
      cardCount: state.cardIds.length,
    });

    this.notifyGroupListeners(groupId, state);

    // Send incremental event
    this.notifyIncrementalListeners(messageId, {
      type: 'group_end',
      groupId,
    });

    return state;
  }

  /**
   * Handle card_create event
   * @param groupId Group ID from card_create event metadata (preferred) or falls back to currentGroupId
   */
  private handleCardCreate(
    messageId: string,
    cardId: string,
    templateId: string,
    templateName: string,
    groupId?: string,
  ): StreamingCardState {
    const now = Date.now();
    // Prefer groupId from event metadata, fall back to currentGroupId
    const effectiveGroupId = groupId || this.currentGroupId || undefined;

    const state: StreamingCardState = {
      cardId,
      templateId,
      templateName,
      status: StreamingCardStatus.CREATING,
      fields: {},
      createdAt: now,
      updatedAt: now,
      groupId: effectiveGroupId,
    };

    this.cards.set(cardId, state);

    // If there's a group (from metadata or current), add this card to it
    if (effectiveGroupId) {
      const groupState = this.groups.get(effectiveGroupId);
      if (groupState) {
        // Only add if not already present
        if (!groupState.cardIds.includes(cardId)) {
          groupState.cardIds.push(cardId);
          groupState.updatedAt = now;
        }
      } else {
        // Group state doesn't exist yet - create it implicitly
        // This handles the case where card_create arrives before card_group_start
        console.log(
          `[StreamingCard] Creating implicit group for card: ${cardId}, groupId: ${effectiveGroupId}`,
        );
        const implicitGroupState: StreamingGroupState = {
          groupId: effectiveGroupId,
          layout: CardGroupLayout.HORIZONTAL, // Default to horizontal layout
          columns: 2, // Default to 2 columns
          status: StreamingGroupStatus.ACTIVE,
          cardIds: [cardId],
          createdAt: now,
          updatedAt: now,
        };
        this.groups.set(effectiveGroupId, implicitGroupState);

        // Track message -> group mapping
        if (!this.messageGroupMap.has(messageId)) {
          this.messageGroupMap.set(messageId, new Set());
        }
        this.messageGroupMap.get(messageId)!.add(effectiveGroupId);

        this.notifyGroupListeners(effectiveGroupId, implicitGroupState);
      }
    }

    // Track message -> card mapping
    if (!this.messageCardMap.has(messageId)) {
      this.messageCardMap.set(messageId, new Set());
    }
    this.messageCardMap.get(messageId)!.add(cardId);

    console.log(`[StreamingCard] Created card: ${cardId}`, {
      templateId,
      templateName,
      groupId: state.groupId,
    });

    this.notifyListeners(cardId, state);

    // Send incremental event
    this.notifyIncrementalListeners(messageId, {
      type: 'card_create',
      groupId: effectiveGroupId || null,
      card: {
        id: cardId,
        code: templateId,
        data: {},
      },
    });

    return state;
  }

  /**
   * Handle card_delta event - process field data
   * @param cardOp Operation type: "add" for array elements, "set" for simple values
   */
  private handleCardDelta(
    cardId: string,
    field: string,
    content: string,
    cardOp: 'add' | 'set' = 'set',
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

    // Handle field content based on operation type
    if (field) {
      if (cardOp === 'add') {
        // For "add" operation, append to array
        // Parse existing value as array, or create new array
        let existingArray: unknown[] = [];
        if (state.fields[field]) {
          try {
            const parsed = JSON.parse(state.fields[field]);
            if (Array.isArray(parsed)) {
              existingArray = parsed;
            }
          } catch {
            // Not valid JSON array, start fresh
            existingArray = [];
          }
        }

        // Parse and append new value
        try {
          const newValue = JSON.parse(content);
          existingArray.push(newValue);
          state.fields[field] = JSON.stringify(existingArray);
        } catch {
          // If content is not valid JSON, append as string to array
          existingArray.push(content);
          state.fields[field] = JSON.stringify(existingArray);
        }
      } else {
        // For "set" operation, replace field value (backend sends complete value)
        // Check if content contains multiple JSON objects separated by newlines
        // This happens for array fields where backend buffers multiple items
        const processedContent = this.processFieldContent(content);
        state.fields[field] = processedContent;
      }
    }

    this.notifyListeners(cardId, state);
    return state;
  }

  /**
   * Process field content to handle special formats
   * - Newline-separated JSON objects are converted to JSON array
   * - Single JSON objects are returned as-is
   * - Plain text is returned as-is
   */
  private processFieldContent(content: string): string {
    if (!content) {
      return content;
    }

    const trimmedContent = content.trim();

    // Check if content contains multiple JSON objects separated by newlines
    // Pattern: {obj1}\n{obj2} or {obj1}\n{obj2}\n{obj3}...
    if (trimmedContent.includes('}\n{')) {
      try {
        // Split by newlines and parse each JSON object
        const lines = trimmedContent.split('\n').filter(line => line.trim());
        const jsonObjects: unknown[] = [];

        for (const line of lines) {
          const trimmedLine = line.trim();
          if (trimmedLine.startsWith('{') && trimmedLine.endsWith('}')) {
            try {
              const parsed = JSON.parse(trimmedLine);
              jsonObjects.push(parsed);
            } catch {
              // Not valid JSON, skip
              console.warn('[StreamingCard] Failed to parse JSON line:', trimmedLine);
            }
          }
        }

        // If we successfully parsed multiple objects, return as JSON array
        if (jsonObjects.length > 0) {
          return JSON.stringify(jsonObjects);
        }
      } catch (e) {
        console.warn('[StreamingCard] Failed to process multi-line JSON:', e);
      }
    }

    // Return content as-is for single values
    return trimmedContent;
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
   * Subscribe to group state changes
   */
  subscribeToGroups(callback: StreamingGroupEventCallback): () => void {
    this.groupListeners.add(callback);
    return () => {
      this.groupListeners.delete(callback);
    };
  }

  /**
   * Notify all group listeners of group state change
   */
  private notifyGroupListeners(
    groupId: string,
    state: StreamingGroupState,
  ): void {
    this.groupListeners.forEach(listener => {
      try {
        listener(groupId, state);
      } catch (error) {
        console.error('[StreamingCard] Group listener error:', error);
      }
    });
  }

  /**
   * Get group state by ID
   */
  getGroup(groupId: string): StreamingGroupState | undefined {
    return this.groups.get(groupId);
  }

  /**
   * Get all groups for a message
   */
  getGroupsForMessage(messageId: string): StreamingGroupState[] {
    const groupIds = this.messageGroupMap.get(messageId);
    if (!groupIds) {
      return [];
    }

    return Array.from(groupIds)
      .map(id => this.groups.get(id))
      .filter((state): state is StreamingGroupState => state !== undefined);
  }

  /**
   * Check if a message has streaming groups
   */
  hasStreamingGroups(messageId: string): boolean {
    return (
      this.messageGroupMap.has(messageId) &&
      this.messageGroupMap.get(messageId)!.size > 0
    );
  }

  /**
   * Get cards belonging to a specific group
   */
  getCardsForGroup(groupId: string): StreamingCardState[] {
    const group = this.groups.get(groupId);
    if (!group) {
      return [];
    }

    return group.cardIds
      .map(cardId => this.cards.get(cardId))
      .filter((state): state is StreamingCardState => state !== undefined);
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
   * Clear all card and group states
   */
  clearAll(): void {
    this.cards.clear();
    this.groups.clear();
    this.messageCardMap.clear();
    this.messageGroupMap.clear();
    this.currentGroupId = null;
  }

  /**
   * Get card count
   */
  getCardCount(): number {
    return this.cards.size;
  }

  // ===== CardStream Payload Methods =====

  /**
   * Build the groups payload for sendMessage('cardstream', payload)
   * Converts internal state to the format expected by the container
   *
   * Output structure:
   * {
   *   groups: [
   *     {
   *       id: 'group-1',
   *       code: 'Vertical',        // Layout type
   *       data: { columns: 2 },    // Layout config
   *       children: [
   *         { id: 'card-1', code: 'templateId', data: {...} },
   *         { id: 'card-2', code: 'templateId', data: {...} }
   *       ]
   *     }
   *   ]
   * }
   */
  buildGroupsPayload(messageId: string): CardStreamPayload {
    const groups: CardStreamGroup[] = [];
    const processedCardIds = new Set<string>();

    // Get all groups for this message
    const messageGroups = this.getGroupsForMessage(messageId);

    // Process grouped cards
    for (const group of messageGroups) {
      const children: CardStreamItem[] = [];

      for (const cardId of group.cardIds) {
        const card = this.cards.get(cardId);
        if (card) {
          children.push(this.buildCardStreamItem(card));
          processedCardIds.add(cardId);
        }
      }

      groups.push({
        id: group.groupId,
        code: this.layoutToCode(group.layout),
        data: {
          columns: group.columns,
        },
        children,
      });
    }

    // Process ungrouped cards (cards without a group)
    const messageCards = this.getCardsForMessage(messageId);
    const ungroupedCards = messageCards.filter(
      card => !processedCardIds.has(card.cardId),
    );

    if (ungroupedCards.length > 0) {
      // Create a default group for ungrouped cards
      const defaultGroupId = `default-${messageId}`;
      const children: CardStreamItem[] = ungroupedCards.map(card =>
        this.buildCardStreamItem(card),
      );

      groups.push({
        id: defaultGroupId,
        code: 'Vertical', // Default layout for ungrouped cards
        data: {
          columns: 1,
        },
        children,
      });
    }

    return { groups };
  }

  /**
   * Build a CardStreamItem from a StreamingCardState
   */
  private buildCardStreamItem(card: StreamingCardState): CardStreamItem {
    // Parse JSON string fields back to objects
    const parsedData: Record<string, unknown> = {};

    for (const [key, value] of Object.entries(card.fields)) {
      try {
        // Try to parse as JSON
        parsedData[key] = JSON.parse(value);
      } catch {
        // If not valid JSON, use the raw string value
        parsedData[key] = value;
      }
    }

    return {
      id: card.cardId,
      code: card.templateId,
      data: parsedData,
    };
  }

  /**
   * Convert CardGroupLayout enum to code string
   */
  private layoutToCode(layout: CardGroupLayout): string {
    switch (layout) {
      case CardGroupLayout.HORIZONTAL:
        return 'Horizontal';
      case CardGroupLayout.VERTICAL:
        return 'Vertical';
      case CardGroupLayout.WATERFALL:
        return 'Waterfall';
      default:
        return 'Vertical';
    }
  }

  /**
   * Subscribe to cardstream payload updates (full state)
   * Called after any card/group event with the full groups payload
   */
  subscribeToCardStream(callback: CardStreamPayloadCallback): () => void {
    this.cardstreamListeners.add(callback);
    return () => {
      this.cardstreamListeners.delete(callback);
    };
  }

  /**
   * Subscribe to incremental cardstream events
   * Called with each individual event (group_start, card_create, card_delta, etc.)
   */
  subscribeToIncremental(callback: CardStreamIncrementalCallback): () => void {
    this.incrementalListeners.add(callback);
    return () => {
      this.incrementalListeners.delete(callback);
    };
  }

  /**
   * Notify all cardstream listeners with the updated payload (full state)
   */
  private notifyCardStreamListeners(messageId: string): void {
    if (this.cardstreamListeners.size === 0) {
      return;
    }

    const payload = this.buildGroupsPayload(messageId);

    console.log('[StreamingCard] CardStream payload update:', {
      messageId,
      groupCount: payload.groups.length,
      groups: payload.groups.map(g => ({
        id: g.id,
        code: g.code,
        childCount: g.children.length,
      })),
    });

    this.cardstreamListeners.forEach(listener => {
      try {
        listener(messageId, payload);
      } catch (error) {
        console.error('[StreamingCard] CardStream listener error:', error);
      }
    });
  }

  /**
   * Notify all incremental listeners with a single event
   */
  private notifyIncrementalListeners(
    messageId: string,
    event: CardStreamIncrementalEvent,
  ): void {
    if (this.incrementalListeners.size === 0) {
      return;
    }

    console.log('[StreamingCard] 📤 Incremental event:', {
      messageId,
      type: event.type,
      ...(event.type === 'card_delta' ? { cardId: event.cardId, field: event.field, op: event.op } : {}),
      ...(event.type === 'card_create' ? { cardId: event.card.id, groupId: event.groupId } : {}),
      ...(event.type === 'group_start' ? { groupId: event.group.id } : {}),
    });

    this.incrementalListeners.forEach(listener => {
      try {
        listener(messageId, event);
      } catch (error) {
        console.error('[StreamingCard] Incremental listener error:', error);
      }
    });
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
 * Check if extra_info indicates a streaming card or group event
 */
export function isStreamingCardEvent(extraInfo: MessageExtraInfo): boolean {
  if (!extraInfo.ynet_type) {
    return false;
  }

  // Check for group events (require group_id)
  if (
    [StreamingCardEventType.GROUP_START, StreamingCardEventType.GROUP_END].includes(
      extraInfo.ynet_type as StreamingCardEventType,
    )
  ) {
    return !!extraInfo.group_id;
  }

  // Check for card events (require card_id)
  return (
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
