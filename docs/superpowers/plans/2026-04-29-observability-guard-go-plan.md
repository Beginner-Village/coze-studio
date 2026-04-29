# Observability — Guard-go Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 Guard-go (`~/projects/ynet/guard-go`) 添加 `/metrics` endpoint、RED + 业务指标（3 来源命中率、AI tokens、检测耗时）、文件日志 + 滚动。Guard-go 当前用 stdlib `log.Printf` 几乎无封装，要新增 `internal/logging/` 包并把 `internal/middleware/logger.go` 改造成结构化日志中间件。

**Architecture:** 新建 `internal/logging/` 提供 `Init()` / `Info()` / `Error()` 等带级别的日志接口，底层 io.Writer 经 lumberjack 双写。`internal/observability/` 注册 RED + 业务指标，`/metrics` 通过 Gin 路由用 `gin.WrapH(promhttp.Handler())` 暴露。业务指标埋点在 `shield_engine.go` 的 3 个检测来源处。

**Tech Stack:** Go 1.22+, Gin, prometheus/client_golang, gopkg.in/natefinch/lumberjack.v2

**Spec:** [docs/superpowers/specs/2026-04-29-observability-stack-design.md](../specs/2026-04-29-observability-stack-design.md)

---

## File Structure

```
internal/
├── logging/                              # NEW: 结构化日志包
│   ├── logger.go                         # 公共 API: Init/Debug/Info/Warn/Error
│   ├── file_writer.go                    # lumberjack wrapper
│   └── file_writer_test.go
├── middleware/
│   └── logger.go                         # MODIFY: log.Printf 换成 logging.Info
├── observability/                        # NEW: 指标
│   ├── metrics.go
│   ├── http_middleware.go
│   └── http_middleware_test.go
├── service/
│   └── shield_engine.go                  # MODIFY: 3 个 source 命中处加埋点
└── ...

cmd/guard/main.go                         # MODIFY: 启动 logging.Init() + 注册 /metrics

~/projects/ynet/obs-stack/
└── dashboards/guard-go.json              # NEW
```

---

## Task 1: 添加 lumberjack + prometheus 依赖

**Files:** `go.mod`, `go.sum`

- [ ] **Step 1: 添加依赖**

```bash
cd ~/projects/ynet/guard-go
go get gopkg.in/natefinch/lumberjack.v2@v2.2.1
go get github.com/prometheus/client_golang@v1.20.5
go mod tidy
```

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

Expected: 通过

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "feat(deps): add lumberjack and prometheus client"
```

---

## Task 2: 新建 internal/logging 包 — file rotation

**Files:**
- Create: `internal/logging/file_writer.go`
- Create: `internal/logging/file_writer_test.go`

- [ ] **Step 1: 写测试**

写入 `internal/logging/file_writer_test.go`：
```go
package logging

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
}

func TestNewWriter_CreatesFileWhenLogFileSet(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	w := NewWriter()
	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("expected log file, got error: %v", err)
	}
}

