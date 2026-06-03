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

import { useWorkflowResourceAction } from '@coze-workflow/components';
import { useUserInfo } from '@coze-foundation/account-adapter';
import { ResType, WorkflowMode } from '@coze-arch/idl/plugin_develop';
import { I18n } from '@coze-arch/i18n';
import { IconCozChat, IconCozWorkflow } from '@coze-arch/coze-design/icons';
import {
  Menu,
  Tag,
  Toast,
  type TableActionProps,
} from '@coze-arch/coze-design';

import { useFolderManagement, type FolderInfo } from '../use-folder-management';
import { BaseLibraryItem } from '../../components/base-library-item';
import WorkflowDefaultIcon from '../../assets/workflow_default_icon.png';
import ImageFlowDefaultIcon from '../../assets/image_flow_default_icon.png';
import { type UseEntityConfigHook } from './types';

const defaultIconMap: { [key in ResType]?: string } = {
  [ResType.Workflow]: WorkflowDefaultIcon,
  [ResType.Imageflow]: ImageFlowDefaultIcon,
};

// Resource type used by the folder backend to scope workflow statistics
const FOLDER_WORKFLOW_RESOURCE_TYPE = 2;
// folder_id '0' means "move out" (back to the top level)
const FOLDER_ROOT_ID = '0';

type FolderActionList = NonNullable<TableActionProps['actionList']>;

// Builds the "move into folder" / "move out of folder" card menu actions.
const buildFolderActions = (params: {
  resId?: string;
  currentFolderId?: string;
  folders: FolderInfo[];
  onMove: (folderId: string, resId?: string) => void;
}): FolderActionList => {
  const { resId, currentFolderId, folders, onMove } = params;
  const actions: FolderActionList = [
    {
      customRender: (
        <Menu.SubMenu
          key="move-into-folder"
          mode="menu"
          data-testid="workspace.library.item.move-into-folder"
        >
          <Menu.Title>
            {I18n.t('workspace_library_folder_move_into') || '移入分类'}
          </Menu.Title>
          {folders.length ? (
            folders.map(folder => (
              <Menu.Item
                key={folder.id}
                disabled={folder.id === currentFolderId}
                onClick={() => onMove(folder.id, resId)}
              >
                {folder.name}
              </Menu.Item>
            ))
          ) : (
            <Menu.Item disabled>
              {I18n.t('workspace_library_folder_empty') || '暂无分类'}
            </Menu.Item>
          )}
        </Menu.SubMenu>
      ),
    },
  ];
  if (currentFolderId) {
    actions.push({
      actionKey: 'move-out-folder',
      actionText: I18n.t('workspace_library_folder_move_out') || '移出分类',
      handler: () => onMove(FOLDER_ROOT_ID, resId),
    });
  }
  return actions;
};

export const useWorkflowConfig: UseEntityConfigHook = ({
  spaceId,
  reloadList,
  getCommonActions,
}) => {
  const userInfo = useUserInfo();

  const { folders, moveResourcesToFolder, refreshFolders } =
    useFolderManagement({
      spaceId,
      resourceType: FOLDER_WORKFLOW_RESOURCE_TYPE,
    });

  const handleMove = (folderId: string, resId?: string): void => {
    if (!resId) {
      return;
    }
    void (async () => {
      try {
        await moveResourcesToFolder(
          folderId,
          [resId],
          FOLDER_WORKFLOW_RESOURCE_TYPE,
        );
        await refreshFolders();
        reloadList();
      } catch (error) {
        Toast.error(
          I18n.t('workspace_library_folder_move_failed') || '移动失败',
        );
      }
    })();
  };

  // Inject folder move actions, then chain the externally provided actions.
  const getWorkflowCommonActions: NonNullable<
    typeof getCommonActions
  > = item => {
    const currentFolderId = folders.find(folder =>
      (folder.resource_ids ?? []).includes(item.res_id ?? ''),
    )?.id;
    return [
      ...buildFolderActions({
        resId: item.res_id,
        currentFolderId,
        folders,
        onMove: handleMove,
      }),
      ...(getCommonActions?.(item) ?? []),
    ];
  };

  const {
    workflowResourceModals,
    handleWorkflowResourceClick,
    renderWorkflowResourceActions,
    openCreateModal,
  } = useWorkflowResourceAction({
    spaceId,
    userId: userInfo?.user_id_str,
    refreshPage: reloadList,
    getCommonActions: getWorkflowCommonActions,
  });

  return {
    modals: workflowResourceModals,
    config: {
      typeFilter: {
        label: I18n.t('library_resource_type_workflow'),
        value: ResType.Workflow,
      },
      onCreate: (isChat: Boolean) => {
        openCreateModal(isChat ? WorkflowMode.ChatFlow : WorkflowMode.Workflow);
      },
      parseParams: params => {
        // After the workflow image stream is merged, the selected workflow needs to also pull out the image stream
        if (params?.res_type_filter?.[0] === ResType.Workflow) {
          return {
            ...params,
            is_get_imageflow: true,
          };
        }
        return params;
      },
      renderCreateMenu: () => (
        <>
          <Menu.Item
            data-testid="workspace.library.header.create.workflow"
            icon={<IconCozWorkflow />}
            onClick={() => {
              openCreateModal(WorkflowMode.Workflow);
            }}
          >
            {I18n.t('library_resource_type_workflow')}
          </Menu.Item>
          {/* The open-source version does not support conversation streaming for the time being */}
          {!IS_OPEN_SOURCE ? (
            <Menu.Item
              data-testid="workspace.library.header.create.chatflow"
              icon={<IconCozChat />}
              onClick={() => {
                openCreateModal(WorkflowMode.ChatFlow);
              }}
            >
              {I18n.t('wf_chatflow_76')}
            </Menu.Item>
          ) : null}
        </>
      ),
      target: [ResType.Workflow, ResType.Imageflow],
      onItemClick: handleWorkflowResourceClick,
      renderItem: item => (
        <BaseLibraryItem
          resourceInfo={item}
          defaultIcon={
            item.res_type !== undefined
              ? defaultIconMap[item.res_type]
              : undefined
          }
          tag={
            item.collaboration_enable === true ? (
              <Tag
                data-testid="workspace.library.item.tag"
                color="brand"
                size="mini"
                className="flex-shrink-0 flex-grow-0"
              >
                {I18n.t('library_filter_tags_collaboration')}
              </Tag>
            ) : null
          }
        />
      ),
      renderResType: item =>
        item.res_type === ResType.Workflow &&
        item.res_sub_type === WorkflowMode.ChatFlow
          ? I18n.t('wf_chatflow_76')
          : I18n.t('library_resource_type_workflow'),
      renderActions: renderWorkflowResourceActions,
    },
  };
};
