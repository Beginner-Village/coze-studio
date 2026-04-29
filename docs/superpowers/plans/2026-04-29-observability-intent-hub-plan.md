# Observability — Intent Hub Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 Intent Hub (`~/projects/ynet/intent-hub`) 添加 `/metrics` endpoint、RED + 业务指标（intent 分类、guard preflight）、文件日志 + 滚动。Intent Hub 是 Python/FastAPI 项目，使用 `prometheus-fastapi-instrumentator` 自动埋点 RED + 暴露 `/metrics`，业务指标手工埋。日志用 stdlib `logging.handlers.RotatingFileHandler`。

**Architecture:** FastAPI 启动时调用 `Instrumentator().instrument(app).expose(app)` 一键挂 `/metrics` 和 RED，业务指标用 `prometheus_client.Counter/Histogram` 在 nodes 关键位置 inc/observe。日志在 `backend/main.py` 启动期初始化 root logger，stdout + RotatingFileHandler 双 handler，env 配置。

**Tech Stack:** Python 3.11+, FastAPI, prometheus-fastapi-instrumentator, prometheus-client, logging stdlib

**Spec:** [docs/superpowers/specs/2026-04-29-observability-stack-design.md](../specs/2026-04-29-observability-stack-design.md)

---

## File Structure

```
backend/
├── main.py                                  # MODIFY: 启动期 init logging + Instrumentator
├── core/
│   └── observability.py                     # NEW: 业务指标 Counter/Histogram 定义
├── core/
│   └── logging_config.py                    # NEW: 集中初始化 root logger
└── engine/nodes/
    ├── intent_router.py                     # MODIFY: intent_classify_* 埋点
    ├── guard_preflight.py                   # MODIFY: guard_preflight_* 埋点
    └── ...

~/projects/ynet/obs-stack/
└── dashboards/intent-hub.json               # NEW
```

---

## Task 1: 添加依赖

**Files:**
- Modify: `pyproject.toml`

- [ ] **Step 1: 添加依赖**

打开 `pyproject.toml`，在 `[project]` 或 `[tool.poetry.dependencies]` 节里加：
```toml
"prometheus-fastapi-instrumentator>=7.0.0",
"prometheus-client>=0.20.0",
```

- [ ] **Step 2: 安装**

```bash
cd ~/projects/ynet/intent-hub
source .venv/bin/activate
pip install -e .
```

或（如果用 uv/poetry）：
```bash
uv pip install -e .
```

Expected: 安装成功

- [ ] **Step 3: 验证 import**

```bash
python -c "from prometheus_fastapi_instrumentator import Instrumentator; from prometheus_client import Counter; print('ok')"
```

Expected: `ok`

- [ ] **Step 4: 提交**

```bash
git add pyproject.toml
git commit -m "feat(deps): add prometheus-fastapi-instrumentator and prometheus-client"
```

---

## Task 2: 集中日志初始化

**Files:**
- Create: `backend/core/logging_config.py`
- Create: `tests/core/test_logging_config.py`

- [ ] **Step 1: 写测试**

写入 `tests/core/test_logging_config.py`：
```python
import logging
import os
import tempfile
from pathlib import Path

import pytest


def test_init_logging_no_file_when_log_file_empty(monkeypatch):
    monkeypatch.setenv("LOG_FILE", "")

    from backend.core.logging_config import init_logging

    init_logging(reset=True)
    root = logging.getLogger()
    file_handlers = [h for h in root.handlers if isinstance(h, logging.handlers.RotatingFileHandler)]
    assert len(file_handlers) == 0


def test_init_logging_creates_file_when_log_file_set(monkeypatch, tmp_path):
    log_path = tmp_path / "app.log"
    monkeypatch.setenv("LOG_FILE", str(log_path))
    monkeypatch.setenv("LOG_MAX_SIZE_MB", "1")
    monkeypatch.setenv("LOG_MAX_BACKUPS", "2")

    from backend.core.logging_config import init_logging

    init_logging(reset=True)
    logging.getLogger().info("hello")

    assert log_path.exists(), f"expected log file {log_path}"


def test_log_level_filter(monkeypatch, capsys):
    monkeypatch.setenv("LOG_LEVEL", "WARN")
    monkeypatch.setenv("LOG_FILE", "")

    from backend.core.logging_config import init_logging

    init_logging(reset=True)
    log = logging.getLogger("test")
    log.debug("debug-msg")
    log.info("info-msg")
    log.warning("warn-msg")
    log.error("error-msg")

    captured = capsys.readouterr()
    out = captured.out + captured.err
    assert "debug-msg" not in out
    assert "info-msg" not in out
    assert "warn-msg" in out
    assert "error-msg" in out
```

