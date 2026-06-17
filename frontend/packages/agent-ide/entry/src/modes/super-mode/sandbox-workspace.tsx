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

import { useCallback, useEffect, useRef, useState } from 'react';

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { DeveloperApi } from '@coze-arch/bot-api';
import { Button, Spin, Toast, Modal, Typography } from '@coze-arch/coze-design';

interface SandboxFile {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
}

const ROOTS = [
  { key: '/workspace', label: '工作区', icon: '🗂️' },
  { key: '/outputs', label: '产出物', icon: '📤' },
  { key: '/uploads', label: '上传区', icon: '📥' },
];

const formatSize = (n: number): string => {
  if (n < 1024) {
    return `${n} B`;
  }
  if (n < 1024 * 1024) {
    return `${(n / 1024).toFixed(1)} KB`;
  }
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
};

const fileIcon = (f: SandboxFile): string => {
  if (f.is_dir) {
    return '📁';
  }
  const ext = f.name.split('.').pop()?.toLowerCase() ?? '';
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp'].includes(ext)) {
    return '🖼️';
  }
  if (['py', 'js', 'ts', 'go', 'java', 'c', 'cpp', 'rs', 'sh'].includes(ext)) {
    return '📜';
  }
  if (['json', 'yaml', 'yml', 'toml', 'xml'].includes(ext)) {
    return '⚙️';
  }
  if (['md', 'txt', 'log'].includes(ext)) {
    return '📝';
  }
  if (['zip', 'tar', 'gz', 'rar', '7z'].includes(ext)) {
    return '🗜️';
  }
  if (['csv', 'xlsx', 'xls'].includes(ext)) {
    return '📊';
  }
  if (ext === 'pdf') {
    return '📕';
  }
  return '📄';
};

