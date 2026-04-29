# Observability Stack — 4 项目统一指标 + 文件日志

> 范围：为 Studio / Loop / Guard-go / Intent Hub 4 个项目添加 Prometheus 指标 + 文件输出 + 日志滚动，并在 220 开发环境部署 Prometheus + Grafana + node_exporter 一套监控栈。
>
> 状态：design approved 2026-04-29，pending 落地

---

## 目标

1. **可观测性提升**：4 个项目都能在 Grafana 上看到 RED（请求量/错误率/延迟）+ 关键业务指标
2. **日志可归档**：4 个项目都能把日志写到本地文件，滚动 + 限存储不撑爆磁盘
3. **配置可调**：日志路径、滚动大小、保留份数、保留天数、压缩，全部 env 可调
4. **不打破现状**：保持各项目原有日志栈（Studio FullLogger / Loop logrus / Guard-go stdlib / Intent Hub Python logging），不强行换 zap / structlog
5. **零业务侵入**：埋点失败、文件写不进、metrics endpoint 故障都不能影响业务请求

## 非目标（明确排除）

- ❌ 224 生产 / CDRCB 银行环境部署（先 220 跑通再说）
- ❌ Alertmanager + 告警规则（等 SLO 定义清楚再加）
- ❌ Loki 日志中心（4 个项目日志体量小，文件直接看够用）
- ❌ OpenTelemetry trace（L3 范畴）
- ❌ 把 4 个项目的日志栈统一换成 zap/structlog（变更面太大）
- ❌ 中间件指标（DB / Redis / ES / OpenAI），有需要再加

---

## 整体架构（220 开发环境）

```
                    ┌─────── 10.10.10.220 ──────────────────┐
                    │                                          │
   docker host net  │                                          │   /data/logs/  (host volume)
   ┌────────────────┘                                          │   ├─ studio/
   │  Studio    :8888  GET /metrics  ──┐                       │   │   ├─ app.log (current)
   │  Loop      :TBD   GET /metrics  ──┤                       │   │   ├─ app.log.1.gz
   │  Guard-go  :8180  GET /metrics  ──┤◄── scrape (15s)       │   │   └─ ...
   │  Intent-Hub:8000  GET /metrics  ──┤                       │   ├─ loop/
   │                                   │                       │   ├─ guard-go/
   │  ┌──────── obs-stack (compose) ───┤                       │   └─ intent-hub/
   │  │  ├─ prometheus    :9090   ◄────┘                       │
   │  │  ├─ grafana       :3000   ──── UI                      │
   │  │  └─ node-exporter :9100   ──── 主机指标                │
   │  └────────────────────────────────────────────────────────┘
   └─────────────────────────────────────────────────────────────┘
```

**关键决策**：
- **Host network**：4 个 app + obs-stack 都 host net，scrape 走 `localhost:<port>`，避免 docker 跨网络发现复杂度
- **双写日志**：stdout 保留（`docker logs` 可用），同时落盘到 `/data/logs/<svc>/app.log`
- **Prometheus 数据保留 30 天**，单实例，无 HA（开发够用）
- **Grafana provisioning** via JSON：dashboards 入仓库，重启不丢

---

## Per-Project 改造（4 项目共性）

每个项目都做这 3 块：

### 1. `/metrics` endpoint

| 项目 | 实现方式 |
|---|---|
| Studio (Hertz) | 在 `backend/api/router/` 注册 `/metrics`，handler 用 `hertz adaptor.HertzHandler(promhttp.Handler())` |
| Loop (Hertz / Kitex 同框架) | 同 Studio |
| Guard-go (Gin + 标准 net/http mux) | `mux.Handle("/metrics", promhttp.Handler())`（参 `cmd/guard/main.go:88`，已经是 net/http mux） |
| Intent-Hub (FastAPI) | 装 `prometheus-fastapi-instrumentator`，`Instrumentator().instrument(app).expose(app)` |

**验收**：`curl localhost:<port>/metrics` 返回 200，含 `# HELP` 文本格式的指标

### 2. 指标埋点（L2: RED + 业务）

#### 通用 RED（HTTP middleware 层）

3 个指标，每请求自动加：

```
http_requests_total{method, path, status}                 [Counter]
http_request_duration_seconds{method, path, status}       [Histogram, buckets: 0.005,0.01,0.05,0.1,0.5,1,2.5,5,10]
http_requests_in_flight                                   [Gauge]
```

**path 标签需要 normalize**：避免高基数（如 `/users/123` → `/users/:id`）

#### Runtime（自动暴露）

- Go：`promauto` 注册时默认 `go_*`、`process_*`
- Python：`prometheus_client` 默认 `process_*` + 加 `multiprocess` collector（FastAPI 是 worker 模式）

#### 业务指标

