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

package dal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

const (
	tableNameAIProduct             = "ai_product"
	tableNameAIProductVersion      = "ai_product_version"
	tableNameAIProductInstallation = "ai_product_installation"
	tableNameAIProductAuditLog     = "ai_product_audit_log"
	tableNameSessionRuntimeConfig  = "super_agent_session_runtime_config"
)

type productPO struct {
	ID               int64  `gorm:"column:id;primaryKey;autoIncrement:true"`
	ProductID        int64  `gorm:"column:product_id;not null"`
	SpaceID          int64  `gorm:"column:space_id;not null"`
	CreatorID        int64  `gorm:"column:creator_id;not null"`
	Name             string `gorm:"column:name;not null"`
	Description      string `gorm:"column:description"`
	Type             string `gorm:"column:type;not null"`
	Status           string `gorm:"column:status;not null"`
	Visibility       string `gorm:"column:visibility;not null"`
	IconURI          string `gorm:"column:icon_uri;not null"`
	CoverURI         string `gorm:"column:cover_uri;not null"`
	Document         string `gorm:"column:document"`
	Feature          string `gorm:"column:feature"`
	SourceRefType    string `gorm:"column:source_ref_type;not null"`
	SourceRefID      int64  `gorm:"column:source_ref_id;not null"`
	LatestVersion    string `gorm:"column:latest_version;not null"`
	PublishedVersion string `gorm:"column:published_version;not null"`
	Official         bool   `gorm:"column:official;not null"`
	Featured         bool   `gorm:"column:featured;not null"`
	InstallCount     int64  `gorm:"column:install_count;not null"`
	DownloadCount    int64  `gorm:"column:download_count;not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null"`
}

func (productPO) TableName() string {
	return tableNameAIProduct
}

type productVersionPO struct {
	ID              int64  `gorm:"column:id;primaryKey;autoIncrement:true"`
	ProductID       int64  `gorm:"column:product_id;not null"`
	Version         string `gorm:"column:version;not null"`
	SourceVersion   string `gorm:"column:source_version;not null"`
	Status          string `gorm:"column:status;not null"`
	ReviewStatus    string `gorm:"column:review_status;not null"`
	ReviewNote      string `gorm:"column:review_note"`
	ReviewerID      int64  `gorm:"column:reviewer_id;not null"`
	ContentHash     string `gorm:"column:content_hash;not null"`
	FeatureSnapshot string `gorm:"column:feature_snapshot"`
	PublishedAt     int64  `gorm:"column:published_at;not null"`
	CreatedAt       int64  `gorm:"column:created_at;not null"`
	UpdatedAt       int64  `gorm:"column:updated_at;not null"`
}

func (productVersionPO) TableName() string {
	return tableNameAIProductVersion
}

type productInstallationPO struct {
	ID             int64  `gorm:"column:id;primaryKey;autoIncrement:true"`
	InstallationID int64  `gorm:"column:installation_id;not null"`
	ProductID      int64  `gorm:"column:product_id;not null"`
	ProductVersion string `gorm:"column:product_version;not null"`
	TargetSpaceID  int64  `gorm:"column:target_space_id;not null"`
	TargetUserID   int64  `gorm:"column:target_user_id;not null"`
	InstalledBy    int64  `gorm:"column:installed_by;not null"`
	Status         string `gorm:"column:status;not null"`
	InstallMode    string `gorm:"column:install_mode;not null"`
	RuntimeConfig  string `gorm:"column:runtime_config"`
	CreatedAt      int64  `gorm:"column:created_at;not null"`
	UpdatedAt      int64  `gorm:"column:updated_at;not null"`
}

func (productInstallationPO) TableName() string {
	return tableNameAIProductInstallation
}

type auditLogPO struct {
	ID             int64  `gorm:"column:id;primaryKey;autoIncrement:true"`
	AuditID        int64  `gorm:"column:audit_id;not null"`
	ProductID      int64  `gorm:"column:product_id;not null"`
	InstallationID int64  `gorm:"column:installation_id;not null"`
	SpaceID        int64  `gorm:"column:space_id;not null"`
	UserID         int64  `gorm:"column:user_id;not null"`
	Action         string `gorm:"column:action;not null"`
	TargetType     string `gorm:"column:target_type;not null"`
	TargetID       string `gorm:"column:target_id;not null"`
	Detail         string `gorm:"column:detail"`
	CreatedAt      int64  `gorm:"column:created_at;not null"`
}

