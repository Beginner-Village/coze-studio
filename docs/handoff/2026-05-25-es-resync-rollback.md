# ES Resync 功能 — 应急回滚 / 故障处理手册

**Date**: 2026-05-25
**Branch**: feat/es-resync
**适用对象**: 现场实施 / 二线运维

> 适用场景: 现场跑了 "重新同步 ES" 按钮 (UI 入口在 Space 详情页 → 数据维护) 后, 发现:
>
> - 智能体 / 资源 / 知识库列表数据消失或不全
> - 知识库召回 (recall) 结果不对 / 命中明显减少
> - 后端 ynet-server 持续报 `[ResyncES]` 相关 error
> - 切片状态 (`knowledge_document_slice.status`) 长时间卡在非 Done 状态
>
> 这是恢复步骤。**先做 §0 evidence 收集再做任何操作**, 不要直接重启 / 删数据。

---

## 0. 第一时间 evidence 收集

在做任何操作前, 先拉以下信息存档 (现场把输出贴到工单 / 群里):

### 0.1 后端日志

```bash
# 假设 ynet-server 用 docker compose 跑
docker logs ynet-server 2>&1 | grep -i 'ResyncES\|resync\|delete_by_query' | tail -200

# 或 K8s
kubectl logs -n ynet -l app=ynet-server --tail=500 | grep -i 'resync\|delete_by_query'
```

关键关键字:
- `[ResyncES] space=<X> done: project_draft=N coze_resource=N kb_entries=N slice_jobs=N` → 同步成功的最终汇总
- `list-index rebuild failed` → 步骤 (b) 失败, 这种情况 ES 可能处于部分清空
- `slice resync partial fail` → 步骤 (c) 失败, 列表已重建但切片重嵌入失败
- `delete_<index> for space <X>` → delete_by_query 报错

### 0.2 MySQL 当前状态

```sql
-- 1. 该 space 下智能体 / 资源 / 知识库的"源真理"行数
USE opencoze;

SELECT space_id, COUNT(*) AS agents
FROM single_agent_draft
WHERE space_id = <SPACE_ID> AND deleted_at IS NULL;

SELECT space_id, COUNT(*) AS apps
FROM app_draft
WHERE space_id = <SPACE_ID> AND deleted_at IS NULL;

SELECT space_id, COUNT(*) AS kbs
FROM knowledge
WHERE space_id = <SPACE_ID> AND deleted_at IS NULL;

-- 2. 切片状态分布 (重点: Init / Processing / Failed / Done 各多少)
SELECT
  k.space_id,
  s.status,
  COUNT(*) AS cnt
FROM knowledge_document_slice s
JOIN knowledge_document d ON d.id = s.document_id
JOIN knowledge k ON k.id = d.knowledge_id
WHERE k.space_id = <SPACE_ID>
GROUP BY k.space_id, s.status
ORDER BY s.status;

-- status 取值参考: Init=0, Processing=1, Done=2, Failed=3
```

### 0.3 ES 当前状态

注: 实际索引名 — 列表索引固定 3 个: `project_draft`, `coze_resource`, `kb_entries`；切片索引按 KB 维度分: `openynet_<kb_id>`。

```bash
ES_HOST=http://192.168.1.220:9200  # 现场改成实际地址

# 1. 集群健康
curl -s "$ES_HOST/_cluster/health?pretty"

# 2. 三个列表索引的总文档数 (不限 space, 是全集)
curl -s "$ES_HOST/_cat/indices/project_draft,coze_resource,kb_entries?v&h=index,docs.count,docs.deleted,store.size"

# 3. 该 space 在三个列表索引中的实际文档数
for IDX in project_draft coze_resource; do
  echo "== $IDX space=<SPACE_ID> =="
  curl -s "$ES_HOST/$IDX/_count" -H 'Content-Type: application/json' \
    -d '{"query":{"term":{"space_id":<SPACE_ID>}}}'
done

# 4. kb_entries 用 kb_id 过滤 (先从 MySQL 拿到这个 space 的 kb_ids 列表)
KB_IDS='1,2,3'  # 替换
curl -s "$ES_HOST/kb_entries/_count" -H 'Content-Type: application/json' \
  -d "{\"query\":{\"terms\":{\"kb_id\":[$KB_IDS]}}}"

# 5. 切片索引 (按 kb_id 一个一个看)
for KB in $KB_IDS; do
  echo "== openynet_$KB =="
  curl -s "$ES_HOST/_cat/indices/openynet_$KB?v&h=index,docs.count,docs.deleted,store.size" 2>/dev/null
done
```

