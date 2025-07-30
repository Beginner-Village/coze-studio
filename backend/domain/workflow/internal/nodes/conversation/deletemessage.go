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

	wf "github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type DeleteMessageConfig struct {
	Manager conversation.ConversationManager
}

type DeleteMessage struct {
	config *DeleteMessageConfig
}

func NewDeleteMessage(_ context.Context, cfg *DeleteMessageConfig) (*DeleteMessage, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Manager == nil {
		return nil, errors.New("manager is required")
	}

	return &DeleteMessage{
		config: cfg,
	}, nil
}

func (c *DeleteMessage) Delete(ctx context.Context, input map[string]any) (map[string]any, error) {
	var (
		execCtx     = execute.GetExeCtx(ctx)
		env         = ternary.IFElse(execCtx.ExeCfg.Mode == vo.ExecuteModeRelease, vo.Online, vo.Draft)
		appID       = execCtx.ExeCfg.AppID
		agentID     = execCtx.ExeCfg.AgentID
		version     = execCtx.ExeCfg.Version
		connectorID = execCtx.ExeCfg.ConnectorID
		userID      = execCtx.ExeCfg.Operator

		successMap = map[string]any{
			"isSuccess": true,
		}
		failedMap = map[string]any{
			"isSuccess": false,
		}
	)

	conversationName, ok := input["conversationName"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("conversationName is required"))
	}
	messageStr, ok := input["messageId"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("messageId is required"))
	}
	messageID, err := strconv.ParseInt(messageStr, 10, 64)
	if err != nil {
		return nil, vo.WrapError(errno.ErrInvalidParameter, err)
	}

	if appID == nil {
		if conversationName != "Default" {
			return nil, vo.WrapError(errno.ErrOnlyDefaultConversationAllowInAgentScenario, fmt.Errorf("only default conversation allow in agent scenario"))
		}

		if agentID == nil || execCtx.ExeCfg.ConversationID == nil {
			return failedMap, nil
		}

		err = c.config.Manager.DeleteMessage(ctx, &conversation.DeleteMessageRequest{ConversationID: *execCtx.ExeCfg.ConversationID, MessageID: messageID})
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}

		return successMap, nil
	}

	t, existed, err := wf.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})
	if err != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
	}

	if existed {
		sts, existed, err := wf.GetRepository().GetStaticConversationByTemplateID(ctx, env, userID, connectorID, t.TemplateID)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}

		if !existed {
			return failedMap, nil
		}

		err = c.config.Manager.DeleteMessage(ctx, &conversation.DeleteMessageRequest{ConversationID: sts.ConversationID, MessageID: messageID})
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}

		return successMap, nil

	} else {
		dyConversation, existed, err := wf.GetRepository().GetDynamicConversationByName(ctx, env, *appID, connectorID, userID, conversationName)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}

		if !existed {
			return failedMap, nil
		}

		err = c.config.Manager.DeleteMessage(ctx, &conversation.DeleteMessageRequest{ConversationID: dyConversation.ConversationID, MessageID: messageID})
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}

		return successMap, nil

	}

}
