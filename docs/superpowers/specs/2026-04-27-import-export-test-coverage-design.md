# Space 导入导出测试覆盖 — 设计文档

**Date:** 2026-04-27
**Status:** Approved (awaiting plan & implementation)
**Topic:** import-export-test-coverage

---

## 背景

最近 1-2 周完成了一系列 Space 导入导出/同步/版本/回滚功能（commit 列表）：

```
fc6d9fe1e feat(sync): auto-associate version and create pre-import snapshot
f886eebc7 feat(sync): add version conflict detection in import preview
c440711d3 feat(api): expose version info in sync history and import preview
da45ac824 test(release): add unit tests for hasher and diff logic
7b956db60 fix(release): embed release_version into ZIP manifest before serialization
9323e9538 feat(sync): update sync mappings after import for incremental detection
34526e3bd feat(import): expose ID mappings from ImportResult for sync mapping
a6474b42e feat(release): add space version management with release, rollback, and diff
5ed964a94 feat(sync): add production import script
f847e5180 feat(sync): implement delete handling in sync import
6f24612fc feat(sync): add sync API handler, router, and models
d057d1a73 feat(sync): add sync service with 3-phase import orchestration
8fbf05e46 feat(sync): add knowledge, folder import to space importer
688101c67 feat(sync): extend validator for v2.0.0 manifest with knowledge support
```

**问题**：核心业务流程几乎没有测试覆盖。

| 文件 | 行数 | 测试 |
|------|------|------|
| `import/space_importer.go` | ~1100 | ❌ 0 |
| `export/resource_collector.go` | ~1300 | ❌ 0 |
| `export/serializer.go` | ~700 | ❌ 0 |
| `sync/sync_service.go` | ~800 | ❌ 0 |
| `import/validator.go` | ~550 | ❌ 0 |
| `import/id_mapper.go` + `reference_rewriter.go` | ~750 | ❌ 0 |
| `release/hasher.go` | ~80 | ✅ 已有 |
| `release/release_service.go` | ~430 | ✅ 部分（parseSemver/diffIDList） |

实测覆盖率粗估 5-10%。

---

## 目标

补齐两层测试：

1. **单元测试**：覆盖纯逻辑函数（id mapping、reference rewriting、validator、serializer round-trip、release rollback/snapshot）
2. **E2E 集成测试**：用 testcontainers 启 MySQL/MinIO/ES/Redis，覆盖 6 个关键场景（happy path + 边界）

全部进 CI，单元测试每次 PR 跑，E2E 在 PR/main 分支跑（独立 job）。

---

## 范围与约束

### 必做
- 7 个新增/增补单元测试文件
- 7 个 E2E 测试场景（用 testcontainers）
- CI Makefile target + GitHub Actions / 现有 CI 集成
- 单元测试覆盖率 > 60%（关键路径）

### 不做（Out of Scope）
- 不重构源码（除非测试无法覆盖的明显死代码）
- 不做性能/压力测试（不在选 (b) Standard 范围内）
- 不做并发安全测试（不在选 (b) 范围内）
- 不做故障注入测试（不在选 (b) 范围内）
- 不改变现有功能行为（仅补测试）
- 不做端到端从 dev → prod 的真实数据迁移测试

### 技术选型

| 选项 | 决定 |
|------|------|
| 单元测试框架 | Go 标准 `testing` + `testify/assert`（项目已用） |
| Mock 框架 | `gomock`（项目已用） |
| 集成测试 | `testcontainers-go` |
| 数据库 | MySQL 8.4（与生产 OB 兼容协议层） |
| 对象存储 | MinIO（与生产一致） |
| 搜索引擎 | Elasticsearch 8.x（与生产一致） |
| 缓存 | Redis 7（与生产一致） |

---

## 架构

### 测试金字塔

```
                    ┌──────────────────────────┐
                    │  E2E (testcontainers)    │  7 个关键场景，慢，真实
                    │  ~30 秒/场景              │
                    └────────────┬─────────────┘
                                 │
              ┌──────────────────┴──────────────────┐
              │      单元测试 (mock-based)           │  ~50 个测试，毫秒级
              │      纯逻辑覆盖                       │
              └─────────────────────────────────────┘
```

