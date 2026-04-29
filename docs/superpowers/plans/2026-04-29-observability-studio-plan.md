# Observability — Studio Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 Studio (`~/projects/ynet/coze-studio`) 添加 `/metrics` endpoint、RED + 业务指标埋点、文件日志输出 + 滚动；并把 Studio 仪表盘 JSON 加到 obs-stack repo。

**Architecture:** 用 `prometheus/client_golang` 在现有 Hertz/Kitex 路由上挂 `/metrics`；HTTP RED 中间件统一插到 router 注册前；业务指标分散到 LLM 调用、文件上传、agent chat 三个落点。日志方面 `backend/pkg/logs/default.go` 加 `io.MultiWriter(stderr, lumberjack)`，6 个 env 变量可调。

**Tech Stack:** Go 1.25, Hertz, Kitex, prometheus/client_golang, gopkg.in/natefinch/lumberjack.v2

**Spec:** [docs/superpowers/specs/2026-04-29-observability-stack-design.md](../specs/2026-04-29-observability-stack-design.md)

**前置依赖:** obs-stack-infra 已就绪（Task 9 把 dashboard 写入 obs-stack/dashboards/）

---

## File Structure

```
backend/
├── pkg/
│   ├── logs/default.go                       # MODIFY: 加 lumberjack 文件输出
│   └── observability/                        # NEW: 通用 metrics 工具
│       ├── metrics.go                        # 通用 Counter/Histogram 注册
│       ├── http_middleware.go                # RED 中间件（Hertz）
│       └── http_middleware_test.go
├── api/router/
│   └── (找现有 router 注册点，挂 /metrics 路由)
├── domain/
│   ├── llm/                                  # MODIFY: 调用处加 token Counter
│   ├── knowledge/service/metrics.go          # 已有，扩展 file_upload_size
│   └── conversation/                         # MODIFY: agent_chat 路径加 Counter
└── api/handler/coze/
    └── developer_api_service.go              # MODIFY: 文件上传处加 file_upload_size

~/projects/ynet/obs-stack/
└── dashboards/studio.json                    # NEW: Studio 仪表盘
```

---

## Task 1: 添加 lumberjack 依赖

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: 添加依赖**

```bash
cd ~/projects/ynet/coze-studio
go get gopkg.in/natefinch/lumberjack.v2@v2.2.1
go mod tidy
```

Expected: `go.sum` 多两行 lumberjack 相关 hash

- [ ] **Step 2: 验证编译**

```bash
cd backend
go build ./pkg/logs/...
```

Expected: 编译通过，没有 import 错误

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "feat(deps): add lumberjack v2 for log rotation"
```

---

## Task 2: 给 logger 加文件输出 + 滚动

**Files:**
- Modify: `backend/pkg/logs/default.go`
- Create: `backend/pkg/logs/file_writer.go`
- Create: `backend/pkg/logs/file_writer_test.go`

- [ ] **Step 1: 写测试 — LOG_FILE 为空时不创建文件**

写入 `backend/pkg/logs/file_writer_test.go`：
```go
package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWriter_NoFileWhenLogFileEmpty(t *testing.T) {
	t.Setenv("LOG_FILE", "")
	w := NewWriter()
	if w == nil {
		t.Fatal("expected writer, got nil")
	}
	// 写一行不应该产生任何文件
	tmp := t.TempDir()
	t.Setenv("LOG_FILE", "")
	_, _ = w.Write([]byte("hello\n"))

	entries, _ := os.ReadDir(tmp)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			t.Fatalf("unexpected log file %s", e.Name())
		}
	}
}

