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

package release

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ReleaseRepo struct {
	db *gorm.DB
}

func NewReleaseRepo(db *gorm.DB) *ReleaseRepo {
	return &ReleaseRepo{db: db}
}

func (r *ReleaseRepo) Create(ctx context.Context, record *SpaceRelease) error {
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("create release: %w", err)
	}
	return nil
}

func (r *ReleaseRepo) GetByVersion(ctx context.Context, spaceID int64, version string) (*SpaceRelease, error) {
	var record SpaceRelease
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND version = ?", spaceID, version).
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get release by version: %w", err)
	}
	return &record, nil
}

func (r *ReleaseRepo) GetLatestPublished(ctx context.Context, spaceID int64) (*SpaceRelease, error) {
	var record SpaceRelease
	err := r.db.WithContext(ctx).
		Where("space_id = ? AND status = ?", spaceID, StatusPublished).
		Order("created_at DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest published release: %w", err)
	}
	return &record, nil
}

func (r *ReleaseRepo) GetLatest(ctx context.Context, spaceID int64) (*SpaceRelease, error) {
	var record SpaceRelease
	err := r.db.WithContext(ctx).
		Where("space_id = ?", spaceID).
		Order("created_at DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest release: %w", err)
	}
	return &record, nil
}

func (r *ReleaseRepo) ListBySpace(ctx context.Context, spaceID int64, status string, limit, offset int) ([]SpaceRelease, int64, error) {
	query := r.db.WithContext(ctx).Where("space_id = ?", spaceID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Model(&SpaceRelease{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count releases: %w", err)
	}

	var records []SpaceRelease
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list releases: %w", err)
	}
	return records, total, nil
}

func (r *ReleaseRepo) UpdateStatus(ctx context.Context, id uint64, status string, publishedAt *int64) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().Unix(),
	}
	if publishedAt != nil {
		updates["published_at"] = *publishedAt
	}
	if err := r.db.WithContext(ctx).Model(&SpaceRelease{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("update release status: %w", err)
	}
	return nil
}

func (r *ReleaseRepo) VersionExists(ctx context.Context, spaceID int64, version string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SpaceRelease{}).
		Where("space_id = ? AND version = ?", spaceID, version).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check version exists: %w", err)
	}
	return count > 0, nil
}
