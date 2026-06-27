# 策略（Strategy）渐进式披露 —— 后端实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 coze-studio 后端新增"策略"资源类型，单智能体绑定后自动获得 3 个渐进式披露工具（列场景 / 列能力项 / 调用能力项），能力项多态支持 工作流/插件/知识库/纯提示词。

**Architecture:** 方案 C（混合）。端到端镜像现有 `database` 资源类型的接线（库资源 + 绑定 agent + 注入工具）。3 个工具用现有 eino `tool.InvokableTool` 抽象表达，第 3 个工具按能力项 `type` 路由进现有执行/检索通路（workflow 执行引擎 / `crossplugin.ExecuteTool` / knowledge recall / 返回内联文本）。

**Tech Stack:** Go、Hertz + hz 代码生成、thriftgo、GORM(gen)、cloudwego/eino（`tool`/`schema`）、Atlas migrations、MySQL。

## Global Constraints

- 仓库根：`/Users/luzhipeng/projects/ynet/coze-studio`；后端根：`backend/`。
- Go module 路径前缀：`github.com/ynet-dev/ynet-studio/backend/...`（见 `node_tool_sandbox.go` import）。
- 资源类型枚举现状到 `ResType_Voice = 9`；策略用 `ResType_Strategy = 10`。
- 工具抽象：eino `tool.InvokableTool`（`Info(ctx)(*schema.ToolInfo,error)` + `InvokableRun(ctx, argumentsInJSON string, ...tool.Option)(string,error)`），import `github.com/cloudwego/eino/components/tool` 与 `github.com/cloudwego/eino/schema`。
- 时间戳统一 `bigint` 毫秒（`autoCreateTime:milli`/`autoUpdateTime:milli`），软删 `gorm.DeletedAt`。
- 模型工具名固定：`strategy_list_scenarios` / `strategy_list_capabilities` / `strategy_invoke_capability`。
- 能力项类型字符串常量：`workflow` / `plugin` / `knowledge` / `prompt`。
- 越权红线：`strategy_invoke_capability` 必须校验 `capability_id ∈ 当前 agent 所绑策略的能力项集合`。
- 验证命令：`cd backend && go build ./...`；测试 `cd backend && go test ./<pkg>/... -run <Name> -v`。
- 提交规范：Conventional Commits（`feat(strategy): ...`）。**未经用户明确要求不 push。**

## Verification 适配说明

每个任务结尾给出与其性质匹配的验证：领域服务/工具 → Go 单测；CRUD/路由 → `go build` + curl；代码生成 → hz 命令 + build；运行时装配 → build + 注释化的手动 agent 运行检查。能写单测处一律先写失败测试（TDD）。

---

## File Structure（决策锁定）

```
backend/
├─ api/model/resource/common/resource_common.go            # [改] +ResType_Strategy=10
├─ idl/data/strategy/strategy_svc.thrift                   # [新] CRUD+发布 IDL（镜像 database_svc.thrift）
├─ api/handler/coze/strategy_service.go                    # [生成/新] HTTP handler（镜像 database_service.go）
├─ api/router/coze/api.go                                  # [生成] 路由（hz 自动）
├─ application/strategy/strategy.go                        # [新] 应用服务 CRUD + 资源事件
├─ application/search/resource_pack.go                     # [改] +strategyPacker + 注册 switch
├─ domain/strategy/
│  ├─ entity/strategy.go                                   # [新] Strategy/Scenario/Capability 实体 + 常量
│  ├─ service/strategy.go                                  # [新] 领域服务接口 + req/resp
│  ├─ service/strategy_impl.go                             # [新] 领域服务实现
│  ├─ repository/repository.go                             # [新] DAO 接口
│  └─ internal/dal/
│     ├─ model/strategy.gen.go                             # [新] GORM 模型 ×3
│     ├─ model/strategy_scenario.gen.go
│     ├─ model/strategy_capability.gen.go
│     └─ query/...                                         # [新] gen query（或手写 DAO）
├─ domain/agent/singleagent/internal/agentflow/
│  ├─ node_tool_strategy.go                                # [新] newStrategyTools + 3 工具 + 分发
│  ├─ agent_flow_builder.go                                # [改] append 策略工具 + strategiesRenderer
│  └─ system_prompt.go                                     # [改] +bound_strategies Jinja2 块
└─ domain/agent/singleagent/entity（SingleAgent 配置）       # [改] +Strategies 绑定字段
docs/ynet-database-sql/01-ynet-studio.sql                  # [改] +3 张表 DDL
docker/atlas/migrations/                                   # [改] atlas migration（按 Makefile 流程）
```

---

## Task 1: 注册资源类型 ResType_Strategy

