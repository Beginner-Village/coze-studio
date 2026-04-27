# Intent Hub — 意图识别中台设计文档

> 日期: 2026-04-16
> 状态: Draft
> 作者: luzhipeng

## 1. 概述

### 1.1 背景

当前多智能体管理分散在工作流节点（IntentDetector + Agent Node）中，缺乏统一的意图分发、上下文管理和智能体注册能力。随着 ChatFlow 智能体数量增长，需要一个独立的意图识别中台来统一管理多智能体的注册、分发和对话编排。

### 1.2 目标

构建一个**完全独立的意图识别中台（Intent Hub）**，具备：

- 智能体注册与管理（ChatFlow 通过接口注册接入）
- 分层意图识别与分发（一级粗分类 → 二级精确匹配）
- 渐进式技能披露（避免 LLM 工具列表过长导致准确率下降）
- 平台级上下文管理（分层上下文，中台管路由层，ChatFlow 管业务层）
- 基于上下文的 Query 自动重组（提取槽位、整合信息后发给子智能体）
- 透明代理模式（用户无感知地与多个 ChatFlow 交互）
- 完整的管理后台（注册、配置、调试、审计）

### 1.3 核心决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| 技术栈 | Python (FastAPI + LangGraph) | 独立服务，LangGraph 提供状态管理和图编排 |
| 架构模式 | FastAPI 管"壳"，LangGraph 管"脑" | HTTP/管理归 FastAPI，AI 路由归 LangGraph，职责分离 |
| 意图披露 | 分层披露（A 方案） | 匹配二级意图架构，解决 LLM 工具数量退化问题 |
| 上下文管理 | 分层上下文（C 方案） | 中台管路由上下文+槽位，ChatFlow 管自己的业务上下文 |
| 用户交互 | 透明代理模式（A 方案） | 中台全程代理，捕获意图切换，用户无感知 |

---

## 2. 系统架构

```
┌──────────────────────────────────────────────────────────────────┐
│                    意图识别中台 (Intent Hub)                       │
│                                                                    │
│  ┌─────────────┐  ┌──────────────────────────────────────────┐   │
│  │  管理后台     │  │           FastAPI 服务层                  │   │
│  │  (Vue/React) │  │                                          │   │
│  │             │──│  /api/admin/*    管理CRUD                 │   │
│  │  - 智能体管理│  │  /api/v1/chat    对外会话(SSE)            │   │
│  │  - 技能分类  │  │  /api/v1/session 会话管理                 │   │
│  │  - 模型配置  │  │  /api/analytics  统计分析                 │   │
│  │  - 会话日志  │  │                                          │   │
│  │  - 数据统计  │  └───────────┬──────────────────────────────┘   │
│  │  - 调试沙盒  │              │                                   │
│  └─────────────┘              │                                   │
│                    ┌──────────▼──────────┐                        │
│                    │   LangGraph 意图引擎  │                       │
│                    │                      │                        │
│                    │  入口路由              │                       │
│                    │    ↓                  │                        │
│                    │  意图切换检测          │                        │
│                    │    ↓                  │                        │
│                    │  一级意图分类(粗粒度)   │                       │
│                    │    ↓                  │                        │
│                    │  技能展开              │   ┌─────────────────┐ │
│                    │    ↓                  │   │  State Store     │ │
│                    │  二级意图选择(ChatFlow) │   │  (Redis + DB)    │ │
│                    │    ↓                  │   │  - 会话状态       │ │
│                    │  槽位提取 + Query重组   │   │  - 槽位数据       │ │
│                    │    ↓                  │   │  - 意图轨迹       │ │
│                    │  ChatFlow调用          │   └─────────────────┘ │
│                    │    ↓                  │                        │
│                    │  结果处理 + 状态更新    │                       │
│                    └─────────────────────┘                        │
└──────────────────────────────────────────────────────────────────┘
          │                    │                    │
    ┌─────▼─────┐        ┌────▼─────┐        ┌────▼─────┐
    │ ChatFlow A │        │ ChatFlow B│        │ ChatFlow C│
    │  (转账)    │        │  (理财)   │        │  (查询)   │
    └───────────┘        └──────────┘        └──────────┘
```

### 2.1 模块职责

