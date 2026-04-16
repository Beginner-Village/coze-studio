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

package spacesync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	spaceexport "github.com/ynet-dev/ynet-studio/backend/application/space/export"
	spaceimport "github.com/ynet-dev/ynet-studio/backend/application/space/import"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	syncImportPrefix   = "sync_imports"
	syncImportTTL      = 30 * time.Minute
	snapshotPrefix     = "space_snapshots"
)

type PendingSyncImport struct {
	Token         string
	SpaceID       int64
	UserID        int64
	SourceSpaceID int64
	Manifest      *spaceexport.Manifest
	TempFileKey   string
	MappingStore  *SyncMappingStore
	Plan          *ImportPlan
	CreatedAt     time.Time
}

type ImportPlan struct {
	Create map[string]int `json:"create"`
	Update map[string]int `json:"update"`
	Delete map[string]int `json:"delete"`
}

type SyncService struct {
	db              *gorm.DB
	exporter        *spaceexport.SpaceExporter
	importer        *spaceimport.SpaceImporter
	objectStorage   storage.Storage
	idGen           idgen.IDGenerator
	historyRepo     *SyncHistoryRepo
	eventBus        service.ResourceEventBus
	projectEventBus service.ProjectEventBus
	pendingCache    map[string]*PendingSyncImport
	mu              sync.Mutex
}

func NewSyncService(db *gorm.DB, exporter *spaceexport.SpaceExporter, importer *spaceimport.SpaceImporter,
	objectStorage storage.Storage, idGen idgen.IDGenerator,
	eventBus service.ResourceEventBus, projectEventBus service.ProjectEventBus) *SyncService {
	return &SyncService{
		db:              db,
		exporter:        exporter,
		importer:        importer,
		objectStorage:   objectStorage,
		idGen:           idGen,
		historyRepo:     NewSyncHistoryRepo(db),
		eventBus:        eventBus,
		projectEventBus: projectEventBus,
		pendingCache:    make(map[string]*PendingSyncImport),
	}
}

func (s *SyncService) ExportSync(ctx context.Context, req *spaceexport.SyncExportRequest) (*spaceexport.ExportResult, error) {
	return s.exporter.ExportSync(ctx, req)
}

func (s *SyncService) ImportPreview(ctx context.Context, spaceID, userID int64, fileContent []byte) (*SyncPreviewResult, error) {
	logs.CtxInfof(ctx, "Starting sync import preview for space_id=%d, user_id=%d", spaceID, userID)

	validator := spaceimport.NewValidator()
	validationResult, err := validator.ValidateAndParse(ctx, fileContent)
	if err != nil {
		return nil, err
	}

	manifest := validationResult.Manifest
	sourceSpaceID := manifest.Source.SpaceID

	// Store ZIP to object storage
	token, err := generateSyncToken()
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to generate sync token"))
	}

	tempFileKey := fmt.Sprintf("%s/%s.zip", syncImportPrefix, token)
	if err = s.objectStorage.PutObject(ctx, tempFileKey, fileContent); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to store temp import file"))
	}

	// Build mapping store and load existing mappings
	mappingStore := NewSyncMappingStore(s.db, sourceSpaceID, spaceID)
	if err = mappingStore.LoadAll(ctx); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to load sync mappings"))
	}

	// Build import plan
	plan := s.buildImportPlan(validationResult.Resources, mappingStore)

	// Detect version conflict
	incomingVersion := manifest.ReleaseVersion
	currentVersion := ""
	if incomingVersion != "" {
		current, _ := s.historyRepo.GetCurrentVersion(ctx, spaceID)
		if current != nil && current.Version != nil {
			currentVersion = *current.Version
		}
		if currentVersion != "" && !isNewerVersion(incomingVersion, currentVersion) {
			validationResult.Warnings = append(validationResult.Warnings,
				fmt.Sprintf("当前已部署版本 %s >= 即将导入的版本 %s，请确认是否需要降级", currentVersion, incomingVersion))
		}
	}

	// Store pending import
	pending := &PendingSyncImport{
		Token:         token,
		SpaceID:       spaceID,
		UserID:        userID,
		SourceSpaceID: sourceSpaceID,
		Manifest:      manifest,
		TempFileKey:   tempFileKey,
		MappingStore:  mappingStore,
		Plan:          plan,
		CreatedAt:     time.Now(),
	}
	s.mu.Lock()
	s.pendingCache[token] = pending
	s.cleanupExpiredTokens()
	s.mu.Unlock()

	logs.CtxInfof(ctx, "Sync import preview completed, token=%s, plan=%+v", token, plan)

	return &SyncPreviewResult{
		ImportToken:     token,
		Manifest:        manifest,
		Plan:            plan,
		Warnings:        validationResult.Warnings,
		TokenExpiresAt:  time.Now().Add(syncImportTTL).Unix(),
		IncomingVersion: incomingVersion,
		CurrentVersion:  currentVersion,
	}, nil
}

