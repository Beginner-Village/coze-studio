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

package release

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"gorm.io/gorm"

	spaceexport "github.com/ynet-dev/ynet-studio/backend/application/space/export"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	releasePackagePrefix = "space_releases"
)

var semverRegex = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

type ReleaseService struct {
	db            *gorm.DB
	exporter      *spaceexport.SpaceExporter
	objectStorage storage.Storage
	repo          *ReleaseRepo
}

func NewReleaseService(db *gorm.DB, exporter *spaceexport.SpaceExporter, objectStorage storage.Storage) *ReleaseService {
	return &ReleaseService{
		db:            db,
		exporter:      exporter,
		objectStorage: objectStorage,
		repo:          NewReleaseRepo(db),
	}
}

// CreateRelease creates a new release version for a space.
// It exports the space, stores the package permanently, and records the release.
func (s *ReleaseService) CreateRelease(ctx context.Context, req *CreateReleaseRequest) (*SpaceRelease, error) {
	// Auto-generate version if not specified
	version := req.Version
	if version == "" {
		var err error
		version, err = s.NextVersion(ctx, req.SpaceID)
		if err != nil {
			return nil, err
		}
	}

	// Validate version format
	if !semverRegex.MatchString(version) {
		return nil, errorx.New(errno.ErrSpaceReleaseInvalidVersion, errorx.KV("version", version))
	}

	// Check uniqueness
	exists, err := s.repo.VersionExists(ctx, req.SpaceID, version)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errorx.New(errno.ErrSpaceReleaseExistsCode, errorx.KV("version", version))
	}

	// Find parent version for incremental since_time
	var parentVersion *string
	var sinceTime int64

	latest, _ := s.repo.GetLatestPublished(ctx, req.SpaceID)
	if latest != nil {
		parentVersion = &latest.Version
	}

	syncType := req.SyncType
	if syncType == "" {
		syncType = "full"
	}

	if syncType == "incremental" {
		if latest == nil {
			return nil, errorx.New(errno.ErrSpaceReleaseInvalidVersion,
				errorx.KV("msg", "cannot create incremental release without a published parent version"))
		}
		// Parse export_time from parent's manifest
		var parentManifest spaceexport.Manifest
		if err = json.Unmarshal(latest.Manifest, &parentManifest); err != nil {
			return nil, fmt.Errorf("parse parent manifest: %w", err)
		}
		exportTime, parseErr := time.Parse(time.RFC3339, parentManifest.ExportTime)
		if parseErr != nil {
			return nil, fmt.Errorf("parse parent export_time: %w", parseErr)
		}
		sinceTime = exportTime.UnixMilli()
	}

	// Export the space (raw, without uploading to temporary storage)
	logs.CtxInfof(ctx, "Creating release %s for space_id=%d, sync_type=%s", version, req.SpaceID, syncType)

	raw, err := s.exporter.ExportSyncRaw(ctx, &spaceexport.SyncExportRequest{
		SpaceID:   req.SpaceID,
		Mode:      syncType,
		SinceTime: sinceTime,
	})
	if err != nil {
		return nil, err
	}

	// Set release version in manifest
	raw.Manifest.ReleaseVersion = version

	// Compute package hash (before modifying manifest, so hash matches the actual ZIP)
	contentHash := HashPackage(raw.ZipContent)

	// Build resource hashes for diff capability.
	// These are stored in the manifest JSON in DB (not in the ZIP itself).
	// Note: hashes may vary for interface{}/map fields; acceptable for MVP.
	if raw.Resources != nil {
		if hashes, hashErr := BuildResourceHashes(raw.Resources); hashErr == nil {
			raw.Manifest.ResourceHashes = hashes
		}
	}

	// Upload to permanent storage (no TTL)
	packageKey := fmt.Sprintf("%s/%d/%s.zip", releasePackagePrefix, req.SpaceID, version)
	if err = s.objectStorage.PutObject(ctx, packageKey, raw.ZipContent); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode,
			errorx.KV("msg", "failed to upload release package"))
	}

	// Marshal manifest and statistics for DB storage
	manifestJSON, err := json.Marshal(raw.Manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	statsJSON, err := json.Marshal(raw.Manifest.Statistics)
	if err != nil {
		return nil, fmt.Errorf("marshal statistics: %w", err)
	}

	now := time.Now().Unix()
	record := &SpaceRelease{
		SpaceID:       req.SpaceID,
		Version:       version,
		SyncType:      syncType,
		ParentVersion: parentVersion,
		Manifest:      manifestJSON,
		Statistics:    statsJSON,
		PackageKey:    packageKey,
		PackageSize:   raw.FileSize,
		ContentHash:   contentHash,
		Status:        StatusDraft,
		CreatedBy:     req.UserID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if req.Tag != "" {
		record.Tag = &req.Tag
	}
	if req.Description != "" {
		record.Description = &req.Description
	}

	if err = s.repo.Create(ctx, record); err != nil {
		return nil, err
	}

	logs.CtxInfof(ctx, "Release %s created for space_id=%d, size=%d, hash=%s", version, req.SpaceID, raw.FileSize, contentHash)
	return record, nil
}

// PublishRelease marks a release as published
func (s *ReleaseService) PublishRelease(ctx context.Context, spaceID int64, version string) error {
	record, err := s.repo.GetByVersion(ctx, spaceID, version)
	if err != nil {
		return err
	}
	if record == nil {
		return errorx.New(errno.ErrSpaceReleaseNotFoundCode, errorx.KV("version", version))
	}
	if record.Status == StatusPublished {
		return nil // already published
	}

	now := time.Now().Unix()
	return s.repo.UpdateStatus(ctx, record.ID, StatusPublished, &now)
}

// DeprecateRelease marks a release as deprecated
func (s *ReleaseService) DeprecateRelease(ctx context.Context, spaceID int64, version string) error {
	record, err := s.repo.GetByVersion(ctx, spaceID, version)
	if err != nil {
		return err
	}
	if record == nil {
		return errorx.New(errno.ErrSpaceReleaseNotFoundCode, errorx.KV("version", version))
	}
	return s.repo.UpdateStatus(ctx, record.ID, StatusDeprecated, nil)
}

// GetRelease returns a release with a temporary download URL
func (s *ReleaseService) GetRelease(ctx context.Context, spaceID int64, version string) (*ReleaseDetail, error) {
	record, err := s.repo.GetByVersion(ctx, spaceID, version)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errorx.New(errno.ErrSpaceReleaseNotFoundCode, errorx.KV("version", version))
	}

	downloadURL, err := s.objectStorage.GetObjectUrl(ctx, record.PackageKey)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode,
			errorx.KV("msg", "failed to generate download URL"))
	}

	return &ReleaseDetail{
		Release:     record,
		DownloadURL: downloadURL,
		ExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
	}, nil
}

