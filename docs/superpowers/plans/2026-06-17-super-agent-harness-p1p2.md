# 超级智能体（Harness Agent）P1+P2 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 Studio 新增「超级智能体」类型（`agent_type=super`）：普通 agent 零改动；超级 agent 在现有运行时上挂 `deep_task` 工具委托 eino DeepAgent 自治，并拥有 `/workspace /uploads /outputs` 固定空间契约。

**Architecture:** 复用现有 `single_agent` 实体+表，加 `agent_type` 列做路由。运行时 `BuildAgent` 读 `AgentType`，`super` 时启用 `deep_task` 工具（`adk.NewAgentTool(deep.New(...))`，eino 原生 agent-as-tool，无需 adk↔compose 桥接）+ 超级 agent 系统提示词。流式/中断/checkpoint 全复用现有 compose/ReAct 链路。

**Tech Stack:** Go 1.25 · eino(compose/react/adk/prebuilt/deep) · gorm.io/gen · testify/suite + goconvey · MySQL 迁移(docker/migrations + docs/ynet-database-sql)

**对应 spec:** `docs/superpowers/specs/2026-06-17-super-agent-harness-design.md`（P1 地基 + P2 引擎跑顺）

**前置:** 分支 `feat/agent-sandbox-superagent`。基线 `cd backend && SESSION_HMAC_SECRET=test go build ./...` 通过。`node_tool_deeptask.go` 已存在（本计划 Task 4 将其完善并接入）。

---

## 文件结构（本计划将创建/修改）

- 加列：`api/model/crossdomain/singleagent/single_agent.go`（实体）
- 加列：`domain/agent/singleagent/internal/dal/model/single_agent_draft.gen.go` + `single_agent_version.gen.go`（PO）
- 改转换：`domain/agent/singleagent/internal/dal/single_agent_draft.go` + `single_agent_version.go`（PO↔Entity 双向）
- 映射：`application/singleagent/create.go`（创建落 AgentType）
- 路由：`domain/agent/singleagent/internal/agentflow/agent_flow_builder.go`（读 AgentType 分支）
- 引擎：`domain/agent/singleagent/internal/agentflow/node_tool_deeptask.go`（deep_task 工具，已存在，完善）
- 提示词：`domain/agent/singleagent/internal/agentflow/system_prompt.go`（超级 agent 纪律）
- 空间：`backend/pkg/agentsandbox/manager.go`（EnsureWorkspaceLayout：建 uploads/outputs）
- 迁移：`docker/migrations/20260617_add_agent_type.sql`（新）+ `docs/ynet-database-sql/01-ynet-studio.sql`（schema）
- 测试：`domain/agent/singleagent/internal/dal/single_agent_draft_test.go`（持久化）、`agentflow/*_test.go`（路由/deep_task/提示词）、`pkg/agentsandbox/*_test.go`（空间契约）

常量约定：`agent_type` 取值 `""`/`normal`=普通；`super`=超级智能体。Go 侧定义常量 `entitySuperAgentType = "super"`（放 agentflow 包，见 Task 3）。

---

## Task 1: 数据模型加 `agent_type` 字段（实体 + PO + 转换）

**Files:**
- Modify: `backend/api/model/crossdomain/singleagent/single_agent.go:87`
- Modify: `backend/domain/agent/singleagent/internal/dal/model/single_agent_draft.gen.go`
- Modify: `backend/domain/agent/singleagent/internal/dal/model/single_agent_version.gen.go`
- Modify: `backend/domain/agent/singleagent/internal/dal/single_agent_draft.go`（两个转换函数）
- Modify: `backend/domain/agent/singleagent/internal/dal/single_agent_version.go`（两个转换函数）

- [ ] **Step 1: 实体加字段**

`single_agent.go`，在 `ForceToolReturn *bool` 之后加：
```go
	AgentType               string // agent_type：""/normal=普通；super=超级智能体（运行时路由）
```

- [ ] **Step 2: PO 加字段（两张表）**

`single_agent_draft.gen.go` 和 `single_agent_version.gen.go`，各自在 `ForceToolReturn *bool ...` 字段之后加：
```go
	AgentType               *string                           `gorm:"column:agent_type;comment:Agent Type for Runtime Routing" json:"agent_type"`
```

- [ ] **Step 3: 改转换函数（draft 表，PO→Entity 与 Entity→PO）**

