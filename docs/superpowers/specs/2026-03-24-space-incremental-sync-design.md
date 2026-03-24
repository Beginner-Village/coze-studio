# 空间增量同步设计文档

## 概述

实现 Ynet Studio 的测试环境到生产环境空间级别增量同步能力。支持全量导出/导入和增量更新，覆盖所有空间资源类型（包括知识库及其原始文件）。

### 核心场景

- 测试环境通过页面 UI 导出空间数据（全量或增量）
- 生产环境通过脚本/API 导入空间数据
- 支持重复导入（幂等），测试环境为唯一真实来源，覆盖生产端修改

### 约束条件

| 维度 | 决定 |
|------|------|
| 对象存储 | 两端独立，原始文件随 ZIP 包迁移 |
| 向量引擎 | 两端一致（同类型引擎） |
| 触发方式 | 测试端页面导出，生产端脚本/API 导入 |
| 同步粒度 | 整个空间 |
| 文件体积 | 未知，需兼容大文件 |
| 冲突策略 | 测试环境覆盖生产（无冲突检测） |

---

## 一、导出包结构

```
space_export_{space_name}_{space_id}_{timestamp}.zip
├── manifest.json                    # 包元数据
├── agents/
│   ├── index.json
│   └── {agent_id}.json
├── plugins/
│   ├── index.json
│   └── {plugin_id}.json
├── workflows/
│   ├── index.json
│   └── {workflow_id}.json
├── variables/
│   ├── index.json
│   └── {variable_id}.json
├── space_models/
│   ├── index.json
│   └── {space_model_id}.json
├── knowledge_bases/
│   ├── index.json
│   └── {knowledge_id}/
│       ├── meta.json
│       ├── documents/
│       │   ├── index.json
│       │   └── {document_id}.json   # 文档元数据 + 分片内容内联
│       └── files/
│           └── {document_id}_{filename}
├── external_knowledge/
│   ├── index.json
│   └── {binding_id}.json
├── folders/
│   ├── index.json
│   ├── {folder_id}.json
│   └── resource_mappings.json
├── deleted_resources.json           # 增量模式：已删除资源列表
└── sync_state.json                  # 同步状态（export_time, 资源快照摘要）
```

### manifest.json 结构

```json
{
  "version": "2.0.0",
  "export_time": "2026-03-24T12:00:00+08:00",
  "sync_type": "full|incremental",
  "since_time": 0,
  "source": {
    "space_id": "100",
    "space_name": "猎鹰测试空间",
    "exporter_id": "1001"
  },
  "statistics": {
    "agents": 5,
    "plugins": 3,
    "workflows": 2,
    "variables": 5,
    "space_models": 4,
    "knowledge_bases": 3,
    "documents": 12,
    "files_total_size": 52428800,
    "external_knowledge": 1,
    "folders": 4
  },
  "id_registry": {
    "agents": [101, 102, 103],
    "plugins": [201, 202],
    "workflows": [301, 302],
    "variables": [401, 402],
    "space_models": [501, 502],
    "knowledge_bases": [601, 602],
    "documents": [701, 702, 703],
    "external_knowledge": [801],
    "folders": [901, 902]
  }
}
```

### deleted_resources.json 结构（增量模式）

```json
{
  "agents": [105],
  "plugins": [],
  "workflows": [],
  "variables": [],
  "space_models": [],
  "knowledge_bases": [],
  "documents": [705, 706],
  "external_knowledge": [],
  "folders": []
}
```

### sync_state.json 结构

```json
{
  "export_time": 1711267200000,
  "source_space_id": 100,
  "resource_snapshot": {
    "agents_count": 5,
    "plugins_count": 3,
    "workflows_count": 2,
    "knowledge_bases_count": 3,
    "documents_count": 12
  }
}
```

---

## 二、数据库新增表

### space_sync_mapping — ID 映射表

在**生产环境**创建，记录源资源 ID 到目标资源 ID 的映射。