func (s *SyncService) ImportConfirm(ctx context.Context, spaceID, userID int64, importToken string) (*SyncImportResult, error) {
	logs.CtxInfof(ctx, "Starting sync import confirm for space_id=%d, token=%s", spaceID, importToken)

	s.mu.Lock()
	pending, ok := s.pendingCache[importToken]
	if !ok {
		s.mu.Unlock()
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "invalid or expired sync import token"))
	}

	if pending.SpaceID != spaceID || pending.UserID != userID {
		s.mu.Unlock()
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "token does not match request"))
	}

	if time.Since(pending.CreatedAt) > syncImportTTL {
		delete(s.pendingCache, importToken)
		s.mu.Unlock()
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "sync import token has expired"))
	}
	s.mu.Unlock()

	// Read ZIP from object storage
	fileContent, err := s.objectStorage.GetObject(ctx, pending.TempFileKey)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to read temp import file"))
	}

	// Re-parse ZIP contents
	validator := spaceimport.NewValidator()
	validationResult, err := validator.ValidateAndParse(ctx, fileContent)
	if err != nil {
		return nil, err
	}

	// Detect release version from manifest
	releaseVersion := pending.Manifest.ReleaseVersion
	var snapshotKey string
	if releaseVersion != "" {
		logs.CtxInfof(ctx, "Importing release version %s into space_id=%d", releaseVersion, spaceID)
		// Create pre-import snapshot for safety (non-blocking: if snapshot fails, still proceed)
		snapshotKey, err = s.CreateSnapshot(ctx, spaceID)
		if err != nil {
			logs.CtxWarnf(ctx, "Failed to create pre-import snapshot for space_id=%d: %v (proceeding with import)", spaceID, err)
			snapshotKey = ""
		}
	}

	// Handle deleted resources before creating/updating
	if validationResult.Deleted != nil {
		if txErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.handleDeletedResources(ctx, tx, validationResult.Deleted, pending.MappingStore)
		}); txErr != nil {
			return nil, errorx.WrapByCode(txErr, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to handle deleted resources"))
		}
	}

	// Execute the sync import using the existing importer in upsert mode
	importResult, err := s.importer.Confirm(ctx, &spaceimport.ConfirmRequest{
		SpaceID:     spaceID,
		UserID:      userID,
		ImportToken: importToken,
	})

	// If the importer's Confirm doesn't find the token (since we're managing our own cache),
	// fall back to using Preview+Confirm flow with the re-parsed data
	if err != nil {
		// Use Preview to register with the importer's cache
		previewResult, previewErr := s.importer.Preview(ctx, &spaceimport.PreviewRequest{
			SpaceID:     spaceID,
			UserID:      userID,
			FileContent: fileContent,
		})
		if previewErr != nil {
			return nil, previewErr
		}

		// Now confirm using the importer's token
		importResult, err = s.importer.Confirm(ctx, &spaceimport.ConfirmRequest{
			SpaceID:     spaceID,
			UserID:      userID,
			ImportToken: previewResult.ImportToken,
		})
		if err != nil {
			return nil, err
		}
	}

	// Update sync mappings with source→target ID pairs
	if importResult.IDMappings != nil {
		s.updateSyncMappings(ctx, pending, importResult.IDMappings)
	}

	// Record sync history with version info
	var histOpts []recordHistoryOpts
	if releaseVersion != "" {
		histOpts = append(histOpts, recordHistoryOpts{
			Version:     releaseVersion,
			SnapshotKey: snapshotKey,
		})
	}
	s.recordSyncHistory(ctx, pending, validationResult.Resources, importResult, histOpts...)

	// Clean up
	s.mu.Lock()
	delete(s.pendingCache, importToken)
	s.mu.Unlock()
	go func() {
		if delErr := s.objectStorage.DeleteObject(context.Background(), pending.TempFileKey); delErr != nil {
			logs.CtxWarnf(ctx, "Failed to delete temp sync import file %s: %v", pending.TempFileKey, delErr)
		}
	}()

	logs.CtxInfof(ctx, "Sync import completed for space_id=%d", spaceID)

	return &SyncImportResult{
		AgentsCreated:    importResult.AgentsCreated,
		PluginsCreated:   importResult.PluginsCreated,
		WorkflowsCreated: importResult.WorkflowsCreated,
		VariablesCreated: importResult.VariablesCreated,
		KnowledgeCreated: importResult.KnowledgeCreated,
		Plan:             pending.Plan,
	}, nil
}

