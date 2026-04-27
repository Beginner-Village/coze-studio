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

package spaceimport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
	resCommon "github.com/ynet-dev/ynet-studio/backend/api/model/resource/common"
	pluginConf "github.com/ynet-dev/ynet-studio/backend/domain/plugin/conf"
	searchEntity "github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// toJSON converts an interface{} to JSON string for database storage
// It properly handles nil pointers and empty slices/maps by returning nil
// instead of the JSON string "null", which would be stored in the database
// and cause issues when reading back.
func toJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	// Use reflection to check if the underlying value is nil
	// This is necessary because an interface{} holding a nil pointer
	// is NOT equal to nil in Go
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	// Check if the result is "null" - this can happen for typed nil values
	if string(data) == "null" {
		return nil
	}
	return string(data)
}

const (
	// ImportTokenTTL is the time-to-live for import tokens (10 minutes)
	ImportTokenTTL = 10 * time.Minute
)

// SpaceImporter handles the import of a space from a ZIP file
type SpaceImporter struct {
	db              *gorm.DB
	idGen           idgen.IDGenerator           // ID generator for creating new IDs
	idMapper        *IDMapper
	rewriter        *ReferenceRewriter
	validator       *Validator
	pendingCache    map[string]*PendingImport   // In-memory cache for simplicity
	eventBus        service.ResourceEventBus    // For ES sync (plugins, workflows)
	projectEventBus service.ProjectEventBus     // For ES sync (agents)
}

// PendingImport represents a pending import operation
type PendingImport struct {
	Token       string
	SpaceID     int64
	UserID      int64
	Manifest    *export.Manifest
	Resources   *export.SpaceResources
	CreatedAt   time.Time
	FileContent []byte
	// ExistingMappings is the set of source→target mappings already known
	// for this (source, target) pair. SyncService populates it from
	// space_sync_mapping so the importer reuses target IDs and updates
	// existing rows rather than creating duplicates.
	ExistingMappings map[string]map[int64]int64
}

// NewSpaceImporter creates a new SpaceImporter
func NewSpaceImporter(db *gorm.DB, idGen idgen.IDGenerator, eventBus service.ResourceEventBus, projectEventBus service.ProjectEventBus) *SpaceImporter {
	return &SpaceImporter{
		db:              db,
		idGen:           idGen,
		idMapper:        NewIDMapper(idGen),
		rewriter:        NewReferenceRewriter(),
		validator:       NewValidator(),
		pendingCache:    make(map[string]*PendingImport),
		eventBus:        eventBus,
		projectEventBus: projectEventBus,
	}
}

// PreviewRequest represents a request to preview an import
type PreviewRequest struct {
	SpaceID     int64
	UserID      int64
	FileContent []byte
	// ExistingMappings forwarded into the resulting PendingImport. See
	// PendingImport.ExistingMappings for the full description.
	ExistingMappings map[string]map[int64]int64
}

// Preview validates and previews an import
func (s *SpaceImporter) Preview(ctx context.Context, req *PreviewRequest) (*PreviewResult, error) {
	logs.CtxInfof(ctx, "Starting import preview for space_id=%d, user_id=%d", req.SpaceID, req.UserID)

	// Validate and parse the import package
	validationResult, err := s.validator.ValidateAndParse(ctx, req.FileContent)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to validate import package: %v", err)
		return nil, err
	}

	// Generate import token
	token, err := s.generateToken()
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "failed to generate import token"))
	}

	// Store pending import
	s.pendingCache[token] = &PendingImport{
		Token:            token,
		SpaceID:          req.SpaceID,
		UserID:           req.UserID,
		Manifest:         validationResult.Manifest,
		Resources:        validationResult.Resources,
		CreatedAt:        time.Now(),
		FileContent:      req.FileContent,
		ExistingMappings: req.ExistingMappings,
	}

	// Clean up expired tokens
	s.cleanupExpiredTokens()

	logs.CtxInfof(ctx, "Import preview completed, token=%s", token)

	return &PreviewResult{
		ImportToken: token,
		Manifest:    validationResult.Manifest,
		Warnings:    validationResult.Warnings,
	}, nil
}

// ConfirmRequest represents a request to confirm an import
type ConfirmRequest struct {
	SpaceID     int64
	UserID      int64
	ImportToken string
}

