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

package rerank

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	appRerank "github.com/ynet-dev/ynet-studio/backend/application/rerank"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

var spaceRerankApp *appRerank.SpaceRerankApp

// InitSpaceRerankApp initializes the space rerank application
func InitSpaceRerankApp(app *appRerank.SpaceRerankApp) {
	spaceRerankApp = app
}

// Request/Response types

type CreateSpaceRerankRequest struct {
	SpaceID      string                 `json:"space_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type"` // openai
	Config       map[string]interface{} `json:"config"`
	SetAsDefault bool                   `json:"set_as_default,omitempty"`
}

type ListSpaceReranksRequest struct {
	SpaceID string `json:"space_id"`
}

type GetSpaceDefaultRerankRequest struct {
	SpaceID string `json:"space_id"`
}

type UpdateSpaceRerankRequest struct {
	SpaceID     string                 `json:"space_id"`
	RerankID    string                 `json:"rerank_id"`
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Type        *string                `json:"type,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

type DeleteSpaceRerankRequest struct {
	SpaceID  string `json:"space_id"`
	RerankID string `json:"rerank_id"`
}

type SetDefaultSpaceRerankRequest struct {
	SpaceID  string `json:"space_id"`
	RerankID string `json:"rerank_id"`
}

type EnableSpaceRerankRequest struct {
	SpaceID  string `json:"space_id"`
	RerankID string `json:"rerank_id"`
}

type DisableSpaceRerankRequest struct {
	SpaceID  string `json:"space_id"`
	RerankID string `json:"rerank_id"`
}

// Response wrapper
type Response struct {
	Code int64       `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func successResp(data interface{}) *Response {
	return &Response{
		Code: 0,
		Msg:  "success",
		Data: data,
	}
}

func errorResp(msg string) *Response {
	return &Response{
		Code: -1,
		Msg:  msg,
	}
}

// CreateSpaceRerank creates a new rerank configuration for a space
func CreateSpaceRerank(ctx context.Context, c *app.RequestContext) {
	var req CreateSpaceRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	userID := getUserIDFromContext(c)

	result, err := spaceRerankApp.CreateSpaceRerank(
		ctx, req.SpaceID, userID, req.Name, req.Description,
		req.Type, req.Config, req.SetAsDefault,
	)
	if err != nil {
		logs.CtxErrorf(ctx, "CreateSpaceRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// ListSpaceReranks lists all reranks for a space
func ListSpaceReranks(ctx context.Context, c *app.RequestContext) {
	var req ListSpaceReranksRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	result, err := spaceRerankApp.ListSpaceReranks(ctx, req.SpaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "ListSpaceReranks failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// GetSpaceDefaultRerank gets the default rerank for a space
func GetSpaceDefaultRerank(ctx context.Context, c *app.RequestContext) {
	var req GetSpaceDefaultRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	result, err := spaceRerankApp.GetSpaceDefaultRerank(ctx, req.SpaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "GetSpaceDefaultRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// UpdateSpaceRerank updates a rerank configuration
func UpdateSpaceRerank(ctx context.Context, c *app.RequestContext) {
	var req UpdateSpaceRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	result, err := spaceRerankApp.UpdateSpaceRerank(
		ctx, req.SpaceID, req.RerankID, req.Name, req.Description,
		req.Type, req.Config,
	)
	if err != nil {
		logs.CtxErrorf(ctx, "UpdateSpaceRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// DeleteSpaceRerank deletes a rerank configuration
func DeleteSpaceRerank(ctx context.Context, c *app.RequestContext) {
	var req DeleteSpaceRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceRerankApp.DeleteSpaceRerank(ctx, req.SpaceID, req.RerankID)
	if err != nil {
		logs.CtxErrorf(ctx, "DeleteSpaceRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// SetDefaultSpaceRerank sets a rerank as the default for a space
func SetDefaultSpaceRerank(ctx context.Context, c *app.RequestContext) {
	var req SetDefaultSpaceRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceRerankApp.SetDefaultSpaceRerank(ctx, req.SpaceID, req.RerankID)
	if err != nil {
		logs.CtxErrorf(ctx, "SetDefaultSpaceRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// EnableSpaceRerank enables a rerank configuration
func EnableSpaceRerank(ctx context.Context, c *app.RequestContext) {
	var req EnableSpaceRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceRerankApp.EnableSpaceRerank(ctx, req.SpaceID, req.RerankID)
	if err != nil {
		logs.CtxErrorf(ctx, "EnableSpaceRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// DisableSpaceRerank disables a rerank configuration
func DisableSpaceRerank(ctx context.Context, c *app.RequestContext) {
	var req DisableSpaceRerankRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceRerankApp.DisableSpaceRerank(ctx, req.SpaceID, req.RerankID)
	if err != nil {
		logs.CtxErrorf(ctx, "DisableSpaceRerank failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(c *app.RequestContext) uint64 {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint64); ok {
			return uid
		}
	}
	return 0
}
