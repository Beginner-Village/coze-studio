# 外部智能体接入完整方案

> 基于 HiAgent（火山引擎智能体）接入实践总结的通用接入方案
>
> **目标**：提供一套标准化流程，使团队能够快速接入其他外部智能体（如百度文心智能体、阿里通义智能体等），避免重复试错。

---

## 📋 目录

1. [架构设计](#1-架构设计)
2. [后端实现](#2-后端实现)
3. [前端实现](#3-前端实现)
4. [测试验证](#4-测试验证)
5. [常见问题与解决方案](#5-常见问题与解决方案)
6. [接入检查清单](#6-接入检查清单)

---

## 1. 架构设计

### 1.1 核心概念

外部智能体接入需要解决以下核心问题：

| 问题 | 解决方案 |
|------|---------|
| **多轮对话上下文保持** | 使用外部智能体的 conversation_id 机制 |
| **会话状态持久化** | 存储到数据库 `conversation.ext` JSON 字段 |
| **会话生命周期管理** | 根据 section_id 判断是否需要重置会话 |
| **线程安全** | 使用 sync.RWMutex 保护共享状态 |
| **前后端类型一致性** | 通过 Thrift IDL 定义统一接口 |

### 1.2 数据流图

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端 UI 层                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ModelSelect 组件                                         │  │
│  │  - 获取模型列表 (包含外部智能体)                           │  │
│  │  - 渲染分组选择器                                          │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓ API 调用
┌─────────────────────────────────────────────────────────────────┐
│                      后端 API 层                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  ModelMgr Service (modelmgr.thrift)                      │  │
│  │  - ListModels: 返回所有可用模型                           │  │
│  │  - 包含外部智能体的元数据 (AgentInfo)                      │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                   Workflow 执行层                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  LLM Node                                                 │  │
│  │  - 检测外部智能体类型                                      │  │
│  │  - 调用对应的 ExternalAgentChatModel                       │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                外部智能体适配层                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  HiAgentChatModel (backend/domain/ynet_agent/)           │  │
│  │  - 实现 schema.ChatModel 接口                             │  │
│  │  - 管理外部智能体会话生命周期                              │  │
│  │  - 会话状态持久化到数据库                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓ HTTP/gRPC
┌─────────────────────────────────────────────────────────────────┐
│                   外部智能体 API                                  │
│  (火山引擎 HiAgent / 百度文心 / 阿里通义 / ...)                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.3 会话状态管理设计

```go
// 会话状态存储结构
type ExecuteConfig struct {
    // ...其他字段

    // 外部智能体会话映射: map[agentID]*ConversationInfo
    ExternalAgentConversations map[string]*ExternalAgentConversationInfo
    externalAgentConversationsMu sync.RWMutex
}

// 通用的外部智能体会话信息
type ExternalAgentConversationInfo struct {
    // 外部智能体的会话 ID
    ExternalConversationID string `json:"external_conversation_id"`

    // 关联的 section_id (用于判断会话边界)
    LastSectionID int64 `json:"last_section_id"`

    // 可选：外部智能体特定的元数据
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

**数据库持久化格式**：

```json
{
  "conversation": {
    "ext": {
      "external_agent_conversations": {
        "hiagent_d1j2ks8dhuh30bfis9g0": {
          "external_conversation_id": "d40n6mh926cock3q4r10",
          "last_section_id": 7566455633650663424,
          "metadata": {
            "provider": "volcengine_hiagent",
            "agent_name": "客服助手"
          }
        },
        "wenxin_agent_xxx": {
          "external_conversation_id": "baidu_conv_12345",
          "last_section_id": 7566455633650663424,
          "metadata": {
            "provider": "baidu_wenxin",
            "agent_name": "文心智能体"
          }
        }
      }
    }
  }
}
```

---

## 2. 后端实现

### 2.1 Thrift IDL 定义

#### 2.1.1 模型管理 IDL (modelmgr.thrift)

```thrift
namespace go modelmgr

// 外部智能体类型枚举
enum ExternalAgentType {
    VOLCENGINE_HIAGENT = 1,  // 火山引擎 HiAgent
    BAIDU_WENXIN = 2,        // 百度文心智能体
    ALI_TONGYI = 3,          // 阿里通义智能体
    CUSTOM = 99,             // 自定义外部智能体
}

// 外部智能体信息
struct AgentInfo {
    1: required string agent_id,                    // 智能体ID
    2: required string agent_name,                  // 智能体名称
    3: required ExternalAgentType agent_type,       // 智能体类型
    4: optional string description,                 // 描述
    5: optional map<string, string> config,         // 配置参数
}

// 模型信息
struct ModelInfo {
    1: required string model_id,
    2: required string model_name,
    3: required string provider,
    4: optional AgentInfo agent_info,  // 👈 关键：外部智能体信息
    // ...其他字段
}

// 模型列表响应
struct ListModelsResponse {
    1: required list<ModelInfo> models,
    253: required i32 code,
    254: required string msg,
}
```

#### 2.1.2 工作流执行配置 (workflow.thrift)

```thrift
namespace go workflow

// 外部智能体会话信息
struct ExternalAgentConversationInfo {
    1: required string external_conversation_id,    // 外部会话ID
    2: required i64 last_section_id,                // 最后的section_id
    3: optional map<string, string> metadata,       // 元数据
}

// 执行配置
struct ExecuteConfig {
    // ...其他字段

    // 外部智能体会话映射: map[agentID]ConversationInfo
    50: optional map<string, ExternalAgentConversationInfo> external_agent_conversations,
}
```

### 2.2 外部智能体适配层实现

#### 2.2.1 通用接口定义

```go
// backend/domain/external_agent/interface.go

package external_agent

import (
    "context"
    "github.com/cloudwego/eino/schema"
)

// ExternalAgentChatModel 外部智能体统一接口
type ExternalAgentChatModel interface {
    schema.ChatModel  // 继承 Eino 的标准 ChatModel 接口

    // GetAgentID 获取智能体ID
    GetAgentID() string

    // GetAgentType 获取智能体类型
    GetAgentType() string

    // EnsureConversation 确保会话存在，返回外部会话ID
    EnsureConversation(ctx context.Context) (string, error)

    // ClearConversation 清除当前会话
    ClearConversation(ctx context.Context) error
}

// ExternalAgentConfig 外部智能体配置
type ExternalAgentConfig struct {
    AgentID       string
    AgentName     string
    AgentType     string
    APIKey        string
    APIEndpoint   string
    Metadata      map[string]string
}
```

#### 2.2.2 HiAgent 实现示例

```go
// backend/domain/ynet_agent/hiagent_model.go

package ynet_agent

import (
    "context"
    "fmt"
    "sync"

    "github.com/cloudwego/eino/schema"
    workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
)

type HiAgentChatModel struct {
    agent          *HiAgent
    client         *http.Client
    conversationMu sync.RWMutex
}

// ============ 核心方法 1: 会话管理 ============

// ensureConversation 确保会话存在
func (h *HiAgentChatModel) ensureConversation(ctx context.Context) (string, error) {
    // 1. 从 ExecuteConfig 获取现有会话信息
    exeCfg := GetExecuteConfigFromContext(ctx)
    existingInfo := exeCfg.GetExternalAgentConversationInfo(h.agent.AgentID)

    // 2. 判断是否可以复用现有会话
    canReuse := false
    if existingInfo != nil && existingInfo.ExternalConversationID != "" {
        if exeCfg.SectionID == nil {
            // 无 section 概念，直接复用
            canReuse = true
        } else if existingInfo.LastSectionID == *exeCfg.SectionID {
            // section 未变化，复用
            canReuse = true
        } else {
            // section 变化了，需要清除旧会话
            logs.CtxInfof(ctx, "section changed (old: %d, new: %d), clearing old conversation",
                existingInfo.LastSectionID, *exeCfg.SectionID)
        }
    }

    if canReuse {
        logs.CtxInfof(ctx, "reusing external agent conversation: %s (section_id: %d)",
            existingInfo.ExternalConversationID, existingInfo.LastSectionID)
        return existingInfo.ExternalConversationID, nil
    }

    // 3. 创建新会话
    externalConvID, err := h.createNewConversation(ctx)
    if err != nil {
        return "", err
    }

    // 4. 保存到内存和数据库
    sectionID := int64(0)
    if exeCfg.SectionID != nil {
        sectionID = *exeCfg.SectionID
    }

    // 保存到 ExecuteConfig 内存
    exeCfg.SetExternalAgentConversationInfo(h.agent.AgentID, &workflowModel.ExternalAgentConversationInfo{
        ExternalConversationID: externalConvID,
        LastSectionID:          sectionID,
        Metadata: map[string]string{
            "provider": "volcengine_hiagent",
            "agent_name": h.agent.Name,
        },
    })

    // 异步保存到数据库
    go func() {
        if err := h.saveConversationToDatabase(ctx, externalConvID, sectionID); err != nil {
            logs.CtxErrorf(ctx, "failed to save external agent conversation to DB: %v", err)
        } else {
            logs.CtxInfof(ctx, "✅ successfully saved external agent conversation to DB")
        }
    }()

    return externalConvID, nil
}

// ============ 核心方法 2: 数据库持久化 ============

// saveConversationToDatabase 保存会话到数据库
func (h *HiAgentChatModel) saveConversationToDatabase(ctx context.Context, externalConvID string, sectionID int64) error {
    exeCfg := GetExecuteConfigFromContext(ctx)
    if exeCfg.ConversationID == nil {
        return fmt.Errorf("conversation_id is nil")
    }

    conversationID := *exeCfg.ConversationID

    // 1. 获取当前 conversation 记录
    conv, err := conversation.DefaultSVC().GetByID(ctx, conversationID)
    if err != nil {
        return fmt.Errorf("failed to get conversation: %w", err)
    }

    // 2. 解析 ext 字段
    ext := make(map[string]interface{})
    if conv.Ext != "" {
        if err := sonic.UnmarshalString(conv.Ext, &ext); err != nil {
            return fmt.Errorf("failed to unmarshal ext: %w", err)
        }
    }

    // 3. 更新 external_agent_conversations 部分
    var externalAgentConvs map[string]interface{}
    if existing, ok := ext["external_agent_conversations"].(map[string]interface{}); ok {
        externalAgentConvs = existing
    } else {
        externalAgentConvs = make(map[string]interface{})
    }

    // 使用标准结构
    externalAgentConvs[h.agent.AgentID] = map[string]interface{}{
        "external_conversation_id": externalConvID,
        "last_section_id":          sectionID,
        "metadata": map[string]string{
            "provider":   "volcengine_hiagent",
            "agent_name": h.agent.Name,
        },
    }
    ext["external_agent_conversations"] = externalAgentConvs

    // 4. 序列化并保存
    extStr, err := sonic.MarshalString(ext)
    if err != nil {
        return fmt.Errorf("failed to marshal ext: %w", err)
    }

    return conversation.DefaultSVC().UpdateExt(ctx, conversationID, extStr)
}

// ============ 核心方法 3: 实现 ChatModel 接口 ============

// Generate 同步生成
func (h *HiAgentChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...schema.ChatModelOption) (*schema.Message, error) {
    // 1. 确保会话存在
    externalConvID, err := h.ensureConversation(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to ensure conversation: %w", err)
    }

    // 2. 调用外部智能体 API
    response, err := h.callExternalAgentAPI(ctx, externalConvID, input, opts...)
    if err != nil {
        return nil, err
    }

    return response, nil
}

// Stream 流式生成
func (h *HiAgentChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...schema.ChatModelOption) (*schema.StreamReader[*schema.Message], error) {
    // 1. 确保会话存在
    externalConvID, err := h.ensureConversation(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to ensure conversation: %w", err)
    }

    // 2. 调用外部智能体流式 API
    return h.streamExternalAgentAPI(ctx, externalConvID, input, opts...)
}

// ============ 核心方法 4: 外部 API 调用 ============

// callExternalAgentAPI 调用外部智能体 API (同步)
func (h *HiAgentChatModel) callExternalAgentAPI(ctx context.Context, convID string, messages []*schema.Message, opts ...schema.ChatModelOption) (*schema.Message, error) {
    // 构造请求
    req := &HiAgentChatRequest{
        AppConversationID: convID,
        Messages:          convertMessages(messages),
        // ...其他参数
    }

    // 发送 HTTP 请求
    resp, err := h.client.Post(h.agent.APIEndpoint, req)
    if err != nil {
        return nil, err
    }

    // 解析响应
    return parseHiAgentResponse(resp)
}

// streamExternalAgentAPI 调用外部智能体 API (流式)
func (h *HiAgentChatModel) streamExternalAgentAPI(ctx context.Context, convID string, messages []*schema.Message, opts ...schema.ChatModelOption) (*schema.StreamReader[*schema.Message], error) {
    // 类似实现，使用 SSE 或 WebSocket
    // ...
}
```

#### 2.2.3 数据库加载逻辑

```go
// backend/domain/workflow/service/executable_impl.go

func loadExternalAgentConversationsFromDB(ctx context.Context, conversationID int64, config *workflowModel.ExecuteConfig) {
    // 1. 从数据库加载 conversation
    conv, err := conversation.DefaultSVC().GetByID(ctx, conversationID)
    if err != nil {
        logs.CtxWarnf(ctx, "failed to load conversation: %v", err)
        return
    }

    // 2. 解析 ext 字段
    if conv.Ext == "" {
        return
    }

    var ext map[string]interface{}
    if err := sonic.UnmarshalString(conv.Ext, &ext); err != nil {
        logs.CtxWarnf(ctx, "failed to unmarshal ext: %v", err)
        return
    }

    // 3. 提取 external_agent_conversations
    externalAgentConvsRaw, ok := ext["external_agent_conversations"].(map[string]interface{})
    if !ok {
        return
    }

    // 4. 遍历每个外部智能体会话
    for agentID, convData := range externalAgentConvsRaw {
        convMap, ok := convData.(map[string]interface{})
        if !ok {
            continue
        }

        // 5. 解析会话信息
        info := &workflowModel.ExternalAgentConversationInfo{}

        // 解析 external_conversation_id
        if externalConvID, ok := convMap["external_conversation_id"].(string); ok {
            info.ExternalConversationID = externalConvID
        }

        // 解析 last_section_id (支持 float64 和 int64)
        if lastSectionID, ok := convMap["last_section_id"].(float64); ok {
            info.LastSectionID = int64(lastSectionID)
        } else if lastSectionID, ok := convMap["last_section_id"].(int64); ok {
            info.LastSectionID = lastSectionID
        }

        // 解析 metadata
        if metadata, ok := convMap["metadata"].(map[string]interface{}); ok {
            info.Metadata = make(map[string]string)
            for k, v := range metadata {
                if strVal, ok := v.(string); ok {
                    info.Metadata[k] = strVal
                }
            }
        }

        // 6. 保存到 ExecuteConfig
        config.ExternalAgentConversations[agentID] = info

        logs.CtxInfof(ctx, "✅ loaded external agent conversation: agent=%s, conv_id=%s, section_id=%d",
            agentID, info.ExternalConversationID, info.LastSectionID)
    }
}
```

### 2.3 LLM Node 集成

```go
// backend/domain/workflow/internal/nodes/llm/llm.go

func (l *llmNode) Generate(ctx context.Context, input map[string]any, opts ...graph.GenerateOption) (map[string]any, error) {
    // 1. 检测是否为外部智能体
    if l.isExternalAgent() {
        return l.generateWithExternalAgent(ctx, input, opts...)
    }

    // 2. 普通模型逻辑
    return l.generateWithNormalModel(ctx, input, opts...)
}

// isExternalAgent 检测是否为外部智能体
func (l *llmNode) isExternalAgent() bool {
    // 检查模型配置中的 agent_info 字段
    return l.config.Model.AgentInfo != nil
}

// generateWithExternalAgent 使用外部智能体生成
func (l *llmNode) generateWithExternalAgent(ctx context.Context, input map[string]any, opts ...graph.GenerateOption) (map[string]any, error) {
    agentInfo := l.config.Model.AgentInfo

    // 1. 根据智能体类型创建对应的 ChatModel
    var chatModel external_agent.ExternalAgentChatModel
    var err error

    switch agentInfo.AgentType {
    case modelmgr.ExternalAgentType_VOLCENGINE_HIAGENT:
        chatModel, err = ynet_agent.NewHiAgentChatModel(ctx, agentInfo)
    case modelmgr.ExternalAgentType_BAIDU_WENXIN:
        chatModel, err = wenxin_agent.NewWenxinChatModel(ctx, agentInfo)
    case modelmgr.ExternalAgentType_ALI_TONGYI:
        chatModel, err = tongyi_agent.NewTongyiChatModel(ctx, agentInfo)
    default:
        return nil, fmt.Errorf("unsupported external agent type: %v", agentInfo.AgentType)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to create external agent chat model: %w", err)
    }

    // 2. 调用外部智能体
    messages := l.buildMessages(input)

    if l.config.Stream {
        return l.streamGenerate(ctx, chatModel, messages, opts...)
    } else {
        return l.syncGenerate(ctx, chatModel, messages, opts...)
    }
}
```

### 2.4 模型管理接口实现

```go
// backend/crossdomain/impl/modelmgr/modelmgr.go

func (m *ModelMgrService) ListModels(ctx context.Context, req *modelmgr.ListModelsRequest) (*modelmgr.ListModelsResponse, error) {
    // 1. 从数据库加载普通模型
    normalModels, err := m.modelRepo.List(ctx, req)
    if err != nil {
        return nil, err
    }

    // 2. 从配置文件加载外部智能体配置
    externalAgents, err := m.loadExternalAgentsFromConfig(ctx, req.SpaceID)
    if err != nil {
        logs.CtxWarnf(ctx, "failed to load external agents: %v", err)
    }

    // 3. 合并模型列表
    allModels := make([]*modelmgr.ModelInfo, 0, len(normalModels)+len(externalAgents))
    allModels = append(allModels, normalModels...)

    for _, agent := range externalAgents {
        allModels = append(allModels, &modelmgr.ModelInfo{
            ModelID:   agent.AgentID,
            ModelName: agent.AgentName,
            Provider:  "external_agent",
            AgentInfo: agent,  // 👈 关键：包含外部智能体信息
        })
    }

    return &modelmgr.ListModelsResponse{
        Models: allModels,
        Code:   0,
        Msg:    "success",
    }, nil
}

// loadExternalAgentsFromConfig 从配置文件加载外部智能体
func (m *ModelMgrService) loadExternalAgentsFromConfig(ctx context.Context, spaceID int64) ([]*modelmgr.AgentInfo, error) {
    configPath := fmt.Sprintf("backend/conf/external_agents/%d/agents.json", spaceID)

    data, err := os.ReadFile(configPath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil  // 没有配置文件，返回空列表
        }
        return nil, err
    }

    var agents []*modelmgr.AgentInfo
    if err := sonic.Unmarshal(data, &agents); err != nil {
        return nil, err
    }

    return agents, nil
}
```

### 2.5 配置文件格式

```json
// backend/conf/external_agents/{space_id}/agents.json

[
  {
    "agent_id": "hiagent_d1j2ks8dhuh30bfis9g0",
    "agent_name": "客服助手",
    "agent_type": 1,
    "description": "火山引擎HiAgent智能体",
    "config": {
      "api_key": "your_api_key_here",
      "api_endpoint": "https://api.volcengine.com/hiagent/v1/chat",
      "app_id": "d1j2ks8dhuh30bfis9g0",
      "timeout": "30s"
    }
  },
  {
    "agent_id": "wenxin_agent_xxx",
    "agent_name": "文心智能体",
    "agent_type": 2,
    "description": "百度文心智能体",
    "config": {
      "api_key": "your_baidu_api_key",
      "api_endpoint": "https://aip.baidubce.com/wenxin/v1/chat",
      "app_id": "wenxin_app_xxx"
    }
  }
]
```

---

## 3. 前端实现

### 3.1 TypeScript 类型定义

```typescript
// frontend/packages/arch/api-schema/src/idl/modelmgr/modelmgr.ts

export enum ExternalAgentType {
  VOLCENGINE_HIAGENT = 1,
  BAIDU_WENXIN = 2,
  ALI_TONGYI = 3,
  CUSTOM = 99,
}

export interface AgentInfo {
  agent_id: string;
  agent_name: string;
  agent_type: ExternalAgentType;
  description?: string;
  config?: Record<string, string>;
}

export interface ModelInfo {
  model_id: string;
  model_name: string;
  provider: string;
  agent_info?: AgentInfo;  // 👈 外部智能体信息
  // ...其他字段
}
```

### 3.2 ModelSelect 组件改造

```tsx
// frontend/packages/workflow/playground/src/components/model-select/index.tsx

import React, { useMemo } from 'react';
import { Select } from '@coze-arch/coze-design';
import { modelmgr } from '@coze-studio/api-schema';

interface ModelSelectProps {
  value?: string;
  onChange?: (modelId: string, modelInfo: modelmgr.ModelInfo) => void;
  spaceId: string;
}

const ModelSelect: React.FC<ModelSelectProps> = ({ value, onChange, spaceId }) => {
  // 1. 获取模型列表
  const { data: modelsData } = useModelList(spaceId);

  // 2. 对模型进行分组
  const groupedModels = useMemo(() => {
    if (!modelsData?.models) return {};

    const groups: Record<string, modelmgr.ModelInfo[]> = {
      'OpenAI': [],
      'Claude': [],
      '外部智能体': [],  // 👈 新增分组
      '其他': [],
    };

    modelsData.models.forEach((model) => {
      // 外部智能体单独分组
      if (model.agent_info) {
        groups['外部智能体'].push(model);
        return;
      }

      // 普通模型按 provider 分组
      if (model.provider.includes('openai')) {
        groups['OpenAI'].push(model);
      } else if (model.provider.includes('claude')) {
        groups['Claude'].push(model);
      } else {
        groups['其他'].push(model);
      }
    });

    return groups;
  }, [modelsData]);

  // 3. 渲染分组选择器
  return (
    <Select
      value={value}
      onChange={(modelId) => {
        const selectedModel = modelsData?.models.find(m => m.model_id === modelId);
        if (selectedModel) {
          onChange?.(modelId, selectedModel);
        }
      }}
      placeholder="请选择模型"
    >
      {Object.entries(groupedModels).map(([groupName, models]) => {
        if (models.length === 0) return null;

        return (
          <Select.OptGroup key={groupName} label={groupName}>
            {models.map((model) => (
              <Select.Option key={model.model_id} value={model.model_id}>
                <div className="flex items-center gap-2">
                  {/* 外部智能体显示特殊图标 */}
                  {model.agent_info && (
                    <span className="text-xs bg-blue-100 text-blue-600 px-2 py-1 rounded">
                      智能体
                    </span>
                  )}
                  <span>{model.model_name}</span>
                </div>
              </Select.Option>
            ))}
          </Select.OptGroup>
        );
      })}
    </Select>
  );
};
```

### 3.3 外部智能体配置 UI

```tsx
// frontend/packages/workflow/playground/src/components/external-agent-config/index.tsx

import React, { useState } from 'react';
import { Form, Input, Select, Button, message } from '@coze-arch/coze-design';
import { modelmgr } from '@coze-studio/api-schema';

interface ExternalAgentConfigProps {
  spaceId: string;
  onSuccess?: () => void;
}

const ExternalAgentConfig: React.FC<ExternalAgentConfigProps> = ({ spaceId, onSuccess }) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      const agentInfo: modelmgr.AgentInfo = {
        agent_id: values.agent_id,
        agent_name: values.agent_name,
        agent_type: values.agent_type,
        description: values.description,
        config: {
          api_key: values.api_key,
          api_endpoint: values.api_endpoint,
          app_id: values.app_id,
        },
      };

      // 调用后端 API 保存配置
      await modelmgr.CreateExternalAgent({
        space_id: spaceId,
        agent_info: agentInfo,
      });

      message.success('外部智能体添加成功');
      onSuccess?.();
    } catch (error) {
      message.error('添加失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form form={form} onFinish={handleSubmit} layout="vertical">
      <Form.Item
        label="智能体类型"
        name="agent_type"
        rules={[{ required: true, message: '请选择智能体类型' }]}
      >
        <Select placeholder="请选择">
          <Select.Option value={modelmgr.ExternalAgentType.VOLCENGINE_HIAGENT}>
            火山引擎 HiAgent
          </Select.Option>
          <Select.Option value={modelmgr.ExternalAgentType.BAIDU_WENXIN}>
            百度文心智能体
          </Select.Option>
          <Select.Option value={modelmgr.ExternalAgentType.ALI_TONGYI}>
            阿里通义智能体
          </Select.Option>
        </Select>
      </Form.Item>

      <Form.Item
        label="智能体ID"
        name="agent_id"
        rules={[{ required: true, message: '请输入智能体ID' }]}
      >
        <Input placeholder="如: hiagent_xxx" />
      </Form.Item>

      <Form.Item
        label="智能体名称"
        name="agent_name"
        rules={[{ required: true, message: '请输入智能体名称' }]}
      >
        <Input placeholder="如: 客服助手" />
      </Form.Item>

      <Form.Item label="描述" name="description">
        <Input.TextArea rows={3} placeholder="智能体功能描述" />
      </Form.Item>

      <Form.Item
        label="API Key"
        name="api_key"
        rules={[{ required: true, message: '请输入API Key' }]}
      >
        <Input.Password placeholder="输入API密钥" />
      </Form.Item>

      <Form.Item
        label="API Endpoint"
        name="api_endpoint"
        rules={[{ required: true, message: '请输入API Endpoint' }]}
      >
        <Input placeholder="https://api.example.com/v1/chat" />
      </Form.Item>

      <Form.Item
        label="App ID"
        name="app_id"
        rules={[{ required: true, message: '请输入App ID' }]}
      >
        <Input placeholder="外部智能体的应用ID" />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit" loading={loading}>
          添加智能体
        </Button>
      </Form.Item>
    </Form>
  );
};

export default ExternalAgentConfig;
```

### 3.4 LLM Node 表单集成

```tsx
// frontend/packages/workflow/playground/src/nodes-v2/llm/llm-form-meta.tsx

import { FormMeta } from '@coze-arch/coze-design';
import ModelSelect from '../../components/model-select';

export const llmFormMeta: FormMeta = {
  fields: [
    {
      name: 'model',
      label: '模型',
      component: ModelSelect,
      required: true,
      // 当模型改变时，检查是否为外部智能体
      onChange: (value, modelInfo, form) => {
        if (modelInfo?.agent_info) {
          // 外部智能体：隐藏某些不适用的配置项
          form.setFieldValue('temperature', undefined);
          form.setFieldValue('max_tokens', undefined);

          // 显示外部智能体特定提示
          message.info('已选择外部智能体，部分高级参数不可用');
        }
      },
    },
    // ...其他字段
  ],
};
```

---

## 4. 测试验证

### 4.1 单元测试

#### 4.1.1 后端单元测试

```go
// backend/domain/ynet_agent/hiagent_model_test.go

package ynet_agent

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
)

func TestHiAgentConversationReuse(t *testing.T) {
    // 测试场景：相同 section_id 下复用会话

    ctx := context.Background()

    // 1. 创建 ExecuteConfig
    sectionID := int64(123456)
    exeCfg := &workflowModel.ExecuteConfig{
        SectionID: &sectionID,
        ExternalAgentConversations: make(map[string]*workflowModel.ExternalAgentConversationInfo),
    }
    ctx = context.WithValue(ctx, "execute_config", exeCfg)

    // 2. 创建 HiAgent 模型
    agent := &HiAgent{AgentID: "test_agent_1"}
    model := &HiAgentChatModel{agent: agent}

    // 3. 第一次调用：创建新会话
    convID1, err := model.ensureConversation(ctx)
    assert.NoError(t, err)
    assert.NotEmpty(t, convID1)

    // 4. 第二次调用：应该复用会话
    convID2, err := model.ensureConversation(ctx)
    assert.NoError(t, err)
    assert.Equal(t, convID1, convID2, "应该复用同一个会话")
}

func TestHiAgentConversationReset(t *testing.T) {
    // 测试场景：section_id 变化时重置会话

    ctx := context.Background()

    // 1. 第一个 section
    sectionID1 := int64(123456)
    exeCfg := &workflowModel.ExecuteConfig{
        SectionID: &sectionID1,
        ExternalAgentConversations: make(map[string]*workflowModel.ExternalAgentConversationInfo),
    }
    ctx = context.WithValue(ctx, "execute_config", exeCfg)

    agent := &HiAgent{AgentID: "test_agent_2"}
    model := &HiAgentChatModel{agent: agent}

    // 2. 第一次调用
    convID1, err := model.ensureConversation(ctx)
    assert.NoError(t, err)

    // 3. 切换到新的 section
    sectionID2 := int64(789012)
    exeCfg.SectionID = &sectionID2

    // 4. 第二次调用：应该创建新会话
    convID2, err := model.ensureConversation(ctx)
    assert.NoError(t, err)
    assert.NotEqual(t, convID1, convID2, "section 变化后应该创建新会话")
}

func TestDatabaseSaveAndLoad(t *testing.T) {
    // 测试场景：数据库保存和加载

    // Mock conversation service
    // ...

    ctx := context.Background()
    conversationID := int64(1001)
    agentID := "test_agent_3"
    externalConvID := "external_conv_123"
    sectionID := int64(999)

    // 1. 保存到数据库
    err := saveExternalAgentConversationToDatabase(ctx, conversationID, agentID, externalConvID, sectionID)
    assert.NoError(t, err)

    // 2. 从数据库加载
    exeCfg := &workflowModel.ExecuteConfig{
        ExternalAgentConversations: make(map[string]*workflowModel.ExternalAgentConversationInfo),
    }
    loadExternalAgentConversationsFromDB(ctx, conversationID, exeCfg)

    // 3. 验证加载结果
    info := exeCfg.ExternalAgentConversations[agentID]
    assert.NotNil(t, info)
    assert.Equal(t, externalConvID, info.ExternalConversationID)
    assert.Equal(t, sectionID, info.LastSectionID)
}
```

#### 4.1.2 前端单元测试

```typescript
// frontend/packages/workflow/playground/src/components/model-select/__tests__/index.test.tsx

import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import ModelSelect from '../index';
import { modelmgr } from '@coze-studio/api-schema';

describe('ModelSelect', () => {
  it('应该正确分组显示外部智能体', () => {
    const mockModels: modelmgr.ModelInfo[] = [
      {
        model_id: 'gpt-4',
        model_name: 'GPT-4',
        provider: 'openai',
      },
      {
        model_id: 'hiagent_1',
        model_name: '客服助手',
        provider: 'external_agent',
        agent_info: {
          agent_id: 'hiagent_1',
          agent_name: '客服助手',
          agent_type: modelmgr.ExternalAgentType.VOLCENGINE_HIAGENT,
        },
      },
    ];

    const { container } = render(
      <ModelSelect value="" onChange={vi.fn()} spaceId="123" />
    );

    // 点击展开下拉框
    fireEvent.click(container.querySelector('.select-trigger'));

    // 验证分组
    expect(screen.getByText('OpenAI')).toBeInTheDocument();
    expect(screen.getByText('外部智能体')).toBeInTheDocument();

    // 验证外部智能体显示特殊标记
    expect(screen.getByText('智能体')).toBeInTheDocument();
  });

  it('选择外部智能体时应该传递完整信息', () => {
    const mockOnChange = vi.fn();
    const agentInfo: modelmgr.AgentInfo = {
      agent_id: 'hiagent_1',
      agent_name: '客服助手',
      agent_type: modelmgr.ExternalAgentType.VOLCENGINE_HIAGENT,
    };

    const { container } = render(
      <ModelSelect value="" onChange={mockOnChange} spaceId="123" />
    );

    // 选择外部智能体
    fireEvent.click(container.querySelector('.select-trigger'));
    fireEvent.click(screen.getByText('客服助手'));

    // 验证回调参数
    expect(mockOnChange).toHaveBeenCalledWith('hiagent_1', expect.objectContaining({
      agent_info: agentInfo,
    }));
  });
});
```

### 4.2 集成测试

```bash
# backend/scripts/test_external_agent.sh

#!/bin/bash

set -e

echo "🚀 启动集成测试..."

# 1. 启动测试环境
echo "1️⃣ 启动测试数据库和服务..."
docker-compose -f docker-compose.test.yml up -d

# 2. 等待服务就绪
echo "2️⃣ 等待服务就绪..."
sleep 10

# 3. 创建测试数据
echo "3️⃣ 初始化测试数据..."
go run scripts/setup_test_data.go

# 4. 运行集成测试
echo "4️⃣ 运行集成测试..."

# 测试场景 1: 创建外部智能体配置
echo "测试场景 1: 创建外部智能体配置"
curl -X POST "http://localhost:8888/api/modelmgr/external_agent/create" \
  -H "Content-Type: application/json" \
  -d '{
    "space_id": "1001",
    "agent_info": {
      "agent_id": "test_hiagent_1",
      "agent_name": "测试客服助手",
      "agent_type": 1,
      "config": {
        "api_key": "test_key",
        "api_endpoint": "https://api.test.com/v1/chat",
        "app_id": "test_app_1"
      }
    }
  }'

# 测试场景 2: 获取模型列表（应包含外部智能体）
echo "测试场景 2: 获取模型列表"
curl "http://localhost:8888/api/modelmgr/models/list?space_id=1001"

# 测试场景 3: 创建工作流并使用外部智能体
echo "测试场景 3: 创建工作流"
WORKFLOW_ID=$(curl -X POST "http://localhost:8888/api/workflow/create" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试工作流",
    "space_id": "1001",
    "nodes": [
      {
        "type": "llm",
        "config": {
          "model_id": "test_hiagent_1"
        }
      }
    ]
  }' | jq -r '.data.workflow_id')

echo "创建的工作流ID: $WORKFLOW_ID"

# 测试场景 4: 执行工作流（第一轮对话）
echo "测试场景 4: 第一轮对话"
EXECUTE_ID_1=$(curl -X POST "http://localhost:8888/api/workflow/execute" \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"input\": {
      \"message\": \"请记住我叫张三\"
    }
  }" | jq -r '.data.execute_id')

echo "第一轮执行ID: $EXECUTE_ID_1"

# 等待执行完成
sleep 5

# 检查执行结果
curl "http://localhost:8888/api/workflow/execute/status?execute_id=$EXECUTE_ID_1"

# 测试场景 5: 第二轮对话（应该记住上下文）
echo "测试场景 5: 第二轮对话（验证上下文保持）"
EXECUTE_ID_2=$(curl -X POST "http://localhost:8888/api/workflow/execute" \
  -H "Content-Type: application/json" \
  -d "{
    \"workflow_id\": \"$WORKFLOW_ID\",
    \"input\": {
      \"message\": \"你还记得我叫什么吗？\"
    }
  }" | jq -r '.data.execute_id')

