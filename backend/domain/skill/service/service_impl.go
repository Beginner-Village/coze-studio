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

package service

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/repository"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// Components holds the dependencies for the skill service.
type Components struct {
	SkillRepo repository.SkillRepository
}

type skillServiceImpl struct {
	repo repository.SkillRepository
}

// NewService creates a new SkillService.
func NewService(c *Components) SkillService {
	return &skillServiceImpl{
		repo: c.SkillRepo,
	}
}

func (s *skillServiceImpl) CreateSkill(ctx context.Context, skill *entity.Skill) (int64, error) {
	// Check duplicate name within the same space
	existing, err := s.repo.GetByName(ctx, skill.SpaceID, skill.Name)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, errorx.New(errno.ErrSkillDuplicateNameCode, errorx.KV("name", skill.Name))
	}

	skill.Status = 1 // active
	return s.repo.Create(ctx, skill)
}

func (s *skillServiceImpl) GetSkill(ctx context.Context, skillID int64) (*entity.Skill, error) {
	return s.repo.Get(ctx, skillID)
}

func (s *skillServiceImpl) GetSkillByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error) {
	return s.repo.GetByName(ctx, spaceID, name)
}

func (s *skillServiceImpl) UpdateSkill(ctx context.Context, skill *entity.Skill) error {
	// If name is being updated, check for duplicates
	if skill.Name != "" {
		existing, err := s.repo.GetByName(ctx, skill.SpaceID, skill.Name)
		if err != nil {
			return err
		}
		if existing != nil && existing.SkillID != skill.SkillID {
			return errorx.New(errno.ErrSkillDuplicateNameCode, errorx.KV("name", skill.Name))
		}
	}

	return s.repo.Update(ctx, skill)
}

func (s *skillServiceImpl) DeleteSkill(ctx context.Context, skillID int64) error {
	return s.repo.Delete(ctx, skillID)
}

func (s *skillServiceImpl) ListSkills(ctx context.Context, req *entity.ListRequest) (*entity.ListResponse, error) {
	return s.repo.List(ctx, req)
}

func (s *skillServiceImpl) MGetSkills(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error) {
	return s.repo.MGet(ctx, skillIDs)
}
