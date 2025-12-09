/*
 * Copyright 2025 coze-dev Authors
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

import React, { useState } from 'react';
import { SpaceType } from './types';

interface CreateSpaceModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: { name: string; description: string; spaceType: SpaceType }) => Promise<void>;
}

export const CreateSpaceModal: React.FC<CreateSpaceModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
}) => {
  const [newSpaceName, setNewSpaceName] = useState('');
  const [newSpaceDescription, setNewSpaceDescription] = useState('');
  const [newSpaceType, setNewSpaceType] = useState<SpaceType>(SpaceType.Personal);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = async () => {
    if (!newSpaceName.trim() || isSubmitting) return;

    setIsSubmitting(true);
    try {
      await onSubmit({
        name: newSpaceName,
        description: newSpaceDescription,
        spaceType: newSpaceType,
      });
      // 重置表单
      setNewSpaceName('');
      setNewSpaceDescription('');
      setNewSpaceType(SpaceType.Personal);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    setNewSpaceName('');
    setNewSpaceDescription('');
    setNewSpaceType(SpaceType.Personal);
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-lg font-semibold mb-4">创建新空间</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">
              空间名称
            </label>
            <input
              type="text"
              value={newSpaceName}
              onChange={e => setNewSpaceName(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="请输入空间名称"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">
              描述（可选）
            </label>
            <textarea
              value={newSpaceDescription}
              onChange={e => setNewSpaceDescription(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 h-20 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="请输入空间描述"
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">
              空间类型
            </label>
            <select
              value={newSpaceType}
              onChange={e => setNewSpaceType(Number(e.target.value) as SpaceType)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value={SpaceType.Personal}>个人空间</option>
              <option value={SpaceType.Team}>团队空间</option>
            </select>
          </div>
        </div>
        <div className="flex space-x-3 mt-6">
          <button
            onClick={handleSubmit}
            disabled={!newSpaceName.trim() || isSubmitting}
            className="flex-1 bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed"
          >
            {isSubmitting ? '创建中...' : '创建'}
          </button>
          <button
            onClick={handleClose}
            disabled={isSubmitting}
            className="flex-1 bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400 disabled:opacity-50"
          >
            取消
          </button>
        </div>
      </div>
    </div>
  );
};

export default CreateSpaceModal;
