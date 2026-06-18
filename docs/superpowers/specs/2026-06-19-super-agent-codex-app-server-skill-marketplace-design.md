# 设计文档 · Codex-like Harness 超级智能体、App Server 与标准技能商城

> 状态：草案 v1，待用户评审。
> 分支：`feat/agent-sandbox-superagent`。
> 日期：2026-06-19。
> 约束：只作用于「沙箱 + 超级智能体」方案；普通 agent、workflow、插件、知识库等现有系统默认行为不变。

## 1. 背景

现有超级智能体已经有一部分地基：

- `single_agent.agent_type` 已用于区分 `super` 与普通 agent。
- `agentflow` 已在 `isSuperAgent` 分支里挂载沙箱工具、`deep_task`、memory、skill_manage、扩展工具。
- 沙箱 workspace 已提供 `/workspace`、`/uploads`、`/outputs` 的文件管理接口。
- `Skill` entity 已有 `Files map[string]string`，可以承载 `SKILL.md` 与脚本/资源文件。

但当前形态还不够：

- 技能仍然偏 prompt-only，更新链路、版本快照、资产管理不完整。
- `skill_manage` 只有 create/list/read，不能 patch/edit/write_file/remove_file/delete，也没有审批、发布、商城。
- 沙箱 workspace 只适合 Studio 内部使用，还不是一个外部可接入的 app-server 能力面。
- Codex-like 的 trace、plan、subagent、MCP、插件/技能分发、审批边界还没有形成统一运行层。

本 spec 要把超级智能体升级成一套独立的 Codex-like Harness 能力层：既能在 Studio 里像 Codex 一样工作，也能作为 app server 被外部系统调用，能够读写文件、写代码、跑测试、生成产物并留下可回放 trace。

## 2. 参考模型

本设计参考以下公开能力模型：

- Codex app 的能力面：多项目/线程、worktrees、terminal/actions、browser、automation、skills、plugins、artifacts、IDE sync。
  参考：`https://developers.openai.com/codex/app`
- Codex skills：技能是 `SKILL.md` 目录包，支持 `scripts/`、`references/`、`assets/`，并使用 progressive disclosure。
  参考：`https://developers.openai.com/codex/skills`
- Codex plugins：插件可打包 skills、apps、MCP servers，并通过 marketplace/distribution 分发。
  参考：`https://developers.openai.com/codex/plugins`
- Codex sandbox：沙箱是自治执行边界，审批策略和沙箱边界共同控制风险。
  参考：`https://developers.openai.com/codex/concepts/sandboxing`
- Codex MCP server：Codex 可以以 MCP server 方式暴露给外部 agent，提供长会话调用能力。
  参考：`https://developers.openai.com/codex/guides/agents-sdk`
- Hermes skills：Hermes 把技能当作 procedural memory，支持标准文件夹、hub/tap/source、audit、quarantine、agent-managed skills 与审批流。
  参考：`https://hermes-agent.nousresearch.com/docs/user-guide/features/skills`

这些参考只用于定义产品与工程目标；实现必须贴合本仓库的 agentflow、agentsandbox、skill domain、marketplace 数据结构。

## 3. 目标

1. 平台提供一种 Codex-like Harness 超级智能体，能力集中在 `agent_type=super`。
2. 超级智能体具备接近 Codex 的主要能力：沙箱文件系统、终端命令、读写代码、运行测试、计划、工具 trace、子任务、标准技能、MCP/插件、产物预览、审批、自动化入口。
3. 提供 app-server 风格外部服务：外部系统可创建/继续/取消 run，流式读取过程，上传文件，下载产物，管理技能，读取 trace。
4. 重构技能系统为标准技能包，支持添加技能、编辑技能、发布技能、安装技能、版本固定、资产管理。
5. 发布维度明确：个人草稿、空间级、全局级、官方精选；全局级有技能商城，所有空间可浏览/安装。
6. 不影响系统其他功能：普通 agent、workflow、插件、知识库、旧技能绑定都走兼容层或原路径。

## 4. 非目标

- 不把普通 agent 迁移到超级智能体 runtime。
- 不重写现有 workflow runtime。
- 不把现有 plugin_marketplace 直接改造成技能商城；技能商城可复用分类/统计思想，但独立建模。
- 不在第一阶段实现所有 Codex app 功能的 UI 细节，例如完整 IDE sync、Chrome extension、desktop computer use。
- 不把测试环境密码、生产密钥、外部 token 写入仓库。

