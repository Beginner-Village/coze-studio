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

import React from 'react';
import { useRouteError, useNavigate } from 'react-router-dom';

/**
 * Admin 专用错误页面组件
 * 不依赖任何需要认证的全局 store
 */
export const AdminError: React.FC = () => {
  const error = useRouteError() as Error | null;
  const navigate = useNavigate();

  console.error('Admin Error:', error);

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center">
      <div className="text-center p-8 bg-white rounded-lg shadow-lg max-w-md">
        <h1 className="text-2xl font-bold text-red-600 mb-4">页面加载错误</h1>
        <p className="text-gray-600 mb-4">
          管理后台页面加载时发生错误，请尝试以下操作：
        </p>
        <ul className="text-left text-gray-500 text-sm mb-6 space-y-2">
          <li>• 刷新页面重试</li>
          <li>• 清除浏览器缓存和 Cookie</li>
          <li>• 检查网络连接</li>
        </ul>
        {error && (
          <details className="text-left mb-6">
            <summary className="cursor-pointer text-gray-500 text-sm">
              错误详情
            </summary>
            <pre className="mt-2 p-2 bg-gray-100 rounded text-xs overflow-auto max-h-40">
              {error.message || String(error)}
            </pre>
          </details>
        )}
        <div className="flex gap-4 justify-center">
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            刷新页面
          </button>
          <button
            onClick={() => navigate('/')}
            className="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300"
          >
            返回首页
          </button>
        </div>
      </div>
    </div>
  );
};

export default AdminError;
