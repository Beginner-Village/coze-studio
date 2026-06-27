# 工作流画布 AI 助手交付文档

日期: 2026-06-24  
环境: 10.10.10.226:8896  
范围: 工作流编辑器右侧 finmallclaw 助手、画布自动编辑工具、多模态图片输入适配、测试环境部署与后续缺口。

## 1. 目标

最终目标是让用户在工作流编辑器点击 AI 后，右侧出现 finmallclaw 对话区域；用户用自然语言描述需求，模型通过工具真实操作左侧画布，包括读取节点目录、添加节点、连线、配置节点、绑定变量、布局、试运行和按错误修复。

关键产品要求:

- 右侧聊天区域不遮挡画布，画布应给聊天列让位。
- 模型不能只回复“已完成”，必须真实调用 `workflow_canvas_*` 工具。
- 一个工作流只能有一个 Start、一个 End，Start/End 只能引用，不能新增。
- 节点添加后必须逐节点配置输入、输出、变量绑定、条件、聚合和 End 返回。
- 变量绑定不能猜，必须从画布上下文和可绑定变量列表里取。
- 测试失败后要定位失败节点并局部修复，不能默认清空画布。
- 多模态图片输入要支持内网场景，优先用 base64/data URL 给模型，不依赖公网可访问 image URL。

## 2. 当前部署快照

测试机:

```bash
ssh dev@10.10.10.226
```

当前容器状态检查结果:

```text
coze-super Up
/ 剩余约 12G
```

主要容器:

- `coze-super`: 后端二进制与前端静态资源所在容器。
- `coze-mysql`: 业务库 `openynet`。
- `coze-redis`: Redis。
- `ynet-loop-minio`: 对象存储。

当前关键 ID:

- 演示 space: `7652614054615187456`
- finmallclaw / 演示超级体: `7652617174313336832`
- 普通测试 bot: `7654425189890916352`
- 常用工作流测试页: `/work_flow?workflow_id=7654449287564099584&space_id=7652614054615187456`

当前模型配置检查:

```sql
select id, model_name, protocol,
       json_extract(capability, '$.input_modal') as input_modal,
       json_extract(conn_config, '$.model') as conn_model
from model_meta
where id in (1,2);
```

截至本交付文档生成时:

- `model_meta.id=1`: `qwen3.7-plus`, `input_modal=["text","image"]`
- finmallclaw draft: `model_id="1"`
- 普通测试 bot draft: 检查到 `model_id="2"`，疑似测试期间 UI 自动保存或手工切换遗留；若要统一用 `qwen3.7-plus`，按下方 SQL 校准。

```sql
update single_agent_draft
set model_info = json_set(coalesce(model_info, json_object()), '$.model_id', '1')
where agent_id in (7654425189890916352,7652617174313336832);

update model_meta
set capability = json_set(coalesce(capability, json_object()), '$.input_modal', json_array('text','image'))
where id = 1;
```

## 3. 已落地能力

### 3.1 右侧 finmallclaw 聊天面板

已实现:

- 工作流编辑器右侧内嵌 finmallclaw 轻量聊天面板。
- 直连 `POST /api/conversation/chat`。
- 使用 `scene=4`, `draft_mode=true`, `space_id`。
- SSE 解析 `type:"tool_response"` 的消息。
- `tool_response.content` 是 ack JSON 字符串，前端解析后分发到画布命令服务。
- 聊天面板打开时，工作流容器给右侧面板让出空间。
- 工具调用展示已从“全部堆在最上面”调整为按文本/工具返回顺序穿插展示。
- 工具显示名已有中文映射，减少英文工具名占用摘要宽度。

主要相关文件:

- `frontend/packages/workflow/playground/src/components/workflow-agent-panel/index.tsx`
- `frontend/packages/workflow/playground/src/components/workflow-agent-panel/message-payload.ts`
- `frontend/packages/workflow/playground/src/components/workflow-container/index.tsx`
- `frontend/packages/workflow/playground/src/services/workflow-agent-command-protocol.ts`