## 5. 核心决策

1. **独立入口**：所有新能力走 `/api/super-agent/*`、`super-agent` service、`agent_type=super` runtime gate。
2. **兼容旧系统**：旧 `/api/draftbot/*` 只保留现有 Studio 内部能力；必要时内部复用 service，但不承载对外 app-server 契约。
3. **标准技能包优先**：技能的 source of truth 是文件夹包，不是 prompt 字段。`prompt` 只作为兼容读取的派生字段。
4. **发布即快照**：发布空间级或全局级技能时，必须快照整个文件树和资产 hash。
5. **安装即绑定版本**：超级智能体绑定的是 `skill_installation` 或 `skill_version`，不是浮动最新草稿。
6. **审批优先**：agent 自创技能、危险命令、外部发布、全局上架都先进入 pending/review 状态。
7. **测试优先落地**：每个阶段都要有本地自动化验证；集成阶段必须在 `10.10.10.226` 测试环境做 Playwright 验收。

## 6. 总体架构

```text
[External Client / Agent / App]
       │
       ├── HTTP/SSE: /api/super-agent/*
       ├── MCP: super-agent-mcp-server
       │
       ▼
[Super Agent App Server]
  ├── Run API: create / reply / stream / cancel
  ├── Workspace API: upload / list / read / download / delete
  ├── Trace API: plan / tool calls / approvals / artifacts
  ├── Skill API: create / edit / install / publish / marketplace
  └── Auth + quota + permission + audit
       │
       ▼
[Super Agent Runtime]
  ├── agent_type=super gate
  ├── agentflow ReAct + deep_task
  ├── sandbox tools: run_bash/read/write/edit/list/grep/glob
  ├── coding loop: inspect/search/edit/test/diff/artifact
  ├── plan/update_plan
  ├── memory_save/memory_recall
  ├── read_skill/skill_manage
  ├── MCP/plugin tools
  └── trace event sink
       │
       ▼
[Per Super Agent Sandbox]
  ├── /workspace
  ├── /uploads
  ├── /outputs
  ├── /skills/<slug>/
  ├── /tmp
  └── .agent/{memory,trace,approvals}
       │
       ▼
[Skill Package + Marketplace]
  ├── skill_package
  ├── skill_version
  ├── skill_asset
  ├── skill_installation
  ├── skill_publication
  └── skill_marketplace_listing
```

## 7. Super Agent Runtime

### 7.1 运行边界

Runtime 只在 `agent_type=super` 时启用。普通 agent 不加载：

- `deep_task`
- 沙箱 bash/文件工具
- `skill_manage`
- memory 工具
- super-agent extension tools
- app-server trace sink

普通 agent 可以继续使用旧的 `read_skill` 和旧绑定结构，但不会进入新沙箱能力面。

### 7.2 文件系统契约

沙箱目录固定为：

```text
/workspace   用户与 agent 的长期工作区
/uploads     用户或外部系统上传的输入
/outputs     生成产物，供 UI 与 app-server 下载
/skills      已安装技能包，只读或受控写
/tmp         临时执行目录，可清理
/.agent      runtime 元数据，默认不直接暴露
```

Studio 现有 workspace 管理可以继续使用 `/workspace`、`/uploads`、`/outputs`。新 app-server API 增加对 `/skills` 的只读预览和对 `/outputs` 的下载 token。

### 7.3 Harness 能力模型

Harness 不是一个更大的 system prompt，而是一套可执行工作循环。超级智能体必须能稳定完成：

1. **Inspect**：读取目录、定位代码、理解项目结构、读取 `AGENTS.md`/README/配置。
2. **Plan**：把复杂任务拆成可更新步骤，并把 plan 事件写入 trace。
3. **Edit**：通过受控文件工具修改 `/workspace` 中的代码、文档、配置、网页、脚本。
4. **Run**：在沙箱中执行命令，例如 install/build/typecheck/test/lint/dev server。
5. **Verify**：根据任务类型运行自动化测试、Playwright、截图、API 请求或产物检查。
6. **Artifact**：把结果写入 `/outputs`，包括 HTML、文档、图片、压缩包、报告、代码包。
7. **Review**：输出修改摘要、风险、测试结果、产物路径、trace 链接。
8. **Persist**：把有价值的流程沉淀为标准 skill 或 memory，但需要审批。

