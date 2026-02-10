/*
 * Copyright 2025 coze-dev Authors
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

	"github.com/coze-dev/coze-studio/backend/domain/skill/entity"
)

// SkillService defines the domain service interface for skill.
type SkillService interface {
	CreateSkill(ctx context.Context, skill *entity.Skill) (int64, error)
	GetSkill(ctx context.Context, skillID int64) (*entity.Skill, error)
	GetSkillByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error)
	UpdateSkill(ctx context.Context, skill *entity.Skill) error
	DeleteSkill(ctx context.Context, skillID int64) error
	ListSkills(ctx context.Context, req *entity.ListRequest) (*entity.ListResponse, error)
	MGetSkills(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error)
}