- [ ] **Step 2: 跑测试看失败**

```bash
cd ~/projects/ynet/intent-hub
pytest tests/core/test_logging_config.py -v
```

Expected: FAIL — `backend.core.logging_config` does not exist

- [ ] **Step 3: 实现 logging_config.py**

写入 `backend/core/logging_config.py`：
```python
"""集中初始化 root logger：stdout + 可选 RotatingFileHandler。

环境变量：
  LOG_FILE            空字符串 = 仅 stdout；非空 = 双写
  LOG_LEVEL           debug/info/warn/error，默认 info
  LOG_MAX_SIZE_MB     单文件 MB，默认 100
  LOG_MAX_BACKUPS     保留旧文件数，默认 7
  LOG_MAX_AGE_DAYS    （Python RotatingFileHandler 不支持，仅 backupCount）
                       超出 backupCount 自动删；如需按天清理，cron 跑
                       find /var/log/intent-hub -mtime +N -delete
  LOG_COMPRESS        （RotatingFileHandler 不支持开箱压缩，留作扩展）
"""
import logging
import logging.handlers
import os
import sys
from pathlib import Path

_INITIALIZED = False


def _level_from_env() -> int:
    name = os.getenv("LOG_LEVEL", "info").upper()
    return {
        "DEBUG": logging.DEBUG,
        "INFO": logging.INFO,
        "WARN": logging.WARNING,
        "WARNING": logging.WARNING,
        "ERROR": logging.ERROR,
        "FATAL": logging.CRITICAL,
        "CRITICAL": logging.CRITICAL,
    }.get(name, logging.INFO)


def _int_env(key: str, default: int) -> int:
    v = os.getenv(key)
    if not v:
        return default
    try:
        return int(v)
    except ValueError:
        return default


def init_logging(*, reset: bool = False) -> None:
    """Initialize the root logger from env. Idempotent unless reset=True.

    Call once at app startup. reset=True is for tests.
    """
    global _INITIALIZED
    if _INITIALIZED and not reset:
        return

    root = logging.getLogger()
    if reset:
        for h in list(root.handlers):
            root.removeHandler(h)

    root.setLevel(_level_from_env())

    fmt = logging.Formatter(
        "%(asctime)s %(levelname)s [%(name)s] %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )

    sh = logging.StreamHandler(sys.stdout)
    sh.setFormatter(fmt)
    root.addHandler(sh)

    log_file = os.getenv("LOG_FILE", "")
    if log_file:
        Path(log_file).parent.mkdir(parents=True, exist_ok=True)
        max_bytes = _int_env("LOG_MAX_SIZE_MB", 100) * 1024 * 1024
        backups = _int_env("LOG_MAX_BACKUPS", 7)
        fh = logging.handlers.RotatingFileHandler(
            log_file,
            maxBytes=max_bytes,
            backupCount=backups,
            encoding="utf-8",
        )
        fh.setFormatter(fmt)
        root.addHandler(fh)

    _INITIALIZED = True
```

- [ ] **Step 4: 跑测试看通过**

```bash
pytest tests/core/test_logging_config.py -v
```

Expected: 3 个 PASS

- [ ] **Step 5: 提交**

```bash
git add backend/core/logging_config.py tests/core/test_logging_config.py
git commit -m "feat(logging): centralized logging init with optional rotating file"
```

---

## Task 3: 业务指标定义

**Files:**
- Create: `backend/core/observability.py`
- Create: `tests/core/test_observability.py`

- [ ] **Step 1: 写测试**

写入 `tests/core/test_observability.py`：
```python
def test_intent_classify_total_increment():
    from backend.core.observability import INTENT_CLASSIFY_TOTAL

    INTENT_CLASSIFY_TOTAL.labels(intent="loan_inquiry").inc()
    val = INTENT_CLASSIFY_TOTAL.labels(intent="loan_inquiry")._value.get()
    assert val >= 1


def test_guard_preflight_total_increment():
    from backend.core.observability import GUARD_PREFLIGHT_TOTAL

    GUARD_PREFLIGHT_TOTAL.labels(result="pass").inc()
    val = GUARD_PREFLIGHT_TOTAL.labels(result="pass")._value.get()
    assert val >= 1


def test_intent_classify_duration_observed():
    from backend.core.observability import INTENT_CLASSIFY_DURATION

    INTENT_CLASSIFY_DURATION.observe(0.5)
    # 不报错即可（_sum 累加）
```

- [ ] **Step 2: 跑测试看失败**

```bash
pytest tests/core/test_observability.py -v
```

