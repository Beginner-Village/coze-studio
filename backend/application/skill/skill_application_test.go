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
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"

	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	openauthentity "github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	mock "github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/orm"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

type testSkillPO struct {
	ID               int64          `gorm:"column:id;primaryKey;autoIncrement:true"`
	SkillID          int64          `gorm:"column:skill_id;not null"`
	SpaceID          int64          `gorm:"column:space_id;not null"`
	Name             string         `gorm:"column:name;not null"`
	Description      *string        `gorm:"column:description"`
	Prompt           *string        `gorm:"column:prompt"`
	Files            *string        `gorm:"column:files"`
	IconURI          string         `gorm:"column:icon_uri;not null"`
	CreatorID        int64          `gorm:"column:creator_id;not null"`
	Status           int8           `gorm:"column:status;not null;default:1"`
	Version          int64          `gorm:"column:version;not null;default:1"`
	PublishScope     int8           `gorm:"column:publish_scope;not null;default:1"`
	PublishedVersion int64          `gorm:"column:published_version;not null;default:0"`
	PublishedAt      int64          `gorm:"column:published_at;not null;default:0"`
	PublishedBy      int64          `gorm:"column:published_by;not null;default:0"`
	ReviewStatus     int8           `gorm:"column:review_status;not null;default:2"`
	ReviewNote       *string        `gorm:"column:review_note"`
	ReviewerID       int64          `gorm:"column:reviewer_id;not null;default:0"`
	ReviewedAt       int64          `gorm:"column:reviewed_at;not null;default:0"`
	CreatedAt        int64          `gorm:"column:created_at;not null"`
	UpdatedAt        int64          `gorm:"column:updated_at;not null"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (testSkillPO) TableName() string {
	return "skill"
}

type testSkillVersionPO struct {
	ID          int64   `gorm:"column:id;primaryKey;autoIncrement:true"`
	SkillID     int64   `gorm:"column:skill_id;not null"`
	Version     int64   `gorm:"column:version;not null"`
	Name        string  `gorm:"column:name;not null"`
	Description *string `gorm:"column:description"`
	Prompt      *string `gorm:"column:prompt"`
	Files       *string `gorm:"column:files"`
	IconURI     string  `gorm:"column:icon_uri;not null"`
	ContentHash string  `gorm:"column:content_hash;not null"`
	CreatedAt   int64   `gorm:"column:created_at;not null"`
}

func (testSkillVersionPO) TableName() string {
	return "skill_version"
}

func TestSkillApplicationServiceUsesOpenAPIAuthUserWhenSessionMissing(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	svc := newTestSkillApplicationService(t, 4000)
	source, err := svc.CreateSkill(ctx, 1, "api-skill", "API skill", "prompt", "", map[string]string{
		"SKILL.md": "---\nname: api-skill\ndescription: API skill\n---\n# API Skill\n",
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(77), source.CreatorID)

	published, err := svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)
	assert.Equal(t, int64(77), published.PublishedBy)

	installed, err := svc.InstallMarketplaceSkill(ctx, source.SkillID, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(77), installed.CreatorID)
}

func TestSkillApplicationServiceInstallMarketplaceSkillCopiesPublishedVersion(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 3000)
	filesV1 := map[string]string{
		"SKILL.md": "---\nname: pdf-tools\ndescription: PDF tools\n---\n# V1\n",
	}
	source, err := svc.CreateSkill(ctx, 1, "pdf-tools", "PDF tools", "v1 prompt", "icon://pdf", filesV1)
	assert.NoError(t, err)

	filesV2 := map[string]string{
		"SKILL.md":            "---\nname: pdf-tools\ndescription: PDF tools\n---\n# PDF Tools\nUse v2.",
		"references/guide.md": "# Guide v2\n",
		"scripts/run.sh":      "echo v2\n",
		"assets/sample.txt":   "sample v2\n",
	}
	source, err = svc.UpdateSkill(ctx, source.SkillID, 1, "", "", "v2 prompt", "", filesV2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), source.Version)

	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)

	filesV3 := map[string]string{
		"SKILL.md": "---\nname: pdf-tools-next\ndescription: Draft\n---\n# Draft V3\n",
	}
	_, err = svc.UpdateSkill(ctx, source.SkillID, 1, "pdf-tools-next", "Draft", "v3 prompt", "", filesV3)
	assert.NoError(t, err)

	installed, err := svc.InstallMarketplaceSkill(ctx, source.SkillID, 2)
	assert.NoError(t, err)
	assert.NotNil(t, installed)
	assert.NotEqual(t, source.SkillID, installed.SkillID)
	assert.Equal(t, int64(2), installed.SpaceID)
	assert.Equal(t, int64(42), installed.CreatorID)
	assert.Equal(t, "pdf-tools", installed.Name)
	assert.Equal(t, "PDF tools", installed.Description)
	assert.Equal(t, "v2 prompt", installed.Prompt)
	assert.Equal(t, filesV2, installed.Files)
	assert.Equal(t, int64(1), installed.Version)
	assert.Equal(t, entity.SkillPublishScopePrivate, installed.PublishScope)
	assert.Equal(t, int64(0), installed.PublishedVersion)
}

func TestSkillApplicationServiceGetMarketplaceSkillReturnsPublishedSnapshotAcrossSpaces(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 3500)
	filesV1 := map[string]string{
		"SKILL.md": "---\nname: doc-tools\ndescription: Doc tools\n---\n# V1\n",
	}
	source, err := svc.CreateSkill(ctx, 1, "doc-tools", "Doc tools", "v1 prompt", "icon://doc", filesV1)
	assert.NoError(t, err)

	filesV2 := map[string]string{
		"SKILL.md":          "---\nname: doc-tools\ndescription: Doc tools\n---\n# Doc Tools\nUse v2.",
		"templates/tpl.md":  "# Template v2\n",
		"scripts/export.sh": "echo v2\n",
	}
	source, err = svc.UpdateSkill(ctx, source.SkillID, 1, "", "", "v2 prompt", "", filesV2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), source.Version)

	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)

	filesV3 := map[string]string{
		"SKILL.md": "---\nname: doc-tools-draft\ndescription: Draft\n---\n# Draft V3\n",
	}
	_, err = svc.UpdateSkill(ctx, source.SkillID, 1, "doc-tools-draft", "Draft", "v3 prompt", "", filesV3)
	assert.NoError(t, err)

	got, err := svc.GetMarketplaceSkill(ctx, source.SkillID, 2)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, source.SkillID, got.SkillID)
	assert.Equal(t, int64(1), got.SpaceID)
	assert.Equal(t, "doc-tools", got.Name)
	assert.Equal(t, "Doc tools", got.Description)
	assert.Equal(t, "v2 prompt", got.Prompt)
	assert.Equal(t, filesV2, got.Files)
	assert.Equal(t, int64(2), got.Version)
	assert.Equal(t, entity.SkillPublishScopeGlobal, got.PublishScope)
	assert.Equal(t, int64(2), got.PublishedVersion)
}

func TestSkillApplicationServicePublishGlobalEntersPendingReview(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 6000)
	withReviewerPermission(t, true)
	source, err := svc.CreateSkill(ctx, 1, "review-global", "review", "prompt", "", map[string]string{
		"SKILL.md": "---\nname: review-global\ndescription: review\n---\n# Review\n",
	})
	assert.NoError(t, err)
	assert.Equal(t, entity.SkillReviewStatusApproved, source.ReviewStatus)

	// Space publishing stays approved (self-governed).
	spacePub, err := svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeSpace)
	assert.NoError(t, err)
	assert.Equal(t, entity.SkillReviewStatusApproved, spacePub.ReviewStatus)

	// Global publishing requires review.
	globalPub, err := svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)
	assert.Equal(t, entity.SkillReviewStatusPending, globalPub.ReviewStatus)

	// Pending global skill must NOT appear in the global marketplace list.
	listed, total, err := svc.ListMarketplaceSkills(ctx, 0, entity.SkillPublishScopeGlobal, 1, 20, "")
	assert.NoError(t, err)
	assert.Equal(t, int32(0), total)
	assert.Empty(t, listed)

	// It should appear in the pending review queue.
	pending, pendingTotal, err := svc.ListPendingReviews(ctx, 1, 1, 20)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), pendingTotal)
	assert.Len(t, pending, 1)
	assert.Equal(t, source.SkillID, pending[0].SkillID)
}

func TestSkillApplicationServiceReviewApproveSurfacesInMarketplace(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 6500)
	withReviewerPermission(t, true)
	source, err := svc.CreateSkill(ctx, 1, "approve-me", "approve", "prompt", "", map[string]string{
		"SKILL.md": "---\nname: approve-me\ndescription: approve\n---\n# Approve\n",
	})
	assert.NoError(t, err)

	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)

	reviewed, err := svc.ReviewSkill(ctx, source.SkillID, true, "looks good")
	assert.NoError(t, err)
	assert.Equal(t, entity.SkillReviewStatusApproved, reviewed.ReviewStatus)
	assert.Equal(t, "looks good", reviewed.ReviewNote)
	assert.Equal(t, int64(42), reviewed.ReviewerID)
	assert.NotZero(t, reviewed.ReviewedAt)

	// Approved global skill now appears in the marketplace, and the review queue empties.
	listed, total, err := svc.ListMarketplaceSkills(ctx, 0, entity.SkillPublishScopeGlobal, 1, 20, "")
	assert.NoError(t, err)
	assert.Equal(t, int32(1), total)
	assert.Len(t, listed, 1)
	assert.Equal(t, source.SkillID, listed[0].SkillID)

	pending, pendingTotal, err := svc.ListPendingReviews(ctx, 1, 1, 20)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), pendingTotal)
	assert.Empty(t, pending)
}

func TestSkillApplicationServiceSyncsProductAfterLifecycleChanges(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 6600)
	syncer := &recordingProductSyncer{}
	svc.ProductSyncer = syncer
	withReviewerPermission(t, true)

	source, err := svc.CreateSkill(ctx, 1, "sync-product", "sync", "prompt", "", map[string]string{
		"SKILL.md": "---\nname: sync-product\ndescription: sync\n---\n# Sync\n",
	})
	assert.NoError(t, err)
	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)
	_, err = svc.ReviewSkill(ctx, source.SkillID, true, "ok")
	assert.NoError(t, err)
	_, err = svc.InstallMarketplaceSkill(ctx, source.SkillID, 2)
	assert.NoError(t, err)

	assert.GreaterOrEqual(t, len(syncer.skillIDs), 4)
	assert.Equal(t, source.SkillID, syncer.skillIDs[0])
	assert.Equal(t, source.SkillID, syncer.skillIDs[1])
	assert.Equal(t, source.SkillID, syncer.skillIDs[2])
	assert.NotEqual(t, source.SkillID, syncer.skillIDs[len(syncer.skillIDs)-1])
}

func TestSkillApplicationServiceReviewRejectKeepsOutOfMarketplace(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 7000)
	withReviewerPermission(t, true)
	source, err := svc.CreateSkill(ctx, 1, "reject-me", "reject", "prompt", "", map[string]string{
		"SKILL.md": "---\nname: reject-me\ndescription: reject\n---\n# Reject\n",
	})
	assert.NoError(t, err)

	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)

	reviewed, err := svc.ReviewSkill(ctx, source.SkillID, false, "not allowed")
	assert.NoError(t, err)
	assert.Equal(t, entity.SkillReviewStatusRejected, reviewed.ReviewStatus)

	listed, total, err := svc.ListMarketplaceSkills(ctx, 0, entity.SkillPublishScopeGlobal, 1, 20, "")
	assert.NoError(t, err)
	assert.Equal(t, int32(0), total)
	assert.Empty(t, listed)
}

// Skill review is restricted to the skill space's owner/admin: a non-manager caller
// (member or non-member) must be denied for both the review action and the pending queue.
func TestSkillApplicationServiceReviewRequiresReviewerRole(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})
	withReviewerPermission(t, false) // caller is NOT a space owner/admin
	svc := newTestSkillApplicationService(t, 8000)
	source, err := svc.CreateSkill(ctx, 1, "needs-review", "x", "prompt", "", map[string]string{
		"SKILL.md": "---\nname: needs-review\ndescription: x\n---\n# X\n",
	})
	assert.NoError(t, err)
	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)

	if _, err := svc.ReviewSkill(ctx, source.SkillID, true, "ok"); err == nil {
		t.Fatalf("non-reviewer must be denied ReviewSkill")
	}
	if _, _, err := svc.ListPendingReviews(ctx, 1, 1, 20); err == nil {
		t.Fatalf("non-reviewer must be denied ListPendingReviews")
	}
}

func TestSkillApplicationServiceRequiresStandardSkillFolderOnCreate(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 5000)

	_, err := svc.CreateSkill(ctx, 1, "prompt-only", "Prompt only", "just prompt", "", nil)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "SKILL.md is required")
	}
}

func TestSkillApplicationServiceRejectsNonStandardSkillFilePaths(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 6000)

	_, err := svc.CreateSkill(ctx, 1, "unsafe", "Unsafe", "", "", map[string]string{
		"SKILL.md":        "---\nname: unsafe\ndescription: Unsafe\n---\n# Unsafe\n",
		"../escape.txt":   "nope",
		"scripts/run.py":  "print('ok')\n",
		"references/a.md": "ok",
	})

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid skill file path")
	}
}

// 标准技能包(对齐 Anthropic / LangChain Agent Skills)允许顶层 README.md /
// requirements.txt / LICENSE 以及任意子目录布局,只要 SKILL.md 存在且无路径穿越/危险扩展名。
func TestSkillApplicationServiceAcceptsStandardTopLevelFiles(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 6100)

	_, err := svc.CreateSkill(ctx, 1, "standard", "Standard", "", "", map[string]string{
		"SKILL.md":           "---\nname: standard\ndescription: Standard\n---\n# Standard\n",
		"README.md":          "# readme\n",
		"requirements.txt":   "requests==2.31.0\n",
		"LICENSE":            "MIT\n",
		"pyproject.toml":     "[project]\nname=\"standard\"\n",
		"src/util/helper.py": "print('ok')\n",
	})

	assert.NoError(t, err)
}

func TestSkillApplicationServiceUpsertsSkillAssetIntoStandardFiles(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 7000)
	source, err := svc.CreateSkill(ctx, 1, "brand-kit", "Brand assets", "", "", map[string]string{
		"SKILL.md": "---\nname: brand-kit\ndescription: Brand assets\n---\n# Brand Kit\n",
	})
	assert.NoError(t, err)

	updated, err := svc.UpsertSkillAsset(ctx, source.SkillID, 1, "assets/logo.png", "data:image/png;base64,iVBORw0KGgo=", "image/png")

	assert.NoError(t, err)
	assert.Equal(t, int64(2), updated.Version)
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", updated.Files["assets/logo.png"])
	assert.Equal(t, "---\nname: brand-kit\ndescription: Brand assets\n---\n# Brand Kit\n", updated.Files["SKILL.md"])
}

func TestSkillApplicationServiceListsAndGetsSkillAssets(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 7500)
	source, err := svc.CreateSkill(ctx, 1, "brand-kit", "Brand assets", "", "", map[string]string{
		"SKILL.md":          "---\nname: brand-kit\ndescription: Brand assets\n---\n# Brand Kit\n",
		"assets/logo.png":   "data:image/png;base64,iVBORw0KGgo=",
		"assets/readme.txt": "hello assets",
		"scripts/run.sh":    "echo ok\n",
	})
	assert.NoError(t, err)

	assets, err := svc.ListSkillAssets(ctx, source.SkillID, 1)

	assert.NoError(t, err)
	if assert.Len(t, assets, 2) {
		assert.Equal(t, "assets/logo.png", assets[0].Path)
		assert.Equal(t, "image/png", assets[0].MIME)
		assert.True(t, assets[0].IsImage)
		assert.Equal(t, int64(len("data:image/png;base64,iVBORw0KGgo=")), assets[0].Size)
		assert.Equal(t, "assets/readme.txt", assets[1].Path)
		assert.Equal(t, "text/plain", assets[1].MIME)
		assert.False(t, assets[1].IsImage)
	}

	asset, err := svc.GetSkillAsset(ctx, source.SkillID, 1, "assets/logo.png")

	assert.NoError(t, err)
	assert.Equal(t, "assets/logo.png", asset.Path)
	assert.Equal(t, "image/png", asset.MIME)
	assert.True(t, asset.IsImage)
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", asset.Content)
}

func TestSkillApplicationServiceRejectsInvalidSkillAssetPath(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 8000)
	source, err := svc.CreateSkill(ctx, 1, "brand-kit", "Brand assets", "", "", map[string]string{
		"SKILL.md": "---\nname: brand-kit\ndescription: Brand assets\n---\n# Brand Kit\n",
	})
	assert.NoError(t, err)

	_, err = svc.UpsertSkillAsset(ctx, source.SkillID, 1, "scripts/logo.png", "data:image/png;base64,iVBORw0KGgo=", "image/png")

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "asset path must be under assets/")
	}
}

func TestSkillApplicationServiceDeletesSkillAssetFromStandardFiles(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 8500)
	source, err := svc.CreateSkill(ctx, 1, "brand-kit", "Brand assets", "", "", map[string]string{
		"SKILL.md":        "---\nname: brand-kit\ndescription: Brand assets\n---\n# Brand Kit\n",
		"assets/logo.png": "data:image/png;base64,iVBORw0KGgo=",
	})
	assert.NoError(t, err)

	updated, err := svc.DeleteSkillAsset(ctx, source.SkillID, 1, "assets/logo.png")

	assert.NoError(t, err)
	assert.Equal(t, int64(2), updated.Version)
	assert.NotContains(t, updated.Files, "assets/logo.png")
	assert.Equal(t, "---\nname: brand-kit\ndescription: Brand assets\n---\n# Brand Kit\n", updated.Files["SKILL.md"])
}

func TestSkillApplicationServiceImportsStandardSkillZipPackage(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 9000)
	content := buildTestSkillZip(t, map[string]string{
		"zip-tools/SKILL.md":            "---\nname: zip-tools\ndescription: ZIP import\n---\n# Zip Tools\n",
		"zip-tools/scripts/run.py":      "print('ok')\n",
		"zip-tools/references/guide.md": "# Guide\n",
		"zip-tools/templates/report.md": "# Report\n",
		"zip-tools/assets/logo.png":     "data:image/png;base64,iVBORw0KGgo=",
	})

	got, err := svc.ImportSkillPackage(ctx, 1, "zip-tools.zip", content, "")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), got.SpaceID)
	assert.Equal(t, "zip-tools", got.Name)
	assert.Equal(t, "ZIP import", got.Description)
	assert.Equal(t, "data:image/png;base64,iVBORw0KGgo=", got.Files["assets/logo.png"])
	assert.Equal(t, "print('ok')\n", got.Files["scripts/run.py"])
	assert.Equal(t, int64(1), got.Version)
}

func TestSkillApplicationServiceValidatesStandardSkillZipPackage(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 9500)
	content := buildTestSkillZip(t, map[string]string{
		"zip-tools/SKILL.md":            "---\nname: zip-tools\ndescription: ZIP validation\nversion: 1.2.3\ncategory: productivity\ntags:\n  - documents\nplatforms:\n  - linux\n---\n# Zip Tools\n",
		"zip-tools/scripts/run.py":      "print('ok')\n",
		"zip-tools/references/guide.md": "# Guide\n",
		"zip-tools/assets/logo.png":     "data:image/png;base64,iVBORw0KGgo=",
	})

	got, err := svc.ValidateSkillPackage(ctx, "zip-tools.zip", content)

	assert.NoError(t, err)
	assert.True(t, got.Valid)
	assert.Empty(t, got.Error)
	assert.Equal(t, "zip-tools.zip", got.Filename)
	assert.Equal(t, "zip-tools", got.Name)
	assert.Equal(t, "ZIP validation", got.Description)
	assert.Equal(t, "1.2.3", got.Metadata.Version)
	assert.Equal(t, "productivity", got.Metadata.Category)
	assert.Equal(t, []string{"documents"}, got.Metadata.Tags)
	assert.Equal(t, []string{"linux"}, got.Metadata.Platforms)
	assert.Equal(t, int32(4), got.FileCount)
	assert.Equal(t, []string{"SKILL.md", "assets/logo.png", "references/guide.md", "scripts/run.py"}, got.FilePaths)
	assert.Equal(t, []string{"assets/logo.png"}, got.AssetPaths)
	assert.Equal(t, []string{"assets/logo.png"}, got.ImagePaths)
}

func TestSkillApplicationServiceValidationReportsInvalidSkillZipPackage(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 9600)
	content := buildTestSkillZip(t, map[string]string{
		"scripts/run.py": "print('ok')\n",
	})

	got, err := svc.ValidateSkillPackage(ctx, "missing-skill-md.zip", content)

	assert.NoError(t, err)
	assert.False(t, got.Valid)
	assert.Equal(t, "missing-skill-md.zip", got.Filename)
	assert.Contains(t, got.Error, "SKILL.md is required")
	assert.NotContains(t, got.Error, "stack=")
	assert.NotContains(t, got.Error, "\n")
	assert.Empty(t, got.FilePaths)
}

func TestSkillApplicationServiceRejectsInvalidSkillZipPackage(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 10000)
	missingSkillMD := buildTestSkillZip(t, map[string]string{
		"scripts/run.py": "print('ok')\n",
	})
	_, err := svc.ImportSkillPackage(ctx, 1, "missing.zip", missingSkillMD, "")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "SKILL.md is required")
	}

	missingRootSkillMD := buildTestSkillZip(t, map[string]string{
		"missing-root/scripts/run.py": "print('ok')\n",
	})
	_, err = svc.ImportSkillPackage(ctx, 1, "missing-root.zip", missingRootSkillMD, "")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "SKILL.md is required")
	}

	unsafePath := buildTestSkillZip(t, map[string]string{
		"SKILL.md":      "---\nname: unsafe\ndescription: Unsafe\n---\n# Unsafe\n",
		"../escape.txt": "nope",
	})
	_, err = svc.ImportSkillPackage(ctx, 1, "unsafe.zip", unsafePath, "")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid skill file path")
	}
}

func TestSkillApplicationServiceExportsStandardSkillZipPackage(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 11000)
	source, err := svc.CreateSkill(ctx, 1, "zip-export-tools", "ZIP export", "", "", map[string]string{
		"SKILL.md":       "---\nname: zip-export-tools\ndescription: ZIP export\n---\n# ZIP Export\n",
		"scripts/run.sh": "echo export\n",
		"assets/logo.png": "data:image/png;base64," +
			base64.StdEncoding.EncodeToString([]byte{0x89, 'P', 'N', 'G'}),
	})
	assert.NoError(t, err)

	pkg, err := svc.ExportSkillPackage(ctx, source.SkillID, 1)

	assert.NoError(t, err)
	assert.Equal(t, "zip-export-tools.zip", pkg.Filename)
	assert.True(t, strings.HasPrefix(pkg.Content, "data:application/zip;base64,"))
	assert.ElementsMatch(t, []string{"SKILL.md", "assets/logo.png", "scripts/run.sh"}, pkg.FilePaths)

	zipFiles := readTestZipDataURL(t, pkg.Content)
	assert.Equal(t, "---\nname: zip-export-tools\ndescription: ZIP export\n---\n# ZIP Export\n", string(zipFiles["zip-export-tools/SKILL.md"]))
	assert.Equal(t, "echo export\n", string(zipFiles["zip-export-tools/scripts/run.sh"]))
	assert.Equal(t, []byte{0x89, 'P', 'N', 'G'}, zipFiles["zip-export-tools/assets/logo.png"])
}

func TestSkillApplicationServiceExportsPublishedMarketplaceSnapshot(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	svc := newTestSkillApplicationService(t, 12000)
	source, err := svc.CreateSkill(ctx, 1, "market-export", "Market export", "", "", map[string]string{
		"SKILL.md": "---\nname: market-export\ndescription: Market export\n---\n# V1\n",
	})
	assert.NoError(t, err)

	_, err = svc.UpdateSkill(ctx, source.SkillID, 1, "", "", "", "", map[string]string{
		"SKILL.md":            "---\nname: market-export\ndescription: Market export\n---\n# V2 Published\n",
		"references/guide.md": "published guide\n",
	})
	assert.NoError(t, err)
	_, err = svc.PublishSkill(ctx, source.SkillID, 1, entity.SkillPublishScopeGlobal)
	assert.NoError(t, err)
	_, err = svc.UpdateSkill(ctx, source.SkillID, 1, "", "", "", "", map[string]string{
		"SKILL.md": "---\nname: market-export-draft\ndescription: Draft\n---\n# V3 Draft\n",
	})
	assert.NoError(t, err)

	pkg, err := svc.ExportSkillPackage(ctx, source.SkillID, 2)

	assert.NoError(t, err)
	assert.Equal(t, "market-export.zip", pkg.Filename)
	zipFiles := readTestZipDataURL(t, pkg.Content)
	assert.Contains(t, string(zipFiles["market-export/SKILL.md"]), "# V2 Published")
	assert.Equal(t, "published guide\n", string(zipFiles["market-export/references/guide.md"]))
	assert.NotContains(t, string(zipFiles["market-export/SKILL.md"]), "V3 Draft")
}

func buildTestSkillZip(t *testing.T, files map[string]string) string {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for path, content := range files {
		w, err := zw.Create(path)
		assert.NoError(t, err)
		_, err = w.Write([]byte(content))
		assert.NoError(t, err)
	}
	assert.NoError(t, zw.Close())

	return "data:application/zip;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func readTestZipDataURL(t *testing.T, content string) map[string][]byte {
	t.Helper()

	parts := strings.SplitN(content, ",", 2)
	assert.Len(t, parts, 2)
	raw, err := base64.StdEncoding.DecodeString(parts[1])
	assert.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	assert.NoError(t, err)
	files := make(map[string][]byte)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		rc, err := file.Open()
		assert.NoError(t, err)
		data, err := io.ReadAll(rc)
		assert.NoError(t, err)
		assert.NoError(t, rc.Close())
		files[file.Name] = data
	}
	return files
}

type recordingProductSyncer struct {
	skillIDs []int64
}

func (r *recordingProductSyncer) SyncSkillProduct(_ context.Context, skillID int64) error {
	r.skillIDs = append(r.skillIDs, skillID)
	return nil
}

// fakeReviewerUserSVC is a minimal crossuser.User that reports a fixed space permission,
// used to test that skill review is gated on space owner/admin (CanManage).
type fakeReviewerUserSVC struct{ canManage bool }

func (f *fakeReviewerUserSVC) GetUserSpaceList(_ context.Context, _ int64) ([]*crossuser.EntitySpace, error) {
	return nil, nil
}
func (f *fakeReviewerUserSVC) CheckSpacePermission(_ context.Context, _, _ int64) (*crossuser.SpacePermission, error) {
	return &crossuser.SpacePermission{IsMember: true, CanManage: f.canManage}, nil
}

// withReviewerPermission installs a fake user service so the caller is treated as a space
// owner/admin (canManage=true) or a plain member (false), then restores the previous SVC.
func withReviewerPermission(t *testing.T, canManage bool) {
	prev := crossuser.DefaultSVC()
	crossuser.SetDefaultSVC(&fakeReviewerUserSVC{canManage: canManage})
	t.Cleanup(func() { crossuser.SetDefaultSVC(prev) })
}

func newTestSkillApplicationService(t *testing.T, idOffset int64) *SkillApplicationService {
	t.Helper()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	var seq int64
	idGen := mock.NewMockIDGenerator(ctrl)
	idGen.EXPECT().GenID(gomock.Any()).DoAndReturn(func(_ context.Context) (int64, error) {
		return atomic.AddInt64(&seq, 1) + idOffset, nil
	}).AnyTimes()

	mockDBGen := orm.NewMockDB()
	mockDBGen.AddTable(&testSkillPO{})
	mockDBGen.AddTable(&testSkillVersionPO{})
	db, err := mockDBGen.DB()
	assert.NoError(t, err)

	return InitService(&ServiceComponents{DB: db, IDGen: idGen})
}

func TestValidateStandardSkillFilesRejectsForbiddenExtensions(t *testing.T) {
	md := "---\nname: demo\ndescription: d\n---\nbody"

	// A clean package (text/script/asset) passes.
	ok := map[string]string{
		"SKILL.md":           md,
		"scripts/run.py":     "print(1)",
		"references/note.md": "note",
		"assets/icon.png":    "data:image/png;base64,AAAA",
	}
	if err := validateStandardSkillFiles(ok); err != nil {
		t.Fatalf("clean standard skill package should pass, got: %v", err)
	}

	// Executable / native-binary file types are rejected anywhere in the package.
	for _, bad := range []string{
		"scripts/tool.exe", "assets/lib.so", "scripts/x.bat", "templates/a.dll",
		"scripts/m.pyc", "references/n.dylib", "assets/app.jar", "scripts/k.msi",
	} {
		files := map[string]string{"SKILL.md": md, bad: "x"}
		if err := validateStandardSkillFiles(files); err == nil {
			t.Fatalf("forbidden executable/binary file %q must be rejected", bad)
		}
	}
}
