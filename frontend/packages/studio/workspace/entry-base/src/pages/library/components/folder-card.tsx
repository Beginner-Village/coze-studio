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

import { useState } from 'react';

import { I18n } from '@coze-arch/i18n';
import {
  IconCozMore,
  IconCozEdit,
  IconCozTrashCan,
} from '@coze-arch/coze-design/icons';
import {
  Space,
  Typography,
  Menu,
  IconButton,
  Modal,
  Input,
  Toast,
} from '@coze-arch/coze-design';

import { type FolderInfo } from '../hooks/use-folder-management';

export const FolderCard: React.FC<{
  folder: FolderInfo;
  gridItemWidth?: number;
  onClick: (folder: FolderInfo) => void;
  onRename?: (folder: FolderInfo, name: string) => Promise<void> | void;
  onDelete?: (folder: FolderInfo) => Promise<void> | void;
}> = ({ folder, gridItemWidth, onClick, onRename, onDelete }) => {
  const [menuVisible, setMenuVisible] = useState(false);
  const [renameOpen, setRenameOpen] = useState(false);
  const [renameValue, setRenameValue] = useState(folder.name);
  const [submitting, setSubmitting] = useState(false);

  const stop = (e: React.MouseEvent) => e.stopPropagation();

  const handleRenameConfirm = async () => {
    const name = renameValue.trim();
    if (!name) {
      Toast.warning(
        I18n.t('workspace_library_folder_name_required') || '请输入分类名称',
      );
      return;
    }
    try {
      setSubmitting(true);
      await onRename?.(folder, name);
      setRenameOpen(false);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = () => {
    Modal.confirm({
      title: I18n.t('workspace_library_folder_delete_title') || '删除分类',
      content:
        I18n.t('workspace_library_folder_delete_tip') ||
        '删除后分类内的工作流会移回未分类，不会被删除。确认删除该分类？',
      okText: I18n.t('Delete') || '删除',
      okButtonProps: { color: 'red' },
      cancelText: I18n.t('Cancel') || '取消',
      onOk: async () => {
        await onDelete?.(folder);
      },
    });
  };

  const showMenu = Boolean(onRename || onDelete);

  return (
    <>
      <div
        className="flex-col cursor-pointer relative group"
        data-testid="workspace.library.folder.card"
        onClick={() => onClick(folder)}
      >
        {showMenu ? (
          <div
            className="absolute top-[8px] right-[8px] z-10 opacity-0 group-hover:opacity-100"
            onClick={stop}
          >
            <Menu
              trigger="click"
              position="bottomRight"
              visible={menuVisible}
              onVisibleChange={setMenuVisible}
              render={
                <Menu.SubMenu mode="menu">
                  {onRename ? (
                    <Menu.Item
                      icon={<IconCozEdit />}
                      data-testid="workspace.library.folder.rename"
                      onClick={() => {
                        setRenameValue(folder.name);
                        setRenameOpen(true);
                        setMenuVisible(false);
                      }}
                    >
                      {I18n.t('workspace_library_folder_rename') || '重命名'}
                    </Menu.Item>
                  ) : null}
                  {onDelete ? (
                    <Menu.Item
                      icon={<IconCozTrashCan />}
                      data-testid="workspace.library.folder.delete"
                      onClick={() => {
                        setMenuVisible(false);
                        handleDelete();
                      }}
                    >
                      {I18n.t('Delete') || '删除'}
                    </Menu.Item>
                  ) : null}
                </Menu.SubMenu>
              }
            >
              <IconButton
                size="small"
                color="secondary"
                icon={<IconCozMore />}
                data-testid="workspace.library.folder.more"
              />
            </Menu>
          </div>
        ) : null}
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
      <Modal
        visible={renameOpen}
        title={I18n.t('workspace_library_folder_rename') || '重命名分类'}
        okText={I18n.t('Confirm') || '确定'}
        cancelText={I18n.t('Cancel') || '取消'}
        confirmLoading={submitting}
        onOk={handleRenameConfirm}
        onCancel={() => setRenameOpen(false)}
      >
        <Input
          value={renameValue}
          onChange={setRenameValue}
          maxLength={50}
          placeholder={
            I18n.t('workspace_library_folder_name_placeholder') ||
            '请输入分类名称'
          }
          onEnterPress={handleRenameConfirm}
        />
      </Modal>
    </>
  );
};
