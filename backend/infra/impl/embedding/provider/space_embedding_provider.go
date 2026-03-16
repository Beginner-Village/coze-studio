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

package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino-ext/components/embedding/openai"

	embEntity "github.com/ynet-dev/ynet-studio/backend/domain/embedding/entity"
	embService "github.com/ynet-dev/ynet-studio/backend/domain/embedding/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
	arkemb "github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/ark"
	httpemb "github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/http"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/wrap"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// SpaceEmbeddingProvider provides embedding instances based on space configuration
type SpaceEmbeddingProvider struct {
	embeddingService embService.SpaceEmbeddingService
	globalEmbedding  embedding.Embedder

	// Cache for space embeddings to avoid repeated database queries
	cache    map[uint64]embedding.Embedder
	cacheMu  sync.RWMutex
}

// NewSpaceEmbeddingProvider creates a new SpaceEmbeddingProvider
func NewSpaceEmbeddingProvider(
	embeddingService embService.SpaceEmbeddingService,
	globalEmbedding embedding.Embedder,
) embedding.Provider {
	return &SpaceEmbeddingProvider{
		embeddingService: embeddingService,
		globalEmbedding:  globalEmbedding,
		cache:            make(map[uint64]embedding.Embedder),
	}
}

// GetEmbedding returns an Embedder for the given space
func (p *SpaceEmbeddingProvider) GetEmbedding(ctx context.Context, spaceID uint64) (embedding.Embedder, error) {
	// Check cache first
	p.cacheMu.RLock()
	if emb, ok := p.cache[spaceID]; ok {
		p.cacheMu.RUnlock()
		return emb, nil
	}
	p.cacheMu.RUnlock()

	// Try to get space-specific embedding
	spaceEmb, err := p.embeddingService.GetDefaultSpaceEmbeddingEntity(ctx, spaceID)
	if err != nil {
		logs.CtxWarnf(ctx, "[SpaceEmbeddingProvider] failed to get space embedding, spaceID=%d, err=%v, using global", spaceID, err)
		// Fall back to global embedding
		if p.globalEmbedding != nil {
			return p.globalEmbedding, nil
		}
		return nil, fmt.Errorf("no embedding configured for space %d and no global fallback", spaceID)
	}

	if spaceEmb == nil || spaceEmb.Status != embEntity.SpaceEmbeddingStatusEnabled {
		logs.CtxInfof(ctx, "[SpaceEmbeddingProvider] space %d has no enabled embedding config, using global", spaceID)
		if p.globalEmbedding != nil {
			return p.globalEmbedding, nil
		}
		return nil, fmt.Errorf("no embedding configured for space %d and no global fallback", spaceID)
	}

	// Parse config from JSON
	config, err := spaceEmb.GetConfigStruct()
	if err != nil {
		logs.CtxErrorf(ctx, "[SpaceEmbeddingProvider] failed to parse embedding config, spaceID=%d, err=%v", spaceID, err)
		if p.globalEmbedding != nil {
			return p.globalEmbedding, nil
		}
		return nil, fmt.Errorf("failed to parse embedding config for space %d: %w", spaceID, err)
	}

	// Create embedding instance from config
	emb, err := p.createEmbeddingFromConfig(ctx, config)
	if err != nil {
		logs.CtxErrorf(ctx, "[SpaceEmbeddingProvider] failed to create embedding from config, spaceID=%d, err=%v", spaceID, err)
		// Fall back to global embedding
		if p.globalEmbedding != nil {
			return p.globalEmbedding, nil
		}
		return nil, fmt.Errorf("failed to create embedding for space %d: %w", spaceID, err)
	}

	// Cache the embedding
	p.cacheMu.Lock()
	p.cache[spaceID] = emb
	p.cacheMu.Unlock()

	return emb, nil
}

// GetGlobalEmbedding returns the global default Embedder
func (p *SpaceEmbeddingProvider) GetGlobalEmbedding(ctx context.Context) (embedding.Embedder, error) {
	if p.globalEmbedding == nil {
		return nil, fmt.Errorf("no global embedding configured")
	}
	return p.globalEmbedding, nil
}

// HasSpaceEmbedding checks if a space has its own embedding configuration
func (p *SpaceEmbeddingProvider) HasSpaceEmbedding(ctx context.Context, spaceID uint64) (bool, error) {
	spaceEmb, err := p.embeddingService.GetDefaultSpaceEmbeddingEntity(ctx, spaceID)
	if err != nil {
		return false, nil // Treat error as no config
	}
	return spaceEmb != nil && spaceEmb.Status == embEntity.SpaceEmbeddingStatusEnabled, nil
}

// InvalidateCache removes the cached embedding for a space
// Call this when space embedding configuration changes
func (p *SpaceEmbeddingProvider) InvalidateCache(spaceID uint64) {
	p.cacheMu.Lock()
	delete(p.cache, spaceID)
	p.cacheMu.Unlock()
}

// InvalidateAllCache clears all cached embeddings
func (p *SpaceEmbeddingProvider) InvalidateAllCache() {
	p.cacheMu.Lock()
	p.cache = make(map[uint64]embedding.Embedder)
	p.cacheMu.Unlock()
}

// createEmbeddingFromConfig creates an Embedder from EmbeddingConfig
func (p *SpaceEmbeddingProvider) createEmbeddingFromConfig(ctx context.Context, config *embEntity.EmbeddingConfig) (embedding.Embedder, error) {
	if config == nil {
		return nil, fmt.Errorf("embedding config is nil")
	}

	batchSize := config.MaxBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	switch config.Type {
	case embEntity.EmbeddingTypeOpenAI:
		if config.OpenAI == nil {
			return nil, fmt.Errorf("OpenAI config is nil")
		}
		cfg := config.OpenAI
		openAICfg := &openai.EmbeddingConfig{
			APIKey:     cfg.APIKey,
			ByAzure:    cfg.ByAzure,
			BaseURL:    cfg.BaseURL,
			APIVersion: cfg.APIVersion,
			Model:      cfg.Model,
		}
		if cfg.RequestDims > 0 {
			dims := cfg.RequestDims
			openAICfg.Dimensions = &dims
		}
		return wrap.NewOpenAIEmbedder(ctx, openAICfg, int64(cfg.Dims), batchSize)

	case embEntity.EmbeddingTypeArk:
		if config.Ark == nil {
			return nil, fmt.Errorf("ARK config is nil")
		}
		cfg := config.Ark
		apiType := ark.APITypeText
		if cfg.APIType == "multimodal" {
			apiType = ark.APITypeMultiModal
		}
		// ARK API has a strict limit of 10 for batch size (applies to all ARK embeddings)
		arkBatchSize := batchSize
		if arkBatchSize > 10 {
			arkBatchSize = 10
		}
		return arkemb.NewArkEmbedder(ctx, &ark.EmbeddingConfig{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
			APIType: &apiType,
		}, int64(cfg.Dims), arkBatchSize)

	case embEntity.EmbeddingTypeOllama:
		if config.Ollama == nil {
			return nil, fmt.Errorf("Ollama config is nil")
		}
		cfg := config.Ollama
		return wrap.NewOllamaEmbedder(ctx, &ollama.EmbeddingConfig{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, int64(cfg.Dims), batchSize)

	case embEntity.EmbeddingTypeHTTP:
		if config.HTTP == nil {
			return nil, fmt.Errorf("HTTP config is nil")
		}
		cfg := config.HTTP
		return httpemb.NewEmbedding(cfg.Addr, int64(cfg.Dims), batchSize)

	default:
		return nil, fmt.Errorf("unsupported embedding type: %s", config.Type)
	}
}
