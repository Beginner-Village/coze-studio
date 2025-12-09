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
	"github.com/coze-dev/coze-studio/backend/application/space/export"
)

// ImportToken represents a pending import operation
type ImportToken struct {
	Token       string           `json:"token"`
	SpaceID     int64            `json:"space_id,string"`
	UserID      int64            `json:"user_id,string"`
	Manifest    *export.Manifest `json:"manifest"`
	CreatedAt   int64            `json:"created_at"`
	ExpiresAt   int64            `json:"expires_at"`
	TempFileKey string           `json:"temp_file_key"`
}

// PreviewResult represents the result of import preview
type PreviewResult struct {
	ImportToken string           `json:"import_token"`
	Manifest    *export.Manifest `json:"manifest"`
	Warnings    []string         `json:"warnings"`
}

// ImportResult represents the final result of an import operation
type ImportResult struct {
	AgentsCreated      int            `json:"agents_created"`
	PluginsCreated     int            `json:"plugins_created"`
	WorkflowsCreated   int            `json:"workflows_created"`
	VariablesCreated   int            `json:"variables_created"`
	SpaceModelsCreated int            `json:"space_models_created"`
	Errors             []*ImportError `json:"errors,omitempty"`
}

// ImportError represents an error that occurred during import
type ImportError struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	Error        string `json:"error"`
}

// IDMapping represents the mapping from old IDs to new IDs
type IDMapping struct {
	OldID int64 `json:"old_id,string"`
	NewID int64 `json:"new_id,string"`
}

// ImportContext holds all context needed during import
type ImportContext struct {
	TargetSpaceID int64
	UserID        int64

	// ID mappings
	AgentIDMap      map[int64]int64
	PluginIDMap     map[int64]int64
	WorkflowIDMap   map[int64]int64
	VariableIDMap   map[int64]int64
	SpaceModelIDMap map[int64]int64 // old space_model.id -> new space_model.id

	// Track which space models were actually created (not skipped)
	CreatedSpaceModelIDs map[int64]bool // new space_model.id -> true if created

	// Fallback model ID for agents when their model is not available
	FallbackModelID *int64

	// Package ID registry for checking in-package references
	PackageIDs *export.IDRegistry
}

// NewImportContext creates a new import context
func NewImportContext(targetSpaceID, userID int64, registry *export.IDRegistry) *ImportContext {
	return &ImportContext{
		TargetSpaceID:        targetSpaceID,
		UserID:               userID,
		AgentIDMap:           make(map[int64]int64),
		PluginIDMap:          make(map[int64]int64),
		WorkflowIDMap:        make(map[int64]int64),
		VariableIDMap:        make(map[int64]int64),
		SpaceModelIDMap:      make(map[int64]int64),
		CreatedSpaceModelIDs: make(map[int64]bool),
		PackageIDs:           registry,
	}
}

// IsInPackageAgent checks if an agent ID is in the package
func (c *ImportContext) IsInPackageAgent(id int64) bool {
	for _, aid := range c.PackageIDs.Agents {
		if aid == id {
			return true
		}
	}
	return false
}

// IsInPackagePlugin checks if a plugin ID is in the package
func (c *ImportContext) IsInPackagePlugin(id int64) bool {
	for _, pid := range c.PackageIDs.Plugins {
		if pid == id {
			return true
		}
	}
	return false
}

// IsInPackageWorkflow checks if a workflow ID is in the package
func (c *ImportContext) IsInPackageWorkflow(id int64) bool {
	for _, wid := range c.PackageIDs.Workflows {
		if wid == id {
			return true
		}
	}
	return false
}

// IsInPackageVariable checks if a variable ID is in the package
func (c *ImportContext) IsInPackageVariable(id int64) bool {
	for _, vid := range c.PackageIDs.Variables {
		if vid == id {
			return true
		}
	}
	return false
}

// RemapAgentID remaps an agent ID or returns 0 if not in package
func (c *ImportContext) RemapAgentID(oldID int64) int64 {
	if !c.IsInPackageAgent(oldID) {
		return 0
	}
	if newID, ok := c.AgentIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// RemapPluginID remaps a plugin ID or returns 0 if not in package
func (c *ImportContext) RemapPluginID(oldID int64) int64 {
	if !c.IsInPackagePlugin(oldID) {
		return 0
	}
	if newID, ok := c.PluginIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// RemapWorkflowID remaps a workflow ID or returns 0 if not in package
func (c *ImportContext) RemapWorkflowID(oldID int64) int64 {
	if !c.IsInPackageWorkflow(oldID) {
		return 0
	}
	if newID, ok := c.WorkflowIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// RemapVariableID remaps a variable ID or returns 0 if not in package
func (c *ImportContext) RemapVariableID(oldID int64) int64 {
	if !c.IsInPackageVariable(oldID) {
		return 0
	}
	if newID, ok := c.VariableIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// IsInPackageSpaceModel checks if a space model ID is in the package
func (c *ImportContext) IsInPackageSpaceModel(id int64) bool {
	for _, smid := range c.PackageIDs.SpaceModels {
		if smid == id {
			return true
		}
	}
	return false
}

// RemapSpaceModelID remaps a space model ID or returns 0 if not in package
func (c *ImportContext) RemapSpaceModelID(oldID int64) int64 {
	if !c.IsInPackageSpaceModel(oldID) {
		return 0
	}
	if newID, ok := c.SpaceModelIDMap[oldID]; ok {
		return newID
	}
	return 0
}

// WasSpaceModelCreated checks if a space model was actually created (not skipped)
func (c *ImportContext) WasSpaceModelCreated(newID int64) bool {
	return c.CreatedSpaceModelIDs[newID]
}

// MarkSpaceModelCreated marks a space model as actually created
func (c *ImportContext) MarkSpaceModelCreated(newID int64) {
	c.CreatedSpaceModelIDs[newID] = true
}

// GetEffectiveModelID returns the model ID to use for an agent
// It returns the remapped model ID if it was created, otherwise returns fallback
func (c *ImportContext) GetEffectiveModelID(oldModelID int64) *int64 {
	if !c.IsInPackageSpaceModel(oldModelID) {
		// Not in package, use fallback
		return c.FallbackModelID
	}

	newID := c.RemapSpaceModelID(oldModelID)
	if newID == 0 {
		return c.FallbackModelID
	}

	if c.WasSpaceModelCreated(newID) {
		return &newID
	}

	// Space model was skipped, use fallback
	return c.FallbackModelID
}