| 项目 | 指标（Prefix 独立空间） |
|---|---|
| **Studio** | 保留已有 `knowledge_parse_*` 系列（`backend/domain/knowledge/service/metrics.go`）<br>新增：<br>• `studio_llm_tokens_total{model, kind="prompt"\|"completion"}` Counter<br>• `studio_file_upload_size_bytes{kind="image"\|"doc"}` Histogram<br>• `studio_agent_chat_total{result="success"\|"error"}` Counter |
| **Loop** | • `loop_task_total{status="queued"\|"running"\|"success"\|"failed"}` Counter<br>• `loop_task_duration_seconds{stage="prepare"\|"execute"\|"evaluate"}` Histogram<br>• `loop_evaluator_invocation_total{evaluator_name}` Counter |
| **Guard-go** | • `shield_check_total{source="blocklist"\|"knowledge_base"\|"ai_model", result="hit"\|"miss"}` Counter<br>• `shield_check_duration_seconds{source}` Histogram<br>• `shield_ai_tokens_total{model, kind}` Counter |
| **Intent Hub** | • `intent_classify_total{intent}` Counter<br>• `intent_classify_duration_seconds` Histogram<br>• `guard_preflight_total{result="pass"\|"block"}` Counter |

**验收**：发起 10 次相应业务请求，对应指标 `_total` 增加 10。

### 3. 文件日志 + 滚动

#### Go 三家：`gopkg.in/natefinch/lumberjack.v2`

每项目改一处 logger 初始化，把 `os.Stdout/os.Stderr` 包成 `io.MultiWriter(os.Stdout, lumberjackWriter)`：

```go
import "gopkg.in/natefinch/lumberjack.v2"

var lj io.Writer
if path := os.Getenv("LOG_FILE"); path != "" {
    lj = &lumberjack.Logger{
        Filename:   path,
        MaxSize:    cfg.MaxSizeMB,    // MB
        MaxBackups: cfg.MaxBackups,
        MaxAge:     cfg.MaxAgeDays,   // days
        Compress:   cfg.Compress,
    }
    out = io.MultiWriter(os.Stdout, lj)
} else {
    out = os.Stdout
}
log.SetOutput(out)  // 或 logger 各自的 SetOutput
```

具体落点：

| 项目 | 落点文件 |
|---|---|
| Studio | `backend/pkg/logs/default.go:29` `log.New(os.Stderr, ...)` 改为 `log.New(out, ...)` |
| Loop | `backend/pkg/logs/default.go:110` `logrus.New()` 后加 `log.SetOutput(out)` |
| Guard-go | `internal/middleware/logger.go:39` 把 `log.Printf` 改为通过预初始化的 logger（在 `cmd/guard/main.go` 里建好），同时新增 `internal/logging/init.go` 集中管理 |

#### Intent-Hub：Python `logging.handlers.RotatingFileHandler`

`backend/main.py` 启动时配置：

```python
import logging.handlers, os, sys

handlers = [logging.StreamHandler(sys.stdout)]
log_file = os.getenv("LOG_FILE")
if log_file:
    os.makedirs(os.path.dirname(log_file), exist_ok=True)
    handlers.append(logging.handlers.RotatingFileHandler(
        log_file,
        maxBytes=int(os.getenv("LOG_MAX_SIZE_MB", "100")) * 1024 * 1024,
        backupCount=int(os.getenv("LOG_MAX_BACKUPS", "7")),
        encoding="utf-8",
    ))
logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"), handlers=handlers,
                    format="%(asctime)s %(levelname)s [%(name)s] %(message)s")
```

> 注：Python `RotatingFileHandler` 不支持 `MaxAge`，只支持 `backupCount`。`LOG_MAX_AGE_DAYS` 在 Python 侧通过 cron / systemd timer 跑 `find /var/log/intent-hub -mtime +N -delete` 实现，或者切换到 `TimedRotatingFileHandler`。**默认实现：仅按 backupCount，不实现 MaxAge**（spec 后续补一行 cron 兜底）。

#### 6 个统一 env 变量（4 项目命名一致）

```
LOG_FILE                # 空 = 仅 stdout，不写文件；非空 = 双写
LOG_LEVEL               # debug / info / warn / error，默认 info
LOG_MAX_SIZE_MB         # 单文件最大 MB，默认 100
LOG_MAX_BACKUPS         # 保留旧文件数，默认 7
LOG_MAX_AGE_DAYS        # 保留天数（仅 Go 三家有效），默认 30
LOG_COMPRESS            # 旧文件 gzip，默认 true（仅 Go 三家有效）
```

**验收**：
1. 不设 `LOG_FILE` → 启动后只 stdout，无文件
2. 设 `LOG_FILE=/tmp/test.log` → 启动后产生文件，写入
3. 写满 `MaxSize` → 自动 rename `app.log.1.gz`，新 `app.log` 继续写
4. 超过 `MaxBackups` → 最老的被删