| 模块 | 技术 | 职责 |
|------|------|------|
| FastAPI 服务层 | FastAPI + SSE (sse-starlette) | HTTP 入口、管理 API、认证鉴权、SSE 推流、审计日志 |
| LangGraph 意图引擎 | LangGraph + LangChain | 分层意图识别、状态流转、上下文管理、ChatFlow 调用编排 |
| State Store | Redis + MySQL | 会话状态持久化、槽位存储、LangGraph Checkpointer |
| 管理后台 | Vue3 / React | ChatFlow 注册、技能配置、模型管理、日志查看、数据统计、调试沙盒 |

### 2.2 与现有系统关系

Intent Hub 是完全独立的项目，不依赖 Studio/Loop/Guard 的代码。它通过 HTTP API 调用已注册的 ChatFlow 智能体（这些 ChatFlow 可以是 Studio 中的 Bot、也可以是任何符合协议的外部服务）。对外暴露的会话 API 与 Studio 的 `/api/conversation/chat` 形式对齐。

---

## 3. 数据模型

### 3.1 chatflow_agent — ChatFlow 智能体注册表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | 雪花 ID |
| name | VARCHAR(128) | 名称，如"转账服务" |
| description | TEXT | 功能描述（供 LLM 意图识别使用） |
| endpoint | VARCHAR(512) | 调用地址 |
| auth_type | ENUM('none','bearer','api_key') | 认证方式 |
| auth_credential | VARCHAR(512) | 加密存储的认证凭证 |
| protocol | ENUM('sse','http_json') | 对接协议 |
| status | ENUM('active','inactive','testing') | 状态 |
| timeout_ms | INT DEFAULT 30000 | 超时时间 |
| metadata | JSON | 扩展字段 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### 3.2 skill_category — 技能分类（一级意图类别）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| name | VARCHAR(128) | 如"金融交易服务" |
| description | TEXT | 给 LLM 看的类别描述 |
| icon | VARCHAR(256) | 管理后台图标 |
| sort_order | INT | 排序权重 |
| status | ENUM('active','inactive') | 状态 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### 3.3 skill_chatflow_mapping — 技能 ↔ ChatFlow 关联

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| category_id | BIGINT FK | 关联 skill_category |
| chatflow_id | BIGINT FK | 关联 chatflow_agent |
| skill_name | VARCHAR(128) | 二级技能名，如"转账" |
| skill_desc | TEXT | 给 LLM 看的技能描述 |
| sort_order | INT | 排序 |
| extract_slots | JSON | 需提取的槽位定义，如 `[{"name":"amount","type":"number","desc":"转账金额","required":true}]` |

### 3.4 model_config — 模型配置

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| name | VARCHAR(128) | 配置名，如"默认意图识别" |
| provider | VARCHAR(64) | openai / azure / local |
| model_name | VARCHAR(128) | gpt-4o / qwen-max 等 |
| api_endpoint | VARCHAR(512) | API 地址 |
| api_key | VARCHAR(512) | 加密存储 |
| temperature | FLOAT DEFAULT 0.1 | |
| max_tokens | INT DEFAULT 1024 | |
| is_default | BOOLEAN | 是否为默认配置 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### 3.5 prompt_template — 系统提示词模板

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| name | VARCHAR(128) | 如"一级意图识别 prompt" |
| type | ENUM('intent_l1','intent_l2','query_rewrite','slot_extract') | 模板类型 |
| content | TEXT | prompt 内容，支持 `{{variables}}` |
| variables | JSON | 可用变量列表 |
| is_active | BOOLEAN | 是否启用 |
| version | INT | 版本号，支持回滚 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### 3.6 conversation — 会话

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| external_id | VARCHAR(64) UNIQUE | 对外暴露的会话 ID |
| user_id | VARCHAR(128) | 用户标识 |
| status | ENUM('active','closed') | |
| current_intent | VARCHAR(128) | 当前所在意图 |
| current_chatflow_id | BIGINT | 当前对话的 ChatFlow |
| slot_state | JSON | 当前累积槽位 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

