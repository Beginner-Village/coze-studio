/*
 * Copyright 2025 ynet-dev Authors
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

import { useEffect, useMemo, useState } from 'react';

import { useChatAreaStoreSet } from '@coze-common/chat-area';
import { DeveloperApi } from '@coze-arch/bot-api';

import { useTraceStore, type RawMessage } from './trace-store';

const SUPER_AGENT_SESSION_SELECT_EVENT = 'coze:super-agent-session-select';
const CONTEXT_COMPACTED_EVENT = 'context.compacted';
const WORKFLOW_CANVAS_DIRECTIVE_MARKER = '你正在工作流编辑器右侧聊天中';
const WORKFLOW_CANVAS_USER_QUERY_MARKER = '用户需求:';

interface SuperAgentTraceEvent {
  id?: string;
  event?: string;
  kind?: string;
  content?: string;
  created_at?: string | number;
  updated_at?: string | number;
  metadata?: Record<string, unknown>;
}

const getMessageID = (message: RawMessage) =>
  String(message?.message_id ?? message?.id ?? '');

const sanitizeWorkflowCanvasDirective = (message: RawMessage): RawMessage => {
  const content = message?.content;
  if (
    String(message?.role) !== 'user' ||
    typeof content !== 'string' ||
    !content.includes(WORKFLOW_CANVAS_DIRECTIVE_MARKER)
  ) {
    return message;
  }

  const queryIndex = content.lastIndexOf(WORKFLOW_CANVAS_USER_QUERY_MARKER);
  if (queryIndex < 0) {
    return message;
  }

  const displayContent = content
    .slice(queryIndex + WORKFLOW_CANVAS_USER_QUERY_MARKER.length)
    .trim();
  return displayContent ? { ...message, content: displayContent } : message;
};

const toContextTraceMessages = (
  events: SuperAgentTraceEvent[] | undefined,
): RawMessage[] =>
  (events ?? [])
    .filter(event => event.event === CONTEXT_COMPACTED_EVENT)
    .map(event => ({
      message_id:
        event.id ||
        `context:compacted:${String(event.metadata?.summary_path ?? '')}`,
      role: 'assistant',
      type: 'context_compacted',
      content: event.content || '',
      created_at: event.created_at,
      updated_at: event.updated_at,
      extra_info: {
        event: event.event,
        kind: event.kind,
        ...(event.metadata ?? {}),
      },
    }));

const mergeTraceMessages = (
  liveMessages: RawMessage[],
  contextMessages: RawMessage[],
) => {
  const seenMessageIDs = new Set(liveMessages.map(getMessageID).filter(Boolean));
  return [
    ...liveMessages,
    ...contextMessages.filter(message => {
      const messageID = getMessageID(message);
      if (!messageID || seenMessageIDs.has(messageID)) {
        return false;
      }
      seenMessageIDs.add(messageID);
      return true;
    }),
  ];
};

/**
 * TraceBridge:挂在 ChatArea context 内部(通过 BotDebugChatAreaComponentProvider 注入),
 * 订阅实时消息并镜像到模块级 trace store。自身渲染 null。
 */
export const TraceBridge: React.FC = () => {
  const { useMessagesStore, useWaitingStore } = useChatAreaStoreSet();
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const messages = useMessagesStore((s: any) => s.messages);
  const pendingReply = useWaitingStore(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (s: any) => Boolean(s.sending || s.waiting || s.responding),
  );
  const setMessages = useTraceStore(s => s.setMessages);
  const setPendingReply = useTraceStore(s => s.setPendingReply);
  const [conversationID, setConversationID] = useState('');
  const [contextMessages, setContextMessages] = useState<RawMessage[]>([]);
  const liveMessages = useMemo(
    () =>
      ((messages ?? []) as RawMessage[]).map(sanitizeWorkflowCanvasDirective),
    [messages],
  );

  useEffect(() => {
    const handleSessionSelect = (event: Event) => {
      const detail = (
        event as CustomEvent<{
          conversationId?: string;
        }>
      ).detail;
      setConversationID(detail?.conversationId || '');
    };
    window.addEventListener(
      SUPER_AGENT_SESSION_SELECT_EVENT,
      handleSessionSelect,
    );
    return () => {
      window.removeEventListener(
        SUPER_AGENT_SESSION_SELECT_EVENT,
        handleSessionSelect,
      );
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    if (!conversationID) {
      setContextMessages([]);
      return () => {
        cancelled = true;
      };
    }
    DeveloperApi.SuperAgentGetTrace({
      conversation_id: conversationID,
      limit: 100,
    })
      .then(resp => {
        if (cancelled) {
          return;
        }
        setContextMessages(
          toContextTraceMessages(
            resp?.data?.events as SuperAgentTraceEvent[] | undefined,
          ),
        );
      })
      .catch(() => {
        if (!cancelled) {
          setContextMessages([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [conversationID]);

  useEffect(() => {
    setMessages(mergeTraceMessages(liveMessages, contextMessages));
  }, [contextMessages, liveMessages, setMessages]);

  useEffect(() => {
    setPendingReply(pendingReply);
  }, [pendingReply, setPendingReply]);

  return null;
};