func (auditLogPO) TableName() string {
	return tableNameAIProductAuditLog
}

type sessionRuntimeConfigPO struct {
	ID               int64  `gorm:"column:id;primaryKey;autoIncrement:true"`
	ConversationID   int64  `gorm:"column:conversation_id;not null"`
	AgentID          int64  `gorm:"column:agent_id;not null"`
	SpaceID          int64  `gorm:"column:space_id;not null"`
	ModelProductID   int64  `gorm:"column:model_product_id;not null"`
	MCPProductIDs    string `gorm:"column:mcp_product_ids"`
	SkillProductIDs  string `gorm:"column:skill_product_ids"`
	ToolPolicy       string `gorm:"column:tool_policy"`
	ContextPolicy    string `gorm:"column:context_policy"`
	ResolvedSnapshot string `gorm:"column:resolved_snapshot"`
	CreatedBy        int64  `gorm:"column:created_by;not null"`
	CreatedAt        int64  `gorm:"column:created_at;not null"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null"`
}

func (sessionRuntimeConfigPO) TableName() string {
	return tableNameSessionRuntimeConfig
}

type ProductDAO struct {
	db    *gorm.DB
	idGen idgen.IDGenerator
}

func NewProductDAO(db *gorm.DB, idGen idgen.IDGenerator) *ProductDAO {
	return &ProductDAO{db: db, idGen: idGen}
}

func (dao *ProductDAO) UpsertProduct(ctx context.Context, product *entity.Product) error {
	if product.ProductID == 0 {
		id, err := dao.idGen.GenID(ctx)
		if err != nil {
			return err
		}
		product.ProductID = id
	}
	now := time.Now().UnixMilli()
	if product.CreatedAt == 0 {
		product.CreatedAt = now
	}
	product.UpdatedAt = now
	po := productToPO(product)
	existing, err := dao.GetProduct(ctx, product.ProductID)
	if err != nil {
		return err
	}
	if existing == nil {
		return dao.db.WithContext(ctx).Create(po).Error
	}
	return dao.db.WithContext(ctx).Model(&productPO{}).Where("product_id = ?", product.ProductID).Updates(productPOToUpdates(po)).Error
}

func (dao *ProductDAO) GetProduct(ctx context.Context, productID int64) (*entity.Product, error) {
	var po productPO
	err := dao.db.WithContext(ctx).Where("product_id = ?", productID).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return productFromPO(&po), nil
}

func (dao *ProductDAO) GetProductBySource(ctx context.Context, sourceType string, sourceID int64) (*entity.Product, error) {
	var po productPO
	err := dao.db.WithContext(ctx).Where("source_ref_type = ? AND source_ref_id = ?", sourceType, sourceID).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return productFromPO(&po), nil
}

