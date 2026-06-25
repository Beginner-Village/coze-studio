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

package agentflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/agentrun"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/modelmgr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/pkg/safego"
)

type AgentState struct {
	Messages                 []*schema.Message
	UserInput                *schema.Message
	ReturnDirectlyToolCallID string
}

type AgentRequest struct {
	UserID         string
	ConversationID int64
	Input          *schema.Message
	History        []*schema.Message

	Identity *singleagent.AgentIdentity

	ResumeInfo   *singleagent.InterruptInfo
	PreCallTools []*agentrun.ToolsRetriever
	Variables    map[string]string
}

type AgentRunner struct {
	runner            compose.Runnable[*AgentRequest, *schema.Message]
	requireCheckpoint bool

	returnDirectlyTools map[string]struct{}
	containWfTool       bool
	modelInfo           *modelmgr.Model

	// deepAgent 非空时（AGENT_ENGINE=deepagents 且构建成功）走实验性 DeepAgents 引擎，
	// 否则走默认 ReAct compose 图。见 deepagents_bridge.go。
	deepAgent adkAgent
	// cpStore 复用现有 checkpoint store（Redis），供 DeepAgents 引擎做中断/恢复。
	cpStore compose.CheckPointStore

	superAgent bool
	sandboxKey string
	// chatModel is reused for super-agent context-compaction LLM summaries
	// (nil => fall back to the rule-based summary; keeps non-super and test paths intact).
	chatModel chatmodel.ToolCallingChatModel
}

const (
	contextSummaryPath                = "/workspace/.agent/context-summary.json"
	defaultContextCompactMaxBytes     = 160 * 1024
	defaultContextCompactRecentMsgs   = 16
	contextSummarySnippetMaxRunes     = 1600
	contextSummaryInjectedMessageHead = "Context summary (auto-compacted)"
	// defaultContextCompactRatio:当模型配置了上下文窗口(Capability.InputTokens)时,
	// 历史占用达到「窗口 × 该比例」即触发压缩。默认 0.85 —— 给「压缩动作本身 + 下一轮回复」
	// 留足头寸,不卡到接近 100% 才压(那样容易溢出)。
	defaultContextCompactRatio = 0.85
	// defaultContextCompactBytesPerToken:字节↔token 的粗略换算(中英文混合经验值约 3 字节/token)。
	// 仅用于触发判断,无需精确;偏小会更早触发(更安全)。
	defaultContextCompactBytesPerToken = 3.0
)

