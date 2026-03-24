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
	"fmt"
	"io"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/ynet-dev/ynet-studio/backend/application/space/export"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	// MaxImportFileSize is the maximum size of an import file (100MB)
	MaxImportFileSize = 100 * 1024 * 1024

	// MaxResourceCount is the maximum number of resources per type
	MaxResourceCount = 500
)

// Validator validates import packages
type Validator struct{}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidationResult contains the validation result
type ValidationResult struct {
	Manifest     *export.Manifest
	Resources    *export.SpaceResources
	Warnings     []string
	FileContents map[int64][]byte // document ID -> file bytes from knowledge_bases/{id}/files/
}

// ValidateAndParse validates and parses an import package
func (v *Validator) ValidateAndParse(ctx context.Context, fileContent []byte) (*ValidationResult, error) {
	// Check file size
	if len(fileContent) > MaxImportFileSize {
		return nil, errorx.New(errno.ErrSpaceImportFailedCode,
			errorx.KV("msg", fmt.Sprintf("file size exceeds maximum limit of %d bytes", MaxImportFileSize)))
	}

	// Open ZIP file
	zipReader, err := zip.NewReader(bytes.NewReader(fileContent), int64(len(fileContent)))
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode,
			errorx.KV("msg", "invalid ZIP file format"))
	}

	// Parse manifest
	manifest, err := v.parseManifest(ctx, zipReader)
	if err != nil {
		return nil, err
	}

	// Validate manifest version
	if err := v.validateManifestVersion(manifest); err != nil {
		return nil, err
	}

	// Validate resource counts
	if err := v.validateResourceCounts(manifest); err != nil {
		return nil, err
	}

	// Parse resources
	resources, warnings, fileContents, err := v.parseResources(ctx, zipReader, manifest)
	if err != nil {
		return nil, err
	}

	logs.CtxInfof(ctx, "Validated import package: agents=%d, plugins=%d, workflows=%d, variables=%d, space_models=%d, knowledge=%d, folders=%d, external_knowledge=%d, warnings=%d",
		len(resources.Agents), len(resources.Plugins), len(resources.Workflows),
		len(resources.Variables), len(resources.SpaceModels),
		len(resources.KnowledgeBases), len(resources.Folders), len(resources.ExternalKnowledge),
		len(warnings))

	// Log IDRegistry for debugging
	logs.CtxDebugf(ctx, "Parsed IDRegistry - Agents: %v", manifest.IDRegistry.Agents)
	logs.CtxDebugf(ctx, "Parsed IDRegistry - Plugins: %v", manifest.IDRegistry.Plugins)
	logs.CtxDebugf(ctx, "Parsed IDRegistry - Workflows: %v", manifest.IDRegistry.Workflows)
	logs.CtxDebugf(ctx, "Parsed IDRegistry - Variables: %v", manifest.IDRegistry.Variables)
	logs.CtxDebugf(ctx, "Parsed IDRegistry - SpaceModels: %v", manifest.IDRegistry.SpaceModels)
	logs.CtxDebugf(ctx, "Parsed IDRegistry - KnowledgeBases: %v", manifest.IDRegistry.KnowledgeBases)
	logs.CtxDebugf(ctx, "Parsed IDRegistry - Folders: %v", manifest.IDRegistry.Folders)

	// Log workflow refs in agents for debugging
	for _, agent := range resources.Agents {
		if len(agent.WorkflowRefs) > 0 {
			for i, wf := range agent.WorkflowRefs {
				logs.CtxDebugf(ctx, "Parsed Agent %s workflow ref[%d]: WorkflowId=%d",
					agent.Name, i, wf.GetWorkflowId())
			}
		}
	}

	return &ValidationResult{
		Manifest:     manifest,
		Resources:    resources,
		Warnings:     warnings,
		FileContents: fileContents,
	}, nil
}

// parseManifest parses the manifest.json from the ZIP
func (v *Validator) parseManifest(ctx context.Context, zipReader *zip.Reader) (*export.Manifest, error) {
	for _, file := range zipReader.File {
		if file.Name == "manifest.json" {
			rc, err := file.Open()
			if err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode,
					errorx.KV("msg", "failed to open manifest.json"))
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode,
					errorx.KV("msg", "failed to read manifest.json"))
			}

			var manifest export.Manifest
			if err := sonic.Unmarshal(data, &manifest); err != nil {
				return nil, errorx.WrapByCode(err, errno.ErrSpaceImportFailedCode,
					errorx.KV("msg", "failed to parse manifest.json"))
			}

			return &manifest, nil
		}
	}

	return nil, errorx.New(errno.ErrSpaceImportFailedCode,
		errorx.KV("msg", "manifest.json not found in package"))
}

// supportedManifestVersions lists all supported manifest versions
var supportedManifestVersions = map[string]bool{
	"1.0.0": true,
	"2.0.0": true,
}