```sql
CREATE TABLE `space_sync_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_space_id` bigint NOT NULL COMMENT '源空间ID（测试环境）',
  `target_space_id` bigint NOT NULL COMMENT '目标空间ID（生产环境）',
  `resource_type` varchar(32) NOT NULL COMMENT 'agent/plugin/workflow/variable/space_model/knowledge/document/slice/folder/external_knowledge',
  `source_resource_id` bigint NOT NULL COMMENT '源资源ID',
  `target_resource_id` bigint NOT NULL COMMENT '目标资源ID',
  `source_updated_at` bigint NOT NULL DEFAULT 0 COMMENT '上次同步时源资源的 updated_at',
  `content_hash` varchar(64) DEFAULT NULL COMMENT '预留字段，当前未使用，后续可用于跳过无变更的 UPDATE 优化',
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source` (`source_space_id`, `resource_type`, `source_resource_id`),
  KEY `idx_target` (`target_space_id`, `resource_type`, `target_resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='跨环境空间同步ID映射';
```

### space_sync_history — 同步历史记录

```sql
CREATE TABLE `space_sync_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_space_id` bigint NOT NULL,
  `target_space_id` bigint NOT NULL,
  `sync_type` varchar(16) NOT NULL COMMENT 'full/incremental',
  `export_time` bigint NOT NULL COMMENT '导出时间戳',
  `import_time` bigint DEFAULT NULL COMMENT '导入时间戳',
  `statistics` json NOT NULL COMMENT '同步统计',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=exported, 1=imported, 2=failed',
  `error_msg` text DEFAULT NULL,
  `package_file_name` varchar(256) DEFAULT NULL,
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_source_target` (`source_space_id`, `target_space_id`, `export_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间同步历史记录';
```

---

## 三、API 接口

### 导出端（测试环境，页面调用）

```
POST /api/space/{space_id}/sync/export
Body: { "mode": "full" | "incremental", "since_time": 1711234567890 }

Response:
{
  "download_url": "https://...",
  "file_name": "space_猎鹰_100_20260324.zip",
  "file_size": 52428800,
  "expires_at": "2026-03-24T13:00:00Z",
  "sync_type": "incremental",
  "statistics": {
    "agents": { "total": 5, "changed": 1 },
    "plugins": { "total": 3, "changed": 0 },
    "workflows": { "total": 2, "changed": 1 },
    "knowledge_bases": { "total": 3, "changed": 2, "files_size": 31457280 },
    "deleted": { "agents": 0, "documents": 1 }
  }
}
```

```
GET /api/space/{space_id}/sync/last-export

Response:
{ "export_time": 1711234567890, "sync_type": "full" }
```

### 导入端（生产环境，脚本调用）

```
POST /api/space/{space_id}/sync/import/preview
Content-Type: multipart/form-data
Body: file=@export.zip

Response:
{
  "import_token": "abc123...",
  "token_expires_at": "2026-03-24T12:30:00Z",
  "manifest": { ... },
  "plan": {
    "create": { "agents": 1, "knowledge_bases": 2, "documents": 5 },
    "update": { "agents": 2, "workflows": 1, "documents": 3 },
    "delete": { "documents": 1 }
  },
  "warnings": ["Model 'gpt-4o' not found, will use fallback"]
}
```

```
POST /api/space/{space_id}/sync/import/confirm
Body: { "import_token": "abc123..." }

Response:
{
  "status": "success",
  "statistics": {
    "created": { "agents": 1, "knowledge_bases": 2 },
    "updated": { "agents": 2, "workflows": 1 },
    "deleted": { "documents": 1 },
    "files_uploaded": 7,
    "vectors_indexed": 1523
  },
  "sync_history_id": 42
}
```

```
GET /api/space/{space_id}/sync/history

Response:
[{ "id": 42, "source_space_id": 100, "sync_type": "incremental", ... }]
```

### 导入 Token

- TTL: 30 分钟（大包解析可能耗时）
- 存储策略：Preview 阶段将上传的 ZIP 存入对象存储（临时 key，30 分钟 TTL），内存中只保存轻量 token 引用（manifest + 临时文件 key）。Confirm 阶段从对象存储读取 ZIP 执行导入。避免大 ZIP 长时间占用内存，也解决服务器重启导致 token 丢失的问题。

### 包体积限制

- 最大包体积：2GB（超出时导出 API 返回错误，提示拆分空间）
- 导入时流式解压，不将整个 ZIP 加载到内存

### 脚本使用示例

```bash
#!/bin/bash
# sync-to-prod.sh
PROD_API="http://10.10.10.220:8080"
SPACE_ID=200
ZIP_FILE=$1

PREVIEW=$(curl -s -X POST "$PROD_API/api/space/$SPACE_ID/sync/import/preview" \
  -F "file=@$ZIP_FILE")
TOKEN=$(echo $PREVIEW | jq -r '.import_token')
echo "Import plan:"
echo $PREVIEW | jq '.plan'

RESULT=$(curl -s -X POST "$PROD_API/api/space/$SPACE_ID/sync/import/confirm" \
  -H "Content-Type: application/json" \
  -d "{\"import_token\": \"$TOKEN\"}")
echo "Import result:"
echo $RESULT | jq '.statistics'
```

---

## 四、导出流程

### 全量导出

1. `ResourceCollector.CollectAll(spaceID)` 采集所有资源
   - 现有：Agents, Plugins, Workflows, Variables, SpaceModels
   - 新增：KnowledgeBases, ExternalKnowledge, Folders
2. 知识库采集 `collectKnowledgeBases(spaceID)`：
   - 查 `knowledge` 表 (WHERE space_id = ? AND deleted_at IS NULL)
   - 对每个知识库，查 `knowledge_document` 表
   - 对每个文档，查 `knowledge_document_slice` 表（分批 100 条）
   - 从对象存储下载原始文件 (document.uri)
3. 文件夹采集 `collectFolders(spaceID)`：
   - 查 `folder` 表
   - 查 `resource_folder_mapping` 表
4. 序列化写入 ZIP 包
5. 写入 `sync_state.json`（export_time = now）
6. 上传 ZIP 到对象存储，返回下载链接

### 增量导出

1. `ResourceCollector.CollectIncremental(spaceID, sinceTime)` 采集变更资源
   - 对每种资源：`WHERE space_id = ? AND updated_at > since_time AND deleted_at IS NULL`
2. 采集已删除资源：
   - 注意：`deleted_at` 是 `DATETIME(3)` 类型，而 `since_time` 是毫秒时间戳，需要类型转换
   - 使用 `.Unscoped()` 绕过 GORM 的自动 `deleted_at IS NULL` 过滤
   - 查询：`Unscoped().WHERE space_id = ? AND deleted_at IS NOT NULL AND UNIX_TIMESTAMP(deleted_at) * 1000 > ?`
   - 写入 `deleted_resources.json`
3. 知识库增量以**文档为最小粒度**：
   - 知识库元数据变更 → 导出 meta.json
   - 文档新增/变更 → 导出 document.json + 全部 slices + 原始文件
   - 文档删除 → 记入 deleted_resources.json
   - 不做分片级别增量（一个文档变了就整体重导）
4. 后续流程与全量导出相同

---

## 五、导入流程

### Step 1: Preview（验证 + 计划）

1. 解压 ZIP，读取 manifest.json
2. 校验版本兼容性（manifest.version）
3. 查 `space_sync_mapping` 表，为每个资源分类：
   - 有映射 → 标记为 UPDATE
   - 无映射 → 标记为 CREATE
4. 读取 `deleted_resources.json`，标记 DELETE 操作
5. 返回导入计划（create/update/delete 统计）+ warnings

### Step 2: Confirm（执行）

#### 执行阶段划分

导入分为三个阶段，将 DB 事务与非事务性操作（对象存储、向量存储）分离：

```
Phase 1: 预处理（事务外）
   ├─ 上传所有知识库原始文件到目标对象存储
   ├─ 记录新 URI 映射 (old_uri → new_uri)
   └─ 失败时清理已上传文件

Phase 2: 数据库事务
   ├─ 0. 处理删除 (deleted_resources)
   │     → 对有映射的资源执行软删除
   │     → 知识库文档删除：删 slices (DB) + 软删 document
   │     → 知识库删除：软删 knowledge
   │     → 清理 mapping 记录
   │
   ├─ 1. Upsert Space Models
   │     → 跨系统匹配：先按 ID，不存在则按 model_entity name 匹配
   │
   ├─ 2. Upsert Plugins
   │     → CREATE: 创建 plugin + plugin_draft
   │     → UPDATE: 更新 plugin_draft 的 manifest, openapi_doc 等
   │
   ├─ 3. Upsert Workflows
   │     → CREATE: 创建 workflow_meta + workflow_draft + workflow_version
   │     → UPDATE: 更新 workflow_draft 的 canvas, input/output_params
   │     → 画布引用重写：plugin_id, workflow_id, knowledge_id
   │
   ├─ 4. Upsert Knowledge Bases (仅 DB 记录)
   │     → 见下方详细流程（此阶段只写 DB，不写向量存储）
   │
   ├─ 5. Upsert External Knowledge Bindings
   │
   ├─ 6. Upsert Agents
   │     → CREATE: 创建 single_agent_draft + agent_tool_draft
   │     → UPDATE: 更新 single_agent_draft 的所有配置字段
   │     → 引用重写：plugin_id, workflow_id, knowledge_id, model_id, variables_meta_id
   │
   ├─ 7. Upsert Variables
   │
   └─ 8. Upsert Folders + Resource Mappings

   每步完成后写入/更新 space_sync_mapping 记录。

Phase 3: 后处理（事务外，事务提交成功后执行）
   ├─ 向量存储操作
   │     ├─ 删除的知识库：删除 collection
   │     ├─ 删除的文档：删除向量存储 partition
   │     ├─ 新增的知识库：创建 collection
   │     └─ 新增/更新的文档：embedding + 写入向量索引
   ├─ ES 搜索索引同步（Agents, Plugins, Workflows）
   ├─ 记录 space_sync_history
   └─ 失败时：记录错误到 history，向量不一致可通过重新导入修复
```

**设计原则**：参考现有 `datacopy.go` 的模式，DB 事务只管数据库操作，向量存储和文件操作在事务外处理，通过 deferred cleanup 保证一致性。

#### 知识库导入详细流程

```
对每个知识库：
├─ 查 mapping: source_knowledge_id → target_id?
│
├─ 已存在 (UPDATE):
│   ├─ 更新 knowledge 表元数据
│   ├─ 对每个文档：
│   │   ├─ 查 mapping: source_document_id → target_doc_id?
│   │   ├─ 已存在 (UPDATE):
│   │   │   ├─ 上传新文件到对象存储，更新 URI
│   │   │   ├─ 更新 document 元数据
│   │   │   ├─ 删除旧 slices (DB + 向量存储 partition)
│   │   │   ├─ 批量创建新 slices
│   │   │   └─ 写入向量存储 (embedding + index)
│   │   └─ 不存在 (CREATE):
│   │       ├─ 上传文件到对象存储
│   │       ├─ 创建 document 记录
│   │       ├─ 批量创建 slices
│   │       ├─ 写入向量存储
│   │       └─ 插入 mapping
│   └─ 更新 mapping 的 source_updated_at
│
└─ 不存在 (CREATE):
    ├─ 创建 knowledge 记录
    ├─ 创建向量存储 collection
    ├─ 对每个文档：
    │   ├─ 上传文件到对象存储
    │   ├─ 创建 document 记录
    │   ├─ 批量创建 slices (100条/批)
    │   └─ 写入向量存储 (embedding + index)
    └─ 插入所有 mapping 记录
```

#### 向量存储与 Embedding 处理

**核心原则：导出包中不包含 embedding 向量，导入时使用目标空间的 embedding 模型重新生成。**

原因：
- 两个环境的 embedding 模型可能版本不同，向量不可直接复用
- embedding 向量体积大，不打入 ZIP 包可显著减小包体积

向量存储操作参考现有 `datacopy.go` 的 `copyDocument` 逻辑：
- 调用 `getManagersForSpace()` 获取目标空间的向量引擎和 embedding 模型
- 调用 `slice2Document()` 转换分片为向量文档（此过程触发 embedding 计算）
- 调用 `ss.Store()` 写入向量索引

**前置检查**：导入 preview 阶段检测目标空间是否配置了 embedding 模型（`space_embedding` 表），如果没有，返回 error 而非 warning，阻止导入。

**大知识库处理**：向量重建在 Phase 3（事务后）执行，分批处理（100 条/批），单个文档失败不影响其他文档。失败的文档记录到导入结果的 `warnings` 中，用户可通过重新导入修复。

### Upsert 核心逻辑

```go
// 伪代码 - 以 Agent 为例
func upsertAgent(agent, importCtx) {
    mapping := queryMapping(sourceSpaceID, "agent", agent.ID)

    if mapping != nil {
        // UPDATE 路径
        targetID := mapping.TargetResourceID
        rewritten := rewriteReferences(agent, importCtx)
        tx.Table("single_agent_draft").
            Where("agent_id = ? AND space_id = ?", targetID, targetSpaceID).
            Updates(rewritten)
        // 更新 agent_tool_draft: 先删后建
        tx.Table("agent_tool_draft").Where("agent_id = ?", targetID).Delete()
        // 重建 tools...
        updateMapping(mapping, agent.UpdatedAt)
    } else {
        // CREATE 路径
        newID := idGen.GenID()
        createAgent(agent, newID, importCtx)
        insertMapping(sourceSpaceID, "agent", agent.ID, targetSpaceID, newID)
    }
}
```

---

## 六、现有导出/导入问题修复

新的 sync 导入需修复现有 `reference_rewriter.go` 中的以下问题：

| # | 问题 | 位置 | 修复方案 |
|---|------|------|---------|
| 1 | `agent.KnowledgeRefs = nil` | reference_rewriter.go:112 | 改为通过 KnowledgeIDMap 重写知识库引用 |
| 2 | `agent.ExternalKnowledge = nil` | reference_rewriter.go:116 | 改为通过 ExternalKnowledgeIDMap 重写 |
| 3 | `agent.DatabaseRefs = nil` | reference_rewriter.go:118 | 保持 nil + 生成 warning（数据库连接不跨环境） |
| 4 | Workflow canvas `knowledge_id = 0` | reference_rewriter.go:269 | 改为通过 KnowledgeIDMap 重写 |
| 5 | Workflow canvas `database_id = 0` | reference_rewriter.go:273 | 保持 0 + 生成 warning |
| 6 | 文件夹结构未导出 | resource_collector.go | 新增 collectFolders() |

### 数据库引用的处理策略

数据库连接信息（地址、账号、密码）与环境绑定，不应跨环境复制。导入时：
- Agent 和 Workflow 中的 `database_id` / `database_config` 设为空
- 在 warnings 中提示："数据库连接需要在生产环境重新配置"

### ImportContext 新增字段

```go
type ImportContext struct {
    // 现有字段
    TargetSpaceID        int64
    UserID               int64
    AgentIDMap           map[int64]int64
    PluginIDMap          map[int64]int64
    WorkflowIDMap        map[int64]int64
    VariableIDMap        map[int64]int64
    SpaceModelIDMap      map[int64]int64
    CreatedSpaceModelIDs map[int64]bool
    FallbackModelID      *int64
    PackageIDs           *export.IDRegistry

    // 新增字段
    KnowledgeIDMap         map[int64]int64  // 知识库 ID 映射
    DocumentIDMap          map[int64]int64  // 文档 ID 映射
    FolderIDMap            map[int64]int64  // 文件夹 ID 映射
    ExternalKnowledgeIDMap map[int64]int64  // 外部知识库绑定 ID 映射

    // Upsert 模式支持
    SyncMode               string                      // "create_only" | "upsert"
    MappingStore           *SyncMappingStore            // 统一的映射查询/缓存层，preview 时加载一次
}
```

`SyncMappingStore` 封装 `space_sync_mapping` 表的查询逻辑，preview 阶段一次性加载目标空间的所有映射到内存，避免导入过程中反复查表：

```go
type SyncMappingStore struct {
    mappings map[string]int64 // key: "type:sourceID" → value: targetID
}

func (s *SyncMappingStore) GetTargetID(resourceType string, sourceID int64) (int64, bool)
func (s *SyncMappingStore) SetMapping(resourceType string, sourceID, targetID int64)
func (s *SyncMappingStore) RemoveMapping(resourceType string, sourceID int64)
```

---

## 七、关键保证

| 保证 | 实现方式 |
|------|---------|
| **幂等性** | 通过 mapping 表判断 create/update，同一包重复导入不产生重复数据 |
| **原子性** | 数据库操作在事务内，失败回滚。文件上传在事务前执行，失败时清理已上传文件 |
| **引用完整性** | 按依赖顺序导入：Model → Plugin → Workflow → Knowledge → Agent |
| **向量一致性** | 文档更新时先删后建 partition，不残留旧向量 |
| **可追溯** | 每次同步记录 space_sync_history |
| **大文件兼容** | ZIP 流式写入，分片批量处理（100条/批） |

---

## 八、代码组织

新增/修改的文件结构：

```
backend/application/space/
├── export/
│   ├── types.go                    # 修改：新增知识库/文件夹导出类型
│   ├── models.go                   # 修改：新增知识库/文件夹 DB Model
│   ├── resource_collector.go       # 修改：新增 collectKnowledgeBases, collectFolders
│   ├── serializer.go               # 修改：支持知识库文件写入 ZIP
│   └── space_exporter.go           # 修改：支持 incremental mode
├── import/
│   ├── types.go                    # 修改：ImportContext 新增字段
│   ├── id_mapper.go                # 修改：新增知识库/文件夹 ID 映射
│   ├── reference_rewriter.go       # 修改：知识库引用重写（不再清空）
│   ├── space_importer.go           # 修改：新增知识库/文件夹导入 + Upsert 逻辑
│   └── validator.go                # 修改：支持 v2.0.0 manifest 验证
└── sync/                           # 新增目录
    ├── sync_service.go             # Sync API 业务逻辑入口
    ├── sync_mapping_repo.go        # space_sync_mapping CRUD
    └── sync_history_repo.go        # space_sync_history CRUD

backend/api/
├── handler/space/
│   └── space_sync_service.go       # 新增：Sync API HTTP handler
├── model/space/
│   └── space_sync.go               # 新增：Sync API 请求/响应模型
└── router/space/
    └── space_sync.go               # 新增：Sync 路由注册

idl/space/
└── space_sync.thrift               # 新增：Sync IDL 定义

docs/ynet-database-sql/
└── 04-ynet-sync.sql                # 新增：Sync 表 DDL
```

---

## 九、不在范围内

以下内容不在本次设计范围内：

- **实时同步 / 自动同步**：本次仅支持手动触发
- **双向同步**：仅支持测试 → 生产单向
- **选择性资源同步**：仅支持整个空间级别
- **数据库连接跨环境迁移**：数据库连接信息与环境绑定，不迁移
- **前端 UI**：生产环境通过脚本操作，测试环境的导出 UI 复用/扩展现有页面
- **Skill 资源导出**：skill 表存在但当前未纳入导出范围，后续按需添加。如果 Agent 引用了 skill，导入时在 warnings 中提示需要手动配置