`single_agent_draft.go` 的 `singleAgentDraftPo2Do`，在 `ForceToolReturn: po.ForceToolReturn,` 后加：
```go
			AgentType:               ptr.From(po.AgentType),
```
`singleAgentDraftDo2Po`，在 `ForceToolReturn: do.ForceToolReturn,` 后加：
```go
		AgentType:               ptr.Of(do.AgentType),
```
确认文件已 import `"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"`（未 import 则加）。

- [ ] **Step 4: 改转换函数（version 表，同 draft）**

`single_agent_version.go` 的 PO→Entity 加 `AgentType: ptr.From(po.AgentType),`；Entity→PO 加 `AgentType: ptr.Of(do.AgentType),`。

- [ ] **Step 5: 编译验证**

Run: `cd backend && SESSION_HMAC_SECRET=test go build ./domain/agent/singleagent/... ./api/model/crossdomain/singleagent/...`
Expected: 退出码 0，无报错。

- [ ] **Step 6: 提交**

```bash
git add backend/api/model/crossdomain/singleagent/single_agent.go \
  backend/domain/agent/singleagent/internal/dal/
git commit -m "feat(agent): add agent_type field through entity/PO/conversions"
```

---

## Task 2: 持久化单测（agent_type 落库与读取）

**Files:**
- Test: `backend/domain/agent/singleagent/internal/dal/single_agent_draft_test.go`

- [ ] **Step 1: 写失败测试**

在该测试 suite 里加（参照文件现有 `SetupSuite` 已注册 `model.SingleAgentDraft`）：
```go
func (s *SingleAgentDraftSuite) TestAgentTypePersist() {
	PatchConvey("agent_type 落库与读取", s.T(), func() {
		ctx := s.ctx
		at := "super"
		err := s.dao.dbQuery.SingleAgentDraft.WithContext(ctx).Create(&model.SingleAgentDraft{
			AgentID: 90001, SpaceID: 100, Name: "super_test", IconURI: "u", AgentType: &at,
		})
		So(err, ShouldBeNil)
		got, err := s.dao.Get(ctx, 90001)
		So(err, ShouldBeNil)
		So(got.AgentType, ShouldEqual, "super")
	})
}
```
> 注：若该 suite 的 `Get` 签名不同，按文件内现有 `Get`/查询方法调整；目标是验证 `AgentType` 经 PO→Entity 转换后等于 "super"。

- [ ] **Step 2: 跑测试确认失败/通过**

Run: `cd backend && SESSION_HMAC_SECRET=test go test ./domain/agent/singleagent/internal/dal/ -run TestSingleAgentDraftSuite/TestAgentTypePersist -v`
Expected: 字段映射正确则 PASS；若 PASS 前先确认编译，红了按报错修。

- [ ] **Step 3: 提交**

```bash
git add backend/domain/agent/singleagent/internal/dal/single_agent_draft_test.go
git commit -m "test(agent): agent_type persist round-trip"
```

---

## Task 3: 运行时路由（BuildAgent 读 AgentType）

**Files:**
- Create: `backend/domain/agent/singleagent/internal/agentflow/super_agent.go`
- Modify: `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go`（在 deep_task 挂载处用 `isSuperAgent(conf)` 判定）

- [ ] **Step 1: 新建 super_agent.go（常量 + 判定）**

```go
package agentflow

// SuperAgentType 是超级智能体的 agent_type 取值。
const SuperAgentType = "super"

// isSuperAgent 报告该 agent 是否为超级智能体。
func isSuperAgent(conf *Config) bool {
	return conf != nil && conf.Agent != nil && conf.Agent.AgentType == SuperAgentType
}
```

- [ ] **Step 2: 写路由单测**

`super_agent_test.go`：
```go
package agentflow

import (
	"testing"

	crossentity "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
)

func TestIsSuperAgent(t *testing.T) {
	mk := func(t string) *Config {
		return &Config{Agent: &entity.SingleAgent{SingleAgent: &crossentity.SingleAgent{AgentType: t}}}
	}
	if isSuperAgent(mk("super")) != true {
		t.Fatal("super => true")
	}
	if isSuperAgent(mk("")) != false || isSuperAgent(mk("normal")) != false {
		t.Fatal("empty/normal => false")
	}
	if isSuperAgent(nil) != false {
		t.Fatal("nil => false")
	}
}
```
> 确认 `entity.SingleAgent` 内嵌字段名（Task 1 探查显示 `entity.SingleAgent{ *crossentity.SingleAgent }`）；import 路径以仓库实际为准。

- [ ] **Step 3: 跑测试**

