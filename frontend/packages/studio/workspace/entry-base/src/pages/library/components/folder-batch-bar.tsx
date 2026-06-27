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

import { IconCozCheckMark, IconCozCross } from '@coze-arch/coze-design/icons';
import { Button, IconButton } from '@coze-arch/coze-design';

export const FolderBatchBar: React.FC<{
  selectedCount: number;
  folderName: string;
  loading?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}> = ({ selectedCount, folderName, loading = false, onCancel, onConfirm }) => (
  <div
    data-testid="workspace.library.folder.batch-bar"
    className="fixed bottom-[32px] left-1/2 -translate-x-1/2 z-[100] flex items-center gap-[12px] px-[12px] py-[8px] rounded-[14px]"
    style={{
      background: 'var(--coz-bg-max, #fff)',
      border: '1px solid var(--coz-stroke-primary, rgba(82, 100, 154, 0.13))',
      boxShadow: '0 4px 12px rgb(29 28 35 / 8%), 0 8px 24px rgb(29 28 35 / 6%)',
    }}
  >
    <IconButton
      color="secondary"
      icon={<IconCozCross />}
      onClick={onCancel}
      data-testid="workspace.library.folder.batch-cancel"
    />
    <span className="text-[14px] coz-fg-primary whitespace-nowrap">
      {`已选 ${selectedCount} 个工作流`}
      <span className="coz-fg-secondary">{' · '}</span>
      {`移入「${folderName}」`}
    </span>
    <Button
      theme="solid"
      type="primary"
      icon={<IconCozCheckMark />}
      loading={loading}
      disabled={selectedCount === 0}
      onClick={onConfirm}
      data-testid="workspace.library.folder.batch-confirm"
    >
      确认
    </Button>
  </div>
);
