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
	"mime"
	"net/http"
	"net/url"
	"path"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
)

// SuperAgentListArtifacts lists generated deliverables under /outputs.
// @router /api/super-agent/artifacts/list [POST]
func SuperAgentListArtifacts(ctx context.Context, c *app.RequestContext) {
	var req application.SuperAgentArtifactListRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ListSuperAgentArtifacts(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentDownloadArtifact downloads a generated deliverable under /outputs.
// @router /api/super-agent/artifacts/download [POST]
func SuperAgentDownloadArtifact(ctx context.Context, c *app.RequestContext) {
	var req application.SuperAgentArtifactDownloadRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	data, err := application.SingleAgentSVC.DownloadSuperAgentArtifact(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	filename := path.Base(data.Path)
	contentType := mime.TypeByExtension(path.Ext(data.Path))
	if contentType == "" && len(data.Content) > 0 {
		contentType = http.DetectContentType(data.Content)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Response.Header.Set("Content-Type", contentType)
	c.Response.Header.Set("Content-Length", fmt.Sprintf("%d", data.Size))
	c.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(filename)))
	c.SetStatusCode(http.StatusOK)
	c.Response.SetBody(data.Content)
}

// SuperAgentDeleteArtifact deletes a generated deliverable under /outputs.
// @router /api/super-agent/artifacts/delete [POST]
func SuperAgentDeleteArtifact(ctx context.Context, c *app.RequestContext) {
	var req application.SuperAgentArtifactDeleteRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.DeleteSuperAgentArtifact(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentMoveArtifact moves or renames a generated deliverable under /outputs.
// @router /api/super-agent/artifacts/move [POST]
func SuperAgentMoveArtifact(ctx context.Context, c *app.RequestContext) {
	var req application.SuperAgentArtifactMoveRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.MoveSuperAgentArtifact(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}
