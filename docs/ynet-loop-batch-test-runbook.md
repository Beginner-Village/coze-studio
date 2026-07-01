# Loop 批量测试（智能体 / 工作流）落地与验证手册

本次改动让 loop 的批量评测（experiment）能**真实运行**「智能体(CozeBot)」「工作流(CozeWorkflow)」——
此前只有 Prompt 有执行器，bot/workflow 在线实验只是 record-only 占位（不真跑）。

## 一、改了什么

### 后端（coze-loop）
- 新增执行器接口 `ICozeTargetRPCAdapter`：`backend/modules/evaluation/domain/component/rpc/cozetarget.go`
- 新增 studio OpenAPI HTTP 适配器：`backend/modules/evaluation/infra/rpc/cozestudio/cozestudio.go`
  - 工作流 → `POST /v1/workflow/run`（同步，输出在 `data`）
  - 智能体 → `POST /v3/chat`（`stream:false`，答案取 role=assistant & type=answer）
  - 鉴权 `Authorization: Bearer <PAT>`
- 新增两个评测执行器：`target_source_cozeworkflow_impl.go` / `target_source_cozebot_impl.go`
- 注册进 `NewSourceTargetOperators`（`domain/service/wire.go`）：CozeWorkflow(4) / CozeBot(1)

### 前端（coze-studio）
- `experiments/create.tsx`：评测对象类型收敛到已支持的 workflow/bot/prompt；创建**真实可执行 target**
  （base type 1/4/2 + source_target_id）并用 **Offline 实验(expt_type=1)**，去掉 record-only 占位；评估器改为可选。
- `experiments/detail.tsx`：新增**逐行结果**面板（每条 item 的目标输出/评估器打分，防御式渲染 + 分页 + 可折叠汇总）。
- `experiments/index.tsx`：实验列表分页。
- `loop-eval-api.ts`：新增 `listExperimentResults`（`/experiments/results/batch_get`）。

## 二、部署必配：loop → studio 调用通道（关键）

在 **loop 服务**（226）注入环境变量，否则选 bot/workflow 跑实验会返回明确错误
`studio openapi not configured`：

| 环境变量 | 说明 | 示例 |
|---|---|---|
| `YNET_STUDIO_OPENAPI_BASE_URL` | studio 服务内网地址（无路径前缀，默认端口 8888） | `http://coze-server:8888` |
| `YNET_STUDIO_OPENAPI_TOKEN` | studio PAT（该 PAT 所属用户须是目标 bot/workflow 所在 space 的成员） | `pat_xxx` |
| `YNET_STUDIO_OPENAPI_USER_ID`（可选） | 智能体 `/v3/chat` 的数据隔离用户标识，缺省 `coze_loop_eval` | `eval_bot` |

PAT 获取：studio 登录后在「个人设置 → API 授权 / 令牌」创建 personal access token。
注意 CDRCB 历史坑：头部必须精确为 `Authorization: Bearer <key>`（大小写、Bearer 前缀）。

## 三、使用前置数据
- 目标 space 内至少 1 个**已发布**工作流（未发布会被 studio 拒绝 `workflow not published`）或 1 个智能体。
- 至少 1 个评估集（列名与工作流入参名对应时按名透传；智能体取 user_query/首个文本列作为提问）。

## 四、验证状态（截至本次改动）

| 层面 | 命令 | 结果 |
|---|---|---|
| 后端编译 | `go build ./modules/evaluation/...` | ✅ 通过 |
| 后端单测（adapter httptest + 两个 operator） | `COZE_LOOP_SESSION_HMAC_KEY=x COZE_LOOP_STUDIO_HMAC_KEY=y go test ./modules/evaluation/infra/rpc/cozestudio/... ./modules/evaluation/domain/service/ -run 'Coze...'` | ✅ 17 项全过 |
| 前端类型检查 | `tsc --noEmit -p tsconfig.json` | ✅ 全项目 exit 0 |
| 前端接口单测 | `vitest run src/pages/observability/__tests__/loop-eval-api.test.ts` | ✅ 4 项全过 |
| **真实环境 E2E** | `BASE_URL=http://10.10.10.226:8896 pnpm playwright test 13-experiment-batch-test` | ⏳ **待在 226 执行**（需上面二/三节配置） |

## 五、跑真实 E2E
1. 按第二节在 loop 配好环境变量并重启 loop。
2. 确认第三节前置数据。
3. 执行：
   ```bash
   cd coze-studio/e2e
   BASE_URL=http://10.10.10.226:8896 pnpm playwright test 13-experiment-batch-test.spec.ts
   ```
4. 期望：创建实验成功；实验详情「结果」tab 出现逐行目标输出。

## 六、2026-07 增量：独立「批量测试」页 + 概览增强 + Excel + bot 收尾

### 前端（coze-studio）
- 新增独立 **「批量测试」页**（`src/pages/batch-test.tsx`，路由 `SpaceSubModuleEnum.BATCH_TEST=/batch-test`，
  工作区侧栏「批量测试」入口）：选 bot/workflow → 下载模板 → 上传 → 一键批跑 → 逐行结果表。
  对用户隐藏 loop 评测集/版本/实验的复杂度（内部仍复用已验证后端编排）。
- **CSV + Excel 双上传**：零依赖 `src/pages/observability/xlsx-lite.ts`（浏览器内置
  `DecompressionStream('deflate-raw')` 解 xlsx 的 DEFLATE + 手写 STORED zip 生成模板），
  不给 monorepo 引入任何依赖，适配银行离线环境。提供「下载 CSV 模板」「下载 Excel 模板」。