### 单元测试覆盖

| 测试文件 | 目标函数 | 测试用例 |
|----------|----------|----------|
| `import/id_mapper_test.go` | `IDMapper.Map`、`Get`、`Has` | 1. 新建映射；2. 已存在映射；3. 跨类型映射；4. 0 ID 边界；5. 大量 ID 性能 |
| `import/reference_rewriter_test.go` | `RewriteReferences` | 1. workflow → tool 引用；2. agent → knowledge；3. agent → workflow；4. 嵌套引用；5. 丢失引用（应保留 placeholder）；6. 循环引用 |
| `import/validator_test.go` | `ValidateManifest` | 1. v2.0.0 manifest 通过；2. 缺必填字段拒绝；3. 未知 resource type 拒绝；4. 版本号格式错误；5. 大量 resource 边界 |
| `export/serializer_test.go` | `SerializeSpace`、`DeserializeSpace` | 1. round-trip 等价（序列化后反序列化数据相等）；2. 空 space；3. 含特殊字符的字段；4. 超长字段截断；5. 大型嵌套对象 |
| `release/hasher_test.go` ✅ | `HashPackage`、`BuildResourceHashes` | 已有 |
| `release/release_service_test.go` ✅ | `parseSemver`、`diffIDList` | 已有，新增：`createSnapshot`、`rollback` 数据正确性 |
| `sync/sync_mapping_repo_test.go` | `Upsert`、`GetByResource`、`Delete` | 1. 新建映射；2. 更新已有；3. 删除；4. 多 mapping 批量 upsert |

### E2E 测试场景（基于已选 (b) Standard 范围）

| # | 测试 | 流程描述 | 主要断言 |
|---|------|----------|----------|
| 1 | `TestExportImportRoundTrip` | 创建 source space (含 2 agent / 1 workflow / 1 knowledge / 3 plugin) → 导出 → 创建 target space → 导入 → 对比 | resource 数量一致；关键字段（name, prompt, schema）等价；ID 已重新分配 |
| 2 | `TestImportTwiceIsUpdate` | 同一 manifest 导入两次 | 第二次 sync_mapping 命中；resources count 不变；fields 更新（修改 manifest 后再导入） |
| 3 | `TestRollbackRestoresState` | 创建 release v1 → 修改 space → 创建 release v2 → 调 rollback API to v1 | space 状态等价于 v1 快照；release_history 记录 rollback；新版本号 v3 |
| 4 | `TestVersionConflictDetected` | 同 release_version 二次 import preview | preview 返回 `version_conflict=true`；preview 含原版本元信息 |
| 5 | `TestCrossReferenceIntegrity` | source space 含 workflow A 调用 tool B、agent C 关联 knowledge D，导入 target | A 引用的 tool ID 是 target B 的新 ID；C 关联的 knowledge ID 是 target D 的新 ID；无悬空引用 |
| 6 | `TestDeleteHandling` | source 第一次有 5 个 agent，第二次 manifest 只剩 3 个，导入 target | 缺失的 2 个 agent 在 target 标记为 deleted（per `f847e5180`）；sync_mapping 同步标记 |
| 7 | `TestESyncFailureGraceful` | mock ES client 模拟失败 → 调 import | DB 事务正常提交（Pending → Indexed 之外的状态）；ES 同步错误仅记 log；返回成功 |

---

## 组件设计

### 测试基础设施

#### `backend/test/integration/space_sync/setup_test.go`

**职责**：启停 testcontainers，提供共享 DB/MinIO/ES/Redis 实例

```go
package space_sync_test

import (
    "context"
    "database/sql"
    "os"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/mysql"
    "github.com/testcontainers/testcontainers-go/modules/minio"
    "github.com/testcontainers/testcontainers-go/modules/elasticsearch"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
    testDB  *sql.DB
    testS3  *MinIOClient
    testES  *ESClient
    testRDB *RedisClient
)

func TestMain(m *testing.M) {
    ctx := context.Background()

    mysqlC, err := mysql.Run(ctx, "mysql:8.4", ...)
    minioC, err := minio.Run(ctx, "minio/minio:latest", ...)
    esC,    err := elasticsearch.Run(ctx, "elasticsearch:8.13.0", ...)
    redisC, err := redis.Run(ctx, "redis:7-alpine", ...)

    // 初始化 schema（用项目现有 init SQL）
    runMigrations(testDB)

    code := m.Run()

    // 清理
    mysqlC.Terminate(ctx)
    minioC.Terminate(ctx)
    esC.Terminate(ctx)
    redisC.Terminate(ctx)

    os.Exit(code)
}
```

