# Space Incremental Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement test-to-production space-level incremental sync with full knowledge base support, upsert logic, and ID mapping persistence.

**Architecture:** Extends the existing `space_exporter/importer` with three new capabilities: (1) knowledge base + folder + external knowledge export/import, (2) upsert-based import via `space_sync_mapping` table, (3) incremental export via `updated_at` filtering. Import executes in three phases: file pre-upload → DB transaction → vector post-processing.

**Tech Stack:** Go, GORM, Hertz HTTP framework, archive/zip, sonic JSON, Thrift IDL

**Spec:** `docs/superpowers/specs/2026-03-24-space-incremental-sync-design.md`

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `docs/ynet-database-sql/04-ynet-sync.sql` | DDL for `space_sync_mapping` and `space_sync_history` tables |
| `backend/application/space/sync/sync_mapping_repo.go` | CRUD operations for `space_sync_mapping` table |
| `backend/application/space/sync/sync_mapping_repo_test.go` | Tests for mapping repo |
| `backend/application/space/sync/sync_history_repo.go` | CRUD operations for `space_sync_history` table |
| `backend/application/space/sync/sync_service.go` | Orchestrator for sync export/import with upsert + 3-phase import |
| `backend/application/space/sync/sync_service_test.go` | Tests for sync service |
| `backend/api/handler/space/space_sync_service.go` | HTTP handlers for sync API endpoints |
| `backend/api/model/space/space_sync.go` | Request/response models for sync API |
| `backend/api/router/space/space_sync.go` | Route registration for `/api/space/{id}/sync/*` |
| `idl/space/space_sync.thrift` | Thrift IDL for sync service |

### Modified Files

| File | Changes |
|------|---------|
| `backend/application/space/export/types.go` | Add `ExportedKnowledge`, `ExportedDocument`, `ExportedSlice`, `ExportedFolder`, `ExportedExternalKnowledge`, `DeletedResources`, `SyncState` types |
| `backend/application/space/export/models.go` | Add DB models for knowledge, document, slice, folder, resource_folder_mapping, external_knowledge_binding |
| `backend/application/space/export/resource_collector.go` | Add `collectKnowledgeBases()`, `collectFolders()`, `collectExternalKnowledge()`, `CollectIncremental()` |
| `backend/application/space/export/serializer.go` | Handle knowledge files in ZIP, write `sync_state.json`, `deleted_resources.json` |
| `backend/application/space/export/space_exporter.go` | Add `ExportSync()` supporting full/incremental modes with file download from object storage |
| `backend/application/space/import/types.go` | Add `KnowledgeIDMap`, `DocumentIDMap`, `FolderIDMap`, `ExternalKnowledgeIDMap`, `SyncMappingStore` to `ImportContext` |
| `backend/application/space/import/id_mapper.go` | Generate IDs for knowledge, document, folder, external knowledge |
| `backend/application/space/import/reference_rewriter.go` | Fix: rewrite knowledge/external knowledge refs instead of nil-ing them; rewrite workflow canvas `knowledge_id` |
| `backend/application/space/import/space_importer.go` | Add `createKnowledge()`, `createFolder()`, `createExternalKnowledge()`; support upsert mode |
| `backend/application/space/import/validator.go` | Support v2.0.0 manifest with knowledge/folder/external knowledge |
| `backend/api/router/register.go` | Add `space.RegisterSync(r)` call |

---

## Task Breakdown

### Task 1: Database DDL + Sync Mapping Repository

**Files:**
- Create: `docs/ynet-database-sql/04-ynet-sync.sql`
- Create: `backend/application/space/sync/sync_mapping_repo.go`
- Create: `backend/application/space/sync/sync_mapping_repo_test.go`
- Create: `backend/application/space/sync/sync_history_repo.go`

- [ ] **Step 1: Create DDL file**

Create `docs/ynet-database-sql/04-ynet-sync.sql`:

```sql
CREATE TABLE IF NOT EXISTS `space_sync_mapping` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_space_id` bigint NOT NULL COMMENT '源空间ID（测试环境）',
  `target_space_id` bigint NOT NULL COMMENT '目标空间ID（生产环境）',
  `resource_type` varchar(32) NOT NULL COMMENT 'agent/plugin/workflow/variable/space_model/knowledge/document/folder/external_knowledge (document is smallest granularity)',
  `source_resource_id` bigint NOT NULL COMMENT '源资源ID',
  `target_resource_id` bigint NOT NULL COMMENT '目标资源ID',
  `source_updated_at` bigint NOT NULL DEFAULT 0 COMMENT '上次同步时源资源的 updated_at',
  `content_hash` varchar(64) DEFAULT NULL COMMENT '预留字段',
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source` (`source_space_id`, `resource_type`, `source_resource_id`),
  KEY `idx_target` (`target_space_id`, `resource_type`, `target_resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='跨环境空间同步ID映射';

CREATE TABLE IF NOT EXISTS `space_sync_history` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_space_id` bigint NOT NULL,
  `target_space_id` bigint NOT NULL,
  `sync_type` varchar(16) NOT NULL COMMENT 'full/incremental',
  `export_time` bigint NOT NULL COMMENT '导出时间戳',
  `import_time` bigint DEFAULT NULL COMMENT '导入时间戳',
  `statistics` json NOT NULL COMMENT '同步统计',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0=exported, 1=imported, 2=failed',
  `error_msg` text DEFAULT NULL,
  `package_file_name` varchar(256) DEFAULT NULL,
  `created_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_source_target` (`source_space_id`, `target_space_id`, `export_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='空间同步历史记录';
```

- [ ] **Step 2: Create SyncMappingRepo**

Create `backend/application/space/sync/sync_mapping_repo.go`:

```go
package sync

import (
    "context"
    "strconv"
    "time"
    "gorm.io/gorm"
)

type SyncMapping struct {
    ID               int64  `gorm:"primaryKey;autoIncrement"`
    SourceSpaceID    int64  `gorm:"column:source_space_id"`
    TargetSpaceID    int64  `gorm:"column:target_space_id"`
    ResourceType     string `gorm:"column:resource_type"`
    SourceResourceID int64  `gorm:"column:source_resource_id"`
    TargetResourceID int64  `gorm:"column:target_resource_id"`
    SourceUpdatedAt  int64  `gorm:"column:source_updated_at"`
    ContentHash      string `gorm:"column:content_hash"`
    CreatedAt        int64  `gorm:"column:created_at"`
    UpdatedAt        int64  `gorm:"column:updated_at"`
}

func (SyncMapping) TableName() string { return "space_sync_mapping" }

// SyncMappingStore provides cached access to sync mappings for a space pair.
// Loaded once at preview time, used throughout import.
type SyncMappingStore struct {
    db            *gorm.DB
    sourceSpaceID int64
    targetSpaceID int64
    mappings      map[string]*SyncMapping // key: "type:sourceID"
}

func NewSyncMappingStore(db *gorm.DB, sourceSpaceID, targetSpaceID int64) *SyncMappingStore {
    return &SyncMappingStore{
        db:            db,
        sourceSpaceID: sourceSpaceID,
        targetSpaceID: targetSpaceID,
        mappings:      make(map[string]*SyncMapping),
    }
}

func mappingKey(resourceType string, sourceID int64) string {
    return resourceType + ":" + strconv.FormatInt(sourceID, 10)
}

// LoadAll loads all existing mappings for the source-target space pair into memory.
func (s *SyncMappingStore) LoadAll(ctx context.Context) error {
    var records []SyncMapping
    err := s.db.WithContext(ctx).
        Where("source_space_id = ? AND target_space_id = ?", s.sourceSpaceID, s.targetSpaceID).
        Find(&records).Error
    if err != nil {
        return err
    }
    for i := range records {
        key := mappingKey(records[i].ResourceType, records[i].SourceResourceID)
        s.mappings[key] = &records[i]
    }
    return nil
}

// GetTargetID returns the target resource ID for a source resource, or 0 if not mapped.
func (s *SyncMappingStore) GetTargetID(resourceType string, sourceID int64) (int64, bool) {
    m, ok := s.mappings[mappingKey(resourceType, sourceID)]
    if !ok {
        return 0, false
    }
    return m.TargetResourceID, true
}

// UpsertMapping creates or updates a mapping record.
// Accepts an optional *gorm.DB to use within a transaction (Phase 2).
// If tx is nil, uses s.db.
func (s *SyncMappingStore) UpsertMapping(ctx context.Context, tx *gorm.DB, resourceType string, sourceID, targetID, sourceUpdatedAt int64) error {
    db := s.db
    if tx != nil {
        db = tx
    }
    now := time.Now().UnixMilli()
    key := mappingKey(resourceType, sourceID)

    if existing, ok := s.mappings[key]; ok {
        existing.SourceUpdatedAt = sourceUpdatedAt
        existing.UpdatedAt = now
        return db.WithContext(ctx).Save(existing).Error
    }

    record := &SyncMapping{
        SourceSpaceID:    s.sourceSpaceID,
        TargetSpaceID:    s.targetSpaceID,
        ResourceType:     resourceType,
        SourceResourceID: sourceID,
        TargetResourceID: targetID,
        SourceUpdatedAt:  sourceUpdatedAt,
        CreatedAt:        now,
        UpdatedAt:        now,
    }
    if err := db.WithContext(ctx).Create(record).Error; err != nil {
        return err
    }
    s.mappings[key] = record
    return nil
}

// RemoveMapping deletes a mapping record.
func (s *SyncMappingStore) RemoveMapping(ctx context.Context, resourceType string, sourceID int64) error {
    key := mappingKey(resourceType, sourceID)
    delete(s.mappings, key)
    return s.db.WithContext(ctx).
        Where("source_space_id = ? AND target_space_id = ? AND resource_type = ? AND source_resource_id = ?",
            s.sourceSpaceID, s.targetSpaceID, resourceType, sourceID).
        Delete(&SyncMapping{}).Error
}
```

- [ ] **Step 3: Create SyncHistoryRepo**

Create `backend/application/space/sync/sync_history_repo.go`:

```go
package sync

import (
    "context"
    "time"
    "gorm.io/gorm"
)

