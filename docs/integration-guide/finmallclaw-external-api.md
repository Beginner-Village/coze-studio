# 外部系统对接 finmallclaw 智能体 —— API 集成指南

本文档面向**平台外部的业务系统**，说明如何通过 HTTP API 调用部署在 Studio 平台上的智能体
（如 `finmallclaw` 这类超级智能体 / Super-Agent），实现「发消息 → 智能体思考并调用工具（含沙箱执行）→ 拿回答」。

> 本文所有接口、请求体、返回体、SSE 事件均已在 **226 测试环境**（`http://10.10.10.226:8896`）
> 用真实 PAT 端到端跑通，实测记录见文末「附录 A」。

---

## 0. 一分钟速览

对接只需三步：

1. **拿一个 API Key（PAT）** —— 在平台「个人访问令牌」页生成，或用登录态调创建接口（见 §2）。
2. **建会话** —— `POST /api/super-agent/sessions/create`，拿到 `conversation_id`（见 §4.1）。
3. **发消息拿回答** —— 二选一：
   - 同步：`POST /api/super-agent/runs/create`，一次请求阻塞返回完整回答（见 §4.2，推荐用于服务端集成）。
   - 流式：`POST /api/super-agent/runs/stream`，SSE 实时推送打字机效果 + 工具调用过程（见 §4.3，推荐用于前端/需要展示中间过程）。

一个最小可用的同步调用：

```bash
BASE=http://10.10.10.226:8896
PAT=pat_xxxxxxxxxxxxxxxx          # 你的 API Key
AGENT=7540939700521926656          # finmallclaw 的智能体 ID
SPACE=7532755646102372352          # 该智能体所属空间 ID

# 1) 建会话
CONV=$(curl -s -X POST "$BASE/api/super-agent/sessions/create" \
  -H "Authorization: Bearer $PAT" -H 'Content-Type: application/json' \
  -d "{\"agent_id\":\"$AGENT\",\"space_id\":\"$SPACE\"}" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["session"]["conversation_id"])')

# 2) 发消息（同步拿回答）
curl -s -X POST "$BASE/api/super-agent/runs/create" \
  -H "Authorization: Bearer $PAT" -H 'Content-Type: application/json' \
  -d "{\"agent_id\":\"$AGENT\",\"space_id\":\"$SPACE\",\"conversation_id\":\"$CONV\",\"user_id\":\"ext-user-1\",\"additional_messages\":[{\"role\":\"user\",\"content\":\"你好\",\"content_type\":\"text\"}]}"
```

---

## 1. 基本信息

| 项 | 值 |
|---|---|
| Base URL（测试环境） | `http://10.10.10.226:8896` |
| 协议 | HTTP/1.1，请求/响应体为 JSON（流式为 SSE） |
| 鉴权 | HTTP Header `Authorization: Bearer <PAT>` |
| 接口前缀 | `/api/super-agent/*` |
| 自描述清单 | `GET /api/super-agent/manifest`（返回全部能力、路由、事件、请求 schema） |
| OpenAPI Schema | `GET /api/super-agent/openapi.json`（3.1.0，可导入 Postman/Apifox） |
| 统一返回包 | `{"code":0,"msg":"success","data":{...}}`，`code!=0` 为错误 |

> **生产环境**请把 `BASE_URL` 换成对外网关地址，其余不变。`manifest` / `openapi.json` 两个自描述接口
> 是「随代码演进的权威说明」，对接前建议先 `curl` 一遍取最新字段。

---

## 2. 鉴权：获取 API Key（PAT）

超级智能体的对外接口用 **个人访问令牌（Personal Access Token, PAT）** 做 Bearer 鉴权。
PAT 归属某个平台用户，**只能调用该用户有权限的智能体**（平台已按资源做 owner 校验，越权会被拒）。

### 方式一：平台 UI 生成（推荐）
登录平台 → 头像 / 设置 → **API 授权 / 个人访问令牌** → 新建，复制形如 `pat_xxx` 的令牌。
令牌明文只在创建时返回一次，请妥善保存。

