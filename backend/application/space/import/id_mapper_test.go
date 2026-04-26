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
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

// fakeIDGen is a deterministic in-memory id generator for tests.
type fakeIDGen struct {
	counter int64
	failOn  int64 // if non-zero, GenID returns error after this many calls
	calls   int64
}

func (f *fakeIDGen) GenID(_ context.Context) (int64, error) {
	atomic.AddInt64(&f.calls, 1)
	if f.failOn > 0 && atomic.LoadInt64(&f.calls) > f.failOn {
		return 0, errors.New("forced gen id failure")
	}
	return atomic.AddInt64(&f.counter, 1) + 1000, nil
}

func (f *fakeIDGen) GenMultiIDs(ctx context.Context, counts int) ([]int64, error) {
	ids := make([]int64, 0, counts)
	for i := 0; i < counts; i++ {
		id, err := f.GenID(ctx)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func sampleResources() *export.SpaceResources {
	return &export.SpaceResources{
		Agents: []*export.ExportedAgent{
			{ID: 100, Name: "agent1"},
			{ID: 200, Name: "agent2"},
		},
		Plugins: []*export.ExportedPlugin{
			{ID: 300, Name: "plugin1"},
		},
		Workflows: []*export.ExportedWorkflow{
			{ID: 400, Name: "workflow1"},
		},
		Variables: []*export.ExportedVariable{
			{ID: 500},
		},
		SpaceModels: []*export.ExportedSpaceModel{
			{ID: 600},
		},
		KnowledgeBases: []*export.ExportedKnowledge{
			{
				ID:   700,
				Name: "kb1",
				Documents: []*export.ExportedDocument{
					{ID: 7001, KnowledgeID: 700, Name: "doc1"},
					{ID: 7002, KnowledgeID: 700, Name: "doc2"},
				},
			},
		},
		Folders: []*export.ExportedFolder{
			{ID: 800},
		},
		ExternalKnowledge: []*export.ExportedExternalKnowledge{
			{ID: 900},
		},
	}
}

func sampleRegistry() *export.IDRegistry {
	return &export.IDRegistry{
		Agents:            []int64{100, 200},
		Plugins:           []int64{300},
		Workflows:         []int64{400},
		Variables:         []int64{500},
		SpaceModels:       []int64{600},
		KnowledgeBases:    []int64{700},
		Documents:         []int64{7001, 7002},
		Folders:           []int64{800},
		ExternalKnowledge: []int64{900},
	}
}

func TestIDMapper_GenerateMapping_AllResourcesMapped(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})

	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)
	require.NotNil(t, importCtx)

	assert.Len(t, importCtx.AgentIDMap, 2)
	assert.Len(t, importCtx.PluginIDMap, 1)
	assert.Len(t, importCtx.WorkflowIDMap, 1)
	assert.Len(t, importCtx.VariableIDMap, 1)
	assert.Len(t, importCtx.SpaceModelIDMap, 1)
	assert.Len(t, importCtx.KnowledgeIDMap, 1)
	assert.Len(t, importCtx.DocumentIDMap, 2)
	assert.Len(t, importCtx.FolderIDMap, 1)
	assert.Len(t, importCtx.ExternalKnowledgeIDMap, 1)

	for srcID, newID := range importCtx.AgentIDMap {
		assert.NotZero(t, newID, "new ID for src %d should not be zero", srcID)
		assert.NotEqual(t, srcID, newID, "new ID should differ from old ID")
	}
}

func TestIDMapper_GenerateMapping_AllNewIDsUnique(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})

	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	seen := map[int64]string{}
	check := func(name string, m map[int64]int64) {
		for _, v := range m {
			if existing, ok := seen[v]; ok {
				t.Errorf("duplicate new ID %d generated for %s and %s", v, existing, name)
			}
			seen[v] = name
		}
	}
	check("agent", importCtx.AgentIDMap)
	check("plugin", importCtx.PluginIDMap)
	check("workflow", importCtx.WorkflowIDMap)
	check("variable", importCtx.VariableIDMap)
	check("space_model", importCtx.SpaceModelIDMap)
	check("knowledge", importCtx.KnowledgeIDMap)
	check("document", importCtx.DocumentIDMap)
	check("folder", importCtx.FolderIDMap)
	check("external_knowledge", importCtx.ExternalKnowledgeIDMap)
}

func TestIDMapper_GenerateMapping_EmptyResources(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})

	empty := &export.SpaceResources{}
	importCtx, err := mapper.GenerateMapping(ctx, empty, &export.IDRegistry{})
	require.NoError(t, err)
	require.NotNil(t, importCtx)

	assert.Empty(t, importCtx.AgentIDMap)
	assert.Empty(t, importCtx.PluginIDMap)
	assert.Empty(t, importCtx.WorkflowIDMap)
	assert.NotNil(t, importCtx.AgentIDMap, "maps should be initialized even when empty")
}

