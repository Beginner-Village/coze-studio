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

package export

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	// ExportFileTTL is the time-to-live for exported files
	ExportFileTTL = 1 * time.Hour

	// ExportFilePrefix is the prefix for exported files in object storage
	ExportFilePrefix = "space_exports"

	// SyncExportFilePrefix is the prefix for sync export files in object storage
	SyncExportFilePrefix = "sync_exports"

	// SyncExportMaxSize is the maximum allowed size for a sync export ZIP (2GB)
	SyncExportMaxSize = 2 * 1024 * 1024 * 1024
)

// SpaceExporter handles the export of a space to a ZIP file
type SpaceExporter struct {
	db            *gorm.DB
	objectStorage storage.Storage
	collector     *ResourceCollector
	serializer    *Serializer
}

// NewSpaceExporter creates a new SpaceExporter
func NewSpaceExporter(db *gorm.DB, objectStorage storage.Storage) *SpaceExporter {
	return &SpaceExporter{
		db:            db,
		objectStorage: objectStorage,
		collector:     NewResourceCollector(db),
		serializer:    NewSerializer(),
	}
}

// ExportRequest represents a request to export a space
type ExportRequest struct {
	SpaceID    int64
	SpaceName  string
	ExporterID int64
}

// SyncExportRequest represents a request to export a space for sync
type SyncExportRequest struct {
	SpaceID        int64
	UserID         int64
	SpaceName      string
	Mode           string // "full" or "incremental"
	SinceTime      int64  // only used when Mode == "incremental"
	ReleaseVersion string // if set, embedded into manifest inside the ZIP
}

// Export exports a space to a ZIP file and returns the download URL
func (e *SpaceExporter) Export(ctx context.Context, req *ExportRequest) (*ExportResult, error) {
	logs.CtxInfof(ctx, "Starting space export for space_id=%d, exporter_id=%d", req.SpaceID, req.ExporterID)

	// Step 1: Collect all resources
	resources, err := e.collector.CollectAll(ctx, req.SpaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to collect resources for space %d: %v", req.SpaceID, err)
		return nil, err
	}

	// Check if space is empty
	if e.isSpaceEmpty(resources) {
		logs.CtxWarnf(ctx, "Space %d is empty, creating empty export", req.SpaceID)
	}

	// Step 2: Build manifest
	manifest := e.serializer.BuildManifest(req.SpaceID, req.SpaceName, req.ExporterID, resources)

	// Step 3: Serialize to ZIP
	zipContent, fileSize, err := e.serializer.SerializeToZip(ctx, manifest, resources)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to serialize space %d to ZIP: %v", req.SpaceID, err)
		return nil, err
	}

	// Step 4: Upload to object storage
	fileName := e.generateFileName(req.SpaceID, req.SpaceName)
	objectKey := fmt.Sprintf("%s/%s", ExportFilePrefix, fileName)

	err = e.objectStorage.PutObject(ctx, objectKey, zipContent)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to upload export file for space %d: %v", req.SpaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("msg", "failed to upload export file"))
	}

	// Step 5: Generate presigned download URL
	downloadURL, err := e.objectStorage.GetObjectUrl(ctx, objectKey)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to generate download URL for space %d: %v", req.SpaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("msg", "failed to generate download URL"))
	}

	expiresAt := time.Now().Add(ExportFileTTL)

	logs.CtxInfof(ctx, "Space export completed for space_id=%d, file_size=%d, expires_at=%v",
		req.SpaceID, fileSize, expiresAt)

	return &ExportResult{
		DownloadURL: downloadURL,
		FileName:    fileName,
		FileSize:    fileSize,
		ExpiresAt:   expiresAt,
		Statistics: Statistics{
			Agents:    len(resources.Agents),
			Plugins:   len(resources.Plugins),
			Workflows: len(resources.Workflows),
			Variables: len(resources.Variables),
		},
	}, nil
}

// SyncExportRawResult holds the raw output of a sync export (before upload)
type SyncExportRawResult struct {
	ZipContent []byte
	Manifest   *Manifest
	Resources  *SpaceResources
	FileSize   int64
}

// exportSyncInternal collects resources and serializes to ZIP without uploading.
// Shared by ExportSync and ExportSyncRaw.
func (e *SpaceExporter) exportSyncInternal(ctx context.Context, req *SyncExportRequest) (*SyncExportRawResult, error) {
	var (
		resources *SpaceResources
		deleted   *DeletedResources
		err       error
	)

	switch req.Mode {
	case "full":
		resources, err = e.collector.CollectAll(ctx, req.SpaceID)
		if err != nil {
			logs.CtxErrorf(ctx, "Failed to collect resources for space %d: %v", req.SpaceID, err)
			return nil, err
		}
	case "incremental":
		resources, deleted, err = e.collector.CollectIncremental(ctx, req.SpaceID, req.SinceTime)
		if err != nil {
			logs.CtxErrorf(ctx, "Failed to collect incremental resources for space %d: %v", req.SpaceID, err)
			return nil, err
		}
	default:
		return nil, errorx.WrapByCode(fmt.Errorf("invalid sync mode: %s", req.Mode), errno.ErrSpaceExportFailedCode, errorx.KV("mode", req.Mode))
	}

	fileContents := e.downloadKnowledgeFiles(ctx, resources)

	manifest := e.serializer.BuildSyncManifest(req.SpaceID, req.SpaceName, req.Mode, req.SinceTime, resources)
	if req.ReleaseVersion != "" {
		manifest.ReleaseVersion = req.ReleaseVersion
	}

	syncState := &SyncState{
		ExportTime:    time.Now().Unix(),
		SourceSpaceID: req.SpaceID,
	}

	zipContent, err := e.serializer.SerializeToSyncZip(ctx, manifest, resources, fileContents, syncState, deleted)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to serialize sync export for space %d: %v", req.SpaceID, err)
		return nil, err
	}

	fileSize := int64(len(zipContent))
	if fileSize > SyncExportMaxSize {
		return nil, errorx.WrapByCode(
			fmt.Errorf("sync export size %d exceeds limit %d", fileSize, SyncExportMaxSize),
			errno.ErrSpaceExportFailedCode,
			errorx.KV("msg", "sync export file too large"),
		)
	}

	return &SyncExportRawResult{
		ZipContent: zipContent,
		Manifest:   manifest,
		Resources:  resources,
		FileSize:   fileSize,
	}, nil
}