这条循环是「像 Codex 一样能写代码和做产物」的验收基础。任何只返回聊天文本、没有文件/命令/验证/产物证据的实现，都不算完成 Harness 能力。

### 7.4 写代码能力

写代码任务必须遵守：

- 所有代码改动发生在超级智能体沙箱的 `/workspace`。
- 修改前先 inspect/search，避免盲写。
- 大改动必须先 plan，再 edit。
- edit 后必须运行项目声明的验证命令；没有项目命令时，至少运行语法检查或目标文件级测试。
- 生成 diff summary，trace 记录被修改文件、命令输出、失败重试。
- 可以把完整代码产物导出到 `/outputs/<artifact>`，供 app-server 下载。

典型任务：

- 创建一个前端 demo。
- 修改已有项目代码并跑测试。
- 生成脚本/CLI。
- 分析数据并产出报告。
- 生成可下载 zip。
- 部署前做 smoke test。

### 7.5 产物能力

产物分级：

- `previewable`：HTML、Markdown、PDF、图片、CSV、JSON，可在 Studio 预览。
- `downloadable`：zip、docx、xlsx、pptx、代码包，通过 app-server 下载。
- `deployable`：静态站点或服务包，二期接入部署工具。
- `trace-linked`：每个产物记录由哪个 run、哪个工具、哪个文件生成。

产物元数据：

- `artifact_id`
- `run_id`
- `path`
- `mime`
- `size`
- `sha256`
- `preview_url`
- `download_url`
- `created_at`

### 7.6 工具与审批

工具分级：

- `safe_read`：list/read/grep/glob/trace read，可自动执行。
- `safe_write`：写 `/workspace`、`/outputs`，默认允许但记录 trace。
- `sandbox_exec`：`run_bash`，根据命令策略决定是否审批。
- `skill_mutation`：创建/编辑技能，必须进入 pending 或草稿。
- `publish_global`：发布到全局商城，必须人工审核。
- `external_network`：web/MCP/外部 API，按 app-server token 与 tool policy 控制。

审批结果写入 trace，并可被外部系统通过 API 查询/处理。

### 7.7 Trace

每个 run 都要产出可回放 trace：

- user input
- model step
- plan update
- tool call start/end/error
- file changes summary
- approval requested/resolved
- skill selected/read/installed/updated
- artifact created
- final answer

Trace 是 app-server 和 Playwright 验收的重要证据。

## 8. App Server 外部服务层

### 8.1 HTTP API

新增独立路由组：

```text
/api/super-agent/runs/create
/api/super-agent/runs/reply
/api/super-agent/runs/stream
/api/super-agent/runs/cancel
/api/super-agent/workspace/list
/api/super-agent/workspace/read
/api/super-agent/workspace/upload
/api/super-agent/workspace/download
/api/super-agent/traces/get
/api/super-agent/approvals/list
/api/super-agent/approvals/resolve
/api/super-agent/skills/create
/api/super-agent/skills/update
/api/super-agent/skills/install
/api/super-agent/skills/publish
/api/super-agent/marketplace/list
/api/super-agent/marketplace/get
```

这些 API 的请求必须包含：

- `space_id`
- `agent_id`
- `user_id` 或鉴权态中的用户
- `run_id`，继续/查询类接口需要
- `client_id`，外部 app 可选

### 8.2 Streaming

首选 SSE：

```text
event: run.started
event: plan.updated
event: tool.started
event: tool.delta
event: tool.completed
event: approval.required
event: artifact.created
event: message.delta
event: run.completed
event: run.failed
```

WebSocket 可作为二期，先不阻塞 HTTP/SSE。

### 8.3 MCP Server

提供一个可选 MCP server，使外部 agent 可以把超级智能体当工具：

- `super_agent_start`
- `super_agent_reply`
- `super_agent_list_files`
- `super_agent_read_file`
- `super_agent_upload_file`
- `super_agent_download_artifact`
- `super_agent_list_skills`
- `super_agent_install_skill`

MCP server 不绕过 HTTP service 权限；它只是另一种协议入口。

### 8.4 鉴权与隔离

外部接入使用独立 app token，不复用 Studio 前端 session。

权限模型：

- token 绑定 `space_id`
- token 可选绑定 `agent_id`
- token 有 scope：`run`、`workspace:read`、`workspace:write`、`skill:install`、`skill:publish`、`approval:resolve`
- token 调用全量入 audit log