type SyncHistory struct {
    ID              int64  `gorm:"primaryKey;autoIncrement"`
    SourceSpaceID   int64  `gorm:"column:source_space_id"`
    TargetSpaceID   int64  `gorm:"column:target_space_id"`
    SyncType        string `gorm:"column:sync_type"`
    ExportTime      int64  `gorm:"column:export_time"`
    ImportTime      *int64 `gorm:"column:import_time"`
    Statistics      string `gorm:"column:statistics;type:json"`
    Status          int32  `gorm:"column:status"`
    ErrorMsg        string `gorm:"column:error_msg"`
    PackageFileName string `gorm:"column:package_file_name"`
    CreatedAt       int64  `gorm:"column:created_at"`
}

func (SyncHistory) TableName() string { return "space_sync_history" }

type SyncHistoryRepo struct {
    db *gorm.DB
}

func NewSyncHistoryRepo(db *gorm.DB) *SyncHistoryRepo {
    return &SyncHistoryRepo{db: db}
}

func (r *SyncHistoryRepo) Create(ctx context.Context, record *SyncHistory) error {
    record.CreatedAt = time.Now().UnixMilli()
    return r.db.WithContext(ctx).Create(record).Error
}

func (r *SyncHistoryRepo) UpdateStatus(ctx context.Context, id int64, status int32, errMsg string) error {
    now := time.Now().UnixMilli()
    return r.db.WithContext(ctx).Model(&SyncHistory{}).Where("id = ?", id).
        Updates(map[string]interface{}{"status": status, "import_time": now, "error_msg": errMsg}).Error
}

func (r *SyncHistoryRepo) ListBySpace(ctx context.Context, targetSpaceID int64, limit int) ([]SyncHistory, error) {
    var records []SyncHistory
    err := r.db.WithContext(ctx).
        Where("target_space_id = ?", targetSpaceID).
        Order("created_at DESC").Limit(limit).
        Find(&records).Error
    return records, err
}

