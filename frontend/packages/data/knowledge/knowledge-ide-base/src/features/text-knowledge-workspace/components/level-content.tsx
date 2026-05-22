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

import classnames from 'classnames';
import { IllustrationNoResult } from '@douyinfe/semi-illustrations';
import {
  useKnowledgeStore,
  type ILevelSegment,
} from '@coze-data/knowledge-stores';
import {
  MergeSliceConfirmModal,
  useMergeSlices,
  type MergeSliceInput,
} from '@coze-data/knowledge-modal-base';
import { LevelTextKnowledgeEditor } from '@coze-data/knowledge-common-components/text-knowledge-editor';
import { I18n } from '@coze-arch/i18n';
import {
  Button,
  Checkbox,
  EmptyState,
  Toast,
  Tooltip,
} from '@coze-arch/coze-design';
import { IconSegmentEmpty } from '@coze-arch/bot-icons';

import { createLevelDocumentChunkByLevelSegment } from '../utils/document-utils';
import styles from '../styles/index.module.less';

export interface LevelContentProps {
  isProcessing: boolean;
  documentId: string;
  levelSegments: ILevelSegment[];
  selectionIDs: string[];
  onLevelSegmentsChange: (chunks: ILevelSegment[]) => void;
  onLevelSegmentDelete: (chunk: ILevelSegment) => void;
}

const MIN_MERGE_COUNT = 2;
const PREVIEW_MAX_LEN = 60;

const previewText = (text: string): string => {
  const trimmed = (text ?? '').replace(/\s+/g, ' ').trim();
  return trimmed.length > PREVIEW_MAX_LEN
    ? `${trimmed.slice(0, PREVIEW_MAX_LEN)}...`
    : trimmed;
};

interface MergeCandidate {
  slice_id: string;
  sequence: number;
  content: string;
}

const useMergeState = (levelSegments: ILevelSegment[]) => {
  const [mergeMode, setMergeMode] = useState(false);
  const [selectedSliceIds, setSelectedSliceIds] = useState<string[]>([]);
  const [mergeModalOpen, setMergeModalOpen] = useState(false);
  const [merging, setMerging] = useState(false);
  const { merge } = useMergeSlices();

  const mergeCandidates: MergeCandidate[] = useMemo(
    () =>
      levelSegments
        .filter(s => Boolean(s.slice_id))
        .map(s => ({
          slice_id: String(s.slice_id),
          sequence: Number(s.slice_sequence ?? 0),
          content: s.text ?? '',
        })),
    [levelSegments],
  );

  const selectedSlices: MergeSliceInput[] = useMemo(
    () => mergeCandidates.filter(c => selectedSliceIds.includes(c.slice_id)),
    [mergeCandidates, selectedSliceIds],
  );

  const isContiguous = useMemo(() => {
    if (selectedSlices.length < MIN_MERGE_COUNT) {
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

  const canMerge = selectedSlices.length >= MIN_MERGE_COUNT && isContiguous;

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
      // Task 3.3 will add polling registration for res.target_slice_id here
    } else if (res.stage === 'update') {
      Toast.error(I18n.t('knowledge_merge_slice_failed_toast'));
    } else if (res.stage === 'delete' && res.partial) {
      Toast.warning(I18n.t('knowledge_merge_slice_partial_toast'));
      setMergeModalOpen(false);
      exitMergeMode();
    }
  }, [merge, selectedSlices, exitMergeMode]);

  return {
    mergeMode,
    setMergeMode,
    selectedSliceIds,
    mergeModalOpen,
    setMergeModalOpen,
    merging,
    mergeCandidates,
    selectedSlices,
    isContiguous,
    canMerge,
    toggleSelected,
    exitMergeMode,
    handleMerge,
  };
};

interface MergeToolbarProps {
  mergeMode: boolean;
  selectedCount: number;
  isContiguous: boolean;
  canMerge: boolean;
  onEnterMode: () => void;
  onCancel: () => void;
  onOpenConfirm: () => void;
}