// ListReleases returns a paginated list of releases for a space
func (s *ReleaseService) ListReleases(ctx context.Context, spaceID int64, status string, page, pageSize int) ([]SpaceRelease, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.ListBySpace(ctx, spaceID, status, pageSize, offset)
}

// DiffVersions compares two versions and returns the difference.
// This is an ID-level diff (which resources were added/modified/removed),
// not a field-level diff.
func (s *ReleaseService) DiffVersions(ctx context.Context, spaceID int64, fromVersion, toVersion string) (*VersionDiff, error) {
	fromRelease, err := s.repo.GetByVersion(ctx, spaceID, fromVersion)
	if err != nil {
		return nil, err
	}
	if fromRelease == nil {
		return nil, errorx.New(errno.ErrSpaceReleaseNotFoundCode, errorx.KV("version", fromVersion))
	}

	toRelease, err := s.repo.GetByVersion(ctx, spaceID, toVersion)
	if err != nil {
		return nil, err
	}
	if toRelease == nil {
		return nil, errorx.New(errno.ErrSpaceReleaseNotFoundCode, errorx.KV("version", toVersion))
	}

	var fromManifest, toManifest spaceexport.Manifest
	if err = json.Unmarshal(fromRelease.Manifest, &fromManifest); err != nil {
		return nil, fmt.Errorf("parse from manifest: %w", err)
	}
	if err = json.Unmarshal(toRelease.Manifest, &toManifest); err != nil {
		return nil, fmt.Errorf("parse to manifest: %w", err)
	}

	diff := &VersionDiff{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Added:       make(map[string][]ResourceSummary),
		Modified:    make(map[string][]ResourceSummary),
		Removed:     make(map[string][]ResourceSummary),
	}

	// Compare ID registries
	diffIDList(diff, "agents", fromManifest.IDRegistry.Agents, toManifest.IDRegistry.Agents,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "agent")
	diffIDList(diff, "plugins", fromManifest.IDRegistry.Plugins, toManifest.IDRegistry.Plugins,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "plugin")
	diffIDList(diff, "workflows", fromManifest.IDRegistry.Workflows, toManifest.IDRegistry.Workflows,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "workflow")
	diffIDList(diff, "variables", fromManifest.IDRegistry.Variables, toManifest.IDRegistry.Variables,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "variable")
	diffIDList(diff, "space_models", fromManifest.IDRegistry.SpaceModels, toManifest.IDRegistry.SpaceModels,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "space_model")
	diffIDList(diff, "knowledge_bases", fromManifest.IDRegistry.KnowledgeBases, toManifest.IDRegistry.KnowledgeBases,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "knowledge")
	diffIDList(diff, "folders", fromManifest.IDRegistry.Folders, toManifest.IDRegistry.Folders,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "folder")
	diffIDList(diff, "external_knowledge", fromManifest.IDRegistry.ExternalKnowledge, toManifest.IDRegistry.ExternalKnowledge,
		fromManifest.ResourceHashes, toManifest.ResourceHashes, "external_knowledge")

	return diff, nil
}

