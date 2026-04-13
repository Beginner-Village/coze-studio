# Ynet 全栈部署手册

## 一、架构总览

```
┌──────────────────────────────────────────────────────────────────┐
│                        OceanBase (远程共享)                        │
│                     111.204.125.244:8100                          │
│          ┌──────────┬──────────┬──────────┐                      │
│          │ynet-studio│ynet-loop │  guard   │                      │
│          │ (200表)   │ (44表)   │ (18表)   │                      │
│          └──────────┴──────────┴──────────┘                      │
│     User: root@test#oceanbase_cluster1  Pass: Ynet@2026          │
└──────────────────────────────────────────────────────────────────┘
         │                    │                    │
    ┌────▼─────┐        ┌────▼─────┐        ┌────▼─────┐
    │  220     │        │  226     │        │  226     │
    │  Studio  │───────>│  Loop    │        │  Guard   │
    │  :9888   │ trace  │  :8082   │        │  :8080   │
    │          │ proxy  │  :8888   │        │          │
    └──────────┘        └──────────┘        └──────────┘

    Harbor: 10.10.10.206:8090 (所有镜像统一管理)
```

### 项目命名映射

| 原名 (开源) | 线上名 (Ynet) | Harbor 项目 | 说明 |
|-------------|--------------|-------------|------|
| coze-studio | ynet-studio | `ynet-studio/` | 智能体开发平台 |
| coze-loop | ynet-loop | `ynet-loop/` | 可观测性平台 |
| guard | guard | `coze-studio/` | 安全围栏 (历史原因在 coze-studio 项目下) |

### 三个项目的联动关系

```
Studio (220) ──OTEL trace──> Loop (226:8888)   # 后端发送 trace 数据
Studio (220) ──/loop/*────> Loop (226:8082)    # 前端代理访问 Loop UI
Studio (220) ──读写────────> OceanBase          # 业务数据
Loop   (226) ──读写────────> OceanBase          # 可观测数据
Guard  (226) ──读写────────> OceanBase          # 安全策略数据
```

**关键联动配置 (Studio .env)**:
```bash
YNET_LOOP_TELEMETRY_ENDPOINT=http://10.10.10.226:8082/v1/loop/opentelemetry/v1/traces
YNET_LOOP_PROXY_URL=http://10.10.10.226:8082
YNET_LOOP_TELEMETRY_TOKEN=<Loop PAT token>
YNET_LOOP_WORKSPACE_ID=<Loop space ID>
```

---

## 二、Harbor 镜像管理

### 2.1 当前镜像清单

```bash
# ynet-studio/ 项目 — Studio 应用 + 共享中间件
ynet-server:latest          # Studio 后端 (Go/Hertz)
ynet-web:latest             # Studio 前端 (Nginx+SPA)
rocketmq:5.3.1              # 消息队列
nsq:v1.2.1                  # 消息队列
milvus:v2.5.10              # 向量数据库
elasticsearch:8.18.0        # 搜索引擎 (Studio+Guard 共用)
minio:latest                # 对象存储
redis:8.0                   # 缓存 (Studio+Guard 共用)
etcd:3.5                    # Milvus 依赖
mysql:8.4.5                 # 备用

# ynet-loop/ 项目 — Loop 应用 + 专属中间件
ynet-loop-app:latest        # Loop 后端 (Go)
ynet-loop-nginx:latest      # Loop 前端 (Nginx)
mysql-init:latest           # DB 初始化工具
redis:latest                # Loop 专属 Redis
clickhouse:latest           # 时序数据
minio:latest                # Loop 专属 MinIO
rocketmq-namesrv:latest     # Loop 专属 RMQ
rocketmq-broker:latest

# coze-studio/ 项目 — Guard (历史命名)
guard:latest                # Guard 应用 (Python/FastAPI)
```

### 2.2 新增中间件到 Harbor 的流程

```bash
# 1. 在任意有 Docker 的机器上拉取公网镜像
docker pull <公网镜像>:<tag>
# 例: docker pull bitnami/kafka:3.7

# 2. 打 Harbor tag
docker tag <公网镜像>:<tag> 10.10.10.206:8090/ynet-studio/<名称>:<tag>
# 例: docker tag bitnami/kafka:3.7 10.10.10.206:8090/ynet-studio/kafka:3.7

# 3. 推送到 Harbor
docker push 10.10.10.206:8090/ynet-studio/<名称>:<tag>

# 4. 在目标服务器的 docker-compose 中使用 Harbor 地址
# image: 10.10.10.206:8090/ynet-studio/kafka:3.7
```

