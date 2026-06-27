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
	"errors"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

// StrategyDAO implements repository.StrategyDAO using GORM.
type StrategyDAO struct {
	db    *gorm.DB
	idgen idgen.IDGenerator
}

func NewStrategyDAO(db *gorm.DB, gen idgen.IDGenerator) *StrategyDAO {
	return &StrategyDAO{db: db, idgen: gen}
}

// ---------------------------------------------------------------------------
// Strategy
// ---------------------------------------------------------------------------

func (d *StrategyDAO) CreateStrategy(ctx context.Context, s *entity.Strategy) (int64, error) {
	id, err := d.idgen.GenID(ctx)
	if err != nil {
		return 0, err
	}
	m := &model.Strategy{
		ID:          id,
		SpaceID:     s.SpaceID,
		AppID:       s.AppID,
		CreatorID:   s.CreatorID,
		Name:        s.Name,
		Description: s.Description,
		IconURI:     s.IconURI,
		Status:      s.Status,
		Version:     s.Version,
	}
	if err := d.db.WithContext(ctx).Create(m).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (d *StrategyDAO) UpdateStrategy(ctx context.Context, s *entity.Strategy) error {
	return d.db.WithContext(ctx).Model(&model.Strategy{}).
		Where("id = ?", s.ID).
		Updates(map[string]any{
			"space_id":    s.SpaceID,
			"app_id":      s.AppID,
			"creator_id":  s.CreatorID,
			"name":        s.Name,
			"description": s.Description,
			"icon_uri":    s.IconURI,
			"status":      s.Status,
			"version":     s.Version,
		}).Error
}

func (d *StrategyDAO) DeleteStrategy(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.Strategy{}, id).Error
}

func (d *StrategyDAO) GetStrategy(ctx context.Context, id int64) (*entity.Strategy, error) {
	var m model.Strategy
	if err := d.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return strategyFromModel(&m), nil
}

func (d *StrategyDAO) ListStrategy(ctx context.Context, spaceID int64, page, size int) ([]*entity.Strategy, int64, error) {
	q := d.db.WithContext(ctx).Model(&model.Strategy{}).Where("space_id = ?", spaceID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var ms []*model.Strategy
	if err := q.Order("id DESC").
		Offset((page - 1) * size).Limit(size).
		Find(&ms).Error; err != nil {
		return nil, 0, err
	}
	out := make([]*entity.Strategy, 0, len(ms))
	for _, m := range ms {
		out = append(out, strategyFromModel(m))
	}
	return out, total, nil
}

func (d *StrategyDAO) PublishStrategy(ctx context.Context, id int64, version string) error {
	return d.db.WithContext(ctx).Model(&model.Strategy{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":  entity.StatusPublished,
			"version": version,
		}).Error
}

// ---------------------------------------------------------------------------
// Scenario
// ---------------------------------------------------------------------------

func (d *StrategyDAO) CreateScenario(ctx context.Context, sc *entity.Scenario) (int64, error) {
	id, err := d.idgen.GenID(ctx)
	if err != nil {
		return 0, err
	}
	m := &model.StrategyScenario{
		ID:          id,
		StrategyID:  sc.StrategyID,
		Name:        sc.Name,
		Description: sc.Description,
		SortOrder:   sc.SortOrder,
	}
	if err := d.db.WithContext(ctx).Create(m).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (d *StrategyDAO) UpdateScenario(ctx context.Context, sc *entity.Scenario) error {
	return d.db.WithContext(ctx).Model(&model.StrategyScenario{}).
		Where("id = ?", sc.ID).
		Updates(map[string]any{
			"strategy_id": sc.StrategyID,
			"name":        sc.Name,
			"description": sc.Description,
			"sort_order":  sc.SortOrder,
		}).Error
}

func (d *StrategyDAO) DeleteScenario(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.StrategyScenario{}, id).Error
}

func (d *StrategyDAO) GetScenario(ctx context.Context, id int64) (*entity.Scenario, error) {
	var m model.StrategyScenario
	if err := d.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return nil, err
	}
	return scenarioFromModel(&m), nil
}