Expected: FAIL — `backend.core.observability` does not exist

- [ ] **Step 3: 实现 observability.py**

写入 `backend/core/observability.py`：
```python
"""Intent Hub 业务指标。RED 由 prometheus-fastapi-instrumentator 自动注册。"""
from prometheus_client import Counter, Histogram

INTENT_CLASSIFY_TOTAL = Counter(
    "intent_classify_total",
    "Intent classifications counted by intent label",
    ["intent"],
)

INTENT_CLASSIFY_DURATION = Histogram(
    "intent_classify_duration_seconds",
    "Wall-clock time of intent classification",
    buckets=(0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
)

GUARD_PREFLIGHT_TOTAL = Counter(
    "guard_preflight_total",
    "Guard preflight outcomes (pass|block|error)",
    ["result"],
)
```

- [ ] **Step 4: 跑测试看通过**

```bash
pytest tests/core/test_observability.py -v
```

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/core/observability.py tests/core/test_observability.py
git commit -m "feat(observability): define intent_classify and guard_preflight metrics"
```

---

## Task 4: 在 main.py 启用 logging 和 Instrumentator

**Files:**
- Modify: `backend/main.py`

- [ ] **Step 1: 改 main.py**

找到 FastAPI app 创建处：
```python
from fastapi import FastAPI
app = FastAPI(...)
```

在 app 创建后立刻加：
```python
from backend.core.logging_config import init_logging
from prometheus_fastapi_instrumentator import Instrumentator

init_logging()  # 在最早一刻初始化日志

# Instrumentator 自动注册 http_requests_total / http_request_duration_seconds
# 它默认指标名带 prefix，下面参数让它输出标准的 http_* 名（与 Studio/Loop/Guard-go 对齐）
Instrumentator(
    should_group_status_codes=False,
    should_ignore_untemplated=False,
    should_respect_env_var=False,
    should_instrument_requests_inprogress=True,
    excluded_handlers=["/metrics"],
    inprogress_name="http_requests_in_flight",
    inprogress_labels=False,
).instrument(app).expose(app, endpoint="/metrics")
```

- [ ] **Step 2: 启动 + curl /metrics**

```bash
cd ~/projects/ynet/intent-hub
LOG_LEVEL=info uvicorn backend.main:app --host 127.0.0.1 --port 8000 &
sleep 3
curl -sf http://localhost:8000/metrics | head -20
```

Expected: 输出 prometheus 文本格式，含 `http_request_duration_seconds_bucket`、`http_requests_in_flight` 等

- [ ] **Step 3: 验证日志生效**

```bash
LOG_FILE=/tmp/intent-hub.log LOG_LEVEL=info uvicorn backend.main:app --host 127.0.0.1 --port 8000 &
sleep 3
ls -la /tmp/intent-hub.log
cat /tmp/intent-hub.log | head -3
kill %1
```

Expected: 文件存在且有 `INFO` 级别日志

- [ ] **Step 4: 提交**

```bash
git add backend/main.py
git commit -m "feat(main): init logging and Prometheus Instrumentator at startup"
```

---

## Task 5: intent_classify 埋点

**Files:**
- Modify: `backend/engine/nodes/intent_router.py` 或类似

- [ ] **Step 1: 找 intent 分类节点**

```bash
cd ~/projects/ynet/intent-hub
grep -rln "intent_router\|classify_intent\|IntentRouter" backend/engine/ | head -5
```

通常是 `backend/engine/nodes/intent_router.py` 之类。

- [ ] **Step 2: 在分类完成处埋点**

```python
import time
from backend.core.observability import INTENT_CLASSIFY_TOTAL, INTENT_CLASSIFY_DURATION


# 在分类函数里:
def classify_intent(state):
    start = time.time()
    intent = ...  # existing classification logic
    INTENT_CLASSIFY_DURATION.observe(time.time() - start)
    INTENT_CLASSIFY_TOTAL.labels(intent=intent or "unknown").inc()
    return intent
```

- [ ] **Step 3: 验证**

```bash
LOG_LEVEL=info uvicorn backend.main:app --host 127.0.0.1 --port 8000 &
sleep 3

# 触发一次 chat (用项目里现有 e2e 测试或手工调用)
curl -X POST http://localhost:8000/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"我想咨询贷款","session_id":"test-1"}'