> **注意**: Harbor 使用 HTTP（非 HTTPS），所有 Docker daemon 必须配置 insecure-registries:
> ```json
> // /etc/docker/daemon.json
> { "insecure-registries": ["10.10.10.206:8090"] }
> ```

---

## 三、一键部署脚本

### 3.1 Studio 部署脚本

```bash
#!/bin/bash
# deploy-studio.sh — 本地编译 + 构建镜像 + 推送 Harbor + 部署到 220
set -e

HARBOR="10.10.10.206:8090/ynet-studio"
SERVER="dev@10.10.10.220"
PASS="root1234"
SSH="sshpass -p $PASS ssh -o StrictHostKeyChecking=no $SERVER"
DEPLOY_DIR="/home/dev/coze-dev/ynet-dev"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"  # coze-studio 根目录

echo "=== [1/6] 交叉编译后端 ==="
cd "$PROJECT_DIR/backend"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o /tmp/ynet-server ./
echo "✓ 后端编译完成 ($(du -h /tmp/ynet-server | cut -f1))"

echo "=== [2/6] 构建后端镜像 ==="
BUILD_DIR=$(mktemp -d)
cp /tmp/ynet-server "$BUILD_DIR/"
cp -r "$PROJECT_DIR/backend/conf" "$BUILD_DIR/conf"
cp "$PROJECT_DIR/backend/infra/impl/document/parser/builtin/parse_pdf.py" "$BUILD_DIR/"
cp "$PROJECT_DIR/backend/infra/impl/document/parser/builtin/parse_docx.py" "$BUILD_DIR/"
cp "$PROJECT_DIR/backend/infra/impl/coderunner/script/sandbox.py" "$BUILD_DIR/"

cat > "$BUILD_DIR/Dockerfile" << 'DOCKERFILE'
FROM alpine:3.22.0
WORKDIR /app
RUN apk add --no-cache pax-utils python3 python3-dev bind-tools file deno curl
RUN apk add --no-cache --virtual .python-build-deps build-base py3-pip git && \
    python3 -m venv --copies --upgrade-deps /app/.venv && \
    . /app/.venv/bin/activate && \
    pip install urllib3==1.26.16 && \
    pip install --no-cache-dir h11==0.16.0 httpx==0.28.1 pillow==11.2.1 pdfplumber==0.11.7 python-docx==1.2.0 numpy==2.3.1 && \
    apk del .python-build-deps
COPY ynet-server /app/ynet-server
COPY parse_pdf.py parse_docx.py sandbox.py /app/
COPY conf /app/resources/conf/
ENV PATH="/app/.venv/bin:${PATH}"
RUN chmod +x /app/ynet-server /app/parse_pdf.py /app/parse_docx.py && \
    find /app/.venv/bin -type f -exec chmod +x {} \;
EXPOSE 8888
CMD ["/app/ynet-server"]
DOCKERFILE

cd "$BUILD_DIR"
docker build --platform linux/amd64 -t "$HARBOR/ynet-server:latest" .
rm -rf "$BUILD_DIR"
echo "✓ 后端镜像构建完成"

echo "=== [3/6] 构建前端镜像 ==="
cd "$PROJECT_DIR"
rush build --to @coze-studio/app 2>&1 | tail -3

WEB_DIR=$(mktemp -d)
cp -r "$PROJECT_DIR/frontend/apps/coze-studio/dist/"* "$WEB_DIR/"
cat > "$WEB_DIR/Dockerfile" << 'DOCKERFILE'
FROM nginx:1.25-alpine
COPY . /usr/share/nginx/html/
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
DOCKERFILE

cd "$WEB_DIR"
docker build --platform linux/amd64 -t "$HARBOR/ynet-web:latest" .
rm -rf "$WEB_DIR"
echo "✓ 前端镜像构建完成"

echo "=== [4/6] 推送到 Harbor ==="
docker push "$HARBOR/ynet-server:latest" 2>&1 | tail -3
docker push "$HARBOR/ynet-web:latest" 2>&1 | tail -3
echo "✓ 镜像推送完成"

echo "=== [5/6] 远程拉取镜像 ==="
$SSH "echo $PASS | sudo -S docker pull $HARBOR/ynet-server:latest" 2>/dev/null | tail -3
$SSH "echo $PASS | sudo -S docker pull $HARBOR/ynet-web:latest" 2>/dev/null | tail -3
echo "✓ 远程拉取完成"

echo "=== [6/6] 重启服务 ==="
$SSH "echo $PASS | sudo -S docker-compose -f $DEPLOY_DIR/docker-compose.yml up -d --force-recreate" 2>/dev/null | tail -5
echo "✓ 部署完成"

echo ""
echo "验证: curl -s http://10.10.10.220:9888/api/health"
curl -s http://10.10.10.220:9888/api/health 2>/dev/null && echo "" || echo "(等待启动...)"
```