// Confirm executes the import operation
func (s *SpaceImporter) Confirm(ctx context.Context, req *ConfirmRequest) (*ImportResult, error) {
	logs.CtxInfof(ctx, "Starting import confirmation for space_id=%d, token=%s", req.SpaceID, req.ImportToken)

	// Get pending import
	pending, ok := s.pendingCache[req.ImportToken]
	if !ok {
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "invalid or expired import token"))
	}

	// Validate token
	if pending.SpaceID != req.SpaceID || pending.UserID != req.UserID {
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "token does not match request"))
	}

	// Check if token is expired
	if time.Since(pending.CreatedAt) > ImportTokenTTL {
		delete(s.pendingCache, req.ImportToken)
		return nil, errorx.New(errno.ErrSpaceImportFailedCode, errorx.KV("msg", "import token has expired"))
	}

	// Execute import
	result, err := s.executeImport(ctx, pending)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to execute import: %v", err)
		return nil, err
	}

	// Remove pending import
	delete(s.pendingCache, req.ImportToken)

	logs.CtxInfof(ctx, "Import completed: agents=%d, plugins=%d, workflows=%d, variables=%d, space_models=%d, knowledge=%d",
		result.AgentsCreated, result.PluginsCreated, result.WorkflowsCreated, result.VariablesCreated, result.SpaceModelsCreated, result.KnowledgeCreated)

	return result, nil
}