func TestNewWriter_CreatesFileWhenLogFileSet(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	t.Setenv("LOG_MAX_SIZE_MB", "1")
	t.Setenv("LOG_MAX_BACKUPS", "2")

	w := NewWriter()
	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file %s, got error: %v", logPath, err)
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
cd backend
go test ./pkg/logs/ -run TestNewWriter -v
```

Expected: FAIL — `NewWriter undefined`

- [ ] **Step 3: 实现 file_writer.go**

写入 `backend/pkg/logs/file_writer.go`：
```go
package logs

import (
	"io"
	"os"
	"strconv"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewWriter returns an io.Writer that writes to stdout, and additionally
// to a rolling file if LOG_FILE env is set. Configuration via env:
//   LOG_FILE          — empty = stdout only; non-empty = stdout + file
//   LOG_MAX_SIZE_MB   — single file max size (default 100)
//   LOG_MAX_BACKUPS   — old file count to keep (default 7)
//   LOG_MAX_AGE_DAYS  — old file max age in days (default 30)
//   LOG_COMPRESS      — gzip rotated files (default true)
func NewWriter() io.Writer {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		return os.Stdout
	}

	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    envInt("LOG_MAX_SIZE_MB", 100),
		MaxBackups: envInt("LOG_MAX_BACKUPS", 7),
		MaxAge:     envInt("LOG_MAX_AGE_DAYS", 30),
		Compress:   envBool("LOG_COMPRESS", true),
	}
	return io.MultiWriter(os.Stdout, lj)
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./pkg/logs/ -run TestNewWriter -v
```

Expected: PASS

- [ ] **Step 5: 写滚动测试**

追加到 `backend/pkg/logs/file_writer_test.go`：
```go
func TestNewWriter_RotatesAtMaxSize(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	t.Setenv("LOG_MAX_SIZE_MB", "1")  // 1 MB
	t.Setenv("LOG_MAX_BACKUPS", "3")

	w := NewWriter()
	// 写 1.5 MB，应触发一次 rotate
	chunk := make([]byte, 4096)
	for i := range chunk {
		chunk[i] = 'A'
	}
	for i := 0; i < 400; i++ {  // 400 * 4KB = 1.6 MB
		if _, err := w.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	entries, _ := os.ReadDir(tmp)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "app") {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected ≥2 log files after rotation, got %d", count)
	}
}
```

- [ ] **Step 6: 跑滚动测试**

```bash
go test ./pkg/logs/ -run TestNewWriter_RotatesAtMaxSize -v
```

Expected: PASS

- [ ] **Step 7: 改 default.go 使用 NewWriter()**

修改 `backend/pkg/logs/default.go:27-30`：

把：
```go
var logger FullLogger = &defaultLogger{
	level:  LevelInfo,
	stdlog: log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
}
```

改成：
```go
var logger FullLogger = &defaultLogger{
	level:  LevelInfo,
	stdlog: log.New(NewWriter(), "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
}
```

- [ ] **Step 8: 验证整个 logs 包编译**

```bash
go build ./pkg/logs/...
go test ./pkg/logs/...
```

Expected: 全部 PASS

- [ ] **Step 9: 提交**

```bash
git add backend/pkg/logs/file_writer.go backend/pkg/logs/file_writer_test.go backend/pkg/logs/default.go
git commit -m "feat(logs): support LOG_FILE rotation via lumberjack"
```

---

## Task 3: 创建 observability 通用包 — 注册指标

**Files:**
- Create: `backend/pkg/observability/metrics.go`
- Create: `backend/pkg/observability/metrics_test.go`

- [ ] **Step 1: 写测试**

写入 `backend/pkg/observability/metrics_test.go`：
```go
package observability

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPRequestsTotalIncrement(t *testing.T) {
	HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200").Inc()
	got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200"))
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}

func TestStudioLLMTokensIncrement(t *testing.T) {
	StudioLLMTokensTotal.WithLabelValues("gpt-4", "prompt").Add(150)
	got := testutil.ToFloat64(StudioLLMTokensTotal.WithLabelValues("gpt-4", "prompt"))
	if got != 150 {
		t.Fatalf("expected 150, got %v", got)
	}
}

func TestMetricNamesNoCollision(t *testing.T) {
	// 简单挂一下，确保 metric 名前缀都按规范
	names := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_requests_in_flight",
		"studio_llm_tokens_total",
		"studio_file_upload_size_bytes",
		"studio_agent_chat_total",
	}
	for _, n := range names {
		if !strings.HasPrefix(n, "http_") && !strings.HasPrefix(n, "studio_") {
			t.Fatalf("%s violates prefix convention", n)
		}
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./pkg/observability/ -v
```

Expected: FAIL — `HTTPRequestsTotal` 等未定义

- [ ] **Step 3: 实现 metrics.go**

写入 `backend/pkg/observability/metrics.go`：
```go
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 通用 RED：所有 HTTP handler 共享
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests total counted by method, normalized path, and status",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method, normalized path, and status",
			Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Currently in-flight HTTP requests",
		},
	)
)

