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

/** Skill information */
export interface SkillInfo {
  skill_id: string;
  space_id: string;
  name: string;
  description: string;
  prompt: string;
  icon_uri: string;
  creator_id: string;
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
  icon_uri?: string;
}

export interface CreateSkillResponse {
  code: number;
  msg: string;
  data: {
    skill_info: SkillInfo;
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
  icon_uri?: string;
}

export interface UpdateSkillResponse {
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

/** Skill CRUD API */
export const CreateSkill = /*#__PURE__*/createAPI<CreateSkillRequest, CreateSkillResponse>({
  "url": "/api/skill/create",
  "method": "POST",
  "name": "CreateSkill",
  "reqType": "CreateSkillRequest",
  "reqMapping": {
    "body": ["space_id", "name", "description", "prompt", "icon_uri"]
  },
  "resType": "CreateSkillResponse",
  "schemaRoot": "api://schemas/idl_skill_skill",
  "service": "skill"
});

export const GetSkill = /*#__PURE__*/createAPI<GetSkillRequest, GetSkillResponse>({
  "url": "/api/skill/get",
  "method": "GET",
  "name": "GetSkill",
  "reqType": "GetSkillRequest",
  "reqMapping": {
    "query": ["skill_id", "space_id"]
  },
  "resType": "GetSkillResponse",
  "schemaRoot": "api://schemas/idl_skill_skill",
  "service": "skill"
});

export const UpdateSkill = /*#__PURE__*/createAPI<UpdateSkillRequest, UpdateSkillResponse>({
  "url": "/api/skill/update",
  "method": "POST",
  "name": "UpdateSkill",
  "reqType": "UpdateSkillRequest",
  "reqMapping": {
    "body": ["skill_id", "space_id", "name", "description", "prompt", "icon_uri"]
  },
  "resType": "UpdateSkillResponse",
  "schemaRoot": "api://schemas/idl_skill_skill",
  "service": "skill"
});

export const DeleteSkill = /*#__PURE__*/createAPI<DeleteSkillRequest, DeleteSkillResponse>({
  "url": "/api/skill/delete",
  "method": "POST",
  "name": "DeleteSkill",
  "reqType": "DeleteSkillRequest",
  "reqMapping": {
    "body": ["skill_id", "space_id"]
  },
  "resType": "DeleteSkillResponse",
  "schemaRoot": "api://schemas/idl_skill_skill",
  "service": "skill"
});

export const ListSkills = /*#__PURE__*/createAPI<ListSkillsRequest, ListSkillsResponse>({
  "url": "/api/skill/list",
  "method": "GET",
  "name": "ListSkills",
  "reqType": "ListSkillsRequest",
  "reqMapping": {
    "query": ["space_id", "page", "page_size", "keyword"]
  },
  "resType": "ListSkillsResponse",
  "schemaRoot": "api://schemas/idl_skill_skill",
  "service": "skill"
});