// executeImport executes the actual import in a transaction
func (s *SpaceImporter) executeImport(ctx context.Context, pending *PendingImport) (*ImportResult, error) {
	result := &ImportResult{}

	// Generate ID mappings, reusing target IDs from prior syncs when available.
	importCtx, err := s.idMapper.GenerateMapping(ctx, pending.Resources, &pending.Manifest.IDRegistry, pending.ExistingMappings)
	if err != nil {
		return nil, err
	}
	importCtx.TargetSpaceID = pending.SpaceID
	importCtx.UserID = pending.UserID

	// Execute in transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 0: Create space models (agents depend on them for model_id)
		for _, spaceModel := range pending.Resources.SpaceModels {
			if err := s.createSpaceModel(ctx, tx, spaceModel, importCtx); err != nil {
				return err
			}
			result.SpaceModelsCreated++
		}

		// Step 0.5: Get fallback model from target space (first available space_model)
		fallbackModelID, err := s.getFirstSpaceModel(ctx, tx, pending.SpaceID)
		if err != nil {
			logs.CtxWarnf(ctx, "Failed to get fallback model for space %d: %v", pending.SpaceID, err)
		} else if fallbackModelID > 0 {
			importCtx.FallbackModelID = &fallbackModelID
			logs.CtxInfof(ctx, "Using fallback model_id=%d for agents with missing models", fallbackModelID)
		} else {
			logs.CtxWarnf(ctx, "No models available in target space %d, agents may have no model configured", pending.SpaceID)
		}

		// Step 1: Create plugins (no dependencies)
		for _, plugin := range pending.Resources.Plugins {
			rewritten := s.rewriter.RewritePlugin(ctx, plugin, importCtx)
			if err := s.createPlugin(ctx, tx, rewritten, importCtx); err != nil {
				return err
			}
			result.PluginsCreated++
		}

		// Step 2: Create workflows (may reference plugins)
		for _, workflow := range pending.Resources.Workflows {
			rewritten := s.rewriter.RewriteWorkflow(ctx, workflow, importCtx)
			if err := s.createWorkflow(ctx, tx, rewritten, importCtx); err != nil {
				return err
			}
			result.WorkflowsCreated++
		}

		// Step 3: Create knowledge bases
		for _, kb := range pending.Resources.KnowledgeBases {
			if err := s.createKnowledge(ctx, tx, kb, importCtx); err != nil {
				return err
			}
			result.KnowledgeCreated++
		}

		// Step 4: Create agents (may reference plugins, workflows, and space models)
		for _, agent := range pending.Resources.Agents {
			rewritten := s.rewriter.RewriteAgent(ctx, agent, importCtx)
			if err := s.createAgent(ctx, tx, rewritten, importCtx); err != nil {
				return err
			}
			result.AgentsCreated++
		}

		// Step 5: Create variables (reference agents)
		for _, variable := range pending.Resources.Variables {
			rewritten := s.rewriter.RewriteVariable(ctx, variable, importCtx)
			if err := s.createVariable(ctx, tx, rewritten, importCtx); err != nil {
				return err
			}
			result.VariablesCreated++
		}

		// Step 6: Create folders
		for _, folder := range pending.Resources.Folders {
			if err := s.createFolder(ctx, tx, folder, importCtx); err != nil {
				return err
			}
		}

		// Step 7: Create folder mappings
		for _, mapping := range pending.Resources.FolderMappings {
			if err := s.createFolderMapping(ctx, tx, mapping, importCtx); err != nil {
				logs.CtxWarnf(ctx, "Failed to create folder mapping: %v", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("msg", "import transaction failed"))
	}

	// Expose ID mappings for sync mapping updates
	result.IDMappings = map[string]map[int64]int64{
		"agent":              importCtx.AgentIDMap,
		"plugin":             importCtx.PluginIDMap,
		"workflow":           importCtx.WorkflowIDMap,
		"variable":           importCtx.VariableIDMap,
		"space_model":        importCtx.SpaceModelIDMap,
		"knowledge":          importCtx.KnowledgeIDMap,
		"document":           importCtx.DocumentIDMap,
		"folder":             importCtx.FolderIDMap,
		"external_knowledge": importCtx.ExternalKnowledgeIDMap,
	}

	// Sync to ES after successful transaction (outside transaction to avoid blocking)
	s.syncToES(ctx, pending.Resources, importCtx)

	return result, nil
}

// createPlugin creates a plugin in the database
func (s *SpaceImporter) createPlugin(ctx context.Context, tx *gorm.DB, plugin *export.ExportedPlugin, importCtx *ImportContext) error {
	newID := importCtx.PluginIDMap[plugin.ID]
	now := time.Now().UnixMilli()

	// Initial version for imported plugins
	initialVersion := "v1.0.0"

	// Create plugin main table (required for plugin queries)
	pluginModel := map[string]interface{}{
		"id":           newID,
		"space_id":     importCtx.TargetSpaceID,
		"developer_id": importCtx.UserID,
		"app_id":       0, // No associated app for imported plugins
		"icon_uri":     plugin.IconURI,
		"server_url":   plugin.ServerURL,
		"plugin_type":  plugin.PluginType,
		"manifest":     toJSON(plugin.Manifest),
		"openapi_doc":  toJSON(plugin.OpenapiDoc),
		"version":      initialVersion,
		"version_desc": "Imported version",
		"created_at":   now,
		"updated_at":   now,
	}

	if err := tx.Table("plugin").Clauses(clause.OnConflict{UpdateAll: true}).Create(pluginModel).Error; err != nil {
		return err
	}

	// Create plugin_draft
	draftModel := map[string]interface{}{
		"id":           newID,
		"space_id":     importCtx.TargetSpaceID,
		"developer_id": importCtx.UserID,
		"app_id":       0, // No associated app for imported plugins
		"icon_uri":     plugin.IconURI,
		"server_url":   plugin.ServerURL,
		"plugin_type":  plugin.PluginType,
		"manifest":     toJSON(plugin.Manifest),
		"openapi_doc":  toJSON(plugin.OpenapiDoc),
		"created_at":   now,
		"updated_at":   now,
	}

	if err := tx.Table("plugin_draft").Clauses(clause.OnConflict{UpdateAll: true}).Create(draftModel).Error; err != nil {
		return err
	}

	// Recreate the plugin's tools so workflow nodes that call api_id can resolve.
	// Each source tool gets a freshly generated ID; the mapping is stored on
	// importCtx so the reference rewriter can rewrite api_id values in workflows.
	for _, tool := range plugin.Tools {
		newToolID, err := s.idGen.GenID(ctx)
		if err != nil {
			return err
		}
		toolRow := map[string]interface{}{
			"id":               newToolID,
			"plugin_id":        newID,
			"sub_url":          tool.SubURL,
			"method":           tool.Method,
			"operation":        toJSON(tool.Operation),
			"activated_status": 0,
			"created_at":       now,
			"updated_at":       now,
		}
		if err := tx.Table("tool").Clauses(clause.OnConflict{UpdateAll: true}).Create(toolRow).Error; err != nil {
			return err
		}
		toolDraftRow := map[string]interface{}{
			"id":               newToolID,
			"plugin_id":        newID,
			"sub_url":          tool.SubURL,
			"method":           tool.Method,
			"operation":        toJSON(tool.Operation),
			"debug_status":     0,
			"activated_status": 0,
			"created_at":       now,
			"updated_at":       now,
		}
		if err := tx.Table("tool_draft").Clauses(clause.OnConflict{UpdateAll: true}).Create(toolDraftRow).Error; err != nil {
			return err
		}
		importCtx.ToolIDMap[tool.ToolID] = newToolID
	}

	return nil
}

// createWorkflow creates a workflow in the database
func (s *SpaceImporter) createWorkflow(ctx context.Context, tx *gorm.DB, workflow *export.ExportedWorkflow, importCtx *ImportContext) error {
	newID := importCtx.WorkflowIDMap[workflow.ID]
	now := time.Now().UnixMilli()

	// Initial version for imported workflows
	initialVersion := "v1.0.0"

	// Create workflow_meta with latest_version set
	metaModel := map[string]interface{}{
		"id":                newID,
		"space_id":          importCtx.TargetSpaceID,
		"name":              workflow.Name,
		"description":       workflow.Desc,
		"icon_uri":          workflow.IconURI,
		"mode":              workflow.Mode,
		"content_type":      0, // Default content type
		"status":            0, // Unpublished
		"app_id":            0, // Library workflow (app_id=0 means it's in the library, not bound to a project)
		"creator_id":        importCtx.UserID,
		"author_id":         importCtx.UserID,
		"updater_id":        importCtx.UserID,
		"latest_version":    initialVersion,
		"latest_version_ts": now,
		"created_at":        now,
		"updated_at":        now,
	}

	if err := tx.Table("workflow_meta").Clauses(clause.OnConflict{UpdateAll: true}).Create(metaModel).Error; err != nil {
		return err
	}

	// Create workflow_version (required for workflow queries that JOIN with workflow_meta)
	versionModel := map[string]interface{}{
		"workflow_id":         newID,
		"version":             initialVersion,
		"version_description": "Imported version",
		"canvas":              toJSON(workflow.Canvas),
		"input_params":        toJSON(workflow.InputParams),
		"output_params":       toJSON(workflow.OutputParams),
		"creator_id":          importCtx.UserID,
		"created_at":          now,
		"commit_id":           "",
	}

	if err := tx.Table("workflow_version").Clauses(clause.OnConflict{UpdateAll: true}).Create(versionModel).Error; err != nil {
		return err
	}

	// Create workflow_draft
	draftModel := map[string]interface{}{
		"id":               newID,
		"canvas":           toJSON(workflow.Canvas),
		"input_params":     toJSON(workflow.InputParams),
		"output_params":    toJSON(workflow.OutputParams),
		"test_run_success": 0,
		"modified":         0,
		"commit_id":        "",
		"updated_at":       now,
	}

	return tx.Table("workflow_draft").Clauses(clause.OnConflict{UpdateAll: true}).Create(draftModel).Error
}

// createAgent creates an agent in the database
func (s *SpaceImporter) createAgent(ctx context.Context, tx *gorm.DB, agent *export.ExportedAgent, importCtx *ImportContext) error {
	newID := importCtx.AgentIDMap[agent.ID]
	now := time.Now().UnixMilli()

	// Log workflow refs before saving
	if len(agent.WorkflowRefs) > 0 {
		for i, wf := range agent.WorkflowRefs {
			logs.CtxDebugf(ctx, "Agent %s workflow ref[%d]: WorkflowId=%d, PluginId=%d, ApiId=%d, Name=%s",
				agent.Name, i, wf.GetWorkflowId(), wf.GetPluginId(), wf.GetApiId(), wf.GetWorkflowName())
		}
	} else {
		logs.CtxDebugf(ctx, "Agent %s has no workflow refs after rewrite", agent.Name)
	}

	// Log the JSON that will be saved
	workflowJSON := toJSON(agent.WorkflowRefs)
	logs.CtxDebugf(ctx, "Agent %s workflow JSON to save: %v", agent.Name, workflowJSON)

	// Update variables_meta_id if it was remapped
	var variablesMetaID *int64
	if agent.VariablesMetaID != nil && *agent.VariablesMetaID != 0 {
		variablesMetaID = agent.VariablesMetaID
	}

	model := map[string]interface{}{
		"agent_id":                   newID,
		"creator_id":                 importCtx.UserID,
		"space_id":                   importCtx.TargetSpaceID,
		"name":                       agent.Name,
		"description":                agent.Desc,
		"icon_uri":                   agent.IconURI,
		"bot_mode":                   agent.BotMode,
		"variables_meta_id":          variablesMetaID,
		"onboarding_info":            toJSON(agent.OnboardingInfo),
		"model_info":                 toJSON(agent.ModelInfo),
		"prompt":                     toJSON(agent.Prompt),
		"plugin":                     toJSON(agent.PluginRefs),
		"knowledge":                  toJSON(agent.KnowledgeRefs),
		"workflow":                   toJSON(agent.WorkflowRefs),
		"database_config":            toJSON(agent.DatabaseRefs),
		"external_knowledge":         toJSON(agent.ExternalKnowledge),
		"suggest_reply":              toJSON(agent.SuggestReply),
		"jump_config":                toJSON(agent.JumpConfig),
		"background_image_info_list": toJSON(agent.BackgroundImageList),
		"shortcut_command":           toJSON(agent.ShortcutCommand),
		"layout_info":                toJSON(agent.LayoutInfo),
		"memory_tool_config":         toJSON(agent.MemoryToolConfig),
		"created_at":                 now,
		"updated_at":                 now,
	}

	if err := tx.Table("single_agent_draft").Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error; err != nil {
		return err
	}

	// Create agent tools
	for _, tool := range agent.AgentTools {
		// Generate new tool ID
		toolNewID, err := s.idGen.GenID(ctx)
		if err != nil {
			return err
		}

		// Remap plugin_id to new ID if in our mapping
		pluginNewID := tool.PluginID
		if newPluginID, ok := importCtx.PluginIDMap[tool.PluginID]; ok {
			pluginNewID = newPluginID
		}

		toolModel := map[string]interface{}{
			"id":           toolNewID,
			"agent_id":     newID,
			"tool_id":      tool.ToolID,
			"plugin_id":    pluginNewID,
			"tool_name":    tool.ToolName,
			"sub_url":      tool.SubURL,
			"method":       tool.Method,
			"tool_version": tool.ToolVersion,
			"operation":    toJSON(tool.Operation),
			"created_at":   now,
		}

		if err := tx.Table("agent_tool_draft").Clauses(clause.OnConflict{UpdateAll: true}).Create(toolModel).Error; err != nil {
			return err
		}
	}

	// Create agent_tool_draft for builtin plugin references (from PluginRefs)
	// This is required for the agent to display plugins correctly
	for _, pluginRef := range agent.PluginRefs {
		toolID := pluginRef.GetApiId()
		pluginID := pluginRef.GetPluginId()

		// Only handle builtin plugins (plugin_id < 1000)
		// Custom plugins are already handled by AgentTools above
		if pluginID <= 0 || pluginID >= 1000 {
			continue
		}

		// Get tool info from plugin config
		toolInfo, exists := pluginConf.GetToolProduct(toolID)
		if !exists {
			logs.CtxWarnf(ctx, "Builtin tool not found in config: tool_id=%d, plugin_id=%d", toolID, pluginID)
			continue
		}

		// Generate new ID for agent_tool_draft
		draftID, err := s.idGen.GenID(ctx)
		if err != nil {
			return err
		}

		builtinToolModel := map[string]interface{}{
			"id":           draftID,
			"agent_id":     newID,
			"tool_id":      toolID,
			"plugin_id":    pluginID,
			"tool_name":    toolInfo.Info.GetName(),
			"sub_url":      toolInfo.Info.GetSubURL(),
			"method":       toolInfo.Info.GetMethod(),
			"tool_version": toolInfo.Info.GetVersion(),
			"operation":    toJSON(toolInfo.Info.Operation),
			"created_at":   now,
		}

		if err := tx.Table("agent_tool_draft").Clauses(clause.OnConflict{UpdateAll: true}).Create(builtinToolModel).Error; err != nil {
			logs.CtxWarnf(ctx, "Failed to create agent_tool_draft for builtin plugin: %v", err)
			// Continue with other plugins instead of failing the entire import
			continue
		}

		logs.CtxDebugf(ctx, "Created agent_tool_draft for builtin plugin: agent_id=%d, tool_id=%d, plugin_id=%d, tool_name=%s",
			newID, toolID, pluginID, toolInfo.Info.GetName())
	}

	return nil
}

// createVariable creates a variable in the database
func (s *SpaceImporter) createVariable(ctx context.Context, tx *gorm.DB, variable *export.ExportedVariable, importCtx *ImportContext) error {
	newID := importCtx.VariableIDMap[variable.ID]
	now := time.Now().UnixMilli()

	// Remap biz_id (agent_id) to new ID
	bizID := variable.BizID
	if variable.BizType == 1 { // 1 = agent
		// Parse old agent ID and remap to new ID
		oldAgentID, err := strconv.ParseInt(variable.BizID, 10, 64)
		if err == nil {
			if newAgentID, ok := importCtx.AgentIDMap[oldAgentID]; ok {
				bizID = strconv.FormatInt(newAgentID, 10)
			}
		}
	}

	// Check if variable with same biz_id + biz_type already exists
	// This handles cases where multiple variables in export have the same biz_id
	var existingCount int64
	tx.Table("variables_meta").
		Where("biz_id = ? AND biz_type = ? AND version = ?", bizID, variable.BizType, "").
		Count(&existingCount)

	if existingCount > 0 {
		// Skip duplicate variable - already created for this agent
		logs.CtxWarnf(ctx, "Skipping duplicate variable for biz_id=%s, biz_type=%d", bizID, variable.BizType)
		return nil
	}

	model := map[string]interface{}{
		"id":            newID,
		"biz_type":      variable.BizType,
		"biz_id":        bizID, // varchar(128)
		"creator_id":    importCtx.UserID,
		"variable_list": toJSON(variable.VariableList),
		"version":       "", // Empty version for imported variables
		"created_at":    now,
		"updated_at":    now,
	}

	return tx.Table("variables_meta").Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error
}

// createSpaceModel creates a space model in the database
// Supports cross-system import by matching model_entity by name if ID doesn't exist
func (s *SpaceImporter) createSpaceModel(ctx context.Context, tx *gorm.DB, spaceModel *export.ExportedSpaceModel, importCtx *ImportContext) error {
	newID := importCtx.SpaceModelIDMap[spaceModel.ID]
	now := time.Now().UnixMilli()

	// Resolve model_entity_id for cross-system compatibility
	modelEntityID, err := s.resolveModelEntityID(ctx, tx, spaceModel)
	if err != nil {
		logs.CtxWarnf(ctx, "Failed to resolve model_entity_id for space_model %d (name=%s): %v",
			spaceModel.ID, spaceModel.ModelEntityName, err)
		// Skip this space_model but don't fail the import
		return nil
	}

	if modelEntityID == 0 {
		logs.CtxWarnf(ctx, "No matching model_entity found for space_model %d (original_entity_id=%d, name=%s), skipping",
			spaceModel.ID, spaceModel.ModelEntityID, spaceModel.ModelEntityName)
		return nil
	}

	model := map[string]interface{}{
		"id":              newID,
		"space_id":        importCtx.TargetSpaceID,
		"model_entity_id": modelEntityID,
		"user_id":         importCtx.UserID,
		"status":          spaceModel.Status,
		"custom_config":   toJSON(spaceModel.CustomConfig),
		"created_at":      now,
		"updated_at":      now,
	}

	logs.CtxInfof(ctx, "Creating space_model: old_id=%d, new_id=%d, model_entity_id=%d (original=%d)",
		spaceModel.ID, newID, modelEntityID, spaceModel.ModelEntityID)

	// Try insert; on (space_id, model_entity_id) collision (re-import to a target
	// that already has this model) reuse the existing space_model id so subsequent
	// agent rewrites point at the right record. Without this the second import of
	// a published version fails with `space_model.uniq_space_model` duplicate.
	if err := tx.Table("space_model").Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error; err != nil {
		if !isDuplicateKeyErr(err) {
			return err
		}
		var existingID int64
		findErr := tx.Table("space_model").
			Where("space_id = ? AND model_entity_id = ?", importCtx.TargetSpaceID, modelEntityID).
			Select("id").
			Take(&existingID).Error
		if findErr != nil {
			return fmt.Errorf("space_model duplicate but lookup failed: %w", findErr)
		}
		logs.CtxInfof(ctx, "space_model already exists for entity=%d, reusing id=%d (was going to create %d)",
			modelEntityID, existingID, newID)
		importCtx.SpaceModelIDMap[spaceModel.ID] = existingID
		importCtx.MarkSpaceModelCreated(existingID)
		return nil
	}

	// Mark this space model as actually created
	importCtx.MarkSpaceModelCreated(newID)
	return nil
}

// isDuplicateKeyErr matches MySQL/MariaDB duplicate key errors. We can't rely
// on errors.Is/As across drivers, so pattern-match the message body.
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "Error 1062")
}