// validateManifestVersion validates the manifest version
func (v *Validator) validateManifestVersion(manifest *export.Manifest) error {
	if manifest.Version == "" {
		return errorx.New(errno.ErrSpaceImportFailedCode,
			errorx.KV("msg", "manifest version is empty"))
	}

	if !supportedManifestVersions[manifest.Version] {
		return errorx.New(errno.ErrSpaceImportFailedCode,
			errorx.KV("msg", fmt.Sprintf("unsupported manifest version: %s, supported: 1.0.0, 2.0.0",
				manifest.Version)))
	}

	return nil
}

// validateResourceCounts validates the resource counts
func (v *Validator) validateResourceCounts(manifest *export.Manifest) error {
	total := manifest.Statistics.Agents + manifest.Statistics.Plugins +
		manifest.Statistics.Workflows + manifest.Statistics.Variables

	if total > MaxResourceCount {
		return errorx.New(errno.ErrSpaceImportFailedCode,
			errorx.KV("msg", fmt.Sprintf("total resource count %d exceeds maximum limit of %d", total, MaxResourceCount)))
	}

	return nil
}

// parseResources parses all resources from the ZIP
func (v *Validator) parseResources(ctx context.Context, zipReader *zip.Reader, manifest *export.Manifest) (*export.SpaceResources, []string, map[int64][]byte, error) {
	resources := &export.SpaceResources{
		Agents:            make([]*export.ExportedAgent, 0),
		Plugins:           make([]*export.ExportedPlugin, 0),
		Workflows:         make([]*export.ExportedWorkflow, 0),
		Variables:         make([]*export.ExportedVariable, 0),
		SpaceModels:       make([]*export.ExportedSpaceModel, 0),
		KnowledgeBases:    make([]*export.ExportedKnowledge, 0),
		Folders:           make([]*export.ExportedFolder, 0),
		FolderMappings:    make([]*export.ExportedFolderMapping, 0),
		ExternalKnowledge: make([]*export.ExportedExternalKnowledge, 0),
	}
	var warnings []string
	fileContents := make(map[int64][]byte) // document ID -> file bytes

	// Create a map of file contents for easy access
	fileMap := make(map[string][]byte)
	for _, file := range zipReader.File {
		rc, err := file.Open()
		if err != nil {
			logs.CtxWarnf(ctx, "Failed to open file %s: %v", file.Name, err)
			continue
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			logs.CtxWarnf(ctx, "Failed to read file %s: %v", file.Name, err)
			continue
		}

		fileMap[file.Name] = data
	}

	// Parse agents
	for _, agentID := range manifest.IDRegistry.Agents {
		filename := fmt.Sprintf("agents/%d.json", agentID)
		data, ok := fileMap[filename]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Agent file not found: %s", filename))
			continue
		}

		var agent export.ExportedAgent
		if err := sonic.Unmarshal(data, &agent); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse agent %d: %v", agentID, err))
			continue
		}

		resources.Agents = append(resources.Agents, &agent)
	}

	// Parse plugins
	for _, pluginID := range manifest.IDRegistry.Plugins {
		filename := fmt.Sprintf("plugins/%d.json", pluginID)
		data, ok := fileMap[filename]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Plugin file not found: %s", filename))
			continue
		}

		var plugin export.ExportedPlugin
		if err := sonic.Unmarshal(data, &plugin); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse plugin %d: %v", pluginID, err))
			continue
		}

		resources.Plugins = append(resources.Plugins, &plugin)
	}

	// Parse workflows
	for _, workflowID := range manifest.IDRegistry.Workflows {
		filename := fmt.Sprintf("workflows/%d.json", workflowID)
		data, ok := fileMap[filename]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Workflow file not found: %s", filename))
			continue
		}

		var workflow export.ExportedWorkflow
		if err := sonic.Unmarshal(data, &workflow); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse workflow %d: %v", workflowID, err))
			continue
		}

		resources.Workflows = append(resources.Workflows, &workflow)
	}

	// Parse variables
	for _, variableID := range manifest.IDRegistry.Variables {
		filename := fmt.Sprintf("variables/%d.json", variableID)
		data, ok := fileMap[filename]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("Variable file not found: %s", filename))
			continue
		}

		var variable export.ExportedVariable
		if err := sonic.Unmarshal(data, &variable); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse variable %d: %v", variableID, err))
			continue
		}

		resources.Variables = append(resources.Variables, &variable)
	}

	// Parse space models
	for _, spaceModelID := range manifest.IDRegistry.SpaceModels {
		filename := fmt.Sprintf("space_models/%d.json", spaceModelID)
		data, ok := fileMap[filename]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("SpaceModel file not found: %s", filename))
			continue
		}

		var spaceModel export.ExportedSpaceModel
		if err := sonic.Unmarshal(data, &spaceModel); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse space model %d: %v", spaceModelID, err))
			continue
		}

		resources.SpaceModels = append(resources.SpaceModels, &spaceModel)
	}

	// Parse knowledge bases with documents (inline slices)
	for _, kbID := range manifest.IDRegistry.KnowledgeBases {
		// Each knowledge base has documents at knowledge_bases/{kbID}/documents/{docID}.json
		// First, try to find the knowledge base index or individual document files
		var kb export.ExportedKnowledge
		kb.ID = kbID
		kb.Documents = make([]*export.ExportedDocument, 0)

		// Try knowledge base metadata file
		kbMetaFile := fmt.Sprintf("knowledge_bases/%d/metadata.json", kbID)
		if data, ok := fileMap[kbMetaFile]; ok {
			if err := sonic.Unmarshal(data, &kb); err != nil {
				warnings = append(warnings, fmt.Sprintf("Failed to parse knowledge base %d metadata: %v", kbID, err))
				continue
			}
		}

		// Parse documents for this knowledge base
		for _, docID := range manifest.IDRegistry.Documents {
			docFile := fmt.Sprintf("knowledge_bases/%d/documents/%d.json", kbID, docID)
			data, ok := fileMap[docFile]
			if !ok {
				continue
			}

			var doc export.ExportedDocument
			if err := sonic.Unmarshal(data, &doc); err != nil {
				warnings = append(warnings, fmt.Sprintf("Failed to parse document %d: %v", docID, err))
				continue
			}

			// Only include documents belonging to this knowledge base
			if doc.KnowledgeID == kbID {
				kb.Documents = append(kb.Documents, &doc)

				// Collect file contents for this document
				fileKey := fmt.Sprintf("knowledge_bases/%d/files/%d", kbID, docID)
				// Try with known extensions
				for fname, fdata := range fileMap {
					if strings.HasPrefix(fname, fileKey) {
						fileContents[docID] = fdata
						break
					}
				}
			}
		}

		resources.KnowledgeBases = append(resources.KnowledgeBases, &kb)
	}

	// Parse folders
	if data, ok := fileMap["folders/index.json"]; ok {
		var folders []*export.ExportedFolder
		if err := sonic.Unmarshal(data, &folders); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse folders/index.json: %v", err))
		} else {
			resources.Folders = folders
		}
	}

	// Parse folder mappings
	if data, ok := fileMap["folders/resource_mappings.json"]; ok {
		var mappings []*export.ExportedFolderMapping
		if err := sonic.Unmarshal(data, &mappings); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse folders/resource_mappings.json: %v", err))
		} else {
			resources.FolderMappings = mappings
		}
	}

	// Parse external knowledge
	for _, ekID := range manifest.IDRegistry.ExternalKnowledge {
		filename := fmt.Sprintf("external_knowledge/%d.json", ekID)
		data, ok := fileMap[filename]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("ExternalKnowledge file not found: %s", filename))
			continue
		}

		var ek export.ExportedExternalKnowledge
		if err := sonic.Unmarshal(data, &ek); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse external knowledge %d: %v", ekID, err))
			continue
		}

		resources.ExternalKnowledge = append(resources.ExternalKnowledge, &ek)
	}

	// Parse deleted_resources.json (for incremental sync)
	if data, ok := fileMap["deleted_resources.json"]; ok {
		var deleted export.DeletedResources
		if err := sonic.Unmarshal(data, &deleted); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse deleted_resources.json: %v", err))
		}
		logs.CtxDebugf(ctx, "Parsed deleted_resources.json")
	}

	// Parse sync_state.json (for incremental sync)
	if data, ok := fileMap["sync_state.json"]; ok {
		var syncState export.SyncState
		if err := sonic.Unmarshal(data, &syncState); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to parse sync_state.json: %v", err))
		}
		logs.CtxDebugf(ctx, "Parsed sync_state.json: export_time=%d, source_space_id=%d", syncState.ExportTime, syncState.SourceSpaceID)
	}

	// Generate warnings for cleared references
	warnings = append(warnings, v.generateClearingWarnings(resources)...)

	return resources, warnings, fileContents, nil
}

