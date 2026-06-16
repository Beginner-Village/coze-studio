# 实施计划 · Studio 超级智能体（agentic coding agent）+ 沙箱独立化

> 目标：在 coze-studio 里落地「Claude Code 式」编码智能体的核心能力，并把沙箱环境
> 抽成一个**自包含、可独立抽出**的模块（现放项目内文件夹，将来可拆为独立项目/服务）。
> 落地策略：走「路线 A」——不碰 eino adk/compose 两套 runtime，在现有 compose/ReAct
> 主路径上以工具+提示词补齐能力。低风险、可增量上线。

## 背景结论（已调研，详见 桌面 PDF 方案）
- eino DeepAgent 的 `edit_file` 本身就是 search-replace（= Claude Code），但它在 adk runtime，
  与现有 compose 主路径不互通 → 本次**不接 adk**，而是在 compose 侧自实现等价工具。
- 现有沙箱 ~1400 行，分散在 4 处，对外仅依赖 cache(Redis)/storage(MinIO) 两个 infra 接口，
  接入点干净（application.go 装配 + node_tool_sandbox/skill 消费）→ 适合抽独立模块。

## 代码现状（已核实）
- `backend/infra/contract/sandbox/sandbox.go` — Runner 接口（Create/Exec/WriteFile/ReadFile/ListFiles/...）
- `backend/infra/impl/sandbox/docker/{runner,command}.go` — Docker 实现
- `backend/domain/sandbox/{manager,registry,reaper}.go` — 会话管理 + MinIO 持久化 + 生命周期
- `backend/crossdomain/contract/sandbox/sandbox.go` — 给 agent 用的 Manager facade 接口
- `backend/domain/agent/.../agentflow/node_tool_sandbox.go` — 工具(run_bash/read/write/list/update_plan)
- 装配：`backend/application/application.go:324` `crosssandbox.SetDefaultSVC(infra.SandboxManager)`

---

## 依赖图（自底向上）

```
[Cache/Blob 注入接口]      (Phase1 新增，模块自有，不依赖 coze infra)
        ↓
[agentsandbox 模块]  Runner(Docker) + SessionManager + Registry + Reaper + Facade
        ↓ 实现
[crossdomain Manager]  Exec/Read/Write/List/SyncSkill  + 新增 EditFile/Grep/Glob
        ↓ 消费
[agent 工具层]  edit_file / grep / glob 工具  + system_prompt 纪律
        ↓
[端到端]  ReAct agent 在沙箱里 编辑→跑命令→搜索
```

关键点：上层（agent 工具）只依赖 crossdomain Manager 接口，**对模块内部无感**；
模块对下只依赖自定义 Cache/Blob 接口，**对 coze infra 无感** → 双向解耦 = 可抽出。

---

## 分阶段（垂直切片，每片一条完整可验证路径）

### Phase 0 · 基线与分支 〔checkpoint 0〕
- 决定分支基底与如何处理当前 52 个未提交的 display-copy 改动（见文末「待你确认」）。
- 建分支，跑一次 `go build ./...` + 相关包 `go test`，确认基线绿。
- **验收**：基线编译/测试通过，工作区干净（feature 与 display-copy 隔离）。

### Phase 1 · 沙箱独立化（extraction-ready 模块）
垂直切片，行为不变前提下迁移：
- **1A** 新建 `backend/pkg/agentsandbox/`（模块根）。定义模块自有的注入接口
  `Cache`、`Blob`（替代直接 import coze 的 infra/contract/cache、storage）。
- **1B** 迁入 Runner 接口 + Docker 实现 + SessionManager/Registry/Reaper + Facade，
  改为依赖 `Cache`/`Blob` 接口；模块内不出现 `backend/domain|application|infra/impl` 的 import。
- **1C** coze 侧薄适配：用现有 infra cache/storage 实现 `Cache`/`Blob`；
  `application.go` 装配指向新模块；`crossdomain/contract/sandbox` 由模块 Facade 满足。
