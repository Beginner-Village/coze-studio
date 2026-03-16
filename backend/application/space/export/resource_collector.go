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

package export

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// ResourceCollector collects all resources from a space for export
type ResourceCollector struct {
	db *gorm.DB
}

// NewResourceCollector creates a new ResourceCollector
func NewResourceCollector(db *gorm.DB) *ResourceCollector {
	return &ResourceCollector{db: db}
}

// CollectAll collects all resources from a space
func (c *ResourceCollector) CollectAll(ctx context.Context, spaceID int64) (*SpaceResources, error) {
	resources := &SpaceResources{
		Agents:      make([]*ExportedAgent, 0),
		Plugins:     make([]*ExportedPlugin, 0),
		Workflows:   make([]*ExportedWorkflow, 0),
		Variables:   make([]*ExportedVariable, 0),
		SpaceModels: make([]*ExportedSpaceModel, 0),
	}

	// Collect agents
	agents, err := c.collectAgents(ctx, spaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to collect agents for space %d: %v", spaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("resource", "agents"))
	}
	resources.Agents = agents
	logs.CtxInfof(ctx, "Collected %d agents from space %d", len(agents), spaceID)

	// Collect plugins
	plugins, err := c.collectPlugins(ctx, spaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to collect plugins for space %d: %v", spaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("resource", "plugins"))
	}
	resources.Plugins = plugins
	logs.CtxInfof(ctx, "Collected %d plugins from space %d", len(plugins), spaceID)

	// Collect workflows from this space
	workflows, err := c.collectWorkflows(ctx, spaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to collect workflows for space %d: %v", spaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("resource", "workflows"))
	}
	logs.CtxInfof(ctx, "Collected %d workflows from space %d", len(workflows), spaceID)

	// Also collect workflows referenced by agents (may be from other spaces)
	referencedWorkflows, err := c.collectReferencedWorkflows(ctx, agents, workflows)
	if err != nil {
		logs.CtxWarnf(ctx, "Failed to collect referenced workflows: %v", err)
		// Continue without referenced workflows, not a critical error
	} else if len(referencedWorkflows) > 0 {
		workflows = append(workflows, referencedWorkflows...)
		logs.CtxInfof(ctx, "Collected %d additional workflows referenced by agents", len(referencedWorkflows))
	}
	resources.Workflows = workflows

	// Collect variables for all agents
	agentIDs := make([]int64, 0, len(agents))
	for _, agent := range agents {
		agentIDs = append(agentIDs, agent.ID)
	}
	variables, err := c.collectVariables(ctx, agentIDs)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to collect variables for space %d: %v", spaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("resource", "variables"))
	}
	resources.Variables = variables
	logs.CtxInfof(ctx, "Collected %d variables from space %d", len(variables), spaceID)

	// Collect space models
	spaceModels, err := c.collectSpaceModels(ctx, spaceID)
	if err != nil {
		logs.CtxErrorf(ctx, "Failed to collect space models for space %d: %v", spaceID, err)
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("resource", "space_models"))
	}
	resources.SpaceModels = spaceModels
	logs.CtxInfof(ctx, "Collected %d space models from space %d", len(spaceModels), spaceID)

	return resources, nil
}

// collectAgents collects all agent drafts from a space
func (c *ResourceCollector) collectAgents(ctx context.Context, spaceID int64) ([]*ExportedAgent, error) {
	var agentDrafts []SingleAgentDraftModel
	err := c.db.WithContext(ctx).
		Table("single_agent_draft").
		Where("space_id = ?", spaceID).
		Where("deleted_at IS NULL").
		Find(&agentDrafts).Error
	if err != nil {
		return nil, err
	}

	// Collect agent tools for each agent
	agentIDs := make([]int64, 0, len(agentDrafts))
	for _, agent := range agentDrafts {
		agentIDs = append(agentIDs, agent.AgentID)
	}

	agentToolsMap, err := c.collectAgentTools(ctx, agentIDs)
	if err != nil {
		return nil, err
	}

	agents := make([]*ExportedAgent, 0, len(agentDrafts))
	for _, agent := range agentDrafts {
		exported := c.convertAgentDraftToExported(&agent)
		if tools, ok := agentToolsMap[agent.AgentID]; ok {
			exported.AgentTools = tools
		}

		// Log workflow refs for debugging
		if len(exported.WorkflowRefs) > 0 {
			for i, wf := range exported.WorkflowRefs {
				logs.CtxDebugf(ctx, "Export: Agent %s (id=%d) workflow ref[%d]: WorkflowId=%d, Name=%s",
					exported.Name, exported.ID, i, wf.GetWorkflowId(), wf.GetWorkflowName())
			}
		} else {
			logs.CtxDebugf(ctx, "Export: Agent %s (id=%d) has no workflow refs", exported.Name, exported.ID)
		}

		agents = append(agents, exported)
	}

	return agents, nil
}

