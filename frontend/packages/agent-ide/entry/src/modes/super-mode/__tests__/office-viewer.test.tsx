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

import { describe, expect, it, vi } from 'vitest';
import { render, waitFor } from '@testing-library/react';
import * as XLSX from 'xlsx';

vi.mock('@coze-arch/coze-design', () => ({
  Spin: () => <div>loading</div>,
}));

import { OfficeViewer } from '../office-viewer';

const bytesToBase64 = (bytes: Uint8Array): string => {
  let bin = '';
  for (let i = 0; i < bytes.length; i++) {
    bin += String.fromCharCode(bytes[i]);
  }
  return btoa(bin);
};

// 构造一个含 XSS payload 的 xlsx(模拟沙箱里不可信文件)
const makeXlsxBase64 = (rows: string[][]): string => {
  const ws = XLSX.utils.aoa_to_sheet(rows);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'Sheet1');
  const out = XLSX.write(wb, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;
  return bytesToBase64(new Uint8Array(out));
};

describe('OfficeViewer · Excel', () => {
  const PAYLOAD = '<img src=x onerror=alert(1)>';

  it('renders cell values as text and does NOT inject HTML (XSS-safe)', async () => {
    const content = makeXlsxBase64([
      ['名称', '值'],
      [PAYLOAD, '123'],
    ]);
    const { container } = render(<OfficeViewer ext="xlsx" content={content} />);

    // 表格渲染出来,文本原样可见
    await waitFor(() => {
      expect(container.querySelector('table')).toBeTruthy();
    });
    expect(container.textContent).toContain(PAYLOAD);
    expect(container.textContent).toContain('名称');
    expect(container.textContent).toContain('123');

    // 关键:payload 不能被解析为真实 DOM —— 不存在被注入的 <img>
    expect(container.querySelector('img')).toBeNull();
    // 单元格里的尖括号是文本,不是元素
    const cells = [...container.querySelectorAll('td')].map(td => td.textContent);
    expect(cells).toContain(PAYLOAD);
  });
});
