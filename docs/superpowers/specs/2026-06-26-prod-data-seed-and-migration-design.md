# 生产真实数据植入 + 部署期自适应数据库迁移 — 设计 spec

> 日期：2026-06-26 · 分支：`feat/agent-sandbox-superagent`
> 目标读者：后续在 ynet-studio 上做"真实场景填充"与"多站点上线迁移"的协作者

## 0. 一句话目标

1. **A — 真实数据植入**：把生产 224 的线上数据**整库克隆替换**进开发环境（仅创作层 + MinIO + ES），让自动化工作流智能体有真实场景可测。
2. **B — 部署期自适应迁移器**：交付一套"部署时一次性把目标库收敛到当前代码 schema、只增不删、对任意起点都成立"的迁移机制，服务 224 / 行内 CDRCB / 未来站点的上线，**现场无需任何二次开发**。

---

## 1. 现状实测（ground truth，2026-06-26 实读）

### 1.1 生产 224（导入源）
- 单机自包含部署，主机 `ubuntu-llm01`；后端镜像 `luzhipeng728/coze-studio-backend:v0211`（构建于 **2026-02-11**，已跑 7 周）。
- 自带 **MySQL 8.4.5**（容器 `coze-mysql`），库名 **`opencoze`**；DB 口令见容器 env（`MYSQL_ROOT_PASSWORD` / `MYSQL_PASSWORD`），脚本运行时探测、不写死在仓库。
- 向量 **Milvus**、对象存储 **MinIO**、MQ **NSQ**、ES 8.18、etcd、Redis，外加 hiagent plugin-runtime。
- 数据量：259 张表、**148 用户**、232 空间、395 space_user、511 agent_draft、**12,706 workflow**、806 plugin、214 知识库、30,370 会话 / 424,203 消息；重型运行数据 `node_execution` 1.25M 行/2GB、`knowledge_document_slice` 745K/872MB；**整库 ~7GB+**。
- 测试账号 `402087139@qq.com` = user_id `7532755646093983744`（首个用户，2025-07-30）。

### 1.2 开发目标库（导入目标）
- **当前服务 = 226 上的 `coze-super` 容器**（super-agent/工作流智能体后端，新构建，4h up）；其 DSN 指向 **226 本机 `coze-mysql` 容器 / 库 `openynet`**（MySQL 8.0.39，root）。**这才是导入目标**。
- `docker/.env.debug` 的 `MYSQL_HOST=10.10.10.220` 是【本地调试】配置、已与线上部署脱节，**不是目标**；220 当前还从本机不可达（隧道）。
- 226 从本机【可达】→ 224(源) 与 226(目标) 我都能连，**导入不再被 220 阻塞**。
- 226 同栈：Milvus(`coze-milvus` v2.5.4) + NSQ + Redis；向量本次不迁。

### 1.3 schema 差距（已实测）
生产 224 缺以下当前代码已有的对象（= 未来 224→上线时要补的，也 = 导入时目标侧要先建好的）：

**缺 12 张表**：`space_sync_mapping`、`space_sync_history`、`space_release`、`super_agent_user_memory`、`super_agent_session_runtime_config`、`ai_product`、`ai_product_version`、`ai_product_installation`、`ai_product_audit_log`、`skill_version`、`skill_publish_marketplace`、`skill_review_status`。

**缺 5 个列**（`single_agent_draft`，`agent_type` 同样缺在 `single_agent_version`）：
| 列 | 定义 | 对导入行的兜底 |
|---|---|---|
| `agent_type` | `VARCHAR(64) DEFAULT NULL` | NULL = 普通体 ✅ |
| `super_agent_tool_config` | `json DEFAULT NULL` | NULL ✅ |
| `source_product_id` | `bigint NOT NULL DEFAULT 0` | 0 = 普通体 ✅ |
| `source_product_version` | `varchar(64) NOT NULL DEFAULT ''` | '' ✅ |

> 兜底默认值都正确，因此 A 的导入用 `--complete-insert` 即可让生产老行平滑落入新 schema。

### 1.4 多站点漂移（B 的根本约束）
- 224：v0211，缺上面 12 表 + 5 列。
- **行内 CDRCB**：中间版本，字段比 224 还**多**一些；OceanBase、库名 `ai_studio`、`sjai` 用户**无 REFERENCES 权限**、历史上 `system_setting.key`→`setting_key` 列名漂移过。
- 各站点停在不同版本 → **禁止"版本号顺序迁移"**，必须**声明式状态收敛**。

