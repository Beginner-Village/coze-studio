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
import {
  SpaceInfo,
  SpaceStatus,
  MemberRoleType,
  getSpaceTypeText,
  getSpaceStatusText,
  getRoleText,
} from './types';

interface SpaceCardProps {
  space: SpaceInfo;
  isExporting: boolean;
  onExport: () => void;
  onImport: () => void;
  onManageMembers: () => void;
  onDelete: () => void;
}

export const SpaceCard: React.FC<SpaceCardProps> = ({
  space,
  isExporting,
  onExport,
  onImport,
  onManageMembers,
  onDelete,
}) => {
  return (
    <div className="border border-gray-200 rounded-md p-4">
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <div className="flex items-center space-x-3 mb-2">
            <h3 className="font-semibold text-lg">{space.name}</h3>
            <span className="px-2 py-1 bg-gray-100 text-gray-800 text-xs rounded">
              {getSpaceTypeText(space.space_type)}
            </span>
            <span
              className={`px-2 py-1 text-xs rounded ${
                space.status === SpaceStatus.Active
                  ? 'bg-green-100 text-green-800'
                  : space.status === SpaceStatus.Inactive
                    ? 'bg-yellow-100 text-yellow-800'
                    : 'bg-gray-100 text-gray-800'
              }`}
            >
              {getSpaceStatusText(space.status)}
            </span>
            {space.current_user_role && (
              <span className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
                {getRoleText(space.current_user_role)}
              </span>
            )}
          </div>

          {space.description && (
            <p className="text-gray-600 mb-2">{space.description}</p>
          )}

          <div className="flex items-center space-x-4 text-sm text-gray-500">
            <span>ID: {space.space_id}</span>
            <span>
              创建时间: {new Date(space.created_at * 1000).toLocaleDateString()}
            </span>
            {space.member_count && <span>成员数: {space.member_count}</span>}
          </div>
        </div>

        <div className="flex space-x-2 ml-4">
          <button
            onClick={onExport}
            disabled={isExporting}
            className="px-3 py-1 text-sm bg-green-100 text-green-800 rounded hover:bg-green-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isExporting ? '导出中...' : '导出'}
          </button>
          <button
            onClick={onImport}
            className="px-3 py-1 text-sm bg-purple-100 text-purple-800 rounded hover:bg-purple-200"
          >
            导入
          </button>
          <button
            onClick={onManageMembers}
            className="px-3 py-1 text-sm bg-blue-100 text-blue-800 rounded hover:bg-blue-200"
          >
            管理成员
          </button>
          <button
            onClick={onDelete}
            className="px-3 py-1 text-sm bg-red-100 text-red-800 rounded hover:bg-red-200"
            disabled={space.current_user_role !== MemberRoleType.Owner}
          >
            删除
          </button>
        </div>
      </div>
    </div>
  );
};

export default SpaceCard;