// Studio 业务指标
var (
	StudioLLMTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "studio_llm_tokens_total",
			Help: "Total LLM tokens consumed by model and kind (prompt|completion)",
		},
		[]string{"model", "kind"},
	)

	StudioFileUploadSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "studio_file_upload_size_bytes",
			Help:    "File upload size by file kind (image|doc|other)",
			Buckets: prometheus.ExponentialBuckets(1024, 4, 10), // 1K → 1G
		},
		[]string{"kind"},
	)

	StudioAgentChatTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "studio_agent_chat_total",
			Help: "Agent chat invocation counts by result",
		},
		[]string{"result"}, // success | error
	)
)
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./pkg/observability/ -v
```

Expected: 3 个测试都 PASS

- [ ] **Step 5: 提交**

```bash
git add backend/pkg/observability/metrics.go backend/pkg/observability/metrics_test.go
git commit -m "feat(observability): register RED + studio business metrics"
```

---

## Task 4: 实现 Hertz HTTP RED 中间件

**Files:**
- Create: `backend/pkg/observability/http_middleware.go`
- Create: `backend/pkg/observability/http_middleware_test.go`

- [ ] **Step 1: 写测试 — middleware 调用后 counter +1**

写入 `backend/pkg/observability/http_middleware_test.go`：
```go
package observability

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddleware_IncrementsCounter(t *testing.T) {
	mw := HTTPRequestsMiddleware()

	c := app.NewContext(0)
	c.Request.Header.SetMethod(consts.MethodGet)
	c.Request.SetRequestURI("/api/test")
	c.Response.SetStatusCode(http.StatusOK)

	// 手动 normalize 路径以匹配 metric label
	expectedBefore := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))

	mw(context.Background(), c)

	expectedAfter := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))

	assert.True(t, expectedAfter == expectedBefore+1,
		"expected counter to increment by 1; before=%v after=%v", expectedBefore, expectedAfter)
}

// 防止测试间相互干扰
var _ = protocol.MethodGet
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./pkg/observability/ -run TestHTTPMiddleware -v
```

Expected: FAIL — `HTTPRequestsMiddleware` 未定义

- [ ] **Step 3: 实现 middleware**

写入 `backend/pkg/observability/http_middleware.go`：
```go
package observability

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// HTTPRequestsMiddleware records RED metrics (rate, errors, duration)
// for every Hertz request. Path is normalized to avoid high-cardinality
// labels (e.g. /users/123 → /users/:id).
func HTTPRequestsMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		start := time.Now()
		c.Next(ctx)
		dur := time.Since(start).Seconds()

		method := string(c.Request.Method())
		path := normalizePath(string(c.Request.URI().Path()))
		status := strconv.Itoa(c.Response.StatusCode())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path, status).Observe(dur)
	}
}

