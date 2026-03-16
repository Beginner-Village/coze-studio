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

	"github.com/ynet-dev/ynet-studio/backend/domain/embedding/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/embedding/repository"
)

type spaceEmbeddingRepository struct {
	db *gorm.DB
}

// NewSpaceEmbeddingRepository creates a new SpaceEmbeddingRepository
func NewSpaceEmbeddingRepository(db *gorm.DB) repository.SpaceEmbeddingRepository {
	return &spaceEmbeddingRepository{db: db}
}

func (r *spaceEmbeddingRepository) Create(ctx context.Context, embedding *entity.SpaceEmbedding) error {
	return r.db.WithContext(ctx).Create(embedding).Error
}

func (r *spaceEmbeddingRepository) GetByID(ctx context.Context, id uint64) (*entity.SpaceEmbedding, error) {
	var embedding entity.SpaceEmbedding
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&embedding).Error
	if err != nil {
		return nil, err
	}
	return &embedding, nil
}

func (r *spaceEmbeddingRepository) GetBySpaceID(ctx context.Context, spaceID uint64) ([]*entity.SpaceEmbedding, error) {
	var embeddings []*entity.SpaceEmbedding
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Order("is_default DESC, created_at ASC").
		Find(&embeddings).Error
	if err != nil {
		return nil, err
	}
	return embeddings, nil
}

func (r *spaceEmbeddingRepository) GetDefaultBySpaceID(ctx context.Context, spaceID uint64) (*entity.SpaceEmbedding, error) {
	var embedding entity.SpaceEmbedding
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND is_default = 1 AND deleted_at IS NULL", spaceID).
		First(&embedding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &embedding, nil
}

func (r *spaceEmbeddingRepository) GetBySpaceIDAndName(ctx context.Context, spaceID uint64, name string) (*entity.SpaceEmbedding, error) {
	var embedding entity.SpaceEmbedding
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND name = ? AND deleted_at IS NULL", spaceID, name).
		First(&embedding).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &embedding, nil
}

func (r *spaceEmbeddingRepository) Update(ctx context.Context, embedding *entity.SpaceEmbedding) error {
	embedding.UpdatedAt = uint64(time.Now().UnixMilli())
	return r.db.WithContext(ctx).
		Model(embedding).
		Select("name", "description", "embedding_type", "config", "max_batch_size", "updated_at").
		Updates(embedding).Error
}

func (r *spaceEmbeddingRepository) UpdateStatus(ctx context.Context, id uint64, status int) error {
	return r.db.WithContext(ctx).
		Model(&entity.SpaceEmbedding{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": uint64(time.Now().UnixMilli()),
		}).Error
}

func (r *spaceEmbeddingRepository) SetDefault(ctx context.Context, spaceID, embeddingID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, clear all defaults for this space
		if err := tx.Model(&entity.SpaceEmbedding{}).
			Where("space_id = ? AND is_default = 1", spaceID).
			Updates(map[string]interface{}{
				"is_default": 0,
				"updated_at": uint64(time.Now().UnixMilli()),
			}).Error; err != nil {
			return err
		}

		// Then, set the new default
		return tx.Model(&entity.SpaceEmbedding{}).
			Where("id = ?", embeddingID).
			Updates(map[string]interface{}{
				"is_default": 1,
				"updated_at": uint64(time.Now().UnixMilli()),
			}).Error
	})
}

func (r *spaceEmbeddingRepository) ClearDefault(ctx context.Context, spaceID uint64) error {
	return r.db.WithContext(ctx).
		Model(&entity.SpaceEmbedding{}).
		Where("space_id = ? AND is_default = 1", spaceID).
		Updates(map[string]interface{}{
			"is_default": 0,
			"updated_at": uint64(time.Now().UnixMilli()),
		}).Error
}

func (r *spaceEmbeddingRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&entity.SpaceEmbedding{}).
		Where("id = ?", id).
		Update("deleted_at", uint64(time.Now().UnixMilli())).Error
}
