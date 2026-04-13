# Deploy-v2 全量部署自动化 — 整合今日踩坑修复

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将今天修复的所有问题整合到 deploy-v2 部署工具中，实现从零一键部署并全量验证

**Architecture:** 更新 deploy-v2 的 docker-compose 模板、observability 配置模板、init.sh 初始化脚本，然后构建最终镜像推送 Harbor，在 220 上从零全量部署测试

**Tech Stack:** Docker Compose, Bash, Go cross-compile, Harbor, OceanBase, RocketMQ, ClickHouse, Playwright MCP

---

### Task 1: 更新 Studio docker-compose.yml — 加入 Loop Session Key

**Files:**
- Modify: `ynet-docker/deploy-v2/studio/docker-compose.yml:74-80`
- Modify: `ynet-docker/deploy-v2/.env.example:33`

**问题背景：** Studio 代理 Loop 的可观测性 API 时，需要注入 session cookie 绕过 Loop 的 session 认证

- [ ] **Step 1: 在 Studio compose 中加入 LOOP_SESSION_KEY 和 LOOP_HOST**

在 `ynet-docker/deploy-v2/studio/docker-compose.yml` 的 Loop 集成区域，确认以下配置已存在（当前版本已有 `LOOP_HOST`/`LOOP_PORT`）：
```yaml
      # Loop 集成
      - YNET_LOOP_PROXY_URL=http://${LOOP_HOST:-10.10.10.220}:${LOOP_PORT:-8888}
      - YNET_LOOP_TELEMETRY_ENABLE=1
      - YNET_LOOP_TELEMETRY_ENDPOINT=http://${LOOP_HOST:-10.10.10.220}:${LOOP_PORT:-8888}/v1/loop/opentelemetry/v1/traces
      - YNET_LOOP_TELEMETRY_TOKEN=${LOOP_TOKEN:-}
      - YNET_LOOP_TRACE_RATIO=1.0
      - YNET_LOOP_WORKSPACE_ID=${LOOP_WORKSPACE_ID:-}
      - YNET_LOOP_SESSION_KEY=${LOOP_SESSION_KEY:-}
```

- [ ] **Step 2: 更新 .env.example 加入新变量**

在 `ynet-docker/deploy-v2/.env.example` 的 Studio 区域后加入 Loop 集成变量：
```bash
# ---------- Loop 集成（由 middleware/init.sh loop-token 自动生成） ----------
LOOP_HOST=10.10.10.220
LOOP_PORT=8888
LOOP_TOKEN=
LOOP_WORKSPACE_ID=
LOOP_SESSION_KEY=
```

- [ ] **Step 3: 验证 compose 文件语法**

Run: `cd ynet-docker/deploy-v2/studio && docker compose config 2>&1 | head -5`
Expected: 正常输出 YAML 格式

---

### Task 2: 创建 Loop observability.yaml 模板

**Files:**
- Create: `ynet-docker/deploy-v2/loop/observability.yaml.tpl`
- Modify: `ynet-docker/deploy-v2/loop/docker-compose.yml`

**问题背景：** Loop 的 observability.yaml 中 RMQ/ClickHouse 地址使用 Docker 内部 hostname，分机部署不可达。且缺少 `cozeloop` platform tenant 配置导致可观测性查询失败。

- [ ] **Step 1: 创建 observability.yaml 模板**

创建 `ynet-docker/deploy-v2/loop/observability.yaml.tpl`，用 `__MW_HOST__` 占位符代替硬编码地址，包含 `cozeloop` tenant 配置。

关键内容：
```yaml
trace_platform_tenants:
  config:
    cozeloop:
      - "ynetloop"
    ynetloop:
      - "ynetloop"
    # ... 其他 tenants

trace_tenant_cfg:
  tenant_table:
    cozeloop:
      365d:
        span_table: "observability_spans"
    ynetloop:
      365d:
        span_table: "observability_spans"

trace_collector_cfg:
  receivers:
    rmq/default:
      addr:
        - "__MW_HOST__:9876"      # ← 占位符
  exporters:
    clickhouse/default:            # ← 使用环境变量配置的 CK
```

- [ ] **Step 2: 在 Loop docker-compose.yml 中挂载 observability.yaml**

修改 `ynet-docker/deploy-v2/loop/docker-compose.yml`：
```yaml
  ynet-loop-app:
    volumes:
      - ./observability.yaml:/ynet-loop/conf/observability.yaml:ro
```