### 0.4 RocketMQ consumer 状态

切片重嵌入是异步走 MQ 的, consumer 不动 = 切片永远不会变 Done。

```bash
# 看 broker 是否健康
docker logs rocketmq-broker --tail=100 2>&1 | grep -iE 'error|warn'

# K8s 同样道理
kubectl logs -n ynet -l app=rocketmq-broker --tail=200

# 关键 topic (consts.go 里硬编码的):
#   openynet_search_app         — 资源列表事件 (coze_resource)
#   openynet_search_resource    — 项目列表事件 (project_draft)
#   openynet_knowledge          — 切片重嵌入事件 (开 N 个 openynet_<kb_id>)
```

把上面四块 (后端 log / MySQL 行数 / ES 行数 / MQ 状态) 一起截下来, 后面修复决策都靠这些数。

---

## 1. 完全回滚到 sync 前状态

这是 "上一步操作不可接受, 必须复原" 的情况。

### 1.1 前提: 你 sync 前做了 backup

#### ES 备份恢复 (推荐用 snapshot)

如果之前配过 ES snapshot repository, 直接还原:

```bash
ES_HOST=http://192.168.1.220:9200
REPO=ynet-backup        # 现场实际名
SNAP=before-resync-20260525

# 1. 关闭三个列表索引 (避免还原冲突)
curl -X POST "$ES_HOST/project_draft,coze_resource,kb_entries/_close"

# 2. 还原
curl -X POST "$ES_HOST/_snapshot/$REPO/$SNAP/_restore" \
  -H 'Content-Type: application/json' \
  -d '{
    "indices": "project_draft,coze_resource,kb_entries",
    "include_global_state": false
  }'

# 3. 打开
curl -X POST "$ES_HOST/project_draft,coze_resource,kb_entries/_open"
```

如果用的是 `elasticdump`:

```bash
# 还原 (假设 backup 存在 /opt/backup/es/*.json)
for IDX in project_draft coze_resource kb_entries; do
  npx elasticdump \
    --input=/opt/backup/es/$IDX-data.json \
    --output=$ES_HOST/$IDX \
    --type=data
done
```

#### MySQL 备份恢复

只需要还原该 space 关联的几张表 (不要全库回滚, 会带走别的 space 的最新写入):

```bash
# 假设有 mysqldump 出来的 .sql, 想要 cherry-pick 表
mysql -h <HOST> -u root -p opencoze < /opt/backup/mysql/single_agent_draft.sql
mysql -h <HOST> -u root -p opencoze < /opt/backup/mysql/app_draft.sql
mysql -h <HOST> -u root -p opencoze < /opt/backup/mysql/knowledge.sql
mysql -h <HOST> -u root -p opencoze < /opt/backup/mysql/knowledge_document.sql
mysql -h <HOST> -u root -p opencoze < /opt/backup/mysql/knowledge_document_slice.sql
```

> 提示: 知识库切片表 (`knowledge_document_slice`) 可能很大, 还原前 `wc -l` 一下确认导出文件完整。

#### 还原后必做的 sanity check

```sql
-- MySQL: 跟 §0.2 同样的查询, 确认行数已经回到 backup 时的状态
SELECT COUNT(*) FROM single_agent_draft WHERE space_id = <SPACE_ID> AND deleted_at IS NULL;
```

```bash
# ES: 跟 §0.3 同样的查询, 确认 docs.count 已经回到 backup 时的状态
curl -s "$ES_HOST/_cat/indices/project_draft,coze_resource,kb_entries?v"
```