// collectAgentTools collects all agent tool bindings
func (c *ResourceCollector) collectAgentTools(ctx context.Context, agentIDs []int64) (map[int64][]*ExportedAgentTool, error) {
	if len(agentIDs) == 0 {
		return make(map[int64][]*ExportedAgentTool), nil
	}

	var tools []AgentToolDraftModel
	err := c.db.WithContext(ctx).
		Table("agent_tool_draft").
		Where("agent_id IN ?", agentIDs).
		Find(&tools).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]*ExportedAgentTool)
	for _, tool := range tools {
		exported := &ExportedAgentTool{
			ToolID:      tool.ToolID,
			PluginID:    tool.PluginID,
			ToolName:    tool.ToolName,
			SubURL:      tool.SubURL,
			Method:      tool.Method,
			ToolVersion: tool.ToolVersion,
			Operation:   tool.Operation,
		}
		result[tool.AgentID] = append(result[tool.AgentID], exported)
	}

	return result, nil
}

// collectPlugins collects all plugin drafts from a space
func (c *ResourceCollector) collectPlugins(ctx context.Context, spaceID int64) ([]*ExportedPlugin, error) {
	var pluginDrafts []PluginDraftModel
	err := c.db.WithContext(ctx).
		Table("plugin_draft").
		Where("space_id = ?", spaceID).
		Where("deleted_at IS NULL").
		Find(&pluginDrafts).Error
	if err != nil {
		return nil, err
	}

	plugins := make([]*ExportedPlugin, 0, len(pluginDrafts))
	for _, plugin := range pluginDrafts {
		plugins = append(plugins, c.convertPluginDraftToExported(&plugin))
	}

	return plugins, nil
}

// collectWorkflows collects all workflow drafts from a space
func (c *ResourceCollector) collectWorkflows(ctx context.Context, spaceID int64) ([]*ExportedWorkflow, error) {
	// First get workflow meta
	var workflowMetas []WorkflowMetaModel
	err := c.db.WithContext(ctx).
		Table("workflow_meta").
		Where("space_id = ?", spaceID).
		Where("deleted_at IS NULL").
		Find(&workflowMetas).Error
	if err != nil {
		return nil, err
	}

	if len(workflowMetas) == 0 {
		return []*ExportedWorkflow{}, nil
	}

	// Get workflow IDs
	workflowIDs := make([]int64, 0, len(workflowMetas))
	for _, meta := range workflowMetas {
		workflowIDs = append(workflowIDs, meta.ID)
	}

	// Get workflow drafts
	var workflowDrafts []WorkflowDraftModel
	err = c.db.WithContext(ctx).
		Table("workflow_draft").
		Where("id IN ?", workflowIDs).
		Where("deleted_at IS NULL").
		Find(&workflowDrafts).Error
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	draftMap := make(map[int64]*WorkflowDraftModel)
	for i := range workflowDrafts {
		draftMap[workflowDrafts[i].ID] = &workflowDrafts[i]
	}

	workflows := make([]*ExportedWorkflow, 0, len(workflowMetas))
	for _, meta := range workflowMetas {
		draft := draftMap[meta.ID]
		workflows = append(workflows, c.convertWorkflowToExported(&meta, draft))
	}

	return workflows, nil
}

// collectVariables collects all variables for the given agent IDs
func (c *ResourceCollector) collectVariables(ctx context.Context, agentIDs []int64) ([]*ExportedVariable, error) {
	if len(agentIDs) == 0 {
		return []*ExportedVariable{}, nil
	}

	// Convert int64 IDs to strings since biz_id is varchar(128)
	bizIDs := make([]string, 0, len(agentIDs))
	for _, id := range agentIDs {
		bizIDs = append(bizIDs, fmt.Sprintf("%d", id))
	}

	var variableMetas []VariablesMetaModel
	err := c.db.WithContext(ctx).
		Table("variables_meta").
		Where("biz_type = ?", 1). // 1 = agent
		Where("biz_id IN ?", bizIDs).
		Find(&variableMetas).Error
	if err != nil {
		return nil, err
	}

	variables := make([]*ExportedVariable, 0, len(variableMetas))
	for _, v := range variableMetas {
		variables = append(variables, c.convertVariableToExported(&v))
	}

	return variables, nil
}

