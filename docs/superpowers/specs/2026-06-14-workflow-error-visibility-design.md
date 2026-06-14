# 工作流/智能体执行错误详情可见性增强 设计

- 日期：2026-06-14
- 状态：待评审
- 模块：backend（工作流执行错误链路 / 模型客户端 / 智能体 agentflow）+ frontend（试跑/调试错误展示）

## 1. 背景与目标

在前端调试工作流（试跑、节点调试）和调试智能体时，节点执行失败只显示**笼统错误**（如"工作流执行失败"/"模型报错"），看不到具体原因。**尤其是模型（LLM）错误**：无法判断是 HTTP 400 还是 500，也看不到模型厂商返回的真实报错（如 400 的参数错误详情、限流、上下文超长、鉴权失败等）。开发/测试时极难定位，常常要去服务器翻后端日志。

### 目标 / 成功标准
- 在**调试/搭建态**（工作流试跑 `ExecuteModeDebug`、节点调试 `ExecuteModeNodeDebug`、智能体调试）下，失败节点能就地看到**具体错误**：
  - 模型错误：HTTP 状态码（400/500/429/401…）+ 厂商原始报错（message/type/code 或原始响应体）。
  - 其它节点错误：底层根因（参数错误、空指针、下游 API 报错等），而非笼统码消息。
- **生产/对外**（`ExecuteModeRelease`、OpenAPI）维持现有脱敏的笼统错误，不回显厂商原始内容。
- 不重复造轮子：与现有 observability/trace 的 `node.error_message` span 属性对齐。

### 非目标（YAGNI）
- 不改造整体错误码体系、不动 OpenAPI 对外错误契约。
- 不为生产态做错误详情存储/查询后台（详情仅调试态就地展示；trace 链路已有的不动）。
- 不默认新增数据库字段——优先复用现有 `node_execution.error_info`；仅当"诊断"阶段证明必须结构化承载时才加字段。

## 2. 现状（已核查的错误流转链路）

```
模型客户端 eino-ext(openai/ark/deepseek/qwen)  返回 err(通常含 HTTP status+厂商 body)
  → LLM 节点 (domain/workflow/internal/nodes/llm/llm.go) 透传 err
  → 执行回调 callback.go 发 NodeError 事件 (Event.Err = 原始 err)
  → event_handle.go 处理:
       wfe = vo.WrapError(errno.ErrWorkflowExecuteFail, event.Err,
                          errorx.KV("cause", vo.UnwrapRootErr(event.Err).Error()))
       errorInfo = wfe.Msg()[:min(1000,...)]      // 写入 node_execution.error_info
       FailReason = wfe.Msg()                       // 写入 workflow_execution.fail_reason
  → application/workflow/workflow.go 把 ErrorInfo/ErrorLevel 返回前端
  → 前端试跑/调试面板展示
```

已核查的关键事实：
- 错误码 `ErrWorkflowExecuteFail` 的消息模板是 `"Workflow execution failure: {cause}"`，包装时带了 `cause = UnwrapRootErr(err).Error()`。**所以 error_info 理论上应含根因**。
- `vo.wfErr.Error()` 保留 `cause`（`"<msg>, cause: <cause>"`），`Unwrap()` 返回 cause；`UnwrapRootErr` 取错误链最深层。
- 已存在执行模式枚举 `vo.ExecuteMode`：`debug` / `node_debug` / `release`，随 `ExecuteConfig` 传递——天然的门控开关。
- `node_execution` 表已有 `error_info` / `error_level` 字段。

**推断的真因（待诊断确认）**：用户仍只看到笼统错误，最可能是
1. 模型客户端返回的 err 根因本身不含 HTTP 状态码/厂商 body（被 eino 或工厂层包装吞掉），导致 `{cause}` 带出来的也是笼统的；或
2. `UnwrapRootErr` 取到的最深层错误恰好丢了 HTTP 上下文；或
3. 前端没有显眼展示 `error_info`（只展示了顶层 workflow 错误）。
精确丢失层必须用真实模型 400/500 复现确定。

## 3. 方案总览（方向 A：诊断先行 → 端到端透传 → 调试态门控 → 前端展示）

### 3.1 阶段 0：复现定位（实现的第一步）
用可运行的后端（参考操作日志 e2e 的启动方式：构建 `bin/openynet`，连 220 开发库，设 `SESSION_HMAC_SECRET`，跑在非 8888 端口），构造模型错误并抓取三处真实数据：
- 构造 **HTTP 400**（坏参数/超长输入）与 **500/401/429**（坏 API key 等）各一次。
- 跑一个含 LLM 节点的工作流 **试跑**（`TestRun`），以及一次**智能体调试**对话触发模型错误。
- 抓取并记录：
  1. 模型客户端返回 err 的类型与 `.Error()` 文本（是否含 status+body）——必要时在 llm 节点临时打点或读后端日志。
  2. `node_execution.error_info` / `workflow_execution.fail_reason` 实际存储值。
  3. 试跑/调试接口返回给前端的 payload 中错误字段实际值。
