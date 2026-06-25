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

/* eslint-disable @coze-arch/max-line-per-function, max-lines --
 * This workbench owns the sandbox tree, editor preview, and upload flows.
 */
import { useCallback, useEffect, useRef, useState } from 'react';

import classNames from 'classnames';
import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { LazyCozeMdBox } from '@coze-common/chat-uikit';
import {
  Button,
  Input,
  Modal,
  Spin,
  Toast,
  Typography,
} from '@coze-arch/coze-design';
import { DeveloperApi } from '@coze-arch/bot-api';

import {
  sortSandboxFiles,
  summarizeHarnessPlan,
  type SandboxFile,
  type SuperAgentHarnessPlanState,
} from './sandbox-workspace-utils';
import {
  iconForFile,
  IcArchive,
  IcUpload,
  IcRefresh,
  IcDownload,
  IcTrash,
  IcChevronRight,
  IcChevronDown,
  IcFile,
  IcPlus,
  IcClose,
  IcFolder,
} from './icons';

import { OfficeViewer } from './office-viewer';
import ws from './sandbox-workspace.module.less';

const ROOTS = [
  { key: '/workspace', label: '文件', Icon: IcFolder },
  { key: '/outputs', label: '产物', Icon: IcArchive },
  { key: '/uploads', label: '上传区', Icon: IcUpload },
];

const IMG_EXT = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'bmp', 'ico'];
const OFFICE_EXT = ['xlsx', 'xls', 'docx', 'doc', 'pptx', 'ppt'];
const CODE_EXT = [
  'py',
  'js',
  'ts',
  'tsx',
  'jsx',
  'go',
  'java',
  'c',
  'cpp',
  'rs',
  'sh',
  'rb',
  'php',
  'vue',
  'css',
  'txt',
  'log',
];

const CODE_KEYWORDS = new Set([
  'async',
  'await',
  'catch',
  'class',
  'const',
  'def',
  'else',
  'export',
  'False',
  'for',
  'from',
  'function',
  'if',
  'import',
  'in',
  'let',
  'new',
  'return',
  'throw',
  'True',
  'try',
  'var',
]);

