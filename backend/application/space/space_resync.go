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
	"fmt"

	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	knowledgesvc "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/service"
	searchsvc "github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// ResyncSVC is the global per-space ES resync application service.
// Wired in initComplexServices after both search/knowledge are ready.
var ResyncSVC *SpaceResyncService

// SpaceResyncService rebuilds ES indices for a single space.
//
// Layered orchestration:
//   - (a) owner permission check (delegated to user domain)
//   - (b) list-index rebuild (project_draft / coze_resource / kb_entries) —
//     synchronous, via search domain
//   - (c) slice/chunk re-embedding (openynet_<kb_id>) — async via MQ, via
//     knowledge domain
//
// Step (c) failures are logged but don't fail the overall call, because
// the synchronous list rebuild is the user-visible portion. If MQ is
// down, slices will need a separate retry but list views work immediately.
type SpaceResyncService struct {
	userSVC      usersvc.User
	searchSVC    searchsvc.Search
	knowledgeSVC knowledgesvc.Knowledge
}

// InitResyncService wires the per-space ES resync service. Must be called
// after both the search and knowledge application services have finished
// init, since both their DomainSVCs are required.
func InitResyncService(userSVC usersvc.User, searchSVC searchsvc.Search, knowledgeSVC knowledgesvc.Knowledge) {
	ResyncSVC = &SpaceResyncService{
		userSVC:      userSVC,
		searchSVC:    searchSVC,
		knowledgeSVC: knowledgeSVC,
	}
}

// ResyncES drops all ES docs for the given space and replays writes from
// MySQL. Only the space owner can trigger this — non-owners get
// ErrSpacePermissionCode.
func (s *SpaceResyncService) ResyncES(ctx context.Context, req *spacemodel.ResyncESRequest) (*spacemodel.ResyncESResponse, error) {
	// 1. permission: only the space owner can trigger a full resync.
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
		return nil, errorx.New(errno.ErrSpacePermissionCode, errorx.KV("msg", "only the space owner can resync ES"))
	}

	// 2. list-index rebuild (sync) — fails the whole call on error.
	counts, err := s.searchSVC.ResyncSpace(ctx, req.SpaceID)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceResyncESCode, errorx.KV("msg", "list-index rebuild failed"))
	}
	if counts == nil {
		counts = &spacemodel.ResyncESCounts{}
	}

	// 3. slice re-embedding (async via MQ) — best-effort; don't fail the
	//    whole call if MQ is hiccupping, because the list rebuild is the
	//    user-visible win.
	queued, sliceErr := s.knowledgeSVC.ResyncSpaceSlices(ctx, req.SpaceID)
	if sliceErr != nil {
		logs.CtxWarnf(ctx, "[ResyncES] space=%d slice resync partial fail: %v (list indices already rebuilt)", req.SpaceID, sliceErr)
	}
	counts.SliceReindexJobs = queued

	logs.CtxInfof(ctx, "[ResyncES] space=%d done: project_draft=%d coze_resource=%d kb_entries=%d slice_jobs=%d",
		req.SpaceID, counts.ProjectDraft, counts.CozeResource, counts.KbEntries, counts.SliceReindexJobs)

	return &spacemodel.ResyncESResponse{
		Code:   0,
		Msg:    "success",
		Counts: counts,
	}, nil
}

// ResyncAllES is the admin bulk rebuild used after a DB-level data sync. It
// purges the three list indices (orphan cleanup) and rebuilds every space in
// req.SpaceIDs, bypassing the per-space owner gate (the domain ResyncSpace
// carries no ownership check). Gated only by requiring a logged-in caller —
// an internal maintenance operation, not a user-facing one.
func (s *SpaceResyncService) ResyncAllES(ctx context.Context, req *spacemodel.ResyncAllESRequest) (*spacemodel.ResyncAllESResponse, error) {
	if _, err := getUserIDFromContext(ctx); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(req.SpaceIDs))
	for _, raw := range req.SpaceIDs {
		id, err := conv.StrToInt64(raw)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errorx.New(errno.ErrSpaceResyncESCode, errorx.KV("msg", "no valid space_ids provided"))
	}

	agg, failed, err := s.searchSVC.ResyncAllSpaces(ctx, ids)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceResyncESCode, errorx.KV("msg", "bulk resync failed"))
	}
	if agg == nil {
		agg = &spacemodel.ResyncESCounts{}
	}

	failedStr := make([]string, 0, len(failed))
	for _, f := range failed {
		failedStr = append(failedStr, conv.Int64ToStr(f))
	}
	logs.CtxInfof(ctx, "[ResyncAllES] requested=%d ok=%d failed=%d project_draft=%d coze_resource=%d kb_entries=%d",
		len(ids), len(ids)-len(failed), len(failed), agg.ProjectDraft, agg.CozeResource, agg.KbEntries)

	return &spacemodel.ResyncAllESResponse{
		Code: 0,
		Msg:  "success",
		Counts: &spacemodel.ResyncAllESCounts{
			Spaces:       len(ids) - len(failed),
			Failed:       len(failed),
			FailedIDs:    failedStr,
			ProjectDraft: agg.ProjectDraft,
			CozeResource: agg.CozeResource,
			KbEntries:    agg.KbEntries,
			Purged:       true,
		},
	}, nil
}
