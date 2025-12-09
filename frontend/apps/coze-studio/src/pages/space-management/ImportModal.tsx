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

import React, { useRef } from 'react';
import { SpaceInfo, ImportPreviewData } from './types';

interface ImportModalProps {
  isOpen: boolean;
  targetSpace: SpaceInfo | null;
  previewData: ImportPreviewData | null;
  loading: boolean;
  error: string | null;
  onClose: () => void;
  onFileSelect: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onConfirmImport: () => void;
  onResetPreview: () => void;
}

export const ImportModal: React.FC<ImportModalProps> = ({
  isOpen,
  targetSpace,
  previewData,
  loading,
  error,
  onClose,
  onFileSelect,
  onConfirmImport,
  onResetPreview,
}) => {
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isOpen || !targetSpace) return null;

  const handleClose = () => {
    onResetPreview();
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-lg">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold">
            导入到空间: {targetSpace.name}
          </h2>
          <button
            onClick={handleClose}
            className="text-gray-500 hover:text-gray-700 text-xl"
          >
            ✕
          </button>
        </div>

        {/* 错误提示 */}
        {error && (
          <div className="mb-4 p-3 bg-red-100 text-red-800 rounded-md">
            {error}
          </div>
        )}

        {/* 文件选择区域 */}
        {!previewData && (
          <div className="mb-4">
            <p className="text-gray-600 mb-3">
              请选择要导入的空间导出文件（.zip格式）
            </p>
            <input
              type="file"
              ref={fileInputRef}
              accept=".zip"
              onChange={onFileSelect}
              disabled={loading}
              className="w-full border border-gray-300 rounded-md p-2"
            />
            {loading && (
              <p className="mt-2 text-blue-600">正在解析文件...</p>
            )}
          </div>
        )}

        {/* 预览信息 */}
        {previewData && (
          <div className="space-y-4">
            <div className="bg-gray-50 p-4 rounded-md">
              <h3 className="font-semibold mb-2">导入预览</h3>
              <div className="space-y-2 text-sm">
                <p>
                  <span className="text-gray-600">来源空间:</span>{' '}
                  <span className="font-medium">
                    {previewData.manifest.source_space_name}
                  </span>
                </p>
                <p>
                  <span className="text-gray-600">导出时间:</span>{' '}
                  <span className="font-medium">
                    {new Date(previewData.manifest.exported_at).toLocaleString()}
                  </span>
                </p>
                <p>
                  <span className="text-gray-600">版本:</span>{' '}
                  <span className="font-medium">
                    {previewData.manifest.version}
                  </span>
                </p>
              </div>
            </div>

            <div className="bg-blue-50 p-4 rounded-md">
              <h3 className="font-semibold mb-2">将导入的资源</h3>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <p>
                  智能体:{' '}
                  <span className="font-medium">
                    {previewData.manifest.statistics.agents}
                  </span>
                </p>
                <p>
                  插件:{' '}
                  <span className="font-medium">
                    {previewData.manifest.statistics.plugins}
                  </span>
                </p>
                <p>
                  工作流:{' '}
                  <span className="font-medium">
                    {previewData.manifest.statistics.workflows}
                  </span>
                </p>
                <p>
                  变量:{' '}
                  <span className="font-medium">
                    {previewData.manifest.statistics.variables}
                  </span>
                </p>
              </div>
            </div>

            {/* 警告信息 */}
            {previewData.warnings && previewData.warnings.length > 0 && (
              <div className="bg-yellow-50 p-4 rounded-md">
                <h3 className="font-semibold mb-2 text-yellow-800">警告</h3>
                <ul className="list-disc list-inside text-sm text-yellow-700">
                  {previewData.warnings.map((warning, index) => (
                    <li key={index}>{warning}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* 操作按钮 */}
            <div className="flex space-x-3 mt-4">
              <button
                onClick={onConfirmImport}
                disabled={loading}
                className="flex-1 bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? '导入中...' : '确认导入'}
              </button>
              <button
                onClick={onResetPreview}
                disabled={loading}
                className="flex-1 bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400 disabled:opacity-50"
              >
                重新选择
              </button>
            </div>
          </div>
        )}

        {/* 关闭按钮 */}
        {!previewData && !loading && (
          <div className="flex justify-end mt-4">
            <button
              onClick={handleClose}
              className="bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400"
            >
              取消
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default ImportModal;
