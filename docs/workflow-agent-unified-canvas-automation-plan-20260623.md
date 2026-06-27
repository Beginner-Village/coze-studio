# 工作流画布统一自动化接入方案

## 目标

把工作流/对话流画布能力从“finmallclaw 右侧聊天的内嵌能力”升级成一套统一的画布自动化协议。平台内置智能体、外部 MCP 客户端、Codex、Claude Code、后续其他界面自动化能力，都通过同一套能力发现、画布编辑、变量绑定、校验、试运行、修复接口工作。

## 当前事实

现有实现已经具备核心闭环:

- 后端 `workflow_canvas_*` 工具让模型发出画布命令。
- 前端 `WorkflowAgentCommandService` 在浏览器内存画布中执行 add/connect/delete/configure/layout/test_run。
- 前端会提供节点目录、画布摘要、可绑定变量、空间资源清单、绑定指南。
- 语义配置器已覆盖 LLM、插件/API、代码、知识库、IF、输出、文本处理、问答、输入、变量聚合、JSON 序列化、卡片选择、智能体、End。
- 系统后端已经提供 `/api/workflow_mcp/mcp` 作为同源 MCP endpoint,复用 `backend/application/workflow/canvasautomation`。
- MCP 已暴露原子写工具 `workflow.add_node/connect/configure_node/set_node_params/delete_node/delete_line/clear_canvas/auto_layout`;外部 MCP 实时模式会把命令投递到打开的浏览器画布,等待 Browser bridge 回传真实 `WorkflowCanvasCommandResult` 后再返回给 MCP client。平台内 finmallclaw 聊天流仍保留 `dispatched_to_canvas` ack,由前端聊天面板继续执行。
- 主后端 MCP route 已接入内存命令中继:外部 MCP 写工具带 `workflow_id/space_id` 时,命令会进入 relay;浏览器可通过 `/api/workflow_mcp/browser_commands` 拉取 `canvas_automation.v0` 命令。
- 前端工作流页已挂载 Browser bridge 轮询 `/api/workflow_mcp/browser_commands`,并把返回的命令交给 `WorkflowAgentCommandService.applyCommandEnvelope` 执行;cursor 存在 sessionStorage 中,避免刷新后重复执行旧命令。
- Browser bridge 已支持双向请求/响应:后端 poll 响应中的 `requests[].request_id/requires_response` 会被前端拆成单条 envelope 执行,需要响应的结果会 POST 到 `/api/workflow_mcp/browser_command_results`。
- Browser bridge 执行 `requires_response` 命令时,即使前端应用命令抛错,也会回传 `status=failed` 的结果,避免外部 MCP 客户端只能等超时。
- 浏览器 relay 已处理后端重启后的 stale cursor:当前端 sessionStorage 中的 cursor 大于后端最新命令 ID 时,后端会重置读取窗口,并把响应 cursor 对齐到实际返回的最新命令。
- MCP `workflow.get_canvas_context` 和 `workflow.get_bindable_variables` 在主后端中已走 browser-live 双向 relay:外部 MCP 客户端发起读取,打开的浏览器画布执行只读命令并回传真实画布摘要/可绑定变量。
- MCP `workflow.test_run` 走后端测试路径返回结构化结果,不需要打开前端试运行抽屉,用于避免聊天面板在测试期间被 UI 重置。
- MCP `workflow.node_smoke_manifest` 提供系统侧 41 节点巡检清单,包含 ready/skip、planned commands、cleanup commands、resource fixture 需求和 temporary workflow opt-in 标记。`workflow.node_smoke_coverage` 在此基础上返回每个节点的 coverage 总表:是否有 manifest/spec/assertions/expected bindable variables、具体 assertions、具体 expected bindable variables、required/planned/cleanup commands、temporary node tags、isolation、是否可临时执行、是否只读、是否需要资源 fixture、是否 partial/sub-canvas。它是外部 Codex/Claude Code 与内部 finmallclaw 共用的节点验收入口。
- MCP `workflow.run_node_smoke` 已接入 browser-live bridge:外部 MCP 客户端可请求打开的浏览器画布按 manifest 执行一个或多个节点 smoke plan,前端 runner 通过 `WorkflowAgentCommandService` 逐步加节点、连线、配置、清理,并把完整 `node_smoke_report` 和面向 agent 修复循环的精简 `node_smoke_summary` 回传给 MCP client。
- MCP `workflow.list_surfaces` 已新增系统级 surface 发现入口。它把普通工作流画布、ChatFlow 画布、ChatFlow 角色配置、ChatFlow 会话模板和空间资源目录统一暴露成机器可读能力面,让模型先确认“哪个界面/配置面可读写、走 browser-live 还是 backend-api、有哪些工具和缺口”,再进入节点级编辑。
- MCP `workflow.list_resource_catalog` 已新增资源目录契约入口。它覆盖插件/API、知识库、智能体、模型、子工作流、数据库、卡片、MCP 服务、HTTP 端点、图像/媒体、触发器、记忆等 12 类资源族,并标明适用节点、已有后端 API 名、必须先发现真实 ID/schema、禁止编造 ID。当前仍只返回 resource contract,不返回 live resource instances。
- MCP `workflow.get_operation_guide` 已新增渐进式操作规程入口。它把“先 surface、再资源目录、再节点能力/规格、再画布上下文、再可绑定变量、再配置、再布局、再 test_run/局部修复”的顺序和 guardrails 结构化返回给 finmallclaw/Codex/Claude Code,避免模型靠长提示词记忆或一错就清空画布。
- 平台内 finmallclaw `workflow_canvas_get_operation_guide` 已同步渐进式操作规程。内部聊天必须先读规程,再按 `workflow_canvas_*` 工具完成节点发现、资源发现、规格读取、画布上下文、可绑定变量、配置、布局、试运行和失败局部修复。
- 外部 MCP `workflow.get_node_spec` 已补齐核心编辑节点的具体规格,覆盖 3/4/5/6/8/13/15/18/22/30/32/58/59/99/100。其余节点也不再返回空泛 fallback,而是返回 singleton/resource-bound/partial/add-only/documentation-only 的安全边界、可用命令和 gaps。Code 节点明确平台 Python Args 运行时,VariableMerge/End/Text/Output 明确真实可绑定变量和 display-only 边界。
- 平台内 finmallclaw `workflow_canvas_get_node_spec` 已同步全量覆盖 41 个 smoke manifest 节点。未完整支持节点不再返回“未找到详细规格”,而是返回内置单例、resource-bound、partial、add-only、documentation-only 的安全边界和可用 `workflow_canvas_*` 工具,避免平台内智能体和外部 MCP 看到两套不同说明。
- 平台内 finmallclaw `workflow_canvas_get_resource_catalog` 已同步资源目录契约。它和外部 MCP `workflow.list_resource_catalog` 一样覆盖 12 类资源族,支持 `node_type`/`family` 过滤,并要求 resource-bound 节点配置前先确认真实资源 ID/schema,禁止编造插件/API、知识库、智能体、模型、MCP、HTTP、数据库等资源。
- 平台内 finmallclaw `workflow_canvas_get_node_smoke_coverage` 已同步节点覆盖度入口。它返回 41 个节点的 manifest/spec/assertions/expected bindable variables 覆盖、具体 assertions、具体 expected bindable variables、required/planned/cleanup tools、temporary node tags、isolation、execute/readonly/skip 分类、resource fixture/partial 缺口和是否能作为下游变量来源,用于在读取具体节点规格前先判断节点边界和 smoke 操作步骤。

主要缺口:

- 真实节点 registry 比已支持的语义配置器更多。循环、批处理、数据库 CRUD、HTTP、MCP、子工作流、触发器、LTM、图像类节点等还没有完整语义配置器。
- 部分节点能添加但不能被 agent 可靠配置，模型容易靠猜表单路径。
- 节点说明、前端配置器、后端工具说明不是同一个机器可读来源。
- 外部工具要做到“实时可见”需要当前工作流页面打开,让浏览器桥在线轮询系统命令中继并执行/回传;无浏览器场景还需要 Backend draft adapter 直接改草稿 JSON。

## 统一协议分层

