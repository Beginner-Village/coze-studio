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

import { Modal, Button } from '@coze-arch/coze-design';

import type { MergeSliceInput } from '../use-merge-slices';

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
      title={`合并 ${slices.length} 个切片`}
      onCancel={onCancel}
      footer={
        <>
          <Button onClick={onCancel} disabled={loading}>
            取消
          </Button>
          <Button
            type="primary"
            onClick={onConfirm}
            loading={loading}
            disabled={loading}
          >
            合并
          </Button>
        </>
      }
    >
      <div
        style={{
          maxHeight: 400,
          overflow: 'auto',
          whiteSpace: 'pre-wrap',
          padding: 12,
          border: '1px solid #eee',
          borderRadius: 4,
        }}
      >
        {preview}
      </div>
    </Modal>
  );
};