### 方式二：接口生成（需登录态 Cookie）
```bash
# a) 登录拿 session cookie
curl -s -c cookies.txt -X POST "$BASE/api/passport/web/email/login/" \
  -H 'Content-Type: application/json' \
  -d '{"email":"<账号邮箱>","password":"<密码>"}'

# b) 用 cookie 创建 PAT（expire_at 为到期 Unix 秒）
EXP=$(( $(date +%s) + 30*24*3600 ))
curl -s -b cookies.txt -X POST \
  "$BASE/api/permission_api/pat/create_personal_access_token_and_permission" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"finmallclaw-integration\",\"expire_at\":$EXP,\"duration_day\":\"30\"}"
# 返回 data.token 即 PAT 明文
```

> 安全建议：为对接方单独建一个 PAT、设置合理有效期、不要把 PAT 写进代码仓库或前端；
> 泄露后到令牌管理页删除即可失效。

---

## 3. 定位对接参数：agent_id / space_id / user_id

| 参数 | 含义 | 怎么拿 |
|---|---|---|
| `agent_id` | finmallclaw 智能体的 ID（雪花 ID，字符串） | 打开该智能体的编辑/详情页，URL 里的 bot/agent id；或让平台侧提供 |
| `space_id` | 智能体所属**空间** ID | 智能体详情页 URL 中的 space id；或平台侧提供 |
| `user_id` | 由对接方自定义的**终端用户标识** | 任意字符串，用于区分不同终端用户的会话与记忆（如 `ext-user-1001`）。不传时后端默认用 `api-user-<PAT用户ID>` |

`agent_id` 与 `bot_id` 二选一即可（含义相同，`bot_id` 为兼容字段）。

> 关于「超级体」：当你用 PAT 调 `/api/super-agent/runs/*` 时，后端会自动把该智能体按
> **超级智能体 harness** 运行（自动注入 `super_agent_app_server=true` 标记），因此无需在数据库里
> 把智能体标成 super，也无需自己传该标记。

---

## 4. 核心接口

### 4.1 建会话 `POST /api/super-agent/sessions/create`

多轮对话前先建一个会话，拿到 `conversation_id`（后续所有轮次都带上它）。

请求体：

| 字段 | 必填 | 说明 |
|---|---|---|
| `agent_id` / `bot_id` | 二选一必填 | 智能体 ID |
| `space_id` | 建议 | 空间 ID |
| `title` | 否 | 会话标题（≤128 字符） |
| `user_id` | 否 | 终端用户标识 |
| `connector_id` | 否 | 渠道标识，一般不传 |

实测返回：
```json
{"code":0,"msg":"success","data":{"session":{
  "session_id":"7659722333023633408",
  "conversation_id":"7659722333023633408",
  "section_id":"7659722333023649792",
  "agent_id":"7642226103964139520",
  "connector_id":"10000010","scene":4,
  "title":"ext-api-test","renamable":true,
  "created_at":1783418081008,"updated_at":1783418081008
}}}
```
取 `data.session.conversation_id` 用于后续发消息。

### 4.2 发消息（同步）`POST /api/super-agent/runs/create`

**一次请求阻塞直到智能体回答完毕**，返回体里直接带完整消息列表。适合服务端后台集成，最省事。

请求体：

| 字段 | 必填 | 说明 |
|---|---|---|
| `agent_id` / `bot_id` | 二选一必填 | 智能体 ID |
| `space_id` | 建议 | 空间 ID |
| `conversation_id` | 建议 | §4.1 拿到的会话 ID（不传则单轮，无上下文） |
| `user_id` | 建议 | 终端用户标识 |
| `additional_messages` | 必填 | 本轮用户消息数组，见下 |
| `custom_variables` | 否 | 提示词变量（JSON） |
| `meta_data` | 否 | 透传业务元数据 |

`additional_messages` 元素结构：
```json
{"role":"user","content":"你的问题","content_type":"text"}
```

实测返回（截断）：
```json
{
  "code":0,"msg":"success",
  "conversation_id":"7659722823681703936",
  "bot_id":"7540939700521926656",
  "status":"completed",
  "messages":[
    {"role":"assistant","type":"function_call","content":"{...run_bash...}"},
    {"role":"assistant","type":"tool_response","content":"exit_code: 0\nstdout:\n..."},
    {"role":"assistant","type":"answer","content":"最终回答文本"},
    {"role":"assistant","type":"verbose","content":"{...generate_answer_finish...}"}
  ]
}
```
**取最终回答**：遍历 `messages`，取 `type == "answer"` 的 `content`。
中间的 `function_call` / `tool_response` 是工具调用轨迹（如沙箱命令执行），需要审计/展示时可用。

