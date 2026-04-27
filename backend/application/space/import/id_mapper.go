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

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// IDMapper manages the mapping from old IDs to new IDs during import
type IDMapper struct {
	idGenerator idgen.IDGenerator
}

// NewIDMapper creates a new IDMapper
func NewIDMapper(idGen idgen.IDGenerator) *IDMapper {
	return &IDMapper{
		idGenerator: idGen,
	}
}

// GenerateMapping generates new IDs for all resources and creates the mapping
func (m *IDMapper) GenerateMapping(ctx context.Context, resources *export.SpaceResources, registry *export.IDRegistry) (*ImportContext, error) {
	importCtx := &ImportContext{
		AgentIDMap:             make(map[int64]int64),
		PluginIDMap:            make(map[int64]int64),
		ToolIDMap:              make(map[int64]int64),
		WorkflowIDMap:          make(map[int64]int64),
		VariableIDMap:          make(map[int64]int64),
		SpaceModelIDMap:        make(map[int64]int64),
		CreatedSpaceModelIDs:   make(map[int64]bool),
		KnowledgeIDMap:         make(map[int64]int64),
		DocumentIDMap:          make(map[int64]int64),
		FolderIDMap:            make(map[int64]int64),
		ExternalKnowledgeIDMap: make(map[int64]int64),
		FileURIMap:             make(map[int64]string),
		PackageIDs:             registry,
	}

	// Generate new IDs for agents
	for _, agent := range resources.Agents {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "agent"))
		}
		importCtx.AgentIDMap[agent.ID] = newID
		logs.CtxDebugf(ctx, "Agent ID mapping: %d -> %d", agent.ID, newID)
	}

	// Generate new IDs for plugins
	for _, plugin := range resources.Plugins {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "plugin"))
		}
		importCtx.PluginIDMap[plugin.ID] = newID
		logs.CtxDebugf(ctx, "Plugin ID mapping: %d -> %d", plugin.ID, newID)
	}

	// Generate new IDs for workflows
	for _, workflow := range resources.Workflows {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "workflow"))
		}
		importCtx.WorkflowIDMap[workflow.ID] = newID
		logs.CtxDebugf(ctx, "Workflow ID mapping: %d -> %d", workflow.ID, newID)
	}

	// Generate new IDs for variables
	for _, variable := range resources.Variables {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "variable"))
		}
		importCtx.VariableIDMap[variable.ID] = newID
		logs.CtxDebugf(ctx, "Variable ID mapping: %d -> %d", variable.ID, newID)
	}

	// Generate new IDs for space models
	for _, spaceModel := range resources.SpaceModels {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "space_model"))
		}
		importCtx.SpaceModelIDMap[spaceModel.ID] = newID
		logs.CtxDebugf(ctx, "SpaceModel ID mapping: %d -> %d", spaceModel.ID, newID)
	}

	// Generate new IDs for knowledge bases and their documents
	for _, kb := range resources.KnowledgeBases {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "knowledge_base"))
		}
		importCtx.KnowledgeIDMap[kb.ID] = newID
		logs.CtxDebugf(ctx, "KnowledgeBase ID mapping: %d -> %d", kb.ID, newID)

		// Generate new IDs for documents within this knowledge base
		for _, doc := range kb.Documents {
			newDocID, err := m.idGenerator.GenID(ctx)
			if err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "document"))
			}
			importCtx.DocumentIDMap[doc.ID] = newDocID
			logs.CtxDebugf(ctx, "Document ID mapping: %d -> %d", doc.ID, newDocID)
		}
	}

	// Generate new IDs for folders
	for _, folder := range resources.Folders {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "folder"))
		}
		importCtx.FolderIDMap[folder.ID] = newID
		logs.CtxDebugf(ctx, "Folder ID mapping: %d -> %d", folder.ID, newID)
	}

	// Generate new IDs for external knowledge
	for _, ek := range resources.ExternalKnowledge {
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", "external_knowledge"))
		}
		importCtx.ExternalKnowledgeIDMap[ek.ID] = newID
		logs.CtxDebugf(ctx, "ExternalKnowledge ID mapping: %d -> %d", ek.ID, newID)
	}

	logs.CtxInfof(ctx, "Generated ID mappings: agents=%d, plugins=%d, workflows=%d, variables=%d, space_models=%d, knowledge=%d, documents=%d, folders=%d, external_knowledge=%d",
		len(importCtx.AgentIDMap), len(importCtx.PluginIDMap),
		len(importCtx.WorkflowIDMap), len(importCtx.VariableIDMap), len(importCtx.SpaceModelIDMap),
		len(importCtx.KnowledgeIDMap), len(importCtx.DocumentIDMap), len(importCtx.FolderIDMap), len(importCtx.ExternalKnowledgeIDMap))

	// Log package IDs for debugging
	if registry != nil {
		logs.CtxDebugf(ctx, "Package IDRegistry - Agents: %v", registry.Agents)
		logs.CtxDebugf(ctx, "Package IDRegistry - Plugins: %v", registry.Plugins)
		logs.CtxDebugf(ctx, "Package IDRegistry - Workflows: %v", registry.Workflows)
		logs.CtxDebugf(ctx, "Package IDRegistry - Variables: %v", registry.Variables)
		logs.CtxDebugf(ctx, "Package IDRegistry - SpaceModels: %v", registry.SpaceModels)
	} else {
		logs.CtxWarnf(ctx, "Package IDRegistry is nil!")
	}

	return importCtx, nil
}

// GetNewAgentID returns the new agent ID for an old agent ID
func (m *IDMapper) GetNewAgentID(importCtx *ImportContext, oldID int64) int64 {
	if newID, ok := importCtx.AgentIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// GetNewPluginID returns the new plugin ID for an old plugin ID
func (m *IDMapper) GetNewPluginID(importCtx *ImportContext, oldID int64) int64 {
	if newID, ok := importCtx.PluginIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// GetNewWorkflowID returns the new workflow ID for an old workflow ID
func (m *IDMapper) GetNewWorkflowID(importCtx *ImportContext, oldID int64) int64 {
	if newID, ok := importCtx.WorkflowIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// GetNewVariableID returns the new variable ID for an old variable ID
func (m *IDMapper) GetNewVariableID(importCtx *ImportContext, oldID int64) int64 {
	if newID, ok := importCtx.VariableIDMap[oldID]; ok {
		return newID
	}
	return 0
}