func (r *AgentRunner) StreamExecute(ctx context.Context, req *AgentRequest) (
	sr *schema.StreamReader[*entity.AgentEvent], err error,
) {
	if req != nil {
		ctx = withToolOutputConversationID(ctx, req.ConversationID)
	}
	// 实验性 DeepAgents 引擎（特性开关，默认关；下游消费的 entity.AgentEvent 类型不变）。
	if r.deepAgent != nil {
		return r.streamExecuteDeep(ctx, req)
	}

	executeID := uuid.New()

	hdl, sr, sw := newReplyCallback(ctx, executeID.String(), r.returnDirectlyTools)

	var composeOpts []compose.Option
	var pipeMsgOpt compose.Option
	var workflowMsgSr *schema.StreamReader[*crossworkflow.WorkflowMessage]
	var workflowMsgSw *schema.StreamWriter[*crossworkflow.WorkflowMessage] // StreamWriter to close when agent finishes
	if r.containWfTool {
		cfReq := crossworkflow.ExecuteConfig{
			AgentID:         &req.Identity.AgentID,
			ConnectorUID:    req.UserID,
			ConnectorID:     req.Identity.ConnectorID,
			BizType:         crossworkflow.BizTypeAgent,
			CustomVariables: req.Variables, // 传递会话级自定义变量到工作流
		}
		if req.Identity.IsDraft {
			cfReq.Mode = crossworkflow.ExecuteModeDebug
		} else {
			cfReq.Mode = crossworkflow.ExecuteModeRelease
		}
		wfConfig := crossworkflow.DefaultSVC().WithExecuteConfig(cfReq)
		composeOpts = append(composeOpts, wfConfig)
		pipeMsgOpt, workflowMsgSr, workflowMsgSw = crossworkflow.DefaultSVC().WithMessagePipe()
		composeOpts = append(composeOpts, pipeMsgOpt)
	}

	composeOpts = append(composeOpts, compose.WithCallbacks(hdl))
	_ = compose.RegisterSerializableType[*AgentState]("agent_state")
	if r.requireCheckpoint {

		defaultCheckPointID := executeID.String()
		if req.ResumeInfo != nil {
			resumeInfo := req.ResumeInfo
			if resumeInfo.InterruptType != singleagent.InterruptEventType_OauthPlugin {
				defaultCheckPointID = resumeInfo.InterruptID
				opts := crossworkflow.DefaultSVC().WithResumeToolWorkflow(resumeInfo.AllWfInterruptData[resumeInfo.ToolCallID], req.Input.Content, resumeInfo.AllWfInterruptData)
				composeOpts = append(composeOpts, opts)
			}
		}

		composeOpts = append(composeOpts, compose.WithCheckPointID(defaultCheckPointID))
	}
	if r.containWfTool && workflowMsgSr != nil {
		safego.Go(ctx, func() {
			r.processWfMidAnswerStream(ctx, sw, workflowMsgSr)
		})
	}
	safego.Go(ctx, func() {
		defer func() {
			if pe := recover(); pe != nil {
				logs.CtxErrorf(ctx, "[AgentRunner] StreamExecute recover, err: %v", pe)

				sw.Send(nil, errors.New("internal server error"))
			}
			// Close workflow message stream writer first, so processWfMidAnswerStream can exit
			if workflowMsgSw != nil {
				workflowMsgSw.Close()
			}
			sw.Close()
		}()
		_, streamErr := r.runner.Stream(ctx, req, composeOpts...)
		if streamErr != nil {
			logs.CtxErrorf(ctx, "[AgentRunner] Stream returned error: %v", streamErr)
			// Error will be propagated through callback's OnError;
			// only send here if it's a setup-level error (before streaming starts)
			sw.Send(nil, streamErr)
		}
	})

	return sr, nil
}

func (r *AgentRunner) processWfMidAnswerStream(ctx context.Context, sw *schema.StreamWriter[*entity.AgentEvent], wfStream *schema.StreamReader[*crossworkflow.WorkflowMessage]) {
	streamInitialized := false
	var srT *schema.StreamReader[*schema.Message]
	var swT *schema.StreamWriter[*schema.Message]
	workflowCallCount := 0

	logs.CtxInfof(ctx, "[WfMidAnswer] Starting workflow mid-answer stream processor")

	defer func() {
		if swT != nil {
			logs.CtxInfof(ctx, "[WfMidAnswer] Closing remaining stream in defer")
			swT.Close()
		}
		logs.CtxInfof(ctx, "[WfMidAnswer] Stream processor finished, total workflow calls: %d", workflowCallCount)
	}()

	for {
		msg, err := wfStream.Recv()

		if err == io.EOF {
			logs.CtxInfof(ctx, "[WfMidAnswer] Workflow stream EOF")
			break
		}
		if msg == nil || msg.DataMessage == nil {
			continue
		}

		if msg.DataMessage.NodeType != crossworkflow.NodeTypeOutputEmitter &&
			msg.DataMessage.NodeType != crossworkflow.NodeTypeCardSelector &&
			msg.DataMessage.NodeType != crossworkflow.NodeTypeAgent {
			continue
		}

		if !streamInitialized {
			streamInitialized = true
			workflowCallCount++
			srT, swT = schema.Pipe[*schema.Message](5)
			logs.CtxInfof(ctx, "[WfMidAnswer] 📤 Creating new tool_mid_answer stream #%d", workflowCallCount)
			sw.Send(&entity.AgentEvent{
				EventType:     singleagent.EventTypeOfToolMidAnswer,
				ToolMidAnswer: srT,
			}, nil)
		}

		extra := make(map[string]any)
		extra["workflow_node_name"] = msg.NodeTitle
		isFinish := msg.DataMessage.Last
		if isFinish {
			extra["is_finish"] = true
		}

		swT.Send(&schema.Message{
			Role:    msg.DataMessage.Role,
			Content: msg.DataMessage.Content,
			Extra:   extra,
		}, nil)

		// 🔥 关键修复：当工作流完成时，立即关闭当前流
		// 这样 push() 函数可以立即收到 EOF，不会阻塞等待整个 agent 结束
		if isFinish {
			logs.CtxInfof(ctx, "[WfMidAnswer] ✅ Workflow #%d finished, closing stream immediately", workflowCallCount)
			swT.Close()
			swT = nil
			streamInitialized = false
		}
	}
}

