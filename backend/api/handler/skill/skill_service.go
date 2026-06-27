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
	"context"
	"fmt"
	"net/http"
	pathutil "path"
	"sort"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/ynet-dev/ynet-studio/backend/api/internal/httputil"
	singleagentApp "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	skillApp "github.com/ynet-dev/ynet-studio/backend/application/skill"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
)

type createSkillRequest struct {
	SpaceID     int64             `json:"space_id,string"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Prompt      string            `json:"prompt"`
	IconURI     string            `json:"icon_uri"`
	Files       map[string]string `json:"files"`
}

type importSkillPackageRequest struct {
	SpaceID  int64  `json:"space_id,string"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
	IconURI  string `json:"icon_uri"`
}

type importRuntimeSkillRequest struct {
	AgentID      int64   `json:"agent_id,string"`
	BotID        int64   `json:"bot_id,string"`
	ConnectorID  *string `json:"connector_id"`
	Name         string  `json:"name"`
	SkillID      int64   `json:"skill_id,string"`
	IconURI      string  `json:"icon_uri"`
	PublishScope int8    `json:"publish_scope"`
}

type validateSkillPackageRequest struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type exportSkillPackageRequest struct {
	SkillID int64 `json:"skill_id,string"`
	SpaceID int64 `json:"space_id,string"`
}

type getSkillRequest struct {
	SkillID int64 `query:"skill_id,string"`
	SpaceID int64 `query:"space_id,string"`
}

type updateSkillRequest struct {
	SkillID     int64             `json:"skill_id,string"`
	SpaceID     int64             `json:"space_id,string"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Prompt      string            `json:"prompt"`
	IconURI     string            `json:"icon_uri"`
	Files       map[string]string `json:"files"`
}

type upsertSkillAssetRequest struct {
	SkillID int64  `json:"skill_id,string"`
	SpaceID int64  `json:"space_id,string"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Mime    string `json:"mime"`
}

type listSkillAssetsRequest struct {
	SkillID int64 `query:"skill_id,string"`
	SpaceID int64 `query:"space_id,string"`
}

type getSkillAssetRequest struct {
	SkillID int64  `query:"skill_id,string"`
	SpaceID int64  `query:"space_id,string"`
	Path    string `query:"path"`
}

type deleteSkillAssetRequest struct {
	SkillID int64  `json:"skill_id,string"`
	SpaceID int64  `json:"space_id,string"`
	Path    string `json:"path"`
}

type deleteSkillRequest struct {
	SkillID int64 `json:"skill_id,string"`
	SpaceID int64 `json:"space_id,string"`
}

type publishSkillRequest struct {
	SkillID int64 `json:"skill_id,string"`
	SpaceID int64 `json:"space_id,string"`
	Scope   int8  `json:"scope"`
}

type reviewSkillRequest struct {
	SkillID int64  `json:"skill_id,string"`
	Approve bool   `json:"approve"`
	Note    string `json:"note"`
}

type listPendingReviewsRequest struct {
	SpaceID  int64 `query:"space_id,string"`
	Page     int32 `query:"page"`
	PageSize int32 `query:"page_size"`
}

type installMarketplaceSkillRequest struct {
	SkillID int64 `json:"skill_id,string"`
	SpaceID int64 `json:"space_id,string"`
}

type getMarketplaceSkillRequest struct {
	SkillID int64 `query:"skill_id,string"`
	SpaceID int64 `query:"space_id,string"`
}

type listSkillsRequest struct {
	SpaceID  int64  `query:"space_id,string"`
	Page     int32  `query:"page"`
	PageSize int32  `query:"page_size"`
	Keyword  string `query:"keyword"`
}

type marketplaceListSkillsRequest struct {
	SpaceID  int64  `query:"space_id,string"`
	Scope    int8   `query:"scope"`
	Page     int32  `query:"page"`
	PageSize int32  `query:"page_size"`
	Keyword  string `query:"keyword"`
}

