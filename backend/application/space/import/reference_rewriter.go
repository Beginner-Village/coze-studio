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
	"bytes"
	"context"
	"encoding/json"
	"strconv"

	"github.com/bytedance/sonic"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// ReferenceRewriter handles rewriting of resource references during import
type ReferenceRewriter struct{}

// NewReferenceRewriter creates a new ReferenceRewriter
func NewReferenceRewriter() *ReferenceRewriter {
	return &ReferenceRewriter{}
}

// RewriteAgent rewrites all references in an agent
func (r *ReferenceRewriter) RewriteAgent(ctx context.Context, agent *export.ExportedAgent, importCtx *ImportContext) *export.ExportedAgent {
	// DEBUG: Print critical info at start
	logs.CtxInfof(ctx, "========== RewriteAgent START ==========")
	logs.CtxInfof(ctx, "Agent: ID=%d, Name=%s", agent.ID, agent.Name)
	logs.CtxInfof(ctx, "Agent WorkflowRefs count: %d", len(agent.WorkflowRefs))
	if importCtx.PackageIDs != nil {
		logs.CtxInfof(ctx, "PackageIDs.Workflows count: %d", len(importCtx.PackageIDs.Workflows))
	} else {
		logs.CtxErrorf(ctx, "PackageIDs is NIL!")
	}
	logs.CtxInfof(ctx, "WorkflowIDMap count: %d", len(importCtx.WorkflowIDMap))

	// Rewrite plugin references. Each ref carries both PluginId and ApiId
	// (the ApiId is the tool inside the plugin). Remap them independently:
	// PluginId via the plugin map, ApiId via the tool map (populated during
	// plugin import). For built-in plugins both IDs are kept as-is so the
	// target system, which ships the same built-ins, can still resolve them.
	if agent.PluginRefs != nil {
		for _, pluginRef := range agent.PluginRefs {
			oldApiID := pluginRef.GetApiId()
			oldPluginID := pluginRef.GetPluginId()
			if importCtx.IsInPackagePlugin(oldPluginID) {
				newPluginID := importCtx.RemapPluginID(oldPluginID)
				pluginRef.PluginId = &newPluginID
				if newToolID, mapped := importCtx.RemapToolID(oldApiID); mapped {
					pluginRef.ApiId = &newToolID
				} else {
					var zero int64 = 0
					pluginRef.ApiId = &zero
					logs.CtxWarnf(ctx, "Plugin %d in-package but tool %d not mapped; clearing api_id", oldPluginID, oldApiID)
				}
			} else if isBuiltinPlugin(oldPluginID) {
				logs.CtxDebugf(ctx, "Keeping built-in plugin reference: plugin_id=%d, api_id=%d", oldPluginID, oldApiID)
			} else {
				logs.CtxWarnf(ctx, "Clearing out-of-package plugin reference: plugin_id=%d, api_id=%d", oldPluginID, oldApiID)
				var zero int64 = 0
				pluginRef.ApiId = &zero
				pluginRef.PluginId = &zero
			}
		}
	}

	// Rewrite workflow references
	if agent.WorkflowRefs != nil {
		logs.CtxDebugf(ctx, "Agent %d has %d workflow refs, package has %d workflows",
			agent.ID, len(agent.WorkflowRefs), len(importCtx.PackageIDs.Workflows))

		validWorkflowRefs := make([]*bot_common.WorkflowInfo, 0, len(agent.WorkflowRefs))
		for _, workflowRef := range agent.WorkflowRefs {
			oldID := workflowRef.GetWorkflowId()
			logs.CtxDebugf(ctx, "Processing workflow ref: oldID=%d, isInPackage=%v",
				oldID, importCtx.IsInPackageWorkflow(oldID))

			if importCtx.IsInPackageWorkflow(oldID) {
				newID := importCtx.RemapWorkflowID(oldID)
				logs.CtxDebugf(ctx, "Remapping workflow: %d -> %d", oldID, newID)
				workflowRef.WorkflowId = &newID
				// Also update PluginId and ApiId if they match the old workflow ID
				// In workflow refs, these fields often mirror the workflow_id
				if workflowRef.GetPluginId() == oldID {
					workflowRef.PluginId = &newID
				}
				if workflowRef.GetApiId() == oldID {
					workflowRef.ApiId = &newID
				}
				validWorkflowRefs = append(validWorkflowRefs, workflowRef)
			} else if oldID != 0 {
				// Log out-of-package workflow reference for debugging
				logs.CtxWarnf(ctx, "Workflow %d not in package for agent %d, will be removed. Package workflows: %v",
					oldID, agent.ID, importCtx.PackageIDs.Workflows)
			}
			// Skip invalid or out-of-package workflow refs (don't add to validWorkflowRefs)
		}
		// Replace with only valid workflow refs (removes invalid ones instead of setting to 0)
		agent.WorkflowRefs = validWorkflowRefs
		logs.CtxDebugf(ctx, "Agent %d workflow refs after rewrite: %d", agent.ID, len(agent.WorkflowRefs))
	}

	// Rewrite knowledge references
	if agent.KnowledgeRefs != nil && agent.KnowledgeRefs.KnowledgeInfo != nil {
		validKnowledgeInfo := make([]*bot_common.KnowledgeInfo, 0, len(agent.KnowledgeRefs.KnowledgeInfo))
		for _, ki := range agent.KnowledgeRefs.KnowledgeInfo {
			oldIDStr := ki.GetId()
			if oldIDStr == "" {
				continue
			}
			oldID, err := strconv.ParseInt(oldIDStr, 10, 64)
			if err != nil {
				logs.CtxWarnf(ctx, "Failed to parse knowledge_info id %q: %v", oldIDStr, err)
				continue
			}
			if importCtx.IsInPackageKnowledge(oldID) {
				newID := importCtx.RemapKnowledgeID(oldID)
				newIDStr := strconv.FormatInt(newID, 10)
				ki.Id = &newIDStr
				validKnowledgeInfo = append(validKnowledgeInfo, ki)
				logs.CtxDebugf(ctx, "Remapped knowledge ref: %d -> %d", oldID, newID)
			} else {
				logs.CtxWarnf(ctx, "Removing out-of-package knowledge ref %d from agent %d", oldID, agent.ID)
			}
		}
		agent.KnowledgeRefs.KnowledgeInfo = validKnowledgeInfo
	}

	// Keep external knowledge references as-is (exported with the package)
	// agent.ExternalKnowledge is preserved

	// Clear database references (not exported)
	agent.DatabaseRefs = nil

	// Rewrite variables_meta_id
	if agent.VariablesMetaID != nil {
		oldID := *agent.VariablesMetaID
		if importCtx.IsInPackageVariable(oldID) {
			newID := importCtx.RemapVariableID(oldID)
			agent.VariablesMetaID = &newID
		} else {
			agent.VariablesMetaID = nil
		}
	}

	// Rewrite agent tools
	if agent.AgentTools != nil {
		validTools := make([]*export.ExportedAgentTool, 0, len(agent.AgentTools))
		for _, tool := range agent.AgentTools {
			if importCtx.IsInPackagePlugin(tool.PluginID) {
				tool.PluginID = importCtx.RemapPluginID(tool.PluginID)
				validTools = append(validTools, tool)
			}
			// Skip tools that reference out-of-package plugins
		}
		agent.AgentTools = validTools
	}

	// Rewrite model_id in ModelInfo
	if agent.ModelInfo != nil && agent.ModelInfo.ModelId != nil {
		oldModelID := *agent.ModelInfo.ModelId
		// Use GetEffectiveModelID which handles:
		// 1. In-package models that were actually created -> remap to new ID
		// 2. In-package models that were skipped -> use fallback
		// 3. Out-of-package models -> use fallback
		effectiveModelID := importCtx.GetEffectiveModelID(oldModelID)
		if effectiveModelID != nil {
			agent.ModelInfo.ModelId = effectiveModelID
			logs.CtxDebugf(ctx, "Rewrote model_id for agent %d: %d -> %d", agent.ID, oldModelID, *effectiveModelID)
		} else {
			// No fallback available, clear model_id
			agent.ModelInfo.ModelId = nil
			logs.CtxWarnf(ctx, "No model available for agent %d (original model_id=%d), cleared model_id", agent.ID, oldModelID)
		}
	} else if agent.ModelInfo != nil && agent.ModelInfo.ModelId == nil {
		// Agent has ModelInfo but no model_id, try to set fallback
		if importCtx.FallbackModelID != nil {
			agent.ModelInfo.ModelId = importCtx.FallbackModelID
			logs.CtxInfof(ctx, "Set fallback model_id=%d for agent %d (had no model)", *importCtx.FallbackModelID, agent.ID)
		}
	}

	logs.CtxDebugf(ctx, "Rewrote references for agent %d", agent.ID)
	return agent
}

