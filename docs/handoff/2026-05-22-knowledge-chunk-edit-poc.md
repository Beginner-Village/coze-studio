# 知识库切片编辑 + 召回优化 — 会话交接文档

> **状态**：Spec B（后端召回 fix）✅ 已在 POC 部署+验证；Spec A（前端切片合并+状态徽章）❌ 代码 ready 但 web 镜像未推 POC（npm install rush 阶段网络卡死被 kill）
>
> **分支**：`feat/wip`（领先 `ynet-main` 20 commits）
>
> **日期**：2026-05-22

---

## 1. POC 环境（K8s / KubeSphere）

| 项 | 值 |
|---|---|
| 集群类型 | KubeSphere 管理的 K8s |
| 企业空间 | `ibbp-workspace` |
| Namespace | `poc` |
| KubeSphere 控制台 | http://console.k8s.ynet.io |
| KubeSphere 账号 | `ibbp-admin` / `hEw#hj9p2` |
| Harbor 镜像仓库 | http://harbor.ynet.io:8000/harbor/projects/12/repositories |
| Harbor 项目（路径前缀）| `bbw-poc-ai`（即 `harbor.ynet.io:8000/bbw-poc-ai/...`）|
| Harbor 账号 | `admin` / `Harbor12345` |
| Studio（猎鹰智能体平台）| http://ai-agent.poc.k8s.ynet.io |
| Studio 测试账号 | `351220960@qq.com` / `123456` |
| Grafana | http://grafana.poc.k8s.ynet.io |
| 测试知识库 URL | `/space/7639215472289775616/knowledge/7640007395921362944`（"test" 知识库，2 doc / 277 chunk / 327 历史命中）|

### 关键 K8s 部署

| Deployment 名 | 镜像名 | 当前 tag | 说明 |
|---|---|---|---|
| `ynet-studio-server` | `harbor.ynet.io:8000/bbw-poc-ai/**studio-server**` | `2026-05-22-r1-chunk-merge-retrieval-fix` ✅ 新 | 后端，已上 Spec B fix |
| `ynet-studio-web` | `harbor.ynet.io:8000/bbw-poc-ai/**ynet-web**` | `2026-05-22-r1-card-url`（旧）| 前端，**Spec A 还没上** |

⚠️ **deployment 名 ≠ 镜像名**：deployment 叫 `ynet-studio-server`，镜像叫 `studio-server`（中间没有 `ynet-` 前缀）；web 同理 `ynet-studio-web` vs `ynet-web`。

### Mac 一次性环境配置

```bash
# 1. hosts（指向 POC Harbor 真实 IP）
sudo sh -c 'echo "192.168.156.41 harbor.ynet.io" >> /etc/hosts'

# 2. Harbor 登录
docker login harbor.ynet.io:8000 -u admin -p Harbor12345

# 3. buildx insecure config（已加到 ynet-docker/buildkitd.toml，新会话拿到代码后用）
docker buildx rm ynet-amd64-builder 2>/dev/null || true
docker buildx create --name ynet-amd64-builder --use --platform linux/amd64 \
  --config ynet-docker/buildkitd.toml
```

### 本地 kubectl 注意

本机 `~/.kube/ynet-config` 是 dev/sit 集群（namespace `dev-ai`/`sit-ai`），**不能** `kubectl -n poc ...`。POC 集群操作只能走 KubeSphere Web 控制台。

---

## 2. 这次会话做了什么

### 2.1 设计 / 规划阶段

走完 `superpowers:brainstorming` → `writing-plans` → `subagent-driven-development` 全流程。三次 scope 调整：

1. 一开始以为"text/image-workspace 没有切片编辑入口" → verify 后发现 **已经有编辑入口**（text 用 LevelTextKnowledgeEditor，image 用 PhotoDetailModal+UpdatePhotoCaption）
2. 客户真实痛点是 (a) 切片**合并**没有 (b) 召回质量差（"aaa 召回 AI 内容且高分"）
3. 决定拆 2 个 spec 并行：Spec A 切片编辑链路（合并+状态徽章）+ Spec B 召回优化