### 0. Surface Discovery Layer

模型第一步不应该直接猜“添加哪些节点”,而是先调用 `workflow.list_surfaces` 获取系统当前暴露的可操作能力面:

- `workflow.canvas`: 普通工作流画布,browser-live,已实现 `canvas_automation.v0`。
- `chatflow.canvas`: ChatFlow 画布,browser-live,复用同一套 `canvas_automation.v0` 节点编辑、变量绑定和 smoke 工具。
- `chatflow.role_settings`: ChatFlow 角色/欢迎语/建议回复等设置面,后端已有 `GetChatFlowRole/CreateChatFlowRole/DeleteChatFlowRole`,MCP 字段级读写仍待封装。
- `chatflow.conversation_templates`: ChatFlow 会话模板面,后端已有 `List/Create/Update/DeleteApplicationConversationDef`,MCP CRUD 仍待封装。
- `space.resource_catalog`: 空间资源目录面,已通过 `workflow.list_resource_catalog` 暴露资源族和发现契约;后续 live list 工具会继续返回真实插件/API、知识库、智能体、模型、卡片、子工作流等资源实例。

surface 发现返回的重点字段:

- `status`: `implemented` / `planned-backend-api` 等,决定能不能直接写。
- `adapter`: `browser-live` 或 `backend-api`,决定是否依赖当前页面打开。
- `command_protocol`: 例如 `canvas_automation.v0`、`surface_settings.v0`、`resource_catalog.v0`。
- `tools`: 已可调用 MCP 工具。
- `backend_apis`: 已存在但尚未封装成 MCP 的系统 API。
- `gaps`: 不能自动完成或不能宣称已完整支持的原因。

这个入口的意义是把“工作流、对话流、右侧角色配置、资源目录”都放到同一套系统 MCP 能力树下。finmallclaw、Codex CLI、Claude Code 和后续 UI 都先读 surface,再读节点目录/节点规格/可绑定变量。

### 0.1 Resource Catalog Layer

资源类节点不能靠模型猜 ID。外部入口使用 `workflow.list_resource_catalog`,平台内 finmallclaw 使用 `workflow_canvas_get_resource_catalog`;二者是所有插件/API、知识库、智能体、模型、数据库、HTTP、MCP、卡片等资源配置前的共同入口。

当前已返回的资源族:

- `plugin_api`: 适用于 3/4,指向 `GetPlaygroundPluginList`,要求按 API schema 绑定参数。
- `knowledge_base`: 适用于 3/6/27,指向 `ListKnowledgeDetail`,要求确认 dataset/knowledge ID。
- `agent`: 适用于 3/100,要求确认真实 bot_id/agent_id 和动态参数 schema。
- `model`: 适用于 3/22,要求确认当前空间允许的模型。
- `workflow`: 适用于 9,要求确认子工作流 ID、版本和输入输出 schema。
- `database`: 适用于 12/42/43/44/46,要求确认表、字段、主键、条件字段和权限。
- `card`: 适用于 99,要求确认卡片模板、选项 schema 和返回字段。
- `mcp_server`: 适用于 61,要求确认 server/tool/input schema/auth。
- `http_endpoint`: 适用于 45,要求确认 method/url/headers/body/response outputs。
- `image_asset`: 适用于 14/16/17/23,要求确认图片资源、图像工作流或生成模型能力。
- `trigger`: 适用于 34/35/36,要求确认触发器类型、schema 和权限。
- `memory`: 适用于 26,要求确认记忆作用域、权限和会话上下文来源。

这个层目前是 `contract-ready`: 已经能防止模型“凭空编造资源”,但尚未返回 live resource instances。后续要继续补 `workflow.list_resources(family, space_id)` 或各族专用工具,把真实资源实例和 schema 通过同一个 MCP 服务吐给内部/外部客户端。

### 0.2 Operation Guide Layer

`workflow.get_operation_guide(task_type)` 是所有客户端进入画布编辑前的统一规程入口。它不是 prompt 文案,而是系统 MCP 返回的机器可读流程:

1. `workflow.list_surfaces`: 确认当前要操作的 surface 和 adapter。
2. `workflow.list_resource_catalog`: 涉及资源类节点时先确认资源族、真实 ID/schema 发现方式和缺口。
3. `workflow.list_node_capabilities` / `workflow.get_node_spec`: 先知道有哪些节点、节点能做什么、哪些节点不能直接完整配置。
4. `workflow.get_canvas_context`: 基于最新真实画布而不是旧快照修改。
5. `workflow.add_node` / `workflow.connect` / delete 系列: 先完成拓扑,Start/End 只连接和配置。
6. `workflow.get_bindable_variables`: 配置任何变量引用前必须读取真实可绑定变量。
7. `workflow.configure_node`: 逐节点配置 input/inputs、outputs、condition、merge_groups、returns、content、prompt、code 或资源参数。
8. `workflow.auto_layout`: 每组 add/connect/configure 后自动或显式布局。
9. `workflow.test_run` / `workflow.explain_failure`: 测试失败后先解释并局部修复对应节点,不要默认 `clear_canvas`。

该规程还固定以下 guardrails:

- `dispatched_to_canvas` 不是成功,必须等待 browser_live 结果或重新读上下文。
- 不要默认清空画布;只有用户明确要求重建或已证明局部修复不可行时才用 `clear_canvas`。
- 一个工作流只能有一个 Start/End。
- `type=13` 是 display-only 输出/纯输出,不能作为变量聚合或 End returns 的真实变量来源;需要下游消费时用 `type=15` 文本处理产出 `output:string`。
- 变量聚合、IF、End 返回文本、Prompt 模板、代码/HTTP/API 参数都必须先读 `workflow.get_bindable_variables`。

### 1. Canvas Automation Protocol v0

所有内部和外部调用都统一成同一个命令 envelope。不同入口只负责 transport 和鉴权，不重新定义业务语义。

```json
{
  "protocol": "canvas_automation.v0",
  "surface": "workflow",
  "space_id": "7652614054615187456",
  "canvas_id": "7654449287564099584",
  "mode": "browser_live",
  "request_id": "agent-turn-uuid",
  "commands": [
    {
      "op": "configure_node",
      "target": "merge",
      "args": {
        "config": {
          "merge_groups": [
            {
              "name": "output",
              "variables": [
                { "from": "text_success", "output": "output" },
                { "from": "text_fail", "output": "output" }
              ]
            }
          ]
        }
      }
    }
  ]
}
```

标准响应:

```json
{
  "protocol": "canvas_automation.v0",
  "request_id": "agent-turn-uuid",
  "status": "ok",
  "results": [
    {
      "op": "configure_node",
      "ok": true,
      "node_id": "130001",
      "diagnostics": []
    }
  ],
  "canvas_context": "节点:\\n开始(100001,type=1) outputs: input:string\\n聚合(130001,type=32) outputs: output:string merge_groups: output=[120001.output]\\n连线: 100001->120001, 120001->130001\\n可绑定变量: 100001.input:string, 120001.output:string, 130001.output:string\\n绑定诊断: none",
  "bindable_variables": "100001.input:string\\n120001.output:string\\n130001.output:string",
  "binding_diagnostics": []
}
```

关键约束:

- 命令必须是语义命令优先: `configure_node` 接收节点意图和绑定变量，不暴露内部表单 path。`set_node_params` 只作为专家兜底。
- `add_node/connect/configure_node/delete/clear/auto_layout/test_run` 在内部聊天、外部 MCP、后端 API 中保持同名同义。
- 每条写命令返回 `ok` 只代表命令已被对应 adapter 接收/应用，不代表工作流可运行。可运行必须看 validate/test_run 和 binding diagnostics。
- 所有引用变量必须能被 `get_bindable_variables` 解释成真实 `node_id.output`。type=13 输出节点如果没有真实 outputs，不能被聚合或 End 返回。

### 2. Capability Layer

机器可读的能力矩阵，记录每个节点:

- type、名称、registry 来源。
- 是否单例、是否可添加。
- supportLevel: singleton / full / resource-bound / partial / add-only / documentation-only。
- 是否有节点目录说明、是否有语义配置器、是否有后端渐进式节点规格。
- 是否需要插件、知识库、数据库、智能体、MCP、图像资源。
- 可用的测试策略: local / resource / sub-canvas / not-executable。
- 当前缺口。

