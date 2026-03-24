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
	"fmt"
	"time"

	"github.com/bytedance/sonic"

	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// Serializer handles serialization of space resources to ZIP format
type Serializer struct{}

// NewSerializer creates a new Serializer
func NewSerializer() *Serializer {
	return &Serializer{}
}

// SerializeToZip serializes space resources to a ZIP file
func (s *Serializer) SerializeToZip(ctx context.Context, manifest *Manifest, resources *SpaceResources) ([]byte, int64, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Write manifest.json
	if err := s.writeJSONFile(zipWriter, "manifest.json", manifest); err != nil {
		return nil, 0, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "manifest.json"))
	}
	logs.CtxInfof(ctx, "Written manifest.json")

	// Write agents
	if err := s.writeAgents(ctx, zipWriter, resources.Agents); err != nil {
		return nil, 0, err
	}

	// Write plugins
	if err := s.writePlugins(ctx, zipWriter, resources.Plugins); err != nil {
		return nil, 0, err
	}

	// Write workflows
	if err := s.writeWorkflows(ctx, zipWriter, resources.Workflows); err != nil {
		return nil, 0, err
	}

	// Write variables
	if err := s.writeVariables(ctx, zipWriter, resources.Variables); err != nil {
		return nil, 0, err
	}

	// Write space models
	if err := s.writeSpaceModels(ctx, zipWriter, resources.SpaceModels); err != nil {
		return nil, 0, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, 0, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("msg", "failed to close zip writer"))
	}

	return buf.Bytes(), int64(buf.Len()), nil
}

// writeJSONFile writes a JSON file to the ZIP archive
func (s *Serializer) writeJSONFile(zipWriter *zip.Writer, filename string, data interface{}) error {
	jsonData, err := sonic.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %w", filename, err)
	}

	writer, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:     filename,
		Method:   zip.Deflate,
		Modified: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to create zip entry for %s: %w", filename, err)
	}

	if _, err := writer.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write data for %s: %w", filename, err)
	}

	return nil
}

// writeAgents writes all agents to the ZIP archive
func (s *Serializer) writeAgents(ctx context.Context, zipWriter *zip.Writer, agents []*ExportedAgent) error {
	// Create index
	index := &ResourceIndex{
		Count: len(agents),
		Items: make([]ResourceIndexItem, 0, len(agents)),
	}

	for _, agent := range agents {
		filename := fmt.Sprintf("agents/%d.json", agent.ID)
		if err := s.writeJSONFile(zipWriter, filename, agent); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filename))
		}

		index.Items = append(index.Items, ResourceIndexItem{
			ID:   agent.ID,
			Name: agent.Name,
			File: filename,
		})
	}

	// Write index
	if err := s.writeJSONFile(zipWriter, "agents/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "agents/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d agents to ZIP", len(agents))
	return nil
}

// writePlugins writes all plugins to the ZIP archive
func (s *Serializer) writePlugins(ctx context.Context, zipWriter *zip.Writer, plugins []*ExportedPlugin) error {
	// Create index
	index := &ResourceIndex{
		Count: len(plugins),
		Items: make([]ResourceIndexItem, 0, len(plugins)),
	}

	for _, plugin := range plugins {
		filename := fmt.Sprintf("plugins/%d.json", plugin.ID)
		if err := s.writeJSONFile(zipWriter, filename, plugin); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filename))
		}

		index.Items = append(index.Items, ResourceIndexItem{
			ID:   plugin.ID,
			Name: plugin.Name,
			File: filename,
		})
	}

	// Write index
	if err := s.writeJSONFile(zipWriter, "plugins/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "plugins/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d plugins to ZIP", len(plugins))
	return nil
}

// writeWorkflows writes all workflows to the ZIP archive
func (s *Serializer) writeWorkflows(ctx context.Context, zipWriter *zip.Writer, workflows []*ExportedWorkflow) error {
	// Create index
	index := &ResourceIndex{
		Count: len(workflows),
		Items: make([]ResourceIndexItem, 0, len(workflows)),
	}

	for _, workflow := range workflows {
		filename := fmt.Sprintf("workflows/%d.json", workflow.ID)
		if err := s.writeJSONFile(zipWriter, filename, workflow); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filename))
		}

		index.Items = append(index.Items, ResourceIndexItem{
			ID:   workflow.ID,
			Name: workflow.Name,
			File: filename,
		})
	}

	// Write index
	if err := s.writeJSONFile(zipWriter, "workflows/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "workflows/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d workflows to ZIP", len(workflows))
	return nil
}

