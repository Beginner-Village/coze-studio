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

package entity

import (
	"encoding/json"
	"strconv"
)

// EmbeddingType represents the type of embedding provider
type EmbeddingType string

const (
	EmbeddingTypeOpenAI EmbeddingType = "openai"
	EmbeddingTypeArk    EmbeddingType = "ark"
	EmbeddingTypeOllama EmbeddingType = "ollama"
	EmbeddingTypeHTTP   EmbeddingType = "http"
)

// SpaceEmbedding represents the space embedding configuration entity
type SpaceEmbedding struct {
	ID            uint64        `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	SpaceID       uint64        `gorm:"not null;comment:空间ID" json:"space_id"`
	UserID        uint64        `gorm:"not null;comment:创建者ID" json:"user_id"`
	Name          string        `gorm:"size:128;not null;comment:配置名称" json:"name"`
	Description   string        `gorm:"size:512;default:'';comment:配置描述" json:"description"`
	EmbeddingType EmbeddingType `gorm:"size:32;not null;comment:Embedding类型" json:"embedding_type"`
	Config        string        `gorm:"type:json;not null;comment:Embedding配置" json:"config"`
	MaxBatchSize  int           `gorm:"not null;default:100;comment:最大批处理大小" json:"max_batch_size"`
	Status        int           `gorm:"not null;default:1;comment:状态: 1启用 2禁用" json:"status"`
	IsDefault     int           `gorm:"not null;default:0;comment:是否为默认配置" json:"is_default"`
	CreatedAt     uint64        `gorm:"not null;default:0;comment:创建时间" json:"created_at"`
	UpdatedAt     uint64        `gorm:"not null;default:0;comment:更新时间" json:"updated_at"`
	DeletedAt     *uint64       `gorm:"comment:删除时间" json:"deleted_at"`
}

func (SpaceEmbedding) TableName() string {
	return "space_embedding"
}

// Status constants
const (
	SpaceEmbeddingStatusEnabled  = 1
	SpaceEmbeddingStatusDisabled = 2
)

// OpenAIEmbeddingConfig represents OpenAI embedding configuration
type OpenAIEmbeddingConfig struct {
	BaseURL     string `json:"base_url"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	ByAzure     bool   `json:"by_azure,omitempty"`
	APIVersion  string `json:"api_version,omitempty"`
	Dims        int    `json:"dims"`
	RequestDims int    `json:"request_dims,omitempty"`
}

// ArkEmbeddingConfig represents ARK embedding configuration
type ArkEmbeddingConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	Dims    int    `json:"dims"`
	APIType string `json:"api_type,omitempty"` // text or multimodal
}

// OllamaEmbeddingConfig represents Ollama embedding configuration
type OllamaEmbeddingConfig struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	Dims    int    `json:"dims"`
}

// HTTPEmbeddingConfig represents HTTP embedding configuration
type HTTPEmbeddingConfig struct {
	Addr string `json:"addr"`
	Dims int    `json:"dims"`
}

// EmbeddingConfig represents the unified embedding configuration
type EmbeddingConfig struct {
	Type         EmbeddingType          `json:"type"`
	MaxBatchSize int                    `json:"max_batch_size,omitempty"`
	OpenAI       *OpenAIEmbeddingConfig `json:"openai_config,omitempty"`
	Ark          *ArkEmbeddingConfig    `json:"ark_config,omitempty"`
	Ollama       *OllamaEmbeddingConfig `json:"ollama_config,omitempty"`
	HTTP         *HTTPEmbeddingConfig   `json:"http_config,omitempty"`
}

// GetConfigStruct parses the JSON config string into EmbeddingConfig
func (s *SpaceEmbedding) GetConfigStruct() (*EmbeddingConfig, error) {
	var config EmbeddingConfig
	if err := json.Unmarshal([]byte(s.Config), &config); err != nil {
		return nil, err
	}
	config.Type = s.EmbeddingType
	config.MaxBatchSize = s.MaxBatchSize
	return &config, nil
}

// SetConfigFromStruct converts EmbeddingConfig to JSON string and sets it
func (s *SpaceEmbedding) SetConfigFromStruct(config *EmbeddingConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	s.Config = string(data)
	s.EmbeddingType = config.Type
	if config.MaxBatchSize > 0 {
		s.MaxBatchSize = config.MaxBatchSize
	}
	return nil
}

// GetDimensions returns the embedding dimensions based on config type
func (s *SpaceEmbedding) GetDimensions() int {
	config, err := s.GetConfigStruct()
	if err != nil {
		return 0
	}

	switch s.EmbeddingType {
	case EmbeddingTypeOpenAI:
		if config.OpenAI != nil {
			return config.OpenAI.Dims
		}
	case EmbeddingTypeArk:
		if config.Ark != nil {
			return config.Ark.Dims
		}
	case EmbeddingTypeOllama:
		if config.Ollama != nil {
			return config.Ollama.Dims
		}
	case EmbeddingTypeHTTP:
		if config.HTTP != nil {
			return config.HTTP.Dims
		}
	}
	return 0
}

// SpaceEmbeddingView is a view model for API responses
type SpaceEmbeddingView struct {
	ID            string           `json:"id"`
	SpaceID       string           `json:"space_id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	EmbeddingType EmbeddingType    `json:"embedding_type"`
	Config        *EmbeddingConfig `json:"config"`
	Status        int              `json:"status"`
	IsDefault     bool             `json:"is_default"`
	Dimensions    int              `json:"dimensions"`
	CreatedAt     uint64           `json:"created_at"`
	UpdatedAt     uint64           `json:"updated_at"`
}

// ToView converts SpaceEmbedding to SpaceEmbeddingView
func (s *SpaceEmbedding) ToView() *SpaceEmbeddingView {
	config, _ := s.GetConfigStruct()

	// Mask sensitive information
	if config != nil {
		if config.OpenAI != nil && config.OpenAI.APIKey != "" {
			config.OpenAI.APIKey = maskAPIKey(config.OpenAI.APIKey)
		}
		if config.Ark != nil && config.Ark.APIKey != "" {
			config.Ark.APIKey = maskAPIKey(config.Ark.APIKey)
		}
	}

	return &SpaceEmbeddingView{
		ID:            formatID(s.ID),
		SpaceID:       formatID(s.SpaceID),
		Name:          s.Name,
		Description:   s.Description,
		EmbeddingType: s.EmbeddingType,
		Config:        config,
		Status:        s.Status,
		IsDefault:     s.IsDefault == 1,
		Dimensions:    s.GetDimensions(),
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
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
