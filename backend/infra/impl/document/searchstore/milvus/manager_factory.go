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

package milvus

import (
	"context"
	"fmt"
	"sync"

	client "github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// ManagerFactoryConfig contains configuration for creating ManagerFactory
type ManagerFactoryConfig struct {
	Client            *client.Client      // required: Milvus client
	EmbeddingProvider embedding.Provider  // required: Embedding provider for space-level embeddings
	GlobalEmbedding   embedding.Embedder  // optional: Global default embedding (fallback)
	EnableHybrid      *bool               // optional: Enable hybrid search
}

// milvusManagerFactory creates milvus managers based on space configuration
type milvusManagerFactory struct {
	config *ManagerFactoryConfig

	// Cache for space managers to avoid repeated creation
	cache   map[uint64][]searchstore.Manager
	cacheMu sync.RWMutex

	// Global managers cache
	globalManagers []searchstore.Manager
	globalOnce     sync.Once
}

// NewManagerFactory creates a new milvus ManagerFactory
func NewManagerFactory(config *ManagerFactoryConfig) (searchstore.ManagerFactory, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("[NewManagerFactory] milvus client not provided")
	}
	if config.EmbeddingProvider == nil {
		return nil, fmt.Errorf("[NewManagerFactory] embedding provider not provided")
	}

	return &milvusManagerFactory{
		config: config,
		cache:  make(map[uint64][]searchstore.Manager),
	}, nil
}

// GetManagers returns SearchStore managers for the given space
func (f *milvusManagerFactory) GetManagers(ctx context.Context, spaceID uint64) ([]searchstore.Manager, error) {
	// Check cache first
	f.cacheMu.RLock()
	if managers, ok := f.cache[spaceID]; ok {
		f.cacheMu.RUnlock()
		return managers, nil
	}
	f.cacheMu.RUnlock()

	// Get embedding for this space
	emb, err := f.config.EmbeddingProvider.GetEmbedding(ctx, spaceID)
	if err != nil {
		logs.CtxWarnf(ctx, "[MilvusManagerFactory] failed to get embedding for space %d: %v, using global", spaceID, err)
		return f.GetGlobalManagers(ctx)
	}

	// Check if space has its own embedding
	hasSpaceEmb, err := f.config.EmbeddingProvider.HasSpaceEmbedding(ctx, spaceID)
	if err != nil {
		logs.CtxWarnf(ctx, "[MilvusManagerFactory] failed to check space embedding: %v", err)
	}
	if !hasSpaceEmb {
		// No space-specific embedding, use global
		return f.GetGlobalManagers(ctx)
	}

	// Create manager with space-specific embedding
	manager, err := f.createManager(ctx, emb)
	if err != nil {
		logs.CtxErrorf(ctx, "[MilvusManagerFactory] failed to create manager for space %d: %v", spaceID, err)
		return nil, err
	}

	managers := []searchstore.Manager{manager}

	// Cache the managers
	f.cacheMu.Lock()
	f.cache[spaceID] = managers
	f.cacheMu.Unlock()

	return managers, nil
}

// GetGlobalManagers returns the global default SearchStore managers
func (f *milvusManagerFactory) GetGlobalManagers(ctx context.Context) ([]searchstore.Manager, error) {
	var initErr error
	f.globalOnce.Do(func() {
		// Try to get global embedding from provider first
		emb, err := f.config.EmbeddingProvider.GetGlobalEmbedding(ctx)
		if err != nil {
			// Fall back to config's global embedding
			if f.config.GlobalEmbedding != nil {
				emb = f.config.GlobalEmbedding
			} else {
				initErr = fmt.Errorf("no global embedding configured: %w", err)
				return
			}
		}

		manager, err := f.createManager(ctx, emb)
		if err != nil {
			initErr = fmt.Errorf("failed to create global manager: %w", err)
			return
		}

		f.globalManagers = []searchstore.Manager{manager}
	})

	if initErr != nil {
		return nil, initErr
	}

	return f.globalManagers, nil
}

// HasSpaceManagers checks if a space has its own manager configuration
func (f *milvusManagerFactory) HasSpaceManagers(ctx context.Context, spaceID uint64) (bool, error) {
	return f.config.EmbeddingProvider.HasSpaceEmbedding(ctx, spaceID)
}

// createManager creates a new milvus manager with the given embedding
func (f *milvusManagerFactory) createManager(ctx context.Context, emb embedding.Embedder) (searchstore.Manager, error) {
	enableHybrid := f.config.EnableHybrid
	if enableHybrid == nil {
		enableHybrid = ptr.Of(emb.SupportStatus() == embedding.SupportDenseAndSparse)
	}

	return NewManager(&ManagerConfig{
		Client:       f.config.Client,
		Embedding:    emb,
		EnableHybrid: enableHybrid,
	})
}

// InvalidateCache removes the cached managers for a space
// Call this when space embedding configuration changes
func (f *milvusManagerFactory) InvalidateCache(spaceID uint64) {
	f.cacheMu.Lock()
	delete(f.cache, spaceID)
	f.cacheMu.Unlock()
}

// InvalidateAllCache clears all cached managers
func (f *milvusManagerFactory) InvalidateAllCache() {
	f.cacheMu.Lock()
	f.cache = make(map[uint64][]searchstore.Manager)
	f.cacheMu.Unlock()
}