**Files:**
- Modify: `backend/api/model/resource/common/resource_common.go:28-40`（枚举）+ 同文件 `String()`/`ResTypeFromString()`

**Interfaces:**
- Produces: `common.ResType_Strategy ResType = 10`

- [ ] **Step 1: 加枚举值**

`resource_common.go` 现有：
```go
const (
	ResType_Plugin    ResType = 1
	// ...
	ResType_Voice     ResType = 9
)
```
改为追加一行：
```go
	ResType_Voice     ResType = 9
	ResType_Strategy  ResType = 10
)
```

- [ ] **Step 2: 补 String()/FromString() 映射**

在同文件的 `func (p ResType) String()` 的 switch 与 `ResTypeFromString(s string)` 的 map 中，仿 `ResType_Voice` 增加 `ResType_Strategy → "Strategy"` 双向映射（定位这两个函数后照葫芦画瓢；若由 thrift 生成，改对应 .thrift 后由 hz 再生成）。

- [ ] **Step 3: 验证 build**

Run: `cd backend && go build ./api/model/resource/common/...`
Expected: 成功，无报错。

- [ ] **Step 4: 提交**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
git add backend/api/model/resource/common/resource_common.go
git commit -m "feat(strategy): register ResType_Strategy=10"
```

---

## Task 2: 建 3 张表 + GORM 模型

**Files:**
- Modify: `docs/ynet-database-sql/01-ynet-studio.sql`（追加 3 张 CREATE TABLE）
- Create: `backend/domain/strategy/internal/dal/model/strategy.gen.go`
- Create: `backend/domain/strategy/internal/dal/model/strategy_scenario.gen.go`
- Create: `backend/domain/strategy/internal/dal/model/strategy_capability.gen.go`

**Interfaces:**
- Produces: GORM 模型 `model.Strategy` / `model.StrategyScenario` / `model.StrategyCapability`，表名 `strategy` / `strategy_scenario` / `strategy_capability`。

- [ ] **Step 1: 追加 DDL（镜像 database 表风格）**

向 `docs/ynet-database-sql/01-ynet-studio.sql` 追加：
```sql
CREATE TABLE IF NOT EXISTS `strategy` (
  `id` bigint unsigned NOT NULL COMMENT 'ID',
  `space_id` bigint unsigned NOT NULL COMMENT 'Space ID',
  `app_id` bigint unsigned DEFAULT NULL COMMENT 'App ID',
  `creator_id` bigint NOT NULL DEFAULT '0' COMMENT 'Creator ID',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Strategy name',
  `description` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Description / L1 hint',
  `icon_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Icon Uri',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '0 draft 1 published',
  `version` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Published version',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time ms',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time ms',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_space_app_creator_deleted` (`space_id`,`app_id`,`creator_id`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='strategy';

CREATE TABLE IF NOT EXISTS `strategy_scenario` (
  `id` bigint unsigned NOT NULL COMMENT 'ID',
  `strategy_id` bigint unsigned NOT NULL COMMENT 'Strategy ID',
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'Scenario name',
  `description` varchar(2000) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Model-facing description',
  `sort_order` int NOT NULL DEFAULT '0' COMMENT 'Sort order',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time ms',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time ms',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_strategy_deleted` (`strategy_id`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='strategy scenario';

CREATE TABLE IF NOT EXISTS `strategy_capability` (
  `id` bigint unsigned NOT NULL COMMENT 'ID',
  `strategy_id` bigint unsigned NOT NULL COMMENT 'Strategy ID',
  `scenario_id` bigint unsigned NOT NULL COMMENT 'Scenario ID',
  `type` varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'workflow/plugin/knowledge/prompt',
  `ref_id` bigint unsigned DEFAULT NULL COMMENT 'workflow_id / plugin_tool_id / knowledge_id',
  `ref_sub_id` bigint unsigned DEFAULT NULL COMMENT 'plugin_id (for plugin type)',
  `ref_version` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'workflow/plugin version',
  `prompt_content` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'inline prompt (prompt type)',
  `retrieve_config` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'json: top_k/min_score (knowledge type)',
  `alias_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT 'Model-facing name override',
  `alias_description` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci COMMENT 'Model-facing curated description',
  `sort_order` int NOT NULL DEFAULT '0' COMMENT 'Sort order',
  `created_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Create Time ms',
  `updated_at` bigint unsigned NOT NULL DEFAULT '0' COMMENT 'Update Time ms',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time',
  PRIMARY KEY (`id`),
  KEY `idx_scenario_deleted` (`scenario_id`,`deleted_at`),
  KEY `idx_strategy_deleted` (`strategy_id`,`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='strategy capability';
```

- [ ] **Step 2: 写 GORM 模型（镜像 `online_database_info.gen.go`）**

`backend/domain/strategy/internal/dal/model/strategy.gen.go`：
```go
package model

import "gorm.io/gorm"

const TableNameStrategy = "strategy"

type Strategy struct {
	ID          int64          `gorm:"column:id;primaryKey;comment:ID" json:"id"`
	SpaceID     int64          `gorm:"column:space_id;not null;comment:Space ID" json:"space_id"`
	AppID       int64          `gorm:"column:app_id;comment:App ID" json:"app_id"`
	CreatorID   int64          `gorm:"column:creator_id;not null;comment:Creator ID" json:"creator_id"`
	Name        string         `gorm:"column:name;not null;comment:Strategy name" json:"name"`
	Description string         `gorm:"column:description;comment:Description / L1 hint" json:"description"`
	IconURI     string         `gorm:"column:icon_uri;comment:Icon Uri" json:"icon_uri"`
	Status      int32          `gorm:"column:status;not null;default:0;comment:0 draft 1 published" json:"status"`
	Version     string         `gorm:"column:version;comment:Published version" json:"version"`
	CreatedAt   int64          `gorm:"column:created_at;not null;autoCreateTime:milli" json:"created_at"`
	UpdatedAt   int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*Strategy) TableName() string { return TableNameStrategy }
```

`strategy_scenario.gen.go`：
```go
package model

import "gorm.io/gorm"

const TableNameStrategyScenario = "strategy_scenario"

type StrategyScenario struct {
	ID          int64          `gorm:"column:id;primaryKey" json:"id"`
	StrategyID  int64          `gorm:"column:strategy_id;not null" json:"strategy_id"`
	Name        string         `gorm:"column:name;not null" json:"name"`
	Description string         `gorm:"column:description" json:"description"`
	SortOrder   int32          `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt   int64          `gorm:"column:created_at;not null;autoCreateTime:milli" json:"created_at"`
	UpdatedAt   int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*StrategyScenario) TableName() string { return TableNameStrategyScenario }
```

`strategy_capability.gen.go`：
```go
package model

import "gorm.io/gorm"

const TableNameStrategyCapability = "strategy_capability"

type StrategyCapability struct {
	ID               int64          `gorm:"column:id;primaryKey" json:"id"`
	StrategyID       int64          `gorm:"column:strategy_id;not null" json:"strategy_id"`
	ScenarioID       int64          `gorm:"column:scenario_id;not null" json:"scenario_id"`
	Type             string         `gorm:"column:type;not null" json:"type"`
	RefID            int64          `gorm:"column:ref_id" json:"ref_id"`
	RefSubID         int64          `gorm:"column:ref_sub_id" json:"ref_sub_id"`
	RefVersion       string         `gorm:"column:ref_version" json:"ref_version"`
	PromptContent    string         `gorm:"column:prompt_content" json:"prompt_content"`
	RetrieveConfig   string         `gorm:"column:retrieve_config" json:"retrieve_config"`
	AliasName        string         `gorm:"column:alias_name" json:"alias_name"`
	AliasDescription string         `gorm:"column:alias_description" json:"alias_description"`
	SortOrder        int32          `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt        int64          `gorm:"column:created_at;not null;autoCreateTime:milli" json:"created_at"`
	UpdatedAt        int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*StrategyCapability) TableName() string { return TableNameStrategyCapability }
```

- [ ] **Step 3: 生成 atlas migration（按现有 Makefile 流程）**

查 `Makefile` 中 `dump_sql_schema` / `atlas-hash` 目标，按既有流程生成 `docker/atlas/migrations/` 下的迁移并更新 hash。若仓库用启动时导入 SQL，则确保新表进了 `01-ynet-studio.sql` 即可。

- [ ] **Step 4: 验证 build**

Run: `cd backend && go build ./domain/strategy/...`
Expected: 成功。

- [ ] **Step 5: 提交**

```bash
git add backend/domain/strategy/internal/dal/model/ docs/ynet-database-sql/01-ynet-studio.sql docker/atlas/migrations/
git commit -m "feat(strategy): add strategy/scenario/capability tables and gorm models"
```

---

## Task 3: 领域实体 + DAO（仓储）

**Files:**
- Create: `backend/domain/strategy/entity/strategy.go`
- Create: `backend/domain/strategy/repository/repository.go`
- Create: `backend/domain/strategy/internal/dal/strategy_dao.go`（GORM 实现）
- Test: `backend/domain/strategy/internal/dal/strategy_dao_test.go`

**Interfaces:**
- Produces:
  - 类型常量 `entity.CapabilityTypeWorkflow="workflow"`、`...Plugin="plugin"`、`...Knowledge="knowledge"`、`...Prompt="prompt"`
  - 实体 `entity.Strategy{ID,SpaceID,AppID,CreatorID,Name,Description,IconURI,Status,Version}`、`entity.Scenario{ID,StrategyID,Name,Description,SortOrder}`、`entity.Capability{ID,StrategyID,ScenarioID,Type,RefID,RefSubID,RefVersion,PromptContent,RetrieveConfig,AliasName,AliasDescription,SortOrder}`
  - DAO 接口 `repository.StrategyDAO`（见 Step 2）

- [ ] **Step 1: 实体 + 常量**

`entity/strategy.go`：
```go
package entity

type CapabilityType = string

const (
	CapabilityTypeWorkflow  CapabilityType = "workflow"
	CapabilityTypePlugin    CapabilityType = "plugin"
	CapabilityTypeKnowledge CapabilityType = "knowledge"
	CapabilityTypePrompt    CapabilityType = "prompt"
)

const (
	StatusDraft     int32 = 0
	StatusPublished int32 = 1
)

type Strategy struct {
	ID, SpaceID, AppID, CreatorID int64
	Name, Description, IconURI    string
	Status                        int32
	Version                       string
	Scenarios                     []*Scenario // GetDetail 时填充
}

type Scenario struct {
	ID, StrategyID int64
	Name, Description string
	SortOrder      int32
	Capabilities   []*Capability // GetDetail 时填充
}

type Capability struct {
	ID, StrategyID, ScenarioID       int64
	Type                             CapabilityType
	RefID, RefSubID                  int64
	RefVersion                       string
	PromptContent, RetrieveConfig    string
	AliasName, AliasDescription      string
	SortOrder                        int32
}
```

- [ ] **Step 2: DAO 接口**

`repository/repository.go`：
```go
package repository

import (
	"context"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

type StrategyDAO interface {
	CreateStrategy(ctx context.Context, s *entity.Strategy) (int64, error)
	UpdateStrategy(ctx context.Context, s *entity.Strategy) error
	DeleteStrategy(ctx context.Context, id int64) error
	GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error)
	ListStrategy(ctx context.Context, spaceID int64, page, size int) ([]*entity.Strategy, int64, error)
	PublishStrategy(ctx context.Context, id int64, version string) error

	CreateScenario(ctx context.Context, sc *entity.Scenario) (int64, error)
	UpdateScenario(ctx context.Context, sc *entity.Scenario) error
	DeleteScenario(ctx context.Context, id int64) error
	ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error)

	CreateCapability(ctx context.Context, c *entity.Capability) (int64, error)
	UpdateCapability(ctx context.Context, c *entity.Capability) error
	DeleteCapability(ctx context.Context, id int64) error
	ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error)
	MGetCapabilities(ctx context.Context, ids []int64) ([]*entity.Capability, error)
	ListCapabilityIDsByStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error) // 越权校验用
}
```

- [ ] **Step 3: 写失败测试（GORM DAO，sqlite 内存或现有测试 harness）**

`strategy_dao_test.go`（用仓库现有 DB 测试约定；若有 `dal` 包的测试 helper 则复用）：
```go
func TestStrategyDAO_CreateAndGet(t *testing.T) {
	dao := newTestStrategyDAO(t) // helper：建内存库+automigrate 3 张表
	id, err := dao.CreateStrategy(context.Background(), &entity.Strategy{
		SpaceID: 1, CreatorID: 2, Name: "对公运营", Description: "银行对公",
	})
	require.NoError(t, err)
	got, err := dao.GetStrategy(context.Background(), id)
	require.NoError(t, err)
	require.Equal(t, "对公运营", got.Name)
}
```

Run: `cd backend && go test ./domain/strategy/internal/dal/... -run TestStrategyDAO_CreateAndGet -v`
Expected: FAIL（`newTestStrategyDAO`/实现未定义）。

- [ ] **Step 4: 实现 GORM DAO**

`internal/dal/strategy_dao.go`：实现 `repository.StrategyDAO`，用 `gorm.DB` + `model.*`，ID 由现有 ID 生成器（参考 database service 的 `d.generator.GenID(ctx)`）。entity↔model 转换在本文件内。`ListCapabilityIDsByStrategies` 用 `WHERE strategy_id IN ?` 查 `strategy_capability.id`。

- [ ] **Step 5: 测试通过**

Run: `cd backend && go test ./domain/strategy/internal/dal/... -v`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add backend/domain/strategy/entity backend/domain/strategy/repository backend/domain/strategy/internal/dal
git commit -m "feat(strategy): add strategy domain entities and gorm DAO"
```

