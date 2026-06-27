/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agentrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/mohae/deepcopy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	messageModel "github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/agentrun"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/message"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crossagent "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/agent"
	crossmessage "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/message"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/internal"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/repository"
	msgEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/imagex"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/pkg/safego"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const agentTracerName = "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun"

type runImpl struct {
	Components

	runProcess *internal.RunProcess
	runEvent   *internal.Event
}

type runtimeDependence struct {
	runID         int64
	agentInfo     *singleagent.SingleAgent
	questionMsgID int64
	runMeta       *entity.AgentRunMeta
	startTime     time.Time

	usage *agentrun.Usage

	// 用于计算每次工具调用的实际耗时，而不是累计时间
	llmStartTime     time.Time // LLM 开始思考的时间（对话开始或上次工具响应的时间）
	lastFuncCallTime time.Time // 上次 function_call 发送的时间，用于计算工具执行时间

	// 可观测性：用于在 trace span 中记录完整对话上下文
	historyMsgCount int    // 历史消息数量
	historyContent  string // 历史消息摘要，用于 trace input
	outputContent   string // 最终回复内容
}

type Components struct {
	RunRecordRepo repository.RunRecordRepo
	ImagexSVC     imagex.ImageX
	TosClient     storage.Storage // 用于将存储 key 在调用模型前展开为 base64
	Cache         cache.Cmdable   // 可为 nil；用于同会话单活跃 run 的幂等防重
}

func NewService(c *Components) Run {
	return &runImpl{
		Components: *c,
		runEvent:   internal.NewMessageEvent(),
		runProcess: internal.NewRunProcess(c.RunRecordRepo),
	}
}

