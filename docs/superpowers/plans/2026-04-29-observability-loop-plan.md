# Observability — Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 Loop (`~/projects/ynet/coze-loop`) 添加 `/metrics` endpoint、RED + 业务指标（任务流水线、evaluator）、文件日志输出 + 滚动；并把端口从 8888 改为 8889 避免和 Studio 冲突。把 Loop 仪表盘加到 obs-stack。

**Architecture:** 用 `prometheus/client_golang` 在 Hertz 路由挂 `/metrics`；HTTP RED 中间件复用 Studio 同款实现（独立包 `pkg/observability/`）。logrus 通过 `log.SetOutput(io.MultiWriter)` 双写到 lumberjack。业务指标在 task scheduler 和 evaluator 调用点埋。

**Tech Stack:** Go 1.22+, Hertz, logrus, prometheus/client_golang, gopkg.in/natefinch/lumberjack.v2

**Spec:** [docs/superpowers/specs/2026-04-29-observability-stack-design.md](../specs/2026-04-29-observability-stack-design.md)

---

## File Structure

```
backend/
├── pkg/
│   ├── logs/default.go                       # MODIFY: logrus.SetOutput(MultiWriter)
│   └── observability/                        # NEW: 复用 Studio 同款（拷过来）
│       ├── metrics.go                        # 通用 RED + Loop 业务
│       ├── http_middleware.go
│       └── http_middleware_test.go
└── cmd/main.go                               # MODIFY: 改默认端口 8889 + 注册 /metrics

~/projects/ynet/obs-stack/
└── dashboards/loop.json                      # NEW: Loop 仪表盘
```

---

## Task 0: 端口从 8888 改成 8889

**Files:**
- Modify: server 启动配置（在 `release/deployment/helm-chart/charts/app/values.yaml` / `backend/cmd/main.go` / 配置 yaml 等）

- [ ] **Step 1: 找当前监听端口配置**

```bash
cd ~/projects/ynet/coze-loop
grep -rn "8888\|listen.*port\|port.*888" backend/cmd/ release/ conf/ 2>/dev/null | head -10
```

记下出现位置。常见有：
- `backend/cmd/main.go` 里 `server.Default(server.WithHostPorts(":8888"))`
- `release/deployment/helm-chart/charts/app/values.yaml` 的 `service.port: 8888`
- 健康检查脚本 `release/.../healthcheck.sh` 里 `localhost:8888/ping`

- [ ] **Step 2: 全部改成 8889**

把所有出现 `8888` 的地方都改成 `8889`，**注意区分**：
- 真正监听端口的位置 → 改
- 第三方服务（如另一个组件刚好也 8888）→ 不动

```bash
grep -rln "8888" backend/cmd/ release/ conf/ 2>/dev/null | \
  xargs -I {} sed -i.bak 's|:8888|:8889|g; s|"8888"|"8889"|g; s|=8888|=8889|g' {}
# 找到的 .bak 检查 diff，确认无误后删
find . -name "*.bak" -delete
```

- [ ] **Step 3: 重新编译验证**

```bash
cd backend
go build ./cmd/...
```

Expected: 编译通过

- [ ] **Step 4: 启动服务验证端口**

```bash
# 在另一终端启动 backend
# 主进程应监听 :8889
ss -ltn | grep :8889
```

Expected: 有 LISTEN 行

- [ ] **Step 5: 提交**

```bash
git add backend/ release/ conf/
git commit -m "feat(loop): change listen port from 8888 to 8889 (avoid Studio conflict)"
```

---

## Task 1: 添加 lumberjack 依赖

**Files:**
- Modify: `go.mod` / `go.sum`

- [ ] **Step 1: 添加依赖**

```bash
cd ~/projects/ynet/coze-loop
go get gopkg.in/natefinch/lumberjack.v2@v2.2.1
go mod tidy
```

- [ ] **Step 2: 验证编译**

```bash
go build ./backend/pkg/logs/...
```

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "feat(deps): add lumberjack v2"
```

---

## Task 2: logrus 双写到 lumberjack

**Files:**
- Modify: `backend/pkg/logs/default.go`
- Create: `backend/pkg/logs/file_writer.go`
- Create: `backend/pkg/logs/file_writer_test.go`

- [ ] **Step 1: 写测试**

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
		t.Fatalf("expected ≥2 log files after rotation, got %d", count)
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./backend/pkg/logs/ -run TestNewWriter -v
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
go test ./backend/pkg/logs/ -run TestNewWriter -v
```

Expected: 3 个测试 PASS

- [ ] **Step 5: 把 logrus 输出接到 NewWriter**

修改 `backend/pkg/logs/default.go`，在 `newDefaultLogger()` 函数（line 109-114）：

