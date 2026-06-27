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

import { ProjectResourceActionKey } from '@coze-arch/bot-api/plugin_develop';

import { type BizResourceType } from './type';

export const WORKFLOW_FOLDER_RESOURCE_TYPE = 2;
export const ROOT_FOLDER_ID = '0';
export const RESOURCE_FOLDER_TYPE = 'folder';
const RESOURCE_FOLDER_ID_PREFIX = 'resource-folder-id-';
const ROOT_KEY = '$-ROOT-$';

interface FolderTreeFolder {
  id: string | number;
  parent_id?: string | number;
  name: string;
  description?: string;
  resource_ids?: Array<string | number>;
}

const normalizeId = (id?: string | number | null): string =>
  id === undefined || id === null ? '' : String(id);

export const getResourceFolderId = (folderId: string | number): string =>
  `${RESOURCE_FOLDER_ID_PREFIX}${folderId}`;

export const getBackendFolderId = (
  resourceFolderId?: string | number,
): string => {
  const id = normalizeId(resourceFolderId);
  if (!id || id === ROOT_KEY) {
    return ROOT_FOLDER_ID;
  }
  return id.startsWith(RESOURCE_FOLDER_ID_PREFIX)
    ? id.slice(RESOURCE_FOLDER_ID_PREFIX.length)
    : id;
};

const byName = (left: BizResourceType, right: BizResourceType): number =>
  (left.name || '').localeCompare(right.name || '');

const sortResourceTree = (items: BizResourceType[]): BizResourceType[] => {
  const folders = items
    .filter(item => item.type === RESOURCE_FOLDER_TYPE)
    .map(item => ({
      ...item,
      children: sortResourceTree(item.children || []),
    }))
    .sort(byName);
  const resources = items
    .filter(item => item.type !== RESOURCE_FOLDER_TYPE)
    .sort(byName);
  return [...folders, ...resources];
};

export const createFolderedResourceTree = ({
  folders,
  resources,
}: {
  folders: FolderTreeFolder[];
  resources: BizResourceType[];
}): BizResourceType[] => {
  const resourceById = new Map(
    resources.map(resource => [String(resource.id), resource]),
  );
  const assignedResourceIds = new Set<string>();
  const folderNodeById = new Map<string, BizResourceType>();

  folders.forEach(folder => {
    const folderId = normalizeId(folder.id);
    const folderNode: BizResourceType = {
      ...folder,
      id: getResourceFolderId(folderId),
      folder_id: folderId,
      type: RESOURCE_FOLDER_TYPE,
      name: folder.name,
      description: folder.description,
      children: [],
      actions: [
        { key: ProjectResourceActionKey.Rename, enable: true },
        { key: ProjectResourceActionKey.Delete, enable: true },
      ],
    };
    folderNodeById.set(folderId, folderNode);
  });

  const rootFolders: BizResourceType[] = [];

  folders.forEach(folder => {
    const folderId = normalizeId(folder.id);
    const node = folderNodeById.get(folderId);
    if (!node) {
      return;
    }
    const parentId = normalizeId(folder.parent_id);
    const parent = parentId ? folderNodeById.get(parentId) : undefined;
    if (parent) {
      parent.children = [...(parent.children || []), node];
    } else {
      rootFolders.push(node);
    }
  });

  folders.forEach(folder => {
    const folderId = normalizeId(folder.id);
    const node = folderNodeById.get(folderId);
    if (!node) {
      return;
    }
    (folder.resource_ids || []).forEach(resourceId => {
      const normalizedResourceId = normalizeId(resourceId);
      const resource = resourceById.get(normalizedResourceId);
      if (!resource) {
        return;
      }
      assignedResourceIds.add(normalizedResourceId);
      node.children = [...(node.children || []), resource];
    });
  });

  const unclassifiedResources = resources.filter(
    resource => !assignedResourceIds.has(String(resource.id)),
  );

  return sortResourceTree([...rootFolders, ...unclassifiedResources]);
};