func (r *AgentRunner) PreHandlerReq(ctx context.Context, req *AgentRequest) *AgentRequest {
	req.Input = r.preHandlerInput(req.Input)
	req.History = r.preHandlerHistory(req.History)
	req.History = r.preHandlerContextCompaction(ctx, req, req.History)
	logs.CtxInfof(ctx, "[AgentRunner] PreHandlerReq, req: %v", conv.DebugJsonToStr(req))

	return req
}

func (r *AgentRunner) preHandlerInput(input *schema.Message) *schema.Message {
	var multiContent []schema.ChatMessagePart

	if len(input.MultiContent) == 0 {
		return input
	}

	unSupportMultiPart := make([]schema.ChatMessagePart, 0, len(input.MultiContent))

	for _, v := range input.MultiContent {
		switch v.Type {
		case schema.ChatMessagePartTypeImageURL:
			if !r.isSupportImage() {
				unSupportMultiPart = append(unSupportMultiPart, v)
			} else {
				multiContent = append(multiContent, v)
			}
		case schema.ChatMessagePartTypeFileURL:
			if !r.isSupportFile() {
				unSupportMultiPart = append(unSupportMultiPart, v)
			} else {
				multiContent = append(multiContent, v)
			}
		case schema.ChatMessagePartTypeAudioURL:
			if !r.isSupportAudio() {
				unSupportMultiPart = append(unSupportMultiPart, v)
			} else {
				multiContent = append(multiContent, v)
			}
		case schema.ChatMessagePartTypeVideoURL:
			if !r.isSupportVideo() {
				unSupportMultiPart = append(unSupportMultiPart, v)
			} else {
				multiContent = append(multiContent, v)
			}
		case schema.ChatMessagePartTypeText:
		default:
			multiContent = append(multiContent, v)
		}
	}

	for _, v := range input.MultiContent {
		if v.Type != schema.ChatMessagePartTypeText {
			continue
		}

		if r.isSupportMultiContent() {
			if len(multiContent) > 0 {
				v.Text = concatContentString(v.Text, unSupportMultiPart)
				multiContent = append(multiContent, v)
			} else {
				input.Content = concatContentString(v.Text, unSupportMultiPart)
			}
		} else {
			input.Content = concatContentString(v.Text, unSupportMultiPart)
		}

	}
	input.MultiContent = multiContent
	return input
}
func concatContentString(textContent string, unSupportTypeURL []schema.ChatMessagePart) string {
	if len(unSupportTypeURL) == 0 {
		return textContent
	}

	// Build a human-readable notice about stripped content
	var imageCount, fileCount, audioCount, videoCount int
	for _, v := range unSupportTypeURL {
		switch v.Type {
		case schema.ChatMessagePartTypeImageURL:
			imageCount++
		case schema.ChatMessagePartTypeFileURL:
			fileCount++
		case schema.ChatMessagePartTypeAudioURL:
			audioCount++
		case schema.ChatMessagePartTypeVideoURL:
			videoCount++
		}
	}

	var notices []string
	if imageCount > 0 {
		notices = append(notices, fmt.Sprintf("%d张图片", imageCount))
	}
	if fileCount > 0 {
		notices = append(notices, fmt.Sprintf("%d个文件", fileCount))
	}
	if audioCount > 0 {
		notices = append(notices, fmt.Sprintf("%d段音频", audioCount))
	}
	if videoCount > 0 {
		notices = append(notices, fmt.Sprintf("%d个视频", videoCount))
	}

	if len(notices) > 0 {
		notice := fmt.Sprintf("\n[系统提示: 当前模型不支持多模态输入，已自动忽略%s。如需处理这些内容，请切换至支持多模态的模型。]", strings.Join(notices, "、"))
		textContent += notice
	}

	return textContent
}

