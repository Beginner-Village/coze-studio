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

import { createAPI } from './../../api/config';

export interface SkillMetadata {
  version?: string;
  category?: string;
  tags?: string[];
  platforms?: string[];
}

export interface SkillAssetSummary {
  count: number;
  image_paths: string[];
}

/** Skill information */
export interface SkillInfo {
  skill_id: string;
  space_id: string;
  name: string;
  description: string;
  prompt: string;
  files?: Record<string, string>;
  metadata?: SkillMetadata;
  asset_summary?: SkillAssetSummary;
  icon_uri: string;
  creator_id: string;
  version?: number;
  publish_scope?: number;
  published_version?: number;
  published_at?: number;
  published_by?: string;
  created_at: number;
  updated_at: number;
}

/** Lightweight reference for Bot binding */
export interface SkillReference {
  skill_id: string;
  skill_name: string;
  skill_description: string;
}

// Create Skill
export interface CreateSkillRequest {
  space_id: string;
  name: string;
  description?: string;
  prompt?: string;
  files?: Record<string, string>;
  icon_uri?: string;
}

export interface CreateSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

export interface ImportSkillPackageRequest {
  space_id: string;
  filename?: string;
  content: string;
  icon_uri?: string;
}

export interface ImportSkillPackageResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

export interface ImportRuntimeSkillRequest {
  agent_id: string;
  bot_id?: string;
  connector_id?: string;
  name: string;
  skill_id?: string;
  icon_uri?: string;
  publish_scope?: number;
}

export interface ImportRuntimeSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

export interface ValidateSkillPackageRequest {
  filename?: string;
  content: string;
}

export interface SkillPackageValidation {
  valid: boolean;
  error?: string;
  filename: string;
  name?: string;
  description?: string;
  metadata?: SkillMetadata;
  file_count: number;
  file_paths: string[];
  asset_paths: string[];
  image_paths: string[];
}

export interface ValidateSkillPackageResponse {
  code: number;
  msg: string;
  data: {
    validation: SkillPackageValidation;
  };
}

export interface ExportSkillPackageRequest {
  space_id: string;
  skill_id: string;
}

export interface SkillPackageExport {
  filename: string;
  content: string;
  size: number;
  file_paths: string[];
}

export interface ExportSkillPackageResponse {
  code: number;
  msg: string;
  data: {
    package: SkillPackageExport;
  };
}

// Get Skill
export interface GetSkillRequest {
  skill_id: string;
  space_id: string;
}

export interface GetSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

// Update Skill
export interface UpdateSkillRequest {
  skill_id: string;
  space_id: string;
  name?: string;
  description?: string;
  prompt?: string;
  files?: Record<string, string>;
  icon_uri?: string;
}

export interface UpdateSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

export interface UpsertSkillAssetRequest {
  skill_id: string;
  space_id: string;
  path: string;
  content: string;
  mime?: string;
}

export interface SkillAssetInfo {
  path: string;
  mime: string;
  size: number;
  is_image: boolean;
}

export interface SkillAssetContent extends SkillAssetInfo {
  content: string;
}

export interface ListSkillAssetsRequest {
  skill_id: string;
  space_id: string;
}

export interface ListSkillAssetsResponse {
  code: number;
  msg: string;
  data: {
    assets: SkillAssetInfo[];
  };
}

export interface GetSkillAssetRequest {
  skill_id: string;
  space_id: string;
  path: string;
}

export interface GetSkillAssetResponse {
  code: number;
  msg: string;
  data: {
    asset: SkillAssetContent;
  };
}

export interface UpsertSkillAssetResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

export interface DeleteSkillAssetRequest {
  skill_id: string;
  space_id: string;
  path: string;
}

export interface DeleteSkillAssetResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

// Delete Skill
export interface DeleteSkillRequest {
  skill_id: string;
  space_id: string;
}

export interface DeleteSkillResponse {
  code: number;
  msg: string;
}

// Publish Skill
export interface PublishSkillRequest {
  skill_id: string;
  space_id: string;
  scope: number;
}

export interface PublishSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

// Install Marketplace Skill
export interface InstallMarketplaceSkillRequest {
  skill_id: string;
  space_id: string;
}

