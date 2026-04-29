# Observability Stack Infrastructure — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 220 上搭一套 Prometheus + Grafana + node_exporter 监控栈，对外通过宿主 IP scrape 4 个被监控应用，将来可整体搬到任意机器（只改 IP）。

**Architecture:** obs-stack 用独立 bridge 网络隔离运行，对外只 expose Grafana(3000) 和 Prometheus(9090)。`prometheus.yml` 用 `10.10.10.220:<port>` 作为 target。Grafana 通过 file provisioning 加载 datasources 和 dashboards，重启不丢数据。node_exporter 跑在被监控机（PoC 阶段就是 220）host network 模式采集主机指标。

**Tech Stack:** Docker Compose v2, Prometheus v2.55.0, Grafana v11.3.0, node_exporter v1.8.2

**Spec:** [docs/superpowers/specs/2026-04-29-observability-stack-design.md](../specs/2026-04-29-observability-stack-design.md)

---

## File Structure

新建独立仓库 `~/projects/ynet/obs-stack/`：

```
obs-stack/
├── docker-compose.yml          # 主 compose，bridge 网络 + 3 个服务
├── prometheus.yml              # Prometheus 配置 + scrape target 列表
├── datasources/
│   └── prometheus.yml          # Grafana 数据源（指向本地 prom）
├── dashboards/
│   ├── dashboards.yml          # Grafana provider 配置
│   └── host-node.json          # 主机指标看板（Grafana 1860）
├── .env.example                # GRAFANA_ADMIN_PASS 等
├── .gitignore
└── README.md                   # 启停 / 端口 / 推广
```

> 4 个应用的 dashboard（`studio.json`/`loop.json`/`guard-go.json`/`intent-hub.json`）由各自的 observability-* plan 添加到 `obs-stack/dashboards/`，本 plan 不创建，但 provider 配置覆盖该目录。

---

## Task 1: 初始化 obs-stack 目录结构

**Files:**
- Create: `~/projects/ynet/obs-stack/.gitignore`
- Create: `~/projects/ynet/obs-stack/.env.example`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p ~/projects/ynet/obs-stack/{datasources,dashboards}
cd ~/projects/ynet/obs-stack
git init -b main
```

Expected: `Initialized empty Git repository in /Users/luzhipeng/projects/ynet/obs-stack/.git/`

- [ ] **Step 2: 写 .gitignore**

写入 `~/projects/ynet/obs-stack/.gitignore`：
```gitignore
.env
*.log
prom-data/
grafana-data/
```

- [ ] **Step 3: 写 .env.example**

写入 `~/projects/ynet/obs-stack/.env.example`：
```bash
# Copy to .env and fill in
GRAFANA_ADMIN_PASS=changeme-strong-password
```

- [ ] **Step 4: 提交**

```bash
cd ~/projects/ynet/obs-stack
git add .gitignore .env.example
git commit -m "chore: init obs-stack repo skeleton"
```

---

## Task 2: 写 docker-compose.yml

**Files:**
- Create: `~/projects/ynet/obs-stack/docker-compose.yml`

- [ ] **Step 1: 写 compose 文件**

写入 `~/projects/ynet/obs-stack/docker-compose.yml`：
```yaml
services:
  prometheus:
    image: prom/prometheus:v2.55.0
    container_name: obs-prometheus
    restart: unless-stopped
    networks: [obs-net]
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prom-data:/prometheus
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.retention.time=30d
      - --web.enable-lifecycle

  grafana:
    image: grafana/grafana:11.3.0
    container_name: obs-grafana
    restart: unless-stopped
    networks: [obs-net]
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./datasources:/etc/grafana/provisioning/datasources:ro
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASS:-admin}
      GF_AUTH_ANONYMOUS_ENABLED: "false"

  node-exporter:
    # node_exporter 跑在被监控机上（PoC 是 220），需要 host network 采主机指标
    # 将来 obs-stack 搬到独立机器时，node-exporter 留在被监控机器上
    image: prom/node-exporter:v1.8.2
    container_name: obs-node-exporter
    restart: unless-stopped
    network_mode: host
    pid: host
    volumes:
      - /:/host:ro,rslave
    command:
      - --path.rootfs=/host

networks:
  obs-net:
    driver: bridge

volumes:
  prom-data:
  grafana-data:
```

- [ ] **Step 2: 验证 compose 语法**

```bash
cd ~/projects/ynet/obs-stack
docker compose config --quiet && echo "OK" || echo "FAIL"
```

Expected: `OK`（不会有任何输出表示 compose 配置合法；非法的话会打错）

- [ ] **Step 3: 提交**

```bash
git add docker-compose.yml
git commit -m "feat(infra): add docker-compose stack with prometheus/grafana/node-exporter"
```

---

## Task 3: 写 prometheus.yml

**Files:**
- Create: `~/projects/ynet/obs-stack/prometheus.yml`

- [ ] **Step 1: 写 scrape 配置**

写入 `~/projects/ynet/obs-stack/prometheus.yml`：
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    monitor: ynet-obs

# Targets 用宿主机 IP（不是 localhost），方便将来 obs-stack 搬到任意机器
# 推广到 224 / CDRCB 时，改这里的 IP 即可

scrape_configs:
  - job_name: prometheus
    static_configs:
      - targets: ["localhost:9090"]

  - job_name: studio
    metrics_path: /metrics
    static_configs:
      - targets: ["10.10.10.220:8888"]
        labels: { service: studio, env: dev-220 }

  - job_name: loop
    metrics_path: /metrics
    static_configs:
      - targets: ["10.10.10.220:8889"]
        labels: { service: loop, env: dev-220 }

  - job_name: guard-go
    metrics_path: /metrics
    static_configs:
      - targets: ["10.10.10.220:8180"]
        labels: { service: guard-go, env: dev-220 }

  - job_name: intent-hub
    metrics_path: /metrics
    static_configs:
      - targets: ["10.10.10.220:8000"]
        labels: { service: intent-hub, env: dev-220 }

  - job_name: node
    static_configs:
      - targets: ["10.10.10.220:9100"]
        labels: { service: host, env: dev-220 }
```

- [ ] **Step 2: 验证 prometheus 配置语法**

```bash
docker run --rm -v $(pwd):/etc/prometheus prom/prometheus:v2.55.0 \
  promtool check config /etc/prometheus/prometheus.yml
```

Expected: `Checking /etc/prometheus/prometheus.yml\n SUCCESS: ...`

- [ ] **Step 3: 提交**

```bash
git add prometheus.yml
git commit -m "feat(infra): add prometheus scrape config for 4 apps + node + self"
```

---

## Task 4: Grafana 数据源 provisioning

**Files:**
- Create: `~/projects/ynet/obs-stack/datasources/prometheus.yml`

- [ ] **Step 1: 写 datasource 配置**

写入 `~/projects/ynet/obs-stack/datasources/prometheus.yml`：
```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://obs-prometheus:9090
    isDefault: true
    editable: false
```

> 注：`obs-prometheus` 是 docker-compose 服务名，Grafana 在 obs-net bridge 网络内通过 DNS 找到 prom 容器，**不要写 localhost**（Grafana 容器内 localhost 是它自己）。

- [ ] **Step 2: 提交**

```bash
git add datasources/prometheus.yml
git commit -m "feat(grafana): provision prometheus datasource"
```

---

## Task 5: Grafana dashboards provisioning 配置

**Files:**
- Create: `~/projects/ynet/obs-stack/dashboards/dashboards.yml`

- [ ] **Step 1: 写 provider 配置**

写入 `~/projects/ynet/obs-stack/dashboards/dashboards.yml`：
```yaml
apiVersion: 1

providers:
  - name: ynet-dashboards
    orgId: 1
    folder: ynet
    folderUid: ynet
    type: file
    disableDeletion: false
    editable: true
    updateIntervalSeconds: 30
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
      foldersFromFilesStructure: false
```

- [ ] **Step 2: 提交**

```bash
git add dashboards/dashboards.yml
git commit -m "feat(grafana): provision file-based dashboards loader"
```

---

## Task 6: 导入 Host Node 仪表盘

**Files:**
- Create: `~/projects/ynet/obs-stack/dashboards/host-node.json`

- [ ] **Step 1: 下载 Grafana 1860 的 JSON**

```bash
cd ~/projects/ynet/obs-stack/dashboards
curl -fsSL "https://grafana.com/api/dashboards/1860/revisions/latest/download" \
  -o host-node.json
ls -la host-node.json
```

Expected: 文件下载成功，大小约 200KB

- [ ] **Step 2: 替换默认 datasource UID**

下载下来的 JSON 里 datasource 字段写的是 `${DS_PROMETHEUS}`（变量占位），Grafana 加载时若没设变量会报错。改成显式名字：

```bash
sed -i.bak 's/"${DS_PROMETHEUS}"/"Prometheus"/g' host-node.json
diff host-node.json.bak host-node.json | head
rm host-node.json.bak
```

Expected: `diff` 输出有变化（`${DS_PROMETHEUS}` 被替换成 `Prometheus`）

- [ ] **Step 3: 提交**

```bash
cd ~/projects/ynet/obs-stack
git add dashboards/host-node.json
git commit -m "feat(grafana): import grafana.com/1860 node-exporter dashboard"
```

---

## Task 7: 本地 smoke test

**Files:** 无新增；只验证 stack 起得来

- [ ] **Step 1: 启动 stack**

```bash
cd ~/projects/ynet/obs-stack
cp .env.example .env
docker compose up -d
sleep 10
docker compose ps
```

Expected: 3 个容器（obs-prometheus / obs-grafana / obs-node-exporter）`State=running`

- [ ] **Step 2: 验证 Prometheus**

```bash
curl -sf http://localhost:9090/-/ready && echo "OK"
curl -sf http://localhost:9090/api/v1/targets | head -c 200
```

Expected: `OK`，targets API 返回 JSON（4 个应用 target 应该是 down，因为应用还没埋点；但 prometheus 自身 + node-exporter 应 up）

- [ ] **Step 3: 验证 Grafana**

```bash
curl -sf http://localhost:3000/api/health && echo ""
```

