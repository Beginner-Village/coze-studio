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

	"github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type CreateConversationConfig struct {
	Manager conversation.ConversationManager
}

type CreateConversation struct {
	config *CreateConversationConfig
}

func NewCreateConversation(_ context.Context, cfg *CreateConversationConfig) (*CreateConversation, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Manager == nil {
		return nil, errors.New("manager is required")
	}
	return &CreateConversation{
		config: cfg,
	}, nil
}

func (c *CreateConversation) Create(ctx context.Context, input map[string]any) (map[string]any, error) {

	var (
		execCtx                 = execute.GetExeCtx(ctx)
		env                     = ternary.IFElse(execCtx.ExeCfg.Mode == vo.ExecuteModeRelease, vo.Online, vo.Draft)
		appID                   = execCtx.ExeCfg.AppID
		agentID                 = execCtx.ExeCfg.AgentID
		version                 = execCtx.ExeCfg.Version
		connectorID             = execCtx.ExeCfg.ConnectorID
		userID                  = execCtx.ExeCfg.Operator
		conversationIDGenerator = workflow.ConversationIDGenerator(func(ctx context.Context, appID int64, userID, connectorID int64) (int64, error) {
			return c.config.Manager.CreateConversation(ctx, &conversation.CreateConversationRequest{
				AppID:       appID,
				UserID:      userID,
				ConnectorID: connectorID,
			})
		})
	)
	if agentID != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, fmt.Errorf("in the agent scenario, create conversation is not available"))
	}

	if appID == nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, errors.New("create conversation node, app id is required"))
	}

	conversationName, ok := input["conversationName"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("conversation name is required"))
	}

	template, existed, err := workflow.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})
	if err != nil {
		return nil, err
	}

	if existed {
		cID, existed, err := workflow.GetRepository().GetOrCreateStaticConversation(ctx, env, conversationIDGenerator, &vo.CreateStaticConversation{
			AppID:       ptr.From(appID),
			TemplateID:  template.TemplateID,
			UserID:      userID,
			ConnectorID: connectorID,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"isSuccess":      true,
			"conversationId": cID,
			"isExisted":      existed,
		}, nil
	}

	cID, existed, err := workflow.GetRepository().GetOrCreateDynamicConversation(ctx, env, conversationIDGenerator, &vo.CreateDynamicConversation{
		AppID:       ptr.From(appID),
		UserID:      userID,
		ConnectorID: connectorID,
		Name:        conversationName,
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"isSuccess":      true,
		"conversationId": cID,
		"isExisted":      existed,
	}, nil

}
