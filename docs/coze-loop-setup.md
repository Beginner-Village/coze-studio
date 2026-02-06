# Coze Loop 可观测性集成指南

## 概述

Coze Studio 通过集成 Coze Loop 实现可观测性功能。追踪数据通过 OTEL 协议从 Coze Studio 上报到 Coze Loop，存储在 ClickHouse 中。

**核心原则：对外只暴露 Coze Studio 一套系统，Coze Loop 作为内部服务不对用户暴露。**

### 当前架构（开发阶段）

```
Coze Studio (localhost:8888)  --OTEL-->  Coze Loop (10.10.10.226:8082)  -->  ClickHouse (10.10.10.226:19000)
       |                                       |
  agent.run span                          存储/查询 traces
  workflow spans                          开发阶段可直接访问 Web UI
```

### 目标架构（正式部署）

```
┌──────────────────────────────────────────────────────────┐
│                    用户可见                                │
│                                                          │
│   Coze Studio (唯一入口)                                  │
│   ├── 智能体开发                                          │
│   ├── 工作流编辑                                          │
│   └── 可观测性页面  ← Coze Studio 自建 UI                 │
│                                                          │
└──────────────────────┬───────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
┌──────────────────────────────────────────────────────────┐
│                   内部服务（不对外暴露）                    │
│                                                          │
│   Coze Studio 后端                                       │
│   ├── 业务逻辑                                            │
│   ├── OTEL 上报 (PAT Token) ──→  Coze Loop 后端          │
│   └── Trace 查询代理 (PAT Token) ──→  Coze Loop API      │
│                                                          │
│   Coze Loop（纯后端服务）                                  │
│   ├── OTEL 接收 + ClickHouse 存储                         │
│   └── Trace 查询 API                                     │
│                                                          │
└──────────────────────────────────────────────────────────┘

认证方式：
- 用户 ↔ Coze Studio：Session Cookie（用户登录态）
- Coze Studio ↔ Coze Loop：PAT Token（服务间认证，配置文件中设置）
- 用户完全不接触 Coze Loop，无需 Coze Loop 账号
```

### 正式部署待实现

1. **Coze Studio 后端**：添加 `/api/observability/*` 代理层，转发请求到 Coze Loop
2. **Coze Studio 前端**：在空间菜单添加"可观测性"页面，自建 trace 列表/详情 UI
3. **部署配置**：Coze Loop 仅内网可达，不暴露公网端口

### 数据映射

| Coze Studio | Coze Loop | 说明 |
|-------------|-----------|------|
| space_id | workspace_id | 请求头 `cozeloop-workspace-id` |
| agent_id | span attribute | 存储在 `agent_id` attribute |
| workflow_id | span attribute | 存储在 workflow span 中 |

---

## 服务器信息

### 远程服务器 (10.10.10.226)

- **SSH**: `ssh dev@10.10.10.226` (sudo 密码: `root1234`)
- **Coze Loop Web UI**: `http://10.10.10.226:8082`
- **Coze Loop App 端口**: `8888` (Docker 内部)

### 远程中间件 (10.10.10.226)

已部署的 Docker 服务（都在 `coze-loop-network` 网络中）：

| 服务 | 容器名 | 端口 | 认证信息 |
|------|--------|------|----------|
| MySQL | coze-loop-mysql | 3306 | user: root, pass: cozeloop-mysql, db: cozeloop-mysql |
| ClickHouse | coze-loop-clickhouse | 8123/19000 | user: default, pass: cozeloop-clickhouse, db: cozeloop-clickhouse |
| Redis | coze-loop-redis | 6379 | pass: cozeloop-redis |
| MinIO | coze-loop-minio | 9000-9001 | user: root, pass: cozeloop-minio, bucket: cozeloop-minio |
| RocketMQ NameSrv | coze-loop-rmq-namesrv | 9876 | - |
| RocketMQ Broker | coze-loop-rmq-broker | - | - |
| **Coze Loop App** | coze-loop-app | 8888 | image: cozedev/coze-loop:1.2.0-dev |
| **Nginx** | coze-loop-nginx | 8082→80 | image: nginx:1.28.0 |