- **实验概览页补全**（`experiments/detail.tsx`）：评测对象（类型+source_id）、评测集名、
  成功/总数、成功率、失败数、耗时（end-start）、创建时间（`start_time` 兜底）；去掉冗余「–」。
- **失败行可见**：结果表对 `eval_target_run_error` 即使 message 为空也展示「执行失败（错误码 N）」，不再静默「-」。

### 后端（coze-loop）
- CozeBot 评测对象输入字段 key 统一为 `input`（`target_source_cozebot_impl.go`，与 CozeWorkflow 一致），
  使「批量测试」的 `target_field_mapping`（input←input）对 bot 同样生效；`extractUserQuery` 优先取 `input`，
  回落 prompt user_query / 首个非空文本字段（向后兼容）。
- `TargetConf.Valid()` 豁免表已含 CozeWorkflow/CozeBot（与 Prompt 同，自行处理输入、不强制连接器 FieldConfs）；
  同步修正受影响的两处单测（`callTarget_EdgeCases` / `TargetConf_Valid_MoreBranches` 改用非豁免类型触发失败，
  并补 CozeBot/CozeWorkflow 豁免正向用例）。

### 验证（本次）
| 层面 | 结果 |
|---|---|
| loop 单测（service/cozestudio/entity 全包） | ✅ 全绿 |
| 前端 tsc + vitest（loop-eval-api + xlsx-lite 往返/多列/转义/空值 8 项） | ✅ 全绿 |
| 本地 dev（proxy→226）workflow 批测 + 真实 DEFLATE+sharedStrings xlsx 上传 | ✅ 3 行含特殊字符全绿 |
| studio `/v3/chat` live 通道（226 loop→coze-super） | ✅ 鉴权/请求体正确，仅报「agent not published」业务错误 |
| **226 部署后 workflow 批测回归** | ✅ 查询余额(demo) 2 行真实输出 57/58ms |

### 部署方式（226，轻量法）
- **前端**：`rsbuild build` → `dist/` 打 tar → scp 到 226 → `docker cp` 进 `coze-super:/app/resources/static/`
  （先 `cp -a static static.bak.<ts>` 备份，再 `rm -rf static/* && tar xzf` 覆盖）。Go 服务直接托管静态文件，**无需重启**。
- **后端**：`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o main ./cmd` → scp → `docker cp` 到
  `ynet-loop-app:/ynet-loop/bin/main`（先备份 `main.bak.<ts>`）→ `docker restart`。
  ⚠️ docker-cp 法在容器 **recreate 后会丢失**，最终应把源码改动纳入正式镜像构建。

## 七、2026-07 补完：bot 绿 E2E + CK 库名修复（两问题均已解决）

### 问题1：bot 批测跑绿（已解决）
- **真因**：批测 bot 列表调用 `intelligenceApi.GetDraftIntelligenceList` **漏传 `status` 参数**（缺省返回空），
  并非「没有 bot」。已在 `batch-test.tsx` 补 `status:[Using,Banned,MoveFailed]`（与 develop 页一致）。
- 用现有智能体 **test(7654425189890916352)** 发布到 **API 渠道**（`/v3/chat` 需 connectorID 1024），
  bot 批测在 226 跑绿：2 行真实答案（账户变更流程 6557ms / 开户材料 6895ms）。

### 问题2：CK 库名（已解决）
- **真因**：`getClickHouseDatabaseName()`（`ck/expt_turn_result_filter.go`）读旧前缀
  `COZE_LOOP_CLICKHOUSE_DATABASE`，而 226 部署走 `YNET_LOOP_CLICKHOUSE_DATABASE`/`infrastructure.yaml`
  的 `ynet-loop-clickhouse`。已改为优先读 `YNET_LOOP_CLICKHOUSE_DATABASE`、回落 `COZE_`、再回落默认。
- 部署新二进制后 `cozeloop-clickhouse does not exist` 错误**消失**（近 3 分钟 0 次）。

### 问题3：filter consumer 缺 user 上下文（已解决）
- CK 库名修复后，`ExptTurnResultFilterConsumer` 不再卡在库名，转而暴露
  `code=602000202 invalid user_id in context`（`modules/foundation/application/auth.go`）——被 CK 错误掩盖的第三个既有 bug。
- **根因**：该 consumer 是唯一**没有** `session.WithCtxUser(ctx,...)` 的 MQ consumer；兄弟
  （`expt_export`/`expt_record_eval`/`expt_scheduler_event`）都在 HandleMessage 里注入了用户上下文。异步无 user → 建结果时 auth 失败。
- **修法（仿既定模式，单点最小改动）**：
  1. `entity.ExptTurnResultFilterEvent` 加可空 `Session *Session`（`event.go`）。
  2. **发布方法** `exptEventPublisher.PublishExptTurnResultFilterEvent`（`producer/expt_event_pub.go`）统一
     `event.Session = entity.NewSession(ctx)`——**一处覆盖全部 8 个生产者**（比逐个改生产者更稳）。
  3. consumer（`consumer/expt_turn_result_filter.go`）nil-safe 注入：
     `if event.Session != nil && event.Session.UserID != "" { ctx = session.WithCtxUser(ctx, &session.User{ID: ...}) }`。
- **验证（226 真实负载）**：新实验 7657538877577494529 的 filter「check」事件已携带
  `Session:{UserID:...}`，consumer 日志 `CompareExptTurnResultFilters finish, all equal`（成功注入用户、查
  `` `ynet-loop-clickhouse` `` 库、比对通过），`invalid user_id`/`cozeloop-clickhouse`/consumer ERROR 计数均为 0。
  该功能此前从未跑通（先被 CK 名挡、再被 user_id 挡），两处皆修后端到端成功。