已落地起点:

- `frontend/packages/workflow/playground/src/services/workflow-agent-node-capabilities.ts`
- `workflow_canvas_node_capability_audit` 会随聊天上下文发送。
- `workflow_canvas_get_canvas_context` 会回传 `node_capabilities`。
- `workflow_canvas_get_node_capability_audit` 可被模型显式调用,用于确认哪些节点是 full/resource-bound/partial/add-only。
- `workflow_canvas_get_node_smoke_manifest` 可被模型显式调用,用于确认 41 个节点的 smoke/readiness、执行隔离方式、临时节点、资源 fixture 需求和必要工具链。
- `workflow_canvas_get_node_smoke_coverage` 可被模型显式调用,用于确认每类节点是否有 spec/assertions/expected bindable variables,并直接读取具体 assertions/expected bindable variables、是否能作为下游变量来源、是否需要资源 fixture 或仍是 partial/sub-canvas。
- `workflow_canvas_get_bindable_variables` 可被模型显式调用,用于配置 input/condition/merge_groups/End returns 前确认真实可绑定变量。

### 3. Command Layer

统一画布命令不绑定某个聊天 UI:

- `list_surfaces`
- `get_node_catalog`
- `get_node_capability_audit`
- `get_node_smoke_manifest`
- `get_node_spec`
- `get_canvas_context`
- `get_bindable_variables`
- `add_node`
- `delete_node`
- `connect`
- `delete_line`
- `configure_node`
- `set_node_params`
- `clear_canvas`
- `auto_layout`
- `validate`
- `test_run`
- `explain_failure`

读命令要求:

- `get_node_catalog`: 只返回节点用途、支持等级和是否需要资源，避免一开始塞完整长文。
- `get_node_spec(type)`: 返回该类型的完整配置说明、输入输出、常见错误、可替代节点。
- `get_canvas_context`: 返回节点、连线、inputs、outputs、returns、merge_groups、绑定诊断、校验错误。
- `get_bindable_variables`: 返回当前真实可绑定变量；可选 `target_node` 后续用于过滤作用域。
- `get_resources`: 返回插件/API、知识库、智能体、数据库、MCP、卡片等资源摘要。

写命令要求:

- 写命令必须可重放、可审计，保存原始 command 和 adapter 执行结果。
- `clear_canvas` 必须带 `reason`，且只在用户明确要求整体重做或上下文证明局部修复不可行时允许。
- `configure_node` 必须根据 node type 走对应 semantic config adapter。没有 semantic adapter 的节点不能宣称 full support。
- `test_run` 默认走后端稳定接口 `/api/workflow_api/agent_test_run`，浏览器内可选择 UI testRun，但结果结构必须标准化。
- MCP 写工具实时模式不直接保存数据库草稿,而是通过系统 relay 把命令发给当前打开的浏览器画布执行;外部 MCP client 等待浏览器返回真实执行结果。平台内 finmallclaw 聊天流仍通过 `{"status":"dispatched_to_canvas","op":"add_node","args":...}` ack 交给聊天面板执行,但二者共享同一套 command envelope 和语义配置器。
- MCP 只读工具当前通过 browser-live relay 请求打开的工作流页执行:poll 返回 `requests`,前端执行 `get_canvas_context/get_bindable_variables`,再把标准结果 POST 回系统后端。
- MCP `workflow.test_run` 是真正后端执行工具,返回 `workflow_test_result`,不返回 `dispatched_to_canvas`,前端桥不会打开试运行抽屉。

内部智能体继续通过 `workflow_canvas_*` 工具调用这些命令。外部 MCP 不是由 Codex/Claude Code 临时自带,而是由我们系统后端提供同源 MCP 服务;Codex、Claude Code、平台超级体只是客户端,transport 不同但能力、规格、命令语义一致。

### 4. Execution Adapter Layer

同一条命令可以有两个执行适配器:

- Browser adapter: 操作当前打开的画布，实时可见，适合内嵌聊天和人工协作。
- Backend draft adapter: 直接读取/修改草稿 JSON，适合外部 MCP、批量生成、离线校验和没有浏览器时的自动化。

短期优先级:

1. Browser adapter 继续承接实时可见体验。
2. Backend test_run 已有雏形，继续补 direct validate/test/explain。
3. Backend draft adapter 后续补齐，解决外部 MCP 没有浏览器时也能改图的问题。

适配器接口:

```ts
interface CanvasAutomationAdapter {
  getCapabilities(input: CanvasIdentity): Promise<NodeCapabilityAudit>;
  getNodeSpec(input: CanvasIdentity & { type: string }): Promise<NodeSpec>;
  getContext(input: CanvasIdentity): Promise<CanvasContext>;
  getBindableVariables(input: CanvasIdentity & { targetNode?: string }): Promise<BindableVariable[]>;
  applyCommands(input: CanvasCommandEnvelope): Promise<CanvasCommandResult>;
  validate(input: CanvasIdentity): Promise<CanvasValidationResult>;
  testRun(input: CanvasIdentity & { input?: Record<string, string> }): Promise<CanvasTestRunResult>;
}
```

Browser adapter 当前可复用:

- `WorkflowAgentCommandService.listNodeTypes`
- `getCanvasJSON/getCanvasSummary/getBindableVariablesSummary`
- `execCommand/runScript`
- `WorkflowRunService.testRun(..., { skipGlobalReload: true })`

Backend adapter 当前可复用:

- `crossworkflow.DefaultSVC().AsyncExecute/GetExecution`
- `/api/workflow_api/agent_test_run`
- `ValidateTree` 和 draft 读取路径
- `backend/cmd/mcptestserver` 的 streamable-http MCP server 模式
- `backend/application/workflow/canvasautomation` 作为系统级 MCP/catalog 能力包;`backend/cmd/workflowmcpserver` 只是独立启动包装,后续主服务路由也复用该包。

Backend adapter 缺口:

- 需要把前端 semantic config 到真实 canvas schema 的转换抽到共享包或后端重写一份等价转换。
- 需要 draft apply/save 的权限校验、版本冲突和回滚策略。
- 需要把前端 binding diagnostics 移植到后端，不能只依赖前端摘要。

### 5. Progressive Skill Layer

模型不能一次拿全量长提示词瞎猜，而是按步骤获取:

1. `list_surfaces`: 先确认 workflow/chatflow/settings/resource catalog 哪些面可读写,以及对应 adapter 和 gaps。
2. `get_node_catalog`: 只给节点用途和支持等级。
3. `get_node_spec(type)`: 用到某类节点前获取完整配置说明。
4. `get_canvas_context`: 获取当前节点、连线、outputs、可绑定变量、校验错误。
5. `get_bindable_variables(node_tag?)`: 针对某个目标节点获取可绑定变量候选。
6. `configure_node`: 只允许语义配置，不让模型直接猜内部表单路径。
7. `validate/test_run`: 用真实错误驱动局部修复。

关键规则:

- 不能把 add/connect ack 当成成功。
- 每组操作后必须校验绑定，从 Start 到 End 审计。
- 试运行失败要定位失败节点和错误类型，局部修复。
- 除非用户明确重做，不能默认 clear_canvas。
- type=13 输出节点只做消息展示，不能作为变量聚合/End returns 的变量源；需要下游消费时用 type=15 文本处理产出真实 output。

## 节点全量测试策略

### 节点验收等级

每个节点最终只能处在以下状态之一:

- `verified-full`: add/configure/connect/validate/test_run 都通过，且可绑定变量与 UI 摘要一致。
- `verified-resource-bound`: add/configure/connect/validate 通过，但运行依赖真实资源；缺资源时必须 skipped-with-reason。
- `verified-add-only`: 能添加、删除、布局，但没有可靠语义配置器；agent 不允许把它当完整执行节点。
- `verified-not-executable`: 注释、展示类或单例节点，只验证边界和禁用误用规则。
- `failing`: 有自动化脚本证明当前存在问题，报告必须包含失败节点、失败命令和建议修复。