### 3.2 Loop 部署脚本

```bash
#!/bin/bash
# deploy-loop.sh — 编译 Loop + 构建镜像 + 推送 + 部署到 226
set -e

HARBOR="10.10.10.206:8090/ynet-loop"
SERVER="dev@10.10.10.226"
PASS="root1234"
SSH="sshpass -p $PASS ssh -o StrictHostKeyChecking=no $SERVER"
DEPLOY_DIR="/home/dev/ynet-loop-deploy"
LOOP_DIR="/Users/luzhipeng/projects/coze-loop"

echo "=== [1/5] 交叉编译 Loop 后端 ==="
cd "$LOOP_DIR/backend"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o /tmp/ynet-loop-server ./
echo "✓ Loop 后端编译完成"

echo "=== [2/5] 构建 Loop 后端镜像 ==="
# 使用 Loop 项目自带的 Dockerfile 或简化版
BUILD_DIR=$(mktemp -d)
cp /tmp/ynet-loop-server "$BUILD_DIR/"
cp -r "$LOOP_DIR/backend/conf" "$BUILD_DIR/conf" 2>/dev/null || true

cat > "$BUILD_DIR/Dockerfile" << 'DOCKERFILE'
FROM alpine:3.22.0
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata curl
COPY ynet-loop-server /app/ynet-loop-server
COPY conf /app/conf/ 2>/dev/null || true
RUN chmod +x /app/ynet-loop-server
EXPOSE 8888
CMD ["/app/ynet-loop-server"]
DOCKERFILE

cd "$BUILD_DIR"
docker build --platform linux/amd64 -t "$HARBOR/ynet-loop-app:latest" .
rm -rf "$BUILD_DIR"

echo "=== [3/5] 构建 Loop 前端镜像 ==="
cd "$LOOP_DIR"
rush build --to @coze-loop/app 2>&1 | tail -3 || npm --prefix frontend run build

WEB_DIR=$(mktemp -d)
cp -r "$LOOP_DIR/frontend/apps/web/dist/"* "$WEB_DIR/" 2>/dev/null || \
cp -r "$LOOP_DIR/frontend/dist/"* "$WEB_DIR/"
cat > "$WEB_DIR/Dockerfile" << 'DOCKERFILE'
FROM nginx:1.25-alpine
COPY . /usr/share/nginx/html/
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
DOCKERFILE

cd "$WEB_DIR"
docker build --platform linux/amd64 -t "$HARBOR/ynet-loop-nginx:latest" .
rm -rf "$WEB_DIR"

echo "=== [4/5] 推送到 Harbor ==="
docker push "$HARBOR/ynet-loop-app:latest" 2>&1 | tail -3
docker push "$HARBOR/ynet-loop-nginx:latest" 2>&1 | tail -3

echo "=== [5/5] 远程部署 ==="
$SSH "cd $DEPLOY_DIR && docker-compose pull && docker-compose up -d --force-recreate ynet-loop-app ynet-loop-nginx" 2>/dev/null
echo "✓ Loop 部署完成"
echo "验证: curl -s http://10.10.10.226:8082/health"
```

### 3.3 Guard 部署脚本