// generateClearingWarnings generates warnings about references that will be cleared
func (v *Validator) generateClearingWarnings(resources *export.SpaceResources) []string {
	var warnings []string

	// Check for knowledge references in agents
	for _, agent := range resources.Agents {
		if agent.KnowledgeRefs != nil {
			warnings = append(warnings,
				fmt.Sprintf("Agent '%s' has knowledge references that will be cleared", agent.Name))
			break // Only warn once
		}
	}

	// Check for knowledge/database references in workflows
	for _, workflow := range resources.Workflows {
		if workflow.Canvas != nil {
			if v.hasKnowledgeOrDatabaseNodes(workflow.Canvas) {
				warnings = append(warnings,
					fmt.Sprintf("Workflow '%s' has knowledge/database nodes that will have their references cleared", workflow.Name))
			}
		}
	}

	return warnings
}

// hasKnowledgeOrDatabaseNodes checks if the canvas has knowledge or database nodes
func (v *Validator) hasKnowledgeOrDatabaseNodes(canvas interface{}) bool {
	canvasBytes, err := sonic.Marshal(canvas)
	if err != nil {
		return false
	}

	canvasStr := string(canvasBytes)
	return strings.Contains(canvasStr, "knowledge_id") || strings.Contains(canvasStr, "database_id")
}
