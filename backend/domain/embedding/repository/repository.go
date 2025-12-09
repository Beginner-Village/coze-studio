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

package repository

import (
	"context"

	"github.com/coze-dev/coze-studio/backend/domain/embedding/entity"
)

// SpaceEmbeddingRepository defines the interface for space embedding data access
type SpaceEmbeddingRepository interface {
	// Create creates a new space embedding configuration
	Create(ctx context.Context, embedding *entity.SpaceEmbedding) error

	// GetByID retrieves a space embedding by ID
	GetByID(ctx context.Context, id uint64) (*entity.SpaceEmbedding, error)

	// GetBySpaceID retrieves all embeddings for a space
	GetBySpaceID(ctx context.Context, spaceID uint64) ([]*entity.SpaceEmbedding, error)

	// GetDefaultBySpaceID retrieves the default embedding for a space
	GetDefaultBySpaceID(ctx context.Context, spaceID uint64) (*entity.SpaceEmbedding, error)

	// GetBySpaceIDAndName retrieves an embedding by space ID and name
	GetBySpaceIDAndName(ctx context.Context, spaceID uint64, name string) (*entity.SpaceEmbedding, error)

	// Update updates a space embedding configuration
	Update(ctx context.Context, embedding *entity.SpaceEmbedding) error

	// UpdateStatus updates the status of a space embedding
	UpdateStatus(ctx context.Context, id uint64, status int) error

	// SetDefault sets an embedding as the default for a space
	SetDefault(ctx context.Context, spaceID, embeddingID uint64) error

	// ClearDefault clears the default flag for all embeddings in a space
	ClearDefault(ctx context.Context, spaceID uint64) error

	// Delete soft-deletes a space embedding
	Delete(ctx context.Context, id uint64) error
}