### 3.7 message — 消息记录

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| conversation_id | BIGINT FK | |
| role | ENUM('user','assistant','system') | |
| content | TEXT | |
| source | ENUM('user','platform','chatflow') | 消息来源 |
| chatflow_id | BIGINT NULL | 来自哪个 ChatFlow |
| intent_snapshot | JSON | 该消息触发时的意图快照 |
| created_at | DATETIME | |

### 3.8 intent_trace — 意图轨迹（审计用）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT PK | |
| conversation_id | BIGINT FK | |
| message_id | BIGINT FK | |
| level | ENUM('l1','l2') | 意图层级 |
| input_text | TEXT | 输入文本 |
| category_id | BIGINT | 一级分类结果 |
| chatflow_id | BIGINT | 二级选择结果 |
| confidence | FLOAT | 置信度 |
| reasoning | TEXT | 模型推理过程 |
| slots_extracted | JSON | 提取的槽位 |
| rewritten_query | TEXT | 重组后的 query |
| latency_ms | INT | 处理延迟 |
| created_at | DATETIME | |

---

## 4. API 设计

### 4.1 对外会话 API（与 Studio 接口形式对齐）

#### 创建会话

```
POST /api/v1/conversation/create
Body: { "user_id": "string", "metadata": {} }
Response: { "conversation_id": "string", "created_at": "datetime" }
```

#### 发起对话（SSE 流式）

```
POST /api/v1/conversation/chat
Body: { "conversation_id": "string", "query": "string", "content_type": "text" }

SSE Events:
  event: intent        → { "level": "l1", "category": "金融交易", "confidence": 0.96 }
  event: intent        → { "level": "l2", "skill": "转账", "chatflow": "ChatFlow-001", "confidence": 0.93 }
  event: slot          → { "extracted": {"payee": "张三", "amount": 5000} }
  event: message_start → { "message_id": "string" }
  event: message_delta → { "content": "请问从" }
  event: message_delta → { "content": "哪个账户转出？" }
  event: message_end   → { "finish_reason": "stop" }
  event: error         → { "code": "string", "message": "string" }
```

#### 获取消息列表

```
GET /api/v1/conversation/{id}/messages
Response: { "messages": [{ "id", "role", "content", "source", "created_at" }] }
```

#### 清除上下文

```
POST /api/v1/conversation/{id}/reset
Response: { "section_id": "string" }
```

#### 中断生成

```
POST /api/v1/conversation/{id}/break
Response: { "success": true }
```

### 4.2 管理后台 API

#### ChatFlow 智能体管理

```
POST   /api/admin/chatflows              创建
GET    /api/admin/chatflows              列表（分页 + 搜索）
GET    /api/admin/chatflows/{id}         详情
PUT    /api/admin/chatflows/{id}         更新
DELETE /api/admin/chatflows/{id}         删除（软删除）
POST   /api/admin/chatflows/{id}/test    连通性测试
```

#### 技能分类管理

```
POST   /api/admin/categories             创建
GET    /api/admin/categories             列表（含关联 ChatFlow 数量）
PUT    /api/admin/categories/{id}        更新
DELETE /api/admin/categories/{id}        删除
PUT    /api/admin/categories/sort        批量排序
```

#### 技能映射（分类 ↔ ChatFlow）

```
POST   /api/admin/categories/{id}/skills        添加技能
PUT    /api/admin/categories/{id}/skills/{sid}   更新技能（描述 / 槽位定义）
DELETE /api/admin/categories/{id}/skills/{sid}   移除技能
```

#### 模型配置

```
POST   /api/admin/models                 创建
GET    /api/admin/models                 列表
PUT    /api/admin/models/{id}            更新
PUT    /api/admin/models/{id}/default    设为默认
```

#### 提示词模板

```
POST   /api/admin/prompts                创建
GET    /api/admin/prompts                列表
PUT    /api/admin/prompts/{id}           更新（自动递增版本号）
GET    /api/admin/prompts/{id}/versions  版本历史
POST   /api/admin/prompts/{id}/rollback  回滚到指定版本
```

#### 会话日志

```
GET    /api/admin/conversations                     列表（分页 + 筛选）
GET    /api/admin/conversations/{id}                详情（含消息 + 意图轨迹）
GET    /api/admin/conversations/{id}/intent-trace   意图轨迹
```

#### 统计分析