echo "第二轮执行ID: $EXECUTE_ID_2"

sleep 5

# 检查第二轮结果（应该包含"张三"）
RESPONSE=$(curl "http://localhost:8888/api/workflow/execute/status?execute_id=$EXECUTE_ID_2")
echo "第二轮响应: $RESPONSE"

if echo "$RESPONSE" | grep -q "张三"; then
  echo "✅ 上下文保持测试通过！"
else
  echo "❌ 上下文保持测试失败！"
  exit 1
fi

# 清理
echo "5️⃣ 清理测试环境..."
docker-compose -f docker-compose.test.yml down

echo "🎉 所有测试通过！"
```

### 4.3 手动测试场景

#### 场景 1: 基础对话流程

1. **准备工作**：
   - 启动后端服务：`make server`
   - 启动前端开发服务器：`cd frontend/apps/coze-studio && npm run dev`

2. **添加外部智能体配置**：
   - 进入"空间设置" → "模型管理" → "外部智能体"
   - 点击"添加智能体"
   - 填写配置信息并保存

3. **创建工作流**：
   - 新建工作流
   - 添加 LLM 节点
   - 在模型选择器中选择刚添加的外部智能体
   - 保存工作流

4. **第一轮对话**：
   - 在 Playground 中输入："请记住我叫陆志鹏"
   - 点击运行
   - **期望结果**：智能体回复确认记住了
   - **后端日志验证**：
     ```
     💾 saving external agent conversation to DB: agent=xxx, conv_id=xxx, section_id=xxx
     ✅ successfully saved external agent conversation to DB
     ```

5. **第二轮对话**（相同 session）：
   - 输入："你还记得我叫什么吗？"
   - 点击运行
   - **期望结果**：智能体回复"你叫陆志鹏"
   - **后端日志验证**：
     ```
     🔄 loading external agent conversations from database...
     DEBUG: loaded last_section_id=xxx (from int64) for agent=xxx
     reusing external agent conversation: xxx (section_id: xxx)
     ```

#### 场景 2: 会话边界测试

1. **创建新 Section**：
   - 在 ChatFlow 中点击"新建对话"（会生成新的 section_id）
   - 或者重新进入 Playground

2. **发送消息**：
   - 输入："你还记得我叫什么吗？"
   - **期望结果**：智能体回复"不记得"（因为会话已重置）
   - **后端日志验证**：
     ```
     section changed (old: xxx, new: yyy), clearing old conversation
     creating new external agent conversation...
     ```

#### 场景 3: 并发对话测试

1. **打开多个浏览器窗口**

2. **同时发起对话**：
   - 窗口 1："我是用户A"
   - 窗口 2："我是用户B"

3. **验证隔离性**：
   - 窗口 1 询问："我是谁？" → 应返回"用户A"
   - 窗口 2 询问："我是谁？" → 应返回"用户B"

4. **后端日志验证**：
   - 应该看到两个不同的 external_conversation_id

---

## 5. 常见问题与解决方案

### 5.1 前端与后端数据表不一致导致查询失败 ⚠️ **重要**

#### 问题描述
前端正常显示外部智能体列表，但执行工作流时后端报错 "record not found"。

#### 日志特征
```
SELECT * FROM `hi_agent` WHERE (agent_id = '4' AND space_id = 1758272617296667)
record not found
create node 大模型 failed: failed to get external agent: record not found
```

#### 根本原因
前端和后端查询了**不同的数据库表**：

1. **前端 `GetHiAgentList` API** 查询 `external_agent_config` 表
2. **后端工作流执行** 查询 `hi_agent` 表
3. 前端传递的 ID (`agent.agent_id || agent.id`) 在后端查询的表中不存在

这是因为系统存在两套外部智能体表结构：
- **旧表**：`hi_agent` (主键：`agent_id` 字符串)
- **新表**：`external_agent_config` (主键：`id` int64，有可选的 `agent_id` 字段)

#### 解决方案

**方案1：统一查询新表** (推荐)

修改后端工作流执行逻辑，直接查询 `external_agent_config` 表：

```go
// backend/crossdomain/impl/modelmgr/modelmgr.go

