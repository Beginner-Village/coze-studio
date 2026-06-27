# 策略（Strategy）—— 渐进式披露引擎 设计文档

- 日期：2026-06-26
- 状态：设计待评审（v2：叶子节点泛化为多态"能力项"）
- 适用仓库：`coze-studio`（Go 后端 + React/TS 前端）
- 交付范围：后端功能设计 + 前端页面设计（本次不含实现）

---

## 1. 背景与目标

### 1.1 灵感来源

参考文章《如何构建一个优秀的垂直领域 Agent》（Shortcut 表格 Agent 作者）。核心论点：

> **好 Agent = 对其任务分布的忠实压缩。** 把上下文当成 CPU 的分层缓存来设计：
> - **L1**：常驻系统提示，极小、即时，覆盖 ~80% 高频任务；
> - **L2**：按需加载的"精选规格"（手写的玩法说明 / 文档），覆盖接下来 ~15%，**一次工具调用**即可发现并载入；
> - **L3**：原始全卷 / 可执行能力兜底，长尾，少数几次调用必能定位并执行。
>
> 关键机制是"元工具墙"：能力的 schema/内容不塞进提示，用一两个 meta 工具按需发现、加载、执行。已加载的内容形成"会话级缓存"。
>
> 原则：**工具不是越多越好**——每多一个工具，提示里多一份 schema、多一个易混淆面，准确率随之下降；要在"信息压缩 ↔ 发现速度"之间，把每个能力放到让全局成本最小的那一层。

### 1.2 要解决的问题

当前一个单智能体若要具备大量能力（工作流、插件、知识库、玩法说明），需要把它们逐个绑定为工具/配置，所有 schema 与说明全部常驻上下文：

- 上下文臃肿、信号被淹没 → 准确率下降；
- 工具数量爆炸 → 选错工具概率上升；
- 每个任务都为"用不到的能力"付费。

### 1.3 目标

在资源库新增一个与"工作流"同级的一等公民资源类型 **策略（Strategy）**，把一批能力按"场景"分层组织。**单智能体绑定策略后，自动获得 3 个渐进式披露工具**，让模型按需下钻发现并调用能力，从而：

- 系统提示里只常驻"一句话 + 3 个工具"，无论策略背后挂多少能力；
- 模型先看"有哪些场景"，再看"某场景下有哪些能力项"，最后才"调用"；
- 把异构能力（工作流/插件/知识库/提示词）统一做成可被 ReAct 循环逐步发现的分层缓存。

### 1.4 关键范围说明

- 这 3 个工具**只注入给绑定了该策略的那个单智能体**，跟随其 ReAct 循环；绑定即注入、解绑即消失。不是对外 MCP、不是多智能体机制。
- 采用**纯三段披露**模型（L1=系统提示一句话、L2=两个 list 工具、L3=invoke 执行），**不做**"披露热度标签"。
- 叶子能力项**引用资源库已有资源**（工作流/插件/知识库），不在策略内重造；唯一例外是"纯提示词"类型，其内容**内联存储**在策略里（它本就是一段手写文本）。
- 知识库可"在 agent 同级直接绑定" **或** "作为策略的能力项披露"，二者并存、不冲突。

---

## 2. 概念模型

### 2.1 三层结构

```
策略 Strategy（资源库新增一等公民，与"工作流"同级）
│
│   L1 提示（常驻系统提示，1 句）：
│   "你绑定了策略「银行对公运营」，处理相关请求前先调用 strategy_list_scenarios"
│
├─ 场景 Scenario（策略下平铺一层，"能力分组"）
│   ├─ 对公开户
│   ├─ 账户异常处理
│   └─ 贷后管理
│
└─ 能力项 Capability（叶子，多态：工作流 / 插件 / 知识库 / 纯提示词）
    ├─ [对公开户]    → 开户预审(工作流)、开户规则说明(提示词)
    ├─ [账户异常处理] → 冻结(工作流)、风控查询(插件)、异常处置手册(知识库)
    └─ [贷后管理]    → 还款提醒(工作流)、催收话术(提示词)
```

### 2.2 L1/L2/L3 与三个工具的对应

