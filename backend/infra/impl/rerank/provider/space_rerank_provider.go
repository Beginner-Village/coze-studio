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

	rerankEntity "github.com/ynet-dev/ynet-studio/backend/domain/rerank/entity"
	rerankService "github.com/ynet-dev/ynet-studio/backend/domain/rerank/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/rerank"
	openaiRerank "github.com/ynet-dev/ynet-studio/backend/infra/impl/document/rerank/openai"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// SpaceRerankProvider provides reranker instances based on space configuration.
// If a space has a rerank model configured, returns a model-based reranker.
// Otherwise, returns nil (caller should fall back to RRF).
type SpaceRerankProvider struct {
	rerankService rerankService.SpaceRerankService

	cache   map[uint64]rerank.Reranker
	cacheMu sync.RWMutex
}

// NewSpaceRerankProvider creates a new SpaceRerankProvider
func NewSpaceRerankProvider(rerankSvc rerankService.SpaceRerankService) *SpaceRerankProvider {
	return &SpaceRerankProvider{
		rerankService: rerankSvc,
		cache:         make(map[uint64]rerank.Reranker),
	}
}

// GetReranker returns a model-based Reranker for the given space, or nil if not configured.
func (p *SpaceRerankProvider) GetReranker(ctx context.Context, spaceID uint64) (rerank.Reranker, error) {
	// Check cache first
	p.cacheMu.RLock()
	if r, ok := p.cache[spaceID]; ok {
		p.cacheMu.RUnlock()
		return r, nil
	}
	p.cacheMu.RUnlock()

	// Try to get space-specific rerank config
	spaceRerank, err := p.rerankService.GetDefaultSpaceRerankEntity(ctx, spaceID)
	if err != nil {
		logs.CtxWarnf(ctx, "[SpaceRerankProvider] failed to get space rerank, spaceID=%d, err=%v", spaceID, err)
		return nil, nil // No config = no model reranker
	}

	if spaceRerank == nil || spaceRerank.Status != rerankEntity.SpaceRerankStatusEnabled {
		return nil, nil // No enabled config
	}

	config, err := spaceRerank.GetConfigStruct()
	if err != nil {
		logs.CtxErrorf(ctx, "[SpaceRerankProvider] failed to parse rerank config, spaceID=%d, err=%v", spaceID, err)
		return nil, nil
	}

	r, err := p.createRerankFromConfig(ctx, config)
	if err != nil {
		logs.CtxErrorf(ctx, "[SpaceRerankProvider] failed to create reranker from config, spaceID=%d, err=%v", spaceID, err)
		return nil, nil
	}

	// Cache the reranker
	p.cacheMu.Lock()
	p.cache[spaceID] = r
	p.cacheMu.Unlock()

	return r, nil
}

// InvalidateCache removes the cached reranker for a space
func (p *SpaceRerankProvider) InvalidateCache(spaceID uint64) {
	p.cacheMu.Lock()
	delete(p.cache, spaceID)
	p.cacheMu.Unlock()
}

func (p *SpaceRerankProvider) createRerankFromConfig(ctx context.Context, config *rerankEntity.RerankConfig) (rerank.Reranker, error) {
	if config == nil {
		return nil, fmt.Errorf("rerank config is nil")
	}

	switch config.Type {
	case rerankEntity.RerankTypeOpenAI:
		if config.OpenAI == nil {
			return nil, fmt.Errorf("OpenAI rerank config is nil")
		}
		return openaiRerank.NewReranker(&openaiRerank.Config{
			BaseURL: config.OpenAI.BaseURL,
			APIKey:  config.OpenAI.APIKey,
			Model:   config.OpenAI.Model,
		}), nil

	default:
		return nil, fmt.Errorf("unsupported rerank type: %s", config.Type)
	}
}
