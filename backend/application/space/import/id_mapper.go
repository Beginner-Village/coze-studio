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

// GenerateMapping generates new IDs for all resources and creates the mapping.
// existing is an optional set of pre-existing source→target mappings keyed by
// resource type ("agent", "plugin", ...). When a source ID is already mapped,
// the existing target ID is reused so re-imports update the same target rows
// instead of creating duplicates. Pass nil for first-time imports.
func (m *IDMapper) GenerateMapping(ctx context.Context, resources *export.SpaceResources, registry *export.IDRegistry, existing map[string]map[int64]int64) (*ImportContext, error) {
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
		ReusedTargetIDs:        make(map[string]map[int64]bool),
		PackageIDs:             registry,
	}

	// resolve picks an existing target ID when one is recorded, otherwise asks
	// the id generator for a fresh one. It also marks reused IDs so the
	// importer knows to UPDATE instead of INSERT for that row.
	resolve := func(resourceType, label string, sourceID int64) (int64, error) {
		if existing != nil {
			if perType, ok := existing[resourceType]; ok {
				if targetID, ok := perType[sourceID]; ok && targetID != 0 {
					if importCtx.ReusedTargetIDs[resourceType] == nil {
						importCtx.ReusedTargetIDs[resourceType] = make(map[int64]bool)
					}
					importCtx.ReusedTargetIDs[resourceType][targetID] = true
					logs.CtxDebugf(ctx, "%s reuse mapping: %d -> %d", label, sourceID, targetID)
					return targetID, nil
				}
			}
		}
		newID, err := m.idGenerator.GenID(ctx)
		if err != nil {
			return 0, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode, errorx.KV("resource", label))
		}
		logs.CtxDebugf(ctx, "%s new mapping: %d -> %d", label, sourceID, newID)
		return newID, nil
	}

	for _, agent := range resources.Agents {
		newID, err := resolve("agent", "Agent", agent.ID)
		if err != nil {
			return nil, err
		}
		importCtx.AgentIDMap[agent.ID] = newID
	}

	for _, plugin := range resources.Plugins {
		newID, err := resolve("plugin", "Plugin", plugin.ID)
		if err != nil {
			return nil, err
		}
		importCtx.PluginIDMap[plugin.ID] = newID
	}

	for _, workflow := range resources.Workflows {
		newID, err := resolve("workflow", "Workflow", workflow.ID)
		if err != nil {
			return nil, err
		}
		importCtx.WorkflowIDMap[workflow.ID] = newID
	}

	for _, variable := range resources.Variables {
		newID, err := resolve("variable", "Variable", variable.ID)
		if err != nil {
			return nil, err
		}
		importCtx.VariableIDMap[variable.ID] = newID
	}

	for _, spaceModel := range resources.SpaceModels {
		newID, err := resolve("space_model", "SpaceModel", spaceModel.ID)
		if err != nil {
			return nil, err
		}
		importCtx.SpaceModelIDMap[spaceModel.ID] = newID
	}

	for _, kb := range resources.KnowledgeBases {
		newID, err := resolve("knowledge", "KnowledgeBase", kb.ID)
		if err != nil {
			return nil, err
		}
		importCtx.KnowledgeIDMap[kb.ID] = newID

		for _, doc := range kb.Documents {
			newDocID, err := resolve("document", "Document", doc.ID)
			if err != nil {
				return nil, err
			}
			importCtx.DocumentIDMap[doc.ID] = newDocID
		}
	}

	for _, folder := range resources.Folders {
		newID, err := resolve("folder", "Folder", folder.ID)
		if err != nil {
			return nil, err
		}
		importCtx.FolderIDMap[folder.ID] = newID
	}

	for _, ek := range resources.ExternalKnowledge {
		newID, err := resolve("external_knowledge", "ExternalKnowledge", ek.ID)
		if err != nil {
			return nil, err
		}
		importCtx.ExternalKnowledgeIDMap[ek.ID] = newID
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
