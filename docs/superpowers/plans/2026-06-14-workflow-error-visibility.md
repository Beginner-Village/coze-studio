# 工作流/智能体执行错误详情可见性增强 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让工作流试跑/节点调试与智能体调试时，失败节点（尤其模型节点）能就地看到具体错误（HTTP 状态码 + 厂商原始报错），而生产/对外维持脱敏。

**Architecture:** 在模型工厂层用装饰器把 eino 模型调用的错误富化为携带 HTTP 状态码 + 厂商报错的 `ModelCallError`（工作流和智能体共用同一条模型调用路径，DRY）；在工作流 `event_handle` 落 `error_info`/`fail_reason` 处按 `ExecuteMode` 门控（debug/node_debug 写完整根因，release 脱敏）；智能体调试链路同理；前端在现有失败展示组件上显眼呈现。

**Tech Stack:** Go（eino / eino-ext chatmodel、errorx、gorm）、前端 React（工作流试跑面板 / 智能体调试窗口）。

**关键参考（已核查）：**
- 模型工厂：`backend/infra/impl/chatmodel/default_factory.go`（`NewFactory` 的 `CreateChatModel` 返回 `chatmodel.ToolCallingChatModel`）
- chatmodel 契约：`backend/infra/contract/chatmodel/chat_model.go`（`ToolCallingChatModel = model.ToolCallingChatModel`，eino 接口含 `Generate`/`Stream`/`WithTools`）
- 工作流错误落库：`backend/domain/workflow/internal/execute/event_handle.go`（节点错误块 ~778-825：`errorInfo = wfe.Msg()[:min(1000,...)]`；line 584 `nodeExec.ErrorInfo = ptr.Of(wfe.Msg())`；line 262 `FailReason`）
- 执行模式：`event.ExeCfg.Mode`（`workflowModel.ExecuteModeDebug` / `ExecuteModeNodeDebug` / `ExecuteModeRelease`，event_handle.go:69/936 已在用）
- 错误根因工具：`backend/domain/workflow/entity/vo/node.go`（`UnwrapRootErr`、`WrapError`、`wfErr`）
- 错误码消息模板：`backend/types/errno/workflow.go`（`ErrWorkflowExecuteFail = "Workflow execution failure: {cause}"`）
- 智能体模型节点：`backend/domain/agent/singleagent/internal/agentflow/node_chat_model.go`

**全部 backend 命令在 `cd /Users/luzhipeng/projects/ynet/coze-studio/backend`，测试 `go test ./...`。新建 .go 必须带 Apache license header。当前分支 `feat/falcon-sandbox-skill-2026-06`（除非另行决定；提交只 `git add` 本功能文件，不要 `-A`）。每个 commit 结尾加 `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`。**

---

## 文件结构

**新建（backend）：**
- `backend/infra/contract/chatmodel/error.go` — `ModelCallError` 类型（携带 `HTTPStatus`/`ProviderMessage`/`Raw`），可被 domain 层类型断言
- `backend/infra/impl/chatmodel/error_enrich.go` — `extractModelHTTPError(err)` 归一 + `wrapModelError(err)` 构造 `ModelCallError`
- `backend/infra/impl/chatmodel/error_enrich_test.go` — 归一函数单测
- `backend/infra/impl/chatmodel/error_decorator.go` — 装饰器：包装 `ToolCallingChatModel` 的 `Generate`/`Stream`/`WithTools`，出错时富化
- `backend/infra/impl/chatmodel/error_decorator_test.go` — 装饰器单测（fake 内层模型）
- `backend/domain/workflow/entity/vo/detailed_cause.go` — `DetailedCause(err) string`：错误链里优先取 `ModelCallError` 的富化消息，否则回退 `UnwrapRootErr`
- `backend/domain/workflow/entity/vo/detailed_cause_test.go`