## 9. 标准技能包

### 9.1 文件结构

```text
skill-slug/
  SKILL.md
  scripts/
  references/
  templates/
  assets/
  agents/
    openai.yaml
  lock.json
  README.md
```

要求：

- `SKILL.md` 必须存在。
- frontmatter 必须有 `name`、`description`。
- `description` 是触发匹配依据，必须简短明确。
- `scripts/` 中脚本默认不自动执行，只有 SKILL.md 指示且工具策略允许时执行。
- `assets/`、`templates/`、`references/` 都纳入版本快照。

### 9.2 Frontmatter

建议字段：

```yaml
---
name: report-writer
description: Use when creating structured business reports from source files.
version: 1.0.0
author: ynet
license: internal
tags: [report, document]
platforms: [super-agent]
permissions:
  filesystem:
    read: ["./references", "./templates", "/workspace", "/uploads"]
    write: ["/outputs"]
  network: false
tool_allowlist: ["read_file", "write_file", "run_bash"]
required_env: []
---
```

`prompt` 字段从 `SKILL.md` body 派生，旧 API 读取时可继续返回。

### 9.3 渐进式披露

运行时分三层：

1. Discovery：系统提示只注入 `name`、`description`、版本、来源、trust level。
2. Activation：模型决定使用技能后调用 `read_skill`，读取完整 `SKILL.md`。
3. Execution：需要脚本/资产时，从 `/skills/<slug>/` 读取文件并执行。

这能避免大量技能挤爆上下文，也保持 Codex/Hermes 风格一致。

## 10. 技能数据模型

### 10.1 兼容原则

现有 `skill` 表保留，逐步升级为 package 当前草稿表。新增表承载版本、资产、安装、发布、商城。

旧字段兼容：

- `skill.prompt`：从 `SKILL.md` body 派生。
- `skill.files`：短期可继续存完整文件树 JSON；中长期迁到对象存储 + asset manifest。
- `skill.version`：当前草稿版本号。

### 10.2 新表建议

`skill_package`

- `skill_id`
- `space_id`
- `scope`: `personal` / `space` / `global`
- `slug`
- `name`
- `description`
- `creator_id`
- `source_type`: `manual` / `agent_generated` / `import_zip` / `marketplace_install` / `github`
- `visibility`: `private` / `space` / `global`
- `status`: `draft` / `active` / `archived` / `disabled`
- `current_version`
- `created_at`
- `updated_at`

`skill_version`

- `skill_id`
- `version`
- `semver`
- `name`
- `description`
- `frontmatter_json`
- `body`
- `files_manifest_json`
- `bundle_uri`
- `content_hash`
- `scan_status`
- `created_by`
- `created_at`

`skill_asset`

- `skill_id`
- `version`
- `path`
- `kind`: `skill_md` / `script` / `reference` / `template` / `asset` / `metadata`
- `mime`
- `size`
- `sha256`
- `storage_uri`
- `scan_status`
- `created_at`

`skill_installation`

- `installation_id`
- `skill_id`
- `version`
- `installed_scope`: `user` / `space` / `agent`
- `space_id`
- `agent_id`
- `user_id`
- `enabled`
- `installed_by`
- `installed_at`

`skill_publication`

- `publication_id`
- `skill_id`
- `version`
- `publish_scope`: `space` / `global`
- `review_status`: `pending` / `approved` / `rejected` / `delisted`
- `reviewer_id`
- `review_note`
- `published_at`

`skill_marketplace_listing`

- `listing_id`
- `skill_id`
- `version`
- `title`
- `summary`
- `category_id`
- `tags_json`
- `trust_level`: `official` / `trusted` / `community`
- `is_featured`
- `heat_score`
- `install_count`
- `favorite_count`
- `rating`
- `status`: `listed` / `hidden` / `delisted`
- `created_at`
- `updated_at`

## 11. 技能生命周期

### 11.1 添加技能

入口：

- Studio UI 创建。
- 上传 zip 创建。
- agent 通过 `skill_manage` 创建。
- 从 marketplace 安装。
- 从外部 URL/GitHub 导入，二期。

创建后默认是个人草稿或空间草稿，不自动上架。

### 11.2 编辑技能

必须支持文件树编辑：