### 2.2 代码（已 commit 到 `feat/wip`，**20 个 commits**）

#### Spec A：切片合并 + 状态徽章（前端为主）

| Commit | 内容 |
|---|---|
| `97cf699d0` | `useMergeSlices` hook with two-step merge + retry |
| `e52ff1f16` | wire useMergeSlices into barrel + error reporting |
| `d61f32de4` | `MergeSliceConfirmModal` with content preview |
| `f0c437880` | MergeSliceConfirmModal i18n + extract styles + testid queries |
| `1f9c6049a` | wire merge toolbar into text-workspace |
| `493cb4eac` | wire merge toolbar into table-workspace |
| `a96fbe257` | `SliceStatusBadge` with 4 states + retry + i18n |
| `86145e2e9` | `useSliceStatusPolling` hook (5s interval, 120s timeout) |
| `8493e25ef` | integrate SliceStatusBadge + polling in 3 workspaces |
| `c6a083ce9` | refactor: lift `ReindexStatusBar` + `useReindexTracking` to shared |

关键路径：
- `frontend/packages/data/knowledge/knowledge-modal-base/src/merge-slice-confirm-modal/`
- `frontend/packages/data/knowledge/knowledge-modal-base/src/hooks/use-merge-slices.ts`
- `frontend/packages/data/knowledge/knowledge-modal-base/src/components/slice-status-badge.tsx`
- `frontend/packages/data/knowledge/knowledge-modal-base/src/hooks/use-slice-status-polling.ts`
- 三个 workspace 的接入点：`text-knowledge-workspace/`、`table-knowledge-workspace/`、`image-knowledge-workspace/`

#### Spec B：召回质量 Phase 1 修复（后端）

| Commit | 内容 |
|---|---|
| `5948d0222` | `isJunkQuery` guard at Retrieve entry — 短 query / 纯标点 / 纯重复字符直接返回空 |
| `8bec3f526` | MinScore floor at 0.3 + agent template 默认值提升到 0.3 |
| `41734a3c5` | **fix**: ES retrieve 中 top1 一直被归一化成 1.0 的 bug → 保留 raw BM25 score |
| `16fb584a4` | log empty results with query — 方便后续 bad case 挖矿 |
| `09c41b2c9` | dump query + per-channel top3 for bad case mining |

关键路径：
- `backend/domain/knowledge/service/retrieve.go`
- `backend/domain/knowledge/service/retrieve_helpers.go`（新增 isJunkQuery 等）

#### 设计文档（已 commit）

- `docs/superpowers/specs/2026-05-22-knowledge-chunk-edit-design.md`（初版 spec，已被 v2 替换内容）
- `docs/superpowers/specs/...` — Spec A v2 / Spec B 调研 / Spec B impl plan（详见 commit `d2659004d`/`dc074a209`/`6fd25aaed`/`88ae632fc`）

### 2.3 POC 部署脚手架（gitignored，**仅本地**）

`.gitignore` 把 `ynet-docker/` 整个目录排除了，下面这些文件只在本地：

| 文件 | 作用 |
|---|---|
| `ynet-docker/build-and-push-poc.sh` | 一键 build + push 两个镜像到 bbw-poc-ai |
| `backend/Dockerfile.poc` | 精简 backend Dockerfile（**去掉 fe-builder stage**，POC 是 split 模式不需要 backend embed 前端 dist，build 时间从 ~30min 降到 ~8min）|
| `ynet-docker/buildkitd.toml` | 加了 `harbor.ynet.io:8000` insecure registry config（**必需**，否则 buildx 走 HTTPS 失败）|
| `ynet-docker/Dockerfile.web.poc-card-url` | POC 前端 overlay（已有，不动）|

镜像 tag 习惯：`2026-05-22-r1-<功能描述>` 比如 `2026-05-22-r1-chunk-merge-retrieval-fix`

### 2.4 POC 部署成果

