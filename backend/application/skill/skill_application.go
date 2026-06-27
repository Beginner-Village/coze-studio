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

package skill

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	pathutil "path"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/repository"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// SkillApplicationSVC is the global skill application service instance.
var SkillApplicationSVC *SkillApplicationService

type ProductSyncer interface {
	SyncSkillProduct(ctx context.Context, skillID int64) error
}

// ServiceComponents holds dependencies for the skill application service.
type ServiceComponents struct {
	IDGen idgen.IDGenerator
	DB    *gorm.DB
}

// SkillApplicationService orchestrates skill domain operations.
type SkillApplicationService struct {
	DomainSVC     service.SkillService
	ProductSyncer ProductSyncer
}

// SkillPackageExport is a reusable standard skill package encoded for JSON transport.
type SkillPackageExport struct {
	Filename  string
	Content   string
	Size      int64
	FilePaths []string
}

// SkillPackageValidation describes the standard-skill package shape before import.
type SkillPackageValidation struct {
	Valid       bool
	Error       string
	Filename    string
	Name        string
	Description string
	Metadata    entity.SkillMetadata
	FileCount   int32
	FilePaths   []string
	AssetPaths  []string
	ImagePaths  []string
}

// SkillAssetInfo describes a standard skill asset without loading all skill files.
type SkillAssetInfo struct {
	Path    string
	MIME    string
	Size    int64
	IsImage bool
}

// SkillAsset contains one standard skill asset's metadata and content.
type SkillAsset struct {
	SkillAssetInfo
	Content string
}

// InitService initializes the skill application service.
func InitService(c *ServiceComponents) *SkillApplicationService {
	domainComponents := &service.Components{
		SkillRepo: repository.NewSkillRepository(c.DB, c.IDGen),
	}

	domainSVC := service.NewService(domainComponents)
	SkillApplicationSVC = &SkillApplicationService{
		DomainSVC: domainSVC,
	}
	return SkillApplicationSVC
}

// CreateSkill creates a new skill.
func (s *SkillApplicationService) CreateSkill(ctx context.Context, spaceID int64, name, description, prompt, iconURI string, files map[string]string) (*entity.Skill, error) {
	if spaceID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}
	if err := validateStandardSkillFiles(files); err != nil {
		return nil, err
	}

	// 真·文件夹技能:若提供了 files 且含 SKILL.md,以 SKILL.md frontmatter 为单一事实源,
	// 解析出 name/description,prompt 取其正文。未显式传入时用解析值。
	if md, ok := files["SKILL.md"]; ok && md != "" {
		fmName, fmDesc, body := parseSkillFrontmatter(md)
		if name == "" {
			name = fmName
		}
		if description == "" {
			description = fmDesc
		}
		if prompt == "" {
			prompt = body
		}
	}

	if name == "" {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "name is required"))
	}

	userID, err := currentSkillUserID(ctx)
	if err != nil {
		return nil, err
	}

	skill := &entity.Skill{
		SpaceID:     spaceID,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Files:       files,
		IconURI:     iconURI,
		CreatorID:   userID,
	}

	skillID, err := s.DomainSVC.CreateSkill(ctx, skill)
	if err != nil {
		return nil, err
	}

	created, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if created != nil {
		if err := s.syncSkillProduct(ctx, created.SkillID); err != nil {
			return nil, err
		}
	}
	return created, nil
}

// ImportSkillPackage creates a standard folder skill from an uploaded zip package.
func (s *SkillApplicationService) ImportSkillPackage(ctx context.Context, spaceID int64, filename, content, iconURI string) (*entity.Skill, error) {
	if spaceID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}
	if err := validateSkillPackageFilename(filename); err != nil {
		return nil, err
	}

	raw, err := decodeSkillPackageContent(content)
	if err != nil {
		return nil, err
	}
	files, err := extractStandardSkillPackage(raw)
	if err != nil {
		return nil, err
	}
	return s.CreateSkill(ctx, spaceID, "", "", "", iconURI, files)
}

