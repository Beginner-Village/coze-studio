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
import { SpaceInfo, SpaceMemberInfo, getRoleText } from './types';

interface MemberModalProps {
  isOpen: boolean;
  space: SpaceInfo | null;
  members: SpaceMemberInfo[];
  onClose: () => void;
}

export const MemberModal: React.FC<MemberModalProps> = ({
  isOpen,
  space,
  members,
  onClose,
}) => {
  if (!isOpen || !space) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-4xl max-h-[80vh] overflow-y-auto">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold">
            {space.name} - 成员管理
          </h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700"
          >
            ✕
          </button>
        </div>

        <div className="space-y-3">
          {members.length === 0 ? (
            <div className="text-center text-gray-500 py-8">暂无成员</div>
          ) : (
            members.map(member => (
              <div
                key={member.user_id}
                className="flex items-center justify-between p-3 border border-gray-200 rounded-md"
              >
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-gray-300 rounded-full flex items-center justify-center">
                    {member.avatar_url ? (
                      <img
                        src={member.avatar_url}
                        alt=""
                        className="w-10 h-10 rounded-full"
                      />
                    ) : (
                      <span className="text-gray-600 font-semibold">
                        {member.username.charAt(0).toUpperCase()}
                      </span>
                    )}
                  </div>
                  <div>
                    <div className="font-semibold">
                      {member.nickname || member.username}
                    </div>
                    <div className="text-sm text-gray-500">
                      @{member.username}
                    </div>
                  </div>
                </div>
                <div className="flex items-center space-x-3">
                  <span className="px-2 py-1 bg-blue-100 text-blue-800 text-sm rounded">
                    {getRoleText(member.role)}
                  </span>
                  <span className="text-sm text-gray-500">
                    加入于 {new Date(member.joined_at * 1000).toLocaleDateString()}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};

export default MemberModal;
