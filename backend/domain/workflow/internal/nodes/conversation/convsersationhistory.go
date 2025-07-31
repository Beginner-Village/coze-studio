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

	"github.com/coze-dev/coze-studio/backend/api/model/ocean/cloud/workflow"
	wf "github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type ConversationHistoryConfig struct {
	Manager conversation.ConversationManager
}

type ConversationHistory struct {
	cfg *ConversationHistoryConfig
}

func NewConversationHistory(_ context.Context, cfg *ConversationHistoryConfig) (*ConversationHistory, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Manager == nil {
		return nil, errors.New("manager is required")
	}
	return &ConversationHistory{
		cfg: cfg,
	}, nil
}

func (ch *ConversationHistory) HistoryMessages(ctx context.Context, input map[string]any) (map[string]any, error) {
	var (
		execCtx     = execute.GetExeCtx(ctx)
		env         = ternary.IFElse(execCtx.ExeCfg.Mode == vo.ExecuteModeRelease, vo.Online, vo.Draft)
		appID       = execCtx.ExeCfg.AppID
		agentID     = execCtx.ExeCfg.AgentID
		connectorID = execCtx.ExeCfg.ConnectorID
		userID      = execCtx.ExeCfg.Operator
		version     = execCtx.ExeCfg.Version
	)
	if agentID != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, fmt.Errorf("in the agent scenario, query conversation list is not available"))
	}
	if appID == nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, fmt.Errorf("query conversation list node, app id is required"))
	}

	conversationName, ok := input["conversationName"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("conversation name is required"))
	}

	rounds, ok := input["rounds"].(int64)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("rounds is required"))
	}

	template, existed, err := wf.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})

	if err != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
	}

	var conversationID int64
	if existed {
		sts, existed, err := wf.GetRepository().GetStaticConversationByTemplateID(ctx, env, userID, connectorID, template.TemplateID)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}
		if existed {
			conversationID = sts.ConversationID
		}

	} else {
		dyConversation, existed, err := wf.GetRepository().GetDynamicConversationByName(ctx, env, *appID, connectorID, userID, conversationName)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}
		if existed {
			conversationID = dyConversation.ConversationID
		}
	}

	if !existed {
		return nil, vo.WrapError(errno.ErrConversationOfAppNotFound, fmt.Errorf("the conversation name does not exist: '%v'", conversationName))
	}

	isChatFlow := execCtx.ExeCfg.WorkflowMode == workflow.WorkflowMode_ChatFlow
	if isChatFlow {
		rounds += 1
	}

	runIDs, err := ch.cfg.Manager.GetLatestRunIDs(ctx, &conversation.GetLatestRunIDsRequest{
		ConversationID: conversationID,
		UserID:         userID,
		AppID:          *appID,
		Rounds:         rounds,
	})

	if err != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
	}

	var messageList []any
	if len(runIDs) == 0 {
		return map[string]any{
			"messageList": messageList,
		}, nil
	}

	if isChatFlow {
		if len(runIDs) == 1 {
			return map[string]any{
				"messageList": messageList,
			}, nil
		}
		runIDs = runIDs[1:] // chatflow needs to filter out this session
	}

	response, err := ch.cfg.Manager.GetMessagesByRunIDs(ctx, &conversation.GetMessagesByRunIDsRequest{
		ConversationID: conversationID,
		RunIDs:         runIDs,
	})
	if err != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
	}

	for _, msg := range response.Messages {
		content, err := ConvertMessageToString(ctx, msg)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}
		messageList = append(messageList, map[string]any{
			"role":    msg.Role,
			"content": content,
		})
	}

	return map[string]any{
		"messageList": messageList,
	}, nil
}
