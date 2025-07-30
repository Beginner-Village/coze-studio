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
	"sync/atomic"

	"github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type CreateMessageConfig struct {
	Creator conversation.ConversationManager
}
type CreateMessage struct {
	config *CreateMessageConfig
}

func NewCreateMessage(_ context.Context, cfg *CreateMessageConfig) (*CreateMessage, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Creator == nil {
		return nil, errors.New("creator is required")
	}
	return &CreateMessage{
		config: cfg,
	}, nil
}

func (c *CreateMessage) getConversationIDByName(ctx context.Context, env vo.Env, appID *int64, version, conversationName string, userID, connectorID int64) (int64, error) {
	template, isExist, err := workflow.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})
	if err != nil {
		return 0, vo.WrapError(errno.ErrConversationNodeInvalidOperation, err)
	}

	conversationIDGenerator := workflow.ConversationIDGenerator(func(ctx context.Context, appID int64, userID, connectorID int64) (int64, error) {
		return c.config.Creator.CreateConversation(ctx, &conversation.CreateConversationRequest{
			AppID:       appID,
			UserID:      userID,
			ConnectorID: connectorID,
		})
	})

	var conversationID int64
	if isExist {
		cID, _, err := workflow.GetRepository().GetOrCreateStaticConversation(ctx, env, conversationIDGenerator, &vo.CreateStaticConversation{
			AppID:       ptr.From(appID),
			TemplateID:  template.TemplateID,
			UserID:      userID,
			ConnectorID: connectorID,
		})
		if err != nil {
			return 0, err
		}
		conversationID = cID
	} else {
		dc, _, err := workflow.GetRepository().GetDynamicConversationByName(ctx, env, *appID, connectorID, userID, conversationName)
		if err != nil {
			return 0, err
		}
		if dc != nil {
			conversationID = dc.ConversationID
		}
	}
	return conversationID, nil
}

func (c *CreateMessage) Create(ctx context.Context, input map[string]any) (map[string]any, error) {
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
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("conversationName is required"))
	}

	role, ok := input["role"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("role is required"))
	}
	if role != "user" && role != "assistant" {
		return nil, vo.WrapError(errno.ErrInvalidParameter, fmt.Errorf("role must be user or assistant"))
	}

	content, ok := input["content"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrConversationNodeInvalidOperation, errors.New("content is required"))
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
				"isSuccess": false,
				"message": map[string]any{
					"messageId":   "0",
					"role":        role,
					"contentType": "text",
					"content":     content,
				},
			}, nil
		}
		conversationID = *execCtx.ExeCfg.ConversationID
		resolvedAppID = *agentID
	} else {
		conversationID, err = c.getConversationIDByName(ctx, env, appID, version, conversationName, userID, connectorID)
		if err != nil {
			return nil, err
		}
		resolvedAppID = *appID
	}

	if conversationID == 0 {
		return map[string]any{
			"isSuccess": false,
			"message": map[string]any{
				"messageId":   "0",
				"role":        role,
				"contentType": "text",
				"content":     content,
			},
		}, nil
	}

	currentConversationID := execCtx.ExeCfg.ConversationID
	isCurrentConversation := currentConversationID != nil && *currentConversationID == conversationID
	var runID int64

	if role == "user" {
		// For user messages, always create a new run and store the ID in the context.
		newRunID, err := workflow.GetRepository().GenID(ctx)
		if err != nil {
			return nil, err
		}
		if execCtx.ExeCfg.RoundID != nil {
			atomic.StoreInt64(execCtx.ExeCfg.RoundID, newRunID)
		}
		runID = newRunID
	} else if isCurrentConversation {
		// For assistant messages in the same conversation, reuse the runID from the context.
		if execCtx.ExeCfg.RoundID == nil {
			// This indicates an inconsistent state, as a user message should have set this.
			return map[string]any{
				"isSuccess": false,
				"message": map[string]any{
					"messageId":   "0",
					"role":        role,
					"contentType": "text",
					"content":     content,
				},
			}, nil
		}
		runID = *execCtx.ExeCfg.RoundID
	} else {
		// For assistant messages in a different conversation or a new workflow run,
		// find the latest runID or create a new one as a fallback.
		runIDs, err := c.config.Creator.GetLatestRunIDs(ctx, &conversation.GetLatestRunIDsRequest{
			ConversationID: conversationID,
			UserID:         userID,
			AppID:          resolvedAppID,
			Rounds:         1,
		})
		if err != nil {
			return nil, err
		}
		if len(runIDs) > 0 && runIDs[0] != 0 {
			runID = runIDs[0]
		} else {
			newRunID, err := workflow.GetRepository().GenID(ctx)
			if err != nil {
				return nil, err
			}
			runID = newRunID
		}
	}

	mID, err := c.config.Creator.CreateMessage(ctx, &conversation.CreateMessageRequest{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		ContentType:    "text",
		UserID:         userID,
		AppID:          resolvedAppID,
		RunID:          runID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	messageOutput := map[string]any{
		"messageId":   mID,
		"role":        role,
		"contentType": "text",
		"content":     content,
	}

	return map[string]any{
		"isSuccess": true,
		"message":   messageOutput,
	}, nil
}