func (r *AgentRunner) preHandlerHistory(history []*schema.Message) []*schema.Message {
	var hm []*schema.Message
	for _, msg := range history {
		if msg.Role == schema.User {
			msg = r.preHandlerInput(msg)
		}
		hm = append(hm, msg)
	}

	// 修复：验证并修复消息序列，确保每个带有 tool_calls 的 assistant 消息
	// 后面都有对应的 tool 响应消息，否则 Qwen 等模型会报错
	hm = r.validateAndFixToolCallSequence(hm)

	return hm
}

type contextCompactionSummary struct {
	Version           string   `json:"version"`
	Summary           string   `json:"summary"`
	UpdatedAt         int64    `json:"updated_at"`
	Trigger           string   `json:"trigger,omitempty"`
	OriginalMessages  int      `json:"original_messages,omitempty"`
	CompactedMessages int      `json:"compacted_messages,omitempty"`
	RetainedMessages  int      `json:"retained_messages,omitempty"`
	OriginalBytes     int      `json:"original_bytes,omitempty"`
	MaxBytes          int      `json:"max_bytes,omitempty"`
	SummaryPath       string   `json:"summary_path,omitempty"`
	KeyFiles          []string `json:"key_files,omitempty"`
	Artifacts         []string `json:"artifacts,omitempty"`
	NextActions       []string `json:"next_actions,omitempty"`
}

func (r *AgentRunner) preHandlerContextCompaction(ctx context.Context, req *AgentRequest, history []*schema.Message) []*schema.Message {
	maxBytes := r.contextCompactThresholdBytes()
	originalBytes := historyApproxBytes(history)
	if !r.superAgent || len(history) == 0 || originalBytes <= maxBytes {
		return history
	}
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return history
	}
	key := r.sandboxKey
	if key == "" && req != nil && req.Identity != nil {
		key = sandboxKeyFor(req.Identity.ConnectorID, req.Identity.AgentID, req.UserID)
	}
	if key == "" {
		return history
	}

	tail := contextCompactionRecentTail(history, contextCompactRecentMessages())
	if len(tail) == 0 || len(tail) >= len(history) {
		return history
	}
	old := history[:len(history)-len(tail)]
	summaryPath := contextSummaryPathForRequest(req)
	summary := buildContextCompactionSummary(ctx, svc, key, req, old, summaryPath, r.chatModel)
	if window := r.modelContextWindowTokens(); window > 0 {
		summary.Trigger = fmt.Sprintf("context_window_ratio_exceeded(window=%d)", window)
	} else {
		summary.Trigger = "history_bytes_exceeded(no_window_configured)"
	}
	summary.OriginalMessages = len(history)
	summary.CompactedMessages = len(old)
	summary.RetainedMessages = len(tail)
	summary.OriginalBytes = originalBytes
	summary.MaxBytes = maxBytes
	summary.SummaryPath = summaryPath
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		logs.CtxWarnf(ctx, "[AgentRunner] marshal context summary failed: %v", err)
		return history
	}
	unlock := lockSandboxFile(key, summaryPath)
	werr := svc.WriteFile(ctx, key, summaryPath, raw)
	unlock()
	if werr != nil {
		logs.CtxWarnf(ctx, "[AgentRunner] write context summary failed: %v", werr)
		return history
	}

	compact := make([]*schema.Message, 0, len(tail)+1)
	compact = append(compact, schema.SystemMessage(formatContextSummaryForModel(summary)))
	compact = append(compact, tail...)
	return r.validateAndFixToolCallSequence(compact)
}

