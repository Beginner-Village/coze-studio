# 设计文档 · 超级智能体（Harness Agent）

> 状态：草案 v1，待用户评审。
> 分支：`feat/agent-sandbox-superagent`。
> 决策已定：全新独立 agent 类型 / 每超级 agent 独立空间 / 混合引擎（方案2）/ 一次覆盖全部愿景（实现分阶段）。

## 1. 目标（Objective）

在 Studio 里新增一种**一等公民的「超级智能体（Harness Agent）」**，与现有普通 agent 并存、互不影响。
它跑在我们这套自治引擎上（DeepAgent 委托 + 沙箱工具），每个超级 agent 有自己独立的空间（沙箱+存储），
靠**绑定不同技能**组合出不同能力，带角色（persona）与长期记忆，未来可无缝接入浏览器/文档等工具。
**用户可以创建很多个这样的超级智能体。**

- 目标用户：Studio 上配置 agent 的开发者/运营；最终面向终端使用者。
- 价值：把平台的 agent 形态从「带工具的对话机器人」升级到「能自主完成多步复杂任务的超级智能体」，且可组合、可扩展。
- 非目标：不改动现有普通 agent 的任何行为；不重写现有 ReAct 主链路；本期不强求纯 adk runtime 替换。

## 2. 范围（Scope）

### In scope（一个 spec 覆盖全部愿景，实现按依赖分阶段）
- **A. 新 agent 类型 + 路由**：`agent_type=super` 判别，超级 agent 走自治引擎，普通 agent 不变。
- **B. 每超级 agent 独立空间**：沙箱 + 固定文件系统契约（workspace/uploads/outputs）+ 持久化。
- **引擎（混合·方案2）**：超级 agent 仍用现有 compose/ReAct 运行时（流式/中断复用），挂 `deep_task` 工具委托复杂自治任务给 eino DeepAgent；按需把子 agent/压缩当工具/中间件加入。
- **C. 技能系统**：对齐 SKILL.md 开放标准；技能绑定/组合；agent 自创技能（skill_manage）；技能管理 UI 优化。
- **E. 角色 + 记忆**：persona 复用；新增每超级 agent 的长期记忆（存储 + save/recall 工具 + 注入）。
- **D. 可组合工具**：以 MCP host + 技能机制为扩展边界，浏览器/doc 等工具后续插入，不改核心。

### Out of scope（明确不做 / 后续）
- 纯 adk.Runner 替换 ReAct 运行时（方案1，二期视效果再上）。
- 多租户级硬隔离沙箱（gVisor/microVM）——本期沿用 Docker 沙箱。
- 具体的浏览器/doc 工具实现本身（本期只做"能插进来"的机制 D，不做工具）。

## 3. 已确认决策
1. ✅ 全新独立 agent 类型（不是现有 agent 的模式开关）。
2. ✅ 每个超级 agent 一个独立空间（沙箱+存储），不按用户共享、不按会话临时。
3. ✅ 引擎用**混合方案2**：现有运行时 + deep_task 委托，低风险、先跑起来。
4. ✅ 一个 spec 覆盖全部愿景；实现分阶段（见 §8）。

## 4. 架构总览

```
[Studio UI]
  ├ 创建/编辑「超级智能体」入口（新 agent_type=super）
  ├ 技能管理页（浏览/建/改/绑定/开关，对齐 SKILL.md）
  └ 对话调试（流式，复用现有）
        │  agent_type=super
        ▼
[Agent 运行时 · agentflow]
  现有 compose/ReAct 引擎（流式/中断/checkpoint 复用）
   ├ 工具层：run_bash/edit_file/grep/glob/read/write/list  (已有)
   ├ + deep_task 工具 = adk.NewAgentTool(DeepAgent)        (路A，已写雏形)
   │     └ DeepAgent：write_todos 自治 + task 子agent(隔离) + 复用同批工具
   ├ + skill_manage 工具（自创/改/版本化技能）            (新)
   ├ + memory save/recall 工具 + system prompt 注入        (新)
   └ persona / 知识库 / 工作流 节点                        (已有)
        │
        ▼
[每超级 agent 独立空间 · agentsandbox 模块]
   Docker 沙箱(per agent+user key) + MinIO 持久化
   固定 FS 契约：/workspace  /uploads  /outputs  /skills/<name>/
        │
        ▼
[扩展边界]
   MCP host（eino-ext）+ 技能机制 → 未来浏览器/doc 工具插入
```

## 5. 组件设计（按子系统）

### 5.1 新 agent 类型 + 路由（A）
- 数据模型：`single_agent` 增加 `agent_type`（默认 `normal`；`super` = 超级智能体）。迁移加一列，默认值保证存量 agent 不受影响。
- 创建流程：UI 新增「创建超级智能体」；后端按 type 落库。
- 路由：`agent_flow_builder` 在组装时读 `agent_type`，`super` 时启用超级 agent 专属工具集（deep_task / skill_manage / memory）与提示词；`normal` 时与现状完全一致。
- **隔离原则**：所有超级 agent 专属逻辑都在 `agent_type=super` 分支内，普通 agent 代码路径零改动。

### 5.2 引擎 · 混合（方案2）
- 复用现有 `compose.Graph + react.Agent + callback_reply_chunk` 流式/中断链路（已验证可用）。
- `deep_task` 工具（`node_tool_deeptask.go`，已写）：`adk.NewAgentTool(deep.New(...))`，把复杂多步任务委托给 DeepAgent 自治（write_todos + 子 agent + 复用同批沙箱工具）。eino 原生 agent-as-tool，无需 adk↔compose 桥接。
- 开关：超级 agent 默认启用 deep_task；普通 agent 受 `DEEP_TASK_ENABLED` 控制（默认关）。
- 提示词：超级 agent 系统提示词增加「遇到多步/探索类任务委托 deep_task；先 read 再 edit；技能匹配时 read_skill」纪律。

