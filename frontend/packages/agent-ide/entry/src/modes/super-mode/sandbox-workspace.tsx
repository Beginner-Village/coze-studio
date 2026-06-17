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

/* eslint-disable @coze-arch/max-line-per-function */
import { useCallback, useEffect, useRef, useState } from 'react';

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { LazyCozeMdBox } from '@coze-common/chat-uikit';
import { DeveloperApi } from '@coze-arch/bot-api';
import { Button, Spin, Toast, Typography } from '@coze-arch/coze-design';

import {
  iconForFile,
  IcUpload,
  IcRefresh,
  IcDownload,
  IcTrash,
  IcChevronRight,
  IcChevronDown,
  IcFile,
} from './icons';

interface SandboxFile {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
}

const ROOTS = [
  { key: '/workspace', label: '工作区' },
  { key: '/outputs', label: '产出物' },
  { key: '/uploads', label: '上传区' },
];

const IMG_EXT = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'];
const OFFICE_EXT = ['xlsx', 'xls', 'docx', 'doc', 'pptx', 'ppt'];

const extOf = (name: string) => name.split('.').pop()?.toLowerCase() ?? '';

const formatSize = (n: number): string => {
  if (n < 1024) {
    return `${n} B`;
  }
  if (n < 1024 * 1024) {
    return `${(n / 1024).toFixed(1)} KB`;
  }
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
};

const base64ToBlobUrl = (b64: string, mime: string): string => {
  const bytes = atob(b64);
  const arr = new Uint8Array(bytes.length);
  for (let i = 0; i < bytes.length; i++) {
    arr[i] = bytes.charCodeAt(i);
  }
  return URL.createObjectURL(new Blob([arr], { type: mime }));
};

const downloadFile = (name: string, content: string, isBinary: boolean) => {
  const url = isBinary
    ? base64ToBlobUrl(content, 'application/octet-stream')
    : URL.createObjectURL(new Blob([content], { type: 'text/plain' }));
  const a = document.createElement('a');
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
};

interface PreviewState {
  name: string;
  path: string;
  content: string;
  isBinary: boolean;
}