- **产出**：一份"丢失层"结论，直接决定 3.2 的改动点（是改模型客户端包装，还是改门控/透传，还是只改前端展示）。

> 该阶段是只读诊断 + 临时打点，不改产品逻辑。结论写入实现计划，后续阶段按结论收敛。

### 3.2 后端：错误详情透传
按阶段 0 结论，做其中必要的子项：
- **(a) 模型客户端错误富化**（若根因丢了 status/body）：在 `infra/impl/chatmodel`（default_factory 构造的模型）调用出错处，把 HTTP 状态码与厂商响应体结构化进错误。优先用 `errorx` 的 KV 承载：`errorx.KV("http_status", "400")`、`errorx.KV("provider_error", <原始 body 或 message>)`，并保证该错误在 Unwrap 链上能被 `UnwrapRootErr` 或专用提取函数取到。
  - eino-ext 各 provider（openai/ark/deepseek/qwen）错误类型不一，统一用一个小工具 `extractModelHTTPError(err) (status int, providerMsg string, ok bool)` 做归一（类型断言 + 字符串兜底解析）。
- **(b) 根因透传确认**：确保 `event_handle.go` 里 `cause` 取的是富化后的根因（含 status+providerMsg），`error_info` 能带出。
- **(c) 结构化承载（按需，YAGNI）**：默认复用 `error_info`，在调试态写入形如
  `"模型调用失败 [HTTP 400]: <厂商原始报错>"` 的可读字符串。
  仅当前端需要分字段渲染（状态码 badge / 可展开原始 body）且字符串不够用时，才新增 `node_execution.error_detail`（JSON：`{http_status, provider_code, provider_message, raw}`，DDL 进 `schema.sql` add-only）。此决定在阶段 0 后敲定。

### 3.3 调试态门控
- 在 `event_handle.go` 构造 `errorInfo` / `FailReason` 处，按当前执行的 `ExecuteConfig.Mode` 分支：
  - `debug` / `node_debug` → 写入/返回**完整根因**（含模型 status + 厂商报错）。
  - `release` → 维持现有脱敏 `wfe.Msg()`。
  - 需确认 `event_handle` 上下文能拿到 Mode（`event.Context` / `ExecuteConfig`），取不到则从 workflow 执行实体透传。
- **智能体调试链路**：`domain/agent/singleagent/internal/agentflow` 中模型/工具节点错误同理，按调试态把详情透传到调试响应。智能体调试态的判定方式在实现时对照 singleagent 的调试入口确认。
- 安全：release 态绝不回显 `provider_error` / 原始 body；调试态仅限有空间权限的搭建者可见（调试入口本身已要求登录+空间权限）。

### 3.4 前端：就地显眼展示
- **工作流试跑/节点调试面板**（先做）：失败节点把后端返回的详细 `error_info`（HTTP 状态 + 厂商报错）显眼展示；若有结构化 `error_detail`，渲染状态码 + 可展开原始 body。
- **智能体调试窗口**（紧随）：模型/工具调用失败时，在调试侧栏或消息区展示具体错误而非笼统提示。
- 定位现有展示组件后在其上增强，不新造页面；文案/i18n 跟随现有风格。

### 3.5 测试
- 复现脚本化：注入式触发模型 400/500（坏 key / 坏参数），验证 **debug 态**接口返回含 status+厂商报错、**release 态**返回脱敏笼统错误。
- 后端单测：`extractModelHTTPError` 对各 provider 错误类型/文本的归一；门控分支（debug vs release）的 error_info 内容差异。
- 手动 e2e：前端试跑一个会触发模型 400 的工作流，确认面板显示具体错误；智能体调试同理；用 release 发布后确认对外脱敏。

## 4. 影响面与风险

- **改动集中在错误构造/展示路径**，不改正常执行逻辑，对成功路径零影响。
- **provider 错误类型差异**：eino-ext 各家错误结构不同，归一函数需类型断言 + 字符串兜底；兜底解析失败时退回原始 `err.Error()`（仍比现状强）。
- **门控正确性**是安全关键：必须确保 release 态不泄露。通过单测覆盖 debug/release 两路 + 手动发布验证。
- **阶段 0 可能改变后续范围**：若诊断发现详情其实已透传、只是前端没展示，则 3.2 大幅缩减、重心转 3.4。spec 以诊断结论为准收敛。

## 5. 待实现时确认的开放点（不阻塞设计）
- `event_handle` / 智能体调试链路能否就地拿到 `ExecuteMode`（取不到则需从执行实体透传）。
- 是否需要新增 `error_detail` 字段（阶段 0 后定）。
- 前端现有失败节点展示组件的位置与增强方式（实现时定位）。
