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

import type { OutputSpan } from '@coze-arch/idl/stone_cozeloop_observability_api';
import type {
  TraceFrontendSpan,
  Tag,
} from '@coze-workflow/test-run-trace/observation-components';

// 直接用常量值，避免 enum 导入在 production build 中被 tree-shake
// TagType.STRING = 0, InputOutputType.TEXT = 0
const TAG_TYPE_STRING = 0;
const INPUT_OUTPUT_TYPE_TEXT = 0;

/**
 * 将 started_at 字符串（毫秒时间戳）解析为毫秒数值
 * TraceTree 组件内部按毫秒处理 start_time 和 duration
 */
function parseStartedAtMs(startedAt: string): number {
  if (!startedAt) return 0;
  const asNum = Number(startedAt);
  if (!isNaN(asNum) && startedAt.trim() !== '') {
    return asNum;
  }
  const ms = new Date(startedAt).getTime();
  return isNaN(ms) ? 0 : ms;
}

/**
 * 将 duration 字符串（毫秒）解析为毫秒数值
 * TraceTree 组件内部按毫秒处理 duration
 */
function parseDurationMs(duration: string): number {
  if (!duration) return 0;
  const ms = Number(duration);
  return isNaN(ms) ? 0 : ms;
}

function convertCustomTags(
  customTags?: Record<string, string>,
): Tag[] {
  if (!customTags) return [];
  return Object.entries(customTags).map(([key, val]) => ({
    key,
    tag_type: TAG_TYPE_STRING,
    value: { v_str: val },
  }));
}

export function convertOutputSpan(span: OutputSpan): TraceFrontendSpan {
  return {
    trace_id: span.trace_id,
    span_id: span.span_id,
    parent_id: span.parent_id,
    name: span.span_name,
    alias_name: span.span_name,
    type: span.span_type || span.type,
    start_time: parseStartedAtMs(span.started_at),
    duration: parseDurationMs(span.duration),
    status_code: span.status_code,
    input: span.input
      ? { type: INPUT_OUTPUT_TYPE_TEXT, content: span.input }
      : undefined,
    output: span.output
      ? { type: INPUT_OUTPUT_TYPE_TEXT, content: span.output }
      : undefined,
    tags: convertCustomTags(span.custom_tags),
  };
}

export function convertOutputSpans(
  spans: OutputSpan[],
): TraceFrontendSpan[] {
  return spans.map(convertOutputSpan);
}