const CODE_TOKEN_RE =
  /(#.*$|\/\/.*$|"(?:\\.|[^"])*"|'(?:\\.|[^'])*'|`(?:\\.|[^`])*`|\b(?:async|await|catch|class|const|def|else|export|False|for|from|function|if|import|in|let|new|return|throw|True|try|var)\b|\b\d+(?:\.\d+)?\b)/g;

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

const editorMeta = (preview?: PreviewState | null) => {
  if (!preview) {
    return { lang: '沙箱', size: '', line: '' };
  }
  const ext = extOf(preview.name);
  const lang =
    ext === 'py'
      ? 'Python'
      : ext === 'md' || ext === 'markdown'
        ? 'Markdown'
        : ext === 'js'
          ? 'JavaScript'
          : ext.toUpperCase() || 'TEXT';
  return {
    lang,
    size: preview.isBinary ? '' : formatSize(preview.content.length),
    line: '',
  };
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

const renderCodeLine = (line: string) => {
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  for (const match of line.matchAll(CODE_TOKEN_RE)) {
    const token = match[0];
    const index = match.index ?? 0;
    if (index > lastIndex) {
      parts.push(line.slice(lastIndex, index));
    }
    const className =
      token.startsWith('#') || token.startsWith('//')
        ? ws.codeComment
        : token.startsWith('"') ||
            token.startsWith("'") ||
            token.startsWith('`')
          ? ws.codeString
          : CODE_KEYWORDS.has(token)
            ? ws.codeKeyword
            : ws.codeNumber;
    parts.push(
      <span key={`${index}-${token}`} className={className}>
        {token}
      </span>,
    );
    lastIndex = index + token.length;
  }
  if (lastIndex < line.length) {
    parts.push(line.slice(lastIndex));
  }
  return parts.length ? parts : ' ';
};

interface PreviewState {
  name: string;
  path: string;
  content: string;
  isBinary: boolean;
}

interface HarnessState {
  plan?: SuperAgentHarnessPlanState | null;
  tool_outputs?: {
    root?: string;
    files?: SandboxFile[];
  };
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
              <tr
                key={i}
                className={i === 0 ? 'coz-mg-secondary font-medium' : ''}
              >
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
    if (isBinary) {
      return <OfficeViewer ext={ext} content={content} />;
    }
    const Icon = iconForFile(name, false);
    return (
      <div className="flex flex-col items-center justify-center h-full gap-[12px] coz-fg-secondary">
        <Icon size={48} className="opacity-50" />
        <div className="text-[13px]">{name}</div>
        <div className="coz-fg-dim text-[12px]">
          无法在线渲染此文档,可先下载用本地软件打开
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
      pretty = content;
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
  if (CODE_EXT.includes(ext)) {
    return (
      <pre className={ws.codePreview}>
        {content.split('\n').map((line, index) => (
          <span className={ws.codeLine} key={index}>
            <span className={ws.codeLineNumber}>{index + 1}</span>
            <span>{renderCodeLine(line)}</span>
          </span>
        ))}
      </pre>
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
  readonly?: boolean;
}> = ({
  file,
  depth,
  activePath,
  listDir,
  onOpen,
  onDelete,
  onDownload,
  readonly,
}) => {
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
        className={classNames(ws.fileRow, active && ws.fileRowActive)}
        style={{ paddingLeft: depth * 14 + 8 }}
        onClick={toggle}
      >
        <span className={ws.twirl}>
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
          className={classNames(ws.fileIcon, file.is_dir && ws.dirIcon)}
        />
        <div className={ws.fileMain}>
          <Typography.Text
            ellipsis={{ showTooltip: true }}
            className={ws.fileName}
          >
            {file.name}
          </Typography.Text>
          {!file.is_dir ? (
            <div className={ws.fileMeta}>{formatSize(file.size)}</div>
          ) : null}
        </div>
        <span className={ws.rowActions}>
          {!file.is_dir ? (
            <span
              className={ws.rowAction}
              title="下载"
              onClick={e => {
                e.stopPropagation();
                onDownload(file);
              }}
            >
              <IcDownload size={14} />
            </span>
          ) : null}
          {!readonly ? (
            <span
              className={classNames(ws.rowAction, ws.rowActionDanger)}
              title="删除"
              onClick={e => {
                e.stopPropagation();
                onDelete(file);
              }}
            >
              <IcTrash size={14} />
            </span>
          ) : null}
        </span>
      </div>
      {file.is_dir && expanded ? (
        loading ? (
          <div
            style={{ paddingLeft: (depth + 1) * 14 + 6 }}
            className="py-[4px]"
          >
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
              readonly={readonly}
            />
          ))
        )
      ) : null}
    </div>
  );
};

export const SandboxWorkspace: React.FC = () => {
  const botId = useBotInfoStore((state: { botId: string }) => state.botId);
  const spaceId = useBotInfoStore(
    (state: { space_id: string }) => state.space_id,
  );
  const [root, setRoot] = useState('/workspace');
  const [files, setFiles] = useState<SandboxFile[]>([]);
  const [loading, setLoading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const [openTabs, setOpenTabs] = useState<PreviewState[]>([]);
  const [preview, setPreview] = useState<PreviewState | null>(null);
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [harnessState, setHarnessState] = useState<HarnessState | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [createName, setCreateName] = useState('');
  const [creating, setCreating] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchHarnessState =
    useCallback(async (): Promise<HarnessState | null> => {
      if (!botId) {
        return null;
      }
      try {
        const resp = await DeveloperApi.SuperAgentGetHarnessState({
          space_id: spaceId,
          bot_id: botId,
        });
        const data = (resp?.data ?? null) as HarnessState | null;
        setHarnessState(data);
        return data;
      } catch {
        return null;
      }
    }, [botId, spaceId]);

  const listDir = useCallback(
    async (path: string): Promise<SandboxFile[]> => {
      if (!botId) {
        return [];
      }
      try {
        const resp = await DeveloperApi.SuperAgentListWorkspaceFiles({
          space_id: spaceId,
          bot_id: botId,
          path,
        });
        const list = (resp?.data?.files as SandboxFile[]) ?? [];
        return sortSandboxFiles(list);
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
      const list = await listDir(path);
      setFiles(list);
      setCounts(prev => ({ ...prev, [path]: list.length }));
      setLoading(false);
    },
    [listDir],
  );

  const refreshCounts = useCallback(async () => {
    if (!botId) {
      return;
    }
    const entries = await Promise.all(
      ROOTS.map(async r => [r.key, (await listDir(r.key)).length] as const),
    );
    setCounts(prev => {
      const next = { ...prev };
      for (const [key, n] of entries) {
        next[key] = n;
      }
      return next;
    });
  }, [botId, listDir]);

  useEffect(() => {
    loadRoot(root);
  }, [root, loadRoot]);

  useEffect(() => {
    refreshCounts();
  }, [refreshCounts]);

  useEffect(() => {
    fetchHarnessState();
  }, [fetchHarnessState]);

  const openFile = async (f: SandboxFile) => {
    if (f.is_dir) {
      return;
    }
    try {
      const resp = await DeveloperApi.SuperAgentReadWorkspaceFile({
        space_id: spaceId,
        bot_id: botId,
        path: f.path,
      });
      const nextPreview = {
        name: f.name,
        path: f.path,
        content: resp?.data?.content ?? '',
        isBinary: Boolean(resp?.data?.is_binary),
      };
      setPreview(nextPreview);
      setOpenTabs(prev => {
        const exists = prev.some(tab => tab.path === nextPreview.path);
        return exists
          ? prev.map(tab => (tab.path === nextPreview.path ? nextPreview : tab))
          : [...prev, nextPreview];
      });
    } catch (e) {
      Toast.error('读取文件失败');
    }
  };

  const activateTab = async (tab: PreviewState) => {
    if (tab.content || tab.isBinary) {
      setPreview(tab);
      return;
    }
    try {
      const resp = await DeveloperApi.SuperAgentReadWorkspaceFile({
        space_id: spaceId,
        bot_id: botId,
        path: tab.path,
      });
      const nextPreview = {
        ...tab,
        content: resp?.data?.content ?? '',
        isBinary: Boolean(resp?.data?.is_binary),
      };
      setPreview(nextPreview);
      setOpenTabs(prev =>
        prev.map(item => (item.path === nextPreview.path ? nextPreview : item)),
      );
    } catch {
      setPreview(tab);
    }
  };

  const closeTab = (tab: PreviewState) => {
    setOpenTabs(prev => {
      const next = prev.filter(item => item.path !== tab.path);
      if (preview?.path === tab.path) {
        setPreview(next[0] ?? null);
      }
      return next;
    });
  };

  const uploadOne = (file: File) => {
    const reader = new FileReader();
    reader.onload = async () => {
      const result = String(reader.result || '');
      const base64 = result.includes(',') ? result.split(',')[1] : result;
      try {
        await DeveloperApi.SuperAgentUploadWorkspaceFile({
          space_id: spaceId,
          bot_id: botId,
          path: `/uploads/${file.name}`,
          content: base64,
          is_base64: true,
        });
        Toast.success(`已上传 ${file.name}`);
        setRoot('/uploads');
        loadRoot('/uploads');
        refreshCounts();
      } catch (e) {
        Toast.error('上传失败');
      }
    };
    reader.readAsDataURL(file);
  };

  const submitCreate = async () => {
    const name = createName.trim().replace(/^\/+/, '');
    if (!name) {
      Toast.error('请输入文件名');
      return;
    }
    setCreating(true);
    try {
      await DeveloperApi.SuperAgentWriteWorkspaceFile({
        space_id: spaceId,
        bot_id: botId,
        path: `${root}/${name}`,
        content: '',
        is_base64: false,
      });
      Toast.success(`已创建 ${name}`);
      setCreateOpen(false);
      setCreateName('');
      loadRoot(root);
      refreshCounts();
    } catch (e) {
      Toast.error('创建文件失败');
    } finally {
      setCreating(false);
    }
  };

  const onDelete = async (f: SandboxFile) => {
    try {
      await DeveloperApi.SuperAgentDeleteWorkspaceFile({
        space_id: spaceId,
        bot_id: botId,
        path: f.path,
      });
      Toast.success('已删除');
      setOpenTabs(prev => prev.filter(tab => tab.path !== f.path));
      if (preview?.path === f.path) {
        setPreview(null);
      }
      loadRoot(root);
      refreshCounts();
    } catch (e) {
      Toast.error('删除失败');
    }
  };

  const onDownload = async (f: SandboxFile) => {
    try {
      const resp = await DeveloperApi.SuperAgentReadWorkspaceFile({
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

  const planSummary = summarizeHarnessPlan(harnessState?.plan);
  const toolOutputCount = harnessState?.tool_outputs?.files?.length ?? 0;
  const meta = editorMeta(preview);

  return (
    <div className={ws.workbench}>
      <section className={ws.panel}>
        <div className={ws.panelHeader}>
          <div className={ws.panelTitle}>沙箱工作区</div>
          <div className={ws.runtime}>
            <span className={ws.runtimeDot} />
            <span>运行中 · 隔离环境</span>
          </div>
        </div>
        <div className={ws.segment}>
          {ROOTS.map(r => {
            const SegmentIcon = r.Icon;
            return (
              <button
                key={r.key}
                type="button"
                onClick={() => setRoot(r.key)}
                className={classNames(
                  ws.segmentButton,
                  root === r.key && ws.segmentButtonActive,
                )}
              >
                <SegmentIcon size={13} />
                <span>{r.label}</span>
                {counts[r.key] !== undefined ? (
                  <span className={ws.segmentCount}>{counts[r.key]}</span>
                ) : null}
              </button>
            );
          })}
        </div>
        <div className={ws.iconRow}>
          <span className={ws.iconSpacer} />
          <button
            className={ws.iconButton}
            title="新建"
            type="button"
            onClick={() => {
              setCreateName('');
              setCreateOpen(true);
            }}
          >
            <IcPlus size={16} />
          </button>
          <button
            className={ws.iconButton}
            title="上传"
            type="button"
            onClick={() => fileInputRef.current?.click()}
          >
            <IcUpload size={16} />
          </button>
          <button
            className={ws.iconButton}
            title="刷新"
            type="button"
            onClick={() => loadRoot(root)}
          >
            <IcRefresh size={16} />
          </button>
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
        {root === '/uploads' ? (
          <div
            className={ws.uploadZone}
            onClick={() => fileInputRef.current?.click()}
          >
            <div className={ws.uploadIcon}>
              <IcUpload size={20} />
            </div>
            <div className={ws.uploadTitle}>
              拖拽文件到此处，或
              <span className={ws.uploadLink}>点击上传</span>
            </div>
            <div className={ws.uploadDesc}>
              支持 图片/PDF/Markdown/代码/CSV，单个 ≤ 50MB
            </div>
          </div>
        ) : null}
        <div
          className={classNames(ws.fileTree, dragOver && ws.fileTreeDrag)}
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
        <div className={ws.fileFooter}>
          <div className={ws.footerPill}>{planSummary}</div>
          <div className={ws.footerPill}>
            工具输出 <b>{toolOutputCount}</b>
          </div>
        </div>
      </section>

      <section className={ws.panel}>
        <div className={ws.editorTabs}>
          {openTabs.map(tab => (
            <button
              key={tab.path}
              type="button"
              className={classNames(
                ws.editorTab,
                preview?.path === tab.path && ws.editorTabActive,
              )}
              onClick={() => activateTab(tab)}
            >
              <IcFile size={14} />
              <span>{tab.name}</span>
              <span
                className={ws.editorTabClose}
                onClick={e => {
                  e.stopPropagation();
                  closeTab(tab);
                }}
              >
                <IcClose size={12} />
              </span>
            </button>
          ))}
        </div>
        <div className={ws.editorArea}>
          {preview ? <FileViewer preview={preview} /> : null}
          {!preview ? (
            <div className={ws.empty}>
              <div className={ws.emptyIcon}>
                <IcFile size={30} />
              </div>
              <div className={ws.emptyTitle}>选择左侧文件查看</div>
              <div className={ws.emptyDesc}>
                支持 图片 / PDF / Markdown / 代码 / CSV 等在线预览
              </div>
            </div>
          ) : null}
        </div>
        <div className={ws.editorStatus}>
          {preview ? (
            <>
              <span>{meta.lang}</span>
              <span>UTF-8</span>
              <span>沙箱 · 隔离环境</span>
              <span className={ws.editorStatusRight}>
                {meta.line ? <span>{meta.line}</span> : null}
                {meta.size ? <span>{meta.size}</span> : null}
              </span>
            </>
          ) : (
            <span>沙箱 · 隔离环境</span>
          )}
        </div>
      </section>

      <Modal
        visible={createOpen}
        title="新建文件"
        okText="创建"
        cancelText="取消"
        okButtonProps={{ loading: creating }}
        onOk={submitCreate}
        onCancel={() => setCreateOpen(false)}
      >
        <Input
          value={createName}
          placeholder={`文件名，将创建在 ${root}/`}
          onChange={setCreateName}
          onEnterPress={submitCreate}
        />
      </Modal>
    </div>
  );
};
