/*
 * Copyright 2025 coze-dev Authors
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

package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/domain/skill/entity"
	"github.com/coze-dev/coze-studio/backend/domain/skill/internal/dal"
	"github.com/coze-dev/coze-studio/backend/infra/contract/idgen"
)

// SkillRepository defines the interface for skill data access.
type SkillRepository interface {
	Create(ctx context.Context, skill *entity.Skill) (int64, error)
	Get(ctx context.Context, skillID int64) (*entity.Skill, error)
	GetByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error)
	Update(ctx context.Context, skill *entity.Skill) error
	Delete(ctx context.Context, skillID int64) error
	List(ctx context.Context, req *entity.ListRequest) (*entity.ListResponse, error)
	MGet(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error)
}

// NewSkillRepository creates a new SkillRepository.
func NewSkillRepository(db *gorm.DB, idGen idgen.IDGenerator) SkillRepository {
	return dal.NewSkillDAO(db, idGen)
}
