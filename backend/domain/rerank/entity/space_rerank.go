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

package entity

import (
	"encoding/json"
	"strconv"
)

// RerankType represents the type of rerank provider
type RerankType string

const (
	RerankTypeOpenAI RerankType = "openai"
)

// SpaceRerank represents the space rerank configuration entity
type SpaceRerank struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	SpaceID    uint64     `gorm:"not null;comment:空间ID" json:"space_id"`
	UserID     uint64     `gorm:"not null;comment:创建者ID" json:"user_id"`
	Name       string     `gorm:"size:128;not null;comment:配置名称" json:"name"`
	Description string    `gorm:"size:512;default:'';comment:配置描述" json:"description"`
	RerankType RerankType `gorm:"size:32;not null;comment:Rerank类型" json:"rerank_type"`
	Config     string     `gorm:"type:json;not null;comment:Rerank配置" json:"config"`
	Status     int        `gorm:"not null;default:1;comment:状态: 1启用 2禁用" json:"status"`
	IsDefault  int        `gorm:"not null;default:0;comment:是否为默认配置" json:"is_default"`
	CreatedAt  uint64     `gorm:"not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt  uint64     `gorm:"not null;default:0;comment:更新时间" json:"updated_at"`
	DeletedAt  *uint64    `gorm:"comment:删除时间" json:"deleted_at"`
}

func (SpaceRerank) TableName() string {
	return "space_rerank"
}

// Status constants
const (
	SpaceRerankStatusEnabled  = 1
	SpaceRerankStatusDisabled = 2
)

// OpenAIRerankConfig represents OpenAI-compatible rerank configuration
type OpenAIRerankConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

// RerankConfig represents the unified rerank configuration
type RerankConfig struct {
	Type   RerankType          `json:"type"`
	OpenAI *OpenAIRerankConfig `json:"openai_config,omitempty"`
}

// GetConfigStruct parses the JSON config string into RerankConfig
func (s *SpaceRerank) GetConfigStruct() (*RerankConfig, error) {
	var config RerankConfig
	if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
		return nil, err
	}
	config.Type = s.RerankType
	return &config, nil
}

// SetConfigFromStruct converts RerankConfig to JSON string and sets it
func (s *SpaceRerank) SetConfigFromStruct(config *RerankConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s.Config = string(data)
	s.RerankType = config.Type
	return nil
}

// SpaceRerankView is a view model for API responses
type SpaceRerankView struct {
	ID         string        `json:"id"`
	SpaceID    string        `json:"space_id"`
	Name       string        `json:"name"`
	Description string       `json:"description"`
	RerankType RerankType    `json:"rerank_type"`
	Config     *RerankConfig `json:"config"`
	Status     int           `json:"status"`
	IsDefault  bool          `json:"is_default"`
	CreatedAt  uint64        `json:"created_at"`
	UpdatedAt  uint64        `json:"updated_at"`
}

// ToView converts SpaceRerank to SpaceRerankView
func (s *SpaceRerank) ToView() *SpaceRerankView {
	config, _ := s.GetConfigStruct()

	// Mask sensitive information
	if config != nil {
		if config.OpenAI != nil && config.OpenAI.APIKey != "" {
			config.OpenAI.APIKey = maskAPIKey(config.OpenAI.APIKey)
		}
	}

	return &SpaceRerankView{
		ID:          formatID(s.ID),
		SpaceID:     formatID(s.SpaceID),
		Name:        s.Name,
		Description: s.Description,
		RerankType:  s.RerankType,
		Config:      config,
		Status:      s.Status,
		IsDefault:   s.IsDefault == 1,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

// maskAPIKey masks the API key for security
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// formatID converts uint64 to string
func formatID(id uint64) string {
	return strconv.FormatUint(id, 10)
}