**修改（backend）：**
- `backend/infra/impl/chatmodel/default_factory.go` — `CreateChatModel` 返回前用装饰器包一层
- `backend/domain/workflow/internal/execute/event_handle.go` — 节点错误/工作流失败落 `error_info`/`fail_reason` 处按 `ExeCfg.Mode` 门控
- 智能体调试错误透传（文件在 Task 5 诊断后定位，预期 `domain/agent/singleagent/internal/agentflow/node_chat_model.go` 或其上层）

**修改/新建（frontend）：** Task 6 定位现有失败节点/调试错误展示组件后增强（工作流试跑面板先做，智能体调试紧随）。

**诊断产出：** `docs/superpowers/notes/2026-06-14-error-visibility-diagnosis.md`

---

## Task 1: 诊断定位（复现 + 抓取，决定后续范围）

**目的：** 用真实模型 400/500 复现，确认详情在"模型客户端 → error_info → 前端 payload"哪一层丢，据此确认/收敛 Task 3-6。**这是只读诊断 + 临时打点，不改产品逻辑。**

**Files:**
- Create: `docs/superpowers/notes/2026-06-14-error-visibility-diagnosis.md`

- [ ] **Step 1: 起后端连开发库（复用操作日志 e2e 的方式）**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
go build -o /Users/luzhipeng/projects/ynet/coze-studio/bin/openynet main.go
cd /Users/luzhipeng/projects/ynet/coze-studio/bin
set -a && source /Users/luzhipeng/projects/ynet/coze-studio/docker/.env.debug && set +a
export SESSION_HMAC_SECRET="oplog-e2e-test-secret-key-32bytes-long"
export LISTEN_ADDR=":8899"; export SERVER_HOST="http://localhost:8899"
nohup ./openynet -start > /tmp/errviz_server.log 2>&1 &
```
注意：8888 可能已被他人实例占用，**务必用 8899**，不要动 8888。等 `nc -z localhost 8899` 通。

- [ ] **Step 2: 构造模型错误并触发工作流试跑**

在开发库里准备/挑选一个含 LLM 节点的工作流。用注册账号（参考操作日志 e2e：`POST /api/passport/web/email/register/v2/` 拿 session cookie）。制造两类错误：
- **HTTP 400**：把该模型配置改成非法参数（如超大 max_tokens / 不支持的字段），或输入超长触发 context length。
- **HTTP 401/500**：把模型的 API Key 临时改坏（DB `model_entity` 或对应配置表），触发鉴权失败。
触发工作流试跑接口（`grep -rn "TestRun\|test_run\|workflow_api.*run" api/router/` 找真实路由），并同样触发一次**智能体调试**对话。

- [ ] **Step 3: 抓取三处真实值**

```bash
# (a) 后端日志里模型客户端返回的原始 err 文本
grep -iE "returns err|cause:|status code|model|invoke" /tmp/errviz_server.log | tail -40
# (b) DB 里实际存储的 error_info / fail_reason
set -a && source /Users/luzhipeng/projects/ynet/coze-studio/docker/.env.debug && set +a
M="/opt/homebrew/opt/mysql-client/bin/mysql"
$M -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" --vertical \
  -e "SELECT error_info,error_level FROM node_execution WHERE status=4 ORDER BY id DESC LIMIT 3;" 2>&1 | grep -v "Using a password"