func (c *runImpl) AgentRun(ctx context.Context, arm *entity.AgentRunMeta) (*schema.StreamReader[*entity.AgentRunResponse], error) {
	sr, sw := schema.Pipe[*entity.AgentRunResponse](20)

	defer func() {
		if pe := recover(); pe != nil {
			logs.CtxErrorf(ctx, "panic recover: %v\n, [stack]:%v", pe, string(debug.Stack()))
			return
		}
	}()

	rtDependence := &runtimeDependence{
		runMeta:   arm,
		startTime: time.Now(),
	}

	safego.Go(ctx, func() {
		defer sw.Close()

		// Create agent execution tracing span
		spanCtx, span := otel.Tracer(agentTracerName).Start(ctx, "agent.run",
			oteltrace.WithSpanKind(oteltrace.SpanKindInternal),
		)
		defer span.End()

		// Build current user input string
		var currentInput string
		if arm.DisplayContent != "" {
			currentInput = arm.DisplayContent
		} else if len(arm.Content) > 0 {
			inputBytes, _ := json.Marshal(arm.Content)
			currentInput = string(inputBytes)
		}

		// 先设置基础属性（cozeloop.input 会在 run 之后用完整上下文覆盖）
		span.SetAttributes(
			attribute.String("cozeloop.workspace_id", fmt.Sprintf("%d", arm.SpaceID)),
			attribute.String("cozeloop.span_type", "Agent"),
			attribute.String("cozeloop.input", strings.ToValidUTF8(currentInput, "")),
			attribute.Int64("agent_id", arm.AgentID),
			attribute.Int64("conversation_id", arm.ConversationID),
			attribute.Int64("space_id", arm.SpaceID),
			attribute.String("user_id", arm.UserID),
			attribute.Bool("is_draft", arm.IsDraft),
		)

		runErr := c.run(spanCtx, sw, rtDependence)

		// run() 完成后，用完整上下文覆盖 cozeloop.input（历史 + 当前输入）
		span.SetAttributes(
			attribute.Int("conversation.history_count", rtDependence.historyMsgCount),
		)
		if rtDependence.historyContent != "" {
			fullInput := rtDependence.historyContent + "\n---\n[当前输入] " + currentInput
			span.SetAttributes(
				attribute.String("cozeloop.input", strings.ToValidUTF8(fullInput, "")),
			)
		}
		if rtDependence.outputContent != "" {
			span.SetAttributes(
				attribute.String("cozeloop.output", strings.ToValidUTF8(rtDependence.outputContent, "")),
			)
		}

		if runErr != nil {
			span.RecordError(runErr)
			span.SetStatus(codes.Error, runErr.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	})

	return sr, nil
}

// runLockTTL 是同会话活跃 run 锁的兜底过期时间（防止异常退出导致死锁）。
const runLockTTL = 10 * time.Minute

// acquireRunLock 用 Redis Incr 实现同会话单活跃 run 的幂等锁（cache.Cmdable 无 SetNX，用原子 Incr 替代）。
// 返回 (release, duplicate)：duplicate=true 表示已有进行中的 run；release 用于完成时释放（可能为 nil）。
func (c *runImpl) acquireRunLock(ctx context.Context, conversationID int64) (func(), bool) {
	if c.Cache == nil || conversationID <= 0 {
		return nil, false
	}
	key := fmt.Sprintf("ynet:run:active:%d", conversationID)
	cnt, err := c.Cache.Incr(ctx, key).Result()
	if err != nil {
		// 缓存异常时不阻断业务（降级为不加锁）。
		logs.CtxWarnf(ctx, "acquireRunLock incr failed, skip dedup: %v", err)
		return nil, false
	}
	// 无论是否首个，都刷新 TTL，避免残留死锁。
	c.Cache.Expire(ctx, key, runLockTTL)
	if cnt > 1 {
		return nil, true
	}
	release := func() {
		c.Cache.Del(context.WithoutCancel(ctx), key)
	}
	return release, false
}

func (c *runImpl) run(ctx context.Context, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) (err error) {

	// 幂等：同一会话同一时刻只允许一个活跃 run，避免重复提交产生重复 run/message。
	if release, dup := c.acquireRunLock(ctx, rtDependence.runMeta.ConversationID); dup {
		c.handlerErr(ctx, rtDependence.runMeta.IsDraft, errors.New("当前会话已有正在进行的请求，请等待上一条完成后再试"), sw)
		return nil
	} else if release != nil {
		defer release()
	}

	agentInfo, err := c.handlerAgent(ctx, rtDependence)
	if err != nil {
		return
	}

	rtDependence.agentInfo = agentInfo

	history, err := c.handlerHistory(ctx, rtDependence)
	if err != nil {
		return
	}
	rtDependence.historyMsgCount = len(history)
	rtDependence.historyContent = buildHistorySummary(history)

	runRecord, err := c.createRunRecord(ctx, sw, rtDependence)

	if err != nil {
		return
	}
	rtDependence.runID = runRecord.ID
	defer func() {
		srRecord := c.buildSendRunRecord(ctx, runRecord, entity.RunStatusCompleted)
		if err != nil {
			srRecord.Error = &entity.RunError{
				Code: errno.ErrConversationAgentRunError,
				Msg:  entity.CauseForDebug(rtDependence.runMeta.IsDraft, err),
			}
			c.runProcess.StepToFailed(ctx, srRecord, sw)
			return
		}
		c.runProcess.StepToComplete(ctx, srRecord, sw, rtDependence.usage)
	}()

	input, err := c.handlerInput(ctx, sw, rtDependence)
	if err != nil {
		return
	}

	rtDependence.questionMsgID = input.ID

	err = c.handlerStreamExecute(ctx, sw, history, input, rtDependence)
	return
}

func (c *runImpl) handlerAgent(ctx context.Context, rtDependence *runtimeDependence) (*singleagent.SingleAgent, error) {
	agentInfo, err := crossagent.DefaultSVC().ObtainAgentByIdentity(ctx, &singleagent.AgentIdentity{
		AgentID: rtDependence.runMeta.AgentID,
		IsDraft: rtDependence.runMeta.IsDraft,
	})
	if err != nil {
		return nil, err
	}

	return agentInfo, nil
}

func (c *runImpl) handlerStreamExecute(ctx context.Context, sw *schema.StreamWriter[*entity.AgentRunResponse], historyMsg []*msgEntity.Message, input *msgEntity.Message, rtDependence *runtimeDependence) (err error) {
	// 检查bot_mode，如果是WorkflowMode(2)，使用内部的AgentRuntime来处理
	if rtDependence.agentInfo != nil && rtDependence.agentInfo.BotMode == 2 {
		// 使用内部的AgentRuntime来处理WorkflowMode
		art := &internal.AgentRuntime{
			RunRecord:     &entity.RunRecordMeta{ID: rtDependence.runID},
			AgentInfo:     rtDependence.agentInfo,
			QuestionMsgID: rtDependence.questionMsgID,
			RunMeta:       rtDependence.runMeta,
			StartTime:     rtDependence.startTime,
			Input:         input,
			HistoryMsg:    historyMsg,
			SW:            sw,
			RunProcess:    c.runProcess,
			RunRecordRepo: c.Components.RunRecordRepo,
			ImagexClient:  c.Components.ImagexSVC,
			StorageClient: c.Components.TosClient,
			MessageEvent:  c.runEvent,
		}

		// 调用Run方法，这会根据bot_mode选择ChatflowRun或AgentStreamExecute
		err = art.Run(ctx)
		// 将内部 AgentRuntime 的输出内容回传给 rtDependence，供 trace span 使用
		rtDependence.outputContent = art.OutputContent
		return err
	}

	// 原有的SingleAgent逻辑
	mainChan := make(chan *entity.AgentRespEvent, 100)

	// 将历史消息和输入转换为 schema.Message 格式
	// 过滤掉内部状态消息，这些不应发送给LLM
	var historySchema []*schema.Message
	for _, msg := range historyMsg {
		// 跳过verbose消息（包括generate_answer_finish）
		if msg.MessageType == message.MessageTypeVerbose {
			continue
		}
		// 跳过knowledge消息 - 知识库检索结果已通过系统提示词注入，
		// 不应作为聊天历史发送给大模型，否则会导致大模型模仿JSON格式输出
		if msg.MessageType == message.MessageTypeKnowledge {
			continue
		}

		schemaMsg := buildSchemaMessage(ctx, msg, c.Components.ImagexSVC, c.Components.TosClient)
		if schemaMsg != nil {
			historySchema = append(historySchema, schemaMsg)
		}
	}

	historySchema = buildAgentHistorySchema(historySchema, rtDependence.agentInfo, rtDependence.runMeta.Ext)

	inputSchema := buildSchemaMessage(ctx, input, c.Components.ImagexSVC, c.Components.TosClient)
	if inputSchema == nil {
		inputSchema = &schema.Message{
			Role:    input.Role,
			Content: input.Content,
		}
	}

	ar := &crossagent.AgentRuntime{
		AgentVersion:     rtDependence.runMeta.Version,
		UserID:           rtDependence.runMeta.UserID,
		ConversationID:   rtDependence.runMeta.ConversationID,
		AgentID:          rtDependence.runMeta.AgentID,
		SpaceID:          rtDependence.runMeta.SpaceID,
		IsDraft:          rtDependence.runMeta.IsDraft,
		ConnectorID:      rtDependence.runMeta.ConnectorID,
		PreRetrieveTools: rtDependence.runMeta.PreRetrieveTools,
		HistoryMsg:       historySchema,
		Input:            inputSchema,
		Variables:        rtDependence.runMeta.CustomVariables, // 传递会话自定义变量
		Ext:              rtDependence.runMeta.Ext,
	}

	streamer, err := crossagent.DefaultSVC().StreamExecute(ctx, ar)
	if err != nil {
		return errors.New(errorx.ErrorWithoutStack(err))
	}

	var wg sync.WaitGroup
	wg.Add(2)
	safego.Go(ctx, func() {
		defer wg.Done()
		c.pull(ctx, mainChan, streamer)
	})

	safego.Go(ctx, func() {
		defer wg.Done()
		c.push(ctx, mainChan, sw, rtDependence)
	})

	wg.Wait()

	return err
}

// defaultHistoryTokenBudget 历史消息的默认 token 预算（粗估），0 表示不裁剪。
const defaultHistoryTokenBudget = 12000

// historyTokenBudget 返回历史 token 预算，可经 AGENT_HISTORY_TOKEN_BUDGET 覆盖（设 0 关闭裁剪）。
func historyTokenBudget() int {
	if v := os.Getenv("AGENT_HISTORY_TOKEN_BUDGET"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return defaultHistoryTokenBudget
}

func historyTokenBudgetForAgent(agentInfo *singleagent.SingleAgent) int {
	if agentInfo != nil && strings.EqualFold(strings.TrimSpace(agentInfo.AgentType), "super") {
		return 0
	}
	return historyTokenBudget()
}

func workflowCanvasModeEnabled(ext map[string]string) bool {
	if len(ext) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(ext["workflow_canvas_mode"])) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func buildAgentHistorySchema(history []*schema.Message, agentInfo *singleagent.SingleAgent, ext map[string]string) []*schema.Message {
	if workflowCanvasModeEnabled(ext) {
		return nil
	}

	// 普通智能体仍按 token 预算裁剪；超级智能体交给 harness context compaction
	// 生成 summary + recent tail，避免旧上下文在摘要前被直接丢弃。
	return trimHistoryByTokenBudget(history, historyTokenBudgetForAgent(agentInfo))
}

// estimateMessageTokens 粗略估算一条 schema.Message 的 token 数（CJK 友好的保守估计：约 3 字节/token）。
func estimateMessageTokens(m *schema.Message) int {
	if m == nil {
		return 0
	}
	n := len(m.Content)
	for _, tc := range m.ToolCalls {
		n += len(tc.Function.Name) + len(tc.Function.Arguments)
	}
	for _, mc := range m.MultiContent {
		n += len(mc.Text)
	}
	return n/3 + 8 // 8: 每条消息的角色/结构开销
}

// trimHistoryByTokenBudget 按 token 预算从最旧端裁剪历史，并去掉裁剪后开头的孤立 tool 消息
// （tool 消息必须紧跟在含 tool_calls 的 assistant 消息之后，否则部分模型 API 会报错）。
// budget<=0 时不裁剪。history 假定为 旧->新 顺序。
func trimHistoryByTokenBudget(history []*schema.Message, budget int) []*schema.Message {
	if budget <= 0 || len(history) == 0 {
		return history
	}
	// 从最新往最旧累加，找到能容纳的起点 start。
	total := 0
	start := 0
	for i := len(history) - 1; i >= 0; i-- {
		total += estimateMessageTokens(history[i])
		if total > budget {
			start = i + 1
			break
		}
	}
	trimmed := history[start:]
	// 去掉开头孤立的 tool 消息（其对应的 assistant tool_call 可能已被裁掉）。
	for len(trimmed) > 0 && trimmed[0].Role == schema.Tool {
		trimmed = trimmed[1:]
	}
	return trimmed
}

// buildHistorySummary 将历史消息构建为可读的摘要字符串，用于 trace span 的 cozeloop.input
func buildHistorySummary(history []*msgEntity.Message) string {
	if len(history) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[历史消息]\n")
	for _, msg := range history {
		if msg == nil {
			continue
		}
		// 跳过 verbose 和 knowledge 类型的内部消息
		if msg.MessageType == message.MessageTypeVerbose || msg.MessageType == message.MessageTypeKnowledge {
			continue
		}
		role := string(msg.Role)
		content := msg.Content
		if msg.DisplayContent != "" {
			content = msg.DisplayContent
		}
		// 截断过长的内容，避免 span 属性过大
		if len(content) > 500 {
			content = content[:500] + "...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n", role, content))
	}
	return sb.String()
}

func buildSchemaMessage(ctx context.Context, msg *msgEntity.Message, imagexClient imagex.ImageX, storageClient storage.Storage) *schema.Message {
	if msg == nil {
		return nil
	}

	if msg.ModelContent != "" {
		var modelMsg schema.Message
		if err := json.Unmarshal([]byte(msg.ModelContent), &modelMsg); err == nil {
			if modelMsg.Role == "" {
				modelMsg.Role = msg.Role
			}
			// 入库时 model_content 的图片只存了 key/URI（不再内联 base64），
			// 调用模型前在此把 URI 展开成自包含的 base64 data URL。
			return internal.ParseMessageURI(ctx, &modelMsg, imagexClient, storageClient)
		}
	}

	return &schema.Message{
		Role:         msg.Role,
		Content:      msg.Content,
		MultiContent: buildSchemaMultiContent(msg.MultiContent, msg.Content),
	}
}

func buildSchemaMultiContent(multi []*message.InputMetaData, fallbackText string) []schema.ChatMessagePart {
	if len(multi) == 0 {
		return nil
	}

	parts := make([]schema.ChatMessagePart, 0, len(multi))
	textExists := false
	for _, item := range multi {
		switch item.Type {
		case message.InputTypeText:
			parts = append(parts, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeText,
				Text: item.Text,
			})
			textExists = true
		case message.InputTypeImage:
			if len(item.FileData) == 0 || item.FileData[0] == nil {
				continue
			}
			fd := item.FileData[0]
			parts = append(parts, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{
					URL: fd.Url,
					URI: fd.URI,
				},
			})
		case message.InputTypeFile:
			if len(item.FileData) == 0 || item.FileData[0] == nil {
				continue
			}
			fd := item.FileData[0]
			parts = append(parts, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeFileURL,
				FileURL: &schema.ChatMessageFileURL{
					URL: fd.Url,
					URI: fd.URI,
				},
			})
		case message.InputTypeAudio:
			if len(item.FileData) == 0 || item.FileData[0] == nil {
				continue
			}
			fd := item.FileData[0]
			parts = append(parts, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeAudioURL,
				AudioURL: &schema.ChatMessageAudioURL{
					URL: fd.Url,
					URI: fd.URI,
				},
			})
		case message.InputTypeVideo:
			if len(item.FileData) == 0 || item.FileData[0] == nil {
				continue
			}
			fd := item.FileData[0]
			parts = append(parts, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeVideoURL,
				VideoURL: &schema.ChatMessageVideoURL{
					URL: fd.Url,
				},
			})
		}
	}

	if !textExists && fallbackText != "" {
		parts = append(parts, schema.ChatMessagePart{
			Type: schema.ChatMessagePartTypeText,
			Text: fallbackText,
		})
	}

	if len(parts) == 0 {
		return nil
	}

	return parts
}

