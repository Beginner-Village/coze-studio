# 空间级操作审计日志（Operation Log）设计

- 日期：2026-06-13
- 状态：待评审
- 模块：backend（采集中间件 / domain / application / API）+ frontend（空间操作日志页）

## 1. 背景与目标

为 ynet-studio（coze-studio）提供**空间（space）级别的操作审计日志**：可追溯"谁、在何时、在哪个空间、对什么资源、做了什么操作、结果如何"。

首要用途：**合规审计 + 历史操作可追溯**。覆盖空间内的写操作（工作流、智能体、知识库、插件等所有调用写接口的行为）。

### 成功标准
- 空间内写操作被可靠记录（异步落库，不阻塞主请求；通道满时丢弃并计数，绝不阻塞业务）。
- 审计日志可读：能直接看到"模块 / 资源 / 动作"的中文语义描述，而非裸 URL。
- 保留期可配置（env，默认 90 天，范围 30–90 天），过期自动清理。
- 提供查询 API（按空间 + 多维筛选 + 分页），仅空间 Owner/Admin 可查。
- 前端提供空间「操作日志」页面。

### 非目标（YAGNI）
- 不记录读操作（GET / 查询类）。
- 不存完整请求/响应 body，不做字段级 before/after diff（仅存摘要元数据）。
- 不做跨空间的全局审计后台、不做导出/告警（后续可扩展）。
- 不做防篡改/加密链（合规要求的不可篡改通过 DB 权限与 add-only 表设计满足，不在本期实现密码学防篡改）。

## 2. 架构总览

顺延项目现有 DDD 分层：

```
API middleware: OperationLogMW (采集)
        │  非阻塞投递
        ▼
   buffered channel (内存缓冲)
        │  批量消费
        ▼
domain/operationlog  (entity / service / repository / internal/dal+gorm-gen)
        │
        ├── 后台 worker: 批量 insert MySQL
        └── 后台清理任务: 删除超过保留期记录
        ▲
        │ 查询
application/operationlog (查询编排 + 权限校验 + operator_name 回查)
        ▲
        │
API handler: POST /api/operation_log/list
        ▲
        │
前端: 空间「操作日志」页
```

## 3. 采集层：`OperationLogMW`

### 3.1 注册位置
在 `backend/main.go` 中间件链中，注册于 `SessionAuthMW` 之后、`I18nMW` 附近（确保 uid 已注入 ctx，可经 `application/base/ctxutil.GetUIDFromCtx` 取得）。

```
s.Use(middleware.SessionAuthMW())
s.Use(middleware.I18nMW())
s.Use(middleware.OperationLogMW())   // 新增
```

### 3.2 路由白名单 + 语义映射（route registry）
新增 `backend/api/middleware/operationlog/route_registry.go`，维护一份显式配置表。每条：

| 字段 | 含义 |
|------|------|
| `Method` | HTTP 方法（POST/PUT/DELETE/PATCH）|
| `PathPattern` | 路由模板（支持 `:id` 等参数段，与 Hertz 路由一致）|
| `Module` | 模块：workflow / agent / knowledge / plugin / space / database / prompt 等 |
| `ResourceType` | 资源类型枚举（复用/对齐 `domain/permission/consts.go` 的 ResourceType）|
| `Action` | create / update / delete / publish / copy / ... |
| `DescTemplate` | 中文描述模板，如 `"更新了工作流「{resource_name}」"` |
| `ResourceIDFrom` | resource_id 抽取来源：path 参数名 / body 字段名 / query 字段名 |
| `ResourceNameFrom` | resource_name 抽取来源（可选）|

只有命中白名单的请求才记录 → 天然去噪（草稿自动保存、心跳、纯读接口不在表内）。新增需要审计的写接口时，往该表加一条即可。

> 第一期覆盖范围：workflow、single agent / project(app)、knowledge、plugin、database、prompt、space member（邀请/移除/改角色）等核心写接口。具体清单在实现阶段对照 `api/router/*` 逐条列出。

