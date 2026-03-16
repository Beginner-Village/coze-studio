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

import React, { useState, useRef } from 'react';
import { useParams } from 'react-router-dom';
import { Layout, Button } from '@coze-arch/coze-design';

const SpaceExportImportPage: React.FC = () => {
  const { space_id } = useParams<{ space_id: string }>();

  // 导出状态
  const [exporting, setExporting] = useState(false);
  const [exportResult, setExportResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  // 导入状态
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);
  const [importResult, setImportResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // 导出空间 - 使用 fetch 直接调用
  const handleExport = async () => {
    if (!space_id) return;

    try {
      setExporting(true);
      setExportResult(null);

      const response = await fetch(`/api/space/${space_id}/export`, {
        method: 'POST',
        headers: {
          Accept: 'application/json, text/plain, */*',
          'Content-Type': 'application/json',
          'Agw-Js-Conv': 'str',
          'x-requested-with': 'XMLHttpRequest',
        },
        body: JSON.stringify({}),
      });

      const data = await response.json();

      if (data.code === 0 || data.code === 200) {
        // 使用返回的下载URL下载文件
        const downloadUrl = data.data.download_url;
        const link = document.createElement('a');
        link.href = downloadUrl;
        link.download = data.data.file_name || `space_${space_id}_export.zip`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        setExportResult({
          success: true,
          message: '导出成功！文件已开始下载。',
        });
      } else {
        setExportResult({
          success: false,
          message: `导出失败: ${data.msg || '未知错误'}`,
        });
      }
    } catch (error: any) {
      console.error('Failed to export space:', error);
      setExportResult({
        success: false,
        message: `导出失败: ${error.message || '未知错误'}`,
      });
    } finally {
      setExporting(false);
    }
  };

  // 处理文件选择
  const handleFileSelect = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    if (!file || !space_id) return;

    // 验证文件类型
    if (!file.name.endsWith('.zip')) {
      setImportError('请选择 .zip 格式的导出文件');
      return;
    }

    try {
      setImportLoading(true);
      setImportError(null);
      setImportResult(null);

      // 读取文件内容并转换为base64
      const fileContent = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          // 移除 data:application/zip;base64, 前缀
          const base64 = result.split(',')[1] || result;
          resolve(base64);
        };
        reader.onerror = reject;
        reader.readAsDataURL(file);
      });

      // 调用预览API
      const previewResponse = await fetch(
        `/api/space/${space_id}/import/preview`,
        {
          method: 'POST',
          headers: {
            Accept: 'application/json, text/plain, */*',
            'Content-Type': 'application/json',
            'Agw-Js-Conv': 'str',
            'x-requested-with': 'XMLHttpRequest',
          },
          body: JSON.stringify({
            file_content: fileContent,
          }),
        },
      );

      const previewData = await previewResponse.json();

      if (previewData.code === 0 || previewData.code === 200) {
        // 确认导入
        const confirmResponse = await fetch(
          `/api/space/${space_id}/import/confirm`,
          {
            method: 'POST',
            headers: {
              Accept: 'application/json, text/plain, */*',
              'Content-Type': 'application/json',
              'Agw-Js-Conv': 'str',
              'x-requested-with': 'XMLHttpRequest',
            },
            body: JSON.stringify({
              import_token: previewData.data.import_token,
            }),
          },
        );

        const confirmData = await confirmResponse.json();

        if (confirmData.code === 0 || confirmData.code === 200) {
          setImportResult({
            success: true,
            message: `导入成功！创建了 ${confirmData.data.statistics.agents_created} 个智能体`,
          });
        } else {
          setImportError(`导入失败: ${confirmData.msg || '未知错误'}`);
        }
      } else {
        setImportError(`预览失败: ${previewData.msg || '未知错误'}`);
      }
    } catch (error: any) {
      console.error('Failed to import:', error);
      setImportError(`导入失败: ${error.message || '未知错误'}`);
    } finally {
      setImportLoading(false);
      // 清空文件输入
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  return (
    <Layout title="猎鹰">
      <Layout.Header className="pb-0">
        <div className="w-full">
          <div className="flex items-center justify-between mb-[16px]">
            <div className="font-[500] text-[20px]">空间导出/导入</div>
          </div>
          <p className="text-gray-500 text-sm">当前空间 ID: {space_id}</p>
        </div>
      </Layout.Header>

      <Layout.Content>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* 导出区域 */}
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold mb-4">导出空间</h2>
            <p className="text-gray-600 text-sm mb-4">
              将当前空间的所有资源（智能体、插件、工作流、变量）打包导出为 ZIP
              文件。
            </p>

            <Button
              theme="solid"
              type="primary"
              onClick={handleExport}
              loading={exporting}
              disabled={exporting}
              block
            >
              {exporting ? '正在导出...' : '导出空间'}
            </Button>

            {/* 导出结果 */}
            {exportResult && (
              <div
                className={`mt-4 p-4 rounded-md ${
                  exportResult.success
                    ? 'bg-green-50 text-green-800'
                    : 'bg-red-50 text-red-800'
                }`}
              >
                <p className="font-medium">{exportResult.message}</p>
              </div>
            )}
          </div>

          {/* 导入区域 */}
          <div className="bg-white rounded-lg border border-gray-200 p-6">
            <h2 className="text-lg font-semibold mb-4">导入到空间</h2>
            <p className="text-gray-600 text-sm mb-4">
              选择之前导出的 ZIP 文件，将资源导入到当前空间。
            </p>

            {/* 错误提示 */}
            {importError && (
              <div className="mb-4 p-3 bg-red-100 text-red-800 rounded-md text-sm">
                {importError}
              </div>
            )}

            {/* 导入成功结果 */}
            {importResult?.success && (
              <div className="mb-4 p-4 bg-green-50 text-green-800 rounded-md">
                <p className="font-medium">{importResult.message}</p>
              </div>
            )}

            {/* 文件选择区域 */}
            <div>
              <input
                type="file"
                ref={fileInputRef}
                accept=".zip"
                onChange={handleFileSelect}
                disabled={importLoading}
                className="hidden"
                id="import-file-input"
              />
              <label
                htmlFor="import-file-input"
                className={`block w-full border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer hover:border-blue-400 hover:bg-blue-50 transition-colors ${
                  importLoading ? 'opacity-50 cursor-not-allowed' : ''
                }`}
              >
                {importLoading ? (
                  <span className="text-blue-600">正在导入...</span>
                ) : (
                  <p className="text-sm text-gray-600">
                    点击选择 ZIP 文件进行导入
                  </p>
                )}
              </label>
            </div>
          </div>
        </div>

        {/* 说明信息 */}
        <div className="mt-8 bg-blue-50 rounded-lg p-6">
          <h3 className="font-semibold text-blue-800 mb-3">使用说明</h3>
          <ul className="text-sm text-blue-700 space-y-2">
            <li>• 导出功能会打包当前空间的所有智能体、插件、工作流和变量</li>
            <li>• 导入时会自动处理 ID 映射，确保导入的资源不会与现有资源冲突</li>
            <li>• 导出的文件有效期为 1 小时</li>
          </ul>
        </div>
      </Layout.Content>
    </Layout>
  );
};

// React Router lazy loading requires a named 'Component' export
export const Component = SpaceExportImportPage;
export default SpaceExportImportPage;
