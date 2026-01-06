# 流式卡片功能设计文档

## 1. 概述

流式卡片功能允许 LLM 在流式输出过程中实时渲染卡片，而不是等待完整 JSON 输出后再渲染。

### 1.1 核心问题

- **传统方式**：LLM 输出完整 JSON `{"contentList":[...]}`，流式过程中 JSON 不完整无法解析
- **流式方式**：使用标记格式，每个 token 都可以增量解析并实时渲染

### 1.2 设计目标

1. 支持流式增量解析，逐字符处理
2. 兼容现有的 ynet_type 消息体系
3. 同时支持智能体界面和 OpenAPI
4. 向后兼容，不影响现有功能

---

## 2. LLM 输出格式设计

### 2.1 标记格式

```
<<CARD:template_id:template_name>>
<<field_name>>field_value
<<field_name2>>field_value2
<</CARD>>
```

### 2.2 示例

```
根据您的查询，这是北京今天的天气情况：

<<CARD:weather_card:天气卡片>>
<<city>>北京
<<temperature>>25°C
<<weather>>晴天
<<humidity>>45%
<<wind>>东南风 3级
<<description>>今天天气不错，适合户外活动。建议穿着轻便衣物。
<</CARD>>

希望对您有帮助！
```

### 2.3 格式规则

| 标记 | 含义 | 说明 |
|-----|------|------|
| `<<CARD:id:name>>` | 卡片开始 | id=模板ID, name=模板名称 |
| `<<field>>` | 字段开始 | field=字段名，后续内容为字段值 |
| `<</CARD>>` | 卡片结束 | 标记卡片数据完成 |

### 2.4 多卡片支持

```
<<CARD:weather_card:天气卡片>>
<<city>>北京
<<temperature>>25°C
<</CARD>>

<<CARD:news_card:新闻卡片>>
<<title>>今日头条
<<content>>重要新闻内容...
<</CARD>>
```

---

## 3. 后端流式解析器设计

### 3.1 状态机

```
          ┌─────────────────────────────────────────────────┐
          │                                                 │
          ▼                                                 │
    ┌──────────┐    '<'     ┌──────────────┐              │
    │   Idle   │ ─────────> │  MaybeTag    │              │
    │ (普通文本) │            │ (可能是标记)  │              │
    └──────────┘            └──────────────┘              │
          ▲                        │                       │
          │                   '<' (确认)                   │
          │                        ▼                       │
          │                 ┌──────────────┐              │
          │                 │   InTag      │              │
          │                 │ (读取标记内容) │              │
          │                 └──────────────┘              │
          │                        │                       │
          │                   '>>' (标记结束)              │
          │                        ▼                       │
          │                 ┌──────────────┐              │
          │ <</CARD>>       │   InCard     │ ─────────────┘
          └─────────────────│ (在卡片内)    │   (新字段<<)
                            └──────────────┘
```

### 3.2 解析器结构

```go
type StreamCardParser struct {
    state        ParserState
    buffer       strings.Builder
    currentCard  *CardState
    currentField string
    cardIDGen    func() string
}

type ParserState int
const (
    StateIdle     ParserState = iota  // 普通文本
    StateMaybeTag                      // 遇到 <
    StateInTag                         // 确认 <<，读标记
    StateInCard                        // 在卡片内
)

type CardState struct {
    ID           string
    TemplateID   string
    TemplateName string
    Fields       map[string]string
}
```

### 3.3 输出事件类型

```go
type CardEventType string
const (
    CardEventCreate CardEventType = "card_create"  // 创建卡片
    CardEventDelta  CardEventType = "card_delta"   // 字段增量
    CardEventDone   CardEventType = "card_done"    // 卡片完成
)

type CardEvent struct {
    Type         CardEventType
    CardID       string
    TemplateID   string            // create 时有值
    TemplateName string            // create 时有值
    Field        string            // delta 时有值
    Delta        string            // delta 时有值（增量内容）
}

type TextEvent struct {
    Text string  // 普通文本增量
}
```

---

## 4. 消息类型扩展

### 4.1 MetaData 字段

在现有 MetaData 基础上扩展：

```go
// 卡片相关 MetaData 字段
const (
    MetaKeyYnetType     = "ynet_type"      // 消息类型
    MetaKeyCardID       = "card_id"        // 卡片唯一ID
    MetaKeyTemplateID   = "template_id"    // 卡片模板ID
    MetaKeyTemplateName = "template_name"  // 卡片模板名称
    MetaKeyCardField    = "card_field"     // 当前字段名
)

// ynet_type 值
const (
    YnetTypeCardCreate = "card_create"  // 创建卡片
    YnetTypeCardDelta  = "card_delta"   // 字段增量更新
    YnetTypeCardDone   = "card_done"    // 卡片完成
)
```

### 4.2 SSE 消息格式

