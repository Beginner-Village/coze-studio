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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

func ptrInt64(v int64) *int64       { return &v }
func ptrString(v string) *string    { return &v }

func newImportContextWithMaps(t *testing.T) *ImportContext {
	t.Helper()
	registry := &export.IDRegistry{
		Agents:    []int64{100},
		Plugins:   []int64{300, 301},
		Workflows: []int64{400},
		Variables: []int64{500},
		KnowledgeBases: []int64{700},
	}
	importCtx := NewImportContext(1, 1, registry)
	importCtx.AgentIDMap[100] = 100100
	importCtx.PluginIDMap[300] = 300300
	importCtx.PluginIDMap[301] = 301301
	importCtx.WorkflowIDMap[400] = 400400
	importCtx.VariableIDMap[500] = 500500
	importCtx.KnowledgeIDMap[700] = 700700
	return importCtx
}

func TestRewriter_RewriteAgent_PluginRefs_RemapInPackage(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID:   100,
		Name: "agent1",
		PluginRefs: []*bot_common.PluginInfo{
			{PluginId: ptrInt64(300), ApiId: ptrInt64(300)},
		},
	}

	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.Len(t, got.PluginRefs, 1)
	assert.Equal(t, int64(300300), got.PluginRefs[0].GetPluginId(), "plugin_id should be remapped")
	assert.Equal(t, int64(300300), got.PluginRefs[0].GetApiId(), "api_id should be remapped")
}

func TestRewriter_RewriteAgent_PluginRefs_BuiltinPreserved(t *testing.T) {
	// Built-in plugins (id<1000) should keep their original IDs.
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID: 100,
		PluginRefs: []*bot_common.PluginInfo{
			// pluginID=4 (Wolfram Alpha-like builtin), api_id is also small
			{PluginId: ptrInt64(4), ApiId: ptrInt64(4)},
		},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.Len(t, got.PluginRefs, 1)
	assert.Equal(t, int64(4), got.PluginRefs[0].GetPluginId(), "builtin plugin should be untouched")
	assert.Equal(t, int64(4), got.PluginRefs[0].GetApiId())
}

func TestRewriter_RewriteAgent_PluginRefs_OutOfPackageCleared(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID: 100,
		PluginRefs: []*bot_common.PluginInfo{
			// Plugin neither in package nor builtin
			{PluginId: ptrInt64(99999), ApiId: ptrInt64(99999)},
		},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.Len(t, got.PluginRefs, 1)
	assert.Equal(t, int64(0), got.PluginRefs[0].GetPluginId(), "out-of-package plugin should be cleared to 0")
	assert.Equal(t, int64(0), got.PluginRefs[0].GetApiId())
}

func TestRewriter_RewriteAgent_WorkflowRefs_RemapInPackage(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID: 100,
		WorkflowRefs: []*bot_common.WorkflowInfo{
			{WorkflowId: ptrInt64(400), PluginId: ptrInt64(400), ApiId: ptrInt64(400)},
		},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.Len(t, got.WorkflowRefs, 1)
	assert.Equal(t, int64(400400), got.WorkflowRefs[0].GetWorkflowId())
	assert.Equal(t, int64(400400), got.WorkflowRefs[0].GetPluginId())
	assert.Equal(t, int64(400400), got.WorkflowRefs[0].GetApiId())
}

func TestRewriter_RewriteAgent_WorkflowRefs_OutOfPackageRemoved(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID: 100,
		WorkflowRefs: []*bot_common.WorkflowInfo{
			{WorkflowId: ptrInt64(400)},   // in package
			{WorkflowId: ptrInt64(99999)}, // out of package
		},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.Len(t, got.WorkflowRefs, 1, "out-of-package workflow refs should be removed entirely")
	assert.Equal(t, int64(400400), got.WorkflowRefs[0].GetWorkflowId())
}