// writeVariables writes all variables to the ZIP archive
func (s *Serializer) writeVariables(ctx context.Context, zipWriter *zip.Writer, variables []*ExportedVariable) error {
	// Create index
	index := &ResourceIndex{
		Count: len(variables),
		Items: make([]ResourceIndexItem, 0, len(variables)),
	}

	for _, variable := range variables {
		filename := fmt.Sprintf("variables/%d.json", variable.ID)
		if err := s.writeJSONFile(zipWriter, filename, variable); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filename))
		}

		index.Items = append(index.Items, ResourceIndexItem{
			ID:   variable.ID,
			Name: fmt.Sprintf("variable_%d", variable.ID),
			File: filename,
		})
	}

	// Write index
	if err := s.writeJSONFile(zipWriter, "variables/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "variables/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d variables to ZIP", len(variables))
	return nil
}

// writeSpaceModels writes all space models to the ZIP archive
func (s *Serializer) writeSpaceModels(ctx context.Context, zipWriter *zip.Writer, spaceModels []*ExportedSpaceModel) error {
	// Create index
	index := &ResourceIndex{
		Count: len(spaceModels),
		Items: make([]ResourceIndexItem, 0, len(spaceModels)),
	}

	for _, sm := range spaceModels {
		filename := fmt.Sprintf("space_models/%d.json", sm.ID)
		if err := s.writeJSONFile(zipWriter, filename, sm); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filename))
		}

		index.Items = append(index.Items, ResourceIndexItem{
			ID:   sm.ID,
			Name: fmt.Sprintf("model_%d", sm.ID),
			File: filename,
		})
	}

	// Write index
	if err := s.writeJSONFile(zipWriter, "space_models/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "space_models/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d space models to ZIP", len(spaceModels))
	return nil
}

// addJSONToZip marshals v as JSON and writes it to the ZIP at the given path
func (s *Serializer) addJSONToZip(w *zip.Writer, path string, v interface{}) error {
	data, err := sonic.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %w", path, err)
	}
	writer, err := w.CreateHeader(&zip.FileHeader{
		Name:     path,
		Method:   zip.Deflate,
		Modified: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to create zip entry for %s: %w", path, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write data for %s: %w", path, err)
	}
	return nil
}

// SerializeToSyncZip serializes space resources to a ZIP file for incremental sync
func (s *Serializer) SerializeToSyncZip(ctx context.Context, manifest *Manifest, resources *SpaceResources,
	fileContents map[int64][]byte, syncState *SyncState, deleted *DeletedResources) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Write manifest.json
	if err := s.addJSONToZip(zipWriter, "manifest.json", manifest); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "manifest.json"))
	}

	// Write existing resource types
	if err := s.writeAgents(ctx, zipWriter, resources.Agents); err != nil {
		return nil, err
	}
	if err := s.writePlugins(ctx, zipWriter, resources.Plugins); err != nil {
		return nil, err
	}
	if err := s.writeWorkflows(ctx, zipWriter, resources.Workflows); err != nil {
		return nil, err
	}
	if err := s.writeVariables(ctx, zipWriter, resources.Variables); err != nil {
		return nil, err
	}
	if err := s.writeSpaceModels(ctx, zipWriter, resources.SpaceModels); err != nil {
		return nil, err
	}

	// Write knowledge bases with nested structure
	if err := s.writeKnowledgeBases(ctx, zipWriter, resources.KnowledgeBases, fileContents); err != nil {
		return nil, err
	}

	// Write folders
	if err := s.writeFolders(ctx, zipWriter, resources.Folders, resources.FolderMappings); err != nil {
		return nil, err
	}

	// Write external knowledge
	if err := s.writeExternalKnowledge(ctx, zipWriter, resources.ExternalKnowledge); err != nil {
		return nil, err
	}

	// Write sync_state.json
	if syncState != nil {
		if err := s.addJSONToZip(zipWriter, "sync_state.json", syncState); err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "sync_state.json"))
		}
	}

	// Write deleted_resources.json
	if deleted != nil {
		if err := s.addJSONToZip(zipWriter, "deleted_resources.json", deleted); err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "deleted_resources.json"))
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("msg", "failed to close zip writer"))
	}

	logs.CtxInfof(ctx, "SerializeToSyncZip: written ZIP with %d bytes", buf.Len())
	return buf.Bytes(), nil
}

