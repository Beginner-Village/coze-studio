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

package skill

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/repository"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// SkillApplicationSVC is the global skill application service instance.
var SkillApplicationSVC *SkillApplicationService

// ServiceComponents holds dependencies for the skill application service.
type ServiceComponents struct {
	IDGen idgen.IDGenerator
	DB    *gorm.DB
}

// SkillApplicationService orchestrates skill domain operations.
type SkillApplicationService struct {
	DomainSVC service.SkillService
}

// InitService initializes the skill application service.
func InitService(c *ServiceComponents) *SkillApplicationService {
	domainComponents := &service.Components{
		SkillRepo: repository.NewSkillRepository(c.DB, c.IDGen),
	}

	domainSVC := service.NewService(domainComponents)
	SkillApplicationSVC = &SkillApplicationService{
		DomainSVC: domainSVC,
	}
	return SkillApplicationSVC
}

// CreateSkill creates a new skill.
func (s *SkillApplicationService) CreateSkill(ctx context.Context, spaceID int64, name, description, prompt, iconURI string, files map[string]string) (*entity.Skill, error) {
	if spaceID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}

	// 真·文件夹技能:若提供了 files 且含 SKILL.md,以 SKILL.md frontmatter 为单一事实源,
	// 解析出 name/description,prompt 取其正文。未显式传入时用解析值。
	if md, ok := files["SKILL.md"]; ok && md != "" {
		fmName, fmDesc, body := parseSkillFrontmatter(md)
		if name == "" {
			name = fmName
		}
		if description == "" {
			description = fmDesc
		}
		if prompt == "" {
			prompt = body
		}
	}

	if name == "" {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "name is required"))
	}

	userID := ctxutil.MustGetUIDFromCtx(ctx)

	skill := &entity.Skill{
		SpaceID:     spaceID,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Files:       files,
		IconURI:     iconURI,
		CreatorID:   userID,
	}

	skillID, err := s.DomainSVC.CreateSkill(ctx, skill)
	if err != nil {
		return nil, err
	}

	return s.DomainSVC.GetSkill(ctx, skillID)
}

// GetSkill gets a skill by ID.
func (s *SkillApplicationService) GetSkill(ctx context.Context, skillID, spaceID int64) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	skill, err := s.DomainSVC.GetSkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, errorx.New(errno.ErrSkillNotFoundCode, errorx.KV("msg", "skill not found"))
	}
	if spaceID > 0 && skill.SpaceID != spaceID {
		return nil, errorx.New(errno.ErrSkillPermissionCode, errorx.KV("msg", "skill does not belong to this space"))
	}

	return skill, nil
}

// UpdateSkill updates a skill.
func (s *SkillApplicationService) UpdateSkill(ctx context.Context, skillID, spaceID int64, name, description, prompt, iconURI string) (*entity.Skill, error) {
	if skillID <= 0 {
		return nil, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	// Verify skill exists and belongs to the space
	existing, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return nil, err
	}

	skill := &entity.Skill{
		SkillID:     existing.SkillID,
		SpaceID:     existing.SpaceID,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		IconURI:     iconURI,
	}

	if err := s.DomainSVC.UpdateSkill(ctx, skill); err != nil {
		return nil, err
	}

	return s.DomainSVC.GetSkill(ctx, skillID)
}

// DeleteSkill deletes a skill.
func (s *SkillApplicationService) DeleteSkill(ctx context.Context, skillID, spaceID int64) error {
	if skillID <= 0 {
		return errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "skill_id is required"))
	}

	// Verify skill exists and belongs to the space
	_, err := s.GetSkill(ctx, skillID, spaceID)
	if err != nil {
		return err
	}

	return s.DomainSVC.DeleteSkill(ctx, skillID)
}

// ListSkills lists skills in a space.
func (s *SkillApplicationService) ListSkills(ctx context.Context, spaceID int64, page, pageSize int32, keyword string) ([]*entity.Skill, int32, error) {
	if spaceID <= 0 {
		return nil, 0, errorx.New(errno.ErrSkillInvalidParamCode, errorx.KV("msg", "space_id is required"))
	}

	resp, err := s.DomainSVC.ListSkills(ctx, &entity.ListRequest{
		SpaceID:  spaceID,
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Skills, resp.Total, nil
}
