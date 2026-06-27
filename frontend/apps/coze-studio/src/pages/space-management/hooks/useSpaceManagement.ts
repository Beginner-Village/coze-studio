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
import { space_management, space_export_import } from '@coze-studio/api-schema';
import { getLocalizedErrorMessage } from '@coze-arch/bot-api';

import type {
  SpaceInfo,
  SpaceMemberInfo,
  SpaceType,
  ImportPreviewData,
} from '../types';

const getDisplayErrorMessage = (message?: string) =>
  getLocalizedErrorMessage(message) || message || '未知错误';

export function useSpaceManagement() {
  const [spaceList, setSpaceList] = useState<SpaceInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [currentSpace, setCurrentSpace] = useState<SpaceInfo | null>(null);
  const [members, setMembers] = useState<SpaceMemberInfo[]>([]);

  // Modal states
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showMemberModal, setShowMemberModal] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);

  // Export/Import states
  const [exportingSpaceId, setExportingSpaceId] = useState<number | null>(null);
  const [importTargetSpace, setImportTargetSpace] = useState<SpaceInfo | null>(null);
  const [importPreviewData, setImportPreviewData] = useState<ImportPreviewData | null>(null);
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);

  // Fetch space list
  const fetchSpaceList = useCallback(async () => {
    try {
      setLoading(true);
      const response = await space_management.GetSpaceList({
        page: 1,
        page_size: 20,
      });

      if (response.code === 200) {
        setSpaceList(response.data || []);
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200) {
        const responseData = error.response?.data;
        if (responseData && responseData.data) {
          setSpaceList(responseData.data);
        }
      }
    } finally {
      setLoading(false);
    }
  }, []);

  // Create space
  const createSpace = useCallback(async (data: { name: string; description: string; spaceType: SpaceType }) => {
    try {
      const response = await space_management.CreateSpace({
        name: data.name,
        description: data.description || undefined,
        space_type: data.spaceType,
      });

      if (response.code === 200) {
        setShowCreateModal(false);
        await fetchSpaceList();
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200) {
        setShowCreateModal(false);
        await fetchSpaceList();
      }
    }
  }, [fetchSpaceList]);

  // Fetch space members
  const fetchSpaceMembers = useCallback(async (spaceId: number) => {
    try {
      const response = await space_management.GetSpaceMembers({
        space_id: spaceId,
        page: 1,
        page_size: 100,
      });

      if (response.code === 200) {
        setMembers(response.data || []);
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200) {
        const responseData = error.response?.data;
        if (responseData && responseData.data) {
          setMembers(responseData.data);
        }
      }
    }
  }, []);

  // Delete space
  const deleteSpace = useCallback(async (spaceId: number) => {
    if (!confirm('确定要删除这个空间吗？此操作不可恢复。')) return;

    try {
      const response = await space_management.DeleteSpace({
        space_id: spaceId,
      });

      if (response.code === 200) {
        await fetchSpaceList();
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200) {
        await fetchSpaceList();
      }
    }
  }, [fetchSpaceList]);

  // Export space
  const exportSpace = useCallback(async (space: SpaceInfo) => {
    try {
      setExportingSpaceId(space.space_id);
      const response = await space_export_import.ExportSpace({
        space_id: String(space.space_id),
      });

      if (response.code === 0 || response.code === 200) {
        const downloadUrl = response.data.download_url;
        const link = document.createElement('a');
        link.href = downloadUrl;
        link.download = response.data.file_name || `space_${space.space_id}_export.zip`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        alert(`导出成功！\n\n统计信息：\n- 智能体: ${response.data.statistics.agents}\n- 插件: ${response.data.statistics.plugins}\n- 工作流: ${response.data.statistics.workflows}\n- 变量: ${response.data.statistics.variables}`);
      } else {
        alert(`导出失败: ${getDisplayErrorMessage(response.msg)}`);
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200 || error.code === '0' || error.code === 0) {
        const responseData = error.response?.data || error.data || error;
        if (responseData && responseData.data && responseData.data.download_url) {
          const downloadUrl = responseData.data.download_url;
          const link = document.createElement('a');
          link.href = downloadUrl;
          link.download = responseData.data.file_name || `space_${space.space_id}_export.zip`;
          document.body.appendChild(link);
          link.click();
          document.body.removeChild(link);
          alert('导出成功！');
          return;
        }
      }
      alert(`导出失败: ${getDisplayErrorMessage(error.message)}`);
    } finally {
      setExportingSpaceId(null);
    }
  }, []);

  // Open import modal
  const openImportModal = useCallback((space: SpaceInfo) => {
    setImportTargetSpace(space);
    setImportPreviewData(null);
    setImportError(null);
    setShowImportModal(true);
  }, []);

  // Handle file select for import
  const handleFileSelect = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file || !importTargetSpace) return;

    if (!file.name.endsWith('.zip')) {
      setImportError('请选择 .zip 格式的导出文件');
      return;
    }

    try {
      setImportLoading(true);
      setImportError(null);

      const fileContent = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          const base64 = result.split(',')[1] || result;
          resolve(base64);
        };
        reader.onerror = reject;
        reader.readAsDataURL(file);
      });

      const response = await space_export_import.ImportPreview({
        space_id: String(importTargetSpace.space_id),
        file_content: fileContent,
      });

      if (response.code === 0 || response.code === 200) {
        setImportPreviewData(response.data as ImportPreviewData);
      } else {
        setImportError(`预览失败: ${getDisplayErrorMessage(response.msg)}`);
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200 || error.code === '0' || error.code === 0) {
        const responseData = error.response?.data || error.data || error;
        if (responseData && responseData.data) {
          setImportPreviewData(responseData.data as ImportPreviewData);
          return;
        }
      }
      setImportError(`预览失败: ${getDisplayErrorMessage(error.message)}`);
    } finally {
      setImportLoading(false);
    }
  }, [importTargetSpace]);

  // Confirm import
  const confirmImport = useCallback(async () => {
    if (!importTargetSpace || !importPreviewData) return;

    try {
      setImportLoading(true);
      setImportError(null);

      const response = await space_export_import.ImportConfirm({
        space_id: String(importTargetSpace.space_id),
        import_token: importPreviewData.import_token,
      });

      if (response.code === 0 || response.code === 200) {
        const stats = response.data.statistics;
        alert(`导入成功！\n\n创建资源：\n- 智能体: ${stats.agents_created}\n- 插件: ${stats.plugins_created}\n- 工作流: ${stats.workflows_created}\n- 变量: ${stats.variables_created}`);
        setShowImportModal(false);
        setImportPreviewData(null);
      } else {
        setImportError(`导入失败: ${getDisplayErrorMessage(response.msg)}`);
      }
    } catch (error: any) {
      if (error.code === '200' || error.code === 200 || error.code === '0' || error.code === 0) {
        const responseData = error.response?.data || error.data || error;
        if (responseData && responseData.data) {
          const stats = responseData.data.statistics;
          alert(`导入成功！\n\n创建资源：\n- 智能体: ${stats.agents_created}\n- 插件: ${stats.plugins_created}\n- 工作流: ${stats.workflows_created}\n- 变量: ${stats.variables_created}`);
          setShowImportModal(false);
          setImportPreviewData(null);
          return;
        }
      }
      setImportError(`导入失败: ${getDisplayErrorMessage(error.message)}`);
    } finally {
      setImportLoading(false);
    }
  }, [importTargetSpace, importPreviewData]);

  // Reset import preview
  const resetImportPreview = useCallback(() => {
    setImportPreviewData(null);
    setImportError(null);
  }, []);

  // Close import modal
  const closeImportModal = useCallback(() => {
    setShowImportModal(false);
    setImportPreviewData(null);
    setImportError(null);
  }, []);

  // Open member modal
  const openMemberModal = useCallback((space: SpaceInfo) => {
    setCurrentSpace(space);
    fetchSpaceMembers(space.space_id);
    setShowMemberModal(true);
  }, [fetchSpaceMembers]);

  // Initialize
  useEffect(() => {
    fetchSpaceList();
  }, [fetchSpaceList]);

  return {
    // State
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

    // Actions
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
  };
}