func transformEventMap(eventType singleagent.EventType) (message.MessageType, error) {
	var eType message.MessageType
	switch eventType {
	case singleagent.EventTypeOfFuncCall:
		return message.MessageTypeFunctionCall, nil
	case singleagent.EventTypeOfKnowledge:
		return message.MessageTypeKnowledge, nil
	case singleagent.EventTypeOfToolsMessage:
		return message.MessageTypeToolResponse, nil
	case singleagent.EventTypeOfChatModelAnswer:
		return message.MessageTypeAnswer, nil
	case singleagent.EventTypeOfToolsAsChatModelStream:
		return message.MessageTypeToolAsAnswer, nil
	case singleagent.EventTypeOfToolMidAnswer:
		return message.MessageTypeToolMidAnswer, nil
	case singleagent.EventTypeOfSuggest:
		return message.MessageTypeFlowUp, nil
	case singleagent.EventTypeOfInterrupt:
		return message.MessageTypeInterrupt, nil
	}
	return eType, errorx.New(errno.ErrReplyUnknowEventType)
}

func (c *runImpl) buildAgentMessage2Create(ctx context.Context, chunk *entity.AgentRespEvent, messageType message.MessageType, rtDependence *runtimeDependence) *message.Message {
	arm := rtDependence.runMeta
	msg := &msgEntity.Message{
		ConversationID: arm.ConversationID,
		RunID:          rtDependence.runID,
		AgentID:        arm.AgentID,
		SectionID:      arm.SectionID,
		UserID:         arm.UserID,
		MessageType:    messageType,
	}
	buildExt := map[string]string{}

	// 累计时间（从对话开始）- 用于不需要区分的场景
	timeCost := fmt.Sprintf("%.1f", float64(time.Since(rtDependence.startTime).Milliseconds())/1000.00)

	// 🔥 计算 LLM 思考时间：从上次工具响应到现在（首次调用则从开始时间算起）
	llmStartTime := rtDependence.llmStartTime
	if llmStartTime.IsZero() {
		llmStartTime = rtDependence.startTime // 首次调用，从对话开始计算
	}
	llmTimeCost := fmt.Sprintf("%.1f", float64(time.Since(llmStartTime).Milliseconds())/1000.00)

	switch messageType {
	case message.MessageTypeQuestion:
		msg.Role = schema.User
		msg.ContentType = arm.ContentType
		for _, content := range arm.Content {
			if content.Type == message.InputTypeText {
				msg.Content = content.Text
				break
			}
		}
		msg.MultiContent = arm.Content
		buildExt = arm.Ext

		msg.DisplayContent = arm.DisplayContent
	case message.MessageTypeAnswer, message.MessageTypeToolAsAnswer:
		msg.Role = schema.Assistant
		msg.ContentType = message.ContentTypeText

	case message.MessageTypeToolResponse:
		msg.Role = schema.Assistant
		msg.ContentType = message.ContentTypeText
		msg.Content = chunk.ToolsMessage[0].Content

		// 回填与对应 function_call 相同的 call_id,供前端按 ID 配对(支持并行工具调用收尾)。
		buildExt[string(msgEntity.MessageExtKeyCallID)] = chunk.ToolsMessage[0].ToolCallID

		// 🔥 计算工具执行时间 = 当前时间 - function_call 发送时间
		toolTimeCost := timeCost // 默认使用累计时间
		if !rtDependence.lastFuncCallTime.IsZero() {
			toolTimeCost = fmt.Sprintf("%.1f", float64(time.Since(rtDependence.lastFuncCallTime).Milliseconds())/1000.00)
		}
		buildExt[string(msgEntity.MessageExtKeyTimeCost)] = toolTimeCost

		modelContent := chunk.ToolsMessage[0]
		mc, err := json.Marshal(modelContent)
		if err == nil {
			msg.ModelContent = string(mc)
		}

		// 🔥 更新 llmStartTime，为下一次 LLM 思考计时
		rtDependence.llmStartTime = time.Now()

	case message.MessageTypeKnowledge:
		msg.Role = schema.Assistant
		msg.ContentType = message.ContentTypeText

		knowledgeContent := c.buildKnowledge(ctx, chunk)
		if knowledgeContent != nil {
			knInfo, err := json.Marshal(knowledgeContent)
			if err == nil {
				msg.Content = string(knInfo)
			}
		}

		buildExt[string(msgEntity.MessageExtKeyTimeCost)] = timeCost

		modelContent := chunk.Knowledge
		mc, err := json.Marshal(modelContent)
		if err == nil {
			msg.ModelContent = string(mc)
		}

	case message.MessageTypeFunctionCall:
		msg.Role = schema.Assistant
		msg.ContentType = message.ContentTypeText

		if len(chunk.FuncCall.ToolCalls) > 0 {
			toolCall := chunk.FuncCall.ToolCalls[0]
			toolCalling, err := json.Marshal(toolCall)
			if err == nil {
				msg.Content = string(toolCalling)
			}
			buildExt[string(msgEntity.MessageExtKeyPlugin)] = toolCall.Function.Name
			buildExt[string(msgEntity.MessageExtKeyToolName)] = toolCall.Function.Name
			// 回填 call_id:前端按 call_id 配对 function_call 与 tool_response。缺失时会退化为
			// 「索引相邻配对」,并行工具调用(消息序为 fc,fc,fc,fc,tr,tr,tr,tr)下只有边界一个能配上,
			// 其余工具永远停在「正在调用」且不单独计时。设置后每个并行调用都能正确收尾。
			buildExt[string(msgEntity.MessageExtKeyCallID)] = toolCall.ID
			// 🔥 使用 LLM 思考时间，而不是累计时间
			buildExt[string(msgEntity.MessageExtKeyTimeCost)] = llmTimeCost

			modelContent := chunk.FuncCall
			mc, err := json.Marshal(modelContent)
			if err == nil {
				msg.ModelContent = string(mc)
			}

			// 🔥 记录 function_call 发送时间，用于计算后续工具执行时间
			rtDependence.lastFuncCallTime = time.Now()
		}
	case message.MessageTypeFlowUp:
		msg.Role = schema.Assistant
		msg.ContentType = message.ContentTypeText
		msg.Content = chunk.Suggest.Content

	case message.MessageTypeVerbose:
		msg.Role = schema.Assistant
		msg.ContentType = message.ContentTypeText

		d := &entity.Data{
			FinishReason: 0,
			FinData:      "",
		}
		dByte, _ := json.Marshal(d)
		afc := &entity.AnswerFinshContent{
			MsgType: entity.MessageSubTypeGenerateFinish,
			Data:    string(dByte),
		}
		afcMarshal, _ := json.Marshal(afc)
		msg.Content = string(afcMarshal)
	case message.MessageTypeInterrupt:
		msg.Role = schema.Assistant
		msg.MessageType = message.MessageTypeVerbose
		msg.ContentType = message.ContentTypeText

		afc := &entity.AnswerFinshContent{
			MsgType: entity.MessageSubTypeInterrupt,
			Data:    "",
		}
		afcMarshal, _ := json.Marshal(afc)
		msg.Content = string(afcMarshal)

		// Add ext to save to context_message
		interruptByte, err := json.Marshal(chunk.Interrupt)
		if err == nil {
			buildExt[string(msgEntity.ExtKeyResumeInfo)] = string(interruptByte)
		}
		buildExt[string(msgEntity.ExtKeyToolCallsIDs)] = chunk.Interrupt.ToolCallID
		rc := &messageModel.RequiredAction{
			Type:              "submit_tool_outputs",
			SubmitToolOutputs: &messageModel.SubmitToolOutputs{},
		}
		msg.RequiredAction = rc
		rcExtByte, err := json.Marshal(rc)
		if err == nil {
			buildExt[string(msgEntity.ExtKeyRequiresAction)] = string(rcExtByte)
		}
	}

	if messageType != message.MessageTypeQuestion {
		botStateExt := c.buildBotStateExt(arm)
		bseString, err := json.Marshal(botStateExt)
		if err == nil {
			buildExt[string(msgEntity.MessageExtKeyBotState)] = string(bseString)
		}
	}
	msg.Ext = buildExt
	return msg
}