| 层 | 工具 / 载体 | 模型看到什么 | 对应文章语义 |
|---|---|---|---|
| **L1** | 系统提示（0 次工具调用） | "有策略 X，先 `list_scenarios`" | 常驻、极小 |
| **L2** | `strategy_list_scenarios` → `strategy_list_capabilities` | 场景目录 → 某场景内能力项（含类型与入参 schema） | 一次"缓存未命中"按需加载 |
| **L3** | `strategy_invoke_capability` | 按能力项类型分发：执行 / 检索 / 注入提示 | 兜底执行与精选规格披露 |

**核心收益**：一个智能体哪怕背后挂 50 个异构能力，系统提示里也只有 1 句话 + 3 个工具的 schema；模型只在处理某类任务时，才下钻加载那一个场景的能力项。

### 2.3 能力项类型（多态叶子）

第 3 个工具 `strategy_invoke_capability` 按 `type` 分发，每种类型复用 coze-studio 现成通路：

| 类型 | invoke 行为 | 复用现有通路 | invoke 入参 | 对应文章 |
|---|---|---|---|---|
| `workflow` | 执行已发布工作流 | workflow 执行引擎（`crossworkflow`） | 工作流自身入参 | L3 执行 |
| `plugin` | 调用插件工具 | `crossplugin.DefaultSVC().ExecuteTool` | 插件工具入参 | L3 执行 |
| `knowledge` | 按 query 检索知识库召回 | knowledge recall 服务 | `{query, top_k?}` | L2/L3 检索 |
| `prompt` | **直接返回内联提示文本**（不执行，只注入上下文） | 无需执行 | `{}`（可选变量，v1 不做） | **L2 精选规格披露** |

> 模型只用 `capability_id` 调用，无需关心底层类型/目标——`type` 与目标解析在后端完成。`list_capabilities` 已把每项的 `type`、面向模型描述、`input_schema` 给到模型。

---

## 3. 架构方案选型

| 方案 | 做法 | 取舍 | 结论 |
|---|---|---|---|
| A 合成插件 | 把策略伪装成含固定工具的"插件"，复用 `agent_tool` 绑定 | 复用最多，但污染 plugin 域；场景/能力项编辑 UI 套不进插件模型；hacky | 否 |
| B 全新链路 | 新资源类型 + 新表 + 新绑定 + 新运行时，全套自建（含执行引擎） | 干净，但各类能力执行链路重复造轮子 | 否 |
| **C 混合** | 新资源类型 + 新表 + 独立编辑 UI；运行时用现有 eino `tool.InvokableTool` 抽象表达 3 个工具，invoke 按类型**路由进现有执行/检索通路** | 建模干净 + 执行零重复；新增代码集中在"存储 / 编辑 / 装配 / 分发"，执行白嫖现有管线 | **采用** |

**工具数量决策**：泛化叶子后**仍固定 3 个工具**（不为新增类型加第 4 个工具）——守文章"少工具"原则，能力项类型是数据、不是工具。

---

## 4. 后端设计（方案 C）

> 端到端镜像现有 **database** 资源类型的接线（它同样是"库资源 + 绑定 agent + 注入工具"）。

### 4.1 注册资源类型

文件：`backend/api/model/resource/common/resource_common.go`（现状枚举到 `ResType_Voice = 9`）。新增：

```go
ResType_Strategy ResType = 10
```

并在 `backend/application/search/resource_pack.go` 的 `NewResourcePacker` switch 增加 `strategyPacker`（仿 `databasePacker`），使策略进入统一资源库列表/筛选/操作管线。

### 4.2 数据模型（3 张新表）

**`strategy`** —— 策略主体
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 策略 ID |
| space_id | bigint | 所属空间 |
| app_id | bigint | 所属应用（可空） |
| creator_id | bigint | 创建者 |
| name | varchar | 名称 |
| description | varchar | 描述（也用作 L1 提示里的一句话领域说明） |
| icon_uri | varchar | 图标 |
| status | tinyint | 0=草稿 1=已发布 |
| version | varchar | 已发布版本号 |
| created_at / updated_at | bigint(ms) | 时间戳 |
| deleted_at | datetime | 软删 |

