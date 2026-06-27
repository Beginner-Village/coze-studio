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

export const getFileExtensionAndName = (fileName: string) => {
  const dotIndex = fileName.lastIndexOf('.');
  if (dotIndex < 0) {
    return {
      nameWithoutExtension: fileName,
      extension: '',
    };
  }
  /**
   * eg: .docx
   */
  const extension = fileName.slice(dotIndex);
  const nameWithoutExtension = fileName.slice(0, dotIndex);
  return {
    extension,
    nameWithoutExtension,
  };
};

const SANDBOX_FILENAME_EXTENSIONS =
  'py|js|ts|tsx|jsx|json|md|txt|csv|xlsx|xls|docx|doc|pptx|ppt|pdf|sh|go|yaml|yml|toml|lock';

const sandboxFilenameRegExp = new RegExp(
  `(^|[\\s,，、])([A-Za-z0-9_-]+\\.(?:${SANDBOX_FILENAME_EXTENSIONS}))(?![A-Za-z0-9_/.-])`,
  'g',
);

export const protectSandboxFilenames = (text: string) =>
  text.replace(sandboxFilenameRegExp, '$1`$2`');