### 1.5 已有可复用资产（不重造轮子）
- **Space 增量同步**：`backend/application/space/sync/*`、路由 `backend/api/router/space/space_sync.go`、`ynet-docker/sync-to-prod.sh`。**仅搬空间内资源，不搬 user/space 本身** → 不满足"含账号的整库导入"，故 A 走 DB 级而非此 API。
- **deploy-v2**：`ynet-docker/deploy-v2/deploy.sh` 已有 `run_mysql()`（本地 mysql 或 `ynet/mysql-client` 容器）、`CREATE DATABASE IF NOT EXISTS`、建表存在性检查、`docs/ynet-studio-seed.sql` 种子。**B 作为其一个阶段接入**。
- **schema 权威来源（注意有漂移，B 要消化）**：`docker/volumes/mysql/schema.sql`(104 表，落后最新列)、`docs/ynet-database-sql/01..105`、`docker/migrations/*.sql`、`docker/atlas/migrations/`。

---

## 2. 已确认决策

| 维度 | 决策 |
|---|---|
| 导入策略 | **merge-via-staging**（不 DROP；226 现有 6 个 dev-only 账号原样保留；雪花表直灌 carry、226 有行的自增表经 staging 去 id 重分配；删 402087139）。原"整库替换"因 226 是 coze-super 活库且需保留 dev-only 而否决 |
| 导入范围 | **仅创作层**（不含会话/消息/执行轨迹） |
| 附属数据 | **带 MinIO 文件 + ES 索引**；**不带向量(Milvus)** |
| 测试账号 | **删 `402087139`**（user + space_user + 其名下空间软删） |
| 主账号 | 用户未点名 → **执行时自动识别**：diff「dev 有、prod 无」的账号，列出+单独备份后再替换，绝不误删 |
| 生产侧 | **全程只读**（SELECT / `mysqldump --single-transaction`，绝不写 224） |

---

## 3. Part A — 真实数据植入（`seed-from-prod.sh`）

### 3.1 原则：merge（不 DROP），按表 id 方案保证零冲突/零丢失
226 是 coze-super 活库，已含 6 个 dev-only 账号(测试号但要保留) + 当前代码 schema(仅缺 4 表，forward-migrate 补)。故**不 DROP**，把生产创作层**合并**进 226：
- **雪花 id 表（66 张：`workflow_*`/`user`/`space`/`knowledge`/`plugin`…）**：`INSERT IGNORE` 直灌、carry 原 id（已验证与 226 不撞 → 零丢失）。
- **226 有行的顺序自增表（5 张：`single_agent_draft`/`skill`/`space_embedding`/`space_user`/`user_memory_config`）**：经 staging 库去 id 重分配（业务键 `agent_id`/`skill_id` 等随行保留；已验证其 id 不被外部引用 → 安全）。
- `model_template`（无业务键的全局模型配置）不导；226 空的自增表按雪花直灌 exact。
- 雪花 id 是否真不撞：已实测 226 的 33 个 workflow_meta id 在 prod 命中 0；user/space 雪花 id 段与 prod 互斥。

### 3.2 表范围 — 用"排除清单"（不会漏表）
- **排除（运行/可观测，不导）**：`conversation`、`message`、`message_*`、`node_execution`、`workflow_execution`、`run_record`、`workflow_snapshot`、`operation_log`、`audit_logs`、`chatflow_conversation_history`、`agent_conversation_mapping`、`statistics_export_file`、`data_copy_task`。
- **排除（决策）**：`knowledge_document_slice`（872MB，没向量=死文本，检索也用不上）。
- **其余全导**（创作层）：`user`/`space`/`space_user`、`single_agent_draft`(+`version`)、`workflow_meta`/`draft`/`version`/`reference`、`plugin`/`plugin_draft`/`plugin_version`/`tool*`、`knowledge`/`knowledge_document`、`folder`/`resource_folder_mapping`、`variables_meta`、`space_model`/`space_embedding`/`space_rerank`、`model_*`、`files`、`app_*`、`template`、`shortcut_command`。动态 `table_<id>`/`ragflow_*`（agent 数据库工具/RAG 运行表）**默认不灌**：它们不在权威 schema，data-only 灌会缺表；按需 `INCLUDE_DYNAMIC=1` 且改用带结构 dump。