### 本地代码

- **coze-studio**: `/Users/luzhipeng/projects/coze-studio-dev-tmp`
- **coze-loop**: `/Users/luzhipeng/projects/coze-loop`

---

## 本地开发使用（推荐）

现在 Coze Loop 已部署到 10.10.10.226，**本地开发不需要再启动 Coze Loop**，直接使用远程服务即可。

### 1. 确认 Coze Studio 配置

配置文件：`docker/.env.debug`（`make server` 使用此文件）

```bash
# Coze Loop Trace 配置
export COZE_LOOP_TELEMETRY_ENABLE="1"
export COZE_LOOP_TELEMETRY_ENDPOINT="http://10.10.10.226:8082/v1/loop/opentelemetry/v1/traces"
export COZE_LOOP_WORKSPACE_ID="7540860295313358849"  # 默认值，实际使用 space_id
export COZE_LOOP_TELEMETRY_TOKEN="pat_32b9ad833b5223787aa0cd873d9e56853923aea5b8bdac58d1fb1b0fe7e6e1fd"
export COZE_SERVICE_NAME="coze-studio-dev"
export COZE_LOOP_TRACE_RATIO="1.0"
```

### 2. 启动 Coze Studio

```bash
# 启动中间件 + 后端
make server

# 或完整环境
make debug
```

### 3. 查看追踪数据

打开 Coze Loop Web UI：`http://10.10.10.226:8082`
1. 登录（需要先注册账号）
2. 选择对应的 workspace
3. 进入 观测 > Trace 页面查看

---

## 本地启动 Coze Loop（备选方案）

如果需要在本地调试 Coze Loop 代码，可以本地启动：

### 方法一：使用启动脚本

```bash
cd /Users/luzhipeng/projects/coze-loop
bash start-local.sh
```

脚本会自动：
1. 设置远程中间件连接参数
2. 设置 `COZE_LOOP_SKIP_CONSUMERS=true`（跳过 RocketMQ consumer）
3. cd 到 backend 目录（需要 `conf/` 目录）
4. 使用 `go run ./cmd/*.go` 启动

### 方法二：后台启动

```bash
nohup bash /Users/luzhipeng/projects/coze-loop/start-local.sh > /tmp/coze-loop.log 2>&1 &
```

### 切换到本地 Coze Loop

修改 `docker/.env.debug`：
```bash
export COZE_LOOP_TELEMETRY_ENDPOINT="http://localhost:8082/v1/loop/opentelemetry/v1/traces"
```

---

## Coze Studio 关键代码

### 环境变量配置

- **实际使用的配置文件**：`docker/.env.debug`（`make server` 通过 `scripts/setup/server.sh` 加载此文件）
- `bin/.env.debug` 是构建过程中从 `docker/.env.debug` 复制过去的，**不要直接修改 bin/ 下的文件**

### 关键代码修改

#### 1. 动态 Workspace 路由 (telemetry.go)

`backend/infra/otel/telemetry.go` 中实现了 `dynamicWSExporter`，根据 span 的 `cozeloop.workspace_id` attribute 动态路由到对应的 workspace endpoint。

#### 2. Agent 级别 Tracing (agent_run_impl.go)

`backend/domain/conversation/agentrun/service/agent_run_impl.go` 中在 `AgentRun()` 方法的 goroutine 里创建了 `agent.run` span 作为根 span：
- 设置 `cozeloop.workspace_id` = space_id
- 设置 `cozeloop.span_type` = "Agent"
- 传递 span context 给子工作流，形成父子关系

#### 3. Workflow 级别 Tracing (tracing.go)

`backend/domain/workflow/internal/execute/tracing.go` 中通过 Eino callbacks 创建 workflow 和 node spans。

**重要修复**：
- 子工作流的 `cozeloop.workspace_id` 始终使用根工作流的 SpaceID（通过 `rootSpaceID` 变量）
- 子工作流节点的 `span_type` 为 `function`（不是 `Workflow`），避免 UI 层级混乱

---

## Coze Loop 侧修改

### 权限检查绕过（DEV 模式）

文件：`coze-loop/backend/modules/observability/infra/rpc/auth/auth.go`

