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
import { Modal, Button } from '@coze-arch/coze-design';

import type { MergeSliceInput } from '../use-merge-slices';

import styles from './index.module.less';

export interface MergeSliceConfirmModalProps {
  visible: boolean;
  slices: MergeSliceInput[];
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const MergeSliceConfirmModal: React.FC<MergeSliceConfirmModalProps> = ({
  visible,
  slices,
  loading = false,
  onConfirm,
  onCancel,
}) => {
  if (!visible) {
    return null;
  }

  const sorted = [...slices].sort((a, b) => a.sequence - b.sequence);
  const preview = sorted.map(s => s.content).join('\n\n');

  return (
    <Modal
      visible={visible}
      title={I18n.t('knowledge_merge_slice_modal_title', {
        num: slices.length,
      })}
      onCancel={onCancel}
      footer={
        <>
          <Button
            data-testid="merge-cancel-btn"
            onClick={onCancel}
            disabled={loading}
          >
            {I18n.t('datasets_createFileModel_CancelBtn')}
          </Button>
          <Button
            data-testid="merge-confirm-btn"
            type="primary"
            onClick={onConfirm}
            loading={loading}
            disabled={loading}
          >
            {I18n.t('workflow_publish_multibranch_merge')}
          </Button>
        </>
      }
    >
      <div className={styles['preview-box']}>{preview}</div>
    </Modal>
  );
};