// RewritePlugin rewrites all references in a plugin
func (r *ReferenceRewriter) RewritePlugin(ctx context.Context, plugin *export.ExportedPlugin, importCtx *ImportContext) *export.ExportedPlugin {
	// Plugins typically don't have references to other resources
	// but we process them in case of future extensions
	logs.CtxDebugf(ctx, "Processed plugin %d (no references to rewrite)", plugin.ID)
	return plugin
}

// RewriteWorkflow rewrites all references in a workflow canvas
func (r *ReferenceRewriter) RewriteWorkflow(ctx context.Context, workflow *export.ExportedWorkflow, importCtx *ImportContext) *export.ExportedWorkflow {
	if workflow.Canvas == nil {
		return workflow
	}

	// Parse canvas JSON and rewrite references
	rewrittenCanvas := r.rewriteCanvasReferences(ctx, workflow.Canvas, importCtx)
	workflow.Canvas = rewrittenCanvas

	logs.CtxDebugf(ctx, "Rewrote references for workflow %d", workflow.ID)
	return workflow
}

// rewriteCanvasReferences recursively rewrites references in workflow canvas
func (r *ReferenceRewriter) rewriteCanvasReferences(ctx context.Context, canvas interface{}, importCtx *ImportContext) interface{} {
	if canvas == nil {
		return nil
	}

	// Convert to map for processing
	// Use sonic for marshaling (fast and correct)
	canvasBytes, err := sonic.Marshal(canvas)
	if err != nil {
		logs.CtxWarnf(ctx, "Failed to marshal canvas: %v", err)
		return canvas
	}

	// IMPORTANT: Use json.Decoder with UseNumber() to avoid precision loss for large int64 IDs
	// float64 can only represent integers up to 2^53 precisely, but workflow_id/plugin_id/agent_id
	// can be 18+ digit numbers that exceed this limit. Using UseNumber() keeps numbers as json.Number
	// (string representation) which preserves full precision.
	var canvasMap map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(canvasBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&canvasMap); err != nil {
		logs.CtxWarnf(ctx, "Failed to unmarshal canvas: %v", err)
		return canvas
	}

	// Rewrite nodes
	if nodes, ok := canvasMap["nodes"].([]interface{}); ok {
		for i, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				nodes[i] = r.rewriteNodeReferences(ctx, nodeMap, importCtx)
			}
		}
		canvasMap["nodes"] = nodes
	}

	return canvasMap
}