---

## Task 4: 领域服务（CRUD + 发布 + 解析）

**Files:**
- Create: `backend/domain/strategy/service/strategy.go`（接口 + req/resp）
- Create: `backend/domain/strategy/service/strategy_impl.go`
- Test: `backend/domain/strategy/service/strategy_impl_test.go`

**Interfaces:**
- Consumes: `repository.StrategyDAO`（Task 3）
- Produces: `service.Strategy` 接口，方法覆盖策略/场景/能力项 CRUD + `Publish` + `GetDetail`(含 scenarios/capabilities) + `ResolveCapability(ctx, id)(*entity.Capability,error)` + `ListCapabilityIDsByAgentStrategies(ctx, strategyIDs)([]int64,error)`。

- [ ] **Step 1: 接口 + req/resp 结构**

`service/strategy.go`：定义 `type Strategy interface { ... }`，方法签名以 `entity.*` 为出入参（仿 `domain/memory/database/service/database.go` 风格：`CreateXxxRequest/Response` 包装）。

- [ ] **Step 2: 写失败测试（用 mock DAO）**

```go
func TestStrategyService_GetDetail(t *testing.T) {
	dao := newMockDAO() // 返回 1 策略 / 1 场景 / 2 能力项
	svc := service.NewStrategyService(dao, fakeIDGen)
	d, err := svc.GetDetail(context.Background(), 100)
	require.NoError(t, err)
	require.Len(t, d.Scenarios, 1)
	require.Len(t, d.Scenarios[0].Capabilities, 2)
}
```

