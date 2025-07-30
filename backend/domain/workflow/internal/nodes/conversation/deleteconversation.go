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

	wf "github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type DeleteConversation struct {
}

func NewDeleteConversation(_ context.Context) *DeleteConversation {
	return &DeleteConversation{}
}

func (c *DeleteConversation) Delete(ctx context.Context, in map[string]any) (map[string]any, error) {

	var (
		execCtx     = execute.GetExeCtx(ctx)
		env         = ternary.IFElse(execCtx.ExeCfg.Mode == vo.ExecuteModeRelease, vo.Online, vo.Draft)
		appID       = execCtx.ExeCfg.AppID
		agentID     = execCtx.ExeCfg.AgentID
		version     = execCtx.ExeCfg.Version
		connectorID = execCtx.ExeCfg.ConnectorID
		userID      = execCtx.ExeCfg.Operator
	)

	if agentID != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, fmt.Errorf("in the agent scenario, delete conversation is not available"))
	}

	if appID == nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, errors.New("delete conversation node, app id is required"))
	}

	cName, ok := in["conversationName"]
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("conversation name is required"))
	}

	conversationName := cName.(string)

	_, existed, err := wf.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})
	if err != nil {
		return nil, err
	}

	if existed {
		return nil, vo.WrapError(errno.ErrConversationNodeInvalidOperation, fmt.Errorf("only conversation created through nodes are allowed to be modified or deleted"))
	}

	dyConversation, existed, err := wf.GetRepository().GetDynamicConversationByName(ctx, env, *appID, connectorID, userID, conversationName)
	if err != nil {
		return nil, err
	}

	if !existed {
		return nil, vo.WrapError(errno.ErrConversationOfAppNotFound, fmt.Errorf("the conversation name does not exist: '%v'", conversationName))
	}

	_, err = wf.GetRepository().DeleteDynamicConversation(ctx, env, dyConversation.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"isSuccess": true,
	}, nil
}
