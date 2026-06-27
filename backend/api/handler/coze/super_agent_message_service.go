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

package coze

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	messageModel "github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	messageEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const superAgentMessageMaxPageSize = 100

type superAgentListMessagesRequest struct {
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	Limit          int    `form:"limit" json:"limit,omitempty"`
	BeforeID       *int64 `form:"before_id" json:"before_id,string,omitempty"`
	AfterID        *int64 `form:"after_id" json:"after_id,string,omitempty"`
	OrderBy        string `form:"order_by" json:"order_by,omitempty"`
	User           string `form:"user_id" json:"user_id,omitempty"`
	ClientID       string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentListMessagesResponse struct {
	Code int                       `json:"code"`
	Msg  string                    `json:"msg"`
	Data superAgentMessageListData `json:"data"`
}

type superAgentMessageListData struct {
	ConversationID string                  `json:"conversation_id"`
	AgentID        string                  `json:"agent_id"`
	Messages       []superAgentMessageItem `json:"messages"`
	HasMore        bool                    `json:"has_more"`
	PrevCursor     string                  `json:"prev_cursor,omitempty"`
	NextCursor     string                  `json:"next_cursor,omitempty"`
	OrderBy        string                  `json:"order_by"`
}

type superAgentMessageItem struct {
	MessageID        string            `json:"message_id"`
	ConversationID   string            `json:"conversation_id"`
	RunID            string            `json:"run_id,omitempty"`
	AgentID          string            `json:"agent_id"`
	SectionID        string            `json:"section_id,omitempty"`
	Role             string            `json:"role"`
	Type             string            `json:"type"`
	Content          string            `json:"content"`
	ContentType      string            `json:"content_type"`
	DisplayContent   string            `json:"display_content,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	Status           int32             `json:"status"`
	Position         int32             `json:"position"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	CreatedAt        int64             `json:"created_at"`
	UpdatedAt        int64             `json:"updated_at"`
}

// SuperAgentListMessages returns stable session messages for App Server clients.
// @router /api/super-agent/messages/list [POST]
func SuperAgentListMessages(ctx context.Context, c *app.RequestContext) {
	var req superAgentListMessagesRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	resp, err := buildSuperAgentMessageList(ctx, req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentListMessagesResponse{
		Code: 0,
		Msg:  "success",
		Data: *resp,
	})
}

func buildSuperAgentMessageList(ctx context.Context, req superAgentListMessagesRequest) (*superAgentMessageListData, error) {
	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if currentConversation == nil {
		return nil, errorx.New(errno.ErrConversationNotFound)
	}
	if err := checkSuperAgentTracePermission(ctx, currentConversation.CreatorID); err != nil {
		return nil, err
	}

	limit := normalizeSuperAgentMessageLimit(req.Limit)
	orderBy := normalizeSuperAgentMessageOrder(req.OrderBy)
	listMeta := &messageEntity.ListMeta{
		ConversationID: req.ConversationID,
		AgentID:        currentConversation.AgentID,
		Limit:          limit,
		OrderBy:        &orderBy,
		Direction:      messageEntity.ScrollPageDirectionNext,
	}
	if req.BeforeID != nil && *req.BeforeID > 0 {
		listMeta.Cursor = *req.BeforeID
		listMeta.Direction = messageEntity.ScrollPageDirectionPrev
	} else if req.AfterID != nil && *req.AfterID > 0 {
		listMeta.Cursor = *req.AfterID
		listMeta.Direction = messageEntity.ScrollPageDirectionNext
	}

	listResult, err := conversation.ConversationSVC.MessageDomainSVC.ListWithoutPair(ctx, listMeta)
	if err != nil {
		return nil, err
	}
	messages := []*messageEntity.Message{}
	hasMore := false
	if listResult != nil {
		messages = append(messages, listResult.Messages...)
		hasMore = listResult.HasMore
	}
	sortSuperAgentMessages(messages, orderBy)

	items := make([]superAgentMessageItem, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		items = append(items, buildSuperAgentMessageItem(msg, currentConversation))
	}

	data := &superAgentMessageListData{
		ConversationID: strconv.FormatInt(currentConversation.ID, 10),
		AgentID:        strconv.FormatInt(currentConversation.AgentID, 10),
		Messages:       items,
		HasMore:        hasMore,
		OrderBy:        orderBy,
	}
	if len(items) > 0 {
		data.PrevCursor = items[0].MessageID
		data.NextCursor = items[len(items)-1].MessageID
	}
	return data, nil
}

func normalizeSuperAgentMessageLimit(limit int) int {
	if limit <= 0 || limit > superAgentMessageMaxPageSize {
		return superAgentMessageMaxPageSize
	}
	return limit
}

func normalizeSuperAgentMessageOrder(orderBy string) string {
	order := strings.ToUpper(strings.TrimSpace(orderBy))
	if order == messageModel.OrderByDesc {
		return messageModel.OrderByDesc
	}
	return messageModel.OrderByAsc
}

func sortSuperAgentMessages(messages []*messageEntity.Message, orderBy string) {
	sort.SliceStable(messages, func(i, j int) bool {
		left := messages[i]
		right := messages[j]
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.CreatedAt == right.CreatedAt {
			if orderBy == messageModel.OrderByDesc {
				return left.ID > right.ID
			}
			return left.ID < right.ID
		}
		if orderBy == messageModel.OrderByDesc {
			return left.CreatedAt > right.CreatedAt
		}
		return left.CreatedAt < right.CreatedAt
	})
}

func buildSuperAgentMessageItem(msg *messageEntity.Message, fallbackConversation *convEntity.Conversation) superAgentMessageItem {
	conversationID := msg.ConversationID
	if conversationID == 0 && fallbackConversation != nil {
		conversationID = fallbackConversation.ID
	}
	agentID := msg.AgentID
	if agentID == 0 && fallbackConversation != nil {
		agentID = fallbackConversation.AgentID
	}
	item := superAgentMessageItem{
		MessageID:        strconv.FormatInt(msg.ID, 10),
		ConversationID:   strconv.FormatInt(conversationID, 10),
		AgentID:          strconv.FormatInt(agentID, 10),
		Role:             string(msg.Role),
		Type:             string(msg.MessageType),
		Content:          msg.Content,
		ContentType:      string(msg.ContentType),
		DisplayContent:   msg.DisplayContent,
		ReasoningContent: msg.ReasoningContent,
		Status:           int32(msg.Status),
		Position:         msg.Position,
		Metadata:         msg.Ext,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
	}
	if msg.RunID > 0 {
		item.RunID = strconv.FormatInt(msg.RunID, 10)
	}
	if msg.SectionID > 0 {
		item.SectionID = strconv.FormatInt(msg.SectionID, 10)
	}
	return item
}
