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
	"strconv"

	"github.com/bytedance/sonic"

	knowledgeModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/knowledge"
	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/internal/events"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/eventbus"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// ResyncSpaceSlices implements Knowledge.ResyncSpaceSlices.
//
// Flow per KB in the space:
//  1. For every searchstore Manager wired into knowledgeSVC (vector +
//     text), call SearchStore.DeleteIndex("openynet_<kb_id>"). Only the ES
//     text store actually drops anything; the vector stores are no-ops and
//     vector wipes happen through their own Manager.Drop path (out of scope
//     here — the resync flow targets ES only).
//  2. Pull every slice for the KB via FindSliceByCondition(KnowledgeID).
//  3. BatchSetStatus(ids, SliceStatusInit, "") so the UI shows
//     "PendingVectoring" until the consumer flips each back to Done.
//  4. Marshal + publish an IndexSliceEvent per slice (sharded by
//     DocumentID, matching CreateSlice/UpdateSlice). The existing
//     indexSlice consumer (event_handle.go:518) fetches the document, runs
//     the embedding pipeline, and writes ES.
//
// Per-KB failures are logged and skipped; the only fatal error is
// knowledgeRepo.ListBySpaceID failing — without a KB list the call would be
// silently empty.
func (k *knowledgeSVC) ResyncSpaceSlices(ctx context.Context, spaceID int64) (int, error) {
	kbs, err := k.knowledgeRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return 0, fmt.Errorf("list kbs by space %d: %w", spaceID, err)
	}

	managers, mErr := k.getManagersForSpace(ctx, uint64(spaceID))
	if mErr != nil {
		// Without managers we can't drop indices, but we can still flip
		// slice status + republish so the next consumer cycle re-creates
		// ES docs (ES upsert is idempotent on doc id). Log + continue.
		logs.CtxWarnf(ctx, "[ResyncSpaceSlices] getManagersForSpace(%d) failed: %v (skipping index drops)", spaceID, mErr)
		managers = nil
	}

	total := 0
	for _, kb := range kbs {
		indexName := getCollectionName(kb.ID)

		// (1) Drop ES index for this KB. Non-ES stores are no-ops; ES
		// uses IgnoreUnavailable=true so a missing index returns nil.
		for _, mgr := range managers {
			ss, err := mgr.GetSearchStore(ctx, indexName)
			if err != nil {
				logs.CtxWarnf(ctx, "[ResyncSpaceSlices] kb=%d type=%s get search store failed: %v",
					kb.ID, mgr.GetType(), err)
				continue
			}
			if err := ss.DeleteIndex(ctx, indexName); err != nil {
				logs.CtxWarnf(ctx, "[ResyncSpaceSlices] kb=%d type=%s DeleteIndex(%s) failed: %v",
					kb.ID, mgr.GetType(), indexName, err)
			}
		}

		// (2) List all slices for this KB.
		slices, _, err := k.sliceRepo.FindSliceByCondition(ctx, &entity.WhereSliceOpt{
			KnowledgeID: kb.ID,
		})
		if err != nil {
			logs.CtxWarnf(ctx, "[ResyncSpaceSlices] kb=%d FindSliceByCondition failed: %v", kb.ID, err)
			continue
		}
		if len(slices) == 0 {
			continue
		}

		// (3) Flip status back to Init so the UI shows PendingVectoring.
		ids := make([]int64, 0, len(slices))
		for _, s := range slices {
			ids = append(ids, s.ID)
		}
		if err := k.sliceRepo.BatchSetStatus(ctx, ids, int32(knowledgeModel.SliceStatusInit), ""); err != nil {
			logs.CtxWarnf(ctx, "[ResyncSpaceSlices] kb=%d BatchSetStatus(init) failed: %v", kb.ID, err)
			continue
		}

		// (4) Publish IndexSliceEvent per slice. The consumer will fetch
		// the document, parse, embed, and write ES.
		for _, s := range slices {
			ev := events.NewIndexSliceEvent(&entity.Slice{
				Info:        knowledgeModel.Info{ID: s.ID},
				KnowledgeID: s.KnowledgeID,
				DocumentID:  s.DocumentID,
			}, nil) // document is fetched inside indexSlice if nil.

			body, err := sonic.Marshal(ev)
			if err != nil {
				logs.CtxWarnf(ctx, "[ResyncSpaceSlices] kb=%d slice=%d marshal failed: %v",
					kb.ID, s.ID, err)
				continue
			}
			if err := k.producer.Send(ctx, body, eventbus.WithShardingKey(strconv.FormatInt(s.DocumentID, 10))); err != nil {
				logs.CtxWarnf(ctx, "[ResyncSpaceSlices] kb=%d slice=%d producer.Send failed: %v",
					kb.ID, s.ID, err)
				continue
			}
			total++
		}
	}

	logs.CtxInfof(ctx, "[ResyncSpaceSlices] space=%d kbs=%d slices_queued=%d", spaceID, len(kbs), total)
	return total, nil
}