### 1.2 没 backup 怎么办

好消息: **ES 数据是从 MySQL 衍生的**, 只要 MySQL 还健康, 可以重新触发 reindex 把 ES 重建一遍。

也就是说: **再点一次 "重新同步 ES" 按钮就行**。Resync 操作是幂等的 (步骤 (b) 先 delete_by_query, 步骤 (c)(d)(e) 全量重写), 跑两遍跟跑一遍结果一样。

如果按钮被 disable / 不在身边, 直接调 API:

```bash
# 现场拿 owner 账号登录后, 从浏览器 devtools 抓 cookie/i_passport_csrf_token
curl -X POST "$STUDIO_HOST/api/space/resync_es" \
  -H 'Content-Type: application/json' \
  -H 'Cookie: <复制现场实际 cookie>' \
  -d '{"space_id":<SPACE_ID>}'
```

返回示例:

```json
{
  "code": 0,
  "msg": "success",
  "counts": {
    "project_draft": 4,
    "coze_resource": 70,
    "kb_entries": 6,
    "slice_reindex_jobs": 231
  }
}
```

`slice_reindex_jobs` 是已入队 MQ 的任务数, **不代表已完成**。完成与否看 §0.2 切片状态分布。

> 兜底注意: 如果连 MySQL 都不健康, 那 ES rebuild 也救不了, 必须先恢复 MySQL (走 DBA 标准 PITR 流程)。

---

## 2. 部分修复 (典型故障)

### 2.1 列表索引重建成功, 但 chunk re-embedding 失败

**症状**: `[ResyncES] space=X done:` log 里 `project_draft / coze_resource / kb_entries` 都有数, 但 `knowledge_document_slice.status` 卡在 `Init` (0) 不变 `Done` (2)。UI 表现: 列表数据回来了, 但知识库召回还是空 / 不完整。

**诊断**:

```sql
SELECT status, COUNT(*) AS cnt
FROM knowledge_document_slice s
JOIN knowledge_document d ON d.id = s.document_id
JOIN knowledge k ON k.id = d.knowledge_id
WHERE k.space_id = <SPACE_ID>
GROUP BY status;
```

如果 `Init` 持续 > 5 分钟不下降, **基本就是 MQ consumer 没在跑 / embedding 服务挂了**。

**修复**:

1. 检查 RocketMQ broker:
   ```bash
   docker logs rocketmq-broker --tail=200 2>&1 | grep -iE 'error|exception'
   # 看到 "TOO_FREQUENT_PULL" / "consumer not exist" 之类的就是 consumer 掉了
   ```

2. 检查 embedding 服务可达性 (现场 LLM/embedding 是配在 model_meta 表的, conn_config JSON 里有 endpoint):
   ```bash
   docker logs ynet-server --tail=500 2>&1 | grep -iE 'embedding|conn_config|model_meta' | tail -30
   ```
   常见报错: `dial tcp <embedding_host>:<port>: i/o timeout`, `401 Unauthorized` (API key 失效)。

3. 最直接的修复 — 重启 ynet-server 让 MQ consumer 重连:
   ```bash
   # docker compose
   docker compose restart ynet-server
   # K8s
   kubectl rollout restart deploy ynet-server -n ynet
   ```

4. 重启完看 5 分钟内 `Init` 数量是否下降, 还是不动就要去查 model_meta 的 endpoint 配置 (走 §3.2 路径)。

### 2.2 某个特定 ES doc 漏了 (sync 完发现某个 agent / 资源 / KB 在 UI 看不到)

**症状**: sync 报 `code: 0`, counts 也对得上 MySQL 总数, 但 owner 反馈 "我那个智能体 / 应用 / 知识库不见了"。

**诊断 — 先确认是 ES 缺数据还是 MySQL 本身就被软删了**:

```sql
-- agent
SELECT id, space_id, name, deleted_at
FROM single_agent_draft
WHERE id = <BOT_ID>;

-- app
SELECT id, space_id, name, deleted_at
FROM app_draft
WHERE id = <APP_ID>;

-- kb
SELECT id, space_id, name, deleted_at
FROM knowledge
WHERE id = <KB_ID>;
```

