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

	crossskill "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/skill"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/service"
)

type impl struct {
	DomainSVC service.SkillService
}

// InitDomainService initializes the cross-domain skill service.
func InitDomainService(c service.SkillService) crossskill.Skill {
	svc := &impl{DomainSVC: c}
	crossskill.SetDefaultSVC(svc)
	return svc
}

func (i *impl) GetSkill(ctx context.Context, skillID int64) (*entity.Skill, error) {
	return i.DomainSVC.GetSkill(ctx, skillID)
}

func (i *impl) GetSkillByName(ctx context.Context, spaceID int64, name string) (*entity.Skill, error) {
	return i.DomainSVC.GetSkillByName(ctx, spaceID, name)
}

func (i *impl) MGetSkills(ctx context.Context, skillIDs []int64) ([]*entity.Skill, error) {
	return i.DomainSVC.MGetSkills(ctx, skillIDs)
}