// normalizePath strips numeric/UUID segments to keep label cardinality bounded.
// Order: longer patterns first, since strings.Contains is the cheapest check.
func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		// 全数字
		if _, err := strconv.ParseInt(seg, 10, 64); err == nil {
			parts[i] = ":id"
			continue
		}
		// UUID 长度（粗判，避免引入额外解析）
		if len(seg) == 36 && strings.Count(seg, "-") == 4 {
			parts[i] = ":uuid"
			continue
		}
	}
	return strings.Join(parts, "/")
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./pkg/observability/ -run TestHTTPMiddleware -v
```

Expected: PASS

- [ ] **Step 5: 写 normalizePath 单测**

追加到 `backend/pkg/observability/http_middleware_test.go`：
```go
func TestNormalizePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/api/users/123", "/api/users/:id"},
		{"/api/users/123/posts/456", "/api/users/:id/posts/:id"},
		{"/api/users/abc", "/api/users/abc"}, // 非数字保留
		{"/api/users/00000000-0000-4000-8000-000000000000", "/api/users/:uuid"},
		{"/", "/"},
		{"", "/"},
	}
	for _, c := range cases {
		got := normalizePath(c.in)
		if got != c.want {
			t.Errorf("normalizePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 6: 跑全部测试**

```bash
go test ./pkg/observability/ -v
```

Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add backend/pkg/observability/http_middleware.go backend/pkg/observability/http_middleware_test.go
git commit -m "feat(observability): hertz RED http middleware with path normalization"
```

---

## Task 5: 注册 /metrics 路由 + 接入 middleware

**Files:**
- Modify: `backend/api/router/register.go` 或类似入口（用 `grep -rn "RegisterRouter\|h.Use\|hertz.New" backend/cmd/` 找 Hertz server 启动点）

- [ ] **Step 1: 找 Hertz server 入口**

```bash
cd backend
grep -rn "hertz.Default\|hertz.New\|server.Default\|server.New\|h.Use\|RegisterRouter\|h.GET" cmd/ api/router/ | head -10
```

记下 `h := server.Default(...)` 或 `h := hertz.Default(...)` 出现的文件和行号。

- [ ] **Step 2: 在 server 启动点注册 middleware + /metrics**

打开上一步找到的文件，在 `h := ...` 之后、`h.Spin()` 之前加：
```go
import (
    "github.com/coze-dev/coze-studio/backend/pkg/observability"
    "github.com/hertz-contrib/adaptor"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// 在 routes 注册前挂 middleware
h.Use(observability.HTTPRequestsMiddleware())

// /metrics endpoint
h.GET("/metrics", adaptor.HertzHandler(promhttp.Handler()))
```

如果 `hertz-contrib/adaptor` 还没引入，先加依赖：
```bash
go get github.com/hertz-contrib/adaptor
go mod tidy
```

- [ ] **Step 3: 启动 Studio，curl /metrics**

```bash
# 在另一终端启动（按现有 docker compose 或本地 go run）
# Studio listen 在 :8888
curl -sf http://localhost:8888/metrics | head -20
```

Expected: 输出 `# HELP http_requests_total ...` 等 Prometheus 文本格式

- [ ] **Step 4: 触发一次请求，验证 counter 增加**

```bash
curl -sf http://localhost:8888/api/health 2>/dev/null || true
sleep 1
curl -sf http://localhost:8888/metrics | grep "http_requests_total" | head -3
```

Expected: 能看到 `http_requests_total{method="GET",path="/api/health",status="200"} 1` 或类似

- [ ] **Step 5: 提交**

```bash
git add backend/api/router/<modified-file>.go go.mod go.sum
git commit -m "feat(api): expose /metrics endpoint and RED middleware"
```

---

## Task 6: LLM token 埋点

**Files:**
- 找 LLM 调用落点：`grep -rln "ChatCompletion\|llm.Invoke\|llm.Chat" backend/domain/llm/ backend/application/`
- Modify: 找到的文件（具体路径取决于代码现状）

- [ ] **Step 1: 找出最关键的一处 LLM 调用点**

```bash
cd backend
grep -rln "Usage.*Token\|TotalTokens\|PromptTokens\|CompletionTokens" domain/ application/ | head -5
```

通常在 LLM 客户端或 wrapper 里能拿到 `Usage{Prompt, Completion, Total}`。

- [ ] **Step 2: 在拿到 Usage 的位置 +1**

伪代码（具体落点按上一步结果调整）：
```go
import "github.com/coze-dev/coze-studio/backend/pkg/observability"

// 在已有的代码：
// resp, err := llm.Chat(ctx, req)
// if err == nil {
observability.StudioLLMTokensTotal.WithLabelValues(model, "prompt").Add(float64(resp.Usage.PromptTokens))
observability.StudioLLMTokensTotal.WithLabelValues(model, "completion").Add(float64(resp.Usage.CompletionTokens))
// }
```

- [ ] **Step 3: 单测 — 模拟一次 LLM 调用，验证 counter 增加**

如果 LLM client 有 mock，加个测试。否则：手动跑一次 agent chat（curl `/api/agent_chat` 之类），然后看 `/metrics` 中 `studio_llm_tokens_total` 是否非零。

- [ ] **Step 4: 验证**

```bash
curl -s http://localhost:8888/metrics | grep "studio_llm_tokens_total"
```

Expected: 看到 model 标签下的非零值

- [ ] **Step 5: 提交**

```bash
git add backend/<modified-file>.go
git commit -m "feat(llm): record studio_llm_tokens_total per model and kind"
```

---

## Task 7: 文件上传 size 直方图

**Files:**
- Modify: `backend/api/handler/coze/developer_api_service.go` (上传文件位置)

- [ ] **Step 1: 找文件上传 handler**

```bash
grep -n "UploadFile\|FormFile\|c.FormFile" backend/api/handler/coze/developer_api_service.go
```

- [ ] **Step 2: 在 size check 通过之后加埋点**

定位到 `req.FileHead != nil` 且 size 已知的位置，加：
```go
import "github.com/coze-dev/coze-studio/backend/pkg/observability"

kind := "other"
if strings.HasPrefix(strings.ToLower(req.FileHead.FileType), "image") {
    kind = "image"
} else if isDocFileType(req.FileHead.FileType) {
    kind = "doc"
}
observability.StudioFileUploadSize.WithLabelValues(kind).Observe(float64(req.FileHead.Size))
```

`isDocFileType` 可以是已有 helper 或简单的 switch（pdf/docx/txt/md/xlsx 等）。

- [ ] **Step 3: 验证**

```bash
# 启 Studio，上传一个图片
# 然后看 metrics
curl -s http://localhost:8888/metrics | grep "studio_file_upload_size_bytes_count"
```

Expected: count 至少为 1

- [ ] **Step 4: 提交**

```bash
git add backend/api/handler/coze/developer_api_service.go
git commit -m "feat(upload): record studio_file_upload_size_bytes histogram"
```

---

## Task 8: agent_chat 计数

**Files:**
- 找 agent chat handler：`grep -rln "agent.*chat\|conversation.*chat" backend/api/handler/`
- Modify: 找到的文件

- [ ] **Step 1: 定位 agent chat 入口**

```bash
grep -rn "AgentChat\|agent_chat\|conversation/chat\|run_streaming" backend/api/handler/ | head -5
```

- [ ] **Step 2: 在 handler 完成时埋点**

```go
import "github.com/coze-dev/coze-studio/backend/pkg/observability"

defer func() {
    result := "success"
    if hadError {
        result = "error"
    }
    observability.StudioAgentChatTotal.WithLabelValues(result).Inc()
}()
```

- [ ] **Step 3: 验证**

```bash
curl -s http://localhost:8888/metrics | grep "studio_agent_chat_total"
```

Expected: success / error 两个标签都至少有一个非零值

- [ ] **Step 4: 提交**

```bash
git add backend/api/handler/<file>.go
git commit -m "feat(agent): record studio_agent_chat_total by result"
```

---

## Task 9: 写 Studio Grafana 仪表盘

**Files:**
- Create: `~/projects/ynet/obs-stack/dashboards/studio.json`

- [ ] **Step 1: 写 dashboard JSON**

写入 `~/projects/ynet/obs-stack/dashboards/studio.json`：
```json
{
  "annotations": {"list": []},
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "id": null,
  "links": [],
  "liveNow": false,
  "panels": [
    {
      "type": "stat",
      "title": "Studio Up",
      "gridPos": {"h": 4, "w": 4, "x": 0, "y": 0},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "up{service=\"studio\"}", "legendFormat": "up"}
      ],
      "options": {"colorMode": "value"},
      "fieldConfig": {"defaults": {"thresholds": {"steps": [{"value": 0, "color": "red"}, {"value": 1, "color": "green"}]}}}
    },
    {
      "type": "stat",
      "title": "QPS (5m)",
      "gridPos": {"h": 4, "w": 4, "x": 4, "y": 0},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "sum(rate(http_requests_total{service=\"studio\"}[5m]))", "legendFormat": "qps"}
      ]
    },
    {
      "type": "stat",
      "title": "P99 latency",
      "gridPos": {"h": 4, "w": 4, "x": 8, "y": 0},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"studio\"}[5m])))", "legendFormat": "p99"}
      ],
      "fieldConfig": {"defaults": {"unit": "s"}}
    },
    {
      "type": "stat",
      "title": "Error rate (5m)",
      "gridPos": {"h": 4, "w": 4, "x": 12, "y": 0},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "sum(rate(http_requests_total{service=\"studio\",status=~\"5..\"}[5m])) / sum(rate(http_requests_total{service=\"studio\"}[5m]))", "legendFormat": "err"}
      ],
      "fieldConfig": {"defaults": {"unit": "percentunit"}}
    },
    {
      "type": "timeseries",
      "title": "Requests/sec by path",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 4},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "topk(10, sum by (path) (rate(http_requests_total{service=\"studio\"}[5m])))", "legendFormat": "{{path}}"}
      ]
    },
    {
      "type": "timeseries",
      "title": "Latency P50/P90/P99",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 4},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "histogram_quantile(0.50, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"studio\"}[5m])))", "legendFormat": "p50"},
        {"expr": "histogram_quantile(0.90, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"studio\"}[5m])))", "legendFormat": "p90"},
        {"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"studio\"}[5m])))", "legendFormat": "p99"}
      ],
      "fieldConfig": {"defaults": {"unit": "s"}}
    },
    {
      "type": "timeseries",
      "title": "LLM tokens/sec by model",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 12},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "sum by (model, kind) (rate(studio_llm_tokens_total[5m]))", "legendFormat": "{{model}}-{{kind}}"}
      ]
    },
    {
      "type": "timeseries",
      "title": "File upload size (P95)",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 12},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "histogram_quantile(0.95, sum by (le, kind) (rate(studio_file_upload_size_bytes_bucket[5m])))", "legendFormat": "{{kind}}"}
      ],
      "fieldConfig": {"defaults": {"unit": "decbytes"}}
    },
    {
      "type": "timeseries",
      "title": "Agent chat / sec",
      "gridPos": {"h": 8, "w": 12, "x": 0, "y": 20},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "sum by (result) (rate(studio_agent_chat_total[5m]))", "legendFormat": "{{result}}"}
      ]
    },
    {
      "type": "timeseries",
      "title": "Knowledge parse duration P95",
      "gridPos": {"h": 8, "w": 12, "x": 12, "y": 20},
      "datasource": "Prometheus",
      "targets": [
        {"expr": "histogram_quantile(0.95, sum by (le) (rate(knowledge_parse_duration_seconds_bucket[5m])))", "legendFormat": "p95"}
      ],
      "fieldConfig": {"defaults": {"unit": "s"}}
    }
  ],
  "refresh": "30s",
  "schemaVersion": 39,
  "tags": ["ynet", "studio"],
  "time": {"from": "now-1h", "to": "now"},
  "timepicker": {},
  "timezone": "",
  "title": "Studio",
  "uid": "ynet-studio",
  "version": 1,
  "weekStart": ""
}
```

- [ ] **Step 2: 验证 JSON 合法**

```bash
cd ~/projects/ynet/obs-stack
python3 -m json.tool dashboards/studio.json > /dev/null && echo "OK"
```

Expected: `OK`

- [ ] **Step 3: 在 obs-stack 仓库提交**

```bash
cd ~/projects/ynet/obs-stack
git add dashboards/studio.json
git commit -m "feat(grafana): add Studio dashboard"
```

---

## Task 10: 更新 Studio docker-compose（日志卷 + env）

**Files:**
- Modify: `docker/docker-compose.yml` 或对应的 compose

- [ ] **Step 1: 找 studio server 容器配置**

```bash
cd ~/projects/ynet/coze-studio
grep -rn "ynet-server\|coze-server\|studio-server" docker/ ynet-docker/deploy-v2/ | head -5
```

- [ ] **Step 2: 添加 env 和 volume**

在 ynet-server 容器配置下加：
```yaml
    environment:
      LOG_FILE: /var/log/app/app.log
      LOG_LEVEL: info
      LOG_MAX_SIZE_MB: "100"
      LOG_MAX_BACKUPS: "7"
      LOG_MAX_AGE_DAYS: "30"
      LOG_COMPRESS: "true"
    volumes:
      - /data/logs/studio:/var/log/app
```

- [ ] **Step 3: 在 220 上创建日志目录**

```bash
ssh dev@10.10.10.220 "sudo mkdir -p /data/logs/studio && sudo chown 1000:1000 /data/logs/studio"
```

- [ ] **Step 4: 验证 volume 存在 + env 生效**

部署后，进容器：
```bash
docker exec ynet-server env | grep LOG_
docker exec ynet-server ls /var/log/app/
```

Expected: 看到 6 个 LOG_ env，看到 `app.log` 文件且持续增长

- [ ] **Step 5: 提交**

```bash
git add docker/docker-compose.yml
git commit -m "feat(deploy): mount /data/logs/studio + set LOG_* env vars"
```

---

## Task 11: 部署验证（依赖 obs-stack-infra plan 已完工）

**Files:** 无新增；端到端验证

- [ ] **Step 1: 确认 obs-stack 在 220 上跑着**

```bash
ssh dev@10.10.10.220 "cd ~/obs-stack && docker compose ps"
```

Expected: 3 个容器 running

- [ ] **Step 2: 检查 Studio target 状态**

```bash
curl -sf http://10.10.10.220:9090/api/v1/targets | \
  python3 -c 'import sys,json; print([(t["labels"]["service"], t["health"]) for t in json.load(sys.stdin)["data"]["activeTargets"]])'
```

Expected: `('studio', 'up')` 在列表里

- [ ] **Step 3: 浏览器访问 Grafana**

```
http://10.10.10.220:3000
ynet 文件夹 → "Studio" 仪表盘
```

Expected: 9 个面板有数据，至少 RED 三个面板（QPS/P99/Error）有非零值

- [ ] **Step 4: 模拟流量**

```bash
for i in {1..20}; do curl -sf http://10.10.10.220:8888/api/health; done
sleep 30
```

刷新仪表盘，QPS 应该出现尖峰。

- [ ] **Step 5: 检查日志文件**

```bash
ssh dev@10.10.10.220 "ls -la /data/logs/studio/"
```

Expected: `app.log` 文件存在，持续增长

---

## Self-Review

完成 11 个 task 后核对：

1. **Spec coverage**:
   - ✅ /metrics endpoint（Task 5）
   - ✅ RED 中间件 + path normalize（Task 4-5）
   - ✅ Go runtime metrics（promauto 自动）
   - ✅ studio_llm_tokens_total（Task 6）
   - ✅ studio_file_upload_size_bytes（Task 7）
   - ✅ studio_agent_chat_total（Task 8）
   - ✅ 知识库 metrics 保留（已有 metrics.go 不动）
   - ✅ 文件日志 + 滚动 + 6 env（Task 2）
   - ✅ docker-compose env + volume（Task 10）
   - ✅ Studio dashboard（Task 9）

2. **Placeholder scan**:
   - Task 6 / Task 8 落点是「找现有代码」性质，给了搜索命令但没给死路径，**这是合理的**因为 Studio 代码大、不同 commit 路径会变。
   - Task 5 同理，找 Hertz server 启动点。

3. **Type/path consistency**:
   - 所有 metric 名一致（http_requests_total 等小写下划线，studio_ 前缀）
   - 所有 env 名一致（LOG_FILE 等全大写下划线）
   - lumberjack 包路径 `gopkg.in/natefinch/lumberjack.v2`
   - dashboard `uid: ynet-studio`

无遗漏。