export interface InstallMarketplaceSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
  };
}

// List Skills
export interface ListSkillsRequest {
  space_id: string;
  page?: number;
  page_size?: number;
  keyword?: string;
}

export interface ListSkillsResponse {
  code: number;
  msg: string;
  data: {
    skill_list: SkillInfo[];
    total: number;
  };
}

// List Marketplace Skills
export interface MarketplaceListSkillsRequest {
  space_id?: string;
  scope?: number;
  page?: number;
  page_size?: number;
  keyword?: string;
}

export interface MarketplaceListSkillsResponse {
  code: number;
  msg: string;
  data: {
    skill_list: SkillInfo[];
    total: number;
  };
}

/** Skill CRUD API */
export const CreateSkill = /*#__PURE__*/ createAPI<
  CreateSkillRequest,
  CreateSkillResponse
>({
  url: '/api/skill/create',
  method: 'POST',
  name: 'CreateSkill',
  reqType: 'CreateSkillRequest',
  reqMapping: {
    body: ['space_id', 'name', 'description', 'prompt', 'files', 'icon_uri'],
  },
  resType: 'CreateSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const GetSkill = /*#__PURE__*/ createAPI<
  GetSkillRequest,
  GetSkillResponse
>({
  url: '/api/skill/get',
  method: 'GET',
  name: 'GetSkill',
  reqType: 'GetSkillRequest',
  reqMapping: {
    query: ['skill_id', 'space_id'],
  },
  resType: 'GetSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const UpdateSkill = /*#__PURE__*/ createAPI<
  UpdateSkillRequest,
  UpdateSkillResponse
>({
  url: '/api/skill/update',
  method: 'POST',
  name: 'UpdateSkill',
  reqType: 'UpdateSkillRequest',
  reqMapping: {
    body: [
      'skill_id',
      'space_id',
      'name',
      'description',
      'prompt',
      'files',
      'icon_uri',
    ],
  },
  resType: 'UpdateSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const DeleteSkill = /*#__PURE__*/ createAPI<
  DeleteSkillRequest,
  DeleteSkillResponse
>({
  url: '/api/skill/delete',
  method: 'POST',
  name: 'DeleteSkill',
  reqType: 'DeleteSkillRequest',
  reqMapping: {
    body: ['skill_id', 'space_id'],
  },
  resType: 'DeleteSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const PublishSkill = /*#__PURE__*/ createAPI<
  PublishSkillRequest,
  PublishSkillResponse
>({
  url: '/api/skill/publish',
  method: 'POST',
  name: 'PublishSkill',
  reqType: 'PublishSkillRequest',
  reqMapping: {
    body: ['skill_id', 'space_id', 'scope'],
  },
  resType: 'PublishSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const InstallMarketplaceSkill = /*#__PURE__*/ createAPI<
  InstallMarketplaceSkillRequest,
  InstallMarketplaceSkillResponse
>({
  url: '/api/skill/marketplace/install',
  method: 'POST',
  name: 'InstallMarketplaceSkill',
  reqType: 'InstallMarketplaceSkillRequest',
  reqMapping: {
    body: ['skill_id', 'space_id'],
  },
  resType: 'InstallMarketplaceSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const MarketplaceGetSkill = /*#__PURE__*/ createAPI<
  GetSkillRequest,
  GetSkillResponse
>({
  url: '/api/skill/marketplace/get',
  method: 'GET',
  name: 'MarketplaceGetSkill',
  reqType: 'GetSkillRequest',
  reqMapping: {
    query: ['skill_id', 'space_id'],
  },
  resType: 'GetSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const ListSkills = /*#__PURE__*/ createAPI<
  ListSkillsRequest,
  ListSkillsResponse
>({
  url: '/api/skill/list',
  method: 'GET',
  name: 'ListSkills',
  reqType: 'ListSkillsRequest',
  reqMapping: {
    query: ['space_id', 'page', 'page_size', 'keyword'],
  },
  resType: 'ListSkillsResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const MarketplaceListSkills = /*#__PURE__*/ createAPI<
  MarketplaceListSkillsRequest,
  MarketplaceListSkillsResponse
>({
  url: '/api/skill/marketplace/list',
  method: 'GET',
  name: 'MarketplaceListSkills',
  reqType: 'MarketplaceListSkillsRequest',
  reqMapping: {
    query: ['space_id', 'scope', 'page', 'page_size', 'keyword'],
  },
  resType: 'MarketplaceListSkillsResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

/** Super Agent App Server Skill APIs */
export const SuperAgentCreateSkill = /*#__PURE__*/ createAPI<
  CreateSkillRequest,
  CreateSkillResponse
>({
  url: '/api/super-agent/skills/create',
  method: 'POST',
  name: 'SuperAgentCreateSkill',
  reqType: 'CreateSkillRequest',
  reqMapping: {
    body: ['space_id', 'name', 'description', 'prompt', 'files', 'icon_uri'],
  },
  resType: 'CreateSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentImportSkillPackage = /*#__PURE__*/ createAPI<
  ImportSkillPackageRequest,
  ImportSkillPackageResponse
>({
  url: '/api/super-agent/skills/import',
  method: 'POST',
  name: 'SuperAgentImportSkillPackage',
  reqType: 'ImportSkillPackageRequest',
  reqMapping: {
    body: ['space_id', 'filename', 'content', 'icon_uri'],
  },
  resType: 'ImportSkillPackageResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentImportRuntimeSkill = /*#__PURE__*/ createAPI<
  ImportRuntimeSkillRequest,
  ImportRuntimeSkillResponse
>({
  url: '/api/super-agent/skills/import-runtime',
  method: 'POST',
  name: 'SuperAgentImportRuntimeSkill',
  reqType: 'ImportRuntimeSkillRequest',
  reqMapping: {
    body: [
      'agent_id',
      'bot_id',
      'connector_id',
      'name',
      'skill_id',
      'icon_uri',
      'publish_scope',
    ],
  },
  resType: 'ImportRuntimeSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentValidateSkillPackage = /*#__PURE__*/ createAPI<
  ValidateSkillPackageRequest,
  ValidateSkillPackageResponse
>({
  url: '/api/super-agent/skills/validate-package',
  method: 'POST',
  name: 'SuperAgentValidateSkillPackage',
  reqType: 'ValidateSkillPackageRequest',
  reqMapping: {
    body: ['filename', 'content'],
  },
  resType: 'ValidateSkillPackageResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentExportSkillPackage = /*#__PURE__*/ createAPI<
  ExportSkillPackageRequest,
  ExportSkillPackageResponse
>({
  url: '/api/super-agent/skills/export',
  method: 'POST',
  name: 'SuperAgentExportSkillPackage',
  reqType: 'ExportSkillPackageRequest',
  reqMapping: {
    body: ['space_id', 'skill_id'],
  },
  resType: 'ExportSkillPackageResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentGetSkill = /*#__PURE__*/ createAPI<
  GetSkillRequest,
  GetSkillResponse
>({
  url: '/api/super-agent/skills/get',
  method: 'GET',
  name: 'SuperAgentGetSkill',
  reqType: 'GetSkillRequest',
  reqMapping: {
    query: ['skill_id', 'space_id'],
  },
  resType: 'GetSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentUpdateSkill = /*#__PURE__*/ createAPI<
  UpdateSkillRequest,
  UpdateSkillResponse
>({
  url: '/api/super-agent/skills/update',
  method: 'POST',
  name: 'SuperAgentUpdateSkill',
  reqType: 'UpdateSkillRequest',
  reqMapping: {
    body: [
      'skill_id',
      'space_id',
      'name',
      'description',
      'prompt',
      'files',
      'icon_uri',
    ],
  },
  resType: 'UpdateSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentListSkillAssets = /*#__PURE__*/ createAPI<
  ListSkillAssetsRequest,
  ListSkillAssetsResponse
>({
  url: '/api/super-agent/skills/assets/list',
  method: 'GET',
  name: 'SuperAgentListSkillAssets',
  reqType: 'ListSkillAssetsRequest',
  reqMapping: {
    query: ['skill_id', 'space_id'],
  },
  resType: 'ListSkillAssetsResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentGetSkillAsset = /*#__PURE__*/ createAPI<
  GetSkillAssetRequest,
  GetSkillAssetResponse
>({
  url: '/api/super-agent/skills/assets/get',
  method: 'GET',
  name: 'SuperAgentGetSkillAsset',
  reqType: 'GetSkillAssetRequest',
  reqMapping: {
    query: ['skill_id', 'space_id', 'path'],
  },
  resType: 'GetSkillAssetResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentUpsertSkillAsset = /*#__PURE__*/ createAPI<
  UpsertSkillAssetRequest,
  UpsertSkillAssetResponse
>({
  url: '/api/super-agent/skills/assets/upsert',
  method: 'POST',
  name: 'SuperAgentUpsertSkillAsset',
  reqType: 'UpsertSkillAssetRequest',
  reqMapping: {
    body: ['skill_id', 'space_id', 'path', 'content', 'mime'],
  },
  resType: 'UpsertSkillAssetResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentDeleteSkillAsset = /*#__PURE__*/ createAPI<
  DeleteSkillAssetRequest,
  DeleteSkillAssetResponse
>({
  url: '/api/super-agent/skills/assets/delete',
  method: 'POST',
  name: 'SuperAgentDeleteSkillAsset',
  reqType: 'DeleteSkillAssetRequest',
  reqMapping: {
    body: ['skill_id', 'space_id', 'path'],
  },
  resType: 'DeleteSkillAssetResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentDeleteSkill = /*#__PURE__*/ createAPI<
  DeleteSkillRequest,
  DeleteSkillResponse
>({
  url: '/api/super-agent/skills/delete',
  method: 'POST',
  name: 'SuperAgentDeleteSkill',
  reqType: 'DeleteSkillRequest',
  reqMapping: {
    body: ['skill_id', 'space_id'],
  },
  resType: 'DeleteSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentPublishSkill = /*#__PURE__*/ createAPI<
  PublishSkillRequest,
  PublishSkillResponse
>({
  url: '/api/super-agent/skills/publish',
  method: 'POST',
  name: 'SuperAgentPublishSkill',
  reqType: 'PublishSkillRequest',
  reqMapping: {
    body: ['skill_id', 'space_id', 'scope'],
  },
  resType: 'PublishSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentListSkills = /*#__PURE__*/ createAPI<
  ListSkillsRequest,
  ListSkillsResponse
>({
  url: '/api/super-agent/skills/list',
  method: 'GET',
  name: 'SuperAgentListSkills',
  reqType: 'ListSkillsRequest',
  reqMapping: {
    query: ['space_id', 'page', 'page_size', 'keyword'],
  },
  resType: 'ListSkillsResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentMarketplaceListSkills = /*#__PURE__*/ createAPI<
  MarketplaceListSkillsRequest,
  MarketplaceListSkillsResponse
>({
  url: '/api/super-agent/marketplace/list',
  method: 'GET',
  name: 'SuperAgentMarketplaceListSkills',
  reqType: 'MarketplaceListSkillsRequest',
  reqMapping: {
    query: ['space_id', 'scope', 'page', 'page_size', 'keyword'],
  },
  resType: 'MarketplaceListSkillsResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentMarketplaceGetSkill = /*#__PURE__*/ createAPI<
  GetSkillRequest,
  GetSkillResponse
>({
  url: '/api/super-agent/marketplace/get',
  method: 'GET',
  name: 'SuperAgentMarketplaceGetSkill',
  reqType: 'GetSkillRequest',
  reqMapping: {
    query: ['skill_id', 'space_id'],
  },
  resType: 'GetSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});

export const SuperAgentInstallMarketplaceSkill = /*#__PURE__*/ createAPI<
  InstallMarketplaceSkillRequest,
  InstallMarketplaceSkillResponse
>({
  url: '/api/super-agent/marketplace/install',
  method: 'POST',
  name: 'SuperAgentInstallMarketplaceSkill',
  reqType: 'InstallMarketplaceSkillRequest',
  reqMapping: {
    body: ['skill_id', 'space_id'],
  },
  resType: 'InstallMarketplaceSkillResponse',
  schemaRoot: 'api://schemas/idl_skill_skill',
  service: 'skill',
});