# (c) 试跑/调试接口返回前端的 payload —— 用 curl 抓接口响应里的错误字段
```

- [ ] **Step 4: 写诊断结论**

在 `docs/superpowers/notes/2026-06-14-error-visibility-diagnosis.md` 记录：
1. 模型客户端 err 是否含 HTTP 状态码 + 厂商 body？（决定是否需要 Task 2/3 的富化）
2. `error_info` 里实际存了什么？（笼统 `Msg()` 还是已含根因？）
3. 前端 payload 里错误字段是什么？（决定 Task 6 重心）
4. **决策**：
   - 若模型 err 已含 status+body 且已进 error_info → Task 2/3 跳过或缩减，重心转 Task 4（门控）+ Task 6（前端展示）。
   - 若模型 err 丢了 status+body → 按 Task 2/3 富化（预期路径）。
   - 若 error_info 已有详情但前端没展示 → Task 4 仍需门控，Task 6 为重点。

- [ ] **Step 5: 停掉测试后端、还原被改坏的配置**

```bash
pkill -f "openynet -start" 2>/dev/null   # 仅杀自己 8899 的；确认不误杀 8888
```
把 Step 2 改坏的 API Key / 模型参数改回原值（如果改了 DB）。

- [ ] **Step 6: Commit 诊断结论**

```bash
git add docs/superpowers/notes/2026-06-14-error-visibility-diagnosis.md
git commit -m "docs(workflow): diagnose model error visibility loss points"
```

> **控制者注意**：Task 1 完成后先读诊断结论，再决定 Task 2-6 哪些执行/调整。以下任务按"模型 err 丢了 status+body"的预期路径写；若诊断推翻该前提，按 Step 4 的决策调整。

---

## Task 2: ModelCallError 类型 + 归一函数

**Files:**
- Create: `backend/infra/contract/chatmodel/error.go`
- Create: `backend/infra/impl/chatmodel/error_enrich.go`
- Test: `backend/infra/impl/chatmodel/error_enrich_test.go`

- [ ] **Step 1: 写 ModelCallError 类型（契约层，可被 domain 断言）**

`backend/infra/contract/chatmodel/error.go`（加 license header）：

```go
package chatmodel

import "fmt"

// ModelCallError 富化后的模型调用错误：携带 HTTP 状态码与厂商原始报错。
// 作为错误链上的一环（Unwrap 返回底层 Raw），其 Error() 文本已包含状态码与厂商消息，
// 供调试态就地展示。生产态由上层决定是否脱敏，不直接回显本类型内容。
type ModelCallError struct {
	HTTPStatus      int    // 0 表示未解析到
	ProviderMessage string // 厂商返回的 message/body（可能为空）
	Raw             error  // 原始底层错误
}

func (e *ModelCallError) Error() string {
	switch {
	case e.HTTPStatus > 0 && e.ProviderMessage != "":
		return fmt.Sprintf("model call failed [HTTP %d]: %s", e.HTTPStatus, e.ProviderMessage)
	case e.HTTPStatus > 0:
		return fmt.Sprintf("model call failed [HTTP %d]: %v", e.HTTPStatus, e.Raw)
	case e.ProviderMessage != "":
		return fmt.Sprintf("model call failed: %s", e.ProviderMessage)
	default:
		return fmt.Sprintf("model call failed: %v", e.Raw)
	}
}

func (e *ModelCallError) Unwrap() error { return e.Raw }

// AsModelCallError 在错误链中查找 ModelCallError。
func AsModelCallError(err error) (*ModelCallError, bool) {
	for err != nil {
		if mce, ok := err.(*ModelCallError); ok {
			return mce, true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil, false
		}
		err = u.Unwrap()
	}
	return nil, false
}
```

- [ ] **Step 2: 写归一函数的失败测试**

`backend/infra/impl/chatmodel/error_enrich_test.go`（加 license header）：

```go
package chatmodel

import (
	"errors"
	"testing"
)

func TestExtractModelHTTPError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsgSub string
	}{
		{"openai-style", errors.New("error, status code: 400, message: invalid 'max_tokens'"), 400, "invalid 'max_tokens'"},
		{"status-only", errors.New("request failed with status 429 Too Many Requests"), 429, "Too Many Requests"},
		{"no-status", errors.New("connection refused"), 0, "connection refused"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, msg := extractModelHTTPError(c.err)
			if status != c.wantStatus {
				t.Fatalf("status: want %d got %d", c.wantStatus, status)
			}
			if c.wantMsgSub != "" && !contains(msg, c.wantMsgSub) {
				t.Fatalf("msg %q does not contain %q", msg, c.wantMsgSub)
			}
		})
	}
}