func (m *modelManager) getHiAgentModel(ctx context.Context, params *model.LLMParams) (eino.BaseChatModel, *modelmgr.Model, error) {
    // 获取数据库连接
    db, err := mysql.New()
    if err != nil {
        return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
    }

    // 定义查询结构
    type ExternalAgentConfig struct {
        ID          int64   `gorm:"column:id;primaryKey"`
        SpaceID     int64   `gorm:"column:space_id"`
        Name        string  `gorm:"column:name"`
        Description *string `gorm:"column:description"`
        Platform    string  `gorm:"column:platform"`
        AgentURL    string  `gorm:"column:agent_url"`
        AgentKey    *string `gorm:"column:agent_key"`
        AgentID     *string `gorm:"column:agent_id"`
        Status      int32   `gorm:"column:status"`
    }

    var agentConfig ExternalAgentConfig

    // 尝试两种查询方式：
    // 1. 先用数字 ID 查询（对应 agent.id）
    queryErr := db.WithContext(ctx).Table("external_agent_config").
        Where("id = ? AND space_id = ? AND deleted_at IS NULL", params.HiAgentID, params.HiAgentSpaceID).
        First(&agentConfig).Error

    if queryErr == gorm.ErrRecordNotFound {
        // 2. 尝试用字符串 agent_id 查询（对应 agent.agent_id）
        queryErr = db.WithContext(ctx).Table("external_agent_config").
            Where("agent_id = ? AND space_id = ? AND deleted_at IS NULL", params.HiAgentID, params.HiAgentSpaceID).
            First(&agentConfig).Error
    }

    if queryErr != nil {
        logs.CtxErrorf(ctx, "❌ Failed to get external agent: agent_id=%s, space_id=%d, error=%v",
            params.HiAgentID, params.HiAgentSpaceID, queryErr)
        return nil, nil, fmt.Errorf("failed to get external agent: %w", queryErr)
    }

    logs.CtxInfof(ctx, "✅ External Agent loaded: id=%d, platform=%s, name=%s",
        agentConfig.ID, agentConfig.Platform, agentConfig.Name)

    // 使用数据库中的 platform 字段区分类型
    platform := agentConfig.Platform

    // 根据平台创建对应的模型
    // ...
}
```

**方案2：统一前端查询旧表**

修改 `GetHiAgentList` API 查询 `hi_agent` 表，但不推荐（旧表结构可能被废弃）。

#### 关键改进点

1. **支持两种 ID 格式查询**：
   - 先尝试用数字 `id` 查询（前端 `agent.id`）
   - 失败后尝试用字符串 `agent_id` 查询（前端 `agent.agent_id`）

2. **使用数据库 platform 字段**：
   - 不再通过 endpoint URL 推断平台类型
   - 直接使用 `external_agent_config.platform` 字段

3. **详细的错误日志**：
   ```go
   logs.CtxInfof(ctx, "✅ External Agent loaded: id=%d, platform=%s, name=%s",
       agentConfig.ID, agentConfig.Platform, agentConfig.Name)
   ```

#### 验证方式

1. **检查前端 API 返回**：
   ```bash
   curl "http://localhost:8888/api/space/{space_id}/hi-agents" | jq
   ```
   确认返回的 `id` 和 `agent_id` 字段。

2. **检查数据库记录**：
   ```sql
   SELECT id, agent_id, name, platform FROM external_agent_config
   WHERE space_id = {space_id} AND deleted_at IS NULL;
   ```

3. **运行工作流测试**：
   - 选择外部智能体
   - 执行工作流
   - 检查后端日志应显示：
     ```
     ✅ External Agent loaded: id=4, platform=dify, name=测试Dify智能体
     ```

#### 预防措施

1. **API 设计时明确 ID 类型**：
   - 使用 `id` 作为数字主键
   - 使用 `external_id` 作为外部系统的字符串ID
   - 避免混淆

2. **统一数据表访问**：
   - 同一模块的所有操作使用同一张表
   - 通过 Repository 模式封装数据访问逻辑

3. **前后端联调测试**：
   - 确保前端传递的 ID 在后端能正确查询到
   - 添加集成测试覆盖完整流程

### 5.2 类型断言失败

#### 问题描述
数据库中正确保存了 `last_section_id`，但加载时为 0。

#### 日志特征
```
[Warn] DEBUG: last_section_id not found in DB for agent=xxx, convMap=map[...last_section_id:7566455633650663424]
[Info] DEBUG: loaded ExternalAgentConversationInfo for agent=xxx: ...last_section_id=0
```

#### 根本原因
不同的 JSON 库对数字的反序列化处理不同：
- 标准 `encoding/json`：将数字反序列化为 `float64`
- `sonic` 库：将整数反序列化为 `int64`

#### 解决方案
在加载逻辑中支持两种类型：

```go
// 支持 float64 (标准库)
if lastSectionID, ok := convMap["last_section_id"].(float64); ok {
    info.LastSectionID = int64(lastSectionID)
} else if lastSectionID, ok := convMap["last_section_id"].(int64); ok {
    // 支持 int64 (sonic库)
    info.LastSectionID = lastSectionID
} else {
    // 降级处理：尝试其他数字类型
    logs.CtxWarnf(ctx, "unexpected type for last_section_id: %T", convMap["last_section_id"])
}
```

### 5.3 前端 Tab 分离：模型选择器架构升级

#### 问题描述
需要将原有的 2 个 Tab（标准模型、HiAgent）拆分为 3 个独立的 Tab（标准模型、HiAgent、Dify），以便：
1. 清晰区分不同的外部智能体平台
2. 支持平台特定的配置 UI
3. 为未来接入更多平台（如百度文心、阿里通义）奠定基础

#### 实现方案

**1. 扩展类型定义**

```typescript
// frontend/packages/workflow/playground/src/typing/index.ts

