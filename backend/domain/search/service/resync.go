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
	"time"

	intelligence "github.com/ynet-dev/ynet-studio/backend/api/model/app/intelligence/common"
	spacemodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	resource "github.com/ynet-dev/ynet-studio/backend/api/model/resource/common"
	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	appentity "github.com/ynet-dev/ynet-studio/backend/domain/app/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// kbEntriesIndex is the ES list index for knowledge bases in a space.
// (Slice / chunk indices use openynet_<kb_id> and are handled by the knowledge domain.)
const kbEntriesIndex = "kb_entries"

// KbInfo is a minimal projection of a knowledge base used by ResyncSpace.
// The application layer adapts the internal *model.Knowledge into this
// public-friendly shape so search/service stays decoupled from the
// knowledge domain's internal packages.
type KbInfo struct {
	ID          int64
	SpaceID     int64
	AppID       int64
	OwnerID     int64
	Name        string
	FormatType  int32
	CreatedAtMs int64
	UpdatedAtMs int64
}

// AgentLister returns single-agent drafts for a space.
type AgentLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*agententity.SingleAgent, error)
}

// AppLister returns draft apps (projects) for a space.
type AppLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*appentity.APP, error)
}

// KbLister returns knowledge bases for a space as KbInfo views.
type KbLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*KbInfo, error)
}

// ResyncSpace drops all ES docs for the given space + replays writes from
// MySQL for project_draft / coze_resource / kb_entries indices.
//
// Knowledge chunk indices (openynet_<kb_id>) are NOT handled here — the
// knowledge domain owns those via knowledgeSvc.ResyncSpaceSlices.
//
// Per-document Create failures are logged and skipped; only the listing /
// delete-by-query failures abort the whole call.
func (s *searchImpl) ResyncSpace(ctx context.Context, spaceID int64) (*spacemodel.ResyncESCounts, error) {
	if s.agentRepo == nil || s.appRepo == nil || s.kbRepo == nil {
		return nil, fmt.Errorf("search.ResyncSpace: resync deps not wired (call SetResyncDeps first)")
	}

	counts := &spacemodel.ResyncESCounts{}

	// (a) Knowledge IDs first — needed for kb_entries terms filter.
	kbs, err := s.kbRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list kbs by space %d: %w", spaceID, err)
	}
	kbIDs := make([]int64, 0, len(kbs))
	for _, kb := range kbs {
		kbIDs = append(kbIDs, kb.ID)
	}

	// (b) delete_by_query on the 3 list indices.
	if _, err := s.esClient.DeleteByQuery(ctx, projectIndexName, map[string]any{
		"term": map[string]any{"space_id": spaceID},
	}); err != nil {
		return nil, fmt.Errorf("delete %s for space %d: %w", projectIndexName, spaceID, err)
	}
	if _, err := s.esClient.DeleteByQuery(ctx, resourceIndexName, map[string]any{
		"term": map[string]any{"space_id": spaceID},
	}); err != nil {
		return nil, fmt.Errorf("delete %s for space %d: %w", resourceIndexName, spaceID, err)
	}
	if len(kbIDs) > 0 {
		if _, err := s.esClient.DeleteByQuery(ctx, kbEntriesIndex, map[string]any{
			"terms": map[string]any{"kb_id": kbIDs},
		}); err != nil {
			return nil, fmt.Errorf("delete %s for kbs %v: %w", kbEntriesIndex, kbIDs, err)
		}
	}

	// (c) rewrite project_draft from agents.
	agents, err := s.agentRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list agents by space %d: %w", spaceID, err)
	}
	for _, a := range agents {
		if err := s.indexAgent(ctx, a); err != nil {
			logs.CtxWarnf(ctx, "resync: index agent %d failed: %v", agentID(a), err)
			continue
		}
		counts.ProjectDraft++
	}

	// (d) rewrite coze_resource from draft apps.
	apps, err := s.appRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list apps by space %d: %w", spaceID, err)
	}
	for _, app := range apps {
		if err := s.indexApp(ctx, app); err != nil {
			logs.CtxWarnf(ctx, "resync: index app %d failed: %v", app.ID, err)
			continue
		}
		counts.CozeResource++
	}

	// (e) rewrite kb_entries from KBs (already fetched in step a).
	for _, kb := range kbs {
		if err := s.indexKbEntry(ctx, kb); err != nil {
			logs.CtxWarnf(ctx, "resync: index kb %d failed: %v", kb.ID, err)
			continue
		}
		counts.KbEntries++
	}

	return counts, nil
}

