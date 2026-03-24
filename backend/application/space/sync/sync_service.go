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
	syncImportPrefix = "sync_imports"
	syncImportTTL    = 30 * time.Minute
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
	s.pendingCache[token] = pending

	// Clean up expired tokens
	s.cleanupExpiredTokens()

	logs.CtxInfof(ctx, "Sync import preview completed, token=%s, plan=%+v", token, plan)

	return &SyncPreviewResult{
		ImportToken:    token,
		Manifest:       manifest,
		Plan:           plan,
		Warnings:       validationResult.Warnings,
		TokenExpiresAt: time.Now().Add(syncImportTTL).Unix(),
	}, nil
}

func (s *SyncService) ImportConfirm(ctx context.Context, spaceID, userID int64, importToken string) (*SyncImportResult, error) {
	logs.CtxInfof(ctx, "Starting sync import confirm for space_id=%d, token=%s", spaceID, importToken)

	pending, ok := s.pendingCache[importToken]
	if !ok {
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "invalid or expired sync import token"))
	}

	if pending.SpaceID != spaceID || pending.UserID != userID {
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "token does not match request"))
	}

	if time.Since(pending.CreatedAt) > syncImportTTL {
		delete(s.pendingCache, importToken)
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "sync import token has expired"))
	}

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

	// Record sync history
	s.recordSyncHistory(ctx, pending, validationResult.Resources, importResult)

	// Clean up
	delete(s.pendingCache, importToken)
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

func (s *SyncService) recordSyncHistory(ctx context.Context, pending *PendingSyncImport, resources *spaceexport.SpaceResources, result *spaceimport.ImportResult) {
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

	if err := s.historyRepo.Create(ctx, record); err != nil {
		logs.CtxWarnf(ctx, "Failed to record sync history: %v", err)
	}
}

func (s *SyncService) cleanupExpiredTokens() {
	now := time.Now()
	for token, pending := range s.pendingCache {
		if now.Sub(pending.CreatedAt) > syncImportTTL {
			delete(s.pendingCache, token)
		}
	}
}

func generateSyncToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type SyncPreviewResult struct {
	ImportToken    string               `json:"import_token"`
	Manifest       *spaceexport.Manifest `json:"manifest"`
	Plan           *ImportPlan           `json:"plan"`
	Warnings       []string              `json:"warnings"`
	TokenExpiresAt int64                 `json:"token_expires_at"`
}

type SyncImportResult struct {
	AgentsCreated    int         `json:"agents_created"`
	PluginsCreated   int         `json:"plugins_created"`
	WorkflowsCreated int         `json:"workflows_created"`
	VariablesCreated int         `json:"variables_created"`
	KnowledgeCreated int         `json:"knowledge_created"`
	Plan             *ImportPlan `json:"plan"`
}