export interface IModelValue {
  // 现有字段
  isHiagent?: boolean;
  hiagentId?: string;
  hiagentSpaceId?: string;

  // 新增：平台标识
  externalAgentPlatform?: 'hiagent' | 'dify';  // 可扩展为联合类型
}
```

**2. 创建平台专用选择器组件**

```typescript
// DifySelector: frontend/packages/workflow/playground/src/nodes-v2/llm/dify-selector/index.tsx
// 过滤 platform === 'dify' 的智能体
const difyAgents = agents.filter(agent => agent.platform === 'dify');

// HiAgentSelector: 修改过滤逻辑
const hiagentAgents = agents.filter(agent =>
  !agent.platform || agent.platform === 'hiagent'
);
```

**3. ModelSelect 组件状态管理**

关键点：避免状态竞争
```typescript
// ❌ 错误：使用计算值会导致竞争
const value = useMemo(() => ..., [_value]);
useEffect(() => {
  setActiveTab(value.isHiagent ? 'hiagent' : 'standard');
}, [value]);

// ✅ 正确：直接使用 props 值
useEffect(() => {
  if (!_value?.isHiagent) {
    setActiveTab('standard');
  } else {
    const newTab = _value?.externalAgentPlatform === 'dify' ? 'dify' : 'hiagent';
    setActiveTab(newTab);
  }
}, [_value?.isHiagent, _value?.externalAgentPlatform]);
```

**4. Tab onChange 处理**

```typescript
<Tabs activeTab={activeTab} onChange={key => {
  if (!key) return;
  setActiveTab(key as 'standard' | 'hiagent' | 'dify');

  // 立即更新父组件状态
  if (key === 'dify') {
    onChange?.({
      isHiagent: true,
      externalAgentPlatform: 'dify',
      hiagentConversationMapping: true,
      // 清除旧数据
      modelName: undefined,
      modelType: undefined,
      hiagentId: undefined,  // ⚠️ 必须清除！
      hiagentSpaceId: undefined,
    });
  }
}}>
  <Tabs.TabPane tab="标准模型" itemKey="standard" />
  <Tabs.TabPane tab="HiAgent" itemKey="hiagent" />
  <Tabs.TabPane tab="Dify" itemKey="dify" />
