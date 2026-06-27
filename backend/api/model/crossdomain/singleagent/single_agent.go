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

package singleagent

import (
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/agentrun"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/plugin"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
)

type EventType string

const (
	EventTypeOfChatModelAnswer        EventType = "chatmodel_answer"
	EventTypeOfToolsAsChatModelStream EventType = "tools_as_chatmodel_answer"
	EventTypeOfToolMidAnswer          EventType = "tool_mid_answer"
	EventTypeOfToolsMessage           EventType = "tools_message"
	EventTypeOfFuncCall               EventType = "func_call"
	EventTypeOfSuggest                EventType = "suggest"
	EventTypeOfKnowledge              EventType = "knowledge"
	EventTypeOfInterrupt              EventType = "interrupt"
)

type AgentEvent struct {
	EventType EventType

	ToolMidAnswer         *schema.StreamReader[*schema.Message]
	ToolAsChatModelAnswer *schema.StreamReader[*schema.Message]

	ChatModelAnswer *schema.StreamReader[*schema.Message]
	ToolsMessage    []*schema.Message
	FuncCall        *schema.Message
	Suggest         *schema.Message
	Knowledge       []*schema.Document
	Interrupt       *InterruptInfo
}

type SingleAgent struct {
	AgentID   int64
	CreatorID int64
	SpaceID   int64
	Name      string
	Desc      string
	IconURI   string
	CreatedAt int64
	UpdatedAt int64
	Version   string
	DeletedAt gorm.DeletedAt

	VariablesMetaID         *int64
	OnboardingInfo          *bot_common.OnboardingInfo
	ModelInfo               *bot_common.ModelInfo
	Prompt                  *bot_common.PromptInfo
	Plugin                  []*bot_common.PluginInfo
	Knowledge               *bot_common.Knowledge
	ExternalKnowledge       *bot_common.ExternalKnowledge
	Workflow                []*bot_common.WorkflowInfo
	SuggestReply            *bot_common.SuggestReplyInfo
	JumpConfig              *bot_common.JumpConfig
	BackgroundImageInfoList []*bot_common.BackgroundImageInfo
	Database                []*bot_common.Database
	BotMode                 bot_common.BotMode
	LayoutInfo              *bot_common.LayoutInfo
	ShortcutCommand         []string
	MemoryToolConfig        *bot_common.MemoryToolConfig
	BoundCards              []*bot_common.BoundCardInfo
	SkillInfoList           []*SkillReference
	ForceToolReturn         *bool
	AgentType               string                // agent_type：""/normal=普通；super=超级智能体（运行时路由）
	Strategies              []int64               // bound strategy IDs (latest-published semantics, no per-strategy version pinning in v1)
	SuperAgentToolConfig    *SuperAgentToolConfig // 超级体能力开关（沙箱/网络/工具权限 + MCP），仅超级体生效
	SourceProductID         int64                 `json:"source_product_id,omitempty"`
	SourceProductVersion    string                `json:"source_product_version,omitempty"`
}

// SuperAgentToolConfig 是超级智能体的能力开关配置（per-agent）。所有开关默认全开
// （nil 表示开启，保持向后兼容）；关闭沙箱即「纯 MCP 模式」。该结构以 JSON 存进
// single_agent_draft.super_agent_tool_config 列，并预留 MCPServers 给 MCP 动态接入。
type SuperAgentToolConfig struct {
	// Sandbox 沙箱总开关。关闭后不挂任何沙箱工具（run_bash/读写/grep/glob/update_plan），
	// 进入「纯 MCP」模式（依赖沙箱的 web_search/web_fetch 也随之失效）。
	Sandbox     *bool `json:"sandbox,omitempty"`
	WebSearch   *bool `json:"web_search,omitempty"`
	WebFetch    *bool `json:"web_fetch,omitempty"`
	RunBash     *bool `json:"run_bash,omitempty"` // 沙箱子开关：仅关 run_bash，保留文件类工具
	DeepTask    *bool `json:"deep_task,omitempty"`
	SkillManage *bool `json:"skill_manage,omitempty"`

	// MCPServers 动态 MCP server 列表（第二块接入）。
	MCPServers []*MCPServerConfig `json:"mcp_servers,omitempty"`
}

// MCPServerConfig 描述一个可动态接入的 MCP server（stdio / sse / streamable-http）。
type MCPServerConfig struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`              // "stdio" | "sse" | "streamable_http"
	Command    string            `json:"command,omitempty"` // stdio: 启动命令
	Args       []string          `json:"args,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	URL        string            `json:"url,omitempty"` // sse / http
	TimeoutSec int64             `json:"timeout_sec,omitempty"`
	Enabled    *bool             `json:"enabled,omitempty"`
}

// 下列 helper 均为 nil-safe：config 或字段为 nil 时一律视作「开启」（默认全开）。
func boolEnabled(v *bool) bool                         { return v == nil || *v }
func (c *SuperAgentToolConfig) SandboxEnabled() bool   { return c == nil || boolEnabled(c.Sandbox) }
func (c *SuperAgentToolConfig) WebSearchEnabled() bool { return c == nil || boolEnabled(c.WebSearch) }
func (c *SuperAgentToolConfig) WebFetchEnabled() bool  { return c == nil || boolEnabled(c.WebFetch) }
func (c *SuperAgentToolConfig) RunBashEnabled() bool   { return c == nil || boolEnabled(c.RunBash) }
func (c *SuperAgentToolConfig) DeepTaskEnabled() bool  { return c == nil || boolEnabled(c.DeepTask) }
func (c *SuperAgentToolConfig) SkillManageEnabled() bool {
	return c == nil || boolEnabled(c.SkillManage)
}

// SkillReference is a lightweight reference for Bot binding.
type SkillReference struct {
	SkillID          int64  `json:"skill_id"`
	SkillName        string `json:"skill_name"`
	SkillDescription string `json:"skill_description"`
}

type InterruptEventType int64

const (
	InterruptEventType_LocalPlugin         InterruptEventType = 1
	InterruptEventType_Question            InterruptEventType = 2
	InterruptEventType_RequireInfos        InterruptEventType = 3
	InterruptEventType_SceneChat           InterruptEventType = 4
	InterruptEventType_InputNode           InterruptEventType = 5
	InterruptEventType_WorkflowLocalPlugin InterruptEventType = 6
	InterruptEventType_OauthPlugin         InterruptEventType = 7
	InterruptEventType_WorkflowLLM         InterruptEventType = 100
)

type InterruptInfo struct {
	AllToolInterruptData map[string]*plugin.ToolInterruptEvent
	AllWfInterruptData   map[string]*crossworkflow.ToolInterruptEvent
	ToolCallID           string
	InterruptType        InterruptEventType
	InterruptID          string

	ChatflowInterrupt *crossworkflow.StateMessage
}

type ExecuteRequest struct {
	Identity *AgentIdentity
	UserID   string

	ConversationID int64

	Input        *schema.Message
	History      []*schema.Message
	ResumeInfo   *InterruptInfo
	PreCallTools []*agentrun.ToolsRetriever

	// Variables 会话级自定义变量，用于覆盖智能体预设变量
	Variables map[string]string

	// Ext 透传运行时扩展字段，例如工作流画布模式标记。
	Ext map[string]string
}

type AgentIdentity struct {
	AgentID int64
	// State   AgentState
	Version     string
	IsDraft     bool
	ConnectorID int64
}