// writeKnowledgeBases writes knowledge bases with nested directory structure
func (s *Serializer) writeKnowledgeBases(ctx context.Context, zipWriter *zip.Writer, knowledgeBases []*ExportedKnowledge, fileContents map[int64][]byte) error {
	index := &ResourceIndex{
		Count: len(knowledgeBases),
		Items: make([]ResourceIndexItem, 0, len(knowledgeBases)),
	}

	for _, kb := range knowledgeBases {
		// Write meta.json (knowledge metadata without documents)
		meta := &ExportedKnowledge{
			ID: kb.ID, Name: kb.Name, Description: kb.Description,
			IconURI: kb.IconURI, FormatType: kb.FormatType,
			Status: kb.Status, CreatedAt: kb.CreatedAt, UpdatedAt: kb.UpdatedAt,
		}
		metaPath := fmt.Sprintf("knowledge_bases/%d/meta.json", kb.ID)
		if err := s.addJSONToZip(zipWriter, metaPath, meta); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", metaPath))
		}

		// Write documents index
		docIndex := &ResourceIndex{
			Count: len(kb.Documents),
			Items: make([]ResourceIndexItem, 0, len(kb.Documents)),
		}
		for _, doc := range kb.Documents {
			docPath := fmt.Sprintf("knowledge_bases/%d/documents/%d.json", kb.ID, doc.ID)
			if err := s.addJSONToZip(zipWriter, docPath, doc); err != nil {
				return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", docPath))
			}
			docIndex.Items = append(docIndex.Items, ResourceIndexItem{
				ID: doc.ID, Name: doc.Name, File: docPath,
			})

			// Write raw file bytes if available
			if fileContents != nil {
				if content, ok := fileContents[doc.ID]; ok && len(content) > 0 {
					filePath := fmt.Sprintf("knowledge_bases/%d/files/%s", kb.ID, doc.ExportFileName)
					writer, err := zipWriter.CreateHeader(&zip.FileHeader{
						Name: filePath, Method: zip.Deflate, Modified: time.Now(),
					})
					if err != nil {
						return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filePath))
					}
					if _, err := writer.Write(content); err != nil {
						return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filePath))
					}
				}
			}
		}
		docIndexPath := fmt.Sprintf("knowledge_bases/%d/documents/index.json", kb.ID)
		if err := s.addJSONToZip(zipWriter, docIndexPath, docIndex); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", docIndexPath))
		}

		index.Items = append(index.Items, ResourceIndexItem{
			ID: kb.ID, Name: kb.Name, File: metaPath,
		})
	}

	if err := s.addJSONToZip(zipWriter, "knowledge_bases/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "knowledge_bases/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d knowledge bases to ZIP", len(knowledgeBases))
	return nil
}

// writeFolders writes folders and resource mappings
func (s *Serializer) writeFolders(ctx context.Context, zipWriter *zip.Writer, folders []*ExportedFolder, mappings []*ExportedFolderMapping) error {
	if err := s.addJSONToZip(zipWriter, "folders/index.json", folders); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "folders/index.json"))
	}
	if err := s.addJSONToZip(zipWriter, "folders/resource_mappings.json", mappings); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "folders/resource_mappings.json"))
	}
	logs.CtxInfof(ctx, "Written %d folders and %d mappings to ZIP", len(folders), len(mappings))
	return nil
}

// writeExternalKnowledge writes external knowledge bindings
func (s *Serializer) writeExternalKnowledge(ctx context.Context, zipWriter *zip.Writer, bindings []*ExportedExternalKnowledge) error {
	index := &ResourceIndex{
		Count: len(bindings),
		Items: make([]ResourceIndexItem, 0, len(bindings)),
	}

	for _, b := range bindings {
		filename := fmt.Sprintf("external_knowledge/%d.json", b.ID)
		if err := s.addJSONToZip(zipWriter, filename, b); err != nil {
			return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", filename))
		}
		index.Items = append(index.Items, ResourceIndexItem{
			ID: b.ID, Name: b.BindingName, File: filename,
		})
	}

	if err := s.addJSONToZip(zipWriter, "external_knowledge/index.json", index); err != nil {
		return errorx.WrapByCode(err, errno.ErrSpaceExportFailedCode, errorx.KV("file", "external_knowledge/index.json"))
	}

	logs.CtxInfof(ctx, "Written %d external knowledge bindings to ZIP", len(bindings))
	return nil
}