// ValidateSkillPackage validates a standard folder skill zip package without importing it.
func (s *SkillApplicationService) ValidateSkillPackage(ctx context.Context, filename, content string) (*SkillPackageValidation, error) {
	validation := &SkillPackageValidation{
		Filename: strings.TrimSpace(filename),
	}
	if err := validateSkillPackageFilename(filename); err != nil {
		validation.Error = skillPackageValidationError(err)
		return validation, nil
	}
	raw, err := decodeSkillPackageContent(content)
	if err != nil {
		validation.Error = skillPackageValidationError(err)
		return validation, nil
	}
	files, err := extractStandardSkillPackage(raw)
	if err != nil {
		validation.Error = skillPackageValidationError(err)
		return validation, nil
	}

	filePaths := make([]string, 0, len(files))
	assetPaths := make([]string, 0)
	imagePaths := make([]string, 0)
	for relPath, fileContent := range files {
		filePaths = append(filePaths, relPath)
		if strings.HasPrefix(relPath, "assets/") {
			assetPaths = append(assetPaths, relPath)
			if strings.HasPrefix(fileContent, "data:image/") || strings.HasPrefix(inferStandardSkillAssetMIME(relPath, fileContent), "image/") {
				imagePaths = append(imagePaths, relPath)
			}
		}
	}
	sort.Strings(filePaths)
	sort.Strings(assetPaths)
	sort.Strings(imagePaths)

	name, description, _ := parseSkillFrontmatter(files["SKILL.md"])
	validation.Valid = true
	validation.Name = name
	validation.Description = description
	validation.Metadata = parseSkillMetadata(files["SKILL.md"])
	validation.FileCount = int32(len(filePaths))
	validation.FilePaths = filePaths
	validation.AssetPaths = assetPaths
	validation.ImagePaths = imagePaths
	return validation, nil
}

// ExportSkillPackage exports a permitted standard skill as a reusable zip package.
func (s *SkillApplicationService) ExportSkillPackage(ctx context.Context, skillID, spaceID int64) (*SkillPackageExport, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}
	if err := s.requireSpaceMember(ctx, spaceID); err != nil {
		return nil, err
	}

	source, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}

	exported := source
	if source.SpaceID != spaceID {
		if !isMarketplaceSkillVisibleToSpace(source, spaceID) {
			return nil, errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "skill is not published to this space"))
		}
		exported, err = s.GetMarketplaceSkill(ctx, skillID, spaceID)
		if err != nil {
			return nil, err
		}
	}

	return buildStandardSkillPackage(exported)
}

// GetSkill gets a skill by ID.
//
// This is the shared entrypoint for the space-scoped skill read/write
// operations (update, asset CRUD, delete, publish), so the cross-tenant IDOR
// gate lives here: the request-supplied space_id is only trusted after the
// caller is confirmed to be a member of that space.
func (s *SkillApplicationService) GetSkill(ctx context.Context, skillID, spaceID int64) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}
	if err := s.requireSpaceMember(ctx, spaceID); err != nil {
		return nil, err
	}

	skill, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}
	if skill.SpaceID != spaceID {
		return nil, errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "skill does not belong to this space"))
	}

	return skill, nil
}

// UpdateSkill updates a skill.
func (s *SkillApplicationService) UpdateSkill(ctx context.Context, skillID, spaceID int64, name, description, prompt, iconURI string, files map[string]string) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	// Verify skill exists and belongs to the space
	existing, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}
	if err := validateStandardSkillFiles(files); err != nil {
		return nil, err
	}

	if md, ok := files["SKILL.md"]; ok && md != "" {
		fmName, fmDesc, body := parseSkillFrontmatter(md)
		if name == "" {
			name = fmName
		}
		if description == "" {
			description = fmDesc
		}
		if prompt == "" {
			prompt = body
		}
	}

	skill := &entity.Skill{
		SkillID:     existing.SkillID,
		SpaceID:     existing.SpaceID,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Files:       files,
		IconURI:     iconURI,
	}

	if err := s.DomainSVC.UpdateSkill(ctx, skill); err != nil {
		return nil, err
	}

	updated, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		if err := s.syncSkillProduct(ctx, updated.SkillID); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// UpsertSkillAsset writes or replaces a file under assets/ in the standard skill package.
