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
import { describe, it, expect } from 'vitest';

import { buildXlsxBlob, parseXlsx } from '../xlsx-lite';

async function roundTrip(rows: string[][]): Promise<string[][]> {
  const blob = buildXlsxBlob(rows);
  const buf = await blob.arrayBuffer();
  return parseXlsx(buf);
}

describe('xlsx-lite build/parse round-trip', () => {
  it('产出合法 ZIP(PK 头)', async () => {
    const blob = buildXlsxBlob([['input'], ['a']]);
    const head = new Uint8Array(await blob.arrayBuffer()).subarray(0, 4);
    expect(Array.from(head)).toEqual([0x50, 0x4b, 0x03, 0x04]);
  });

  it('单列 input + 示例行往返一致', async () => {
    const rows = [['input'], ['我有哪些银行卡'], ['查询我的余额']];
    const parsed = await roundTrip(rows);
    expect(parsed).toEqual(rows);
  });

  it('多列 + 特殊字符(逗号/引号/尖括号/&/unicode)保真', async () => {
    const rows = [
      ['input', 'note'],
      ['a,b "quoted"', '<tag> & 中文😀'],
      ['line', ''],
    ];
    const parsed = await roundTrip(rows);
    expect(parsed[0]).toEqual(['input', 'note']);
    expect(parsed[1]).toEqual(['a,b "quoted"', '<tag> & 中文😀']);
    // 空单元格保留为占位,首列仍可取
    expect(parsed[2][0]).toBe('line');
  });

  it('空值/前导列缺失时按列索引对齐', async () => {
    const rows = [['input'], [''], ['x']];
    const parsed = await roundTrip(rows);
    expect(parsed.map(r => r[0])).toEqual(['input', '', 'x']);
  });
});