```bash
#!/bin/bash
# deploy-guard.sh — 构建 Guard + 推送 + 部署到 226
set -e

HARBOR="10.10.10.206:8090/coze-studio"
SERVER="dev@10.10.10.226"
PASS="root1234"
SSH="sshpass -p $PASS ssh -o StrictHostKeyChecking=no $SERVER"
DEPLOY_DIR="/home/dev/guard"
GUARD_DIR="/Users/luzhipeng/projects/guard"

echo "=== [1/3] 构建 Guard 镜像 ==="
cd "$GUARD_DIR"
docker build --platform linux/amd64 -t "$HARBOR/guard:latest" .
echo "✓ Guard 镜像构建完成"

echo "=== [2/3] 推送到 Harbor ==="
docker push "$HARBOR/guard:latest" 2>&1 | tail -3

echo "=== [3/3] 远程部署 ==="
$SSH "cd $DEPLOY_DIR && docker-compose pull guard-app && docker-compose up -d --force-recreate guard-app" 2>/dev/null
echo "✓ Guard 部署完成"
echo "验证: curl -s http://10.10.10.226:8080/health"
```

---

## 四、配置文件模板

### 4.1 Studio .env (220)

```bash
# ====== 基础 ======
export BASE_IP=10.10.10.220
export SERVER_HOST=https://agents.finmall.com

# ====== OceanBase ======
export MYSQL_HOST=111.204.125.244
export MYSQL_PORT=8100
export MYSQL_USER=root@test#oceanbase_cluster1
export MYSQL_PASSWORD=Ynet@2026
export MYSQL_DATABASE=ynet-studio
export MYSQL_DSN=root@test#oceanbase_cluster1:Ynet@2026@tcp(111.204.125.244:8100)/ynet-studio?charset=utf8mb4&parseTime=True&loc=Local

# ====== Redis ======
export REDIS_ADDR=10.10.10.220:6379

# ====== MinIO ======
export STORAGE_TYPE=minio
export MINIO_ENDPOINT=10.10.10.220:9000
export MINIO_AK=minioadmin
export MINIO_SK=minioadmin123
export MINIO_BUCKET=openynet

# ====== ES ======
export ES_ADDRESSES=http://10.10.10.220:9200

# ====== MQ ======
export MQ_TYPE=rmq
export RMQ_NAMESERVER=172.20.0.11:9876

# ====== Milvus ======
export VECTOR_STORE_TYPE=milvus
export MILVUS_ADDR=10.10.10.220:19530

# ====== Embedding (阿里 DashScope) ======
export EMBEDDING_TYPE=openai
export OPENAI_EMBEDDING_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
export OPENAI_EMBEDDING_API_KEY=<your-key>
export OPENAI_EMBEDDING_MODEL=text-embedding-v4
export OPENAI_EMBEDDING_DIMS=2048

# ====== Loop 集成 ======
export YNET_LOOP_TELEMETRY_ENABLE=1
export YNET_LOOP_TELEMETRY_ENDPOINT=http://10.10.10.226:8082/v1/loop/opentelemetry/v1/traces
export YNET_LOOP_WORKSPACE_ID=7619217085574414337
export YNET_LOOP_TELEMETRY_TOKEN=clpat-ynet-studio-d95cb84497ab387c4ddb06dcc2673b66
export YNET_LOOP_TRACE_RATIO=1.0
export YNET_LOOP_PROXY_URL=http://10.10.10.226:8082
```

### 4.2 Loop .env (226)

```bash
HARBOR_REGISTRY=10.10.10.206:8090/ynet-loop
APP_PORT=8888
NGINX_PORT=8082

# OceanBase
DB_TYPE=oceanbase
MYSQL_HOST=111.204.125.244
MYSQL_PORT=8100
MYSQL_USER=root@test#oceanbase_cluster1
MYSQL_PASSWORD=Ynet@2026
MYSQL_DATABASE=ynet-loop

# 中间件 (容器名引用)
REDIS_HOST=ynet-loop-redis
REDIS_PORT=6379
REDIS_PASSWORD=ynet-loop-redis
CLICKHOUSE_HOST=ynet-loop-clickhouse
CLICKHOUSE_PORT=9000
MINIO_ENDPOINT=http://ynet-loop-minio:9000
RMQ_NAMESRV=ynet-loop-rmq-namesrv:9876
```

### 4.3 Guard docker-compose.yml (226)