func TestRewriter_RewriteAgent_KnowledgeRefs_RemapAndDrop(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID: 100,
		KnowledgeRefs: &bot_common.Knowledge{
			KnowledgeInfo: []*bot_common.KnowledgeInfo{
				{Id: ptrString("700")},   // in package
				{Id: ptrString("99999")}, // out of package
				{Id: ptrString("")},      // skipped (empty)
				{Id: ptrString("not-a-number")},
			},
		},
	}

	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.NotNil(t, got.KnowledgeRefs)
	require.Len(t, got.KnowledgeRefs.KnowledgeInfo, 1, "only the in-package knowledge should remain")
	assert.Equal(t, "700700", got.KnowledgeRefs.KnowledgeInfo[0].GetId())
}

func TestRewriter_RewriteAgent_VariablesMetaID_InPackage(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()
	agent := &export.ExportedAgent{
		ID:              100,
		VariablesMetaID: ptrInt64(500),
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.NotNil(t, got.VariablesMetaID)
	assert.Equal(t, int64(500500), *got.VariablesMetaID)
}

func TestRewriter_RewriteAgent_VariablesMetaID_OutOfPackageCleared(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()
	agent := &export.ExportedAgent{
		ID:              100,
		VariablesMetaID: ptrInt64(99999),
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	assert.Nil(t, got.VariablesMetaID, "out-of-package variables_meta_id should be cleared")
}

func TestRewriter_RewriteAgent_AgentTools_OnlyInPackagePluginsKept(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	agent := &export.ExportedAgent{
		ID: 100,
		AgentTools: []*export.ExportedAgentTool{
			{PluginID: 300, ToolName: "in"},
			{PluginID: 99999, ToolName: "out"},
		},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.Len(t, got.AgentTools, 1)
	assert.Equal(t, int64(300300), got.AgentTools[0].PluginID)
	assert.Equal(t, "in", got.AgentTools[0].ToolName)
}

func TestRewriter_RewriteAgent_DatabaseRefs_AlwaysCleared(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()
	agent := &export.ExportedAgent{
		ID:           100,
		DatabaseRefs: []*bot_common.Database{{}},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	assert.Nil(t, got.DatabaseRefs, "database refs should always be cleared (not exported)")
}

func TestRewriter_RewriteAgent_ModelInfo_RemapWhenCreated(t *testing.T) {
	registry := &export.IDRegistry{SpaceModels: []int64{600}, Agents: []int64{100}}
	importCtx := NewImportContext(1, 1, registry)
	importCtx.SpaceModelIDMap[600] = 600600
	importCtx.MarkSpaceModelCreated(600600)

	r := NewReferenceRewriter()
	agent := &export.ExportedAgent{
		ID:        100,
		ModelInfo: &bot_common.ModelInfo{ModelId: ptrInt64(600)},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.NotNil(t, got.ModelInfo.ModelId)
	assert.Equal(t, int64(600600), *got.ModelInfo.ModelId)
}

func TestRewriter_RewriteAgent_ModelInfo_FallbackWhenSkipped(t *testing.T) {
	registry := &export.IDRegistry{SpaceModels: []int64{600}, Agents: []int64{100}}
	importCtx := NewImportContext(1, 1, registry)
	importCtx.SpaceModelIDMap[600] = 600600
	// not marked as created → skipped
	fallback := int64(999)
	importCtx.FallbackModelID = &fallback

	r := NewReferenceRewriter()
	agent := &export.ExportedAgent{
		ID:        100,
		ModelInfo: &bot_common.ModelInfo{ModelId: ptrInt64(600)},
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	require.NotNil(t, got.ModelInfo.ModelId)
	assert.Equal(t, fallback, *got.ModelInfo.ModelId)
}

func TestRewriter_RewriteAgent_ModelInfo_NoFallbackClearsModelID(t *testing.T) {
	registry := &export.IDRegistry{SpaceModels: []int64{}, Agents: []int64{100}}
	importCtx := NewImportContext(1, 1, registry)

	r := NewReferenceRewriter()
	agent := &export.ExportedAgent{
		ID:        100,
		ModelInfo: &bot_common.ModelInfo{ModelId: ptrInt64(99999)}, // not in package, no fallback
	}
	got := r.RewriteAgent(context.Background(), agent, importCtx)
	assert.Nil(t, got.ModelInfo.ModelId, "no fallback → ModelId should be cleared")
}

func TestRewriter_RewriteWorkflow_NilCanvasNoop(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	wf := &export.ExportedWorkflow{ID: 400, Canvas: nil}
	got := r.RewriteWorkflow(context.Background(), wf, importCtx)
	assert.Nil(t, got.Canvas)
}

func TestRewriter_RewriteWorkflow_RemapsNodeReferences(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	canvas := map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{
				"data": map[string]interface{}{
					"agent_id":     int64(100),
					"plugin_id":    int64(300),
					"workflow_id":  int64(400),
					"knowledge_id": int64(700),
					"database_id":  int64(8888),
				},
			},
		},
	}

	wf := &export.ExportedWorkflow{ID: 400, Canvas: canvas}
	got := r.RewriteWorkflow(context.Background(), wf, importCtx)

	rewritten, ok := got.Canvas.(map[string]interface{})
	require.True(t, ok)
	nodes := rewritten["nodes"].([]interface{})
	data := nodes[0].(map[string]interface{})["data"].(map[string]interface{})

	// agent_id, plugin_id, workflow_id, knowledge_id should be remapped
	// database_id should be cleared to 0
	assert.Equal(t, int64(100100), toInt64(t, data["agent_id"]))
	assert.Equal(t, int64(300300), toInt64(t, data["plugin_id"]))
	assert.Equal(t, int64(400400), toInt64(t, data["workflow_id"]))
	assert.Equal(t, int64(700700), toInt64(t, data["knowledge_id"]))
	assert.Equal(t, int64(0), toInt64(t, data["database_id"]))
}

func TestRewriter_RewriteWorkflow_OutOfPackageNodeRefsClearedToZero(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	canvas := map[string]interface{}{
		"nodes": []interface{}{
			map[string]interface{}{
				"data": map[string]interface{}{
					"agent_id":    int64(99999),
					"plugin_id":   int64(99999),
					"workflow_id": int64(99999),
				},
			},
		},
	}
	wf := &export.ExportedWorkflow{ID: 400, Canvas: canvas}
	got := r.RewriteWorkflow(context.Background(), wf, importCtx)

	rewritten, _ := got.Canvas.(map[string]interface{})
	nodes := rewritten["nodes"].([]interface{})
	data := nodes[0].(map[string]interface{})["data"].(map[string]interface{})
	assert.Equal(t, int64(0), toInt64(t, data["agent_id"]))
	assert.Equal(t, int64(0), toInt64(t, data["plugin_id"]))
	assert.Equal(t, int64(0), toInt64(t, data["workflow_id"]))
}

func TestRewriter_RewriteVariable_AgentBizIDRemapped(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()

	v := &export.ExportedVariable{
		ID:      500,
		BizType: 1, // agent
		BizID:   strconv.FormatInt(100, 10),
	}
	got := r.RewriteVariable(context.Background(), v, importCtx)
	assert.Equal(t, "100100", got.BizID)
}

func TestRewriter_RewriteVariable_NonAgentLeavesBizIDAlone(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()
	v := &export.ExportedVariable{
		ID:      500,
		BizType: 2, // not agent
		BizID:   "some-other-id",
	}
	got := r.RewriteVariable(context.Background(), v, importCtx)
	assert.Equal(t, "some-other-id", got.BizID)
}

func TestRewriter_RewritePlugin_NoOp(t *testing.T) {
	importCtx := newImportContextWithMaps(t)
	r := NewReferenceRewriter()
	p := &export.ExportedPlugin{ID: 300, Name: "p"}
	got := r.RewritePlugin(context.Background(), p, importCtx)
	assert.Same(t, p, got, "RewritePlugin should return the same pointer (no-op)")
}

// toInt64 is a tiny test helper to extract an int64 from interface{} after JSON round-trip.
func toInt64(t *testing.T, v interface{}) int64 {
	t.Helper()
	got, ok := getInt64FromInterface(v)
	require.True(t, ok, "expected int64-convertible value, got %#v", v)
	return got
}
