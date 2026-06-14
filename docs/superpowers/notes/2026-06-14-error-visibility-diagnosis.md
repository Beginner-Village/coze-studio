# 错误可见性诊断结论（Task 1）

- 日期：2026-06-14
- 方法：静态源码分析（eino-ext 客户端错误类型 + 后端错误链 + 前端展示组件）。结论已足够指导实现；活体复现放到 Task 7 端到端验证一并确认。

## 核查的三层

### 1. 模型客户端返回的错误是否含 HTTP 状态码 + 厂商消息？
- **openai 协议**（`eino-ext/components/model/openai@v0.1.13`）：`Generate`/`Stream` 出错返回 `convOrigAPIError(err)`，转换为 `types.go` 的 `APIError{Code, Message, Type, HTTPStatus, HTTPStatusCode}`，其 `Error()` = `"error, status code: 400, status: ..., message: ..."`。**详情完整**。该 `APIError` 是**叶子错误（无 Unwrap）**。
- **ark / qwen / deepseek 协议**：未发现 `convOrigAPIError` / `APIError` / `HTTPStatusCode` 等富化逻辑——这些协议的错误**可能较笼统**（取决于各自底层 SDK）。
- 结论：**富化只在 openai 协议保证**；其它协议不一致。OpenAI 兼容网关若返回非标准错误体，go-openai 可能也解析不出 `APIError`，退化为笼统。

### 2. error_info 实际存了什么？
- `event_handle.go` 节点错误：`wfe = vo.WrapError(errno.ErrWorkflowExecuteFail, event.Err, errorx.KV("cause", vo.UnwrapRootErr(event.Err).Error()))`；`error_info = wfe.Msg()` = 模板 `"Workflow execution failure: {cause}"`。
- 对 openai 协议：`UnwrapRootErr` 会停在叶子 `APIError`，`{cause}` = 完整富文本 → **error_info 理论上已含 400 + 厂商消息**。
- 对其它协议/兼容网关：`{cause}` 取到的根因可能笼统 → error_info 笼统。
- 截断到 1000 字符，足够。

### 3. 前端是否展示 error_info？
- 工作流试跑面板**已渲染** `error_info`：`packages/workflow/playground/src/components/test-run/execute-result/execute-result-side-sheet/components/error-item.tsx`（解构 `{ errorInfo, errorLevel }` 展示）；`hooks/use-node-error-list.ts` 收集 `errorInfo`（多条 `join(';')`）。
- 结论：**前端不是主战场**——它已经在显示 error_info 全文。用户看到"笼统"≈ error_info 本身笼统。

## 决策（收敛后续任务）

| 任务 | 调整 | 理由 |
|---|---|---|
| Task 2/3 装饰器 | **保留，作为主修** | 在模型调用源头统一把任意 provider 的错误归一为 `ModelCallError`（带 HTTP 状态+厂商消息），保证详情成型并进入错误链，覆盖 openai 兼容网关/ark/qwen 等富化缺失的情况。比依赖各协议自有转换更可靠。 |
| Task 4 DetailedCause + 门控 | **保留** | `DetailedCause` 优先取 `ModelCallError` 富文本写入 error_info（debug）；release 用 `wfe.Msg()` 脱敏。注意：既然详情会流入 error_info，**release 脱敏从"nice-to-have"升级为安全必需**。 |
| Task 5 智能体调试 | **保留** | 智能体走同一模型工厂，装饰器自动受益；只需确认调试态把详细错误透传到调试响应。 |
| Task 6 前端 | **降级** | 前端已渲染 errorInfo。改为"验证 + 微调"：确保不被截断成一行、多行/长文本可读、必要时高亮 `[HTTP 4xx]`。不是主改。 |
| Task 7 e2e | **保留** | 活体复现确认：debug 态 error_info 含 400/厂商消息且前端可见；release 态脱敏。 |

## 待 Task 7 活体确认的点
- 真实模型 400/500 时，eino graph 是否用 `%w` 保留 Unwrap 链（使 `AsModelCallError` / `UnwrapRootErr` 能取到装饰器注入的 `ModelCallError`）。现有 `UnwrapRootErr` 机制能工作，间接表明链路保留 `%w`；Task 7 直接验证。