func (s *SyncService) GetLastExport(ctx context.Context, spaceID int64) (*SyncHistory, error) {
	return s.historyRepo.GetLastExport(ctx, spaceID)
}

func (s *SyncService) GetHistory(ctx context.Context, spaceID int64) ([]SyncHistory, error) {
	return s.historyRepo.ListBySpace(ctx, spaceID, 50)
}

// handleDeletedResources processes deleted resources from the import package.
// For each deleted resource that has a mapping in the target space, soft-delete it.
func (s *SyncService) handleDeletedResources(ctx context.Context, tx *gorm.DB, deleted *spaceexport.DeletedResources, mappingStore *SyncMappingStore) error {
	if deleted == nil {
		return nil
	}
	now := time.Now()

	// Delete agents
	for _, sourceID := range deleted.Agents {
		targetID, exists := mappingStore.GetTargetID("agent", sourceID)
		if !exists {
			continue
		}
		tx.Table("single_agent_draft").Where("agent_id = ?", targetID).Update("deleted_at", now)
		tx.Table("agent_tool_draft").Where("agent_id = ?", targetID).Delete(&struct{}{})
		mappingStore.RemoveMapping(ctx, "agent", sourceID)
	}

	// Delete plugins
	for _, sourceID := range deleted.Plugins {
		targetID, exists := mappingStore.GetTargetID("plugin", sourceID)
		if !exists {
			continue
		}
		tx.Table("plugin_draft").Where("id = ?", targetID).Update("deleted_at", now)
		tx.Table("plugin").Where("id = ?", targetID).Update("deleted_at", now)
		mappingStore.RemoveMapping(ctx, "plugin", sourceID)
	}

	// Delete workflows
	for _, sourceID := range deleted.Workflows {
		targetID, exists := mappingStore.GetTargetID("workflow", sourceID)
		if !exists {
			continue
		}
		tx.Table("workflow_meta").Where("id = ?", targetID).Update("deleted_at", now)
		tx.Table("workflow_draft").Where("id = ?", targetID).Update("deleted_at", now)
		mappingStore.RemoveMapping(ctx, "workflow", sourceID)
	}

	// Delete knowledge bases (+ their documents and slices)
	for _, sourceID := range deleted.KnowledgeBases {
		targetID, exists := mappingStore.GetTargetID("knowledge", sourceID)
		if !exists {
			continue
		}
		tx.Table("knowledge_document_slice").Where("knowledge_id = ?", targetID).Update("deleted_at", now)
		tx.Table("knowledge_document").Where("knowledge_id = ?", targetID).Update("deleted_at", now)
		tx.Table("knowledge").Where("id = ?", targetID).Update("deleted_at", now)
		mappingStore.RemoveMapping(ctx, "knowledge", sourceID)
	}

	// Delete individual documents
	for _, sourceID := range deleted.Documents {
		targetID, exists := mappingStore.GetTargetID("document", sourceID)
		if !exists {
			continue
		}
		tx.Table("knowledge_document_slice").Where("document_id = ?", targetID).Update("deleted_at", now)
		tx.Table("knowledge_document").Where("id = ?", targetID).Update("deleted_at", now)
		mappingStore.RemoveMapping(ctx, "document", sourceID)
	}

	// Delete folders
	for _, sourceID := range deleted.Folders {
		targetID, exists := mappingStore.GetTargetID("folder", sourceID)
		if !exists {
			continue
		}
		tx.Table("folder").Where("id = ?", targetID).Update("deleted_at", now)
		tx.Table("resource_folder_mapping").Where("folder_id = ?", targetID).Delete(&struct{}{})
		mappingStore.RemoveMapping(ctx, "folder", sourceID)
	}

	return nil
}

