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

package embedding

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	appEmbedding "github.com/coze-dev/coze-studio/backend/application/embedding"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

var spaceEmbeddingApp *appEmbedding.SpaceEmbeddingApp

// InitSpaceEmbeddingApp initializes the space embedding application
func InitSpaceEmbeddingApp(app *appEmbedding.SpaceEmbeddingApp) {
	spaceEmbeddingApp = app
}

// Request/Response types

type CreateSpaceEmbeddingRequest struct {
	SpaceID      string                 `json:"space_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type"` // openai, ark, ollama, http
	Config       map[string]interface{} `json:"config"`
	MaxBatchSize int                    `json:"max_batch_size,omitempty"`
	SetAsDefault bool                   `json:"set_as_default,omitempty"`
}

type ListSpaceEmbeddingsRequest struct {
	SpaceID string `json:"space_id"`
}

type GetSpaceDefaultEmbeddingRequest struct {
	SpaceID string `json:"space_id"`
}

type UpdateSpaceEmbeddingRequest struct {
	SpaceID      string                 `json:"space_id"`
	EmbeddingID  string                 `json:"embedding_id"`
	Name         *string                `json:"name,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Type         *string                `json:"type,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	MaxBatchSize *int                   `json:"max_batch_size,omitempty"`
}

type DeleteSpaceEmbeddingRequest struct {
	SpaceID     string `json:"space_id"`
	EmbeddingID string `json:"embedding_id"`
}

type SetDefaultSpaceEmbeddingRequest struct {
	SpaceID     string `json:"space_id"`
	EmbeddingID string `json:"embedding_id"`
}

type EnableSpaceEmbeddingRequest struct {
	SpaceID     string `json:"space_id"`
	EmbeddingID string `json:"embedding_id"`
}

type DisableSpaceEmbeddingRequest struct {
	SpaceID     string `json:"space_id"`
	EmbeddingID string `json:"embedding_id"`
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

// CreateSpaceEmbedding creates a new embedding configuration for a space
// @router /api/embedding/space/create [POST]
func CreateSpaceEmbedding(ctx context.Context, c *app.RequestContext) {
	var req CreateSpaceEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	// Get user ID from context
	userID := getUserIDFromContext(c)

	result, err := spaceEmbeddingApp.CreateSpaceEmbedding(
		ctx,
		req.SpaceID,
		userID,
		req.Name,
		req.Description,
		req.Type,
		req.Config,
		req.MaxBatchSize,
		req.SetAsDefault,
	)
	if err != nil {
		logs.CtxErrorf(ctx, "CreateSpaceEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// ListSpaceEmbeddings lists all embeddings for a space
// @router /api/embedding/space/list [POST]
func ListSpaceEmbeddings(ctx context.Context, c *app.RequestContext) {
	var req ListSpaceEmbeddingsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	result, err := spaceEmbeddingApp.ListSpaceEmbeddings(ctx, req.SpaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "ListSpaceEmbeddings failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// GetSpaceDefaultEmbedding gets the default embedding for a space
// @router /api/embedding/space/default [POST]
func GetSpaceDefaultEmbedding(ctx context.Context, c *app.RequestContext) {
	var req GetSpaceDefaultEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	result, err := spaceEmbeddingApp.GetSpaceDefaultEmbedding(ctx, req.SpaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "GetSpaceDefaultEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// UpdateSpaceEmbedding updates an embedding configuration
// @router /api/embedding/space/update [POST]
func UpdateSpaceEmbedding(ctx context.Context, c *app.RequestContext) {
	var req UpdateSpaceEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	result, err := spaceEmbeddingApp.UpdateSpaceEmbedding(
		ctx,
		req.SpaceID,
		req.EmbeddingID,
		req.Name,
		req.Description,
		req.Type,
		req.Config,
		req.MaxBatchSize,
	)
	if err != nil {
		logs.CtxErrorf(ctx, "UpdateSpaceEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(result))
}

// DeleteSpaceEmbedding deletes an embedding configuration
// @router /api/embedding/space/delete [POST]
func DeleteSpaceEmbedding(ctx context.Context, c *app.RequestContext) {
	var req DeleteSpaceEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceEmbeddingApp.DeleteSpaceEmbedding(ctx, req.SpaceID, req.EmbeddingID)
	if err != nil {
		logs.CtxErrorf(ctx, "DeleteSpaceEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// SetDefaultSpaceEmbedding sets an embedding as the default for a space
// @router /api/embedding/space/set-default [POST]
func SetDefaultSpaceEmbedding(ctx context.Context, c *app.RequestContext) {
	var req SetDefaultSpaceEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceEmbeddingApp.SetDefaultSpaceEmbedding(ctx, req.SpaceID, req.EmbeddingID)
	if err != nil {
		logs.CtxErrorf(ctx, "SetDefaultSpaceEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// EnableSpaceEmbedding enables an embedding configuration
// @router /api/embedding/space/enable [POST]
func EnableSpaceEmbedding(ctx context.Context, c *app.RequestContext) {
	var req EnableSpaceEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceEmbeddingApp.EnableSpaceEmbedding(ctx, req.SpaceID, req.EmbeddingID)
	if err != nil {
		logs.CtxErrorf(ctx, "EnableSpaceEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// DisableSpaceEmbedding disables an embedding configuration
// @router /api/embedding/space/disable [POST]
func DisableSpaceEmbedding(ctx context.Context, c *app.RequestContext) {
	var req DisableSpaceEmbeddingRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(consts.StatusBadRequest, errorResp(err.Error()))
		return
	}

	err := spaceEmbeddingApp.DisableSpaceEmbedding(ctx, req.SpaceID, req.EmbeddingID)
	if err != nil {
		logs.CtxErrorf(ctx, "DisableSpaceEmbedding failed: %v", err)
		c.JSON(consts.StatusOK, errorResp(err.Error()))
		return
	}

	c.JSON(consts.StatusOK, successResp(nil))
}

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(c *app.RequestContext) uint64 {
	// Try to get user ID from context or header
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint64); ok {
			return uid
		}
	}
	return 0
}
