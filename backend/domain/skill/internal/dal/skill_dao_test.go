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
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	mock "github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/orm"
)

func TestSkillDAO_VersionSnapshot(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var seq int64
	idGen := mock.NewMockIDGenerator(ctrl)
	idGen.EXPECT().GenID(gomock.Any()).DoAndReturn(func(_ context.Context) (int64, error) {
		return atomic.AddInt64(&seq, 1) + 1000, nil
	}).AnyTimes()

	mockDBGen := orm.NewMockDB()
	mockDBGen.AddTable(&skillPO{})
	mockDBGen.AddTable(&skillVersionPO{})
	db, err := mockDBGen.DB()
	assert.NoError(t, err)

	dao := NewSkillDAO(db, idGen)

	skillID, err := dao.Create(ctx, &entity.Skill{
		SpaceID: 1,
		Name:    "demo",
		Prompt:  "v1 prompt",
		Status:  entity.SkillStatusActive,
	})
	assert.NoError(t, err)

	// Create starts at version 1.
	got, err := dao.Get(ctx, skillID)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, int64(1), got.Version)

	v1, err := dao.GetVersion(ctx, skillID, 1)
	assert.NoError(t, err)
	assert.NotNil(t, v1)
	assert.Equal(t, int64(1), v1.Version)
	assert.Equal(t, "v1 prompt", v1.Prompt)

	// Update bumps version to 2 and writes a standard folder skill snapshot.
	filesV2 := map[string]string{
		"SKILL.md":            "---\nname: demo\ndescription: Demo skill\n---\n# Demo\nUse v2.",
		"references/guide.md": "# Guide\n",
		"scripts/run.sh":      "echo ok\n",
		"assets/example.txt":  "asset\n",
	}
	err = dao.Update(ctx, &entity.Skill{SkillID: skillID, Prompt: "v2 prompt", Files: filesV2})
	assert.NoError(t, err)

	got, err = dao.Get(ctx, skillID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), got.Version)
	assert.Equal(t, "v2 prompt", got.Prompt)
	assert.Equal(t, filesV2, got.Files)

	// The snapshot for version 2 is retrievable and content-hashed.
	v2, err := dao.GetVersion(ctx, skillID, 2)
	assert.NoError(t, err)
	assert.NotNil(t, v2)
	assert.Equal(t, int64(2), v2.Version)
	assert.Equal(t, "v2 prompt", v2.Prompt)
	assert.Equal(t, filesV2, v2.Files)
	assert.Equal(t, contentHash("v2 prompt", filesV2), v2.ContentHash)

	latest, err := dao.GetLatestVersion(ctx, skillID)
	assert.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, int64(2), latest.Version)

	// A second update yields version 3.
	filesV3 := map[string]string{"SKILL.md": "# V3\n"}
	err = dao.Update(ctx, &entity.Skill{SkillID: skillID, Prompt: "v3 prompt", Files: filesV3})
	assert.NoError(t, err)
	latest, err = dao.GetLatestVersion(ctx, skillID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), latest.Version)
	assert.Equal(t, filesV3, latest.Files)
}

func TestSkillDAO_MarketplacePublish(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var seq int64
	idGen := mock.NewMockIDGenerator(ctrl)
	idGen.EXPECT().GenID(gomock.Any()).DoAndReturn(func(_ context.Context) (int64, error) {
		return atomic.AddInt64(&seq, 1) + 2000, nil
	}).AnyTimes()

	mockDBGen := orm.NewMockDB()
	mockDBGen.AddTable(&skillPO{})
	mockDBGen.AddTable(&skillVersionPO{})
	db, err := mockDBGen.DB()
	assert.NoError(t, err)

	dao := NewSkillDAO(db, idGen)

	globalID, err := dao.Create(ctx, &entity.Skill{
		SpaceID:     1,
		Name:        "global-pdf",
		Description: "Reusable PDF skill",
		Prompt:      "global prompt",
		Status:      entity.SkillStatusActive,
	})
	assert.NoError(t, err)
	spaceID, err := dao.Create(ctx, &entity.Skill{
		SpaceID:     1,
		Name:        "space-xlsx",
		Description: "Space spreadsheet skill",
		Prompt:      "space prompt",
		Status:      entity.SkillStatusActive,
	})
	assert.NoError(t, err)
	otherSpaceID, err := dao.Create(ctx, &entity.Skill{
		SpaceID:     2,
		Name:        "other-space-docx",
		Description: "Other space document skill",
		Prompt:      "other prompt",
		Status:      entity.SkillStatusActive,
	})
	assert.NoError(t, err)

	assert.NoError(t, dao.Publish(ctx, globalID, entity.SkillPublishScopeGlobal, entity.SkillReviewStatusApproved, 1, 100, 1000))
	assert.NoError(t, dao.Publish(ctx, spaceID, entity.SkillPublishScopeSpace, entity.SkillReviewStatusApproved, 1, 100, 2000))
	assert.NoError(t, dao.Publish(ctx, otherSpaceID, entity.SkillPublishScopeSpace, entity.SkillReviewStatusApproved, 1, 100, 3000))

	resp, err := dao.ListMarketplace(ctx, &entity.MarketplaceListRequest{SpaceID: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Total)
	assert.Equal(t, []int64{spaceID, globalID}, []int64{resp.Skills[0].SkillID, resp.Skills[1].SkillID})
	assert.Equal(t, entity.SkillPublishScopeSpace, resp.Skills[0].PublishScope)
	assert.Equal(t, int64(1), resp.Skills[0].PublishedVersion)

	resp, err = dao.ListMarketplace(ctx, &entity.MarketplaceListRequest{Scope: entity.SkillPublishScopeGlobal})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, globalID, resp.Skills[0].SkillID)

	resp, err = dao.ListMarketplace(ctx, &entity.MarketplaceListRequest{SpaceID: 2})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Total)
	assert.Equal(t, []int64{otherSpaceID, globalID}, []int64{resp.Skills[0].SkillID, resp.Skills[1].SkillID})

	resp, err = dao.ListMarketplace(ctx, &entity.MarketplaceListRequest{SpaceID: 1, Keyword: "PDF"})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, globalID, resp.Skills[0].SkillID)

	assert.NoError(t, dao.Publish(ctx, globalID, entity.SkillPublishScopePrivate, entity.SkillReviewStatusApproved, 0, 0, 0))
	resp, err = dao.ListMarketplace(ctx, &entity.MarketplaceListRequest{SpaceID: 1})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Total)
	assert.Equal(t, spaceID, resp.Skills[0].SkillID)
}