每个节点的最小证据:

- 能通过节点目录发现。
- 能通过能力矩阵说明 supportLevel。
- 如果 canAdd=true，能在空白画布 add 后出现。
- 如果 supportLevel=full/resource-bound，必须有 semantic config 单测。
- 如果 runtimeSmoke=local，必须能构造最小可运行工作流并试运行。
- 如果 runtimeSmoke=resource，必须验证“缺资源时跳过并给原因”和“有资源 fixture 时能配置”两条路径。
- 如果 runtimeSmoke=sub-canvas，必须在循环/批处理子画布测试集中验证。
- Smoke 报告必须区分 `status` 和 `executionStatus`: `status` 只说明计划支持等级,`executionStatus` 才说明该节点是否已有真实执行证据。
- 没有 `executedCommands` 覆盖 requiredCommands 的节点必须进入 `evidenceGaps`,不能因为场景登记为 `verified-full` 就算跑通。

当前节点分组:

- `full/local 或可本地验证`: 5 代码、8 条件分支、13 输出/纯输出、15 文本处理、18 问答、30 输入、32 变量聚合、58 JSON序列化、2 End。
- `full/resource-bound`: 3 大模型、4 插件/API、6 知识库检索、99 卡片选择、100 智能体。
- `partial 优先补齐`: 45 HTTP、59 JSON解析、22 意图识别、21 循环、28 批处理、19 跳出循环、29 继续循环。
- `resource-bound 待资源适配`: 9 子工作流、12 SQL自定义、14 图像流、16 生成图片、17 图片引用、23 图片画布、26 长期记忆、27 知识库写入、34/35/36 触发器、42/43/44/46 数据库 CRUD、61 MCP。
- `add-only/非执行`: 11 变量、20 变量赋值、31 注释。

### 覆盖矩阵测试

目的: 防止“面板有节点，agent 不知道”。

已落地:

- 单测校验每个可见 registry type 都在能力矩阵登记。
- 单测输出当前 full/partial/add-only 缺口。
- Smoke report 已新增执行证据维度: `executionStatus/executedCommands/missingCommands/evidence/evidenceGaps`,避免把静态场景登记误判成真实跑过。
- Smoke plan 已新增逐节点执行计划维度: `mode/isolation/ready/steps/cleanupSteps/missingConcreteCommands`。资源缺失、子画布未支持、缺少具体 connect/configure 步骤会被显式列出,不会让模型盲猜。
- Smoke executor 已新增执行器适配层: Playwright/MCP 只需要实现 `executeCommand/captureEvidence`,执行失败或抛异常时也会跑 cleanup,再输出 report 可消费的 `WorkflowAgentNodeSmokeExecutionResult`。
- Smoke CommandService adapter 已新增浏览器内执行适配: 直接把 plan step 包成 `canvas_automation.v0` envelope 调 `WorkflowAgentCommandService.applyCommandEnvelope`,保证内嵌 finmallclaw、Browser bridge、节点巡检共享同一条画布执行路径。
- Smoke runner 已新增批量编排层: `runWorkflowAgentNodeSmokeSuite` 会生成 plans/readiness,顺序执行每个 plan,收集 results,并生成最终 smoke report。
- Smoke manifest 已新增统一巡检清单层: `buildWorkflowAgentNodeSmokeManifest` 将 41 个节点的 plan/readiness/skip reason/临时节点清理/是否需要资源 fixture 汇总成 JSON-friendly 结构,后续 MCP、Codex CLI、平台内置智能体和可视化巡检 UI 都应读取同一份 manifest,避免各入口维护不同节点清单。
- 本地可验证节点已补 concrete setup: 2 End、5 代码、8 条件分支、13 输出/纯输出、15 文本处理、18 问答、30 输入、32 变量聚合、58 JSON 序列化。场景不再只写说明: IF/JSON/VariableMerge/End 都会先创建或读取真实上游变量,调用 `get_bindable_variables`,再配置 condition/input/merge_groups/返回文本。
- VariableMerge smoke 明确使用 type=15 Text 作为真实 output 来源,禁止把 type=13 输出节点当成聚合变量。End smoke 明确只配置内置 end,通过返回文本 + `streaming_output=true` 绑定上游变量,不新增第二个 End。
- CommandService smoke adapter 默认拒绝执行 `temporary-workflow` 计划;调用方必须显式确认当前 canvas 是临时巡检画布,避免把 End 配置或临时节点误写到用户正在编辑的工作流。

下一步:

- 把实际运行时 `command.listNodeTypes()` 的结果和静态矩阵对比，发现新增节点自动报警。
- 将审计结果在 UI 中隐藏给用户、显式给 agent。

### 配置器单测

目的: 防止“说明写了，但 configure_node 实际没改对表单”。

每个 full 节点必须有:

- semantic config -> form params 的单测。
- 固定输入、变量输入、清空输入、outputs 声明、下游绑定案例。
- canvas summary 能正确读回 outputs/inputs/returns/merge_groups。

### 浏览器集成测试

目的: 验证真实 FlowGram 画布是否能动态执行。

每个节点至少测试:

- add_node 后节点出现。
- configure_node 后右侧表单/画布摘要反映配置。
- connect 后线出现。
- auto_layout 后节点不堆叠。
- validate 没有结构性错误。

测试运行方式:

- 本地优先: `make server` 指向远程数据库/资源，`make fe` 或 `API_PROXY_TARGET=http://10.10.10.226:8896` 启动前端热更新。
- 远程回归: 只在本地巡检通过后部署到 226，再跑少量冒烟。
- 浏览器必须用可视化模式，避免无头测试看不到布局/弹窗/右侧配置面板问题。

报告格式:

```json
{
  "node_type": "32",
  "node_name": "变量聚合",
  "status": "verified-full",
  "commands": ["add_node", "connect", "configure_node", "validate", "test_run"],
  "bindable_variables": ["text_success.output", "text_fail.output", "merge.output"],
  "diagnostics": [],
  "evidence": {
    "workflow_id": "generated",
    "screenshot": "reports/workflow-node-32.png",
    "test_run_status": "success"
  }
}
```

### 复杂工作流测试

优先构造 4 类复杂场景:

1. 转账业务多意图: 转账、查询转账记录、缺账号追问、缺卡号查最近收款人、确认后调用 mock API。
2. 分支汇合: LLM/IF/文本处理/变量聚合/End returns，覆盖 type=13 与 type=15 的边界。
3. 资源类流程: 插件/API、知识库、智能体、HTTP、MCP。
4. 子画布流程: 循环、批处理、break/continue。

每个复杂场景都要有:

- 构图脚本。
- 预期可绑定变量清单。
- 预期校验结果。
- 试运行输入和结果断言。
- 失败自动修复样例。

## 已验证证据

### 2026-06-24 Codex CLI MCP 验证

验证环境:

- 工作流页: `http://10.10.10.226:8896/work_flow?workflow_id=7654449287564099584&space_id=7652614054615187456`
- 系统 MCP endpoint: `/api/workflow_mcp/mcp`
- Codex MCP 配置名: `ynet_workflow`
- 本地开发代理: `scripts/workflow-mcp-cookie-proxy.mjs`,仅用于开发期把已有 session 注入到同源 MCP 请求;生产应替换为 PAT 或开发者 API token。

验证结果:

- `MCP_OK 41`: Codex CLI 能通过 `workflow.list_node_capabilities` 读取 41 个节点能力。
- `MCP_CONTEXT_OK`: Codex CLI 能通过 `workflow.get_canvas_context` 读取当前打开浏览器画布的真实上下文。
- `MCP_WRITE_OK codex_cli_probe_20260624011138`: Codex CLI 能调用 `workflow.add_node` 在当前打开画布实时添加节点,随后调用 `workflow.delete_node` 删除该探针节点,并通过 `workflow.get_canvas_context` 验证写入和清理完成。
- `MCP_NODE_SMOKE_TOOL_OK`: Codex CLI 能通过 `ynet_workflow_local` 调用 `workflow.run_node_smoke(node_type=32)`,独立 MCP 包装服务在无浏览器 bridge 时返回 `status=dispatched_to_canvas/op=run_node_smoke`。这证明工具已进入 Codex 可调用工具集;真实 `node_smoke_report` 仍需要打开工作流页并让 Browser bridge 执行。
- `MCP_NODE_SMOKE_ROUTE_OK`: 主后端路由级测试已覆盖 `POST /api/workflow_mcp/mcp tools/call workflow.run_node_smoke -> GET /api/workflow_mcp/browser_commands -> POST /api/workflow_mcp/browser_command_results -> MCP response`,并确认响应中带回 `node_smoke_report`。
- `FRONTEND_NODE_SMOKE_SUMMARY_OK`: 前端 Browser bridge 的 `run_node_smoke` 回传已新增 `node_smoke_summary`,包含 total/executed/failed/skipped/partial/pending/notRequired、failingNodes、skippedNodes、partialNodes、evidenceGaps 和 nextActions,让 finmallclaw/Codex/Claude Code 无需扫描完整报告即可继续局部修复。
- `FRONTEND_NODE_SMOKE_SUMMARY_REGRESSION_OK`: 前端 `workflow-agent-node-smoke-report` 与 `workflow-agent-node-smoke-browser-command` 回归已验证 Browser bridge 会把失败节点、跳过节点、partial 节点、evidence gaps 和 nextActions 汇总到 `node_smoke_summary`。
- `MCP_CODEX_EXEC_MANIFEST_OK`: `codex exec` 使用已配置的 `ynet_workflow_local` streamable HTTP MCP,实际调用 `workflow.node_smoke_manifest`,并按 schema 输出 `{"used_mcp_tool":true,"manifest_total":41,"variable_merge_expected":["merge.output"],"has_end_text_assertion":true,"has_end_streaming_assertion":true,"has_output_display_only_assertion":true}`。
- `MCP_NODE_SMOKE_COVERAGE_UNIT_OK`: 后端 MCP 已注册 `workflow.node_smoke_coverage`,单测验证 total_nodes/manifest_nodes 均为 41,无 missing manifest/spec,并能区分 executable、readonly、resource fixture、partial/sub-canvas 节点;type=32 有 spec/assertions/expected_bindable_variables,type=13 被标记为不可作为下游稳定变量来源。
- `MCP_CODEX_EXEC_NODE_SMOKE_COVERAGE_OK`: `codex exec` 实际调用 `ynet_workflow_local.workflow.node_smoke_coverage`,并按 schema 输出 `{"used_mcp_tool":true,"total_nodes":41,"manifest_nodes":41,"missing_manifest_empty":true,"missing_spec_empty":true,"type32_has_merge_output":true,"type32_can_be_downstream":true,"type13_cannot_be_downstream":true,"type13_mentions_display_only":true,"type45_is_partial":true}`。
- `MCP_CODEX_EXEC_NODE_SMOKE_COVERAGE_ASSERTIONS_OK`: `codex exec` 实际调用 `ynet_workflow_local.workflow.node_smoke_coverage`,并按 schema 输出 `{"used_mcp_tool":true,"total_nodes":41,"manifest_nodes":41,"merge_expected_bindable_variables":["merge.output"],"merge_assertions_include_bindable_guard":true,"output_assertions_include_display_only":true,"coverage_gaps_empty_array":true}`。
- `MCP_CODEX_EXEC_NODE_SMOKE_COVERAGE_COMMAND_PLAN_OK`: `codex exec` 实际调用 `ynet_workflow_local.workflow.node_smoke_coverage`,并按 schema 输出 `{"used_mcp_tool":true,"total_nodes":41,"type32_required_has_bindable":true,"type32_required_has_configure":true,"type32_planned_has_auto_layout":true,"type32_cleanup_has_delete_node":true,"type32_temp_tags":["smoke_32_variable_merge","smoke_32_text_primary","smoke_32_text_fallback"],"type32_isolation":"temporary-workflow"}`。
- `MCP_CODEX_EXEC_SPEC_OK`: `codex exec` 实际调用 `workflow.get_node_spec(type=5)`, `workflow.get_node_spec(type=32)` 和 `workflow.node_smoke_manifest`,并按 schema 输出 `{"used_mcp_tool":true,"code_spec_mentions_python_args":true,"code_spec_mentions_args_params":true,"merge_spec_mentions_bindable_variables":true,"manifest_total":41,"has_end_text_streaming":true}`。
- `MCP_CODEX_EXEC_SURFACES_OK`: `codex exec` 实际调用 `workflow.list_surfaces`,并按 schema 输出 `{"used_mcp_tool":true,"surface_total":5,"has_workflow_canvas":true,"has_chatflow_canvas":true,"chatflow_canvas_protocol":"canvas_automation.v0","has_role_settings_backend_api":true,"has_resource_catalog":true}`。
- `MCP_CODEX_EXEC_RESOURCE_CATALOG_OK`: `codex exec` 实际调用 `workflow.list_resource_catalog` 两次,一次全量、一次 `node_type=4` 过滤,并按 schema 输出 `{"used_mcp_tool":true,"catalog_total":12,"has_plugin_api":true,"has_knowledge_base":true,"has_mcp_server":true,"has_http_endpoint":true,"type4_filtered_all_cover_type4":true,"plugin_api_mentions_no_fabricated_ids":true}`。
- `MCP_CODEX_EXEC_OPERATION_GUIDE_OK`: `codex exec` 实际调用 `ynet_workflow_local.workflow.get_operation_guide(task_type=build_workflow)`,并按 schema 输出 `{"used_mcp_tool":true,"task_type":"build_workflow","has_surface_first":true,"has_resource_catalog":true,"has_bindable_before_configure":true,"has_no_clear_canvas_guard":true,"has_auto_layout_after_mutation":true,"has_test_run":true,"has_failure_repair_checklist":true}`。
- `FINMALLCLAW_OPERATION_GUIDE_UNIT_OK`: 平台内 `workflow_canvas_get_operation_guide` 已通过单测验证,返回内部 `workflow_canvas_*` 渐进式工具顺序、`binding_audit`、`failure_repair` 和不要默认清空画布等 guardrails。
- `FINMALLCLAW_RESOURCE_CATALOG_UNIT_OK`: 平台内 `workflow_canvas_get_resource_catalog` 已通过单测验证,全量返回 12 类资源族,`node_type=4` 过滤只返回 `plugin_api`,且提示词强制在渐进式设计阶段读取资源目录。
- `FINMALLCLAW_NODE_SMOKE_COVERAGE_UNIT_OK`: 平台内 `workflow_canvas_get_node_smoke_coverage` 已通过单测验证,total_nodes/manifest_nodes 均为 41,无 missing manifest/spec,type=32 直接返回变量聚合 assertion 和 `expected_bindable_variables:["merge.output"]`,且可作为下游来源,type=13 被标记为 display-only 且不可作为下游稳定来源,type=45 保留 HTTP partial gap。
- `MCP_SURFACE_DISCOVERY_UNIT_OK`: 后端单测已确认 `workflow.list_surfaces` 注册并返回 `workflow.canvas/chatflow.canvas/chatflow.role_settings/chatflow.conversation_templates`,其中 workflow/chatflow canvas 共用 `canvas_automation.v0`,ChatFlow 设置面指向现有后端 API。

本轮修复过的问题:

- `workflow.delete_node` 的前端协议映射缺失 `node_tag -> node` 参数,导致外部 MCP 删除节点请求无法真正落到 `WorkflowAgentCommandService.deleteNode`。
- 外部 MCP 写工具不能只返回 `dispatched_to_canvas` ack,否则客户端无法确认浏览器是否执行成功;现在实时模式会等待 Browser bridge 的真实执行结果。
- 后端重启后前端 sessionStorage cursor 可能大于后端最新 relay command ID,导致新命令被跳过或重复处理旧命令;relay 已处理 stale cursor。
- Browser bridge 执行需要响应的命令时,前端异常必须回传 `status=failed`,否则 MCP client 只能等超时。

本轮自动化检查:

- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation ./api/router/workflowmcp ./api/router -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation ./api/router/workflowmcp -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./application/workflow/canvasautomation ./api/router/workflowmcp -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation -run 'TestWorkflowMCP(ServerRegistersSystemTools|ListSurfacesAdvertisesWorkflowAndChatflow)' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation -run 'TestWorkflowMCP(ServerRegistersSystemTools|ListResourceCatalog|ListSurfacesAdvertisesWorkflowAndChatflow)' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation -run 'TestWorkflowMCP(ServerRegistersSystemTools|OperationGuideEnforcesProgressiveEditingProtocol|ListSurfacesAdvertisesWorkflowAndChatflow)' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation -run 'TestWorkflowMCP(OperationGuideEnforcesProgressiveEditingProtocol|ListSurfacesAdvertisesWorkflowAndChatflow|ServerRegistersSystemTools|NodeSmokeCoverageReportsEveryNodeAndBoundaries)' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation ./api/router/workflowmcp ./domain/agent/singleagent/internal/agentflow -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestWorkflowCanvasGetOperationGuideReturnsProgressiveProtocol|TestSuperAgentWorkflowCanvasPromptRequiresContextAndConfigureNode' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestWorkflowCanvasGetNodeSmokeCoverageReportsEveryNodeAndBoundaries|TestWorkflowCanvasGetOperationGuideReturnsProgressiveProtocol|TestWorkflowCanvasGetResourceCatalogReturnsResourceFamilies|TestSuperAgentWorkflowCanvasPromptRequiresContextAndConfigureNode' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./application/workflow/canvasautomation ./api/router/workflowmcp -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestWorkflowCanvasGetNodeSpec(CoversEverySmokeManifestNode|FallbackExplainsSupportBoundaries)' -count=1`
- `SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestWorkflowCanvasGetResourceCatalog|TestSuperAgentWorkflowCanvasPromptRequiresContextAndConfigureNode' -count=1`
- `pnpm vitest run src/services/__tests__/workflow-agent-browser-command-relay.test.ts src/services/__tests__/workflow-agent-command-protocol.test.ts src/services/__tests__/workflow-agent-command-readonly.test.ts`
- `pnpm vitest run src/services/__tests__/workflow-agent-node-smoke-report.test.ts src/services/__tests__/workflow-agent-node-smoke-browser-command.test.ts`
- `pnpm vitest run src/services/__tests__/workflow-agent-node-smoke-*.test.ts src/services/__tests__/workflow-agent-test-run-guard.test.ts`
- `pnpm tsc --noEmit -p tsconfig.json`
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o openynet main.go`
- `IS_OPEN_SOURCE=false npx rsbuild build`

仍不能宣称完成的范围:

- 41 个节点已有能力登记和部分场景定义,但还没有逐节点可视化浏览器巡检证据;不能宣称所有节点都能 add/configure/connect/validate/test_run。
- `workflow.test_run` 后端路径可用,但复杂失败定位和自动修复仍需要和节点级 diagnostics 继续打通。
- 无浏览器场景的 Backend draft adapter 仍未完成,外部 MCP 目前要实时改图仍依赖当前工作流页面打开并挂载 Browser bridge。

## 外部 MCP 接入方案

### MCP server 工具集

系统后端提供 `backend/application/workflow/canvasautomation` 作为统一 MCP 能力层;`backend/cmd/workflowmcpserver` 是同一能力的独立启动包装,便于本地/运维验证。工具集:

- `workflow.get_operation_guide`
- `workflow.list_node_capabilities`
- `workflow.list_surfaces`
- `workflow.list_resource_catalog`
- `workflow.node_smoke_manifest`
- `workflow.node_smoke_coverage`
- `workflow.run_node_smoke`
- `workflow.get_node_spec`
- `workflow.get_canvas_context`
- `workflow.get_bindable_variables`
- `workflow.add_node`
- `workflow.connect`
- `workflow.configure_node`
- `workflow.set_node_params`
- `workflow.delete_node`
- `workflow.delete_line`
- `workflow.clear_canvas`
- `workflow.auto_layout`
- `workflow.test_run`
- `workflow.explain_failure`

MCP 工具 schema 和内部工具保持一一对应:

- `workflow.get_operation_guide` = 系统渐进式操作规程入口,返回 surface/resource/node/context/bind/configure/layout/test/fix 的顺序、guardrails 和绑定/失败修复 checklist。
- `workflow.list_surfaces` = 系统 surface 能力发现入口,返回 workflow/chatflow/settings/resource catalog 的 adapter、工具、后端 API 和 gaps。
- `workflow.list_resource_catalog` = 内部 `workflow_canvas_get_resource_catalog`,系统资源目录契约入口,返回资源族、适用节点、已有后端 API、绑定规则、禁止编造 ID 和 live list 缺口。
- `workflow.list_node_capabilities` = 内部 `workflow_canvas_get_node_capability_audit`
- `workflow.node_smoke_manifest` = 内部 `workflow_canvas_get_node_smoke_manifest`
- `workflow.node_smoke_coverage` = 系统侧 41 节点覆盖报告,用于在读取具体 spec 前先判断节点能否执行 smoke、是否需要资源 fixture、是否 partial/sub-canvas、是否有 spec/assertions/bindable expectations。
- `workflow.run_node_smoke` = 外部 MCP browser-live 节点巡检触发器,由前端 Browser bridge 调用统一 smoke runner 并回传 `node_smoke_report`
- `workflow.get_node_spec` = 内部 `workflow_canvas_get_node_spec`
- `workflow.get_canvas_context` = 内部 `workflow_canvas_get_canvas_context`
- `workflow.get_bindable_variables` = 内部 `workflow_canvas_get_bindable_variables`
- `workflow.add_node/connect/configure_node/set_node_params/delete_node/delete_line/clear_canvas/auto_layout` = 内部 `workflow_canvas_*` 写命令语义;外部 MCP 实时模式返回浏览器真实执行结果,平台内聊天流返回前端可执行 ack
- `workflow.test_run` = `/api/workflow_api/agent_test_run`

MCP 服务由系统提供,不直接耦合某个 LLM。Codex、Claude Code、平台超级体、后续其它自动化入口只要会调 MCP/HTTP 工具，就可以使用同一套协议。

认证方式:

- 本地开发: token + space_id/workflow_id。
- 平台内: 当前 session/user。
- 外部 Codex/Claude Code: personal access token 或开发者 API token。

### 内部/外部同源

内部聊天:

`agent tool call -> workflow_canvas_* -> SSE ack -> Browser adapter -> canvas`

外部 MCP 实时模式:

`MCP client -> 系统 MCP 服务 workflow.add_node/run_node_smoke 等工具(带 workflow_id/space_id) -> backend relay -> Browser bridge 轮询 /api/workflow_mcp/browser_commands -> 当前画布实时变化或节点 smoke runner 执行 -> Browser bridge 回传执行结果 -> MCP tool result`

外部 MCP 离线/批量模式:

`MCP client -> 系统 MCP 服务 workflow.* tools -> Backend draft adapter -> draft JSON -> validate/test_run -> 前端刷新可见`

二者共享:

- 节点能力矩阵。
- 节点规格。
- 命令 schema。
- 校验/试运行结果格式。
- 错误解释规则。

推荐接入形态:

1. 平台内嵌聊天默认走 Browser adapter，用户实时看到节点出现、连线、布局、调试。
2. Codex/Claude Code 本机开发时优先走 MCP server + Browser bridge，适合可视化协作和调试前端 UI 行为。
3. 批量生成、夜间巡检、CI 回归走 MCP server + Backend draft adapter，不依赖浏览器。
4. 其它页面功能后续也按同一套模式开放: capability -> context -> command -> validate/test -> diagnostics。

实时模式的必要条件:

- 当前编辑器页面必须打开并挂载 Browser bridge。
- MCP 写工具外部调用必须携带 `workflow_id` 和 `space_id`;否则后端无法知道哪一个打开的画布要执行命令。
- Browser bridge 已轮询 `/api/workflow_mcp/browser_commands?workflow_id=...&space_id=...&after_id=...`,把返回的 `commands` 交给 `WorkflowAgentCommandService.applyCommandEnvelope`。
- 平台内聊天流仍可直接从 tool_response content 读取原始 JSON ack,至少包含 `status/op/args`。
- 外部 MCP 写命令如果带 `requires_response`,Browser bridge 必须把成功或失败结果 POST 回后端;MCP client 以这个结果作为工具调用结果。
- 前端桥只把平台内聊天流的 `status=dispatched_to_canvas` 当作本地写命令执行;`workflow.test_run` 的后端结果只展示给 agent,不触发前端试运行抽屉。
- 一组写操作后自动执行 `auto_layout`,或由模型显式调用 `workflow.auto_layout`,避免节点堆叠。
- `workflow.run_node_smoke` 必须在浏览器实时模式下执行;它不是后端直接改草稿,而是把节点 smoke plan 交给当前画布里的统一前端 runner,因此可复用真实节点配置器、变量绑定和清理逻辑。

## 分阶段落地

### Phase 1: 覆盖矩阵和缺口可见化

- 已新增节点能力矩阵和单测。
- 聊天上下文携带节点能力审计。
- 后端 `get_canvas_context` 回传能力审计。

完成标准:

- 所有可见节点都有能力登记。
- 新增节点未登记时测试失败。
- agent 能看到当前哪些节点不是 full support。

补充完成标准:

- 能生成节点能力 JSON 报告，明确 41 个可见节点的当前状态。
- `binding diagnostics` 进入 `get_canvas_context`，模型看到非 none 时必须局部修复。
- 每个 `full` 节点都有 semantic config 单测。

### Phase 2: 每个节点补语义配置器

按优先级补:

1. HTTP、JSON解析、子工作流、MCP。
2. 数据库 CRUD/SQL。
3. 循环、批处理、break/continue。
4. 变量、变量赋值、LTM、触发器。
5. 图像类节点。

完成标准:

- supportLevel 从 partial/add-only 升级到 full/resource-bound 时必须带单测。
- `get_node_spec`、前端能力矩阵、语义配置器同步更新。

推荐第一批:

- HTTP(type=45): method/url/headers/query/body/auth/outputs。
- JSON解析(type=59): input JSON、schema/paths、outputs。
- MCP(type=61): server/tool/resource discovery、tool args、outputs。
- 子工作流(type=9): workflow_id、子流程 input schema、outputs。

这四类覆盖后，复杂外部接口和组合流程的表达能力会明显提升。

### Phase 3: 浏览器节点巡检

用本地 `make server` + `make fe` 或远程 226 API proxy 跑巡检:

- 自动创建测试 workflow。
- 按节点类型逐个 add/configure/connect/validate。
- 生成 HTML/JSON 报告。

完成标准:

- 每个节点都有一条 smoke 结果。
- resource-bound 节点如果缺资源,报告为 skipped-with-reason,不能默默 pass。

建议实现路径:

- 新增 `frontend/packages/workflow/playground/src/services/workflow-agent-node-smoke-scenarios.ts`，按节点类型定义最小场景。
- 新增 `workflow-agent-node-smoke-plan.ts`，把场景转换成可执行计划: 本地节点走临时 workflow,资源节点必须有 fixture,子画布节点单独跳过并列缺口,执行后统一产出 evidence。
- 新增 `workflow-agent-node-smoke-executor.ts`，把计划交给 MCP/Playwright adapter 执行,统一处理失败、异常、清理和证据回传。
- 新增 `workflow-agent-node-smoke-command-service-adapter.ts`，在浏览器内把巡检计划直接桥接到 `WorkflowAgentCommandService.applyCommandEnvelope`。
- 新增 `workflow-agent-node-smoke-runner.ts`，统一编排 41 个节点计划的执行、skip/not-ready 结果和最终 report 输出。
- 新增 `workflow-agent-node-smoke-manifest.ts`，导出执行前 readiness manifest: 每个节点包含 `mode/isolation/ready/plannedCommands/cleanupCommands/temporaryNodeTags/skipReason`,作为外部 MCP 和内部 finmallclaw 共同读取的巡检入口。
- 新增 `workflow-agent-node-smoke-browser-command.ts`，让 Browser bridge 能消费 `run_node_smoke` 命令,按需过滤单个 `node_type`,调用统一 runner,并把 `node_smoke_report` 放回 MCP 命令结果。
- 已补 `workflow-agent-node-smoke-scenarios` 的本地节点 concrete setup: Code、Output、Text、Input、IF、Question、VariableMerge、JsonStringify、End。默认 suite 的 `notReady` 对本地 verified-full 节点应为空;resource-bound/sub-canvas/unsupported 节点继续显式 skip 或列缺口。
- `runWorkflowAgentNodeSmokeSuiteWithCommandService` 如需执行本地节点计划,必须传 `allowTemporaryWorkflowPlans=true`,且 canvas 必须是自动创建的临时巡检 workflow。
- 新增 Playwright 可视化巡检脚本，逐节点创建临时 workflow、执行命令、截图、保存 JSON 报告。
- 巡检脚本只调用统一 command protocol，不直接点 UI 内部实现，确保和 agent/MCP 同源。

### Phase 4: MCP server

暴露同一套协议给外部工具。

完成标准:

- Codex/Claude Code 能通过 MCP 获取节点目录、读取画布、应用命令、试运行。
- 平台内 finmallclaw 与外部 MCP 的命令 schema 一致。

当前已落地:

- `backend/application/workflow/canvasautomation`
- `backend/cmd/workflowmcpserver` 仅作为系统能力的独立启动器。
- 主后端路由 `/api/workflow_mcp/mcp`。
- 主后端浏览器命令轮询路由 `/api/workflow_mcp/browser_commands`。
- 前端 Browser bridge 轮询服务 `workflow-agent-browser-command-relay` 和工作流页挂载逻辑。
- 只读工具: capabilities/spec/context/bindable_variables。
- 巡检清单工具: 外部 `workflow.node_smoke_manifest` / 内部 `workflow_canvas_get_node_smoke_manifest`,用于统一获取 41 节点 smoke readiness。
- 巡检覆盖工具: 外部 `workflow.node_smoke_coverage` / 内部 `workflow_canvas_get_node_smoke_coverage`,用于先获取 41 节点全量 coverage: manifest/spec 是否缺失、执行/只读/资源 fixture/partial-sub-canvas 分类、关键绑定断言和 expected bindable variables 的具体内容、required/planned/cleanup command plan、temporary node tags、isolation、节点是否能作为下游变量来源。
- 内外部巡检清单均已补充 `assertions` 与 `expected_bindable_variables`,明确变量聚合、输出/纯输出、文本处理、结束节点返回文本/流式输出等绑定契约,减少模型盲猜。
- 巡检执行工具: 外部 `workflow.run_node_smoke`,通过 Browser bridge 触发前端 smoke runner,支持 `node_type` 单节点过滤、`allow_temporary_workflow` 和 `include_skipped`,并回传完整 `node_smoke_report` 与精简 `node_smoke_summary`。
- 写工具: add_node/connect/configure_node/set_node_params/delete_node/delete_line/clear_canvas/auto_layout,支持 Browser bridge 实时执行并回传真实执行结果。
- Surface 发现工具: `workflow.list_surfaces`,返回 `workflow.canvas/chatflow.canvas/chatflow.role_settings/chatflow.conversation_templates/space.resource_catalog`,用于让所有客户端先确认可操作面、adapter、协议、现有后端 API 和 gaps。
- 资源目录工具: `workflow.list_resource_catalog`,返回 12 类资源族 contract,覆盖所有 resource-bound/resource-smoke 节点,支持 `node_type`/`family` 过滤;当前仍不返回 live resource instances。
- 渐进式操作规程工具: `workflow.get_operation_guide`,返回 build/fix 流程的必备工具、严格顺序、guardrails、绑定审计 checklist 和失败局部修复 checklist,用于替代超长提示词记忆。
- 平台内渐进式操作规程工具: `workflow_canvas_get_operation_guide`,返回同一套内部 `workflow_canvas_*` 编辑顺序和 guardrails,用于约束 finmallclaw 右侧聊天不要跳过资源/变量/规格读取,也不要遇错就默认清空画布。
- 平台内资源目录工具: `workflow_canvas_get_resource_catalog`,与外部 MCP 资源目录保持同一资源族和同一禁编造 ID 约束。
- 测试工具: test_run/explain_failure,其中 test_run 走后端执行,不打开前端试运行抽屉。
- Codex CLI 已完成 capabilities/context/write/delete 的 browser-live 验证。
- Codex CLI 已配置 `ynet_workflow_local` 指向本地系统 MCP `http://127.0.0.1:9901/mcp`,并验证可调用 `workflow.node_smoke_manifest`:返回 `total=41`,包含 type=32 变量聚合,且 type=32 要求 `workflow.get_bindable_variables`,同时存在 resource fixture skip 节点。
- Codex CLI `codex exec` 已真实触发 `ynet_workflow_local.workflow.node_smoke_manifest` MCP tool call,并验证 manifest 中 type=32 暴露 `expected_bindable_variables=["merge.output"]`,type=2 暴露 `End 可以返回文本`/`streaming_output=true`,type=13 暴露 `display-only`。
- Codex CLI `codex exec` 已真实触发 `workflow.node_smoke_coverage`,验证 total_nodes/manifest_nodes 均为 41,missing_manifest/spec 均为 0,type=32 有 expected bindable,type=13 不能作为下游变量来源,HTTP 保留 partial gap,MCP 节点需要 resource fixture。
- Codex CLI `codex exec` 已真实触发 `workflow.get_node_spec(type=5/type=32)`,验证 Code 节点规格包含 Python `async def main(args: Args)`、`args.params` 和禁止 `args.strip/get`,变量聚合规格包含 `workflow.get_bindable_variables` 和真实可绑定变量要求。
- Codex CLI `codex exec` 已真实触发 `workflow.list_surfaces`,验证当前 surface 总数为 5,普通工作流画布和 ChatFlow 画布均存在,且 ChatFlow 画布复用 `canvas_automation.v0`。
- Codex CLI `codex exec` 已真实触发 `workflow.list_resource_catalog` 全量和 `node_type=4` 过滤调用,验证 catalog 总数为 12,包含 plugin_api/knowledge_base/mcp_server/http_endpoint,且插件/API 目录明确 `do_not_fabricate_ids=true`。
- Codex CLI `codex exec` 已真实触发 `workflow.get_operation_guide(task_type=build_workflow)`,验证规程包含先 surface/resource/node/context,再读 bindable variables,再 configure,并包含不默认 clear_canvas、每组变更后 auto_layout、test_run 和 failure_repair checklist。
- 平台内 `workflow_canvas_get_node_spec` 已有回归测试覆盖 41 个 `workflow_canvas_get_node_smoke_manifest` 节点,确认所有节点都有具体规格或安全边界,不会再返回“未找到该节点的详细规格”。
- 平台内 `workflow_canvas_get_operation_guide` 已有回归测试覆盖内部渐进式规程,并验证 `SuperAgentWorkflowCanvasPrompt` 要求先调用该工具。
- 平台内 `workflow_canvas_get_resource_catalog` 已有回归测试覆盖全量资源目录和 `node_type=4` 过滤,并验证 `SuperAgentWorkflowCanvasPrompt` 要求先读取资源目录。
- Codex CLI 已验证可调用 `workflow.run_node_smoke(node_type=32, allow_temporary_workflow=true, include_skipped=false)`,本地 MCP 包装服务返回 `status=dispatched_to_canvas/op=run_node_smoke`;浏览器实际执行报告留给 Browser bridge 在线场景验证。
- 后端 MCP 已注册 `workflow.run_node_smoke`;单元测试覆盖它会投递 `op=run_node_smoke` 的 browser-live 命令、等待前端结果,并返回 `node_smoke_report`。
- 主后端路由测试已验证 browser-live 双向 relay 能把 `node_smoke_report` 从 `/api/workflow_mcp/browser_command_results` 带回原始 MCP tool response。
- 前端已验证 `pnpm vitest run src/services/__tests__/workflow-agent-node-smoke-*.test.ts`、`pnpm tsc --noEmit -p tsconfig.json` 和 `IS_OPEN_SOURCE=false npx rsbuild build`,确认 `node_smoke_summary` 接入 Browser bridge 后单测、类型检查和产物构建均通过。

