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

package oceanbase

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// ManagerFactoryConfig contains configuration for creating ManagerFactory.
type ManagerFactoryConfig struct {
	DB                *gorm.DB           // required: OceanBase connection
	EmbeddingProvider embedding.Provider // required: space-level embedding provider
	GlobalEmbedding   embedding.Embedder // optional: global fallback
	BatchSize         int                // optional: default 100
}

type obManagerFactory struct {
	config *ManagerFactoryConfig

	cache   map[uint64][]searchstore.Manager
	cacheMu sync.RWMutex

	globalManagers []searchstore.Manager
	globalOnce     sync.Once
}

// NewManagerFactory creates a new OceanBase ManagerFactory.
func NewManagerFactory(config *ManagerFactoryConfig) (searchstore.ManagerFactory, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("[NewManagerFactory] OceanBase DB not provided")
	}
	if config.EmbeddingProvider == nil {
		return nil, fmt.Errorf("[NewManagerFactory] embedding provider not provided")
	}

	return &obManagerFactory{
		config: config,
		cache:  make(map[uint64][]searchstore.Manager),
	}, nil
}

func (f *obManagerFactory) GetManagers(ctx context.Context, spaceID uint64) ([]searchstore.Manager, error) {
	f.cacheMu.RLock()
	if managers, ok := f.cache[spaceID]; ok {
		f.cacheMu.RUnlock()
		return managers, nil
	}
	f.cacheMu.RUnlock()

	emb, err := f.config.EmbeddingProvider.GetEmbedding(ctx, spaceID)
	if err != nil {
		logs.CtxWarnf(ctx, "[OBManagerFactory] failed to get embedding for space %d: %v, using global", spaceID, err)
		return f.GetGlobalManagers(ctx)
	}

	hasSpaceEmb, err := f.config.EmbeddingProvider.HasSpaceEmbedding(ctx, spaceID)
	if err != nil {
		logs.CtxWarnf(ctx, "[OBManagerFactory] failed to check space embedding: %v", err)
	}
	if !hasSpaceEmb {
		return f.GetGlobalManagers(ctx)
	}

	manager, err := f.createManager(emb)
	if err != nil {
		logs.CtxErrorf(ctx, "[OBManagerFactory] failed to create manager for space %d: %v", spaceID, err)
		return nil, err
	}

	managers := []searchstore.Manager{manager}

	f.cacheMu.Lock()
	f.cache[spaceID] = managers
	f.cacheMu.Unlock()

	return managers, nil
}

func (f *obManagerFactory) GetGlobalManagers(ctx context.Context) ([]searchstore.Manager, error) {
	var initErr error
	f.globalOnce.Do(func() {
		emb, err := f.config.EmbeddingProvider.GetGlobalEmbedding(ctx)
		if err != nil {
			if f.config.GlobalEmbedding != nil {
				emb = f.config.GlobalEmbedding
			} else {
				initErr = fmt.Errorf("no global embedding configured: %w", err)
				return
			}
		}

		manager, err := f.createManager(emb)
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

func (f *obManagerFactory) HasSpaceManagers(ctx context.Context, spaceID uint64) (bool, error) {
	return f.config.EmbeddingProvider.HasSpaceEmbedding(ctx, spaceID)
}

func (f *obManagerFactory) createManager(emb embedding.Embedder) (searchstore.Manager, error) {
	batchSize := f.config.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	return NewManager(&ManagerConfig{
		DB:        f.config.DB,
		Embedding: emb,
		BatchSize: batchSize,
	})
}
