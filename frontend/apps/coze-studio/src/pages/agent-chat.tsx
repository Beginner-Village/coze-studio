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

import { useSearchParams } from 'react-router-dom';
import { useEffect, useMemo, useRef, useState } from 'react';

import { BuilderChat } from '@coze-studio/open-chat';

const MESSAGE_SOURCE_IFRAME = 'ynet-sdk-iframe';

/**
 * AgentChatPage - Embeddable chat page for WebSDK iframe integration.
 *
 * URL params:
 *   - bot_id: The bot/project ID
 *   - token: Authentication token (PAT)
 *   - workflow_id: Optional workflow ID
 *   - mode: Should be 'websdk'
 *   - title: Optional display title
 */
export default function AgentChatPage() {
  const [searchParams] = useSearchParams();
  const [token, setToken] = useState(searchParams.get('token') || '');

  const botId = searchParams.get('bot_id') || '';
  const workflowId = searchParams.get('workflow_id') || '';
  const title = searchParams.get('title') || '';
  const expectedParentOrigin = searchParams.get('parent_origin') || '';
  const parentOrigin = useRef<string>(expectedParentOrigin);

  // Listen for postMessage from parent window
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (expectedParentOrigin && event.origin !== expectedParentOrigin) {
        console.warn(
          '[agent-chat] reject message from untrusted origin:',
          event.origin,
          'expected:',
          expectedParentOrigin,
        );
        return;
      }
      const { data } = event;
      if (!data || !data.type || data.source === MESSAGE_SOURCE_IFRAME) {
        return;
      }

      parentOrigin.current = event.origin;

      switch (data.type) {
        case 'INIT':
          if (data.payload?.token) {
            setToken(data.payload.token);
          }
          break;
        case 'UPDATE_TOKEN':
          if (data.payload?.token) {
            setToken(data.payload.token);
          }
          break;
        default:
          break;
      }
    };

    window.addEventListener('message', handleMessage);

    // Notify parent that iframe is ready (only when parent_origin is known)
    if (window.parent !== window && expectedParentOrigin) {
      window.parent.postMessage(
        { source: MESSAGE_SOURCE_IFRAME, type: 'READY', payload: {} },
        expectedParentOrigin,
      );
    }

    return () => window.removeEventListener('message', handleMessage);
  }, [expectedParentOrigin]);

  const workflow = useMemo(
    () => ({
      id: workflowId || undefined,
    }),
    [workflowId],
  );

  const project = useMemo(
    () => ({
      id: botId,
      type: 'bot' as const,
      mode: 'websdk' as const,
      name: title || undefined,
    }),
    [botId, title],
  );

  const auth = useMemo(
    () => ({
      type: 'external' as const,
      token,
      refreshToken: () => {
        // Request new token from parent (only when parent_origin is known)
        if (window.parent !== window && expectedParentOrigin) {
          window.parent.postMessage(
            {
              source: MESSAGE_SOURCE_IFRAME,
              type: 'TOKEN_EXPIRED',
              payload: {},
            },
            expectedParentOrigin,
          );
        }
        return Promise.resolve('');
      },
    }),
    [token, expectedParentOrigin],
  );

  if (!botId) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          color: '#86909c',
          fontSize: '14px',
        }}
      >
        Missing bot_id parameter
      </div>
    );
  }

  return (
    <div style={{ width: '100%', height: '100vh', overflow: 'hidden' }}>
      <BuilderChat
        workflow={workflow}
        project={project}
        auth={auth}
        userInfo={undefined}
        areaUi={{
          header: { isShow: !!title },
          input: { isShow: true },
          uploadable: true,
        }}
        setting={{}}
      />
    </div>
  );
}
