include "../base.thrift"

namespace go admin

// ==================== 管理员管理 ====================

struct AdminUserInfo {
    1: required string user_id,
    2: required string username,
    3: required string avatar_url,
    4: required string role,        // super_admin / admin
    5: required i64 created_at,
}

struct CheckAdminInitRequest {
    255: optional base.Base Base (api.none="true"),
}

struct CheckAdminInitResponse {
    1: required bool initialized,   // 系统是否已初始化
    2: optional bool is_admin,      // 当前用户是否是管理员
    3: optional string role,        // 当前用户角色
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct InitAdminRequest {
    1: optional string user_id (api.body="user_id"),  // 可选，不传则使用当前登录用户
    255: optional base.Base Base (api.none="true"),
}

struct InitAdminResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct ListAdminUsersRequest {
    255: optional base.Base Base (api.none="true"),
}

struct ListAdminUsersResponse {
    1: required list<AdminUserInfo> items,
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct AddAdminRequest {
    1: required string user_id (api.body="user_id"),
    2: optional string role (api.body="role"),  // 默认 admin
    255: optional base.Base Base (api.none="true"),
}

struct AddAdminResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct RemoveAdminRequest {
    1: required string user_id (api.body="user_id"),
    255: optional base.Base Base (api.none="true"),
}

struct RemoveAdminResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// ==================== 公共模型管理 ====================

struct PublicModelItem {
    1: required string id,
    2: required string name,
    3: optional string description,
    4: required string model_type,   // llm / embedding / rerank / tts
    5: required string protocol,
    6: optional string icon_url,
    7: optional string icon_uri,
    8: required i64 created_at,
    9: required i32 status,          // 1=启用 2=禁用
    10: optional i64 context_length,
}

struct ListPublicModelsRequest {
    1: optional string model_type (api.query="model_type"),   // 按类型过滤
    2: optional string keyword (api.query="keyword"),         // 搜索关键词
    255: optional base.Base Base (api.none="true"),
}

struct ListPublicModelsResponse {
    1: required list<PublicModelItem> items,
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct CreatePublicModelRequest {
    1: required string name (api.body="name"),
    2: optional string description (api.body="description"),
    3: required string model_type (api.body="model_type"),  // llm / embedding
    4: required string protocol (api.body="protocol"),
    5: required string base_url (api.body="base_url"),
    6: required string api_key (api.body="api_key"),
    7: required string model (api.body="model"),
    8: optional string icon_uri (api.body="icon_uri"),
    9: optional i32 input_tokens (api.body="input_tokens"),
    10: optional i32 output_tokens (api.body="output_tokens"),
    11: optional bool function_call (api.body="function_call"),
    12: optional bool json_mode (api.body="json_mode"),
    13: optional bool reasoning (api.body="reasoning"),
    14: optional bool enable_thinking (api.body="enable_thinking"),
    255: optional base.Base Base (api.none="true"),
}

struct CreatePublicModelResponse {
    1: optional PublicModelItem data,
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct GetPublicModelRequest {
    1: required string model_id (api.path="model_id"),
    255: optional base.Base Base (api.none="true"),
}

struct GetPublicModelResponse {
    1: optional PublicModelDetailOutput data,
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct PublicModelDetailOutput {
    1: required string id,
    2: required string name,
    3: optional string description,
    4: required string model_type,
    5: required string protocol,
    6: required string base_url,
    7: required string api_key,
    8: required string model,
    9: optional string icon_uri,
    10: optional string icon_url,
    11: optional i32 input_tokens,
    12: optional i32 output_tokens,
    13: optional bool function_call,
    14: optional bool json_mode,
    15: optional bool reasoning,
    16: optional bool enable_thinking,
    17: required i64 created_at,
    18: required i32 status,
}

struct UpdatePublicModelRequest {
    1: required string model_id (api.path="model_id"),
    2: optional string name (api.body="name"),
    3: optional string description (api.body="description"),
    4: optional string base_url (api.body="base_url"),
    5: optional string api_key (api.body="api_key"),
    6: optional string model (api.body="model"),
    7: optional string icon_uri (api.body="icon_uri"),
    8: optional i32 input_tokens (api.body="input_tokens"),
    9: optional i32 output_tokens (api.body="output_tokens"),
    10: optional bool function_call (api.body="function_call"),
    11: optional bool json_mode (api.body="json_mode"),
    12: optional bool reasoning (api.body="reasoning"),
    13: optional bool enable_thinking (api.body="enable_thinking"),
    255: optional base.Base Base (api.none="true"),
}

struct UpdatePublicModelResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct DeletePublicModelRequest {
    1: required string model_id (api.body="model_id"),
    255: optional base.Base Base (api.none="true"),
}

struct DeletePublicModelResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct EnablePublicModelRequest {
    1: required string model_id (api.body="model_id"),
    255: optional base.Base Base (api.none="true"),
}

struct EnablePublicModelResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct DisablePublicModelRequest {
    1: required string model_id (api.body="model_id"),
    255: optional base.Base Base (api.none="true"),
}

struct DisablePublicModelResponse {
    253: required i32 code,
    254: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// ==================== 服务定义 ====================

service AdminService {
    // 初始化检查(无需管理员权限)
    CheckAdminInitResponse CheckAdminInit(1: CheckAdminInitRequest req) (api.get="/api/admin/check")

    // 首次初始化超级管理员(仅在未初始化时可用)
    InitAdminResponse InitAdmin(1: InitAdminRequest req) (api.post="/api/admin/init")

    // 管理员列表(需要管理员权限)
    ListAdminUsersResponse ListAdminUsers(1: ListAdminUsersRequest req) (api.get="/api/admin/users")

    // 添加管理员(需要超级管理员权限)
    AddAdminResponse AddAdmin(1: AddAdminRequest req) (api.post="/api/admin/users/add")

    // 移除管理员(需要超级管理员权限)
    RemoveAdminResponse RemoveAdmin(1: RemoveAdminRequest req) (api.post="/api/admin/users/remove")

    // 公共模型 CRUD(需要管理员权限)
    ListPublicModelsResponse ListPublicModels(1: ListPublicModelsRequest req) (api.get="/api/admin/models")
    GetPublicModelResponse GetPublicModel(1: GetPublicModelRequest req) (api.get="/api/admin/models/:model_id")
    CreatePublicModelResponse CreatePublicModel(1: CreatePublicModelRequest req) (api.post="/api/admin/models/create")
    UpdatePublicModelResponse UpdatePublicModel(1: UpdatePublicModelRequest req) (api.post="/api/admin/models/:model_id/update")
    DeletePublicModelResponse DeletePublicModel(1: DeletePublicModelRequest req) (api.post="/api/admin/models/delete")

    // 启用/禁用公共模型
    EnablePublicModelResponse EnablePublicModel(1: EnablePublicModelRequest req) (api.post="/api/admin/models/enable")
    DisablePublicModelResponse DisablePublicModel(1: DisablePublicModelRequest req) (api.post="/api/admin/models/disable")
}
