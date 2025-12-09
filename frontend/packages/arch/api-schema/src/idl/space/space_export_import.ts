/*
 * Copyright 2025 coze-dev Authors
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

import { createAPI } from './../../api/config';

/** Export statistics */
export interface ExportStatistics {
  agents: number;
  plugins: number;
  workflows: number;
  variables: number;
}

/** Export data */
export interface ExportData {
  download_url: string;
  expires_at: number;
  file_name: string;
  file_size: number;
  statistics: ExportStatistics;
}

/** Export space request */
export interface ExportSpaceRequest {
  space_id: string;
}

/** Export space response */
export interface ExportSpaceResponse {
  code: number;
  msg: string;
  data: ExportData;
}

/** Import manifest */
export interface ImportManifest {
  version: string;
  source_space_name: string;
  exported_at: number;
  statistics: ExportStatistics;
}

/** Import preview data */
export interface ImportPreviewData {
  import_token: string;
  manifest: ImportManifest;
  warnings: string[];
  token_expires_at: number;
}

/** Import preview request */
export interface ImportPreviewRequest {
  space_id: string;
  file_content: string; // base64 encoded file content
}

/** Import preview response */
export interface ImportPreviewResponse {
  code: number;
  msg: string;
  data: ImportPreviewData;
}

/** Import result statistics */
export interface ImportResultStatistics {
  agents_created: number;
  plugins_created: number;
  workflows_created: number;
  variables_created: number;
}

/** Resource reference */
export interface ResourceReference {
  id: string;
  name: string;
  resource_type: string;
}

/** Import confirm data */
export interface ImportConfirmData {
  statistics: ImportResultStatistics;
  created_resources: ResourceReference[];
}

/** Import confirm request */
export interface ImportConfirmRequest {
  space_id: string;
  import_token: string;
}

/** Import confirm response */
export interface ImportConfirmResponse {
  code: number;
  msg: string;
  data: ImportConfirmData;
}

/** Export a space to ZIP package */
export const ExportSpace = /*#__PURE__*/ createAPI<
  ExportSpaceRequest,
  ExportSpaceResponse
>({
  url: '/api/space/{space_id}/export',
  method: 'POST',
  name: 'ExportSpace',
  reqType: 'ExportSpaceRequest',
  reqMapping: {
    path: ['space_id'],
  },
  resType: 'ExportSpaceResponse',
  schemaRoot: 'api://schemas/idl_space_space_export_import',
  service: 'space_export_import',
});

/** Preview import operation */
export const ImportPreview = /*#__PURE__*/ createAPI<
  ImportPreviewRequest,
  ImportPreviewResponse
>({
  url: '/api/space/{space_id}/import/preview',
  method: 'POST',
  name: 'ImportPreview',
  reqType: 'ImportPreviewRequest',
  reqMapping: {
    path: ['space_id'],
    body: ['file_content'],
  },
  resType: 'ImportPreviewResponse',
  schemaRoot: 'api://schemas/idl_space_space_export_import',
  service: 'space_export_import',
});

/** Confirm and execute import */
export const ImportConfirm = /*#__PURE__*/ createAPI<
  ImportConfirmRequest,
  ImportConfirmResponse
>({
  url: '/api/space/{space_id}/import/confirm',
  method: 'POST',
  name: 'ImportConfirm',
  reqType: 'ImportConfirmRequest',
  reqMapping: {
    path: ['space_id'],
    body: ['import_token'],
  },
  resType: 'ImportConfirmResponse',
  schemaRoot: 'api://schemas/idl_space_space_export_import',
  service: 'space_export_import',
});