- 编辑 `SKILL.md`
- 新增/删除/重命名文件
- 上传 assets/templates
- 编辑 scripts/references
- 查看 diff
- 保存为新草稿版本

保存时执行：

1. path 校验。
2. frontmatter 校验。
3. manifest/hash 生成。
4. 脚本扫描。
5. 版本快照。

### 11.3 发布技能

发布维度：

- **个人草稿**：只创建者可见。
- **空间级发布**：同一 space 所有人可安装/绑定。
- **全局发布**：所有 space 可在商城浏览/安装，必须审核。
- **官方精选**：平台管理员认证，默认最高 trust level。

发布规则：

- 空间级发布可由 space 管理员审核。
- 全局发布必须平台管理员审核。
- 全局发布必须完成安全扫描。
- 已发布版本不可变；修改必须产生新版本。

### 11.4 安装技能

安装不是复制并修改原技能，而是创建 `skill_installation`：

- 安装到用户：个人可用。
- 安装到空间：空间内可绑定。
- 安装到超级智能体：该 agent runtime 直接可用。

绑定时默认 pin 当前版本；用户可手动更新。

### 11.5 Agent 自学习技能

`skill_manage` 升级为：

- `create`
- `patch`
- `edit`
- `write_file`
- `remove_file`
- `delete`
- `list`
- `read`
- `diff`
- `submit_for_review`

agent 写入的技能默认进入 `agent_generated` 草稿，不直接全局发布。用户审批后才能绑定或上架。

## 12. 全局技能商城

商城功能：

- 浏览/搜索/分类/标签。
- 查看详情：说明、文件树、版本、作者、trust level、扫描结果、使用示例。
- 安装到用户/空间/超级智能体。
- 收藏、安装次数、使用次数。
- 更新检查、升级、回滚、卸载。
- 审核队列：pending/rejected/approved/delisted。

商城页面建议：

- `空间技能`：本 space 草稿与已发布。
- `全局商城`：所有可安装技能。
- `我的安装`：用户/空间/agent 的安装状态。
- `审核管理`：管理员可见。

## 13. 资产管理

资产不是附件，而是技能和超级智能体的能力组成。

资产分类：

- skill assets：`assets/`、`templates/`、`references/`、`scripts/`
- workspace assets：`/workspace`
- uploaded inputs：`/uploads`
- generated outputs：`/outputs`
- marketplace media：icon、screenshots、example outputs

资产要求：

- 所有发布版本记录 path、hash、size、mime、storage_uri。
- zip 导入导出必须保持相对路径。
- 大文件走对象存储，不放 DB。
- UI 预览必须能识别 markdown/json/csv/code/image/pdf/office。
- 删除 published asset 不允许原地删除，只能新版本移除。

## 14. 前端设计

### 14.1 Super Mode

当前 2:1 工作区 + 聊天布局保留，但升级为 Codex-like workspace：

- 左侧：文件树、编辑器、预览、产物。
- 右侧：聊天、plan、trace、审批。
- 顶部或抽屉：模型、人设、技能、MCP、app-server token。

### 14.2 Trace Panel

展示：

- plan steps
- tool calls
- file changes
- approval requests
- artifacts
- skill activity

### 14.3 技能管理

空间内新增或升级页面：

- 文件夹技能编辑器。
- frontmatter 表单 + `SKILL.md` markdown。
- assets/templates/references/scripts 文件树。
- 发布目标选择：个人、空间、全局。
- 发布前检查结果。
- 全局商城入口。

## 15. 安全与隔离

### 15.1 不影响现有系统

硬边界：

- 新 API 路由使用 `/api/super-agent/*`。
- Runtime 只从 `agent_type=super` 分支进入。
- DB 新表与旧表兼容，不删除旧字段。
- 旧 Skill API 可继续工作，映射到 package 的兼容层。
- 普通 agent 不挂 `run_bash` 和超级体专属扩展。

### 15.2 扫描

扫描项：

- shell 危险命令。
- 访问敏感路径。
- 网络访问。
- 外部下载。
- 硬编码 secret。
- 超大文件。
- 可执行文件类型。

扫描结果：

- `safe`
- `warning`
- `blocked`

`blocked` 不能上架，管理员也不能绕过危险项，只能修改内容后重扫。

### 15.3 审计

记录：

- 外部 app-server token 调用。
- run 创建/取消。
- 文件读写/下载。
- 命令执行。
- 技能安装/发布/审核。
- 审批动作。

