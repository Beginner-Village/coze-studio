include "../base.thrift"

namespace go embedding

// 空间Embedding配置管理接口

// Embedding类型枚举
enum EmbeddingType {
    OPENAI = 1,
    ARK = 2,
    OLLAMA = 3,
    HTTP = 4,
}

// OpenAI Embedding配置
struct OpenAIEmbeddingConfig {
    1: required string base_url,
    2: required string api_key,
    3: required string model,
    4: optional bool by_azure,
    5: optional string api_version,
    6: required i32 dims,
    7: optional i32 request_dims,
}

// ARK Embedding配置
struct ArkEmbeddingConfig {
    1: optional string base_url,
    2: required string api_key,
    3: required string model,
    4: required i32 dims,
    5: optional string api_type, // text or multimodal
}

// Ollama Embedding配置
struct OllamaEmbeddingConfig {
    1: required string base_url,
    2: required string model,
    3: required i32 dims,
}

// HTTP Embedding配置
struct HttpEmbeddingConfig {
    1: required string addr,
    2: required i32 dims,
}

// 统一的Embedding配置
struct EmbeddingConfig {
    1: required EmbeddingType type,
    2: optional i32 max_batch_size,
    3: optional OpenAIEmbeddingConfig openai_config,
    4: optional ArkEmbeddingConfig ark_config,
    5: optional OllamaEmbeddingConfig ollama_config,
    6: optional HttpEmbeddingConfig http_config,
}

// 空间Embedding配置
struct SpaceEmbeddingConfig {
    1: required string id,
    2: required string space_id,
    3: required string name,
    4: optional string description,
    5: required EmbeddingConfig config,
    6: required i32 status, // 1: enabled, 2: disabled
    7: required bool is_default, // 是否为默认配置
    8: required i64 created_at,
    9: required i64 updated_at,
}

// === 请求和响应结构 ===

// 创建空间Embedding配置
struct CreateSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required string name (api.body="name"),
    3: optional string description (api.body="description"),
    4: required EmbeddingConfig config (api.body="config"),
    5: optional bool set_as_default (api.body="set_as_default"),

    255: optional base.Base Base (api.none="true"),
}

struct CreateSpaceEmbeddingResponse {
    1: optional SpaceEmbeddingConfig data,
    2: required i64 code,
    3: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 获取空间Embedding配置列表
struct ListSpaceEmbeddingsRequest {
    1: required string space_id (api.body="space_id"),

    255: optional base.Base Base (api.none="true"),
}

struct ListSpaceEmbeddingsResponse {
    1: optional list<SpaceEmbeddingConfig> data,
    2: required i64 code,
    3: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 获取空间默认Embedding配置
struct GetSpaceDefaultEmbeddingRequest {
    1: required string space_id (api.body="space_id"),

    255: optional base.Base Base (api.none="true"),
}

struct GetSpaceDefaultEmbeddingResponse {
    1: optional SpaceEmbeddingConfig data,
    2: required i64 code,
    3: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 更新空间Embedding配置
struct UpdateSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required string embedding_id (api.body="embedding_id"),
    3: optional string name (api.body="name"),
    4: optional string description (api.body="description"),
    5: optional EmbeddingConfig config (api.body="config"),

    255: optional base.Base Base (api.none="true"),
}

struct UpdateSpaceEmbeddingResponse {
    1: optional SpaceEmbeddingConfig data,
    2: required i64 code,
    3: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 删除空间Embedding配置
struct DeleteSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required string embedding_id (api.body="embedding_id"),

    255: optional base.Base Base (api.none="true"),
}

struct DeleteSpaceEmbeddingResponse {
    1: required i64 code,
    2: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 设置默认Embedding配置
struct SetDefaultSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required string embedding_id (api.body="embedding_id"),

    255: optional base.Base Base (api.none="true"),
}

struct SetDefaultSpaceEmbeddingResponse {
    1: required i64 code,
    2: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 启用/禁用Embedding配置
struct EnableSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required string embedding_id (api.body="embedding_id"),

    255: optional base.Base Base (api.none="true"),
}

struct EnableSpaceEmbeddingResponse {
    1: required i64 code,
    2: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

struct DisableSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required string embedding_id (api.body="embedding_id"),

    255: optional base.Base Base (api.none="true"),
}

struct DisableSpaceEmbeddingResponse {
    1: required i64 code,
    2: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// 测试Embedding配置
struct TestSpaceEmbeddingRequest {
    1: required string space_id (api.body="space_id"),
    2: required EmbeddingConfig config (api.body="config"),
    3: optional string test_text (api.body="test_text"),

    255: optional base.Base Base (api.none="true"),
}

struct TestSpaceEmbeddingResponse {
    1: optional bool success,
    2: optional i32 dims,
    3: optional string error_message,
    4: required i64 code,
    5: required string msg,
    255: required base.BaseResp BaseResp (api.none="true"),
}

// === 服务定义 ===

service SpaceEmbeddingService {
    // 空间Embedding配置管理
    CreateSpaceEmbeddingResponse CreateSpaceEmbedding(1: CreateSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/create', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    ListSpaceEmbeddingsResponse ListSpaceEmbeddings(1: ListSpaceEmbeddingsRequest request)
        (api.post='/api/embedding/space/list', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    GetSpaceDefaultEmbeddingResponse GetSpaceDefaultEmbedding(1: GetSpaceDefaultEmbeddingRequest request)
        (api.post='/api/embedding/space/default', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    UpdateSpaceEmbeddingResponse UpdateSpaceEmbedding(1: UpdateSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/update', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    DeleteSpaceEmbeddingResponse DeleteSpaceEmbedding(1: DeleteSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/delete', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    SetDefaultSpaceEmbeddingResponse SetDefaultSpaceEmbedding(1: SetDefaultSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/set-default', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    EnableSpaceEmbeddingResponse EnableSpaceEmbedding(1: EnableSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/enable', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    DisableSpaceEmbeddingResponse DisableSpaceEmbedding(1: DisableSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/disable', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")

    TestSpaceEmbeddingResponse TestSpaceEmbedding(1: TestSpaceEmbeddingRequest request)
        (api.post='/api/embedding/space/test', api.category="embedding", api.gen_path="embedding", agw.preserve_base = "true")
}
