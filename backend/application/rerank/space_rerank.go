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

package rerank

import (
	"context"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/service"
	infraRepo "github.com/ynet-dev/ynet-studio/backend/infra/impl/rerank/repository"
)

// SpaceRerankApp is the application layer for space rerank
type SpaceRerankApp struct {
	service service.SpaceRerankService
}

// NewSpaceRerankApp creates a new SpaceRerankApp
func NewSpaceRerankApp(db *gorm.DB) *SpaceRerankApp {
	repo := infraRepo.NewSpaceRerankRepository(db)
	svc := service.NewSpaceRerankService(repo)
	return &SpaceRerankApp{
		service: svc,
	}
}

// GetService returns the underlying service for provider usage
func (app *SpaceRerankApp) GetService() service.SpaceRerankService {
	return app.service
}

// CreateSpaceRerank creates a new rerank configuration for a space
func (app *SpaceRerankApp) CreateSpaceRerank(
	ctx context.Context,
	spaceIDStr string,
	userID uint64,
	name, description string,
	rerankType string,
	config map[string]interface{},
	setAsDefault bool,
) (*entity.SpaceRerankView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	rerankConfig, err := buildRerankConfig(entity.RerankType(rerankType), config)
	if err != nil {
		return nil, err
	}

	rerank, err := app.service.CreateSpaceRerank(ctx, spaceID, userID, name, description, rerankConfig, setAsDefault)
	if err != nil {
		return nil, err
	}

	return rerank.ToView(), nil
}

// ListSpaceReranks lists all reranks for a space
func (app *SpaceRerankApp) ListSpaceReranks(ctx context.Context, spaceIDStr string) ([]*entity.SpaceRerankView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	return app.service.ListSpaceReranks(ctx, spaceID)
}

// GetSpaceDefaultRerank gets the default rerank for a space
func (app *SpaceRerankApp) GetSpaceDefaultRerank(ctx context.Context, spaceIDStr string) (*entity.SpaceRerankView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	return app.service.GetSpaceDefaultRerank(ctx, spaceID)
}

// UpdateSpaceRerank updates a rerank configuration
func (app *SpaceRerankApp) UpdateSpaceRerank(
	ctx context.Context,
	spaceIDStr, rerankIDStr string,
	name, description *string,
	rerankType *string,
	config map[string]interface{},
) (*entity.SpaceRerankView, error) {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	rerankID, err := strconv.ParseUint(rerankIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid rerank_id: %w", err)
	}

	var rerankConfig *entity.RerankConfig
	if config != nil && rerankType != nil {
		rerankConfig, err = buildRerankConfig(entity.RerankType(*rerankType), config)
		if err != nil {
			return nil, err
		}
	}

	rerank, err := app.service.UpdateSpaceRerank(ctx, spaceID, rerankID, name, description, rerankConfig)
	if err != nil {
		return nil, err
	}

	return rerank.ToView(), nil
}

// DeleteSpaceRerank deletes a rerank configuration
func (app *SpaceRerankApp) DeleteSpaceRerank(ctx context.Context, spaceIDStr, rerankIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	rerankID, err := strconv.ParseUint(rerankIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid rerank_id: %w", err)
	}

	return app.service.DeleteSpaceRerank(ctx, spaceID, rerankID)
}

// SetDefaultSpaceRerank sets a rerank as the default for a space
func (app *SpaceRerankApp) SetDefaultSpaceRerank(ctx context.Context, spaceIDStr, rerankIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	rerankID, err := strconv.ParseUint(rerankIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid rerank_id: %w", err)
	}

	return app.service.SetDefaultSpaceRerank(ctx, spaceID, rerankID)
}

// EnableSpaceRerank enables a rerank configuration
func (app *SpaceRerankApp) EnableSpaceRerank(ctx context.Context, spaceIDStr, rerankIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	rerankID, err := strconv.ParseUint(rerankIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid rerank_id: %w", err)
	}

	return app.service.EnableSpaceRerank(ctx, spaceID, rerankID)
}

// DisableSpaceRerank disables a rerank configuration
func (app *SpaceRerankApp) DisableSpaceRerank(ctx context.Context, spaceIDStr, rerankIDStr string) error {
	spaceID, err := strconv.ParseUint(spaceIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid space_id: %w", err)
	}

	rerankID, err := strconv.ParseUint(rerankIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid rerank_id: %w", err)
	}

	return app.service.DisableSpaceRerank(ctx, spaceID, rerankID)
}

// buildRerankConfig converts the config map to RerankConfig struct
func buildRerankConfig(rerankType entity.RerankType, config map[string]interface{}) (*entity.RerankConfig, error) {
	rerankConfig := &entity.RerankConfig{
		Type: rerankType,
	}

	getString := func(key string) string {
		if v, ok := config[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	switch rerankType {
	case entity.RerankTypeOpenAI:
		rerankConfig.OpenAI = &entity.OpenAIRerankConfig{
			BaseURL: getString("base_url"),
			APIKey:  getString("api_key"),
			Model:   getString("model"),
		}
		if rerankConfig.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("api_key is required for OpenAI rerank")
		}
		if rerankConfig.OpenAI.Model == "" {
			return nil, fmt.Errorf("model is required for OpenAI rerank")
		}

	default:
		return nil, fmt.Errorf("unsupported rerank type: %s", rerankType)
	}

	return rerankConfig, nil
}