#### `backend/test/integration/space_sync/helpers.go`

**职责**：fixture 生成器、断言工具

```go
// CreateTestSpace 创建一个填充测试数据的 space
func CreateTestSpace(t *testing.T, opts ...SpaceOpt) *TestSpace

// SpaceFixtures 提供常用 fixture 组合
type SpaceFixtures struct {
    SmallSpace func() *TestSpace        // 1 agent / 0 workflow
    MediumSpace func() *TestSpace       // 2 agent / 1 workflow / 1 knowledge / 3 plugin
    LargeSpace func() *TestSpace        // 50 agent / 10 workflow / 5 knowledge
    SpaceWithRefs func() *TestSpace     // 含跨资源引用
}

// AssertSpacesEquivalent 对比两个 space 的核心字段（忽略 ID/timestamp）
func AssertSpacesEquivalent(t *testing.T, expected, actual *TestSpace)

// CleanupSpace 测试后清理 space 数据
func CleanupSpace(t *testing.T, spaceID int64)
```

### CI 集成

#### `Makefile` 新增 target

```makefile
.PHONY: test-unit test-integration test-all

test-unit:
	go test -v ./backend/... -short

test-integration:
	go test -v ./backend/test/integration/... -timeout 10m

test-all: test-unit test-integration
```

`-short` flag 在单元测试 `setup_test.go` 中检查：
```go
func TestMain(m *testing.M) {
    if testing.Short() {
        os.Exit(0)  // 跳过 integration 测试
    }
    // ...
}
```

#### CI workflow（项目现有 CI 脚本添加）

新增 step：
```yaml
- name: Run integration tests
  run: make test-integration
  env:
    TESTCONTAINERS_RYUK_DISABLED: true  # CI 环境优化
```

---

## 文件结构

### 新建
```
backend/application/space/import/id_mapper_test.go
backend/application/space/import/reference_rewriter_test.go
backend/application/space/import/validator_test.go
backend/application/space/export/serializer_test.go
backend/application/space/sync/sync_mapping_repo_test.go

backend/test/integration/space_sync/
├── setup_test.go              # testcontainers 启停
├── helpers.go                 # fixture + 断言
├── round_trip_test.go         # E2E 1
├── increment_test.go          # E2E 2
├── rollback_test.go           # E2E 3
├── version_conflict_test.go   # E2E 4
├── reference_test.go          # E2E 5
├── delete_test.go             # E2E 6
└── es_failure_test.go         # E2E 7

Makefile (新增 target)
```

### 修改
```
backend/application/space/release/release_service_test.go  # 增补 snapshot/rollback 测试
.github/workflows/test.yml  # 或现有 CI 脚本，新增 integration job
```

### 不改
- 全部业务源码（除非发现 bug 必须修，单独提 PR）

---

## 数据流（E2E 关键流程）

### TestExportImportRoundTrip

```
[Setup]
  → CreateTestSpace(MediumSpace) → sourceSpaceID
  → CreateEmptyTargetSpace() → targetSpaceID

[Export]
  → SpaceExporter.Export(sourceSpaceID, opts)
  → 返回 exportZIP (in-memory bytes 或 MinIO 路径)

[Import]
  → SyncService.ImportPreview(targetSpaceID, exportZIP)
  → 验证 preview 返回正确的 resources 列表
  → SyncService.ImportConfirm(targetSpaceID, exportZIP, options)
  → 返回 importResult

[Assert]
  → loadSpace(targetSpaceID) → targetSpaceData
  → AssertSpacesEquivalent(sourceSpaceData, targetSpaceData)
  → 验证：
     - agents.count == sources.agents.count
     - 每个 agent.name 匹配
     - 每个 agent 的 ID 是新分配（!= source ID）
     - sync_mapping 表中有对应的映射记录
```

