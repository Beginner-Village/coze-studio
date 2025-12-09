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

package export

import (
	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/api/model/app/bot_common"
)

// SingleAgentDraftModel represents the single_agent_draft table
type SingleAgentDraftModel struct {
	ID                      int64                            `gorm:"column:id;primaryKey"`
	AgentID                 int64                            `gorm:"column:agent_id"`
	CreatorID               int64                            `gorm:"column:creator_id"`
	SpaceID                 int64                            `gorm:"column:space_id"`
	Name                    string                           `gorm:"column:name"`
	Description             *string                          `gorm:"column:description"`
	IconURI                 string                           `gorm:"column:icon_uri"`
	CreatedAt               int64                            `gorm:"column:created_at"`
	UpdatedAt               int64                            `gorm:"column:updated_at"`
	DeletedAt               gorm.DeletedAt                   `gorm:"column:deleted_at"`
	BotMode                 bot_common.BotMode               `gorm:"column:bot_mode"`
	VariablesMetaID         *int64                           `gorm:"column:variables_meta_id"`
	OnboardingInfo          *bot_common.OnboardingInfo       `gorm:"column:onboarding_info;serializer:json"`
	ModelInfo               *bot_common.ModelInfo            `gorm:"column:model_info;serializer:json"`
	Prompt                  *bot_common.PromptInfo           `gorm:"column:prompt;serializer:json"`
	Plugin                  []*bot_common.PluginInfo         `gorm:"column:plugin;serializer:json"`
	Knowledge               *bot_common.Knowledge            `gorm:"column:knowledge;serializer:json"`
	ExternalKnowledge       *bot_common.ExternalKnowledge    `gorm:"column:external_knowledge;serializer:json"`
	Workflow                []*bot_common.WorkflowInfo       `gorm:"column:workflow;serializer:json"`
	SuggestReply            *bot_common.SuggestReplyInfo     `gorm:"column:suggest_reply;serializer:json"`
	JumpConfig              *bot_common.JumpConfig           `gorm:"column:jump_config;serializer:json"`
	BackgroundImageInfoList []*bot_common.BackgroundImageInfo `gorm:"column:background_image_info_list;serializer:json"`
	DatabaseConfig          []*bot_common.Database           `gorm:"column:database_config;serializer:json"`
	ShortcutCommand         []string                         `gorm:"column:shortcut_command;serializer:json"`
	LayoutInfo              *bot_common.LayoutInfo           `gorm:"column:layout_info;serializer:json"`
	MemoryToolConfig        *bot_common.MemoryToolConfig     `gorm:"column:memory_tool_config;serializer:json"`
}

func (SingleAgentDraftModel) TableName() string {
	return "single_agent_draft"
}

// AgentToolDraftModel represents the agent_tool_draft table
// Note: This table does NOT have deleted_at or updated_at columns
type AgentToolDraftModel struct {
	ID          int64       `gorm:"column:id;primaryKey"`
	AgentID     int64       `gorm:"column:agent_id"`
	ToolID      int64       `gorm:"column:tool_id"`
	PluginID    int64       `gorm:"column:plugin_id"`
	ToolName    string      `gorm:"column:tool_name"`
	SubURL      string      `gorm:"column:sub_url"`
	Method      string      `gorm:"column:method"`
	ToolVersion string      `gorm:"column:tool_version"`
	Operation   interface{} `gorm:"column:operation;serializer:json"`
	CreatedAt   int64       `gorm:"column:created_at"`
}

func (AgentToolDraftModel) TableName() string {
	return "agent_tool_draft"
}

// PluginDraftModel represents the plugin_draft table
// Note: name/description are stored in manifest.name_for_human/description_for_human
type PluginDraftModel struct {
	ID          int64          `gorm:"column:id;primaryKey"`
	SpaceID     int64          `gorm:"column:space_id"`
	DeveloperID int64          `gorm:"column:developer_id"`
	AppID       int64          `gorm:"column:app_id"`
	IconURI     string         `gorm:"column:icon_uri"`
	ServerURL   string         `gorm:"column:server_url"`
	PluginType  int32          `gorm:"column:plugin_type"`
	Manifest    interface{}    `gorm:"column:manifest;serializer:json"`
	OpenapiDoc  interface{}    `gorm:"column:openapi_doc;serializer:json"`
	CreatedAt   int64          `gorm:"column:created_at"`
	UpdatedAt   int64          `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (PluginDraftModel) TableName() string {
	return "plugin_draft"
}

// WorkflowMetaModel represents the workflow_meta table
type WorkflowMetaModel struct {
	ID          int64          `gorm:"column:id;primaryKey"`
	SpaceID     int64          `gorm:"column:space_id"`
	AppID       *int64         `gorm:"column:app_id"`
	Name        string         `gorm:"column:name"`
	Description *string        `gorm:"column:description"`
	IconURI     *string        `gorm:"column:icon_uri"`
	Mode        int32          `gorm:"column:mode"`
	Status      int32          `gorm:"column:status"`
	CreatorID   int64          `gorm:"column:creator_id"`
	UpdaterID   int64          `gorm:"column:updater_id"`
	CreatedAt   int64          `gorm:"column:created_at"`
	UpdatedAt   int64          `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (WorkflowMetaModel) TableName() string {
	return "workflow_meta"
}

// WorkflowDraftModel represents the workflow_draft table
type WorkflowDraftModel struct {
	ID           int64          `gorm:"column:id;primaryKey"`
	Canvas       interface{}    `gorm:"column:canvas;serializer:json"`
	InputParams  interface{}    `gorm:"column:input_params;serializer:json"`
	OutputParams interface{}    `gorm:"column:output_params;serializer:json"`
	CommitID     string         `gorm:"column:commit_id"`
	CreatedAt    int64          `gorm:"column:created_at"`
	UpdatedAt    int64          `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (WorkflowDraftModel) TableName() string {
	return "workflow_draft"
}

// VariablesMetaModel represents the variables_meta table
// Note: This table does NOT have a deleted_at column
// Note: biz_id is varchar(128) in the database, not bigint
type VariablesMetaModel struct {
	ID           int64       `gorm:"column:id;primaryKey"`
	BizType      int32       `gorm:"column:biz_type"`
	BizID        string      `gorm:"column:biz_id"` // varchar(128)
	CreatorID    int64       `gorm:"column:creator_id"`
	Version      string      `gorm:"column:version"`
	VariableList interface{} `gorm:"column:variable_list;serializer:json"`
	CreatedAt    int64       `gorm:"column:created_at"`
	UpdatedAt    int64       `gorm:"column:updated_at"`
}

func (VariablesMetaModel) TableName() string {
	return "variables_meta"
}

// SpaceModelModel represents the space_model table
type SpaceModelModel struct {
	ID            int64          `gorm:"column:id;primaryKey"`
	SpaceID       int64          `gorm:"column:space_id"`
	ModelEntityID int64          `gorm:"column:model_entity_id"`
	UserID        int64          `gorm:"column:user_id"`
	Status        int32          `gorm:"column:status"`
	CustomConfig  interface{}    `gorm:"column:custom_config;serializer:json"`
	CreatedAt     int64          `gorm:"column:created_at"`
	UpdatedAt     int64          `gorm:"column:updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (SpaceModelModel) TableName() string {
	return "space_model"
}