Expected: `{"database":"ok",...}`

- [ ] **Step 4: 验证 datasource provisioned**

```bash
# 用默认 admin/<env 中的密码> 登录拿到 cookie
PASS=$(grep GRAFANA_ADMIN_PASS .env | cut -d= -f2)
curl -sf -u admin:"$PASS" http://localhost:3000/api/datasources \
  | python3 -c 'import sys,json; d=json.load(sys.stdin); print([x["name"] for x in d])'
```

Expected: `['Prometheus']`

- [ ] **Step 5: 验证 host-node dashboard 加载**

```bash
curl -sf -u admin:"$PASS" http://localhost:3000/api/search?folderIds=0 \
  | python3 -c 'import sys,json; print([d["title"] for d in json.load(sys.stdin)])'
```

Expected: 列表包含 `Node Exporter Full`（host-node.json 的 dashboard title）

- [ ] **Step 6: 浏览器肉眼确认**

```bash
echo "Open http://10.10.10.220:3000 in browser"
echo "Login: admin / $PASS"
echo "Folder 'ynet' should contain 'Node Exporter Full' dashboard"
echo "The dashboard should show CPU/memory/disk graphs (since node_exporter is up)"
```

Expected: 浏览器看 dashboard 有数据点

- [ ] **Step 7: 关停（验证完）**

```bash
docker compose down
# data volumes 留着，下次起来还在
```

---

## Task 8: 写 README

**Files:**
- Create: `~/projects/ynet/obs-stack/README.md`

- [ ] **Step 1: 写运维文档**

写入 `~/projects/ynet/obs-stack/README.md`：
```markdown
# Observability Stack

220 开发环境的统一监控栈：Prometheus + Grafana + node_exporter，
监控 4 个应用：Studio / Loop / Guard-go / Intent-Hub。

## 快速启动

```bash
cp .env.example .env
# 编辑 .env，设置 GRAFANA_ADMIN_PASS
docker compose up -d
```

UI:
- Grafana: http://10.10.10.220:3000  (admin / .env 里的密码)
- Prometheus: http://10.10.10.220:9090

## 拓扑

```
[4 个应用]    ←── scrape 15s ───    [obs-stack]
10.10.10.220                        bridge net
:8888 Studio                        (在 obs-net 内)
:8889 Loop
:8180 Guard-go
:8000 Intent-Hub
:9100 node_exporter
```

obs-stack 通过宿主 IP scrape 各应用的 `/metrics`，
将来 obs-stack 整体搬迁到任意机器只需改 `prometheus.yml` 中的 IP。

## 文件

- `docker-compose.yml` — 服务定义
- `prometheus.yml` — scrape 配置
- `datasources/prometheus.yml` — Grafana 数据源（指向 obs-prometheus 容器）
- `dashboards/dashboards.yml` — Grafana 仪表盘 provider
- `dashboards/host-node.json` — 主机指标（Grafana 1860）
- `dashboards/<app>.json` — 各应用仪表盘（由各 observability-* plan 添加）

## 操作

```bash
# 起
docker compose up -d

# 停
docker compose down

# 重载 Prometheus 配置（修改 prometheus.yml 后）
curl -X POST http://localhost:9090/-/reload

# 看抓取状态
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job:.labels.job, health}'

# 查日志
docker compose logs prometheus
docker compose logs grafana
```

## 推广到其他环境

把整个 `obs-stack/` 目录拷到目标机器，改两处：
1. `prometheus.yml` 的 `targets` IP（改成目标网络里被监控应用的 IP）
2. `.env` 的 `GRAFANA_ADMIN_PASS`

被监控应用不需要任何修改，只要 `/metrics` endpoint 在新 IP 上能通。

## Backup

数据卷：
- `prom-data` — 30 天指标
- `grafana-data` — dashboards 状态、用户、annotations

```bash
docker run --rm -v obs-stack_grafana-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/grafana-$(date +%Y%m%d).tar.gz -C /data .
```
```

- [ ] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: add obs-stack README with topology and ops guide"
```

---

## Self-Review

完成 8 个 task 后核对：

1. **Spec coverage**:
   - ✅ docker-compose（spec §3 / Task 2）
   - ✅ prometheus.yml 用宿主 IP（spec §3 / Task 3）
   - ✅ Grafana datasource（spec §3 / Task 4）
   - ✅ Grafana dashboards provider（spec §3 / Task 5）
   - ✅ host-node 仪表盘（spec §4 / Task 6）
   - ⏳ 4 个应用 dashboard 由各 observability-* plan 添加（不在本 plan 范围）

2. **Placeholder scan**:
   - prometheus.yml 中 `10.10.10.220` 是 PoC 阶段固定 IP，不算 placeholder（搬迁时改）
   - `${GRAFANA_ADMIN_PASS:-admin}` 是 docker compose 默认值语法，不算 placeholder

3. **Type/path consistency**:
   - 容器名一致（obs-prometheus / obs-grafana / obs-node-exporter）
   - 网络名一致（obs-net）
   - 路径一致（`/etc/grafana/provisioning/datasources` 和 `/etc/grafana/provisioning/dashboards`）

无遗漏。