func (c *runImpl) handlerHistory(ctx context.Context, rtDependence *runtimeDependence) ([]*msgEntity.Message, error) {

	conversationTurns := entity.ConversationTurnsDefault

	if rtDependence.agentInfo != nil && rtDependence.agentInfo.ModelInfo != nil && rtDependence.agentInfo.ModelInfo.ShortMemoryPolicy != nil && ptr.From(rtDependence.agentInfo.ModelInfo.ShortMemoryPolicy.HistoryRound) > 0 {
		conversationTurns = ptr.From(rtDependence.agentInfo.ModelInfo.ShortMemoryPolicy.HistoryRound)
	}

	runRecordList, err := c.RunRecordRepo.List(ctx, &entity.ListRunRecordMeta{
		ConversationID: rtDependence.runMeta.ConversationID,
		SectionID:      rtDependence.runMeta.SectionID,
		Limit:          conversationTurns,
	})
	if err != nil {
		return nil, err
	}

	if len(runRecordList) == 0 {
		return nil, nil
	}

	runIDS := c.getRunID(runRecordList)

	history, err := crossmessage.DefaultSVC().GetByRunIDs(ctx, rtDependence.runMeta.ConversationID, runIDS)
	if err != nil {
		return nil, err
	}

	return history, nil
}

func (c *runImpl) getRunID(rr []*entity.RunRecordMeta) []int64 {
	ids := make([]int64, 0, len(rr))
	for _, r := range rr {
		ids = append(ids, r.ID)
	}

	return ids
}

func (c *runImpl) createRunRecord(ctx context.Context, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) (*entity.RunRecordMeta, error) {
	runPoData, err := c.RunRecordRepo.Create(ctx, rtDependence.runMeta)
	if err != nil {
		logs.CtxErrorf(ctx, "RunRecordRepo.Create error: %v", err)
		return nil, err
	}

	srRecord := c.buildSendRunRecord(ctx, runPoData, entity.RunStatusCreated)

	c.runProcess.StepToCreate(ctx, srRecord, sw)

	err = c.runProcess.StepToInProgress(ctx, srRecord, sw)
	if err != nil {
		logs.CtxErrorf(ctx, "runProcess.StepToInProgress error: %v", err)
		return nil, err
	}

	return runPoData, nil
}

func (c *runImpl) handlerInput(ctx context.Context, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) (*msgEntity.Message, error) {
	msgMeta := c.buildAgentMessage2Create(ctx, nil, message.MessageTypeQuestion, rtDependence)

	cm, err := crossmessage.DefaultSVC().Create(ctx, msgMeta)
	if err != nil {
		return nil, err
	}

	ackErr := c.handlerAckMessage(ctx, cm, sw)
	if ackErr != nil {
		return msgMeta, ackErr
	}
	return cm, nil
}

func (c *runImpl) pull(ctx context.Context, mainChan chan *entity.AgentRespEvent, events *schema.StreamReader[*crossagent.AgentEvent]) {
	defer func() {
		logs.CtxDebugf(ctx, "[PULL-DEBUG] closing mainChan")
		close(mainChan)
	}()

	for {
		logs.CtxDebugf(ctx, "[PULL-DEBUG] waiting for events.Recv()...")
		rm, re := events.Recv()
		logs.CtxDebugf(ctx, "[PULL-DEBUG] events.Recv() returned, eventType=%v, err=%v", func() string {
			if rm != nil {
				return string(rm.EventType)
			}
			return "nil"
		}(), re)
		if re != nil {
			errChunk := &entity.AgentRespEvent{
				Err: re,
			}
			mainChan <- errChunk
			return
		}

		eventType, tErr := transformEventMap(rm.EventType)

		if tErr != nil {
			errChunk := &entity.AgentRespEvent{
				Err: tErr,
			}
			mainChan <- errChunk
			return
		}

		respChunk := &entity.AgentRespEvent{
			EventType:    eventType,
			ModelAnswer:  rm.ChatModelAnswer,
			ToolsMessage: rm.ToolsMessage,
			FuncCall:     rm.FuncCall,
			Knowledge:    rm.Knowledge,
			Suggest:      rm.Suggest,
			Interrupt:    rm.Interrupt,

			ToolMidAnswer: rm.ToolMidAnswer,
			ToolAsAnswer:  rm.ToolAsChatModelAnswer,
		}

		logs.CtxDebugf(ctx, "[PULL-DEBUG] sending event to mainChan, type=%v", eventType)
		mainChan <- respChunk
		logs.CtxDebugf(ctx, "[PULL-DEBUG] sent event to mainChan, type=%v", eventType)
	}
}

