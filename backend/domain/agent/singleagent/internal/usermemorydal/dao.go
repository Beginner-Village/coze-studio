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

// Package usermemorydal is the GORM-backed implementation of the per-user
// super-agent memory contract (crossdomain/contract/usermemory.Manager).
package usermemorydal

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	usermemory "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/usermemory"
)

const tableNameUserMemory = "super_agent_user_memory"

// defaultRecallLimit caps how many non-profile entries recall returns.
const defaultRecallLimit = 20

// userMemoryPO is the persistent object for the super_agent_user_memory table.
type userMemoryPO struct {
	ID                   int64          `gorm:"column:id;primaryKey;autoIncrement:true"`
	UserID               int64          `gorm:"column:user_id;not null"`
	SpaceID              int64          `gorm:"column:space_id;not null;default:0"`
	AgentID              int64          `gorm:"column:agent_id;not null;default:0"`
	Kind                 int8           `gorm:"column:kind;not null;default:3"`
	MemKey               string         `gorm:"column:mem_key;not null;default:''"`
	Content              string         `gorm:"column:content;not null"`
	Tags                 string         `gorm:"column:tags;not null;default:''"`
	SourceConversationID int64          `gorm:"column:source_conversation_id;not null;default:0"`
	SourceRunID          int64          `gorm:"column:source_run_id;not null;default:0"`
	Status               int8           `gorm:"column:status;not null;default:1"`
	CreatedAt            int64          `gorm:"column:created_at;not null"`
	UpdatedAt            int64          `gorm:"column:updated_at;not null"`
	DeletedAt            gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (userMemoryPO) TableName() string {
	return tableNameUserMemory
}

// UserMemoryDAO implements usermemory.Manager over a *gorm.DB.
type UserMemoryDAO struct {
	db *gorm.DB
}

var _ usermemory.Manager = (*UserMemoryDAO)(nil)

// NewUserMemoryDAO creates a new UserMemoryDAO.
func NewUserMemoryDAO(db *gorm.DB) *UserMemoryDAO {
	return &UserMemoryDAO{db: db}
}

func (dao *UserMemoryDAO) Save(ctx context.Context, m *usermemory.UserMemory) (*usermemory.UserMemory, error) {
	now := time.Now().UnixMilli()
	if m.Status == 0 {
		m.Status = usermemory.StatusActive
	}
	if m.Kind == 0 {
		m.Kind = usermemory.KindFact
	}
	key := strings.TrimSpace(m.MemKey)

	// Supersede an existing active entry when a stable key matches.
	if key != "" {
		var existing userMemoryPO
		err := dao.db.WithContext(ctx).
			Where("user_id = ? AND mem_key = ? AND status = ?", m.UserID, key, usermemory.StatusActive).
			Order("id DESC").
			First(&existing).Error
		if err == nil {
			existing.Content = m.Content
			existing.Tags = m.Tags
			existing.Kind = int8(m.Kind)
			if m.SpaceID != 0 {
				existing.SpaceID = m.SpaceID
			}
			if m.AgentID != 0 {
				existing.AgentID = m.AgentID
			}
			if m.SourceConversationID != 0 {
				existing.SourceConversationID = m.SourceConversationID
			}
			if m.SourceRunID != 0 {
				existing.SourceRunID = m.SourceRunID
			}
			existing.UpdatedAt = now
			if err := dao.db.WithContext(ctx).Save(&existing).Error; err != nil {
				return nil, err
			}
			return dao.po2do(&existing), nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	po := &userMemoryPO{
		UserID:               m.UserID,
		SpaceID:              m.SpaceID,
		AgentID:              m.AgentID,
		Kind:                 int8(m.Kind),
		MemKey:               key,
		Content:              m.Content,
		Tags:                 m.Tags,
		SourceConversationID: m.SourceConversationID,
		SourceRunID:          m.SourceRunID,
		Status:               m.Status,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := dao.db.WithContext(ctx).Create(po).Error; err != nil {
		return nil, err
	}
	return dao.po2do(po), nil
}

func (dao *UserMemoryDAO) Recall(ctx context.Context, userID int64, q usermemory.RecallQuery) ([]*usermemory.UserMemory, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = defaultRecallLimit
	}

	seen := make(map[int64]struct{})
	out := make([]*usermemory.UserMemory, 0, limit+8)

	// Profile entries are always returned (the durable "who the user is").
	var profiles []*userMemoryPO
	if err := dao.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND kind = ?", userID, usermemory.StatusActive, int8(usermemory.KindProfile)).
		Order("updated_at DESC").
		Find(&profiles).Error; err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if _, ok := seen[p.ID]; ok {
			continue
		}
		seen[p.ID] = struct{}{}
		out = append(out, dao.po2do(p))
	}

	// Other kinds, filtered by query/kind, most-recent first.
	tx := dao.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND kind <> ?", userID, usermemory.StatusActive, int8(usermemory.KindProfile))
	if q.Kind != nil {
		tx = tx.Where("kind = ?", int8(*q.Kind))
	}
	if query := strings.TrimSpace(q.Query); query != "" {
		like := "%" + query + "%"
		tx = tx.Where("content LIKE ? OR tags LIKE ?", like, like)
	}
	var others []*userMemoryPO
	if err := tx.Order("updated_at DESC").Limit(limit).Find(&others).Error; err != nil {
		return nil, err
	}
	for _, p := range others {
		if _, ok := seen[p.ID]; ok {
			continue
		}
		seen[p.ID] = struct{}{}
		out = append(out, dao.po2do(p))
	}
	return out, nil
}

func (dao *UserMemoryDAO) po2do(po *userMemoryPO) *usermemory.UserMemory {
	return &usermemory.UserMemory{
		ID:                   po.ID,
		UserID:               po.UserID,
		SpaceID:              po.SpaceID,
		AgentID:              po.AgentID,
		Kind:                 usermemory.MemoryKind(po.Kind),
		MemKey:               po.MemKey,
		Content:              po.Content,
		Tags:                 po.Tags,
		SourceConversationID: po.SourceConversationID,
		SourceRunID:          po.SourceRunID,
		Status:               po.Status,
		CreatedAt:            po.CreatedAt,
		UpdatedAt:            po.UpdatedAt,
	}
}