## 16. 测试环境记录

用户提供的测试环境：

- Host：`10.10.10.226`
- SSH：`dev@10.10.10.226`，仅用于部署/服务排查；凭据不写入仓库。
- 当前部署入口：`http://10.10.10.226:8896/`
- 测试账号：由会话提供，禁止写入仓库、日志、测试脚本、截图、trace。
- 已探测入口：
  - `http://10.10.10.226:8896/`：当前部署版本入口，Playwright MCP 已验证可进入 Studio。
  - `http://10.10.10.226:8080/sign`：返回前端页面，可作为 Playwright UI smoke 的候选入口。
  - `http://10.10.10.226:8888/sign`：返回 Hertz 404，像是后端服务但不是前端入口。
  - `http://10.10.10.226:9888/sign`：当前连接失败，不作为默认入口。
  - `http://10.10.10.226/`：nginx 默认页或静态入口，需要确认是否代理到 Studio。
- 用途：部署后的联调、UI 验收、端到端验证。
- 凭据：由会话提供，禁止写入仓库、日志、测试脚本、截图、trace。测试时通过本地环境变量、临时输入或 Playwright secret 注入。

Playwright MCP 实测记录（2026-06-19）：

- 已打开 `http://10.10.10.226:8896/`，自动进入 `Personal Space`。
- 已进入空间 `7652614054615187456`。
- 已打开超级智能体 `演示超级体`，agent id 为 `7652617174313336832`。
- Super Mode UI 已显示：自主规划、独立沙箱、技能 & MCP、长期记忆。
- 工作区已验证可列出 `/workspace`，包含 `.agent/`、`node_modules/`、`create_excel.py`、`generate_glm52_doc.js`、`generate_glm_news.js`、`glm5_model_comparison.md`、`hello.txt`、`package.json`、`primes.py`、`primes.txt` 等。
- 产出物已验证可列出 `/outputs`，包含 `glm5_model_comparison.xlsx`、`GLM-5.2_技术规格文档.docx`、`glm52_news.docx`。
- 只读对话验证通过：发送“只读取并列出 `/workspace` 下文件名，不要创建/删除/修改文件”，运行过程显示调用 `list_files` 1 个工具，并返回文件列表。
- 技能页现状：已有 4 个技能（pptx/pdf/docx/xlsx），但卡片直接展示长篇 `SKILL.md` 内容，仍缺少标准文件夹编辑、版本、发布、商城化体验。
- 商店现状：全局商店有项目商店、智能体、外部应用、插件商店，未看到技能商城入口。
- 前端 console 发现 1 个错误：`updateRespondingInImmer: cannot find related function call, expect index -1`。它出现在 tool_response 事件处理阶段，后续实现需要修复或收敛。

测试前准备：

1. 确认服务地址、端口、登录路径。
2. 确认当前部署版本或 commit。
3. 创建或找到一个 `agent_type=super` 的测试 agent。
4. 准备一个空间级测试技能、一个全局发布候选技能、一个包含 assets/scripts 的 zip。
5. 测试完清理临时技能、临时 agent、临时 app token。

## 17. Playwright 验收策略

Playwright 不是可选项。涉及 UI 或 app-server 端到端时必须跑。

### 17.1 Super Mode UI

用 Playwright 验证：

- 登录测试环境。
- 进入指定 space。
- 创建或打开超级智能体。
- 工作区能看到 `/workspace`、`/uploads`、`/outputs`。
- 上传文件到 `/uploads`。
- 对话触发 agent 读取上传文件并生成 `/outputs` 产物。
- 预览/下载产物。
- trace 面板显示 tool call 和 artifact。

### 17.2 Harness 写代码与产物

用 Playwright + 沙箱命令验证：

- 上传或初始化一个小型代码项目。
- 要求超级智能体修改代码。
- trace 中出现 inspect/search/edit/run/test 步骤。
- 沙箱中生成 diff 或变更文件。
- 至少运行一次验证命令，失败时能看到重试或错误总结。
- 生成一个可预览或可下载产物到 `/outputs`。
- UI 能预览产物，app-server 能下载同一产物。

### 17.3 标准技能

用 Playwright 验证：

