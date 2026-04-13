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

package repository

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/domain/rerank/entity"
)

// SpaceRerankRepository defines the interface for space rerank data access
type SpaceRerankRepository interface {
	// Create creates a new space rerank configuration
	Create(ctx context.Context, rerank *entity.SpaceRerank) error

	// GetByID retrieves a space rerank by ID
	GetByID(ctx context.Context, id uint64) (*entity.SpaceRerank, error)

	// GetBySpaceID retrieves all reranks for a space
	GetBySpaceID(ctx context.Context, spaceID uint64) ([]*entity.SpaceRerank, error)

	// GetDefaultBySpaceID retrieves the default rerank for a space
	GetDefaultBySpaceID(ctx context.Context, spaceID uint64) (*entity.SpaceRerank, error)

	// GetBySpaceIDAndName retrieves a rerank by space ID and name
	GetBySpaceIDAndName(ctx context.Context, spaceID uint64, name string) (*entity.SpaceRerank, error)

	// Update updates a space rerank configuration
	Update(ctx context.Context, rerank *entity.SpaceRerank) error

	// UpdateStatus updates the status of a space rerank
	UpdateStatus(ctx context.Context, id uint64, status int) error

	// SetDefault sets a rerank as the default for a space
	SetDefault(ctx context.Context, spaceID, rerankID uint64) error

	// ClearDefault clears the default flag for all reranks in a space
	ClearDefault(ctx context.Context, spaceID uint64) error

	// Delete soft-deletes a space rerank
	Delete(ctx context.Context, id uint64) error
}
