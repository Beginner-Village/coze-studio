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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
)

func TestServiceInstallRejectsUnpublishedGlobalProduct(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryProductRepo()
	repo.products[100] = &entity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusReviewing,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "1",
	}
	svc := NewService(&Components{ProductRepo: repo})

	installed, err := svc.Install(ctx, 100, 1, 42, "")

	require.Error(t, err)
	assert.Nil(t, installed)
	assert.Contains(t, err.Error(), "not installable")
	assert.Empty(t, repo.audits)
}

func TestServiceInstallPinsPublishedVersion(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryProductRepo()
	repo.products[100] = &entity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "4",
		LatestVersion:    "5",
	}
	svc := NewService(&Components{ProductRepo: repo})

	installed, err := svc.Install(ctx, 100, 1, 42, "")

	require.NoError(t, err)
	require.NotNil(t, installed)
	assert.Equal(t, int64(100), installed.ProductID)
	assert.Equal(t, int64(1), installed.TargetSpaceID)
	assert.Equal(t, int64(42), installed.InstalledBy)
	assert.Equal(t, "4", installed.ProductVersion)
	assert.Equal(t, entity.AIProductInstallationActive, installed.Status)
	require.Len(t, repo.audits, 1)
	assert.Equal(t, entity.AIProductAuditActionInstall, repo.audits[0].Action)
}

func TestServiceUpgradeMovesInstallationToLatestPublishedVersion(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryProductRepo()
	repo.products[100] = &entity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "5",
	}
	repo.installations[installKey(100, 1, 0)] = &entity.ProductInstallation{
		InstallationID: 1000,
		ProductID:      100,
		ProductVersion: "4",
		TargetSpaceID:  1,
		InstalledBy:    42,
		Status:         entity.AIProductInstallationActive,
	}
	svc := NewService(&Components{ProductRepo: repo})

	upgraded, err := svc.Upgrade(ctx, 100, 1, 42)

	require.NoError(t, err)
	require.NotNil(t, upgraded)
	assert.Equal(t, "5", upgraded.ProductVersion)
	assert.Equal(t, entity.AIProductInstallationActive, upgraded.Status)
	require.Len(t, repo.audits, 1)
	assert.Equal(t, entity.AIProductAuditActionUpgrade, repo.audits[0].Action)
	assert.Equal(t, "4", repo.audits[0].Detail["from_version"])
	assert.Equal(t, "5", repo.audits[0].Detail["to_version"])
}

func TestServiceUninstallMarksInstallationInactive(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryProductRepo()
	repo.products[100] = &entity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "4",
	}
	repo.installations[installKey(100, 1, 0)] = &entity.ProductInstallation{
		InstallationID: 1000,
		ProductID:      100,
		ProductVersion: "4",
		TargetSpaceID:  1,
		InstalledBy:    42,
		Status:         entity.AIProductInstallationActive,
	}
	svc := NewService(&Components{ProductRepo: repo})

	err := svc.Uninstall(ctx, 100, 1, 42)

	require.NoError(t, err)
	got := repo.installations[installKey(100, 1, 0)]
	require.NotNil(t, got)
	assert.Equal(t, "4", got.ProductVersion)
	assert.Equal(t, entity.AIProductInstallationUninstalled, got.Status)
	require.Len(t, repo.audits, 1)
	assert.Equal(t, entity.AIProductAuditActionUninstall, repo.audits[0].Action)
}