func (s *SyncService) buildImportPlan(resources *spaceexport.SpaceResources, mappingStore *SyncMappingStore) *ImportPlan {
	plan := &ImportPlan{
		Create: make(map[string]int),
		Update: make(map[string]int),
		Delete: make(map[string]int),
	}

	countResources := func(resourceType string, ids []int64) {
		for _, id := range ids {
			if _, exists := mappingStore.GetTargetID(resourceType, id); exists {
				plan.Update[resourceType]++
			} else {
				plan.Create[resourceType]++
			}
		}
	}

	agentIDs := make([]int64, len(resources.Agents))
	for i, a := range resources.Agents {
		agentIDs[i] = a.ID
	}
	countResources("agent", agentIDs)

	pluginIDs := make([]int64, len(resources.Plugins))
	for i, p := range resources.Plugins {
		pluginIDs[i] = p.ID
	}
	countResources("plugin", pluginIDs)

	workflowIDs := make([]int64, len(resources.Workflows))
	for i, w := range resources.Workflows {
		workflowIDs[i] = w.ID
	}
	countResources("workflow", workflowIDs)

	variableIDs := make([]int64, len(resources.Variables))
	for i, v := range resources.Variables {
		variableIDs[i] = v.ID
	}
	countResources("variable", variableIDs)

	kbIDs := make([]int64, len(resources.KnowledgeBases))
	for i, kb := range resources.KnowledgeBases {
		kbIDs[i] = kb.ID
	}
	countResources("knowledge_base", kbIDs)

	return plan
}

type recordHistoryOpts struct {
	Version             string
	SnapshotKey         string
	RollbackFromVersion string
}

func (s *SyncService) recordSyncHistory(ctx context.Context, pending *PendingSyncImport, resources *spaceexport.SpaceResources, result *spaceimport.ImportResult, opts ...recordHistoryOpts) {
	now := time.Now().Unix()
	stats, _ := json.Marshal(map[string]interface{}{
		"agents_created":    result.AgentsCreated,
		"plugins_created":   result.PluginsCreated,
		"workflows_created": result.WorkflowsCreated,
		"variables_created": result.VariablesCreated,
		"knowledge_created": result.KnowledgeCreated,
		"plan":              pending.Plan,
	})

	syncType := "full"
	if pending.Manifest.SyncType != "" {
		syncType = pending.Manifest.SyncType
	}

	record := &SyncHistory{
		SourceSpaceID: pending.SourceSpaceID,
		TargetSpaceID: pending.SpaceID,
		SyncType:      syncType,
		ExportTime:    now,
		ImportTime:    &now,
		Statistics:    json.RawMessage(stats),
		Status:        1, // success
		CreatedAt:     now,
	}

	if len(opts) > 0 {
		o := opts[0]
		if o.Version != "" {
			record.Version = &o.Version
		}
		if o.SnapshotKey != "" {
			record.SnapshotKey = &o.SnapshotKey
		}
		if o.RollbackFromVersion != "" {
			record.RollbackFromVersion = &o.RollbackFromVersion
		}
	}

	if err := s.historyRepo.Create(ctx, record); err != nil {
		logs.CtxWarnf(ctx, "Failed to record sync history: %v", err)
	}
}