func (s *SkillApplicationService) UpsertSkillAsset(ctx context.Context, skillID, spaceID int64, relPath, content, mime string) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}
	existing, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}
	assetPath, err := sanitizeStandardSkillAssetPath(relPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(content) == "" {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "content is required"))
	}
	if len(content) > 10*1024*1024 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "asset content is too large"))
	}
	if mime == "" {
		mime = inferStandardSkillAssetMIME(assetPath, content)
	}
	if !isAllowedStandardSkillAssetMIME(mime) {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "unsupported asset mime"))
	}

	files := cloneSkillFiles(existing.Files)
	if files == nil {
		files = map[string]string{}
	}
	files[assetPath] = content
	if err := validateStandardSkillFiles(files); err != nil {
		return nil, err
	}
	if err := s.DomainSVC.UpdateSkill(ctx, &entity.Skill{
		SkillID: existing.SkillID,
		SpaceID: existing.SpaceID,
		Files:   files,
	}); err != nil {
		return nil, err
	}
	updated, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		if err := s.syncSkillProduct(ctx, updated.SkillID); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// ListSkillAssets returns lightweight metadata for files under assets/.
func (s *SkillApplicationService) ListSkillAssets(ctx context.Context, skillID, spaceID int64) ([]*SkillAssetInfo, error) {
	existing, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for relPath := range existing.Files {
		if strings.HasPrefix(relPath, "assets/") {
			paths = append(paths, relPath)
		}
	}
	sort.Strings(paths)

	assets := make([]*SkillAssetInfo, 0, len(paths))
	for _, relPath := range paths {
		content := existing.Files[relPath]
		assets = append(assets, skillAssetInfoFromContent(relPath, content))
	}
	return assets, nil
}

// GetSkillAsset returns one file under assets/.
func (s *SkillApplicationService) GetSkillAsset(ctx context.Context, skillID, spaceID int64, relPath string) (*SkillAsset, error) {
	existing, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}
	assetPath, err := sanitizeStandardSkillAssetPath(relPath)
	if err != nil {
		return nil, err
	}
	content, ok := existing.Files[assetPath]
	if !ok {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "asset not found"))
	}
	return &SkillAsset{
		SkillAssetInfo: *skillAssetInfoFromContent(assetPath, content),
		Content:        content,
	}, nil
}

// DeleteSkillAsset removes a file under assets/ from the standard skill package.
func (s *SkillApplicationService) DeleteSkillAsset(ctx context.Context, skillID, spaceID int64, relPath string) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}
	existing, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}
	assetPath, err := sanitizeStandardSkillAssetPath(relPath)
	if err != nil {
		return nil, err
	}

	files := cloneSkillFiles(existing.Files)
	if _, ok := files[assetPath]; !ok {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "asset not found"))
	}
	delete(files, assetPath)
	if err := validateStandardSkillFiles(files); err != nil {
		return nil, err
	}
	if err := s.DomainSVC.UpdateSkill(ctx, &entity.Skill{
		SkillID: existing.SkillID,
		SpaceID: existing.SpaceID,
		Files:   files,
	}); err != nil {
		return nil, err
	}
	updated, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		if err := s.syncSkillProduct(ctx, updated.SkillID); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// DeleteSkill deletes a skill.
func (s *SkillApplicationService) DeleteSkill(ctx context.Context, skillID, spaceID int64) error {
	if skillID <= 0 {
		return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	// Verify skill exists and belongs to the space
	_, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return err
	}

	return s.DomainSVC.DeleteSkill(ctx, skillID)
}

// PublishSkill publishes or unpublishes a skill.
func (s *SkillApplicationService) PublishSkill(ctx context.Context, skillID, spaceID int64, scope int8) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}
	if !isValidPublishScope(scope) {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid publish scope"))
	}

	skill, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}

	version := skill.Version
	publishedAt := time.Now().UnixMilli()
	publishedBy, err := currentSkillUserID(ctx)
	if err != nil {
		return nil, err
	}
	if scope == entity.SkillPublishScopePrivate {
		version = 0
		publishedAt = 0
		publishedBy = 0
	}

	// Global publishing requires platform review; space/private are self-governed.
	reviewStatus := entity.SkillReviewStatusApproved
	if scope == entity.SkillPublishScopeGlobal {
		reviewStatus = entity.SkillReviewStatusPending
	}

	if err := s.DomainSVC.PublishSkill(ctx, skillID, scope, reviewStatus, version, publishedBy, publishedAt); err != nil {
		return nil, err
	}

	published, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if published != nil {
		if err := s.syncSkillProduct(ctx, published.SkillID); err != nil {
			return nil, err
		}
	}
	return published, nil
}

