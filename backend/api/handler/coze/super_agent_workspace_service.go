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

	developer_api "github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
)

// SuperAgentListWorkspaceFiles .
// @router /api/super-agent/workspace/list [POST]
func SuperAgentListWorkspaceFiles(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ListSandboxFilesRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ListSuperAgentWorkspaceFiles(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentReadWorkspaceFile .
// @router /api/super-agent/workspace/read [POST]
func SuperAgentReadWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ReadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ReadSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentDownloadWorkspaceFile .
// @router /api/super-agent/workspace/download [POST]
func SuperAgentDownloadWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.DownloadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	data, err := application.SingleAgentSVC.DownloadSuperAgentWorkspaceFile(ctx, &req)
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

// SuperAgentUploadWorkspaceFile .
// @router /api/super-agent/workspace/upload [POST]
func SuperAgentUploadWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.UploadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.UploadSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentWriteWorkspaceFile .
// @router /api/super-agent/workspace/write [POST]
func SuperAgentWriteWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	SuperAgentUploadWorkspaceFile(ctx, c)
}

// SuperAgentDeleteWorkspaceFile .
// @router /api/super-agent/workspace/delete [POST]
func SuperAgentDeleteWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.DeleteSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.DeleteSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentMoveWorkspaceFile .
// @router /api/super-agent/workspace/move [POST]
func SuperAgentMoveWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.MoveSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.MoveSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentCreateWorkspaceDirectory .
// @router /api/super-agent/workspace/mkdir [POST]
func SuperAgentCreateWorkspaceDirectory(ctx context.Context, c *app.RequestContext) {
	var req developer_api.CreateSandboxDirectoryRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.CreateSuperAgentWorkspaceDirectory(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentStatWorkspacePath .
// @router /api/super-agent/workspace/stat [POST]
func SuperAgentStatWorkspacePath(ctx context.Context, c *app.RequestContext) {
	var req developer_api.StatSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.StatSuperAgentWorkspacePath(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentGrepWorkspace .
// @router /api/super-agent/workspace/grep [POST]
func SuperAgentGrepWorkspace(ctx context.Context, c *app.RequestContext) {
	var req developer_api.GrepSandboxFilesRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.GrepSuperAgentWorkspace(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentGlobWorkspace .
// @router /api/super-agent/workspace/glob [POST]
func SuperAgentGlobWorkspace(ctx context.Context, c *app.RequestContext) {
	var req developer_api.GlobSandboxFilesRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.GlobSuperAgentWorkspace(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentEditWorkspaceFile .
// @router /api/super-agent/workspace/edit [POST]
func SuperAgentEditWorkspaceFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.EditSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.EditSuperAgentWorkspaceFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentApplyWorkspacePatch .
// @router /api/super-agent/workspace/patch [POST]
func SuperAgentApplyWorkspacePatch(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ApplySandboxPatchRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ApplySuperAgentWorkspacePatch(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// SuperAgentExecSandboxCommand .
// @router /api/super-agent/sandbox/exec [POST]
func SuperAgentExecSandboxCommand(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ExecSandboxCommandRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.RunSuperAgentSandboxCommand(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}
