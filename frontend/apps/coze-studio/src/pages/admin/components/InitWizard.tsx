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
import { Button, Toast } from '@coze-arch/coze-design';
import { admin } from '@coze-studio/api-schema';

interface InitWizardProps {
  onSuccess: () => void;
}

/**
 * Admin 初始化向导
 * 注意：此组件独立于全局布局，不依赖 useUserInfo 等全局 hooks
 * 后端会自动从session获取当前登录用户的ID
 */
export const InitWizard: React.FC<InitWizardProps> = ({ onSuccess }) => {
  const [loading, setLoading] = useState(false);

  const handleInit = async () => {
    setLoading(true);
    try {
      // 不传 user_id，后端会自动使用当前登录用户
      const response = await admin.InitAdmin({});
      if (response.code === 0) {
        Toast.success('初始化成功！您已成为超级管理员');
        onSuccess();
      } else {
        console.error('初始化失败:', response.msg);
        Toast.error(response.msg || '初始化失败');
      }
    } catch (error: any) {
      // 处理API客户端特殊情况
      if (error.code === '0' || error.code === 0) {
        Toast.success('初始化成功！您已成为超级管理员');
        onSuccess();
      } else {
        console.error('初始化失败:', error);
        Toast.error('初始化失败，请确保您已登录');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center h-screen bg-gray-50">
      <div className="bg-white rounded-lg shadow-lg p-8 max-w-md w-full">
        <h2 className="text-2xl font-semibold mb-2 text-center">初始化管理后台</h2>
        <p className="text-gray-500 text-center mb-6">
          首次访问需要设置超级管理员
        </p>

        <div className="mb-6 p-4 bg-blue-50 rounded-lg">
          <p className="text-sm text-blue-700 text-center">
            点击下方按钮，将当前登录账户设置为超级管理员
          </p>
        </div>

        <Button
          type="primary"
          block
          loading={loading}
          onClick={handleInit}
        >
          确认初始化（使用当前账户）
        </Button>

        <p className="text-xs text-gray-400 text-center mt-4">
          超级管理员可以管理公共模型和其他管理员
        </p>
      </div>
    </div>
  );
};

export default InitWizard;