curl -s http://localhost:8000/metrics | grep "intent_classify"
```

Expected: 至少一个 intent label 值非零

- [ ] **Step 4: 提交**

```bash
git add backend/engine/nodes/intent_router.py
git commit -m "feat(intent): record intent_classify_total/duration"
```

---

## Task 6: guard_preflight 埋点

**Files:**
- Modify: `backend/engine/nodes/guard_preflight.py`

- [ ] **Step 1: 找 guard preflight 调用结果**

```bash
cd ~/projects/ynet/intent-hub
cat backend/engine/nodes/guard_preflight.py | head -50
```

通常会有 `passed/blocked` 类标志。

- [ ] **Step 2: 在 preflight 结果处埋点**

```python
from backend.core.observability import GUARD_PREFLIGHT_TOTAL


# 在已有的 preflight check 之后:
result = "pass"
if guard_response.action == "block":
    result = "block"
elif guard_response is None:  # 调用错误
    result = "error"

GUARD_PREFLIGHT_TOTAL.labels(result=result).inc()
```

- [ ] **Step 3: 验证**

```bash
curl -s http://localhost:8000/metrics | grep "guard_preflight"
```

Expected: 至少一个 result 标签值非零

- [ ] **Step 4: 提交**

```bash
git add backend/engine/nodes/guard_preflight.py
git commit -m "feat(guard): record guard_preflight_total by result"
```

---

## Task 7: 写 Intent Hub Grafana 仪表盘

**Files:**
- Create: `~/projects/ynet/obs-stack/dashboards/intent-hub.json`

- [ ] **Step 1: 写 dashboard JSON**

写入 `~/projects/ynet/obs-stack/dashboards/intent-hub.json`：
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
    {"type": "stat", "title": "Intent-Hub Up", "gridPos": {"h": 4, "w": 4, "x": 0, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "up{service=\"intent-hub\"}"}], "options": {"colorMode": "value"}, "fieldConfig": {"defaults": {"thresholds": {"steps": [{"value": 0, "color": "red"}, {"value": 1, "color": "green"}]}}}},
    {"type": "stat", "title": "QPS (5m)", "gridPos": {"h": 4, "w": 4, "x": 4, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(http_request_duration_seconds_count{service=\"intent-hub\"}[5m]))"}]},
    {"type": "stat", "title": "P99 latency", "gridPos": {"h": 4, "w": 4, "x": 8, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"intent-hub\"}[5m])))"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "stat", "title": "Error rate", "gridPos": {"h": 4, "w": 4, "x": 12, "y": 0}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(http_request_duration_seconds_count{service=\"intent-hub\",status=~\"5..\"}[5m])) / sum(rate(http_request_duration_seconds_count{service=\"intent-hub\"}[5m]))"}], "fieldConfig": {"defaults": {"unit": "percentunit"}}},
    {"type": "timeseries", "title": "Requests/sec by handler", "gridPos": {"h": 8, "w": 12, "x": 0, "y": 4}, "datasource": "Prometheus", "targets": [{"expr": "topk(10, sum by (handler) (rate(http_request_duration_seconds_count{service=\"intent-hub\"}[5m])))", "legendFormat": "{{handler}}"}]},
    {"type": "timeseries", "title": "Latency P50/P90/P99", "gridPos": {"h": 8, "w": 12, "x": 12, "y": 4}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.50, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"intent-hub\"}[5m])))", "legendFormat": "p50"}, {"expr": "histogram_quantile(0.90, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"intent-hub\"}[5m])))", "legendFormat": "p90"}, {"expr": "histogram_quantile(0.99, sum by (le) (rate(http_request_duration_seconds_bucket{service=\"intent-hub\"}[5m])))", "legendFormat": "p99"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "piechart", "title": "Intent distribution (1h)", "gridPos": {"h": 8, "w": 12, "x": 0, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "sum by (intent) (increase(intent_classify_total[1h]))", "legendFormat": "{{intent}}"}]},
    {"type": "timeseries", "title": "Intent classify duration P95", "gridPos": {"h": 8, "w": 12, "x": 12, "y": 12}, "datasource": "Prometheus", "targets": [{"expr": "histogram_quantile(0.95, sum by (le) (rate(intent_classify_duration_seconds_bucket[5m])))", "legendFormat": "p95"}], "fieldConfig": {"defaults": {"unit": "s"}}},
    {"type": "timeseries", "title": "Guard preflight pass-rate", "gridPos": {"h": 8, "w": 24, "x": 0, "y": 20}, "datasource": "Prometheus", "targets": [{"expr": "sum(rate(guard_preflight_total{result=\"pass\"}[5m])) / sum(rate(guard_preflight_total[5m]))", "legendFormat": "pass rate"}, {"expr": "sum by (result) (rate(guard_preflight_total[5m]))", "legendFormat": "{{result}}"}]}
  ],
  "refresh": "30s",
  "schemaVersion": 39,
  "tags": ["ynet", "intent-hub"],
  "time": {"from": "now-1h", "to": "now"},
  "title": "Intent Hub",
  "uid": "ynet-intent-hub",
  "version": 1
}
```