// indexAgent writes a single agent into the project_draft ES index, mirroring
// the document shape produced by application/singleagent/create.go on Created.
func (s *searchImpl) indexAgent(ctx context.Context, a *agententity.SingleAgent) error {
	if a == nil || a.SingleAgent == nil {
		return fmt.Errorf("nil agent")
	}
	updateMs := a.UpdatedAt
	createMs := a.CreatedAt
	if updateMs == 0 {
		updateMs = time.Now().UnixMilli()
	}
	if createMs == 0 {
		createMs = updateMs
	}
	doc := &entity.ProjectDocument{
		ID:           a.AgentID,
		Type:         intelligence.IntelligenceType_Bot,
		Status:       intelligence.IntelligenceStatus_Using,
		Name:         ptr.Of(a.Name),
		SpaceID:      ptr.Of(a.SpaceID),
		OwnerID:      ptr.Of(a.CreatorID),
		CreateTimeMS: ptr.Of(createMs),
		UpdateTimeMS: ptr.Of(updateMs),
	}
	return s.esClient.Create(ctx, projectIndexName, conv.Int64ToStr(a.AgentID), doc)
}

// indexApp writes a single draft app into the coze_resource ES index — but
// also into the project_draft index, which is where intelligence lists pull
// apps from (cf. application/app/app.go DraftProjectCreate).
//
// NOTE: in production, apps live in project_draft AND a row of type Project in
// coze_resource is *not* written by the app create flow. For the resync we
// mirror the source-of-truth: apps go to project_draft only.
func (s *searchImpl) indexApp(ctx context.Context, app *appentity.APP) error {
	if app == nil {
		return fmt.Errorf("nil app")
	}
	doc := &entity.ProjectDocument{
		ID:           app.ID,
		Type:         intelligence.IntelligenceType_Project,
		Status:       intelligence.IntelligenceStatus_Using,
		Name:         app.Name,
		SpaceID:      ptr.Of(app.SpaceID),
		OwnerID:      ptr.Of(app.OwnerID),
		CreateTimeMS: ptr.Of(app.CreatedAtMS),
		UpdateTimeMS: ptr.Of(app.UpdatedAtMS),
	}
	if app.PublishStatus != nil && *app.PublishStatus == appentity.PublishStatusOfPublishDone {
		doc.HasPublished = ptr.Of(1)
		if app.PublishedAtMS != nil {
			doc.PublishTimeMS = app.PublishedAtMS
		}
	}
	return s.esClient.Create(ctx, resourceIndexName, conv.Int64ToStr(app.ID), doc)
}

// indexKbEntry writes a single KB into the kb_entries ES index. Document
// shape mirrors what the knowledge create flow publishes to coze_resource
// (ResType_Knowledge), with kb_id repeated as the doc id.
func (s *searchImpl) indexKbEntry(ctx context.Context, kb *KbInfo) error {
	if kb == nil {
		return fmt.Errorf("nil kb")
	}
	var appID *int64
	if kb.AppID != 0 {
		appID = ptr.Of(kb.AppID)
	}
	doc := &entity.ResourceDocument{
		ResID:         kb.ID,
		ResType:       resource.ResType_Knowledge,
		ResSubType:    ptr.Of(kb.FormatType),
		Name:          ptr.Of(kb.Name),
		OwnerID:       ptr.Of(kb.OwnerID),
		SpaceID:       ptr.Of(kb.SpaceID),
		APPID:         appID,
		PublishStatus: ptr.Of(resource.PublishStatus_Published),
		CreateTimeMS:  ptr.Of(kb.CreatedAtMs),
		UpdateTimeMS:  ptr.Of(kb.UpdatedAtMs),
		PublishTimeMS: ptr.Of(kb.CreatedAtMs),
	}
	return s.esClient.Create(ctx, kbEntriesIndex, conv.Int64ToStr(kb.ID), doc)
}

// agentID safely extracts the AgentID for logging; protects against nil panic
// when the embedded crossdomain.SingleAgent is unset (shouldn't happen in
// practice, but indexAgent already guards).
func agentID(a *agententity.SingleAgent) int64 {
	if a == nil || a.SingleAgent == nil {
		return 0
	}
	return a.AgentID
}