func contextSummaryPathForRequest(req *AgentRequest) string {
	if req != nil && req.ConversationID > 0 {
		return fmt.Sprintf("/workspace/.agent/sessions/%d/context-summary.json", req.ConversationID)
	}
	return contextSummaryPath
}

func contextCompactMaxBytes() int {
	return envInt("AGENT_CONTEXT_COMPACT_MAX_BYTES", defaultContextCompactMaxBytes)
}

// modelContextWindowTokens 返回当前模型的上下文窗口(max input tokens):
//   - 优先用模型配置的 Capability.InputTokens(用户在模型配置里填的精确值);
//   - 未配置时按模型名推断已知家族的窗口(兜底,见 inferContextWindowByModelName);
//   - 仍无法识别返回 0 → 上层退回固定字节阈值(对未知小窗口模型最安全)。
func (r *AgentRunner) modelContextWindowTokens() int {
	if r == nil || r.modelInfo == nil {
		return 0
	}
	if r.modelInfo.Meta.Capability != nil {
		if n := r.modelInfo.Meta.Capability.InputTokens; n > 0 {
			return n
		}
	}
	return inferContextWindowByModelName(r.modelInfo.Name)
}

// inferContextWindowByModelName 按模型名推断已知家族的上下文窗口(token)。
// 仅对能明确识别的家族返回窗口;不认识的返回 0(由上层走字节兜底,避免给小模型猜过大窗口)。
func inferContextWindowByModelName(name string) int {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return 0
	}
	switch {
	case strings.Contains(n, "glm-5") || strings.Contains(n, "glm5") || strings.Contains(n, "glm-4.6") || strings.Contains(n, "glm-4-plus"):
		return 1000000 // GLM-5.x / 长上下文 GLM:1M
	case strings.Contains(n, "claude"):
		return 200000 // Claude:200K
	case strings.Contains(n, "deepseek"):
		return 128000 // DeepSeek:128K
	case strings.Contains(n, "gpt-4o") || strings.Contains(n, "gpt-4.1") || strings.Contains(n, "gpt-4-turbo") || strings.Contains(n, "o1") || strings.Contains(n, "o3"):
		return 128000 // OpenAI 主流长上下文:128K
	case strings.Contains(n, "qwen"):
		return 128000 // 通义千问主流:128K
	case strings.Contains(n, "gemini"):
		return 1000000 // Gemini 1.5/2.x:1M
	default:
		return 0 // 未知模型:走字节兜底
	}
}

// contextCompactThresholdBytes 计算触发压缩的字节阈值:
//   - 若模型配置了上下文窗口 → 阈值 = 窗口token × ratio × 每token字节数(随模型自适应);
//   - 否则 → 退回固定字节阈值(默认 160KB),保证永不溢出/不崩。
//
// 沿用字节比较(无需引入分词器),把阈值从模型窗口推导即可做到「按窗口百分比压缩」。
func (r *AgentRunner) contextCompactThresholdBytes() int {
	window := r.modelContextWindowTokens()
	if window <= 0 {
		return contextCompactMaxBytes()
	}
	ratio := envFloat("AGENT_CONTEXT_COMPACT_RATIO", defaultContextCompactRatio)
	bytesPerToken := envFloat("AGENT_CONTEXT_COMPACT_BYTES_PER_TOKEN", defaultContextCompactBytesPerToken)
	threshold := int(float64(window) * ratio * bytesPerToken)
	if threshold <= 0 {
		return contextCompactMaxBytes()
	}
	return threshold
}