// NextVersion computes the next auto-incremented version for a space
func (s *ReleaseService) NextVersion(ctx context.Context, spaceID int64) (string, error) {
	latest, err := s.repo.GetLatest(ctx, spaceID)
	if err != nil {
		return "", err
	}
	if latest == nil {
		return "v1.0.0", nil
	}

	major, minor, patch, ok := parseSemver(latest.Version)
	if !ok {
		return "v1.0.0", nil
	}
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch+1), nil
}

// GetRepo exposes the repo for use by other services (e.g., sync rollback)
func (s *ReleaseService) GetRepo() *ReleaseRepo {
	return s.repo
}

// --- internal helpers ---

func parseSemver(version string) (major, minor, patch int, ok bool) {
	matches := semverRegex.FindStringSubmatch(version)
	if len(matches) != 4 {
		return 0, 0, 0, false
	}
	major, _ = strconv.Atoi(matches[1])
	minor, _ = strconv.Atoi(matches[2])
	patch, _ = strconv.Atoi(matches[3])
	return major, minor, patch, true
}

func diffIDList(diff *VersionDiff, category string, fromIDs, toIDs []int64,
	fromHashes, toHashes map[string]string, hashPrefix string) {

	fromSet := make(map[int64]bool, len(fromIDs))
	for _, id := range fromIDs {
		fromSet[id] = true
	}
	toSet := make(map[int64]bool, len(toIDs))
	for _, id := range toIDs {
		toSet[id] = true
	}

	// Added: in toIDs but not in fromIDs
	for _, id := range toIDs {
		if !fromSet[id] {
			diff.Added[category] = append(diff.Added[category], ResourceSummary{ID: id})
		}
	}

	// Removed: in fromIDs but not in toIDs
	for _, id := range fromIDs {
		if !toSet[id] {
			diff.Removed[category] = append(diff.Removed[category], ResourceSummary{ID: id})
		}
	}

	// Modified: in both, but hash changed (if hashes available)
	if len(fromHashes) > 0 && len(toHashes) > 0 {
		for _, id := range toIDs {
			if !fromSet[id] {
				continue // already counted as added
			}
			key := fmt.Sprintf("%s:%d", hashPrefix, id)
			fh := fromHashes[key]
			th := toHashes[key]
			if fh != "" && th != "" && fh != th {
				diff.Modified[category] = append(diff.Modified[category], ResourceSummary{ID: id})
			}
		}
	}
}