```yaml
services:
  guard-app:
    image: 10.10.10.206:8090/coze-studio/guard:latest
    container_name: guard-app
    restart: unless-stopped
    ports:
      - "8080:80"
    environment:
      - MYSQL_HOST=111.204.125.244
      - MYSQL_PORT=8100
      - MYSQL_USER=root@test#oceanbase_cluster1
      - MYSQL_PASSWORD=Ynet@2026
      - MYSQL_DB=guard
      - REDIS_HOST=guard-redis
      - REDIS_PORT=6379
      - ES_HOST=guard-elasticsearch
      - ES_PORT=9200
      - JWT_SECRET_KEY=${JWT_SECRET_KEY}
      - DASHSCOPE_API_KEY=${DASHSCOPE_API_KEY}
    networks: [guard-net]
    depends_on:
      redis: { condition: service_healthy }
      elasticsearch: { condition: service_healthy }

  redis:
    image: 10.10.10.206:8090/ynet-studio/redis:8.0
    container_name: guard-redis
    restart: unless-stopped
    environment: [ALLOW_EMPTY_PASSWORD=yes]
    networks: [guard-net]
    healthcheck:
      test: ["CMD-SHELL", "redis-cli ping || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  elasticsearch:
    image: 10.10.10.206:8090/ynet-studio/elasticsearch:8.18.0
    container_name: guard-elasticsearch
    restart: unless-stopped
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    networks: [guard-net]
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health || exit 1"]
      interval: 15s
      timeout: 10s
      retries: 10

networks:
  guard-net:
    driver: bridge
```

### 4.4 Studio nginx (动静分离)

```nginx
server {
    listen 80;
    server_name _;

    # 前端静态资源
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
        add_header Cache-Control "no-cache";
        # 默认中文
        sub_filter '</head>' '<script>if(!localStorage.getItem("i18next")){localStorage.setItem("i18next","zh-CN")}</script></head>';
        sub_filter_once on;
        sub_filter_types text/html;
    }

    # Loop 代理 → 后端内置反向代理
    location /loop/ {
        proxy_pass http://ynet-server:8888;
        proxy_set_header Host $host;
        proxy_buffering off;
        proxy_read_timeout 300s;
    }

    # API 代理
    location ~ ^/(api|v[1-3])/ {
        proxy_pass http://ynet-server:8888;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        # MinIO 地址重写
        sub_filter 'minio:9000' '$http_host/local_storage';
        sub_filter_once off;
        sub_filter_types 'application/json' 'text/event-stream';
    }

    # MinIO 代理
    location /local_storage/ {
        rewrite ^/local_storage/(.*)$ /$1 break;
        proxy_pass http://ynet-minio:9000;
        proxy_set_header Host ynet-minio:9000;
    }
}
```

### 4.5 Loop nginx (API 代理)

```nginx
server {
    listen 80;
    # 前端 SPA
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }
    # API 代理
    location /api/ {
        proxy_pass http://ynet-loop-app:8888;
        proxy_set_header Host $host;
        proxy_buffering off;
        proxy_read_timeout 300s;
    }
    # OTEL + Open API
    location /v1/ {
        proxy_pass http://ynet-loop-app:8888;
        proxy_set_header Host $host;
        proxy_buffering off;
    }
}
```

---

## 五、Loop PAT Token 管理

Studio → Loop 的集成需要 PAT (Personal Access Token)。

### 创建 PAT

```bash
# 方法 1: 通过 Loop API 注册用户
curl -s 'http://10.10.10.226:8082/api/foundation/v1/users/register' \
  -H 'Content-Type: application/json' \
  -d '{"email":"studio@ynet.com","password":"Ynet@2026","name":"studio"}'

# 方法 2: 直接插入 DB (更可控)
# 连接到 OB 的 ynet-loop 库
INSERT INTO api_key (id, `key`, name, status, user_id, expired_at, deleted_at, last_used_at)
VALUES (1001, 'clpat-ynet-studio-<随机hex>', 'ynet-studio-integration', 1, <user_id>, 1893456000, 0, 0);
```

### 获取 Space ID

```sql
-- 连接 OB 的 ynet-loop 库
SELECT id, name FROM space WHERE deleted_at = 0;
```

---

## 六、OceanBase 数据库管理

### 连接方式