把：
```go
func newDefaultLogger() Logger {
	log := logrus.New()
	log.SetFormatter(&customFormatter{})
	log.SetLevel(logrus.InfoLevel)
	return &defaultLogger{log: log}
}
```

改成：
```go
func newDefaultLogger() Logger {
	log := logrus.New()
	log.SetFormatter(&customFormatter{})
	log.SetLevel(logrus.InfoLevel)
	log.SetOutput(NewWriter())
	return &defaultLogger{log: log}
}
```

- [ ] **Step 6: 验证编译 + 测试**

```bash
go build ./backend/pkg/logs/...
go test ./backend/pkg/logs/...
```

Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add backend/pkg/logs/file_writer.go backend/pkg/logs/file_writer_test.go backend/pkg/logs/default.go
git commit -m "feat(logs): logrus dual-write to stdout + rolling file"
```

---

## Task 3: observability 通用包 — 注册 RED 和 Loop 业务指标

**Files:**
- Create: `backend/pkg/observability/metrics.go`
- Create: `backend/pkg/observability/metrics_test.go`

- [ ] **Step 1: 写测试**

写入 `backend/pkg/observability/metrics_test.go`：
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

func TestLoopTaskTotalIncrement(t *testing.T) {
	LoopTaskTotal.WithLabelValues("success").Inc()
	got := testutil.ToFloat64(LoopTaskTotal.WithLabelValues("success"))
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./backend/pkg/observability/ -v
```

Expected: FAIL — undefined

- [ ] **Step 3: 实现 metrics.go**

写入 `backend/pkg/observability/metrics.go`：
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

// Loop 业务指标
var (
	LoopTaskTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loop_task_total",
			Help: "Tasks counted by status: queued|running|success|failed",
		},
		[]string{"status"},
	)
	LoopTaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loop_task_duration_seconds",
			Help:    "Task wall-clock by stage",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1s → 400s
		},
		[]string{"stage"},
	)
	LoopEvaluatorInvocationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loop_evaluator_invocation_total",
			Help: "Evaluator calls by name",
		},
		[]string{"evaluator_name"},
	)
)
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./backend/pkg/observability/ -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/pkg/observability/metrics.go backend/pkg/observability/metrics_test.go
git commit -m "feat(observability): register RED + loop business metrics"
```

---

## Task 4: 实现 Hertz HTTP RED 中间件

**Files:**
- Create: `backend/pkg/observability/http_middleware.go`
- Create: `backend/pkg/observability/http_middleware_test.go`

- [ ] **Step 1: 写测试**

写入 `backend/pkg/observability/http_middleware_test.go`：
```go
package observability

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddleware_IncrementsCounter(t *testing.T) {
	mw := HTTPRequestsMiddleware()
	c := app.NewContext(0)
	c.Request.Header.SetMethod(consts.MethodGet)
	c.Request.SetRequestURI("/api/test")
	c.Response.SetStatusCode(http.StatusOK)

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))
	mw(context.Background(), c)
	after := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))

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
go test ./backend/pkg/observability/ -run TestHTTPMiddleware -v
```

Expected: FAIL

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
go test ./backend/pkg/observability/ -v
```

Expected: 4 个测试都 PASS

- [ ] **Step 5: 提交**

```bash
git add backend/pkg/observability/http_middleware.go backend/pkg/observability/http_middleware_test.go
git commit -m "feat(observability): hertz RED middleware for loop"
```

---

## Task 5: 注册 /metrics + 接 middleware

**Files:**
- Modify: `backend/cmd/main.go` 或 server 启动文件

- [ ] **Step 1: 找 server 启动点**

```bash
grep -n "server.Default\|server.New\|hertz.Default\|h.Use\|h.GET\|h.Spin" backend/cmd/main.go
```

- [ ] **Step 2: 加 middleware + /metrics**

在 `h := server.Default(...)` 之后、`h.Spin()` 之前加：
```go
import (
    "github.com/coze-dev/coze-loop/backend/pkg/observability"
    "github.com/hertz-contrib/adaptor"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

h.Use(observability.HTTPRequestsMiddleware())
h.GET("/metrics", adaptor.HertzHandler(promhttp.Handler()))
```

如需依赖：
```bash
go get github.com/hertz-contrib/adaptor
go mod tidy
```

- [ ] **Step 3: 启动 + curl /metrics**

```bash
# 启 backend，listen :8889
curl -sf http://localhost:8889/metrics | head -20
```

Expected: 输出 `# HELP http_requests_total ...` 等

- [ ] **Step 4: 提交**

```bash
git add backend/cmd/main.go go.mod go.sum
git commit -m "feat(api): expose /metrics + RED middleware on :8889"
```

---

## Task 6: 任务流水线 metrics 埋点

**Files:**
- 找 task scheduler/runner：`grep -rln "task.*Run\|task.*Execute\|task.*Schedule\|task.*Start" backend/`
- Modify: 找到的文件

