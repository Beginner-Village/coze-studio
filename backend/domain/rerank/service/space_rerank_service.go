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

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/repository"
)

// SpaceRerankService defines the interface for space rerank business logic
type SpaceRerankService interface {
	// CreateSpaceRerank creates a new rerank configuration for a space
	CreateSpaceRerank(ctx context.Context, spaceID, userID uint64, name, description string, config *entity.RerankConfig, setAsDefault bool) (*entity.SpaceRerank, error)

	// ListSpaceReranks lists all reranks for a space
	ListSpaceReranks(ctx context.Context, spaceID uint64) ([]*entity.SpaceRerankView, error)

	// GetSpaceDefaultRerank gets the default rerank for a space
	GetSpaceDefaultRerank(ctx context.Context, spaceID uint64) (*entity.SpaceRerankView, error)

	// GetSpaceRerank gets a specific rerank configuration
	GetSpaceRerank(ctx context.Context, rerankID uint64) (*entity.SpaceRerankView, error)

	// UpdateSpaceRerank updates a rerank configuration
	UpdateSpaceRerank(ctx context.Context, spaceID, rerankID uint64, name, description *string, config *entity.RerankConfig) (*entity.SpaceRerank, error)

	// DeleteSpaceRerank deletes a rerank configuration
	DeleteSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error

	// SetDefaultSpaceRerank sets a rerank as the default for a space
	SetDefaultSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error

	// EnableSpaceRerank enables a rerank configuration
	EnableSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error

	// DisableSpaceRerank disables a rerank configuration
	DisableSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error

	// GetSpaceRerankEntity gets the raw entity for creating reranker
	GetSpaceRerankEntity(ctx context.Context, rerankID uint64) (*entity.SpaceRerank, error)

	// GetDefaultSpaceRerankEntity gets the default rerank entity for a space
	GetDefaultSpaceRerankEntity(ctx context.Context, spaceID uint64) (*entity.SpaceRerank, error)
}

type spaceRerankService struct {
	repo repository.SpaceRerankRepository
}

// NewSpaceRerankService creates a new SpaceRerankService
func NewSpaceRerankService(repo repository.SpaceRerankRepository) SpaceRerankService {
	return &spaceRerankService{
		repo: repo,
	}
}