func TestNewWriter_RotatesAtMaxSize(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "app.log")
	t.Setenv("LOG_FILE", logPath)
	t.Setenv("LOG_MAX_SIZE_MB", "1")
	t.Setenv("LOG_MAX_BACKUPS", "3")
	w := NewWriter()
	chunk := make([]byte, 4096)
	for i := range chunk {
		chunk[i] = 'A'
	}
	for i := 0; i < 400; i++ {
		_, _ = w.Write(chunk)
	}
	entries, _ := os.ReadDir(tmp)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "app") {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected ≥2 log files, got %d", count)
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/logging/ -v
```

Expected: FAIL — undefined

- [ ] **Step 3: 实现 file_writer.go**

写入 `internal/logging/file_writer.go`：
```go
package logging

import (
	"io"
	"os"
	"strconv"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewWriter returns an io.Writer that writes to stdout, plus a rolling
// file when LOG_FILE is set. Configuration via env (defaults in parens):
//   LOG_FILE          (empty: stdout only)
//   LOG_MAX_SIZE_MB   (100)
//   LOG_MAX_BACKUPS   (7)
//   LOG_MAX_AGE_DAYS  (30)
//   LOG_COMPRESS      (true)
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
go test ./internal/logging/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/logging/file_writer.go internal/logging/file_writer_test.go
git commit -m "feat(logging): rolling file writer"
```

---

## Task 3: 实现 logger.go — 带级别的日志接口

**Files:**
- Create: `internal/logging/logger.go`
- Create: `internal/logging/logger_test.go`

- [ ] **Step 1: 写测试**

写入 `internal/logging/logger_test.go`：
```go
package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	t.Setenv("LOG_LEVEL", "warn")
	Init(&buf)

	Debug("debug-msg")
	Info("info-msg")
	Warn("warn-msg")
	Error("error-msg")

	out := buf.String()
	if strings.Contains(out, "debug-msg") {
		t.Errorf("debug should be filtered, got: %s", out)
	}
	if strings.Contains(out, "info-msg") {
		t.Errorf("info should be filtered, got: %s", out)
	}
	if !strings.Contains(out, "warn-msg") {
		t.Errorf("warn should pass: %s", out)
	}
	if !strings.Contains(out, "error-msg") {
		t.Errorf("error should pass: %s", out)
	}
}

func TestFormatHasLevel(t *testing.T) {
	var buf bytes.Buffer
	t.Setenv("LOG_LEVEL", "info")
	Init(&buf)
	Info("hello")

	if !strings.Contains(buf.String(), "INFO") {
		t.Errorf("expected level prefix INFO in: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("expected message hello in: %s", buf.String())
	}
}

func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	t.Setenv("LOG_LEVEL", "info")
	Init(&buf)
	Infof("user=%s id=%d", "alice", 42)

	if !strings.Contains(buf.String(), "user=alice id=42") {
		t.Errorf("formatting failed: %s", buf.String())
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/logging/ -run TestLevel -v
```

Expected: FAIL — undefined

- [ ] **Step 3: 实现 logger.go**

写入 `internal/logging/logger.go`：
```go
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

func parseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "INFO"
	}
}

var (
	mu        sync.Mutex
	writer    io.Writer = os.Stdout
	level     Level     = LevelInfo
	stdLogger *log.Logger
)

// Init configures the global logger. If w is nil, uses NewWriter() (env-driven).
// Reads LOG_LEVEL from env.
func Init(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	if w == nil {
		w = NewWriter()
	}
	writer = w
	level = parseLevel(os.Getenv("LOG_LEVEL"))
	stdLogger = log.New(w, "", 0)
}

func init() {
	Init(nil)
}

func logAt(lvl Level, msg string) {
	mu.Lock()
	defer mu.Unlock()
	if lvl < level {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	stdLogger.Printf("%s %s %s", lvl, ts, msg)
}

func Debug(msg string)                     { logAt(LevelDebug, msg) }
func Info(msg string)                      { logAt(LevelInfo, msg) }
func Warn(msg string)                      { logAt(LevelWarn, msg) }
func Error(msg string)                     { logAt(LevelError, msg) }
func Fatal(msg string)                     { logAt(LevelFatal, msg); os.Exit(1) }
func Debugf(format string, v ...any)       { logAt(LevelDebug, fmt.Sprintf(format, v...)) }
func Infof(format string, v ...any)        { logAt(LevelInfo, fmt.Sprintf(format, v...)) }
func Warnf(format string, v ...any)        { logAt(LevelWarn, fmt.Sprintf(format, v...)) }
func Errorf(format string, v ...any)       { logAt(LevelError, fmt.Sprintf(format, v...)) }
func Fatalf(format string, v ...any)       { logAt(LevelFatal, fmt.Sprintf(format, v...)); os.Exit(1) }
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/logging/ -v
```

Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/logging/logger.go internal/logging/logger_test.go
git commit -m "feat(logging): leveled logger with env-driven config"
```

---

## Task 4: 替换 middleware/logger.go 用新 logger

**Files:**
- Modify: `internal/middleware/logger.go`

- [ ] **Step 1: 改 Logger middleware**

把 `internal/middleware/logger.go` 整个内容替换为：
```go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"

	"github.com/Beginner-Village/guard-go/internal/logging"
)

const (
	traceIDKey   = "trace_id"
	requestIDKey = "request_id"
)

// Logger logs request method/path/status/latency with trace + request IDs.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = ulid.Make().String()
		}
		requestID := ulid.Make().String()

		c.Set(traceIDKey, traceID)
		c.Set(requestIDKey, requestID)
		c.Header("X-Trace-ID", traceID)
		c.Header("X-Request-ID", requestID)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logging.Infof("[%s] [%s] %s %s %d %v",
			traceID, requestID,
			c.Request.Method, c.Request.URL.Path,
			status, latency,
		)
	}
}
```

> 注意 import path `github.com/Beginner-Village/guard-go` 来自 go.mod 的 module 名，如果不一样按实际改。

- [ ] **Step 2: 验证编译**

```bash
go build ./...
```

Expected: 通过

- [ ] **Step 3: 提交**

```bash
git add internal/middleware/logger.go
git commit -m "refactor(middleware): use new logging package"
```

---

## Task 5: 在 main.go 调用 logging.Init() + 设置端口

**Files:** `cmd/guard/main.go`

- [ ] **Step 1: 在 main 函数顶部加 Init**

打开 `cmd/guard/main.go`，在 `main()` 第一行加：
```go
import "github.com/Beginner-Village/guard-go/internal/logging"

func main() {
    logging.Init(nil)  // 用 env-driven writer
    // ... existing code
}
```

- [ ] **Step 2: 验证编译**

```bash
go build ./cmd/guard/
```

- [ ] **Step 3: 启动一次手动验证日志输出**

```bash
LOG_LEVEL=info LOG_FILE=/tmp/guard.log ./guard &
sleep 2
ls -la /tmp/guard.log
cat /tmp/guard.log | head
kill %1
```

Expected: `/tmp/guard.log` 存在，里面有 `INFO ... HTTP server listening on ...` 类日志

- [ ] **Step 4: 提交**

```bash
git add cmd/guard/main.go
git commit -m "feat(main): initialize logging at startup"
```

---

## Task 6: observability 包 — 注册 RED + 业务指标

**Files:**
- Create: `internal/observability/metrics.go`
- Create: `internal/observability/metrics_test.go`

- [ ] **Step 1: 写测试**

写入 `internal/observability/metrics_test.go`：
```go
package observability

import (
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

func TestShieldCheckTotalIncrement(t *testing.T) {
	ShieldCheckTotal.WithLabelValues("blocklist", "hit").Inc()
	got := testutil.ToFloat64(ShieldCheckTotal.WithLabelValues("blocklist", "hit"))
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}

func TestShieldAITokensIncrement(t *testing.T) {
	ShieldAITokensTotal.WithLabelValues("Qwen3Guard-Gen-4B", "completion").Add(50)
	got := testutil.ToFloat64(ShieldAITokensTotal.WithLabelValues("Qwen3Guard-Gen-4B", "completion"))
	if got != 50 {
		t.Fatalf("expected 50, got %v", got)
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/observability/ -v
```

Expected: FAIL — undefined

- [ ] **Step 3: 实现 metrics.go**

写入 `internal/observability/metrics.go`：
```go
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 通用 RED
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "HTTP requests"},
		[]string{"method", "path", "status"},
	)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP latency",
			Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{Name: "http_requests_in_flight", Help: "In-flight"},
	)
)

// Shield 业务指标
var (
	ShieldCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shield_check_total",
			Help: "Shield check by source (blocklist|knowledge_base|ai_model) and result (hit|miss)",
		},
		[]string{"source", "result"},
	)
	ShieldCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "shield_check_duration_seconds",
			Help:    "Shield check latency by source",
			Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5},
		},
		[]string{"source"},
	)
	ShieldAITokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shield_ai_tokens_total",
			Help: "AI model tokens used by shield",
		},
		[]string{"model", "kind"},
	)
)
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/observability/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/observability/metrics.go internal/observability/metrics_test.go
git commit -m "feat(observability): RED + shield business metrics"
```

---

## Task 7: Gin RED 中间件

**Files:**
- Create: `internal/observability/http_middleware.go`
- Create: `internal/observability/http_middleware_test.go`

- [ ] **Step 1: 写测试**

写入 `internal/observability/http_middleware_test.go`：
```go
package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddleware_IncrementsCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(HTTPRequestsMiddleware())
	r.GET("/foo", func(c *gin.Context) { c.Status(http.StatusOK) })

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	r.ServeHTTP(w, req)

	after := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200"))
	if after != before+1 {
		t.Fatalf("expected counter +1, got before=%v after=%v", before, after)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/api/users/123", "/api/users/:id"},
		{"/", "/"},
		{"", "/"},
		{"/api/users/00000000-0000-4000-8000-000000000000", "/api/users/:uuid"},
	}
	for _, c := range cases {
		if got := normalizePath(c.in); got != c.want {
			t.Errorf("normalizePath(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/observability/ -run TestHTTPMiddleware -v
```

Expected: FAIL

- [ ] **Step 3: 实现 middleware**

写入 `internal/observability/http_middleware.go`：
```go
package observability

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func HTTPRequestsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		start := time.Now()
		c.Next()
		dur := time.Since(start).Seconds()

		method := c.Request.Method
		// Gin 把 ":id" 这种路径参数原样保留在 c.FullPath()，比 URL.Path 更稳定
		path := c.FullPath()
		if path == "" {
			path = normalizePath(c.Request.URL.Path)
		}
		status := strconv.Itoa(c.Writer.Status())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path, status).Observe(dur)
	}
}

func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		if _, err := strconv.ParseInt(seg, 10, 64); err == nil {
			parts[i] = ":id"
			continue
		}
		if len(seg) == 36 && strings.Count(seg, "-") == 4 {
			parts[i] = ":uuid"
		}
	}
	return strings.Join(parts, "/")
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/observability/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/observability/http_middleware.go internal/observability/http_middleware_test.go
git commit -m "feat(observability): gin RED middleware with FullPath"
```

---

## Task 8: 在 main.go 注册 /metrics + 接 middleware

**Files:**
- Modify: `cmd/guard/main.go`

- [ ] **Step 1: 当前 mux 用法定位**

```bash
grep -n "mux\|router\|gin.Default\|gin.New" cmd/guard/main.go internal/router/ 2>/dev/null | head -10
```

如果当前 main.go 用 `http.ServeMux`，要看是不是只有 health 那种简单 handler。如果 router 在别的地方用 Gin，就在 Gin engine 上挂。

- [ ] **Step 2: 在 Gin engine 上挂 /metrics + middleware**

假设 main.go 中有 `r := gin.New()`：
```go
import (
    "github.com/Beginner-Village/guard-go/internal/observability"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

r.Use(observability.HTTPRequestsMiddleware())
r.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

如果 main.go 同时用 net/http mux 启动（line 88 `http.ListenAndServe(cfg.Server.ListenAddr, mux)`），把 `/metrics` 同时挂到 mux 上：
```go
mux.Handle("/metrics", promhttp.Handler())
```

> 取决于现有架构，可能两边都加（Gin engine 给业务路由，net/http mux 给 health/metrics）。具体看 main.go 现状。

- [ ] **Step 3: 启动 + curl /metrics**

```bash
LISTEN_PORT=8180 ./guard &
sleep 2
curl -sf http://localhost:8180/metrics | head -20
```

Expected: 输出 prometheus 文本格式

- [ ] **Step 4: 提交**

```bash
git add cmd/guard/main.go
git commit -m "feat(api): expose /metrics and RED middleware"
```

---

## Task 9: 业务指标埋点 — 3 来源命中率 + 检测耗时

**Files:**
- Modify: `internal/service/shield_engine.go`

- [ ] **Step 1: 找 3 个来源的检测函数**

```bash
grep -n "checkBlocklist\|checkKnowledge\|checkAI\|DetectionSource" internal/service/shield_engine.go | head -20
```

通常会有类似：
- `checkBlocklist(ctx, content) -> hit, err`
- `checkKnowledgeBase(ctx, content) -> hit, err`
- `checkAIModel(ctx, content) -> hit, err`

- [ ] **Step 2: 在每个 check 函数顶部 + 退出时埋点**

伪代码（实际调整）：
```go
import "github.com/Beginner-Village/guard-go/internal/observability"

func (e *Engine) checkBlocklist(ctx context.Context, content string) (hit bool, err error) {
    defer func(start time.Time) {
        observability.ShieldCheckDuration.WithLabelValues("blocklist").Observe(time.Since(start).Seconds())
        result := "miss"
        if hit {
            result = "hit"
        }
        observability.ShieldCheckTotal.WithLabelValues("blocklist", result).Inc()
    }(time.Now())
    
    // ... existing logic
}
```

对 `checkKnowledgeBase` 用 source label `knowledge_base`，`checkAIModel` 用 `ai_model`。

- [ ] **Step 3: AI 模型 token 埋点**

`checkAIModel` 拿到 LLM response 后：
```go
if resp.Usage.PromptTokens > 0 {
    observability.ShieldAITokensTotal.WithLabelValues(modelName, "prompt").Add(float64(resp.Usage.PromptTokens))
}
if resp.Usage.CompletionTokens > 0 {
    observability.ShieldAITokensTotal.WithLabelValues(modelName, "completion").Add(float64(resp.Usage.CompletionTokens))
}
```

- [ ] **Step 4: 启动 + 触发一次检测 + 验证**

```bash
LISTEN_PORT=8180 ./guard &
sleep 3

curl -X POST http://localhost:8180/api/shield/check/text \
  -H 'Content-Type: application/json' \
  -d '{"product_code":"test","business_code":"test","content":"hello world"}'

curl -s http://localhost:8180/metrics | grep "shield_check"
curl -s http://localhost:8180/metrics | grep "shield_ai_tokens"
```

Expected: shield_check_total 至少有一个 source 命中（如 ai_model），shield_ai_tokens 有非零 prompt/completion

- [ ] **Step 5: 提交**

```bash
git add internal/service/shield_engine.go
git commit -m "feat(shield): record per-source check counts/duration + AI tokens"
```

---

## Task 10: 写 Guard-go Grafana 仪表盘

**Files:**
- Create: `~/projects/ynet/obs-stack/dashboards/guard-go.json`

- [ ] **Step 1: 写 dashboard JSON**

写入 `~/projects/ynet/obs-stack/dashboards/guard-go.json`：
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
    {"type": "stat", "title": "Guard-go Up", "gridPos": {"h": 4, "w": 4, "x": 0, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "up{service=\"guard-go\"}"}], "options": {"colorMode": "value"}, "fieldConfig": {"defaults": {"thresholds": {"steps": [{"value": 0, "color": "red"}, {"value": 1, "color": "green"}]}}}},
    {"type": "stat", "title": "QPS (5m)", "gridPos": {"h": 4, "w": 4, "x": 4, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(http_requests_total{service=\"guard-go\"}[5m]))"}]},
    {"type": "stat", "title": "P99 latency", "gridPos": {"h": 4, "w": 4, "x": 8, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"guard-go\"}[5m])))"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "stat", "title": "Error rate", "gridPos": {"h": 4, "w": 4, "x": 12, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(http_requests_total{service=\"guard-go\",status=~\"5..\"}[5m])) / sum(rate(http_requests_total{service=\"guard-go\"}[5m]))"}], "fieldConfig": {"defaults": {"unit": "percentunit"}}},
    {"type": "timeseries", "title": "Requests/sec by path", "gridPos": {"h": 8, "w": 12, "x": 0, "y": 4}, "datasource": "Prometheus", "targets": [{"expr": "topk(10, sum by (path) (rate(http_requests_total{service=\"guard-go\"}[5m])))", "legendFormat": "{{path}}"}]},
    {"type": "timeseries", "title": "Latency P50/P90/P99", "gridPos": {"h": 8, "w": 12, "x": 12, "y": 4}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.50, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"guard-go\"}[5m])))", "legendFormat": "p50"}, {"expr": "histogram_quantile(0.90, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"guard-go\"}[5m])))", "legendFormat": "p90"}, {"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"guard-go\"}[5m])))", "legendFormat": "p99"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "piechart", "title": "Hit ratio by source", "gridPos": {"h": 8, "w": 8, "x": 0, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "sum by (source) (increase(shield_check_total{result=\"hit\"}[1h]))", "legendFormat": "{{source}}"}]},
    {"type": "timeseries", "title": "Check duration P95 by source", "gridPos": {"h": 8, "w": 8, "x": 8, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.95, sum by (le, source) (rate(shield_check_duration_seconds_bucket[5m])))", "legendFormat": "{{source}}"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "timeseries", "title": "AI tokens / sec", "gridPos": {"h": 8, "w": 8, "x": 16, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "sum by (model, kind) (rate(shield_ai_tokens_total[5m]))", "legendFormat": "{{model}}-{{kind}}"}]}
  ],
  "refresh": "30s",
  "schemaVersion": 39,
  "tags": ["ynet", "guard-go"],
  "time": {"from": "now-1h", "to": "now"},
  "title": "Guard-go",
  "uid": "ynet-guard-go",
  "version": 1
}
```

- [ ] **Step 2: 验证 JSON**

```bash
cd ~/projects/ynet/obs-stack
python3 -m json.tool dashboards/guard-go.json > /dev/null && echo "OK"
```

- [ ] **Step 3: 提交**

```bash
cd ~/projects/ynet/obs-stack
git add dashboards/guard-go.json
git commit -m "feat(grafana): add Guard-go dashboard"
```

---

## Task 11: 更新 docker-compose（日志卷 + env）

**Files:**
- Modify: `docker-compose.yml` 或对应部署模板

- [ ] **Step 1: 加 env + volume**

```yaml
services:
  guard-go-app:
    image: ynet/guard-go:latest
    environment:
      LISTEN_PORT: "8180"
      LOG_FILE: /var/log/app/app.log
      LOG_LEVEL: info
      LOG_MAX_SIZE_MB: "100"
      LOG_MAX_BACKUPS: "7"
      LOG_MAX_AGE_DAYS: "30"
      LOG_COMPRESS: "true"
      # ... 其他原有 env
    volumes:
      - /data/logs/guard-go:/var/log/app
```

- [ ] **Step 2: 220 上建目录**

```bash
ssh dev@10.10.10.220 "sudo mkdir -p /data/logs/guard-go && sudo chown 1000:1000 /data/logs/guard-go"
```

- [ ] **Step 3: 重启 + 验证**

```bash
ssh dev@10.10.10.220 "cd /opt/guard-go && docker compose up -d"
sleep 10
curl -sf http://10.10.10.220:8180/metrics | head -3
ssh dev@10.10.10.220 "ls /data/logs/guard-go/"
```

- [ ] **Step 4: 提交**

```bash
git add docker-compose.yml
git commit -m "feat(deploy): mount /data/logs/guard-go + LOG env vars"
```

---

## Task 12: 部署验证

- [ ] **Step 1: target up**

```bash
curl -sf http://10.10.10.220:9090/api/v1/targets | grep -o '"service":"guard-go","health":"up"'
```

- [ ] **Step 2: Grafana 仪表盘**

浏览器开 `http://10.10.10.220:3000`，"Guard-go" 仪表盘要有 RED + 3 来源 + AI tokens 数据。

- [ ] **Step 3: 模拟流量**

```bash
for i in {1..20}; do
  curl -s -X POST http://10.10.10.220:8180/api/shield/check/text \
    -H 'Content-Type: application/json' \
    -d '{"product_code":"test","business_code":"test","content":"hello"}'
done
sleep 30
```

刷新仪表盘，shield_check 统计应增加。

---

## Self-Review

1. **Spec coverage**:
   - ✅ /metrics endpoint via Gin（Task 8）
   - ✅ RED middleware + path（Task 7-8）
   - ✅ shield_check_total / duration（Task 9）
   - ✅ shield_ai_tokens_total（Task 9）
   - ✅ 文件日志 + 滚动 + 6 env（Task 2-3）
   - ✅ docker-compose env + volume（Task 11）
   - ✅ Guard-go dashboard（Task 10）
   - ✅ 替换 stdlib log.Printf 为 logging package（Task 4）

2. **Placeholder scan**:
   - import path `github.com/Beginner-Village/guard-go` 来自实际 go.mod，agent 实施时检查一致即可

3. **Type/path consistency**:
   - 端口 8180 在 docker-compose 与 Prometheus target 一致
   - logging package 函数签名一致（Init/Info/Warn/Error...）
   - dashboard `uid: ynet-guard-go`

无遗漏。
