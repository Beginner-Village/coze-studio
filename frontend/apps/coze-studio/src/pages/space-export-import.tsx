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

import {
  useRef,
  useState,
  type ChangeEvent,
  type FC,
  type RefObject,
} from 'react';
import { useParams } from 'react-router-dom';
import { Button } from '@coze-arch/coze-design';
import {
  IconCozDownload,
  IconCozInfoCircle,
  IconCozUpload,
} from '@coze-arch/coze-design/icons';
import { getLocalizedErrorMessage } from '@coze-arch/bot-api';

const getDisplayErrorMessage = (message?: string) =>
  getLocalizedErrorMessage(message) || message || '未知错误';

const getUnknownErrorMessage = (error: unknown) =>
  error instanceof Error ? getDisplayErrorMessage(error.message) : '未知错误';

const requestHeaders = {
  Accept: 'application/json, text/plain, */*',
  'Content-Type': 'application/json',
  'Agw-Js-Conv': 'str',
  'x-requested-with': 'XMLHttpRequest',
};

const importLabelClassName =
  'w-full h-[44px] rounded-[10px] cursor-pointer border-[1.5px] border-dashed border-[var(--coz-stroke-plus)] bg-[var(--coz-bg-secondary)] coz-fg-secondary flex items-center justify-center gap-[8px] text-[14px] transition-colors hover:border-[rgb(53,138,255)] hover:text-[rgb(53,138,255)] hover:bg-[rgba(53,138,255,.05)]';

interface ResultMessage {
  success: boolean;
  message: string;
}

const isSuccessCode = (code?: number) => code === 0 || code === 200;

const readFileAsBase64 = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      const base64 = result.split(',')[1] || result;
      resolve(base64);
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });

const downloadByUrl = (url: string, filename: string) => {
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
};