所有权限检查方法（`CheckWorkspacePermission`, `CheckViewPermission`, `CheckQueryPermission`, `CheckIngestPermission`, `CheckTaskPermission`）在 `COZE_LOOP_SKIP_CONSUMERS=true` 时跳过 RPC 权限检查。

### 直接写入 ClickHouse（绕过 RocketMQ）

文件：`coze-loop/backend/modules/observability/domain/trace/service/trace_service.go`

`IngestTraces()` 方法在 DEV 模式下直接写入 ClickHouse，不通过 RocketMQ consumer。

---

## 远程部署 Coze Loop（构建和更新流程）

当修改了 Coze Loop 代码后，需要重新部署到 10.10.10.226 的完整流程。

### 前置条件

- 本地 coze-loop 代码：`/Users/luzhipeng/projects/coze-loop`
- 远程服务器：`dev@10.10.10.226`（sudo 密码：`root1234`）
- Docker 镜像基础：`cozedev/coze-loop:1.2.0`（远程服务器已有）

### 步骤一：交叉编译 Go 二进制

```bash
cd /Users/luzhipeng/projects/coze-loop/backend

# ARM Mac 交叉编译为 linux/amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o /tmp/coze-loop-main ./cmd/*.go
```

### 步骤二：上传到远程服务器

```bash
scp /tmp/coze-loop-main dev@10.10.10.226:/tmp/coze-loop-main
```

### 步骤三：在远程服务器构建 Docker 镜像

```bash
ssh dev@10.10.10.226

# 创建 Dockerfile
cat > /tmp/Dockerfile.coze-loop <<'EOF'
FROM cozedev/coze-loop:1.2.0
COPY coze-loop-main /coze-loop/bin/main
RUN chmod +x /coze-loop/bin/main
EOF

# 构建镜像（在 /tmp 目录，因为二进制文件在那里）
cd /tmp
echo root1234 | sudo -S docker build -f Dockerfile.coze-loop -t cozedev/coze-loop:1.2.0-dev .
```

### 步骤四：重启容器

```bash
# 停止旧容器
echo root1234 | sudo -S docker stop coze-loop-app coze-loop-nginx
echo root1234 | sudo -S docker rm coze-loop-app coze-loop-nginx

# 启动 app 容器
echo root1234 | sudo -S docker run -d \
  --name coze-loop-app \
  --restart always \
  --network coze-loop-network \
  -p 8888:8888 \
  -v coze-loop-nginx-data:/coze-loop/resources \
  -v /home/dev/coze-dev/coze-loop/release/deployment/docker-compose/bootstrap/app:/coze-loop/bootstrap \
  -v /home/dev/coze-dev/coze-loop/release/deployment/docker-compose/conf:/coze-loop/conf \
  -e COZE_LOOP_REDIS_DOMAIN=coze-loop-redis \
  -e COZE_LOOP_REDIS_PORT=6379 \
  -e COZE_LOOP_REDIS_PASSWORD=cozeloop-redis \
  -e COZE_LOOP_MYSQL_DOMAIN=coze-loop-mysql \
  -e COZE_LOOP_MYSQL_PORT=3306 \
  -e COZE_LOOP_MYSQL_USER=root \
  -e COZE_LOOP_MYSQL_PASSWORD=cozeloop-mysql \
  -e COZE_LOOP_MYSQL_DATABASE=cozeloop-mysql \
  -e COZE_LOOP_CLICKHOUSE_DOMAIN=coze-loop-clickhouse \
  -e COZE_LOOP_CLICKHOUSE_PORT=9000 \
  -e COZE_LOOP_CLICKHOUSE_USER=default \
  -e COZE_LOOP_CLICKHOUSE_PASSWORD=cozeloop-clickhouse \
  -e COZE_LOOP_CLICKHOUSE_DATABASE=cozeloop-clickhouse \
  -e COZE_LOOP_OSS_PROTOCOL=http \
  -e COZE_LOOP_OSS_DOMAIN=coze-loop-minio \
  -e COZE_LOOP_OSS_PORT=9000 \
  -e COZE_LOOP_OSS_USER=root \
  -e COZE_LOOP_OSS_PASSWORD=cozeloop-minio \
  -e COZE_LOOP_OSS_REGION=us-east-1 \
  -e COZE_LOOP_OSS_BUCKET=cozeloop-minio \
  -e COZE_LOOP_RMQ_NAMESRV_DOMAIN=coze-loop-rmq-namesrv \
  -e COZE_LOOP_RMQ_NAMESRV_PORT=9876 \
  --entrypoint sh \
  cozedev/coze-loop:1.2.0-dev \
  /coze-loop/bootstrap/entrypoint.sh

# 等待 app 启动（约 10 秒）
sleep 10
echo root1234 | sudo -S docker logs --tail 5 coze-loop-app
# 确认看到 "Completed!" 后继续

# 启动 nginx 容器
echo root1234 | sudo -S docker run -d \
  --name coze-loop-nginx \
  --restart always \
  --network coze-loop-network \
  -p 8082:80 \
  -v coze-loop-nginx-data:/usr/share/nginx/html:ro \
  -v /home/dev/coze-dev/coze-loop/release/deployment/docker-compose/bootstrap/nginx:/coze-loop-nginx/bootstrap \
  -e COZE_LOOP_OSS_PROTOCOL=http \
  -e COZE_LOOP_OSS_DOMAIN=coze-loop-minio \
  -e COZE_LOOP_OSS_PORT=9000 \
  -e COZE_LOOP_OSS_BUCKET=cozeloop-minio \
  --entrypoint sh \
  nginx:1.28.0 \
  /coze-loop-nginx/bootstrap/entrypoint.sh
```