// BuildSyncManifest builds a manifest with sync metadata
func (s *Serializer) BuildSyncManifest(spaceID int64, spaceName string, mode string, sinceTime int64, resources *SpaceResources) *Manifest {
	registry := IDRegistry{
		Agents:            make([]int64, 0, len(resources.Agents)),
		Plugins:           make([]int64, 0, len(resources.Plugins)),
		Workflows:         make([]int64, 0, len(resources.Workflows)),
		Variables:         make([]int64, 0, len(resources.Variables)),
		SpaceModels:       make([]int64, 0, len(resources.SpaceModels)),
		KnowledgeBases:    make([]int64, 0, len(resources.KnowledgeBases)),
		Documents:         make([]int64, 0),
		ExternalKnowledge: make([]int64, 0, len(resources.ExternalKnowledge)),
		Folders:           make([]int64, 0, len(resources.Folders)),
	}

	totalDocs := 0
	var filesTotalSize int64
	for _, agent := range resources.Agents {
		registry.Agents = append(registry.Agents, agent.ID)
	}
	for _, plugin := range resources.Plugins {
		registry.Plugins = append(registry.Plugins, plugin.ID)
	}
	for _, workflow := range resources.Workflows {
		registry.Workflows = append(registry.Workflows, workflow.ID)
	}
	for _, variable := range resources.Variables {
		registry.Variables = append(registry.Variables, variable.ID)
	}
	for _, sm := range resources.SpaceModels {
		registry.SpaceModels = append(registry.SpaceModels, sm.ID)
	}
	for _, kb := range resources.KnowledgeBases {
		registry.KnowledgeBases = append(registry.KnowledgeBases, kb.ID)
		for _, doc := range kb.Documents {
			registry.Documents = append(registry.Documents, doc.ID)
			totalDocs++
			filesTotalSize += doc.Size
		}
	}
	for _, ek := range resources.ExternalKnowledge {
		registry.ExternalKnowledge = append(registry.ExternalKnowledge, ek.ID)
	}
	for _, f := range resources.Folders {
		registry.Folders = append(registry.Folders, f.ID)
	}

	return &Manifest{
		Version:    ManifestVersion,
		SyncType:   mode,
		SinceTime:  sinceTime,
		ExportTime: time.Now().UTC().Format(time.RFC3339),
		Source: SourceInfo{
			SpaceID:   spaceID,
			SpaceName: spaceName,
		},
		Statistics: Statistics{
			Agents:            len(resources.Agents),
			Plugins:           len(resources.Plugins),
			Workflows:         len(resources.Workflows),
			Variables:         len(resources.Variables),
			SpaceModels:       len(resources.SpaceModels),
			KnowledgeBases:    len(resources.KnowledgeBases),
			Documents:         totalDocs,
			FilesTotalSize:    filesTotalSize,
			ExternalKnowledge: len(resources.ExternalKnowledge),
			Folders:           len(resources.Folders),
		},
		IDRegistry: registry,
	}
}

// BuildManifest builds the manifest for export
func (s *Serializer) BuildManifest(spaceID int64, spaceName string, exporterID int64, resources *SpaceResources) *Manifest {
	// Build ID registry
	registry := IDRegistry{
		Agents:      make([]int64, 0, len(resources.Agents)),
		Plugins:     make([]int64, 0, len(resources.Plugins)),
		Workflows:   make([]int64, 0, len(resources.Workflows)),
		Variables:   make([]int64, 0, len(resources.Variables)),
		SpaceModels: make([]int64, 0, len(resources.SpaceModels)),
	}

	for _, agent := range resources.Agents {
		registry.Agents = append(registry.Agents, agent.ID)
	}
	for _, plugin := range resources.Plugins {
		registry.Plugins = append(registry.Plugins, plugin.ID)
	}
	for _, workflow := range resources.Workflows {
		registry.Workflows = append(registry.Workflows, workflow.ID)
	}
	for _, variable := range resources.Variables {
		registry.Variables = append(registry.Variables, variable.ID)
	}
	for _, spaceModel := range resources.SpaceModels {
		registry.SpaceModels = append(registry.SpaceModels, spaceModel.ID)
	}

	return &Manifest{
		Version:    ManifestVersion,
		ExportTime: time.Now().UTC().Format(time.RFC3339),
		Source: SourceInfo{
			SpaceID:    spaceID,
			SpaceName:  spaceName,
			ExporterID: exporterID,
		},
		Statistics: Statistics{
			Agents:      len(resources.Agents),
			Plugins:     len(resources.Plugins),
			Workflows:   len(resources.Workflows),
			Variables:   len(resources.Variables),
			SpaceModels: len(resources.SpaceModels),
		},
		IDRegistry: registry,
	}
}
