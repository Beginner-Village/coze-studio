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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	productentity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	productservice "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/service"
	skillentity "github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

func TestSyncSkillProductMapsApprovedGlobalSkillToPublishedProduct(t *testing.T) {
	ctx := context.Background()
	productRepo := newMemoryProductRepo()
	skillSvc := &fakeSkillService{skills: map[int64]*skillentity.Skill{
		10: {
			SkillID:          10,
			SpaceID:          1,
			CreatorID:        42,
			Name:             "pdf-tools",
			Description:      "PDF tools",
			Files:            map[string]string{"SKILL.md": "# PDF", "assets/logo.png": "data:image/png;base64,AAAA"},
			Metadata:         skillentity.SkillMetadata{Category: "document", Tags: []string{"pdf"}},
			IconURI:          "icon://pdf",
			PublishScope:     skillentity.SkillPublishScopeGlobal,
			PublishedVersion: 2,
			Version:          3,
			ReviewStatus:     skillentity.SkillReviewStatusApproved,
		},
	}}
	svc := newTestProductApplicationService(productRepo, skillSvc)

	err := svc.SyncSkillProduct(ctx, 10)

	require.NoError(t, err)
	product, err := productRepo.GetProductBySource(ctx, productentity.SourceRefTypeSkill, 10)
	require.NoError(t, err)
	require.NotNil(t, product)
	assert.Equal(t, productentity.AIProductTypeStandardSkill, product.Type)
	assert.Equal(t, productentity.AIProductStatusPublished, product.Status)
	assert.Equal(t, productentity.AIProductVisibilityGlobal, product.Visibility)
	assert.Equal(t, "2", product.PublishedVersion)
	assert.Equal(t, "3", product.LatestVersion)
	assert.Equal(t, int64(10), product.Feature["skill_id"])
	assert.Equal(t, int64(2), product.Feature["file_count"])
	assert.Equal(t, int64(1), product.Feature["asset_count"])
}

func TestSyncSkillProductMapsPendingGlobalSkillToReviewingProduct(t *testing.T) {
	ctx := context.Background()
	productRepo := newMemoryProductRepo()
	skillSvc := &fakeSkillService{skills: map[int64]*skillentity.Skill{
		10: {
			SkillID:          10,
			SpaceID:          1,
			CreatorID:        42,
			Name:             "pdf-tools",
			Description:      "PDF tools",
			Files:            map[string]string{"SKILL.md": "# PDF"},
			PublishScope:     skillentity.SkillPublishScopeGlobal,
			PublishedVersion: 2,
			Version:          2,
			ReviewStatus:     skillentity.SkillReviewStatusPending,
		},
	}}
	svc := newTestProductApplicationService(productRepo, skillSvc)

	err := svc.SyncSkillProduct(ctx, 10)

	require.NoError(t, err)
	product, err := productRepo.GetProductBySource(ctx, productentity.SourceRefTypeSkill, 10)
	require.NoError(t, err)
	require.NotNil(t, product)
	assert.Equal(t, productentity.AIProductStatusReviewing, product.Status)
	assert.Equal(t, productentity.AIProductVisibilityGlobal, product.Visibility)
}

func TestSyncSkillProductKeepsExistingProductIDForSource(t *testing.T) {
	ctx := context.Background()
	productRepo := newMemoryProductRepo()
	productRepo.products[999] = &productentity.Product{
		ProductID:     999,
		SourceRefType: productentity.SourceRefTypeSkill,
		SourceRefID:   10,
		Name:          "old-name",
		Type:          productentity.AIProductTypeStandardSkill,
		Status:        productentity.AIProductStatusDraft,
		Visibility:    productentity.AIProductVisibilityPrivate,
	}
	skillSvc := &fakeSkillService{skills: map[int64]*skillentity.Skill{
		10: {
			SkillID:      10,
			SpaceID:      1,
			CreatorID:    42,
			Name:         "new-name",
			Description:  "new desc",
			Files:        map[string]string{"SKILL.md": "# New"},
			PublishScope: skillentity.SkillPublishScopeSpace,
			Version:      4,
			ReviewStatus: skillentity.SkillReviewStatusApproved,
		},
	}}
	svc := newTestProductApplicationService(productRepo, skillSvc)

	err := svc.SyncSkillProduct(ctx, 10)

	require.NoError(t, err)
	product, err := productRepo.GetProductBySource(ctx, productentity.SourceRefTypeSkill, 10)
	require.NoError(t, err)
	require.NotNil(t, product)
	assert.Equal(t, int64(999), product.ProductID)
	assert.Equal(t, "new-name", product.Name)
	assert.Equal(t, productentity.AIProductVisibilitySpace, product.Visibility)
}