Run: `cd backend && go test ./domain/strategy/service/... -run TestStrategyService_GetDetail -v`
Expected: FAIL。

- [ ] **Step 3: 实现服务**

`strategy_impl.go`：`GetDetail` 组装 strategy→scenarios→capabilities 树；`Publish` 置 `status=1`+`version`；`ResolveCapability` = `dao.MGetCapabilities([id])[0]`；`ListCapabilityIDsByAgentStrategies` 透传 DAO。CRUD 直接委托 DAO + 基本校验（name 非空等）。

- [ ] **Step 4: 测试通过**

Run: `cd backend && go test ./domain/strategy/service/... -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/domain/strategy/service
git commit -m "feat(strategy): add strategy domain service (crud/publish/resolve)"
```

---

## Task 5: IDL + 应用层 + HTTP 接口

**Files:**
- Create: `idl/data/strategy/strategy_svc.thrift`（镜像 `idl/data/database/database_svc.thrift`）
- Generate/Create: `backend/api/handler/coze/strategy_service.go`
- Modify (generated): `backend/api/router/coze/api.go`
- Create: `backend/application/strategy/strategy.go`

**Interfaces:**
- Consumes: `service.Strategy`（Task 4）
- Produces: HTTP 路由 `/api/strategy/*`；`strategyApp.StrategyApplicationSVC`