**`strategy_scenario`** —— 场景（策略下平铺一层）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 场景 ID |
| strategy_id | bigint | 所属策略 |
| name | varchar | 场景名（面向模型，简短） |
| description | varchar | 场景描述（面向模型："这类任务进这里"） |
| sort_order | int | 排序 |
| created_at / updated_at | bigint(ms) | 时间戳 |
| deleted_at | datetime | 软删 |

**`strategy_capability`** —— 能力项（多态叶子，核心表）
| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 能力项 ID（模型用它调用） |
| strategy_id | bigint | 冗余，便于按策略聚合 |
| scenario_id | bigint | 所属场景 |
| type | varchar(16) | `workflow` / `plugin` / `knowledge` / `prompt` |
| ref_id | bigint NULL | 被引用资源 ID：workflow_id / plugin_tool_id / knowledge_id；`prompt` 为空 |
| ref_sub_id | bigint NULL | 仅 `plugin` 用：plugin_id（工具所属插件）；其余空 |
| ref_version | varchar NULL | workflow_version / plugin_version；空=最新已发布 |
| prompt_content | text NULL | 仅 `prompt` 用：内联提示文本 |
| retrieve_config | json NULL | 仅 `knowledge` 用：`{top_k, min_score, ...}` |
| alias_name | varchar NULL | 面向模型的名称覆盖；空则回落资源自身名 |
| alias_description | text NULL | 面向模型的"精选描述/玩法说明"（披露质量关键，空则回落资源自身描述） |
| sort_order | int | 排序 |
| created_at / updated_at | bigint(ms) | 时间戳 |
| deleted_at | datetime | 软删 |

> **`alias_description` 是质量关键。** 文章里 L2 的精髓是"手写的、含注意事项的精选规格，而非 schema dump"。策略作者可为每个能力项写一段面向模型的描述（何时用、前置条件、坑）。

### 4.3 注入给单智能体的 3 个工具

设计原则：**无论绑定几个策略、能力项多少种类型，固定只注入 3 个工具**，靠参数在树上导航。

#### 工具 1：`strategy_list_scenarios`
- 面向模型描述：列出指定策略下可用的"场景（能力分组）"。处理某领域任务前先调用本工具了解可做哪些事。
- 入参：
```json
{ "type": "object", "properties": {
  "strategy_id": { "type": "string", "description": "策略 ID；仅绑定一个策略时可省略" }
}}
```
- 返回：
```json
[ { "strategy_id":"...", "scenario_id":"...", "name":"对公开户", "description":"对公账户的开立与预审", "capability_count":2 } ]
```

#### 工具 2：`strategy_list_capabilities`
- 面向模型描述：列出某场景下的能力项及其类型与入参 schema，据此选择并准备调用。
- 入参：
```json
{ "type": "object", "required": ["scenario_id"], "properties": {
  "scenario_id": { "type": "string" }
}}
```
- 返回：
```json
[
  { "capability_id":"...", "type":"workflow", "name":"开户预审", "description":"<alias_description 优先>", "input_schema": { "type":"object", "properties": {} } },
  { "capability_id":"...", "type":"knowledge", "name":"异常处置手册", "description":"按问题检索处置手册", "input_schema": { "type":"object", "required":["query"], "properties": { "query":{"type":"string"} } } },
  { "capability_id":"...", "type":"prompt", "name":"开户规则说明", "description":"对公开户的合规要点", "input_schema": { "type":"object", "properties": {} } }
]
```
> **直接返回 `input_schema`**：模型一步即可据此组装参数并调用，避免新增"取 schema"工具。

#### 工具 3：`strategy_invoke_capability`
- 面向模型描述：按 `capability_id` 调用该能力项并返回结果（工作流执行 / 插件调用 / 知识库检索 / 返回提示文本）。
- 入参：
```json
{ "type": "object", "required": ["capability_id"], "properties": {
  "capability_id": { "type": "string" },
  "arguments": { "type": "object", "description": "按工具2返回的 input_schema 提供；prompt 类型可省略" }
}}
```
- 返回：能力项结果（透传执行/检索输出，或提示文本）。
- 后端分发（按 `strategy_capability.type`）：
  - `workflow` → 校验 ∈ 当前绑定 → 路由进 workflow 执行引擎（同 `node_tool_workflow.go` invoke 路径）；
  - `plugin` → `crossplugin.DefaultSVC().ExecuteTool(...)`（同 `pluginInvokableTool.InvokableRun`）；
  - `knowledge` → knowledge recall 服务，入参 `{query, top_k?}`；
  - `prompt` → 直接返回 `prompt_content`。
