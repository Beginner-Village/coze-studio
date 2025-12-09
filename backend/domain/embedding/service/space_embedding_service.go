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
	"fmt"
	"time"

	"github.com/coze-dev/coze-studio/backend/domain/embedding/entity"
	"github.com/coze-dev/coze-studio/backend/domain/embedding/repository"
)

// SpaceEmbeddingService defines the interface for space embedding business logic
type SpaceEmbeddingService interface {
	// CreateSpaceEmbedding creates a new embedding configuration for a space
	CreateSpaceEmbedding(ctx context.Context, spaceID, userID uint64, name, description string, config *entity.EmbeddingConfig, setAsDefault bool) (*entity.SpaceEmbedding, error)

	// ListSpaceEmbeddings lists all embeddings for a space
	ListSpaceEmbeddings(ctx context.Context, spaceID uint64) ([]*entity.SpaceEmbeddingView, error)

	// GetSpaceDefaultEmbedding gets the default embedding for a space
	GetSpaceDefaultEmbedding(ctx context.Context, spaceID uint64) (*entity.SpaceEmbeddingView, error)

	// GetSpaceEmbedding gets a specific embedding configuration
	GetSpaceEmbedding(ctx context.Context, embeddingID uint64) (*entity.SpaceEmbeddingView, error)

	// UpdateSpaceEmbedding updates an embedding configuration
	UpdateSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64, name, description *string, config *entity.EmbeddingConfig) (*entity.SpaceEmbedding, error)

	// DeleteSpaceEmbedding deletes an embedding configuration
	DeleteSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error

	// SetDefaultSpaceEmbedding sets an embedding as the default for a space
	SetDefaultSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error

	// EnableSpaceEmbedding enables an embedding configuration
	EnableSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error

	// DisableSpaceEmbedding disables an embedding configuration
	DisableSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error

	// GetSpaceEmbeddingEntity gets the raw entity for creating embedder
	GetSpaceEmbeddingEntity(ctx context.Context, embeddingID uint64) (*entity.SpaceEmbedding, error)

	// GetDefaultSpaceEmbeddingEntity gets the default embedding entity for a space
	GetDefaultSpaceEmbeddingEntity(ctx context.Context, spaceID uint64) (*entity.SpaceEmbedding, error)
}

type spaceEmbeddingService struct {
	repo repository.SpaceEmbeddingRepository
}

// NewSpaceEmbeddingService creates a new SpaceEmbeddingService
func NewSpaceEmbeddingService(repo repository.SpaceEmbeddingRepository) SpaceEmbeddingService {
	return &spaceEmbeddingService{
		repo: repo,
	}
}

