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

import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { useSearchParams } from 'react-router-dom';

import classNames from 'classnames';
import {
  BotDebugChatArea,
  BotDebugChatAreaComponentProvider,
} from '@coze-agent-ide/chat-debug-area';

import { type AgentChatAreaProps } from '../single-mode/section-area/agent-chat-area';
import { TraceBridge } from './codex-trace/trace-bridge';
import { CodexTracePanel } from './codex-trace/codex-trace-panel';

import cs from './super-chat-area.module.less';

// 回复区字号:用户可调,持久化到 localStorage。范围 12–22px,默认 14px。
const FONT_KEY = 'super-chat-font-size';
const FONT_MIN = 12;
const FONT_MAX = 22;
const FONT_DEFAULT = 14;
const FONT_STEP = 1;

const clampFont = (n: number): number =>
  Math.min(FONT_MAX, Math.max(FONT_MIN, Math.round(n)));

const useChatFontSize = (): [number, (next: number) => void] => {
  const [size, setSize] = useState<number>(() => {
    if (typeof window === 'undefined') {
      return FONT_DEFAULT;
    }
    const raw = window.localStorage.getItem(FONT_KEY);
    const parsed = raw ? Number(raw) : NaN;
    return Number.isFinite(parsed) ? clampFont(parsed) : FONT_DEFAULT;
  });
  const update = useCallback((next: number) => {
    const v = clampFont(next);
    setSize(v);
    try {
      window.localStorage.setItem(FONT_KEY, String(v));
    } catch {
      /* localStorage 不可用时忽略 */
    }
  }, []);
  return [size, update];
};

const FontSizeControl: React.FC<{
  size: number;
  onChange: (next: number) => void;
}> = ({ size, onChange }) => (
  <div className={cs.fontControl} role="group" aria-label="调整回复字体大小">
    <span className={cs.fontGlyph} aria-hidden="true">
      A
    </span>
    <button
      type="button"
      className={cs.fontBtn}
      disabled={size <= FONT_MIN}
      aria-label="缩小字体"
      title="缩小字体"
      onClick={() => onChange(size - FONT_STEP)}
    >
      −
    </button>
    <span className={cs.fontValue} title="当前回复字号">
      {size}
    </span>
    <button
      type="button"
      className={cs.fontBtn}
      disabled={size >= FONT_MAX}
      aria-label="放大字体"
      title="放大字体"
      onClick={() => onChange(size + FONT_STEP)}
    >
      +
    </button>
  </div>
);

export type SuperChatAreaProps = Pick<
  AgentChatAreaProps,
  | 'renderChatTitleNode'
  | 'chatSlot'
  | 'chatHeaderClassName'
  | 'chatAreaReadOnly'
> & {
  chatInputTopSlot?: ReactNode;
  messageView?: 'trace' | 'native';
  title?: ReactNode;
};

export const SuperChatArea: React.FC<SuperChatAreaProps> = ({
  renderChatTitleNode,
  chatSlot,
  chatHeaderClassName,
  chatAreaReadOnly,
  chatInputTopSlot,
  messageView = 'trace',
  title = '预览与调试',
}) => {
  const extraTitleNode = renderChatTitleNode?.({
    pageFrom: undefined,
    showBackground: false,
  });
  // 员工聊天模式:标题用虚拟员工名（URL ?name=），而不是编辑器的「预览与调试」。
  const [searchParams] = useSearchParams();
  const effectiveTitle =
    searchParams.get('employeeChat') === '1'
      ? searchParams.get('name') || '员工对话'
      : title;
  const [fontSize, setFontSize] = useChatFontSize();
  const traceInject = useMemo(
    () => ({
      chatInputIntegration: {
        renderChatInputTopSlot: () => (
          <>
            <TraceBridge />
            {chatInputTopSlot}
          </>
        ),
      },
      messageGroupWrapper: () => null,
      onboarding: () => null,
    }),
    [chatInputTopSlot],
  );
  const nativeInject = useMemo(
    () =>
      chatInputTopSlot
        ? {
            chatInputIntegration: {
              renderChatInputTopSlot: () => chatInputTopSlot,
            },
          }
        : {},
    [chatInputTopSlot],
  );

  return (
    <div className={cs.chatShell}>
      <div className={classNames(cs.chatHeader, chatHeaderClassName)}>
        <div className={cs.chatTitle}>{effectiveTitle}</div>
        <div className={cs.headerActions}>
          <FontSizeControl size={fontSize} onChange={setFontSize} />
          {extraTitleNode}
        </div>
      </div>
      {messageView === 'native' ? (
        <div className={classNames(cs.superChat, cs.nativeChat)}>
          <BotDebugChatAreaComponentProvider value={nativeInject}>
            <div className={cs.nativeChatPane}>
              <BotDebugChatArea readOnly={chatAreaReadOnly} />
              {chatSlot ? (
                <div className={cs.nativeChatSlot}>{chatSlot}</div>
              ) : null}
            </div>
          </BotDebugChatAreaComponentProvider>
        </div>
      ) : (
        <div className={cs.superChat}>
          <div
            className={cs.tracePane}
            style={
              { '--codex-answer-font': `${fontSize}px` } as React.CSSProperties
            }
          >
            <CodexTracePanel />
          </div>
          <BotDebugChatAreaComponentProvider value={traceInject}>
            <div className={cs.composerPane}>
              <BotDebugChatArea readOnly={chatAreaReadOnly} />
              {chatSlot}
            </div>
          </BotDebugChatAreaComponentProvider>
        </div>
      )}
    </div>
  );
};