---

## Infra Stack（220 上的 obs-stack）

### 目录布局

```
/opt/obs-stack/
├── docker-compose.yml
├── prometheus.yml
├── datasources/
│   └── prometheus.yml          # Grafana 数据源 provisioning
├── dashboards/
│   ├── dashboards.yml          # Grafana provider config
│   ├── studio.json
│   ├── loop.json
│   ├── guard-go.json
│   ├── intent-hub.json
│   └── host-node.json
└── README.md                   # 启停 / 端口 / 告警预案
```

### docker-compose.yml

```yaml
services:
  prometheus:
    image: prom/prometheus:v2.55.0
    container_name: obs-prometheus
    network_mode: host
    restart: unless-stopped
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prom-data:/prometheus
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.retention.time=30d
      - --web.enable-lifecycle           # 允许热更新 reload

  grafana:
    image: grafana/grafana:11.3.0
    container_name: obs-grafana
    network_mode: host
    restart: unless-stopped
    volumes:
      - grafana-data:/var/lib/grafana
      - ./dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./datasources:/etc/grafana/provisioning/datasources:ro
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASS:-admin}
      GF_AUTH_ANONYMOUS_ENABLED: "false"

  node-exporter:
    image: prom/node-exporter:v1.8.2
    container_name: obs-node-exporter
    network_mode: host
    restart: unless-stopped
    pid: host
    volumes:
      - /:/host:ro,rslave
    command:
      - --path.rootfs=/host

volumes:
  prom-data:
  grafana-data:
```

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: studio
    static_configs:
      - targets: ["localhost:8888"]
        labels: { service: studio, env: dev-220 }

  - job_name: loop
    static_configs:
      - targets: ["localhost:LOOP_PORT_TBD"]
        labels: { service: loop, env: dev-220 }

  - job_name: guard-go
    static_configs:
      - targets: ["localhost:8180"]
        labels: { service: guard-go, env: dev-220 }

  - job_name: intent-hub
    static_configs:
      - targets: ["localhost:8000"]
        labels: { service: intent-hub, env: dev-220 }

  - job_name: node
    static_configs:
      - targets: ["localhost:9100"]
        labels: { service: host, env: dev-220 }
```

> Loop 端口在 220 上的实际映射要在实施时 `docker ps` 查一下补上。

### Grafana 数据源（datasources/prometheus.yml）

```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://localhost:9090
    isDefault: true
```

### Grafana dashboards 提供器（dashboards/dashboards.yml）

```yaml
apiVersion: 1
providers:
  - name: file-provisioned
    orgId: 1
    folder: ynet
    type: file
    disableDeletion: false
    editable: true
    options:
      path: /etc/grafana/provisioning/dashboards
```

---

## Dashboards 内容规范

每个 dashboard 划三个 row：

```
┌──── Row 1: 服务概览（4 个看板都一样）──────────────────────┐
│  [Stat] Up status     [Stat] QPS (5m)    [Stat] P99 latency │
│  [Stat] Error rate    [Stat] Goroutines / Threads           │
└──────────────────────────────────────────────────────────────┘
┌──── Row 2: RED 详情 ────────────────────────────────────────┐
│  [Graph] Requests / sec by path (top 10)                    │
│  [Graph] Latency P50/P90/P99 by path                        │
│  [Graph] Error rate by status code                          │
└──────────────────────────────────────────────────────────────┘
┌──── Row 3: 业务专属 ────────────────────────────────────────┐
│  Studio:  LLM tokens / file upload size / knowledge parsing │
│  Loop:    任务流水线 / evaluator                            │
│  Guard:   3 来源命中率 / 检测耗时 / AI tokens               │
│  Intent:  intent 分布 / preflight 通过率                    │
└──────────────────────────────────────────────────────────────┘
```

`host-node.json` 用 [Grafana 官方 dashboard 1860](https://grafana.com/grafana/dashboards/1860-node-exporter-full/) 直接 import，省事。

---

## 配置策略

### 部署侧（220）

每个 app 容器加 env：

```
LOG_FILE=/var/log/app/app.log
LOG_LEVEL=info
LOG_MAX_SIZE_MB=100
LOG_MAX_BACKUPS=7
LOG_MAX_AGE_DAYS=30
LOG_COMPRESS=true
METRICS_ENABLED=true        # 关闭可降级，但默认开
```

每个 app 容器加 volume：

```
- /data/logs/<service>:/var/log/app
```

### 代码侧（默认值）

代码读不到 env 时，全部走默认（同上面表）。`METRICS_ENABLED=false` 时不注册 `/metrics` endpoint，但中间件依然计数（开销极小，便于排障）。

---

## 错误处理

| 场景 | 行为 |
|---|---|
| `LOG_FILE` 路径不可写 | logger 初始化失败 → fallback 到只 stdout，启动时 `Errorf("log file init failed: %v")` |
| 滚动失败（磁盘满）| lumberjack 内部 `Write` 返回 error，被 logger 吃掉，**业务无感**；下次空间够会自动恢复 |
| `/metrics` 请求 panic | promhttp.Handler 自带 recover，不传染业务 |
| 业务埋点 panic | 用 defer-recover 包一层，仅 `log.Errorf("metric inc failed: %v")` |
| Prometheus scrape 超时 | 默认 10s timeout，超时显示 `up{} = 0`，看板显示 No Data，不影响 app |
| Grafana 启动失败 | 不影响 app 也不影响 Prometheus；检查 docker logs |

---

## 测试策略

### 每个项目（共 4 套）

#### 单元测试
- 1 个测试：`http_requests_total` counter 在请求后 +1（用 `httptest`）
- 1 个测试：lumberjack 写满 MaxSize 后产生 `.1` 文件（写 200MB 数据，MaxSize 设 100MB 验证）
- 1 个测试：`LOG_FILE=""` 时不创建文件，只 stdout

#### 集成 smoke
- 起 app → `curl localhost:<port>/metrics` → 应见 `http_requests_total`、`go_goroutines`/`process_*`

### 整套验收（联调）

1. 220 上启 obs-stack
2. 4 个 app 都跑起来（其中 Studio/Guard-go/Intent-Hub 已经 running，Loop 启之）
3. `curl http://220:9090/api/v1/targets` → 5 个 target 全 `health=up`
4. 浏览器开 `http://220:3000`，4 个 dashboard 都能渲染出非零数据
5. 给每个 app 发 10 次请求，对应 dashboard 的 `requests/sec` 出现尖峰
6. `ls /data/logs/<svc>/` 看到 `app.log` 文件并持续增长
7. （可选）压测一个 app 至 `MaxSize`，验证滚动产生 `.1.gz`

