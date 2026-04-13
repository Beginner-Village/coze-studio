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
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/repository"
)

type spaceRerankRepository struct {
	db *gorm.DB
}

// NewSpaceRerankRepository creates a new SpaceRerankRepository
func NewSpaceRerankRepository(db *gorm.DB) repository.SpaceRerankRepository {
	return &spaceRerankRepository{db: db}
}

func (r *spaceRerankRepository) Create(ctx context.Context, rerank *entity.SpaceRerank) error {
	return r.db.WithContext(ctx).Create(rerank).Error
}

func (r *spaceRerankRepository) GetByID(ctx context.Context, id uint64) (*entity.SpaceRerank, error) {
	var rerank entity.SpaceRerank
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&rerank).Error
	if err != nil {
		return nil, err
	}
	return &rerank, nil
}

func (r *spaceRerankRepository) GetBySpaceID(ctx context.Context, spaceID uint64) ([]*entity.SpaceRerank, error) {
	var reranks []*entity.SpaceRerank
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Order("is_default DESC, created_at ASC").
		Find(&reranks).Error
	if err != nil {
		return nil, err
	}
	return reranks, nil
}

func (r *spaceRerankRepository) GetDefaultBySpaceID(ctx context.Context, spaceID uint64) (*entity.SpaceRerank, error) {
	var rerank entity.SpaceRerank
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND is_default = 1 AND deleted_at IS NULL", spaceID).
		First(&rerank).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &rerank, nil
}

func (r *spaceRerankRepository) GetBySpaceIDAndName(ctx context.Context, spaceID uint64, name string) (*entity.SpaceRerank, error) {
	var rerank entity.SpaceRerank
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND name = ? AND deleted_at IS NULL", spaceID, name).
		First(&rerank).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &rerank, nil
}

func (r *spaceRerankRepository) Update(ctx context.Context, rerank *entity.SpaceRerank) error {
	rerank.UpdatedAt = uint64(time.Now().UnixMilli())
	return r.db.WithContext(ctx).
		Model(rerank).
		Select("name", "description", "rerank_type", "config", "updated_at").
		Updates(rerank).Error
}

func (r *spaceRerankRepository) UpdateStatus(ctx context.Context, id uint64, status int) error {
	return r.db.WithContext(ctx).
		Model(&entity.SpaceRerank{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": uint64(time.Now().UnixMilli()),
		}).Error
}

func (r *spaceRerankRepository) SetDefault(ctx context.Context, spaceID, rerankID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, clear all defaults for this space
		if err := tx.Model(&entity.SpaceRerank{}).
			Where("space_id = ? AND is_default = 1", spaceID).
			Updates(map[string]interface{}{
				"is_default": 0,
				"updated_at": uint64(time.Now().UnixMilli()),
			}).Error; err != nil {
			return err
		}

		// Then, set the new default
		return tx.Model(&entity.SpaceRerank{}).
			Where("id = ?", rerankID).
			Updates(map[string]interface{}{
				"is_default": 1,
				"updated_at": uint64(time.Now().UnixMilli()),
			}).Error
	})
}

func (r *spaceRerankRepository) ClearDefault(ctx context.Context, spaceID uint64) error {
	return r.db.WithContext(ctx).
		Model(&entity.SpaceRerank{}).
		Where("space_id = ? AND is_default = 1", spaceID).
		Updates(map[string]interface{}{
			"is_default": 0,
			"updated_at": uint64(time.Now().UnixMilli()),
		}).Error
}

func (r *spaceRerankRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&entity.SpaceRerank{}).
		Where("id = ?", id).
		Update("deleted_at", uint64(time.Now().UnixMilli())).Error
}