- [ ] **Step 3: 在 deploy.sh 或 init.sh 中添加模板渲染逻辑**

部署 Loop 前，用 `sed` 替换模板中的 `__MW_HOST__`：
```bash
sed "s/__MW_HOST__/$MW_HOST/g" loop/observability.yaml.tpl > loop/observability.yaml
```

---

### Task 3: 更新 init.sh — Loop token 自动设置 session_key

**Files:**
- Modify: `ynet-docker/deploy-v2/middleware/init.sh:328-368`

**问题背景：** `init_loop_token()` 创建 PAT token 和获取 workspace_id，但没有同时获取 session_key 用于 Studio 代理可观测性 API。

- [ ] **Step 1: 在 init_loop_token 中提取 session_key**

在 `init_loop_token()` 函数中，注册用户后从 DB 读取 session_key：
```bash
  local session_key=$(run_mysql "ynet-loop" "SELECT session_key FROM user WHERE id=$user_id AND deleted_at=0 LIMIT 1;" | tr -d '[:space:]')

  echo "  YNET_LOOP_SESSION_KEY=$session_key"
```

- [ ] **Step 2: 同样更新 deploy.sh 中的 do_setup_loop_token**

在 `deploy.sh` 的 `do_setup_loop_token()` 中也读取 session_key 并保存到 `LOOP_SESSION_KEY`。

---

### Task 4: 构建最终版 Studio 镜像推送 Harbor

**Files:**
- Binary: `/tmp/openynet-server` (已编译好的最终版)

**前提：** Docker Desktop 已运行

- [ ] **Step 1: 确认 Docker 可用**

Run: `docker info >/dev/null 2>&1 && echo OK`

- [ ] **Step 2: 构建并推送 Studio 镜像**

```bash
cd /tmp/studio-build
docker buildx build --platform linux/amd64 \
  -f Dockerfile.patch \
  -t 10.10.10.206:8090/ynet-studio/ynet-server:latest \
  --push .
```

- [ ] **Step 3: 验证镜像已推送**

```bash
curl -s http://10.10.10.206:8090/v2/ynet-studio/ynet-server/tags/list
```

---

### Task 5: 构建最终版 Loop 镜像推送 Harbor

**Files:**
- Binary: `/tmp/ynet-loop` (已编译好的最终版)

**问题：** Loop 没有 Dockerfile.patch，需要创建

- [ ] **Step 1: 获取当前 Loop 镜像名**

从 220 上查看当前 Loop 镜像：
```bash
ssh dev@220 "docker inspect ynet-loop-app --format '{{.Config.Image}}'"
```

- [ ] **Step 2: 创建 Loop Dockerfile.patch 并构建推送**

```dockerfile
FROM <current-loop-image>
COPY ynet-loop /ynet-loop/bin/main
```

```bash
docker buildx build --platform linux/amd64 \
  -f Dockerfile.patch \
  -t 10.10.10.206:8090/ynet-loop/app:latest \
  --push .
```

---

### Task 6: 220 全量清理

**目标：** 清理 220 上所有旧容器和数据，准备从零部署

- [ ] **Step 1: 停止并删除所有 ynet 容器**

```bash
ssh dev@220 "
  docker stop ynet-server ynet-web ynet-loop-app ynet-loop-nginx guard-app 2>/dev/null
  docker rm ynet-server ynet-web ynet-loop-app ynet-loop-nginx guard-app 2>/dev/null
  echo 'CLEAN DONE'
"
```

- [ ] **Step 2: 清理旧镜像**

```bash
ssh dev@220 "docker image prune -f"
```

---

### Task 7: 中间件初始化验证

**目标：** 验证 226 上的中间件都在运行且数据可达

- [ ] **Step 1: 验证所有中间件连接**

```bash
ssh dev@220 "
  # Redis
  nc -z 10.10.10.226 6379 && echo 'Redis OK' || echo 'Redis FAIL'
  # ES
  curl -sf http://10.10.10.226:9200 >/dev/null && echo 'ES OK' || echo 'ES FAIL'
  # ClickHouse
  nc -z 10.10.10.226 19000 && echo 'CK OK' || echo 'CK FAIL'
  # MinIO
  nc -z 10.10.10.226 9000 && echo 'MinIO OK' || echo 'MinIO FAIL'
  # RMQ
  nc -z 10.10.10.226 9876 && echo 'RMQ OK' || echo 'RMQ FAIL'
  # OceanBase
  nc -z 111.204.125.244 8100 && echo 'OB OK' || echo 'OB FAIL'
"
```