func TestIDMapper_GenerateMapping_PropagatesIDGenError(t *testing.T) {
	ctx := context.Background()
	// Generator fails after the first call → fails on the second agent.
	mapper := NewIDMapper(&fakeIDGen{failOn: 1})

	_, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.Error(t, err)
}

func TestIDMapper_GenerateMapping_RegistryAttached(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	registry := sampleRegistry()

	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), registry)
	require.NoError(t, err)

	assert.Same(t, registry, importCtx.PackageIDs, "PackageIDs should reference the supplied registry")
}

func TestIDMapper_GetNewAgentID_Hit(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})

	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	got := mapper.GetNewAgentID(importCtx, 100)
	assert.NotZero(t, got)
	assert.Equal(t, importCtx.AgentIDMap[100], got)
}

func TestIDMapper_GetNewAgentID_Miss(t *testing.T) {
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx := NewImportContext(1, 1, sampleRegistry())

	got := mapper.GetNewAgentID(importCtx, 9999)
	assert.Equal(t, int64(0), got, "missing mapping should return 0")
}

func TestIDMapper_GetNewPluginID_Hit(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	got := mapper.GetNewPluginID(importCtx, 300)
	assert.Equal(t, importCtx.PluginIDMap[300], got)
}

func TestIDMapper_GetNewWorkflowID_Hit(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	got := mapper.GetNewWorkflowID(importCtx, 400)
	assert.Equal(t, importCtx.WorkflowIDMap[400], got)
}

func TestIDMapper_GetNewVariableID_Hit(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	got := mapper.GetNewVariableID(importCtx, 500)
	assert.Equal(t, importCtx.VariableIDMap[500], got)
}

func TestIDMapper_GetNewVariableID_Miss(t *testing.T) {
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx := NewImportContext(1, 1, sampleRegistry())
	got := mapper.GetNewVariableID(importCtx, 999999)
	assert.Equal(t, int64(0), got)
}

func TestImportContext_RemapAgentID_NotInPackageReturnsZero(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	// Even though the map *might* contain the value, RemapAgentID checks IsInPackageAgent
	// against the registry — so an ID not in the registry returns 0.
	got := importCtx.RemapAgentID(8888)
	assert.Equal(t, int64(0), got)
}

func TestImportContext_RemapAgentID_HitsRegistryAndMap(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	got := importCtx.RemapAgentID(100)
	assert.NotZero(t, got)
	assert.Equal(t, importCtx.AgentIDMap[100], got)
}

func TestImportContext_GetEffectiveModelID_FallbackWhenNotInPackage(t *testing.T) {
	importCtx := NewImportContext(1, 1, &export.IDRegistry{SpaceModels: []int64{600}})
	fallback := int64(42)
	importCtx.FallbackModelID = &fallback

	got := importCtx.GetEffectiveModelID(9999) // not in package
	require.NotNil(t, got)
	assert.Equal(t, fallback, *got)
}

func TestImportContext_GetEffectiveModelID_FallbackWhenSkipped(t *testing.T) {
	importCtx := NewImportContext(1, 1, &export.IDRegistry{SpaceModels: []int64{600}})
	fallback := int64(42)
	importCtx.FallbackModelID = &fallback
	importCtx.SpaceModelIDMap[600] = 1234 // mapped, but never marked Created

	got := importCtx.GetEffectiveModelID(600)
	require.NotNil(t, got)
	assert.Equal(t, fallback, *got, "skipped space model should fall back")
}

func TestImportContext_GetEffectiveModelID_RemappedWhenCreated(t *testing.T) {
	importCtx := NewImportContext(1, 1, &export.IDRegistry{SpaceModels: []int64{600}})
	importCtx.SpaceModelIDMap[600] = 1234
	importCtx.MarkSpaceModelCreated(1234)

	got := importCtx.GetEffectiveModelID(600)
	require.NotNil(t, got)
	assert.Equal(t, int64(1234), *got)
}

func TestImportContext_RemapDocumentID_RegistryAware(t *testing.T) {
	ctx := context.Background()
	mapper := NewIDMapper(&fakeIDGen{})
	importCtx, err := mapper.GenerateMapping(ctx, sampleResources(), sampleRegistry())
	require.NoError(t, err)

	got := importCtx.RemapDocumentID(7001)
	assert.NotZero(t, got)
	assert.Equal(t, importCtx.DocumentIDMap[7001], got)

	gotMiss := importCtx.RemapDocumentID(99999)
	assert.Zero(t, gotMiss)
}
