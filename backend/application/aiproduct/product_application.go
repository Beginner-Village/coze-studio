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

package aiproduct

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	productentity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	productrepo "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/repository"
	productservice "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/service"
	skillentity "github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	skillservice "github.com/ynet-dev/ynet-studio/backend/domain/skill/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

var ProductApplicationSVC *ProductApplicationService

type ServiceComponents struct {
	IDGen    idgen.IDGenerator
	DB       *gorm.DB
	SkillSVC skillservice.SkillService
}

type ProductApplicationService struct {
	DomainSVC   productservice.Service
	ProductRepo productrepo.Repository
	SkillSVC    skillservice.SkillService
}

func InitService(c *ServiceComponents) *ProductApplicationService {
	repo := productrepo.NewRepository(c.DB, c.IDGen)
	domainSVC := productservice.NewService(&productservice.Components{ProductRepo: repo})
	ProductApplicationSVC = &ProductApplicationService{
		DomainSVC:   domainSVC,
		ProductRepo: repo,
		SkillSVC:    c.SkillSVC,
	}
	return ProductApplicationSVC
}

func (s *ProductApplicationService) ListMarketplaceProducts(ctx context.Context, req *productentity.ListProductsRequest) (*productentity.ListProductsResult, error) {
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		req = &productentity.ListProductsRequest{}
	}
	req.UserID = userID
	return s.DomainSVC.ListMarketplace(ctx, req)
}

func (s *ProductApplicationService) GetVisibleProduct(ctx context.Context, productID, spaceID int64) (*productentity.Product, error) {
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return nil, err
	}
	return s.DomainSVC.GetVisibleProduct(ctx, productID, spaceID, userID)
}

func (s *ProductApplicationService) InstallProduct(ctx context.Context, productID, spaceID int64, version string) (*productentity.ProductInstallation, error) {
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireProductSpaceManager(ctx, spaceID, userID); err != nil {
		return nil, err
	}
	return s.DomainSVC.Install(ctx, productID, spaceID, userID, version)
}

func (s *ProductApplicationService) UpgradeProduct(ctx context.Context, productID, spaceID int64) (*productentity.ProductInstallation, error) {
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireProductSpaceManager(ctx, spaceID, userID); err != nil {
		return nil, err
	}
	return s.DomainSVC.Upgrade(ctx, productID, spaceID, userID)
}

func (s *ProductApplicationService) UninstallProduct(ctx context.Context, productID, spaceID int64) error {
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return err
	}
	if err := requireProductSpaceManager(ctx, spaceID, userID); err != nil {
		return err
	}
	return s.DomainSVC.Uninstall(ctx, productID, spaceID, userID)
}

func (s *ProductApplicationService) GetSessionRuntimeConfig(ctx context.Context, conversationID int64) (*productentity.SessionRuntimeConfig, error) {
	if conversationID <= 0 {
		return nil, fmt.Errorf("conversation_id is required")
	}
	return s.ProductRepo.GetSessionRuntimeConfig(ctx, conversationID)
}

func (s *ProductApplicationService) UpsertSessionRuntimeConfig(ctx context.Context, config *productentity.SessionRuntimeConfig) (*productentity.SessionRuntimeConfig, error) {
	if config == nil {
		return nil, fmt.Errorf("runtime config is required")
	}
	if config.ConversationID <= 0 {
		return nil, fmt.Errorf("conversation_id is required")
	}
	if config.SpaceID <= 0 {
		return nil, fmt.Errorf("space_id is required")
	}
	if config.AgentID <= 0 {
		return nil, fmt.Errorf("agent_id is required")
	}
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return nil, err
	}
	config.CreatedBy = userID
	config.MCPProductIDs = dedupePositiveInt64s(config.MCPProductIDs)
	config.SkillProductIDs = dedupePositiveInt64s(config.SkillProductIDs)
	snapshot, err := s.ResolveSessionRuntimeConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	config.ResolvedSnapshot = snapshot
	existing, err := s.ProductRepo.GetSessionRuntimeConfig(ctx, config.ConversationID)
	if err != nil {
		return nil, err
	}
	if err := s.ProductRepo.UpsertSessionRuntimeConfig(ctx, config); err != nil {
		return nil, err
	}
	action := productentity.AIProductAuditActionSessionBind
	if existing != nil {
		action = productentity.AIProductAuditActionSessionUpdate
	}
	_ = s.DomainSVC.RecordAudit(ctx, &productentity.AuditLog{
		SpaceID:    config.SpaceID,
		UserID:     userID,
		Action:     action,
		TargetType: "conversation",
		TargetID:   strconv.FormatInt(config.ConversationID, 10),
		Detail: map[string]any{
			"agent_id":          config.AgentID,
			"model_product_id":  config.ModelProductID,
			"mcp_product_ids":   config.MCPProductIDs,
			"skill_product_ids": config.SkillProductIDs,
		},
	})
	return s.ProductRepo.GetSessionRuntimeConfig(ctx, config.ConversationID)
}