如果 `deleted_at IS NOT NULL` → 是用户自己删过, 不是 sync 漏掉。直接告知用户即可。

如果 MySQL 有数 ES 没有:

```bash
# 现场用 ES query 验证
curl -s "$ES_HOST/project_draft/_doc/<BOT_ID>"
# 404 → 确实漏了
```

**修复 — 单 doc 重写**:

最简单还是再触发一次 space-wide resync (整个 space 重跑只多花 1-2 分钟):

```bash
curl -X POST "$STUDIO_HOST/api/space/resync_es" \
  -H 'Cookie: ...' \
  -d '{"space_id":<SPACE_ID>}'
```

如果不想动整个 space, 可以做 "保存一次该资源触发增量 indexer":
- agent: UI 打开该智能体, 改一下描述再保存
- app: UI 打开该 App draft, 编辑保存
- kb: 在 KB 详情页改一下名字保存

(这些保存动作会推到 `openynet_search_app` / `openynet_search_resource` 等 MQ topic, consumer 会 upsert ES。)

### 2.3 ES delete_by_query 跑一半 server crash 了

**症状**: 后端 log 里出现 `delete project_draft for space X: context canceled` / `connection reset by peer`。ES 处于 "部分清空" 状态 — 三个列表索引可能 project_draft 清完了, coze_resource 清一半, kb_entries 没动。

**修复 — resync 操作是幂等的, 再点一次按钮**:

代码层面 `domain/search/service/resync.go` 的设计是:
1. `DeleteByQuery` 全部 indices (失败就 return err, 不再继续)
2. 从 MySQL `ListBySpaceID` 拿全量
3. 逐条 `Create/Upsert`

所以重跑一次:
- 已删的索引会再被 delete (no-op, 0 docs deleted)
- 没删完的索引会被删干净
- 然后全部从 MySQL 重新写

**只要 MySQL 健康, 重跑就是安全的**, 不会丢数据。

```bash
curl -X POST "$STUDIO_HOST/api/space/resync_es" \
  -H 'Cookie: ...' \
  -d '{"space_id":<SPACE_ID>}'
```

### 2.4 跑完 sync 后 HTTP 返回 504 / timeout, 但其实后端在继续跑

**症状**: 前端 Toast 显示 "同步失败: timeout" (来自 `Toast.error`), 但 5 分钟后再 `_cat/indices` 看 ES, docs.count 其实已经对上了。

**原因**: nginx / ingress 的 client timeout 默认 60s, 大 space (>100 KB or >10k slices) 跑完整 sync 可能超过这个。HTTP 已经断开, 但 ynet-server 进程在继续完成 delete + reindex + 入队。

**判断**:

```bash
# 看后端 log 有没有最终的 "done" 行
docker logs ynet-server --tail=1000 2>&1 | grep "\[ResyncES\] space=<SPACE_ID> done"
```

如果有 `done:` 行 → sync 实际成功, 前端误报。告知用户刷新页面看实际效果。
如果没有 → sync 半路挂了, 走 §2.3。

---

## 3. 已知 gotcha

### 3.1 跨 space 数据残留

**场景**: space 之前 owner 是 A, 后来用户 X 把它转给 B (现在 owner_id=B)。理论上 ES 里旧的 doc 也应该 owner_id=B, 但如果在 owner 切换之前 ES 已经写过, 历史 doc 可能还是 owner_id=A。

**触发 sync 修复**: `delete_by_query{space_id: X}` 是按 space_id 过滤 (不看 owner), 会把这个 space 的所有 doc 一锅清掉, 然后从 MySQL rewrite, owner_id 全部用最新的 B。**所以 sync 一遍就修了**。

如果用户报 "我把 space 转给 B 但召回结果里还是显示 owner=A" → 让他点一次 "重新同步 ES" 即可。

### 3.2 model_meta.conn_config 错配 → embedding 失败

