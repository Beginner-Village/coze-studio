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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const tableNameSkillVersion = "skill_version"

// skillVersionPO is the persistent object for the skill_version table. Each row
// is an immutable snapshot of a skill's content at a specific version.
type skillVersionPO struct {
	ID          int64   `gorm:"column:id;primaryKey;autoIncrement:true"`
	SkillID     int64   `gorm:"column:skill_id;not null"`
	Version     int64   `gorm:"column:version;not null"`
	Name        string  `gorm:"column:name;not null"`
	Description *string `gorm:"column:description"`
	Prompt      *string `gorm:"column:prompt"`
	IconURI     string  `gorm:"column:icon_uri;not null"`
	ContentHash string  `gorm:"column:content_hash;not null"`
	CreatedAt   int64   `gorm:"column:created_at;not null"`
}

func (skillVersionPO) TableName() string {
	return tableNameSkillVersion
}

// CreateVersion persists an immutable snapshot of a skill version.
func (dao *SkillDAO) CreateVersion(ctx context.Context, v *entity.SkillVersion) error {
	po := skillVersionDo2po(v)
	if err := dao.db.WithContext(ctx).Create(po).Error; err != nil {
		return errorx.WrapByCode(err, errno.ErrSkillCreateCode)
	}
	return nil
}

// GetVersion returns the snapshot for a specific skill version, or nil if absent.
func (dao *SkillDAO) GetVersion(ctx context.Context, skillID, version int64) (*entity.SkillVersion, error) {
	var po skillVersionPO
	err := dao.db.WithContext(ctx).
		Where("skill_id = ? AND version = ?", skillID, version).
		First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillNotFoundCode)
	}
	return skillVersionPo2do(&po), nil
}

// GetLatestVersion returns the highest-version snapshot for a skill, or nil if none.
func (dao *SkillDAO) GetLatestVersion(ctx context.Context, skillID int64) (*entity.SkillVersion, error) {
	var po skillVersionPO
	err := dao.db.WithContext(ctx).
		Where("skill_id = ?", skillID).
		Order("version DESC").
		First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillNotFoundCode)
	}
	return skillVersionPo2do(&po), nil
}

// createVersionFromSkill writes a snapshot row derived from the current skill PO.
func (dao *SkillDAO) createVersionFromSkill(ctx context.Context, po *skillPO, now int64) error {
	prompt := ""
	if po.Prompt != nil {
		prompt = *po.Prompt
	}
	v := &entity.SkillVersion{
		SkillID:     po.SkillID,
		Version:     po.Version,
		Name:        po.Name,
		Prompt:      prompt,
		IconURI:     po.IconURI,
		ContentHash: contentHash(prompt),
		CreatedAt:   now,
	}
	if po.Description != nil {
		v.Description = *po.Description
	}
	return dao.CreateVersion(ctx, v)
}

func contentHash(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

func skillVersionDo2po(v *entity.SkillVersion) *skillVersionPO {
	createdAt := v.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().UnixMilli()
	}
	po := &skillVersionPO{
		SkillID:     v.SkillID,
		Version:     v.Version,
		Name:        v.Name,
		IconURI:     v.IconURI,
		ContentHash: v.ContentHash,
		CreatedAt:   createdAt,
	}
	if v.Description != "" {
		po.Description = &v.Description
	}
	if v.Prompt != "" {
		po.Prompt = &v.Prompt
	}
	return po
}

func skillVersionPo2do(po *skillVersionPO) *entity.SkillVersion {
	v := &entity.SkillVersion{
		SkillID:     po.SkillID,
		Version:     po.Version,
		Name:        po.Name,
		IconURI:     po.IconURI,
		ContentHash: po.ContentHash,
		CreatedAt:   po.CreatedAt,
	}
	if po.Description != nil {
		v.Description = *po.Description
	}
	if po.Prompt != nil {
		v.Prompt = *po.Prompt
	}
	return v
}