// ReviewSkill records a platform review decision for a globally-published skill.
func (s *SkillApplicationService) ReviewSkill(ctx context.Context, skillID int64, approve bool, note string) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	skill, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}

	if err := s.requireSpaceManager(ctx, skill.SpaceID); err != nil {
		return nil, err
	}
	reviewerID, err := currentSkillUserID(ctx)
	if err != nil {
		return nil, err
	}

	reviewStatus := entity.SkillReviewStatusRejected
	if approve {
		reviewStatus = entity.SkillReviewStatusApproved
	}

	if err := s.DomainSVC.ReviewSkill(ctx, skillID, reviewStatus, note, reviewerID, time.Now().UnixMilli()); err != nil {
		return nil, err
	}

	reviewed, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if reviewed != nil {
		if err := s.syncSkillProduct(ctx, reviewed.SkillID); err != nil {
			return nil, err
		}
	}
	return reviewed, nil
}

// ListPendingReviews lists a space's skills awaiting review. Only that space's
// owner/admin may view its review queue.
func (s *SkillApplicationService) ListPendingReviews(ctx context.Context, spaceID int64, page, pageSize int32) ([]*entity.Skill, int32, error) {
	if err := s.requireSpaceManager(ctx, spaceID); err != nil {
		return nil, 0, err
	}
	resp, err := s.DomainSVC.ListPendingReviews(ctx, &entity.PendingReviewListRequest{
		SpaceID:  spaceID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.Skills, resp.Total, nil
}

// ListSkills lists skills in a space.
func (s *SkillApplicationService) ListSkills(ctx context.Context, spaceID int64, page, pageSize int32, keyword string) ([]*entity.Skill, int32, error) {
	if err := s.requireSpaceMember(ctx, spaceID); err != nil {
		return nil, 0, err
	}

	resp, err := s.DomainSVC.ListSkills(ctx, &entity.ListRequest{
		SpaceID:  spaceID,
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Skills, resp.Total, nil
}

// ListMarketplaceSkills lists published skills visible to a space.
func (s *SkillApplicationService) ListMarketplaceSkills(ctx context.Context, spaceID int64, scope int8, page, pageSize int32, keyword string) ([]*entity.Skill, int32, error) {
	if scope == entity.SkillPublishScopePrivate || (scope != 0 && !isValidPublishScope(scope)) {
		return nil, 0, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid publish scope"))
	}
	if scope == entity.SkillPublishScopeSpace && spaceID <= 0 {
		return nil, 0, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}

	resp, err := s.DomainSVC.ListMarketplaceSkills(ctx, &entity.MarketplaceListRequest{
		SpaceID:  spaceID,
		Scope:    scope,
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Skills, resp.Total, nil
}

// GetMarketplaceSkill returns the published snapshot for a marketplace-visible skill.
func (s *SkillApplicationService) GetMarketplaceSkill(ctx context.Context, skillID, targetSpaceID int64) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	source, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}
	if !isMarketplaceSkillVisibleToSpace(source, targetSpaceID) {
		return nil, errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "skill is not published to this space"))
	}
	if source.PublishedVersion <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "published version is required"))
	}

	snapshot, err := s.DomainSVC.GetSkillVersion(ctx, source.SkillID, source.PublishedVersion)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "published skill version not found"))
	}

	return publishedSnapshotToSkill(source, snapshot), nil
}

