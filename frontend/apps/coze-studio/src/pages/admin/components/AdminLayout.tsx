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

import React, { useEffect, useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Spin, Tabs } from '@coze-arch/coze-design';
import { admin } from '@coze-studio/api-schema';

import { InitWizard } from './InitWizard';

export const AdminLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const [_initialized, setInitialized] = useState(false);
  const [isAdmin, setIsAdmin] = useState(false);
  const [showInitWizard, setShowInitWizard] = useState(false);

  useEffect(() => {
    checkAdminInit();
  }, []);

  const checkAdminInit = async () => {
    try {
      const response = await admin.CheckAdminInit({});
      if (response.code === 0) {
        setInitialized(response.initialized);
        setIsAdmin(response.is_admin || false);

        if (!response.initialized) {
          setShowInitWizard(true);
        } else if (!response.is_admin) {
          // 非管理员，跳转到首页
          navigate('/');
          return;
        }
      }
    } catch (error: any) {
      // 处理API客户端特殊情况
      if (error.code === '0' || error.code === 0) {
        const responseData = error.response?.data || error;
        setInitialized(responseData.initialized);
        setIsAdmin(responseData.is_admin || false);

        if (!responseData.initialized) {
          setShowInitWizard(true);
        } else if (!responseData.is_admin) {
          navigate('/');
          return;
        }
      } else {
        console.error('检查管理员初始化状态失败:', error);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleInitSuccess = () => {
    setShowInitWizard(false);
    setInitialized(true);
    setIsAdmin(true);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Spin size="large" />
      </div>
    );
  }

  if (showInitWizard) {
    return <InitWizard onSuccess={handleInitSuccess} />;
  }

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <h2 className="text-xl font-semibold mb-2">无权限访问</h2>
          <p className="text-gray-500">您没有管理员权限，无法访问此页面</p>
        </div>
      </div>
    );
  }

  const activeKey = location.pathname.includes('/users') ? 'users' : 'models';

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <h1 className="text-xl font-semibold">管理后台</h1>
        </div>
        <div className="max-w-7xl mx-auto px-6">
          <Tabs
            activeKey={activeKey}
            onChange={(key) => navigate(`/admin/${key}`)}
          >
            <Tabs.TabPane tab="公共模型" itemKey="models" />
            <Tabs.TabPane tab="管理员" itemKey="users" />
          </Tabs>
        </div>
      </div>
      <div className="max-w-7xl mx-auto px-6 py-6">
        <Outlet />
      </div>
    </div>
  );
};

export { AdminLayout as Component };
export default AdminLayout;