### 步骤五：验证

```bash
# 检查容器状态
echo root1234 | sudo -S docker ps --filter name=coze-loop-app --filter name=coze-loop-nginx

# 测试 OTEL 端点
curl -s http://10.10.10.226:8082/v1/loop/opentelemetry/v1/traces \
  -X POST -H "Content-Type: application/x-protobuf" \
  -H "Authorization: Bearer pat_32b9ad833b5223787aa0cd873d9e56853923aea5b8bdac58d1fb1b0fe7e6e1fd" \
  -d ''
# 预期返回：{"code":602000101,"msg":"no access permission"}（空请求正常）

# 测试 Web UI
curl -s -o /dev/null -w "%{http_code}" http://10.10.10.226:8082/
# 预期返回：200
```

### 关键配置文件（远程服务器）

| 文件路径 | 说明 |
|---------|------|
| `/home/dev/coze-dev/coze-loop/release/deployment/docker-compose/conf/observability.yaml` | 可观测性配置（MQ topic、collector 等） |
| `/home/dev/coze-dev/coze-loop/release/deployment/docker-compose/bootstrap/app/entrypoint.sh` | App 启动脚本 |
| `/home/dev/coze-dev/coze-loop/release/deployment/docker-compose/bootstrap/nginx/entrypoint.sh` | Nginx 启动脚本（包含 /v1/ 代理） |
| `/home/dev/coze-dev/coze-loop/release/deployment/docker-compose/.env` | Docker 环境变量 |

### 注意事项

1. **observability.yaml 需同步**：本地代码 `coze-loop/backend/conf/observability.yaml` 需要与远程保持一致，但地址要替换：
   - 本地用 `10.10.10.226:9876`（直连远程 RocketMQ）
   - 远程用 `coze-loop-rmq-namesrv:9876`（Docker 内部网络）

2. **Nginx 已添加 /v1/ 代理**：修改了 `bootstrap/nginx/entrypoint.sh`，添加了 `/v1/` 路径的反向代理，使 OTEL traces 可通过 8082 端口访问。

3. **不使用 docker-compose 启动 app/nginx**：因为 docker-compose 中 app 依赖所有中间件服务的 health check，而中间件不是通过 docker-compose 管理的（是独立 docker run 的），所以用 `docker run` 直接启动。

---

## 验证追踪数据

### 方法一：ClickHouse 直接查询

