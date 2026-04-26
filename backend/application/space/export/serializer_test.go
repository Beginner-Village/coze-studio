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
	"archive/zip"
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleManifest() *Manifest {
	return &Manifest{
		Version:    ManifestVersion,
		ExportTime: "2026-04-27T00:00:00Z",
		Source: SourceInfo{
			SpaceID:    100,
			SpaceName:  "test-space",
			ExporterID: 999,
		},
		Statistics: Statistics{Agents: 1},
		IDRegistry: IDRegistry{Agents: []int64{1}},
	}
}

func sampleResourcesForSerialize() *SpaceResources {
	return &SpaceResources{
		Agents: []*ExportedAgent{
			{ID: 1, Name: "agent1", Desc: "first agent"},
			{ID: 2, Name: "agent2", Desc: "second agent"},
		},
		Plugins: []*ExportedPlugin{
			{ID: 100, Name: "plugin1"},
		},
		Workflows: []*ExportedWorkflow{
			{ID: 200, Name: "wf1"},
		},
		Variables: []*ExportedVariable{
			{ID: 300, BizType: 1, BizID: "1"},
		},
		SpaceModels: []*ExportedSpaceModel{
			{ID: 400, ModelEntityID: 1, ModelEntityName: "gpt-4"},
		},
		KnowledgeBases:    []*ExportedKnowledge{},
		Folders:           []*ExportedFolder{},
		FolderMappings:    []*ExportedFolderMapping{},
		ExternalKnowledge: []*ExportedExternalKnowledge{},
	}
}

// readZipFile is a tiny helper that reads a single file from ZIP bytes.
func readZipFile(t *testing.T, zipBytes []byte, name string) ([]byte, bool) {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	require.NoError(t, err)
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			require.NoError(t, err)
			defer rc.Close()
			data, err := io.ReadAll(rc)
			require.NoError(t, err)
			return data, true
		}
	}
	return nil, false
}

func zipFileNames(t *testing.T, zipBytes []byte) []string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	require.NoError(t, err)
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

func TestSerializer_SerializeToZip_RoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	resources := sampleResourcesForSerialize()

	zipBytes, size, err := s.SerializeToZip(ctx, manifest, resources)
	require.NoError(t, err)
	require.NotEmpty(t, zipBytes)
	assert.Equal(t, int64(len(zipBytes)), size, "size return matches buffer length")

	names := zipFileNames(t, zipBytes)
	// Spot check that expected entries are there.
	for _, expected := range []string{
		"manifest.json",
		"agents/index.json",
		"agents/1.json",
		"agents/2.json",
		"plugins/index.json",
		"plugins/100.json",
		"workflows/200.json",
		"variables/300.json",
		"space_models/400.json",
	} {
		assert.Contains(t, names, expected)
	}
}

func TestSerializer_SerializeToZip_AgentRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	original := &ExportedAgent{
		ID:   42,
		Name: "agent42",
		Desc: "alpha & beta",
	}
	resources := &SpaceResources{Agents: []*ExportedAgent{original}}

	zipBytes, _, err := s.SerializeToZip(ctx, manifest, resources)
	require.NoError(t, err)

	data, ok := readZipFile(t, zipBytes, "agents/42.json")
	require.True(t, ok)

	var got ExportedAgent
	require.NoError(t, sonic.Unmarshal(data, &got))
	assert.Equal(t, original.ID, got.ID)
	assert.Equal(t, original.Name, got.Name)
	assert.Equal(t, original.Desc, got.Desc)
}

func TestSerializer_SerializeToZip_ManifestRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	manifest.ReleaseVersion = "1.2.3"

	zipBytes, _, err := s.SerializeToZip(ctx, manifest, sampleResourcesForSerialize())
	require.NoError(t, err)

	data, ok := readZipFile(t, zipBytes, "manifest.json")
	require.True(t, ok)
	var got Manifest
	require.NoError(t, sonic.Unmarshal(data, &got))
	assert.Equal(t, manifest.Version, got.Version)
	assert.Equal(t, manifest.Source.SpaceID, got.Source.SpaceID)
	assert.Equal(t, manifest.Source.SpaceName, got.Source.SpaceName)
	assert.Equal(t, manifest.ReleaseVersion, got.ReleaseVersion)
}

func TestSerializer_SerializeToZip_EmptyResources(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	empty := &SpaceResources{}
	zipBytes, _, err := s.SerializeToZip(ctx, sampleManifest(), empty)
	require.NoError(t, err)

	names := zipFileNames(t, zipBytes)
	// Empty index files should still be present
	for _, expected := range []string{
		"manifest.json",
		"agents/index.json",
		"plugins/index.json",
		"workflows/index.json",
		"variables/index.json",
		"space_models/index.json",
	} {
		assert.Contains(t, names, expected)
	}
}

func TestSerializer_SpecialCharactersInNames(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	resources := &SpaceResources{
		Agents: []*ExportedAgent{
			{ID: 1, Name: "测试 中文 \"quotes\" \\back\\slash 🚀", Desc: "newline\nin desc"},
		},
	}
	zipBytes, _, err := s.SerializeToZip(ctx, manifest, resources)
	require.NoError(t, err)

	data, ok := readZipFile(t, zipBytes, "agents/1.json")
	require.True(t, ok)
	var got ExportedAgent
	require.NoError(t, sonic.Unmarshal(data, &got))
	assert.Equal(t, resources.Agents[0].Name, got.Name)
	assert.Equal(t, resources.Agents[0].Desc, got.Desc)
}