func envFloat(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func contextCompactRecentMessages() int {
	return envInt("AGENT_CONTEXT_COMPACT_RECENT_MESSAGES", defaultContextCompactRecentMsgs)
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func historyApproxBytes(history []*schema.Message) int {
	total := 0
	for _, msg := range history {
		if msg == nil {
			continue
		}
		total += len(msg.Content)
		for _, part := range msg.MultiContent {
			total += len(part.Text)
			if part.ImageURL != nil {
				total += len(part.ImageURL.URL)
			}
			if part.FileURL != nil {
				total += len(part.FileURL.URL)
			}
			if part.AudioURL != nil {
				total += len(part.AudioURL.URL)
			}
			if part.VideoURL != nil {
				total += len(part.VideoURL.URL)
			}
		}
	}
	return total
}

func contextCompactionRecentTail(history []*schema.Message, keep int) []*schema.Message {
	if keep <= 0 || keep >= len(history) {
		return history
	}
	start := len(history) - keep
	for start > 0 && history[start] != nil && history[start].Role != schema.User {
		start--
	}
	for start < len(history) && history[start] != nil && history[start].Role == schema.Tool {
		start++
	}
	return history[start:]
}

func buildContextCompactionSummary(ctx context.Context, svc crosssandbox.Manager, key string, req *AgentRequest, old []*schema.Message, summaryPath string, chatModel chatmodel.ToolCallingChatModel) *contextCompactionSummary {
	previous := readExistingContextSummary(ctx, svc, key, summaryPath)
	if previous == nil && summaryPath != contextSummaryPath {
		previous = readExistingContextSummary(ctx, svc, key, contextSummaryPath)
	}
	lines := make([]string, 0, 8)
	if previous != nil && strings.TrimSpace(previous.Summary) != "" {
		lines = append(lines, "Previous summary: "+previous.Summary)
	}
	lines = append(lines, summarizeMessagesForContext(old)...)
	if len(lines) == 0 {
		lines = append(lines, "Older conversation context was compacted.")
	}
	summary := strings.Join(lines, "\n")
	if len([]rune(summary)) > contextSummarySnippetMaxRunes {
		rs := []rune(summary)
		summary = string(rs[:contextSummarySnippetMaxRunes]) + "..."
	}

	// Prefer a higher-quality LLM summary when a chat model is available; fall back
	// to the rule-based summary above on any error/timeout (keeps the hot path safe).
	if chatModel != nil {
		if llm := llmCompactionSummary(ctx, chatModel, old, previousSummaryText(previous)); strings.TrimSpace(llm) != "" {
			summary = strings.TrimSpace(llm)
			if len([]rune(summary)) > contextSummarySnippetMaxRunes {
				rs := []rune(summary)
				summary = string(rs[:contextSummarySnippetMaxRunes]) + "..."
			}
		}
	}

	nextActions := make([]string, 0, 1)
	if req != nil && req.Input != nil && strings.TrimSpace(req.Input.Content) != "" {
		nextActions = append(nextActions, strings.TrimSpace(req.Input.Content))
	}

	allText := summary
	if req != nil && req.Input != nil {
		allText += "\n" + req.Input.Content
	}
	return &contextCompactionSummary{
		Version:     "v1",
		Summary:     summary,
		UpdatedAt:   time.Now().Unix(),
		KeyFiles:    uniqueMatches(allText, `/workspace/[^\s,，。；;'"）)]+`),
		Artifacts:   uniqueMatches(allText, `/outputs/[^\s,，。；;'"）)]+`),
		NextActions: nextActions,
	}
}

// contextCompactionLLMTimeout bounds the synchronous LLM summary call so a slow
// model never stalls the user-facing run; on timeout we fall back to rule-based.
const contextCompactionLLMTimeout = 25 * time.Second

// contextCompactionLLMInputBudget caps how many bytes of older turns we feed the
// summarizer (keeps the most recent older turns when over budget).
const contextCompactionLLMInputBudget = 120_000

func previousSummaryText(previous *contextCompactionSummary) string {
	if previous == nil {
		return ""
	}
	return strings.TrimSpace(previous.Summary)
}

// llmCompactionSummary produces a higher-quality structured summary of the older
// turns via the chat model (Hermes-style checkpoint). Returns "" on any error so
// the caller falls back to the rule-based summary. Best-effort, timeout-bounded.
func llmCompactionSummary(ctx context.Context, chatModel chatmodel.ToolCallingChatModel, old []*schema.Message, previous string) string {
	if chatModel == nil || len(old) == 0 {
		return ""
	}
	transcript := renderMessagesForCompaction(old, contextCompactionLLMInputBudget)
	if strings.TrimSpace(transcript) == "" {
		return ""
	}

	var sb strings.Builder
	if previous != "" {
		sb.WriteString("Earlier checkpoint to UPDATE (carry forward still-relevant facts, move finished items to completed):\n")
		sb.WriteString(previous)
		sb.WriteString("\n\n")
	}
	sb.WriteString("Older conversation turns to compact:\n")
	sb.WriteString(transcript)

	sys := schema.SystemMessage("You compress earlier conversation turns into a faithful, concise CHECKPOINT for the agent to keep working. " +
		"Do NOT answer the user or perform any task — only summarize. Write reference-only notes, not a transcript copy. " +
		"Cover, only when present: Goal/Task; Completed actions (tool + outcome); Active state (working dir, files created/edited, test/build status); Key decisions; Open questions / pending user asks; Relevant files and /outputs artifacts (full paths). " +
		"Rewrite pending-sounding actions as past-tense facts so they are not re-executed. Keep it tight (a few short sections).")
	usr := schema.UserMessage(sb.String())

	cctx, cancel := context.WithTimeout(ctx, contextCompactionLLMTimeout)
	defer cancel()
	out, err := chatModel.Generate(cctx, []*schema.Message{sys, usr})
	if err != nil || out == nil {
		logs.CtxWarnf(ctx, "[AgentRunner] LLM context summary failed, using rule-based fallback: %v", err)
		return ""
	}
	return out.Content
}

// renderMessagesForCompaction renders messages as "role: content" lines, capping
// per-message length and total bytes (keeping the most recent when over budget).
func renderMessagesForCompaction(messages []*schema.Message, budgetBytes int) string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		if msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		content := strings.Join(strings.Fields(msg.Content), " ")
		if len([]rune(content)) > 1000 {
			content = string([]rune(content)[:1000]) + "..."
		}
		lines = append(lines, string(msg.Role)+": "+content)
	}
	// Keep the most recent lines within budget.
	total := 0
	start := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		total += len(lines[i]) + 1
		if total > budgetBytes {
			break
		}
		start = i
	}
	return strings.Join(lines[start:], "\n")
}

