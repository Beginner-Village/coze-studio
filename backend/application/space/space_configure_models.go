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

package space

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"

	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// ConfigureModelsSVC is the global one-shot per-space model reconfig service.
// Wired in initComplexServices alongside ResyncSVC.
var ConfigureModelsSVC *SpaceConfigureModelsService

// InMemoryCacheInvalidator can flush in-process cached objects for a space.
// Both SpaceRerankProvider and SpaceEmbeddingProvider satisfy this.
type InMemoryCacheInvalidator interface {
	InvalidateCache(spaceID uint64)
}

// SpaceConfigureModelsService atomically rewrites the chat / embedder / rerank
// configs for a single space and clears ALL caches (Redis per-space model
// cache + in-process rerank/embedding provider caches) so the new values
// take effect on the very next request without a pod restart.
type SpaceConfigureModelsService struct {
	userSVC              usersvc.User
	db                   *gorm.DB
	redis                cache.Cmdable
	rerankCacheInvalid   InMemoryCacheInvalidator
	embeddingCacheInvalid InMemoryCacheInvalidator
}

// InitConfigureModelsService wires the per-space reconfig service. Must be
// called after the user application service is up so its DomainSVC is ready.
func InitConfigureModelsService(userSVC usersvc.User, db *gorm.DB, redis cache.Cmdable, rerankInv, embeddingInv InMemoryCacheInvalidator) {
	ConfigureModelsSVC = &SpaceConfigureModelsService{
		userSVC:              userSVC,
		rerankCacheInvalid:   rerankInv,
		embeddingCacheInvalid: embeddingInv,
		db:      db,
		redis:   redis,
	}
}

// ConfigureModels applies the chat/embedder/rerank reconfig for the given
// space and clears the model cache. Only the space owner can call this.
//
// Any of chat / embedder / rerank in the request may be nil — that section is
// then skipped. If all three are nil we still run permission + cache cleanup
// (the operator may want to nuke a stale cache without touching configs).
func (s *SpaceConfigureModelsService) ConfigureModels(ctx context.Context, req *spacemodel.ConfigureModelsRequest) (*spacemodel.ConfigureModelsResponse, error) {
	// 1. permission — only the space owner can rewrite per-space configs.
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	spaceInfo, err := s.userSVC.GetSpaceByID(ctx, req.SpaceID)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceNotFoundCode, errorx.KV("msg", fmt.Sprintf("get space %d failed", req.SpaceID)))
	}
	if spaceInfo == nil {
		return nil, errorx.New(errno.ErrSpaceNotFoundCode, errorx.KV("msg", fmt.Sprintf("space %d not found", req.SpaceID)))
	}
	if spaceInfo.OwnerID != userID {
		return nil, errorx.New(errno.ErrSpacePermissionCode, errorx.KV("msg", "only the space owner can reconfigure models"))
	}

	counts := &spacemodel.ConfigureModelsCounts{Warnings: []string{}}

	// 2. DB writes — single transaction so a partial failure rolls back.
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if req.Chat != nil {
			n, err := updateModelMeta(tx, req.Chat)
			if err != nil {
				return fmt.Errorf("update model_meta: %w", err)
			}
			counts.ModelMetaUpdated = n
		}
		if req.Embedder != nil {
			n, err := updateSpaceEmbedding(tx, req.SpaceID, req.Embedder)
			if err != nil {
				return fmt.Errorf("update space_embedding: %w", err)
			}
			counts.SpaceEmbeddingUpdated = n
		}
		if req.Rerank != nil {
			n, err := updateSpaceRerank(tx, req.SpaceID, req.Rerank)
			if err != nil {
				return fmt.Errorf("update space_rerank: %w", err)
			}
			counts.SpaceRerankUpdated = n
		}
		return nil
	}); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceConfigureModelsCode, errorx.KV("msg", err.Error()))
	}

	// 3. cache invalidation — clear ALL caches so the next request re-reads
	// from MySQL. Three layers: Redis per-space model cache + in-process
	// rerank provider cache + in-process embedding provider cache.
	deleted, warning := s.invalidateSpaceModelCache(ctx, req.SpaceID)
	counts.RedisKeysDeleted = deleted
	if warning != "" {
		counts.Warnings = append(counts.Warnings, warning)
	}
	spaceIDU := uint64(req.SpaceID)
	if s.rerankCacheInvalid != nil {
		s.rerankCacheInvalid.InvalidateCache(spaceIDU)
		logs.CtxInfof(ctx, "[ConfigureModels] invalidated in-memory rerank cache for space %d", req.SpaceID)
	}
	if s.embeddingCacheInvalid != nil {
		s.embeddingCacheInvalid.InvalidateCache(spaceIDU)
		logs.CtxInfof(ctx, "[ConfigureModels] invalidated in-memory embedding cache for space %d", req.SpaceID)
	}

	logs.CtxInfof(ctx, "[ConfigureModels] space=%d model_meta=%d space_embedding=%d space_rerank=%d redis_deleted=%d warnings=%d",
		req.SpaceID, counts.ModelMetaUpdated, counts.SpaceEmbeddingUpdated, counts.SpaceRerankUpdated, counts.RedisKeysDeleted, len(counts.Warnings))

	return &spacemodel.ConfigureModelsResponse{
		Code: 0,
		Msg:  "success",
		Data: counts,
	}, nil
}