func (c *runImpl) push(ctx context.Context, mainChan chan *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) {

	var err error
	defer func() {
		if err != nil {
			logs.CtxErrorf(ctx, "run.push error: %v", err)
			c.handlerErr(ctx, rtDependence.runMeta.IsDraft, err, sw)
		}
	}()

	reasoningContent := bytes.NewBuffer([]byte{})

	var firstAnswerMsg *msgEntity.Message
	var reasoningMsg *msgEntity.Message
	isSendFinishAnswer := false
	var preToolResponseMsg *msgEntity.Message
	toolResponseMsgContent := bytes.NewBuffer([]byte{})
	for {
		var chunk *entity.AgentRespEvent
		var ok bool
		select {
		case <-ctx.Done():
			// 客户端断连/请求取消：停止消费并落 cancelled，避免空烧 token。
			logs.CtxWarnf(ctx, "run.push canceled: %v", ctx.Err())
			if rtDependence.runID > 0 {
				detached := context.WithoutCancel(ctx)
				now := time.Now().UnixMilli()
				_ = c.RunRecordRepo.UpdateByID(detached, rtDependence.runID, &entity.UpdateMeta{
					Status:    entity.RunStatusCancelled,
					UpdatedAt: now,
				})
			}
			return
		case chunk, ok = <-mainChan:
		}
		if !ok || chunk == nil {
			return
		}
		logs.CtxDebugf(ctx, "[PUSH-DEBUG] received event from mainChan, event_type=%v, has_stream=%v", chunk.EventType, chunk.ModelAnswer != nil || chunk.ToolAsAnswer != nil || chunk.ToolMidAnswer != nil)
		if chunk.Err != nil {
			if errors.Is(chunk.Err, io.EOF) {
				if !isSendFinishAnswer {
					isSendFinishAnswer = true
					if firstAnswerMsg != nil && len(reasoningContent.String()) > 0 {
						c.saveReasoningContent(ctx, firstAnswerMsg, reasoningContent.String())
						reasoningContent.Reset()
					}

					finishErr := c.handlerFinalAnswerFinish(ctx, sw, rtDependence)
					if finishErr != nil {
						err = finishErr
						return
					}
				}
				return
			}
			c.handlerErr(ctx, rtDependence.runMeta.IsDraft, chunk.Err, sw)
			return
		}

		switch chunk.EventType {
		case message.MessageTypeFunctionCall:
			err = c.handlerFunctionCall(ctx, chunk, sw, rtDependence)
			if err != nil {
				return
			}

			if preToolResponseMsg == nil {
				var cErr error
				preToolResponseMsg, cErr = c.PreCreateAnswer(ctx, rtDependence)
				if cErr != nil {
					err = cErr
					return
				}
			}
		case message.MessageTypeToolResponse:
			err = c.handlerTooResponse(ctx, chunk, sw, rtDependence, preToolResponseMsg, toolResponseMsgContent.String())
			if err != nil {
				return
			}
			preToolResponseMsg = nil // reset
		case message.MessageTypeKnowledge:
			err = c.handlerKnowledge(ctx, chunk, sw, rtDependence)
			if err != nil {
				return
			}
		case message.MessageTypeToolMidAnswer:
			fullMidAnswerContent := bytes.NewBuffer([]byte{})
			var usage *msgEntity.UsageExt
			toolMidAnswerMsg, cErr := c.PreCreateAnswer(ctx, rtDependence)

			if cErr != nil {
				err = cErr
				return
			}

			var preMsgIsFinish = false
			for {
				streamMsg, receErr := chunk.ToolMidAnswer.Recv()
				if receErr != nil {
					if errors.Is(receErr, io.EOF) {
						break
					}
					err = receErr
					return
				}
				if preMsgIsFinish {
					toolMidAnswerMsg, cErr = c.PreCreateAnswer(ctx, rtDependence)
					if cErr != nil {
						err = cErr
						return
					}
					preMsgIsFinish = false
				}
				if streamMsg == nil {
					continue
				}
				if firstAnswerMsg == nil && len(streamMsg.Content) > 0 {
					if reasoningMsg != nil {
						toolMidAnswerMsg = deepcopy.Copy(reasoningMsg).(*msgEntity.Message)
					}
					firstAnswerMsg = deepcopy.Copy(toolMidAnswerMsg).(*msgEntity.Message)
				}

				if streamMsg.Extra != nil {
					if val, ok := streamMsg.Extra["workflow_node_name"]; ok && val != nil {
						// 🔥 修复：确保Ext map已初始化，避免nil map赋值panic
						if toolMidAnswerMsg.Ext == nil {
							toolMidAnswerMsg.Ext = make(map[string]string)
						}
						toolMidAnswerMsg.Ext["message_title"] = val.(string)
					}
				}

				sendMidAnswerMsg := c.buildSendMsg(ctx, toolMidAnswerMsg, false, rtDependence)
				sendMidAnswerMsg.Content = streamMsg.Content
				toolResponseMsgContent.WriteString(streamMsg.Content)
				fullMidAnswerContent.WriteString(streamMsg.Content)

				c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, sendMidAnswerMsg, sw)

				if streamMsg != nil && streamMsg.ResponseMeta != nil {
					usage = c.handlerUsage(streamMsg.ResponseMeta)
				}

				if streamMsg.Extra["is_finish"] == true {
					preMsgIsFinish = true
					sendMidAnswerMsg := c.buildSendMsg(ctx, toolMidAnswerMsg, false, rtDependence)
					sendMidAnswerMsg.Content = fullMidAnswerContent.String()
					fullMidAnswerContent.Reset()
					hfErr := c.handlerAnswer(ctx, sendMidAnswerMsg, sw, usage, rtDependence, toolMidAnswerMsg)
					if hfErr != nil {
						err = hfErr
						return
					}
				}
			}

		case message.MessageTypeToolAsAnswer:
			var usage *msgEntity.UsageExt
			fullContent := bytes.NewBuffer([]byte{})
			toolAsAnswerMsg, cErr := c.PreCreateAnswer(ctx, rtDependence)
			if cErr != nil {
				err = cErr
				return
			}
			if firstAnswerMsg == nil {
				firstAnswerMsg = toolAsAnswerMsg
			}

			// Initialize streaming card handler if agent has bound cards
			var toolAsAnswerCardHandler *internal.StreamCardHandler
			if rtDependence.agentInfo != nil && len(rtDependence.agentInfo.BoundCards) > 0 {
				toolAsAnswerCardHandler = internal.NewStreamCardHandler(rtDependence.agentInfo.BoundCards)
			}

			for {
				streamMsg, receErr := chunk.ToolAsAnswer.Recv()
				if receErr != nil {
					if errors.Is(receErr, io.EOF) {

						// Flush any remaining card buffer content
						if toolAsAnswerCardHandler != nil && toolAsAnswerCardHandler.IsEnabled() {
							flushOutputs := toolAsAnswerCardHandler.Flush()
							for _, out := range flushOutputs {
								if out.ShouldSend() {
									flushMsg := c.buildSendMsg(ctx, toolAsAnswerMsg, false, rtDependence)
									out.ApplyToMessage(flushMsg)
									fullContent.WriteString(out.GetOutputContent())
									c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, flushMsg, sw)
								}
							}
						}

						answer := c.buildSendMsg(ctx, toolAsAnswerMsg, false, rtDependence)
						// Use JSON format for storage if there are completed cards
						if toolAsAnswerCardHandler != nil && toolAsAnswerCardHandler.HasCompletedCards() {
							finalContent, jsonErr := toolAsAnswerCardHandler.BuildFinalContent()
							if jsonErr == nil && finalContent != "" {
								answer.Content = finalContent
							} else {
								answer.Content = fullContent.String()
							}
						} else {
							answer.Content = fullContent.String()
						}
						hfErr := c.handlerAnswer(ctx, answer, sw, usage, rtDependence, toolAsAnswerMsg)
						if hfErr != nil {
							err = hfErr
							return
						}
						break
					}
					err = receErr
					return
				}

				if streamMsg != nil && streamMsg.ResponseMeta != nil {
					usage = c.handlerUsage(streamMsg.ResponseMeta)
				}

				// Process content through streaming card handler if enabled
				if toolAsAnswerCardHandler != nil && toolAsAnswerCardHandler.IsEnabled() {
					outputs, _ := toolAsAnswerCardHandler.ProcessContent(streamMsg.Content)
					for _, out := range outputs {
						if out.ShouldSend() {
							sendMsg := c.buildSendMsg(ctx, toolAsAnswerMsg, false, rtDependence)
							out.ApplyToMessage(sendMsg)
							fullContent.WriteString(out.GetOutputContent())
							c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, sendMsg, sw)
						}
					}
				} else {
					// Original behavior: send content directly
					sendMsg := c.buildSendMsg(ctx, toolAsAnswerMsg, false, rtDependence)
					fullContent.WriteString(streamMsg.Content)
					sendMsg.Content = streamMsg.Content
					c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, sendMsg, sw)
				}
			}

		case message.MessageTypeAnswer:
			logs.CtxDebugf(ctx, "[PUSH-DEBUG] START processing MessageTypeAnswer, will block until stream EOF")
			fullContent := bytes.NewBuffer([]byte{})
			var usage *msgEntity.UsageExt
			var isToolCalls = false
			var modelAnswerMsg *msgEntity.Message

			// Initialize streaming card handler if agent has bound cards
			var cardHandler *internal.StreamCardHandler
			if rtDependence.agentInfo != nil && len(rtDependence.agentInfo.BoundCards) > 0 {
				cardHandler = internal.NewStreamCardHandler(rtDependence.agentInfo.BoundCards)
			}

			for {
				streamMsg, receErr := chunk.ModelAnswer.Recv()
				logs.CtxDebugf(ctx, "[PUSH-DEBUG] ModelAnswer.Recv() returned, hasMsg=%v, err=%v, isToolCalls=%v", streamMsg != nil, receErr, streamMsg != nil && len(streamMsg.ToolCalls) > 0)
				if receErr != nil {
					if errors.Is(receErr, io.EOF) {
						logs.CtxDebugf(ctx, "[PUSH-DEBUG] MessageTypeAnswer stream EOF, isToolCalls=%v, modelAnswerMsg=%v", isToolCalls, modelAnswerMsg != nil)

						if isToolCalls {
							logs.CtxDebugf(ctx, "[PUSH-DEBUG] END MessageTypeAnswer (tool_calls, skipping answer)")
							break
						}
						if modelAnswerMsg == nil {
							break
						}

						// Flush any remaining card buffer content
						if cardHandler != nil && cardHandler.IsEnabled() {
							flushOutputs := cardHandler.Flush()
							for _, out := range flushOutputs {
								if out.ShouldSend() {
									flushMsg := c.buildSendMsg(ctx, modelAnswerMsg, false, rtDependence)
									out.ApplyToMessage(flushMsg)
									fullContent.WriteString(out.GetOutputContent())
									c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, flushMsg, sw)
								}
							}
						}

						answer := c.buildSendMsg(ctx, modelAnswerMsg, false, rtDependence)
						// Use JSON format for storage if there are completed cards
						if cardHandler != nil && cardHandler.HasCompletedCards() {
							finalContent, jsonErr := cardHandler.BuildFinalContent()
							if jsonErr == nil && finalContent != "" {
								answer.Content = finalContent
							} else {
								answer.Content = fullContent.String()
							}
						} else {
							answer.Content = fullContent.String()
						}
						hfErr := c.handlerAnswer(ctx, answer, sw, usage, rtDependence, modelAnswerMsg)
						if hfErr != nil {
							err = hfErr
							return
						}
						break
					}
					err = receErr
					return
				}

				if streamMsg != nil && len(streamMsg.ToolCalls) > 0 {
					isToolCalls = true
				}

				if streamMsg != nil && streamMsg.ResponseMeta != nil {
					usage = c.handlerUsage(streamMsg.ResponseMeta)
				}

				if streamMsg != nil && len(streamMsg.ReasoningContent) == 0 && len(streamMsg.Content) == 0 {
					continue
				}

				if len(streamMsg.ReasoningContent) > 0 {
					if reasoningMsg == nil {
						reasoningMsg, err = c.PreCreateAnswer(ctx, rtDependence)
						if err != nil {
							return
						}
					}

					sendReasoningMsg := c.buildSendMsg(ctx, reasoningMsg, false, rtDependence)
					reasoningContent.WriteString(streamMsg.ReasoningContent)
					sendReasoningMsg.ReasoningContent = ptr.Of(streamMsg.ReasoningContent)
					c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, sendReasoningMsg, sw)
				}
				if len(streamMsg.Content) > 0 {

					if modelAnswerMsg == nil {
						modelAnswerMsg, err = c.PreCreateAnswer(ctx, rtDependence)
						if err != nil {
							return
						}
						if firstAnswerMsg == nil {
							if reasoningMsg != nil {
								modelAnswerMsg.ID = reasoningMsg.ID
							}
							firstAnswerMsg = modelAnswerMsg
						}
					}

					// Process content through streaming card handler if enabled
					if cardHandler != nil && cardHandler.IsEnabled() {
						outputs, _ := cardHandler.ProcessContent(streamMsg.Content)
						for _, out := range outputs {
							if out.ShouldSend() {
								sendAnswerMsg := c.buildSendMsg(ctx, modelAnswerMsg, false, rtDependence)
								out.ApplyToMessage(sendAnswerMsg)
								fullContent.WriteString(out.GetOutputContent())
								c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, sendAnswerMsg, sw)
							}
						}
					} else {
						// Original behavior: send content directly
						sendAnswerMsg := c.buildSendMsg(ctx, modelAnswerMsg, false, rtDependence)
						fullContent.WriteString(streamMsg.Content)
						sendAnswerMsg.Content = streamMsg.Content
						c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, sendAnswerMsg, sw)
					}
				}
			}

		case message.MessageTypeFlowUp:
			if isSendFinishAnswer {

				if firstAnswerMsg != nil && len(reasoningContent.String()) > 0 {
					c.saveReasoningContent(ctx, firstAnswerMsg, reasoningContent.String())
				}

				isSendFinishAnswer = true
				finishErr := c.handlerFinalAnswerFinish(ctx, sw, rtDependence)
				if finishErr != nil {
					err = finishErr
					return
				}
			}

			err = c.handlerSuggest(ctx, chunk, sw, rtDependence)
			if err != nil {
				return
			}

		case message.MessageTypeInterrupt:
			err = c.handlerInterrupt(ctx, chunk, sw, rtDependence, firstAnswerMsg, reasoningContent.String())
			if err != nil {
				return
			}
		}
	}
}