func (s *ProductApplicationService) DeleteSessionRuntimeConfig(ctx context.Context, conversationID int64) error {
	if conversationID <= 0 {
		return fmt.Errorf("conversation_id is required")
	}
	return s.ProductRepo.DeleteSessionRuntimeConfig(ctx, conversationID)
}

func (s *ProductApplicationService) ResolveSessionRuntimeConfig(ctx context.Context, config *productentity.SessionRuntimeConfig) (map[string]any, error) {
	userID, err := currentProductUserID(ctx)
	if err != nil {
		return nil, err
	}
	snapshot := map[string]any{
		"conversation_id": config.ConversationID,
		"agent_id":        config.AgentID,
		"space_id":        config.SpaceID,
	}
	if config.ModelProductID > 0 {
		model, err := s.resolveRuntimeProduct(ctx, config.ModelProductID, config.SpaceID, userID, productentity.AIProductTypeModel)
		if err != nil {
			return nil, err
		}
		snapshot["model"] = model
	}
	mcpProducts, err := s.resolveRuntimeProducts(ctx, config.MCPProductIDs, config.SpaceID, userID, productentity.AIProductTypeMCPServer)
	if err != nil {
		return nil, err
	}
	if len(mcpProducts) > 0 {
		snapshot["mcp_servers"] = mcpProducts
	}
	skillProducts, err := s.resolveRuntimeProducts(ctx, config.SkillProductIDs, config.SpaceID, userID, productentity.AIProductTypeStandardSkill)
	if err != nil {
		return nil, err
	}
	if len(skillProducts) > 0 {
		snapshot["skills"] = skillProducts
	}
	if len(config.ToolPolicy) > 0 {
		snapshot["tool_policy"] = config.ToolPolicy
	}
	if len(config.ContextPolicy) > 0 {
		snapshot["context_policy"] = config.ContextPolicy
	}
	return snapshot, nil
}

func (s *ProductApplicationService) SyncSkillProduct(ctx context.Context, skillID int64) error {
	if s == nil || s.DomainSVC == nil || s.ProductRepo == nil || s.SkillSVC == nil {
		return fmt.Errorf("ai product service is not initialized")
	}
	if skillID <= 0 {
		return fmt.Errorf("skill_id is required")
	}
	skill, err := s.SkillSVC.GetSkill(ctx, skillID)
	if err != nil {
		return err
	}
	if skill == nil {
		return fmt.Errorf("skill %d not found", skillID)
	}
	existing, err := s.ProductRepo.GetProductBySource(ctx, productentity.SourceRefTypeSkill, skillID)
	if err != nil {
		return err
	}
	product := productFromSkill(skill, existing)
	version := productVersionFromSkill(skill)
	_, err = s.DomainSVC.SyncProduct(ctx, product, version)
	return err
}

func productFromSkill(skill *skillentity.Skill, existing *productentity.Product) *productentity.Product {
	product := &productentity.Product{}
	if existing != nil {
		cp := *existing
		product = &cp
	}
	product.SpaceID = skill.SpaceID
	product.CreatorID = skill.CreatorID
	product.Name = skill.Name
	product.Description = skill.Description
	product.Type = productentity.AIProductTypeStandardSkill
	product.Status = skillProductStatus(skill)
	product.Visibility = skillProductVisibility(skill.PublishScope)
	product.IconURI = skill.IconURI
	product.Feature = skillProductFeature(skill)
	product.SourceRefType = productentity.SourceRefTypeSkill
	product.SourceRefID = skill.SkillID
	product.LatestVersion = versionString(skill.Version)
	product.PublishedVersion = versionString(skill.PublishedVersion)
	return product
}

func productVersionFromSkill(skill *skillentity.Skill) *productentity.ProductVersion {
	version := skill.Version
	if version <= 0 {
		version = skill.PublishedVersion
	}
	if version <= 0 {
		return nil
	}
	return &productentity.ProductVersion{
		Version:         strconv.FormatInt(version, 10),
		SourceVersion:   strconv.FormatInt(version, 10),
		Status:          skillProductStatus(skill),
		ReviewStatus:    skillReviewStatusString(skill.ReviewStatus),
		ReviewNote:      skill.ReviewNote,
		ReviewerID:      skill.ReviewerID,
		FeatureSnapshot: skillProductFeature(skill),
		PublishedAt:     skill.PublishedAt,
	}
}

func skillProductStatus(skill *skillentity.Skill) productentity.AIProductStatus {
	if skill.PublishScope == skillentity.SkillPublishScopeGlobal {
		switch skill.ReviewStatus {
		case skillentity.SkillReviewStatusPending:
			return productentity.AIProductStatusReviewing
		case skillentity.SkillReviewStatusRejected:
			return productentity.AIProductStatusArchived
		}
	}
	if skill.PublishScope != skillentity.SkillPublishScopePrivate && skill.PublishedVersion > 0 && skill.ReviewStatus == skillentity.SkillReviewStatusApproved {
		return productentity.AIProductStatusPublished
	}
	return productentity.AIProductStatusDraft
}