func contains(s, sub string) bool { return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 3: 写归一 + 包装实现**

`backend/infra/impl/chatmodel/error_enrich.go`（加 license header）：

```go
package chatmodel

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

var statusCodeRe = regexp.MustCompile(`(?i)status(?:\s+code)?[:\s]+(\d{3})`)

// extractModelHTTPError 从各家 eino-ext provider 的错误里尽力解析 HTTP 状态码与厂商消息。
// 解析不到状态码时返回 0；消息至少回退为 err.Error()。
func extractModelHTTPError(err error) (status int, providerMsg string) {
	if err == nil {
		return 0, ""
	}
	s := err.Error()
	if m := statusCodeRe.FindStringSubmatch(s); len(m) == 2 {
		status, _ = strconv.Atoi(m[1])
	}
	// 优先取 "message: ..." 之后的部分作为厂商消息
	if idx := strings.Index(strings.ToLower(s), "message:"); idx >= 0 {
		providerMsg = strings.TrimSpace(s[idx+len("message:"):])
	} else {
		providerMsg = s
	}
	return status, providerMsg
}

// wrapModelError 把原始模型错误富化为 ModelCallError；err 为 nil 时返回 nil。
func wrapModelError(err error) error {
	if err == nil {
		return nil
	}
	// 已经富化过则不重复包装
	if _, ok := chatmodel.AsModelCallError(err); ok {
		return err
	}
	status, msg := extractModelHTTPError(err)
	return &chatmodel.ModelCallError{HTTPStatus: status, ProviderMessage: msg, Raw: err}
}
```

- [ ] **Step 4: 跑测试**

Run: `cd backend && go test ./infra/impl/chatmodel/ -run TestExtractModelHTTPError -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/infra/contract/chatmodel/error.go backend/infra/impl/chatmodel/error_enrich.go backend/infra/impl/chatmodel/error_enrich_test.go
git commit -m "feat(chatmodel): add ModelCallError and HTTP error extraction"
```

---

## Task 3: 模型调用错误装饰器 + 工厂接入

**Files:**
- Create: `backend/infra/impl/chatmodel/error_decorator.go`
- Test: `backend/infra/impl/chatmodel/error_decorator_test.go`
- Modify: `backend/infra/impl/chatmodel/default_factory.go`

eino 接口（`chatmodel.ToolCallingChatModel = model.ToolCallingChatModel`）含：
`Generate(ctx, []*schema.Message, ...Option) (*schema.Message, error)`、
`Stream(ctx, []*schema.Message, ...Option) (*schema.StreamReader[*schema.Message], error)`、
`WithTools([]*schema.ToolInfo) (ToolCallingChatModel, error)`。

- [ ] **Step 1: 写装饰器**

`backend/infra/impl/chatmodel/error_decorator.go`（加 license header）：

```go
package chatmodel

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

// errEnrichModel 包装内层模型，把 Generate/Stream 的错误富化为 ModelCallError。
type errEnrichModel struct {
	inner chatmodel.ToolCallingChatModel
}

func withErrorEnrichment(m chatmodel.ToolCallingChatModel) chatmodel.ToolCallingChatModel {
	if m == nil {
		return nil
	}
	return &errEnrichModel{inner: m}
}

func (m *errEnrichModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	out, err := m.inner.Generate(ctx, input, opts...)
	if err != nil {
		return nil, wrapModelError(err)
	}
	return out, nil
}

func (m *errEnrichModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	out, err := m.inner.Stream(ctx, input, opts...)
	if err != nil {
		return nil, wrapModelError(err)
	}
	return out, nil
}

func (m *errEnrichModel) WithTools(tools []*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &errEnrichModel{inner: inner}, nil
}
```

> 注意：eino 的 `Stream` 错误也可能在**读取流时**才返回（首包之后）。本装饰器只富化"建流即返回"的错误。流内错误的富化（若诊断 Task 1 显示模型 400/500 是建流即报，则本装饰器足够；若是流中报，需在流读取处理处富化——按 Task 1 结论决定是否追加）。先实现建流错误富化，覆盖最常见的 400/401 鉴权/参数错误（这些在建流前就返回）。

- [ ] **Step 2: 写装饰器测试**

`backend/infra/impl/chatmodel/error_decorator_test.go`（加 license header）：

```go
package chatmodel

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

type fakeModel struct{ err error }

func (f *fakeModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, f.err
}
func (f *fakeModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, f.err
}
func (f *fakeModel) WithTools(tools []*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	return f, nil
}

func TestDecoratorEnrichesGenerateError(t *testing.T) {
	raw := errors.New("error, status code: 400, message: invalid request")
	m := withErrorEnrichment(&fakeModel{err: raw})
	_, err := m.Generate(context.Background(), nil)
	mce, ok := chatmodel.AsModelCallError(err)
	if !ok {
		t.Fatalf("expected ModelCallError, got %T: %v", err, err)
	}
	if mce.HTTPStatus != 400 {
		t.Fatalf("want status 400, got %d", mce.HTTPStatus)
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./infra/impl/chatmodel/ -run TestDecorator -v`
Expected: PASS

- [ ] **Step 4: 工厂接入装饰器**

修改 `backend/infra/impl/chatmodel/default_factory.go` 的 `CreateChatModel`（先 grep 找该方法实现位置；它调用 `protocol2Builder[protocol](ctx, config)` 拿到模型）。在返回前包一层：

```go
// CreateChatModel 内，拿到 builder 产物 m, err 后：
if err != nil {
	return nil, err
}
return withErrorEnrichment(m), nil
```

确认 `CreateChatModel` 的真实结构（可能在 default_factory.go 或 singleton.go），把返回点替换为包装后的模型。**不要**改 builder 们本身。

- [ ] **Step 5: 编译 + 全包测试**

Run:
```bash
cd backend && go build ./infra/impl/chatmodel/... && go test ./infra/impl/chatmodel/... -v
```
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/infra/impl/chatmodel/error_decorator.go backend/infra/impl/chatmodel/error_decorator_test.go backend/infra/impl/chatmodel/default_factory.go
git commit -m "feat(chatmodel): enrich model call errors via factory decorator"
```

---

## Task 4: 根因提取 + event_handle 按 ExecuteMode 门控（工作流）

**Files:**
- Create: `backend/domain/workflow/entity/vo/detailed_cause.go`
- Test: `backend/domain/workflow/entity/vo/detailed_cause_test.go`
- Modify: `backend/domain/workflow/internal/execute/event_handle.go`

- [ ] **Step 1: 写 DetailedCause 提取器**

`backend/domain/workflow/entity/vo/detailed_cause.go`（加 license header）：

```go
package vo

import "github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"

// DetailedCause 返回用于调试展示的详细根因文本：
// 错误链中若存在 ModelCallError，优先用其富化文本（含 HTTP 状态码 + 厂商报错）；
// 否则回退到最深层根因。
func DetailedCause(err error) string {
	if err == nil {
		return ""
	}
	if mce, ok := chatmodel.AsModelCallError(err); ok {
		return mce.Error()
	}
	return UnwrapRootErr(err).Error()
}
```

- [ ] **Step 2: 写测试**

`backend/domain/workflow/entity/vo/detailed_cause_test.go`（加 license header）：

```go
package vo

import (
	"errors"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

func TestDetailedCausePrefersModelError(t *testing.T) {
	mce := &chatmodel.ModelCallError{HTTPStatus: 400, ProviderMessage: "invalid max_tokens", Raw: errors.New("raw")}
	got := DetailedCause(mce)
	if got != "model call failed [HTTP 400]: invalid max_tokens" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestDetailedCauseFallsBackToRoot(t *testing.T) {
	err := errors.New("plain root error")
	if DetailedCause(err) != "plain root error" {
		t.Fatalf("unexpected: %q", DetailedCause(err))
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./domain/workflow/entity/vo/ -run TestDetailedCause -v`
Expected: PASS

- [ ] **Step 4: event_handle 门控（节点错误块）**

修改 `backend/domain/workflow/internal/execute/event_handle.go`。先读 ~778-826 节点错误块与 ~245-265 工作流失败块，确认 `exeCfg`（或 `event.ExeCfg`）在作用域内。

节点错误块当前（~800）：
```go
errorInfo = wfe.Msg()[:min(1000, len(wfe.Msg()))]
```
改为按模式选择详细 or 脱敏：
```go
if exeCfg.Mode == workflowModel.ExecuteModeDebug || exeCfg.Mode == workflowModel.ExecuteModeNodeDebug {
	detail := vo.DetailedCause(event.Err)
	errorInfo = detail[:min(1000, len(detail))]
} else {
	errorInfo = wfe.Msg()[:min(1000, len(wfe.Msg()))]
}
```
对 ~245-265 的 `FailReason` 块（`errMsg := wfe.Msg()[:min(1000,...)]`）做同样的模式门控，debug/node_debug 用 `vo.DetailedCause(event.Err)`，release 用 `wfe.Msg()`。
line 584 `nodeExec.ErrorInfo = ptr.Of(wfe.Msg())` 若在另一分支，也同样门控。
确认 `exeCfg` 变量来源：若该函数签名是 `exeCfg workflowModel.ExecuteConfig`（event_handle.go:901 有此签名）直接用；若用 `event.ExeCfg` 则用之。import 已有 `workflowModel` 与 `vo`（确认；缺则补 `vo "github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"`）。

- [ ] **Step 5: 编译 + 相关测试**

Run:
```bash
cd backend && go build ./domain/workflow/... && go test ./domain/workflow/entity/vo/... -v
```
Expected: 编译通过；vo 测试 PASS

- [ ] **Step 6: Commit**

```bash
git add backend/domain/workflow/entity/vo/detailed_cause.go backend/domain/workflow/entity/vo/detailed_cause_test.go backend/domain/workflow/internal/execute/event_handle.go
git commit -m "feat(workflow): surface detailed error cause in debug mode, sanitize in release"
```

---

## Task 5: 智能体调试错误透传

**Files:**
- Modify: 智能体模型节点错误处理（预期 `backend/domain/agent/singleagent/internal/agentflow/node_chat_model.go`，实现时定位）

> 前提：Task 3 的装饰器已让智能体走的同一个模型工厂产出富化错误（智能体也经 `CreateChatModel`）。本任务只需确保智能体**调试态**把该详细错误透传到调试响应，而非笼统化。

- [ ] **Step 1: 定位智能体调试错误处理点**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
grep -rn "Mode\|debug\|Debug\|isDebug\|err\b\|Error" domain/agent/singleagent/internal/agentflow/node_chat_model.go | grep -iE "err|debug" | head
grep -rln "debug\|Debug\|isDebug\|ExecuteMode" domain/agent/singleagent --include="*.go" | head
```
确认：① 智能体模型调用错误如何返回到调试响应；② 是否有"调试态"标志（类似 workflow 的 Mode）。

- [ ] **Step 2: 透传详细错误（调试态）**

在智能体调试态错误返回处，用 `chatmodel.AsModelCallError(err)` 提取富化文本（含 HTTP 状态 + 厂商报错）填入返回给前端的错误字段；非调试态维持现有脱敏。具体落点按 Step 1 定位结果（若智能体调试本就直接把 err.Error() 回传，则装饰器已自动带出详细信息，本任务可能仅需确认 + 加测试）。

> 若 Step 1 发现智能体调试已能透传 `err.Error()`（装饰器富化后即含详情），则本任务退化为"验证 + 加一条断言测试"，在报告说明。

- [ ] **Step 3: 编译**

Run: `cd backend && go build ./domain/agent/... && go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/domain/agent/singleagent/internal/agentflow/
git commit -m "feat(agent): surface detailed model error in debug mode"
```

---

## Task 6: 前端就地展示详细错误

**Files:**
- 工作流试跑/节点调试失败展示组件（实现时定位）
- 智能体调试错误展示组件（实现时定位）

- [ ] **Step 1: 定位现有失败错误展示**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
grep -rln "error_info\|errorInfo\|error_level\|errorLevel\|FailReason\|fail_reason" packages apps --include="*.tsx" --include="*.ts" | grep -v node_modules | head
```
找到工作流试跑面板里渲染节点 `errorInfo` 的组件，以及智能体调试错误展示处。打开看现有展示方式（是否截断/折叠/只显示首行）。

- [ ] **Step 2: 增强工作流试跑面板展示（先做）**

在失败节点展示组件里，把 `errorInfo`（现在调试态已含 HTTP 状态 + 厂商报错）**完整、可读地**展示：
- 显眼标题（如"节点执行失败"）+ 完整 `errorInfo` 文本（支持换行/可展开，不要截断到一行）。
- 若文本含 `[HTTP <code>]` 可选高亮状态码。
- 跟随现有组件库（`@coze-arch/coze-design`）与样式风格，不新造页面。

- [ ] **Step 3: 增强智能体调试错误展示（紧随）**

在智能体调试窗口，模型/工具调用失败时把详细错误展示在调试侧栏或消息区（而非笼统提示）。复用 Step 2 的展示思路。

- [ ] **Step 4: 构建验证**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
node_modules/.bin/eslint <你改动的文件...>   # 0 error
# 如能构建对应包则 rush build -t <pkg>，否则人工核对 import/类型真实存在
```

- [ ] **Step 5: Commit**

```bash
git add frontend/<你改动的文件...>
git commit -m "feat(workflow): display detailed node error in debug panels"
```

---

## Task 7: 端到端手动验证

**Files:** 无（验证）

- [ ] **Step 1: 起后端（同 Task 1 Step 1，端口 8899）**

- [ ] **Step 2: 调试态验证（详情可见）**

触发一个会模型 400 的工作流试跑 + 智能体调试：
- 确认前端面板显示具体错误：含 `[HTTP 400]` + 厂商报错（如参数错误详情），不再是笼统"模型报错"。
- 用 401（坏 key）再验证一次，确认能区分鉴权失败。

- [ ] **Step 3: 生产态验证（脱敏）**

把同一工作流**发布**后用 release 路径（或 OpenAPI）触发同样的模型错误，确认返回的是脱敏的笼统错误，**不含**厂商原始 body / `provider_error`。

- [ ] **Step 4: 还原 + 停服**

还原被改坏的模型配置；`pkill -f "openynet -start"`（仅自己的 8899）。

- [ ] **Step 5: 记录验证结论**

把 debug/release 两路的实际返回贴进诊断 notes 文件末尾作为验证证据；如有收尾修改一并提交。

---

## Self-Review 检查结果

- **Spec 覆盖**：诊断先行(Task1) / 模型错误富化含 HTTP 状态+厂商报错(Task2,3) / 根因透传(Task4 DetailedCause) / 按 ExecuteMode 门控 debug vs release(Task4,5) / 工作流试跑先做+智能体调试紧随(Task6) / 复用 error_info 不动表(Task4 写入 error_info，未加字段) / 调试态可见生产脱敏(Task4,5,7) / 测试与 e2e(Task2-4 单测 + Task7 手动) — 全部有对应任务。
- **占位符**：Task1 是诊断任务（合理的调查步骤，非空泛 TODO）；Task5/Task6 的"实现时定位"均给了 grep 命令与预期落点，并标注了"若诊断显示已透传则退化为验证"的明确分支——非空白占位。无 "TBD"。
- **类型一致性**：`ModelCallError{HTTPStatus,ProviderMessage,Raw}`、`AsModelCallError`、`extractModelHTTPError`、`wrapModelError`、`withErrorEnrichment`、`vo.DetailedCause` 在各任务间签名一致；event_handle 用 `exeCfg.Mode`/`event.Err` 与现有代码一致。
- **诊断依赖说明**：Task1 结论可能收敛 Task2/3（若详情已透传则缩减），plan 已在 Task1 Step4 与各任务注明分支，控制者据诊断调整。