### 4.3 发消息（流式 SSE）`POST /api/super-agent/runs/stream`

请求体同 §4.2。响应为 `text/event-stream`，按事件实时推送。适合前端打字机效果、实时展示工具执行。

事件序列（实测）：
```
event:conversation.chat.created      # 本轮 run 创建
event:conversation.chat.in_progress  # 开始处理
event:conversation.ack               # 已收到用户消息（回显）
event:conversation.message.delta     # 回答增量（多条，逐字/逐段）
event:conversation.message.completed # 一条完整消息（answer / function_call / tool_response / verbose）
event:conversation.chat.completed    # 本轮结束
event:conversation.stream.done       # 流结束（收到即可关闭连接）
```

全部事件类型（来自 manifest）：
`conversation.ack`、`conversation.chat.created`、`conversation.chat.in_progress`、
`conversation.message.delta`、`conversation.message.completed`、`conversation.chat.completed`、
`conversation.chat.failed`、`conversation.chat.cancelled`、`conversation.stream.done`、`conversation.error`。

- 结束事件：`conversation.stream.done`
- 错误事件：`conversation.error`（或 `conversation.chat.failed`）

每个 `data:` 是一段 JSON。`message.completed` 的 `type` 字段区分内容：
| type | 含义 |
|---|---|
| `ack` | 用户输入回显 |
| `function_call` | 智能体决定调用某工具（如 `run_bash`），`content` 内含工具名与参数 |
| `tool_response` | 工具返回（如沙箱 `exit_code/stdout/stderr`），`meta_data.time_cost` 为耗时秒数 |
| `answer` | 面向用户的最终回答文本 |
| `verbose` | 内部状态标记（如 `generate_answer_finish`），可忽略 |

### 4.4 多轮对话

用**同一个 `conversation_id`** 反复调 §4.2 / §4.3 即可，平台自动维护上下文与记忆。
（另有 `POST /api/super-agent/runs/reply` 语义等价，`conversation_id` 必填。）

### 4.5 拉取历史消息 `POST /api/super-agent/messages/list`

| 字段 | 必填 | 说明 |
|---|---|---|
| `conversation_id` | 必填 | 会话 ID |
| `limit` | 否 | 每页条数（≤100） |
| `order_by` | 否 | `ASC` / `DESC` |
| `before_id` / `after_id` | 否 | 游标翻页 |

实测返回 `data.messages`（或 `message_list`）为消息数组，含 `role/type/content`。

### 4.6 取消运行 `POST /api/super-agent/runs/cancel`
请求体带 `run_id`（或 `conversation_id`），用于中断一次进行中的流式运行。

---

## 5. 沙箱能力说明（重点）

finmallclaw 这类超级体可以在**隔离沙箱容器**里执行命令 / 读写文件（工具 `run_bash`、`write_file` 等）。
是否给某次运行挂载这些沙箱工具，取决于两个条件（满足其一即挂载）：

1. 后端环境变量 `SANDBOX_TOOLS_ENABLED=true`（全局强制开启）；**或**
2. 该智能体**绑定了至少一个技能（Skill）**。

前置条件（平台侧，已在 226 就绪）：
- 后端 `SANDBOX_ENABLED=true`（沙箱总开关，2026-07-07 已在 226 重新开启）；
- 后端容器挂载了 `/var/run/docker.sock`，且本机存在沙箱基础镜像 `ynet-sandbox:rich`。

运行时每个智能体会话会拉起一个独立沙箱容器，命名形如 `ynet-sb-a<agentId>-<hash>`，空闲后自动回收。

> 若你调用后智能体回复「无法在沙箱中执行命令」，说明该智能体**没绑技能**且未开全局
> `SANDBOX_TOOLS_ENABLED`，此时它只做纯对话。给该智能体绑定一个技能即可获得沙箱执行能力。

---

## 6. 错误处理

