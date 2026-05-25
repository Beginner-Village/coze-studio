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

	"gorm.io/gorm"
)

// The following adapters wrap raw *gorm.DB into the per-resource Lister
// interfaces used by ResyncSpace. They live here (not in each domain) for
// two reasons:
//
//  1. The PO model packages are under internal/dal/model/ paths which Go's
//     internal-import rule forbids from importing into other domains.
//  2. The shape we need is much smaller than the full domain entity — just a
//     few columns for the ES projection. Threading a new method through 4
//     existing domain repository interfaces would carry a lot of noise for a
//     single use-site.
//
// All queries are scoped by space_id, ordered by id ASC, and respect any
// soft-delete column the gorm scope auto-applies via DeletedAt (which only
// fires when a *struct* with a DeletedAt field is the target — the raw
// table-name + Scan() form below explicitly adds `deleted_at IS NULL` where
// applicable to mirror that behaviour).

// NewWorkflowDBLister wraps *gorm.DB so it satisfies WorkflowLister.
// Reads from workflow_meta (the single source of truth for a workflow's
// space binding; workflow_draft / workflow_version are per-version blobs).
func NewWorkflowDBLister(db *gorm.DB) WorkflowLister {
	return &workflowDBLister{db: db}
}

type workflowDBLister struct {
	db *gorm.DB
}

func (l *workflowDBLister) ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*WorkflowInfo, error) {
	type row struct {
		ID        int64
		SpaceID   int64
		AppID     int64
		CreatorID int64
		Name      string
		Mode      int32
		Status    int32
		CreatedAt int64
		UpdatedAt int64
	}
	q := l.db.WithContext(ctx).
		Table("workflow_meta").
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []row
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*WorkflowInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, &WorkflowInfo{
			ID:          r.ID,
			SpaceID:     r.SpaceID,
			AppID:       r.AppID,
			OwnerID:     r.CreatorID,
			Name:        r.Name,
			Mode:        r.Mode,
			HasPublish:  r.Status == 1,
			CreatedAtMs: r.CreatedAt,
			UpdatedAtMs: r.UpdatedAt,
		})
	}
	return out, nil
}

// NewPluginDBLister wraps *gorm.DB so it satisfies PluginLister.
// Reads from plugin_draft. Name lives inside the manifest JSON column; the
// resync indexes a stable placeholder if the manifest is unparsable, which
// the live event-bus path would update on next plugin edit.
func NewPluginDBLister(db *gorm.DB) PluginLister {
	return &pluginDBLister{db: db}
}

type pluginDBLister struct {
	db *gorm.DB
}

func (l *pluginDBLister) ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*PluginInfoView, error) {
	type row struct {
		ID          int64
		SpaceID     int64
		DeveloperID int64
		AppID       int64
		PluginType  int32
		Manifest    string
		CreatedAt   int64
		UpdatedAt   int64
	}
	q := l.db.WithContext(ctx).
		Table("plugin_draft").
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []row
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*PluginInfoView, 0, len(rows))
	for _, r := range rows {
		name := extractPluginName(r.Manifest)
		out = append(out, &PluginInfoView{
			ID:          r.ID,
			SpaceID:     r.SpaceID,
			AppID:       r.AppID,
			OwnerID:     r.DeveloperID,
			Name:        name,
			PluginType:  r.PluginType,
			CreatedAtMs: r.CreatedAt,
			UpdatedAtMs: r.UpdatedAt,
		})
	}
	return out, nil
}

// extractPluginName picks a human-readable name from the manifest JSON blob
// without panicking on malformed input. It looks for either "name_for_human"
// or "name_for_model" (the two name fields used by the manifest schema).
// Returns "" if neither key is found / json is broken; the caller will index
// an empty-name doc which is fine for search recall + visible-list purposes.
func extractPluginName(manifest string) string {
	if manifest == "" {
		return ""
	}
	// Minimal forgiving extractor — avoids pulling sonic just for this.
	// Pattern: "name_for_human": "<value>" OR "name_for_model": "<value>"
	for _, key := range []string{`"name_for_human"`, `"name_for_model"`} {
		idx := indexOf(manifest, key)
		if idx < 0 {
			continue
		}
		rest := manifest[idx+len(key):]
		colon := indexOf(rest, `:`)
		if colon < 0 {
			continue
		}
		rest = rest[colon+1:]
		// Skip whitespace
		for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\n') {
			rest = rest[1:]
		}
		if len(rest) == 0 || rest[0] != '"' {
			continue
		}
		rest = rest[1:]
		end := indexOf(rest, `"`)
		if end < 0 {
			continue
		}
		return rest[:end]
	}
	return ""
}

func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// NewPromptDBLister wraps *gorm.DB so it satisfies PromptLister.
// Reads from prompt_resource.
func NewPromptDBLister(db *gorm.DB) PromptLister {
	return &promptDBLister{db: db}
}

type promptDBLister struct {
	db *gorm.DB
}

func (l *promptDBLister) ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*PromptInfo, error) {
	type row struct {
		ID        int64
		SpaceID   int64
		Name      string
		CreatorID int64
		Status    int32
		CreatedAt int64
		UpdatedAt int64
	}
	// prompt_resource has no soft-delete column; Status=1 marks valid rows
	// (Status=0 is "invalid" per the schema comment).
	q := l.db.WithContext(ctx).
		Table("prompt_resource").
		Where("space_id = ? AND status = 1", spaceID).
		Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []row
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*PromptInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, &PromptInfo{
			ID:          r.ID,
			SpaceID:     r.SpaceID,
			OwnerID:     r.CreatorID,
			Name:        r.Name,
			CreatedAtMs: r.CreatedAt,
			UpdatedAtMs: r.UpdatedAt,
		})
	}
	return out, nil
}

// NewDatabaseDBLister wraps *gorm.DB so it satisfies DatabaseLister.
// Reads from draft_database_info (the source of truth for the listing UI;
// online_database_info is the published-snapshot table and IDs there are
// referenced by draft rows via related_online_id).
func NewDatabaseDBLister(db *gorm.DB) DatabaseLister {
	return &databaseDBLister{db: db}
}

type databaseDBLister struct {
	db *gorm.DB
}

func (l *databaseDBLister) ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*DatabaseInfo, error) {
	type row struct {
		ID        int64
		SpaceID   int64
		AppID     int64
		CreatorID int64
		TableName string `gorm:"column:table_name"`
		CreatedAt int64
		UpdatedAt int64
	}
	q := l.db.WithContext(ctx).
		Table("draft_database_info").
		Where("space_id = ? AND deleted_at IS NULL", spaceID).
		Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []row
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*DatabaseInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, &DatabaseInfo{
			ID:          r.ID,
			SpaceID:     r.SpaceID,
			AppID:       r.AppID,
			OwnerID:     r.CreatorID,
			Name:        r.TableName,
			CreatedAtMs: r.CreatedAt,
			UpdatedAtMs: r.UpdatedAt,
		})
	}
	return out, nil
}