func (c *runImpl) saveReasoningContent(ctx context.Context, firstAnswerMsg *msgEntity.Message, reasoningContent string) {
	_, err := crossmessage.DefaultSVC().Edit(ctx, &message.Message{
		ID:               firstAnswerMsg.ID,
		ReasoningContent: reasoningContent,
	})
	if err != nil {
		logs.CtxInfof(ctx, "save reasoning content failed, err: %v", err)
	}
}

func (c *runImpl) handlerInterrupt(ctx context.Context, chunk *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence, firstAnswerMsg *msgEntity.Message, reasoningCOntent string) error {
	interruptData, cType, err := c.parseInterruptData(ctx, chunk.Interrupt)
	if err != nil {
		return err
	}
	preMsg, err := c.PreCreateAnswer(ctx, rtDependence)
	if err != nil {
		return err
	}
	deltaAnswer := &entity.ChunkMessageItem{
		ID:             preMsg.ID,
		ConversationID: preMsg.ConversationID,
		SectionID:      preMsg.SectionID,
		RunID:          preMsg.RunID,
		AgentID:        preMsg.AgentID,
		Role:           entity.RoleType(preMsg.Role),
		Content:        interruptData,
		MessageType:    preMsg.MessageType,
		ContentType:    cType,
		ReplyID:        preMsg.RunID,
		Ext:            preMsg.Ext,
		IsFinish:       false,
	}

	c.runEvent.SendMsgEvent(entity.RunEventMessageDelta, deltaAnswer, sw)
	finalAnswer := deepcopy.Copy(deltaAnswer).(*entity.ChunkMessageItem)
	if len(reasoningCOntent) > 0 && firstAnswerMsg == nil {
		finalAnswer.ReasoningContent = ptr.Of(reasoningCOntent)
	}
	err = c.handlerAnswer(ctx, finalAnswer, sw, nil, rtDependence, preMsg)
	if err != nil {
		return err
	}

	err = c.handlerInterruptVerbose(ctx, chunk, sw, rtDependence)
	if err != nil {
		return err
	}
	return nil
}