func TestInstallProductRequiresSpaceManager(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})
	withProductSpacePermission(t, false)

	productRepo := newMemoryProductRepo()
	productRepo.products[100] = &productentity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             productentity.AIProductTypeStandardSkill,
		Status:           productentity.AIProductStatusPublished,
		Visibility:       productentity.AIProductVisibilityGlobal,
		PublishedVersion: "4",
	}
	svc := newTestProductApplicationService(productRepo, &fakeSkillService{})

	installed, err := svc.InstallProduct(ctx, 100, 1, "")

	require.Error(t, err)
	assert.Nil(t, installed)
	assert.Contains(t, err.Error(), "space manager")
	assert.Empty(t, productRepo.installations)
}

func TestInstallProductUsesLoggedInUserAndPinsVersion(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})
	withProductSpacePermission(t, true)

	productRepo := newMemoryProductRepo()
	productRepo.products[100] = &productentity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             productentity.AIProductTypeStandardSkill,
		Status:           productentity.AIProductStatusPublished,
		Visibility:       productentity.AIProductVisibilityGlobal,
		PublishedVersion: "4",
		LatestVersion:    "5",
	}
	svc := newTestProductApplicationService(productRepo, &fakeSkillService{})

	installed, err := svc.InstallProduct(ctx, 100, 1, "")

	require.NoError(t, err)
	require.NotNil(t, installed)
	assert.Equal(t, int64(42), installed.InstalledBy)
	assert.Equal(t, "4", installed.ProductVersion)
	assert.Equal(t, productentity.AIProductInstallationActive, installed.Status)
}

func TestUpsertSessionRuntimeConfigResolvesInstalledSkillProducts(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	productRepo := newMemoryProductRepo()
	productRepo.products[100] = &productentity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             productentity.AIProductTypeStandardSkill,
		Status:           productentity.AIProductStatusPublished,
		Visibility:       productentity.AIProductVisibilityGlobal,
		SourceRefType:    productentity.SourceRefTypeSkill,
		SourceRefID:      10,
		PublishedVersion: "4",
		Feature:          map[string]any{"entry_file": "SKILL.md"},
	}
	productRepo.installations[productentity.InstallationKey(100, 1, 42)] = &productentity.ProductInstallation{
		InstallationID: 1001,
		ProductID:      100,
		ProductVersion: "4",
		TargetSpaceID:  1,
		TargetUserID:   42,
		Status:         productentity.AIProductInstallationActive,
		InstallMode:    "space",
	}
	svc := newTestProductApplicationService(productRepo, &fakeSkillService{})

	config, err := svc.UpsertSessionRuntimeConfig(ctx, &productentity.SessionRuntimeConfig{
		ConversationID:  200,
		AgentID:         300,
		SpaceID:         1,
		SkillProductIDs: []int64{100, 100, 0},
		ToolPolicy:      map[string]any{"web_search": true},
	})

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, []int64{100}, config.SkillProductIDs)
	require.Contains(t, config.ResolvedSnapshot, "skills")
	skills, ok := config.ResolvedSnapshot["skills"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, skills, 1)
	assert.Equal(t, int64(100), skills[0]["product_id"])
	assert.Equal(t, int64(1001), skills[0]["installation_id"])
	assert.Equal(t, productentity.AIProductAuditActionSessionBind, productRepo.audits[0].Action)
}