- **越权防护**：`capability_id` 必须 ∈ 当前智能体所绑策略的能力项集合，禁止借工具调任意资源。

### 4.4 智能体绑定模型

与"工作流挂到智能体"同构：在单智能体草稿/发布配置（`entity.SingleAgent`）新增 `strategies: [{strategy_id, version}]`。草稿态编辑器增删；发布态随智能体发布固化（version 默认"最新已发布"）。

### 4.5 运行时注入

文件：`backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go`（第 239–255 行那串 `agentTools = append(...)`，工具抽象为 eino `tool.InvokableTool`/`tool.BaseTool`）。

1. 新增 `node_tool_strategy.go`，实现 `newStrategyTools(ctx, conf *strategyConfig) ([]tool.InvokableTool, error)`，构造 3 个工具（每个实现 eino `Info()` + `InvokableRun()`，参考 `node_tool_plugin.go` / `node_tool_sandbox.go`）。
2. 在 builder 装配处追加（仿现有 db/av 工具）：
```go
agentTools = append(agentTools, slices.Transform(strategyTools, func(a tool.InvokableTool) tool.BaseTool { return a })...)
```
3. **L1 提示注入**：在 `system_prompt.go` 的 `REACT_SYSTEM_PROMPT_JINJA2` 增加 `{% if bound_strategies %}…{% endif %}` 块；在 builder 增加 `strategiesRenderer` lambda 节点 + 输出键 `bound_strategies`（仿 `skillsRenderer`/`placeholderOfAvailableSkills`）。

运行时数据流：

```
模型 ReAct 循环
  │  (L1: 系统提示已知"有策略X")
  ├─► strategy_list_scenarios(X)            → 场景目录
  ├─► strategy_list_capabilities(场景A)     → 能力项 + type + input_schema
  └─► strategy_invoke_capability(cap, args)
            │  校验 cap ∈ 已绑策略能力项
            ├─ workflow → workflow 执行引擎
            ├─ plugin   → crossplugin.ExecuteTool
            ├─ knowledge→ knowledge recall
            └─ prompt   → 返回内联文本
                  └─► 结果回灌模型
```

### 4.6 管理面 API（CRUD / 发布）

走现有 thrift IDL + hz 代码生成 + application 层（仿 `idl/data/database/database_svc.thrift` → `backend/api/handler/coze/` → `backend/application/...`）：

- 策略：`Create / Update / Delete / GetDetail / List(随通用资源列表)` / `Publish`
- 场景：`CreateScenario / UpdateScenario / DeleteScenario / ReorderScenario`
- 能力项：`AddCapability / UpdateCapability(alias/prompt/retrieve) / RemoveCapability / ReorderCapability`
- （可选 P1）`PreviewStrategyTools`：返回"3 个工具分别会给模型看到什么"。

### 4.7 新增领域目录结构

```
backend/domain/strategy/
  ├─ entity/                # Strategy / Scenario / Capability 领域模型
  ├─ service/               # CRUD、发布、能力项解析与分发数据准备
  └─ internal/
      ├─ dal/model/         # GORM gen 模型（3 张表）
      └─ repo/              # 仓储实现
backend/application/strategy/   # 应用层，对接 API + 资源事件
backend/api/model/.../strategy/ # thrift 生成的请求/响应模型
backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy.go # 3 工具 + 分发
```

---

## 5. 前端设计（贴 `@coze-arch/coze-design` 现有风格）

### 5.1 资源库入口注册

- 新增 `entry-base/src/pages/library/hooks/use-entity-configs/use-strategy-config.tsx`（仿 `use-database-config.tsx`）：`typeFilter`「策略」、`target:[ResType.Strategy]`、`onCreate` 开"新建策略"弹窗、`onItemClick` 跳编辑器、`renderActions` 删除等。
- 前端枚举 `ResType` 加 `Strategy = 10`（`frontend/packages/arch/idl/src/auto-generated/plugin_develop/namespaces/resource_resource_common.ts`）。
- 在 `entry-adapter/src/pages/library/index.tsx` 注册（仿 `useDatabaseConfig`：import + 调用 + 加进 `entityConfigs` 数组 + 渲染 `strategyModals`）。