type skillInfoResponse struct {
	SkillID          string                 `json:"skill_id"`
	SpaceID          string                 `json:"space_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Prompt           string                 `json:"prompt"`
	Files            map[string]string      `json:"files,omitempty"`
	Metadata         *skillMetadataResponse `json:"metadata,omitempty"`
	AssetSummary     skillAssetSummary      `json:"asset_summary"`
	IconURI          string                 `json:"icon_uri"`
	CreatorID        string                 `json:"creator_id"`
	Version          int64                  `json:"version"`
	PublishScope     int8                   `json:"publish_scope"`
	PublishedVersion int64                  `json:"published_version"`
	PublishedAt      int64                  `json:"published_at"`
	PublishedBy      string                 `json:"published_by"`
	ReviewStatus     int8                   `json:"review_status"`
	ReviewNote       string                 `json:"review_note,omitempty"`
	ReviewerID       string                 `json:"reviewer_id"`
	ReviewedAt       int64                  `json:"reviewed_at"`
	CreatedAt        int64                  `json:"created_at"`
	UpdatedAt        int64                  `json:"updated_at"`
}

type skillMetadataResponse struct {
	Version   string   `json:"version,omitempty"`
	Category  string   `json:"category,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
}

type skillAssetSummary struct {
	Count      int32    `json:"count"`
	ImagePaths []string `json:"image_paths"`
}

type skillPackageExportResponse struct {
	Filename  string   `json:"filename"`
	Content   string   `json:"content"`
	Size      int64    `json:"size"`
	FilePaths []string `json:"file_paths"`
}

type skillPackageValidationResponse struct {
	Valid       bool                   `json:"valid"`
	Error       string                 `json:"error,omitempty"`
	Filename    string                 `json:"filename"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    *skillMetadataResponse `json:"metadata,omitempty"`
	FileCount   int32                  `json:"file_count"`
	FilePaths   []string               `json:"file_paths"`
	AssetPaths  []string               `json:"asset_paths"`
	ImagePaths  []string               `json:"image_paths"`
}

type skillAssetResponse struct {
	Path    string `json:"path"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
	IsImage bool   `json:"is_image"`
	Content string `json:"content,omitempty"`
}

