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
	"time"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
)

// ManifestVersion is the current version of the export format
const ManifestVersion = "1.0.0"

// Manifest represents the metadata of an exported space package
type Manifest struct {
	Version    string           `json:"version"`
	ExportTime string           `json:"export_time"`
	Source     SourceInfo       `json:"source"`
	Statistics Statistics       `json:"statistics"`
	IDRegistry IDRegistry       `json:"id_registry"`
}

// SourceInfo contains information about the source space
type SourceInfo struct {
	SpaceID    int64  `json:"space_id,string"`
	SpaceName  string `json:"space_name"`
	ExporterID int64  `json:"exporter_id,string"`
}

// Statistics contains counts of exported resources
type Statistics struct {
	Agents      int `json:"agents"`
	Plugins     int `json:"plugins"`
	Workflows   int `json:"workflows"`
	Variables   int `json:"variables"`
	SpaceModels int `json:"space_models"`
}

// IDRegistry contains all resource IDs in the package for quick lookup
type IDRegistry struct {
	Agents      []int64 `json:"agents"`
	Plugins     []int64 `json:"plugins"`
	Workflows   []int64 `json:"workflows"`
	Variables   []int64 `json:"variables"`
	SpaceModels []int64 `json:"space_models"`
}

// ResourceIndex represents the index file for each resource type
type ResourceIndex struct {
	Count int                    `json:"count"`
	Items []ResourceIndexItem    `json:"items"`
}

// ResourceIndexItem represents a single item in the resource index
type ResourceIndexItem struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
	File string `json:"file"`
}

// ExportedAgent represents an exported agent with all its data
type ExportedAgent struct {
	ID        int64  `json:"id,string"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	IconURI   string `json:"icon_uri"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`

	// Core configurations
	BotMode             bot_common.BotMode              `json:"bot_mode"`
	VariablesMetaID     *int64                          `json:"variables_meta_id,string,omitempty"`
	OnboardingInfo      *bot_common.OnboardingInfo      `json:"onboarding_info,omitempty"`
	ModelInfo           *bot_common.ModelInfo           `json:"model_info,omitempty"`
	Prompt              *bot_common.PromptInfo          `json:"prompt,omitempty"`
	SuggestReply        *bot_common.SuggestReplyInfo    `json:"suggest_reply,omitempty"`
	JumpConfig          *bot_common.JumpConfig          `json:"jump_config,omitempty"`
	LayoutInfo          *bot_common.LayoutInfo          `json:"layout_info,omitempty"`
	ShortcutCommand     []string                        `json:"shortcut_command,omitempty"`
	MemoryToolConfig    *bot_common.MemoryToolConfig    `json:"memory_tool_config,omitempty"`
	BackgroundImageList []*bot_common.BackgroundImageInfo `json:"background_image_list,omitempty"`

	// References to other resources (will be remapped during import)
	PluginRefs    []*bot_common.PluginInfo   `json:"plugin_refs,omitempty"`
	KnowledgeRefs *bot_common.Knowledge      `json:"knowledge_refs,omitempty"`
	WorkflowRefs  []*bot_common.WorkflowInfo `json:"workflow_refs,omitempty"`
	DatabaseRefs  []*bot_common.Database     `json:"database_refs,omitempty"`

	// External knowledge (will be cleared during import)
	ExternalKnowledge *bot_common.ExternalKnowledge `json:"external_knowledge,omitempty"`

	// Agent tools (N:N relationship with plugins)
	AgentTools []*ExportedAgentTool `json:"agent_tools,omitempty"`
}

// ExportedAgentTool represents an agent tool binding
type ExportedAgentTool struct {
	ToolID      int64       `json:"tool_id,string"`
	PluginID    int64       `json:"plugin_id,string"`
	ToolName    string      `json:"tool_name"`
	SubURL      string      `json:"sub_url,omitempty"`
	Method      string      `json:"method,omitempty"`
	ToolVersion string      `json:"tool_version,omitempty"`
	Operation   interface{} `json:"operation,omitempty"`
}

// ExportedPlugin represents an exported plugin with all its data
// Note: name and description are stored in manifest.name_for_human/description_for_human
type ExportedPlugin struct {
	ID         int64  `json:"id,string"`
	Name       string `json:"name"`       // Extracted from manifest.name_for_human
	Desc       string `json:"desc"`       // Extracted from manifest.description_for_human
	IconURI    string `json:"icon_uri"`
	ServerURL  string `json:"server_url,omitempty"`
	PluginType int32  `json:"plugin_type"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`

	// Plugin configuration
	Manifest   interface{} `json:"manifest,omitempty"`
	OpenapiDoc interface{} `json:"openapi_doc,omitempty"`

	// Tools defined in this plugin
	Tools []*ExportedTool `json:"tools,omitempty"`
}

// ExportedTool represents a tool within a plugin
type ExportedTool struct {
	ToolID     int64       `json:"tool_id,string"`
	ToolName   string      `json:"tool_name"`
	ToolDesc   string      `json:"tool_desc,omitempty"`
	SubURL     string      `json:"sub_url,omitempty"`
	Method     string      `json:"method,omitempty"`
	Operation  interface{} `json:"operation,omitempty"`
}

// ExportedWorkflow represents an exported workflow with all its data
type ExportedWorkflow struct {
	ID          int64  `json:"id,string"`
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	IconURI     string `json:"icon_uri"`
	Mode        int32  `json:"mode"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`

	// Draft content
	Canvas       interface{} `json:"canvas,omitempty"`
	InputParams  interface{} `json:"input_params,omitempty"`
	OutputParams interface{} `json:"output_params,omitempty"`
}

// ExportedVariable represents an exported variable meta
// Note: biz_id is varchar(128) in the database
type ExportedVariable struct {
	ID        int64  `json:"id,string"`
	BizType   int32  `json:"biz_type"`
	BizID     string `json:"biz_id"` // varchar(128) in database
	CreatorID int64  `json:"creator_id,string"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`

	// Variable list
	VariableList interface{} `json:"variable_list,omitempty"`
}

// ExportedSpaceModel represents an exported space model configuration
type ExportedSpaceModel struct {
	ID              int64       `json:"id,string"`
	ModelEntityID   int64       `json:"model_entity_id,string"`
	ModelEntityName string      `json:"model_entity_name"` // For cross-system matching by name
	Status          int32       `json:"status"`
	CustomConfig    interface{} `json:"custom_config,omitempty"`
	CreatedAt       int64       `json:"created_at"`
	UpdatedAt       int64       `json:"updated_at"`
}

// SpaceResources contains all resources collected from a space
type SpaceResources struct {
	Agents      []*ExportedAgent      `json:"agents"`
	Plugins     []*ExportedPlugin     `json:"plugins"`
	Workflows   []*ExportedWorkflow   `json:"workflows"`
	Variables   []*ExportedVariable   `json:"variables"`
	SpaceModels []*ExportedSpaceModel `json:"space_models"`
}

// ExportResult represents the result of an export operation
type ExportResult struct {
	DownloadURL string    `json:"download_url"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ExpiresAt   time.Time `json:"expires_at"`
	Statistics  Statistics `json:"statistics"`
}