const MergeToolbar: React.FC<MergeToolbarProps> = ({
  mergeMode,
  selectedCount,
  isContiguous,
  canMerge,
  onEnterMode,
  onCancel,
  onOpenConfirm,
}) => {
  if (!mergeMode) {
    return (
      <div className="flex items-center justify-between gap-2 py-1">
        <div />
        <Button
          color="secondary"
          onClick={onEnterMode}
          data-testid="merge-enter-mode-btn"
        >
          {I18n.t('workflow_publish_multibranch_merge')}
        </Button>
      </div>
    );
  }

  const showNonContiguousHint =
    selectedCount >= MIN_MERGE_COUNT && !isContiguous;

  return (
    <div className="flex items-center justify-between gap-2 py-1">
      <div className="flex items-center gap-2 text-sm coz-fg-secondary">
        <span>
          {I18n.t('knowledge_merge_slice_selected_count', {
            num: selectedCount,
          })}
        </span>
        {showNonContiguousHint ? (
          <span className="coz-fg-hglt-red">
            {I18n.t('knowledge_merge_button_non_contiguous_tip')}
          </span>
        ) : null}
      </div>
      <div className="flex items-center gap-2">
        <Button color="secondary" onClick={onCancel}>
          {I18n.t('datasets_createFileModel_CancelBtn')}
        </Button>
        <Tooltip
          content={I18n.t('knowledge_merge_button_non_contiguous_tip')}
          trigger={canMerge ? 'custom' : 'hover'}
          visible={canMerge ? false : undefined}
        >
          <Button
            type="primary"
            disabled={!canMerge}
            onClick={onOpenConfirm}
            data-testid="merge-open-confirm-btn"
          >
            {I18n.t('workflow_publish_multibranch_merge')}
          </Button>
        </Tooltip>
      </div>
    </div>
  );
};

interface MergeCandidateListProps {
  candidates: MergeCandidate[];
  selectedSliceIds: string[];
  onToggle: (sliceId: string) => void;
}

const MergeCandidateList: React.FC<MergeCandidateListProps> = ({
  candidates,
  selectedSliceIds,
  onToggle,
}) => (
  <div className="flex flex-col gap-1 border border-solid coz-stroke-primary rounded-[6px] p-2 max-h-[200px] overflow-auto">
    {candidates.map(c => (
      <label
        key={c.slice_id}
        className="flex items-start gap-2 px-1 py-1 cursor-pointer hover:coz-mg-secondary-hovered rounded"
      >
        <Checkbox
          checked={selectedSliceIds.includes(c.slice_id)}
          onChange={() => onToggle(c.slice_id)}
          data-testid={`merge-checkbox-${c.slice_id}`}
        />
        <span className="text-xs coz-fg-secondary shrink-0 w-10">
          #{c.sequence}
        </span>
        <span className="text-sm coz-fg-primary truncate">
          {previewText(c.content)}
        </span>
      </label>
    ))}
  </div>
);

export const LevelContent: React.FC<LevelContentProps> = ({
  isProcessing,
  documentId,
  levelSegments,
  selectionIDs,
  onLevelSegmentsChange,
  onLevelSegmentDelete,
}) => {
  const canEdit = useKnowledgeStore(state => state.canEdit);
  const searchValue = useKnowledgeStore(state => state.searchValue);

  // Convert hierarchical segmented data into an editor-usable format
  const renderLevelSegmentsData = levelSegments.map(item =>
    createLevelDocumentChunkByLevelSegment(item),
  );

  const {
    mergeMode,
    setMergeMode,
    selectedSliceIds,
    mergeModalOpen,
    setMergeModalOpen,
    merging,
    mergeCandidates,
    selectedSlices,
    isContiguous,
    canMerge,
    toggleSelected,
    exitMergeMode,
    handleMerge,
  } = useMergeState(levelSegments);

  if (levelSegments.length === 0) {
    return (
      <div className={classnames(styles['empty-content'])}>
        <EmptyState
          size="large"
          icon={
            searchValue ? (
              <IllustrationNoResult style={{ width: 150, height: '100%' }} />
            ) : (
              <IconSegmentEmpty style={{ width: 150, height: '100%' }} />
            )
          }
          title={
            isProcessing
              ? I18n.t('content_view_003')
              : searchValue
                ? I18n.t('knowledge_no_result')
                : I18n.t('dataset_segment_empty_desc')
          }
        />
      </div>
    );
  }

  const showMergeToolbar = canEdit && mergeCandidates.length >= MIN_MERGE_COUNT;

  return (
    <div className="flex flex-col gap-2 w-full">
      {showMergeToolbar ? (
        <MergeToolbar
          mergeMode={mergeMode}
          selectedCount={selectedSlices.length}
          isContiguous={isContiguous}
          canMerge={canMerge}
          onEnterMode={() => setMergeMode(true)}
          onCancel={exitMergeMode}
          onOpenConfirm={() => setMergeModalOpen(true)}
        />
      ) : null}

      {mergeMode ? (
        <MergeCandidateList
          candidates={mergeCandidates}
          selectedSliceIds={selectedSliceIds}
          onToggle={toggleSelected}
        />
      ) : null}

      <LevelTextKnowledgeEditor
        chunks={renderLevelSegmentsData}
        selectionIDs={selectionIDs}
        documentId={documentId}
        readonly={!canEdit}
        onChange={onLevelSegmentsChange}
        onDeleteChunk={onLevelSegmentDelete}
      />

      <MergeSliceConfirmModal
        visible={mergeModalOpen}
        slices={selectedSlices}
        loading={merging}
        onConfirm={handleMerge}
        onCancel={() => setMergeModalOpen(false)}
      />
    </div>
  );
};
