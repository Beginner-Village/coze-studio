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

// WorkflowInfo is a minimal projection of a workflow_meta row used by ResyncSpace.
type WorkflowInfo struct {
	ID          int64
	SpaceID     int64
	AppID       int64 // 0 if library workflow (not bound to any app)
	OwnerID     int64
	Name        string
	Mode        int32 // workflow.WorkflowMode raw value
	HasPublish  bool  // true if metadata says it has at least one published version
	CreatedAtMs int64
	UpdatedAtMs int64
}

// PluginInfoView is a minimal projection of a plugin_draft row used by ResyncSpace.
type PluginInfoView struct {
	ID          int64
	SpaceID     int64
	AppID       int64 // 0 if library plugin
	OwnerID     int64
	Name        string
	PluginType  int32
	CreatedAtMs int64
	UpdatedAtMs int64
}

// PromptInfo is a minimal projection of a prompt_resource row used by ResyncSpace.
type PromptInfo struct {
	ID          int64
	SpaceID     int64
	OwnerID     int64
	Name        string
	CreatedAtMs int64
	UpdatedAtMs int64
}

// DatabaseInfo is a minimal projection of a draft_database_info row used by ResyncSpace.
type DatabaseInfo struct {
	ID          int64
	SpaceID     int64
	AppID       int64 // 0 if library database
	OwnerID     int64
	Name        string
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

// WorkflowLister returns workflow_meta rows for a space as WorkflowInfo views.
type WorkflowLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*WorkflowInfo, error)
}

// PluginLister returns plugin_draft rows for a space as PluginInfoView views.
type PluginLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*PluginInfoView, error)
}

// PromptLister returns prompt_resource rows for a space as PromptInfo views.
type PromptLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*PromptInfo, error)
}

// DatabaseLister returns draft_database_info rows for a space as DatabaseInfo views.
type DatabaseLister interface {
	ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*DatabaseInfo, error)
}