- [ ] **Step 1: 定位 task 状态变更点**

通常 Loop 有几个文件类似 `backend/domain/task/scheduler.go`、`backend/application/task/runner.go`。

```bash
grep -rln "TaskStatus\|StatusRunning\|StatusSuccess\|StatusFailed" backend/ | head -5
```

- [ ] **Step 2: 在状态变更处加 counter**

类似地：
```go
import "github.com/coze-dev/coze-loop/backend/pkg/observability"

// 当任务变 running:
observability.LoopTaskTotal.WithLabelValues("running").Inc()
// 当任务完成:
observability.LoopTaskTotal.WithLabelValues("success").Inc() // or "failed"
```

- [ ] **Step 3: 在 task 执行的 wall-clock 处加 histogram**

如果任务有明显阶段（prepare/execute/evaluate），分阶段加：
```go
start := time.Now()
// prepare ...
observability.LoopTaskDuration.WithLabelValues("prepare").Observe(time.Since(start).Seconds())

start = time.Now()
// execute ...
observability.LoopTaskDuration.WithLabelValues("execute").Observe(time.Since(start).Seconds())
```

如果没有明确分阶段，整段算 `total`：
```go
defer func(start time.Time) {
    observability.LoopTaskDuration.WithLabelValues("total").Observe(time.Since(start).Seconds())
}(time.Now())
```

- [ ] **Step 4: 验证**

```bash
# 触发一次任务
# 然后看 metrics
curl -s http://localhost:8889/metrics | grep "loop_task"
```

Expected: counter 和 histogram 都有非零数

- [ ] **Step 5: 提交**

```bash
git add backend/<modified-files>.go
git commit -m "feat(task): record loop_task_total/duration_seconds"
```

---

## Task 7: Evaluator 调用计数

**Files:**
- 找 evaluator 调用入口：`grep -rln "evaluator\.Run\|Evaluator.*Invoke\|evaluator.*Execute" backend/`
- Modify: 找到的文件

- [ ] **Step 1: 定位**

```bash
grep -rn "Evaluator\|evaluator" backend/domain/ backend/application/ | grep -i "run\|invoke\|exec" | head -5
```

- [ ] **Step 2: 在调用入口加**

```go
import "github.com/coze-dev/coze-loop/backend/pkg/observability"

func (e *Evaluator) Run(ctx context.Context, ...) (...) {
    observability.LoopEvaluatorInvocationTotal.WithLabelValues(e.Name).Inc()
    // ... existing code
}
```

- [ ] **Step 3: 验证**

```bash
curl -s http://localhost:8889/metrics | grep "loop_evaluator_invocation_total"
```

Expected: 至少一个 evaluator_name 标签下非零

- [ ] **Step 4: 提交**

```bash
git add backend/<file>.go
git commit -m "feat(evaluator): record loop_evaluator_invocation_total"
```

---

## Task 8: 写 Loop Grafana 仪表盘

**Files:**
- Create: `~/projects/ynet/obs-stack/dashboards/loop.json`

- [ ] **Step 1: 写 dashboard JSON**

