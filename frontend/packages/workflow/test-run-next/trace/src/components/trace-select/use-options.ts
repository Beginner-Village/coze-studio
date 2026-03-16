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

import { useCallback, useEffect, useRef, useState } from 'react';

import dayjs from 'dayjs';
import { useMemoizedFn } from 'ahooks';
import { SpanStatus, type Span } from '@coze-arch/bot-api/workflow_api';

import { convertOutputSpanToSpan } from '../../cozeloop-converter';
import { listSpans } from '../../cozeloop-api';
import { useTraceListStore } from '../../contexts';
import { MAX_TRACE_LENGTH, MAX_TRACE_TIME } from '../../constants';

const getDefaultDate = (): [Date, Date] => {
  const end = dayjs().endOf('day').toDate();
  const start = dayjs()
    .subtract(MAX_TRACE_TIME - 1, 'day')
    .startOf('day')
    .toDate();

  return [start, end];
};

export const useOptions = (workflowId: string) => {
  const [date, setDate] = useState<[Date, Date]>(getDefaultDate());
  const [status, setStatus] = useState<SpanStatus>(SpanStatus.Unknown);
  const [options, setOptions] = useState<Span[]>([]);

  const optionsCacheRef = useRef(new Map<string, Span>());

  const { ready, span, patch, spaceId } = useTraceListStore(store => ({
    span: store.span,
    ready: store.ready,
    patch: store.patch,
    spaceId: store.spaceId,
  }));

  const fetch = useMemoizedFn(async () => {
    const searchParams = new URLSearchParams(location.search);
    const executeMode = searchParams.get('execute_mode');
    const executeId = searchParams.get('execute_id');

    const resp = await listSpans({
      workspace_id: spaceId,
      start_time: String(date[0].getTime()),
      end_time: String(date[1].getTime()),
      filters: {
        filter_fields: [
          {
            field_name: 'span_type',
            values: ['Workflow'],
            query_type: 'eq',
          },
        ],
      },
      page_size: MAX_TRACE_LENGTH,
      order_bys: [{ field: 'started_at', is_asc: false }],
    });

    const outputSpans = resp.spans || [];

    // Client-side filter by workflowId (custom_tags.id)
    let filtered = outputSpans.filter(s => s.custom_tags?.id === workflowId);

    // Client-side status filter
    if (status !== SpanStatus.Unknown) {
      filtered = filtered.filter(s => s.status_code === status);
    }

    // Client-side execute_mode filter
    if (executeMode) {
      filtered = filtered.filter(
        s => s.custom_tags?.execute_mode === executeMode,
      );
    }

    const next: Span[] = filtered.map(convertOutputSpanToSpan);
    let maybeInitialSpan = next[0];

    // Handle execute_id from URL: find matching span
    if (executeId && !ready && !span) {
      const matchByExecuteId = filtered.find(
        s =>
          s.custom_tags?.execute_id === executeId ||
          s.custom_tags?.root_execute_id === executeId,
      );
      if (matchByExecuteId) {
        const converted = convertOutputSpanToSpan(matchByExecuteId);
        maybeInitialSpan = converted;
        const exists = next.find(i => i.log_id === converted.log_id);
        if (!exists) {
          next.unshift(converted);
        }
      }
    }

    next.forEach(s => {
      if (s.log_id) {
        optionsCacheRef.current.set(s.log_id, s);
      }
    });
    setOptions(next);
    // If it is not initialized, it is initialized once
    if (!ready && !span && maybeInitialSpan) {
      patch({ span: maybeInitialSpan });
    }
    if (!ready) {
      patch({ ready: true });
    }
  });

  const handleDateChange = useCallback(
    (next: [Date, Date]) => {
      const [start, end] = next;
      // The date selected by the time selector is 0:00 on the day, which needs to be converted to 11:59:59 on the day.
      setDate([start, dayjs(end).endOf('day').toDate()] as [Date, Date]);
    },
    [setDate],
  );

  useEffect(() => {
    fetch();
  }, [date, status, fetch]);

  return {
    date,
    status,
    setStatus,
    options,
    optionsCacheRef,
    fetch,
    onDateChange: handleDateChange,
  };
};