```
GET    /api/admin/analytics/overview     总览（会话数、消息数、活跃 ChatFlow）
GET    /api/admin/analytics/intents      意图分布（按分类 / ChatFlow / 时间）
GET    /api/admin/analytics/performance  性能指标（延迟、成功率、置信度分布）
```

---

## 5. LangGraph 意图引擎

### 5.1 State 定义

```python
from typing import TypedDict, Optional

class IntentState(TypedDict):
    # 输入
    user_input: str
    conversation_id: str
    chat_history: list[dict]           # [{role, content, source}]

    # 一级意图
    l1_category_id: Optional[int]
    l1_category_name: Optional[str]
    l1_confidence: Optional[float]

    # 二级意图
    available_skills: list[dict]       # 当前类别下的技能列表
    l2_chatflow_id: Optional[int]
    l2_skill_name: Optional[str]
    l2_confidence: Optional[float]

    # 槽位 & Query
    slot_definitions: list[dict]       # 该技能需要的槽位定义
    extracted_slots: dict              # 本轮提取的槽位
    accumulated_slots: dict            # 累积槽位（跨轮次）
    rewritten_query: Optional[str]     # 重组后的 query

    # ChatFlow 交互
    chatflow_endpoint: Optional[str]
    chatflow_response: Optional[str]

    # 流转控制
    current_phase: str                 # idle | classifying | in_chatflow | switching
    intent_switched: bool              # 是否检测到意图切换
    error: Optional[str]
```

### 5.2 图结构

```
                    ┌──────────┐
                    │  入口路由  │
                    └────┬─────┘
                         │
                ┌────────▼────────┐
          ┌─────┤  当前在ChatFlow中？├─────┐
          │ 否   └─────────────────┘  是  │
          │                               │
    ┌─────▼──────┐              ┌─────────▼─────────┐
    │ 一级意图分类 │              │   意图切换检测       │
    │ (粗粒度技能) │              │  (继续 or 重新分类)  │
    └─────┬──────┘              └─────────┬─────────┘
          │                          ┌────┴────┐
          │                       继续│         │切换
          │                          │         │
    ┌─────▼──────┐    ┌──────────────▼┐   回到一级
    │ 技能展开    │    │ 直接转发ChatFlow│   分类节点
    │ (加载二级)  │    └───────┬───────┘
    └─────┬──────┘            │
          │                   │
    ┌─────▼──────┐            │
    │ 二级意图选择 │            │
    │ (选ChatFlow)│            │
    └─────┬──────┘            │
          │                   │
    ┌─────▼──────┐            │
    │ 槽位提取 +  │            │
    │ Query 重组  │            │
    └─────┬──────┘            │
          │                   │
    ┌─────▼───────────────────▼──┐
    │       ChatFlow 调用         │
    └─────────────┬──────────────┘
                  │
    ┌─────────────▼──────────────┐
    │    结果处理 + 状态更新       │
    └─────────────┬──────────────┘
                  │
                输出
```

### 5.3 核心节点逻辑

#### 入口路由

```python
def route_entry(state: IntentState) -> str:
    """判断当前是否在某个 ChatFlow 会话中"""
    if state["current_phase"] == "in_chatflow":
        return "intent_switch_check"
    return "classify_l1"
```

#### 意图切换检测

```python
def intent_switch_check(state: IntentState) -> IntentState:
    """判断用户是继续当前 ChatFlow 流程，还是想做别的事
    
    Prompt:
      当前正在 [{current_intent}] 流程中。
      用户说: "{user_input}"
      判断用户是在继续当前流程，还是想做别的事？
      返回: {"continue": true/false, "reason": "..."}
    
    continue=True  → 路由到 call_chatflow（直接转发）
    continue=False → 路由到 classify_l1（重新走分类）
    """
```

#### 一级意图分类

```python
def classify_l1(state: IntentState) -> IntentState:
    """分层披露第一步: 粗粒度分类
    
    从 DB 加载所有 active 的 skill_category，
    构造 LangChain Tool 列表（每个 category 是一个 Tool）:
      tools = [
        Tool(name="金融交易服务", description="处理转账、缴费、兑换等金融交易"),
        Tool(name="账户管理服务", description="查询余额、修改账户设置等"),
        ...
      ]
    LLM function calling → 选择一个 category
    记录 l1_category_id, l1_confidence
    """
```

