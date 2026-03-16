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

import { lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Spin } from '@coze-arch/coze-design';

import { AdminLayout } from './components/AdminLayout';

// 懒加载子页面
const AdminModelsPage = lazy(() => import('./models'));
const AdminModelCreatePage = lazy(() => import('./models/create'));
const AdminModelEditPage = lazy(() => import('./models/edit'));
const AdminUsersPage = lazy(() => import('./users'));

const Loading = () => (
  <div className="flex items-center justify-center h-screen">
    <Spin size="large" />
  </div>
);

// Admin 路由组件
const AdminRouter = () => {
  return (
    <Suspense fallback={<Loading />}>
      <Routes>
        <Route element={<AdminLayout />}>
          <Route index element={<Navigate to="models" replace />} />
          <Route path="models" element={<AdminModelsPage />} />
          <Route path="models/create" element={<AdminModelCreatePage />} />
          <Route path="models/:model_id/edit" element={<AdminModelEditPage />} />
          <Route path="users" element={<AdminUsersPage />} />
        </Route>
      </Routes>
    </Suspense>
  );
};

export default AdminRouter;

// 重新导出组件
export { AdminLayout } from './components/AdminLayout';
export { InitWizard } from './components/InitWizard';