- [ ] **Step 2: 重新初始化数据库（如需要）**

```bash
cd middleware && ./init.sh db
```

- [ ] **Step 3: 确保 ES 索引、RMQ topics、MinIO buckets 存在**

```bash
cd middleware && ./init.sh es && ./init.sh rmq && ./init.sh minio
```

---

### Task 8: 部署 Guard

- [ ] **Step 1: 拉取镜像并启动 Guard**

```bash
ssh dev@220 "
  cd /home/dev/ynet-deploy/guard
  docker-compose pull
  docker-compose up -d
"
```

- [ ] **Step 2: 验证 Guard 健康**

```bash
curl -sf http://10.10.10.220:8080/health
```

---

### Task 9: 部署 Loop

- [ ] **Step 1: 渲染 observability.yaml 模板**

```bash
sed "s/__MW_HOST__/10.10.10.226/g" loop/observability.yaml.tpl > loop/observability.yaml
```

- [ ] **Step 2: 拉取镜像并启动 Loop**

```bash
ssh dev@220 "
  cd /home/dev/ynet-deploy/loop
  docker-compose pull
  docker-compose up -d
"
```

- [ ] **Step 3: 等待 Loop 健康**

```bash
# 等待 healthcheck 通过
for i in $(seq 1 30); do
  curl -sf http://10.10.10.220:8888/ping && break
  sleep 2
done
```

---

### Task 10: 创建 Loop Token + 获取 Session Key

- [ ] **Step 1: 执行 init.sh loop-token**

```bash
cd middleware && ./init.sh loop-token
```
输出包含：LOOP_TOKEN、LOOP_WORKSPACE_ID、LOOP_SESSION_KEY

- [ ] **Step 2: 更新 .env 文件**

将输出的三个值填入 `.env`

---

### Task 11: 部署 Studio

- [ ] **Step 1: 拉取镜像并启动 Studio**

```bash
ssh dev@220 "
  cd /home/dev/ynet-deploy/studio
  docker-compose pull
  docker-compose up -d
"
```

- [ ] **Step 2: 验证 Studio 可访问**

```bash
curl -sf http://10.10.10.220:9888/ | head -1
```

---

### Task 12: 端到端功能验证 — 知识库

- [ ] **Step 1: 创建知识库并上传文档**

通过浏览器：
1. 导航到知识库页面
2. 创建知识库 → 添加自定义文本 → 完成

- [ ] **Step 2: 测试语义/全文/混合检索**

通过检索测试面板，三种模式都应返回结果

---

### Task 13: 端到端功能验证 — 智能体 + 知识库

- [ ] **Step 1: 创建或使用已有智能体**

- [ ] **Step 2: 绑定知识库**

- [ ] **Step 3: 发送消息验证知识库召回**

应显示"已搜索知识库"标签，回复引用知识库内容

---

### Task 14: 端到端功能验证 — 可观测性

- [ ] **Step 1: 发送对话触发 trace**

- [ ] **Step 2: 验证 Studio OTEL export 成功**

```bash
docker logs ynet-server 2>&1 | grep 'successfully exported'
```

- [ ] **Step 3: 验证 Loop 接收并写入 ClickHouse**

```bash
# RMQ 有消息
mqadmin topicStatus -t trace_ingestion_event | grep 'Max Offset'
# ClickHouse 有数据
clickhouse-client -q 'SELECT count() FROM observability_spans'
```

- [ ] **Step 4: 验证可观测性页面显示 trace 数据**

导航到 `/space/{id}/observability`，应显示 trace 记录

---

### Task 15: 最终确认并更新部署文档

- [ ] **Step 1: 确认所有功能正常**

| 功能 | 状态 |
|------|------|
| 知识库语义检索 | |
| 知识库全文检索 | |
| 知识库混合检索 | |
| 智能体对话 | |
| 智能体+知识库 | |
| 可观测性 trace | |
| 可观测性页面 | |

- [ ] **Step 2: 更新 PITFALLS.md 加入今日踩坑**

在 `docs/PITFALLS.md` 中追加今天发现的新问题和解决方案

- [ ] **Step 3: 提交所有改动**

```bash
git add -A
git commit -m "feat(deploy): integrate observability fixes and full automation"
```