func (dao *ProductDAO) ListProducts(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error) {
	if req == nil {
		req = &entity.ListProductsRequest{}
	}
	query := dao.db.WithContext(ctx).Model(&productPO{})
	if req.Type != "" {
		query = query.Where("type = ?", string(req.Type))
	}
	if req.Visibility != "" {
		query = query.Where("visibility = ?", string(req.Visibility))
	}
	if req.Status != "" {
		query = query.Where("status = ?", string(req.Status))
	}
	if req.Keyword != "" {
		like := "%" + strings.TrimSpace(req.Keyword) + "%"
		query = query.Where("(name LIKE ? OR description LIKE ?)", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	var pos []productPO
	if err := query.Order("updated_at DESC, product_id DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	products := make([]*entity.Product, 0, len(pos))
	for i := range pos {
		products = append(products, productFromPO(&pos[i]))
	}
	return &entity.ListProductsResult{Products: products, Total: int32(total)}, nil
}

func (dao *ProductDAO) UpsertVersion(ctx context.Context, version *entity.ProductVersion) error {
	now := time.Now().UnixMilli()
	if version.CreatedAt == 0 {
		version.CreatedAt = now
	}
	version.UpdatedAt = now
	po := versionToPO(version)
	var existing productVersionPO
	err := dao.db.WithContext(ctx).Where("product_id = ? AND version = ?", version.ProductID, version.Version).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dao.db.WithContext(ctx).Create(po).Error
	}
	if err != nil {
		return err
	}
	return dao.db.WithContext(ctx).Model(&productVersionPO{}).
		Where("product_id = ? AND version = ?", version.ProductID, version.Version).
		Updates(productVersionPOToUpdates(po)).Error
}

func (dao *ProductDAO) ListVersions(ctx context.Context, productID int64) ([]*entity.ProductVersion, error) {
	var pos []productVersionPO
	if err := dao.db.WithContext(ctx).Where("product_id = ?", productID).Order("published_at DESC, id DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	versions := make([]*entity.ProductVersion, 0, len(pos))
	for i := range pos {
		versions = append(versions, versionFromPO(&pos[i]))
	}
	return versions, nil
}

func (dao *ProductDAO) InstallProduct(ctx context.Context, installation *entity.ProductInstallation) error {
	if installation.InstallationID == 0 {
		id, err := dao.idGen.GenID(ctx)
		if err != nil {
			return err
		}
		installation.InstallationID = id
	}
	now := time.Now().UnixMilli()
	if installation.CreatedAt == 0 {
		installation.CreatedAt = now
	}
	installation.UpdatedAt = now
	return dao.db.WithContext(ctx).Create(installationToPO(installation)).Error
}

func (dao *ProductDAO) UpdateInstallation(ctx context.Context, installation *entity.ProductInstallation) error {
	installation.UpdatedAt = time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Model(&productInstallationPO{}).
		Where("installation_id = ?", installation.InstallationID).
		Updates(productInstallationPOToUpdates(installationToPO(installation))).Error
}

func (dao *ProductDAO) GetInstallation(ctx context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error) {
	query := dao.db.WithContext(ctx).Where("product_id = ? AND target_space_id = ?", productID, spaceID)
	if userID > 0 {
		query = query.Where("target_user_id IN ?", []int64{0, userID})
	} else {
		query = query.Where("target_user_id = 0")
	}
	var po productInstallationPO
	err := query.Order("target_user_id DESC, updated_at DESC").First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return installationFromPO(&po), nil
}

func (dao *ProductDAO) ListInstallations(ctx context.Context, req *entity.ListInstallationsRequest) ([]*entity.ProductInstallation, error) {
	if req == nil {
		req = &entity.ListInstallationsRequest{}
	}
	query := dao.db.WithContext(ctx).Model(&productInstallationPO{})
	if req.SpaceID > 0 {
		query = query.Where("target_space_id = ?", req.SpaceID)
	}
	if req.UserID > 0 {
		query = query.Where("target_user_id IN ?", []int64{0, req.UserID})
	}
	if req.ProductID > 0 {
		query = query.Where("product_id = ?", req.ProductID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", string(req.Status))
	}
	var pos []productInstallationPO
	if err := query.Order("updated_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	installations := make([]*entity.ProductInstallation, 0, len(pos))
	for i := range pos {
		installations = append(installations, installationFromPO(&pos[i]))
	}
	return installations, nil
}

func (dao *ProductDAO) CreateAudit(ctx context.Context, audit *entity.AuditLog) error {
	if audit.AuditID == 0 {
		id, err := dao.idGen.GenID(ctx)
		if err != nil {
			return err
		}
		audit.AuditID = id
	}
	if audit.CreatedAt == 0 {
		audit.CreatedAt = time.Now().UnixMilli()
	}
	return dao.db.WithContext(ctx).Create(auditToPO(audit)).Error
}

func (dao *ProductDAO) ListAudits(ctx context.Context, req *entity.ListAuditsRequest) ([]*entity.AuditLog, error) {
	if req == nil {
		req = &entity.ListAuditsRequest{}
	}
	query := dao.db.WithContext(ctx).Model(&auditLogPO{})
	if req.ProductID > 0 {
		query = query.Where("product_id = ?", req.ProductID)
	}
	if req.SpaceID > 0 {
		query = query.Where("space_id = ?", req.SpaceID)
	}
	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	var pos []auditLogPO
	if err := query.Order("created_at DESC, id DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	audits := make([]*entity.AuditLog, 0, len(pos))
	for i := range pos {
		audits = append(audits, auditFromPO(&pos[i]))
	}
	return audits, nil
}

func (dao *ProductDAO) UpsertSessionRuntimeConfig(ctx context.Context, config *entity.SessionRuntimeConfig) error {
	now := time.Now().UnixMilli()
	if config.CreatedAt == 0 {
		config.CreatedAt = now
	}
	config.UpdatedAt = now
	po := sessionRuntimeConfigToPO(config)
	var existing sessionRuntimeConfigPO
	err := dao.db.WithContext(ctx).Where("conversation_id = ?", config.ConversationID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dao.db.WithContext(ctx).Create(po).Error
	}
	if err != nil {
		return err
	}
	config.ID = existing.ID
	if config.CreatedAt == now {
		config.CreatedAt = existing.CreatedAt
		po.CreatedAt = existing.CreatedAt
	}
	return dao.db.WithContext(ctx).Model(&sessionRuntimeConfigPO{}).
		Where("conversation_id = ?", config.ConversationID).
		Updates(sessionRuntimeConfigPOToUpdates(po)).Error
}

func (dao *ProductDAO) GetSessionRuntimeConfig(ctx context.Context, conversationID int64) (*entity.SessionRuntimeConfig, error) {
	var po sessionRuntimeConfigPO
	err := dao.db.WithContext(ctx).Where("conversation_id = ?", conversationID).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return sessionRuntimeConfigFromPO(&po), nil
}

func (dao *ProductDAO) DeleteSessionRuntimeConfig(ctx context.Context, conversationID int64) error {
	return dao.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Delete(&sessionRuntimeConfigPO{}).Error
}

func productToPO(do *entity.Product) *productPO {
	return &productPO{
		ID:               do.ID,
		ProductID:        do.ProductID,
		SpaceID:          do.SpaceID,
		CreatorID:        do.CreatorID,
		Name:             do.Name,
		Description:      do.Description,
		Type:             string(do.Type),
		Status:           string(do.Status),
		Visibility:       string(do.Visibility),
		IconURI:          do.IconURI,
		CoverURI:         do.CoverURI,
		Document:         do.Document,
		Feature:          encodeMap(do.Feature),
		SourceRefType:    do.SourceRefType,
		SourceRefID:      do.SourceRefID,
		LatestVersion:    do.LatestVersion,
		PublishedVersion: do.PublishedVersion,
		Official:         do.Official,
		Featured:         do.Featured,
		InstallCount:     do.InstallCount,
		DownloadCount:    do.DownloadCount,
		CreatedAt:        do.CreatedAt,
		UpdatedAt:        do.UpdatedAt,
	}
}

func productFromPO(po *productPO) *entity.Product {
	return &entity.Product{
		ID:               po.ID,
		ProductID:        po.ProductID,
		SpaceID:          po.SpaceID,
		CreatorID:        po.CreatorID,
		Name:             po.Name,
		Description:      po.Description,
		Type:             entity.AIProductType(po.Type),
		Status:           entity.AIProductStatus(po.Status),
		Visibility:       entity.AIProductVisibility(po.Visibility),
		IconURI:          po.IconURI,
		CoverURI:         po.CoverURI,
		Document:         po.Document,
		Feature:          decodeMap(po.Feature),
		SourceRefType:    po.SourceRefType,
		SourceRefID:      po.SourceRefID,
		LatestVersion:    po.LatestVersion,
		PublishedVersion: po.PublishedVersion,
		Official:         po.Official,
		Featured:         po.Featured,
		InstallCount:     po.InstallCount,
		DownloadCount:    po.DownloadCount,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}

func versionToPO(do *entity.ProductVersion) *productVersionPO {
	return &productVersionPO{
		ID:              do.ID,
		ProductID:       do.ProductID,
		Version:         do.Version,
		SourceVersion:   do.SourceVersion,
		Status:          string(do.Status),
		ReviewStatus:    do.ReviewStatus,
		ReviewNote:      do.ReviewNote,
		ReviewerID:      do.ReviewerID,
		ContentHash:     do.ContentHash,
		FeatureSnapshot: encodeMap(do.FeatureSnapshot),
		PublishedAt:     do.PublishedAt,
		CreatedAt:       do.CreatedAt,
		UpdatedAt:       do.UpdatedAt,
	}
}

func versionFromPO(po *productVersionPO) *entity.ProductVersion {
	return &entity.ProductVersion{
		ID:              po.ID,
		ProductID:       po.ProductID,
		Version:         po.Version,
		SourceVersion:   po.SourceVersion,
		Status:          entity.AIProductStatus(po.Status),
		ReviewStatus:    po.ReviewStatus,
		ReviewNote:      po.ReviewNote,
		ReviewerID:      po.ReviewerID,
		ContentHash:     po.ContentHash,
		FeatureSnapshot: decodeMap(po.FeatureSnapshot),
		PublishedAt:     po.PublishedAt,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
}

func installationToPO(do *entity.ProductInstallation) *productInstallationPO {
	return &productInstallationPO{
		ID:             do.ID,
		InstallationID: do.InstallationID,
		ProductID:      do.ProductID,
		ProductVersion: do.ProductVersion,
		TargetSpaceID:  do.TargetSpaceID,
		TargetUserID:   do.TargetUserID,
		InstalledBy:    do.InstalledBy,
		Status:         string(do.Status),
		InstallMode:    do.InstallMode,
		RuntimeConfig:  encodeMap(do.RuntimeConfig),
		CreatedAt:      do.CreatedAt,
		UpdatedAt:      do.UpdatedAt,
	}
}

func installationFromPO(po *productInstallationPO) *entity.ProductInstallation {
	return &entity.ProductInstallation{
		ID:             po.ID,
		InstallationID: po.InstallationID,
		ProductID:      po.ProductID,
		ProductVersion: po.ProductVersion,
		TargetSpaceID:  po.TargetSpaceID,
		TargetUserID:   po.TargetUserID,
		InstalledBy:    po.InstalledBy,
		Status:         entity.AIProductInstallationStatus(po.Status),
		InstallMode:    po.InstallMode,
		RuntimeConfig:  decodeMap(po.RuntimeConfig),
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

func auditToPO(do *entity.AuditLog) *auditLogPO {
	return &auditLogPO{
		ID:             do.ID,
		AuditID:        do.AuditID,
		ProductID:      do.ProductID,
		InstallationID: do.InstallationID,
		SpaceID:        do.SpaceID,
		UserID:         do.UserID,
		Action:         do.Action,
		TargetType:     do.TargetType,
		TargetID:       do.TargetID,
		Detail:         encodeMap(do.Detail),
		CreatedAt:      do.CreatedAt,
	}
}

func auditFromPO(po *auditLogPO) *entity.AuditLog {
	return &entity.AuditLog{
		ID:             po.ID,
		AuditID:        po.AuditID,
		ProductID:      po.ProductID,
		InstallationID: po.InstallationID,
		SpaceID:        po.SpaceID,
		UserID:         po.UserID,
		Action:         po.Action,
		TargetType:     po.TargetType,
		TargetID:       po.TargetID,
		Detail:         decodeMap(po.Detail),
		CreatedAt:      po.CreatedAt,
	}
}

func sessionRuntimeConfigToPO(do *entity.SessionRuntimeConfig) *sessionRuntimeConfigPO {
	return &sessionRuntimeConfigPO{
		ID:               do.ID,
		ConversationID:   do.ConversationID,
		AgentID:          do.AgentID,
		SpaceID:          do.SpaceID,
		ModelProductID:   do.ModelProductID,
		MCPProductIDs:    encodeInt64Slice(do.MCPProductIDs),
		SkillProductIDs:  encodeInt64Slice(do.SkillProductIDs),
		ToolPolicy:       encodeMap(do.ToolPolicy),
		ContextPolicy:    encodeMap(do.ContextPolicy),
		ResolvedSnapshot: encodeMap(do.ResolvedSnapshot),
		CreatedBy:        do.CreatedBy,
		CreatedAt:        do.CreatedAt,
		UpdatedAt:        do.UpdatedAt,
	}
}

func sessionRuntimeConfigFromPO(po *sessionRuntimeConfigPO) *entity.SessionRuntimeConfig {
	return &entity.SessionRuntimeConfig{
		ID:               po.ID,
		ConversationID:   po.ConversationID,
		AgentID:          po.AgentID,
		SpaceID:          po.SpaceID,
		ModelProductID:   po.ModelProductID,
		MCPProductIDs:    decodeInt64Slice(po.MCPProductIDs),
		SkillProductIDs:  decodeInt64Slice(po.SkillProductIDs),
		ToolPolicy:       decodeMap(po.ToolPolicy),
		ContextPolicy:    decodeMap(po.ContextPolicy),
		ResolvedSnapshot: decodeMap(po.ResolvedSnapshot),
		CreatedBy:        po.CreatedBy,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}

func productPOToUpdates(po *productPO) map[string]any {
	return map[string]any{
		"space_id":          po.SpaceID,
		"creator_id":        po.CreatorID,
		"name":              po.Name,
		"description":       po.Description,
		"type":              po.Type,
		"status":            po.Status,
		"visibility":        po.Visibility,
		"icon_uri":          po.IconURI,
		"cover_uri":         po.CoverURI,
		"document":          po.Document,
		"feature":           po.Feature,
		"source_ref_type":   po.SourceRefType,
		"source_ref_id":     po.SourceRefID,
		"latest_version":    po.LatestVersion,
		"published_version": po.PublishedVersion,
		"official":          po.Official,
		"featured":          po.Featured,
		"install_count":     po.InstallCount,
		"download_count":    po.DownloadCount,
		"updated_at":        po.UpdatedAt,
	}
}

func productVersionPOToUpdates(po *productVersionPO) map[string]any {
	return map[string]any{
		"source_version":   po.SourceVersion,
		"status":           po.Status,
		"review_status":    po.ReviewStatus,
		"review_note":      po.ReviewNote,
		"reviewer_id":      po.ReviewerID,
		"content_hash":     po.ContentHash,
		"feature_snapshot": po.FeatureSnapshot,
		"published_at":     po.PublishedAt,
		"updated_at":       po.UpdatedAt,
	}
}

func productInstallationPOToUpdates(po *productInstallationPO) map[string]any {
	return map[string]any{
		"product_version": po.ProductVersion,
		"target_space_id": po.TargetSpaceID,
		"target_user_id":  po.TargetUserID,
		"installed_by":    po.InstalledBy,
		"status":          po.Status,
		"install_mode":    po.InstallMode,
		"runtime_config":  po.RuntimeConfig,
		"updated_at":      po.UpdatedAt,
	}
}

func sessionRuntimeConfigPOToUpdates(po *sessionRuntimeConfigPO) map[string]any {
	return map[string]any{
		"agent_id":          po.AgentID,
		"space_id":          po.SpaceID,
		"model_product_id":  po.ModelProductID,
		"mcp_product_ids":   po.MCPProductIDs,
		"skill_product_ids": po.SkillProductIDs,
		"tool_policy":       po.ToolPolicy,
		"context_policy":    po.ContextPolicy,
		"resolved_snapshot": po.ResolvedSnapshot,
		"created_by":        po.CreatedBy,
		"created_at":        po.CreatedAt,
		"updated_at":        po.UpdatedAt,
	}
}

func encodeMap(m map[string]any) string {
	// Return valid empty JSON (not "") for empty/nil maps: these strings are
	// written into JSON-typed columns (feature, runtime_config, …) where an
	// empty string is rejected as invalid JSON. decodeMap handles "{}" and "".
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func decodeMap(s string) map[string]any {
	if s == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

func encodeInt64Slice(v []int64) string {
	if len(v) == 0 {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeInt64Slice(s string) []int64 {
	if s == "" {
		return nil
	}
	var v []int64
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil
	}
	return v
}