// updateModelMeta rewrites conn_config for ALL active model_meta rows. The
// model_meta table is global (one row per LLM the studio knows about), not
// per-space — every space shares the same chat LLM in this customer's
// deployment, so a single UPDATE matches the customer's manual SQL.
func updateModelMeta(tx *gorm.DB, chat *spacemodel.ChatModelConfig) (int64, error) {
	connConfig, err := json.Marshal(map[string]any{
		"base_url":        chat.BaseURL,
		"api_key":         chat.APIKey,
		"model":           chat.Model,
		"enable_thinking": false,
	})
	if err != nil {
		return 0, err
	}
	res := tx.Exec(
		"UPDATE model_meta SET conn_config = ? WHERE deleted_at IS NULL",
		string(connConfig),
	)
	return res.RowsAffected, res.Error
}

// updateSpaceEmbedding rewrites config for the embedder row(s) of a single space.
func updateSpaceEmbedding(tx *gorm.DB, spaceID int64, emb *spacemodel.EmbedderModelConfig) (int64, error) {
	config, err := json.Marshal(map[string]any{
		"type": "openai",
		"openai_config": map[string]any{
			"dims":     emb.Dims,
			"model":    emb.Model,
			"api_key":  emb.APIKey,
			"base_url": emb.BaseURL,
		},
		"max_batch_size": 100,
	})
	if err != nil {
		return 0, err
	}
	res := tx.Exec(
		"UPDATE space_embedding SET config = ? WHERE space_id = ? AND deleted_at IS NULL",
		string(config), spaceID,
	)
	return res.RowsAffected, res.Error
}

// updateSpaceRerank rewrites config for the rerank row(s) of a single space.
func updateSpaceRerank(tx *gorm.DB, spaceID int64, rerank *spacemodel.RerankModelConfig) (int64, error) {
	config, err := json.Marshal(map[string]any{
		"type": "openai",
		"openai_config": map[string]any{
			"model":    rerank.Model,
			"api_key":  rerank.APIKey,
			"base_url": rerank.BaseURL,
		},
	})
	if err != nil {
		return 0, err
	}
	res := tx.Exec(
		"UPDATE space_rerank SET config = ? WHERE space_id = ? AND deleted_at IS NULL",
		string(config), spaceID,
	)
	return res.RowsAffected, res.Error
}

// invalidateSpaceModelCache deletes every `space:<space_id>:model:<model_entity_id>`
// Redis key that could have been written for this space. The cache abstraction
// doesn't expose SCAN, so we enumerate (space_model rows ∪ public model_entity
// rows) — the same union the cache writer could have inserted — and DEL each.
//
// Returns (deleted_count, warning). A Redis nil interface or a query failure
// is reported as a warning, never as an error: cache misses are recoverable
// (5-min TTL) and we'd rather not roll back a successful DB write.
func (s *SpaceConfigureModelsService) invalidateSpaceModelCache(ctx context.Context, spaceID int64) (int64, string) {
	if s.redis == nil {
		return 0, "redis not configured; cache will rely on TTL expiry"
	}

	// Union of (a) space-private models and (b) public models. Either set may
	// have been cached for this space at any point.
	type idRow struct {
		ID uint64 `gorm:"column:id"`
	}
	var ids []idRow
	const query = `
SELECT id FROM model_entity
WHERE deleted_at IS NULL
  AND (
    id IN (SELECT model_entity_id FROM space_model WHERE space_id = ? AND deleted_at IS NULL)
    OR is_public = 1
  )`
	if err := s.db.WithContext(ctx).Raw(query, spaceID).Scan(&ids).Error; err != nil {
		return 0, fmt.Sprintf("enumerate model_entity for cache cleanup failed: %v", err)
	}
	if len(ids) == 0 {
		return 0, ""
	}

	keys := make([]string, 0, len(ids))
	for _, r := range ids {
		keys = append(keys, fmt.Sprintf("space:%d:model:%d", spaceID, r.ID))
	}

	// Some Redis backends (notably cluster mode) reject multi-key DEL when
	// the keys hash to different slots. All our keys share the same `space:<id>:`
	// prefix so they would normally land in the same slot — but to stay robust
	// across stand-alone vs cluster deployments we DEL one key at a time.
	var deleted int64
	for _, k := range keys {
		n, err := s.redis.Del(ctx, k).Result()
		if err != nil {
			return deleted, fmt.Sprintf("redis DEL failed at key %s: %v (deleted %d of %d so far)", k, err, deleted, len(keys))
		}
		deleted += n
	}
	return deleted, ""
}