### 3.3 流水线（merge，不 DROP）
```
[0] 预检    : 连 224(只读)+226；探测库/凭据；行数
[1] 备份    : mysqldump 226 openynet 全量 → backup/*.sql.gz（红线，可回滚）
[2] 补表    : forward-migrate 给 226 补缺的 4 张表（CREATE IF NOT EXISTS）
[3] 删冲突  : 删 226 的 402087139（解 email 唯一冲突）
[—] 分类    : 创作层表 → carry(雪花/226空) vs reassign(226 有行的自增)
[4] 直灌    : 雪花表 mysqldump --insert-ignore → 直灌 226（carry id，零丢失）
[5] 重分配  : reassign 表(5张) → staging → INSERT(去 id) SELECT 进 226
[6] 清 402  : 删合并后 prod 侧 402087139（账号删、其 space 软删）
[7] MinIO   : mc mirror 224 桶 → 226 桶（头像/知识库文档/上传件）
[8] ES      : 后端重建索引（创作层资源）
[9] 校验    : 行数 + dev-only 6 账号仍在 + 抽查空间可打开
```

### 3.4 生产只读取数（关键命令）
```bash
# 在 224 上、容器内执行；--single-transaction 一致性快照、不锁表、不写库
docker exec coze-mysql mysqldump -uroot -proot \
  --no-create-info --complete-insert --single-transaction --quick \
  --skip-triggers --set-gtid-purged=OFF --default-character-set=utf8mb4 \
  --ignore-table=opencoze.node_execution \
  --ignore-table=opencoze.workflow_execution \
  --ignore-table=opencoze.run_record \
  --ignore-table=opencoze.workflow_snapshot \
  --ignore-table=opencoze.conversation \
  --ignore-table=opencoze.message \
  --ignore-table=opencoze.knowledge_document_slice \
  ... opencoze | gzip > /tmp/opencoze-authoring-<ts>.sql.gz
```
- `--complete-insert` **必须**：目标多出的 5 新列才能靠 DEFAULT 接住，否则 `INSERT VALUES` 列数不匹配。
- `--no-create-info`：只要数据，建表交给已就位的新 schema。

### 3.5 账号处理
```sql
SET @uid = (SELECT id FROM user WHERE email='402087139@qq.com');
UPDATE space SET deleted_at=UNIX_TIMESTAMP()*1000 WHERE owner_id=@uid AND deleted_at=0;
DELETE FROM space_user WHERE user_id=@uid;
DELETE FROM user WHERE id=@uid;   -- 或软删，按目标表语义
```
（merge 不 DROP → 226 的 6 个 dev-only 账号原样保留，无需导出/回灌。删的是 226 与 prod 各一份 402087139。）

### 3.6 执行前置
目标已确认为 **226 `coze-mysql/openynet`**（非 220）；224 源 + 226 目标本机均可达 → 可直接跑（本机做中转：224 dump → 本机 → 226 load）。脚本参数化 `SRC_*/DST_*`，默认 `DST_SSH=dev@10.10.10.226`。**注意**：226 `openynet` 是 `coze-super` 在用的活库，替换前会停/重启服务更稳。

---

## 4. Part B — 部署期自适应迁移器（`gen-forward-migrate` + `forward-migrate-<ver>.sql`）

### 4.1 解法：声明式状态收敛 + 只增不删
不管目标停在哪个版本，都把它**收敛**到"当前代码期望 schema"，且**绝不 DROP**。逐条对应用户痛点：
- 缺库 → `CREATE DATABASE IF NOT EXISTS`
- 缺表 → `CREATE TABLE IF NOT EXISTS <完整 DDL>`
- 缺列 → 守卫式 `ADD COLUMN`（见 4.3）
- 列漂移（改名/改类型，纯增量表达不了）→ **known-fixups** 幂等清单（见 4.4）

### 4.2 产物形态（关键：现场零开发）
- **生成器在我方**每次发版自动从权威 schema 产出一个**自包含、幂等的 `.sql`**。
- **现场只需 mysql 客户端跑这一个文件**（行内离线、无 Atlas 也能用）→ 复用 deploy-v2 `run_mysql`。
- 生成器实现（避免 SQL 解析）：
  1. 起一个**临时参考 MySQL** 容器；
  2. 按序应用权威 schema（`schema.sql` + `docs/ynet-database-sql/04,05,99-105` + `docker/migrations/*`）→ 得到"期望态"；
  3. 用 `SHOW CREATE TABLE` 取每张期望表的完整 DDL → 转 `CREATE TABLE IF NOT EXISTS`；
  4. 从 `information_schema.columns` 取每张表期望列 → 生成守卫 `__add_col` 调用；
  5. 排除动态/运行表模式（`table\_%`、`ragflow\_%`、运行表）；
  6. 拼接 known-fixups → 输出 `forward-migrate-<ver>.sql`。