Run: `cd backend && SESSION_HMAC_SECRET=test go test ./domain/agent/singleagent/internal/agentflow/ -run TestIsSuperAgent -v`
Expected: PASS。

- [ ] **Step 4: 提交**

```bash
git add backend/domain/agent/singleagent/internal/agentflow/super_agent.go \
  backend/domain/agent/singleagent/internal/agentflow/super_agent_test.go
git commit -m "feat(agent): super agent type detection (isSuperAgent)"
```

---

## Task 4: deep_task 工具接入（超级 agent 启用）

**Files:**
- Modify: `backend/domain/agent/singleagent/internal/agentflow/node_tool_deeptask.go`（已存在；改 gate）
- Modify: `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go:313`（agentTools 组装完、`var isReActAgent` 之前插入挂载）

- [ ] **Step 1: 改 deep_task 的启用条件**

`node_tool_deeptask.go` 的 `deepTaskEnabled()` 改为接受 super 判定：保留 env 开关给普通 agent，超级 agent 默认开。即在 builder 调用处用 `isSuperAgent(conf) || deepTaskEnabled()`。`node_tool_deeptask.go` 本身不改逻辑，仅确认 `newDeepTaskTool(ctx, chatModel, agentTools)` 签名可用。

- [ ] **Step 2: builder 挂载 deep_task**

`agent_flow_builder.go`，在第 313 行（`if deepAgentsEnabled() {...}` 块之后）、第 315 行 `var isReActAgent bool` 之前插入：
```go
	if isSuperAgent(conf) || deepTaskEnabled() {
		if dt, derr := newDeepTaskTool(ctx, chatModel, agentTools); derr != nil {
			logs.CtxWarnf(ctx, "[BuildAgent] build deep_task tool failed: %v", derr)
		} else if dt != nil {
			agentTools = append(agentTools, dt)
			logs.CtxInfof(ctx, "[BuildAgent] mounted deep_task tool (super=%v)", isSuperAgent(conf))
		}
	}
```
> deep agent 用挂载前的 agentTools 构建（不含 deep_task），避免递归；append 在其后。`chatModel` 已在 BuildAgent 作用域内（见同文件 reactConfig.ToolCallingModel: chatModel）。

- [ ] **Step 3: 编译验证（含 eino adk/deep 依赖）**

Run: `cd backend && SESSION_HMAC_SECRET=test go build ./domain/agent/singleagent/...`
Expected: 退出码 0。若 `adk.NewAgentTool`/`deep.New` 报签名不符，按 `go doc github.com/cloudwego/eino/adk NewAgentTool` 与 `go doc github.com/cloudwego/eino/adk/prebuilt/deep.Config` 校正。

- [ ] **Step 4: 工具计数单测**

在 `agentflow` 包加 `deep_task_test.go`：构造一个最小 `fakeToolCallingChatModel`（或复用现有），调用 `newDeepTaskTool(ctx, model, nil)`，断言返回的 `tool.BaseTool` 的 `Info(ctx).Name == "deep_task"`。
```go
func TestNewDeepTaskToolInfo(t *testing.T) {
	// 用现有可构造的 fake chat model；若无，跳过并标注需要真实 model 的集成测试。
	// 断言： info.Name == deepTaskName ("deep_task")
}
```
> 若包内无可用 fake ToolCallingChatModel，本步降级为「构建期不 panic + 编译通过」，并在 Task 6 的 220 E2E 做真实验证（标注）。

- [ ] **Step 5: 提交**

```bash
git add backend/domain/agent/singleagent/internal/agentflow/
git commit -m "feat(agent): mount deep_task tool for super agents (DeepAgent delegation)"
```

---

## Task 5: 空间契约（/workspace /uploads /outputs）+ 超级 agent 提示词

**Files:**
- Modify: `backend/pkg/agentsandbox/manager.go`（新增 EnsureWorkspaceLayout）
- Modify: `backend/crossdomain/contract/sandbox/sandbox.go`（接口加 EnsureWorkspaceLayout，可选）
- Modify: `backend/domain/agent/singleagent/internal/agentflow/system_prompt.go`（超级 agent 纪律）

- [ ] **Step 1: manager 加 EnsureWorkspaceLayout**

`manager.go` 加方法（建固定目录，幂等）：
```go
// EnsureWorkspaceLayout 确保固定文件系统契约目录存在：/workspace /uploads /outputs。
func (m *Manager) EnsureWorkspaceLayout(ctx context.Context, key string) error {
	if err := m.EnsureSandbox(ctx, key); err != nil {
		return err
	}
	_, err := m.Exec(ctx, key, "mkdir -p /workspace /uploads /outputs", 0)
	return err
}
```

