# Ynet Studio OpenAPI 测试文档

## 测试环境配置

| 配置项 | 值 |
|--------|-----|
| 服务地址 | `http://localhost:8888` |
| PAT 令牌 | `pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8` |
| 智能体 ID | `7582154049642823680` |
| 工作流 ID | `7582154798485471232` |
| 空间 ID | `7582149611540709376` |

---

## 一、智能体 OpenAPI 测试

### 1.1 创建会话

```bash
curl -X POST "http://localhost:8888/v1/conversation/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "bot_id": "7582154049642823680"
  }'
```

**预期响应**：
```json
{
  "code": 0,
  "msg": "",
  "data": {
    "id": "7582418689991901184",
    "created_at": 1765419424,
    "meta_data": null,
    "connector_id": "1024",
    "last_section_id": "7582418689991917568"
  }
}
```

> 📝 记录返回的 `id` 作为 `conversation_id`，用于后续对话

---

### 1.2 发起对话（流式）

```bash
curl -X POST "http://localhost:8888/v3/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "bot_id": "7582154049642823680",
    "conversation_id": "替换为上一步返回的conversation_id",
    "user_id": "test_user_001",
    "stream": true,
    "additional_messages": [
      {
        "role": "user",
        "content": "你好，请介绍一下你自己",
        "content_type": "text"
      }
    ]
  }'
```

**预期响应**（SSE 流式）：
```
event:conversation.chat.created
data:{"id":"xxx","conversation_id":"xxx","bot_id":"7582154049642823680",...}

event:conversation.message.delta
data:{"content":"你好","content_type":"text",...}

event:conversation.message.completed
data:{"content":"完整回复内容",...}

event:conversation.chat.completed
data:{"status":"completed",...}

event:conversation.stream.done
data:
```

---

### 1.3 快速测试（无需提前创建会话）

如果不传 `conversation_id`，系统会自动创建新会话：

```bash
curl -X POST "http://localhost:8888/v3/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "bot_id": "7582154049642823680",
    "user_id": "test_user_001",
    "stream": true,
    "additional_messages": [
      {
        "role": "user",
        "content": "你好",
        "content_type": "text"
      }
    ]
  }'
```

---

### 1.4 测试记忆功能

**第一轮：让智能体记住名字**
```bash
curl -X POST "http://localhost:8888/v3/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "bot_id": "7582154049642823680",
    "user_id": "test_user_001",
    "stream": true,
    "additional_messages": [
      {
        "role": "user",
        "content": "我叫张三，请记住我的名字",
        "content_type": "text"
      }
    ]
  }'
```

**第二轮：验证记忆**
```bash
curl -X POST "http://localhost:8888/v3/chat" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "bot_id": "7582154049642823680",
    "user_id": "test_user_001",
    "stream": true,
    "additional_messages": [
      {
        "role": "user",
        "content": "你还记得我叫什么名字吗？",
        "content_type": "text"
      }
    ]
  }'
```

> ✅ 智能体应该能回忆起 "张三" 这个名字

---

## 二、工作流 OpenAPI 测试

### 2.1 同步运行工作流

```bash
curl -X POST "http://localhost:8888/v1/workflow/run" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "workflow_id": "7582154798485471232",
    "parameters": {
      "input": "你好，工作流测试"
    }
  }'
```

**预期响应**：
```json
{
  "code": 0,
  "msg": null,
  "data": "{\"output\":\"你好呀！😊 很高兴见到你～...\"}",
  "token": 44,
  "cost": "0.00000",
  "debug_url": "http://127.0.0.1:3000/work_flow?execute_id=xxx&space_id=xxx&workflow_id=xxx&execute_mode=2",
  "execute_id": "xxx"
}
```

---

### 2.2 流式运行工作流

```bash
curl -X POST "http://localhost:8888/v1/workflow/stream_run" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "workflow_id": "7582154798485471232",
    "parameters": {
      "input": "请用3句话介绍人工智能"
    }
  }'
```

**预期响应**（SSE 流式）：
```
id: 0
event: message
data: {"content":"{\"output\":\"...\"}","content_type":"text","node_seq_id":"0","node_id":"900001","node_is_finish":true,"node_type":"End","node_title":"结束"}

id: 1
event: done
data: {"debug_url":"http://127.0.0.1:3000/work_flow?execute_id=xxx..."}
```

---

### 2.3 创建工作流会话（可选）

工作流也支持会话模式：

```bash
curl -X POST "http://localhost:8888/v1/workflow/conversation/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer pat_10690f59830f261f9750a14a12d929f126ca3a41cf30e3c6ce0e96dffaf7e2e8" \
  -d '{
    "workflow_id": "7582154798485471232"
  }'
```

---

## 三、API 端点汇总

| 功能 | 方法 | 端点 |
|------|------|------|
| 创建智能体会话 | POST | `/v1/conversation/create` |
| 智能体对话 | POST | `/v3/chat` |
| 查询会话列表 | GET | `/v1/conversations` |
| 清空会话 | POST | `/v1/conversations/:conversation_id/clear` |
| 工作流同步运行 | POST | `/v1/workflow/run` |
| 工作流流式运行 | POST | `/v1/workflow/stream_run` |
| 创建工作流会话 | POST | `/v1/workflow/conversation/create` |
| 获取工作流信息 | GET | `/v1/workflows/:workflow_id` |
| 获取运行历史 | GET | `/v1/workflow/get_run_history` |

---

## 四、请求参数说明

### 智能体对话参数 (`/v3/chat`)

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| bot_id | string | ✅ | 智能体 ID |
| user_id | string | ✅ | 用户标识，用于数据隔离 |
| conversation_id | string | ❌ | 会话 ID，不传则自动创建 |
| stream | boolean | ❌ | 是否流式返回，默认 true |
| additional_messages | array | ❌ | 消息列表 |
| custom_variables | object | ❌ | 自定义变量 |
| meta_data | object | ❌ | 元数据 |

### 工作流运行参数 (`/v1/workflow/run`)

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workflow_id | string | ✅ | 工作流 ID |
| parameters | object | ❌ | 输入参数，key-value 格式 |
| bot_id | string | ❌ | 关联的智能体 ID |
| ext | object | ❌ | 扩展参数 |

---

## 五、错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 700012006 | Session 认证失败 |
| 700012007 | PAT 令牌无效 |
| 4001xxx | 参数错误 |
| 5001xxx | 服务内部错误 |

---

## 六、测试检查清单

- [ ] 创建智能体会话成功
- [ ] 智能体对话正常响应
- [ ] 智能体记忆功能正常
- [ ] 工作流同步运行成功
- [ ] 工作流流式运行成功
- [ ] PAT 令牌认证正常

---

*文档更新时间：2025-12-11*
