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

package spacesync

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type SyncHistory struct {
	ID                  uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	SourceSpaceID       int64           `gorm:"column:source_space_id"`
	TargetSpaceID       int64           `gorm:"column:target_space_id"`
	SyncType            string          `gorm:"column:sync_type"`
	Version             *string         `gorm:"column:version"`
	ExportTime          int64           `gorm:"column:export_time"`
	ImportTime          *int64          `gorm:"column:import_time"`
	Statistics          json.RawMessage `gorm:"column:statistics;type:json"`
	Status              int8            `gorm:"column:status"`
	ErrorMsg            *string         `gorm:"column:error_msg"`
	PackageFileName     *string         `gorm:"column:package_file_name"`
	SnapshotKey         *string         `gorm:"column:snapshot_key"`
	RollbackFromVersion *string         `gorm:"column:rollback_from_version"`
	CreatedAt           int64           `gorm:"column:created_at"`
}

func (SyncHistory) TableName() string {
	return "space_sync_history"
}

type SyncHistoryRepo struct {
	db *gorm.DB
}

func NewSyncHistoryRepo(db *gorm.DB) *SyncHistoryRepo {
	return &SyncHistoryRepo{db: db}
}

func (r *SyncHistoryRepo) Create(ctx context.Context, record *SyncHistory) error {
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("create sync history: %w", err)
	}
	return nil
}

func (r *SyncHistoryRepo) UpdateStatus(ctx context.Context, id uint64, status int8, errMsg *string) error {
	updates := map[string]interface{}{
		"status":    status,
		"error_msg": errMsg,
	}
	if err := r.db.WithContext(ctx).Model(&SyncHistory{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("update sync history status: %w", err)
	}
	return nil
}

func (r *SyncHistoryRepo) ListBySpace(ctx context.Context, targetSpaceID int64, limit int) ([]SyncHistory, error) {
	var records []SyncHistory
	err := r.db.WithContext(ctx).
		Where("target_space_id = ?", targetSpaceID).
		Order("export_time DESC").
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("list sync history: %w", err)
	}
	return records, nil
}

func (r *SyncHistoryRepo) GetLastExport(ctx context.Context, sourceSpaceID int64) (*SyncHistory, error) {
	var record SyncHistory
	err := r.db.WithContext(ctx).
		Where("source_space_id = ?", sourceSpaceID).
		Order("export_time DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get last export: %w", err)
	}
	return &record, nil
}

func (r *SyncHistoryRepo) GetCurrentVersion(ctx context.Context, targetSpaceID int64) (*SyncHistory, error) {
	var record SyncHistory
	err := r.db.WithContext(ctx).
		Where("target_space_id = ? AND status = 1 AND version IS NOT NULL AND version != ''", targetSpaceID).
		Order("created_at DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get current version: %w", err)
	}
	return &record, nil
}

func (r *SyncHistoryRepo) GetByVersion(ctx context.Context, targetSpaceID int64, version string) (*SyncHistory, error) {
	var record SyncHistory
	err := r.db.WithContext(ctx).
		Where("target_space_id = ? AND version = ? AND status = 1", targetSpaceID, version).
		Order("created_at DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get history by version: %w", err)
	}
	return &record, nil
}