// Helper functions for conversion

func (c *ResourceCollector) convertAgentDraftToExported(agent *SingleAgentDraftModel) *ExportedAgent {
	return &ExportedAgent{
		ID:                  agent.AgentID,
		Name:                agent.Name,
		Desc:                getStringValue(agent.Description),
		IconURI:             agent.IconURI,
		CreatedAt:           agent.CreatedAt,
		UpdatedAt:           agent.UpdatedAt,
		BotMode:             agent.BotMode,
		VariablesMetaID:     agent.VariablesMetaID,
		OnboardingInfo:      agent.OnboardingInfo,
		ModelInfo:           agent.ModelInfo,
		Prompt:              agent.Prompt,
		SuggestReply:        agent.SuggestReply,
		JumpConfig:          agent.JumpConfig,
		LayoutInfo:          agent.LayoutInfo,
		ShortcutCommand:     agent.ShortcutCommand,
		MemoryToolConfig:    agent.MemoryToolConfig,
		BackgroundImageList: agent.BackgroundImageInfoList,
		PluginRefs:          agent.Plugin,
		KnowledgeRefs:       agent.Knowledge,
		WorkflowRefs:        agent.Workflow,
		DatabaseRefs:        agent.DatabaseConfig,
		ExternalKnowledge:   agent.ExternalKnowledge,
	}
}

func (c *ResourceCollector) convertPluginDraftToExported(plugin *PluginDraftModel) *ExportedPlugin {
	// Extract name and description from manifest
	name, desc := extractPluginNameDesc(plugin.Manifest)

	return &ExportedPlugin{
		ID:         plugin.ID,
		Name:       name,
		Desc:       desc,
		IconURI:    plugin.IconURI,
		ServerURL:  plugin.ServerURL,
		PluginType: plugin.PluginType,
		CreatedAt:  plugin.CreatedAt,
		UpdatedAt:  plugin.UpdatedAt,
		Manifest:   plugin.Manifest,
		OpenapiDoc: plugin.OpenapiDoc,
	}
}

// extractPluginNameDesc extracts name and description from manifest
func extractPluginNameDesc(manifest interface{}) (string, string) {
	if manifest == nil {
		return "", ""
	}
	manifestMap, ok := manifest.(map[string]interface{})
	if !ok {
		return "", ""
	}
	name := ""
	desc := ""
	if v, ok := manifestMap["name_for_human"].(string); ok {
		name = v
	}
	if v, ok := manifestMap["description_for_human"].(string); ok {
		desc = v
	}
	return name, desc
}

func (c *ResourceCollector) convertWorkflowToExported(meta *WorkflowMetaModel, draft *WorkflowDraftModel) *ExportedWorkflow {
	exported := &ExportedWorkflow{
		ID:        meta.ID,
		Name:      meta.Name,
		Desc:      getStringValue(meta.Description),
		IconURI:   getStringValue(meta.IconURI),
		Mode:      meta.Mode,
		CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt,
	}

	if draft != nil {
		exported.Canvas = draft.Canvas
		exported.InputParams = draft.InputParams
		exported.OutputParams = draft.OutputParams
	}

	return exported
}

func (c *ResourceCollector) convertVariableToExported(v *VariablesMetaModel) *ExportedVariable {
	return &ExportedVariable{
		ID:           v.ID,
		BizType:      v.BizType,
		BizID:        v.BizID,
		CreatorID:    v.CreatorID,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
		VariableList: v.VariableList,
	}
}

// Helper function to convert *string to string
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// collectSpaceModels collects all space model configurations from a space
func (c *ResourceCollector) collectSpaceModels(ctx context.Context, spaceID int64) ([]*ExportedSpaceModel, error) {
	var spaceModels []SpaceModelModel
	err := c.db.WithContext(ctx).
		Table("space_model").
		Where("space_id = ?", spaceID).
		Where("deleted_at IS NULL").
		Find(&spaceModels).Error
	if err != nil {
		return nil, err
	}

	if len(spaceModels) == 0 {
		return []*ExportedSpaceModel{}, nil
	}

	// Collect model_entity_ids to query names
	entityIDs := make([]int64, 0, len(spaceModels))
	for _, sm := range spaceModels {
		entityIDs = append(entityIDs, sm.ModelEntityID)
	}

	// Query model_entity names for cross-system matching
	entityNameMap, err := c.getModelEntityNames(ctx, entityIDs)
	if err != nil {
		// Log error but continue without names
		logs.CtxWarnf(ctx, "Failed to get model entity names: %v", err)
		entityNameMap = make(map[int64]string)
	}

	result := make([]*ExportedSpaceModel, 0, len(spaceModels))
	for _, sm := range spaceModels {
		exported := c.convertSpaceModelToExported(&sm)
		exported.ModelEntityName = entityNameMap[sm.ModelEntityID]
		result = append(result, exported)
	}

	return result, nil
}