func (d *StrategyDAO) ListScenarios(ctx context.Context, strategyID int64) ([]*entity.Scenario, error) {
	var ms []*model.StrategyScenario
	if err := d.db.WithContext(ctx).
		Where("strategy_id = ?", strategyID).
		Order("sort_order ASC, id ASC").
		Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.Scenario, 0, len(ms))
	for _, m := range ms {
		out = append(out, scenarioFromModel(m))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Capability
// ---------------------------------------------------------------------------

func (d *StrategyDAO) CreateCapability(ctx context.Context, c *entity.Capability) (int64, error) {
	id, err := d.idgen.GenID(ctx)
	if err != nil {
		return 0, err
	}
	m := &model.StrategyCapability{
		ID:               id,
		StrategyID:       c.StrategyID,
		ScenarioID:       c.ScenarioID,
		Type:             c.Type,
		RefID:            c.RefID,
		RefSubID:         c.RefSubID,
		RefVersion:       c.RefVersion,
		PromptContent:    c.PromptContent,
		RetrieveConfig:   c.RetrieveConfig,
		AliasName:        c.AliasName,
		AliasDescription: c.AliasDescription,
		SortOrder:        c.SortOrder,
	}
	if err := d.db.WithContext(ctx).Create(m).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (d *StrategyDAO) UpdateCapability(ctx context.Context, c *entity.Capability) error {
	return d.db.WithContext(ctx).Model(&model.StrategyCapability{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"strategy_id":       c.StrategyID,
			"scenario_id":       c.ScenarioID,
			"type":              c.Type,
			"ref_id":            c.RefID,
			"ref_sub_id":        c.RefSubID,
			"ref_version":       c.RefVersion,
			"prompt_content":    c.PromptContent,
			"retrieve_config":   c.RetrieveConfig,
			"alias_name":        c.AliasName,
			"alias_description": c.AliasDescription,
			"sort_order":        c.SortOrder,
		}).Error
}

func (d *StrategyDAO) DeleteCapability(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Delete(&model.StrategyCapability{}, id).Error
}

func (d *StrategyDAO) ListCapabilities(ctx context.Context, scenarioID int64) ([]*entity.Capability, error) {
	var ms []*model.StrategyCapability
	if err := d.db.WithContext(ctx).
		Where("scenario_id = ?", scenarioID).
		Order("sort_order ASC, id ASC").
		Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.Capability, 0, len(ms))
	for _, m := range ms {
		out = append(out, capabilityFromModel(m))
	}
	return out, nil
}

func (d *StrategyDAO) MGetCapabilities(ctx context.Context, ids []int64) ([]*entity.Capability, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var ms []*model.StrategyCapability
	if err := d.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&ms).Error; err != nil {
		return nil, err
	}
	out := make([]*entity.Capability, 0, len(ms))
	for _, m := range ms {
		out = append(out, capabilityFromModel(m))
	}
	return out, nil
}

func (d *StrategyDAO) ListCapabilityIDsByStrategies(ctx context.Context, strategyIDs []int64) ([]int64, error) {
	if len(strategyIDs) == 0 {
		return nil, nil
	}
	var ids []int64
	if err := d.db.WithContext(ctx).
		Model(&model.StrategyCapability{}).
		Where("strategy_id IN ?", strategyIDs).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// ---------------------------------------------------------------------------
// entity <-> model converters
// ---------------------------------------------------------------------------

func strategyFromModel(m *model.Strategy) *entity.Strategy {
	return &entity.Strategy{
		ID:          m.ID,
		SpaceID:     m.SpaceID,
		AppID:       m.AppID,
		CreatorID:   m.CreatorID,
		Name:        m.Name,
		Description: m.Description,
		IconURI:     m.IconURI,
		Status:      m.Status,
		Version:     m.Version,
	}
}

func scenarioFromModel(m *model.StrategyScenario) *entity.Scenario {
	return &entity.Scenario{
		ID:          m.ID,
		StrategyID:  m.StrategyID,
		Name:        m.Name,
		Description: m.Description,
		SortOrder:   m.SortOrder,
	}
}

func capabilityFromModel(m *model.StrategyCapability) *entity.Capability {
	return &entity.Capability{
		ID:               m.ID,
		StrategyID:       m.StrategyID,
		ScenarioID:       m.ScenarioID,
		Type:             m.Type,
		RefID:            m.RefID,
		RefSubID:         m.RefSubID,
		RefVersion:       m.RefVersion,
		PromptContent:    m.PromptContent,
		RetrieveConfig:   m.RetrieveConfig,
		AliasName:        m.AliasName,
		AliasDescription: m.AliasDescription,
		SortOrder:        m.SortOrder,
	}
}