</Tabs>
```

**⚠️ 重要：使用 `itemKey` 而不是 `key`**

```typescript
// ❌ 错误：@coze-arch/coze-design 的 Tabs 不使用 key prop
<Tabs.TabPane tab="HiAgent" key="hiagent" />

// ✅ 正确：使用 itemKey prop
<Tabs.TabPane tab="HiAgent" itemKey="hiagent" />
```

**5. 后端参数解析**

```go
// backend/domain/workflow/internal/nodes/llm/llm.go

case "externalAgentPlatform":
    if param.Input.Value.Content == nil {
        continue
    }
    strVal, ok := param.Input.Value.Content.(string)
    if !ok {
        continue
    }
    p.ExternalAgentPlatform = strVal
```

#### 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| Tab 切换混乱 | 使用计算值导致状态竞争 | 直接使用 `_value` props |
| onChange 收到 undefined | 使用了错误的 prop 名称 | 使用 `itemKey` 而不是 `key` |
| 切换后数据未清除 | 忘记清除旧字段 | 显式设置为 `undefined` |
| 节点卡片标题未更新 | 字段名不一致 | 同时设置 `subtitle` 和 `subTitle` |

#### 架构优势

1. **平台隔离**：每个平台有独立的 UI 和逻辑
2. **易扩展**：新增平台只需添加新的 Tab 和 Selector 组件
3. **类型安全**：使用 TypeScript 联合类型保证类型正确性
4. **向后兼容**：保留 `isHiagent` 字段，兼容旧数据

### 5.4 会话状态丢失

#### 问题描述
第二轮对话时智能体忘记了第一轮的内容。

#### 排查步骤

1. **检查数据库保存**：
   ```sql
   SELECT id, ext FROM conversation WHERE id = <conversation_id>;
   ```
   验证 `ext` 字段中是否包含 `external_agent_conversations`。

2. **检查数据库加载**：
   在 `loadExternalAgentConversationsFromDB` 添加日志：
   ```go
   logs.CtxInfof(ctx, "DEBUG: raw ext from DB: %s", conv.Ext)
   ```

3. **检查 section_id 比较**：
   在 `ensureConversation` 添加日志：
   ```go
   logs.CtxInfof(ctx, "DEBUG: existingInfo.LastSectionID=%d, current SectionID=%d",
       existingInfo.LastSectionID, *exeCfg.SectionID)
   ```

#### 常见原因

| 原因 | 解决方案 |
|------|---------|
| 数据库保存失败 | 检查 `UpdateExt` 方法是否正确执行 |
| 类型断言失败 | 添加多种类型支持（见 5.1） |
| section_id 变化 | 确认是否符合业务逻辑，考虑调整会话边界判断 |
| ExecuteConfig 未正确传递 | 检查 context 中是否正确设置了 ExecuteConfig |

### 5.3 并发安全问题

#### 问题描述
多个请求并发访问时出现数据竞争或 panic。

#### 解决方案

1. **使用读写锁保护共享状态**：
   ```go
   type ExecuteConfig struct {
       ExternalAgentConversations map[string]*ExternalAgentConversationInfo
       externalAgentConversationsMu sync.RWMutex  // 👈 保护锁
   }

   // 读操作使用读锁
   func (c *ExecuteConfig) GetExternalAgentConversationInfo(agentID string) *ExternalAgentConversationInfo {
       c.externalAgentConversationsMu.RLock()
       defer c.externalAgentConversationsMu.RUnlock()

       return c.ExternalAgentConversations[agentID]
   }

   // 写操作使用写锁
   func (c *ExecuteConfig) SetExternalAgentConversationInfo(agentID string, info *ExternalAgentConversationInfo) {
       c.externalAgentConversationsMu.Lock()
       defer c.externalAgentConversationsMu.Unlock()

       if c.ExternalAgentConversations == nil {
           c.ExternalAgentConversations = make(map[string]*ExternalAgentConversationInfo)
       }
       c.ExternalAgentConversations[agentID] = info
   }
   ```

2. **双重检查锁定模式（Double-Check Locking）**：
   ```go
   func (h *ExternalAgentChatModel) ensureConversation(ctx context.Context) (string, error) {
       // 第一次检查（读锁）
       existingInfo := exeCfg.GetExternalAgentConversationInfo(h.agentID)
       if existingInfo != nil && canReuse(existingInfo) {
           return existingInfo.ExternalConversationID, nil
       }

       // 加写锁创建新会话
       h.conversationMu.Lock()
       defer h.conversationMu.Unlock()

       // 第二次检查（避免重复创建）
       existingInfo = exeCfg.GetExternalAgentConversationInfo(h.agentID)
       if existingInfo != nil && canReuse(existingInfo) {
           return existingInfo.ExternalConversationID, nil
       }

       // 创建新会话
       return h.createNewConversation(ctx)
   }
   ```

### 5.4 前端模型选择不显示

#### 问题描述
前端模型选择器中看不到外部智能体选项。

#### 排查步骤

1. **检查后端 API 返回**：
   ```bash
   curl "http://localhost:8888/api/modelmgr/models/list?space_id=1001" | jq
   ```
   验证响应中是否包含 `agent_info` 字段。

2. **检查前端 API 调用**：
   在浏览器控制台查看网络请求，确认数据正确接收。

3. **检查分组逻辑**：
   在 `ModelSelect` 组件中添加 console.log：
   ```typescript
   console.log('Grouped models:', groupedModels);
   ```

4. **检查条件渲染**：
   确认 `groupedModels['外部智能体']` 不为空。

#### 常见原因

| 原因 | 解决方案 |
|------|---------|
| 配置文件路径错误 | 检查 `backend/conf/external_agents/{space_id}/agents.json` 是否存在 |
| API 返回格式不匹配 | 对比 Thrift IDL 定义和实际返回数据 |
| 前端类型定义不一致 | 运行 `npm run update` 重新生成 TypeScript 类型 |
| 分组逻辑错误 | 检查 `model.agent_info` 判断条件 |

### 5.5 外部智能体 API 调用失败

#### 问题描述
调用外部智能体 API 时返回 401、403 或超时错误。

#### 排查步骤

1. **检查配置文件**：
   ```bash
   cat backend/conf/external_agents/{space_id}/agents.json
   ```
   验证 `api_key`、`api_endpoint`、`app_id` 是否正确。

2. **检查网络连通性**：
   ```bash
   curl -v "https://api.volcengine.com/hiagent/v1/chat"
   ```

3. **检查请求参数**：
   在 `callExternalAgentAPI` 添加日志：
   ```go
   logs.CtxInfof(ctx, "DEBUG: request to external agent: %+v", req)
   ```

4. **检查响应内容**：
   ```go
   logs.CtxInfof(ctx, "DEBUG: response from external agent: status=%d, body=%s", resp.StatusCode, string(body))
   ```

#### 常见错误码

| 错误码 | 原因 | 解决方案 |
|-------|------|---------|
| 401 | API Key 无效或过期 | 更新配置文件中的 `api_key` |
| 403 | 权限不足或 App ID 错误 | 检查 `app_id` 配置，确认账号权限 |
| 404 | API Endpoint 错误 | 检查 `api_endpoint` 是否正确 |
| 429 | 请求频率超限 | 添加重试逻辑，或升级 API 套餐 |
| 500 | 外部服务内部错误 | 联系外部智能体服务商 |
| Timeout | 请求超时 | 增加超时时间，或检查网络状况 |

#### 重试逻辑示例

```go
func (h *ExternalAgentChatModel) callExternalAgentAPIWithRetry(ctx context.Context, req *Request) (*Response, error) {
    maxRetries := 3
    baseDelay := 1 * time.Second

    for i := 0; i < maxRetries; i++ {
        resp, err := h.callExternalAgentAPI(ctx, req)

        if err == nil {
            return resp, nil
        }

        // 判断是否需要重试
        if !isRetryableError(err) {
            return nil, err
        }

        // 指数退避
        delay := baseDelay * time.Duration(1<<uint(i))
        logs.CtxWarnf(ctx, "retry %d/%d after %v: %v", i+1, maxRetries, delay, err)
        time.Sleep(delay)
    }

    return nil, fmt.Errorf("failed after %d retries", maxRetries)
}