- HTTP 200 且 `code==0` 为成功；`code!=0` 时 `msg` 为错误原因。
- 401 / `ErrUserAuthenticationFailed`：`Authorization` 头缺失或 PAT 无效/过期。
- 越权（调了非本 PAT 用户拥有的智能体）：返回权限错误。
- 流式过程中出错：会收到 `conversation.error` 或 `conversation.chat.failed` 事件。
- `model not found`：目标空间未绑定模型，需在平台为该空间配置可用模型。

---

## 7. 完整对接示例（Node.js，同步方式）

```js
const BASE = 'http://10.10.10.226:8896';
const PAT = process.env.FINMALLCLAW_PAT;      // pat_xxx
const AGENT = '7540939700521926656';
const SPACE = '7532755646102372352';

async function call(path, body) {
  const r = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${PAT}`, 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  return r.json();
}

async function main() {
  // 1) 建会话
  const s = await call('/api/super-agent/sessions/create', { agent_id: AGENT, space_id: SPACE });
  const conversation_id = s.data.session.conversation_id;

  // 2) 发消息（同步）
  const resp = await call('/api/super-agent/runs/create', {
    agent_id: AGENT, space_id: SPACE, conversation_id, user_id: 'ext-user-1001',
    additional_messages: [{ role: 'user', content: '帮我算一下 1+1', content_type: 'text' }],
  });

  // 3) 取最终回答
  const answer = (resp.messages || []).filter(m => m.type === 'answer').map(m => m.content).join('');
  console.log('回答:', answer);
}
main();
```

---

## 附录 A：226 环境端到端实测记录（2026-07-07）

在 `http://10.10.10.226:8896` 用真实 PAT 跑通，证据：

1. **鉴权**：登录测试账号 → 创建 PAT `pat_7028...`（成功，`code:0`）。
2. **建会话**：`sessions/create` 返回 `conversation_id=7659722333023633408`，`connector_id=10000010`（API 渠道）。
3. **同步发消息**：`runs/create` 直接返回 `status:"completed"` + 完整 `messages`，`answer` = `1+1等于2`。
4. **流式发消息**：`runs/stream` 事件序列完整 —— `chat.created → chat.in_progress → ack →
   message.delta ×N → message.completed → chat.completed → stream.done`。
5. **沙箱真实执行**（用一个绑定了技能的智能体 `7540939700521926656`）：
   - `function_call`：工具 `run_bash`，参数 `echo SANDBOX_OK_$(date +%s)`；
   - `tool_response`：`exit_code: 0` / `stdout: SANDBOX_OK_1783418201`，耗时 3.3s；
   - 对应沙箱容器 `ynet-sb-a7540939700521926656-ud6bc5b127c2d80fadcb2  Up`（基于 `ynet-sandbox:rich`）。
6. **多轮 / 历史**：`messages/list` 正确返回同会话历史消息。

> 说明：实测所用 `agent_id=7540939700521926656` 是测试账号下一个绑定了技能的智能体，用于验证
> **外部 API + 沙箱执行**全链路。对接你自己的 `finmallclaw` 时，把 `agent_id / space_id / PAT`
> 换成 finmallclaw 对应的值即可，流程与返回结构完全一致。

## 附录 B：全部 super-agent 接口清单

| 路由 | 用途 |
|---|---|
| `POST /api/super-agent/sessions/{create,get,list,rename,delete}` | 会话管理 |
| `POST /api/super-agent/runs/{create,stream,reply,get,list,cancel}` | 运行（发消息）|
| `POST /api/super-agent/messages/list` | 历史消息 |
| `POST /api/super-agent/runtime-config/{get,update,delete}` | 会话级运行时配置（换模型/绑 MCP/技能）|
| `POST /api/super-agent/workspace/{list,read,upload,download,delete}` | 沙箱工作区文件 |
| `POST /api/super-agent/artifacts/{list,download,move,delete}` | 产物文件 |
| `POST /api/super-agent/approvals/{list,resolve}` | 高危操作审批 |
| `POST /api/super-agent/traces/get` | 运行轨迹 |
| `GET  /api/super-agent/manifest`、`GET /api/super-agent/openapi.json` | 自描述 |

完整字段以 `GET /api/super-agent/openapi.json` 为准。
