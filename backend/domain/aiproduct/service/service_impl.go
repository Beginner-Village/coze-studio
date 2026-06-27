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

package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/repository"
)

type productServiceImpl struct {
	repo repository.Repository
}

func (s *productServiceImpl) SyncProduct(ctx context.Context, product *entity.Product, version *entity.ProductVersion) (*entity.Product, error) {
	if product == nil {
		return nil, fmt.Errorf("product is required")
	}
	if product.Type == "" {
		return nil, fmt.Errorf("product type is required")
	}
	if product.Name == "" {
		return nil, fmt.Errorf("product name is required")
	}
	if product.Status == "" {
		product.Status = entity.AIProductStatusDraft
	}
	if product.Visibility == "" {
		product.Visibility = entity.AIProductVisibilityPrivate
	}
	if err := s.repo.UpsertProduct(ctx, product); err != nil {
		return nil, err
	}
	if version != nil {
		version.ProductID = product.ProductID
		if err := s.repo.UpsertVersion(ctx, version); err != nil {
			return nil, err
		}
	}
	return s.repo.GetProduct(ctx, product.ProductID)
}

func (s *productServiceImpl) ListMarketplace(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error) {
	if req == nil {
		req = &entity.ListProductsRequest{}
	}
	all, err := s.repo.ListProducts(ctx, req)
	if err != nil {
		return nil, err
	}
	if all == nil {
		return &entity.ListProductsResult{}, nil
	}
	products := make([]*entity.Product, 0, len(all.Products))
	for _, product := range all.Products {
		if isMarketplaceVisible(product, req.SpaceID) {
			cp := *product
			products = append(products, &cp)
		}
	}
	sort.SliceStable(products, func(i, j int) bool {
		leftPriority := marketplacePriority(products[i], req.SpaceID)
		rightPriority := marketplacePriority(products[j], req.SpaceID)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if products[i].UpdatedAt != products[j].UpdatedAt {
			return products[i].UpdatedAt > products[j].UpdatedAt
		}
		return products[i].ProductID > products[j].ProductID
	})
	return &entity.ListProductsResult{Products: products, Total: int32(len(products))}, nil
}

func (s *productServiceImpl) GetVisibleProduct(ctx context.Context, productID, spaceID, userID int64) (*entity.Product, error) {
	product, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, nil
	}
	if canSeeProduct(product, spaceID, userID) {
		return product, nil
	}
	return nil, fmt.Errorf("product %d not found or access denied", productID)
}

func (s *productServiceImpl) Install(ctx context.Context, productID, spaceID, userID int64, version string) (*entity.ProductInstallation, error) {
	if productID <= 0 || spaceID <= 0 || userID <= 0 {
		return nil, fmt.Errorf("product_id, space_id and user_id are required")
	}
	product, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !isInstallable(product, spaceID, userID) {
		return nil, fmt.Errorf("product %d is not installable", productID)
	}
	if version == "" {
		version = product.PublishedVersion
	}
	if version == "" {
		return nil, fmt.Errorf("product %d has no published version", productID)
	}

	installation, err := s.repo.GetInstallation(ctx, productID, spaceID, 0)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	if installation == nil {
		installation = &entity.ProductInstallation{
			ProductID:      productID,
			TargetSpaceID:  spaceID,
			TargetUserID:   0,
			InstalledBy:    userID,
			ProductVersion: version,
			Status:         entity.AIProductInstallationActive,
			InstallMode:    "space",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.repo.InstallProduct(ctx, installation); err != nil {
			return nil, err
		}
	} else {
		installation.ProductVersion = version
		installation.Status = entity.AIProductInstallationActive
		installation.InstalledBy = userID
		installation.UpdatedAt = now
		if err := s.repo.UpdateInstallation(ctx, installation); err != nil {
			return nil, err
		}
	}
	_ = s.recordProductAudit(ctx, product, installation, userID, entity.AIProductAuditActionInstall, map[string]any{
		"to_version":   version,
		"product_type": string(product.Type),
		"visibility":   string(product.Visibility),
	})
	return installation, nil
}

func (s *productServiceImpl) Uninstall(ctx context.Context, productID, spaceID, userID int64) error {
	if productID <= 0 || spaceID <= 0 || userID <= 0 {
		return fmt.Errorf("product_id, space_id and user_id are required")
	}
	product, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		return err
	}
	installation, err := s.repo.GetInstallation(ctx, productID, spaceID, 0)
	if err != nil {
		return err
	}
	if installation == nil {
		return fmt.Errorf("product %d is not installed", productID)
	}
	installation.Status = entity.AIProductInstallationUninstalled
	installation.UpdatedAt = time.Now().UnixMilli()
	if err := s.repo.UpdateInstallation(ctx, installation); err != nil {
		return err
	}
	_ = s.recordProductAudit(ctx, product, installation, userID, entity.AIProductAuditActionUninstall, map[string]any{
		"from_version": installation.ProductVersion,
		"product_type": productTypeString(product),
	})
	return nil
}