func isRetryableError(err error) bool {
    // 网络错误、超时、429、500 等可以重试
    // 401、403、404 等不应重试
    // 实现具体判断逻辑
}
```

---

## 6. 接入检查清单

使用此清单确保完整实现外部智能体接入：

### 6.1 后端实现清单

- [ ] **Thrift IDL 定义**
  - [ ] 定义 `ExternalAgentType` 枚举
  - [ ] 定义 `AgentInfo` 结构体
  - [ ] 在 `ModelInfo` 中添加 `agent_info` 字段
  - [ ] 定义 `ExternalAgentConversationInfo` 结构体
  - [ ] 在 `ExecuteConfig` 中添加 `external_agent_conversations` 字段

- [ ] **代码生成**
  - [ ] 运行 `hz update -idl` 生成后端代码
  - [ ] 运行前端 `npm run update` 生成 TypeScript 类型

- [ ] **外部智能体适配层**
  - [ ] 创建 `ExternalAgentChatModel` 接口
  - [ ] 实现具体的外部智能体 ChatModel（如 `HiAgentChatModel`）
  - [ ] 实现 `ensureConversation` 方法
  - [ ] 实现会话创建逻辑
  - [ ] 实现会话复用判断逻辑
  - [ ] 实现数据库保存逻辑（`saveConversationToDatabase`）
  - [ ] 实现数据库加载逻辑（`loadExternalAgentConversationsFromDB`）
  - [ ] 添加类型断言的多种支持（float64 + int64）
  - [ ] 实现 `Generate` 方法（同步）
  - [ ] 实现 `Stream` 方法（流式）
  - [ ] 添加并发安全保护（sync.RWMutex）

- [ ] **LLM Node 集成**
  - [ ] 添加外部智能体检测逻辑（`isExternalAgent`）
  - [ ] 实现 `generateWithExternalAgent` 方法
  - [ ] 根据 `agent_type` 创建对应的 ChatModel
  - [ ] 处理同步和流式两种模式

- [ ] **模型管理接口**
  - [ ] 实现 `ListModels` 接口（包含外部智能体）
  - [ ] 实现 `loadExternalAgentsFromConfig` 方法
  - [ ] 实现 `CreateExternalAgent` 接口（可选）

- [ ] **配置文件**
  - [ ] 创建 `backend/conf/external_agents/{space_id}/agents.json`
  - [ ] 填写正确的 API Key、Endpoint、App ID

- [ ] **单元测试**
  - [ ] 会话复用测试
  - [ ] 会话重置测试
  - [ ] 数据库保存加载测试
  - [ ] 并发安全测试

### 6.2 前端实现清单

- [ ] **TypeScript 类型**
  - [ ] 验证 `ExternalAgentType` 枚举已生成
  - [ ] 验证 `AgentInfo` 接口已生成
  - [ ] 验证 `ModelInfo` 包含 `agent_info` 字段

- [ ] **ModelSelect 组件**
  - [ ] 添加外部智能体分组逻辑
  - [ ] 渲染外部智能体特殊标记
  - [ ] onChange 回调传递完整 `ModelInfo`

- [ ] **LLM Node 表单**
  - [ ] 集成 `ModelSelect` 组件
  - [ ] 处理模型切换逻辑
  - [ ] 根据是否为外部智能体显示/隐藏配置项

- [ ] **外部智能体配置 UI（可选）**
  - [ ] 创建配置表单组件
  - [ ] 实现新增外部智能体功能
  - [ ] 实现编辑外部智能体功能
  - [ ] 实现删除外部智能体功能

- [ ] **单元测试**
  - [ ] ModelSelect 分组测试
  - [ ] 外部智能体选择测试
  - [ ] 配置表单验证测试

### 6.3 集成测试清单

- [ ] **基础流程测试**
  - [ ] 添加外部智能体配置
  - [ ] 模型列表中显示外部智能体
  - [ ] 创建使用外部智能体的工作流
  - [ ] 第一轮对话成功
  - [ ] 数据库正确保存会话信息

- [ ] **上下文保持测试**
  - [ ] 第二轮对话复用会话
  - [ ] 数据库正确加载会话信息
  - [ ] 外部智能体记住上下文

- [ ] **会话边界测试**
  - [ ] section_id 变化时重置会话
  - [ ] 新会话不包含旧上下文

- [ ] **并发测试**
  - [ ] 多个用户同时对话
  - [ ] 会话隔离正确
  - [ ] 无数据竞争或 panic

- [ ] **错误处理测试**
  - [ ] API Key 错误时的提示
  - [ ] 网络错误时的重试
  - [ ] 超时时的降级处理

### 6.4 文档清单

- [ ] **技术文档**
  - [ ] 架构设计说明
  - [ ] API 接口文档
  - [ ] 数据库 schema 说明
  - [ ] 配置文件格式说明

- [ ] **开发指南**
  - [ ] 新外部智能体接入步骤
  - [ ] 代码示例和模板
  - [ ] 常见问题和解决方案

- [ ] **运维文档**
  - [ ] 部署指南
  - [ ] 监控和告警配置
  - [ ] 故障排查手册

---

## 7. 新外部智能体接入示例

### 7.1 接入百度文心智能体

以百度文心智能体为例，展示完整接入流程。

#### Step 1: 更新 IDL 定义

```thrift
// idl/modelmgr/modelmgr.thrift
enum ExternalAgentType {
    VOLCENGINE_HIAGENT = 1,
    BAIDU_WENXIN = 2,  // 👈 新增
    ALI_TONGYI = 3,
    CUSTOM = 99,
}
```

#### Step 2: 创建适配器实现

```go
// backend/domain/wenxin_agent/wenxin_model.go

