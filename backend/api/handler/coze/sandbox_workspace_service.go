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

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	developer_api "github.com/ynet-dev/ynet-studio/backend/api/model/app/developer_api"
	application "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
)

// ListSandboxFiles .
// @router /api/draftbot/sandbox/list [POST]
func ListSandboxFiles(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ListSandboxFilesRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ListSandboxFiles(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// ReadSandboxFile .
// @router /api/draftbot/sandbox/read [POST]
func ReadSandboxFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.ReadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.ReadSandboxFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// UploadSandboxFile .
// @router /api/draftbot/sandbox/upload [POST]
func UploadSandboxFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.UploadSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.UploadSandboxFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// DeleteSandboxFile .
// @router /api/draftbot/sandbox/delete [POST]
func DeleteSandboxFile(ctx context.Context, c *app.RequestContext) {
	var req developer_api.DeleteSandboxFileRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := application.SingleAgentSVC.DeleteSandboxFile(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}
