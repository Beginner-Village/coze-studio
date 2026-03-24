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

package space

import (
	"context"
	"io"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	spaceModel "github.com/ynet-dev/ynet-studio/backend/api/model/space"
	spaceApp "github.com/ynet-dev/ynet-studio/backend/application/space"
	spaceexport "github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

// SyncExport exports a space for sync
// @router /api/space/{space_id}/sync/export [POST]
func SyncExport(ctx context.Context, c *app.RequestContext) {
	var req spaceModel.SyncExportRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	result, err := spaceApp.SyncSVC.ExportSync(ctx, &spaceexport.SyncExportRequest{
		SpaceID: req.SpaceID,
		Mode:    req.Mode,
		SinceTime: req.SinceTime,
	})
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, &spaceModel.SyncExportResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.SyncExportData{
			DownloadURL: result.DownloadURL,
			FileName:    result.FileName,
			FileSize:    result.FileSize,
			ExpiresAt:   result.ExpiresAt.UnixMilli(),
		},
	})
}

// SyncImportPreview previews a sync import operation
// @router /api/space/{space_id}/sync/import/preview [POST]
func SyncImportPreview(ctx context.Context, c *app.RequestContext) {
	spaceIDStr := c.Param("space_id")
	spaceID, err := strconv.ParseInt(spaceIDStr, 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.String(consts.StatusBadRequest, "file is required")
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.String(consts.StatusBadRequest, "failed to open uploaded file")
		return
	}
	defer f.Close()

	fileContent, err := io.ReadAll(f)
	if err != nil {
		c.String(consts.StatusBadRequest, "failed to read uploaded file")
		return
	}

	result, err := spaceApp.SyncSVC.ImportPreview(ctx, spaceID, 0, fileContent)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, &spaceModel.SyncImportPreviewResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.SyncImportPreviewData{
			ImportToken: result.ImportToken,
			Plan: &spaceModel.SyncPlan{
				Create: result.Plan.Create,
				Update: result.Plan.Update,
				Delete: result.Plan.Delete,
			},
			Warnings:       result.Warnings,
			TokenExpiresAt: result.TokenExpiresAt,
		},
	})
}

// SyncImportConfirm confirms and executes a sync import operation
// @router /api/space/{space_id}/sync/import/confirm [POST]
func SyncImportConfirm(ctx context.Context, c *app.RequestContext) {
	var req spaceModel.SyncImportConfirmRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	result, err := spaceApp.SyncSVC.ImportConfirm(ctx, req.SpaceID, 0, req.ImportToken)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	var plan *spaceModel.SyncPlan
	if result.Plan != nil {
		plan = &spaceModel.SyncPlan{
			Create: result.Plan.Create,
			Update: result.Plan.Update,
			Delete: result.Plan.Delete,
		}
	}

	c.JSON(consts.StatusOK, &spaceModel.SyncImportConfirmResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.SyncImportConfirmData{
			AgentsCreated:    result.AgentsCreated,
			PluginsCreated:   result.PluginsCreated,
			WorkflowsCreated: result.WorkflowsCreated,
			VariablesCreated: result.VariablesCreated,
			KnowledgeCreated: result.KnowledgeCreated,
			Plan:             plan,
		},
	})
}

// SyncLastExport gets the last sync export info for a space
// @router /api/space/{space_id}/sync/last-export [GET]
func SyncLastExport(ctx context.Context, c *app.RequestContext) {
	spaceIDStr := c.Param("space_id")
	spaceID, err := strconv.ParseInt(spaceIDStr, 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}

	record, err := spaceApp.SyncSVC.GetLastExport(ctx, spaceID)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	resp := &spaceModel.SyncLastExportResponse{
		Code: 0,
		Msg:  "success",
	}

	if record != nil {
		resp.Data = &spaceModel.SyncLastExportData{
			ExportTime: record.ExportTime,
			SyncType:   record.SyncType,
		}
	}

	c.JSON(consts.StatusOK, resp)
}

// SyncHistory gets the sync history for a space
// @router /api/space/{space_id}/sync/history [GET]
func SyncHistory(ctx context.Context, c *app.RequestContext) {
	spaceIDStr := c.Param("space_id")
	spaceID, err := strconv.ParseInt(spaceIDStr, 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}

	records, err := spaceApp.SyncSVC.GetHistory(ctx, spaceID)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	items := make([]*spaceModel.SyncHistoryItem, 0, len(records))
	for _, r := range records {
		items = append(items, &spaceModel.SyncHistoryItem{
			ID:            r.ID,
			SourceSpaceID: r.SourceSpaceID,
			TargetSpaceID: r.TargetSpaceID,
			SyncType:      r.SyncType,
			ExportTime:    r.ExportTime,
			ImportTime:    r.ImportTime,
			Status:        r.Status,
		})
	}

	c.JSON(consts.StatusOK, &spaceModel.SyncHistoryResponse{
		Code: 0,
		Msg:  "success",
		Data: items,
	})
}
