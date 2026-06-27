# 超级体记忆(memory)重新设计 —— per-user + DB

Date: 2026-06-20

## 背景与决策

旧设计(`agentflow/node_tool_memory.go`):memory = 沙箱文件 `/workspace/.agent/memory.json` 里的**扁平字符串数组**,`memory_save` 纯追加,`memory_recall` 倒全量(query 被忽略)。问题:只增不改/无去重、倒全量撑爆上下文、无结构无分类、无 provenance、**作用域绑死单 agent 沙箱**、不可检索。不符合"会进化超级体"的需求。

用户决策(2026-06-20):
- **作用域 = 按用户(grows with you)**:记忆跟着 `user_id` 走,同一用户在任意超级体/会话共享同一份画像+事实;`agent_id`/`space_id` 仅作 provenance 与可选过滤,不做隔离。
- **存储 = 数据库(OB/MySQL)**:结构化行 + 可索引 + 跨会话/跨 agent 共享 + 并发安全 + 可审计;为语义召回预留向量(OB)。

## 数据模型

新表 `super_agent_user_memory`(迁移 `docs/ynet-database-sql/101-super-agent-user-memory.sql`):

| 列 | 类型 | 说明 |
|---|---|---|
| id | bigint PK | 雪花/自增 |
| user_id | bigint NOT NULL | 归属用户(per-user 作用域核心) |
| space_id | bigint default 0 | 可选过滤/provenance |
| agent_id | bigint default 0 | 哪个 agent 观察到(provenance) |
| kind | tinyint NOT NULL | 1 profile / 2 preference / 3 fact / 4 project / 5 feedback |
| mem_key | varchar(128) default '' | 稳定键,用于 upsert/supersede(如 "timezone"、"tone");非空时同 (user_id,mem_key) 唯一 |
| content | text NOT NULL | 记忆内容 |
| tags | varchar(512) default '' | 逗号/JSON 标签,辅助检索 |
| source_conversation_id | bigint default 0 | 来源会话 |
| source_run_id | bigint default 0 | 来源 run |
| status | tinyint default 1 | 1 active / 2 archived(superseded) |
| created_at / updated_at | bigint(ms) | |

索引:`(user_id, status)`、`(user_id, kind, status)`、唯一 `(user_id, mem_key)`(mem_key 非空)。Phase B 再加向量列 + ANN 索引(OB)做语义召回。

## 工具 API(替换扁平版)

- **memory_save(content, kind?, key?, tags?)**:kind 默认 fact;若 key 非空且 (user_id,key) 已存在 → **更新**(supersede,content+updated_at);否则插入。近重复内容去重(同 user)。profile/preference 类鼓励带 key。
- **memory_recall(query?, kind?, limit?)**:**profile 类常驻全返**;其余按 query 过滤(Phase A:content/tags LIKE 或全文;Phase B:向量语义)+ kind 过滤 + limit(默认 ~20)。**不再倒全量**。
- 每轮开场可由系统提示/复盘 fork 调 recall;复盘 fork 写入时"先 recall 去重、能覆盖就覆盖"。

## 落地分层(照搬 skill 的 DB 模式)

- `domain/agent/singleagent/.../usermemory/entity`(或复用 singleagent entity):`UserMemory` 实体 + `MemoryKind`。
- repository 接口:`Save/Upsert(ctx, *UserMemory)`、`ListByUser(ctx, userID, filter)`、`Search(ctx, userID, query, kind, limit)`、`Supersede`。
- `internal/dal`:GORM DAO(对齐 `domain/skill/internal/dal`)。
- 一个轻服务封装"save with upsert/dedup"、"recall with profile-always + relevance"。
- **工具改线**:`superAgentExtensionFactory` 由 `func(key string)` 改为 `func(deps superAgentToolDeps)`,`deps={SandboxKey,UserID,SpaceID,AgentID}`;5 个工厂(web_fetch/web_search/memory_save/memory_recall/skill_manage)统一适配(多数只用 SandboxKey)。memory 工具用 UserID 走 DB 服务。

## 增量(TDD)

1. 迁移 SQL(101)+ GORM model/entity。
2. repository 接口 + DAO + 单测(upsert by key、search by user/kind/query、profile-always)。
3. memory 服务(save-with-upsert-dedup、recall-profile+relevance)+ 单测。
4. 工厂 deps 重构 + memory 工具改走服务 + 单测(per-user 隔离、upsert、recall 不倒全量)。
5. 接入闭环复盘 fork(复盘 agent 的 memory_save 走新服务)。
6. 真实双会话端到端 + 部署 226 留证。

## 兼容/迁移

旧沙箱 `memory.json` 不删(数据安全);可选一次性导入脚本把旧条目按 user 归集到新表(默认 kind=fact)。新代码只读写 DB;读取时若 DB 空且旧文件有内容,可懒迁移(Phase A 可不做,先并存)。
