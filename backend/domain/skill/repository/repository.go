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

package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/internal/dal"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

// SkillRepository defines the interface for skill data access.
type SkillRepository interface {
	Create(ctx context.Context, skill *entity.Skill) (int64, error)
	Get(ctx context.Context, skillID int64) (*entity.Skill, error)
	GetByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error)
	Update(ctx context.Context, skill *entity.Skill) error
	Delete(ctx context.Context, skillID int64) error
	Publish(ctx context.Context, skillID int64, scope, reviewStatus int8, version, publisherID, publishedAt int64) error
	SetReviewStatus(ctx context.Context, skillID int64, reviewStatus int8, note string, reviewerID, reviewedAt int64) error
	ListPendingReviews(ctx context.Context, req *entity.PendingReviewListRequest) (*entity.ListResponse, error)
	List(ctx context.Context, req *entity.ListRequest) (*entity.ListResponse, error)
	ListMarketplace(ctx context.Context, req *entity.MarketplaceListRequest) (*entity.ListResponse, error)
	MGet(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error)

	// CreateVersion persists an immutable snapshot of a skill version.
	CreateVersion(ctx context.Context, v *entity.SkillVersion) error
	// GetVersion returns the snapshot for a specific skill version, or nil if absent.
	GetVersion(ctx context.Context, skillID, version int64) (*entity.SkillVersion, error)
	// GetLatestVersion returns the highest-version snapshot for a skill, or nil if none.
	GetLatestVersion(ctx context.Context, skillID int64) (*entity.SkillVersion, error)
}

// NewSkillRepository creates a new SkillRepository.
func NewSkillRepository(db *gorm.DB, idGen idgen.IDGenerator) SkillRepository {
	return dal.NewSkillDAO(db, idGen)
}
