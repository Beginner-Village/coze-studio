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

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/common"
	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/conversation"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	agentrun "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/service"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	conversationService "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/service"
	message "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/service"
	"github.com/ynet-dev/ynet-studio/backend/domain/shortcutcmd/service"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/slices"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

type ConversationApplicationService struct {
	appContext *ServiceComponents

	AgentRunDomainSVC     agentrun.Run
	ConversationDomainSVC conversationService.Conversation
	MessageDomainSVC      message.Message

	ShortcutDomainSVC service.ShortcutCmd
}

var ConversationSVC = new(ConversationApplicationService)

type OpenapiAgentRunApplication struct {
	appContext *ServiceComponents  // 添加appContext以访问ImageX服务

	ShortcutDomainSVC service.ShortcutCmd
}

var ConversationOpenAPISVC = new(OpenapiAgentRunApplication)

func (c *ConversationApplicationService) ClearHistory(ctx context.Context, req *conversation.ClearConversationHistoryRequest) (*conversation.ClearConversationHistoryResponse, error) {
	resp := new(conversation.ClearConversationHistoryResponse)

	conversationID := req.ConversationID

	// get conversation
	currentRes, err := c.ConversationDomainSVC.GetByID(ctx, conversationID)
	if err != nil {
		return resp, err
	}
	if currentRes == nil {
		return resp, errorx.New(errno.ErrConversationNotFound)
	}
	// check user
	userID := ctxutil.GetUIDFromCtx(ctx)
	if userID == nil || *userID != currentRes.CreatorID {
		return resp, errorx.New(errno.ErrConversationNotFound, errorx.KV("msg", "user not match"))
	}

	// 清除上下文 = 在「同一个会话」内开一个新分段(会话 ID 保持不变),而不是删掉会话再
	// 另建一个新会话。原先 Delete+Create 的写法会把旧会话标记 Deleted,但只回传
	// NewSectionID、不回传新会话 ID,前端仍持旧(已删)会话 ID → 之后所有发送/再清除都查
	// 不到会话(status=Normal 过滤)而失败,智能体被彻底卡死。改为新分段后会话 ID 始终有效。
	convRes, err := c.ConversationDomainSVC.NewConversationCtx(ctx, &entity.NewConversationCtxRequest{
		ID: conversationID,
	})
	if err != nil {
		return resp, err
	}
	resp.NewSectionID = convRes.SectionID
	return resp, nil
}

func (c *ConversationApplicationService) CreateSection(ctx context.Context, conversationID int64) (int64, error) {
	currentRes, err := c.ConversationDomainSVC.GetByID(ctx, conversationID)
	if err != nil {
		return 0, err
	}

	if currentRes == nil {
		return 0, errorx.New(errno.ErrConversationNotFound, errorx.KV("msg", "conversation not found"))
	}
	var userID int64
	if currentRes.ConnectorID == consts.CozeConnectorID {
		userID = ctxutil.MustGetUIDFromCtx(ctx)
	} else {
		userID = ctxutil.MustGetUIDFromApiAuthCtx(ctx)
	}

	if userID != currentRes.CreatorID {
		return 0, errorx.New(errno.ErrConversationNotFound, errorx.KV("msg", "user not match"))
	}

	convRes, err := c.ConversationDomainSVC.NewConversationCtx(ctx, &entity.NewConversationCtxRequest{
		ID: conversationID,
	})
	if err != nil {
		return 0, err
	}

	// 清理会话即用户「重开」的意图：主动释放可能残留的活跃 run 锁，避免上一条 run
	// 异常未释放时，用户清理后再发消息仍被「已有正在进行的请求」卡住。
	if c.AgentRunDomainSVC != nil {
		_ = c.AgentRunDomainSVC.ReleaseRunLock(ctx, conversationID)
	}

	return convRes.SectionID, nil
}

func (c *ConversationApplicationService) CreateConversation(ctx context.Context, agentID int64, connectorID int64) (*conversation.CreateConversationResponse, error) {
	resp := new(conversation.CreateConversationResponse)
	apiKeyInfo := ctxutil.GetApiAuthFromCtx(ctx)
	userID := apiKeyInfo.UserID
	if connectorID != consts.WebSDKConnectorID {
		connectorID = apiKeyInfo.ConnectorID
	}

	conversationData, err := c.ConversationDomainSVC.Create(ctx, &entity.CreateMeta{
		AgentID:     agentID,
		UserID:      userID,
		ConnectorID: connectorID,
		Scene:       common.Scene_SceneOpenApi,
	})
	if err != nil {
		return nil, err
	}
	resp.ConversationData = &conversation.ConversationData{
		Id:            conversationData.ID,
		LastSectionID: &conversationData.SectionID,
		ConnectorID:   &conversationData.ConnectorID,
		CreatedAt:     conversationData.CreatedAt / 1000,
	}
	return resp, nil
}

func (c *ConversationApplicationService) ListConversation(ctx context.Context, req *conversation.ListConversationsApiRequest) (*conversation.ListConversationsApiResponse, error) {

	resp := new(conversation.ListConversationsApiResponse)

	apiKeyInfo := ctxutil.GetApiAuthFromCtx(ctx)
	userID := apiKeyInfo.UserID
	connectorID := apiKeyInfo.ConnectorID

	if userID == 0 {
		return resp, errorx.New(errno.ErrConversationNotFound)
	}
	if ptr.From(req.ConnectorID) == consts.WebSDKConnectorID {
		connectorID = ptr.From(req.ConnectorID)
	}

	conversationDOList, hasMore, err := c.ConversationDomainSVC.List(ctx, &entity.ListMeta{
		UserID:      userID,
		AgentID:     req.GetBotID(),
		ConnectorID: connectorID,
		Scene:       common.Scene_SceneOpenApi,
		Page:        int(req.GetPageNum()),
		Limit:       int(req.GetPageSize()),
	})
	if err != nil {
		return resp, err
	}
	conversationData := slices.Transform(conversationDOList, func(conv *entity.Conversation) *conversation.ConversationData {
		return &conversation.ConversationData{
			Id:            conv.ID,
			LastSectionID: &conv.SectionID,
			ConnectorID:   &conv.ConnectorID,
			CreatedAt:     conv.CreatedAt / 1000,
		}
	})

	resp.Data = &conversation.ListConversationData{
		Conversations: conversationData,
		HasMore:       hasMore,
	}
	return resp, nil
}
