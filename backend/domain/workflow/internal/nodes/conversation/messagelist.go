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

package conversation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type MessageListConfig struct {
	Lister conversation.ConversationManager
}
type MessageList struct {
	config *MessageListConfig
}

func NewMessageList(_ context.Context, cfg *MessageListConfig) (*MessageList, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	if cfg.Lister == nil {
		return nil, errors.New("lister is required")
	}

	return &MessageList{
		config: cfg,
	}, nil

}

func (m *MessageList) getConversationIDByName(ctx context.Context, env vo.Env, appID *int64, version, conversationName string, userID, connectorID int64) (int64, error) {
	template, isExist, err := workflow.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})

	if err != nil {
		return 0, vo.WrapError(errno.ErrConversationNodeInvalidOperation, err)
	}

	var conversationID int64
	if isExist {
		sc, _, err := workflow.GetRepository().GetStaticConversationByTemplateID(ctx, env, userID, connectorID, template.TemplateID)
		if err != nil {
			return 0, vo.WrapError(errno.ErrConversationNodeInvalidOperation, err)
		}
		if sc != nil {
			conversationID = sc.ConversationID
		}
	} else {
		dc, _, err := workflow.GetRepository().GetDynamicConversationByName(ctx, env, *appID, connectorID, userID, conversationName)
		if err != nil {
			return 0, vo.WrapError(errno.ErrConversationNodeInvalidOperation, err)
		}
		if dc != nil {
			conversationID = dc.ConversationID
		}
	}

	return conversationID, nil
}

func (m *MessageList) List(ctx context.Context, input map[string]any) (map[string]any, error) {
	var (
		execCtx     = execute.GetExeCtx(ctx)
		env         = ternary.IFElse(execCtx.ExeCfg.Mode == vo.ExecuteModeRelease, vo.Online, vo.Draft)
		appID       = execCtx.ExeCfg.AppID
		agentID     = execCtx.ExeCfg.AgentID
		version     = execCtx.ExeCfg.Version
		connectorID = execCtx.ExeCfg.ConnectorID
		userID      = execCtx.ExeCfg.Operator
	)

	conversationName, ok := input["conversationName"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrConversationNodeInvalidOperation, errors.New("ConversationName is required"))
	}

	var conversationID int64
	var err error
	var resolvedAppID int64
	if appID == nil {
		if conversationName != "Default" {
			return nil, vo.WrapError(errno.ErrOnlyDefaultConversationAllowInAgentScenario, errors.New("conversation node only allow in application"))
		}
		if agentID == nil || execCtx.ExeCfg.ConversationID == nil {
			return map[string]any{
				"messageList": []any{},
				"firstId":     "0",
				"lastId":      "0",
				"hasMore":     false,
			}, nil
		}
		conversationID = *execCtx.ExeCfg.ConversationID
		resolvedAppID = *agentID
	} else {
		conversationID, err = m.getConversationIDByName(ctx, env, appID, version, conversationName, userID, connectorID)
		if err != nil {
			return nil, err
		}
		resolvedAppID = *appID
	}

	req := &conversation.MessageListRequest{
		UserID:         userID,
		AppID:          resolvedAppID,
		ConversationID: conversationID,
	}

	if req.ConversationID == 0 {
		return map[string]any{
			"messageList": []any{},
			"firstId":     "0",
			"lastId":      "0",
			"hasMore":     false,
		}, nil
	}

	limit, ok := input["Limit"].(int64)
	if ok {
		if limit > 0 && limit <= 50 {
			req.Limit = limit
		} else {
			req.Limit = 50
		}
	} else {
		req.Limit = 50
	}
	beforeID, ok := input["beforeId"].(string)
	if ok {

		req.BeforeID = &beforeID
	}
	afterID, ok := input["afterId"].(string)
	if ok {

		req.AfterID = &afterID
	}

	if beforeID != "" && afterID != "" {
		return nil, vo.WrapError(errno.ErrInvalidParameter, fmt.Errorf("BeforeID and AfterID cannot be set at the same time"))
	}

	ml, err := m.config.Lister.MessageList(ctx, req)
	if err != nil {
		return nil, err
	}

	var messageList []any
	for _, msg := range ml.Messages {
		content, err := convertMessageToString(ctx, msg)
		if err != nil {
			return nil, err
		}
		messageList = append(messageList, map[string]any{
			"messageId":   strconv.FormatInt(msg.ID, 10),
			"role":        msg.Role,
			"contentType": msg.ContentType,
			"content":     content,
		})
	}

	// TODO: After the List interface is updated, the firstId and lastId from the response can be returned directly without extra processing
	var firstId, lastId any = "0", "0"
	if len(messageList) > 0 {
		if firstMsg, ok := messageList[0].(map[string]any); ok {
			firstId = firstMsg["messageId"]
		}
		if lastMsg, ok := messageList[len(messageList)-1].(map[string]any); ok {
			lastId = lastMsg["messageId"]
		}
	}

	return map[string]any{
		"messageList": messageList,
		"firstId":     firstId,
		"lastId":      lastId,
		"hasMore":     ml.HasMore,
	}, nil

}

func convertMessageToString(ctx context.Context, msg *conversation.Message) (string, error) {
	if msg.MultiContent != nil {
		sb := strings.Builder{}
		for idx, m := range msg.MultiContent {
			if m.Uri != nil {
				url, err := workflow.GetRepository().GetObjectUrl(ctx, ptr.From(m.Uri))
				if err != nil {
					return "", err
				}
				sb.WriteString(url)

			} else if m.Text != nil {
				sb.WriteString(ptr.From(m.Text))
			}
			if idx < len(msg.MultiContent)-1 {
				sb.WriteString(",")
			}
		}
		return sb.String(), nil
	} else if msg.Text != nil {
		return ptr.From(msg.Text), nil
	} else {
		return "", vo.WrapError(errno.ErrInvalidParameter, errors.New("message is invalid"))
	}

}