// CreateSnapshot exports the current state of a space as a full snapshot for rollback.
// The snapshot is stored permanently in object storage.
func (s *SyncService) CreateSnapshot(ctx context.Context, spaceID int64) (string, error) {
	logs.CtxInfof(ctx, "Creating snapshot for space_id=%d", spaceID)

	raw, err := s.exporter.ExportSyncRaw(ctx, &spaceexport.SyncExportRequest{
		SpaceID: spaceID,
		Mode:    "full",
	})
	if err != nil {
		return "", fmt.Errorf("export snapshot: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	snapshotKey := fmt.Sprintf("%s/%d/%s.zip", snapshotPrefix, spaceID, timestamp)

	if err = s.objectStorage.PutObject(ctx, snapshotKey, raw.ZipContent); err != nil {
		return "", fmt.Errorf("upload snapshot: %w", err)
	}

	logs.CtxInfof(ctx, "Snapshot created for space_id=%d, key=%s, size=%d", spaceID, snapshotKey, raw.FileSize)
	return snapshotKey, nil
}

// RollbackToVersion rolls back a space to a previous version by importing its release package.
// It creates a snapshot of the current state before rolling back.
func (s *SyncService) RollbackToVersion(ctx context.Context, spaceID, userID int64, targetVersion, releasePackageKey string) (*RollbackResult, error) {
	logs.CtxInfof(ctx, "Rolling back space_id=%d to version=%s", spaceID, targetVersion)

	// Get current version
	currentHistory, _ := s.historyRepo.GetCurrentVersion(ctx, spaceID)
	currentVersion := ""
	if currentHistory != nil && currentHistory.Version != nil {
		currentVersion = *currentHistory.Version
	}

	// Create snapshot before rollback
	snapshotKey, err := s.CreateSnapshot(ctx, spaceID)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to create pre-rollback snapshot"))
	}

	// Read the release package from object storage
	zipContent, err := s.objectStorage.GetObject(ctx, releasePackageKey)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to read release package"))
	}

	// Execute import via standard preview+confirm flow
	previewResult, err := s.importer.Preview(ctx, &spaceimport.PreviewRequest{
		SpaceID:     spaceID,
		UserID:      userID,
		FileContent: zipContent,
	})
	if err != nil {
		return nil, err
	}

	importResult, err := s.importer.Confirm(ctx, &spaceimport.ConfirmRequest{
		SpaceID:     spaceID,
		UserID:      userID,
		ImportToken: previewResult.ImportToken,
	})
	if err != nil {
		return nil, err
	}

	// Record sync history with rollback info
	pending := &PendingSyncImport{
		SpaceID:       spaceID,
		UserID:        userID,
		SourceSpaceID: 0,
		Manifest:      previewResult.Manifest,
		Plan:          &ImportPlan{},
	}

	s.recordSyncHistory(ctx, pending, nil, importResult, recordHistoryOpts{
		Version:             targetVersion,
		SnapshotKey:         snapshotKey,
		RollbackFromVersion: currentVersion,
	})

	logs.CtxInfof(ctx, "Rollback completed for space_id=%d, from=%s to=%s", spaceID, currentVersion, targetVersion)

	return &RollbackResult{
		RolledBackFrom: currentVersion,
		RolledBackTo:   targetVersion,
		SnapshotKey:    snapshotKey,
	}, nil
}

