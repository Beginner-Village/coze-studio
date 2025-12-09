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
)

// Provider provides embedding instances based on space configuration
// It supports both space-level embedding configuration and global fallback
type Provider interface {
	// GetEmbedding returns an Embedder for the given space
	// If the space has no embedding configuration, it returns the global default
	// If no global default is configured, it returns an error
	GetEmbedding(ctx context.Context, spaceID uint64) (Embedder, error)

	// GetGlobalEmbedding returns the global default Embedder
	// This is used as fallback when space has no embedding configuration
	GetGlobalEmbedding(ctx context.Context) (Embedder, error)

	// HasSpaceEmbedding checks if a space has its own embedding configuration
	HasSpaceEmbedding(ctx context.Context, spaceID uint64) (bool, error)
}