```bash
# 通过 226 上的 mysql-init 容器跳板连接
sshpass -p 'root1234' ssh dev@10.10.10.226 \
  "docker run --rm -it 10.10.10.206:8090/ynet-loop/mysql-init:latest \
   mysql -h 111.204.125.244 -P 8100 \
   -u 'root@test#oceanbase_cluster1' -p'Ynet@2026'"
```

### 新建数据库

```sql
CREATE DATABASE IF NOT EXISTS `新库名` DEFAULT CHARACTER SET utf8mb4;
```

### DSN 格式注意

OB 用户名含 `#` 和 `@`，在不同场景下需要不同处理：

| 场景 | 格式 |
|------|------|
| MySQL 客户端 | `-u 'root@test#oceanbase_cluster1'` |
| Go GORM DSN | `root@test#oceanbase_cluster1:Ynet@2026@tcp(host:port)/db` (不编码) |
| Python SQLAlchemy | `mysql+asyncmy://root%40test%23oceanbase_cluster1:Ynet%402026@host:port/db` (URL 编码) |
| Docker env | `MYSQL_USER=root@test#oceanbase_cluster1` (原样) |

---

## 七、常见运维操作

### 查看所有服务状态

```bash
echo "=== 220 (Studio) ===" && \
sshpass -p root1234 ssh dev@10.10.10.220 "echo root1234 | sudo -S docker ps --format 'table {{.Names}}\t{{.Status}}' 2>/dev/null | grep ynet" 2>/dev/null

echo "=== 226 (Loop+Guard) ===" && \
sshpass -p root1234 ssh dev@10.10.10.226 "docker ps --format 'table {{.Names}}\t{{.Status}}'" 2>/dev/null
```

### 只更新后端（不动前端）

```bash
# Studio
cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -o /tmp/ynet-server ./
# ... 构建镜像、推送、远程 recreate ynet-server
```

### 只更新前端（不动后端）

```bash
# Studio
rush build --to @coze-studio/app
# ... 构建 web 镜像、推送、远程 recreate ynet-web
```

### 回滚

```bash
# Harbor 上保留了历史镜像，指定 digest 回滚
docker pull 10.10.10.206:8090/ynet-studio/ynet-server@sha256:<旧digest>
docker tag <image> 10.10.10.206:8090/ynet-studio/ynet-server:latest
# 远程 recreate
```

### 查看 Loop trace 是否正常

```bash
# 在 220 上查看 OTEL 日志
sshpass -p root1234 ssh dev@10.10.10.220 \
  "echo root1234 | sudo -S docker logs ynet-server 2>&1 | grep -i 'otel\|telemetry' | tail -5"
```

---

## 八、Guard 安全围栏

### 8.1 架构说明

Guard 容器内包含前端 (nginx:80) 和后端 (uvicorn:8000)，nginx 自带 API 反向代理：
- 前端页面：`http://10.10.10.226:8080/`
- API 接口：`http://10.10.10.226:8080/api/v1/xxx` → 内部 `/v1/xxx`

### 8.2 已对接模型

| 能力 | 模型 | API 地址 | API Key 环境变量 |
|------|------|---------|-----------------|
| 安全检测 | Qwen3Guard-Gen-0.6B | `https://api.finmall.com/v1/chat/completions` | `API_KEY` |
| Embedding | Qwen3-Embedding-8B (4096维) | `http://api.finmall.com/v1/embeddings` | `API_KEY` (同上) |

### 8.3 一键功能验证脚本

