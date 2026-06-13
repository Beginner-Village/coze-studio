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

package operationlog

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/ynet-dev/ynet-studio/backend/api/internal/httputil"
	model "github.com/ynet-dev/ynet-studio/backend/api/model/data/operationlog"
	appoplog "github.com/ynet-dev/ynet-studio/backend/application/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

// ListOperationLog queries a space's operation-audit log. Only the space
// Owner/Admin may call (permission is enforced in the application layer).
// @router /api/operation_log/list [POST]
func ListOperationLog(ctx context.Context, c *app.RequestContext) {
	var req model.ListRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	f := &entity.ListFilter{
		SpaceID:      req.SpaceID,
		OperatorID:   req.OperatorID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Keyword:      req.Keyword,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	items, total, err := appoplog.OperationLogApplicationSVC.ListOperationLogs(ctx, f)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	dtos := make([]model.LogItemDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, model.LogItemDTO{
			ID:           it.ID,
			OperatorID:   it.OperatorID,
			OperatorName: it.OperatorName,
			Module:       it.Module,
			ResourceType: it.ResourceType,
			ResourceID:   it.ResourceID,
			ResourceName: it.ResourceName,
			Action:       it.Action,
			Description:  it.Description,
			Status:       int32(it.Status),
			ClientIP:     it.ClientIP,
			CreatedAt:    it.CreatedAt,
		})
	}

	c.JSON(consts.StatusOK, &model.ListResponse{Code: 0, Msg: "success", Logs: dtos, Total: total})
}