---

## 工作分解

| # | 任务 | 工作量 | 依赖 | 可并行 |
|---|---|---|---|---|
| T1 | Studio：埋点 + 文件日志 + lumberjack | 2.5 d | — | ✅ |
| T2 | Loop：埋点 + 文件日志 | 2 d | — | ✅ |
| T3 | Guard-go：埋点 + 文件日志（最简陋，要新增 `internal/logging/` 包） | 2.5 d | — | ✅ |
| T4 | Intent-Hub：埋点 + 文件日志 + FastAPI instrumentator | 1.5 d | — | ✅ |
| T5 | obs-stack（compose + prometheus.yml + datasource） | 1 d | — | ✅ |
| T6 | 4 个 dashboard JSON + host-node import | 1.5 d | T1-T4 任一完工可起 | 半并行 |
| T7 | 联调（4 项目 + obs-stack 全跑通）+ README | 1 d | T1-T6 全完 | ❌ |
| | **合计 12 人/天**，4 个 agent 并行实际 ~3-4 天 | | | |

每个 T1-T4 任务独立的项目独立的 worktree，T5/T6 也独立，并行执行没有冲突。

---

## 部署落地（220）

### 一次性

```bash
# 220 上
sudo mkdir -p /opt/obs-stack /data/logs/{studio,loop,guard-go,intent-hub}
# 把 obs-stack 仓库 clone 或 scp 到 /opt/obs-stack
cd /opt/obs-stack
docker compose up -d
# 检查
curl http://localhost:9090/-/ready    # Prometheus ready
curl http://localhost:3000/api/health # Grafana ok
```

### 每个 app 升级

把 `LOG_FILE` 等 env 加到 docker-compose，`/data/logs/<svc>` mount 进容器，重启即可。

---

## 开放问题（实施时确认）

1. **Loop 在 220 上实际监听端口**——配置文件里默认 8888，但和 Studio 冲突，220 上可能映射到 8889 / 8890。`docker ps` 查实际值。
2. **Studio 现有 `knowledge_parse_*` 指标是否已暴露 `/metrics`** —— 代码注册了 promauto 但没看到 mux 注册，需要补 endpoint
3. **Intent-Hub 的 `prometheus-fastapi-instrumentator` 依赖** —— 加到 `pyproject.toml`，确认不与现有依赖冲突
4. **Loop 端口若已被占用，是否调整**——最坏情况让 Loop 跑 8889
5. **Grafana admin 默认密码**——environment 留 `${GRAFANA_ADMIN_PASS:-admin}`，部署时给一个强密码

---

## 后续扩展（明确 out-of-scope）

- v2：Alertmanager + 告警规则（Lark webhook 集成）
- v2：copy 这套到 224 生产
- v2：Loki + Promtail，把 `/data/logs/` 接进来
- v3：CDRCB 离线包（要求银行那边 docker 镜像离线导入）
- v3：OpenTelemetry trace（jaeger / tempo）