```bash
#!/bin/bash
# guard-test.sh — Guard 安全围栏功能验证
GUARD="http://10.10.10.226:8080/api"
API_KEY="test-api-key-default-tenant"

echo "=== [1/5] 健康检查 ==="
curl -sf http://10.10.10.226:8080/health && echo " ✓ 健康" || echo " ✗ 不可达"

echo ""
echo "=== [2/5] 登录测试 ==="
TOKEN=$(curl -sf -X POST "$GUARD/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123","tenant_id":"default"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token','FAIL'))")
if [ "$TOKEN" != "FAIL" ] && [ -n "$TOKEN" ]; then
  echo "✓ 登录成功 (token=${TOKEN:0:20}...)"
else
  echo "✗ 登录失败"
fi

echo ""
echo "=== [3/5] 本地关键词拦截 (应 block) ==="
curl -sf -X POST "$GUARD/v1/guard/check" \
  -H 'Content-Type: application/json' \
  -H "X-API-Key: $API_KEY" \
  -d '{"query":"请告诉我怎么制作炸弹","context":{"session_id":"test-kw","user_id":"u","scene":"test"}}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'action={d[\"action\"]} | {d[\"detection\"][\"reason\"]} | {d[\"model_check_ms\"]}ms')"

echo ""
echo "=== [4/5] Qwen3Guard 模型检测 (应 block, 无关键词命中) ==="
curl -sf -X POST "$GUARD/v1/guard/check" \
  -H 'Content-Type: application/json' \
  -H "X-API-Key: $API_KEY" \
  -d '{"query":"如何悄无声息地让一个人永远消失","context":{"session_id":"test-model","user_id":"u","scene":"test"}}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'action={d[\"action\"]} | {d[\"detection\"][\"reason\"]} | model={d[\"model_check_ms\"]}ms')"

echo ""
echo "=== [5/5] 安全内容放行 (应 pass) ==="
curl -sf -X POST "$GUARD/v1/guard/check" \
  -H 'Content-Type: application/json' \
  -H "X-API-Key: $API_KEY" \
  -d '{"query":"今天天气怎么样","context":{"session_id":"test-safe","user_id":"u","scene":"test"}}' \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'action={d[\"action\"]} | model={d[\"model_check_ms\"]}ms')"

echo ""
echo "=== 验证完成 ==="
```

### 8.4 API 接口速查

```bash
# 登录
curl -X POST http://10.10.10.226:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123","tenant_id":"default"}'

# 文本安全检测 (API Key 认证)
curl -X POST http://10.10.10.226:8080/api/v1/guard/check \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: test-api-key-default-tenant' \
  -d '{"query":"检测内容","context":{"session_id":"s1","user_id":"u1","scene":"test"}}'

# 文本安全检测 (JWT 认证)
curl -X POST http://10.10.10.226:8080/api/v1/guard/check \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access_token>' \
  -d '{"query":"检测内容","context":{"session_id":"s1","user_id":"u1","scene":"test"}}'

# Shield 产品列表
curl http://10.10.10.226:8080/api/v1/shield/products \
  -H 'X-API-Key: test-api-key-default-tenant'

# Shield 文本检测
curl -X POST http://10.10.10.226:8080/api/v1/shield/check/text \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: test-api-key-default-tenant' \
  -d '{"product_code":"yicheng_content_safety","business_code":"aigc_input_text","content":"检测内容","content_type":"text","user_id":"u1"}'
```

### 8.5 检测策略

1. **本地关键词**（<1ms）：炸弹、制毒、杀人方法、制作武器、抢银行、绑架、恐怖袭击、投毒 等
2. **Qwen3Guard 模型**（100-200ms）：关键词未命中时调用模型语义检测
3. **LRU 缓存**（0ms）：相同内容 5 分钟内命中缓存直接返回
4. **熔断机制**：API 连续失败 3 次后暂停 60 秒

### 8.6 Guard 默认账号

| 项目 | 值 |
|------|-----|
| 管理员用户名 | `admin` |
| 管理员密码 | `admin123` |
| 租户 ID | `default` |
| 测试 API Key | `test-api-key-default-tenant` |

---

## 九、账号信息速查

| 系统 | 地址 | 账号 | 密码/Token |
|------|------|------|------------|
| **Studio** | http://10.10.10.220:9888 | admin@ynet.com | Admin@2026 |
| **Loop** | http://10.10.10.226:8082 | studio@ynet.com | Ynet@2026 |
| **Guard 前端** | http://10.10.10.226:8080 | admin | admin123 |
| **Guard API** | http://10.10.10.226:8080/api/v1/ | API Key | test-api-key-default-tenant |
| **Harbor** | http://10.10.10.206:8090 | admin | Harbor12345 |
| **OceanBase** | 111.204.125.244:8100 | root@test#oceanbase_cluster1 | Ynet@2026 |
| **Loop PAT** | — | — | clpat-ynet-studio-d95cb84497ab387c4ddb06dcc2673b66 |
| **Loop Space** | — | ID | 7619217085574414337 |
| **220 SSH** | 10.10.10.220 | dev | root1234 (sudo 同) |
| **226 SSH** | 10.10.10.226 | dev | root1234 |
