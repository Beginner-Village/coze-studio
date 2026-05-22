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

import { useCallback, useMemo, useState } from 'react';

import { I18n } from '@coze-arch/i18n';
import { Toast } from '@coze-arch/coze-design';

import { useMergeSlices, type MergeSliceInput } from '../use-merge-slices';

export const MERGE_MIN_COUNT = 2;

export interface MergeCandidate {
  slice_id: string;
  sequence: number;
  content: string;
}

export interface UseMergeStateOptions {
  /**
   * Called with the new target slice id after a successful merge.
   * Used by callers that want to wire polling (e.g. Task 3.3).
   */
  onMergeSuccess?: (targetSliceId: string) => void;
}

/**
 * Generic merge state hook reused by text/table/image workspaces.
 *
 * Accepts a pre-projected list of merge candidates so it stays decoupled from
 * any specific slice shape (text level segment, table row, image slice ...).
 */
export const useMergeState = (
  candidates: MergeCandidate[],
  options: UseMergeStateOptions = {},
) => {
  const { onMergeSuccess } = options;
  const [mergeMode, setMergeMode] = useState(false);
  const [selectedSliceIds, setSelectedSliceIds] = useState<string[]>([]);
  const [mergeModalOpen, setMergeModalOpen] = useState(false);
  const [merging, setMerging] = useState(false);
  const { merge } = useMergeSlices();

  const selectedSlices: MergeSliceInput[] = useMemo(
    () => candidates.filter(c => selectedSliceIds.includes(c.slice_id)),
    [candidates, selectedSliceIds],
  );

  const isContiguous = useMemo(() => {
    if (selectedSlices.length < MERGE_MIN_COUNT) {
      return false;
    }
    const sorted = [...selectedSlices].sort((a, b) => a.sequence - b.sequence);
    for (let i = 1; i < sorted.length; i++) {
      if (sorted[i].sequence !== sorted[i - 1].sequence + 1) {
        return false;
      }
    }
    return true;
  }, [selectedSlices]);

  const canMerge = selectedSlices.length >= MERGE_MIN_COUNT && isContiguous;

  const toggleSelected = useCallback((sliceId: string) => {
    setSelectedSliceIds(prev =>
      prev.includes(sliceId)
        ? prev.filter(id => id !== sliceId)
        : [...prev, sliceId],
    );
  }, []);

  const exitMergeMode = useCallback(() => {
    setMergeMode(false);
    setSelectedSliceIds([]);
  }, []);

  const handleMerge = useCallback(async () => {
    setMerging(true);
    const res = await merge(selectedSlices);
    setMerging(false);
    if (res.ok) {
      Toast.success(I18n.t('knowledge_merge_slice_success_toast'));
      setMergeModalOpen(false);
      exitMergeMode();
      onMergeSuccess?.(res.target_slice_id);
    } else if (res.stage === 'update') {
      Toast.error(I18n.t('knowledge_merge_slice_failed_toast'));
    } else if (res.stage === 'delete' && res.partial) {
      Toast.warning(I18n.t('knowledge_merge_slice_partial_toast'));
      setMergeModalOpen(false);
      exitMergeMode();
    }
  }, [merge, selectedSlices, exitMergeMode, onMergeSuccess]);

  return {
    mergeMode,
    setMergeMode,
    selectedSliceIds,
    mergeModalOpen,
    setMergeModalOpen,
    merging,
    selectedSlices,
    isContiguous,
    canMerge,
    toggleSelected,
    exitMergeMode,
    handleMerge,
  };
};
