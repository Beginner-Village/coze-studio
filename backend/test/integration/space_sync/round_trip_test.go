// Copyright 2025 ynet-dev Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package space_sync_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
	spaceimport "github.com/ynet-dev/ynet-studio/backend/application/space/import"
)

// TestE2E_ExportImportRoundTrip exercises Serializer → Validator across the full ZIP
// pipeline. We don't write to MySQL here because Serializer/Validator are pure code,
// but we run this in the integration suite to keep the round-trip path covered alongside
// the persistence-layer E2E.
func TestE2E_ExportImportRoundTrip_FullManifest(t *testing.T) {
	ctx := context.Background()

	// Build a realistic manifest + resources covering all entity types.
	resources := &export.SpaceResources{
		Agents: []*export.ExportedAgent{
			{ID: 1001, Name: "agent_alpha", Desc: "first"},
			{ID: 1002, Name: "agent_beta", Desc: "second"},
		},
		Plugins: []*export.ExportedPlugin{
			{ID: 2001, Name: "plugin_alpha"},
		},
		Workflows: []*export.ExportedWorkflow{
			{ID: 3001, Name: "workflow_alpha"},
		},
		Variables: []*export.ExportedVariable{
			{ID: 4001, BizType: 1, BizID: "1001"},
		},
		SpaceModels: []*export.ExportedSpaceModel{
			{ID: 5001, ModelEntityID: 1, ModelEntityName: "gpt-4"},
		},
		KnowledgeBases:    []*export.ExportedKnowledge{},
		Folders:           []*export.ExportedFolder{},
		FolderMappings:    []*export.ExportedFolderMapping{},
		ExternalKnowledge: []*export.ExportedExternalKnowledge{},
	}

	serializer := export.NewSerializer()
	manifest := serializer.BuildManifest(100, "test-space", 999, resources)

	zipBytes, size, err := serializer.SerializeToZip(ctx, manifest, resources)
	require.NoError(t, err)
	require.NotEmpty(t, zipBytes)
	assert.Greater(t, size, int64(0))

	validator := spaceimport.NewValidator()
	res, err := validator.ValidateAndParse(ctx, zipBytes)
	require.NoError(t, err)
	require.NotNil(t, res.Manifest)

	assert.Equal(t, "2.0.0", res.Manifest.Version)
	assert.Equal(t, int64(100), res.Manifest.Source.SpaceID)
	require.Len(t, res.Resources.Agents, 2)
	require.Len(t, res.Resources.Plugins, 1)
	require.Len(t, res.Resources.Workflows, 1)
	require.Len(t, res.Resources.Variables, 1)
	require.Len(t, res.Resources.SpaceModels, 1)

	// Names survive round-trip.
	assert.Equal(t, "agent_alpha", res.Resources.Agents[0].Name)
	assert.Equal(t, "plugin_alpha", res.Resources.Plugins[0].Name)
	assert.Equal(t, "workflow_alpha", res.Resources.Workflows[0].Name)
}

func TestE2E_ExportImportRoundTrip_LargeIDsSurvive(t *testing.T) {
	ctx := context.Background()
	bigID := int64(7383829382949000000) // > 2^53, would lose precision in float64

	resources := &export.SpaceResources{
		Agents: []*export.ExportedAgent{
			{ID: bigID, Name: "big-id-agent"},
		},
	}
	serializer := export.NewSerializer()
	manifest := serializer.BuildManifest(100, "big", 1, resources)
	zipBytes, _, err := serializer.SerializeToZip(ctx, manifest, resources)
	require.NoError(t, err)

	res, err := spaceimport.NewValidator().ValidateAndParse(ctx, zipBytes)
	require.NoError(t, err)
	require.Len(t, res.Resources.Agents, 1)
	assert.Equal(t, bigID, res.Resources.Agents[0].ID, "large int64 must round-trip without precision loss")
}

func TestE2E_ExportImportRoundTrip_SyncZipWithDeletedResources(t *testing.T) {
	ctx := context.Background()

	resources := &export.SpaceResources{
		Agents: []*export.ExportedAgent{{ID: 1, Name: "a1"}},
	}
	serializer := export.NewSerializer()
	manifest := serializer.BuildSyncManifest(100, "src", "incremental", 9999, resources)
	deleted := &export.DeletedResources{
		Agents:    []int64{500, 501},
		Workflows: []int64{700},
	}
	syncState := &export.SyncState{ExportTime: 1234567890, SourceSpaceID: 100}

	zipBytes, err := serializer.SerializeToSyncZip(ctx, manifest, resources, nil, syncState, deleted)
	require.NoError(t, err)

	res, err := spaceimport.NewValidator().ValidateAndParse(ctx, zipBytes)
	require.NoError(t, err)
	require.NotNil(t, res.Deleted)
	assert.Equal(t, []int64{500, 501}, res.Deleted.Agents)
	assert.Equal(t, []int64{700}, res.Deleted.Workflows)
}
