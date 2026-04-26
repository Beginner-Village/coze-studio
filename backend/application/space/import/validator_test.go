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
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

// buildTestZip is a tiny helper that produces a ZIP with the given (filename, content) pairs.
func buildTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func validManifest() *export.Manifest {
	return &export.Manifest{
		Version: export.ManifestVersion,
		Source: export.SourceInfo{
			SpaceID:    1,
			SpaceName:  "test",
			ExporterID: 1,
		},
		Statistics: export.Statistics{Agents: 1},
		IDRegistry: export.IDRegistry{
			Agents: []int64{100},
		},
	}
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestValidator_validateManifestVersion_Valid(t *testing.T) {
	v := NewValidator()
	for _, ver := range []string{"1.0.0", "2.0.0"} {
		m := validManifest()
		m.Version = ver
		assert.NoError(t, v.validateManifestVersion(m), "version %s should be supported", ver)
	}
}

func TestValidator_validateManifestVersion_Empty(t *testing.T) {
	v := NewValidator()
	m := validManifest()
	m.Version = ""
	err := v.validateManifestVersion(m)
	require.Error(t, err)
}

func TestValidator_validateManifestVersion_Unsupported(t *testing.T) {
	v := NewValidator()
	m := validManifest()
	m.Version = "0.9.0"
	err := v.validateManifestVersion(m)
	require.Error(t, err)
}

func TestValidator_validateResourceCounts_WithinLimit(t *testing.T) {
	v := NewValidator()
	m := validManifest()
	m.Statistics = export.Statistics{
		Agents: 100, Plugins: 100, Workflows: 100, Variables: 100,
	} // total = 400, under 500
	assert.NoError(t, v.validateResourceCounts(m))
}

func TestValidator_validateResourceCounts_ExceedsLimit(t *testing.T) {
	v := NewValidator()
	m := validManifest()
	m.Statistics = export.Statistics{
		Agents: 200, Plugins: 200, Workflows: 200, Variables: 200,
	} // total = 800, over 500
	err := v.validateResourceCounts(m)
	require.Error(t, err)
}

func TestValidator_validateResourceCounts_AtLimitOK(t *testing.T) {
	v := NewValidator()
	m := validManifest()
	m.Statistics = export.Statistics{
		Agents: 250, Plugins: 250,
	} // total = 500, equal to limit, should pass (uses '>')
	assert.NoError(t, v.validateResourceCounts(m))
}

func TestValidator_ValidateAndParse_RoundTrip(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()

	m := &export.Manifest{
		Version:    export.ManifestVersion,
		Source:     export.SourceInfo{SpaceID: 1, SpaceName: "src", ExporterID: 99},
		Statistics: export.Statistics{Agents: 1, Plugins: 1, Workflows: 1, Variables: 1},
		IDRegistry: export.IDRegistry{
			Agents:    []int64{100},
			Plugins:   []int64{300},
			Workflows: []int64{400},
			Variables: []int64{500},
		},
	}
	files := map[string][]byte{
		"manifest.json":         mustJSON(t, m),
		"agents/100.json":       mustJSON(t, &export.ExportedAgent{ID: 100, Name: "a"}),
		"plugins/300.json":      mustJSON(t, &export.ExportedPlugin{ID: 300, Name: "p"}),
		"workflows/400.json":    mustJSON(t, &export.ExportedWorkflow{ID: 400, Name: "w"}),
		"variables/500.json":    mustJSON(t, &export.ExportedVariable{ID: 500}),
	}
	zipBytes := buildTestZip(t, files)

	res, err := v.ValidateAndParse(ctx, zipBytes)
	require.NoError(t, err)
	require.NotNil(t, res.Manifest)
	assert.Equal(t, "2.0.0", res.Manifest.Version)
	require.Len(t, res.Resources.Agents, 1)
	assert.Equal(t, "a", res.Resources.Agents[0].Name)
	require.Len(t, res.Resources.Plugins, 1)
	require.Len(t, res.Resources.Workflows, 1)
	require.Len(t, res.Resources.Variables, 1)
}

func TestValidator_ValidateAndParse_TooLarge(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	huge := make([]byte, MaxImportFileSize+1)
	_, err := v.ValidateAndParse(ctx, huge)
	require.Error(t, err)
}

func TestValidator_ValidateAndParse_InvalidZip(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	_, err := v.ValidateAndParse(ctx, []byte("not a zip file"))
	require.Error(t, err)
}

func TestValidator_ValidateAndParse_MissingManifest(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	zipBytes := buildTestZip(t, map[string][]byte{
		"agents/100.json": mustJSON(t, &export.ExportedAgent{ID: 100, Name: "a"}),
	})
	_, err := v.ValidateAndParse(ctx, zipBytes)
	require.Error(t, err)
}

func TestValidator_ValidateAndParse_BadManifestJSON(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	zipBytes := buildTestZip(t, map[string][]byte{
		"manifest.json": []byte("{not valid json"),
	})
	_, err := v.ValidateAndParse(ctx, zipBytes)
	require.Error(t, err)
}

func TestValidator_ValidateAndParse_UnsupportedVersion(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	m := validManifest()
	m.Version = "3.0.0" // unsupported
	zipBytes := buildTestZip(t, map[string][]byte{
		"manifest.json": mustJSON(t, m),
	})
	_, err := v.ValidateAndParse(ctx, zipBytes)
	require.Error(t, err)
}

func TestValidator_ValidateAndParse_MissingResourceFileEmitsWarning(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	m := validManifest()
	zipBytes := buildTestZip(t, map[string][]byte{
		"manifest.json": mustJSON(t, m),
		// agents/100.json is missing — should yield a warning, not an error
	})
	res, err := v.ValidateAndParse(ctx, zipBytes)
	require.NoError(t, err)
	require.NotEmpty(t, res.Warnings, "missing resource file should produce warnings")
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "agents/100.json") {
			found = true
			break
		}
	}
	assert.True(t, found, "warning should mention missing agents/100.json")
}