写入 `~/projects/ynet/obs-stack/dashboards/loop.json`：
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
    {"type": "stat", "title": "Loop Up", "gridPos": {"h": 4, "w": 4, "x": 0, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "up{service=\"loop\"}"}], "options": {"colorMode": "value"}, "fieldConfig": {"defaults": {"thresholds": {"steps": [{"value": 0, "color": "red"}, {"value": 1, "color": "green"}]}}}},
    {"type": "stat", "title": "QPS (5m)", "gridPos": {"h": 4, "w": 4, "x": 4, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(http_requests_total{service=\"loop\"}[5m]))"}]},
    {"type": "stat", "title": "P99 latency", "gridPos": {"h": 4, "w": 4, "x": 8, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"loop\"}[5m])))"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "stat", "title": "Error rate (5m)", "gridPos": {"h": 4, "w": 4, "x": 12, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(http_requests_total{service=\"loop\",status=~\"5..\"}[5m])) / sum(rate(http_requests_total{service=\"loop\"}[5m]))"}], "fieldConfig": {"defaults": {"unit": "percentunit"}}},
    {"type": "timeseries", "title": "Requests/sec by path", "gridPos": {"h": 8, "w": 12, "x": 0, "y": 4}, "datasource": "Prometheus", "targets": [{"expr": "topk(10, sum by (path) (rate(http_requests_total{service=\"loop\"}[5m])))", "legendFormat": "{{path}}"}]},
    {"type": "timeseries", "title": "Latency P50/P90/P99", "gridPos": {"h": 8, "w": 12, "x": 12, "y": 4}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.50, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"loop\"}[5m])))", "legendFormat": "p50"}, {"expr": "histogram_quantile(0.90, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"loop\"}[5m])))", "legendFormat": "p90"}, {"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"loop\"}[5m])))", "legendFormat": "p99"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "timeseries", "title": "Task pipeline (events/sec)", "gridPos": {"h": 8, "w": 12, "x": 0, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "sum by (status) (rate(loop_task_total[5m]))", "legendFormat": "{{status}}"}]},
    {"type": "timeseries", "title": "Task duration P95 by stage", "gridPos": {"h": 8, "w": 12, "x": 12, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.95, sum by (le, stage) (rate(loop_task_duration_seconds_bucket[5m])))", "legendFormat": "{{stage}}"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "timeseries", "title": "Evaluator invocations/sec", "gridPos": {"h": 8, "w": 24, "x": 0, "y": 20}, "datasource": "Prometheus", "targets": [{"expr": "topk(10, sum by (evaluator_name) (rate(loop_evaluator_invocation_total[5m])))", "legendFormat": "{{evaluator_name}}"}]}
  ],
  "refresh": "30s",
  "schemaVersion": 39,
  "tags": ["ynet", "loop"],
  "time": {"from": "now-1h", "to": "now"},
  "title": "Loop",
  "uid": "ynet-loop",
  "version": 1
}
```

- [ ] **Step 2: 验证 JSON**

```bash
cd ~/projects/ynet/obs-stack
python3 -m json.tool dashboards/loop.json > /dev/null && echo "OK"
```

- [ ] **Step 3: 提交**

```bash
cd ~/projects/ynet/obs-stack
git add dashboards/loop.json
git commit -m "feat(grafana): add Loop dashboard"
```

---

## Task 9: 更新 Loop docker-compose（端口 + 日志卷 + env）

**Files:**
- Modify: Loop 的 docker-compose 或 helm values

- [ ] **Step 1: 找部署配置**

```bash
cd ~/projects/ynet/coze-loop
find release/ docker/ -name "docker-compose*.yml" -o -name "values.yaml" 2>/dev/null
```

- [ ] **Step 2: 改端口映射 + 加 env + 加 volume**

如果是 docker-compose：
```yaml
services:
  loop-app:
    image: ynet/loop:latest
    ports:
      - "8889:8889"   # 从 8888:8888 改
    environment:
      LOG_FILE: /var/log/app/app.log
      LOG_LEVEL: info
      LOG_MAX_SIZE_MB: "100"
      LOG_MAX_BACKUPS: "7"
      LOG_MAX_AGE_DAYS: "30"
      LOG_COMPRESS: "true"
    volumes:
      - /data/logs/loop:/var/log/app
```

- [ ] **Step 3: 220 上建目录**

```bash
ssh dev@10.10.10.220 "sudo mkdir -p /data/logs/loop && sudo chown 1000:1000 /data/logs/loop"
```

- [ ] **Step 4: 重启 + 验证**

```bash
ssh dev@10.10.10.220 "cd /opt/loop && docker compose up -d"
sleep 10
curl -sf http://10.10.10.220:8889/metrics | head -3
ssh dev@10.10.10.220 "ls /data/logs/loop/"
```

Expected: metrics 有输出，`app.log` 文件存在

- [ ] **Step 5: 提交**

```bash
git add release/ docker/ # 实际改的文件
git commit -m "feat(deploy): loop on :8889 with /data/logs/loop volume + LOG env"
```

---

## Task 10: 部署验证

- [ ] **Step 1: Prometheus target up**

```bash
curl -sf http://10.10.10.220:9090/api/v1/targets | grep -o '"service":"loop","health":"up"'
```

Expected: 有匹配

- [ ] **Step 2: Grafana 仪表盘渲染**

浏览器开 `http://10.10.10.220:3000`，ynet 文件夹下的 "Loop" 仪表盘要有非零数据。

- [ ] **Step 3: 模拟流量 + 验证**

```bash
for i in {1..20}; do curl -sf http://10.10.10.220:8889/api/health; done
sleep 30
# QPS 出现尖峰
```

---

## Self-Review

1. **Spec coverage**:
   - ✅ 端口改 8889（Task 0）
   - ✅ /metrics endpoint（Task 5）
   - ✅ RED 中间件（Task 4-5）
   - ✅ loop_task_total / duration（Task 6）
   - ✅ loop_evaluator_invocation_total（Task 7）
   - ✅ 文件日志 + 滚动 + 6 env（Task 2）
   - ✅ docker-compose 端口/env/volume（Task 9）
   - ✅ Loop dashboard（Task 8）

2. **Placeholder scan**:
   - Task 6 / 7 用 grep 找落点合理（Loop 代码大）
   - 没有未定义符号

3. **Type/path consistency**:
   - 端口 8889 在所有相关位置
   - lumberjack 包路径一致
   - logrus + io.MultiWriter 用法一致

无遗漏。