func readExistingContextSummary(ctx context.Context, svc crosssandbox.Manager, key string, summaryPath string) *contextCompactionSummary {
	if svc == nil {
		return nil
	}
	raw, err := svc.ReadFile(ctx, key, summaryPath)
	if err != nil || strings.TrimSpace(string(raw)) == "" {
		return nil
	}
	summary := &contextCompactionSummary{}
	if err := json.Unmarshal(raw, summary); err != nil {
		return nil
	}
	return summary
}

func summarizeMessagesForContext(messages []*schema.Message) []string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		if msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		role := string(msg.Role)
		content := strings.Join(strings.Fields(msg.Content), " ")
		if len([]rune(content)) > 220 {
			content = string([]rune(content)[:220]) + "..."
		}
		lines = append(lines, role+": "+content)
		if len(lines) >= 12 {
			break
		}
	}
	return lines
}

func uniqueMatches(text string, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		match = strings.TrimRight(match, ".。")
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		out = append(out, match)
	}
	return out
}

func formatContextSummaryForModel(summary *contextCompactionSummary) string {
	if summary == nil {
		return contextSummaryInjectedMessageHead + "\nOlder conversation context was compacted."
	}
	summaryPath := strings.TrimSpace(summary.SummaryPath)
	if summaryPath == "" {
		summaryPath = contextSummaryPath
	}
	parts := []string{
		contextSummaryInjectedMessageHead,
		"Persisted summary: " + summaryPath,
		"Summary:\n" + summary.Summary,
	}
	if len(summary.KeyFiles) > 0 {
		parts = append(parts, "Key files: "+strings.Join(summary.KeyFiles, ", "))
	}
	if len(summary.Artifacts) > 0 {
		parts = append(parts, "Artifacts: "+strings.Join(summary.Artifacts, ", "))
	}
	if len(summary.NextActions) > 0 {
		parts = append(parts, "Next actions: "+strings.Join(summary.NextActions, "; "))
	}
	return strings.Join(parts, "\n")
}