package wenxin_agent

import (
    "context"
    "fmt"
    "github.com/cloudwego/eino/schema"
    workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
)

type WenxinAgent struct {
    AgentID     string
    AgentName   string
    APIKey      string
    APIEndpoint string
    AppID       string
}

type WenxinChatModel struct {
    agent *WenxinAgent
    // ...其他字段
}

// NewWenxinChatModel 创建文心智能体模型
func NewWenxinChatModel(ctx context.Context, agentInfo *modelmgr.AgentInfo) (*WenxinChatModel, error) {
    agent := &WenxinAgent{
        AgentID:     agentInfo.AgentID,
        AgentName:   agentInfo.AgentName,
        APIKey:      agentInfo.Config["api_key"],
        APIEndpoint: agentInfo.Config["api_endpoint"],
        AppID:       agentInfo.Config["app_id"],
    }

    return &WenxinChatModel{
        agent: agent,
    }, nil
}

// ensureConversation 实现会话管理（与 HiAgent 类似）
func (w *WenxinChatModel) ensureConversation(ctx context.Context) (string, error) {
    // 复用通用逻辑
    // ...
}

// Generate 实现同步生成
func (w *WenxinChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...schema.ChatModelOption) (*schema.Message, error) {
    externalConvID, err := w.ensureConversation(ctx)
    if err != nil {
        return nil, err
    }

    // 调用百度文心 API
    req := &WenxinChatRequest{
        ConversationID: externalConvID,
        Messages:       convertMessages(input),
        // 文心特有参数
        User: "user_id",
    }

    resp, err := w.callWenxinAPI(ctx, req)
    if err != nil {
        return nil, err
    }

    return parseWenxinResponse(resp), nil
}

// Stream 实现流式生成
func (w *WenxinChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...schema.ChatModelOption) (*schema.StreamReader[*schema.Message], error) {
    // 类似实现
    // ...
}