### 3.3 采集流程
1. `ctx.Next(c)` 执行业务逻辑。
2. 查 route registry：未命中 → 直接返回，不记录。
3. 提取字段：
   - `operator_id`：`ctxutil.GetUIDFromCtx(c)`（取不到则跳过，未登录写操作不在审计范围）。
   - `space_id`：按 query → form → body JSON 顺序提取 `space_id`（项目内统一为 `space_id`，`json:"space_id,string"`）。取不到则记 0（空间无关写操作，仍记录但归到无空间）。
   - `resource_id` / `resource_name`：按 registry 的 `ResourceIDFrom` / `ResourceNameFrom` 抽取（path/body/query）。
   - `method`、`path`（PathOriginal）、`client_ip`、`status`（HTTP code）、`log_id`、`duration_ms`。
   - `status`：依据 HTTP code 与响应 body 的业务 code 判定 success / fail；`error_code` 记录业务错误码（失败时）。
   - `request_summary`：从 body 中按白名单字段裁剪出的简短摘要（**不存完整 body**），便于阅读，长度截断（如 ≤512 字符）。
4. 组装 `OperationLogEvent`，**非阻塞** `select { case ch <- event: default: metrics.dropCount++ }` 投递到缓冲 channel。

> body 读取：参考 `api/middleware/log.go` 的 `AccessLogMW`，`ctx.Next` 之后 `ctx.Request.Body()` 已缓冲可读，不影响业务。

## 4. domain/operationlog

目录结构（对齐 `domain/shortcutcmd`、`domain/statistics`）：

```
domain/operationlog/
  entity/operation_log.go          # OperationLog 实体、OperationLogEvent、查询过滤条件
  service/operation_log.go         # 接口
  service/operation_log_impl.go    # 实现：异步 worker、清理、查询
  repository/repository.go         # 仓储接口
  internal/dal/dao.go              # gorm dao
  internal/dal/model/operation_log.gen.go
  internal/dal/query/*.gen.go
```

### 4.1 数据表 `operation_log`

| 列 | 类型 | 说明 |
|----|------|------|
| `id` | bigint PK | idgen 生成 |
| `space_id` | bigint, index | 空间 ID（0=无空间）|
| `operator_id` | bigint, index | 操作者 user id |
| `module` | varchar(64) | 模块 |
| `resource_type` | int | 资源类型枚举 |
| `resource_id` | bigint | 资源 ID（可空/0）|
| `resource_name` | varchar(255) | 资源名（采集时能取到则存）|
| `action` | varchar(32) | 动作 |
| `description` | varchar(512) | 渲染后的中文语义描述 |
| `method` | varchar(8) | HTTP 方法 |
| `path` | varchar(255) | 请求路径 |
| `request_summary` | varchar(512) | 请求摘要（裁剪，非完整 body）|
| `status` | tinyint | 1=success 2=fail |
| `error_code` | varchar(64) | 业务错误码（失败时）|
| `client_ip` | varchar(64) | 客户端 IP |
| `duration_ms` | int | 耗时 |
| `log_id` | varchar(64) | 链路 log id |
| `created_at` | bigint | 毫秒时间戳 |

索引：`idx_space_created (space_id, created_at)`、`idx_space_operator (space_id, operator_id)`、`idx_space_restype (space_id, resource_type)`。

> **不存 `operator_name`**：写路径只存 `operator_id`，查询时批量回查 user 表解析显示名，避免每次写入多一次查询，也避免用户改名后历史名不一致。

DDL 追加到 `docker/volumes/mysql/schema.sql`，由现有启动自愈（add-only）机制建表。

### 4.2 异步落库 worker
- service 初始化时启动一个后台 goroutine。
- 从缓冲 channel 批量取（批大小如 100 或定时 1s flush，取先到者）→ idgen 批量生成 id → gorm 批量 insert。
- 单批失败：记 error 日志 + 计数，不重试到阻塞（审计日志尽力而为，不影响主流程）。