| 镜像 | tag | 状态 |
|---|---|---|
| `studio-server` | `2026-05-22-r1-chunk-merge-retrieval-fix` | ✅ **已推送** |
| `ynet-web` | `2026-05-22-r1-chunk-merge-retrieval-fix` | ❌ **未推送**（build 多次卡死被 kill）|

#### 已切换的 deployment

✅ `ynet-studio-server` 已经指向新 tag，pod `ynet-studio-server-b5c7655c7-7lvjs` 在跑。

❌ `ynet-studio-web` 还在跑老镜像 `ynet-web:2026-05-22-r1-card-url`。

### 2.5 Playwright 验证（已通过）

| Query | 期望 | 实际 |
|---|---|---|
| `aaa` | 空（isJunkQuery 拦截）| ✅ "暂无命中结果" |
| `!!!` | 空（all punct）| ✅ "暂无命中结果" |
| `test` | 命中 chunks | ✅ 命中 3 个（前期股本.xlsx 等）|

→ Spec B 在 POC 已经验证生效，acceptance gate 满足。

---

## 3. 已知踩坑

1. **POC 是 K8s split 模式**：前端走 nginx pod，后端走 server pod。所以 backend 镜像里 embed 前端 dist 是浪费，用 `backend/Dockerfile.poc` 跳过 fe-builder
2. **buildx 默认 HTTPS**，POC Harbor 是 HTTP:8000：
   - `ynet-docker/buildkitd.toml` 加 `[registry."harbor.ynet.io:8000"]` http=true insecure=true
   - 脚本里 `--push` 改成 `--output type=registry,registry.insecure=true`
3. **deployment 名 ≠ 镜像名**（deployment `ynet-studio-server` ↔ image `studio-server`）
4. **`ynet-docker/` 被 `.gitignore`**：部署脚本是 local-only，跨机器要手动复制
5. **npm install rush 不稳**：`Dockerfile.web.poc-card-url` 的 fe-builder stage 装 rush 会 ECONNRESET，多次重试；或者超时被 kill
6. **本地 kubectl 不通 POC**：只能用 KubeSphere 控制台或 playwright 操控网页
7. **Mac M 系列必须** `docker buildx --platform linux/amd64`，否则起 pod 会 ImagePullBackOff

---

## 4. 还缺什么

### 4.1 Web 镜像没推到 POC（核心阻塞）

- 卡在 `Dockerfile.web.poc-card-url` 的 `npm install -g @microsoft/rush` 步骤
- 多次失败原因：ECONNRESET（network reset）/ 超时 / 被 kill
- 影响：Spec A 的切片合并按钮、SliceStatusBadge、polling 在 POC **看不到**

### 4.2 Spec A 在 POC 没法验证

- 切片合并 UI（text/table workspace 列表行多选 + 合并按钮 + MergeSliceConfirmModal）
- 编辑/合并后的状态徽章（"重新索引中" 转圈 → "Done"）
- 失败重试按钮

### 4.3 还没合回 `ynet-main`

20 个 commits 留在 `feat/wip`，没合并到主干。

### 4.4 设计阶段遗留 P1 项（不阻塞这一波，长期改进）

- Spec A 原本规划的 image-workspace UpdatePhotoCaption 是否真触发 indexSlice — verify 任务 `#7` 还 pending
- Spec B 后续可以加 hybrid 召回融合调参、rerank 模型升级、query rewrite、chunk size/overlap 调优（独立 spec，按客户 bad case 驱动）

---

## 5. 后续要做什么（新会话执行清单）

### Step 1: 解决 web 镜像 push 不上的问题

`ynet-docker/build-and-push-poc.sh` 重跑，如果还卡在 npm install rush，试以下任一 mitigation：

**Option A：换备用 npm registry**

改 `ynet-docker/Dockerfile.web.poc-card-url` 第 11 行 `npm config set registry`：

```dockerfile
RUN npm config set registry https://registry.npm.taobao.org \
    && npm config set fetch-retries 5 \
    && npm config set fetch-retry-mintimeout 20000 \
    && npm config set fetch-retry-maxtimeout 120000
```

**Option B：拆步加 retry**

```dockerfile
RUN for i in 1 2 3 4 5; do \
      npm install -g @microsoft/rush && break || sleep 10; \
    done
```