- [ ] **Step 2: 单测（用现有 fakeRunner）**

`pkg/agentsandbox/workspace_test.go`：
```go
func TestEnsureWorkspaceLayout(t *testing.T) {
	m := testManager(newFakeRunner())
	if err := m.EnsureWorkspaceLayout(context.Background(), "u1"); err != nil {
		t.Fatalf("layout: %v", err)
	}
}
```
> fakeRunner.Exec 返回 ok（见 fakes_test.go），断言无错即可；真实建目录在 220 E2E 验证。

- [ ] **Step 3: 超级 agent 系统提示词**

`system_prompt.go`：在「Task Execution Discipline」段加（仅超级 agent 注入，用模板变量或在 super 分支拼接）：
```
- For a complex, multi-step task (writing/refactoring code, multi-file investigation, anything needing many steps or parallel exploration), prefer delegating it to the "deep_task" tool, which plans autonomously and can spawn its own sub-agents. Pass it one clear, self-contained task description.
- Your files live under /workspace (working files), /uploads (user uploads), /outputs (deliverables you produce for the user). Put final deliverables in /outputs.
```
> 若 system_prompt 是全局常量，给超级 agent 单独追加一段（在 super 分支构建 systemPrompt 时拼接），避免影响普通 agent。

- [ ] **Step 4: 编译 + 测试**

Run: `cd backend && SESSION_HMAC_SECRET=test go build ./... && go test ./pkg/agentsandbox/... -run TestEnsureWorkspaceLayout -v`
Expected: build 0；测试 PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/pkg/agentsandbox/ backend/crossdomain/contract/sandbox/ \
  backend/domain/agent/singleagent/internal/agentflow/system_prompt.go
git commit -m "feat(agent): workspace layout contract + super-agent prompt discipline"
```

---

## Task 6: 全量验证 + 220 E2E

**Files:** 无（验证）

- [ ] **Step 1: 全量编译 + vet**

Run: `cd backend && SESSION_HMAC_SECRET=test go build ./... && go vet ./domain/agent/singleagent/... ./pkg/agentsandbox/...`
Expected: 均退出码 0。

- [ ] **Step 2: 相关包测试**

Run: `cd backend && SESSION_HMAC_SECRET=test go test ./domain/agent/singleagent/internal/agentflow/... ./domain/agent/singleagent/internal/dal/... ./pkg/agentsandbox/...`
Expected: 全 ok。

- [ ] **Step 3: 220 部署 E2E（参照本仓 tasks/TEST-REPORT.md 的部署方式）**

- 交叉编译 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o openynet .`，scp 到 220，overlay 镜像（含 docker CLI），跑测试容器（端口 8896、挂 docker.sock、MINIO_ENDPOINT=10.10.10.220:9000）。
- 跑迁移 SQL（`docker/migrations/20260617_add_agent_type.sql`）到 openynet 库（**先备份**，或用单独测试库）。
- 浏览器（402087139@qq.com）创建一个 agent，手动把其 `agent_type` 置 `super`（或经 UI/SQL），发一个需要 deep_task 的复杂任务，验证：deep_task 被调用、自治多步完成、产物落 /outputs、普通 agent 不受影响。
- 验收对照 spec §11。

- [ ] **Step 4: 收尾提交（测试报告更新）**

```bash
git add docs/superpowers/
git commit -m "docs(super-agent): P1+P2 verified (220 E2E)"
```

---

## 后续计划（不在本 P1+P2）
- P3 技能：`docs/superpowers/plans/<date>-super-agent-p3-skills.md`（SKILL.md 标准化 + 绑定 + skill_manage 自创 + 安全扫描 + UI）
- P4 记忆：memory 存储 + save/recall + 注入
- P5 工具生态：MCP host + 示范工具

## 风险/备注
- `developer_api.DraftBotCreateRequest` 是否已有 `agent_type` 字段需在实现 Task 起手时确认；若 IDL 生成的 request 无此字段，UI 暂经「编辑/SQL」设置 agent_type，request 字段作为 P3 UI 工作的一部分补（不阻塞 P1+P2 引擎验证）。
- deep_task 内部步骤默认不流式（黑盒返回）；如需展示中间步骤，后续开 `EmitInternalEvents`。
- 迁移在生产库执行前必须备份（spec §9 边界）。