- [ ] **Step 1: 写 IDL（镜像 database_svc.thrift）**

`idl/data/strategy/strategy_svc.thrift` 定义 service + 请求/响应（List/Get/Add/Update/Delete/Publish 策略；场景与能力项的增删改排序）。路由前缀 `/api/strategy/...`，`api.post` 注解仿 database。

- [ ] **Step 2: 代码生成**

按仓库 hz 流程（参考 `database` 生成方式）：
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
hz update -idl idl/data/strategy/strategy_svc.thrift
```
生成 model（`backend/api/model/data/strategy/...`）+ handler 桩 + 路由。

- [ ] **Step 3: 应用层（镜像 `application/memory/database.go`）**

`application/strategy/strategy.go`：`StrategyApplicationService{ DomainSVC service.Strategy; eventbus search.ResourceEventBus }` + `StrategyApplicationSVC` 单例；每个方法做 uid/space 鉴权（仿 database 的 `crossuser.DefaultSVC().GetUserSpaceList`），调 DomainSVC，Add/Publish 后 `eventbus.PublishResources(... ResType_Strategy ...)`（仿 database `AddDatabase`）。

- [ ] **Step 4: handler 接线（镜像 `database_service.go`）**

`backend/api/handler/coze/strategy_service.go`：每个 handler `BindAndValidate` → 调 `strategyApp.StrategyApplicationSVC.Xxx` → `c.JSON`。确认 `api.go` 路由已注册（hz 生成）。

- [ ] **Step 5: 验证 build + curl 冒烟**

Run: `cd backend && go build ./...`
Expected: 成功。
启动后端后：
```bash
curl -s -X POST localhost:8888/api/strategy/add -H 'Content-Type: application/json' \
  -d '{"space_id":<sid>,"name":"测试策略","description":"d"}'