#### 技能展开

```python
def expand_skills(state: IntentState) -> IntentState:
    """根据一级分类，从 DB 加载该 category 下的 skill_chatflow_mapping
    
    将具体技能列表写入 state["available_skills"]
    同时加载对应的 slot_definitions
    """
```

#### 二级意图选择

```python
def classify_l2(state: IntentState) -> IntentState:
    """分层披露第二步: 从展开的技能中选择具体 ChatFlow
    
    tools = [
        Tool(name="转账", description="用户发起转账请求"),
        Tool(name="缴费", description="水电煤气等缴费"),
        ...
    ]
    LLM function calling → 选择具体 ChatFlow
    记录 l2_chatflow_id, l2_confidence
    """
```

#### 槽位提取 + Query 重组

```python
def extract_and_rewrite(state: IntentState) -> IntentState:
    """从对话上下文中提取所需槽位，重组 query
    
    Prompt:
      用户原始输入: "{user_input}"
      历史对话: {chat_history}
      需要提取的信息: {slot_definitions}
      
      1. 从上下文中提取以下字段
      2. 将提取结果和用户意图重组为一句清晰的 query
      
    输出:
      slots: {"payee": "张三", "amount": 5000}
      query: "用户要从工资卡转账5000元给张三，请协助完成转账流程"
    
    合并 extracted_slots 到 accumulated_slots
    """
```

#### ChatFlow 调用

```python
async def call_chatflow(state: IntentState) -> IntentState:
    """调用注册的 ChatFlow endpoint
    
    根据 protocol (sse/http_json) 选择调用方式:
    - SSE: 建立流式连接，逐 chunk 通过 FastAPI SSE 透传给用户
    - HTTP JSON: 发送请求，等待完整响应
    
    请求体:
      {
        "query": state["rewritten_query"],
        "conversation_id": state["conversation_id"],
        "slots": state["accumulated_slots"],
        "metadata": { "source": "intent_hub" }
      }
    """
```

#### 结果处理

```python
def process_result(state: IntentState) -> IntentState:
    """更新状态:
    - 合并本轮槽位到 accumulated_slots
    - 将 ChatFlow 响应摘要写入 chat_history
    - 记录 intent_trace（写 DB）
    - 设置 current_phase = "in_chatflow"
    - 更新 conversation 表的 current_intent, current_chatflow_id, slot_state
    """
```

### 5.4 图组装

```python
from langgraph.graph import StateGraph, END
from langgraph.checkpoint.postgres import PostgresSaver

graph = StateGraph(IntentState)

# 添加节点
graph.add_node("intent_switch_check", intent_switch_check)
graph.add_node("classify_l1", classify_l1)
graph.add_node("expand_skills", expand_skills)
graph.add_node("classify_l2", classify_l2)
graph.add_node("extract_and_rewrite", extract_and_rewrite)
graph.add_node("call_chatflow", call_chatflow)
graph.add_node("process_result", process_result)

# 入口路由
graph.set_conditional_entry_point(
    route_entry,
    {
        "classify_l1": "classify_l1",
        "intent_switch_check": "intent_switch_check",
    }
)

# 意图切换检测 → 继续当前 or 重新分类
graph.add_conditional_edges(
    "intent_switch_check",
    lambda s: "call_chatflow" if not s["intent_switched"] else "classify_l1",
)

# 主流程
graph.add_edge("classify_l1", "expand_skills")
graph.add_edge("expand_skills", "classify_l2")
graph.add_edge("classify_l2", "extract_and_rewrite")
graph.add_edge("extract_and_rewrite", "call_chatflow")
graph.add_edge("call_chatflow", "process_result")
graph.add_edge("process_result", END)

# 编译（带持久化）
checkpointer = PostgresSaver.from_conn_string("postgresql://...")
app = graph.compile(checkpointer=checkpointer)
```

### 5.5 完整请求链路示例

