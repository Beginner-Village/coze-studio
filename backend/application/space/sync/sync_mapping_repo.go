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
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SyncMapping struct {
	ID               uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	SourceSpaceID    int64  `gorm:"column:source_space_id"`
	TargetSpaceID    int64  `gorm:"column:target_space_id"`
	ResourceType     string `gorm:"column:resource_type"`
	SourceResourceID int64  `gorm:"column:source_resource_id"`
	TargetResourceID int64  `gorm:"column:target_resource_id"`
	SourceUpdatedAt  int64  `gorm:"column:source_updated_at"`
	ContentHash      string `gorm:"column:content_hash"`
	CreatedAt        int64  `gorm:"column:created_at"`
	UpdatedAt        int64  `gorm:"column:updated_at"`
}

func (SyncMapping) TableName() string {
	return "space_sync_mapping"
}

type SyncMappingStore struct {
	db            *gorm.DB
	sourceSpaceID int64
	targetSpaceID int64
	mappings      map[string]*SyncMapping
}

func NewSyncMappingStore(db *gorm.DB, sourceSpaceID, targetSpaceID int64) *SyncMappingStore {
	return &SyncMappingStore{
		db:            db,
		sourceSpaceID: sourceSpaceID,
		targetSpaceID: targetSpaceID,
		mappings:      make(map[string]*SyncMapping),
	}
}

func mappingKey(resourceType string, sourceID int64) string {
	return resourceType + ":" + strconv.FormatInt(sourceID, 10)
}

func (s *SyncMappingStore) LoadAll(ctx context.Context) error {
	var rows []SyncMapping
	err := s.db.WithContext(ctx).
		Where("source_space_id = ? AND target_space_id = ?", s.sourceSpaceID, s.targetSpaceID).
		Find(&rows).Error
	if err != nil {
		return err
	}
	s.mappings = make(map[string]*SyncMapping, len(rows))
	for i := range rows {
		key := mappingKey(rows[i].ResourceType, rows[i].SourceResourceID)
		s.mappings[key] = &rows[i]
	}
	return nil
}

func (s *SyncMappingStore) GetTargetID(resourceType string, sourceID int64) (int64, bool) {
	key := mappingKey(resourceType, sourceID)
	m, ok := s.mappings[key]
	if !ok {
		return 0, false
	}
	return m.TargetResourceID, true
}

func (s *SyncMappingStore) UpsertMapping(ctx context.Context, tx *gorm.DB, resourceType string, sourceID, targetID, sourceUpdatedAt int64) error {
	db := s.db
	if tx != nil {
		db = tx
	}

	now := time.Now().Unix()
	key := mappingKey(resourceType, sourceID)

	record := SyncMapping{
		SourceSpaceID:    s.sourceSpaceID,
		TargetSpaceID:    s.targetSpaceID,
		ResourceType:     resourceType,
		SourceResourceID: sourceID,
		TargetResourceID: targetID,
		SourceUpdatedAt:  sourceUpdatedAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "source_space_id"},
				{Name: "resource_type"},
				{Name: "source_resource_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"target_resource_id", "source_updated_at", "updated_at",
			}),
		}).
		Create(&record).Error
	if err != nil {
		return fmt.Errorf("upsert sync mapping: %w", err)
	}

	s.mappings[key] = &record
	return nil
}

func (s *SyncMappingStore) RemoveMapping(ctx context.Context, resourceType string, sourceID int64) error {
	err := s.db.WithContext(ctx).
		Where("source_space_id = ? AND resource_type = ? AND source_resource_id = ?",
			s.sourceSpaceID, resourceType, sourceID).
		Delete(&SyncMapping{}).Error
	if err != nil {
		return fmt.Errorf("remove sync mapping: %w", err)
	}

	key := mappingKey(resourceType, sourceID)
	delete(s.mappings, key)
	return nil
}