**创建卡片**
```
event: conversation.message.delta
data: {
  "id": "msg_123",
  "type": "answer",
  "content": "",
  "content_type": "card",
  "meta_data": {
    "ynet_type": "card_create",
    "card_id": "card_uuid_001",
    "template_id": "weather_card",
    "template_name": "天气卡片"
  }
}
```

**字段增量**
```
event: conversation.message.delta
data: {
  "id": "msg_123",
  "type": "answer",
  "content": "北",
  "content_type": "card",
  "meta_data": {
    "ynet_type": "card_delta",
    "card_id": "card_uuid_001",
    "card_field": "city"
  }
}
```

**卡片完成**
```
event: conversation.message.delta
data: {
  "id": "msg_123",
  "type": "answer",
  "content": "",
  "content_type": "card",
  "meta_data": {
    "ynet_type": "card_done",
    "card_id": "card_uuid_001"
  }
}
```

---

## 5. 前端处理逻辑

### 5.1 状态管理

```typescript
interface StreamingCard {
  id: string;
  templateId: string;
  templateName: string;
  fields: Record<string, string>;
  status: 'streaming' | 'complete';
}

// 卡片状态 Map
const streamingCards = new Map<string, StreamingCard>();
```

### 5.2 事件处理

```typescript
function handleCardEvent(message: SSEMessage) {
  const { meta_data, content } = message;
  const ynetType = meta_data?.ynet_type;
  const cardId = meta_data?.card_id;

  switch (ynetType) {
    case 'card_create':
      // 创建新卡片
      streamingCards.set(cardId, {
        id: cardId,
        templateId: meta_data.template_id,
        templateName: meta_data.template_name,
        fields: {},
        status: 'streaming'
      });
      // 渲染空卡片骨架
      renderCardSkeleton(cardId);
      break;

    case 'card_delta':
      // 增量更新字段
      const card = streamingCards.get(cardId);
      if (card) {
        const field = meta_data.card_field;
        card.fields[field] = (card.fields[field] || '') + content;
        // 更新卡片渲染
        updateCardField(cardId, field, card.fields[field]);
      }
      break;

    case 'card_done':
      // 标记卡片完成
      const doneCard = streamingCards.get(cardId);
      if (doneCard) {
        doneCard.status = 'complete';
        // 发送完整数据到 iframe
        sendCardDataToIframe(cardId, doneCard);
      }
      break;
  }
}
```

### 5.3 渲染策略

1. **card_create**：显示卡片骨架/加载状态
2. **card_delta**：实时更新字段值（可用打字机效果）
3. **card_done**：通过 postMessage 发送完整数据到 iframe

---

## 6. 提示词模板

### 6.1 卡片格式说明提示词

```
**可用卡片**
当需要以结构化卡片形式展示内容时，请使用以下标记格式输出：

### 1. 天气卡片
卡片代码: `weather_card`
参数说明：
- `city` (string): 城市名称 (必填)
- `temperature` (string): 温度
- `weather` (string): 天气状况
- `humidity` (string): 湿度
- `wind` (string): 风力风向

输出格式：
<<CARD:weather_card:天气卡片>>
<<city>>城市名称
<<temperature>>温度值
<<weather>>天气状况
<<humidity>>湿度值
<<wind>>风力信息
<</CARD>>

**重要提示**：
1. 卡片标记必须使用 `<<` 和 `>>` 包裹
2. 卡片开始标记格式：`<<CARD:卡片代码:卡片名称>>`
3. 字段标记格式：`<<字段名>>字段值`
4. 卡片结束标记：`<</CARD>>`
5. 字段值可以包含任意文本，直到下一个 `<<` 出现
```

---

## 7. 实现计划

### Phase 1: 核心解析器
- [ ] 实现 `StreamCardParser` 状态机
- [ ] 单元测试覆盖各种边界情况

### Phase 2: 消息管道集成
- [ ] 在 agentflow 流式输出管道中集成解析器
- [ ] 扩展消息 MetaData 字段
- [ ] 更新 `buildARSM2ApiMessage` 处理逻辑

### Phase 3: 前端支持
- [ ] 扩展 `SpecialAnswerContent` 组件支持流式
- [ ] 实现卡片骨架和增量渲染
- [ ] 更新 postMessage 通信逻辑

### Phase 4: 提示词集成
- [ ] 更新 `node_bound_cards.go` 生成新格式提示词
- [ ] 测试 LLM 输出格式正确性

---

## 8. 兼容性考虑

### 8.1 向后兼容

- 普通文本消息不受影响
- 未使用卡片的智能体正常工作
- 旧版 JSON 卡片格式继续支持（通过 content_type 区分）

### 8.2 降级策略

如果流式解析失败：
1. 收集完整输出后尝试 JSON 解析
2. 作为普通文本消息返回

---

## 9. 测试用例

### 9.1 正常流程
- 单卡片输出
- 多卡片连续输出
- 卡片与普通文本混合

### 9.2 边界情况
- 字段值包含特殊字符
- 卡片标记不完整
- 网络中断后恢复

### 9.3 性能测试
- 大量字段的卡片
- 高频率增量更新