// createKnowledge creates a knowledge base with its documents and slices in the DB
func (s *SpaceImporter) createKnowledge(ctx context.Context, tx *gorm.DB, kb *export.ExportedKnowledge, importCtx *ImportContext) error {
	newKnowledgeID := importCtx.KnowledgeIDMap[kb.ID]
	now := time.Now().UnixMilli()

	knowledgeModel := map[string]interface{}{
		"id":          newKnowledgeID,
		"name":        kb.Name,
		"app_id":      0,
		"creator_id":  importCtx.UserID,
		"space_id":    importCtx.TargetSpaceID,
		"created_at":  now,
		"updated_at":  now,
		"status":      1,
		"description": kb.Description,
		"icon_uri":    kb.IconURI,
		"format_type": kb.FormatType,
	}
	if err := tx.Table("knowledge").Clauses(clause.OnConflict{UpdateAll: true}).Create(knowledgeModel).Error; err != nil {
		return err
	}

	for _, doc := range kb.Documents {
		if err := s.createDocument(ctx, tx, doc, newKnowledgeID, importCtx); err != nil {
			return err
		}
	}
	return nil
}

// createDocument creates a document and its slices in the DB
func (s *SpaceImporter) createDocument(ctx context.Context, tx *gorm.DB, doc *export.ExportedDocument, knowledgeID int64, importCtx *ImportContext) error {
	newDocID := importCtx.DocumentIDMap[doc.ID]
	now := time.Now().UnixMilli()

	// Get new URI from Phase 1 file upload (if available)
	newURI := ""
	if uri, ok := importCtx.FileURIMap[doc.ID]; ok {
		newURI = uri
	}

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
		"status":         1,
		"parse_rule":     toJSON(doc.ParseRule),
		"table_info":     toJSON(doc.TableInfo),
	}
	if err := tx.Table("knowledge_document").Clauses(clause.OnConflict{UpdateAll: true}).Create(docModel).Error; err != nil {
		return err
	}

	// Create slices
	for _, slice := range doc.Slices {
		newSliceID, err := s.idGen.GenID(ctx)
		if err != nil {
			return err
		}
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
			"status":       1,
		}
		if err := tx.Table("knowledge_document_slice").Clauses(clause.OnConflict{UpdateAll: true}).Create(sliceModel).Error; err != nil {
			return err
		}
	}
	return nil
}