### 5.2 策略列表

复用资源库列表（`BaseLibraryPage`），由 §5.1 entity config 渲染——卡片、网格/列表切换、筛选、搜索全部白嫖。卡片副信息显示"N 个场景 · M 个能力项 + 发布状态"。

### 5.3 策略编辑器（新页面）

整页路由（非弹窗，因需两级树 + 逐项编辑）：`apps/coze-studio/src/routes.tsx` 加 `/strategy`（仿 `work_flow` 的 lazy + loader）。布局：

```
┌─ Header: 策略名 / 描述 / [预览] [发布] ─────────────────────────┐
├──────────────┬─────────────────────────────────────────────────┤
│ 场景列表(左)  │  选中场景的能力项(主区)                            │
│ ▸ 对公开户    │  场景名 + 场景描述(可编辑)                         │
│ ▸ 账户异常 ✓  │  ┌───────────────────────────────────────────┐  │
│ ▸ 贷后管理    │  │ 能力项列表: [类型标签] 名称 | 面向模型描述 | 操作│ │
│              │  │ [+ 添加能力项 ▾]                              │ │
│ [+ 添加场景]  │  │   ├ 工作流  → WorkflowModalBase 选已发布工作流 │ │
│              │  │   ├ 插件    → 选插件工具                       │ │
│              │  │   ├ 知识库  → 选 KB + 检索参数                 │ │
│              │  │   └ 提示词  → 内联富文本编辑                   │ │
└──────────────┴─────────────────────────────────────────────────┘
```

- 左侧场景可拖拽排序、重命名、删除；
- "添加能力项"是一个分类型下拉：工作流/插件复用现有选择器（`WorkflowModalBase` / 插件选择器），知识库选 KB 并配检索参数，提示词内联编辑文本；
- 每个能力项行内可编辑"面向模型的别名/描述"（`alias_name`/`alias_description`）；能力项可跨场景移动、排序；
- 能力项前置**类型标签 Tag**（工作流/插件/知识库/提示词，配色区分）；
- （可选 P1）"预览"抽屉：展示 3 个工具对模型的实际返回。

### 5.4 智能体绑定

- 复用技能区弹窗模式（`agent-ide/.../external-knowledge-modal.tsx` 或 `BotWorkflowModal` 为范本）做 `BotStrategyModal`，从资源库选策略。
- 绑定后在技能列表显示策略卡，副标题标注"提供 3 个渐进披露工具：列场景 / 列能力 / 调用"。
- store：`frontend/packages/studio/stores/bot-detail/src/store/bot-skill/store.ts` 仿 `workflows`（state 默认 `[]`、类型、`updateSkillStrategies`、`transformDto2Vo.strategy`/`transformVo2Dto.strategy`、init）新增 `strategies`。

### 5.5 i18n

`studio-i18n-resource/src/locales/{zh-CN,en}.json` 新增：`library_resource_type_strategy`、`navigation_workspace_library_strategy`、`strategy_create`、`strategy_scenario_add`、`strategy_capability_add`、能力项类型标签、`strategy_model_facing_desc`、`strategy_bind_hint` 等。

### 5.6 设计规范复用

组件全用 `@coze-arch/coze-design`（Button/Modal/Table/Tree/Collapse/Input/Tag/Typography/Toast/Menu）；图标 `@coze-arch/coze-design/icons` `IconCoz*`；样式 CSS Modules `*.module.less` + `--coz-*` token；卡片圆角 14px、按钮 4px；主按钮 `type="primary" theme="solid"`。与库页、工作流编辑器视觉一致。

---

## 6. 关键决策与边界

