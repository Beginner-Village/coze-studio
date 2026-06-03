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

import { I18n } from '@coze-arch/i18n';
import { Space, Typography } from '@coze-arch/coze-design';

import { type FolderInfo } from '../hooks/use-folder-management';

export const FolderCard: React.FC<{
  folder: FolderInfo;
  gridItemWidth?: number;
  onClick: (folder: FolderInfo) => void;
}> = ({ folder, gridItemWidth, onClick }) => (
  <div
    className="flex-col cursor-pointer"
    data-testid="workspace.library.folder.card"
    onClick={() => onClick(folder)}
  >
    <div className="w-full h-[122px] flex items-center justify-center bg-[#F9FAFD] rounded-[6px]">
      <div className="text-[56px] leading-none">📁</div>
    </div>
    <div className="flex flex-col gap-[2px] mt-[10px]">
      <div className="h-[20px] flex-shrink-0">
        <Space spacing={4} className="w-full">
          <Typography.Text
            data-testid="workspace.library.folder.name"
            className="h-[20px] text-[16px] coz-fg-primary leading-[20px]"
            style={{
              maxWidth: gridItemWidth ? `${gridItemWidth - 64}px` : '',
            }}
            ellipsis={{ showTooltip: true }}
          >
            <span className="font-[600]">{folder.name}</span>
          </Typography.Text>
        </Space>
      </div>
      <div className="mt-[12px] h-[16px] text-[12px] coz-fg-secondary leading-[16px]">
        {I18n.t('workspace_library_folder_count', {
          count: folder.resource_count ?? folder.resource_ids?.length ?? 0,
        }) ||
          `${folder.resource_count ?? folder.resource_ids?.length ?? 0} 个工作流`}
      </div>
    </div>
  </div>
);
