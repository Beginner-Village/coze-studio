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

// 空间类型枚举
export enum SpaceType {
  Personal = 1,
  Team = 2,
}

// 空间状态枚举
export enum SpaceStatus {
  Active = 1,
  Inactive = 2,
  Archived = 3,
}

// 成员角色枚举
export enum MemberRoleType {
  Owner = 1,
  Admin = 2,
  Member = 3,
}

export interface SpaceInfo {
  space_id: number;
  name: string;
  description?: string;
  icon_url?: string;
  space_type: SpaceType;
  status: SpaceStatus;
  owner_id: number;
  creator_id: number;
  created_at: number;
  updated_at?: number;
  member_count?: number;
  current_user_role?: MemberRoleType;
}

export interface SpaceMemberInfo {
  user_id: number;
  username: string;
  nickname?: string;
  avatar_url?: string;
  role: MemberRoleType;
  joined_at: number;
  last_active_at?: number;
}

// 导入预览数据
export interface ImportPreviewData {
  import_token: string;
  manifest: {
    version: string;
    source_space_name: string;
    exported_at: number;
    statistics: {
      agents: number;
      plugins: number;
      workflows: number;
      variables: number;
    };
  };
  warnings: string[];
  token_expires_at: number;
}

// 辅助函数：获取空间类型显示文本
export const getSpaceTypeText = (type: SpaceType): string => {
  switch (type) {
    case SpaceType.Personal:
      return '个人空间';
    case SpaceType.Team:
      return '团队空间';
    default:
      return '未知';
  }
};

// 辅助函数：获取空间状态显示文本
export const getSpaceStatusText = (status: SpaceStatus): string => {
  switch (status) {
    case SpaceStatus.Active:
      return '活跃';
    case SpaceStatus.Inactive:
      return '不活跃';
    case SpaceStatus.Archived:
      return '已归档';
    default:
      return '未知';
  }
};

// 辅助函数：获取角色显示文本
export const getRoleText = (role: MemberRoleType): string => {
  switch (role) {
    case MemberRoleType.Owner:
      return '拥有者';
    case MemberRoleType.Admin:
      return '管理员';
    case MemberRoleType.Member:
      return '成员';
    default:
      return '未知';
  }
};