### 5.3 每超级 agent 空间（B）
- 复用 `agentsandbox` 模块（已独立化）。沙箱 key = `(connector, agent, user)` → 天然每超级 agent 一套。
- 固定 FS 契约：`/workspace`（工作区，持久）、`/uploads`（用户上传）、`/outputs`（产物）、`/skills/<name>/`（技能注入）。落地为沙箱内目录约定 + 解析。
- 持久化：MinIO checkpoint（已有）；reaper 生命周期（已有）。
- 存储：超级 agent 的"自己的东西"= /workspace + /outputs，跨会话保留。

### 5.4 技能系统（C）
- **格式标准化**：技能落为 `SKILL.md` + YAML frontmatter（name/description/allowed-tools），对齐 agentskills.io（与 Claude Code/Hermes 互通）。
- **渐进披露**：被动——绑定技能的 name+description 进 system prompt；主动——`read_skill` 注入全文 + 脚本到 `/skills/<name>/`（复用现有注入）。
- **绑定/组合**：超级 agent 绑定技能集合（复用现有 SkillInfoList），不同组合 = 不同能力。
- **自创技能**：新增 `skill_manage` 工具（create/edit/list/delete），agent 把 SKILL.md 写入自己的技能空间，原子写 + 版本。安全：写入前过内容扫描（参照 Hermes skills_guard 思路）。
- **UI 优化**：技能管理页——列表/搜索/新建/编辑/绑定到超级 agent/启停；展示 frontmatter 与脚本。

### 5.5 角色 + 记忆（E）
- persona：复用现有。
- 长期记忆：每超级 agent 一份记忆（结构化 JSON：user/facts/preferences）。
  - 存储：先用沙箱内文件（`/workspace/.agent/memory.json`，跟随空间持久化），后续可迁 DB。
  - 工具：`memory_save` / `memory_recall`，agent 自己读写。
  - 注入：会话开始把记忆摘要注入 system prompt。
  - 设计参照 Letta（memory blocks）/ DeerFlow（总结边界异步更新）。

### 5.6 可组合工具扩展（D）
- 机制：MCP host（eino-ext `components/tool/mcp`）+ 技能机制。
- 未来浏览器/doc 工具 = 一个 MCP server 或一个带脚本的技能，绑定即用，核心零改动。
- 本期只做"能插进来"的边界，不实现具体工具。

## 6. 数据流（一次超级 agent 对话）
```
用户消息 → 路由(agent_type=super) → 现有 compose 图
  → ReAct 决策：简单直接答；复杂 → 调 deep_task
     → DeepAgent 自治：write_todos → 子agent/沙箱工具(在本 agent 空间) → 汇总
  → 技能匹配 → read_skill 注入 → 按技能脚本执行
  → 记忆：开场注入摘要，过程中 memory_save
  → 大输出 offload 落盘(已有) → 流式回前端(已有)
```

## 7. 测试策略
- 单元：路由判别、deep_task 构建、skill_manage（create/edit/版本/安全扫描）、memory save/recall、FS 契约解析。
- 集成：超级 agent 端到端（绑技能→自治多步→产物落 /outputs→记忆持久）。
- 构建：`go build ./...` + `go vet`。
- E2E（220 部署）：创建超级 agent → 复杂任务（写代码/查资料/多步）→ 验证 deep_task 自治、技能生效、记忆跨会话、空间隔离。
- 绿灯门槛：单元+构建全绿；E2E 关键路径通过。

## 8. 实现阶段（依赖顺序；spec 覆盖全部，按此分批落地）
1. **P1 地基**：`agent_type` 字段+迁移+路由；超级 agent 启用 deep_task；FS 契约（workspace/uploads/outputs）。
2. **P2 引擎跑顺**：deep_task 在超级 agent 上端到端可用；提示词纪律；220 验证自治。
3. **P3 技能**：SKILL.md 标准化 + 绑定 + skill_manage 自创 + 安全扫描 + 技能管理 UI。
4. **P4 记忆**：memory 存储 + save/recall 工具 + 注入。
5. **P5 工具生态**：MCP host 接好；示范接一个工具（如浏览器或 doc）验证可组合。

## 9. 边界（Boundaries）
- **Always**：改代码后 build+test；每阶段一个 checkpoint 提交；普通 agent 行为零回归。
- **Ask first**：DB 迁移上生产；改动现有 agent 公共路径；引入新外部依赖。
- **Never**：force-push；改 main/上游；删用户数据；本期强行做纯 adk runtime 替换。

## 10. 风险与规避
- DeepAgent 在工具内运行的流式可见性：deep_task 内部步骤默认不流式（黑盒返回结果）→ P2 视需要再开 EmitInternalEvents。
- 技能自创安全：skill_manage 写入前内容扫描 + 限制可写路径在 /skills。
- 记忆膨胀：摘要注入 + 大小上限。
- 迁移风险：`agent_type` 加列默认值，存量 agent 不受影响。

## 11. 验收标准（Acceptance）
- 普通 agent 行为 100% 不变（回归测试）。
- 能在 UI 创建「超级智能体」，绑定技能，跑通一个需要 deep_task 自治的多步复杂任务。
- 每超级 agent 空间隔离（A 的文件 B 看不到）、跨会话持久。
- agent 能自己创建一个技能并在下一轮用上。
- 记忆跨会话生效。
- 浏览器/doc 类工具能以 MCP/技能方式插入（示范一个）。