### 2026-06-24 部署状态

- 本地构建已通过:后端 Linux 二进制 `backend/openynet`,前端 `frontend/apps/coze-studio/dist`。
- 已生成待上传包:
  - `/tmp/coze-studio-dist-20260624160652.tgz`,sha256 `c73e32e5c6ad8400934e67cd4ffa462bd6239d7146d279eec98451c75f6dea82`
  - `/tmp/coze-openynet-20260624161650.tgz`,sha256 `9f08722f7aab880db4b9282dc579171dc54b7da917d20680e17a5fa272196c58`
- 2026-06-24 17:02 通过 `hz-on` 启动杭州 OpenVPN 后,226 SSH 与 HTTP 恢复正常。
- 后端包 `/tmp/coze-openynet-20260624161650.tgz` 已上传 226 并替换 `coze-super:/app/openynet`,保留回滚备份 `/app/openynet.bak-unified-mcp-20260624090547`,已重启 `coze-super`。
- 部署后健康检查通过:`curl -I http://10.10.10.226:8896/` 返回 200,`/api/workflow_mcp/surfaces` 返回鉴权错误而非 404/崩溃,确认新后端进程正常接管路由。
- 已清理 `coze-super:/app` 旧 openynet 备份,磁盘可用空间恢复到 13.0G。

