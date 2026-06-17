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
import {
  Button,
  Spin,
  Empty,
  Toast,
  Modal,
  Typography,
  Tooltip,
} from '@coze-arch/coze-design';

interface SandboxFile {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
}

const ROOTS = [
  { key: '/workspace', label: '工作区', desc: '智能体的主工作目录' },
  { key: '/outputs', label: '产出物', desc: '任务生成的文件' },
  { key: '/uploads', label: '上传区', desc: '你提供给智能体的素材' },
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

export const SandboxWorkspace: React.FC = () => {
  const botId = useBotInfoStore(state => state.botId);
  const spaceId = useBotInfoStore(state => state.space_id);
  const [root, setRoot] = useState('/workspace');
  const [files, setFiles] = useState<SandboxFile[]>([]);
  const [loading, setLoading] = useState(false);
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

  const onUpload = async (file: File) => {
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

  return (
    <div className="flex flex-col h-full">
      {/* 顶部:目录切换 + 操作 */}
      <div className="flex items-center gap-[8px] px-[8px] pb-[10px] flex-wrap">
        {ROOTS.map(r => (
          <Tooltip key={r.key} content={r.desc}>
            <div
              onClick={() => setRoot(r.key)}
              className={`cursor-pointer px-[12px] py-[5px] rounded-[8px] text-[13px] font-medium transition-colors ${
                root === r.key
                  ? 'coz-mg-hglt coz-fg-hglt'
                  : 'coz-fg-secondary hover:coz-mg-secondary'
              }`}
            >
              {r.label}
            </div>
          </Tooltip>
        ))}
        <div className="flex-1" />
        <Button
          size="small"
          color="primary"
          onClick={() => fileInputRef.current?.click()}
        >
          上传文件
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
              onUpload(f);
            }
            e.target.value = '';
          }}
        />
      </div>

      {/* 文件列表 */}
      <div className="flex-1 overflow-auto px-[4px]">
        {loading ? (
          <div className="flex justify-center py-[40px]">
            <Spin />
          </div>
        ) : files.length === 0 ? (
          <Empty
            title="空目录"
            description="智能体运行后产生的文件会出现在这里"
            className="py-[40px]"
          />
        ) : (
          <div className="flex flex-col gap-[4px]">
            {files.map(f => (
              <div
                key={f.path}
                className="group flex items-center gap-[10px] px-[12px] py-[9px] rounded-[8px] coz-bg-primary hover:coz-mg-secondary cursor-pointer"
                onClick={() => openFile(f)}
              >
                <span className="text-[16px]">{f.is_dir ? '📁' : '📄'}</span>
                <Typography.Text
                  ellipsis={{ showTooltip: true }}
                  className="flex-1 !text-[13px]"
                >
                  {f.name}
                </Typography.Text>
                {!f.is_dir ? (
                  <span className="text-[12px] coz-fg-dim">
                    {formatSize(f.size)}
                  </span>
                ) : null}
                <Button
                  size="mini"
                  color="secondary"
                  className="opacity-0 group-hover:opacity-100"
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
