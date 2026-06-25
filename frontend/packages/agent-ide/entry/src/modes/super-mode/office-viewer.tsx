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

/* Office 文档在线预览:Excel(SheetJS) / Word(docx-preview) / PPT(pptx-preview)。
 * 三个库都通过动态 import 懒加载,完全在浏览器侧渲染,不占用沙箱内存。 */
import { useEffect, useRef, useState } from 'react';

import { Spin } from '@coze-arch/coze-design';

import './office-viewer.css';

const base64ToBytes = (b64: string): Uint8Array => {
  const bin = atob(b64);
  const arr = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) {
    arr[i] = bin.charCodeAt(i);
  }
  return arr;
};

interface OfficeViewerProps {
  ext: string;
  content: string; // base64
}

// ---- Excel ----
// 安全说明:xlsx 来自沙箱(agent 生成 / 用户上传),内容不可信。绝不使用
// SheetJS 的 sheet_to_html + dangerouslySetInnerHTML(单元格内容不转义 → XSS);
// 改为 sheet_to_json 取二维数组,交给 React 渲染 <td>,由 React 逐格转义。
interface SheetData {
  name: string;
  rows: string[][];
}

const cellToText = (v: unknown): string => {
  if (v === null || v === undefined) {
    return '';
  }
  return typeof v === 'string' ? v : String(v);
};

const ExcelViewer: React.FC<{ content: string }> = ({ content }) => {
  const [sheets, setSheets] = useState<SheetData[]>([]);
  const [active, setActive] = useState(0);
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const XLSX = await import('xlsx');
        const wb = XLSX.read(base64ToBytes(content), { type: 'array' });
        const out: SheetData[] = wb.SheetNames.map(name => {
          const raw = XLSX.utils.sheet_to_json<unknown[]>(wb.Sheets[name], {
            header: 1,
            blankrows: false,
            defval: '',
          });
          const maxCols = raw.reduce((m, r) => Math.max(m, r.length), 0);
          const rows = raw.map(r => {
            const cells = Array.from({ length: maxCols }, (_, i) =>
              cellToText(r[i]),
            );
            return cells;
          });
          return { name, rows };
        });
        if (alive) {
          setSheets(out);
        }
      } catch (e) {
        if (alive) {
          setError('解析 Excel 失败');
        }
      }
    })();
    return () => {
      alive = false;
    };
  }, [content]);

  if (error) {
    return <Centered text={error} />;
  }
  if (!sheets.length) {
    return <Loading />;
  }
  const current = sheets[active];
  return (
    <div className="flex flex-col h-full">
      {sheets.length > 1 ? (
        <div className="flex items-center gap-[4px] px-[10px] py-[6px] border-b coz-stroke-primary overflow-x-auto shrink-0">
          {sheets.map((s, i) => (
            <div
              key={s.name}
              onClick={() => setActive(i)}
              className={`cursor-pointer px-[10px] py-[4px] rounded-[6px] text-[12px] whitespace-nowrap ${
                i === active
                  ? 'coz-mg-hglt coz-fg-hglt font-medium'
                  : 'coz-fg-secondary hover:coz-mg-secondary'
              }`}
            >
              {s.name}
            </div>
          ))}
        </div>
      ) : null}
      <div className="flex-1 min-h-0 overflow-auto p-[12px] coz-office-sheet">
        {current && current.rows.length ? (
          <table>
            <tbody>
              {current.rows.map((row, ri) => (
                // eslint-disable-next-line react/no-array-index-key
                <tr key={ri}>
                  {row.map((cell, ci) => (
                    // eslint-disable-next-line react/no-array-index-key
                    <td key={ci}>{cell}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <Centered text="空白表格" />
        )}
      </div>
    </div>
  );
};

// ---- Word ----
const WordViewer: React.FC<{ content: string }> = ({ content }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<'loading' | 'done' | 'error'>('loading');

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const { renderAsync } = await import('docx-preview');
        const blob = new Blob([base64ToBytes(content)], {
          type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        });
        if (!ref.current) {
          return;
        }
        ref.current.innerHTML = '';
        await renderAsync(blob, ref.current, undefined, {
          className: 'docx',
          inWrapper: true,
          ignoreWidth: true,
          ignoreHeight: true,
          experimental: true,
        });
        if (alive) {
          setState('done');
        }
      } catch (e) {
        if (alive) {
          setState('error');
        }
      }
    })();
    return () => {
      alive = false;
    };
  }, [content]);

  return (
    <div className="h-full overflow-auto coz-bg-secondary">
      {state === 'loading' ? <Loading /> : null}
      {state === 'error' ? <Centered text="解析 Word 失败" /> : null}
      <div ref={ref} className="coz-office-docx flex justify-center py-[16px]" />
    </div>
  );
};

// ---- PowerPoint ----
// 用 @aiden0z/pptx-renderer:浏览器原生高保真渲染(HTML/SVG DOM,支持图片/渐变/
// 图表/SmartArt/中文),JSZip3,纯前端打包进产物,服务器无需联网、无需任何转换服务。
interface PptxViewerInstance {
  destroy?: () => void;
}

const PptViewer: React.FC<{ content: string }> = ({ content }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<'loading' | 'done' | 'error'>('loading');

  useEffect(() => {
    let alive = true;
    let viewer: PptxViewerInstance | undefined;
    (async () => {
      try {
        const { PptxViewer } = await import('@aiden0z/pptx-renderer');
        if (!ref.current) {
          return;
        }
        ref.current.innerHTML = '';
        // PreviewInput 支持 Uint8Array;fitMode:'contain' 自动等比适配容器宽度。
        viewer = (await PptxViewer.open(
          base64ToBytes(content),
          ref.current,
          { renderMode: 'list', fitMode: 'contain' },
        )) as unknown as PptxViewerInstance;
        if (alive) {
          setState('done');
        }
      } catch (e) {
        if (alive) {
          setState('error');
        }
      }
    })();
    return () => {
      alive = false;
      try {
        viewer?.destroy?.();
      } catch {
        /* noop */
      }
    };
  }, [content]);

  return (
    <div className="h-full overflow-y-auto overflow-x-hidden coz-bg-secondary py-[12px]">
      {state === 'loading' ? <Loading /> : null}
      {state === 'error' ? (
        <Centered text="无法在线渲染此 PPT,请点右上角「下载」查看" />
      ) : null}
      <div ref={ref} className="coz-office-pptx" />
    </div>
  );
};

const Loading: React.FC = () => (
  <div className="flex justify-center py-[40px]">
    <Spin />
  </div>
);

const Centered: React.FC<{ text: string }> = ({ text }) => (
  <div className="flex items-center justify-center h-full coz-fg-secondary text-[13px]">
    {text}
  </div>
);

export const OfficeViewer: React.FC<OfficeViewerProps> = ({ ext, content }) => {
  if (['xlsx', 'xls'].includes(ext)) {
    return <ExcelViewer content={content} />;
  }
  if (['docx', 'doc'].includes(ext)) {
    return <WordViewer content={content} />;
  }
  if (['pptx', 'ppt'].includes(ext)) {
    return <PptViewer content={content} />;
  }
  return <Centered text="暂不支持该格式预览" />;
};