// createFolder creates a folder in the DB
func (s *SpaceImporter) createFolder(ctx context.Context, tx *gorm.DB, folder *export.ExportedFolder, importCtx *ImportContext) error {
	newID := importCtx.FolderIDMap[folder.ID]
	now := time.Now().UnixMilli()

	parentID := folder.ParentID
	if parentID != 0 {
		if newParentID, ok := importCtx.FolderIDMap[parentID]; ok {
			parentID = newParentID
		}
	}

	model := map[string]interface{}{
		"id":         newID,
		"space_id":   importCtx.TargetSpaceID,
		"parent_id":  parentID,
		"name":       folder.Name,
		"creator_id": importCtx.UserID,
		"created_at": now,
		"updated_at": now,
	}
	return tx.Table("folder").Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error
}

// createFolderMapping creates a resource-to-folder mapping in the DB
func (s *SpaceImporter) createFolderMapping(ctx context.Context, tx *gorm.DB, mapping *export.ExportedFolderMapping, importCtx *ImportContext) error {
	// Remap resource ID based on type
	newResourceID := mapping.ResourceID
	switch mapping.ResourceType {
	case 1: // agent
		if id, ok := importCtx.AgentIDMap[mapping.ResourceID]; ok {
			newResourceID = id
		}
	case 2: // workflow
		if id, ok := importCtx.WorkflowIDMap[mapping.ResourceID]; ok {
			newResourceID = id
		}
	case 3: // knowledge
		if id, ok := importCtx.KnowledgeIDMap[mapping.ResourceID]; ok {
			newResourceID = id
		}
	case 5: // plugin
		if id, ok := importCtx.PluginIDMap[mapping.ResourceID]; ok {
			newResourceID = id
		}
	}

	newFolderID := mapping.FolderID
	if id, ok := importCtx.FolderIDMap[mapping.FolderID]; ok {
		newFolderID = id
	}

	newID, err := s.idGen.GenID(ctx)
	if err != nil {
		return err
	}
	model := map[string]interface{}{
		"id":            newID,
		"space_id":      importCtx.TargetSpaceID,
		"resource_id":   newResourceID,
		"resource_type": mapping.ResourceType,
		"folder_id":     newFolderID,
		"created_at":    time.Now().UnixMilli(),
	}
	return tx.Table("resource_folder_mapping").Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error
}