### 4.3 保留期清理
- env `OPERATION_LOG_RETENTION_DAYS`（默认 90，未配或越界则归一到 [30,90]）。
- 后台定时任务（每天一次）：`DELETE ... WHERE created_at < now - retentionDays`，分批删除避免大事务。

### 4.4 查询接口（domain service）
- `ListOperationLogs(ctx, filter)`：filter = { space_id, operator_id?, resource_type?, action?, start_time?, end_time?, keyword?, page, page_size } → 返回 records + total。
- keyword 模糊匹配 `description` / `resource_name`。

## 5. application/operationlog

- `ListOperationLogs`：
  1. 取 operator uid（`ctxutil`）与请求 space_id。
  2. **权限校验**：调用 `domain/user/service.CheckMemberPermission(ctx, spaceID, uid)`，要求 `canManage == true`（即 Owner/Admin）；否则返回权限错误（复用 errno 体系）。
  3. 调 domain service 查询。
  4. 批量回查 `operator_id → operator_name`（user 服务），组装返回。

## 6. API 层

- 新增 thrift/IDL + handler：`POST /api/operation_log/list`
  - 入参：`space_id`（required）、`operator_id`、`resource_type`、`action`、`start_time`、`end_time`、`keyword`、`page`、`page_size`。
  - 出参：`{ logs: [...], total, page, page_size }`，每条含 operator_id、operator_name、module、resource_type、resource_name、action、description、status、created_at 等。
- 路由注册：在 `api/router` 新增 operation_log 路由组，注册到 `register.go`。
- 走 WebAPI 鉴权（已有 session 中间件链覆盖）。

## 7. 前端：空间「操作日志」页

- 入口：workspace 子菜单（`packages/foundation/space-ui-adapter` 的 workspace-sub-menu）加「操作日志」项；仅 Owner/Admin 可见（依据当前用户在该空间的角色）。
- 页面（新增于 `apps/coze-studio/src/pages/` 或对应 workspace 包）：
  - 筛选栏：操作人、资源类型、动作、时间范围、关键词。
  - 分页表格：时间 / 操作人 / 模块 / 资源 / 动作描述 / 状态。
  - 调 `POST /api/operation_log/list`。
- API client 走项目现有 api-schema（`packages/arch/api-schema`）生成方式，新增 operation_log 的 idl。

## 8. 配置

| env | 默认 | 说明 |
|-----|------|------|
| `OPERATION_LOG_RETENTION_DAYS` | 90 | 保留天数，归一到 [30,90] |
| `OPERATION_LOG_BUFFER_SIZE` | 4096 | 采集缓冲 channel 容量（可选）|
| `OPERATION_LOG_ENABLED` | true | 总开关（可选，便于排障/压测时关闭）|

## 9. 测试策略

- 单元测试：
  - route registry 匹配（命中/未命中、path 参数抽取）。
  - 字段提取（space_id 从 query/form/body 三处）。
  - 权限校验（Owner/Admin 通过，Member 拒绝）。
  - 清理任务的时间边界。
- 集成测试：middleware → channel → worker → DB 落库全链路（用 testcontainers/现有 test 基建对齐 `domain/statistics/service/statistics_test.go` 风格）。
- 手动端到端：在前端页面触发若干写操作，验证日志出现、筛选/分页正确、Member 账号访问被拒。

## 10. 风险与权衡

- **采集为尽力而为**：channel 满即丢弃 + 计数。合规上若要求"零丢失"，需后续改为同步写或加持久化队列（本期不做，已与需求方确认异步可接受）。
- **路由白名单需维护**：新增写接口要补登记，否则不被审计。通过在 code review checklist 与文档中提示缓解。
- **request_summary 摘要可能含敏感字段**：白名单字段裁剪 + 长度截断，避免整包敏感数据落库；敏感字段（密码/密钥）显式排除。
- **schema 自愈为 add-only**：表结构后续变更需走自愈脚本兼容路径。
```