func (s *spaceEmbeddingService) CreateSpaceEmbedding(ctx context.Context, spaceID, userID uint64, name, description string, config *entity.EmbeddingConfig, setAsDefault bool) (*entity.SpaceEmbedding, error) {
	// Check if name already exists in this space
	existing, err := s.repo.GetBySpaceIDAndName(ctx, spaceID, name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("embedding configuration with name '%s' already exists in this space", name)
	}

	now := uint64(time.Now().UnixMilli())
	embedding := &entity.SpaceEmbedding{
		SpaceID:      spaceID,
		UserID:       userID,
		Name:         name,
		Description:  description,
		MaxBatchSize: 100, // default
		Status:       entity.SpaceEmbeddingStatusEnabled,
		IsDefault:    0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if config.MaxBatchSize > 0 {
		embedding.MaxBatchSize = config.MaxBatchSize
	}

	if err := embedding.SetConfigFromStruct(config); err != nil {
		return nil, fmt.Errorf("failed to set config: %w", err)
	}

	// If this should be the default, clear existing defaults first
	if setAsDefault {
		if err := s.repo.ClearDefault(ctx, spaceID); err != nil {
			return nil, fmt.Errorf("failed to clear existing default: %w", err)
		}
		embedding.IsDefault = 1
	} else {
		// Check if there's any existing embedding, if not, make this the default
		existingEmbeddings, err := s.repo.GetBySpaceID(ctx, spaceID)
		if err == nil && len(existingEmbeddings) == 0 {
			embedding.IsDefault = 1
		}
	}

	if err := s.repo.Create(ctx, embedding); err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	return embedding, nil
}

func (s *spaceEmbeddingService) ListSpaceEmbeddings(ctx context.Context, spaceID uint64) ([]*entity.SpaceEmbeddingView, error) {
	embeddings, err := s.repo.GetBySpaceID(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	views := make([]*entity.SpaceEmbeddingView, 0, len(embeddings))
	for _, e := range embeddings {
		views = append(views, e.ToView())
	}
	return views, nil
}

func (s *spaceEmbeddingService) GetSpaceDefaultEmbedding(ctx context.Context, spaceID uint64) (*entity.SpaceEmbeddingView, error) {
	embedding, err := s.repo.GetDefaultBySpaceID(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	if embedding == nil {
		return nil, nil
	}
	return embedding.ToView(), nil
}

func (s *spaceEmbeddingService) GetSpaceEmbedding(ctx context.Context, embeddingID uint64) (*entity.SpaceEmbeddingView, error) {
	embedding, err := s.repo.GetByID(ctx, embeddingID)
	if err != nil {
		return nil, err
	}
	return embedding.ToView(), nil
}

func (s *spaceEmbeddingService) UpdateSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64, name, description *string, config *entity.EmbeddingConfig) (*entity.SpaceEmbedding, error) {
	embedding, err := s.repo.GetByID(ctx, embeddingID)
	if err != nil {
		return nil, fmt.Errorf("embedding not found: %w", err)
	}

	if embedding.SpaceID != spaceID {
		return nil, fmt.Errorf("embedding does not belong to this space")
	}

	if name != nil && *name != embedding.Name {
		// Check if new name already exists
		existing, err := s.repo.GetBySpaceIDAndName(ctx, spaceID, *name)
		if err == nil && existing != nil && existing.ID != embeddingID {
			return nil, fmt.Errorf("embedding configuration with name '%s' already exists", *name)
		}
		embedding.Name = *name
	}

	if description != nil {
		embedding.Description = *description
	}

	if config != nil {
		if err := embedding.SetConfigFromStruct(config); err != nil {
			return nil, fmt.Errorf("failed to set config: %w", err)
		}
	}

	embedding.UpdatedAt = uint64(time.Now().UnixMilli())

	if err := s.repo.Update(ctx, embedding); err != nil {
		return nil, fmt.Errorf("failed to update embedding: %w", err)
	}

	return embedding, nil
}

func (s *spaceEmbeddingService) DeleteSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error {
	embedding, err := s.repo.GetByID(ctx, embeddingID)
	if err != nil {
		return fmt.Errorf("embedding not found: %w", err)
	}

	if embedding.SpaceID != spaceID {
		return fmt.Errorf("embedding does not belong to this space")
	}

	if embedding.IsDefault == 1 {
		// If deleting the default, we need to set another one as default
		embeddings, err := s.repo.GetBySpaceID(ctx, spaceID)
		if err != nil {
			return err
		}
		for _, e := range embeddings {
			if e.ID != embeddingID && e.Status == entity.SpaceEmbeddingStatusEnabled {
				if err := s.repo.SetDefault(ctx, spaceID, e.ID); err != nil {
					return fmt.Errorf("failed to set new default: %w", err)
				}
				break
			}
		}
	}

	return s.repo.Delete(ctx, embeddingID)
}

func (s *spaceEmbeddingService) SetDefaultSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error {
	embedding, err := s.repo.GetByID(ctx, embeddingID)
	if err != nil {
		return fmt.Errorf("embedding not found: %w", err)
	}

	if embedding.SpaceID != spaceID {
		return fmt.Errorf("embedding does not belong to this space")
	}

	if embedding.Status != entity.SpaceEmbeddingStatusEnabled {
		return fmt.Errorf("cannot set disabled embedding as default")
	}

	return s.repo.SetDefault(ctx, spaceID, embeddingID)
}

func (s *spaceEmbeddingService) EnableSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error {
	embedding, err := s.repo.GetByID(ctx, embeddingID)
	if err != nil {
		return fmt.Errorf("embedding not found: %w", err)
	}

	if embedding.SpaceID != spaceID {
		return fmt.Errorf("embedding does not belong to this space")
	}

	return s.repo.UpdateStatus(ctx, embeddingID, entity.SpaceEmbeddingStatusEnabled)
}

func (s *spaceEmbeddingService) DisableSpaceEmbedding(ctx context.Context, spaceID, embeddingID uint64) error {
	embedding, err := s.repo.GetByID(ctx, embeddingID)
	if err != nil {
		return fmt.Errorf("embedding not found: %w", err)
	}

	if embedding.SpaceID != spaceID {
		return fmt.Errorf("embedding does not belong to this space")
	}

	if embedding.IsDefault == 1 {
		return fmt.Errorf("cannot disable the default embedding, please set another embedding as default first")
	}

	return s.repo.UpdateStatus(ctx, embeddingID, entity.SpaceEmbeddingStatusDisabled)
}

func (s *spaceEmbeddingService) GetSpaceEmbeddingEntity(ctx context.Context, embeddingID uint64) (*entity.SpaceEmbedding, error) {
	return s.repo.GetByID(ctx, embeddingID)
}

func (s *spaceEmbeddingService) GetDefaultSpaceEmbeddingEntity(ctx context.Context, spaceID uint64) (*entity.SpaceEmbedding, error) {
	return s.repo.GetDefaultBySpaceID(ctx, spaceID)
}