// getModelEntityNames queries model_entity table to get names by IDs
func (c *ResourceCollector) getModelEntityNames(ctx context.Context, entityIDs []int64) (map[int64]string, error) {
	if len(entityIDs) == 0 {
		return make(map[int64]string), nil
	}

	type modelEntityRow struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}

	var entities []modelEntityRow
	err := c.db.WithContext(ctx).
		Table("model_entity").
		Select("id, name").
		Where("id IN ?", entityIDs).
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]string, len(entities))
	for _, e := range entities {
		result[e.ID] = e.Name
	}
	return result, nil
}

func (c *ResourceCollector) convertSpaceModelToExported(sm *SpaceModelModel) *ExportedSpaceModel {
	return &ExportedSpaceModel{
		ID:            sm.ID,
		ModelEntityID: sm.ModelEntityID,
		Status:        sm.Status,
		CustomConfig:  sm.CustomConfig,
		CreatedAt:     sm.CreatedAt,
		UpdatedAt:     sm.UpdatedAt,
	}
}

// collectReferencedWorkflows collects workflows that are referenced by agents but not in the current space
func (c *ResourceCollector) collectReferencedWorkflows(ctx context.Context, agents []*ExportedAgent, existingWorkflows []*ExportedWorkflow) ([]*ExportedWorkflow, error) {
	// Build a set of already collected workflow IDs
	existingIDs := make(map[int64]bool)
	for _, wf := range existingWorkflows {
		existingIDs[wf.ID] = true
	}

	// Collect all referenced workflow IDs from agents
	referencedIDs := make(map[int64]bool)
	for _, agent := range agents {
		if agent.WorkflowRefs != nil {
			for _, wfRef := range agent.WorkflowRefs {
				wfID := wfRef.GetWorkflowId()
				if wfID != 0 && !existingIDs[wfID] {
					referencedIDs[wfID] = true
					logs.CtxDebugf(ctx, "Agent %s references workflow %d (not in current space)", agent.Name, wfID)
				}
			}
		}
	}

	if len(referencedIDs) == 0 {
		return nil, nil
	}

	// Convert to slice
	workflowIDs := make([]int64, 0, len(referencedIDs))
	for id := range referencedIDs {
		workflowIDs = append(workflowIDs, id)
	}

	logs.CtxInfof(ctx, "Found %d referenced workflows not in current space: %v", len(workflowIDs), workflowIDs)

	// Query workflow metadata
	var workflowMetas []WorkflowMetaModel
	err := c.db.WithContext(ctx).
		Table("workflow_meta").
		Where("id IN ?", workflowIDs).
		Where("deleted_at IS NULL").
		Find(&workflowMetas).Error
	if err != nil {
		return nil, err
	}

	if len(workflowMetas) == 0 {
		logs.CtxWarnf(ctx, "No workflow metadata found for referenced workflows: %v", workflowIDs)
		return nil, nil
	}

	// Get workflow drafts
	var workflowDrafts []WorkflowDraftModel
	err = c.db.WithContext(ctx).
		Table("workflow_draft").
		Where("id IN ?", workflowIDs).
		Where("deleted_at IS NULL").
		Find(&workflowDrafts).Error
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	draftMap := make(map[int64]*WorkflowDraftModel)
	for i := range workflowDrafts {
		draftMap[workflowDrafts[i].ID] = &workflowDrafts[i]
	}

	// Convert to exported format
	workflows := make([]*ExportedWorkflow, 0, len(workflowMetas))
	for _, meta := range workflowMetas {
		draft := draftMap[meta.ID]
		workflows = append(workflows, c.convertWorkflowToExported(&meta, draft))
		logs.CtxDebugf(ctx, "Collected referenced workflow: ID=%d, Name=%s, SpaceID=%d", meta.ID, meta.Name, meta.SpaceID)
	}

	return workflows, nil
}