### TestRollbackRestoresState

```
[Setup]
  → CreateTestSpace(SmallSpace) → spaceID
  → ReleaseService.Release(spaceID, "1.0.0", "initial") → release_v1
  → 修改 space（agent.prompt = "modified"）
  → ReleaseService.Release(spaceID, "1.1.0", "after-modify") → release_v2

[Rollback]
  → ReleaseService.Rollback(spaceID, "1.0.0")

[Assert]
  → loadSpace(spaceID) → currentState
  → 断言 agent.prompt == initial value（不是 "modified"）
  → release_history 表中最新一条 type=rollback, target_version=1.0.0
  → 新版本号自动 = 1.2.0（递增）
```

### TestESyncFailureGraceful

```
[Setup]
  → 用 mock ES client 替换默认实现
  → ESClient.On("Index").Return(errors.New("ES unavailable"))

[Action]
  → SyncService.ImportConfirm(spaceID, zip)

[Assert]
  → 返回 success（不是 error）
  → DB 中所有 resources 已创建（事务提交了）
  → 日志中含 "ES sync failed" 但不 panic
  → import_history 标记 ES sync = false
```

---

## 错误处理

### 测试错误处理原则

1. 所有 E2E 测试用 `t.Cleanup()` 注册清理函数（删 space、清 ES index、清 MinIO bucket）
2. testcontainers 启动失败时输出明确日志（容器拉取超时、端口冲突）
3. 单测 mock 失败时用 `gomock.WithCallerInfo` 输出调用栈
4. CI 环境 testcontainers 用 `Reuse=true` 避免每个测试重启容器（仅限 read-only 测试）

---

## 测试矩阵（断言清单）

| E2E 测试 | 数据正确性 | sync_mapping 正确性 | 错误处理 | 性能 |
|----------|-----------|---------------------|----------|------|
| RoundTrip | ✅ 字段相等 | ✅ 全部映射存在 | - | - |
| IncrementUpdate | ✅ 字段更新 | ✅ 第二次命中 | - | - |
| Rollback | ✅ 状态回退 | - | - | - |
| VersionConflict | - | - | ✅ preview 返回 conflict | - |
| CrossReference | ✅ ID 改写正确 | ✅ 所有引用映射存在 | - | - |
| DeleteHandling | ✅ deleted 标记 | ✅ 标记同步 | - | - |
| ESFailure | ✅ DB 提交 | - | ✅ ES 失败优雅降级 | - |

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| testcontainers 在 CI 慢（启动 4 个容器 ~1min） | 单独 job，并行 unit test，不阻塞快速反馈 |
| OB vs MySQL 在某些 SQL 行为差异 | 测试用 MySQL（OB 兼容协议层），生产前另跑 OB smoke test |
| Mac M 系列芯片 ARM 镜像问题 | testcontainers 用 multi-arch 镜像；README 说明本地需 colima 或 docker desktop |
| Schema 升级时 fixture 需同步 | fixture 用 Go 代码生成（不用 SQL dump），schema 变化时跟着改 |
| 测试 flaky（容器启动慢导致 timeout） | 每个测试 timeout 设 30s；setup 阶段 timeout 设 5min |
| 测试间相互污染（共享 DB） | 每个测试 t.Run 子测试用独立 spaceID，结束清理 |

---

## 验收标准

1. ✅ 所有新增单元测试通过（5 个文件）
2. ✅ 所有 E2E 测试通过（7 个文件）
3. ✅ 单元测试覆盖率 > 60%（针对 `space/import`、`space/export`、`space/sync`、`space/release`）
4. ✅ `make test-unit` 在 30 秒内跑完
5. ✅ `make test-integration` 在 5 分钟内跑完
6. ✅ CI workflow 新增 integration job 跑通
7. ✅ 测试可重复运行（多次跑结果一致）
8. ✅ README 或 CONTRIBUTING 文档说明如何本地跑测试
9. ✅ 不破坏现有源码（git diff 仅含测试文件 + Makefile + CI 配置）
