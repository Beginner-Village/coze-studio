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

import { it, expect } from 'vitest';

import {
  getFileExtensionAndName,
  protectSandboxFilenames,
} from '../../src/utils/file-name';

it('should get file extension by xxx.extension case', () => {
  const fileName = '《史蒂夫·乔布斯传》官方正式中文版电子书.pdf';
  const { nameWithoutExtension, extension } = getFileExtensionAndName(fileName);
  expect(extension).toBe('.pdf');
  expect(nameWithoutExtension).toBe('《史蒂夫·乔布斯传》官方正式中文版电子书');
});

it('not get file extension by xxx case', () => {
  const fileName = 'Visual Studio Code';
  const { nameWithoutExtension, extension } = getFileExtensionAndName(fileName);
  expect(extension).toBe('');
  expect(nameWithoutExtension).toBe('Visual Studio Code');
});

it('should protect sandbox filenames before markdown render', () => {
  const text = [
    'create_excel.py',
    'generate_glm52_doc.js',
    'hello.txt',
    'package-lock.json',
  ].join('\n');

  expect(protectSandboxFilenames(text)).toBe(
    [
      '`create_excel.py`',
      '`generate_glm52_doc.js`',
      '`hello.txt`',
      '`package-lock.json`',
    ].join('\n'),
  );
});

it('should keep explicit markdown links and urls unchanged', () => {
  const text =
    '[hello.txt](https://example.com/hello.txt) https://example.com/create_excel.py';

  expect(protectSandboxFilenames(text)).toBe(text);
});
