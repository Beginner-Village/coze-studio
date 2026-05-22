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

import { useCallback, useState } from 'react';

import { useSliceStatusPolling } from '../use-slice-status-polling';
import type { SliceBadgeStatus } from '../slice-status-badge';

export interface UseReindexTrackingResult {
  editedIds: string[];
  statusMap: Record<string, SliceBadgeStatus>;
  handleSliceEdited: (sliceId: string) => void;
}

/**
 * Bundles the editedIds bookkeeping + polling for a workspace. Returns the
 * statusMap (for rendering) and a stable handler the caller passes to merge
 * state and any edit callbacks.
 */
export const useReindexTracking = (
  documentId: string,
): UseReindexTrackingResult => {
  const [editedIds, setEditedIds] = useState<string[]>([]);
  const handleSliceEdited = useCallback((sliceId: string) => {
    if (!sliceId) {
      return;
    }
    setEditedIds(prev => (prev.includes(sliceId) ? prev : [...prev, sliceId]));
  }, []);
  const { statusMap } = useSliceStatusPolling({
    documentId,
    watchedIds: editedIds,
  });
  return { editedIds, statusMap, handleSliceEdited };
};