```
Expected: 返回新建策略 id。

- [ ] **Step 6: 提交**

```bash
git add idl/data/strategy backend/api/model/data/strategy backend/api/handler/coze/strategy_service.go backend/api/router/coze/api.go backend/application/strategy
git commit -m "feat(strategy): add strategy CRUD idl, handlers and application service"
```

---

## Task 6: 资源库列表接入（strategyPacker）

**Files:**
- Modify: `backend/application/search/resource_pack.go`（+`strategyPacker` + `NewResourcePacker` switch）

**Interfaces:**
- Consumes: `service.Strategy`（经 `ServiceComponents`）、`common.ResType_Strategy`

- [ ] **Step 1: 写 strategyPacker（镜像 `databasePacker`）**

在 `resource_pack.go` 增加：
```go
type strategyPacker struct{ resourceBasePacker }

func (s *strategyPacker) GetDataInfo(ctx context.Context) (*dataInfo, error) {
	st, err := s.appContext.StrategyDomainSVC.GetDetail(ctx, s.resID)
	if err != nil { return nil, err }
	return &dataInfo{ iconURI: ptr.Of(st.IconURI), desc: ptr.Of(st.Description) }, nil
}
func (s *strategyPacker) GetActions(ctx context.Context) []*common.ResourceAction {
	return []*common.ResourceAction{{Key: common.ActionKey_Delete, Enable: true}}
}
```
（如 `ServiceComponents` 无 `StrategyDomainSVC` 字段则补一个，并在依赖装配处注入。）

- [ ] **Step 2: 注册进 switch**

`NewResourcePacker` 增加：
```go
	case common.ResType_Strategy:
		return &strategyPacker{resourceBasePacker: base}, nil
```

- [ ] **Step 3: 验证 build**

Run: `cd backend && go build ./application/search/...`
Expected: 成功。

- [ ] **Step 4: 提交**

```bash
git add backend/application/search/resource_pack.go
git commit -m "feat(strategy): integrate strategy into resource library packer"
```

---

## Task 7: 单智能体绑定字段

**Files:**
- Modify: `backend/domain/agent/singleagent/entity`（`SingleAgent` 配置结构 + 草稿/发布持久化）

**Interfaces:**
- Produces: `SingleAgent.Strategies []*BoundStrategy{ StrategyID int64; Version string }`，并被 `Config.Agent` 携带进 agent flow builder。

- [ ] **Step 1: 加绑定结构**

在 `SingleAgent` 配置（bot skill 配置部分）增加 `Strategies []*BoundStrategy`，定义 `BoundStrategy{StrategyID, Version}`。

- [ ] **Step 2: 持久化往返**

在单智能体草稿读写（draft get/update）与发布（publish）处，仿 `workflows`/`database` 的存取，把 `Strategies` 读出/写入（DTO↔entity）。定位现有 workflow 绑定字段的存取点，照其增加 strategies 分支。

- [ ] **Step 3: 验证 build + 往返测试**

Run: `cd backend && go build ./domain/agent/...`
若有单智能体序列化测试，加一条断言 strategies 往返不丢；否则 build 通过即可。

- [ ] **Step 4: 提交**

```bash
git add backend/domain/agent/singleagent
git commit -m "feat(strategy): persist bound strategies on single agent config"
```

---

## Task 8: 3 个工具 + list 实现

**Files:**
- Create: `backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy.go`
- Test: `backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy_test.go`

**Interfaces:**
- Consumes: `service.Strategy`（经 cross-domain contract；若无则新增 `crossstrategy` 契约，仿 `crossplugin`/`crossworkflow`）
- Produces:
  - `type strategyConfig struct{ spaceID int64; userID string; agentIdentity *entity.AgentIdentity; strategyIDs []int64 }`
  - `func newStrategyTools(ctx, conf *strategyConfig)([]tool.InvokableTool, error)` 返回 3 个工具
  - 三个工具结构 `listScenariosTool`/`listCapabilitiesTool`/`invokeCapabilityTool`，各实现 eino `Info()`+`InvokableRun()`

- [ ] **Step 1: 写失败测试（Info + list）**

```go
func TestStrategyTools_Info(t *testing.T) {
	tools, err := newStrategyTools(context.Background(), &strategyConfig{strategyIDs: []int64{1}, svc: fakeStrategySvc()})
	require.NoError(t, err)
	require.Len(t, tools, 3)
	info0, _ := tools[0].Info(context.Background())
	require.Equal(t, "strategy_list_scenarios", info0.Name)
}