> 注：Instrumentator 默认输出的 RED metric 标签是 `handler` 而不是 `path`，所以 dashboard 用 `handler`。

- [ ] **Step 2: 验证 JSON**

```bash
cd ~/projects/ynet/obs-stack
python3 -m json.tool dashboards/intent-hub.json > /dev/null && echo "OK"
```

- [ ] **Step 3: 提交**

```bash
cd ~/projects/ynet/obs-stack
git add dashboards/intent-hub.json
git commit -m "feat(grafana): add Intent Hub dashboard"
```

---

## Task 8: 更新部署脚本（日志 env + 卷）

**Files:**
- Modify: `scripts/start.sh` 或 `docker-compose.yml`（按 Intent Hub 当前部署方式）

- [ ] **Step 1: 找当前启动方式**

```bash
cd ~/projects/ynet/intent-hub
ls scripts/
find . -maxdepth 3 -name "docker-compose*.yml" -o -name "Dockerfile" 2>/dev/null
```

- [ ] **Step 2: 加 env 变量**

如果是 docker-compose：
```yaml
services:
  intent-hub:
    image: ynet/intent-hub:latest
    environment:
      LOG_FILE: /var/log/app/app.log
      LOG_LEVEL: info
      LOG_MAX_SIZE_MB: "100"
      LOG_MAX_BACKUPS: "7"
      LOG_MAX_AGE_DAYS: "30"
    volumes:
      - /data/logs/intent-hub:/var/log/app
    ports:
      - "8000:8000"
```

如果是 systemd / 直接 uvicorn 启动，加 env 到 EnvironmentFile 或 launch script。

- [ ] **Step 3: 220 上建目录**

```bash
ssh dev@10.10.10.220 "sudo mkdir -p /data/logs/intent-hub && sudo chown 1000:1000 /data/logs/intent-hub"
```

- [ ] **Step 4: 验证**

```bash
ssh dev@10.10.10.220 "cd /opt/intent-hub && docker compose up -d"
sleep 10
curl -sf http://10.10.10.220:8000/metrics | head -3
ssh dev@10.10.10.220 "ls /data/logs/intent-hub/"
```

- [ ] **Step 5: 提交**

```bash
git add docker-compose.yml scripts/start.sh
git commit -m "feat(deploy): mount /data/logs/intent-hub + LOG env vars"
```

---

## Task 9: 部署验证

- [ ] **Step 1: target up**

```bash
curl -sf http://10.10.10.220:9090/api/v1/targets | grep -o '"service":"intent-hub","health":"up"'
```

- [ ] **Step 2: Grafana 仪表盘**

浏览器开 `http://10.10.10.220:3000`，"Intent Hub" 仪表盘要有 RED + intent + guard 数据。

- [ ] **Step 3: 模拟流量**

```bash
for i in {1..10}; do
  curl -s -X POST http://10.10.10.220:8000/api/v1/chat \
    -H 'Content-Type: application/json' \
    -d '{"message":"test","session_id":"smoke-'$i'"}'
done
sleep 30
```

刷新仪表盘，QPS 应有尖峰，intent 分布饼图应有切片。

---

## Self-Review

1. **Spec coverage**:
   - ✅ /metrics endpoint via Instrumentator（Task 4）
   - ✅ RED 指标自动（Task 4，Instrumentator 提供）
   - ✅ intent_classify_total / duration（Task 5）
   - ✅ guard_preflight_total（Task 6）
   - ✅ 文件日志 + 滚动 + 6 env（Task 2，注意 LOG_MAX_AGE_DAYS / LOG_COMPRESS Python 不生效，spec 已说明）
   - ✅ docker-compose env + volume（Task 8）
   - ✅ Intent Hub dashboard（Task 7）

2. **Placeholder scan**:
   - Task 5 / 6 通过 grep 找节点位置合理
   - Task 8 取决于实际部署方式（docker-compose / systemd），plan 给了两套指引

3. **Type/path consistency**:
   - 端口 8000 与 prometheus.yml 一致
   - metric 名（`intent_classify_total`、`guard_preflight_total`）与 dashboard query 一致
   - `init_logging(reset=False)` 默认幂等

4. **Python 特殊点**：
   - Instrumentator 默认 RED 用 `handler` 标签而非 `path`，dashboard 已对齐
   - `RotatingFileHandler` 不支持 MaxAge / Compress，spec 已说明用 cron 兜底（不在 plan 范围）

无遗漏。
