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
/**
 * xlsx-lite —— 零依赖的极简 .xlsx 读写(单工作表 string[][])。
 *
 * 读取用浏览器内置 `DecompressionStream('deflate-raw')` 解 ZIP 内的 DEFLATE 数据,
 * 写入用手写的 STORED(不压缩)ZIP。因此不给 monorepo 引入任何新依赖,也适配银行离线环境。
 * 面向 Chrome / 现代浏览器(DecompressionStream 需 Chrome 103+)。
 */

/* eslint-disable @typescript-eslint/no-magic-numbers -- ZIP/CRC32/xlsx 二进制格式的固定字节偏移与常量，逐一具名收益低 */

const ZIP_LOCAL_SIG = 0x04034b50;
const ZIP_CENTRAL_SIG = 0x02014b50;
const ZIP_EOCD_SIG = 0x06054b50;

const CRC_TABLE = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) {
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    }
    t[n] = c >>> 0;
  }
  return t;
})();

function crc32(bytes: Uint8Array): number {
  let c = 0xffffffff;
  for (let i = 0; i < bytes.length; i++) {
    c = CRC_TABLE[(c ^ bytes[i]) & 0xff] ^ (c >>> 8);
  }
  return (c ^ 0xffffffff) >>> 0;
}

async function inflateRaw(data: Uint8Array): Promise<Uint8Array> {
  if (typeof DecompressionStream === 'undefined') {
    throw new Error(
      '当前浏览器不支持解析 Excel(需要 DecompressionStream),请改用 CSV',
    );
  }
  const ds = new DecompressionStream('deflate-raw');
  const stream = new Blob([data]).stream().pipeThrough(ds);
  const buf = await new Response(stream).arrayBuffer();
  return new Uint8Array(buf);
}

/** 解析 ZIP,返回 文件名 → 原始字节。仅支持 STORED(0)与 DEFLATE(8)。 */
async function unzip(bytes: Uint8Array): Promise<Record<string, Uint8Array>> {
  const dv = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let eocd = -1;
  for (let i = bytes.length - 22; i >= 0; i--) {
    if (dv.getUint32(i, true) === ZIP_EOCD_SIG) {
      eocd = i;
      break;
    }
  }
  if (eocd < 0) {
    throw new Error('无效的 Excel 文件(未找到 ZIP 结构)');
  }
  const cdCount = dv.getUint16(eocd + 10, true);
  let cdOff = dv.getUint32(eocd + 16, true);
  const out: Record<string, Uint8Array> = {};
  const dec = new TextDecoder();
  for (let n = 0; n < cdCount; n++) {
    if (dv.getUint32(cdOff, true) !== ZIP_CENTRAL_SIG) {
      break;
    }
    const method = dv.getUint16(cdOff + 10, true);
    const compSize = dv.getUint32(cdOff + 20, true);
    const fnLen = dv.getUint16(cdOff + 28, true);
    const extraLen = dv.getUint16(cdOff + 30, true);
    const commentLen = dv.getUint16(cdOff + 32, true);
    const localOff = dv.getUint32(cdOff + 42, true);
    const name = dec.decode(bytes.subarray(cdOff + 46, cdOff + 46 + fnLen));
    const lhFnLen = dv.getUint16(localOff + 26, true);
    const lhExtraLen = dv.getUint16(localOff + 28, true);
    const dataStart = localOff + 30 + lhFnLen + lhExtraLen;
    const comp = bytes.subarray(dataStart, dataStart + compSize);

    out[name] = method === 0 ? comp : await inflateRaw(comp);
    cdOff += 46 + fnLen + extraLen + commentLen;
  }
  return out;
}

function colToIndex(ref: string): number {
  const m = /^([A-Z]+)/.exec(ref);
  if (!m) {
    return -1;
  }
  let n = 0;
  for (const ch of m[1]) {
    n = n * 26 + (ch.charCodeAt(0) - 64);
  }
  return n - 1;
}