func TestSerializer_LargeIDPrecision(t *testing.T) {
	// Verify int64 IDs > 2^53 survive serialization (relevant for distributed ID generators).
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	bigID := int64(7383829382949000000)
	resources := &SpaceResources{
		Agents: []*ExportedAgent{
			{ID: bigID, Name: "big"},
		},
	}
	zipBytes, _, err := s.SerializeToZip(ctx, manifest, resources)
	require.NoError(t, err)

	// Note: filename uses %d which prints exact int64.
	data, ok := readZipFile(t, zipBytes, "agents/7383829382949000000.json")
	require.True(t, ok)

	var got ExportedAgent
	require.NoError(t, sonic.Unmarshal(data, &got))
	assert.Equal(t, bigID, got.ID, "large int64 ID must round-trip without precision loss")
}

func TestSerializer_BuildManifest_PopulatesRegistry(t *testing.T) {
	s := NewSerializer()
	resources := sampleResourcesForSerialize()
	m := s.BuildManifest(100, "test", 999, resources)

	require.NotNil(t, m)
	assert.Equal(t, ManifestVersion, m.Version)
	assert.Equal(t, int64(100), m.Source.SpaceID)
	assert.Equal(t, "test", m.Source.SpaceName)
	assert.Equal(t, int64(999), m.Source.ExporterID)
	assert.Equal(t, []int64{1, 2}, m.IDRegistry.Agents)
	assert.Equal(t, []int64{100}, m.IDRegistry.Plugins)
	assert.Equal(t, []int64{200}, m.IDRegistry.Workflows)
	assert.Equal(t, len(resources.Agents), m.Statistics.Agents)
}

func TestSerializer_BuildSyncManifest_IncludesDocuments(t *testing.T) {
	s := NewSerializer()
	resources := &SpaceResources{
		Agents: []*ExportedAgent{{ID: 1}},
		KnowledgeBases: []*ExportedKnowledge{
			{
				ID: 700,
				Documents: []*ExportedDocument{
					{ID: 7001, Size: 100},
					{ID: 7002, Size: 200},
				},
			},
		},
	}
	m := s.BuildSyncManifest(100, "test", "incremental", 1234, resources)
	require.NotNil(t, m)
	assert.Equal(t, "incremental", m.SyncType)
	assert.Equal(t, int64(1234), m.SinceTime)
	assert.Equal(t, []int64{700}, m.IDRegistry.KnowledgeBases)
	assert.ElementsMatch(t, []int64{7001, 7002}, m.IDRegistry.Documents)
	assert.Equal(t, 2, m.Statistics.Documents)
	assert.Equal(t, int64(300), m.Statistics.FilesTotalSize)
}

func TestSerializer_SerializeToSyncZip_IncludesDeletedAndSyncState(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	resources := sampleResourcesForSerialize()
	syncState := &SyncState{ExportTime: 1234567890, SourceSpaceID: 100}
	deleted := &DeletedResources{Agents: []int64{99, 100}}

	zipBytes, err := s.SerializeToSyncZip(ctx, manifest, resources, nil, syncState, deleted)
	require.NoError(t, err)

	names := zipFileNames(t, zipBytes)
	assert.Contains(t, names, "sync_state.json")
	assert.Contains(t, names, "deleted_resources.json")

	data, ok := readZipFile(t, zipBytes, "deleted_resources.json")
	require.True(t, ok)
	var gotDeleted DeletedResources
	require.NoError(t, sonic.Unmarshal(data, &gotDeleted))
	assert.Equal(t, []int64{99, 100}, gotDeleted.Agents)
}

func TestSerializer_SerializeToSyncZip_KnowledgeStructure(t *testing.T) {
	ctx := context.Background()
	s := NewSerializer()
	manifest := sampleManifest()
	resources := &SpaceResources{
		KnowledgeBases: []*ExportedKnowledge{
			{
				ID:   700,
				Name: "kb1",
				Documents: []*ExportedDocument{
					{ID: 7001, KnowledgeID: 700, Name: "doc1", ExportFileName: "doc1.txt"},
				},
			},
		},
	}
	fileContents := map[int64][]byte{7001: []byte("hello world")}

	zipBytes, err := s.SerializeToSyncZip(ctx, manifest, resources, fileContents, nil, nil)
	require.NoError(t, err)
	names := zipFileNames(t, zipBytes)

	assert.Contains(t, names, "knowledge_bases/700/meta.json")
	assert.Contains(t, names, "knowledge_bases/700/documents/7001.json")
	assert.Contains(t, names, "knowledge_bases/700/documents/index.json")
	assert.Contains(t, names, "knowledge_bases/700/files/doc1.txt")

	// Verify file content
	body, ok := readZipFile(t, zipBytes, "knowledge_bases/700/files/doc1.txt")
	require.True(t, ok)
	assert.Equal(t, "hello world", string(body))
}