// ResyncSpace drops all ES docs for the given space + replays writes from
// MySQL for project_draft / coze_resource / kb_entries indices.
//
// Knowledge chunk indices (openynet_<kb_id>) are NOT handled here — the
// knowledge domain owns those via knowledgeSvc.ResyncSpaceSlices.
//
// coze_resource holds 5 different resource types (Plugin, Workflow,
// Knowledge, Prompt, Database) sourced from 5 different MySQL tables.
// kb_entries is a separate list index for knowledge bases.
//
// Per-document Create failures are logged and skipped; only the listing /
// delete-by-query failures abort the whole call.
func (s *searchImpl) ResyncSpace(ctx context.Context, spaceID int64) (*spacemodel.ResyncESCounts, error) {
	if s.agentRepo == nil || s.appRepo == nil || s.kbRepo == nil ||
		s.workflowRepo == nil || s.pluginRepo == nil ||
		s.promptRepo == nil || s.databaseRepo == nil {
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

	// (c) rewrite project_draft from agents + apps.
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

	apps, err := s.appRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list apps by space %d: %w", spaceID, err)
	}
	for _, app := range apps {
		if err := s.indexApp(ctx, app); err != nil {
			logs.CtxWarnf(ctx, "resync: index app %d failed: %v", app.ID, err)
			continue
		}
		counts.ProjectDraft++
	}

	// (d) rewrite coze_resource — 5 resource types from 5 different MySQL tables.

	// (d.1) Workflow (ResType=2) from workflow_meta.
	workflows, err := s.workflowRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list workflows by space %d: %w", spaceID, err)
	}
	for _, w := range workflows {
		if err := s.indexWorkflow(ctx, w); err != nil {
			logs.CtxWarnf(ctx, "resync: index workflow %d failed: %v", w.ID, err)
			continue
		}
		counts.CozeResource++
	}

	// (d.2) Plugin (ResType=1) from plugin_draft.
	plugins, err := s.pluginRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list plugins by space %d: %w", spaceID, err)
	}
	for _, p := range plugins {
		if err := s.indexPlugin(ctx, p); err != nil {
			logs.CtxWarnf(ctx, "resync: index plugin %d failed: %v", p.ID, err)
			continue
		}
		counts.CozeResource++
	}

	// (d.3) Prompt (ResType=6) from prompt_resource.
	prompts, err := s.promptRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list prompts by space %d: %w", spaceID, err)
	}
	for _, p := range prompts {
		if err := s.indexPrompt(ctx, p); err != nil {
			logs.CtxWarnf(ctx, "resync: index prompt %d failed: %v", p.ID, err)
			continue
		}
		counts.CozeResource++
	}

	// (d.4) Database (ResType=7) from draft_database_info.
	databases, err := s.databaseRepo.ListBySpaceID(ctx, spaceID, 0)
	if err != nil {
		return nil, fmt.Errorf("list databases by space %d: %w", spaceID, err)
	}
	for _, d := range databases {
		if err := s.indexDatabase(ctx, d); err != nil {
			logs.CtxWarnf(ctx, "resync: index database %d failed: %v", d.ID, err)
			continue
		}
		counts.CozeResource++
	}

	// (d.5) Knowledge (ResType=4) from KBs — mirror what the create flow writes
	// (cf. domain/knowledge/service eventbus publish) so SearchResources can
	// find Knowledge entries inside coze_resource too.
	for _, kb := range kbs {
		if err := s.indexKnowledgeResource(ctx, kb); err != nil {
			logs.CtxWarnf(ctx, "resync: index knowledge resource %d failed: %v", kb.ID, err)
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

// indexApp writes a draft app into the project_draft ES index — that's the
// index intelligence lists pull projects from (cf. application/app/app.go).
// Apps deliberately do NOT go to coze_resource (no row of type Project is
// written by the app create flow).
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
	return s.esClient.Create(ctx, projectIndexName, conv.Int64ToStr(app.ID), doc)
}

// indexWorkflow writes a workflow into coze_resource (ResType_Workflow).
// Mirrors application/workflow/workflow.go CreateWorkflow's Resource doc shape.
func (s *searchImpl) indexWorkflow(ctx context.Context, w *WorkflowInfo) error {
	if w == nil {
		return fmt.Errorf("nil workflow")
	}
	var appID *int64
	if w.AppID != 0 {
		appID = ptr.Of(w.AppID)
	}
	publishStatus := resource.PublishStatus_UnPublished
	if w.HasPublish {
		publishStatus = resource.PublishStatus_Published
	}
	doc := &entity.ResourceDocument{
		ResID:         w.ID,
		ResType:       resource.ResType_Workflow,
		ResSubType:    ptr.Of(w.Mode),
		Name:          ptr.Of(w.Name),
		OwnerID:       ptr.Of(w.OwnerID),
		SpaceID:       ptr.Of(w.SpaceID),
		APPID:         appID,
		PublishStatus: ptr.Of(publishStatus),
		CreateTimeMS:  ptr.Of(w.CreatedAtMs),
		UpdateTimeMS:  ptr.Of(w.UpdatedAtMs),
	}
	return s.esClient.Create(ctx, resourceIndexName, conv.Int64ToStr(w.ID), doc)
}

// indexPlugin writes a plugin into coze_resource (ResType_Plugin).
// Mirrors application/plugin/plugin.go RegisterPluginMeta's Resource doc shape.
func (s *searchImpl) indexPlugin(ctx context.Context, p *PluginInfoView) error {
	if p == nil {
		return fmt.Errorf("nil plugin")
	}
	var appID *int64
	if p.AppID != 0 {
		appID = ptr.Of(p.AppID)
	}
	doc := &entity.ResourceDocument{
		ResID:         p.ID,
		ResType:       resource.ResType_Plugin,
		ResSubType:    ptr.Of(p.PluginType),
		Name:          ptr.Of(p.Name),
		OwnerID:       ptr.Of(p.OwnerID),
		SpaceID:       ptr.Of(p.SpaceID),
		APPID:         appID,
		PublishStatus: ptr.Of(resource.PublishStatus_UnPublished),
		CreateTimeMS:  ptr.Of(p.CreatedAtMs),
		UpdateTimeMS:  ptr.Of(p.UpdatedAtMs),
	}
	return s.esClient.Create(ctx, resourceIndexName, conv.Int64ToStr(p.ID), doc)
}

// indexPrompt writes a prompt into coze_resource (ResType_Prompt).
// Mirrors application/prompt/prompt.go UpsertPromptResource's Resource doc shape.
func (s *searchImpl) indexPrompt(ctx context.Context, p *PromptInfo) error {
	if p == nil {
		return fmt.Errorf("nil prompt")
	}
	doc := &entity.ResourceDocument{
		ResID:         p.ID,
		ResType:       resource.ResType_Prompt,
		Name:          ptr.Of(p.Name),
		OwnerID:       ptr.Of(p.OwnerID),
		SpaceID:       ptr.Of(p.SpaceID),
		PublishStatus: ptr.Of(resource.PublishStatus_Published),
		CreateTimeMS:  ptr.Of(p.CreatedAtMs),
		UpdateTimeMS:  ptr.Of(p.UpdatedAtMs),
	}
	return s.esClient.Create(ctx, resourceIndexName, conv.Int64ToStr(p.ID), doc)
}

// indexDatabase writes a database into coze_resource (ResType_Database).
// Mirrors application/memory/database.go AddDatabase's Resource doc shape.
func (s *searchImpl) indexDatabase(ctx context.Context, d *DatabaseInfo) error {
	if d == nil {
		return fmt.Errorf("nil database")
	}
	var appID *int64
	if d.AppID != 0 {
		appID = ptr.Of(d.AppID)
	}
	doc := &entity.ResourceDocument{
		ResID:         d.ID,
		ResType:       resource.ResType_Database,
		Name:          ptr.Of(d.Name),
		OwnerID:       ptr.Of(d.OwnerID),
		SpaceID:       ptr.Of(d.SpaceID),
		APPID:         appID,
		PublishStatus: ptr.Of(resource.PublishStatus_UnPublished),
		CreateTimeMS:  ptr.Of(d.CreatedAtMs),
		UpdateTimeMS:  ptr.Of(d.UpdatedAtMs),
	}
	return s.esClient.Create(ctx, resourceIndexName, conv.Int64ToStr(d.ID), doc)
}

// indexKnowledgeResource writes a knowledge base entry into coze_resource
// (ResType_Knowledge). The create flow publishes a Resource event so SearchResources
// can find KBs; resync mirrors that.
func (s *searchImpl) indexKnowledgeResource(ctx context.Context, kb *KbInfo) error {
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
	return s.esClient.Create(ctx, resourceIndexName, conv.Int64ToStr(kb.ID), doc)
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
