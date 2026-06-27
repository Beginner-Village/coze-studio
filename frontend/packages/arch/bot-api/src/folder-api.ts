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

import { axiosInstance, type BotAPIRequestConfig } from './axios';

// Folder 资源管理接口（非 IDL 生成，走 /api/plugin_api/*）

export interface FolderInfo {
  id: string;
  space_id: string;
  parent_id?: string;
  name: string;
  description: string;
  creator_id: string;
  created_at: number;
  updated_at: number;
  resource_ids?: Array<string | number>;
  resource_count?: number;
}

export interface GetFolderListRequest {
  space_id: string; // 使用字符串避免 JS bigint 精度丢失
  resource_type?: number;
}

export interface GetFolderListResponse {
  code: number;
  msg?: string;
  data: FolderInfo[];
}

export interface CreateFolderRequest {
  space_id: string;
  parent_id?: string;
  name: string;
  description?: string;
}

export interface CreateFolderResponse {
  code: number;
  msg?: string;
}

export interface MoveResourcesToFolderRequest {
  space_id: string;
  folder_id: string; // '0' 表示移出文件夹
  resource_ids: string[];
  resource_type: number;
}

export interface MoveResourcesToFolderResponse {
  code: number;
  msg?: string;
}

export interface UpdateFolderRequest {
  space_id: string;
  folder_id: string;
  name: string;
  description?: string;
}

export interface UpdateFolderResponse {
  code: number;
  msg?: string;
}

export interface DeleteFolderRequest {
  space_id: string;
  folder_id: string;
}

export interface DeleteFolderResponse {
  code: number;
  msg?: string;
}

class FolderApiService {
  /**
   * List folders for the given space, optionally scoped by resource type.
   *
   * Each returned FolderInfo carries `resource_ids` / `resource_count`
   * computed server-side for the requested resource type.
   */
  async getFolderList(
    data: GetFolderListRequest,
    config?: BotAPIRequestConfig,
  ): Promise<GetFolderListResponse> {
    return await axiosInstance.post('/api/plugin_api/get_folder_list', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  /**
   * Create a folder under the given space.
   */
  async createFolder(
    data: CreateFolderRequest,
    config?: BotAPIRequestConfig,
  ): Promise<CreateFolderResponse> {
    return await axiosInstance.post('/api/plugin_api/create_folder', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  /**
   * Move resources into a folder. `folder_id === '0'` moves them out.
   */
  async moveResourcesToFolder(
    data: MoveResourcesToFolderRequest,
    config?: BotAPIRequestConfig,
  ): Promise<MoveResourcesToFolderResponse> {
    return await axiosInstance.post(
      '/api/plugin_api/move_resources_to_folder',
      data,
      {
        headers: { 'Agw-Js-Conv': 'str' },
        ...config,
      },
    );
  }

  /**
   * Rename a folder / update its description.
   */
  async updateFolder(
    data: UpdateFolderRequest,
    config?: BotAPIRequestConfig,
  ): Promise<UpdateFolderResponse> {
    return await axiosInstance.post('/api/plugin_api/update_folder', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }

  /**
   * Delete a folder. Resources inside are moved back to the top level.
   */
  async deleteFolder(
    data: DeleteFolderRequest,
    config?: BotAPIRequestConfig,
  ): Promise<DeleteFolderResponse> {
    return await axiosInstance.post('/api/plugin_api/delete_folder', data, {
      headers: { 'Agw-Js-Conv': 'str' },
      ...config,
    });
  }
}

export const folderApi = new FolderApiService();