func TestStrategyTools_ListCapabilities(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{strategyIDs: []int64{1}, svc: fakeStrategySvc()})
	out, err := tools[1].InvokableRun(context.Background(), `{"scenario_id":"10"}`)
	require.NoError(t, err)
	require.Contains(t, out, `"type":"prompt"`) // fake 返回含 prompt 能力项
}
```

Run: `cd backend && go test ./domain/agent/singleagent/internal/agentflow/ -run TestStrategyTools -v`
Expected: FAIL。

- [ ] **Step 2: 实现 3 工具（eino `tool.InvokableTool`，镜像 `node_tool_plugin.go`/`node_tool_sandbox.go`）**

每个工具 struct 持有 `svc service.Strategy` + `conf`。`Info()` 用 `schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{...})` 声明上面文档 §4.3 的入参 schema。`listScenariosTool.InvokableRun` → `svc.ListScenarios` 取场景（带 capability_count）→ JSON 串。`listCapabilitiesTool.InvokableRun` → `svc.ListCapabilities(scenario_id)`，每项映射 `{capability_id,type,name(alias优先),description(alias优先),input_schema}`：
  - workflow：取工作流入参 schema（`crossworkflow` 查 workflow 的 input params 转 JSON schema）
  - plugin：取插件工具 operation 转 JSON schema
  - knowledge：固定 `{query:string, top_k?:int}`
  - prompt：空对象 `{}`
`invokeCapabilityTool` 在 Task 9 实现 InvokableRun 主体；本任务先建 struct 与 Info。

- [ ] **Step 3: 测试通过（Info + list）**

Run: `cd backend && go test ./domain/agent/singleagent/internal/agentflow/ -run TestStrategyTools -v`
Expected: PASS。

- [ ] **Step 4: 提交**

```bash
git add backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy.go backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy_test.go
git commit -m "feat(strategy): add 3 progressive-disclosure tools (scenarios/capabilities list)"
```

---

## Task 9: invoke 分发（4 类）

**Files:**
- Modify: `backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy.go`（`invokeCapabilityTool.InvokableRun`）
- Test: 同 `node_tool_strategy_test.go`

**Interfaces:**
- Consumes: `crossworkflow`（workflow 执行）、`crossplugin.DefaultSVC().ExecuteTool`（plugin）、knowledge recall 跨域服务（mirror `externalKnowledgeTools` 在 `agent_flow_builder.go` 用到的 recall 通路）、`service.Strategy.ResolveCapability` + `ListCapabilityIDsByAgentStrategies`

- [ ] **Step 1: 写失败测试（4 类 + 越权）**

```go
func TestInvoke_Prompt(t *testing.T) {
	tool := newInvokeTool(strategyIDs(1), fakeSvcWithPrompt("cap=5", "对公开户合规要点..."))
	out, err := tool.InvokableRun(ctx, `{"capability_id":"5"}`)
	require.NoError(t, err)
	require.Contains(t, out, "对公开户合规要点")
}
func TestInvoke_RejectsUnbound(t *testing.T) {
	tool := newInvokeTool(strategyIDs(1), fakeSvcCapNotInStrategy("99"))
	_, err := tool.InvokableRun(ctx, `{"capability_id":"99"}`)
	require.Error(t, err) // 越权
}
```

Run: `cd backend && go test ./domain/agent/singleagent/internal/agentflow/ -run TestInvoke -v`
Expected: FAIL。

- [ ] **Step 2: 实现分发**

```go
func (t *invokeCapabilityTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var req struct{ CapabilityID string `json:"capability_id"`; Arguments json.RawMessage `json:"arguments"` }
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil { return argParseErrMsg(err), nil }
	capID, _ := strconv.ParseInt(req.CapabilityID, 10, 64)

	// 越权校验
	allowed, err := t.svc.ListCapabilityIDsByAgentStrategies(ctx, t.conf.strategyIDs)
	if err != nil { return "", err }
	if !contains(allowed, capID) { return "Error: capability not bound to this agent", nil }

	cap, err := t.svc.ResolveCapability(ctx, capID)
	if err != nil { return "", err }

	switch cap.Type {
	case entity.CapabilityTypePrompt:
		return cap.PromptContent, nil
	case entity.CapabilityTypeWorkflow:
		return t.runWorkflow(ctx, cap, string(req.Arguments)) // 复用 crossworkflow 执行
	case entity.CapabilityTypePlugin:
		return t.runPlugin(ctx, cap, string(req.Arguments))    // 复用 crossplugin.ExecuteTool
	case entity.CapabilityTypeKnowledge:
		return t.runKnowledge(ctx, cap, string(req.Arguments)) // 复用 knowledge recall
	default:
		return "Error: unknown capability type", nil
	}
}
```
- `runWorkflow`：仿 `node_tool_workflow.go`/`workflow_tool.go`，用 `cap.RefID`(+`RefVersion`) 取 workflow tool 并 invoke arguments。
- `runPlugin`：构造 `service.ExecuteToolRequest{ PluginID: cap.RefSubID, ToolID: cap.RefID, ArgumentsInJson: args, ... }` 调 `crossplugin.DefaultSVC().ExecuteTool`（镜像 `pluginInvokableTool.InvokableRun`）。
- `runKnowledge`：解析 `{query, top_k}`，调 knowledge recall 跨域服务（定位 `externalKnowledgeTools` 所用 recall；用 `cap.RefID` 作 dataset/knowledge id），返回召回片段拼接文本。

- [ ] **Step 3: 测试通过**

Run: `cd backend && go test ./domain/agent/singleagent/internal/agentflow/ -run TestInvoke -v`
Expected: PASS（prompt/越权先绿；workflow/plugin/knowledge 用 fake 跨域 svc 验证分支路由）。

- [ ] **Step 4: 提交**

```bash
git add backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy.go backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy_test.go
git commit -m "feat(strategy): implement invoke_capability dispatch (workflow/plugin/knowledge/prompt)"
```

---

## Task 10: 运行时装配 + L1 提示注入

**Files:**
- Modify: `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go`（append 工具 + strategiesRenderer 节点 + 边）
- Modify: `backend/domain/agent/singleagent/internal/agentflow/system_prompt.go`（Jinja2 块）

**Interfaces:**
- Consumes: `newStrategyTools`（Task 8/9）、`conf.Agent.Strategies`（Task 7）

- [ ] **Step 1: append 策略工具**

在 `agent_flow_builder.go` 现有 `agentTools = append(...)`（239–255 行）之后插入：
```go
	var strategyTools []tool.InvokableTool
	if len(conf.Agent.Strategies) > 0 {
		strategyTools, err = newStrategyTools(ctx, &strategyConfig{
			spaceID:       conf.Agent.SpaceID,
			userID:        conf.UserID,
			agentIdentity: conf.Identity,
			strategyIDs:   strategyIDsOf(conf.Agent.Strategies),
			svc:           crossstrategy.DefaultSVC(),
		})
		if err != nil { return nil, err }
	}
	agentTools = append(agentTools, slices.Transform(strategyTools, func(a tool.InvokableTool) tool.BaseTool { return a })...)
