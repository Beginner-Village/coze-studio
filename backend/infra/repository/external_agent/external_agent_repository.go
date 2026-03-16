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

package external_agent

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/mysql"
	"github.com/ynet-dev/ynet-studio/backend/api/model/ynet_agent"
)

type ExternalAgentRepository struct {
	db *gorm.DB
}

func NewExternalAgentRepository() (*ExternalAgentRepository, error) {
	db, err := mysql.New()
	if err != nil {
		return nil, err
	}
	return &ExternalAgentRepository{
		db: db,
	}, nil
}

// GetListBySpaceID 根据空间ID获取外部智能体列表
func (r *ExternalAgentRepository) GetListBySpaceID(ctx context.Context, spaceID int64, page, pageSize int32) ([]*ynet_agent.HiAgentInfo, int32, error) {
	// 计算偏移量
	offset := (page - 1) * pageSize

	// 定义表结构
	type ExternalAgentConfig struct {
		ID          int64   `gorm:"column:id"`
		SpaceID     int64   `gorm:"column:space_id"`
		Name        string  `gorm:"column:name"`
		Description *string `gorm:"column:description"`
		Platform    string  `gorm:"column:platform"`
		AgentURL    string  `gorm:"column:agent_url"`
		AgentKey    *string `gorm:"column:agent_key"`
		AgentID     *string `gorm:"column:agent_id"`
		AppID       *string `gorm:"column:app_id"`
		Icon        *string `gorm:"column:icon"`
		Category    *string `gorm:"column:category"`
		Status      int32   `gorm:"column:status"`
		Metadata    *string `gorm:"column:metadata"`
		CreatedBy   int64   `gorm:"column:created_by"`
		UpdatedBy   *int64  `gorm:"column:updated_by"`
		CreatedAt   string  `gorm:"column:created_at"`
		UpdatedAt   string  `gorm:"column:updated_at"`
	}

	// 查询总数
	var total int64
	err := r.db.WithContext(ctx).Table("external_agent_config").
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count agents: %w", err)
	}

	// 查询数据
	var configs []ExternalAgentConfig
	err = r.db.WithContext(ctx).Table("external_agent_config").
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Order("created_at DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&configs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query agents: %w", err)
	}

	// 转换为响应格式
	var agents []*ynet_agent.HiAgentInfo
	for _, config := range configs {
		agent := &ynet_agent.HiAgentInfo{
			ID:          config.ID,
			SpaceID:     config.SpaceID,
			Name:        config.Name,
			Description: config.Description,
			Platform:    config.Platform,
			AgentURL:    config.AgentURL,
			AgentID:     config.AgentID,
			AppID:       config.AppID,
			Icon:        config.Icon,
			Category:    config.Category,
			Status:      config.Status,
			Metadata:    config.Metadata,
			CreatedBy:   config.CreatedBy,
			UpdatedBy:   config.UpdatedBy,
			CreatedAt:   config.CreatedAt,
			UpdatedAt:   config.UpdatedAt,
		}

		// 不返回密钥明文
		agent.AgentKey = nil

		agents = append(agents, agent)
	}

	return agents, int32(total), nil
}

// ExternalAgentConfigModel 数据库模型
type ExternalAgentConfigModel struct {
	ID          int64   `gorm:"column:id;primaryKey;autoIncrement"`
	SpaceID     int64   `gorm:"column:space_id"`
	Name        string  `gorm:"column:name"`
	Description *string `gorm:"column:description"`
	Platform    string  `gorm:"column:platform"`
	AgentURL    string  `gorm:"column:agent_url"`
	AgentKey    *string `gorm:"column:agent_key"`
	AgentID     *string `gorm:"column:agent_id"`
	AppID       *string `gorm:"column:app_id"`
	Icon        *string `gorm:"column:icon"`
	Category    *string `gorm:"column:category"`
	Status      int32   `gorm:"column:status"`
	Metadata    *string `gorm:"column:metadata"`
	CreatedBy   int64   `gorm:"column:created_by"`
}

func (ExternalAgentConfigModel) TableName() string {
	return "external_agent_config"
}

// CreateAgent 创建外部智能体
func (r *ExternalAgentRepository) CreateAgent(ctx context.Context, agent *ynet_agent.HiAgentInfo) (*ynet_agent.HiAgentInfo, error) {
	model := &ExternalAgentConfigModel{
		SpaceID:     agent.SpaceID,
		Name:        agent.Name,
		Description: agent.Description,
		Platform:    agent.Platform,
		AgentURL:    agent.AgentURL,
		AgentKey:    agent.AgentKey,
		AgentID:     agent.AgentID,
		AppID:       agent.AppID,
		Icon:        agent.Icon,
		Category:    agent.Category,
		Status:      agent.Status,
		Metadata:    agent.Metadata,
		CreatedBy:   agent.CreatedBy,
	}

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create agent: %w", result.Error)
	}

	agent.ID = model.ID
	return agent, nil
}

// UpdateAgent 更新外部智能体
func (r *ExternalAgentRepository) UpdateAgent(ctx context.Context, id int64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&ExternalAgentConfigModel{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)
	return result.Error
}

// DeleteAgent 软删除外部智能体
func (r *ExternalAgentRepository) DeleteAgent(ctx context.Context, id int64) error {
	query := `UPDATE external_agent_config SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL`
	result := r.db.WithContext(ctx).Exec(query, id)
	return result.Error
}