func skillProductVisibility(scope int8) productentity.AIProductVisibility {
	switch scope {
	case skillentity.SkillPublishScopeGlobal:
		return productentity.AIProductVisibilityGlobal
	case skillentity.SkillPublishScopeSpace:
		return productentity.AIProductVisibilitySpace
	default:
		return productentity.AIProductVisibilityPrivate
	}
}

func skillProductFeature(skill *skillentity.Skill) map[string]any {
	fileCount := int64(len(skill.Files))
	assetCount := int64(0)
	hasScripts := false
	for relPath := range skill.Files {
		if strings.HasPrefix(relPath, "assets/") {
			assetCount++
		}
		if strings.HasPrefix(relPath, "scripts/") {
			hasScripts = true
		}
	}
	capabilities := []string{"standard_skill_package"}
	if hasScripts {
		capabilities = append(capabilities, "execute_scripts")
	}
	sort.Strings(capabilities)
	return map[string]any{
		"skill_id":        skill.SkillID,
		"skill_version":   skill.Version,
		"file_count":      fileCount,
		"asset_count":     assetCount,
		"entry_file":      "SKILL.md",
		"category":        skill.Metadata.Category,
		"tags":            cloneStrings(skill.Metadata.Tags),
		"platforms":       cloneStrings(skill.Metadata.Platforms),
		"capabilities":    capabilities,
		"publish_scope":   skill.PublishScope,
		"review_status":   skill.ReviewStatus,
		"source_ref_type": productentity.SourceRefTypeSkill,
	}
}

func skillReviewStatusString(status int8) string {
	switch status {
	case skillentity.SkillReviewStatusPending:
		return "pending"
	case skillentity.SkillReviewStatusApproved:
		return "approved"
	case skillentity.SkillReviewStatusRejected:
		return "rejected"
	default:
		return ""
	}
}

func versionString(v int64) string {
	if v <= 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

func (s *ProductApplicationService) resolveRuntimeProducts(ctx context.Context, productIDs []int64, spaceID, userID int64, expectedType productentity.AIProductType) ([]map[string]any, error) {
	products := make([]map[string]any, 0, len(productIDs))
	for _, productID := range productIDs {
		product, err := s.resolveRuntimeProduct(ctx, productID, spaceID, userID, expectedType)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (s *ProductApplicationService) resolveRuntimeProduct(ctx context.Context, productID, spaceID, userID int64, expectedType productentity.AIProductType) (map[string]any, error) {
	product, err := s.DomainSVC.GetVisibleProduct(ctx, productID, spaceID, userID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, fmt.Errorf("product %d is not visible in space %d", productID, spaceID)
	}
	if product.Type != expectedType {
		return nil, fmt.Errorf("product %d type %s does not match %s", productID, product.Type, expectedType)
	}
	installation, err := s.ProductRepo.GetInstallation(ctx, productID, spaceID, userID)
	if err != nil {
		return nil, err
	}
	if installation == nil || installation.Status != productentity.AIProductInstallationActive {
		return nil, fmt.Errorf("product %d is not installed in space %d", productID, spaceID)
	}
	return map[string]any{
		"product_id":        product.ProductID,
		"name":              product.Name,
		"type":              string(product.Type),
		"visibility":        string(product.Visibility),
		"version":           installation.ProductVersion,
		"source_ref_type":   product.SourceRefType,
		"source_ref_id":     product.SourceRefID,
		"feature":           product.Feature,
		"installation_id":   installation.InstallationID,
		"installation_mode": installation.InstallMode,
	}, nil
}

func dedupePositiveInt64s(in []int64) []int64 {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(in))
	out := make([]int64, 0, len(in))
	for _, v := range in {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func currentProductUserID(ctx context.Context) (int64, error) {
	if userID := ctxutil.GetUIDFromCtx(ctx); userID != nil {
		return *userID, nil
	}
	apiAuth := ctxutil.GetApiAuthFromCtx(ctx)
	if apiAuth != nil && apiAuth.UserID != 0 {
		return apiAuth.UserID, nil
	}
	return 0, fmt.Errorf("user context is required")
}

func requireProductSpaceManager(ctx context.Context, spaceID, userID int64) error {
	if spaceID <= 0 {
		return fmt.Errorf("space_id is required")
	}
	if userID <= 0 {
		return fmt.Errorf("user context is required")
	}
	userSVC := crossuser.DefaultSVC()
	if userSVC == nil {
		return fmt.Errorf("space manager permission service is not initialized")
	}
	perm, err := userSVC.CheckSpacePermission(ctx, spaceID, userID)
	if err != nil {
		return err
	}
	if perm == nil || !perm.CanManage {
		return fmt.Errorf("space manager role required")
	}
	return nil
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