// validateAndFixToolCallSequence 验证并修复 tool_calls 消息序列
// Qwen/OpenAI API 要求：assistant 消息的 tool_calls 必须有对应的 tool 响应消息
func (r *AgentRunner) validateAndFixToolCallSequence(messages []*schema.Message) []*schema.Message {
	if len(messages) == 0 {
		return messages
	}

	var result []*schema.Message

	for i := 0; i < len(messages); i++ {
		msg := messages[i]
		result = append(result, msg)

		// 检查是否是带有 tool_calls 的 assistant 消息
		if msg.Role == schema.Assistant && len(msg.ToolCalls) > 0 {
			// 收集所有需要响应的 tool_call_id
			pendingToolCallIDs := make(map[string]string) // id -> function name
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" {
					pendingToolCallIDs[tc.ID] = tc.Function.Name
				}
			}

			// 检查后续消息是否有对应的 tool 响应
			j := i + 1
			for j < len(messages) {
				nextMsg := messages[j]
				if nextMsg.Role == schema.Tool && nextMsg.ToolCallID != "" {
					delete(pendingToolCallIDs, nextMsg.ToolCallID)
				} else if nextMsg.Role == schema.User || nextMsg.Role == schema.Assistant {
					// 遇到 user 或 assistant 消息，停止检查
					break
				}
				j++
			}

			// 如果有未响应的 tool_calls，插入占位符 tool 消息
			if len(pendingToolCallIDs) > 0 {
				logs.Warnf("[AgentRunner] Found %d tool_calls without responses, inserting placeholders", len(pendingToolCallIDs))
			}
			for toolCallID, funcName := range pendingToolCallIDs {
				logs.Warnf("[AgentRunner] Inserting placeholder for missing tool response: toolCallID=%s, funcName=%s", toolCallID, funcName)
				placeholderToolMsg := &schema.Message{
					Role:       schema.Tool,
					Content:    "[工具调用结果丢失，请重新发起请求]",
					ToolCallID: toolCallID,
					ToolName:   funcName,
				}
				result = append(result, placeholderToolMsg)
			}
		}
	}

	return result
}

func (r *AgentRunner) isSupportMultiContent() bool {
	return len(r.modelInfo.Meta.Capability.InputModal) > 1
}
func (r *AgentRunner) isSupportImage() bool {
	return slices.Contains(r.modelInfo.Meta.Capability.InputModal, modelmgr.ModalImage)
}
func (r *AgentRunner) isSupportFile() bool {
	return slices.Contains(r.modelInfo.Meta.Capability.InputModal, modelmgr.ModalFile)
}
func (r *AgentRunner) isSupportAudio() bool {
	return slices.Contains(r.modelInfo.Meta.Capability.InputModal, modelmgr.ModalAudio)
}
func (r *AgentRunner) isSupportVideo() bool {
	return slices.Contains(r.modelInfo.Meta.Capability.InputModal, modelmgr.ModalVideo)
}