type response struct {
	Code int32       `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func entityToResponse(s *entity.Skill) *skillInfoResponse {
	if s == nil {
		return nil
	}
	return &skillInfoResponse{
		SkillID:          int64ToString(s.SkillID),
		SpaceID:          int64ToString(s.SpaceID),
		Name:             s.Name,
		Description:      s.Description,
		Prompt:           s.Prompt,
		Files:            s.Files,
		Metadata:         skillMetadataToResponse(s),
		AssetSummary:     skillAssetSummaryFromFiles(s.Files),
		IconURI:          s.IconURI,
		CreatorID:        int64ToString(s.CreatorID),
		Version:          s.Version,
		PublishScope:     s.PublishScope,
		PublishedVersion: s.PublishedVersion,
		PublishedAt:      s.PublishedAt,
		PublishedBy:      int64ToString(s.PublishedBy),
		ReviewStatus:     s.ReviewStatus,
		ReviewNote:       s.ReviewNote,
		ReviewerID:       int64ToString(s.ReviewerID),
		ReviewedAt:       s.ReviewedAt,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func skillMetadataToResponse(s *entity.Skill) *skillMetadataResponse {
	metadata := s.Metadata
	if isEmptySkillMetadata(metadata) && s.Files != nil {
		metadata = skillApp.ParseSkillMetadata(s.Files["SKILL.md"])
	}
	return skillMetadataValueToResponse(metadata)
}

func skillMetadataValueToResponse(metadata entity.SkillMetadata) *skillMetadataResponse {
	if isEmptySkillMetadata(metadata) {
		return nil
	}
	return &skillMetadataResponse{
		Version:   metadata.Version,
		Category:  metadata.Category,
		Tags:      metadata.Tags,
		Platforms: metadata.Platforms,
	}
}

func isEmptySkillMetadata(metadata entity.SkillMetadata) bool {
	return metadata.Version == "" &&
		metadata.Category == "" &&
		len(metadata.Tags) == 0 &&
		len(metadata.Platforms) == 0
}

func skillAssetSummaryFromFiles(files map[string]string) skillAssetSummary {
	paths := make([]string, 0)
	imagePaths := make([]string, 0)
	for filePath, content := range files {
		if !strings.HasPrefix(filePath, "assets/") {
			continue
		}
		paths = append(paths, filePath)
		if isImageSkillAsset(filePath, content) {
			imagePaths = append(imagePaths, filePath)
		}
	}
	sort.Strings(paths)
	sort.Strings(imagePaths)
	return skillAssetSummary{Count: int32(len(paths)), ImagePaths: imagePaths}
}

func skillPackageExportToResponse(pkg *skillApp.SkillPackageExport) *skillPackageExportResponse {
	if pkg == nil {
		return nil
	}
	return &skillPackageExportResponse{
		Filename:  pkg.Filename,
		Content:   pkg.Content,
		Size:      pkg.Size,
		FilePaths: pkg.FilePaths,
	}
}

func skillPackageValidationToResponse(validation *skillApp.SkillPackageValidation) *skillPackageValidationResponse {
	if validation == nil {
		return nil
	}
	return &skillPackageValidationResponse{
		Valid:       validation.Valid,
		Error:       validation.Error,
		Filename:    validation.Filename,
		Name:        validation.Name,
		Description: validation.Description,
		Metadata:    skillMetadataValueToResponse(validation.Metadata),
		FileCount:   validation.FileCount,
		FilePaths:   validation.FilePaths,
		AssetPaths:  validation.AssetPaths,
		ImagePaths:  validation.ImagePaths,
	}
}

func skillAssetInfoToResponse(asset *skillApp.SkillAssetInfo) *skillAssetResponse {
	if asset == nil {
		return nil
	}
	return &skillAssetResponse{
		Path:    asset.Path,
		Mime:    asset.MIME,
		Size:    asset.Size,
		IsImage: asset.IsImage,
	}
}

func skillAssetToResponse(asset *skillApp.SkillAsset) *skillAssetResponse {
	if asset == nil {
		return nil
	}
	resp := skillAssetInfoToResponse(&asset.SkillAssetInfo)
	resp.Content = asset.Content
	return resp
}

func isImageSkillAsset(filePath string, content string) bool {
	if strings.HasPrefix(content, "data:image/") {
		return true
	}
	switch strings.ToLower(pathutil.Ext(filePath)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg":
		return true
	default:
		return false
	}
}

func int64ToString(v int64) string {
	return fmt.Sprintf("%d", v)
}

// CreateSkill handles POST /api/skill/create
func CreateSkill(ctx context.Context, c *app.RequestContext) {
	var req createSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.CreateSkill(ctx, req.SpaceID, req.Name, req.Description, req.Prompt, req.IconURI, req.Files)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ImportSkillPackage handles POST /api/super-agent/skills/import
func ImportSkillPackage(ctx context.Context, c *app.RequestContext) {
	var req importSkillPackageRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.ImportSkillPackage(ctx, req.SpaceID, req.Filename, req.Content, req.IconURI)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ImportRuntimeSkill handles POST /api/super-agent/skills/import-runtime
func ImportRuntimeSkill(ctx context.Context, c *app.RequestContext) {
	var req importRuntimeSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}
	if singleagentApp.SingleAgentSVC == nil {
		httputil.InternalError(ctx, c, fmt.Errorf("single agent service is not available"))
		return
	}

	skill, err := singleagentApp.SingleAgentSVC.ImportSuperAgentRuntimeSkill(ctx, &singleagentApp.SuperAgentRuntimeSkillImportRequest{
		AgentID:      req.AgentID,
		BotID:        req.BotID,
		ConnectorID:  req.ConnectorID,
		Name:         req.Name,
		SkillID:      req.SkillID,
		IconURI:      req.IconURI,
		PublishScope: req.PublishScope,
	})
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ValidateSkillPackage handles POST /api/super-agent/skills/validate-package
func ValidateSkillPackage(ctx context.Context, c *app.RequestContext) {
	var req validateSkillPackageRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	validation, err := skillApp.SkillApplicationSVC.ValidateSkillPackage(ctx, req.Filename, req.Content)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"validation": skillPackageValidationToResponse(validation)},
	})
}

// ExportSkillPackage handles POST /api/super-agent/skills/export
func ExportSkillPackage(ctx context.Context, c *app.RequestContext) {
	var req exportSkillPackageRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	pkg, err := skillApp.SkillApplicationSVC.ExportSkillPackage(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"package": skillPackageExportToResponse(pkg)},
	})
}

// UpsertSkillAsset handles POST /api/super-agent/skills/assets/upsert
func UpsertSkillAsset(ctx context.Context, c *app.RequestContext) {
	var req upsertSkillAssetRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.UpsertSkillAsset(ctx, req.SkillID, req.SpaceID, req.Path, req.Content, req.Mime)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ListSkillAssets handles GET /api/super-agent/skills/assets/list
func ListSkillAssets(ctx context.Context, c *app.RequestContext) {
	var req listSkillAssetsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	assets, err := skillApp.SkillApplicationSVC.ListSkillAssets(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	assetResponses := make([]*skillAssetResponse, 0, len(assets))
	for _, asset := range assets {
		assetResponses = append(assetResponses, skillAssetInfoToResponse(asset))
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"assets": assetResponses},
	})
}

// GetSkillAsset handles GET /api/super-agent/skills/assets/get
func GetSkillAsset(ctx context.Context, c *app.RequestContext) {
	var req getSkillAssetRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	asset, err := skillApp.SkillApplicationSVC.GetSkillAsset(ctx, req.SkillID, req.SpaceID, req.Path)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"asset": skillAssetToResponse(asset)},
	})
}

// DeleteSkillAsset handles POST /api/super-agent/skills/assets/delete
func DeleteSkillAsset(ctx context.Context, c *app.RequestContext) {
	var req deleteSkillAssetRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.DeleteSkillAsset(ctx, req.SkillID, req.SpaceID, req.Path)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// GetSkill handles GET /api/skill/get
func GetSkill(ctx context.Context, c *app.RequestContext) {
	var req getSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.GetSkill(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// UpdateSkill handles POST /api/skill/update
func UpdateSkill(ctx context.Context, c *app.RequestContext) {
	var req updateSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.UpdateSkill(ctx, req.SkillID, req.SpaceID, req.Name, req.Description, req.Prompt, req.IconURI, req.Files)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// DeleteSkill handles POST /api/skill/delete
func DeleteSkill(ctx context.Context, c *app.RequestContext) {
	var req deleteSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	err := skillApp.SkillApplicationSVC.DeleteSkill(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
	})
}

// PublishSkill handles POST /api/skill/publish
func PublishSkill(ctx context.Context, c *app.RequestContext) {
	var req publishSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.PublishSkill(ctx, req.SkillID, req.SpaceID, req.Scope)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ReviewSkill handles POST /api/skill/review
func ReviewSkill(ctx context.Context, c *app.RequestContext) {
	var req reviewSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.ReviewSkill(ctx, req.SkillID, req.Approve, req.Note)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ListPendingReviews handles GET /api/skill/review/pending
func ListPendingReviews(ctx context.Context, c *app.RequestContext) {
	var req listPendingReviewsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skills, total, err := skillApp.SkillApplicationSVC.ListPendingReviews(ctx, req.SpaceID, req.Page, req.PageSize)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	skillResponses := make([]*skillInfoResponse, 0, len(skills))
	for _, s := range skills {
		skillResponses = append(skillResponses, entityToResponse(s))
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"skill_list": skillResponses,
			"total":      total,
		},
	})
}

// InstallMarketplaceSkill handles POST /api/skill/marketplace/install
func InstallMarketplaceSkill(ctx context.Context, c *app.RequestContext) {
	var req installMarketplaceSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.InstallMarketplaceSkill(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// GetMarketplaceSkill handles GET /api/skill/marketplace/get
func GetMarketplaceSkill(ctx context.Context, c *app.RequestContext) {
	var req getMarketplaceSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.GetMarketplaceSkill(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// ListSkills handles GET /api/skill/list
func ListSkills(ctx context.Context, c *app.RequestContext) {
	var req listSkillsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skills, total, err := skillApp.SkillApplicationSVC.ListSkills(ctx, req.SpaceID, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	skillResponses := make([]*skillInfoResponse, 0, len(skills))
	for _, s := range skills {
		skillResponses = append(skillResponses, entityToResponse(s))
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"skill_list": skillResponses,
			"total":      total,
		},
	})
}

// ListMarketplaceSkills handles GET /api/skill/marketplace/list
func ListMarketplaceSkills(ctx context.Context, c *app.RequestContext) {
	var req marketplaceListSkillsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skills, total, err := skillApp.SkillApplicationSVC.ListMarketplaceSkills(ctx, req.SpaceID, req.Scope, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	skillResponses := make([]*skillInfoResponse, 0, len(skills))
	for _, s := range skills {
		skillResponses = append(skillResponses, entityToResponse(s))
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"skill_list": skillResponses,
			"total":      total,
		},
	})
}
