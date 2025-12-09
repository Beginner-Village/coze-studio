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

package embedding

import (
	"context"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/domain/embedding/entity"
	"github.com/coze-dev/coze-studio/backend/domain/embedding/service"
	infraRepo "github.com/coze-dev/coze-studio/backend/infra/impl/embedding/repository"
)

// SpaceEmbeddingApp is the application layer for space embedding
type SpaceEmbeddingApp struct {
	service service.SpaceEmbeddingService
}

// NewSpaceEmbeddingApp creates a new SpaceEmbeddingApp
func NewSpaceEmbeddingApp(db *gorm.DB) *SpaceEmbeddingApp {
	repo := infraRepo.NewSpaceEmbeddingRepository(db)
	svc := service.NewSpaceEmbeddingService(repo)
	return &SpaceEmbeddingApp{
		service: svc,
	}
}

// CreateSpaceEmbedding creates a new embedding configuration for a space
func (app *SpaceEmbeddingApp) CreateSpaceEmbedding(
	ctx context.Context,
	spaceIDStr string,
	userID uint64,
	name, description string,
	embeddingType string,
	config map[string]interface{},
	maxBatchSize int,
	setAsDefault bool,
) (*entity.SpaceEmbeddingView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	embConfig, err := buildEmbeddingConfig(entity.EmbeddingType(embeddingType), config, maxBatchSize)
	if err != nil {
		return nil, err
	}

	embedding, err := app.service.CreateSpaceEmbedding(ctx, spaceID, userID, name, description, embConfig, setAsDefault)
	if err != nil {
		return nil, err
	}

	return embedding.ToView(), nil
}

// ListSpaceEmbeddings lists all embeddings for a space
func (app *SpaceEmbeddingApp) ListSpaceEmbeddings(ctx context.Context, spaceIDStr string) ([]*entity.SpaceEmbeddingView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	return app.service.ListSpaceEmbeddings(ctx, spaceID)
}

// GetSpaceDefaultEmbedding gets the default embedding for a space
func (app *SpaceEmbeddingApp) GetSpaceDefaultEmbedding(ctx context.Context, spaceIDStr string) (*entity.SpaceEmbeddingView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	return app.service.GetSpaceDefaultEmbedding(ctx, spaceID)
}

// UpdateSpaceEmbedding updates an embedding configuration
func (app *SpaceEmbeddingApp) UpdateSpaceEmbedding(
	ctx context.Context,
	spaceIDStr, embeddingIDStr string,
	name, description *string,
	embeddingType *string,
	config map[string]interface{},
	maxBatchSize *int,
) (*entity.SpaceEmbeddingView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	embeddingID, err := strconv.ParseUint(embeddingIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid embedding_id: %w", err)
	}

	var embConfig *entity.EmbeddingConfig
	if config != nil && embeddingType != nil {
		batchSize := 100
		if maxBatchSize != nil {
			batchSize = *maxBatchSize
		}
		embConfig, err = buildEmbeddingConfig(entity.EmbeddingType(*embeddingType), config, batchSize)
		if err != nil {
			return nil, err
		}
	}

	embedding, err := app.service.UpdateSpaceEmbedding(ctx, spaceID, embeddingID, name, description, embConfig)
	if err != nil {
		return nil, err
	}

	return embedding.ToView(), nil
}

// DeleteSpaceEmbedding deletes an embedding configuration
func (app *SpaceEmbeddingApp) DeleteSpaceEmbedding(ctx context.Context, spaceIDStr, embeddingIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	embeddingID, err := strconv.ParseUint(embeddingIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid embedding_id: %w", err)
	}

	return app.service.DeleteSpaceEmbedding(ctx, spaceID, embeddingID)
}

// SetDefaultSpaceEmbedding sets an embedding as the default for a space
func (app *SpaceEmbeddingApp) SetDefaultSpaceEmbedding(ctx context.Context, spaceIDStr, embeddingIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	embeddingID, err := strconv.ParseUint(embeddingIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid embedding_id: %w", err)
	}

	return app.service.SetDefaultSpaceEmbedding(ctx, spaceID, embeddingID)
}