func (s *spaceRerankService) CreateSpaceRerank(ctx context.Context, spaceID, userID uint64, name, description string, config *entity.RerankConfig, setAsDefault bool) (*entity.SpaceRerank, error) {
	// Check if name already exists in this space
	existing, err := s.repo.GetBySpaceIDAndName(ctx, spaceID, name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("rerank configuration with name '%s' already exists in this space", name)
	}

	now := uint64(time.Now().UnixMilli())
	rerank := &entity.SpaceRerank{
		SpaceID:     spaceID,
		UserID:      userID,
		Name:        name,
		Description: description,
		Status:      entity.SpaceRerankStatusEnabled,
		IsDefault:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := rerank.SetConfigFromStruct(config); err != nil {
		return nil, fmt.Errorf("failed to set config: %w", err)
	}

	// If this should be the default, clear existing defaults first
	if setAsDefault {
		if err := s.repo.ClearDefault(ctx, spaceID); err != nil {
			return nil, fmt.Errorf("failed to clear existing default: %w", err)
		}
		rerank.IsDefault = 1
	} else {
		// Check if there's any existing rerank, if not, make this the default
		existingReranks, err := s.repo.GetBySpaceID(ctx, spaceID)
		if err == nil && len(existingReranks) == 0 {
			rerank.IsDefault = 1
		}
	}

	if err := s.repo.Create(ctx, rerank); err != nil {
		return nil, fmt.Errorf("failed to create rerank: %w", err)
	}

	return rerank, nil
}

func (s *spaceRerankService) ListSpaceReranks(ctx context.Context, spaceID uint64) ([]*entity.SpaceRerankView, error) {
	reranks, err := s.repo.GetBySpaceID(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	views := make([]*entity.SpaceRerankView, 0, len(reranks))
	for _, r := range reranks {
		views = append(views, r.ToView())
	}
	return views, nil
}

func (s *spaceRerankService) GetSpaceDefaultRerank(ctx context.Context, spaceID uint64) (*entity.SpaceRerankView, error) {
	rerank, err := s.repo.GetDefaultBySpaceID(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	if rerank == nil {
		return nil, nil
	}
	return rerank.ToView(), nil
}

func (s *spaceRerankService) GetSpaceRerank(ctx context.Context, rerankID uint64) (*entity.SpaceRerankView, error) {
	rerank, err := s.repo.GetByID(ctx, rerankID)
	if err != nil {
		return nil, err
	}
	return rerank.ToView(), nil
}

func (s *spaceRerankService) UpdateSpaceRerank(ctx context.Context, spaceID, rerankID uint64, name, description *string, config *entity.RerankConfig) (*entity.SpaceRerank, error) {
	rerank, err := s.repo.GetByID(ctx, rerankID)
	if err != nil {
		return nil, fmt.Errorf("rerank not found: %w", err)
	}

	if rerank.SpaceID != spaceID {
		return nil, fmt.Errorf("rerank does not belong to this space")
	}

	if name != nil && *name != rerank.Name {
		// Check if new name already exists
		existing, err := s.repo.GetBySpaceIDAndName(ctx, spaceID, *name)
		if err == nil && existing != nil && existing.ID != rerankID {
			return nil, fmt.Errorf("rerank configuration with name '%s' already exists", *name)
		}
		rerank.Name = *name
	}

	if description != nil {
		rerank.Description = *description
	}

	if config != nil {
		if err := rerank.SetConfigFromStruct(config); err != nil {
			return nil, fmt.Errorf("failed to set config: %w", err)
		}
	}

	rerank.UpdatedAt = uint64(time.Now().UnixMilli())

	if err := s.repo.Update(ctx, rerank); err != nil {
		return nil, fmt.Errorf("failed to update rerank: %w", err)
	}

	return rerank, nil
}

func (s *spaceRerankService) DeleteSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error {
	rerank, err := s.repo.GetByID(ctx, rerankID)
	if err != nil {
		return fmt.Errorf("rerank not found: %w", err)
	}

	if rerank.SpaceID != spaceID {
		return fmt.Errorf("rerank does not belong to this space")
	}

	if rerank.IsDefault == 1 {
		// If deleting the default, we need to set another one as default
		reranks, err := s.repo.GetBySpaceID(ctx, spaceID)
		if err != nil {
			return err
		}
		for _, r := range reranks {
			if r.ID != rerankID && r.Status == entity.SpaceRerankStatusEnabled {
				if err := s.repo.SetDefault(ctx, spaceID, r.ID); err != nil {
					return fmt.Errorf("failed to set new default: %w", err)
				}
				break
			}
		}
	}

	return s.repo.Delete(ctx, rerankID)
}

func (s *spaceRerankService) SetDefaultSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error {
	rerank, err := s.repo.GetByID(ctx, rerankID)
	if err != nil {
		return fmt.Errorf("rerank not found: %w", err)
	}

	if rerank.SpaceID != spaceID {
		return fmt.Errorf("rerank does not belong to this space")
	}

	if rerank.Status != entity.SpaceRerankStatusEnabled {
		return fmt.Errorf("cannot set disabled rerank as default")
	}

	return s.repo.SetDefault(ctx, spaceID, rerankID)
}

func (s *spaceRerankService) EnableSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error {
	rerank, err := s.repo.GetByID(ctx, rerankID)
	if err != nil {
		return fmt.Errorf("rerank not found: %w", err)
	}

	if rerank.SpaceID != spaceID {
		return fmt.Errorf("rerank does not belong to this space")
	}

	return s.repo.UpdateStatus(ctx, rerankID, entity.SpaceRerankStatusEnabled)
}

func (s *spaceRerankService) DisableSpaceRerank(ctx context.Context, spaceID, rerankID uint64) error {
	rerank, err := s.repo.GetByID(ctx, rerankID)
	if err != nil {
		return fmt.Errorf("rerank not found: %w", err)
	}

	if rerank.SpaceID != spaceID {
		return fmt.Errorf("rerank does not belong to this space")
	}

	if rerank.IsDefault == 1 {
		return fmt.Errorf("cannot disable the default rerank, please set another rerank as default first")
	}

	return s.repo.UpdateStatus(ctx, rerankID, entity.SpaceRerankStatusDisabled)
}

func (s *spaceRerankService) GetSpaceRerankEntity(ctx context.Context, rerankID uint64) (*entity.SpaceRerank, error) {
	return s.repo.GetByID(ctx, rerankID)
}

func (s *spaceRerankService) GetDefaultSpaceRerankEntity(ctx context.Context, spaceID uint64) (*entity.SpaceRerank, error) {
	return s.repo.GetDefaultBySpaceID(ctx, spaceID)
}