// ---- 文件查看器(按类型内联渲染) ----
const FileViewer: React.FC<{ preview: PreviewState }> = ({ preview }) => {
  const { name, content, isBinary } = preview;
  const ext = extOf(name);

  if (isBinary && IMG_EXT.includes(ext)) {
    const mime = ext === 'svg' ? 'svg+xml' : ext === 'jpg' ? 'jpeg' : ext;
    return (
      <div className="flex justify-center p-[16px]">
        <img
          src={`data:image/${mime};base64,${content}`}
          alt={name}
          className="max-w-full object-contain rounded-[6px]"
        />
      </div>
    );
  }
  if (!isBinary && ext === 'svg') {
    const svgUrl = `data:image/svg+xml;base64,${btoa(
      unescape(encodeURIComponent(content)),
    )}`;
    return (
      <div className="flex justify-center p-[16px]">
        <img src={svgUrl} alt={name} className="max-w-full object-contain" />
      </div>
    );
  }
  if (ext === 'pdf') {
    const url = isBinary ? base64ToBlobUrl(content, 'application/pdf') : '';
    return <iframe title={name} src={url} className="w-full h-full border-0" />;
  }
  if (ext === 'md' || ext === 'markdown') {
    return (
      <div className="px-[16px] py-[12px]">
        <LazyCozeMdBox
          markDown={content}
          autoFixSyntax={{ autoFixEnding: false }}
        />
      </div>
    );
  }
  if (ext === 'html' || ext === 'htm') {
    return (
      <iframe
        title={name}
        srcDoc={content}
        sandbox="allow-scripts"
        className="w-full h-full border-0"
      />
    );
  }
  if (ext === 'csv') {
    const rows = content
      .split('\n')
      .filter(l => l.trim())
      .slice(0, 300)
      .map(l => l.split(','));
    return (
      <div className="p-[12px] overflow-auto">
        <table className="text-[12px] border-collapse">
          <tbody>
            {rows.map((r, i) => (
              <tr key={i} className={i === 0 ? 'coz-mg-secondary font-medium' : ''}>
                {r.map((c, j) => (
                  <td
                    key={j}
                    className="border coz-stroke-primary px-[8px] py-[4px] whitespace-nowrap"
                  >
                    {c}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }
  if (OFFICE_EXT.includes(ext)) {
    const Icon = iconForFile(name, false);
    return (
      <div className="flex flex-col items-center justify-center h-full gap-[12px] coz-fg-secondary">
        <Icon size={48} className="opacity-50" />
        <div className="text-[13px]">{name}</div>
        <div className="coz-fg-dim text-[12px]">
          Office 文档在线预览即将支持,可先下载用本地软件打开
        </div>
        <Button
          color="primary"
          onClick={() => downloadFile(name, content, isBinary)}
        >
          下载文件
        </Button>
      </div>
    );
  }
  if (ext === 'json') {
    let pretty = content;
    try {
      pretty = JSON.stringify(JSON.parse(content), null, 2);
    } catch {
      /* keep raw */
    }
    return (
      <pre className="text-[12px] font-mono leading-[1.65] coz-fg-primary whitespace-pre-wrap break-all p-[14px]">
        {pretty}
      </pre>
    );
  }
  if (isBinary) {
    return (
      <div className="flex items-center justify-center h-full coz-fg-secondary text-[13px]">
        二进制文件,暂不支持预览（{formatSize(content.length)}）
      </div>
    );
  }
  return (
    <pre className="text-[12px] font-mono leading-[1.65] coz-fg-primary whitespace-pre-wrap break-all p-[14px]">
      {content}
    </pre>
  );
};

// ---- 文件树节点(懒加载子目录) ----
const TreeNode: React.FC<{
  file: SandboxFile;
  depth: number;
  activePath?: string;
  listDir: (path: string) => Promise<SandboxFile[]>;
  onOpen: (f: SandboxFile) => void;
  onDelete: (f: SandboxFile) => void;
  onDownload: (f: SandboxFile) => void;
}> = ({ file, depth, activePath, listDir, onOpen, onDelete, onDownload }) => {
  const [expanded, setExpanded] = useState(false);
  const [children, setChildren] = useState<SandboxFile[] | null>(null);
  const [loading, setLoading] = useState(false);
  const Icon = iconForFile(file.name, file.is_dir, expanded);
  const active = activePath === file.path;

  const toggle = async () => {
    if (!file.is_dir) {
      onOpen(file);
      return;
    }
    const next = !expanded;
    setExpanded(next);
    if (next && children === null) {
      setLoading(true);
      setChildren(await listDir(file.path));
      setLoading(false);
    }
  };

  return (
    <div>
      <div
        className={`group flex items-center gap-[6px] py-[4px] pr-[6px] rounded-[6px] cursor-pointer text-[13px] ${
          active ? 'coz-mg-hglt' : 'hover:coz-mg-secondary'
        }`}
        style={{ paddingLeft: depth * 14 + 6 }}
        onClick={toggle}
      >
        <span className="w-[12px] coz-fg-dim shrink-0 flex items-center">
          {file.is_dir ? (
            expanded ? (
              <IcChevronDown size={12} />
            ) : (
              <IcChevronRight size={12} />
            )
          ) : null}
        </span>
        <Icon
          size={15}
          className={file.is_dir ? 'coz-fg-hglt' : 'coz-fg-secondary'}
        />
        <Typography.Text
          ellipsis={{ showTooltip: true }}
          className={`flex-1 !text-[13px] ${active ? 'coz-fg-hglt' : 'coz-fg-primary'}`}
        >
          {file.name}
        </Typography.Text>
        {!file.is_dir ? (
          <span className="text-[11px] coz-fg-dim shrink-0">
            {formatSize(file.size)}
          </span>
        ) : null}
        {!file.is_dir ? (
          <span
            className="opacity-0 group-hover:opacity-100 shrink-0 coz-fg-dim hover:coz-fg-primary"
            title="下载"
            onClick={e => {
              e.stopPropagation();
              onDownload(file);
            }}
          >
            <IcDownload size={14} />
          </span>
        ) : null}
        <span
          className="opacity-0 group-hover:opacity-100 shrink-0 coz-fg-dim hover:coz-fg-hglt-red"
          title="删除"
          onClick={e => {
            e.stopPropagation();
            onDelete(file);
          }}
        >
          <IcTrash size={14} />
        </span>
      </div>
      {file.is_dir && expanded ? (
        loading ? (
          <div style={{ paddingLeft: (depth + 1) * 14 + 6 }} className="py-[4px]">
            <Spin size="small" />
          </div>
        ) : (
          (children ?? []).map(c => (
            <TreeNode
              key={c.path}
              file={c}
              depth={depth + 1}
              activePath={activePath}
              listDir={listDir}
              onOpen={onOpen}
              onDelete={onDelete}
              onDownload={onDownload}
            />
          ))
        )
      ) : null}
    </div>
  );
};

export const SandboxWorkspace: React.FC = () => {
  const botId = useBotInfoStore(state => state.botId);
  const spaceId = useBotInfoStore(state => state.space_id);
  const [root, setRoot] = useState('/workspace');
  const [files, setFiles] = useState<SandboxFile[]>([]);
  const [loading, setLoading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [preview, setPreview] = useState<PreviewState | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const listDir = useCallback(
    async (path: string): Promise<SandboxFile[]> => {
      if (!botId) {
        return [];
      }
      try {
        const resp = await DeveloperApi.ListSandboxFiles({
          space_id: spaceId,
          bot_id: botId,
          path,
        });
        const list = (resp?.data?.files as SandboxFile[]) ?? [];
        return list.sort((a, b) =>
          a.is_dir === b.is_dir
            ? a.name.localeCompare(b.name)
            : a.is_dir
              ? -1
              : 1,
        );
      } catch (e) {
        Toast.error('读取沙箱目录失败');
        return [];
      }
    },
    [botId, spaceId],
  );

  const loadRoot = useCallback(
    async (path: string) => {
      setLoading(true);
      setFiles(await listDir(path));
      setLoading(false);
    },
    [listDir],
  );

  useEffect(() => {
    loadRoot(root);
  }, [root, loadRoot]);

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
        name: f.name,
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
        loadRoot('/uploads');
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
      if (preview?.path === f.path) {
        setPreview(null);
      }
      loadRoot(root);
    } catch (e) {
      Toast.error('删除失败');
    }
  };

  const onDownload = async (f: SandboxFile) => {
    try {
      const resp = await DeveloperApi.ReadSandboxFile({
        space_id: spaceId,
        bot_id: botId,
        path: f.path,
      });
      downloadFile(
        f.name,
        resp?.data?.content ?? '',
        Boolean(resp?.data?.is_binary),
      );
    } catch (e) {
      Toast.error('下载失败');
    }
  };

  return (
    <div className="flex h-full">
      {/* 左:文件树 */}
      <div className="w-[270px] shrink-0 flex flex-col border-r coz-stroke-primary">
        <div className="flex items-center gap-[4px] px-[8px] py-[8px]">
          <div className="flex items-center gap-[2px] p-[2px] rounded-[8px] coz-mg-secondary flex-1">
            {ROOTS.map(r => (
              <div
                key={r.key}
                onClick={() => setRoot(r.key)}
                className={`flex-1 text-center cursor-pointer px-[6px] py-[4px] rounded-[6px] text-[12px] font-medium transition-all ${
                  root === r.key
                    ? 'coz-bg-max coz-fg-plus shadow-sm'
                    : 'coz-fg-secondary hover:coz-fg-primary'
                }`}
              >
                {r.label}
              </div>
            ))}
          </div>
          <span
            className="shrink-0 coz-fg-secondary hover:coz-fg-primary cursor-pointer p-[4px]"
            title="上传"
            onClick={() => fileInputRef.current?.click()}
          >
            <IcUpload size={16} />
          </span>
          <span
            className="shrink-0 coz-fg-secondary hover:coz-fg-primary cursor-pointer p-[4px]"
            title="刷新"
            onClick={() => loadRoot(root)}
          >
            <IcRefresh size={16} />
          </span>
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
        <div
          className={`flex-1 overflow-auto px-[6px] pb-[8px] ${
            dragOver ? 'coz-mg-hglt' : ''
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
            <div className="flex justify-center py-[40px]">
              <Spin />
            </div>
          ) : files.length === 0 ? (
            <div className="text-center py-[40px] coz-fg-dim text-[12px] select-none">
              空目录
              <br />
              产物会出现在这里
            </div>
          ) : (
            files.map(f => (
              <TreeNode
                key={f.path}
                file={f}
                depth={0}
                activePath={preview?.path}
                listDir={listDir}
                onOpen={openFile}
                onDelete={onDelete}
                onDownload={onDownload}
              />
            ))
          )}
        </div>
      </div>

      {/* 右:内联预览区 */}
      <div className="flex-1 min-w-0 flex flex-col">
        {preview ? (
          <>
            <div className="flex items-center gap-[10px] px-[14px] py-[8px] border-b coz-stroke-primary shrink-0">
              <span className="font-mono text-[12px] coz-fg-secondary flex-1 truncate">
                {preview.path}
              </span>
              <Button
                size="small"
                color="primary"
                icon={<IcDownload size={14} />}
                onClick={() =>
                  downloadFile(preview.name, preview.content, preview.isBinary)
                }
              >
                下载
              </Button>
            </div>
            <div className="flex-1 min-h-0 overflow-auto">
              <FileViewer preview={preview} />
            </div>
          </>
        ) : (
          <div className="flex-1 flex flex-col items-center justify-center coz-fg-dim gap-[10px] select-none">
            <IcFile size={40} className="opacity-40" />
            <div className="text-[13px]">选择左侧文件查看</div>
            <div className="text-[12px]">
              支持图片 / PDF / Markdown / 代码 / CSV 等在线预览
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
