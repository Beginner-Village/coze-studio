/*
 * Copyright 2025 coze-dev Authors
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
package internal

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"sync"

	"github.com/cloudwego/eino/schema"

	"github.com/coze-dev/coze-studio/backend/api/model/crossdomain/agentrun"
	"github.com/coze-dev/coze-studio/backend/api/model/crossdomain/message"
	crossworkflow "github.com/coze-dev/coze-studio/backend/crossdomain/contract/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/conversation/agentrun/entity"
	msgEntity "github.com/coze-dev/coze-studio/backend/domain/conversation/message/entity"
	"github.com/coze-dev/coze-studio/backend/infra/contract/imagex"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

func (art *AgentRuntime) ChatflowRun(ctx context.Context, imagex imagex.ImageX) (err error) {

	mh := &MessageEventHandler{
		sw:           art.SW,
		messageEvent: art.MessageEvent,
	}
	resumeInfo := parseResumeInfo(ctx, art.GetHistory())
	
	// 从LayoutInfo获取chatflow的WorkflowId
	// LayoutInfo存储的是智能体绑定的主chatflow，Workflow列表存储的是作为工具的工作流
	var wfID int64
	agentInfo := art.GetAgentInfo()
	if agentInfo.LayoutInfo != nil && agentInfo.LayoutInfo.WorkflowId != "" {
		wfID, _ = strconv.ParseInt(agentInfo.LayoutInfo.WorkflowId, 10, 64)
	}

	if wfID == 0 {
		mh.handlerErr(ctx, errorx.New(errno.ErrAgentRunWorkflowNotFound))
		return
	}
	var wfStreamer *schema.StreamReader[*crossworkflow.WorkflowMessage]

	// 根据 IsDraft 标志选择执行模式：
	// - IsDraft=true (草稿/测试模式): 使用 Debug 模式，访问 DraftTable
	// - IsDraft=false (发布模式): 使用 Release 模式，访问 OnlineTable
	executeMode := ternary.IFElse(art.GetRunMeta().IsDraft, crossworkflow.ExecuteModeDebug, crossworkflow.ExecuteModeRelease)

	// 将 UserID 从 string 转换为 int64，用于 Operator 字段
	// Operator 字段在 prefetchChatHistory 中用于查询聊天历史记录
	userID, _ := strconv.ParseInt(art.GetRunMeta().UserID, 10, 64)

	executeConfig := crossworkflow.ExecuteConfig{
		ID:              wfID,
		ConnectorID:     art.GetRunMeta().ConnectorID,
		ConnectorUID:    art.GetRunMeta().UserID,
		Operator:        userID, // 修复：添加 Operator 字段，与 ChatFlow 直接运行保持一致
		AgentID:         ptr.Of(art.GetRunMeta().AgentID),
		Mode:            executeMode,
		BizType:         crossworkflow.BizTypeAgent,
		SyncPattern:     crossworkflow.SyncPatternStream,
		CustomVariables: art.GetRunMeta().CustomVariables, // 传递会话自定义变量到工作流，用于覆盖预设变量
	}

	if resumeInfo != nil {
		wfStreamer, err = crossworkflow.DefaultSVC().StreamResume(ctx, &crossworkflow.ResumeRequest{
			ResumeData: concatWfInput(art),
			EventID:    resumeInfo.ChatflowInterrupt.InterruptEvent.ID,
			ExecuteID:  resumeInfo.ChatflowInterrupt.ExecuteID,
		}, executeConfig)
	} else {
		executeConfig.ConversationID = &art.GetRunMeta().ConversationID
		executeConfig.SectionID = &art.GetRunMeta().SectionID
		executeConfig.InitRoundID = &art.RunRecord.ID
		executeConfig.RoundID = &art.RunRecord.ID
		executeConfig.UserMessage = transMessageToSchemaMessage(ctx, []*msgEntity.Message{art.GetInput()}, imagex)[0]
		executeConfig.MaxHistoryRounds = ptr.Of(getAgentHistoryRounds(art.GetAgentInfo()))
		wfStreamer, err = crossworkflow.DefaultSVC().StreamExecute(ctx, executeConfig, map[string]any{
			"USER_INPUT": concatWfInput(art),
		})
	}
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	safego.Go(ctx, func() {
		defer wg.Done()
		art.pullWfStream(ctx, wfStreamer, mh)
	})
	wg.Wait()
	return err
}

func concatWfInput(rtDependence *AgentRuntime) string {
	var input string
	for _, content := range rtDependence.RunMeta.Content {
		if content.Type == message.InputTypeText {
			input = content.Text + "," + input
		} else {
			for _, file := range content.FileData {
				input += file.Url + ","
			}
		}
	}
	return input
}

func (art *AgentRuntime) pullWfStream(ctx context.Context, events *schema.StreamReader[*crossworkflow.WorkflowMessage], mh *MessageEventHandler) {

	fullAnswerContent := bytes.NewBuffer([]byte{})
	var usage *msgEntity.UsageExt

	preAnswerMsg, cErr := preCreateAnswer(ctx, art)

	if cErr != nil {
		return
	}

	var preMsgIsFinish = false
	var lastAnswerMsg *entity.ChunkMessageItem

	for {
		st, re := events.Recv()
		if re != nil {
			if errors.Is(re, io.EOF) {

				if lastAnswerMsg != nil && usage != nil {
					art.SetUsage(&agentrun.Usage{
						LlmPromptTokens:     usage.InputTokens,
						LlmCompletionTokens: usage.OutputTokens,
						LlmTotalTokens:      usage.TotalCount,
					})
					_ = mh.handlerWfUsage(ctx, lastAnswerMsg, usage)
				}
				// 记录最终输出内容，供可观测性 trace span 使用
				if lastAnswerMsg != nil {
					art.OutputContent = lastAnswerMsg.Content
				}

				finishErr := mh.handlerFinalAnswerFinish(ctx, art)
				if finishErr != nil {
					logs.CtxErrorf(ctx, "handlerFinalAnswerFinish error: %v", finishErr)
					return
				}
				return
			}
			logs.CtxErrorf(ctx, "pullWfStream Recv error: %v", re)
			mh.handlerErr(ctx, re)
			return
		}
		if st == nil {
			continue
		}
		if st.StateMessage != nil {
			if st.StateMessage.Status == crossworkflow.WorkflowFailed {
				mh.handlerErr(ctx, st.StateMessage.LastError)
				continue
			}
			if st.StateMessage.Usage != nil {
				usage = &msgEntity.UsageExt{
					InputTokens:  st.StateMessage.Usage.InputTokens,
					OutputTokens: st.StateMessage.Usage.OutputTokens,
					TotalCount:   st.StateMessage.Usage.InputTokens + st.StateMessage.Usage.OutputTokens,
				}
			}

			if st.StateMessage.InterruptEvent != nil { // interrupt
				mh.handlerWfInterruptMsg(ctx, st.StateMessage, art)
				continue
			}

		}

		if st.DataMessage == nil {
			continue
		}

		switch st.DataMessage.Type {
		case crossworkflow.Answer:

			// input node & question node skip
			if st.DataMessage != nil && (st.DataMessage.NodeType == crossworkflow.NodeTypeInputReceiver || st.DataMessage.NodeType == crossworkflow.NodeTypeQuestion) {
				break
			}

			if preMsgIsFinish {
				preAnswerMsg, cErr = preCreateAnswer(ctx, art)
				if cErr != nil {
					return
				}
				preMsgIsFinish = false
			}
			if st.DataMessage.Content != "" {
				fullAnswerContent.WriteString(st.DataMessage.Content)
			}
			if st.DataMessage.Last {
				preMsgIsFinish = true
				sendAnswerMsg := buildSendMsg(ctx, preAnswerMsg, false, art)
				sendAnswerMsg.Content = fullAnswerContent.String()
				fullAnswerContent.Reset()
				hfErr := mh.handlerAnswer(ctx, sendAnswerMsg, usage, art, preAnswerMsg)
				if hfErr != nil {
					return
				}
				lastAnswerMsg = sendAnswerMsg
			}
			sendAnswerMsg := buildSendMsg(ctx, preAnswerMsg, false, art)
			sendAnswerMsg.Content = st.DataMessage.Content

			mh.messageEvent.SendMsgEvent(entity.RunEventMessageDelta, sendAnswerMsg, mh.sw)
		}
	}
}
