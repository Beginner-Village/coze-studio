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

import { memo, useEffect, useRef, useState, type FC } from 'react';

import { type Message, type ContentType } from '@coze-common/chat-core';
import { MdBoxLazy } from '@coze-arch/bot-md-box-adapter/lazy';

type IProps = Record<'message', Message<ContentType>>;

// export const BizMessageInnerAddonBottom: FC<IProps> = p =>
//   p.message.role === 'assistant' && p.message.reasoning_content ? (
//     <div className="my-[8px] px-[14px] border-solid border-[0] border-l-[0.25em] border-l-[var(--color-border-default)] text-[var(--color-fg-muted)]">
//       {p.message.reasoning_content}
//     </div>
//   ) : null;

export const BizMessageInnerAddonBottom: FC<IProps> = memo(
  p => {
    const [reasoningFinished, setReasoningFinished] = useState(false);
    // 思维链默认折叠,用户可手动展开
    const [expanded, setExpanded] = useState(false);
    const ref = useRef(p.message.reasoning_content);

    useEffect(() => {
      setReasoningFinished(ref.current === p.message.reasoning_content);

      return () => {
        ref.current = p.message.reasoning_content;
      };
      // Content used to trigger reasoning rerender
    }, [p.message.reasoning_content, p.message.content]);

    if (!(p.message.role === 'assistant' && p.message.reasoning_content)) {
      return null;
    }

    const reasoning = p.message.reasoning_content;
    const streaming = !p.message.is_finish && !reasoningFinished;

    return (
      <div className="my-[8px]">
        {/* 折叠态:只显示「思考中/已深度思考 · N 字」,点击展开 */}
        <div
          onClick={() => setExpanded(v => !v)}
          className="inline-flex items-center gap-[6px] px-[10px] py-[4px] rounded-[8px] cursor-pointer select-none coz-mg-secondary coz-fg-secondary text-[13px] hover:coz-mg-primary transition-colors"
        >
          <span>{streaming ? '思考中' : '已深度思考'}</span>
          <span className="coz-fg-dim">· {reasoning.length} 字</span>
          <svg
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2.2"
            strokeLinecap="round"
            strokeLinejoin="round"
            style={{
              transform: expanded ? 'rotate(180deg)' : 'none',
              transition: 'transform 0.15s',
            }}
          >
            <path d="M6 9l6 6 6-6" />
          </svg>
        </div>
        {expanded ? (
          <div className="mt-[6px]">
            <MdBoxLazy
              markDown={`${reasoning.replace(/^/gm, '> ')}`}
              showIndicator={streaming}
            ></MdBoxLazy>
          </div>
        ) : null}
      </div>
    );
  },
  (prev, next) =>
    prev.message.role === next.message.role &&
    prev.message.is_finish === next.message.is_finish &&
    prev.message.reasoning_content === next.message.reasoning_content &&
    prev.message.content === next.message.content,
);
