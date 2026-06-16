# SPEC · Studio 超级智能体（agentic coding agent）+ 独立沙箱模块

> 规格文档。先于 plan.md / 实现。任何实现以本文档的验收标准为准。
> 状态：草案 v1，待确认。

---

## 1. 目标（Objective）

在 coze-studio 里，把现有 agent 升级为「Claude Code 式」编码/任务智能体：能在隔离沙箱里
**跑命令、读写/精确编辑文件、搜索代码、多步自治、按技能完成日常任务**，并把整个沙箱环境
做成一个**自包含、可独立抽出**的模块。

- **目标用户**：Studio 平台上配置 agent 的开发者/运营；最终面向终端用户的智能体使用者。
- **价值**：让平台具备市面头部智能体（Claude Code / Codex / OpenHands）的核心能力。
- **落地策略**：走「路线 A」——不改 eino adk/compose 两套 runtime，在现有 compose/ReAct
  主路径上以**工具 + 提示词**补齐能力。低风险、可增量上线。adk 自治循环留作后续（路线 B）。

## 2. 范围（Scope）

### In scope（本次）
1. **独立沙箱模块**：把沙箱（Runner+Docker 实现+会话管理+持久化）迁入 `backend/pkg/agentsandbox/`，
   对下只依赖模块自有的 `Cache`/`Blob` 注入接口，不依赖 coze 的 domain/application/infra-impl。
2. **编码工具集**：`edit_file`(search-replace)、`grep`、`glob`（✅ 已实现并验证，见 §7）。
3. **context 压缩**：工具输出超限时做摘要而非粗暴截断。
4. **两轴审批**：危险操作（写类 run_bash / 删 / 网络）前置审批门 + plan 只读模式。
5. **测试验证**：单元 + 集成 + 构建 + （可选）真实沙箱端到端冒烟。

### Out of scope（后续，不在本次）
- repo_map（接 codegraph）、adk.Runner 接通 DeepAgent 自治循环、子 agent fan-out、provider 多模型路由、Windows 沙箱后端。

### Non-goals
- 不重写现有 ReAct 主路径；不融合 adk 与 compose 两套 runtime；不引入新外部沙箱服务（E2B 等）。

## 3. 需求与验收标准（Requirements & Acceptance）

| # | 需求 | 验收标准 |
|---|---|---|
| R1 | 沙箱独立模块 | `backend/pkg/agentsandbox/` 内 `grep -r "backend/domain\|backend/application\|backend/infra/impl"` 命中数为 0；模块仅依赖自有 Cache/Blob 接口 + Runner 接口；`go build ./...` 通过 |
| R2 | 行为不变 | run_bash/read/write/list/SyncSkill 迁移后行为不变；现有 domain/sandbox 测试全部迁移并通过 |
| R3 | edit_file | search-replace 语义：0 匹配报 not found；多匹配且非 replace_all 报错且文件不变；replace_all 全替换；返回替换次数。单测覆盖✅ |
| R4 | grep/glob | grep 优先 ripgrep 回退 grep，返回 file:line；glob 按文件名模式返回路径；空 pattern 报错 |
| R5 | 提示词纪律 | 系统提示词包含「先 read 再 edit、优先 edit_file、grep/glob 定位」纪律 |
| R6 | context 压缩 | 工具输出超阈值时摘要（保留关键信息），单测验证压缩前后预算 |
| R7 | 两轴审批 | 危险工具调用前触发审批门；plan 只读模式不写文件/不执行；单测覆盖门控判定 |
| R8 | 全量验证 | `go build ./...` + `go vet` + 相关包 `go test` 全绿；输出测试报告 |

## 4. 命令（Commands）

```bash
# 后端根目录: backend/
go build ./...                               # 全量编译
go vet ./domain/sandbox/... ./pkg/agentsandbox/...
go test ./domain/sandbox/... ./pkg/agentsandbox/... ./domain/agent/singleagent/...
go test -run Docker ./pkg/agentsandbox/docker/...   # 集成测试（需本机 Docker，可选）
```

## 5. 项目结构（Project Structure）

```
backend/pkg/agentsandbox/          # ← 独立沙箱模块（可整体抽出）
  types.go        # Runner 接口 + Create/Exec/Read/Write/List 请求响应 + State
  deps.go         # Cache / Blob 注入接口（模块自有，替代 infra/contract/cache、storage）
  manager.go      # 会话级 Manager（facade）+ EditFile/Grep/Glob
  registry.go     # 活沙箱注册表（内存 + Redis-via-Cache）
  reaper.go       # 空闲回收
  docker/         # Docker Runner 实现（接 Runner 接口）
backend/crossdomain/contract/sandbox/   # 消费侧 Manager 接口（不变，类型指向 agentsandbox）
backend/application/base/appinfra/      # 装配：infra cache/storage → agentsandbox Cache/Blob 适配
backend/domain/agent/.../agentflow/     # 工具层：edit_file/grep/glob + 审批门 + 提示词
```

## 6. 代码风格（Code Style）

- 遵循 coze-studio 现有 Go 风格与目录约定；Conventional Commits。
- surgical edit，不重构无关代码；不给未改代码加注释/类型。
- 接口只增不改（crossdomain Manager 加方法不动旧签名），避免破坏其它实现。
- 新模块对外 API 稳定、依赖倒置（注入 Cache/Blob/Runner）。

## 7. 测试策略（Testing Strategy）

- **单元**：纯 Go + 内存 fakeRunner/fakeCache/fakeBlob。覆盖 EditFile（5 例✅）、Grep/Glob 校验、
  context 压缩、审批门判定、会话生命周期回归。
- **集成**：Docker Runner（`-run Docker`，需本机 Docker；缺环境降级 mock 并标注）。
- **构建**：`go build ./...` + `go vet`。
- **端到端（可选）**：起一个绑定技能的 agent，让它 read→edit_file→run_bash 跑一条真实链路。
- 绿灯门槛：合并前全部单元 + 构建 + vet 必须绿。

## 8. 边界（Boundaries）

- **Always**：改代码后跑 build+test；每个 Phase 一个 checkpoint 提交；保持行为兼容。
- **Ask first**：删除/移动既有公共包导致跨仓影响；改动 application 装配的运行时行为；引入新外部依赖。
- **Never**：force-push；改 main/上游；删用户数据；把 adk 与 compose 强行融合（本次不碰 runtime）。

## 9. 已完成（截至本 spec）
- ✅ Phase 0：从 ynet-main 拉 `feat/agent-sandbox-superagent`，基线绿。
- ✅ Phase 2（部分）：edit_file/grep/glob 工具 + 提示词 + 单测，已提交 `7ff0687f0`。

## 10. 已确认的规格决策（2026-06-17）
1. ✅ 模块目录：`backend/pkg/agentsandbox/`
2. ✅ context 压缩：**结构化截断 + 落盘引用（DeepAgent 式）**——超阈值的工具输出写进沙箱
   `/workspace/.agent/tooloutputs/<id>.txt`，回灌给模型的是「头尾摘要 + 文件引用路径」，不调额外 LLM。
3. ✅ 两轴审批：**按 Codex 默认**——sandbox mode(read-only/workspace-write/full) × approval
   policy(never/on-failure/on-request)；写类 run_bash / 删除 / 网络访问 走审批门，读与沙箱内编辑放行。

> SPEC 已确认，进入 plan 对齐与实现。
