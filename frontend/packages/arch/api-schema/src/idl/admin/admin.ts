import * as base from './../base';
export { base };
import { createAPI } from './../../api/config';
/** ==================== 管理员管理 ==================== */
export interface AdminUserInfo {
  user_id: string,
  username: string,
  avatar_url: string,
  /** super_admin / admin */
  role: string,
  created_at: number,
}
export interface CheckAdminInitRequest {}
export interface CheckAdminInitResponse {
  /** 系统是否已初始化 */
  initialized: boolean,
  /** 当前用户是否是管理员 */
  is_admin?: boolean,
  /** 当前用户角色 */
  role?: string,
  code: number,
  msg: string,
}
export interface InitAdminRequest {
  /** 可选，不传则使用当前登录用户 */
  user_id?: string
}
export interface InitAdminResponse {
  code: number,
  msg: string,
}
export interface ListAdminUsersRequest {}
export interface ListAdminUsersResponse {
  items: AdminUserInfo[],
  code: number,
  msg: string,
}
export interface AddAdminRequest {
  user_id: string,
  /** 默认 admin */
  role?: string,
}
export interface AddAdminResponse {
  code: number,
  msg: string,
}
export interface RemoveAdminRequest {
  user_id: string
}
export interface RemoveAdminResponse {
  code: number,
  msg: string,
}
/** ==================== 公共模型管理 ==================== */
export interface PublicModelItem {
  id: string,
  name: string,
  description?: string,
  /** llm / embedding / rerank / tts */
  model_type: string,
  protocol: string,
  icon_url?: string,
  icon_uri?: string,
  created_at: number,
  /** 1=启用 2=禁用 */
  status: number,
  context_length?: number,
}
export interface ListPublicModelsRequest {
  /** 按类型过滤 */
  model_type?: string,
  /** 搜索关键词 */
  keyword?: string,
}
export interface ListPublicModelsResponse {
  items: PublicModelItem[],
  code: number,
  msg: string,
}
export interface CreatePublicModelRequest {
  name: string,
  description?: string,
  /** llm / embedding */
  model_type: string,
  protocol: string,
  base_url: string,
  api_key: string,
  model: string,
  icon_uri?: string,
  input_tokens?: number,
  output_tokens?: number,
  function_call?: boolean,
  json_mode?: boolean,
  reasoning?: boolean,
  enable_thinking?: boolean,
}
export interface CreatePublicModelResponse {
  data?: PublicModelItem,
  code: number,
  msg: string,
}
export interface GetPublicModelRequest {
  model_id: string
}
export interface GetPublicModelResponse {
  data?: PublicModelDetailOutput,
  code: number,
  msg: string,
}
export interface PublicModelDetailOutput {
  id: string,
  name: string,
  description?: string,
  model_type: string,
  protocol: string,
  base_url: string,
  api_key: string,
  model: string,
  icon_uri?: string,
  icon_url?: string,
  input_tokens?: number,
  output_tokens?: number,
  function_call?: boolean,
  json_mode?: boolean,
  reasoning?: boolean,
  enable_thinking?: boolean,
  created_at: number,
  status: number,
}
export interface UpdatePublicModelRequest {
  model_id: string,
  name?: string,
  description?: string,
  base_url?: string,
  api_key?: string,
  model?: string,
  icon_uri?: string,
  input_tokens?: number,
  output_tokens?: number,
  function_call?: boolean,
  json_mode?: boolean,
  reasoning?: boolean,
  enable_thinking?: boolean,
}
export interface UpdatePublicModelResponse {
  code: number,
  msg: string,
}
export interface DeletePublicModelRequest {
  model_id: string
}
export interface DeletePublicModelResponse {
  code: number,
  msg: string,
}
export interface EnablePublicModelRequest {
  model_id: string
}
export interface EnablePublicModelResponse {
  code: number,
  msg: string,
}
export interface DisablePublicModelRequest {
  model_id: string
}
export interface DisablePublicModelResponse {
  code: number,
  msg: string,
}
/** 初始化检查(无需管理员权限) */
export const CheckAdminInit = /*#__PURE__*/createAPI<CheckAdminInitRequest, CheckAdminInitResponse>({
  "url": "/api/admin/check",
  "method": "GET",
  "name": "CheckAdminInit",
  "reqType": "CheckAdminInitRequest",
  "reqMapping": {},
  "resType": "CheckAdminInitResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
/** 首次初始化超级管理员(仅在未初始化时可用) */
export const InitAdmin = /*#__PURE__*/createAPI<InitAdminRequest, InitAdminResponse>({
  "url": "/api/admin/init",
  "method": "POST",
  "name": "InitAdmin",
  "reqType": "InitAdminRequest",
  "reqMapping": {
    "body": ["user_id"]
  },
  "resType": "InitAdminResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
/** 管理员列表(需要管理员权限) */
export const ListAdminUsers = /*#__PURE__*/createAPI<ListAdminUsersRequest, ListAdminUsersResponse>({
  "url": "/api/admin/users",
  "method": "GET",
  "name": "ListAdminUsers",
  "reqType": "ListAdminUsersRequest",
  "reqMapping": {},
  "resType": "ListAdminUsersResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
/** 添加管理员(需要超级管理员权限) */
export const AddAdmin = /*#__PURE__*/createAPI<AddAdminRequest, AddAdminResponse>({
  "url": "/api/admin/users/add",
  "method": "POST",
  "name": "AddAdmin",
  "reqType": "AddAdminRequest",
  "reqMapping": {
    "body": ["user_id", "role"]
  },
  "resType": "AddAdminResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
/** 移除管理员(需要超级管理员权限) */
export const RemoveAdmin = /*#__PURE__*/createAPI<RemoveAdminRequest, RemoveAdminResponse>({
  "url": "/api/admin/users/remove",
  "method": "POST",
  "name": "RemoveAdmin",
  "reqType": "RemoveAdminRequest",
  "reqMapping": {
    "body": ["user_id"]
  },
  "resType": "RemoveAdminResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
/** 公共模型 CRUD(需要管理员权限) */
export const ListPublicModels = /*#__PURE__*/createAPI<ListPublicModelsRequest, ListPublicModelsResponse>({
  "url": "/api/admin/models",
  "method": "GET",
  "name": "ListPublicModels",
  "reqType": "ListPublicModelsRequest",
  "reqMapping": {
    "query": ["model_type", "keyword"]
  },
  "resType": "ListPublicModelsResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
export const GetPublicModel = /*#__PURE__*/createAPI<GetPublicModelRequest, GetPublicModelResponse>({
  "url": "/api/admin/models/:model_id",
  "method": "GET",
  "name": "GetPublicModel",
  "reqType": "GetPublicModelRequest",
  "reqMapping": {
    "path": ["model_id"]
  },
  "resType": "GetPublicModelResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
export const CreatePublicModel = /*#__PURE__*/createAPI<CreatePublicModelRequest, CreatePublicModelResponse>({
  "url": "/api/admin/models/create",
  "method": "POST",
  "name": "CreatePublicModel",
  "reqType": "CreatePublicModelRequest",
  "reqMapping": {
    "body": ["name", "description", "model_type", "protocol", "base_url", "api_key", "model", "icon_uri", "input_tokens", "output_tokens", "function_call", "json_mode", "reasoning", "enable_thinking"]
  },
  "resType": "CreatePublicModelResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
export const UpdatePublicModel = /*#__PURE__*/createAPI<UpdatePublicModelRequest, UpdatePublicModelResponse>({
  "url": "/api/admin/models/:model_id/update",
  "method": "POST",
  "name": "UpdatePublicModel",
  "reqType": "UpdatePublicModelRequest",
  "reqMapping": {
    "path": ["model_id"],
    "body": ["name", "description", "base_url", "api_key", "model", "icon_uri", "input_tokens", "output_tokens", "function_call", "json_mode", "reasoning", "enable_thinking"]
  },
  "resType": "UpdatePublicModelResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
export const DeletePublicModel = /*#__PURE__*/createAPI<DeletePublicModelRequest, DeletePublicModelResponse>({
  "url": "/api/admin/models/delete",
  "method": "POST",
  "name": "DeletePublicModel",
  "reqType": "DeletePublicModelRequest",
  "reqMapping": {
    "body": ["model_id"]
  },
  "resType": "DeletePublicModelResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
/** 启用/禁用公共模型 */
export const EnablePublicModel = /*#__PURE__*/createAPI<EnablePublicModelRequest, EnablePublicModelResponse>({
  "url": "/api/admin/models/enable",
  "method": "POST",
  "name": "EnablePublicModel",
  "reqType": "EnablePublicModelRequest",
  "reqMapping": {
    "body": ["model_id"]
  },
  "resType": "EnablePublicModelResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});
export const DisablePublicModel = /*#__PURE__*/createAPI<DisablePublicModelRequest, DisablePublicModelResponse>({
  "url": "/api/admin/models/disable",
  "method": "POST",
  "name": "DisablePublicModel",
  "reqType": "DisablePublicModelRequest",
  "reqMapping": {
    "body": ["model_id"]
  },
  "resType": "DisablePublicModelResponse",
  "schemaRoot": "api://schemas/idl_admin_admin",
  "service": "admin"
});