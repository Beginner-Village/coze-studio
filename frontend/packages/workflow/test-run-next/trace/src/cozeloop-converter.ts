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
import type { Span } from '@coze-arch/bot-api/workflow_api';

import type { TraceFrontendSpan, Tag } from './observation-components';

// TagType.STRING = 0, InputOutputType.TEXT = 0
const TAG_TYPE_STRING = 0;
const INPUT_OUTPUT_TYPE_TEXT = 0;

/**
 * OTEL custom_tags key → old system tag key alias mapping.
 * The old workflow system expects keys like `workflow_id`, `workflow_node_id`,
 * but OTEL exports them as `id`, `node.id`.
 */
const TAG_KEY_ALIASES: Record<string, string> = {
  id: 'workflow_id',
  'node.id': 'workflow_node_id',
};

function parseStartedAtMs(startedAt: string): number {
  if (!startedAt) {
    return 0;
  }
  const asNum = Number(startedAt);
  if (!isNaN(asNum) && startedAt.trim() !== '') {
    return asNum;
  }
  const ms = new Date(startedAt).getTime();
  return isNaN(ms) ? 0 : ms;
}

function parseDurationMs(duration: string): number {
  if (!duration) {
    return 0;
  }
  const ms = Number(duration);
  return isNaN(ms) ? 0 : ms;
}

function convertCustomTagsToTags(customTags?: Record<string, string>): Tag[] {
  if (!customTags) {
    return [];
  }
  const tags: Tag[] = [];
  for (const [key, val] of Object.entries(customTags)) {
    tags.push({ key, tag_type: TAG_TYPE_STRING, value: { v_str: val } });
    const alias = TAG_KEY_ALIASES[key];
    if (alias) {
      tags.push({
        key: alias,
        tag_type: TAG_TYPE_STRING,
        value: { v_str: val },
      });
    }
  }
  return tags;
}

/**
 * Convert OutputSpan (from CozeLoop API) to Span (for trace select dropdown).
 * Maps trace_id to both trace_id and log_id so existing code that reads
 * span.log_id continues to work.
 */
export function convertOutputSpanToSpan(outputSpan: OutputSpan): Span {
  return {
    trace_id: outputSpan.trace_id,
    log_id: outputSpan.trace_id,
    span_id: outputSpan.span_id,
    parent_id: outputSpan.parent_id,
    name: outputSpan.span_name,
    type: outputSpan.span_type || outputSpan.type,
    start_time: parseStartedAtMs(outputSpan.started_at),
    duration: parseDurationMs(outputSpan.duration),
    status_code: outputSpan.status_code,
    tags: convertCustomTagsToTags(outputSpan.custom_tags),
  };
}

/**
 * Convert OutputSpan (from CozeLoop API) to TraceFrontendSpan (for trace detail view).
 */
export function convertOutputSpanToTraceFrontendSpan(
  outputSpan: OutputSpan,
): TraceFrontendSpan {
  return {
    trace_id: outputSpan.trace_id,
    log_id: outputSpan.trace_id,
    span_id: outputSpan.span_id,
    parent_id: outputSpan.parent_id,
    name: outputSpan.span_name,
    alias_name: outputSpan.span_name,
    type: outputSpan.span_type || outputSpan.type,
    start_time: parseStartedAtMs(outputSpan.started_at),
    duration: parseDurationMs(outputSpan.duration),
    status_code: outputSpan.status_code,
    input: outputSpan.input
      ? { type: INPUT_OUTPUT_TYPE_TEXT, content: outputSpan.input }
      : undefined,
    output: outputSpan.output
      ? { type: INPUT_OUTPUT_TYPE_TEXT, content: outputSpan.output }
      : undefined,
    tags: convertCustomTagsToTags(outputSpan.custom_tags),
  };
}
