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

	// Update bumps version to 2 and writes a snapshot.
	err = dao.Update(ctx, &entity.Skill{SkillID: skillID, Prompt: "v2 prompt"})
	assert.NoError(t, err)

	got, err = dao.Get(ctx, skillID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), got.Version)
	assert.Equal(t, "v2 prompt", got.Prompt)

	// The snapshot for version 2 is retrievable and content-hashed.
	v2, err := dao.GetVersion(ctx, skillID, 2)
	assert.NoError(t, err)
	assert.NotNil(t, v2)
	assert.Equal(t, int64(2), v2.Version)
	assert.Equal(t, "v2 prompt", v2.Prompt)
	assert.Equal(t, contentHash("v2 prompt"), v2.ContentHash)

	latest, err := dao.GetLatestVersion(ctx, skillID)
	assert.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, int64(2), latest.Version)

	// A second update yields version 3.
	err = dao.Update(ctx, &entity.Skill{SkillID: skillID, Prompt: "v3 prompt"})
	assert.NoError(t, err)
	latest, err = dao.GetLatestVersion(ctx, skillID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), latest.Version)
}
