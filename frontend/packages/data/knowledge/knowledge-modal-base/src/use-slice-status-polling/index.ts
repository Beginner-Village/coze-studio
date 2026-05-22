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

import { useEffect, useRef, useState } from 'react';

import { KnowledgeApi } from '@coze-arch/bot-api';

import type { SliceBadgeStatus } from '../slice-status-badge';

const POLL_INTERVAL_MS = 5000;
const TIMEOUT_MS = 120000;

// Backend slice.status enum (see backend/api/model/data/knowledge/slice.go &
// backend/domain/knowledge/internal/dal/model/progress.go)
const STATUS_INIT = 0;
const STATUS_DONE = 1;
const STATUS_PROCESSING = 2;
const STATUS_DEACTIVE = 9;

export interface UseSliceStatusPollingParams {
  documentId: string;
  watchedIds: string[];
}

const mapStatus = (status: number): SliceBadgeStatus => {
  switch (status) {
    case STATUS_INIT:
      return 'Init';
    case STATUS_DONE:
      return 'Done';
    case STATUS_PROCESSING:
      return 'Processing';
    case STATUS_DEACTIVE:
      // Deactive - treat as Init for UX (re-indexing pending)
      return 'Init';
    default:
      return status > STATUS_DONE ? 'Failed' : 'Init';
  }
};

export const useSliceStatusPolling = ({
  documentId,
  watchedIds,
}: UseSliceStatusPollingParams) => {
  const [statusMap, setStatusMap] = useState<Record<string, SliceBadgeStatus>>(
    {},
  );
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const startTsRef = useRef<number>(0);
  const watchedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (watchedIds.length === 0) {
      return;
    }
    watchedRef.current = new Set(watchedIds);
    startTsRef.current = Date.now();
    const seen: Record<string, SliceBadgeStatus> = {};

    const stopTimer = () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };

    const poll = async () => {
      try {
        const req = {
          document_id: documentId,
        } satisfies Parameters<typeof KnowledgeApi.ListSlice>[0];
        const resp = await KnowledgeApi.ListSlice(req);
        const typedResp = resp as
          | { slices?: Array<{ slice_id: string; status: number }> }
          | undefined;
        const slices = typedResp?.slices ?? [];
        const updates: Record<string, SliceBadgeStatus> = {};
        for (const s of slices) {
          const id = String(s.slice_id);
          if (!watchedRef.current.has(id)) {
            continue;
          }
          const mapped = mapStatus(s.status);
          updates[id] = mapped;
          seen[id] = mapped;
        }
        if (Object.keys(updates).length > 0) {
          setStatusMap(prev => ({ ...prev, ...updates }));
        }

        const watchedSize = watchedRef.current.size;
        const seenIds = Object.keys(seen);
        const allSeen = seenIds.length === watchedSize;
        const allDone = allSeen && seenIds.every(id => seen[id] === 'Done');

        if (allDone) {
          stopTimer();
          return;
        }

        if (Date.now() - startTsRef.current > TIMEOUT_MS) {
          const timeoutUpdates: Record<string, SliceBadgeStatus> = {};
          for (const id of watchedRef.current) {
            if (seen[id] !== 'Done') {
              timeoutUpdates[id] = 'Timeout';
            }
          }
          if (Object.keys(timeoutUpdates).length > 0) {
            setStatusMap(prev => ({ ...prev, ...timeoutUpdates }));
          }
          stopTimer();
        }
      } catch (err) {
        // 轮询错误不中断 UI，下一 tick 会自动重试；仅打印调试日志
        if (typeof console !== 'undefined') {
          console.warn('[useSliceStatusPolling] poll failed:', err);
        }
      }
    };

    void poll();
    timerRef.current = setInterval(() => {
      void poll();
    }, POLL_INTERVAL_MS);

    return () => {
      stopTimer();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 用 watchedIds.join(',') 做稳定的依赖键，避免新数组引用导致重复 effect
  }, [documentId, watchedIds.join(',')]);

  return { statusMap };
};