// InstallMarketplaceSkill copies a published skill snapshot into the target space as a private skill.
func (s *SkillApplicationService) InstallMarketplaceSkill(ctx context.Context, sourceSkillID, targetSpaceID int64) (*entity.Skill, error) {
	if sourceSkillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}
	if targetSpaceID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}

	source, err := s.DomainSVC.GetSkill(ctx, sourceSkillID)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}
	if !isMarketplaceSkillVisibleToSpace(source, targetSpaceID) {
		return nil, errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "skill is not published to this space"))
	}
	if source.PublishedVersion <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "published version is required"))
	}

	snapshot, err := s.DomainSVC.GetSkillVersion(ctx, source.SkillID, source.PublishedVersion)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "published skill version not found"))
	}

	name, err := s.uniqueInstalledSkillName(ctx, targetSpaceID, snapshot.Name)
	if err != nil {
		return nil, err
	}
	userID, err := currentSkillUserID(ctx)
	if err != nil {
		return nil, err
	}
	installed := &entity.Skill{
		SpaceID:      targetSpaceID,
		Name:         name,
		Description:  snapshot.Description,
		Prompt:       snapshot.Prompt,
		Files:        cloneSkillFiles(snapshot.Files),
		IconURI:      snapshot.IconURI,
		CreatorID:    userID,
		Status:       entity.SkillStatusActive,
		PublishScope: entity.SkillPublishScopePrivate,
	}

	installedID, err := s.DomainSVC.CreateSkill(ctx, installed)
	if err != nil {
		return nil, err
	}
	installedSkill, err := s.DomainSVC.GetSkill(ctx, installedID)
	if err != nil {
		return nil, err
	}
	if installedSkill != nil {
		if err := s.syncSkillProduct(ctx, installedSkill.SkillID); err != nil {
			return nil, err
		}
	}
	return installedSkill, nil
}

func (s *SkillApplicationService) syncSkillProduct(ctx context.Context, skillID int64) error {
	if s == nil || s.ProductSyncer == nil || skillID <= 0 {
		return nil
	}
	return s.ProductSyncer.SyncSkillProduct(ctx, skillID)
}

func isValidPublishScope(scope int8) bool {
	switch scope {
	case entity.SkillPublishScopePrivate, entity.SkillPublishScopeSpace, entity.SkillPublishScopeGlobal:
		return true
	default:
		return false
	}
}

const (
	maxSkillPackageBytes     = 25 * 1024 * 1024
	maxSkillPackageFiles     = 200
	maxSkillPackageFileBytes = 5 * 1024 * 1024
)

func decodeSkillPackageContent(content string) ([]byte, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "content is required"))
	}
	if len(content) > maxSkillPackageBytes*2 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill package is too large"))
	}
	if strings.HasPrefix(content, "data:") {
		idx := strings.Index(content, ",")
		if idx < 0 {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill package content"))
		}
		content = content[idx+1:]
	}
	content = strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(content)

	raw, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill package content"))
	}
	if len(raw) > maxSkillPackageBytes {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill package is too large"))
	}
	return raw, nil
}

func validateSkillPackageFilename(filename string) error {
	if strings.TrimSpace(filename) != "" && !strings.HasSuffix(strings.ToLower(strings.TrimSpace(filename)), ".zip") {
		return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill package must be a .zip file"))
	}
	return nil
}

func skillPackageValidationError(err error) string {
	if err == nil {
		return ""
	}
	var statusErr errorx.StatusError
	if errors.As(err, &statusErr) && strings.TrimSpace(statusErr.Msg()) != "" {
		return strings.TrimSpace(statusErr.Msg())
	}
	msg := strings.TrimSpace(errorx.ErrorWithoutStack(err))
	if idx := strings.Index(msg, "\n"); idx >= 0 {
		msg = strings.TrimSpace(msg[:idx])
	}
	return msg
}

func extractStandardSkillPackage(raw []byte) (map[string]string, error) {
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill zip package"))
	}

	entries := make(map[string]*zip.File)
	paths := make([]string, 0)
	var totalUncompressed uint64
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || isIgnorableSkillPackageEntry(entry.Name) {
			continue
		}
		relPath, err := normalizeSkillPackageEntryPath(entry.Name)
		if err != nil {
			return nil, err
		}
		if isIgnorableSkillPackageEntry(relPath) {
			continue
		}
		totalUncompressed += entry.UncompressedSize64
		if totalUncompressed > maxSkillPackageBytes {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill package is too large"))
		}
		entries[relPath] = entry
		paths = append(paths, relPath)
	}
	if len(paths) == 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill package is empty"))
	}
	if len(paths) > maxSkillPackageFiles {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill package has too many files"))
	}

	root := detectSkillPackageRoot(paths)
	files := make(map[string]string, len(entries))
	for relPath, entry := range entries {
		skillPath := relPath
		if root != "" {
			skillPath = strings.TrimPrefix(relPath, root+"/")
		}
		if skillPath == "" || isIgnorableSkillPackageEntry(skillPath) {
			continue
		}
		if !isValidStandardSkillFilePath(skillPath) {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file path: %s", relPath)))
		}

		content, err := readSkillPackageEntry(entry)
		if err != nil {
			return nil, err
		}
		files[skillPath] = encodeSkillPackageFileContent(skillPath, content)
	}
	if err := validateStandardSkillFiles(files); err != nil {
		return nil, err
	}
	return files, nil
}