func TestUpsertSessionRuntimeConfigRejectsUninstalledProduct(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	productRepo := newMemoryProductRepo()
	productRepo.products[100] = &productentity.Product{
		ProductID:        100,
		Name:             "PDF tools",
		Type:             productentity.AIProductTypeStandardSkill,
		Status:           productentity.AIProductStatusPublished,
		Visibility:       productentity.AIProductVisibilityGlobal,
		PublishedVersion: "4",
	}
	svc := newTestProductApplicationService(productRepo, &fakeSkillService{})

	config, err := svc.UpsertSessionRuntimeConfig(ctx, &productentity.SessionRuntimeConfig{
		ConversationID:  200,
		AgentID:         300,
		SpaceID:         1,
		SkillProductIDs: []int64{100},
	})

	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "not installed")
	assert.Empty(t, productRepo.runtimeConfigs)
}

func newTestProductApplicationService(repo *memoryProductRepo, skillSvc *fakeSkillService) *ProductApplicationService {
	return &ProductApplicationService{
		DomainSVC:   serviceForTest(repo),
		ProductRepo: repo,
		SkillSVC:    skillSvc,
	}
}

func serviceForTest(repo *memoryProductRepo) productservice.Service {
	return productservice.NewService(&productservice.Components{ProductRepo: repo})
}

type fakeSkillService struct {
	skills   map[int64]*skillentity.Skill
	versions map[int64]map[int64]*skillentity.SkillVersion
}

func (f *fakeSkillService) CreateSkill(context.Context, *skillentity.Skill) (int64, error) {
	return 0, nil
}
func (f *fakeSkillService) GetSkill(_ context.Context, skillID int64) (*skillentity.Skill, error) {
	skill := f.skills[skillID]
	if skill == nil {
		return nil, nil
	}
	cp := *skill
	return &cp, nil
}
func (f *fakeSkillService) GetSkillByName(context.Context, int64, string) (*skillentity.Skill, error) {
	return nil, nil
}
func (f *fakeSkillService) UpdateSkill(context.Context, *skillentity.Skill) error { return nil }
func (f *fakeSkillService) DeleteSkill(context.Context, int64) error              { return nil }
func (f *fakeSkillService) PublishSkill(context.Context, int64, int8, int8, int64, int64, int64) error {
	return nil
}
func (f *fakeSkillService) ReviewSkill(context.Context, int64, int8, string, int64, int64) error {
	return nil
}
func (f *fakeSkillService) ListPendingReviews(context.Context, *skillentity.PendingReviewListRequest) (*skillentity.ListResponse, error) {
	return nil, nil
}
func (f *fakeSkillService) ListSkills(context.Context, *skillentity.ListRequest) (*skillentity.ListResponse, error) {
	return nil, nil
}
func (f *fakeSkillService) ListMarketplaceSkills(context.Context, *skillentity.MarketplaceListRequest) (*skillentity.ListResponse, error) {
	return nil, nil
}
func (f *fakeSkillService) MGetSkills(context.Context, []int64) ([]*skillentity.Skill, error) {
	return nil, nil
}
func (f *fakeSkillService) GetSkillVersion(_ context.Context, skillID, version int64) (*skillentity.SkillVersion, error) {
	if f.versions == nil || f.versions[skillID] == nil {
		return nil, errors.New("version not found")
	}
	return f.versions[skillID][version], nil
}

type memoryProductRepo struct {
	products       map[int64]*productentity.Product
	installations  map[string]*productentity.ProductInstallation
	runtimeConfigs map[int64]*productentity.SessionRuntimeConfig
	audits         []*productentity.AuditLog
	nextID         int64
}

func newMemoryProductRepo() *memoryProductRepo {
	return &memoryProductRepo{
		products:       map[int64]*productentity.Product{},
		installations:  map[string]*productentity.ProductInstallation{},
		runtimeConfigs: map[int64]*productentity.SessionRuntimeConfig{},
		audits:         []*productentity.AuditLog{},
		nextID:         1000,
	}
}

