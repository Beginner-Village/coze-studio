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

	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/repository"
	searchsvc "github.com/ynet-dev/ynet-studio/backend/domain/search/service"
)

// NewKbInfoLister wraps a KnowledgeRepo so it satisfies searchsvc.KbLister.
// It lives in the knowledge domain because the conversion reads fields of
// the internal *model.Knowledge type — application packages cannot import
// that internal package directly (Go internal-import rule).
//
// Used by the per-space ES resync flow:
//
//	searchSvc.SetResyncDeps(agentRepo, appRepo, kbservice.NewKbInfoLister(kbRepo))
func NewKbInfoLister(repo repository.KnowledgeRepo) searchsvc.KbLister {
	return &kbInfoLister{repo: repo}
}

type kbInfoLister struct {
	repo repository.KnowledgeRepo
}

func (a *kbInfoLister) ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*searchsvc.KbInfo, error) {
	raw, err := a.repo.ListBySpaceID(ctx, spaceID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*searchsvc.KbInfo, 0, len(raw))
	for _, kb := range raw {
		if kb == nil {
			continue
		}
		out = append(out, &searchsvc.KbInfo{
			ID:          kb.ID,
			SpaceID:     kb.SpaceID,
			AppID:       kb.AppID,
			OwnerID:     kb.CreatorID,
			Name:        kb.Name,
			FormatType:  kb.FormatType,
			CreatedAtMs: kb.CreatedAt,
			UpdatedAtMs: kb.UpdatedAt,
		})
	}
	return out, nil
}
