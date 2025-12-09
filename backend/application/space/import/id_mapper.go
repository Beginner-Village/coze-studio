/*
 * Copyright 2025 coze-dev Authors
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

	"github.com/coze-dev/coze-studio/backend/application/space/export"
	"github.com/coze-dev/coze-studio/backend/infra/contract/idgen"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/types/errno"
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
		AgentIDMap:           make(map[int64]int64),
		PluginIDMap:          make(map[int64]int64),
		WorkflowIDMap:        make(map[int64]int64),
		VariableIDMap:        make(map[int64]int64),
		SpaceModelIDMap:      make(map[int64]int64),
		CreatedSpaceModelIDs: make(map[int64]bool),
		PackageIDs:           registry,
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

	logs.CtxInfof(ctx, "Generated ID mappings: agents=%d, plugins=%d, workflows=%d, variables=%d, space_models=%d",
		len(importCtx.AgentIDMap), len(importCtx.PluginIDMap),
		len(importCtx.WorkflowIDMap), len(importCtx.VariableIDMap), len(importCtx.SpaceModelIDMap))

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