// GetCurrentVersion returns the current deployed version for a space
func (s *SyncService) GetCurrentVersion(ctx context.Context, spaceID int64) (*SyncHistory, error) {
	return s.historyRepo.GetCurrentVersion(ctx, spaceID)
}

// RollbackResult contains the result of a rollback operation
type RollbackResult struct {
	RolledBackFrom string `json:"rolled_back_from"`
	RolledBackTo   string `json:"rolled_back_to"`
	SnapshotKey    string `json:"snapshot_key"`
}

func (s *SyncService) cleanupExpiredTokens() {
	now := time.Now()
	for token, pending := range s.pendingCache {
		if now.Sub(pending.CreatedAt) > syncImportTTL {
			delete(s.pendingCache, token)
		}
	}
}

// isNewerVersion returns true if version a is strictly newer than version b.
// Uses semver parsing for correct multi-digit comparison (e.g., v1.0.10 > v1.0.9).
func isNewerVersion(a, b string) bool {
	aMaj, aMin, aPatch, aOk := parseSemverSimple(a)
	bMaj, bMin, bPatch, bOk := parseSemverSimple(b)
	if !aOk || !bOk {
		return a > b // fallback to string comparison
	}
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	if aMin != bMin {
		return aMin > bMin
	}
	return aPatch > bPatch
}

// parseSemverSimple parses a semver string like "v1.2.3" into components.
func parseSemverSimple(version string) (major, minor, patch int, ok bool) {
	if len(version) < 2 || version[0] != 'v' {
		return 0, 0, 0, false
	}
	parts := [3]int{}
	idx := 0
	for _, ch := range version[1:] {
		if ch == '.' {
			idx++
			if idx > 2 {
				return 0, 0, 0, false
			}
			continue
		}
		if ch < '0' || ch > '9' {
			return 0, 0, 0, false
		}
		parts[idx] = parts[idx]*10 + int(ch-'0')
	}
	if idx != 2 {
		return 0, 0, 0, false
	}
	return parts[0], parts[1], parts[2], true
}

// updateSyncMappings records source→target ID mappings after a successful import.
// This enables future incremental imports to detect existing resources.
func (s *SyncService) updateSyncMappings(ctx context.Context, pending *PendingSyncImport, idMappings map[string]map[int64]int64) {
	mappingStore := NewSyncMappingStore(s.db, pending.SourceSpaceID, pending.SpaceID)

	for resourceType, mappings := range idMappings {
		for sourceID, targetID := range mappings {
			if err := mappingStore.UpsertMapping(ctx, nil, resourceType, sourceID, targetID, time.Now().UnixMilli()); err != nil {
				logs.CtxWarnf(ctx, "Failed to upsert sync mapping %s:%d->%d: %v", resourceType, sourceID, targetID, err)
			}
		}
	}

	logs.CtxInfof(ctx, "Updated sync mappings for space_id=%d, source_space_id=%d", pending.SpaceID, pending.SourceSpaceID)
}

func generateSyncToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type SyncPreviewResult struct {
	ImportToken    string                `json:"import_token"`
	Manifest       *spaceexport.Manifest `json:"manifest"`
	Plan           *ImportPlan           `json:"plan"`
	Warnings       []string              `json:"warnings"`
	TokenExpiresAt int64                 `json:"token_expires_at"`
	// Version info for display
	IncomingVersion string `json:"incoming_version,omitempty"`
	CurrentVersion  string `json:"current_version,omitempty"`
}

type SyncImportResult struct {
	AgentsCreated    int         `json:"agents_created"`
	PluginsCreated   int         `json:"plugins_created"`
	WorkflowsCreated int         `json:"workflows_created"`
	VariablesCreated int         `json:"variables_created"`
	KnowledgeCreated int         `json:"knowledge_created"`
	Plan             *ImportPlan `json:"plan"`
}
