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

	// SkillExecution 控制「该智能体是否允许执行技能脚本(沙箱)」。区别于上面各开关(仅超级体生效),
	// 该项对普通(非超级)智能体同样生效:关闭=只读技能——只读 SKILL.md、不挂 run_bash 等沙箱工具、
	// read_skill 不注入脚本。nil=允许(向后兼容)。
	SkillExecution *bool `json:"skill_execution,omitempty"`

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

// SkillExecutionEnabled 报告是否允许「该智能体执行技能脚本(沙箱)」。也作用于普通体:关闭时
// 普通体把技能当只读文档处理。nil=允许(向后兼容)。
func (c *SuperAgentToolConfig) SkillExecutionEnabled() bool {
	return c == nil || boolEnabled(c.SkillExecution)
}

// mcpSecretRedacted 是任何「读取/回显」路径上替换 MCP env 密钥值的哨兵。写入(update)时
// 若某个 env 值仍等于该哨兵，表示「保留已存密钥」(见 MergeMCPServerSecrets)。
const mcpSecretRedacted = "__ynet_redacted__"

// RedactEnvSecrets 返回 env 的副本，把每个值替换成脱敏哨兵(保留 key，让前端仍能看到有哪些变量)。
// env 为空时原样返回。绝不修改入参。
func RedactEnvSecrets(env map[string]string) map[string]string {
	if len(env) == 0 {
		return env
	}
	out := make(map[string]string, len(env))
	for k := range env {
		out[k] = mcpSecretRedacted
	}
	return out
}

// RedactedForRead 返回可安全回传给客户端的配置副本：每个 MCP server 的 env 值都被替换成
// 脱敏哨兵，凭证(如 Authorization: Bearer …)永不出后端。bool 开关与非密钥字段原样拷贝。
// c 为 nil 时返回 nil，且绝不修改入参。
func (c *SuperAgentToolConfig) RedactedForRead() *SuperAgentToolConfig {
	if c == nil {
		return nil
	}
	cp := *c
	if len(c.MCPServers) > 0 {
		cp.MCPServers = make([]*MCPServerConfig, len(c.MCPServers))
		for i, srv := range c.MCPServers {
			if srv == nil {
				continue
			}
			s := *srv
			s.Env = RedactEnvSecrets(srv.Env)
			cp.MCPServers[i] = &s
		}
	}
	return &cp
}

// MergeMCPServerSecrets 在写入时回填被脱敏的密钥：对 newCfg 里每个 server，凡 env 值仍等于
// 脱敏哨兵的，用 oldCfg 里同名 server、同 key 的已存值替换；哨兵但旧配置里找不到对应值的，
// 直接丢弃(绝不把哨兵字面量落库)。这样客户端可以把读到的(含脱敏 env 的)整份配置原样存回，
// 而不会冲掉真实凭证。直接就地修改 newCfg。
func MergeMCPServerSecrets(newCfg, oldCfg *SuperAgentToolConfig) {
	if newCfg == nil || len(newCfg.MCPServers) == 0 {
		return
	}
	oldByName := map[string]*MCPServerConfig{}
	if oldCfg != nil {
		for _, srv := range oldCfg.MCPServers {
			if srv != nil {
				oldByName[srv.Name] = srv
			}
		}
	}
	for _, srv := range newCfg.MCPServers {
		if srv == nil || len(srv.Env) == 0 {
			continue
		}
		old := oldByName[srv.Name]
		merged := make(map[string]string, len(srv.Env))
		for k, v := range srv.Env {
			if v != mcpSecretRedacted {
				merged[k] = v
				continue
			}
			if old != nil {
				if prev, ok := old.Env[k]; ok {
					merged[k] = prev
				}
			}
			// 哨兵但旧配置无对应值 → 丢弃该 key，绝不落库哨兵字面量。
		}
		srv.Env = merged
	}
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
