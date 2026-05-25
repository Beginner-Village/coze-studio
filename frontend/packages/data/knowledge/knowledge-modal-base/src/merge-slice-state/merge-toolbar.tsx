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

import React from 'react';

import { I18n } from '@coze-arch/i18n';
import { Button, Tooltip } from '@coze-arch/coze-design';

import { MERGE_MIN_COUNT } from './use-merge-state';

export interface MergeToolbarProps {
  mergeMode: boolean;
  selectedCount: number;
  isContiguous: boolean;
  canMerge: boolean;
  onEnterMode: () => void;
  onCancel: () => void;
  onOpenConfirm: () => void;
}

export const MergeToolbar: React.FC<MergeToolbarProps> = ({
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
    selectedCount >= MERGE_MIN_COUNT && !isContiguous;

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