func normalizeSkillPackageEntryPath(entryName string) (string, error) {
	entryName = strings.TrimSpace(entryName)
	if entryName == "" || strings.Contains(entryName, "\\") || strings.HasPrefix(entryName, "/") {
		return "", errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file path: %s", entryName)))
	}
	parts := strings.Split(entryName, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file path: %s", entryName)))
		}
	}
	return strings.Join(parts, "/"), nil
}

func isIgnorableSkillPackageEntry(entryName string) bool {
	entryName = strings.TrimSpace(entryName)
	return entryName == "" ||
		entryName == ".DS_Store" ||
		strings.HasSuffix(entryName, "/.DS_Store") ||
		strings.HasPrefix(entryName, "__MACOSX/")
}

func detectSkillPackageRoot(paths []string) string {
	for _, relPath := range paths {
		if relPath == "SKILL.md" {
			return ""
		}
	}
	root := ""
	for _, relPath := range paths {
		parts := strings.Split(relPath, "/")
		if len(parts) < 2 {
			return ""
		}
		if root == "" {
			root = parts[0]
			continue
		}
		if root != parts[0] {
			return ""
		}
	}
	if isStandardSkillRootName(root) {
		return ""
	}
	return root
}

func isStandardSkillRootName(name string) bool {
	switch name {
	case "scripts", "references", "templates", "assets":
		return true
	default:
		return false
	}
}

func readSkillPackageEntry(entry *zip.File) ([]byte, error) {
	if entry.UncompressedSize64 > maxSkillPackageFileBytes {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("skill file is too large: %s", entry.Name)))
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file: %s", entry.Name)))
	}
	defer reader.Close()

	content, err := io.ReadAll(io.LimitReader(reader, maxSkillPackageFileBytes+1))
	if err != nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file: %s", entry.Name)))
	}
	if len(content) > maxSkillPackageFileBytes {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("skill file is too large: %s", entry.Name)))
	}
	return content, nil
}

func encodeSkillPackageFileContent(relPath string, content []byte) string {
	text := string(content)
	if strings.HasPrefix(text, "data:") {
		return text
	}
	if strings.HasPrefix(relPath, "assets/") {
		mime := inferStandardSkillAssetMIME(relPath, "")
		if strings.HasPrefix(mime, "image/") {
			return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(content)
		}
	}
	return text
}

func buildStandardSkillPackage(skill *entity.Skill) (*SkillPackageExport, error) {
	if skill == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}
	if err := validateStandardSkillFiles(skill.Files); err != nil {
		return nil, err
	}

	filePaths := make([]string, 0, len(skill.Files))
	for relPath := range skill.Files {
		filePaths = append(filePaths, relPath)
	}
	sort.Strings(filePaths)

	root := sanitizeSkillPackageRootName(skill.Name)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, relPath := range filePaths {
		content, err := decodeSkillPackageExportFileContent(skill.Files[relPath])
		if err != nil {
			return nil, err
		}
		header := &zip.FileHeader{
			Name:   root + "/" + relPath,
			Method: zip.Deflate,
		}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file: %s", relPath)))
		}
		if _, err := writer.Write(content); err != nil {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file: %s", relPath)))
		}
	}
	if err := zw.Close(); err != nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill zip package"))
	}

	return &SkillPackageExport{
		Filename:  root + ".zip",
		Content:   "data:application/zip;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
		Size:      int64(buf.Len()),
		FilePaths: filePaths,
	}, nil
}

func decodeSkillPackageExportFileContent(content string) ([]byte, error) {
	if !strings.HasPrefix(content, "data:") {
		return []byte(content), nil
	}
	idx := strings.Index(content, ",")
	if idx < 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill file content"))
	}
	metadata := strings.ToLower(content[:idx])
	payload := content[idx+1:]
	if strings.Contains(metadata, ";base64") {
		payload = strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(payload)
		raw, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill file content"))
		}
		return raw, nil
	}
	raw, err := url.PathUnescape(payload)
	if err != nil {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "invalid skill file content"))
	}
	return []byte(raw), nil
}