**Option C：用 host 网络**

build 命令加 `--network=host`（buildkitd 已经允许 network.host entitlement）

**Option D：本地先装好 rush 再 COPY 进去**

复杂，但最稳。优先 A+B。

### Step 2: push 成功后切 deployment image tag

用 playwright（参考本会话）：
1. 登录 `http://console.k8s.ynet.io`（ibbp-admin / hEw#hj9p2）
2. 导航到 `/ibbp-workspace/clusters/default/projects/poc/deployments/ynet-studio-web`
3. 编辑 YAML，把 `image:` 改成 `harbor.ynet.io:8000/bbw-poc-ai/ynet-web:2026-05-22-r1-chunk-merge-retrieval-fix`
4. 保存 → pod 自动滚动

### Step 3: 验证 Spec A（playwright 或人工）

进 `http://ai-agent.poc.k8s.ynet.io/space/7639215472289775616/knowledge/7640007395921362944`

- **合并切片**：text-workspace 多选 2 个相邻 chunk → 点合并按钮 → MergeSliceConfirmModal 弹出 → 确认 → 看新 chunk 生成 + 老 2 个消失
- **状态徽章**：编辑任意 chunk → 保存 → 看行尾出现"重新索引中"转圈 → 5 秒后变 Done（轮询）
- **失败重试**：制造 embedding 失败场景（比如断 embedding 服务）→ 看 SliceStatusBadge 显示"重试"按钮 → 点击重新触发

### Step 4: 合回 `ynet-main`

`feat/wip` 上 20 个 commits review OK 后：

```bash
git checkout ynet-main
git pull origin ynet-main
git merge --no-ff feat/wip -m "feat(knowledge): chunk merge + status badge + retrieval quality fix"
git push origin ynet-main
```

（或者 PR 走 review 流程，看团队规范。）

### Step 5: 重命名 / 删除 `feat/wip` 分支

合完之后：
```bash
git branch -d feat/wip
git push origin :feat/wip  # 删远程
```

或者保留改名 `feat/2026-05-knowledge-chunk-edit-and-retrieval-fix` 作 archive。

---

## 6. 相关文件速查

### 代码
- 后端召回：`backend/domain/knowledge/service/retrieve.go` + `retrieve_helpers.go`
- 前端切片编辑/合并：`frontend/packages/data/knowledge/knowledge-modal-base/src/`
  - `merge-slice-confirm-modal/`
  - `hooks/use-merge-slices.ts`
  - `hooks/use-slice-status-polling.ts`
  - `components/slice-status-badge.tsx`
  - `components/reindex-status-bar.tsx`
  - `hooks/use-reindex-tracking.ts`
- 三个 workspace 接入：`frontend/packages/data/knowledge/knowledge-ide-base/src/features/{text,table,image}-knowledge-workspace/`

### 设计 / 计划文档（已 commit）
- `docs/superpowers/specs/2026-05-22-knowledge-chunk-edit-design.md`
- `docs/superpowers/specs/...spec-a-v2...md` / `...spec-b...md`
- `docs/superpowers/plans/2026-05-22-...md`

### POC 部署（**本地** ynet-docker/ gitignored）
- `ynet-docker/build-and-push-poc.sh`
- `ynet-docker/buildkitd.toml`
- `ynet-docker/Dockerfile.web.poc-card-url`
- `backend/Dockerfile.poc`

### 本交接文档
- `docs/handoff/2026-05-22-knowledge-chunk-edit-poc.md`（你正在读）

---

## 7. 上下文背景（如果新人接手）

- 客户：成都农商银行（CDRCB），但 POC 是云端 K8s 集群，不是银行内网
- 客户痛点：知识库召回准（aaa 不该命中 AI）+ 切片可以手动改/合并/看到 re-indexing 状态
- POC 环境性质：客户验收前的最后落地，跟 224 生产、220 开发完全独立
- 部署习惯：Mac 本机 build → push Harbor → KubeSphere 控制台改 deployment image tag → pod 自动滚动