// ExportSyncRaw exports a space for sync and returns raw ZIP bytes + manifest.
// Unlike ExportSync, this does NOT upload to object storage — caller handles storage.
func (e *SpaceExporter) ExportSyncRaw(ctx context.Context, req *SyncExportRequest) (*SyncExportRawResult, error) {
	logs.CtxInfof(ctx, "Starting raw sync export for space_id=%d, mode=%s", req.SpaceID, req.Mode)
	return e.exportSyncInternal(ctx, req)
}

// ExportSync exports a space for sync (full or incremental) and returns the download URL
func (e *SpaceExporter) ExportSync(ctx context.Context, req *SyncExportRequest) (*ExportResult, error) {
	logs.CtxInfof(ctx, "Starting sync export for space_id=%d, mode=%s, since_time=%d", req.SpaceID, req.Mode, req.SinceTime)

	raw, err := e.exportSyncInternal(ctx, req)
	if err != nil {
		return nil, err
	}

	// Upload to object storage
	fileName := e.generateFileName(req.SpaceID, req.SpaceName)
	objectKey := fmt.Sprintf("%s/%s", SyncExportFilePrefix, fileName)

	if err = e.objectStorage.PutObject(ctx, objectKey, raw.ZipContent); err != nil {
		logs.CtxErrorf(ctx, "Failed to upload sync export file for space %d: %v", req.SpaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("msg", "failed to upload sync export file"))
	}

	// Generate presigned download URL
	downloadURL, err := e.objectStorage.GetObjectUrl(ctx, objectKey)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to generate download URL for space %d: %v", req.SpaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("msg", "failed to generate download URL"))
	}

	expiresAt := time.Now().Add(ExportFileTTL)

	logs.CtxInfof(ctx, "Sync export completed for space_id=%d, mode=%s, file_size=%d, expires_at=%v",
		req.SpaceID, req.Mode, raw.FileSize, expiresAt)

	return &ExportResult{
		DownloadURL: downloadURL,
		FileName:    fileName,
		FileSize:    raw.FileSize,
		ExpiresAt:   expiresAt,
		Statistics:  raw.Manifest.Statistics,
	}, nil
}

// downloadKnowledgeFiles downloads original files for knowledge base documents from object storage
func (e *SpaceExporter) downloadKnowledgeFiles(ctx context.Context, resources *SpaceResources) map[int64][]byte {
	fileContents := make(map[int64][]byte)
	for _, kb := range resources.KnowledgeBases {
		for _, doc := range kb.Documents {
			if doc.URI == "" {
				continue
			}
			content, err := e.objectStorage.GetObject(ctx, doc.URI)
			if err != nil {
				logs.CtxWarnf(ctx, "Failed to download file for document %d (uri=%s): %v", doc.ID, doc.URI, err)
				continue
			}
			fileContents[doc.ID] = content
		}
	}
	return fileContents
}

// isSpaceEmpty checks if the space has no resources
func (e *SpaceExporter) isSpaceEmpty(resources *SpaceResources) bool {
	return len(resources.Agents) == 0 &&
		len(resources.Plugins) == 0 &&
		len(resources.Workflows) == 0 &&
		len(resources.Variables) == 0
}

// generateFileName generates a filename for the export
func (e *SpaceExporter) generateFileName(spaceID int64, spaceName string) string {
	timestamp := time.Now().Format("20060102_150405")
	// Sanitize space name for filename
	safeName := sanitizeFileName(spaceName)
	if safeName == "" {
		safeName = "space"
	}
	return fmt.Sprintf("%s_%d_%s.zip", safeName, spaceID, timestamp)
}

// sanitizeFileName removes or replaces characters that are not safe for filenames
func sanitizeFileName(name string) string {
	if name == "" {
		return ""
	}

	// Replace unsafe characters with underscore
	result := make([]rune, 0, len(name))
	for _, r := range name {
		if isFilenameChar(r) {
			result = append(result, r)
		} else if len(result) > 0 && result[len(result)-1] != '_' {
			result = append(result, '_')
		}
	}

	// Trim trailing underscore
	for len(result) > 0 && result[len(result)-1] == '_' {
		result = result[:len(result)-1]
	}

	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}

	return string(result)
}

// isFilenameChar checks if a character is safe for filenames
func isFilenameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' ||
		// Allow Chinese characters
		(r >= 0x4E00 && r <= 0x9FFF)
}
