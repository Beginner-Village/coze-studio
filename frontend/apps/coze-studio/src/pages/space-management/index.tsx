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

import React, { useState } from 'react';

import { SpaceCard } from './SpaceCard';
import { MemberModal } from './MemberModal';
import { ImportModal } from './ImportModal';
import { useSpaceManagement } from './hooks/useSpaceManagement';
import { DataMaintenanceSection } from './DataMaintenanceSection';
import { CreateSpaceModal } from './CreateSpaceModal';

const SpaceManagementPage: React.FC = () => {
  const [maintenanceSpaceId, setMaintenanceSpaceId] = useState<string>('');
  const {
    spaceList,
    loading,
    currentSpace,
    members,
    showCreateModal,
    showMemberModal,
    showImportModal,
    exportingSpaceId,
    importTargetSpace,
    importPreviewData,
    importLoading,
    importError,
    setShowCreateModal,
    setShowMemberModal,
    createSpace,
    deleteSpace,
    exportSpace,
    openImportModal,
    handleFileSelect,
    confirmImport,
    resetImportPreview,
    closeImportModal,
    openMemberModal,
  } = useSpaceManagement();

  return (
    <div className="p-8 max-w-6xl mx-auto">
      <div className="mb-6">
        <a
          href="/space"
          className="text-blue-500 hover:text-blue-700 underline"
        >
          ← 返回工作空间
        </a>
      </div>

      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-bold">空间管理</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600"
        >
          创建新空间
        </button>
      </div>

      {/* Modals */}
      <CreateSpaceModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSubmit={createSpace}
      />

      <MemberModal
        isOpen={showMemberModal}
        space={currentSpace}
        members={members}
        onClose={() => setShowMemberModal(false)}
      />

      <ImportModal
        isOpen={showImportModal}
        targetSpace={importTargetSpace}
        previewData={importPreviewData}
        loading={importLoading}
        error={importError}
        onClose={closeImportModal}
        onFileSelect={handleFileSelect}
        onConfirmImport={confirmImport}
        onResetPreview={resetImportPreview}
      />

      {/* Space List */}
      <div className="bg-white rounded-lg shadow-md">
        <div className="p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold">空间列表</h2>
        </div>

        {loading ? (
          <div className="p-6 text-center">加载中...</div>
        ) : spaceList.length === 0 ? (
          <div className="p-6 text-center text-gray-500">暂无空间</div>
        ) : (
          <div className="p-6">
            <div className="grid grid-cols-1 gap-4">
              {spaceList.map(space => (
                <SpaceCard
                  key={space.space_id}
                  space={space}
                  isExporting={exportingSpaceId === space.space_id}
                  onExport={() => exportSpace(space)}
                  onImport={() => openImportModal(space)}
                  onManageMembers={() => openMemberModal(space)}
                  onDelete={() => deleteSpace(space.space_id)}
                />
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Data Maintenance */}
      {spaceList.length > 0 && (
        <div className="bg-white rounded-lg shadow-md mt-8">
          <div className="p-6 border-b border-gray-200">
            <h2 className="text-lg font-semibold">数据维护</h2>
            <p className="text-sm text-gray-500 mt-1">
              对指定空间执行 ES 重新同步等维护操作
            </p>
          </div>
          <div className="p-6">
            <div className="mb-4">
              <label className="text-sm text-gray-700 mr-2">选择空间:</label>
              <select
                value={maintenanceSpaceId}
                onChange={e => setMaintenanceSpaceId(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1 text-sm"
              >
                <option value="">-- 请选择 --</option>
                {spaceList.map(sp => (
                  <option key={sp.space_id} value={String(sp.space_id)}>
                    {sp.name} (ID: {sp.space_id})
                  </option>
                ))}
              </select>
            </div>
            {maintenanceSpaceId ? (
              <DataMaintenanceSection spaceId={maintenanceSpaceId} />
            ) : null}
          </div>
        </div>
      )}
    </div>
  );
};

export default SpaceManagementPage;