- 创建文件夹技能。
- 编辑 `SKILL.md`。
- 上传 `assets/` 或 `templates/` 文件。
- 保存后版本递增。
- 绑定到超级智能体。
- 对话触发 `read_skill`。
- 沙箱 `/skills/<slug>/` 中存在完整文件树。

### 17.4 发布与商城

用 Playwright 验证：

- 空间级发布：同 space 另一个用户/agent 可安装。
- 全局发布：进入审核队列。
- 审核通过后出现在全局技能商城。
- 从全局商城安装到另一个 space。
- 升级/回滚版本。
- 下架后不可新安装，已安装版本按策略继续或提示升级。

### 17.5 App Server

用 Playwright 或 APIRequestContext 验证：

- 创建 app token。
- 调 `/api/super-agent/runs/create`。
- 读取 SSE stream。
- 上传文件。
- 查询 workspace。
- 下载 artifact。
- 查询 trace。
- 触发审批并 resolve。

### 17.6 不回归验证

必须验证：

- 普通 agent 仍能正常对话。
- 普通 agent 不出现沙箱 bash 工具。
- 普通 workflow 不受 skill package 迁移影响。
- 旧 `/api/skill/list`、`/api/skill/get` 仍兼容。

## 18. 实现阶段

### P0 Spec 与迁移评审

- 完成本 spec。
- 写实施计划。
- 确认 DB migration 与兼容策略。

### P1 Runtime 与 App Server 骨架

- 新增 `/api/super-agent/*` 路由组。
- 建立 run/create/reply/stream/cancel 的 service 边界。
- trace sink 写入内存或 DB 草案。
- 不改变普通 agent。

### P2 Workspace 与 Artifact

- app-server workspace API。
- artifact 下载 token。
- Playwright 验证 `/uploads -> agent -> /outputs`。

### P3 Harness 写代码闭环

- inspect/search/edit/run/test/diff 事件进入 trace。
- 沙箱 `/workspace` 支持代码项目。
- 命令执行输出可流式或可查询。
- 产物写入 `/outputs` 并生成 artifact 元数据。
- Playwright 验证一个前端 demo 或小型代码项目改动。

### P4 标准技能包

- frontmatter parser 升级。
- skill files update 链路。
- skill version 快照包含 files/assets。
- `skill_manage` 升级为多 action + pending。

### P5 发布维度与商城

- `skill_publication`。
- `skill_marketplace_listing`。
- 空间级发布。
- 全局发布审核。
- 安装与版本 pin。

### P6 MCP 与外部接入

- super-agent MCP server。
- token scope。
- 外部 agent 调用验收。

### P7 Codex-like 体验补齐

- trace panel。
- plan panel。
- subagent/deep_task 可视化。
- automations 入口。

## 19. 验收标准

必须全部满足：

1. 普通 agent 行为不变，有回归证据。
2. 超级智能体能完成文件上传、沙箱处理、代码修改、测试验证、产物输出、trace 查看。
3. app-server 可被外部系统创建 run、流式读取、上传文件、下载产物、查询 trace。
4. 技能是标准文件夹包，包含 `SKILL.md`、assets/scripts/references/templates。
5. 支持空间级发布和全局发布。
6. 全局技能商城可浏览、安装、升级、下架。
7. agent 自创技能进入 pending，不会直接污染全局。
8. 发布版本不可变，安装版本可 pin/rollback。
9. Playwright 在 `10.10.10.226` 跑通 UI、Harness 写代码、技能发布、商城安装、app-server 核心链路。
10. 密钥和测试密码没有写入仓库、日志、截图或 trace。

## 20. 开放问题

1. 全局技能发布审核角色：沿用现有管理员权限，还是新增技能审核员角色？
2. app-server token 是否复用现有 open platform token 体系，还是单独表？
3. 技能资产短期继续存在 DB `files` JSON，还是第一阶段直接迁对象存储？
4. 全局商城是否复用现有 marketplace 分类表，还是建立 skill 专属分类？
5. 自动化任务是第一期只预留 API，还是 P6 就提供 UI？

## 21. 结论

推荐按「独立 Codex-like Harness Super Agent Runtime + App Server + 标准技能包 + 空间/全局发布 + 全局技能商城」推进。

这条路线有三个好处：

1. 能对齐 Codex/Hermes 的核心能力模型。
2. 能给外部系统一个稳定 app-server 接入面。
3. 能用 `agent_type=super` 和 `/api/super-agent/*` 把风险圈住，不影响现有系统。