/** 读取 .xlsx 首个工作表为二维字符串数组(含表头行)。 */
export async function parseXlsx(buf: ArrayBuffer): Promise<string[][]> {
  const files = await unzip(new Uint8Array(buf));
  const dec = new TextDecoder();
  const parser = new DOMParser();

  const shared: string[] = [];
  const ssRaw = files['xl/sharedStrings.xml'];
  if (ssRaw) {
    const doc = parser.parseFromString(dec.decode(ssRaw), 'application/xml');
    doc.querySelectorAll('si').forEach(si => {
      let s = '';
      si.querySelectorAll('t').forEach(t => {
        s += t.textContent || '';
      });
      shared.push(s);
    });
  }

  let sheetKey = 'xl/worksheets/sheet1.xml';
  if (!files[sheetKey]) {
    sheetKey =
      Object.keys(files).find(k => /^xl\/worksheets\/.*\.xml$/.test(k)) ||
      sheetKey;
  }
  const sheetRaw = files[sheetKey];
  if (!sheetRaw) {
    return [];
  }
  const doc = parser.parseFromString(dec.decode(sheetRaw), 'application/xml');
  const rows: string[][] = [];
  doc.querySelectorAll('sheetData > row').forEach(rowEl => {
    const cells: string[] = [];
    rowEl.querySelectorAll('c').forEach(c => {
      const ref = c.getAttribute('r') || '';
      const colIdx = colToIndex(ref);
      const t = c.getAttribute('t');
      let val = '';
      if (t === 's') {
        const idx = parseInt(c.querySelector('v')?.textContent || '0', 10);
        val = shared[idx] ?? '';
      } else if (t === 'inlineStr') {
        val =
          c.querySelector('is > t')?.textContent ??
          c.querySelector('t')?.textContent ??
          '';
      } else {
        val = c.querySelector('v')?.textContent ?? '';
      }
      if (colIdx >= 0) {
        cells[colIdx] = val;
      }
    });
    for (let i = 0; i < cells.length; i++) {
      if (cells[i] === undefined) {
        cells[i] = '';
      }
    }
    rows.push(cells);
  });
  return rows;
}

function xmlEscape(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function colName(i: number): string {
  let s = '';
  let n = i + 1;
  while (n > 0) {
    const m = (n - 1) % 26;
    s = String.fromCharCode(65 + m) + s;
    n = Math.floor((n - 1) / 26);
  }
  return s;
}

function sheetXml(rows: string[][]): string {
  const body = rows
    .map((row, r) => {
      const cells = row
        .map(
          (val, c) =>
            `<c r="${colName(c)}${r + 1}" t="inlineStr"><is><t xml:space="preserve">${xmlEscape(
              val ?? '',
            )}</t></is></c>`,
        )
        .join('');
      return `<row r="${r + 1}">${cells}</row>`;
    })
    .join('');
  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${body}</sheetData></worksheet>`;
}

function zipStored(files: Record<string, string>): Blob {
  const enc = new TextEncoder();
  const chunks: Uint8Array[] = [];
  const central: Uint8Array[] = [];
  let offset = 0;
  for (const [name, content] of Object.entries(files)) {
    const nameBytes = enc.encode(name);
    const data = enc.encode(content);
    const crc = crc32(data);

    const lh = new Uint8Array(30 + nameBytes.length);
    const ldv = new DataView(lh.buffer);
    ldv.setUint32(0, ZIP_LOCAL_SIG, true);
    ldv.setUint16(4, 20, true);
    ldv.setUint32(14, crc, true);
    ldv.setUint32(18, data.length, true);
    ldv.setUint32(22, data.length, true);
    ldv.setUint16(26, nameBytes.length, true);
    lh.set(nameBytes, 30);
    chunks.push(lh, data);

    const ch = new Uint8Array(46 + nameBytes.length);
    const cdv = new DataView(ch.buffer);
    cdv.setUint32(0, ZIP_CENTRAL_SIG, true);
    cdv.setUint16(4, 20, true);
    cdv.setUint16(6, 20, true);
    cdv.setUint32(16, crc, true);
    cdv.setUint32(20, data.length, true);
    cdv.setUint32(24, data.length, true);
    cdv.setUint16(28, nameBytes.length, true);
    cdv.setUint32(42, offset, true);
    ch.set(nameBytes, 46);
    central.push(ch);

    offset += lh.length + data.length;
  }
  const cdStart = offset;
  let cdSize = 0;
  for (const c of central) {
    chunks.push(c);
    cdSize += c.length;
  }
  const eocd = new Uint8Array(22);
  const edv = new DataView(eocd.buffer);
  edv.setUint32(0, ZIP_EOCD_SIG, true);
  edv.setUint16(8, central.length, true);
  edv.setUint16(10, central.length, true);
  edv.setUint32(12, cdSize, true);
  edv.setUint32(16, cdStart, true);
  chunks.push(eocd);
  return new Blob(chunks as BlobPart[], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  });
}

/** 用二维字符串数组构建单工作表 .xlsx(STORED,不压缩),返回可下载 Blob。 */
export function buildXlsxBlob(rows: string[][]): Blob {
  const files: Record<string, string> = {
    '[Content_Types].xml':
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>',
    '_rels/.rels':
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>',
    'xl/workbook.xml':
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Sheet1" sheetId="1" r:id="rId1"/></sheets></workbook>',
    'xl/_rels/workbook.xml.rels':
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>',
    'xl/worksheets/sheet1.xml': sheetXml(rows),
  };
  return zipStored(files);
}