func sanitizeSkillPackageRootName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var builder strings.Builder
	lastDash := false
	for _, ch := range name {
		allowed := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-'
		if allowed {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	root := strings.Trim(builder.String(), "-._")
	if root == "" {
		return "skill-package"
	}
	return root
}

// forbiddenSkillFileExtensions blocks executable / native-binary file types from
// standard skill packages (defense-in-depth for the company skill marketplace).
var forbiddenSkillFileExtensions = map[string]struct{}{
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".bat": {}, ".cmd": {},
	".com": {}, ".scr": {}, ".msi": {}, ".app": {}, ".jar": {}, ".class": {},
	".pyc": {}, ".pyo": {}, ".o": {}, ".a": {}, ".bin": {}, ".deb": {},
	".rpm": {}, ".dmg": {}, ".pkg": {}, ".node": {},
}

// hasForbiddenSkillExtension reports whether a skill file path carries a blocked
// executable/native-binary extension.
func hasForbiddenSkillExtension(relPath string) bool {
	ext := strings.ToLower(pathutil.Ext(relPath))
	_, bad := forbiddenSkillFileExtensions[ext]
	return bad
}

func validateStandardSkillFiles(files map[string]string) error {
	if strings.TrimSpace(files["SKILL.md"]) == "" {
		return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "SKILL.md is required"))
	}

	for relPath := range files {
		if !isValidStandardSkillFilePath(relPath) {
			return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file path: %s", relPath)))
		}
		if hasForbiddenSkillExtension(relPath) {
			return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("forbidden executable/binary file type in skill package: %s", relPath)))
		}
	}
	return nil
}

// isValidStandardSkillFilePath 只做安全校验,不限制目录结构。
// 标准技能包(对齐 Anthropic / LangChain Agent Skills)允许任意相对路径布局,
// 顶层常见 README.md / requirements.txt / LICENSE / pyproject.toml,以及任意自定义
// 子目录都应被接受。真正的安全防护由以下机制保证,与目录无关:
//   - 这里拦截路径穿越:无 `..`(经 pathutil.Clean 校验)、无绝对路径、无反斜杠、无前后空白
//   - hasForbiddenSkillExtension 拦截可执行/原生二进制(.exe/.so/.pyc 等)
//   - 上层限制单包总大小与文件数量
func isValidStandardSkillFilePath(relPath string) bool {
	if relPath == "" || relPath != strings.TrimSpace(relPath) {
		return false
	}
	if strings.Contains(relPath, "\\") || strings.HasPrefix(relPath, "/") {
		return false
	}
	// pathutil.Clean 规整后若与原值不同,说明含 `.`/重复分隔符等可疑片段,拒绝。
	if pathutil.Clean(relPath) != relPath {
		return false
	}
	// 显式拦截路径穿越:任一片段为 `..`/`.`/空 都拒绝
	// (pathutil.Clean 不会移除前导 `..`,必须单独判断,否则 `../escape` 会漏过)。
	for _, part := range strings.Split(relPath, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func sanitizeStandardSkillAssetPath(relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if !strings.HasPrefix(relPath, "assets/") {
		return "", errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "asset path must be under assets/"))
	}
	if !isValidStandardSkillFilePath(relPath) {
		return "", errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", fmt.Sprintf("invalid skill file path: %s", relPath)))
	}
	return relPath, nil
}

func inferStandardSkillAssetMIME(relPath string, content string) string {
	if strings.HasPrefix(content, "data:") {
		rest := strings.TrimPrefix(content, "data:")
		if idx := strings.IndexAny(rest, ";,"); idx > 0 {
			return rest[:idx]
		}
	}
	switch strings.ToLower(pathutil.Ext(relPath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func isAllowedStandardSkillAssetMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/png", "image/jpeg", "image/webp", "image/gif", "image/svg+xml", "application/json", "text/plain", "text/markdown", "application/octet-stream":
		return true
	default:
		return false
	}
}

func skillAssetInfoFromContent(relPath, content string) *SkillAssetInfo {
	mime := inferStandardSkillAssetMIME(relPath, content)
	return &SkillAssetInfo{
		Path:    relPath,
		MIME:    mime,
		Size:    int64(len(content)),
		IsImage: strings.HasPrefix(mime, "image/") || strings.HasPrefix(content, "data:image/"),
	}
}

func currentSkillUserID(ctx context.Context) (int64, error) {
	if userID := ctxutil.GetUIDFromCtx(ctx); userID != nil {
		return *userID, nil
	}
	apiAuth := ctxutil.GetApiAuthFromCtx(ctx)
	if apiAuth != nil && apiAuth.UserID != 0 {
		return apiAuth.UserID, nil
	}
	return 0, errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "user context is required"))
}