// getFirstSpaceModel gets the first available space_model in the target space
// Returns the space_model.id or 0 if none found
func (s *SpaceImporter) getFirstSpaceModel(ctx context.Context, tx *gorm.DB, spaceID int64) (int64, error) {
	type spaceModelRow struct {
		ID int64 `gorm:"column:id"`
	}
	var model spaceModelRow
	err := tx.Table("space_model").
		Select("id").
		Where("space_id = ?", spaceID).
		Where("deleted_at IS NULL").
		Order("created_at ASC").
		First(&model).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return model.ID, nil
}

// resolveModelEntityID resolves the model_entity_id for cross-system import
// It first tries to use the original ID, then falls back to matching by name
func (s *SpaceImporter) resolveModelEntityID(ctx context.Context, tx *gorm.DB, spaceModel *export.ExportedSpaceModel) (int64, error) {
	// First, check if the original model_entity_id exists in the target system
	var existsCount int64
	err := tx.Table("model_entity").
		Where("id = ?", spaceModel.ModelEntityID).
		Count(&existsCount).Error
	if err != nil {
		return 0, err
	}

	if existsCount > 0 {
		// Original ID exists, use it directly
		logs.CtxInfof(ctx, "Found model_entity by original ID: %d", spaceModel.ModelEntityID)
		return spaceModel.ModelEntityID, nil
	}

	// Original ID doesn't exist, try to match by name
	if spaceModel.ModelEntityName == "" {
		// No name available for matching
		logs.CtxWarnf(ctx, "model_entity_id %d not found and no name available for matching", spaceModel.ModelEntityID)
		return 0, nil
	}

	// Query model_entity by name
	type modelEntityRow struct {
		ID int64 `gorm:"column:id"`
	}
	var entity modelEntityRow
	err = tx.Table("model_entity").
		Select("id").
		Where("name = ?", spaceModel.ModelEntityName).
		First(&entity).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logs.CtxWarnf(ctx, "No model_entity found with name '%s' for cross-system matching", spaceModel.ModelEntityName)
			return 0, nil
		}
		return 0, err
	}

	logs.CtxInfof(ctx, "Matched model_entity by name '%s': original_id=%d -> new_id=%d",
		spaceModel.ModelEntityName, spaceModel.ModelEntityID, entity.ID)
	return entity.ID, nil
}