func TestValidator_ValidateAndParse_ResourceCountsOverLimit(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	m := validManifest()
	m.Statistics = export.Statistics{Agents: 600} // over 500 limit
	zipBytes := buildTestZip(t, map[string][]byte{
		"manifest.json": mustJSON(t, m),
	})
	_, err := v.ValidateAndParse(ctx, zipBytes)
	require.Error(t, err)
}

func TestValidator_HasKnowledgeOrDatabaseNodes(t *testing.T) {
	v := NewValidator()
	canvas1 := map[string]interface{}{"nodes": []interface{}{
		map[string]interface{}{"data": map[string]interface{}{"knowledge_id": 1}},
	}}
	assert.True(t, v.hasKnowledgeOrDatabaseNodes(canvas1))

	canvas2 := map[string]interface{}{"nodes": []interface{}{
		map[string]interface{}{"data": map[string]interface{}{"database_id": 1}},
	}}
	assert.True(t, v.hasKnowledgeOrDatabaseNodes(canvas2))

	canvas3 := map[string]interface{}{"nodes": []interface{}{
		map[string]interface{}{"data": map[string]interface{}{"agent_id": 1}},
	}}
	assert.False(t, v.hasKnowledgeOrDatabaseNodes(canvas3))
}

func TestValidator_GenerateClearingWarnings(t *testing.T) {
	v := NewValidator()
	resources := &export.SpaceResources{
		Agents: []*export.ExportedAgent{
			{Name: "agent1"}, // no refs → no warning
		},
	}
	got := v.generateClearingWarnings(resources)
	assert.Empty(t, got)
}

func TestValidator_ValidateAndParse_DeletedResources(t *testing.T) {
	ctx := context.Background()
	v := NewValidator()
	m := validManifest()
	m.IDRegistry = export.IDRegistry{} // no resources
	m.Statistics = export.Statistics{}

	zipBytes := buildTestZip(t, map[string][]byte{
		"manifest.json": mustJSON(t, m),
		"deleted_resources.json": mustJSON(t, &export.DeletedResources{
			Agents:  []int64{1, 2},
			Plugins: []int64{3},
		}),
	})
	res, err := v.ValidateAndParse(ctx, zipBytes)
	require.NoError(t, err)
	require.NotNil(t, res.Deleted)
	assert.Equal(t, []int64{1, 2}, res.Deleted.Agents)
	assert.Equal(t, []int64{3}, res.Deleted.Plugins)
}