export const SandboxWorkspace: React.FC = () => {
  const botId = useBotInfoStore(state => state.botId);
  const spaceId = useBotInfoStore(state => state.space_id);
  const [root, setRoot] = useState('/workspace');
  const [files, setFiles] = useState<SandboxFile[]>([]);
  const [loading, setLoading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [preview, setPreview] = useState<{
    path: string;
    content: string;
    isBinary: boolean;
  } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const load = useCallback(
    async (path: string) => {
      if (!botId) {
        return;
      }
      setLoading(true);
      try {
        const resp = await DeveloperApi.ListSandboxFiles({
          space_id: spaceId,
          bot_id: botId,
          path,
        });
        setFiles((resp?.data?.files as SandboxFile[]) ?? []);
      } catch (e) {
        setFiles([]);
        Toast.error('读取沙箱目录失败');
      } finally {
        setLoading(false);
      }
    },
    [botId, spaceId],
  );

  useEffect(() => {
    load(root);
  }, [root, load]);

  const openFile = async (f: SandboxFile) => {
    if (f.is_dir) {
      return;
    }
    try {
      const resp = await DeveloperApi.ReadSandboxFile({
        space_id: spaceId,
        bot_id: botId,
        path: f.path,
      });
      setPreview({
        path: f.path,
        content: resp?.data?.content ?? '',
        isBinary: Boolean(resp?.data?.is_binary),
      });
    } catch (e) {
      Toast.error('读取文件失败');
    }
  };

  const uploadOne = (file: File) => {
    const reader = new FileReader();
    reader.onload = async () => {
      const result = String(reader.result || '');
      const base64 = result.includes(',') ? result.split(',')[1] : result;
      try {
        await DeveloperApi.UploadSandboxFile({
          space_id: spaceId,
          bot_id: botId,
          path: `/uploads/${file.name}`,
          content: base64,
          is_base64: true,
        });
        Toast.success(`已上传 ${file.name}`);
        setRoot('/uploads');
        load('/uploads');
      } catch (e) {
        Toast.error('上传失败');
      }
    };
    reader.readAsDataURL(file);
  };

  const onDelete = async (f: SandboxFile) => {
    try {
      await DeveloperApi.DeleteSandboxFile({
        space_id: spaceId,
        bot_id: botId,
        path: f.path,
      });
      Toast.success('已删除');
      load(root);
    } catch (e) {
      Toast.error('删除失败');
    }
  };

  const totalSize = files.reduce((acc, f) => acc + (f.is_dir ? 0 : f.size), 0);

  return (
    <div className="flex flex-col h-full">
      {/* 目录分段切换 */}
      <div className="flex items-center gap-[6px] px-[10px] pt-[2px] pb-[10px]">
        <div className="flex items-center gap-[2px] p-[3px] rounded-[10px] coz-mg-secondary">
          {ROOTS.map(r => (
            <div
              key={r.key}
              onClick={() => setRoot(r.key)}
              className={`flex items-center gap-[5px] cursor-pointer px-[12px] py-[5px] rounded-[8px] text-[13px] font-medium transition-all ${
                root === r.key
                  ? 'coz-bg-max coz-fg-plus shadow-sm'
                  : 'coz-fg-secondary hover:coz-fg-primary'
              }`}
            >
              <span className="text-[13px]">{r.icon}</span>
              {r.label}
            </div>
          ))}
        </div>
        <div className="flex-1" />
        <Button size="small" color="primary" onClick={() => fileInputRef.current?.click()}>
          上传
        </Button>
        <Button size="small" color="secondary" onClick={() => load(root)}>
          刷新
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={e => {
            const f = e.target.files?.[0];
            if (f) {
              uploadOne(f);
            }
            e.target.value = '';
          }}
        />
      </div>

      {/* 路径 + 统计 */}
      <div className="flex items-center justify-between px-[14px] pb-[8px] text-[12px] coz-fg-dim">
        <span className="font-mono">{root}</span>
        <span>
          {files.length} 项 · {formatSize(totalSize)}
        </span>
      </div>

      {/* 文件列表 / 拖拽上传区 */}
      <div
        className={`flex-1 overflow-auto mx-[8px] mb-[8px] rounded-[12px] border border-dashed transition-colors ${
          dragOver ? 'coz-stroke-hglt coz-mg-hglt' : 'coz-stroke-primary'
        }`}
        onDragOver={e => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={e => {
          e.preventDefault();
          setDragOver(false);
          const f = e.dataTransfer.files?.[0];
          if (f) {
            uploadOne(f);
          }
        }}
      >
        {loading ? (
          <div className="flex justify-center py-[48px]">
            <Spin />
          </div>
        ) : files.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full py-[48px] text-center select-none">
            <div className="text-[40px] mb-[8px] opacity-70">📭</div>
            <div className="text-[14px] coz-fg-secondary font-medium">
              空目录
            </div>
            <div className="text-[12px] coz-fg-dim mt-[4px] max-w-[200px]">
              智能体运行产生的文件会出现在这里,也可拖拽文件到此上传
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-[2px] p-[6px]">
            {files.map(f => (
              <div
                key={f.path}
                className="group flex items-center gap-[10px] px-[10px] py-[8px] rounded-[8px] hover:coz-mg-secondary cursor-pointer transition-colors"
                onClick={() => openFile(f)}
              >
                <span className="text-[18px] leading-none shrink-0">
                  {fileIcon(f)}
                </span>
                <Typography.Text
                  ellipsis={{ showTooltip: true }}
                  className="flex-1 !text-[13px] coz-fg-primary"
                >
                  {f.name}
                </Typography.Text>
                {!f.is_dir ? (
                  <span className="text-[11px] coz-fg-dim shrink-0">
                    {formatSize(f.size)}
                  </span>
                ) : (
                  <span className="text-[11px] coz-fg-dim shrink-0">目录</span>
                )}
                <Button
                  size="mini"
                  color="secondary"
                  className="opacity-0 group-hover:opacity-100 shrink-0"
                  onClick={e => {
                    e.stopPropagation();
                    onDelete(f);
                  }}
                >
                  删除
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 文件预览 */}
      <Modal
        visible={Boolean(preview)}
        title={preview?.path}
        onCancel={() => setPreview(null)}
        footer={null}
        width={680}
      >
        {preview?.isBinary ? (
          <div className="coz-fg-secondary py-[20px] text-center">
            二进制文件,无法预览(大小 {formatSize(preview.content.length)})
          </div>
        ) : (
          <pre className="text-[12px] coz-fg-primary whitespace-pre-wrap break-all max-h-[60vh] overflow-auto coz-bg-secondary p-[12px] rounded-[8px]">
            {preview?.content}
          </pre>
        )}
      </Modal>
    </div>
  );
};
