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

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
)

type superAgentToolConfigRequest struct {
	AgentID int64                            `json:"agent_id,string"`
	BotID   int64                            `json:"bot_id,string"`
	SpaceID int64                            `json:"space_id,string"`
	Config  *crossagent.SuperAgentToolConfig `json:"config"`
}

type superAgentToolConfigData struct {
	Config *crossagent.SuperAgentToolConfig `json:"config"`
}

type superAgentToolConfigResponse struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data superAgentToolConfigData `json:"data"`
}

func resolveSuperAgentConfigAgentID(req *superAgentToolConfigRequest) int64 {
	if req.AgentID > 0 {
		return req.AgentID
	}
	return req.BotID
}

// SuperAgentGetToolConfig returns the per-agent capability switches for a super-agent.
// A nil config means "all enabled" (default). @router /api/super-agent/config/get [POST]
func SuperAgentGetToolConfig(ctx context.Context, c *app.RequestContext) {
	var req superAgentToolConfigRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	agentID := resolveSuperAgentConfigAgentID(&req)
	if agentID <= 0 {
		invalidParamRequestResponse(c, "agent_id or bot_id is required")
		return
	}
	// 用登录态身份做属主校验（忽略请求体里的 user，避免越权 IDOR）。
	callerUserID, err := resolveSuperAgentSessionUserID(ctx, "")
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	cfg, err := application.SingleAgentSVC.GetSuperAgentToolConfig(ctx, agentID, callerUserID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentToolConfigResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentToolConfigData{Config: cfg},
	})
}

// SuperAgentUpdateToolConfig persists the per-agent capability switches.
// @router /api/super-agent/config/update [POST]
func SuperAgentUpdateToolConfig(ctx context.Context, c *app.RequestContext) {
	var req superAgentToolConfigRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	agentID := resolveSuperAgentConfigAgentID(&req)
	if agentID <= 0 {
		invalidParamRequestResponse(c, "agent_id or bot_id is required")
		return
	}
	callerUserID, err := resolveSuperAgentSessionUserID(ctx, "")
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if err := application.SingleAgentSVC.UpdateSuperAgentToolConfig(ctx, agentID, callerUserID, req.Config); err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentToolConfigResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentToolConfigData{Config: req.Config},
	})
}