// generateToken generates a random import token
func (s *SpaceImporter) generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// cleanupExpiredTokens removes expired tokens from the cache
func (s *SpaceImporter) cleanupExpiredTokens() {
	now := time.Now()
	for token, pending := range s.pendingCache {
		if now.Sub(pending.CreatedAt) > ImportTokenTTL {
			delete(s.pendingCache, token)
		}
	}
}

// syncToES synchronizes imported resources to Elasticsearch
// This is called after successful database transaction to ensure ES consistency
func (s *SpaceImporter) syncToES(ctx context.Context, resources *export.SpaceResources, importCtx *ImportContext) {
	if s.eventBus == nil {
		logs.CtxWarnf(ctx, "EventBus is nil, skipping ES sync for imported resources")
		return
	}

	now := time.Now().UnixMilli()

	// Sync plugins to ES
	for _, plugin := range resources.Plugins {
		newID := importCtx.PluginIDMap[plugin.ID]
		err := s.eventBus.PublishResources(ctx, &searchEntity.ResourceDomainEvent{
			OpType: searchEntity.Created,
			Resource: &searchEntity.ResourceDocument{
				ResType:       resCommon.ResType_Plugin,
				ResSubType:    ptr.Of(plugin.PluginType),
				ResID:         newID,
				Name:          ptr.Of(plugin.Name),
				OwnerID:       ptr.Of(importCtx.UserID),
				SpaceID:       ptr.Of(importCtx.TargetSpaceID),
				APPID:         ptr.Of(int64(0)), // Library plugin (app_id=0)
				PublishStatus: ptr.Of(resCommon.PublishStatus_UnPublished), // Default to unpublished
				CreateTimeMS:  ptr.Of(now),
				UpdateTimeMS:  ptr.Of(now),
			},
		})
		if err != nil {
			logs.CtxErrorf(ctx, "Failed to sync plugin %d to ES: %v", newID, err)
		}
	}

	// Sync workflows to ES
	for _, workflow := range resources.Workflows {
		newID := importCtx.WorkflowIDMap[workflow.ID]
		err := s.eventBus.PublishResources(ctx, &searchEntity.ResourceDomainEvent{
			OpType: searchEntity.Created,
			Resource: &searchEntity.ResourceDocument{
				ResType:       resCommon.ResType_Workflow,
				ResID:         newID,
				Name:          ptr.Of(workflow.Name),
				OwnerID:       ptr.Of(importCtx.UserID),
				SpaceID:       ptr.Of(importCtx.TargetSpaceID),
				APPID:         ptr.Of(int64(0)), // Library workflow (app_id=0)
				PublishStatus: ptr.Of(resCommon.PublishStatus_UnPublished), // Default to unpublished
				CreateTimeMS:  ptr.Of(now),
				UpdateTimeMS:  ptr.Of(now),
			},
		})
		if err != nil {
			logs.CtxErrorf(ctx, "Failed to sync workflow %d to ES: %v", newID, err)
		}
	}

	// Sync agents to ES using ProjectEventBus
	if s.projectEventBus != nil {
		for _, agent := range resources.Agents {
			newID := importCtx.AgentIDMap[agent.ID]
			err := s.projectEventBus.PublishProject(ctx, &searchEntity.ProjectDomainEvent{
				OpType: searchEntity.Created,
				Project: &searchEntity.ProjectDocument{
					Status:  1, // Using status
					Type:    1, // Bot type
					ID:      newID,
					SpaceID: ptr.Of(importCtx.TargetSpaceID),
					OwnerID: ptr.Of(importCtx.UserID),
					Name:    ptr.Of(agent.Name),
				},
			})
			if err != nil {
				logs.CtxErrorf(ctx, "Failed to sync agent %d to ES: %v", newID, err)
			}
		}
		logs.CtxInfof(ctx, "Synced %d agents to ES", len(resources.Agents))
	} else {
		logs.CtxWarnf(ctx, "ProjectEventBus is nil, skipping agent ES sync for %d agents", len(resources.Agents))
	}
}