**场景**: 知识库的切片重嵌入需要 embedding 模型, embedding 模型 endpoint 配在 `model_meta.conn_config` 这张表里的 JSON 字段。如果运维改过 LLM 服务地址 / API key 但 DB 没同步, 切片会全卡在 `Init` 或者 `Failed`。

**诊断**:

```sql
SELECT id, name, conn_config
FROM model_meta
WHERE meta_type = 'embedding' AND status = 1;
```

确认每个 active 的 embedding 模型, `conn_config` JSON 里:
- `base_url` 是当前可达地址
- `api_key` 没过期
- `model` 名字跟现场实际 LLM 服务上 deploy 的名字一致

**典型坑**:
- 现场 LLM API key 失效 → 401 Unauthorized → `Failed` 状态
- LLM 服务从 K8s svc 改为 NodePort 但 conn_config 没改 → connection refused
- 模型名拼错 (`qwen3-7B` vs `qwen3-7b`) → 400 Bad Request

修完 model_meta 后, **不需要重启 ynet-server**, 重新跑一次 resync 就会用新 conn_config。

### 3.3 切片表很大时 RocketMQ broker 内存压力

**场景**: 一个 space 有 5000+ slices, sync 一次性把 5000 个 reindex 任务塞进 `openynet_knowledge` topic, broker store 占用瞬间飙升, 可能撑爆 8GB 内存的小机器。

**诊断**: `docker stats rocketmq-broker` 看 mem 使用。

**当前 mitigation** (代码层面没做限流, 是已知问题): 大 space resync 前手动分 KB 触发 (用户点 KB 列表里单个 KB 的 "重建索引" 而不是 space-wide 按钮)。

写在这里是因为 §5 的扩展方向会提到 — 长期方案是 async task + 分批入队。

### 3.4 RocketMQ topic 不存在 (新部署忘了建)

**场景**: 全新环境第一次 deploy, RocketMQ 启动了但没 auto-create topic 权限, sync 会报:

```
slice resync partial fail: send to topic openynet_knowledge: TOPIC_NOT_EXIST
```

**修复**:

```bash
# 进 broker 容器
docker exec -it rocketmq-broker bash

# 用 mqadmin 建 topic
sh mqadmin updateTopic -n localhost:9876 -t openynet_knowledge -c DefaultCluster
sh mqadmin updateTopic -n localhost:9876 -t openynet_search_app -c DefaultCluster
sh mqadmin updateTopic -n localhost:9876 -t openynet_search_resource -c DefaultCluster
```

(deploy-v2 的标准脚本里已经包了这步, 这里只是出问题时手动补救。)

---

## 4. 监控建议 (post-sync)

跑完一次 resync, 建议看以下指标确认健康:

### 4.1 ES 集群健康

```bash
curl -s "$ES_HOST/_cluster/health?pretty"
```

- `status: green` 或 `yellow` (单节点 dev 环境一般 yellow) → OK
- `status: red` → 有 primary shard 失败, 立刻查 `_cluster/allocation/explain`

### 4.2 切片状态分布 (sync 后 5 分钟拉一次, 15 分钟再拉一次)

```sql
SELECT status, COUNT(*)
FROM knowledge_document_slice s
JOIN knowledge_document d ON d.id = s.document_id
JOIN knowledge k ON k.id = d.knowledge_id
WHERE k.space_id = <SPACE_ID>
GROUP BY status;
```

- `Init` (0) 应该在 5 分钟内大幅下降, 15 分钟内基本清零
- `Done` (2) 应该单调上升
- `Failed` (3) 一定要看, 出现就要走 §3.2 排查 embedding 配置

### 4.3 后端 log 关键字白名单

```bash
docker logs ynet-server --tail=2000 2>&1 | grep -iE 'non-retryable|panic|fatal'
```

应为空。出现 `non-retryable error` 就要立刻 dump 完整 log 上报。

### 4.4 (可选) Prometheus 指标

如果 obs 链路已打通 (参考 [Studio-Loop 可观测性整合](feedback_loop_studio_integration.md)):