| 主题 | 决策 |
|---|---|
| 披露模型 | 纯三段披露（策略>场景>能力项），不做热度标签；L1 系统提示只放策略名 + 一句引导。 |
| 工具数量 | 固定 3 个工具，靠参数导航；泛化叶子不加第 4 个工具。 |
| 叶子多态 | 能力项 type ∈ {workflow, plugin, knowledge, prompt}；invoke 按 type 分发，复用现有执行/检索通路。 |
| 能力来源 | workflow/plugin/knowledge **引用**资源库已有；prompt **内联**存储。 |
| `list_capabilities` 返回 | 带 type + `input_schema`，省掉"取 schema"工具。 |
| 越权防护 | `invoke` 校验 capability ∈ 当前智能体所绑策略能力项集合。 |
| 失效保护 | 被引用资源被删/下架时，`list/invoke` 返回明确错误给模型，不崩。 |
| 知识库双绑 | 可在 agent 同级直接绑 KB，也可作能力项披露，二者并存。 |
| 版本策略 | v1 跟随现有资源 draft+publish；绑定默认"最新已发布"。 |
| 作用域 | 这 3 个工具仅注入给绑定该策略的单智能体；空间隔离沿用现有权限。 |
| 命名 | 中文"策略/场景/能力项"；模型工具名 `strategy_list_scenarios` / `strategy_list_capabilities` / `strategy_invoke_capability`。 |

---

## 7. 改动文件清单（已核对锚点）

### 后端
| 文件 | 改动 |
|---|---|
| `backend/api/model/resource/common/resource_common.go` | 新增 `ResType_Strategy = 10` + `String()/FromString()` |
| `idl/data/strategy/strategy_svc.thrift`（新增） | 策略/场景/能力项 CRUD + 发布 IDL，hz 生成 |
| `backend/api/handler/coze/strategy_service.go`（生成/新增） | HTTP handler（仿 database_service.go） |
| `backend/application/strategy/**`（新增） | 应用层 CRUD + 资源事件发布 |
| `backend/domain/strategy/**`（新增） | entity / service / dal / repo |
| `backend/application/search/resource_pack.go` | 新增 `strategyPacker` + 注册进 `NewResourcePacker` |
| `backend/domain/agent/singleagent/internal/agentflow/node_tool_strategy.go`（新增） | `newStrategyTools` 构造 3 工具 + 按 type 分发 |
| `backend/domain/agent/singleagent/internal/agentflow/agent_flow_builder.go` | append 策略工具（仿 239–255 行）+ strategiesRenderer 节点 |
| `backend/domain/agent/singleagent/internal/agentflow/system_prompt.go` | 加 `bound_strategies` Jinja2 块 |
| `entity.SingleAgent` 绑定配置 | 新增 `strategies` 字段 |
| DB DDL（`docs/ynet-database-sql/*.sql` + atlas migration） | 建 3 张新表 |

### 前端
| 文件 | 改动 |
|---|---|
| `entry-base/src/pages/library/hooks/use-entity-configs/use-strategy-config.tsx`（新增） | 策略 entity config |
| `.../use-entity-configs/index.ts` + `entry-adapter/.../library/index.tsx` | 导出 + 注册 `useStrategyConfig` |
| `frontend/packages/arch/idl/src/auto-generated/plugin_develop/namespaces/resource_resource_common.ts` | `ResType.Strategy = 10` |
| `apps/coze-studio/src/routes.tsx` | 新增 `/strategy` 路由 + lazy import |
| `frontend/packages/.../strategy-*`（新增包/目录） | 策略编辑器页 + `BotStrategyModal` |
| `frontend/packages/studio/stores/bot-detail/src/store/bot-skill/store.ts` + `transform.ts` + `types/skill.ts` | 新增 `strategies` slice |
| `studio-i18n-resource/src/locales/{zh-CN,en}.json` | 新增 i18n 键 |

---

## 8. 分期建议

- **MVP（P0）**：资源类型注册 + 3 张表 + 策略/场景/能力项 CRUD + 编辑器（场景树 + 四类能力项 + alias）+ 智能体绑定 + 运行时注入 3 工具 + 四类 invoke 分发 + 越权/失效保护。
- **P1**：编辑器"预览 3 工具返回"调试抽屉；`PreviewStrategyTools` API；prompt 类型变量插值。
- **P2**：能力项 pin 具体版本；策略复制/导入导出；披露热度标签（L1 高频场景直接进系统提示）。

---

## 9. 开放问题（待评审确认，均已给默认值）

1. 策略编辑器整页路由（采用）vs 大弹窗。
2. "预览 3 工具返回值"调试面板是否进 MVP（默认 P1）。
3. 单智能体可绑多个策略（默认支持）vs 限制 1 个。