### 3.2 画布命令总线

已实现的前端命令能力:

- 新增节点
- 连接节点
- 配置节点
- 修改节点高级参数
- 删除节点
- 删除连线
- 清空非 Start/End 节点
- 自动布局
- 读取画布上下文
- 读取可绑定变量
- 触发试运行
- 工具 ack 去重
- tag 到 nodeId 的映射
- 操作后自动布局

主要相关文件:

- `frontend/packages/workflow/playground/src/services/workflow-agent-command-service.ts`
- `frontend/packages/workflow/playground/src/services/workflow-agent-command-readonly.ts`
- `frontend/packages/workflow/playground/src/services/workflow-agent-command-auto-layout.ts`
- `frontend/packages/workflow/playground/src/services/workflow-agent-clear-canvas.ts`
- `frontend/packages/workflow/playground/src/services/workflow-agent-test-run-guard.ts`
- `frontend/packages/workflow/playground/src/services/workflow-agent-canvas-summary.ts`
- `frontend/packages/workflow/playground/src/services/workflow-agent-semantic-config.ts`

### 3.3 后端画布工具

已注册到超级体工具体系的 `workflow_canvas_*` 能力包括:

- `workflow_canvas_get_operation_guide`
- `workflow_canvas_get_node_catalog`
- `workflow_canvas_get_node_spec`
- `workflow_canvas_get_node_capability_audit`
- `workflow_canvas_get_node_smoke_manifest`
- `workflow_canvas_get_node_smoke_coverage`
- `workflow_canvas_get_resource_catalog`
- `workflow_canvas_get_canvas_context`
- `workflow_canvas_get_bindable_variables`
- `workflow_canvas_add_node`
- `workflow_canvas_connect`
- `workflow_canvas_configure_node`
- `workflow_canvas_set_node_params`
- `workflow_canvas_delete_node`
- `workflow_canvas_delete_line`
- `workflow_canvas_clear_canvas`
- `workflow_canvas_auto_layout`
- `workflow_canvas_test_run`

后端工具的执行方式:

- 后端 tool run 不直接改数据库画布。
- 后端返回确定格式 ack:

```json
{"status":"dispatched_to_canvas","op":"add_node","args":{}}
```

- 前端收到 ack 后在当前浏览器画布里执行真实操作。

主要相关文件:

- `backend/domain/agent/singleagent/internal/agentflow/node_tool_workflow_canvas.go`
- `backend/domain/agent/singleagent/internal/agentflow/workflow_canvas_operation_guide.go`
- `backend/domain/agent/singleagent/internal/agentflow/workflow_canvas_node_catalog.go`
- `backend/domain/agent/singleagent/internal/agentflow/workflow_canvas_node_smoke_manifest.go`
- `backend/domain/agent/singleagent/internal/agentflow/workflow_canvas_node_smoke_coverage.go`
- `backend/domain/agent/singleagent/internal/agentflow/workflow_canvas_resource_catalog.go`

### 3.4 渐进式节点知识

已实现的设计方向:

1. 先用 `workflow_canvas_get_node_catalog` 获取可用节点目录。
2. 需要具体节点时，再用 `workflow_canvas_get_node_spec(type)` 获取完整配置说明。
3. 资源型节点先用 `workflow_canvas_get_resource_catalog` 获取插件、知识库、智能体、模型、数据库、MCP 等资源。
4. 配置条件、变量聚合、End returns 前必须调用 `workflow_canvas_get_bindable_variables`。
5. 完成一组 add/connect/configure/delete 后调用 `workflow_canvas_auto_layout` 和 `workflow_canvas_get_canvas_context` 审计。
6. 试运行失败后要求局部修复，不允许默认清空画布。

已经补充过的重点规则:

- Start/End 是 singleton，不能新增。
- type=13 输出节点偏展示，不应作为变量聚合的稳定下游来源。
- 固定文案分支需要下游变量时，优先用 type=15 文本处理节点产出真实 `output`，再聚合。
- 变量聚合 type=32 的 `merge_groups.variables` 必须来自可绑定变量列表。
- End 支持“返回变量”和“返回文本”，返回文本可把多个变量拼进文本里，并支持流式输出。
- 不需要输入的文本处理/输出节点，应支持清空默认 input。

### 3.5 多模态图片输入

已修复/实现:

- 前端上传 host 从错误的 `localhost:8888` 修到当前请求 host，226 页面上传不再打本机 localhost。
- 普通智能体和 finmallclaw workflow 面板发送图片时保留 `content_type:"mix"` 和 `item_list`，不再把图片丢成纯文本。
- 后端 `/api/conversation/chat` 的 mix 图片解析支持:
  - 上传后的 object key。
  - `data:image/...;base64,...`。
  - 常见裸 base64 前缀自动补 `data:image/png;base64,`。
  - 普通 URL 兜底。
- 对内网对象存储图片，后端优先读取 object bytes 并转成 data URL 给模型，避免百炼/vLLM 拉不到内网 MinIO URL。
- OpenAPI 路径也按同样思路处理: 消息入库仍可保存短 URL，模型输入单独使用 data URL。
- `qwen3.7-plus` 模型能力已配置 `input_modal=["text","image"]`。

主要相关文件:

- `backend/api/handler/coze/upload_service.go`
- `backend/api/handler/coze/upload_service_test.go`
- `backend/application/conversation/agent_run.go`
- `backend/application/conversation/openapi_agent_run.go`
- `backend/application/conversation/agent_run_multimodal_test.go`
- `frontend/packages/common/chat-area/chat-area/src/store/batch-upload-file.ts`
- `frontend/packages/common/chat-area/chat-area/src/hooks/file/use-upload.ts`
- `frontend/packages/common/chat-area/chat-area/src/utils/upload.ts`
- `frontend/packages/common/chat-area/chat-core/src/message/presend-local-message/presend-local-message-factory.ts`

已完成验证:

- 后端单测通过:

```bash
cd backend
SESSION_HMAC_SECRET=test-secret MOCKEY_CHECK_GCFLAGS=false \
go test ./application/conversation \
  -run 'TestParseMultiContentUsesDataURLForImageModelInput|TestParseMultiContentAcceptsDataURLImageWithoutObjectKey|TestActiveAgentRunRegistryCancelsAndUnregisters' \
  -count=1

SESSION_HMAC_SECRET=test-secret MOCKEY_CHECK_GCFLAGS=false \
go test ./domain/agent/singleagent/internal/agentflow \
  -run 'TestHistoryApproxBytesHandlesPartialMultiContent|TestContextCompactThresholdBytes' \
  -count=1
```

- 浏览器登录态直发 data URL 到 `/api/conversation/chat` 返回 200，模型能读出测试图标题 `MULTIMODAL OK`。

未完成验证:

- 上传 key -> 后端读对象 -> data URL -> `qwen3.7-plus` 的完整大图 OCR 验证被中断，下一轮应继续。
- 普通智能体 UI 选择图片发送后的端到端验证还需补。
- finmallclaw 面板 UI 选择图片发送后的端到端验证还需补。

## 4. 测试服务器部署步骤

### 4.1 VPN

如果本机无法访问 10.10.10.226，先检查或启动:

```bash
which hz-on
hz-on
pgrep -fl 'openvpn|hz-on'
```

当前机器上已经看到 openvpn 进程在运行。

### 4.2 后端部署

本地构建 Linux 二进制:

```bash
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/openynet main.go
```

上传和替换容器内二进制:

```bash
export SSHPASS='<server-password>'
sshpass -e scp -o StrictHostKeyChecking=no /tmp/openynet dev@10.10.10.226:/tmp/openynet

sshpass -e ssh -o StrictHostKeyChecking=no dev@10.10.10.226 '
set -e
TS=$(date +%Y%m%d%H%M%S)
docker cp coze-super:/app/openynet /tmp/openynet.bak-$TS
docker cp /tmp/openynet coze-super:/app/openynet
docker exec coze-super chmod +x /app/openynet
docker restart coze-super
docker ps --filter name=coze-super --format "{{.Names}} {{.Status}}"
'
```

健康检查:

```bash
sshpass -e ssh -o StrictHostKeyChecking=no dev@10.10.10.226 '
curl -sS -I http://127.0.0.1:8896/ | head
docker logs --since 2m coze-super 2>&1 | tail -80
'
```

### 4.3 前端部署

构建:

```bash
cd frontend/apps/coze-studio
IS_OPEN_SOURCE=false npx rsbuild build
```

上传静态资源，保留线上 `config.js`:

```bash
cd frontend/apps/coze-studio
tar --exclude='./config.js' -czf /tmp/coze-studio-dist.tgz -C dist .

export SSHPASS='<server-password>'
sshpass -e scp -o StrictHostKeyChecking=no /tmp/coze-studio-dist.tgz dev@10.10.10.226:/tmp/coze-studio-dist.tgz

sshpass -e ssh -o StrictHostKeyChecking=no dev@10.10.10.226 '
set -e
TS=$(date +%Y%m%d%H%M%S)
rm -rf /tmp/coze-studio-dist-$TS
mkdir -p /tmp/coze-studio-dist-$TS
tar -xzf /tmp/coze-studio-dist.tgz -C /tmp/coze-studio-dist-$TS
docker cp coze-super:/app/resources/static/config.js /tmp/config.js.$TS
docker cp /tmp/coze-studio-dist-$TS/. coze-super:/app/resources/static/
docker cp /tmp/config.js.$TS coze-super:/app/resources/static/config.js
'
```

前端只替换静态资源时通常不需要重启后端；若遇到 chunk 404 或浏览器缓存旧 bundle，需要刷新页面或清缓存。

### 4.4 本地开发连远程后端

推荐前端本地热更新 + API 代理到 226，避免每次上传静态资源:

```bash
cd frontend/apps/coze-studio
API_PROXY_TARGET=http://10.10.10.226:8896 pnpm dev -- --port 8897
```

打开:

```text
http://localhost:8897/work_flow?workflow_id=<workflow_id>&space_id=7652614054615187456
```

如果需要本地后端:

```bash
make server
make fe
```

但当前更高效的方式是本地前端代理远程后端。

## 5. 验证清单

### 5.1 基础验证

- 页面可加载，无 chunk 404。
- `/api/common/upload/apply_upload_action` 返回的 `UploadHosts` 不应是 `localhost:8888`，应是当前 host 或可访问域名。
- 普通智能体上传图片后，请求体应为 `content_type:"mix"`。
- finmallclaw 面板上传图片后，请求体也应为 `content_type:"mix"`，并保留 image item。
- 模型配置为 `qwen3.7-plus` 时，不应再返回“当前模型不支持多模态输入”。

### 5.2 多模态验证

用一张大字图，内容建议:

```text
MULTIMODAL OK
CODE 42
```

测试项:

- API 直传 data URL。
- API 直传裸 base64。
- 前端上传图片后用 object key 发送。
- 普通智能体页面发送。
- 工作流右侧 finmallclaw 面板发送。

预期:

```text
title=MULTIMODAL OK, code=42
```

### 5.3 画布工具验证

最小流程:

1. 让 finmallclaw “读取节点目录，做一个 Start -> 文本处理 -> End 的流程”。
2. 检查是否调用:
   - `workflow_canvas_get_node_catalog`
   - `workflow_canvas_get_node_spec`
   - `workflow_canvas_get_canvas_context`
   - `workflow_canvas_add_node`
   - `workflow_canvas_connect`
   - `workflow_canvas_get_bindable_variables`
   - `workflow_canvas_configure_node`
   - `workflow_canvas_auto_layout`