func (r *SyncHistoryRepo) GetLastExport(ctx context.Context, sourceSpaceID int64) (*SyncHistory, error) {
    var record SyncHistory
    err := r.db.WithContext(ctx).
        Where("source_space_id = ? AND status IN (0, 1)", sourceSpaceID).
        Order("export_time DESC").First(&record).Error
    if err == gorm.ErrRecordNotFound {
        return nil, nil
    }
    return &record, err
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd backend && go build ./application/space/sync/...`

- [ ] **Step 5: Commit**

```
git add docs/ynet-database-sql/04-ynet-sync.sql backend/application/space/sync/
git commit -m "feat(sync): add DDL and sync mapping/history repositories"
```

---

### Task 2: Export Types Extension — Knowledge, Folder, External Knowledge

**Files:**
- Modify: `backend/application/space/export/types.go`
- Modify: `backend/application/space/export/models.go`

- [ ] **Step 1: Add exported types to types.go**

Add after the existing `ExportedSpaceModel` struct:

```go
// ExportedKnowledge represents an exported knowledge base
type ExportedKnowledge struct {
    ID          int64  `json:"id,string"`
    Name        string `json:"name"`
    Description string `json:"description"`
    IconURI     string `json:"icon_uri"`
    FormatType  int32  `json:"format_type"` // 0=Text, 1=Table, 2=Images
    Status      int32  `json:"status"`
    CreatedAt   int64  `json:"created_at"`
    UpdatedAt   int64  `json:"updated_at"`

    Documents []*ExportedDocument `json:"documents"`
}

// ExportedDocument represents an exported knowledge document
type ExportedDocument struct {
    ID            int64       `json:"id,string"`
    KnowledgeID   int64       `json:"knowledge_id,string"`
    Name          string      `json:"name"`
    FileExtension string      `json:"file_extension"`
    DocumentType  int32       `json:"document_type"`
    URI           string      `json:"uri"`
    Size          int64       `json:"size"`
    SliceCount    int64       `json:"slice_count"`
    CharCount     int64       `json:"char_count"`
    SourceType    int32       `json:"source_type"`
    Status        int32       `json:"status"`
    ParseRule     interface{} `json:"parse_rule,omitempty"`
    TableInfo     interface{} `json:"table_info,omitempty"`
    CreatedAt     int64       `json:"created_at"`
    UpdatedAt     int64       `json:"updated_at"`

    // Inline slices
    Slices []*ExportedSlice `json:"slices"`

    // File name within the ZIP (e.g., "12345_document.pdf")
    ExportFileName string `json:"export_file_name,omitempty"`
}

// ExportedSlice represents an exported document slice
type ExportedSlice struct {
    ID         int64   `json:"id,string"`
    DocumentID int64   `json:"document_id,string"`
    Content    string  `json:"content"`
    Sequence   float64 `json:"sequence"`
    Status     int32   `json:"status"`
    CreatedAt  int64   `json:"created_at"`
    UpdatedAt  int64   `json:"updated_at"`
}

// ExportedFolder represents an exported folder
type ExportedFolder struct {
    ID          int64  `json:"id,string"`
    ParentID    int64  `json:"parent_id,string"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    CreatedAt   int64  `json:"created_at"`
    UpdatedAt   int64  `json:"updated_at"`
}

// ExportedFolderMapping represents a resource-to-folder mapping
type ExportedFolderMapping struct {
    ResourceID   int64  `json:"resource_id,string"`
    ResourceType int32  `json:"resource_type"` // 1=agent,2=workflow,3=knowledge,4=database,5=plugin
    FolderID     int64  `json:"folder_id,string"`
}

// ExportedExternalKnowledge represents an exported external knowledge binding
type ExportedExternalKnowledge struct {
    ID          int64       `json:"id,string"`
    BindingKey  string      `json:"binding_key"`
    BindingName string      `json:"binding_name"`
    BindingType int32       `json:"binding_type"`
    ExtraConfig interface{} `json:"extra_config,omitempty"`
    Status      int32       `json:"status"`
    CreatedAt   int64       `json:"created_at"`
    UpdatedAt   int64       `json:"updated_at"`
}

// DeletedResources lists resources deleted since last export (incremental mode)
type DeletedResources struct {
    Agents            []int64 `json:"agents"`
    Plugins           []int64 `json:"plugins"`
    Workflows         []int64 `json:"workflows"`
    Variables         []int64 `json:"variables"`
    SpaceModels       []int64 `json:"space_models"`
    KnowledgeBases    []int64 `json:"knowledge_bases"`
    Documents         []int64 `json:"documents"`
    ExternalKnowledge []int64 `json:"external_knowledge"`
    Folders           []int64 `json:"folders"`
}

// SyncState records export state for incremental sync
type SyncState struct {
    ExportTime    int64 `json:"export_time"`
    SourceSpaceID int64 `json:"source_space_id,string"`
}
```

Update `SpaceResources` to include new fields:

```go
type SpaceResources struct {
    Agents            []*ExportedAgent            `json:"agents"`
    Plugins           []*ExportedPlugin           `json:"plugins"`
    Workflows         []*ExportedWorkflow         `json:"workflows"`
    Variables         []*ExportedVariable         `json:"variables"`
    SpaceModels       []*ExportedSpaceModel       `json:"space_models"`
    KnowledgeBases    []*ExportedKnowledge        `json:"knowledge_bases"`
    Folders           []*ExportedFolder           `json:"folders"`
    FolderMappings    []*ExportedFolderMapping    `json:"folder_mappings"`
    ExternalKnowledge []*ExportedExternalKnowledge `json:"external_knowledge"`
}
```

Update `Manifest` to include sync-related fields:

```go
type Manifest struct {
    Version    string     `json:"version"`
    ExportTime string     `json:"export_time"`
    SyncType   string     `json:"sync_type"` // "full" or "incremental"
    SinceTime  int64      `json:"since_time,omitempty"`
    Source     SourceInfo `json:"source"`
    Statistics Statistics `json:"statistics"`
    IDRegistry IDRegistry `json:"id_registry"`
}
```

Update `Statistics`:

```go
type Statistics struct {
    Agents            int   `json:"agents"`
    Plugins           int   `json:"plugins"`
    Workflows         int   `json:"workflows"`
    Variables         int   `json:"variables"`
    SpaceModels       int   `json:"space_models"`
    KnowledgeBases    int   `json:"knowledge_bases"`
    Documents         int   `json:"documents"`
    FilesTotalSize    int64 `json:"files_total_size"`
    ExternalKnowledge int   `json:"external_knowledge"`
    Folders           int   `json:"folders"`
}
```

Update `IDRegistry`:

```go
type IDRegistry struct {
    Agents            []int64 `json:"agents"`
    Plugins           []int64 `json:"plugins"`
    Workflows         []int64 `json:"workflows"`
    Variables         []int64 `json:"variables"`
    SpaceModels       []int64 `json:"space_models"`
    KnowledgeBases    []int64 `json:"knowledge_bases"`
    Documents         []int64 `json:"documents"`
    ExternalKnowledge []int64 `json:"external_knowledge"`
    Folders           []int64 `json:"folders"`
}
```

Update `ManifestVersion`:

```go
const ManifestVersion = "2.0.0"
```

- [ ] **Step 2: Add DB models to models.go**

Add after existing models:

```go
// KnowledgeModel maps to the knowledge table
type KnowledgeModel struct {
    ID          int64   `gorm:"column:id;primaryKey"`
    Name        string  `gorm:"column:name"`
    AppID       int64   `gorm:"column:app_id"`
    CreatorID   int64   `gorm:"column:creator_id"`
    SpaceID     int64   `gorm:"column:space_id"`
    CreatedAt   int64   `gorm:"column:created_at"`
    UpdatedAt   int64   `gorm:"column:updated_at"`
    Status      int32   `gorm:"column:status"`
    Description *string `gorm:"column:description"`
    IconURI     *string `gorm:"column:icon_uri"`
    FormatType  int32   `gorm:"column:format_type"`
}

func (KnowledgeModel) TableName() string { return "knowledge" }

// KnowledgeDocumentModel maps to the knowledge_document table
type KnowledgeDocumentModel struct {
    ID            int64       `gorm:"column:id;primaryKey"`
    KnowledgeID   int64       `gorm:"column:knowledge_id"`
    Name          string      `gorm:"column:name"`
    FileExtension string      `gorm:"column:file_extension"`
    DocumentType  int32       `gorm:"column:document_type"`
    URI           *string     `gorm:"column:uri"`
    Size          int64       `gorm:"column:size"`
    SliceCount    int64       `gorm:"column:slice_count"`
    CharCount     int64       `gorm:"column:char_count"`
    CreatorID     int64       `gorm:"column:creator_id"`
    SpaceID       int64       `gorm:"column:space_id"`
    CreatedAt     int64       `gorm:"column:created_at"`
    UpdatedAt     int64       `gorm:"column:updated_at"`
    SourceType    int32       `gorm:"column:source_type"`
    Status        int32       `gorm:"column:status"`
    FailReason    *string     `gorm:"column:fail_reason"`
    ParseRule     interface{} `gorm:"column:parse_rule;serializer:json"`
    TableInfo     interface{} `gorm:"column:table_info;serializer:json"`
}

func (KnowledgeDocumentModel) TableName() string { return "knowledge_document" }

// KnowledgeDocumentSliceModel maps to the knowledge_document_slice table
type KnowledgeDocumentSliceModel struct {
    ID          int64   `gorm:"column:id;primaryKey"`
    KnowledgeID int64   `gorm:"column:knowledge_id"`
    DocumentID  int64   `gorm:"column:document_id"`
    Content     *string `gorm:"column:content"`
    Sequence    float64 `gorm:"column:sequence"`
    CreatedAt   int64   `gorm:"column:created_at"`
    UpdatedAt   int64   `gorm:"column:updated_at"`
    CreatorID   int64   `gorm:"column:creator_id"`
    SpaceID     int64   `gorm:"column:space_id"`
    Status      int32   `gorm:"column:status"`
    FailReason  *string `gorm:"column:fail_reason"`
    Hit         int64   `gorm:"column:hit"`
}

func (KnowledgeDocumentSliceModel) TableName() string { return "knowledge_document_slice" }

// FolderModel maps to the folder table
type FolderModel struct {
    ID          int64   `gorm:"column:id;primaryKey"`
    SpaceID     int64   `gorm:"column:space_id"`
    ParentID    int64   `gorm:"column:parent_id"`
    Name        string  `gorm:"column:name"`
    Description *string `gorm:"column:description"`
    CreatorID   int64   `gorm:"column:creator_id"`
    CreatedAt   int64   `gorm:"column:created_at"`
    UpdatedAt   int64   `gorm:"column:updated_at"`
}

func (FolderModel) TableName() string { return "folder" }

// ResourceFolderMappingModel maps to the resource_folder_mapping table
type ResourceFolderMappingModel struct {
    ID           int64 `gorm:"column:id;primaryKey"`
    SpaceID      int64 `gorm:"column:space_id"`
    ResourceID   int64 `gorm:"column:resource_id"`
    ResourceType int32 `gorm:"column:resource_type"`
    FolderID     int64 `gorm:"column:folder_id"`
    CreatedAt    int64 `gorm:"column:created_at"`
    UpdatedAt    int64 `gorm:"column:updated_at"`
}

func (ResourceFolderMappingModel) TableName() string { return "resource_folder_mapping" }

// ExternalKnowledgeBindingModel maps to the external_knowledge_binding table
type ExternalKnowledgeBindingModel struct {
    ID          int64       `gorm:"column:id;primaryKey"`
    UserID      int64       `gorm:"column:user_id"`
    BindingKey  string      `gorm:"column:binding_key"`
    BindingName string      `gorm:"column:binding_name"`
    BindingType int32       `gorm:"column:binding_type"`
    ExtraConfig interface{} `gorm:"column:extra_config;serializer:json"`
    Status      int32       `gorm:"column:status"`
    CreatedAt   int64       `gorm:"column:created_at"`
    UpdatedAt   int64       `gorm:"column:updated_at"`
}

func (ExternalKnowledgeBindingModel) TableName() string { return "external_knowledge_binding" }
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./application/space/export/...`

- [ ] **Step 4: Commit**

```
git add backend/application/space/export/types.go backend/application/space/export/models.go
git commit -m "feat(sync): add export types for knowledge, folder, external knowledge"
```

---

### Task 3: Resource Collector — Knowledge Bases

**Files:**
- Modify: `backend/application/space/export/resource_collector.go`

- [ ] **Step 1: Add collectKnowledgeBases method**

Add to `resource_collector.go`:

```go
// collectKnowledgeBases collects all knowledge bases and their documents/slices from a space
func (c *ResourceCollector) collectKnowledgeBases(ctx context.Context, spaceID int64) ([]*ExportedKnowledge, error) {
    var knowledges []KnowledgeModel
    err := c.db.WithContext(ctx).
        Table("knowledge").
        Where("space_id = ? AND deleted_at IS NULL", spaceID).
        Find(&knowledges).Error
    if err != nil {
        return nil, err
    }

    result := make([]*ExportedKnowledge, 0, len(knowledges))
    for _, k := range knowledges {
        exported := &ExportedKnowledge{
            ID:          k.ID,
            Name:        k.Name,
            Description: getStringValue(k.Description),
            IconURI:     getStringValue(k.IconURI),
            FormatType:  k.FormatType,
            Status:      k.Status,
            CreatedAt:   k.CreatedAt,
            UpdatedAt:   k.UpdatedAt,
        }

        // Collect documents for this knowledge base
        docs, err := c.collectDocuments(ctx, k.ID)
        if err != nil {
            logs.CtxErrorf(ctx, "Failed to collect documents for knowledge %d: %v", k.ID, err)
            return nil, err
        }
        exported.Documents = docs
        result = append(result, exported)
    }

    return result, nil
}

// collectDocuments collects all documents and their slices for a knowledge base
func (c *ResourceCollector) collectDocuments(ctx context.Context, knowledgeID int64) ([]*ExportedDocument, error) {
    var docs []KnowledgeDocumentModel
    err := c.db.WithContext(ctx).
        Table("knowledge_document").
        Where("knowledge_id = ? AND deleted_at IS NULL", knowledgeID).
        Find(&docs).Error
    if err != nil {
        return nil, err
    }

    result := make([]*ExportedDocument, 0, len(docs))
    for _, doc := range docs {
        exported := &ExportedDocument{
            ID:            doc.ID,
            KnowledgeID:   doc.KnowledgeID,
            Name:          doc.Name,
            FileExtension: doc.FileExtension,
            DocumentType:  doc.DocumentType,
            URI:           getStringValue(doc.URI),
            Size:          doc.Size,
            SliceCount:    doc.SliceCount,
            CharCount:     doc.CharCount,
            SourceType:    doc.SourceType,
            Status:        doc.Status,
            ParseRule:     doc.ParseRule,
            TableInfo:     doc.TableInfo,
            CreatedAt:     doc.CreatedAt,
            UpdatedAt:     doc.UpdatedAt,
        }

        // Generate export file name
        exported.ExportFileName = fmt.Sprintf("%d_%s", doc.ID, doc.Name)

        // Collect slices in batches of 100
        slices, err := c.collectSlices(ctx, doc.ID)
        if err != nil {
            logs.CtxErrorf(ctx, "Failed to collect slices for document %d: %v", doc.ID, err)
            return nil, err
        }
        exported.Slices = slices
        result = append(result, exported)
    }

    return result, nil
}

// collectSlices collects all slices for a document in batches
func (c *ResourceCollector) collectSlices(ctx context.Context, documentID int64) ([]*ExportedSlice, error) {
    var allSlices []*ExportedSlice
    batchSize := 100
    offset := 0

    for {
        var slices []KnowledgeDocumentSliceModel
        err := c.db.WithContext(ctx).
            Table("knowledge_document_slice").
            Where("document_id = ? AND deleted_at IS NULL", documentID).
            Order("sequence ASC").
            Offset(offset).Limit(batchSize).
            Find(&slices).Error
        if err != nil {
            return nil, err
        }
        if len(slices) == 0 {
            break
        }

        for _, s := range slices {
            allSlices = append(allSlices, &ExportedSlice{
                ID:         s.ID,
                DocumentID: s.DocumentID,
                Content:    getStringValue(s.Content),
                Sequence:   s.Sequence,
                Status:     s.Status,
                CreatedAt:  s.CreatedAt,
                UpdatedAt:  s.UpdatedAt,
            })
        }

        if len(slices) < batchSize {
            break
        }
        offset += batchSize
    }

    return allSlices, nil
}
```

- [ ] **Step 2: Wire into CollectAll**

In `CollectAll()`, after the space models collection block, add:

```go
    // Collect knowledge bases
    knowledgeBases, err := c.collectKnowledgeBases(ctx, spaceID)
    if err != nil {
        logs.CtxErrorf(ctx, "Failed to collect knowledge bases for space %d: %v", spaceID, err)
        return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("resource", "knowledge_bases"))
    }
    resources.KnowledgeBases = knowledgeBases
    logs.CtxInfof(ctx, "Collected %d knowledge bases from space %d", len(knowledgeBases), spaceID)
```

- [ ] **Step 3: Verify compilation**

Run: `cd backend && go build ./application/space/export/...`

- [ ] **Step 4: Commit**

```
git commit -m "feat(sync): add knowledge base collection to resource collector"
```

---

### Task 4: Resource Collector — Folders & External Knowledge

**Files:**
- Modify: `backend/application/space/export/resource_collector.go`

- [ ] **Step 1: Add collectFolders and collectExternalKnowledge**

```go
// collectFolders collects all folders and resource-folder mappings from a space
func (c *ResourceCollector) collectFolders(ctx context.Context, spaceID int64) ([]*ExportedFolder, []*ExportedFolderMapping, error) {
    var folders []FolderModel
    err := c.db.WithContext(ctx).
        Table("folder").
        Where("space_id = ? AND deleted_at IS NULL", spaceID).
        Find(&folders).Error
    if err != nil {
        return nil, nil, err
    }

    exportedFolders := make([]*ExportedFolder, 0, len(folders))
    for _, f := range folders {
        exportedFolders = append(exportedFolders, &ExportedFolder{
            ID:          f.ID,
            ParentID:    f.ParentID,
            Name:        f.Name,
            Description: getStringValue(f.Description),
            CreatedAt:   f.CreatedAt,
            UpdatedAt:   f.UpdatedAt,
        })
    }

    var mappings []ResourceFolderMappingModel
    err = c.db.WithContext(ctx).
        Table("resource_folder_mapping").
        Where("space_id = ?", spaceID).
        Find(&mappings).Error
    if err != nil {
        return exportedFolders, nil, err
    }

    exportedMappings := make([]*ExportedFolderMapping, 0, len(mappings))
    for _, m := range mappings {
        exportedMappings = append(exportedMappings, &ExportedFolderMapping{
            ResourceID:   m.ResourceID,
            ResourceType: m.ResourceType,
            FolderID:     m.FolderID,
        })
    }

    return exportedFolders, exportedMappings, nil
}

// collectExternalKnowledge collects external knowledge bindings
// Note: external_knowledge_binding is per-user, not per-space.
// We collect bindings referenced by agents in this space.
func (c *ResourceCollector) collectExternalKnowledge(ctx context.Context, agents []*ExportedAgent) ([]*ExportedExternalKnowledge, error) {
    // Collect unique binding IDs from agents' ExternalKnowledge refs
    bindingIDs := make(map[int64]bool)
    for _, agent := range agents {
        if agent.ExternalKnowledge == nil {
            continue
        }
        for _, ek := range agent.ExternalKnowledge.GetExternalKnowledge() {
            if id := ek.GetId(); id != 0 {
                bindingIDs[id] = true
            }
        }
    }

    if len(bindingIDs) == 0 {
        return []*ExportedExternalKnowledge{}, nil
    }

    ids := make([]int64, 0, len(bindingIDs))
    for id := range bindingIDs {
        ids = append(ids, id)
    }

    var bindings []ExternalKnowledgeBindingModel
    err := c.db.WithContext(ctx).
        Table("external_knowledge_binding").
        Where("id IN ?", ids).
        Find(&bindings).Error
    if err != nil {
        return nil, err
    }

    result := make([]*ExportedExternalKnowledge, 0, len(bindings))
    for _, b := range bindings {
        result = append(result, &ExportedExternalKnowledge{
            ID:          b.ID,
            BindingKey:  b.BindingKey,
            BindingName: b.BindingName,
            BindingType: b.BindingType,
            ExtraConfig: b.ExtraConfig,
            Status:      b.Status,
            CreatedAt:   b.CreatedAt,
            UpdatedAt:   b.UpdatedAt,
        })
    }

    return result, nil
}
```

- [ ] **Step 2: Wire into CollectAll**

Add after knowledge bases collection:

```go
    // Collect folders
    folders, folderMappings, err := c.collectFolders(ctx, spaceID)
    if err != nil {
        logs.CtxWarnf(ctx, "Failed to collect folders for space %d: %v", spaceID, err)
        // Non-critical, continue without folders
    } else {
        resources.Folders = folders
        resources.FolderMappings = folderMappings
        logs.CtxInfof(ctx, "Collected %d folders from space %d", len(folders), spaceID)
    }

    // Collect external knowledge bindings
    externalKnowledge, err := c.collectExternalKnowledge(ctx, agents)
    if err != nil {
        logs.CtxWarnf(ctx, "Failed to collect external knowledge: %v", err)
    } else {
        resources.ExternalKnowledge = externalKnowledge
        logs.CtxInfof(ctx, "Collected %d external knowledge bindings", len(externalKnowledge))
    }
```

- [ ] **Step 3: Verify compilation + commit**

```
cd backend && go build ./application/space/export/...
git commit -m "feat(sync): add folder and external knowledge collection"
```

---

### Task 5: Incremental Export — CollectIncremental + Deleted Resources

**Files:**
- Modify: `backend/application/space/export/resource_collector.go`

- [ ] **Step 1: Add CollectIncremental method**

```go
// CollectIncremental collects only resources changed since sinceTime
func (c *ResourceCollector) CollectIncremental(ctx context.Context, spaceID int64, sinceTime int64) (*SpaceResources, *DeletedResources, error) {
    resources := &SpaceResources{
        Agents:            make([]*ExportedAgent, 0),
        Plugins:           make([]*ExportedPlugin, 0),
        Workflows:         make([]*ExportedWorkflow, 0),
        Variables:         make([]*ExportedVariable, 0),
        SpaceModels:       make([]*ExportedSpaceModel, 0),
        KnowledgeBases:    make([]*ExportedKnowledge, 0),
        Folders:           make([]*ExportedFolder, 0),
        FolderMappings:    make([]*ExportedFolderMapping, 0),
        ExternalKnowledge: make([]*ExportedExternalKnowledge, 0),
    }

    // Collect changed agents
    var agentDrafts []SingleAgentDraftModel
    err := c.db.WithContext(ctx).Table("single_agent_draft").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Find(&agentDrafts).Error
    if err != nil {
        return nil, nil, err
    }
    agentIDs := make([]int64, 0)
    for _, a := range agentDrafts {
        agentIDs = append(agentIDs, a.AgentID)
    }
    agentToolsMap, _ := c.collectAgentTools(ctx, agentIDs)
    for _, a := range agentDrafts {
        exported := c.convertAgentDraftToExported(&a)
        if tools, ok := agentToolsMap[a.AgentID]; ok {
            exported.AgentTools = tools
        }
        resources.Agents = append(resources.Agents, exported)
    }

    // Collect changed plugins
    var pluginDrafts []PluginDraftModel
    c.db.WithContext(ctx).Table("plugin_draft").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Find(&pluginDrafts)
    for _, p := range pluginDrafts {
        resources.Plugins = append(resources.Plugins, c.convertPluginDraftToExported(&p))
    }

    // Collect changed workflows
    var workflowMetas []WorkflowMetaModel
    c.db.WithContext(ctx).Table("workflow_meta").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Find(&workflowMetas)
    wfIDs := make([]int64, 0)
    for _, m := range workflowMetas {
        wfIDs = append(wfIDs, m.ID)
    }
    if len(wfIDs) > 0 {
        var wfDrafts []WorkflowDraftModel
        c.db.WithContext(ctx).Table("workflow_draft").Where("id IN ?", wfIDs).Find(&wfDrafts)
        draftMap := make(map[int64]*WorkflowDraftModel)
        for i := range wfDrafts {
            draftMap[wfDrafts[i].ID] = &wfDrafts[i]
        }
        for _, m := range workflowMetas {
            resources.Workflows = append(resources.Workflows, c.convertWorkflowToExported(&m, draftMap[m.ID]))
        }
    }

    // Collect changed variables directly (not just for changed agents)
    var changedVars []VariablesMetaModel
    c.db.WithContext(ctx).Table("variables_meta").
        Where("biz_type = 1 AND updated_at > ?", sinceTime).
        Find(&changedVars)
    for _, v := range changedVars {
        // Filter to variables belonging to agents in this space
        resources.Variables = append(resources.Variables, c.convertVariableToExported(&v))
    }

    // Collect changed space models
    var spaceModels []SpaceModelModel
    c.db.WithContext(ctx).Table("space_model").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Find(&spaceModels)
    for _, sm := range spaceModels {
        resources.SpaceModels = append(resources.SpaceModels, c.convertSpaceModelToExported(&sm))
    }

    // Collect changed knowledge bases (document-level granularity)
    var changedKnowledges []KnowledgeModel
    c.db.WithContext(ctx).Table("knowledge").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Find(&changedKnowledges)

    // Also check for knowledge bases with changed documents
    var docChangedKnowledgeIDs []int64
    c.db.WithContext(ctx).Table("knowledge_document").
        Select("DISTINCT knowledge_id").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Pluck("knowledge_id", &docChangedKnowledgeIDs)

    // Merge knowledge IDs
    knowledgeIDSet := make(map[int64]bool)
    for _, k := range changedKnowledges {
        knowledgeIDSet[k.ID] = true
    }
    for _, id := range docChangedKnowledgeIDs {
        knowledgeIDSet[id] = true
    }

    // Collect each changed knowledge base individually (not N+1 — single query per KB)
    if len(knowledgeIDSet) > 0 {
        changedIDs := make([]int64, 0, len(knowledgeIDSet))
        for id := range knowledgeIDSet {
            changedIDs = append(changedIDs, id)
        }
        var kbModels []KnowledgeModel
        c.db.WithContext(ctx).Table("knowledge").
            Where("id IN ? AND deleted_at IS NULL", changedIDs).
            Find(&kbModels)
        for _, k := range kbModels {
            exported := &ExportedKnowledge{
                ID: k.ID, Name: k.Name, Description: getStringValue(k.Description),
                IconURI: getStringValue(k.IconURI), FormatType: k.FormatType,
                Status: k.Status, CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt,
            }
            docs, err := c.collectDocuments(ctx, k.ID)
            if err != nil {
                logs.CtxWarnf(ctx, "Failed to collect documents for knowledge %d: %v", k.ID, err)
                continue
            }
            exported.Documents = docs
            resources.KnowledgeBases = append(resources.KnowledgeBases, exported)
        }
    }

    // Collect deleted resources
    deleted := c.collectDeletedResources(ctx, spaceID, sinceTime)

    // Collect changed folders (filter by updated_at in incremental mode)
    var changedFolders []FolderModel
    c.db.WithContext(ctx).Table("folder").
        Where("space_id = ? AND updated_at > ? AND deleted_at IS NULL", spaceID, sinceTime).
        Find(&changedFolders)
    for _, f := range changedFolders {
        resources.Folders = append(resources.Folders, &ExportedFolder{
            ID: f.ID, ParentID: f.ParentID, Name: f.Name,
            Description: getStringValue(f.Description), CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
        })
    }
    // Always export full folder mappings (small table, needed for correctness)
    var mappings []ResourceFolderMappingModel
    c.db.WithContext(ctx).Table("resource_folder_mapping").Where("space_id = ?", spaceID).Find(&mappings)
    for _, m := range mappings {
        resources.FolderMappings = append(resources.FolderMappings, &ExportedFolderMapping{
            ResourceID: m.ResourceID, ResourceType: m.ResourceType, FolderID: m.FolderID,
        })
    }

    return resources, deleted, nil
}

// collectDeletedResources finds resources soft-deleted since sinceTime
// Uses Unscoped() to bypass GORM's automatic deleted_at IS NULL filter
// Converts deleted_at (DATETIME) to milliseconds for comparison with sinceTime
func (c *ResourceCollector) collectDeletedResources(ctx context.Context, spaceID int64, sinceTime int64) *DeletedResources {
    deleted := &DeletedResources{}

    collectDeleted := func(table, idCol string) []int64 {
        var ids []int64
        c.db.WithContext(ctx).Unscoped().Table(table).
            Select(idCol).
            Where("space_id = ? AND deleted_at IS NOT NULL AND UNIX_TIMESTAMP(deleted_at) * 1000 > ?", spaceID, sinceTime).
            Pluck(idCol, &ids)
        return ids
    }

    deleted.Agents = collectDeleted("single_agent_draft", "agent_id")
    deleted.Plugins = collectDeleted("plugin_draft", "id")
    deleted.Workflows = collectDeleted("workflow_meta", "id")
    deleted.Variables = collectDeletedVariables(ctx, c.db, spaceID, sinceTime)
    deleted.SpaceModels = collectDeleted("space_model", "id")
    deleted.KnowledgeBases = collectDeleted("knowledge", "id")
    deleted.Documents = collectDeleted("knowledge_document", "id")
    deleted.Folders = collectDeleted("folder", "id")

    return deleted
}
```

- [ ] **Step 2: Verify compilation + commit**

```
cd backend && go build ./application/space/export/...
git commit -m "feat(sync): add incremental export and deleted resource collection"
```

---

### Task 6: Serializer Extension — Knowledge Files + Sync State

**Files:**
- Modify: `backend/application/space/export/serializer.go`

- [ ] **Step 1: Add SerializeToSyncZip method and helpers**

Add a new top-level serialize method that wraps the existing logic and adds knowledge/folder/sync support:

```go
// SerializeToSyncZip creates a ZIP containing all resources + knowledge files + sync metadata
func (s *Serializer) SerializeToSyncZip(ctx context.Context, manifest *Manifest, resources *SpaceResources,
    fileContents map[int64][]byte, syncState *SyncState, deleted *DeletedResources) ([]byte, error) {

    buf := new(bytes.Buffer)
    w := zip.NewWriter(buf)
    defer w.Close()

    // Existing resources (reuse existing addResourceToZip patterns)
    s.addManifestToZip(w, manifest)
    s.addResourcesToZip(w, "agents", resources.Agents)
    s.addResourcesToZip(w, "plugins", resources.Plugins)
    s.addResourcesToZip(w, "workflows", resources.Workflows)
    s.addResourcesToZip(w, "variables", resources.Variables)
    s.addResourcesToZip(w, "space_models", resources.SpaceModels)

    // Knowledge bases with nested structure
    s.addKnowledgeBasesToZip(w, resources.KnowledgeBases, fileContents)

    // Folders
    s.addFoldersToZip(w, resources.Folders, resources.FolderMappings)

    // External knowledge
    s.addResourcesToZip(w, "external_knowledge", resources.ExternalKnowledge)

    // Sync metadata
    s.addJSONToZip(w, "sync_state.json", syncState)
    if deleted != nil {
        s.addJSONToZip(w, "deleted_resources.json", deleted)
    }

    w.Close()
    return buf.Bytes(), nil
}

// addKnowledgeBasesToZip writes nested knowledge_bases/ directory structure
func (s *Serializer) addKnowledgeBasesToZip(w *zip.Writer, kbs []*ExportedKnowledge, fileContents map[int64][]byte) error {
    // Write index
    s.addJSONToZip(w, "knowledge_bases/index.json", kbs)

    for _, kb := range kbs {
        prefix := fmt.Sprintf("knowledge_bases/%d/", kb.ID)

        // Write meta.json (knowledge metadata without documents)
        meta := *kb
        meta.Documents = nil
        s.addJSONToZip(w, prefix+"meta.json", &meta)

        // Write documents with inline slices
        s.addJSONToZip(w, prefix+"documents/index.json", kb.Documents)
        for _, doc := range kb.Documents {
            s.addJSONToZip(w, fmt.Sprintf("%sdocuments/%d.json", prefix, doc.ID), doc)

            // Write original file if available
            if content, ok := fileContents[doc.ID]; ok && len(content) > 0 {
                f, _ := w.Create(fmt.Sprintf("%sfiles/%s", prefix, doc.ExportFileName))
                f.Write(content)
            }
        }
    }
    return nil
}

// addFoldersToZip writes folders/ directory
func (s *Serializer) addFoldersToZip(w *zip.Writer, folders []*ExportedFolder, mappings []*ExportedFolderMapping) error {
    s.addJSONToZip(w, "folders/index.json", folders)
    s.addJSONToZip(w, "folders/resource_mappings.json", mappings)
    return nil
}

// addJSONToZip writes a JSON file into the ZIP
func (s *Serializer) addJSONToZip(w *zip.Writer, path string, v interface{}) error {
    data, err := sonic.Marshal(v)
    if err != nil {
        return err
    }
    f, err := w.Create(path)
    if err != nil {
        return err
    }
    _, err = f.Write(data)
    return err
}
```

Also update `BuildManifest()` → add `BuildSyncManifest()`:

```go
func (s *Serializer) BuildSyncManifest(spaceID int64, mode string, sinceTime int64, resources *SpaceResources) *Manifest {
    // Count documents and file sizes
    var docCount int
    var filesSize int64
    for _, kb := range resources.KnowledgeBases {
        docCount += len(kb.Documents)
    }

    return &Manifest{
        Version:    ManifestVersion, // "2.0.0"
        ExportTime: time.Now().Format(time.RFC3339),
        SyncType:   mode,
        SinceTime:  sinceTime,
        Source:     SourceInfo{SpaceID: spaceID},
        Statistics: Statistics{
            Agents:         len(resources.Agents),
            Plugins:        len(resources.Plugins),
            Workflows:      len(resources.Workflows),
            Variables:      len(resources.Variables),
            SpaceModels:    len(resources.SpaceModels),
            KnowledgeBases: len(resources.KnowledgeBases),
            Documents:      docCount,
            FilesTotalSize: filesSize,
            ExternalKnowledge: len(resources.ExternalKnowledge),
            Folders:        len(resources.Folders),
        },
        IDRegistry: buildIDRegistry(resources),
    }
}
```

- [ ] **Step 2: Verify compilation + commit**

```
cd backend && go build ./application/space/export/...
git commit -m "feat(sync): extend serializer for knowledge files and sync state"
```

---

### Task 7: Space Exporter — Sync Export Mode

**Files:**
- Modify: `backend/application/space/export/space_exporter.go`

- [ ] **Step 1: Add ExportSync method**

Add a new method that supports full/incremental modes and downloads knowledge base files from object storage:

```go
type SyncExportRequest struct {
    SpaceID   int64
    UserID    int64
    Mode      string // "full" or "incremental"
    SinceTime int64  // only used when Mode == "incremental"
}

func (e *SpaceExporter) ExportSync(ctx context.Context, req *SyncExportRequest) (*ExportResult, error) {
    var resources *SpaceResources
    var deleted *DeletedResources
    var err error

    if req.Mode == "incremental" && req.SinceTime > 0 {
        resources, deleted, err = e.collector.CollectIncremental(ctx, req.SpaceID, req.SinceTime)
    } else {
        resources, err = e.collector.CollectAll(ctx, req.SpaceID)
        req.Mode = "full"
    }
    if err != nil {
        return nil, err
    }

    // Download knowledge base original files from object storage
    fileContents := make(map[int64][]byte)
    for _, kb := range resources.KnowledgeBases {
        for _, doc := range kb.Documents {
            if doc.URI != "" {
                content, err := e.objectStorage.GetObject(ctx, doc.URI)
                if err != nil {
                    logs.CtxWarnf(ctx, "Failed to download file for document %d: %v", doc.ID, err)
                    continue
                }
                fileContents[doc.ID] = content
            }
        }
    }

    // Build manifest with sync info
    manifest := e.serializer.BuildSyncManifest(req.SpaceID, req.Mode, req.SinceTime, resources)

    // Serialize to ZIP with knowledge files
    syncState := &SyncState{
        ExportTime:    time.Now().UnixMilli(),
        SourceSpaceID: req.SpaceID,
    }
    zipContent, err := e.serializer.SerializeToSyncZip(ctx, manifest, resources, fileContents, syncState, deleted)
    if err != nil {
        return nil, err
    }

    // Upload and return URL (same as existing Export)
    // ... (reuse existing upload logic)
}
```

- [ ] **Step 2: Verify compilation + commit**

```
cd backend && go build ./application/space/export/...
git commit -m "feat(sync): add sync export mode with file download"
```

---

### Task 8: Import Context Extension + Reference Rewriter Fix

**Files:**
- Modify: `backend/application/space/import/types.go`
- Modify: `backend/application/space/import/id_mapper.go`
- Modify: `backend/application/space/import/reference_rewriter.go`

- [ ] **Step 1: Extend ImportContext in types.go**

Add new fields to `ImportContext`:

```go
    // New ID mappings
    KnowledgeIDMap         map[int64]int64
    DocumentIDMap          map[int64]int64
    FolderIDMap            map[int64]int64
    ExternalKnowledgeIDMap map[int64]int64

    // Sync mode
    SyncMode     string                         // "create_only" | "upsert"
    MappingStore *spacesync.SyncMappingStore     // nil for create_only mode
```

Add `IsInPackageKnowledge`, `RemapKnowledgeID`, `IsInPackageDocument`, `RemapDocumentID` methods following the existing pattern.

- [ ] **Step 2: Extend IDMapper in id_mapper.go**

Add ID generation for knowledge bases, documents, folders, and external knowledge in `GenerateMapping()`, following the existing pattern for agents/plugins/workflows.

- [ ] **Step 3: Fix reference_rewriter.go — Knowledge refs**

In `RewriteAgent()`, replace:

```go
// Line 112-113: Clear knowledge references (not exported)
agent.KnowledgeRefs = nil
```

With:

```go
// Rewrite knowledge references
if agent.KnowledgeRefs != nil {
    knowledgeList := agent.KnowledgeRefs.GetKnowledge()
    if len(knowledgeList) > 0 {
        for _, kRef := range knowledgeList {
            oldID := kRef.GetDatasetId()
            if importCtx.IsInPackageKnowledge(oldID) {
                newID := importCtx.RemapKnowledgeID(oldID)
                kRef.DatasetId = &newID
            }
        }
    }
}
```

Replace:
```go
// Line 116-117: Clear external knowledge references
agent.ExternalKnowledge = nil
```

With:
```go
// Rewrite external knowledge references
if agent.ExternalKnowledge != nil {
    for _, ek := range agent.ExternalKnowledge.GetExternalKnowledge() {
        oldID := ek.GetId()
        if importCtx.IsInPackageExternalKnowledge(oldID) {
            newID := importCtx.RemapExternalKnowledgeID(oldID)
            ek.Id = &newID
        }
    }
}
```

Keep `agent.DatabaseRefs = nil` as-is (database connections are environment-specific).

- [ ] **Step 4: Fix reference_rewriter.go — Workflow canvas knowledge_id**

In `rewriteNodeReferences()`, replace:

```go
// Line 269-271: Clear knowledge_id
if _, ok := data["knowledge_id"]; ok {
    data["knowledge_id"] = 0
}
```

With:

```go
// Rewrite knowledge_id
if knowledgeID, ok := getInt64FromInterface(data["knowledge_id"]); ok && knowledgeID != 0 {
    if importCtx.IsInPackageKnowledge(knowledgeID) {
        data["knowledge_id"] = importCtx.RemapKnowledgeID(knowledgeID)
    } else {
        data["knowledge_id"] = 0
    }
}
```

- [ ] **Step 5: Verify compilation + commit**

```
cd backend && go build ./application/space/import/...
git commit -m "fix(sync): rewrite knowledge refs instead of clearing them"
```

---

### Task 9: Validator Extension — v2.0.0 Manifest

**Files:**
- Modify: `backend/application/space/import/validator.go`

- [ ] **Step 1: Update validator to handle v2.0.0 manifest**

The validator needs to:
- Accept both "1.0.0" and "2.0.0" manifest versions
- Parse knowledge bases, folders, and external knowledge from the ZIP
- Parse `deleted_resources.json` and `sync_state.json` if present
- Validate knowledge file existence in ZIP

Follow the existing pattern in `ValidateAndParse()` for parsing resource directories.

- [ ] **Step 2: Verify compilation + commit**

```
cd backend && go build ./application/space/import/...
git commit -m "feat(sync): extend validator for v2.0.0 manifest"
```

---

### Task 10: Space Importer — Knowledge Import + Upsert Logic

**Files:**
- Modify: `backend/application/space/import/space_importer.go`

This is the largest task. It adds:
1. Knowledge base creation (DB records)
2. Folder creation
3. External knowledge creation
4. Upsert logic for all resource types

- [ ] **Step 1: Add createKnowledge method**

```go
// createKnowledge creates a knowledge base with its documents and slices in the database
// Note: vector store operations happen in Phase 3 (post-transaction)
func (s *SpaceImporter) createKnowledge(ctx context.Context, tx *gorm.DB, kb *export.ExportedKnowledge, importCtx *ImportContext) error {
    newKnowledgeID := importCtx.KnowledgeIDMap[kb.ID]
    now := time.Now().UnixMilli()

    // Create knowledge record
    knowledgeModel := map[string]interface{}{
        "id":          newKnowledgeID,
        "name":        kb.Name,
        "app_id":      0,
        "creator_id":  importCtx.UserID,
        "space_id":    importCtx.TargetSpaceID,
        "created_at":  now,
        "updated_at":  now,
        "status":      1, // effective
        "description": kb.Description,
        "icon_uri":    kb.IconURI,
        "format_type": kb.FormatType,
    }
    if err := tx.Table("knowledge").Create(knowledgeModel).Error; err != nil {
        return err
    }

    // Create documents and slices
    for _, doc := range kb.Documents {
        if err := s.createDocument(ctx, tx, doc, newKnowledgeID, importCtx); err != nil {
            return err
        }
    }

    return nil
}

func (s *SpaceImporter) createDocument(ctx context.Context, tx *gorm.DB, doc *export.ExportedDocument, knowledgeID int64, importCtx *ImportContext) error {
    newDocID := importCtx.DocumentIDMap[doc.ID]
    now := time.Now().UnixMilli()

    // URI will be set to the new object storage path after file upload (Phase 1)
    newURI := importCtx.FileURIMap[doc.ID] // Set during Phase 1

    docModel := map[string]interface{}{
        "id":             newDocID,
        "knowledge_id":   knowledgeID,
        "name":           doc.Name,
        "file_extension": doc.FileExtension,
        "document_type":  doc.DocumentType,
        "uri":            newURI,
        "size":           doc.Size,
        "slice_count":    doc.SliceCount,
        "char_count":     doc.CharCount,
        "creator_id":     importCtx.UserID,
        "space_id":       importCtx.TargetSpaceID,
        "created_at":     now,
        "updated_at":     now,
        "source_type":    doc.SourceType,
        "status":         1, // enable
        "parse_rule":     toJSON(doc.ParseRule),
        "table_info":     toJSON(doc.TableInfo),
    }
    if err := tx.Table("knowledge_document").Create(docModel).Error; err != nil {
        return err
    }

    // Create slices in batches
    for i := 0; i < len(doc.Slices); i += 100 {
        end := i + 100
        if end > len(doc.Slices) {
            end = len(doc.Slices)
        }
        batch := doc.Slices[i:end]
        for _, slice := range batch {
            newSliceID, _ := s.idGen.GenID(ctx)
            sliceModel := map[string]interface{}{
                "id":           newSliceID,
                "knowledge_id": knowledgeID,
                "document_id":  newDocID,
                "content":      slice.Content,
                "sequence":     slice.Sequence,
                "created_at":   now,
                "updated_at":   now,
                "creator_id":   importCtx.UserID,
                "space_id":     importCtx.TargetSpaceID,
                "status":       1, // done
            }
            if err := tx.Table("knowledge_document_slice").Create(sliceModel).Error; err != nil {
                return err
            }
        }
    }

    return nil
}
```

- [ ] **Step 2: Add createFolder and createExternalKnowledge methods**

```go
func (s *SpaceImporter) createFolder(ctx context.Context, tx *gorm.DB, folder *export.ExportedFolder, importCtx *ImportContext) error {
    newID := importCtx.FolderIDMap[folder.ID]
    now := time.Now().UnixMilli()

    // Remap parent_id if it's a child folder
    parentID := folder.ParentID
    if parentID != 0 {
        if newParentID, ok := importCtx.FolderIDMap[parentID]; ok {
            parentID = newParentID
        }
    }

    model := map[string]interface{}{
        "id":          newID,
        "space_id":    importCtx.TargetSpaceID,
        "parent_id":   parentID,
        "name":        folder.Name,
        "description": folder.Description,
        "creator_id":  importCtx.UserID,
        "created_at":  now,
        "updated_at":  now,
    }
    return tx.Table("folder").Create(model).Error
}
```

- [ ] **Step 3: Add upsert wrapper logic**

For each resource type, add an `upsertXxx` method that wraps the existing `createXxx`:

```go
func (s *SpaceImporter) upsertResource(ctx context.Context, tx *gorm.DB, resourceType string, sourceID int64,
    importCtx *ImportContext, createFn func() error, updateFn func(targetID int64) error) error {

    if importCtx.MappingStore == nil {
        // create_only mode
        return createFn()
    }

    targetID, exists := importCtx.MappingStore.GetTargetID(resourceType, sourceID)
    if exists {
        // UPDATE path
        return updateFn(targetID)
    }
    // CREATE path
    return createFn()
}
```

Then add specific update methods for each resource type:

```go
// updateAgent updates an existing agent during upsert import
func (s *SpaceImporter) updateAgent(ctx context.Context, tx *gorm.DB, agent *export.ExportedAgent, targetID int64, importCtx *ImportContext) error {
    now := time.Now().UnixMilli()
    rewritten := s.rewriter.RewriteAgent(ctx, agent, importCtx)

    updates := map[string]interface{}{
        "name":                       rewritten.Name,
        "description":                rewritten.Desc,
        "icon_uri":                   rewritten.IconURI,
        "bot_mode":                   rewritten.BotMode,
        "onboarding_info":            toJSON(rewritten.OnboardingInfo),
        "model_info":                 toJSON(rewritten.ModelInfo),
        "prompt":                     toJSON(rewritten.Prompt),
        "plugin":                     toJSON(rewritten.PluginRefs),
        "knowledge":                  toJSON(rewritten.KnowledgeRefs),
        "workflow":                   toJSON(rewritten.WorkflowRefs),
        "suggest_reply":              toJSON(rewritten.SuggestReply),
        "shortcut_command":           toJSON(rewritten.ShortcutCommand),
        "memory_tool_config":         toJSON(rewritten.MemoryToolConfig),
        "background_image_info_list": toJSON(rewritten.BackgroundImageList),
        "layout_info":                toJSON(rewritten.LayoutInfo),
        "updated_at":                 now,
    }
    if err := tx.Table("single_agent_draft").Where("agent_id = ?", targetID).Updates(updates).Error; err != nil {
        return err
    }

    // Delete old tools, recreate new ones
    tx.Table("agent_tool_draft").Where("agent_id = ?", targetID).Delete(&struct{}{})
    for _, tool := range rewritten.AgentTools {
        // ... create tool (same logic as createAgent)
    }
    return nil
}

// updatePlugin updates an existing plugin during upsert import
func (s *SpaceImporter) updatePlugin(ctx context.Context, tx *gorm.DB, plugin *export.ExportedPlugin, targetID int64, importCtx *ImportContext) error {
    now := time.Now().UnixMilli()
    updates := map[string]interface{}{
        "icon_uri":    plugin.IconURI,
        "server_url":  plugin.ServerURL,
        "plugin_type": plugin.PluginType,
        "manifest":    toJSON(plugin.Manifest),
        "openapi_doc": toJSON(plugin.OpenapiDoc),
        "updated_at":  now,
    }
    // Update both plugin and plugin_draft tables
    tx.Table("plugin").Where("id = ?", targetID).Updates(updates)
    return tx.Table("plugin_draft").Where("id = ?", targetID).Updates(updates).Error
}

// updateWorkflow updates an existing workflow during upsert import
func (s *SpaceImporter) updateWorkflow(ctx context.Context, tx *gorm.DB, workflow *export.ExportedWorkflow, targetID int64, importCtx *ImportContext) error {
    now := time.Now().UnixMilli()
    rewritten := s.rewriter.RewriteWorkflow(ctx, workflow, importCtx)

    // Update workflow_meta
    tx.Table("workflow_meta").Where("id = ?", targetID).Updates(map[string]interface{}{
        "name": rewritten.Name, "description": rewritten.Desc,
        "icon_uri": rewritten.IconURI, "updated_at": now,
    })

    // Update workflow_draft
    return tx.Table("workflow_draft").Where("id = ?", targetID).Updates(map[string]interface{}{
        "canvas": toJSON(rewritten.Canvas), "input_params": toJSON(rewritten.InputParams),
        "output_params": toJSON(rewritten.OutputParams), "updated_at": now,
    }).Error
}

// updateKnowledge updates an existing knowledge base during upsert import
// Only updates DB records; vector rebuild happens in Phase 3
func (s *SpaceImporter) updateKnowledge(ctx context.Context, tx *gorm.DB, kb *export.ExportedKnowledge, targetKnowledgeID int64, importCtx *ImportContext) error {
    now := time.Now().UnixMilli()
    tx.Table("knowledge").Where("id = ?", targetKnowledgeID).Updates(map[string]interface{}{
        "name": kb.Name, "description": kb.Description, "icon_uri": kb.IconURI, "updated_at": now,
    })

    for _, doc := range kb.Documents {
        targetDocID, exists := importCtx.MappingStore.GetTargetID("document", doc.ID)
        if exists {
            // Update existing document: replace URI, delete old slices, create new slices
            newURI := importCtx.FileURIMap[doc.ID]
            tx.Table("knowledge_document").Where("id = ?", targetDocID).Updates(map[string]interface{}{
                "name": doc.Name, "uri": newURI, "size": doc.Size,
                "slice_count": doc.SliceCount, "char_count": doc.CharCount, "updated_at": now,
            })
            // Delete old slices
            tx.Table("knowledge_document_slice").Where("document_id = ?", targetDocID).Delete(&struct{}{})
            // Create new slices (same logic as createDocument)
            for _, slice := range doc.Slices {
                newSliceID, _ := s.idGen.GenID(ctx)
                tx.Table("knowledge_document_slice").Create(map[string]interface{}{
                    "id": newSliceID, "knowledge_id": targetKnowledgeID, "document_id": targetDocID,
                    "content": slice.Content, "sequence": slice.Sequence,
                    "created_at": now, "updated_at": now,
                    "creator_id": importCtx.UserID, "space_id": importCtx.TargetSpaceID, "status": 1,
                })
            }
        } else {
            // New document in existing knowledge base
            s.createDocument(ctx, tx, doc, targetKnowledgeID, importCtx)
        }
    }
    return nil
}
```

- [ ] **Step 4: Wire knowledge/folder/external knowledge into executeImport**

In `executeImport()`, add steps between workflow and agent creation:

```go
    // Step 4: Create knowledge bases (DB records only, vectors in Phase 3)
    for _, kb := range pending.Resources.KnowledgeBases {
        // ... upsert logic
    }

    // Step 5: Create external knowledge bindings
    // Step 6: Agents (with knowledge refs now rewritten)
    // Step 7: Variables
    // Step 8: Folders + resource mappings
```

- [ ] **Step 5: Verify compilation + commit**

```
cd backend && go build ./application/space/import/...
git commit -m "feat(sync): add knowledge import and upsert logic"
```

---

### Task 11: Sync Service — Orchestrator with 3-Phase Import

**Files:**
- Create: `backend/application/space/sync/sync_service.go`

- [ ] **Step 1: Create SyncService**

This is the top-level orchestrator that coordinates the three phases:

```go
package sync

type SyncService struct {
    db              *gorm.DB
    exporter        *spaceexport.SpaceExporter
    importer        *spaceimport.SpaceImporter
    objectStorage   storage.Storage
    knowledgeSVC    knowledge.Knowledge  // For vector store operations
    idGen           idgen.IDGenerator
    mappingStore    *SyncMappingStore
    historyRepo     *SyncHistoryRepo
    eventBus        service.ResourceEventBus
    projectEventBus service.ProjectEventBus
}

// ExportSync handles sync export (full or incremental)
func (s *SyncService) ExportSync(ctx context.Context, req *SyncExportRequest) (*SyncExportResponse, error)

// ImportPreview validates the package, stores ZIP to temp object storage, builds import plan.
// ZIP storage: uploaded file is stored to object storage with key "sync_imports/{token}.zip"
// and 30-minute TTL. Only the lightweight token + manifest is kept in memory.
func (s *SyncService) ImportPreview(ctx context.Context, req *SyncImportPreviewRequest) (*SyncImportPreviewResponse, error) {
    // 1. Validate and parse ZIP
    // 2. Store ZIP to object storage: objectStorage.PutObject(ctx, "sync_imports/"+token+".zip", fileContent)
    // 3. Check package size limit (max 2GB)
    // 4. Check target space has embedding model configured (space_embedding table)
    //    → If no embedding model AND package contains knowledge bases → return error
    // 5. Load SyncMappingStore for source-target space pair
    // 6. Build import plan (create/update/delete counts)
    // 7. Store lightweight PendingImport{Token, SpaceID, Manifest, TempFileKey} in memory
    // 8. Return plan + warnings
}

// ImportConfirm reads ZIP from object storage and executes the 3-phase import
func (s *SyncService) ImportConfirm(ctx context.Context, req *SyncImportConfirmRequest) (*SyncImportConfirmResponse, error) {
    // 1. Get pending import by token
    // 2. Read ZIP from object storage: objectStorage.GetObject(ctx, pending.TempFileKey)
    // 3. Execute 3-phase import
    // 4. Delete temp ZIP from object storage
}

// GetLastExport returns the last export timestamp for a space
func (s *SyncService) GetLastExport(ctx context.Context, spaceID int64) (*LastExportResponse, error)

// GetHistory returns sync history for a space
func (s *SyncService) GetHistory(ctx context.Context, spaceID int64) ([]SyncHistory, error)
```

The `ImportConfirm` method implements the 3-phase approach:

```go
func (s *SyncService) ImportConfirm(ctx context.Context, req *SyncImportConfirmRequest) (*SyncImportConfirmResponse, error) {
    // Phase 1: Upload files to object storage
    fileURIMap, err := s.uploadKnowledgeFiles(ctx, pending)
    if err != nil {
        s.cleanupUploadedFiles(ctx, fileURIMap)
        return nil, err
    }

    // Phase 2: DB transaction
    result, err := s.executeDBTransaction(ctx, pending, fileURIMap)
    if err != nil {
        s.cleanupUploadedFiles(ctx, fileURIMap)
        return nil, err
    }

    // Phase 3: Post-transaction processing (vector store + ES sync)
    s.rebuildVectorIndexes(ctx, pending, result)
    s.syncToES(ctx, pending.Resources, result.ImportCtx)
    s.recordHistory(ctx, pending, result)

    return result.Response, nil
}
```

- [ ] **Step 2: Implement Phase 1 — File upload**

```go
func (s *SyncService) uploadKnowledgeFiles(ctx context.Context, pending *PendingImport) (map[int64]string, error) {
    fileURIMap := make(map[int64]string) // documentID -> new URI
    for _, kb := range pending.Resources.KnowledgeBases {
        for _, doc := range kb.Documents {
            fileContent := pending.FileContents[doc.ID]
            if len(fileContent) == 0 {
                continue
            }
            objectKey := fmt.Sprintf("knowledge/%d/%d_%s", pending.SpaceID, doc.ID, doc.Name)
            if err := s.objectStorage.PutObject(ctx, objectKey, fileContent); err != nil {
                return fileURIMap, err
            }
            fileURIMap[doc.ID] = objectKey
        }
    }
    return fileURIMap, nil
}
```

- [ ] **Step 3: Implement Phase 3 — Vector rebuild**

```go
func (s *SyncService) rebuildVectorIndexes(ctx context.Context, pending *PendingImport, result *ImportResult) {
    // For each knowledge base that was created or updated:
    // 1. Get or create collection
    // 2. For each document, call slice2Document + ss.Store
    // Reference: datacopy.go copyDocument() logic
    // Failures are logged but don't fail the import
}
```

- [ ] **Step 4: Verify compilation + commit**

```
cd backend && go build ./application/space/sync/...
git commit -m "feat(sync): add sync service with 3-phase import"
```

---

### Task 12: IDL + API Models + Handler + Router

**Files:**
- Create: `idl/space/space_sync.thrift`
- Create: `backend/api/model/space/space_sync.go`
- Create: `backend/api/handler/space/space_sync_service.go`
- Create: `backend/api/router/space/space_sync.go`
- Modify: `backend/api/router/register.go`

- [ ] **Step 1: Create Thrift IDL**

Create `idl/space/space_sync.thrift` following the pattern of `space_export_import.thrift`:

```thrift
namespace go space

struct SyncExportRequest {
    1: required i64 space_id
    2: required string mode     // "full" or "incremental"
    3: optional i64 since_time
}

struct SyncExportResponse {
    1: required string download_url
    2: required string file_name
    3: required i64 file_size
    4: required string expires_at
    5: required string sync_type
}

// ... (Import preview/confirm request/response structs)
```

- [ ] **Step 2: Create API models**

Create `backend/api/model/space/space_sync.go` with Go structs matching the API design in the spec. Use struct tags for JSON binding. Don't auto-generate from Thrift — write manually for simplicity.

```go
package space

type SyncExportRequest struct {
    Mode      string `json:"mode" binding:"required"`
    SinceTime int64  `json:"since_time,omitempty"`
}

type SyncExportResponse struct {
    DownloadURL string      `json:"download_url"`
    FileName    string      `json:"file_name"`
    FileSize    int64       `json:"file_size"`
    ExpiresAt   string      `json:"expires_at"`
    SyncType    string      `json:"sync_type"`
    Statistics  interface{} `json:"statistics"`
}

type SyncImportPreviewResponse struct {
    ImportToken    string      `json:"import_token"`
    TokenExpiresAt string     `json:"token_expires_at"`
    Manifest       interface{} `json:"manifest"`
    Plan           interface{} `json:"plan"`
    Warnings       []string    `json:"warnings"`
}

type SyncImportConfirmRequest struct {
    ImportToken string `json:"import_token" binding:"required"`
}

type SyncImportConfirmResponse struct {
    Status        string      `json:"status"`
    Statistics    interface{} `json:"statistics"`
    SyncHistoryID int64       `json:"sync_history_id,string"`
}
```

- [ ] **Step 3: Create HTTP handler**

Create `backend/api/handler/space/space_sync_service.go` following the pattern of `space_export_import_service.go`:

```go
package space

func SyncExport(ctx context.Context, c *app.RequestContext) {
    spaceID := getSpaceID(c)
    var req spacemodel.SyncExportRequest
    if err := c.BindJSON(&req); err != nil {
        // error response
        return
    }
    result, err := spacesync.SyncSVC.ExportSync(ctx, &sync.SyncExportRequest{
        SpaceID:   spaceID,
        Mode:      req.Mode,
        SinceTime: req.SinceTime,
    })
    // respond with result
}

func SyncImportPreview(ctx context.Context, c *app.RequestContext) { /* ... */ }
func SyncImportConfirm(ctx context.Context, c *app.RequestContext) { /* ... */ }
func SyncLastExport(ctx context.Context, c *app.RequestContext) { /* ... */ }
func SyncHistory(ctx context.Context, c *app.RequestContext) { /* ... */ }
```

- [ ] **Step 4: Create router**

Create `backend/api/router/space/space_sync.go`:

```go
package space

func RegisterSync(r *server.Hertz) {
    spaceGroup := r.Group("/api/space/:space_id")
    syncGroup := spaceGroup.Group("/sync")
    {
        syncGroup.POST("/export", handler.SyncExport)
        syncGroup.GET("/last-export", handler.SyncLastExport)
        syncGroup.POST("/import/preview", handler.SyncImportPreview)
        syncGroup.POST("/import/confirm", handler.SyncImportConfirm)
        syncGroup.GET("/history", handler.SyncHistory)
    }
}
```

- [ ] **Step 5: Register in main router**

In `backend/api/router/register.go`, add:

```go
space.RegisterSync(r)
```

- [ ] **Step 6: Verify compilation + commit**

```
cd backend && go build ./...
git commit -m "feat(sync): add sync API handler, router, and models"
```

---

### Task 13: Service Initialization + Wiring

**Files:**
- Create: `backend/application/space/space_sync.go`
- Modify: `backend/application/base/appinfra/app_infra.go` (or wherever services are initialized)

- [ ] **Step 1: Create service wrapper**

Create `backend/application/space/space_sync.go`:

```go
package space

import (
    spacesync "github.com/ynet-dev/ynet-studio/backend/application/space/sync"
    // ... other imports
)

var SyncSVC *spacesync.SyncService

func InitSyncService(db *gorm.DB, objectStorage storage.Storage, idGen idgen.IDGenerator,
    eventBus service.ResourceEventBus, projectEventBus service.ProjectEventBus,
    knowledgeSVC knowledge.Knowledge) {
    SyncSVC = spacesync.NewSyncService(db, objectStorage, idGen, eventBus, projectEventBus, knowledgeSVC)
}
```

- [ ] **Step 2: Wire into app initialization**

Find the initialization location (check `app_infra.go` or `main.go`) and add `space.InitSyncService(...)` call after the existing `InitSpaceExportImportService`.

- [ ] **Step 3: Verify full build**

Run: `cd backend && go build ./...`

- [ ] **Step 4: Commit**

```
git commit -m "feat(sync): wire sync service into application initialization"
```

---

### Task 14: Delete Handling in Import

**Files:**
- Modify: `backend/application/space/sync/sync_service.go`

- [ ] **Step 1: Implement delete handling in Phase 2**

Add to the DB transaction phase:

```go
func (s *SyncService) handleDeletedResources(ctx context.Context, tx *gorm.DB, deleted *export.DeletedResources, importCtx *ImportContext) error {
    // For each deleted resource type:
    // 1. Look up mapping (source_id -> target_id)
    // 2. Soft-delete the target resource
    // 3. Remove the mapping record
    // 4. For knowledge: also delete documents and slices

    for _, sourceAgentID := range deleted.Agents {
        targetID, exists := importCtx.MappingStore.GetTargetID("agent", sourceAgentID)
        if !exists {
            continue
        }
        now := time.Now()
        tx.Table("single_agent_draft").Where("agent_id = ?", targetID).Update("deleted_at", now)
        tx.Table("agent_tool_draft").Where("agent_id = ?", targetID).Delete(&struct{}{})
        importCtx.MappingStore.RemoveMapping(ctx, "agent", sourceAgentID)
    }

    // Similar for plugins, workflows, knowledge_bases, documents, folders
    // Knowledge deletion: soft-delete knowledge + all documents + all slices
    // Vector store cleanup happens in Phase 3

    return nil
}
```

- [ ] **Step 2: Track deleted knowledge for Phase 3 vector cleanup**

Store deleted knowledge/document IDs so Phase 3 can clean up vector store collections and partitions.

- [ ] **Step 3: Verify compilation + commit**

```
cd backend && go build ./application/space/sync/...
git commit -m "feat(sync): implement delete handling in sync import"
```

---

### Task 15: End-to-End Integration Test

**Files:**
- Create: `backend/application/space/sync/sync_service_test.go`

- [ ] **Step 1: Write integration test for full export → import cycle**

```go
func TestSyncService_FullExportImport(t *testing.T) {
    // Setup: Create a space with agents, plugins, workflows, knowledge bases
    // Step 1: Export the space
    // Step 2: Import into a new space
    // Step 3: Verify all resources exist in target space
    // Step 4: Verify mapping records created
}
```

- [ ] **Step 2: Write integration test for incremental sync**

```go
func TestSyncService_IncrementalSync(t *testing.T) {
    // Setup: Full export → import (establishes mappings)
    // Step 1: Modify an agent, add a new knowledge doc, delete a workflow
    // Step 2: Incremental export
    // Step 3: Import incremental package
    // Step 4: Verify: agent updated, new doc created, workflow deleted
    // Step 5: Verify: mapping records updated accordingly
}
```

- [ ] **Step 3: Write test for idempotent import**

```go
func TestSyncService_IdempotentImport(t *testing.T) {
    // Import the same package twice
    // Verify no duplicate resources created
}
```

- [ ] **Step 4: Run tests + commit**

```
cd backend && go test ./application/space/sync/... -v
git commit -m "test(sync): add integration tests for sync service"
```

---

### Task 16: Import Script

**Files:**
- Create: `ynet-docker/sync-to-prod.sh`

- [ ] **Step 1: Create the import script**

```bash
#!/bin/bash
set -e

PROD_API="${PROD_API:-http://localhost:8080}"
SPACE_ID="${SPACE_ID:?'SPACE_ID is required'}"
ZIP_FILE="${1:?'Usage: sync-to-prod.sh <export.zip>'}"

echo "=== Ynet Space Sync Import ==="
echo "API: $PROD_API"
echo "Space: $SPACE_ID"
echo "File: $ZIP_FILE"

# Preview
echo -e "\n--- Preview ---"
PREVIEW=$(curl -sf -X POST "$PROD_API/api/space/$SPACE_ID/sync/import/preview" \
  -F "file=@$ZIP_FILE")
TOKEN=$(echo "$PREVIEW" | jq -r '.import_token')
echo "$PREVIEW" | jq '.plan'

WARNINGS=$(echo "$PREVIEW" | jq -r '.warnings[]? // empty')
if [ -n "$WARNINGS" ]; then
    echo -e "\nWarnings:"
    echo "$WARNINGS"
fi

# Confirm
read -p "Proceed with import? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo -e "\n--- Importing ---"
RESULT=$(curl -sf -X POST "$PROD_API/api/space/$SPACE_ID/sync/import/confirm" \
  -H "Content-Type: application/json" \
  -d "{\"import_token\": \"$TOKEN\"}")

echo "$RESULT" | jq '.statistics'
echo -e "\nSync complete. History ID: $(echo "$RESULT" | jq -r '.sync_history_id')"
```

- [ ] **Step 2: Make executable + commit**

```
chmod +x ynet-docker/sync-to-prod.sh
git add ynet-docker/sync-to-prod.sh
git commit -m "feat(sync): add production import script"
```

---

## Dependency Graph

```
Task 1 (DDL + Repos)
  └─► Task 2 (Export Types)
       ├─► Task 3 (Knowledge Collection)
       ├─► Task 4 (Folder/ExtKnowledge Collection)
       └─► Task 8 (ImportContext + RefRewriter Fix)
            └─► Task 9 (Validator)
                 └─► Task 10 (Knowledge Import + Upsert)

Task 5 (Incremental Export) ──► depends on Task 3, 4
Task 6 (Serializer) ──► depends on Task 2
Task 7 (Exporter Sync Mode) ──► depends on Task 5, 6

Task 11 (Sync Service) ──► depends on Task 1, 7, 10
Task 12 (API Layer) ──► depends on Task 11
Task 13 (Wiring) ──► depends on Task 12
Task 14 (Delete Handling) ──► depends on Task 11
Task 15 (Integration Tests) ──► depends on Task 13, 14
Task 16 (Import Script) ──► depends on Task 12
```

**Recommended execution order:** 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14 → 15 → 16
