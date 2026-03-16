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

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
)

// Skill defines the cross-domain interface for skill service.
// Used by agent runtime to load skill details at runtime.
type Skill interface {
	GetSkill(ctx context.Context, skillID int64) (*entity.Skill, error)
	GetSkillByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error)
	MGetSkills(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error)
}

var defaultSVC Skill

func DefaultSVC() Skill {
	return defaultSVC
}

func SetDefaultSVC(c Skill) {
	defaultSVC = c
}
