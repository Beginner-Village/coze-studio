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

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze/superagenttrace"
	developer_api "github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	msgEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const superAgentTraceMaxPageSize int32 = 100

type superAgentTraceContract struct {
	Events      []string `json:"events"`
	MaxPageSize int32    `json:"max_page_size"`
}

type superAgentTraceGetRequest struct {
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	RunID          *int64 `form:"run_id" json:"run_id,string,omitempty"`
	Limit          int32  `form:"limit" json:"limit,omitempty"`
}

type superAgentTraceGetResponse struct {
	Code int                   `json:"code"`
	Msg  string                `json:"msg"`
	Data *superagenttrace.Data `json:"data"`
}

// SuperAgentGetTrace exposes a replayable historical trace for an App Server conversation.
// @router /api/super-agent/traces/get [POST]
func SuperAgentGetTrace(ctx context.Context, c *app.RequestContext) {
	var req superAgentTraceGetRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}

	data, err := buildSuperAgentTraceData(ctx, req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, &superAgentTraceGetResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func buildSuperAgentTraceData(ctx context.Context, req superAgentTraceGetRequest) (*superagenttrace.Data, error) {
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

	limit := req.Limit
	if limit <= 0 || limit > superAgentTraceMaxPageSize {
		limit = superAgentTraceMaxPageSize
	}
	runRecords, err := conversation.ConversationSVC.AgentRunDomainSVC.List(ctx, &agentrunEntity.ListRunRecordMeta{
		ConversationID: req.ConversationID,
		Limit:          limit,
		OrderBy:        "asc",
	})
	if err != nil {
		return nil, err
	}
	runRecords = filterSuperAgentTraceRuns(runRecords, req.RunID)
	runIDs := make([]int64, 0, len(runRecords))
	for _, runRecord := range runRecords {
		runIDs = append(runIDs, runRecord.ID)
	}
	messages := make([]*msgEntity.Message, 0)
	if len(runIDs) > 0 {
		messages, err = conversation.ConversationSVC.MessageDomainSVC.GetByRunIDs(ctx, req.ConversationID, runIDs)
		if err != nil {
			return nil, err
		}
	}

	data := superagenttrace.BuildData(req.ConversationID, runRecords, messages)
	if currentConversation.AgentID > 0 && application.SingleAgentSVC != nil {
		harnessState, harnessErr := application.SingleAgentSVC.GetSuperAgentHarnessState(ctx, &developer_api.SuperAgentHarnessStateRequest{
			AgentID:        currentConversation.AgentID,
			ConversationID: currentConversation.ID,
		})
		if harnessErr != nil {
			logs.CtxWarnf(ctx, "[buildSuperAgentTraceData] read harness state failed: %v", harnessErr)
		} else if harnessState != nil && harnessState.Data != nil {
			appendContextCompactedTraceEvent(data, harnessState.Data.Context, data.ConversationID, "")
			appendPlanUpdatedTraceEvent(data, harnessState.Data.Plan, data.ConversationID, "")
		}
	}

	return data, nil
}

func checkSuperAgentTracePermission(ctx context.Context, creatorID int64) error {
	if apiKeyInfo := ctxutil.GetApiAuthFromCtx(ctx); apiKeyInfo != nil {
		if apiKeyInfo.UserID == creatorID {
			return nil
		}
		return errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "permission denied"))
	}
	if userID := ctxutil.GetUIDFromCtx(ctx); userID != nil && *userID == creatorID {
		return nil
	}
	return errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "permission denied"))
}

func filterSuperAgentTraceRuns(runRecords []*agentrunEntity.RunRecordMeta, runID *int64) []*agentrunEntity.RunRecordMeta {
	if runID == nil {
		return runRecords
	}
	filtered := make([]*agentrunEntity.RunRecordMeta, 0, 1)
	for _, runRecord := range runRecords {
		if runRecord.ID == *runID {
			filtered = append(filtered, runRecord)
		}
	}
	return filtered
}