- **验收**：
  - `grep -r "backend/domain\|backend/application\|infra/impl" backend/pkg/agentsandbox` 仅命中注入适配（理想为 0）。
  - run_bash/read/write/list 行为不变；`docker/runner_integration_test.go` 通过（需 Docker，可选）。
  - `go build ./...` 通过。
- 〔checkpoint 1〕迁移后人工 review 边界是否真解耦。

### Phase 2 · edit_file + grep + glob（Claude Code 式核心）
端到端切片：
- **2A** 模块 Facade + crossdomain Manager 接口新增：
  - `EditFile(ctx,key,path,oldStr,newStr string,replaceAll bool)`：Go 内 Read→search-replace→Write；
    `replaceAll=false` 且多处匹配 → 报错要求 replace_all；0 处 → "string not found"。
  - `Grep(ctx,key,pattern,path)`：沙箱内 Exec `rg`(无则 `grep -rn`)。
  - `Glob(ctx,key,pattern)`：沙箱内 Exec `find`。
- **2B** `node_tool_sandbox.go` 加 `editFileTool/grepTool/globTool`（Info + InvokableRun），
  注册进 `newSandboxTools`。
- **2C** `system_prompt.go` 增加 edit_file/grep/glob 使用纪律（抄 DeepAgent：先 read 再 edit、
  优先 edit 不要整文件重写、唯一匹配等）。
- **2D** 单测（纯 Go，内存假 Runner）：EditFile 的 唯一匹配 / 多匹配报错 / replaceAll / 未找到 四条路径。
- **验收**：单测通过；`go build ./...` 通过；mock manager 冒烟验证三个工具的 ToolInfo schema 合法。
- 〔checkpoint 2〕review edit 语义与提示词。

### Phase 3 · context 压缩 + 两轴审批（本次纳入，见 SPEC §10）
- **3A 压缩**：DeepAgent 式——工具输出超阈值时写进沙箱 `/workspace/.agent/tooloutputs/<id>.txt`，
  回灌给模型「头尾摘要 + 文件引用路径」，不调额外 LLM。替代现有粗暴 16KB 截断（truncateForModel）。
- **3B 审批**：Codex 两轴——sandbox mode × approval policy；写类 run_bash/删除/网络 走审批门，
  读与沙箱内编辑放行；plan 只读模式不写不执行。工具调用前置 hook。
- **验收**：审批门判定单测；压缩落盘+引用单测。

### Phase 4 · 测试与验证（贯穿，收口）
- 单元：EditFile/Grep/Glob、沙箱模块迁移后回归。
- 集成：docker runner（如环境允许）。
- 构建：`go build ./...` + `go vet`。
- （可选）真实沙箱端到端冒烟：起一个 agent 让它 read→edit→run。
- **验收**：全部绿；输出测试报告。

### Phase 5 · 后续（不在本次范围，留接口）
- repo_map（接 codegraph）、adk.Runner 接通 DeepAgent 自治循环、子 agent fan-out、provider 多模型路由。

---

## 本次执行范围（建议）
Phase 0–2 + Phase 4 = **沙箱独立化 + edit_file/grep/glob + 测试验证**。
Phase 3 视 checkpoint 决定；Phase 5 明确为后续。

## 风险与规避
- 沙箱迁移破坏现有装配 → 行为回归测试 + 保持 crossdomain 接口签名不变（只增不改）。
- Docker 集成测试依赖本机 Docker → 缺环境时降级为 mock + 标注。
- eino 工具 schema 写错 → 仿现有 writeFileTool 模式，单测校验 ToolInfo。

## 待你确认（checkpoint 0）
1. **分支基底**：从 `main`（干净基线）还是从当前 `chore/ynet-display-copy`（含 studio 现状）拉？
2. **52 个未提交的 display-copy 改动**怎么处理：stash 暂存（保留可恢复）/ 先提交到 display-copy 分支 / 原样带过来？
3. **本次范围**：按建议做 Phase 0–2+4，还是把 Phase 3（审批+压缩）也纳入？
