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

import { useState, useCallback, useEffect } from 'react';

import { folderApi, type FolderInfo } from '@coze-arch/bot-api';

export type { FolderInfo };

export interface UseFolderManagementProps {
  spaceId: string;
  /** Resource type used to scope folder resource statistics (e.g. 2 = workflow) */
  resourceType?: number;
  onSuccess?: () => void;
}

export interface UseFolderManagementReturn {
  folders: FolderInfo[];
  loading: boolean;
  createFolder: (name: string, description?: string) => Promise<void>;
  moveResourcesToFolder: (
    folderId: string,
    resourceIds: string[],
    resourceType: number,
  ) => Promise<void>;
  refreshFolders: () => Promise<void>;
}

export const useFolderManagement = ({
  spaceId,
  resourceType,
  onSuccess,
}: UseFolderManagementProps): UseFolderManagementReturn => {
  const [folders, setFolders] = useState<FolderInfo[]>([]);
  const [loading, setLoading] = useState(false);

  const refreshFolders = useCallback(async () => {
    if (!spaceId) {
      return;
    }

    setLoading(true);
    try {
      const response = await folderApi.getFolderList({
        space_id: spaceId,
        ...(resourceType !== undefined ? { resource_type: resourceType } : {}),
      });

      if (response.code === 0) {
        setFolders(response.data || []);
      }
    } catch (error) {
      console.error('Failed to fetch folders:', error);
    } finally {
      setLoading(false);
    }
  }, [spaceId, resourceType]);

  const createFolder = useCallback(
    async (name: string, description = '') => {
      if (!spaceId) {
        return;
      }

      try {
        const response = await folderApi.createFolder({
          space_id: spaceId,
          name,
          description,
        });

        if (response.code === 0) {
          await refreshFolders();
          onSuccess?.();
        }
      } catch (error) {
        console.error('Failed to create folder:', error);
        throw error;
      }
    },
    [spaceId, refreshFolders, onSuccess],
  );

  const moveResourcesToFolder = useCallback(
    async (
      folderId: string,
      resourceIds: string[],
      moveResourceType: number,
    ) => {
      if (!spaceId) {
        return;
      }

      try {
        const response = await folderApi.moveResourcesToFolder({
          space_id: spaceId,
          folder_id: folderId,
          resource_ids: resourceIds,
          resource_type: moveResourceType,
        });

        if (response.code === 0) {
          onSuccess?.();
        }
      } catch (error) {
        console.error('Failed to move resources to folder:', error);
        throw error;
      }
    },
    [spaceId, onSuccess],
  );

  useEffect(() => {
    refreshFolders();
  }, [refreshFolders]);

  return {
    folders,
    loading,
    createFolder,
    moveResourcesToFolder,
    refreshFolders,
  };
};
