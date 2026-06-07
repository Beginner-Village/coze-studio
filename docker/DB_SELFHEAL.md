# 数据库自愈（DB Self-Heal）

镜像启动时自动**对齐数据库 schema** 并**灌入模型内部数据**，然后再启动应用。
目标：升级换镜像时无需任何外部脚本/人工 SQL，零外部上传，全部自包含在镜像里，
适配内网/离线现场。

## 病根

原先 schema 只在 MySQL **首次 initdb** 时执行 `docker/volumes/mysql/schema.sql`。
升级只换后端镜像、MySQL 数据卷不变 → schema.sql 不会重跑 → 新版本需要的
**缺表 / 缺列 / 模型内部数据**全部缺失，应用报错。

## 原理

后端镜像（`backend/Dockerfile` 最终 alpine 层）的 `ENTRYPOINT` 改为
`/app/db-selfheal.sh`，它在拉起 `/app/openynet` 之前依次做：

1. **等待 MySQL 就绪**：循环 `mysql ... -e "SELECT 1"`，最多约 60s（30 次 × 2s）。
   超时则打印 warn 并直接启动应用（不阻塞，应用自身还会再做连接重试）。
2. **schema 自愈**：`python3 /app/db/schema-sync.py --ref-sql /app/db/schema.sql ... --apply`
   - 把 `schema.sql`（裸 mysqldump）临时 load 进目标库的一个 scratch 参考库，
     和线上库做 diff；
   - **只加不删**：参考库有而线上没有的【表】→ `CREATE TABLE`；缺的【列】→
     `ALTER TABLE ADD COLUMN`；**绝不 DROP、绝不改已存在的表/列**，不动现场数据；
   - 跑完删除 scratch 库；
   - 因为是 diff 后只补差异，所以**不怕 schema.sql 里的 `CREATE TABLE` 撞已存在**。
3. **模型内部数据**：`mysql ... < /app/db/model-seed.sql`，里面全是
   `INSERT IGNORE`，**幂等**，重复启动不会重复插入或报错。
   覆盖表：`model_template` / `model_entity` / `model_meta` / `template`。
4. **交还控制权**：`exec /app/openynet`，保留原 `CMD ["/app/openynet"]` 语义
   （PID 1、信号转发正常）。

每一步（2、3）单步失败用 `|| echo warn` 兜底，不致命，保证应用总能被拉起，
失败原因打印在启动日志里便于排查。

## 开关（逃生口）

| 环境变量 | 默认 | 说明 |
| --- | --- | --- |
| `DB_SELFHEAL` | `true` | `false` 时**完全跳过**自愈，直接 `exec /app/openynet` |

需要的 MySQL 连接环境变量（与 `docker/.env.example` 一致）：
`MYSQL_HOST` `MYSQL_PORT` `MYSQL_USER` `MYSQL_PASSWORD` `MYSQL_DATABASE`。

## 模型密钥占位符（重要）

`docker/volumes/mysql/model-seed.sql` 里的模型 API Key 已**脱敏**为占位符
`REPLACE_ME_MODEL_API_KEY`（不会提交真实密钥到仓库）。

现场要让模型真正可用，需要把占位符换成真实 key，二选一：

- **构建前替换**（推荐，烤进镜像）：在 CI/构建机上对 `model-seed.sql`
  做一次 `sed -i 's/REPLACE_ME_MODEL_API_KEY/<真实key>/g'` 再 build；
- **现场库内替换**：进 MySQL 后 `UPDATE` 对应表把占位符替换成真实 key，
  或在管理后台界面里重新填模型 key。

> 注意：占位符**不要**直接 commit 成真实 key。

## 如何更新参考数据

镜像内的自愈参考物来自仓库这两个文件，更新它们后重新 build 镜像即可：

- `docker/volumes/mysql/schema.sql` —— schema 参考（mysqldump 产物）。
  升级带来新表/新列时，用新版本库重新导出结构覆盖它：
  `mysqldump --no-data -h<host> -P<port> -u<user> -p <db> > docker/volumes/mysql/schema.sql`
  （`--no-data` 仅结构即可；schema-sync 只关心结构。）
- `docker/volumes/mysql/model-seed.sql` —— 模型内部数据（`INSERT IGNORE`）。
  新增/调整内置模型后重新导出，并记得把真实密钥再脱敏回
  `REPLACE_ME_MODEL_API_KEY` 再提交。

镜像内对应路径：`/app/db/schema.sql`、`/app/db/model-seed.sql`、
`/app/db/schema-sync.py`、`/app/db-selfheal.sh`。

## 与 Atlas 的关系

仓库另有 `docker/atlas/` + `docker/migrations/`（Atlas 声明式迁移）。
本自愈机制是**离线现场的轻量兜底**：不依赖 Atlas 二进制、不写迁移历史表，
只做"补缺表/缺列 + 幂等灌种子数据"，与 Atlas 流程不冲突。
若现场已具备 Atlas 工具链并希望以 Atlas 为准，可设 `DB_SELFHEAL=false` 关闭本机制。
（是否统一改用 Atlas 由现场流程决定，见交付时的开放问题。）