3. 检查画布是否只有一个 Start 和一个 End。
4. 检查文本处理节点是否没有多余未绑定 input。
5. 检查 End 是否正确返回文本或变量。

复杂流程:

- 意图识别 -> IF/选择器 -> 多个文本处理/LLM 分支 -> 变量聚合 -> End。
- 重点检查变量聚合只使用真实可绑定变量，不聚合 type=13 输出节点。

失败修复流程:

- 故意制造 End returns 空、变量不存在、BlockID empty。
- 要求 agent 试运行并修复。
- 预期行为是读取上下文和可绑定变量后局部修复，不清空画布。

## 6. 仍缺的功能和风险

### 6.1 节点能力未全部闭环

当前工具和说明已经覆盖大量节点，但不是所有节点都完成了“真实可执行语义配置器 + smoke 验证”。

已相对完整:

- Start/End 引用规则
- LLM
- 代码
- IF/选择器
- 文本处理
- 输出/纯输出
- 输入
- 变量聚合
- JSON 序列化
- 问答
- 智能体
- 卡片选择

仍需补齐或强化:

- 插件/API: 需要真实 plugin/api schema 和 fixture。
- 知识库检索/写入: 需要 dataset schema 和写入策略。
- SQL/数据库节点: 需要表字段发现、条件绑定和输出 schema。
- HTTP 请求: 需要 method/url/header/query/body/response schema 的语义配置器。
- 子工作流: 需要子工作流 schema 发现。
- 循环/批处理: 需要子画布、数组变量、循环输出的完整配置器。
- MCP 节点: 需要 server/tool 发现和参数 schema。
- 触发器、长期记忆、图像类节点: 资源发现和配置器仍不足。

### 6.2 画布上下文实时性

`workflow_canvas_get_canvas_context` 在同一轮对话里可能是用户发送消息前的快照。模型刚调用 add/connect/configure 后，如果马上读取上下文，可能仍看到旧状态。

现有提示已经要求模型结合本轮 ack 自检，但这不是最佳工程解。

建议下一步:

- 把 `get_canvas_context` 做成真正的浏览器实时读命令。
- 或者后端 tool run 等待前端执行结果回传后再返回最终上下文。
- 让工具返回从前端执行后的真实结果，而不是只返回 dispatched ack。

### 6.3 试运行结果获取

目前 `workflow_canvas_test_run` 能触发前端试运行，但稳定拿到结构化测试结果还不够。用户之前遇到的问题是测试会打开/覆盖原生试运行面板，导致右侧聊天区域被隐藏或重载。

建议下一步:

- 后端提供稳定 workflow test-run API，不依赖前端面板状态。
- tool 返回结构化结果:

```json
{
  "ok": false,
  "errors": [
    {"node_id":"...", "node_name":"...", "message":"引用变量不存在"}
  ],
  "outputs": {}
}
```

- finmallclaw 只消费这个结果并继续修复，不让原生试运行 UI 打断会话。

### 6.4 变量绑定仍是最大风险

典型失败:

- 输出节点 type=13 没有真实稳定输出变量，却被下游聚合。
- 文本处理节点固定文本不需要 input，但默认 input 没清掉。
- 变量聚合引用了旧节点、未定义变量或 display-only 输出。
- End returns 为空或引用不存在。

需要把“绑定审计”产品化:

- 每次 configure 后自动跑一次轻量校验。
- 给模型返回每个节点的可绑定变量、已绑定变量、未绑定字段。
- 对 type=32 变量聚合单独提供 `configure_merge_node` 这类强 schema 工具，减少自由 JSON 猜测。

### 6.5 多模态日志风险