```
用户: "我想转5000给张三"

1. route_entry         → current_phase=idle → 走 classify_l1
2. classify_l1         → Tools: [金融交易, 账户管理, 理财咨询, 通用]
                         LLM 选择: 金融交易服务 (confidence: 0.96)
                         SSE → event:intent {level:"l1", category:"金融交易"}

3. expand_skills       → 加载: [转账, 缴费, 兑换]

4. classify_l2         → Tools: [转账, 缴费, 兑换]
                         LLM 选择: 转账 (confidence: 0.93)
                         SSE → event:intent {level:"l2", skill:"转账"}

5. extract_and_rewrite → 提取: {payee:"张三", amount:5000, from:null}
                         重组: "用户要转账5000元给张三，付款账户待确认"
                         SSE → event:slot {payee:"张三", amount:5000}

6. call_chatflow       → POST ChatFlow:转账 endpoint
                         SSE → event:message_delta 透传 ChatFlow 流式响应
                         ChatFlow 回复: "请问从哪个账户转出？"

7. process_result      → 更新 conversation: current_phase=in_chatflow
                         记录 intent_trace
                         SSE → event:message_end

---

用户: "工资卡"  (第二轮)

1. route_entry         → current_phase=in_chatflow → 走 intent_switch_check
2. intent_switch_check → "工资卡" 是对转账流程的回答 → continue=True
3. call_chatflow       → 直接转发 "工资卡" 给转账 ChatFlow
                         ChatFlow 回复: "确认从工资卡转5000元给张三？"
4. process_result      → 更新槽位: {from:"工资卡"}

---

用户: "算了不转了，帮我查下余额"  (意图切换)

1. route_entry         → current_phase=in_chatflow → 走 intent_switch_check
2. intent_switch_check → 检测到意图切换 → intent_switched=True
3. classify_l1         → LLM 选择: 账户管理服务
4. expand_skills       → 加载: [余额查询, 账户设置]
5. classify_l2         → LLM 选择: 余额查询
6. extract_and_rewrite → 提取: {} (无特殊槽位)
                         重组: "用户要查询账户余额"
7. call_chatflow       → POST ChatFlow:查询 endpoint
8. process_result      → 更新: current_intent=账户查询, 清空旧槽位
```

---

## 6. 管理后台页面规划

共 8 个核心页面：

### 6.1 Dashboard 概览页

- 今日核心指标卡片：会话数、消息数、活跃智能体数、平均延迟、意图准确率
- 意图分布饼图（按一级分类）
- 最近 24h 会话趋势折线图
- 最近告警列表（ChatFlow 超时率上升等）
- 最近注册/上线的 ChatFlow

### 6.2 智能体管理页

- 列表：名称、状态（active/inactive/testing）、Endpoint、协议、今日调用量、操作
- 操作：编辑、连通性测试、启用/禁用
- 注册/编辑弹窗：名称、描述（供 LLM 使用）、Endpoint、认证方式、Token、协议选择、超时配置
- 连通性测试按钮

### 6.3 技能分类页

- 树形列表，支持拖拽排序
- 一级分类：名称、描述、图标、关联技能数
- 二级技能：技能名 → 关联的 ChatFlow、技能描述
- 槽位定义编辑器：表格形式编辑字段名、类型、描述、是否必填

### 6.4 模型配置页

- 列表：配置名、Provider、模型名、是否默认
- 编辑：Provider、模型名、API Endpoint、API Key、Temperature、Max Tokens
- 设为默认按钮

### 6.5 提示词管理页

- 按类型筛选：intent_l1 / intent_l2 / query_rewrite / slot_extract
- 编辑器：左侧 Prompt 内容（支持 `{{变量}}`），右侧可用变量列表
- 预览渲染效果
- 版本历史列表，支持回滚到任意版本

### 6.6 会话日志页

- 列表：会话 ID、用户、消息数、意图轨迹摘要、状态、创建时间
- 筛选：时间范围、状态、智能体
- 详情页：左侧对话流（标注消息来源：用户/中台/ChatFlow），右侧意图轨迹时间线

### 6.7 数据统计页

- 意图分布：按一级分类、按 ChatFlow、按时间段
- 性能指标：各环节延迟分布、成功率、置信度分布
- 趋势分析：日/周/月维度

### 6.8 调试沙盒页