```bash
# 查询今天的 span 数量
curl -s "http://10.10.10.226:8123/?user=default&password=cozeloop-clickhouse&database=cozeloop-clickhouse" \
  --data-binary "SELECT count() FROM observability_spans WHERE start_time > $(date +%s)000000000"

# 查看最近的 agent span
curl -s "http://10.10.10.226:8123/?user=default&password=cozeloop-clickhouse&database=cozeloop-clickhouse" \
  --data-binary "SELECT span_name, span_type, space_id, trace_id, duration FROM observability_spans WHERE span_type='Agent' ORDER BY start_time DESC LIMIT 5"

# 查看某个 trace 的完整层级
curl -s "http://10.10.10.226:8123/?user=default&password=cozeloop-clickhouse&database=cozeloop-clickhouse" \
  --data-binary "SELECT span_id, span_name, span_type, parent_id FROM observability_spans WHERE trace_id='<trace_id>' ORDER BY start_time ASC"
```

### 方法二：Coze Loop Web UI

1. 打开 `http://10.10.10.226:8082`
2. 登录后选择对应的 workspace
3. 进入 观测 > Trace 页面查看

### 方法三：触发测试追踪

```bash
# 通过 OpenAPI 触发智能体对话（会产生 agent + workflow spans）
curl -X POST "http://localhost:8888/v3/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "bot_id": "<agent_id>",
    "user_id": "test_user",
    "stream": true,
    "auto_save_history": true,
    "additional_messages": [{"role": "user", "content": "你好", "content_type": "text"}]
  }'
```

---

## Trace 层级结构

一次智能体对话产生的 trace 层级：

```
agent.run (Type: Agent, root span)
├── workflow_name (Type: Workflow)
│   ├── 开始 (Type: WorkflowStart)
│   ├── 节点1 (Type: function/plugin/llm...)
│   ├── 子工作流节点 (Type: function)  ← 注意：子工作流节点的 span_type 是 function
│   │   └── sub_workflow_name (Type: Workflow)
│   │       ├── 开始 (Type: WorkflowStart)
│   │       ├── 子节点1 (Type: function/plugin/llm...)
│   │       └── 结束 (Type: WorkflowEnd)
│   ├── 节点2 (Type: function/plugin/llm...)
│   └── 结束 (Type: WorkflowEnd)
└── ...
```

---

## 常见问题

### Q: Coze Loop 启动失败 `panic: lstat conf: no such file or directory`
**A**: 必须从 `coze-loop/backend/` 目录启动，因为需要读取 `conf/` 目录。使用 `start-local.sh` 脚本会自动处理。

### Q: Coze Loop 容器 `panic: trace topic required`
**A**: `observability.yaml` 缺少 MQ producer 配置。需要确保包含以下配置项：
- `trace_ingest_tenant_config` (含 mq_producer)
- `annotation_mq_producer_config`
- `span_with_annotation_mq_producer_config`
- `backfill_mq_producer_config`

同步本地 `coze-loop/backend/conf/observability.yaml` 到远程，注意替换地址为 Docker 内部网络名。

### Q: API 返回 `{"code":602000702,"msg":"Service Internal Error"}`
**A**: 内部 API (`/api/`) 需要 session cookie 认证。直接用 curl 带 Bearer token 会触发 session 验证失败。改用浏览器访问或使用 OpenAPI 端点。

### Q: 上报数据后 ClickHouse 没数据
**A**: BatchSpanProcessor 默认每 5 秒批量导出。等几秒再查。确认 `COZE_LOOP_TELEMETRY_ENABLE=1` 且 Coze Loop 服务在运行。

### Q: workspace_id 如何关联
**A**: Coze Studio 的 `space_id` = Coze Loop 的 `workspace_id`。在 span 的 `cozeloop.workspace_id` attribute 中设置。

### Q: 子工作流 span 显示"trace不在当前空间"
**A**: 子工作流的 `cozeloop.workspace_id` 必须使用根工作流的 SpaceID。已在 `tracing.go` 中通过 `rootSpaceID` 变量修复。

### Q: `make server` 用的是哪个 .env 文件？
**A**: 用的是 `docker/.env.debug`。`make server` → `scripts/setup/server.sh` → 当 `APP_ENV=debug` 时加载 `docker/.env.debug`。`bin/.env.debug` 是构建过程中复制过去的副本。