func (c *runImpl) parseInterruptData(_ context.Context, interruptData *singleagent.InterruptInfo) (string, message.ContentType, error) {

	type msg struct {
		Type        string `json:"type,omitempty"`
		ContentType string `json:"content_type"`
		Content     any    `json:"content"` // either optionContent or string
		ID          string `json:"id,omitempty"`
	}

	defaultContentType := message.ContentTypeText
	switch interruptData.InterruptType {
	case singleagent.InterruptEventType_OauthPlugin:
		data := interruptData.AllToolInterruptData[interruptData.ToolCallID].ToolNeedOAuth.Message
		return data, defaultContentType, nil
	case singleagent.InterruptEventType_Question:
		var iData map[string][]*msg
		err := json.Unmarshal([]byte(interruptData.AllWfInterruptData[interruptData.ToolCallID].InterruptData), &iData)
		if err != nil {
			return "", defaultContentType, err
		}
		if len(iData["messages"]) == 0 {
			return "", defaultContentType, errorx.New(errno.ErrInterruptDataEmpty)
		}
		interruptMsg := iData["messages"][0]

		if interruptMsg.ContentType == "text" {
			return interruptMsg.Content.(string), defaultContentType, nil
		} else if interruptMsg.ContentType == "option" || interruptMsg.ContentType == "form_schema" {
			iMarshalData, err := json.Marshal(interruptMsg)
			if err != nil {
				return "", defaultContentType, err
			}
			return string(iMarshalData), message.ContentTypeCard, nil
		}
	case singleagent.InterruptEventType_InputNode:
		data := interruptData.AllWfInterruptData[interruptData.ToolCallID].InterruptData
		return data, message.ContentTypeCard, nil
	case singleagent.InterruptEventType_WorkflowLLM:
		toolInterruptEvent := interruptData.AllWfInterruptData[interruptData.ToolCallID].ToolInterruptEvent
		data := toolInterruptEvent.InterruptData
		if singleagent.InterruptEventType(toolInterruptEvent.EventType) == singleagent.InterruptEventType_InputNode {
			return data, message.ContentTypeCard, nil
		}
		if singleagent.InterruptEventType(toolInterruptEvent.EventType) == singleagent.InterruptEventType_Question {
			var iData map[string][]*msg
			err := json.Unmarshal([]byte(data), &iData)
			if err != nil {
				return "", defaultContentType, err
			}
			if len(iData["messages"]) == 0 {
				return "", defaultContentType, errorx.New(errno.ErrInterruptDataEmpty)
			}
			interruptMsg := iData["messages"][0]

			if interruptMsg.ContentType == "text" {
				return interruptMsg.Content.(string), defaultContentType, nil
			} else if interruptMsg.ContentType == "option" || interruptMsg.ContentType == "form_schema" {
				iMarshalData, err := json.Marshal(interruptMsg)
				if err != nil {
					return "", defaultContentType, err
				}
				return string(iMarshalData), message.ContentTypeCard, nil
			}
		}
		return "", defaultContentType, errorx.New(errno.ErrUnknowInterruptType)

	}
	return "", defaultContentType, errorx.New(errno.ErrUnknowInterruptType)
}

func (c *runImpl) handlerUsage(meta *schema.ResponseMeta) *msgEntity.UsageExt {
	if meta == nil || meta.Usage == nil {
		return nil
	}

	return &msgEntity.UsageExt{
		TotalCount:   int64(meta.Usage.TotalTokens),
		InputTokens:  int64(meta.Usage.PromptTokens),
		OutputTokens: int64(meta.Usage.CompletionTokens),
	}
}

func (c *runImpl) handlerErr(_ context.Context, isDebug bool, err error, sw *schema.StreamWriter[*entity.AgentRunResponse]) {
	errMsg := entity.CauseForDebug(isDebug, err)
	c.runEvent.SendErrEvent(entity.RunEventError, sw, &entity.RunError{
		Code: errno.ErrAgentRun,
		Msg:  errMsg,
	})
}

func (c *runImpl) PreCreateAnswer(ctx context.Context, rtDependence *runtimeDependence) (*msgEntity.Message, error) {
	arm := rtDependence.runMeta
	msgMeta := &msgEntity.Message{
		ConversationID: arm.ConversationID,
		RunID:          rtDependence.runID,
		AgentID:        arm.AgentID,
		SectionID:      arm.SectionID,
		UserID:         arm.UserID,
		Role:           schema.Assistant,
		MessageType:    message.MessageTypeAnswer,
		ContentType:    message.ContentTypeText,
		Ext:            arm.Ext,
	}

	if arm.Ext == nil {
		msgMeta.Ext = map[string]string{}
	}

	botStateExt := c.buildBotStateExt(arm)
	bseString, err := json.Marshal(botStateExt)
	if err != nil {
		return nil, err
	}

	// 🔥 修复：确保Ext map已初始化，避免nil map panic
	if msgMeta.Ext == nil {
		msgMeta.Ext = make(map[string]string)
	}
	if _, ok := msgMeta.Ext[string(msgEntity.MessageExtKeyBotState)]; !ok {
		msgMeta.Ext[string(msgEntity.MessageExtKeyBotState)] = string(bseString)
	}

	msgMeta.Ext = arm.Ext
	return crossmessage.DefaultSVC().PreCreate(ctx, msgMeta)
}

func (c *runImpl) handlerAnswer(ctx context.Context, msg *entity.ChunkMessageItem, sw *schema.StreamWriter[*entity.AgentRunResponse], usage *msgEntity.UsageExt, rtDependence *runtimeDependence, preAnswerMsg *msgEntity.Message) error {

	if len(msg.Content) == 0 && len(ptr.From(msg.ReasoningContent)) == 0 {
		return nil
	}

	msg.IsFinish = true

	if msg.Ext == nil {
		msg.Ext = map[string]string{}
	}
	if usage != nil {
		msg.Ext[string(msgEntity.MessageExtKeyToken)] = strconv.FormatInt(usage.TotalCount, 10)
		msg.Ext[string(msgEntity.MessageExtKeyInputTokens)] = strconv.FormatInt(usage.InputTokens, 10)
		msg.Ext[string(msgEntity.MessageExtKeyOutputTokens)] = strconv.FormatInt(usage.OutputTokens, 10)

		rtDependence.usage = &agentrun.Usage{
			LlmPromptTokens:     usage.InputTokens,
			LlmCompletionTokens: usage.OutputTokens,
			LlmTotalTokens:      usage.TotalCount,
		}
	}

	if _, ok := msg.Ext[string(msgEntity.MessageExtKeyTimeCost)]; !ok {
		msg.Ext[string(msgEntity.MessageExtKeyTimeCost)] = fmt.Sprintf("%.1f", float64(time.Since(rtDependence.startTime).Milliseconds())/1000.00)
	}

	buildModelContent := &schema.Message{
		Role:    schema.Assistant,
		Content: msg.Content,
	}

	mc, err := json.Marshal(buildModelContent)
	if err != nil {
		return err
	}
	preAnswerMsg.Content = msg.Content
	preAnswerMsg.ReasoningContent = ptr.From(msg.ReasoningContent)
	preAnswerMsg.Ext = msg.Ext
	preAnswerMsg.ContentType = msg.ContentType
	preAnswerMsg.ModelContent = string(mc)
	preAnswerMsg.CreatedAt = 0
	preAnswerMsg.UpdatedAt = 0

	_, err = crossmessage.DefaultSVC().Create(ctx, preAnswerMsg)
	if err != nil {
		return err
	}

	// 记录最终回复内容到 rtDependence，用于 trace span output
	rtDependence.outputContent = msg.Content

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, msg, sw)

	return nil
}

func (c *runImpl) buildBotStateExt(arm *entity.AgentRunMeta) *msgEntity.BotStateExt {
	agentID := strconv.FormatInt(arm.AgentID, 10)
	botStateExt := &msgEntity.BotStateExt{
		AgentID:   agentID,
		AgentName: arm.Name,
		Awaiting:  agentID,
		BotID:     agentID,
	}

	return botStateExt
}

func (c *runImpl) handlerFunctionCall(ctx context.Context, chunk *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) error {
	cm := c.buildAgentMessage2Create(ctx, chunk, message.MessageTypeFunctionCall, rtDependence)

	cmData, err := crossmessage.DefaultSVC().Create(ctx, cm)
	if err != nil {
		return err
	}

	sendMsg := c.buildSendMsg(ctx, cmData, true, rtDependence)

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, sendMsg, sw)
	return nil
}