仍需补齐:

- MCP 工具认证、space/workflow 权限策略和外部 token 方案。
- 继续扩展 226/本地可视浏览器验证: configure_node、test_run、复杂失败修复、打开/关闭 AI 面板后的 bridge 在线状态。
- `workflow.get_canvas_context/get_bindable_variables` 的后端 draft 读取实现;当前实时模式仍依赖浏览器上下文。
- Backend draft adapter 写入能力,让无浏览器客户端也能直接修改草稿。

### Phase 5: 后端草稿适配器

让外部无浏览器也能直接改 draft JSON。

完成标准:

- `workflow.apply_commands` 能直接修改草稿。
- 修改后前端刷新可见。
- backend validate/test_run 可返回节点级错误和输出。

必须防住:

- 草稿版本冲突: apply 前后都带 commit_id 或 revision。
- 权限边界: space/workflow/user 统一鉴权。
- 回滚: 每次批量 apply 保存前置 snapshot。
- 并发: 同一 workflow 写锁或乐观锁。

## 下一步执行建议

优先级从“把 MCP 做出来”转为“把 MCP 接入和节点巡检打牢”:

1. 在已打通的 Codex CLI MCP 实时链路上继续扩展验证: `workflow.configure_node -> 表单变化`、`workflow.test_run -> 聊天不重置`、复杂失败后 `explain_failure -> 局部修复`。
2. 用 `workflow.run_node_smoke` 在打开的浏览器画布上跑单节点和全量本地节点 smoke report,优先覆盖变量聚合、输出/结束返回文本、文本处理无输入、IF/多分支绑定。
3. 用可视化 Playwright 按 manifest 创建临时 workflow 跑本地前端,生成节点 smoke report。
4. 把 HTTP/JSON解析/MCP/子工作流补成 semantic config，并把能力矩阵升级。
5. 补 `workflow.get_canvas_context/get_bindable_variables` 的后端 draft 实现,让外部 MCP 不依赖浏览器也能读图。
6. 最后做 Backend draft adapter 写入能力，让外部无浏览器也能画图。

## 风险与处理

- 表单结构复杂: 所有 agent-facing 配置必须走语义配置器，不让模型猜内部 path。
- 资源依赖多: resource-bound 节点必须先发现真实资源，没有资源就 skipped-with-reason。
- 子画布复杂: 循环/批处理单独作为二级协议处理，不能和普通节点混在一起粗暴配置。
- 外部写权限: MCP 写操作必须有 workflow_id、space_id、用户身份和权限校验。
- JSON 一次性生成过大: 复杂流程使用增量命令 + 校验闭环，不推荐一次性输出整图 JSON；整图 JSON 只用于导入/导出、diff、回滚和批量迁移。