- 我方可控环境（224/dev）另可用 **Atlas 声明式 `schema apply --dry-run`** 做精确预览/交叉校验。

### 4.3 守卫式 ADD COLUMN（MySQL 8.4 / OB 都不支持 `ADD COLUMN IF NOT EXISTS`）
```sql
DROP PROCEDURE IF EXISTS __add_col;
DELIMITER //
CREATE PROCEDURE __add_col(IN t VARCHAR(128), IN c VARCHAR(128), IN def TEXT)
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                 WHERE table_schema=DATABASE() AND table_name=t AND column_name=c) THEN
    SET @ddl=CONCAT('ALTER TABLE `',t,'` ADD COLUMN `',c,'` ',def);
    PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;
  END IF;
END //
DELIMITER ;
-- 每个期望列一行（由生成器产出），例：
CALL __add_col('single_agent_draft','agent_type','VARCHAR(64) DEFAULT NULL');
CALL __add_col('single_agent_draft','super_agent_tool_config','json DEFAULT NULL');
CALL __add_col('single_agent_draft','source_product_id','bigint NOT NULL DEFAULT 0');
CALL __add_col('single_agent_draft','source_product_version','varchar(64) NOT NULL DEFAULT ""');
CALL __add_col('single_agent_version','agent_type','VARCHAR(64) DEFAULT NULL');
DROP PROCEDURE __add_col;
```
幂等、只增不删、跑几遍都安全。

### 4.4 known-fixups（站点特有、纯增量表达不了的）
```sql
-- 例：行内 system_setting.key → setting_key（空表零风险，information_schema 守卫）
-- 仅当旧列在、新列不在时才改名；否则跳过。
```
每条都守卫，可重复执行。

### 4.5 安全 + 接入 deploy-v2
- **铁律**：备份 → 干跑(只读 diff/`--dry-run` 打印 plan) → 应用 → 校验(表数/列数) → 可回滚。
- 作为 `deploy.sh` 的 **"schema 收敛"阶段**：排在建库后、seed/启动前；用 `run_mysql` 跑 `forward-migrate-<ver>.sql`。
- 参数化库名/中间件；OB 无 FK（schema 本就无 FK）、权限受限场景只用 CREATE/ALTER/INSERT。

---

## 5. 落成物

| 文件 | 作用 |
|---|---|
| `docs/superpowers/specs/2026-06-26-prod-data-seed-and-migration-design.md` | 本设计（已提交） |
| `ynet-docker/seed-from-prod.sh` | A：克隆替换导入（参数化、只读源、备份、账号、MinIO/ES、校验） |
| `ynet-docker/forward-migrate/gen-forward-migrate.sh` | B：生成器（参考库 → 幂等迁移 SQL） |
| `ynet-docker/forward-migrate/forward-migrate.sample.sql` | B：样例产物（含守卫 ADD COLUMN + known-fixups 骨架） |

---

## 6. 执行顺序与验证

1. 写 spec（本文件）→ 提交。
2. 写 A/B 脚本 → 提交。
3. **B 只读自验**：以当前代码 schema 为期望态，对 224 跑差集检测，验证恰好检出 12 表 + 5 列。
4. **A 执行**（阻塞于 220 连通 + 用户放行 + 主账号确认）：备份 → 导入 → 端到端校验一个空间可完整打开。

## 7. 风险与回滚
- 生产只读，零写入；任何异常不影响 224。
- 目标库替换前**全量备份**，可一键回滚到 `backend/openynet-<ts>.sql.gz`。
- 目标改为 226（可达），原 220 阻塞解除；执行仅需用户放行 + 替换前停 coze-super。
- 动态 `table_<id>` 与 ragflow 表：A 默认带、B 默认排除（不参与 schema 收敛、也绝不删）。

## 8. 待确认 / Open items
- 主账号：执行期自动识别 + 二次确认（不阻塞当前）。
- ES 重建：确认后端有可调的"重建索引"入口；否则补一段按 space 重新 index 的脚本。
- 动态表 + ragflow 是否纳入 A 的导入（默认纳入，体积可控；可关）。
