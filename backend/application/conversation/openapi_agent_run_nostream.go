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

package conversation

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/run"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

// ChatV3NoStreamResponse 非流式响应结构
type ChatV3NoStreamResponse struct {
	Code           int                       `json:"code"`
	Msg            string                    `json:"msg"`
	ConversationID string                    `json:"conversation_id"`
	BotID          string                    `json:"bot_id"`
	ChatID         string                    `json:"chat_id"`       // run_id
	Status         string                    `json:"status"`        // created, in_progress, completed, failed
	Messages       []*run.ChatV3MessageDetail `json:"messages"`
	Usage          *ChatV3Usage              `json:"usage,omitempty"`
}

// ChatV3Usage token 使用统计
type ChatV3Usage struct {
	TokenCount       int `json:"token_count"`
	OutputCount      int `json:"output_count"`
	InputCount       int `json:"input_count"`
}

// OpenapiAgentRunNoStream 非流式 Agent 运行
func (a *OpenapiAgentRunApplication) OpenapiAgentRunNoStream(ctx context.Context, ar *run.ChatV3Request) (*ChatV3NoStreamResponse, error) {
	apiKeyInfo := ctxutil.GetApiAuthFromCtx(ctx)
	creatorID := apiKeyInfo.UserID
	connectorID := apiKeyInfo.ConnectorID

	if ptr.From(ar.ConnectorID) == consts.WebSDKConnectorID {
		connectorID = ptr.From(ar.ConnectorID)
	}

	agentInfo, caErr := a.checkAgent(ctx, ar, connectorID)
	if caErr != nil {
		logs.CtxErrorf(ctx, "checkAgent err:%v", caErr)
		return nil, caErr
	}

	conversationData, ccErr := a.checkConversation(ctx, ar, creatorID, connectorID)
	if ccErr != nil {
		logs.CtxErrorf(ctx, "checkConversation err:%v", ccErr)
		return nil, ccErr
	}

	spaceID := agentInfo.SpaceID
	arr, err := a.buildAgentRunRequest(ctx, ar, connectorID, spaceID, conversationData, agentInfo)
	if err != nil {
		logs.CtxErrorf(ctx, "buildAgentRunRequest err:%v", err)
		return nil, err
	}

	streamer, err := ConversationSVC.AgentRunDomainSVC.AgentRun(ctx, arr)
	if err != nil {
		return nil, err
	}

	// 收集所有响应
	return a.collectStreamResponse(ctx, streamer, conversationData, ar.BotID)
}

// collectStreamResponse 收集流式响应并转换为非流式响应
func (a *OpenapiAgentRunApplication) collectStreamResponse(
	ctx context.Context,
	streamer *schema.StreamReader[*entity.AgentRunResponse],
	conversationData *convEntity.Conversation,
	botID int64,
) (*ChatV3NoStreamResponse, error) {
	resp := &ChatV3NoStreamResponse{
		Code:           0,
		Msg:            "success",
		ConversationID: strconv.FormatInt(conversationData.ID, 10),
		BotID:          strconv.FormatInt(botID, 10),
		Status:         "in_progress",
		Messages:       make([]*run.ChatV3MessageDetail, 0),
	}

	// 用于累积 delta 消息的 map，key 为 message_id
	deltaMessages := make(map[string]*run.ChatV3MessageDetail)

	for {
		chunk, recvErr := streamer.Recv()
		logs.CtxInfof(ctx, "nostream chunk:%v, err:%v", conv.DebugJsonToStr(chunk), recvErr)

		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			return nil, recvErr
		}

		switch chunk.Event {
		case entity.RunEventError:
			resp.Code = int(chunk.Error.Code)
			resp.Msg = chunk.Error.Msg
			resp.Status = "failed"
			return resp, nil

		case entity.RunEventStreamDone:
			resp.Status = "completed"

		case entity.RunEventAck:
			if chunk.ChunkMessageItem != nil {
				resp.ChatID = strconv.FormatInt(chunk.ChunkMessageItem.RunID, 10)
			}

		case entity.RunEventCompleted:
			resp.Status = "completed"

		case entity.RunEventFailed:
			resp.Status = "failed"

		case entity.RunEventMessageDelta:
			// 累积 delta 消息
			if chunk.ChunkMessageItem != nil {
				msgID := strconv.FormatInt(chunk.ChunkMessageItem.ID, 10)

				// 跳过特殊消息
				if shouldSkipMessage(chunk.ChunkMessageItem) {
					continue
				}

				if existing, ok := deltaMessages[msgID]; ok {
					// 累积内容
					existing.Content += chunk.ChunkMessageItem.Content
					if chunk.ChunkMessageItem.ReasoningContent != nil && *chunk.ChunkMessageItem.ReasoningContent != "" {
						if existing.ReasoningContent == nil {
							existing.ReasoningContent = ptr.Of("")
						}
						*existing.ReasoningContent += *chunk.ChunkMessageItem.ReasoningContent
					}
				} else {
					// 新消息
					msg := buildNoStreamMessage(chunk.ChunkMessageItem)
					deltaMessages[msgID] = msg
				}
			}

		case entity.RunEventMessageCompleted:
			// 完整消息，替换或添加
			if chunk.ChunkMessageItem != nil {
				// 跳过特殊消息
				if shouldSkipMessage(chunk.ChunkMessageItem) {
					continue
				}

				msg := buildNoStreamMessage(chunk.ChunkMessageItem)
				msgID := msg.ID

				// 如果之前有 delta 消息，用完整消息替换
				if _, ok := deltaMessages[msgID]; ok {
					deltaMessages[msgID] = msg
				} else {
					// 直接添加完整消息
					resp.Messages = append(resp.Messages, msg)
				}
			}
		}
	}

	// 将累积的 delta 消息添加到响应中
	for _, msg := range deltaMessages {
		resp.Messages = append(resp.Messages, msg)
	}

	return resp, nil
}

// shouldSkipMessage 判断是否应该跳过该消息
func shouldSkipMessage(item *entity.ChunkMessageItem) bool {
	if item == nil {
		return true
	}

	// 跳过 THINKING- 前缀的消息
	if strings.HasPrefix(item.Content, "THINKING-") {
		return true
	}

	// 跳过包含 message_title 的中间消息（输出节点）
	if messageTitle, exists := item.Ext["message_title"]; exists && messageTitle != "" {
		return true
	}

	// 跳过卡片消息
	if isCardMessage(item.Content) {
		return true
	}

	return false
}

// buildNoStreamMessage 构建非流式消息
func buildNoStreamMessage(item *entity.ChunkMessageItem) *run.ChatV3MessageDetail {
	msg := &run.ChatV3MessageDetail{
		ID:             strconv.FormatInt(item.ID, 10),
		ConversationID: strconv.FormatInt(item.ConversationID, 10),
		BotID:          strconv.FormatInt(item.AgentID, 10),
		Role:           string(item.Role),
		Type:           string(item.MessageType),
		Content:        item.Content,
		ContentType:    string(item.ContentType),
		ChatID:         strconv.FormatInt(item.RunID, 10),
		CreatedAt:      ptr.Of(item.CreatedAt / 1000),
	}

	if item.ReasoningContent != nil && *item.ReasoningContent != "" {
		msg.ReasoningContent = item.ReasoningContent
	}

	// 复制 MetaData
	if len(item.Ext) > 0 {
		msg.MetaData = make(map[string]string)
		for k, v := range item.Ext {
			msg.MetaData[k] = v
		}
	}

	return msg
}
