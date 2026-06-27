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
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const tableNameSkill = "skill"

// skillPO is the persistent object for skill table.
type skillPO struct {
	ID               int64          `gorm:"column:id;primaryKey;autoIncrement:true"`
	SkillID          int64          `gorm:"column:skill_id;not null"`
	SpaceID          int64          `gorm:"column:space_id;not null"`
	Name             string         `gorm:"column:name;not null"`
	Description      *string        `gorm:"column:description"`
	Prompt           *string        `gorm:"column:prompt"`
	Files            *string        `gorm:"column:files"`
	IconURI          string         `gorm:"column:icon_uri;not null"`
	CreatorID        int64          `gorm:"column:creator_id;not null"`
	Status           int8           `gorm:"column:status;not null;default:1"`
	Version          int64          `gorm:"column:version;not null;default:1"`
	PublishScope     int8           `gorm:"column:publish_scope;not null;default:1"`
	PublishedVersion int64          `gorm:"column:published_version;not null;default:0"`
	PublishedAt      int64          `gorm:"column:published_at;not null;default:0"`
	PublishedBy      int64          `gorm:"column:published_by;not null;default:0"`
	ReviewStatus     int8           `gorm:"column:review_status;not null;default:2"`
	ReviewNote       *string        `gorm:"column:review_note"`
	ReviewerID       int64          `gorm:"column:reviewer_id;not null;default:0"`
	ReviewedAt       int64          `gorm:"column:reviewed_at;not null;default:0"`
	CreatedAt        int64          `gorm:"column:created_at;not null"`
	UpdatedAt        int64          `gorm:"column:updated_at;not null"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (skillPO) TableName() string {
	return tableNameSkill
}

// SkillDAO implements SkillRepository.
type SkillDAO struct {
	db    *gorm.DB
	idGen idgen.IDGenerator
}

// NewSkillDAO creates a new SkillDAO.
func NewSkillDAO(db *gorm.DB, idGen idgen.IDGenerator) *SkillDAO {
	return &SkillDAO{db: db, idGen: idGen}
}

func (dao *SkillDAO) Create(ctx context.Context, skill *entity.Skill) (int64, error) {
	id, err := dao.idGen.GenID(ctx)
	if err != nil {
		return 0, errorx.WrapByCode(err, errno.ErrSkillIDGenFailCode, errorx.KV("msg", "Create"))
	}

	now := time.Now().UnixMilli()
	po := dao.do2po(skill)
	po.SkillID = id
	po.Version = 1
	po.CreatedAt = now
	po.UpdatedAt = now

	if err := dao.db.WithContext(ctx).Create(po).Error; err != nil {
		return 0, errorx.WrapByCode(err, errno.ErrSkillCreateCode)
	}
	if err := dao.createVersionFromSkill(ctx, po, now); err != nil {
		return 0, errorx.WrapByCode(err, errno.ErrSkillCreateCode)
	}
	return id, nil
}

func (dao *SkillDAO) Get(ctx context.Context, skillID int64) (*entity.Skill, error) {
	var po skillPO
	// Only load active skills so disabled skills are not used at runtime.
	err := dao.db.WithContext(ctx).
		Where("skill_id = ? AND status = ?", skillID, entity.SkillStatusActive).
		First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillNotFoundCode)
	}
	return dao.po2do(&po), nil
}

func (dao *SkillDAO) GetByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error) {
	var po skillPO
	err := dao.db.WithContext(ctx).Where("space_id = ? AND name = ?", spaceID, name).First(&po).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillNotFoundCode)
	}
	return dao.po2do(&po), nil
}

func (dao *SkillDAO) Update(ctx context.Context, skill *entity.Skill) error {
	now := time.Now().UnixMilli()
	updates := map[string]interface{}{
		"updated_at": now,
		// Bump the content version on every update so each change is snapshotted.
		"version": gorm.Expr("version + 1"),
	}
	if skill.Name != "" {
		updates["name"] = skill.Name
	}
	if skill.Description != "" {
		updates["description"] = skill.Description
	}
	if skill.Prompt != "" {
		updates["prompt"] = skill.Prompt
	}
	if skill.Files != nil {
		if b, err := json.Marshal(skill.Files); err == nil {
			updates["files"] = string(b)
		}
	}
	if skill.IconURI != "" {
		updates["icon_uri"] = skill.IconURI
	}

	err := dao.db.WithContext(ctx).Model(&skillPO{}).Where("skill_id = ?", skill.SkillID).Updates(updates).Error
	if err != nil {
		return errorx.WrapByCode(err, errno.ErrSkillUpdateCode)
	}

	// Read back the now-current row (without status filter, version may have
	// just incremented) and persist an immutable snapshot of it.
	var po skillPO
	if err := dao.db.WithContext(ctx).Where("skill_id = ?", skill.SkillID).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errorx.WrapByCode(err, errno.ErrSkillUpdateCode)
	}
	if err := dao.createVersionFromSkill(ctx, &po, now); err != nil {
		return errorx.WrapByCode(err, errno.ErrSkillUpdateCode)
	}
	return nil
}

func (dao *SkillDAO) Delete(ctx context.Context, skillID int64) error {
	err := dao.db.WithContext(ctx).Where("skill_id = ?", skillID).Delete(&skillPO{}).Error
	if err != nil {
		return errorx.WrapByCode(err, errno.ErrSkillDeleteCode)
	}
	return nil
}

func (dao *SkillDAO) Publish(ctx context.Context, skillID int64, scope, reviewStatus int8, version, publisherID, publishedAt int64) error {
	err := dao.db.WithContext(ctx).Model(&skillPO{}).
		Where("skill_id = ? AND status = ?", skillID, entity.SkillStatusActive).
		Updates(map[string]interface{}{
			"publish_scope":     scope,
			"published_version": version,
			"published_at":      publishedAt,
			"published_by":      publisherID,
			"review_status":     reviewStatus,
			"updated_at":        time.Now().UnixMilli(),
		}).Error
	if err != nil {
		return errorx.WrapByCode(err, errno.ErrSkillUpdateCode)
	}
	return nil
}

// SetReviewStatus records a platform review decision for a skill.
func (dao *SkillDAO) SetReviewStatus(ctx context.Context, skillID int64, reviewStatus int8, note string, reviewerID, reviewedAt int64) error {
	err := dao.db.WithContext(ctx).Model(&skillPO{}).
		Where("skill_id = ? AND status = ?", skillID, entity.SkillStatusActive).
		Updates(map[string]interface{}{
			"review_status": reviewStatus,
			"review_note":   note,
			"reviewer_id":   reviewerID,
			"reviewed_at":   reviewedAt,
			"updated_at":    time.Now().UnixMilli(),
		}).Error
	if err != nil {
		return errorx.WrapByCode(err, errno.ErrSkillUpdateCode)
	}
	return nil
}

// ListPendingReviews lists global-scope skills awaiting platform review.
func (dao *SkillDAO) ListPendingReviews(ctx context.Context, req *entity.PendingReviewListRequest) (*entity.ListResponse, error) {
	query := dao.db.WithContext(ctx).Model(&skillPO{}).
		Where("status = ? AND publish_scope = ? AND review_status = ?",
			entity.SkillStatusActive, entity.SkillPublishScopeGlobal, entity.SkillReviewStatusPending)
	if req.SpaceID > 0 {
		query = query.Where("space_id = ?", req.SpaceID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var pos []skillPO
	err := query.Order("published_at DESC, updated_at DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&pos).Error
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	skills := make([]*entity.Skill, 0, len(pos))
	for _, po := range pos {
		skills = append(skills, dao.po2do(&po))
	}
	return &entity.ListResponse{Skills: skills, Total: int32(total)}, nil
}

func (dao *SkillDAO) List(ctx context.Context, req *entity.ListRequest) (*entity.ListResponse, error) {
	// Only list active skills so disabled skills are hidden.
	query := dao.db.WithContext(ctx).Model(&skillPO{}).
		Where("space_id = ? AND status = ?", req.SpaceID, entity.SkillStatusActive)

	if req.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+req.Keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var pos []skillPO
	err := query.Order("created_at DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&pos).Error
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	skills := make([]*entity.Skill, 0, len(pos))
	for _, po := range pos {
		skills = append(skills, dao.po2do(&po))
	}

	return &entity.ListResponse{
		Skills: skills,
		Total:  int32(total),
	}, nil
}

func (dao *SkillDAO) ListMarketplace(ctx context.Context, req *entity.MarketplaceListRequest) (*entity.ListResponse, error) {
	query := dao.db.WithContext(ctx).Model(&skillPO{}).
		Where("status = ?", entity.SkillStatusActive)

	switch req.Scope {
	case entity.SkillPublishScopeGlobal:
		// Global marketplace only surfaces approved skills.
		query = query.Where("publish_scope = ? AND review_status = ?",
			entity.SkillPublishScopeGlobal, entity.SkillReviewStatusApproved)
	case entity.SkillPublishScopeSpace:
		query = query.Where("publish_scope = ? AND space_id = ?", entity.SkillPublishScopeSpace, req.SpaceID)
	default:
		if req.SpaceID > 0 {
			// Space-scoped skills are self-governed (no review); global must be approved.
			query = query.Where(
				"((publish_scope = ? AND review_status = ?) OR (publish_scope = ? AND space_id = ?))",
				entity.SkillPublishScopeGlobal,
				entity.SkillReviewStatusApproved,
				entity.SkillPublishScopeSpace,
				req.SpaceID,
			)
		} else {
			query = query.Where("publish_scope = ? AND review_status = ?",
				entity.SkillPublishScopeGlobal, entity.SkillReviewStatusApproved)
		}
	}

	if req.Keyword != "" {
		like := "%" + req.Keyword + "%"
		query = query.Where("(name LIKE ? OR description LIKE ?)", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var pos []skillPO
	err := query.Order("published_at DESC, updated_at DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&pos).Error
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	skills := make([]*entity.Skill, 0, len(pos))
	for _, po := range pos {
		skills = append(skills, dao.po2do(&po))
	}

	return &entity.ListResponse{
		Skills: skills,
		Total:  int32(total),
	}, nil
}

func (dao *SkillDAO) MGet(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error) {
	if len(skillIDs) == 0 {
		return nil, nil
	}

	var pos []skillPO
	// Only load active skills so disabled skills are not used at runtime.
	err := dao.db.WithContext(ctx).
		Where("skill_id IN ? AND status = ?", skillIDs, entity.SkillStatusActive).
		Find(&pos).Error
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSkillListCode)
	}

	skills := make([]*entity.Skill, 0, len(pos))
	for _, po := range pos {
		skills = append(skills, dao.po2do(&po))
	}
	return skills, nil
}

func (dao *SkillDAO) do2po(do *entity.Skill) *skillPO {
	po := &skillPO{
		SkillID:          do.SkillID,
		SpaceID:          do.SpaceID,
		Name:             do.Name,
		IconURI:          do.IconURI,
		CreatorID:        do.CreatorID,
		Status:           do.Status,
		Version:          do.Version,
		PublishScope:     do.PublishScope,
		PublishedVersion: do.PublishedVersion,
		PublishedAt:      do.PublishedAt,
		PublishedBy:      do.PublishedBy,
		ReviewStatus:     do.ReviewStatus,
		ReviewerID:       do.ReviewerID,
		ReviewedAt:       do.ReviewedAt,
		CreatedAt:        do.CreatedAt,
		UpdatedAt:        do.UpdatedAt,
	}
	if do.ReviewNote != "" {
		po.ReviewNote = &do.ReviewNote
	}
	if do.Description != "" {
		po.Description = &do.Description
	}
	if do.Prompt != "" {
		po.Prompt = &do.Prompt
	}
	if do.Files != nil {
		if b, err := json.Marshal(do.Files); err == nil {
			s := string(b)
			po.Files = &s
		}
	}
	return po
}

func (dao *SkillDAO) po2do(po *skillPO) *entity.Skill {
	do := &entity.Skill{
		SkillID:          po.SkillID,
		SpaceID:          po.SpaceID,
		Name:             po.Name,
		IconURI:          po.IconURI,
		CreatorID:        po.CreatorID,
		Status:           po.Status,
		Version:          po.Version,
		PublishScope:     po.PublishScope,
		PublishedVersion: po.PublishedVersion,
		PublishedAt:      po.PublishedAt,
		PublishedBy:      po.PublishedBy,
		ReviewStatus:     po.ReviewStatus,
		ReviewerID:       po.ReviewerID,
		ReviewedAt:       po.ReviewedAt,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
	if po.ReviewNote != nil {
		do.ReviewNote = *po.ReviewNote
	}
	if po.Description != nil {
		do.Description = *po.Description
	}
	if po.Prompt != nil {
		do.Prompt = *po.Prompt
	}
	if po.Files != nil && *po.Files != "" {
		m := map[string]string{}
		if err := json.Unmarshal([]byte(*po.Files), &m); err == nil {
			do.Files = m
		}
	}
	return do
}
