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
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	aiproduct "github.com/ynet-dev/ynet-studio/backend/application/aiproduct"
)

// superAgentPublishAgentAppRequest is the request body for the publish endpoint.
type superAgentPublishAgentAppRequest struct {
	AgentID int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	SpaceID int64  `form:"space_id" json:"space_id,string,omitempty"`
	User    string `form:"user_id" json:"user_id,omitempty"`
	Name    string `form:"name" json:"name,omitempty"`
	Version string `form:"version" json:"version,omitempty"`
}

// superAgentBuildStatusRequest is the request body for the build-status endpoint.
type superAgentBuildStatusRequest struct {
	ProductID int64  `form:"product_id" json:"product_id,string,omitempty"`
	SpaceID   int64  `form:"space_id" json:"space_id,string,omitempty"`
	Version   string `form:"version" json:"version,omitempty"`
	User      string `form:"user_id" json:"user_id,omitempty"`
}

// superAgentRecruitAgentAppRequest is the request body for the recruit endpoint.
type superAgentRecruitAgentAppRequest struct {
	ProductID int64  `form:"product_id" json:"product_id,string,omitempty"`
	SpaceID   int64  `form:"space_id" json:"space_id,string,omitempty"`
	User      string `form:"user_id" json:"user_id,omitempty"`
}

type superAgentPublishAgentAppResponse struct {
	Code int                          `json:"code"`
	Msg  string                       `json:"msg"`
	Data superAgentPublishAgentAppData `json:"data"`
}

type superAgentPublishAgentAppData struct {
	ProductID string `json:"product_id"`
	Version   string `json:"version"`
}

type superAgentBuildStatusResponse struct {
	Code int                       `json:"code"`
	Msg  string                    `json:"msg"`
	Data superAgentBuildStatusData `json:"data"`
}

type superAgentBuildStatusData struct {
	ProductID   string `json:"product_id"`
	Version     string `json:"version"`
	BuildStatus string `json:"build_status"`
}

type superAgentRecruitAgentAppResponse struct {
	Code int                           `json:"code"`
	Msg  string                        `json:"msg"`
	Data superAgentRecruitAgentAppData `json:"data"`
}

type superAgentRecruitAgentAppData struct {
	ShadowAgentID string `json:"shadow_agent_id"`
}

// SuperAgentPublishAgentApp publishes a super-agent as an agent_app AI product.
// The caller (user_id, space_id) is resolved from the request context; the body
// user_id field is accepted only as a fallback for API-key callers — it is NEVER
// used for permission decisions independently of context resolution.
//
// @router /api/super-agent/agent-app/publish [POST]
func SuperAgentPublishAgentApp(ctx context.Context, c *app.RequestContext) {
	var req superAgentPublishAgentAppRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.AgentID <= 0 {
		invalidParamRequestResponse(c, "agent_id is required")
		return
	}
	if req.SpaceID <= 0 {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}
	// Resolve caller from context (session/API-key), fall back to explicit user_id.
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	svc := aiproduct.AgentAppSVC
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("agent_app service is not initialized (sandbox disabled)"))
		return
	}
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("agent-%d", req.AgentID)
	}
	version := req.Version
	if version == "" {
		version = "v1.0.0"
	}
	productID, publishedVersion, err := svc.PublishAgentApp(ctx, aiproduct.PublishReq{
		AgentID: req.AgentID,
		SpaceID: req.SpaceID,
		UserID:  userID,
		Name:    name,
		Version: version,
	})
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentPublishAgentAppResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentPublishAgentAppData{
			ProductID: fmt.Sprintf("%d", productID),
			Version:   publishedVersion,
		},
	})
}

// SuperAgentAgentAppBuildStatus returns the sandbox template build status for a
// published agent_app product version.
//
// @router /api/super-agent/agent-app/build-status [POST]
func SuperAgentAgentAppBuildStatus(ctx context.Context, c *app.RequestContext) {
	var req superAgentBuildStatusRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ProductID <= 0 {
		invalidParamRequestResponse(c, "product_id is required")
		return
	}
	// Resolve caller from context to authenticate the request.
	if _, err := resolveSuperAgentSessionUserID(ctx, req.User); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	// Guard: AgentAppSVC may be nil when sandbox is disabled.
	if aiproduct.AgentAppSVC == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("agent_app service is not initialized (sandbox disabled)"))
		return
	}
	// Use the shared product application service to look up the product.
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	product, err := svc.GetVisibleProduct(ctx, req.ProductID, req.SpaceID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	if product == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("product not found: %d", req.ProductID))
		return
	}
	buildStatus := "unknown"
	if product.Feature != nil {
		if templateRaw, ok := product.Feature["template"]; ok {
			if templateMap, ok := templateRaw.(map[string]any); ok {
				if s, ok := templateMap["build_status"].(string); ok && s != "" {
					buildStatus = s
				}
			}
		}
	}
	c.JSON(http.StatusOK, &superAgentBuildStatusResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentBuildStatusData{
			ProductID:   fmt.Sprintf("%d", product.ProductID),
			Version:     req.Version,
			BuildStatus: buildStatus,
		},
	})
}

// SuperAgentRecruitAgentApp installs a published agent_app product into a space
// and materialises a read-only shadow agent draft in that space.
//
// @router /api/super-agent/agent-app/recruit [POST]
func SuperAgentRecruitAgentApp(ctx context.Context, c *app.RequestContext) {
	var req superAgentRecruitAgentAppRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ProductID <= 0 {
		invalidParamRequestResponse(c, "product_id is required")
		return
	}
	if req.SpaceID <= 0 {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}
	// Resolve caller from context (session/API-key), fall back to explicit user_id.
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	svc := aiproduct.AgentAppSVC
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("agent_app service is not initialized (sandbox disabled)"))
		return
	}
	shadowAgentID, err := svc.RecruitAgentApp(ctx, req.ProductID, req.SpaceID, userID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentRecruitAgentAppResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentRecruitAgentAppData{
			ShadowAgentID: fmt.Sprintf("%d", shadowAgentID),
		},
	})
}
