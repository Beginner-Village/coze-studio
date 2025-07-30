package conversation

import (
	"context"
	"errors"
	"fmt"

	wf "github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

type ClearConversationHistoryConfig struct {
	Manager conversation.ConversationManager
}

type ClearConversationHistory struct {
	cfg *ClearConversationHistoryConfig
}

func NewClearConversationHistory(_ context.Context, cfg *ClearConversationHistoryConfig) (*ClearConversationHistory, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Manager == nil {
		return nil, errors.New("manager is required")
	}

	return &ClearConversationHistory{
		cfg: cfg,
	}, nil
}

func (c *ClearConversationHistory) Clear(ctx context.Context, in map[string]any) (map[string]any, error) {

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

	conversationName, ok := in["conversationName"].(string)
	if !ok {
		return nil, vo.WrapError(errno.ErrInvalidParameter, errors.New("conversation name is required"))
	}

	t, existed, err := wf.GetRepository().GetConversationTemplate(ctx, env, vo.GetConversationTemplatePolicy{
		AppID:   appID,
		Name:    ptr.Of(conversationName),
		Version: ptr.Of(version),
	})

	if err != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
	}
	var conversationID int64
	if existed {
		ret, existed, err := wf.GetRepository().GetStaticConversationByTemplateID(ctx, env, userID, connectorID, t.TemplateID)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}
		if existed {
			conversationID = ret.ConversationID
		}
	} else {
		ret, existed, err := wf.GetRepository().GetDynamicConversationByName(ctx, env, *appID, connectorID, userID, conversationName)
		if err != nil {
			return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
		}
		if existed {
			conversationID = ret.ConversationID
		}
	}

	if !existed {
		return map[string]any{
			"isSuccess": false,
		}, nil
	}

	err = c.cfg.Manager.ClearConversationHistory(ctx, &conversation.ClearConversationHistoryReq{
		ConversationID: conversationID,
	})
	if err != nil {
		return nil, vo.WrapError(errno.ErrConversationNodesNotAvailable, err)
	}
	return map[string]any{
		"isSuccess": true,
	}, nil

}