// rewriteNodeReferences rewrites references in a single workflow node
func (r *ReferenceRewriter) rewriteNodeReferences(ctx context.Context, node map[string]interface{}, importCtx *ImportContext) map[string]interface{} {
	// Get node data
	data, ok := node["data"].(map[string]interface{})
	if !ok {
		return node
	}

	// Reference fields can show up at multiple depths inside node.data
	// (e.g. data.agent_id at the top, data.inputs.fcParam.pluginFCParam.pluginList[].api_id
	// nested several levels deep inside LLM nodes). Walk the whole subtree so
	// we never miss one.
	rewriteRefsRecursive(data, importCtx)
	node["data"] = data
	return node
}

// rewriteRefsRecursive walks an arbitrary JSON tree (decoded with UseNumber so
// large IDs survive without precision loss) and rewrites any ID fields it
// recognizes. It rewrites in place and returns nothing because callers own
// the parent map/slice.
func rewriteRefsRecursive(v interface{}, importCtx *ImportContext) {
	switch val := v.(type) {
	case map[string]interface{}:
		// Replace this node's own ID fields, then descend.
		rewriteIDFields(val, importCtx)
		for _, child := range val {
			rewriteRefsRecursive(child, importCtx)
		}
	case []interface{}:
		for _, item := range val {
			rewriteRefsRecursive(item, importCtx)
		}
	}
}