- `ynet_es_resync_duration_seconds` — 单次 sync 耗时, 单 space < 60s 为健康
- `ynet_es_resync_failure_total{step="list_index|slice"}` — 失败计数, 应保持 0
- `ynet_mq_consumer_lag{topic="openynet_knowledge"}` — consumer lag, sync 完后 5 分钟内应回落

---

## 5. 升级方案 (将来如果出问题反复)

当前 (`feat/es-resync` 第一版) 设计是 **同步 HTTP + best-effort MQ 入队**, 对 < 5000 slices 的 space 够用。如果现场反复出现以下情况, 考虑实施 [设计 spec §5](../../docs/specs/) 里提到的扩展:

### 5.1 async task pattern (避免 30s/60s HTTP timeout)

把 `/api/space/resync_es` 改成两段:

- `POST /api/space/resync_es` → 立即返回 `task_id`, 后端 goroutine 异步跑
- `GET /api/space/resync_es/status?task_id=X` → 前端轮询, 拿进度 / 完成状态

适用大 space (>200 KB / >50k slices)。

### 5.2 进度条

后端把当前阶段写到 Redis (`resync:task:<task_id>` → JSON `{stage: "delete|index|enqueue", progress: 0.7}`), 前端 polling 拉显示给用户。

### 5.3 分布式锁 (避免同一个 space 并发 sync 互相踩)

当前 Batch 2 测试已经验证了同 space 并发是 "幂等且无脏数据" 的, 但会让 ES 短暂 double-write 浪费资源。生产建议加 Redis SETNX `resync:lock:space:<id>` (TTL 5min), 同 space 第二次请求返回 `409 Conflict`。

### 5.4 数据对账 (定时 ES ↔ MySQL diff)

写个 cron 任务每天比对:

```
expected = SELECT COUNT(*) FROM single_agent_draft WHERE space_id=X AND deleted_at IS NULL
actual = curl ES/project_draft/_count {space_id: X}
if abs(expected - actual) > 0:
    alert + auto-trigger resync
```

可以提前发现 drift, 不用等用户报障。

---

## 附录: 一键 evidence dump 脚本 (现场可粘)

```bash
#!/bin/bash
# 用法: ./dump-resync-state.sh <SPACE_ID> <KB_IDS_CSV>
# 示例: ./dump-resync-state.sh 7639215472289775616 1,2,3

SPACE_ID=$1
KB_IDS=$2
ES_HOST=${ES_HOST:-http://localhost:9200}
OUT=resync-state-$(date +%Y%m%d-%H%M%S).txt

{
  echo "==== ES cluster health ===="
  curl -s "$ES_HOST/_cluster/health?pretty"

  echo "==== List indices size ===="
  curl -s "$ES_HOST/_cat/indices/project_draft,coze_resource,kb_entries?v"

  echo "==== Slice indices size ===="
  for KB in $(echo "$KB_IDS" | tr , ' '); do
    curl -s "$ES_HOST/_cat/indices/openynet_$KB?v" 2>/dev/null
  done

  echo "==== Space doc counts in ES ===="
  for IDX in project_draft coze_resource; do
    echo "-- $IDX --"
    curl -s "$ES_HOST/$IDX/_count" -H 'Content-Type: application/json' \
      -d "{\"query\":{\"term\":{\"space_id\":$SPACE_ID}}}"
  done
  echo "-- kb_entries --"
  KB_ARR=$(echo "$KB_IDS" | sed 's/,/,/g')
  curl -s "$ES_HOST/kb_entries/_count" -H 'Content-Type: application/json' \
    -d "{\"query\":{\"terms\":{\"kb_id\":[$KB_ARR]}}}"

  echo "==== Recent backend log ===="
  docker logs ynet-server --tail=300 2>&1 | grep -iE 'resync|delete_by_query|consumer' | tail -100
} > "$OUT" 2>&1

echo "wrote $OUT"
```

把这个脚本输出贴到工单里, 后端基本能直接定位问题。