- 左侧：对话窗口（可输入测试消息）
- 右侧：实时调试面板，展示每一步的决策过程
  - 一级意图识别结果 + 置信度
  - 展开的技能列表
  - 二级意图选择结果 + 置信度
  - 提取的槽位
  - 重组的 Query
  - ChatFlow 调用响应状态
  - 各环节延迟
- 底部：清空对话、导出调试日志

---

## 7. 技术选型

| 组件 | 技术 | 版本 |
|------|------|------|
| Web 框架 | FastAPI | ≥0.110 |
| AI 编排 | LangGraph + LangChain | langgraph≥0.2, langchain≥0.3 |
| 数据库 | MySQL 8.x | |
| 缓存 | Redis 7.x | |
| SSE | sse-starlette | |
| ORM | SQLAlchemy 2.x | |
| 前端 | Vue3 + Element Plus 或 React + Ant Design | |
| 部署 | Docker + Docker Compose | |
| Python | ≥3.11 | |

---

## 8. 项目结构

```
intent-hub/
├── backend/
│   ├── main.py                      # FastAPI 入口
│   ├── api/
│   │   ├── v1/
│   │   │   ├── conversation.py      # 对外会话 API
│   │   │   └── health.py            # 健康检查
│   │   ├── admin/
│   │   │   ├── chatflow.py          # ChatFlow CRUD
│   │   │   ├── category.py          # 技能分类 CRUD
│   │   │   ├── model_config.py      # 模型配置
│   │   │   ├── prompt.py            # 提示词管理
│   │   │   ├── conversation_log.py  # 会话日志
│   │   │   └── analytics.py         # 统计分析
│   │   └── middleware/
│   │       ├── auth.py              # 认证
│   │       └── logging.py           # 请求日志
│   ├── engine/
│   │   ├── graph.py                 # LangGraph 图定义和组装
│   │   ├── state.py                 # IntentState 定义
│   │   └── nodes/
│   │       ├── classify_l1.py       # 一级意图分类
│   │       ├── expand_skills.py     # 技能展开
│   │       ├── classify_l2.py       # 二级意图选择
│   │       ├── extract_rewrite.py   # 槽位提取 + Query 重组
│   │       ├── call_chatflow.py     # ChatFlow 调用
│   │       ├── intent_switch.py     # 意图切换检测
│   │       └── process_result.py    # 结果处理
│   ├── models/
│   │   ├── chatflow_agent.py        # ORM 模型
│   │   ├── skill_category.py
│   │   ├── skill_mapping.py
│   │   ├── model_config.py
│   │   ├── prompt_template.py
│   │   ├── conversation.py
│   │   ├── message.py
│   │   └── intent_trace.py
│   ├── services/
│   │   ├── chatflow_service.py      # ChatFlow 业务逻辑
│   │   ├── category_service.py
│   │   ├── model_service.py
│   │   ├── prompt_service.py
│   │   └── analytics_service.py
│   ├── core/
│   │   ├── config.py                # 配置管理
│   │   ├── database.py              # DB 连接
│   │   ├── redis.py                 # Redis 连接
│   │   └── security.py              # 加密 / 认证工具
│   ├── alembic/                     # 数据库迁移
│   ├── requirements.txt
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── views/
│   │   │   ├── Dashboard.vue
│   │   │   ├── ChatflowManage.vue
│   │   │   ├── SkillCategory.vue
│   │   │   ├── ModelConfig.vue
│   │   │   ├── PromptManage.vue
│   │   │   ├── ConversationLog.vue
│   │   │   ├── Analytics.vue
│   │   │   └── DebugSandbox.vue
│   │   ├── api/                     # API 请求封装
│   │   ├── components/              # 通用组件
│   │   ├── router/
│   │   └── stores/
│   ├── package.json
│   └── Dockerfile
├── docker-compose.yml
└── docs/
    └── api.md
```

---

## 9. 部署架构

```
docker-compose.yml:
  ├── intent-hub-api     (FastAPI 后端, port 8000)
  ├── intent-hub-web     (前端静态, port 3000, nginx)
  ├── mysql              (数据库, port 3306)
  └── redis              (缓存, port 6379)
```

生产环境可复用现有的 MySQL 和 Redis 实例，仅部署 api + web 两个容器。