func (c *runImpl) handlerAckMessage(_ context.Context, input *msgEntity.Message, sw *schema.StreamWriter[*entity.AgentRunResponse]) error {
	sendMsg := &entity.ChunkMessageItem{
		ID:             input.ID,
		ConversationID: input.ConversationID,
		SectionID:      input.SectionID,
		AgentID:        input.AgentID,
		Role:           entity.RoleType(input.Role),
		MessageType:    message.MessageTypeAck,
		ReplyID:        input.ID,
		Content:        input.Content,
		ContentType:    message.ContentTypeText,
		IsFinish:       true,
	}

	c.runEvent.SendMsgEvent(entity.RunEventAck, sendMsg, sw)

	return nil
}

func (c *runImpl) handlerTooResponse(ctx context.Context, chunk *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence, preToolResponseMsg *msgEntity.Message, toolResponseMsgContent string) error {

	cm := c.buildAgentMessage2Create(ctx, chunk, message.MessageTypeToolResponse, rtDependence)

	var cmData *message.Message
	var err error

	if preToolResponseMsg != nil {
		cm.ID = preToolResponseMsg.ID
		cm.CreatedAt = preToolResponseMsg.CreatedAt
		cm.UpdatedAt = preToolResponseMsg.UpdatedAt
		if len(toolResponseMsgContent) > 0 {
			cm.Content = toolResponseMsgContent + "\n" + cm.Content
		}
	}

	cmData, err = crossmessage.DefaultSVC().Create(ctx, cm)
	if err != nil {
		return err
	}

	sendMsg := c.buildSendMsg(ctx, cmData, true, rtDependence)

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, sendMsg, sw)

	return nil
}

func (c *runImpl) handlerSuggest(ctx context.Context, chunk *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) error {
	cm := c.buildAgentMessage2Create(ctx, chunk, message.MessageTypeFlowUp, rtDependence)

	cmData, err := crossmessage.DefaultSVC().Create(ctx, cm)
	if err != nil {
		return err
	}

	sendMsg := c.buildSendMsg(ctx, cmData, true, rtDependence)

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, sendMsg, sw)

	return nil
}

func (c *runImpl) handlerKnowledge(ctx context.Context, chunk *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) error {
	cm := c.buildAgentMessage2Create(ctx, chunk, message.MessageTypeKnowledge, rtDependence)
	cmData, err := crossmessage.DefaultSVC().Create(ctx, cm)
	if err != nil {
		return err
	}

	sendMsg := c.buildSendMsg(ctx, cmData, true, rtDependence)

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, sendMsg, sw)
	return nil
}

func (c *runImpl) buildKnowledge(_ context.Context, chunk *entity.AgentRespEvent) *msgEntity.VerboseInfo {
	var recallDatas []msgEntity.RecallDataInfo
	for _, kOne := range chunk.Knowledge {
		recallDatas = append(recallDatas, msgEntity.RecallDataInfo{
			Slice: kOne.Content,
			Meta: msgEntity.MetaInfo{
				Dataset: msgEntity.DatasetInfo{
					ID:   kOne.MetaData["dataset_id"].(string),
					Name: kOne.MetaData["dataset_name"].(string),
				},
				Document: msgEntity.DocumentInfo{
					ID:   kOne.MetaData["document_id"].(string),
					Name: kOne.MetaData["document_name"].(string),
				},
			},
			Score: kOne.Score(),
		})
	}

	verboseData := &msgEntity.VerboseData{
		Chunks:     recallDatas,
		OriReq:     "",
		StatusCode: 0,
	}
	data, err := json.Marshal(verboseData)
	if err != nil {
		return nil
	}
	knowledgeInfo := &msgEntity.VerboseInfo{
		MessageType: string(entity.MessageSubTypeKnowledgeCall),
		Data:        string(data),
	}
	return knowledgeInfo
}

func (c *runImpl) handlerFinalAnswerFinish(ctx context.Context, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) error {
	cm := c.buildAgentMessage2Create(ctx, nil, message.MessageTypeVerbose, rtDependence)
	cmData, err := crossmessage.DefaultSVC().Create(ctx, cm)
	if err != nil {
		return err
	}

	sendMsg := c.buildSendMsg(ctx, cmData, true, rtDependence)

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, sendMsg, sw)
	return nil
}

func (c *runImpl) handlerInterruptVerbose(ctx context.Context, chunk *entity.AgentRespEvent, sw *schema.StreamWriter[*entity.AgentRunResponse], rtDependence *runtimeDependence) error {
	cm := c.buildAgentMessage2Create(ctx, chunk, message.MessageTypeInterrupt, rtDependence)
	cmData, err := crossmessage.DefaultSVC().Create(ctx, cm)
	if err != nil {
		return err
	}

	sendMsg := c.buildSendMsg(ctx, cmData, true, rtDependence)

	c.runEvent.SendMsgEvent(entity.RunEventMessageCompleted, sendMsg, sw)
	return nil
}

func (c *runImpl) buildSendMsg(_ context.Context, msg *msgEntity.Message, isFinish bool, rtDependence *runtimeDependence) *entity.ChunkMessageItem {

	copyMap := make(map[string]string)
	for k, v := range msg.Ext {
		copyMap[k] = v
	}

	return &entity.ChunkMessageItem{
		ID:               msg.ID,
		ConversationID:   msg.ConversationID,
		SectionID:        msg.SectionID,
		AgentID:          msg.AgentID,
		Content:          msg.Content,
		Role:             entity.RoleTypeAssistant,
		ContentType:      msg.ContentType,
		MessageType:      msg.MessageType,
		ReplyID:          rtDependence.questionMsgID,
		Type:             msg.MessageType,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
		RunID:            rtDependence.runID,
		Ext:              copyMap,
		IsFinish:         isFinish,
		ReasoningContent: ptr.Of(msg.ReasoningContent),
	}
}

func (c *runImpl) buildSendRunRecord(_ context.Context, runRecord *entity.RunRecordMeta, runStatus entity.RunStatus) *entity.ChunkRunItem {
	return &entity.ChunkRunItem{
		ID:             runRecord.ID,
		ConversationID: runRecord.ConversationID,
		AgentID:        runRecord.AgentID,
		SectionID:      runRecord.SectionID,
		Status:         runStatus,
		CreatedAt:      runRecord.CreatedAt,
	}
}

func (c *runImpl) Delete(ctx context.Context, runID []int64) error {
	return c.RunRecordRepo.Delete(ctx, runID)
}

func (c *runImpl) Create(ctx context.Context, runRecord *entity.AgentRunMeta) (*entity.RunRecordMeta, error) {
	return c.RunRecordRepo.Create(ctx, runRecord)
}

func (c *runImpl) GetByID(ctx context.Context, runID int64) (*entity.RunRecordMeta, error) {
	runRecord, err := c.RunRecordRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	meta := &entity.RunRecordMeta{
		ID:             runRecord.ID,
		ConversationID: runRecord.ConversationID,
		SectionID:      runRecord.SectionID,
		AgentID:        runRecord.AgentID,
		Status:         entity.RunStatus(runRecord.Status),
		Usage:          runRecord.Usage,
		Ext:            runRecord.Ext,
		CreatedAt:      runRecord.CreatedAt,
		UpdatedAt:      runRecord.UpdatedAt,
		CompletedAt:    runRecord.CompletedAt,
		FailedAt:       runRecord.FailedAt,
	}
	if runRecord.LastError != "" {
		var runErr entity.RunError
		if err := json.Unmarshal([]byte(runRecord.LastError), &runErr); err == nil {
			meta.Error = &runErr
		}
	}
	return meta, nil
}

func (c *runImpl) List(ctx context.Context, listMeta *entity.ListRunRecordMeta) ([]*entity.RunRecordMeta, error) {
	return c.RunRecordRepo.List(ctx, listMeta)
}
