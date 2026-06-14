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
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/agentrun"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
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
	UserID  string
	Input   *schema.Message
	History []*schema.Message

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
}

func (r *AgentRunner) StreamExecute(ctx context.Context, req *AgentRequest) (
	sr *schema.StreamReader[*entity.AgentEvent], err error,
) {
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
