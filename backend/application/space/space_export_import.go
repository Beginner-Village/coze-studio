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
	"time"

	"gorm.io/gorm"

	spaceModel "github.com/ynet-dev/ynet-studio/backend/api/model/space"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	spaceexport "github.com/ynet-dev/ynet-studio/backend/application/space/export"
	spaceimport "github.com/ynet-dev/ynet-studio/backend/application/space/import"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// getUserIDFromContext safely gets user ID from context
func getUserIDFromContext(ctx context.Context) (int64, error) {
	userIDPtr := ctxutil.GetUIDFromCtx(ctx)
	if userIDPtr == nil {
		return 0, errorx.New(errno.ErrUserSessionInvalidateCode, errorx.KV("msg", "user not authenticated"))
	}
	return *userIDPtr, nil
}

// SpaceExportImportService provides space export and import functionality
var SpaceExportImportSVC *SpaceExportImportService

// SpaceExportImportService handles space export and import operations
type SpaceExportImportService struct {
	exporter *spaceexport.SpaceExporter
	importer *spaceimport.SpaceImporter
}

// InitSpaceExportImportService initializes the space export/import service
func InitSpaceExportImportService(db *gorm.DB, objectStorage storage.Storage, idGen idgen.IDGenerator, eventBus service.ResourceEventBus, projectEventBus service.ProjectEventBus) {
	SpaceExportImportSVC = &SpaceExportImportService{
		exporter: spaceexport.NewSpaceExporter(db, objectStorage),
		importer: spaceimport.NewSpaceImporter(db, idGen, eventBus, projectEventBus),
	}
}

// ExportSpace exports a space to a ZIP package
func (s *SpaceExportImportService) ExportSpace(ctx context.Context, req *spaceModel.ExportSpaceRequest) (*spaceModel.ExportSpaceResponse, error) {
	// Get current user ID from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	logs.CtxInfof(ctx, "ExportSpace request: space_id=%d, user_id=%d", req.SpaceID, userID)

	// Call exporter
	result, err := s.exporter.Export(ctx, &spaceexport.ExportRequest{
		SpaceID:    req.SpaceID,
		ExporterID: userID,
	})
	if err != nil {
		logs.CtxErrorf(ctx, "Export failed: %v", err)
		return nil, err
	}

	// Convert to response model
	resp := &spaceModel.ExportSpaceResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.ExportData{
			DownloadURL: result.DownloadURL,
			ExpiresAt:   result.ExpiresAt.UnixMilli(),
			FileName:    result.FileName,
			FileSize:    result.FileSize,
			Statistics: &spaceModel.ExportStatistics{
				Agents:    int32(result.Statistics.Agents),
				Plugins:   int32(result.Statistics.Plugins),
				Workflows: int32(result.Statistics.Workflows),
				Variables: int32(result.Statistics.Variables),
			},
		},
	}

	return resp, nil
}

// ImportPreview previews an import operation
func (s *SpaceExportImportService) ImportPreview(ctx context.Context, req *spaceModel.ImportPreviewRequest) (*spaceModel.ImportPreviewResponse, error) {
	// Get current user ID from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	logs.CtxInfof(ctx, "ImportPreview request: space_id=%d, user_id=%d", req.SpaceID, userID)

	// Call importer preview
	result, err := s.importer.Preview(ctx, &spaceimport.PreviewRequest{
		SpaceID:     req.SpaceID,
		UserID:      userID,
		FileContent: req.FileContent,
	})
	if err != nil {
		logs.CtxErrorf(ctx, "Import preview failed: %v", err)
		return nil, err
	}

	// Parse export time
	var exportedAt int64
	if exportTime, err := time.Parse(time.RFC3339, result.Manifest.ExportTime); err == nil {
		exportedAt = exportTime.UnixMilli()
	}

	// Convert to response model
	resp := &spaceModel.ImportPreviewResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.ImportPreviewData{
			ImportToken: result.ImportToken,
			Manifest: &spaceModel.ImportManifest{
				Version:         result.Manifest.Version,
				SourceSpaceName: result.Manifest.Source.SpaceName,
				ExportedAt:      exportedAt,
				Statistics: &spaceModel.ExportStatistics{
					Agents:    int32(result.Manifest.Statistics.Agents),
					Plugins:   int32(result.Manifest.Statistics.Plugins),
					Workflows: int32(result.Manifest.Statistics.Workflows),
					Variables: int32(result.Manifest.Statistics.Variables),
				},
			},
			Warnings:       result.Warnings,
			TokenExpiresAt: time.Now().Add(spaceimport.ImportTokenTTL).UnixMilli(),
		},
	}

	return resp, nil
}

// ImportConfirm confirms and executes an import operation
func (s *SpaceExportImportService) ImportConfirm(ctx context.Context, req *spaceModel.ImportConfirmRequest) (*spaceModel.ImportConfirmResponse, error) {
	// Get current user ID from context
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	logs.CtxInfof(ctx, "ImportConfirm request: space_id=%d, user_id=%d", req.SpaceID, userID)

	// Call importer confirm
	result, err := s.importer.Confirm(ctx, &spaceimport.ConfirmRequest{
		SpaceID:     req.SpaceID,
		UserID:      userID,
		ImportToken: req.ImportToken,
	})
	if err != nil {
		logs.CtxErrorf(ctx, "Import confirm failed: %v", err)
		return nil, err
	}

	// Convert to response model
	resp := &spaceModel.ImportConfirmResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.ImportConfirmData{
			Statistics: &spaceModel.ImportResultStatistics{
				AgentsCreated:    int32(result.AgentsCreated),
				PluginsCreated:   int32(result.PluginsCreated),
				WorkflowsCreated: int32(result.WorkflowsCreated),
				VariablesCreated: int32(result.VariablesCreated),
			},
			CreatedResources: make([]*spaceModel.ResourceReference, 0),
		},
	}

	return resp, nil
}