func TestServiceListMarketplaceHidesPrivateAndReviewingProducts(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryProductRepo()
	repo.products[100] = &entity.Product{
		ProductID:        100,
		Name:             "Published global",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "1",
	}
	repo.products[101] = &entity.Product{
		ProductID:        101,
		Name:             "Space product",
		SpaceID:          1,
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilitySpace,
		PublishedVersion: "1",
	}
	repo.products[102] = &entity.Product{
		ProductID:        102,
		Name:             "Reviewing global",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusReviewing,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "1",
	}
	repo.products[103] = &entity.Product{
		ProductID:        103,
		Name:             "Private product",
		SpaceID:          1,
		CreatorID:        42,
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilityPrivate,
		PublishedVersion: "1",
	}
	svc := NewService(&Components{ProductRepo: repo})

	listed, err := svc.ListMarketplace(ctx, &entity.ListProductsRequest{
		SpaceID:  1,
		UserID:   42,
		Type:     entity.AIProductTypeStandardSkill,
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	require.NotNil(t, listed)
	require.Len(t, listed.Products, 2)
	assert.Equal(t, int32(2), listed.Total)
	assert.Equal(t, int64(101), listed.Products[0].ProductID)
	assert.Equal(t, int64(100), listed.Products[1].ProductID)
}

func TestServiceAuditRecordsInstallUpgradeUninstall(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryProductRepo()
	repo.products[100] = &entity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             entity.AIProductTypeStandardSkill,
		Status:           entity.AIProductStatusPublished,
		Visibility:       entity.AIProductVisibilityGlobal,
		PublishedVersion: "4",
	}
	svc := NewService(&Components{ProductRepo: repo})

	_, err := svc.Install(ctx, 100, 1, 42, "")
	require.NoError(t, err)
	repo.products[100].PublishedVersion = "5"
	_, err = svc.Upgrade(ctx, 100, 1, 42)
	require.NoError(t, err)
	err = svc.Uninstall(ctx, 100, 1, 42)
	require.NoError(t, err)

	require.Len(t, repo.audits, 3)
	assert.Equal(t, entity.AIProductAuditActionInstall, repo.audits[0].Action)
	assert.Equal(t, entity.AIProductAuditActionUpgrade, repo.audits[1].Action)
	assert.Equal(t, entity.AIProductAuditActionUninstall, repo.audits[2].Action)
}

type memoryProductRepo struct {
	products      map[int64]*entity.Product
	versions      map[int64][]*entity.ProductVersion
	installations map[string]*entity.ProductInstallation
	audits        []*entity.AuditLog
	nextID        int64
}

func newMemoryProductRepo() *memoryProductRepo {
	return &memoryProductRepo{
		products:      map[int64]*entity.Product{},
		versions:      map[int64][]*entity.ProductVersion{},
		installations: map[string]*entity.ProductInstallation{},
		audits:        []*entity.AuditLog{},
		nextID:        9000,
	}
}

func installKey(productID, spaceID, userID int64) string {
	return entity.InstallationKey(productID, spaceID, userID)
}

func (m *memoryProductRepo) UpsertProduct(_ context.Context, product *entity.Product) error {
	if product.ProductID == 0 {
		m.nextID++
		product.ProductID = m.nextID
	}
	cp := *product
	m.products[product.ProductID] = &cp
	return nil
}

func (m *memoryProductRepo) GetProduct(_ context.Context, productID int64) (*entity.Product, error) {
	product := m.products[productID]
	if product == nil {
		return nil, nil
	}
	cp := *product
	return &cp, nil
}

func (m *memoryProductRepo) GetProductBySource(_ context.Context, sourceType string, sourceID int64) (*entity.Product, error) {
	for _, product := range m.products {
		if product.SourceRefType == sourceType && product.SourceRefID == sourceID {
			cp := *product
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memoryProductRepo) ListProducts(_ context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	products := make([]*entity.Product, 0, len(m.products))
	for _, product := range m.products {
		if req.Type != "" && product.Type != req.Type {
			continue
		}
		cp := *product
		products = append(products, &cp)
	}
	return &entity.ListProductsResult{Products: products, Total: int32(len(products))}, nil
}

func (m *memoryProductRepo) UpsertVersion(_ context.Context, version *entity.ProductVersion) error {
	m.versions[version.ProductID] = append(m.versions[version.ProductID], version)
	return nil
}

func (m *memoryProductRepo) ListVersions(_ context.Context, productID int64) ([]*entity.ProductVersion, error) {
	return m.versions[productID], nil
}

func (m *memoryProductRepo) InstallProduct(_ context.Context, installation *entity.ProductInstallation) error {
	if installation.InstallationID == 0 {
		m.nextID++
		installation.InstallationID = m.nextID
	}
	cp := *installation
	m.installations[installKey(installation.ProductID, installation.TargetSpaceID, installation.TargetUserID)] = &cp
	return nil
}

func (m *memoryProductRepo) UpdateInstallation(_ context.Context, installation *entity.ProductInstallation) error {
	if installation == nil {
		return errors.New("installation is required")
	}
	cp := *installation
	m.installations[installKey(installation.ProductID, installation.TargetSpaceID, installation.TargetUserID)] = &cp
	return nil
}

func (m *memoryProductRepo) GetInstallation(_ context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error) {
	installation := m.installations[installKey(productID, spaceID, userID)]
	if installation == nil && userID != 0 {
		installation = m.installations[installKey(productID, spaceID, 0)]
	}
	if installation == nil {
		return nil, nil
	}
	cp := *installation
	return &cp, nil
}

func (m *memoryProductRepo) ListInstallations(_ context.Context, req *entity.ListInstallationsRequest) ([]*entity.ProductInstallation, error) {
	out := make([]*entity.ProductInstallation, 0, len(m.installations))
	for _, installation := range m.installations {
		if req.SpaceID > 0 && installation.TargetSpaceID != req.SpaceID {
			continue
		}
		if req.ProductID > 0 && installation.ProductID != req.ProductID {
			continue
		}
		if req.Status != "" && installation.Status != req.Status {
			continue
		}
		cp := *installation
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memoryProductRepo) CreateAudit(_ context.Context, audit *entity.AuditLog) error {
	if audit.AuditID == 0 {
		m.nextID++
		audit.AuditID = m.nextID
	}
	cp := *audit
	m.audits = append(m.audits, &cp)
	return nil
}

func (m *memoryProductRepo) ListAudits(_ context.Context, req *entity.ListAuditsRequest) ([]*entity.AuditLog, error) {
	out := make([]*entity.AuditLog, 0, len(m.audits))
	for _, audit := range m.audits {
		if req.ProductID > 0 && audit.ProductID != req.ProductID {
			continue
		}
		if req.SpaceID > 0 && audit.SpaceID != req.SpaceID {
			continue
		}
		cp := *audit
		out = append(out, &cp)
	}
	return out, nil
}

func (m *memoryProductRepo) UpsertSessionRuntimeConfig(context.Context, *entity.SessionRuntimeConfig) error {
	return nil
}

func (m *memoryProductRepo) GetSessionRuntimeConfig(context.Context, int64) (*entity.SessionRuntimeConfig, error) {
	return nil, nil
}

func (m *memoryProductRepo) DeleteSessionRuntimeConfig(context.Context, int64) error {
	return nil
}