// callWenxinAPI 调用百度文心 API
func (w *WenxinChatModel) callWenxinAPI(ctx context.Context, req *WenxinChatRequest) (*WenxinChatResponse, error) {
    // 构造 HTTP 请求
    httpReq, err := http.NewRequestWithContext(ctx, "POST", w.agent.APIEndpoint, nil)
    if err != nil {
        return nil, err
    }

    // 设置认证头（百度使用 OAuth 2.0）
    accessToken, err := w.getAccessToken(ctx)
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Authorization", "Bearer "+accessToken)

    // 发送请求
    // ...
}

// getAccessToken 获取百度 OAuth 2.0 access_token
func (w *WenxinChatModel) getAccessToken(ctx context.Context) (string, error) {
    // 实现 OAuth 2.0 客户端凭证流程
    // ...
}
```

#### Step 3: 注册到 LLM Node

```go
// backend/domain/workflow/internal/nodes/llm/llm.go

func (l *llmNode) generateWithExternalAgent(ctx context.Context, input map[string]any, opts ...graph.GenerateOption) (map[string]any, error) {
    agentInfo := l.config.Model.AgentInfo

    var chatModel external_agent.ExternalAgentChatModel
    var err error

    switch agentInfo.AgentType {
    case modelmgr.ExternalAgentType_VOLCENGINE_HIAGENT:
        chatModel, err = ynet_agent.NewHiAgentChatModel(ctx, agentInfo)
    case modelmgr.ExternalAgentType_BAIDU_WENXIN:  // 👈 新增
        chatModel, err = wenxin_agent.NewWenxinChatModel(ctx, agentInfo)
    case modelmgr.ExternalAgentType_ALI_TONGYI:
        chatModel, err = tongyi_agent.NewTongyiChatModel(ctx, agentInfo)
    default:
        return nil, fmt.Errorf("unsupported external agent type: %v", agentInfo.AgentType)
    }

    // ...后续逻辑
}
```

#### Step 4: 添加配置文件

```json
// backend/conf/external_agents/1001/agents.json

[
  {
    "agent_id": "wenxin_agent_001",
    "agent_name": "文心智能体-客服助手",
    "agent_type": 2,
    "description": "百度文心智能体，支持多轮对话",
    "config": {
      "api_key": "your_baidu_api_key",
      "secret_key": "your_baidu_secret_key",
      "api_endpoint": "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions",
      "app_id": "wenxin_app_001"
    }
  }
]
```

#### Step 5: 前端支持

```tsx
// frontend/packages/workflow/playground/src/components/model-select/index.tsx

// 无需修改！通用的分组逻辑自动支持新的外部智能体
// 只要后端返回的 ModelInfo 中包含 agent_info 即可
```

#### Step 6: 测试验证

```bash
# 运行集成测试
bash backend/scripts/test_external_agent.sh

# 手动测试
# 1. 启动服务
make server

# 2. 获取模型列表
curl "http://localhost:8888/api/modelmgr/models/list?space_id=1001" | jq

# 应该看到：
# {
#   "models": [
#     ...
#     {
#       "model_id": "wenxin_agent_001",
#       "model_name": "文心智能体-客服助手",
#       "provider": "external_agent",
#       "agent_info": {
#         "agent_id": "wenxin_agent_001",
#         "agent_type": 2,
#         ...
#       }
#     }
#   ]
# }

# 3. 在前端创建工作流并测试多轮对话
```

---

## 8. 总结

### 8.1 核心设计原则

1. **统一接口**：所有外部智能体都实现 `schema.ChatModel` 接口，保证一致性
2. **适配器模式**：每个外部智能体有独立的适配器，隔离具体实现细节
3. **数据库持久化**：会话状态存储到 `conversation.ext` JSON 字段，支持跨请求保持
4. **会话边界管理**：使用 `section_id` 判断会话边界，自动重置上下文
5. **并发安全**：使用 `sync.RWMutex` 保护共享状态
6. **类型兼容性**：支持多种 JSON 库的数字反序列化类型

### 8.2 关键技术点

| 技术点 | 实现方式 | 文件位置 |
|--------|---------|---------|
| Thrift IDL 定义 | 定义统一的外部智能体接口 | `idl/modelmgr/modelmgr.thrift` |
| 会话状态管理 | ExecuteConfig + Database | `backend/api/model/crossdomain/workflow/workflow.go` |
| 数据库持久化 | conversation.ext JSON 字段 | `backend/crossdomain/impl/conversation/conversation.go` |
| 会话加载逻辑 | 支持多种类型断言 | `backend/domain/workflow/service/executable_impl.go` |
| 外部智能体适配器 | 实现 ChatModel 接口 | `backend/domain/ynet_agent/hiagent_model.go` |
| LLM Node 集成 | 根据 agent_type 创建模型 | `backend/domain/workflow/internal/nodes/llm/llm.go` |
| 前端模型选择 | 自动分组外部智能体 | `frontend/packages/workflow/playground/src/components/model-select/` |

### 8.3 接入新智能体的步骤摘要

1. **更新 Thrift IDL**：添加新的 `ExternalAgentType` 枚举值
2. **生成代码**：运行 `hz update` 和 `npm run update`
3. **创建适配器**：实现 `ExternalAgentChatModel` 接口
4. **注册到 LLM Node**：在 switch case 中添加新类型
5. **添加配置文件**：在 `backend/conf/external_agents/` 下添加配置
6. **测试验证**：运行单元测试和集成测试
7. **更新文档**：补充接入说明和常见问题

### 8.4 后续优化方向

1. **动态配置加载**：支持从 Web UI 管理外部智能体配置，无需重启服务
2. **监控和告警**：添加外部智能体调用的成功率、延迟等指标监控
3. **降级策略**：当外部智能体不可用时，自动切换到备用模型
4. **成本控制**：统计外部智能体调用次数和费用，支持配额管理
5. **多租户隔离**：不同 space 的外部智能体配置完全隔离
6. **会话历史管理**：支持查看和导出外部智能体的对话历史

---

## 9. 参考资料

- [HiAgent 官方文档](https://www.volcengine.com/docs/hiagent/)
- [百度文心智能体 API](https://cloud.baidu.com/doc/WENXINWORKSHOP/index.html)
- [阿里通义千问 API](https://help.aliyun.com/zh/dashscope/)
- [Eino Framework](https://github.com/cloudwego/eino)
- [Go Context Best Practices](https://go.dev/blog/context)
- [Sonic JSON Library](https://github.com/bytedance/sonic)

---

## 附录 A: 错误码定义

```go
// backend/types/errno/external_agent.go

package errno

const (
    // 外部智能体相关错误码 (200xxx)
    ErrExternalAgentNotFound          = 200001  // 外部智能体不存在
    ErrExternalAgentConfigInvalid     = 200002  // 外部智能体配置无效
    ErrExternalAgentAPIKeyInvalid     = 200003  // API Key 无效
    ErrExternalAgentAPICallFailed     = 200004  // 外部 API 调用失败
    ErrExternalAgentConversationNotFound = 200005  // 会话不存在
    ErrExternalAgentTimeout           = 200006  // 外部智能体超时
    ErrExternalAgentQuotaExceeded     = 200007  // 配额超限
)

// 错误消息映射
var ExternalAgentErrorMessages = map[int]string{
    ErrExternalAgentNotFound:          "外部智能体不存在",
    ErrExternalAgentConfigInvalid:     "外部智能体配置无效",
    ErrExternalAgentAPIKeyInvalid:     "API Key 无效或已过期",
    ErrExternalAgentAPICallFailed:     "调用外部智能体 API 失败",
    ErrExternalAgentConversationNotFound: "会话不存在或已过期",
    ErrExternalAgentTimeout:           "外部智能体响应超时",
    ErrExternalAgentQuotaExceeded:     "外部智能体配额已用尽",
}
```

---

## 附录 B: 数据库 Schema

```sql
-- conversation 表
CREATE TABLE `conversation` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '会话ID',
  `space_id` bigint NOT NULL COMMENT '空间ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `ext` text COMMENT '扩展字段（JSON格式）',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_space_user` (`space_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='会话表';

-- ext 字段格式示例
-- {
--   "external_agent_conversations": {
--     "hiagent_xxx": {
--       "external_conversation_id": "d40n6mh926cock3q4r10",
--       "last_section_id": 7566455633650663424,
--       "metadata": {
--         "provider": "volcengine_hiagent",
--         "agent_name": "客服助手"
--       }
--     }
--   }
-- }
```

---

**文档版本**: v1.0
**最后更新**: 2025-10-29
**维护者**: 后端团队
**状态**: ✅ 已验证

---

## ⚠️ 重要提示

此文档基于 HiAgent 实际接入经验总结，已在生产环境验证。接入其他外部智能体时：

1. **严格遵循此流程**：避免重复踩坑
2. **保持架构一致性**：使用相同的会话管理和持久化机制
3. **充分测试**：特别是多轮对话和会话边界场景
4. **及时更新文档**：发现新问题后补充到"常见问题"章节

有任何疑问，请联系后端架构团队。