func (m *memoryProductRepo) UpsertProduct(_ context.Context, product *productentity.Product) error {
	if product.ProductID == 0 {
		m.nextID++
		product.ProductID = m.nextID
	}
	cp := *product
	m.products[product.ProductID] = &cp
	return nil
}
func (m *memoryProductRepo) GetProduct(_ context.Context, productID int64) (*productentity.Product, error) {
	product := m.products[productID]
	if product == nil {
		return nil, nil
	}
	cp := *product
	return &cp, nil
}
func (m *memoryProductRepo) GetProductBySource(_ context.Context, sourceType string, sourceID int64) (*productentity.Product, error) {
	for _, product := range m.products {
		if product.SourceRefType == sourceType && product.SourceRefID == sourceID {
			cp := *product
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *memoryProductRepo) ListProducts(context.Context, *productentity.ListProductsRequest) (*productentity.ListProductsResult, error) {
	return nil, nil
}
func (m *memoryProductRepo) UpsertVersion(context.Context, *productentity.ProductVersion) error {
	return nil
}
func (m *memoryProductRepo) ListVersions(context.Context, int64) ([]*productentity.ProductVersion, error) {
	return nil, nil
}
func (m *memoryProductRepo) InstallProduct(_ context.Context, installation *productentity.ProductInstallation) error {
	if installation.InstallationID == 0 {
		m.nextID++
		installation.InstallationID = m.nextID
	}
	cp := *installation
	m.installations[productentity.InstallationKey(installation.ProductID, installation.TargetSpaceID, installation.TargetUserID)] = &cp
	return nil
}
func (m *memoryProductRepo) UpdateInstallation(_ context.Context, installation *productentity.ProductInstallation) error {
	cp := *installation
	m.installations[productentity.InstallationKey(installation.ProductID, installation.TargetSpaceID, installation.TargetUserID)] = &cp
	return nil
}
func (m *memoryProductRepo) GetInstallation(_ context.Context, productID, spaceID, userID int64) (*productentity.ProductInstallation, error) {
	installation := m.installations[productentity.InstallationKey(productID, spaceID, userID)]
	if installation == nil && userID != 0 {
		installation = m.installations[productentity.InstallationKey(productID, spaceID, 0)]
	}
	if installation == nil {
		return nil, nil
	}
	cp := *installation
	return &cp, nil
}
func (m *memoryProductRepo) ListInstallations(context.Context, *productentity.ListInstallationsRequest) ([]*productentity.ProductInstallation, error) {
	return nil, nil
}
func (m *memoryProductRepo) CreateAudit(_ context.Context, audit *productentity.AuditLog) error {
	cp := *audit
	m.audits = append(m.audits, &cp)
	return nil
}
func (m *memoryProductRepo) ListAudits(context.Context, *productentity.ListAuditsRequest) ([]*productentity.AuditLog, error) {
	return nil, nil
}
func (m *memoryProductRepo) UpsertSessionRuntimeConfig(_ context.Context, config *productentity.SessionRuntimeConfig) error {
	cp := *config
	cp.MCPProductIDs = append([]int64(nil), config.MCPProductIDs...)
	cp.SkillProductIDs = append([]int64(nil), config.SkillProductIDs...)
	m.runtimeConfigs[config.ConversationID] = &cp
	return nil
}
func (m *memoryProductRepo) GetSessionRuntimeConfig(_ context.Context, conversationID int64) (*productentity.SessionRuntimeConfig, error) {
	config := m.runtimeConfigs[conversationID]
	if config == nil {
		return nil, nil
	}
	cp := *config
	cp.MCPProductIDs = append([]int64(nil), config.MCPProductIDs...)
	cp.SkillProductIDs = append([]int64(nil), config.SkillProductIDs...)
	return &cp, nil
}
func (m *memoryProductRepo) DeleteSessionRuntimeConfig(_ context.Context, conversationID int64) error {
	delete(m.runtimeConfigs, conversationID)
	return nil
}

type fakeProductUserSVC struct{ canManage bool }

func (f *fakeProductUserSVC) GetUserSpaceList(_ context.Context, _ int64) ([]*crossuser.EntitySpace, error) {
	return nil, nil
}

func (f *fakeProductUserSVC) CheckSpacePermission(_ context.Context, _, _ int64) (*crossuser.SpacePermission, error) {
	return &crossuser.SpacePermission{IsMember: true, CanManage: f.canManage}, nil
}

func withProductSpacePermission(t *testing.T, canManage bool) {
	prev := crossuser.DefaultSVC()
	crossuser.SetDefaultSVC(&fakeProductUserSVC{canManage: canManage})
	t.Cleanup(func() { crossuser.SetDefaultSVC(prev) })
}