```

- [ ] **Step 2: L1 提示块**

`system_prompt.go` 的 `REACT_SYSTEM_PROMPT_JINJA2`，在 `available_skills` 块后加：
```jinja2
{% if bound_strategies %}
----- Start Of Strategies -----
{{ bound_strategies }}
----- End Of Strategies -----
{% endif %}
```
并在 `agent_flow_builder.go` 增加 `const placeholderOfBoundStrategies = "bound_strategies"`、`strategiesRenderer`（lambda：把 `conf.Agent.Strategies` 渲染成 "你绑定了策略「X」，处理相关请求前先调用 strategy_list_scenarios"）+ `AddLambdaNode(keyOfStrategiesRender, ..., WithOutputKey(placeholderOfBoundStrategies))` + `AddEdge(keyOfStrategiesRender, keyOfPromptTemplate)`（仿 skillsRenderer 548–597 行）。无绑定时渲染空串（Jinja2 `{% if %}` 跳过）。

- [ ] **Step 3: 验证 build + 手动 agent 运行**

Run: `cd backend && go build ./...`
Expected: 成功。
手动：给一个测试 agent 绑定一个策略（直接写 DB 或经 Task 5 接口），跑一次对话，确认：系统提示出现策略引导句；模型可依次调用 3 工具；prompt 类型能力项返回内联文本。

- [ ] **Step 4: 提交**

```bash
git add backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go backend/domain/agent/singleagent/internal/agentflow/system_prompt.go
git commit -m "feat(strategy): inject strategy tools and L1 prompt hint into single agent runtime"
```

---

## Self-Review（落计划时执行）

- **Spec 覆盖**：§4.1→T1；§4.2→T2；§4.7/DAO→T3；服务→T4；§4.6 API→T5；资源库→T6；§4.4 绑定→T7；§4.3 三工具→T8；invoke 四类分发→T9；§4.5 运行时+L1→T10。全覆盖。
- **越权**：T9 Step2 显式校验。**失效保护**：runWorkflow/runPlugin/runKnowledge 对 ref 失效返回 `Error: ...` 文本而非 panic（实现时在各 run* 包 err→友好串）。
- **类型一致**：工具名/能力项类型常量在 Global Constraints 固定，T8/T9 复用同名常量。
- **跨域契约**：T8/T9 依赖 `crossstrategy.DefaultSVC()`；若仓库要求跨域调用走 contract，则在 `backend/crossdomain/contract/strategy` 新增接口并在装配处注册（仿 `crossplugin`）。此为 T8 的前置子步骤。

## 依赖顺序

T1 → T2 → T3 → T4 →（T5、T6、T7 可并行）→ T8 → T9 → T10。
