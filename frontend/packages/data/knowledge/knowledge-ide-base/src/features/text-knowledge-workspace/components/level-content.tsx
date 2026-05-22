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

import { useCallback, useMemo } from 'react';

import classnames from 'classnames';
import { IllustrationNoResult } from '@douyinfe/semi-illustrations';
import {
  useKnowledgeStore,
  type ILevelSegment,
} from '@coze-data/knowledge-stores';
import {
  MergeSliceConfirmModal,
  useMergeState,
  MergeToolbar,
  MergeCandidateList,
  MERGE_MIN_COUNT,
  ReindexStatusBar,
  useReindexTracking,
  type MergeCandidate,
} from '@coze-data/knowledge-modal-base';
import { LevelTextKnowledgeEditor } from '@coze-data/knowledge-common-components/text-knowledge-editor';
import { I18n } from '@coze-arch/i18n';
import { EmptyState } from '@coze-arch/coze-design';
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

  // Track slice ids that have been edited/merged so we can poll their re-index
  // status. We accumulate ids (dedup) and rely on useSliceStatusPolling to
  // transition each one to "Done" — at which point we hide it from the bar.
  const { editedIds, statusMap, handleSliceEdited } =
    useReindexTracking(documentId);

  // Diff incoming chunks against current levelSegments to detect which slice
  // text was edited (LevelTextKnowledgeEditor doesn't expose a per-chunk
  // edited callback, so we intercept the existing onChange).
  const handleLevelSegmentsChange = useCallback(
    (chunks: ILevelSegment[]) => {
      const prevMap = new Map(
        levelSegments
          .filter(s => Boolean(s.slice_id))
          .map(s => [String(s.slice_id), s.text ?? '']),
      );
      for (const c of chunks) {
        if (!c.slice_id) {
          continue;
        }
        const id = String(c.slice_id);
        const prevText = prevMap.get(id);
        if (prevText !== undefined && prevText !== (c.text ?? '')) {
          handleSliceEdited(id);
        }
      }
      onLevelSegmentsChange(chunks);
    },
    [levelSegments, onLevelSegmentsChange, handleSliceEdited],
  );

  const {
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
  } = useMergeState(mergeCandidates, {
    onMergeSuccess: handleSliceEdited,
  });

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

  const showMergeToolbar = canEdit && mergeCandidates.length >= MERGE_MIN_COUNT;

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

      <ReindexStatusBar
        watchedIds={editedIds}
        statusMap={statusMap}
        testId="text-reindex-status-bar"
      />

      <LevelTextKnowledgeEditor
        chunks={renderLevelSegmentsData}
        selectionIDs={selectionIDs}
        documentId={documentId}
        readonly={!canEdit}
        onChange={handleLevelSegmentsChange}
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