func (s *productServiceImpl) Upgrade(ctx context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error) {
	if productID <= 0 || spaceID <= 0 || userID <= 0 {
		return nil, fmt.Errorf("product_id, space_id and user_id are required")
	}
	product, err := s.repo.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !isInstallable(product, spaceID, userID) {
		return nil, fmt.Errorf("product %d is not installable", productID)
	}
	if product.PublishedVersion == "" {
		return nil, fmt.Errorf("product %d has no published version", productID)
	}
	installation, err := s.repo.GetInstallation(ctx, productID, spaceID, 0)
	if err != nil {
		return nil, err
	}
	if installation == nil || installation.Status == entity.AIProductInstallationUninstalled {
		return nil, fmt.Errorf("product %d is not installed", productID)
	}
	fromVersion := installation.ProductVersion
	installation.ProductVersion = product.PublishedVersion
	installation.Status = entity.AIProductInstallationActive
	installation.InstalledBy = userID
	installation.UpdatedAt = time.Now().UnixMilli()
	if err := s.repo.UpdateInstallation(ctx, installation); err != nil {
		return nil, err
	}
	_ = s.recordProductAudit(ctx, product, installation, userID, entity.AIProductAuditActionUpgrade, map[string]any{
		"from_version": fromVersion,
		"to_version":   product.PublishedVersion,
		"product_type": string(product.Type),
	})
	return installation, nil
}

func (s *productServiceImpl) ListInstalled(ctx context.Context, spaceID, userID int64, productType entity.AIProductType) ([]*entity.ProductInstallation, error) {
	return s.repo.ListInstallations(ctx, &entity.ListInstallationsRequest{
		SpaceID: spaceID,
		UserID:  userID,
		Type:    productType,
		Status:  entity.AIProductInstallationActive,
	})
}

func (s *productServiceImpl) RecordAudit(ctx context.Context, audit *entity.AuditLog) error {
	if audit == nil {
		return fmt.Errorf("audit is required")
	}
	if audit.CreatedAt == 0 {
		audit.CreatedAt = time.Now().UnixMilli()
	}
	return s.repo.CreateAudit(ctx, audit)
}

func (s *productServiceImpl) recordProductAudit(ctx context.Context, product *entity.Product, installation *entity.ProductInstallation, userID int64, action string, detail map[string]any) error {
	if detail == nil {
		detail = map[string]any{}
	}
	productID := int64(0)
	spaceID := int64(0)
	if product != nil {
		productID = product.ProductID
		if _, ok := detail["product_type"]; !ok {
			detail["product_type"] = string(product.Type)
		}
		if _, ok := detail["visibility"]; !ok {
			detail["visibility"] = string(product.Visibility)
		}
	}
	installationID := int64(0)
	if installation != nil {
		installationID = installation.InstallationID
		spaceID = installation.TargetSpaceID
	}
	return s.RecordAudit(ctx, &entity.AuditLog{
		ProductID:      productID,
		InstallationID: installationID,
		SpaceID:        spaceID,
		UserID:         userID,
		Action:         action,
		TargetType:     "ai_product",
		TargetID:       strconv.FormatInt(productID, 10),
		Detail:         detail,
	})
}

func isInstallable(product *entity.Product, spaceID, userID int64) bool {
	if product == nil || product.Status != entity.AIProductStatusPublished || product.PublishedVersion == "" {
		return false
	}
	switch product.Visibility {
	case entity.AIProductVisibilityGlobal:
		return true
	case entity.AIProductVisibilitySpace:
		return product.SpaceID == spaceID
	case entity.AIProductVisibilityPrivate:
		return product.CreatorID == userID
	default:
		return false
	}
}

func canSeeProduct(product *entity.Product, spaceID, userID int64) bool {
	if product == nil {
		return false
	}
	if product.CreatorID == userID && userID > 0 {
		return true
	}
	if product.Status != entity.AIProductStatusPublished {
		return false
	}
	switch product.Visibility {
	case entity.AIProductVisibilityGlobal:
		return true
	case entity.AIProductVisibilitySpace:
		return product.SpaceID == spaceID
	default:
		return false
	}
}

func isMarketplaceVisible(product *entity.Product, spaceID int64) bool {
	if product == nil || product.Status != entity.AIProductStatusPublished {
		return false
	}
	switch product.Visibility {
	case entity.AIProductVisibilityGlobal:
		return true
	case entity.AIProductVisibilitySpace:
		return product.SpaceID == spaceID
	default:
		return false
	}
}

func marketplacePriority(product *entity.Product, spaceID int64) int {
	if product.Visibility == entity.AIProductVisibilitySpace && product.SpaceID == spaceID {
		return 0
	}
	if product.Visibility == entity.AIProductVisibilityGlobal {
		return 1
	}
	return 2
}

func productTypeString(product *entity.Product) string {
	if product == nil {
		return ""
	}
	return string(product.Type)
}