// EnableSpaceEmbedding enables an embedding configuration
func (app *SpaceEmbeddingApp) EnableSpaceEmbedding(ctx context.Context, spaceIDStr, embeddingIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	embeddingID, err := strconv.ParseUint(embeddingIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid embedding_id: %w", err)
	}

	return app.service.EnableSpaceEmbedding(ctx, spaceID, embeddingID)
}

// DisableSpaceEmbedding disables an embedding configuration
func (app *SpaceEmbeddingApp) DisableSpaceEmbedding(ctx context.Context, spaceIDStr, embeddingIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	embeddingID, err := strconv.ParseUint(embeddingIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid embedding_id: %w", err)
	}

	return app.service.DisableSpaceEmbedding(ctx, spaceID, embeddingID)
}

// GetDefaultEmbeddingEntity gets the raw entity for a space's default embedding
func (app *SpaceEmbeddingApp) GetDefaultEmbeddingEntity(ctx context.Context, spaceIDStr string) (*entity.SpaceEmbedding, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	return app.service.GetDefaultSpaceEmbeddingEntity(ctx, spaceID)
}

// buildEmbeddingConfig converts the config map to EmbeddingConfig struct
func buildEmbeddingConfig(embType entity.EmbeddingType, config map[string]interface{}, maxBatchSize int) (*entity.EmbeddingConfig, error) {
	embConfig := &entity.EmbeddingConfig{
		Type:         embType,
		MaxBatchSize: maxBatchSize,
	}

	getString := func(key string) string {
		if v, ok := config[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	getInt := func(key string) int {
		if v, ok := config[key]; ok {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			}
		}
		return 0
	}

	getBool := func(key string) bool {
		if v, ok := config[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
		return false
	}

	switch embType {
	case entity.EmbeddingTypeOpenAI:
		embConfig.OpenAI = &entity.OpenAIEmbeddingConfig{
			BaseURL:     getString("base_url"),
			APIKey:      getString("api_key"),
			Model:       getString("model"),
			ByAzure:     getBool("by_azure"),
			APIVersion:  getString("api_version"),
			Dims:        getInt("dims"),
			RequestDims: getInt("request_dims"),
		}
		if embConfig.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("api_key is required for OpenAI embedding")
		}
		if embConfig.OpenAI.Model == "" {
			return nil, fmt.Errorf("model is required for OpenAI embedding")
		}
		if embConfig.OpenAI.Dims <= 0 {
			return nil, fmt.Errorf("dims must be positive for OpenAI embedding")
		}

	case entity.EmbeddingTypeArk:
		embConfig.Ark = &entity.ArkEmbeddingConfig{
			BaseURL: getString("base_url"),
			APIKey:  getString("api_key"),
			Model:   getString("model"),
			Dims:    getInt("dims"),
			APIType: getString("api_type"),
		}
		if embConfig.Ark.APIKey == "" {
			return nil, fmt.Errorf("api_key is required for ARK embedding")
		}
		if embConfig.Ark.Model == "" {
			return nil, fmt.Errorf("model is required for ARK embedding")
		}
		if embConfig.Ark.Dims <= 0 {
			return nil, fmt.Errorf("dims must be positive for ARK embedding")
		}

	case entity.EmbeddingTypeOllama:
		embConfig.Ollama = &entity.OllamaEmbeddingConfig{
			BaseURL: getString("base_url"),
			Model:   getString("model"),
			Dims:    getInt("dims"),
		}
		if embConfig.Ollama.BaseURL == "" {
			return nil, fmt.Errorf("base_url is required for Ollama embedding")
		}
		if embConfig.Ollama.Model == "" {
			return nil, fmt.Errorf("model is required for Ollama embedding")
		}
		if embConfig.Ollama.Dims <= 0 {
			return nil, fmt.Errorf("dims must be positive for Ollama embedding")
		}

	case entity.EmbeddingTypeHTTP:
		embConfig.HTTP = &entity.HTTPEmbeddingConfig{
			Addr: getString("addr"),
			Dims: getInt("dims"),
		}
		if embConfig.HTTP.Addr == "" {
			return nil, fmt.Errorf("addr is required for HTTP embedding")
		}
		if embConfig.HTTP.Dims <= 0 {
			return nil, fmt.Errorf("dims must be positive for HTTP embedding")
		}

	default:
		return nil, fmt.Errorf("unsupported embedding type: %s", embType)
	}

	return embConfig, nil
}