当前后端把图片转成 data URL 给模型，这对内网是正确方向。但需要继续检查日志链路，避免把完整 base64 打进日志。

建议:

- `PreHandlerReq` / request debug log 对 `data:image/...;base64,...` 做截断或 redaction。
- 只记录 MIME、字节大小、hash、是否 data URL。

### 6.6 模型元数据需要清理

当前 `model_meta.id=2` 的 `model_name` 显示为 `gemini-3.5-flash`，但 `conn_config.model` 是 `qwen3.7-max`，语义不一致。

建议:

- 清理模型配置表，避免 UI 显示和实际调用模型不一致。
- 明确标记哪些模型支持 image。
- 对阿里百炼和 vLLM OpenAI-compatible 统一使用 `image_url.url = data:image/...;base64,...`。

### 6.7 工作区很脏，不能全量提交

当前仓库存在大量历史改动和临时 Playwright 文件。交付前必须只挑选本功能相关文件提交，不要 `git add .`。

临时文件示例:

- `mm-chat-response-187.txt`
- `mm-chat-network.txt`
- `mm-upload-network.txt`
- `ordinary-chat-request-*.json`
- `super-agent-playwright-*.json`

建议提交前清理或加入本地忽略，不要进入正式 commit。

## 7. 建议下一阶段计划

优先级 P0:

1. 完成多模态 UI 端到端验证:
   - 普通智能体上传图片。
   - finmallclaw 面板上传图片。
   - object key 路径读对象转 data URL。
2. 给 data URL 日志脱敏。
3. 校准测试 bot 和 finmallclaw 的模型都使用 `qwen3.7-plus`。
4. 修复试运行不应隐藏/重载聊天面板的问题，优先走后端 test-run API。

优先级 P1:

1. 把 `get_canvas_context` 改为执行后实时上下文，而不是旧快照。
2. 为 type=32 变量聚合、End 返回、文本处理固定文本提供强 schema 配置工具。
3. 每次 add/connect/configure/delete 后自动布局和自动绑定审计。
4. 建立节点 smoke 测试矩阵，用临时 workflow 逐节点验证可添加、可配置、可运行、可清理。

优先级 P2:

1. 补齐资源型节点的资源发现和 fixture:
   - 插件/API
   - 知识库
   - 数据库
   - 子工作流
   - MCP
   - 卡片
2. 建一个复杂转账场景作为回归样例:
   - 转账意图识别
   - 缺账号追问
   - 缺卡号查最近收款人
   - 查询最近转账记录
   - 风险校验
   - 转账确认
   - 成功/失败统一返回文本

## 8. 回滚方案

后端二进制替换前会备份:

```text
/tmp/openynet.bak-<timestamp>
```

回滚:

```bash
export SSHPASS='<server-password>'
sshpass -e ssh -o StrictHostKeyChecking=no dev@10.10.10.226 '
docker cp /tmp/openynet.bak-<timestamp> coze-super:/app/openynet
docker exec coze-super chmod +x /app/openynet
docker restart coze-super
'
```

前端静态资源回滚需要有对应 dist 包或从镜像/备份恢复。部署时建议额外备份:

```bash
docker cp coze-super:/app/resources/static /tmp/static.bak-<timestamp>
```

## 9. 结论

当前已经完成的是“能让模型通过工具调画布”的主链路，以及多模态在内网场景下必须走 base64/data URL 的后端适配。真正要达到可交付产品，还差两个硬点:

1. 画布工具必须从 ack 模式升级为执行后可回读真实结果，尤其是上下文、可绑定变量和试运行结果。
2. 每类节点都要有可执行的语义配置器和 smoke 验证，尤其是变量聚合、End 返回、资源型节点和循环/批处理。

下一阶段不建议继续堆提示词。应该用“节点能力矩阵 + 强 schema 配置工具 + 后端 test-run API + 自动绑定审计”把模型从猜测拉回确定性流程。