// rewriteIDFields handles the actual remapping for ID fields that may exist on
// a single map. Keep this aligned with the resource types tracked in
// ImportContext.
func rewriteIDFields(data map[string]interface{}, importCtx *ImportContext) {
	if agentID, ok := getInt64FromInterface(data["agent_id"]); ok && agentID != 0 {
		if importCtx.IsInPackageAgent(agentID) {
			data["agent_id"] = importCtx.RemapAgentID(agentID)
		} else {
			data["agent_id"] = 0
		}
	}

	if pluginID, ok := getInt64FromInterface(data["plugin_id"]); ok && pluginID != 0 {
		if importCtx.IsInPackagePlugin(pluginID) {
			data["plugin_id"] = importCtx.RemapPluginID(pluginID)
		} else {
			// Built-in / out-of-package plugins: leave alone so the target
			// system's own copy (if any) can satisfy the reference. Workflows
			// targeting unknown plugins will fail at runtime rather than be
			// silently dropped.
		}
	}

	if apiID, ok := getInt64FromInterface(data["api_id"]); ok && apiID != 0 {
		if newID, mapped := importCtx.RemapToolID(apiID); mapped {
			data["api_id"] = newID
		}
		// Unmapped api_id: leave as-is for the same reason as plugin_id above.
	}

	if workflowID, ok := getInt64FromInterface(data["workflow_id"]); ok && workflowID != 0 {
		if importCtx.IsInPackageWorkflow(workflowID) {
			data["workflow_id"] = importCtx.RemapWorkflowID(workflowID)
		} else {
			data["workflow_id"] = 0
		}
	}

	if knowledgeID, ok := getInt64FromInterface(data["knowledge_id"]); ok && knowledgeID != 0 {
		if importCtx.IsInPackageKnowledge(knowledgeID) {
			data["knowledge_id"] = importCtx.RemapKnowledgeID(knowledgeID)
		} else {
			data["knowledge_id"] = 0
		}
	}

	// database_id is not exported; always zero on import.
	if _, ok := data["database_id"]; ok {
		data["database_id"] = 0
	}
}

// RewriteVariable rewrites all references in a variable
func (r *ReferenceRewriter) RewriteVariable(ctx context.Context, variable *export.ExportedVariable, importCtx *ImportContext) *export.ExportedVariable {
	// Rewrite biz_id for agent variables
	if variable.BizType == 1 { // 1 = agent
		oldBizID := variable.BizID
		// Parse string BizID to int64 for remapping
		oldAgentID, err := strconv.ParseInt(oldBizID, 10, 64)
		if err == nil && importCtx.IsInPackageAgent(oldAgentID) {
			newAgentID := importCtx.RemapAgentID(oldAgentID)
			variable.BizID = strconv.FormatInt(newAgentID, 10)
		}
	}

	logs.CtxDebugf(ctx, "Rewrote references for variable %d", variable.ID)
	return variable
}

// getInt64FromInterface safely extracts an int64 from an interface{}
// IMPORTANT: Handles precision loss for large int64 values (> 2^53) when parsed from JSON
// JSON numbers are typically parsed as float64, which can only represent integers up to 2^53 precisely
// When using json.Decoder with UseNumber(), numbers are parsed as json.Number (string) which preserves precision
func getInt64FromInterface(v interface{}) (int64, bool) {
	if v == nil {
		return 0, false
	}

	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case json.Number:
		// json.Number is used when UseNumber() is enabled on json.Decoder
		// This preserves precision for large integers (> 2^53)
		n, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	case float64:
		// WARNING: float64 can only precisely represent integers up to 2^53 (9007199254740991)
		// For larger values, precision is lost. This is a known limitation.
		// The fix is to use string representation for large IDs in JSON.
		return int64(val), true
	case float32:
		return int64(val), true
	case string:
		// Support string representation of int64 to avoid precision loss
		if val == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

// isBuiltinPlugin checks if a plugin ID represents a built-in system plugin
// Built-in plugins have small IDs (typically < 1000) assigned by the system
// Examples: plugin_id=4 is Wolfram Alpha calculator
func isBuiltinPlugin(pluginID int64) bool {
	return pluginID > 0 && pluginID < 1000
}