const ExportCard: FC<{ spaceId: string }> = ({ spaceId }) => {
  const [exporting, setExporting] = useState(false);
  const [exportResult, setExportResult] = useState<ResultMessage | null>(null);

  const handleExport = async () => {
    try {
      setExporting(true);
      setExportResult(null);

      const response = await fetch(`/api/space/${spaceId}/export`, {
        method: 'POST',
        headers: requestHeaders,
        body: JSON.stringify({}),
      });

      const data = await response.json();
      if (isSuccessCode(data.code)) {
        downloadByUrl(
          data.data.download_url,
          data.data.file_name || `space_${spaceId}_export.zip`,
        );
        setExportResult({
          success: true,
          message: '导出成功！文件已开始下载。',
        });
      } else {
        setExportResult({
          success: false,
          message: `导出失败: ${getDisplayErrorMessage(data.msg)}`,
        });
      }
    } catch (error: unknown) {
      console.error('Failed to export space:', error);
      setExportResult({
        success: false,
        message: `导出失败: ${getUnknownErrorMessage(error)}`,
      });
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className="bg-[var(--coz-bg-max)] rounded-[14px] border border-solid coz-stroke-primary p-[24px]">
      <h2 className="m-0 text-[16px] leading-[24px] font-[600] coz-fg-plus">
        导出空间
      </h2>
      <p className="mt-[10px] mb-[20px] text-[13px] leading-[20px] coz-fg-secondary">
        将当前空间的所有资源（智能体、插件、工作流、变量）打包导出为 ZIP
        文件。
      </p>
      <Button
        theme="solid"
        type="primary"
        className="!h-[44px] !rounded-[10px]"
        icon={<IconCozDownload />}
        onClick={handleExport}
        loading={exporting}
        disabled={exporting}
        block
      >
        {exporting ? '正在导出...' : '导出空间'}
      </Button>
      {exportResult ? <ResultAlert result={exportResult} /> : null}
    </div>
  );
};

const ImportCard: FC<{
  spaceId: string;
  fileInputRef: RefObject<HTMLInputElement>;
}> = ({ spaceId, fileInputRef }) => {
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);
  const [importResult, setImportResult] = useState<ResultMessage | null>(null);

  const handleFileSelect = async (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }

    if (!file.name.endsWith('.zip')) {
      setImportError('请选择 .zip 格式的导出文件');
      return;
    }

    try {
      setImportLoading(true);
      setImportError(null);
      setImportResult(null);

      const fileContent = await readFileAsBase64(file);
      const previewResponse = await fetch(
        `/api/space/${spaceId}/import/preview`,
        {
          method: 'POST',
          headers: requestHeaders,
          body: JSON.stringify({
            file_content: fileContent,
          }),
        },
      );

      const previewData = await previewResponse.json();

      if (isSuccessCode(previewData.code)) {
        const confirmResponse = await fetch(
          `/api/space/${spaceId}/import/confirm`,
          {
            method: 'POST',
            headers: requestHeaders,
            body: JSON.stringify({
              import_token: previewData.data.import_token,
            }),
          },
        );

        const confirmData = await confirmResponse.json();

        if (isSuccessCode(confirmData.code)) {
          setImportResult({
            success: true,
            message: `导入成功！创建了 ${confirmData.data.statistics.agents_created} 个智能体`,
          });
        } else {
          setImportError(
            `导入失败: ${getDisplayErrorMessage(confirmData.msg)}`,
          );
        }
      } else {
        setImportError(`预览失败: ${getDisplayErrorMessage(previewData.msg)}`);
      }
    } catch (error: unknown) {
      console.error('Failed to import:', error);
      setImportError(`导入失败: ${getUnknownErrorMessage(error)}`);
    } finally {
      setImportLoading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  return (
    <div className="bg-[var(--coz-bg-max)] rounded-[14px] border border-solid coz-stroke-primary p-[24px]">
      <h2 className="m-0 text-[16px] leading-[24px] font-[600] coz-fg-plus">
        导入到空间
      </h2>
      <p className="mt-[10px] mb-[20px] text-[13px] leading-[20px] coz-fg-secondary">
        选择之前导出的 ZIP 文件，将资源导入到当前空间。
      </p>
      {importError ? <ErrorAlert message={importError} /> : null}
      {importResult?.success ? <ResultAlert result={importResult} /> : null}
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
        className={`${importLabelClassName} ${
          importLoading ? 'opacity-50 cursor-not-allowed' : ''
        }`}
      >
        {importLoading ? (
          <span className="text-blue-600">正在导入...</span>
        ) : (
          <>
            <IconCozUpload />
            <span>点击选择 ZIP 文件进行导入</span>
          </>
        )}
      </label>
    </div>
  );
};

const ResultAlert: FC<{ result: ResultMessage }> = ({ result }) => (
  <div
    className={`mt-[18px] p-[14px] rounded-[10px] text-[13px] ${
      result.success ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'
    }`}
  >
    <p className="font-medium">{result.message}</p>
  </div>
);

const ErrorAlert: FC<{ message: string }> = ({ message }) => (
  <div className="mb-[14px] p-[14px] bg-red-50 text-red-800 rounded-[10px] text-[13px]">
    {message}
  </div>
);

const UsageNote = () => (
  <div className="mt-[18px] bg-[var(--coz-bg-max)] rounded-[14px] border border-solid coz-stroke-primary border-l-[3px] border-l-[rgb(53,138,255)] py-[20px] px-[24px]">
    <h3 className="m-0 mb-[12px] text-[14px] leading-[22px] font-[600] coz-fg-plus flex items-center gap-[8px]">
      <IconCozInfoCircle className="coz-fg-hglt" />
      使用说明
    </h3>
    <ul className="m-0 p-0 list-none text-[13px] leading-[20px] coz-fg-secondary space-y-[9px]">
      <li>• 导出功能会打包当前空间的所有智能体、插件、工作流和变量</li>
      <li>• 导入时会自动处理 ID 映射，确保导入的资源不会与现有资源冲突</li>
      <li>• 导出的文件有效期为 1 小时</li>
    </ul>
  </div>
);

const SpaceExportImportPage: FC = () => {
  const { space_id } = useParams<{ space_id: string }>();
  const fileInputRef = useRef<HTMLInputElement>(null);

  return (
    <div className="h-full flex flex-col overflow-hidden bg-[rgb(244,246,251)]">
      <div className="flex-none px-[32px] pt-[26px] pb-[18px] bg-[var(--coz-bg-max)] border-0 border-b-[1px] border-solid coz-stroke-primary">
        <div className="flex items-start justify-between gap-[16px]">
          <div>
            <h1 className="m-0 text-[24px] leading-[32px] font-[600] coz-fg-plus">
              空间导出/导入
            </h1>
            <p className="m-0 mt-[8px] text-[13px] leading-[20px] coz-fg-secondary">
              当前空间 ID: <b className="font-[500]">{space_id}</b>
            </p>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-[32px] pt-[22px] pb-[40px]">
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-[18px]">
          {space_id ? <ExportCard spaceId={space_id} /> : null}
          {space_id ? (
            <ImportCard spaceId={space_id} fileInputRef={fileInputRef} />
          ) : null}
        </div>
        <UsageNote />
      </div>
    </div>
  );
};

// React Router lazy loading requires a named 'Component' export
export const Component = SpaceExportImportPage;
export default SpaceExportImportPage;
