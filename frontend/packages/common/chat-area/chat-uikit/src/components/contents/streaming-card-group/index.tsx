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

import { useState, useEffect, type FC } from 'react';

import {
  type StreamingGroupState,
  StreamingGroupStatus,
  getStreamingCardStateManager,
} from '@coze-common/chat-core';

import { StreamingCardContent } from '../streaming-card-content';

import './index.less';

export interface StreamingCardGroupProps {
  groupId: string;
  messageId: string;
  onGroupComplete?: (groupId: string, cardIds: string[]) => void;
}

/**
 * Streaming Card Group Component
 *
 * Renders a group of cards with specified layout (horizontal, vertical, waterfall).
 * Each card has its own iframe.
 * Handles real-time updates as cards are added to the group during streaming.
 */
export const StreamingCardGroup: FC<StreamingCardGroupProps> = props => {
  const { groupId, messageId, onGroupComplete } = props;

  const [groupState, setGroupState] = useState<StreamingGroupState | null>(null);

  // Get initial group state
  useEffect(() => {
    const manager = getStreamingCardStateManager();
    const initialState = manager.getGroup(groupId);
    if (initialState) {
      setGroupState(initialState);
    }
  }, [groupId]);

  // Subscribe to group state updates
  useEffect(() => {
    const manager = getStreamingCardStateManager();

    const unsubscribe = manager.subscribeToGroups((updatedGroupId, state) => {
      if (updatedGroupId === groupId) {
        setGroupState({ ...state });

        // Notify completion
        if (
          state.status === StreamingGroupStatus.COMPLETE &&
          onGroupComplete
        ) {
          onGroupComplete(groupId, state.cardIds);
        }
      }
    });

    return unsubscribe;
  }, [groupId, onGroupComplete]);

  // Also subscribe to card events to re-render when cards are added
  useEffect(() => {
    const manager = getStreamingCardStateManager();

    const unsubscribe = manager.subscribe((cardId, cardState) => {
      // If this card belongs to our group, refresh group state
      if (cardState.groupId === groupId) {
        const currentGroupState = manager.getGroup(groupId);
        if (currentGroupState) {
          setGroupState({ ...currentGroupState });
        }
      }
    });

    return unsubscribe;
  }, [groupId]);

  // Generate layout class based on group settings
  const getLayoutClass = (): string => {
    if (!groupState) {
      return '';
    }

    const layoutClass = `layout-${groupState.layout}`;
    const columnsClass = `columns-${groupState.columns}`;

    return `${layoutClass} ${columnsClass}`;
  };

  // Render nothing if group state not available
  if (!groupState) {
    return null;
  }

  const isComplete = groupState.status === StreamingGroupStatus.COMPLETE;
  const cardCount = groupState.cardIds.length;

  return (
    <div
      className={`streaming-card-group ${getLayoutClass()} ${isComplete ? 'complete' : 'streaming'}`}
      data-group-id={groupId}
      data-layout={groupState.layout}
      data-columns={groupState.columns}
    >
      {/* Group header - optional, can be shown in debug mode */}
      {process.env.NODE_ENV === 'development' && (
        <div className="streaming-card-group-header">
          <span className="group-info">
            Group: {groupState.layout} ({cardCount} cards)
          </span>
        </div>
      )}

      {/* Cards container with layout - each card has its own iframe */}
      <div className="streaming-card-group-cards">
        {groupState.cardIds.map((cardId, index) => (
          <div
            key={cardId}
            className="streaming-card-group-item"
            data-index={index}
          >
            <StreamingCardContent
              cardId={cardId}
              messageId={messageId}
            />
          </div>
        ))}
      </div>
    </div>
  );
};

StreamingCardGroup.displayName = 'StreamingCardGroup';

export default StreamingCardGroup;