// requireSpaceManager gates skill review (approve/reject + the review queue) to the
// skill space's owner/admin — reusing the existing space RoleType, no separate reviewer
// config or env var needed. A skill published to global scope is reviewed by an
// owner/admin of its own space (they vouch for it before it reaches the global market).
// SECURITY: members below admin, or non-members, are denied — a normal user cannot
// self-approve their own skill onto the marketplace or read the review queue.
func (s *SkillApplicationService) requireSpaceManager(ctx context.Context, spaceID int64) error {
	if spaceID <= 0 {
		return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}
	uid, err := currentSkillUserID(ctx)
	if err != nil {
		return err
	}
	perm, err := crossuser.DefaultSVC().CheckSpacePermission(ctx, spaceID, uid)
	if err != nil {
		return err
	}
	if perm == nil || !perm.CanManage {
		return errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "space owner/admin role required to review skills"))
	}
	return nil
}

// requireSpaceMember asserts the caller belongs to the given space (any role).
// SECURITY: skill read/write endpoints take a request-supplied space_id, so the
// space_id alone is NOT authorization. This member-level gate (mirroring
// requireSpaceManager but accepting normal members, not just owner/admin) closes
// the cross-tenant IDOR: a caller can only operate on skills in a space they are
// actually a member of.
func (s *SkillApplicationService) requireSpaceMember(ctx context.Context, spaceID int64) error {
	if spaceID <= 0 {
		return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}
	uid, err := currentSkillUserID(ctx)
	if err != nil {
		return err
	}
	perm, err := crossuser.DefaultSVC().CheckSpacePermission(ctx, spaceID, uid)
	if err != nil {
		return err
	}
	if perm == nil || !perm.IsMember {
		return errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "you do not have access to this space"))
	}
	return nil
}

func isMarketplaceSkillVisibleToSpace(skill *entity.Skill, targetSpaceID int64) bool {
	if skill.PublishScope == entity.SkillPublishScopeGlobal {
		return true
	}
	return skill.PublishScope == entity.SkillPublishScopeSpace && skill.SpaceID == targetSpaceID
}

func publishedSnapshotToSkill(source *entity.Skill, snapshot *entity.SkillVersion) *entity.Skill {
	return &entity.Skill{
		SkillID:          source.SkillID,
		SpaceID:          source.SpaceID,
		Name:             snapshot.Name,
		Description:      snapshot.Description,
		Prompt:           snapshot.Prompt,
		Files:            cloneSkillFiles(snapshot.Files),
		IconURI:          snapshot.IconURI,
		CreatorID:        source.CreatorID,
		Status:           source.Status,
		PublishScope:     source.PublishScope,
		PublishedVersion: source.PublishedVersion,
		PublishedAt:      source.PublishedAt,
		PublishedBy:      source.PublishedBy,
		Version:          snapshot.Version,
		CreatedAt:        source.CreatedAt,
		UpdatedAt:        source.UpdatedAt,
	}
}

func cloneSkillFiles(files map[string]string) map[string]string {
	if files == nil {
		return nil
	}
	cloned := make(map[string]string, len(files))
	for path, content := range files {
		cloned[path] = content
	}
	return cloned
}

func (s *SkillApplicationService) uniqueInstalledSkillName(ctx context.Context, spaceID int64, base string) (string, error) {
	if base == "" {
		base = "imported-skill"
	}
	existing, err := s.DomainSVC.GetSkillByName(ctx, spaceID, base)
	if err != nil {
		return "", err
	}
	if existing == nil {
		return base, nil
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s (%d)", base, i)
		existing, err = s.DomainSVC.GetSkillByName(ctx, spaceID, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return "", errorx.New(errno.ErrSkillDuplicateNameCode, errorx.KV("name", base))
}
