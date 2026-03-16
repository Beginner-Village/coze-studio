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

import { useEffect, useState } from 'react';

import dayjs from 'dayjs';
import { useMemoizedFn } from 'ahooks';
import { type TraceFrontendSpan } from '@coze-arch/bot-api/workflow_api';

import { sortSpans } from '../../utils';
import { convertOutputSpanToTraceFrontendSpan } from '../../cozeloop-converter';
import { getTrace } from '../../cozeloop-api';
import { useTraceListStore } from '../../contexts';
import { MAX_TRACE_TIME } from '../../constants';

export const useTrace = () => {
  const [loading, setLoading] = useState(false);
  const [spans, setSpans] = useState<TraceFrontendSpan[] | null>(null);

  const { span, spaceId } = useTraceListStore(store => ({
    span: store.span,
    spaceId: store.spaceId,
  }));

  const fetch = useMemoizedFn(async (traceId: string) => {
    setLoading(true);
    /** When querying the log, the start and end time must be passed. Since the user can check the range within 7 days, he can directly fake the 7-day time interval. */
    const now = dayjs().endOf('day').valueOf();
    const end = dayjs()
      .subtract(MAX_TRACE_TIME, 'day')
      .startOf('day')
      .valueOf();

    try {
      const resp = await getTrace({
        workspace_id: spaceId,
        trace_id: traceId,
        start_time: String(end),
        end_time: String(now),
      });
      if (!resp || !resp.spans) {
        return;
      }
      const converted = resp.spans.map(convertOutputSpanToTraceFrontendSpan);
      const next = sortSpans(converted);
      setSpans(next);
    } finally {
      setLoading(false);
    }
  });

  useEffect(() => {
    // log_id is mapped from trace_id by the converter
    const traceId = span?.trace_id || span?.log_id;
    if (traceId) {
      fetch(traceId);
    }
  }, [span, fetch]);

  return { spans, loading };
};
