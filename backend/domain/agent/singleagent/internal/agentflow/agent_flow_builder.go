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

package agentflow

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	canvasauto "github.com/ynet-dev/ynet-studio/backend/application/workflow/canvasautomation"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/modelmgr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/slices"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// Session key is now retrieved from database when needed

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type Config struct {
	Agent         *entity.SingleAgent
	UserID        string
	Identity      *entity.AgentIdentity
	ModelMgr      modelmgr.Manager
	ModelFactory  chatmodel.Factory
	CPStore       compose.CheckPointStore
	Embedder      embedding.Embedder
	SessionCookie string // 用户的session cookie，用于调用RAGFlow API
	Ext           map[string]string
}

const (
	keyOfPersonRender           = "persona_render"
	keyOfBoundCardsRender       = "bound_cards_render"
	keyOfSkillsRender           = "skills_render"
	keyOfKnowledgeRetriever     = "knowledge_retriever"
	keyOfKnowledgeRetrieverPack = "knowledge_retriever_pack"
	keyOfPromptVariables        = "prompt_variables"
	keyOfPromptTemplate         = "prompt_template"
	keyOfReActAgent             = "react_agent"
	keyOfReActAgentToolsNode    = "agent_tool"
	keyOfReActAgentChatModel    = "re_act_chat_model"
	keyOfLLM                    = "llm"
	keyOfToolsPreRetriever      = "tools_pre_retriever"
)

func BuildAgent(ctx context.Context, conf *Config) (r *AgentRunner, err error) {
	// Session key will be retrieved from database when needed by external knowledge tool

	workflowCanvasMode := isSuperAgent(conf) && workflowCanvasModeEnabled(conf.Ext)

	persona := conf.Agent.Prompt.GetPrompt()
	if isSuperAgent(conf) {
		if workflowCanvasMode {
			persona = persona + "\n\n" + SuperAgentWorkflowCanvasPrompt
		} else {
			persona = persona + "\n\n" + SuperAgentExtraPrompt
		}
		// Hermes 式:开场自动注入该用户的 USER.md/MEMORY.md 长期记忆,让超级体"一上来就懂这个人"。
		// sandboxKeyFor 是纯函数,可在此提前算(与下方 sandboxKey 一致)。
		memKey := sandboxKeyFor(conf.Identity.ConnectorID, conf.Agent.AgentID, conf.UserID)
		if mem := loadSuperAgentMemoryDocs(ctx, memKey); mem != "" {
			persona = persona + "\n\n" + mem
		}
	}

	// 策略 L1 提示:绑定策略后,强约束智能体走 scenes→caps→run 渐进披露流程,
	// 显著提升账户/业务类请求的触达可靠性(避免凭空臆测或向用户索要可由 run 获取的信息)。
	if len(conf.Agent.Strategies) > 0 {
		persona = persona + "\n\n" + StrategyL1Prompt
	}

	avConf := &variableConf{
		Agent:       conf.Agent,
		UserID:      conf.UserID,
		ConnectorID: conf.Identity.ConnectorID,
		Embedder:    conf.Embedder,
	}
	promptVars := &promptVariables{
		Agent: conf.Agent,
		avs:   nil, // 不再直接注入变量
	}

	personaVars := &personaRender{
		personaVariableNames: extractJinja2Placeholder(persona),
		persona:              persona,
		variables:            nil, // 不再直接注入变量到 persona
	}

	// Create bound cards renderer
	boundCardsRenderer := newBoundCardsRender(conf.Agent.BoundCards, nil)

	// Create skills renderer
	skillsRenderer := newSkillsRender(conf.Agent.SkillInfoList, isSuperAgent(conf))

	// Load model info first so it can be used for knowledge retrieval
	modelInfo, err := loadModelInfo(ctx, conf.ModelMgr, ptr.From(conf.Agent.ModelInfo.ModelId), conf.Agent.SpaceID)
	if err != nil {
		return nil, err
	}

	kr, err := newKnowledgeRetriever(ctx, &retrieverConfig{
		knowledgeConfig: conf.Agent.Knowledge,
		modelInfo:       modelInfo, // Pass model info for query rewrite and NL2SQL
	})
	if err != nil {
		return nil, err
	}

	// 超级智能体常把大段文件内容塞进 write_file/run_bash 等工具调用的参数里。
	// max_tokens 只限制「模型单次输出长度」(与输入上下文无关),设小了会把工具调用的
	// JSON 截断成残缺串(报 "unexpected end of JSON input")。现代模型(qwen3-max 等)
	// 最大输出本身就很大,这里对超级体**彻底不传 max_tokens**,让模型用满额输出。
	// 需同时清掉:① 智能体级覆盖;② 模型基础配置里的默认值(否则会回落到那个默认)。
	if isSuperAgent(conf) {
		if conf.Agent.ModelInfo != nil {
			conf.Agent.ModelInfo.MaxTokens = nil
		}
		if modelInfo != nil {
			modelInfo.Meta.ConnConfig.MaxTokens = nil
		}
	}

	chatModel, err := newChatModel(ctx, &config{
		modelFactory:      conf.ModelFactory,
		modelInfo:         modelInfo,
		agentModelSetting: conf.Agent.ModelInfo,
	})
	if err != nil {
		return nil, err
	}

	requireCheckpoint := false
	var pluginTools []tool.InvokableTool
	if !workflowCanvasMode {
		pluginTools, err = newPluginTools(ctx, &toolConfig{
			spaceID:       conf.Agent.SpaceID,
			userID:        conf.UserID,
			agentIdentity: conf.Identity,
			toolConf:      conf.Agent.Plugin,
		})
		if err != nil {
			return nil, err
		}
	}
	tr := newPreToolRetriever(&toolPreCallConf{})

	var wfTools []workflow.ToolFromWorkflow
	returnDirectlyTools := make(map[string]struct{})
	if !workflowCanvasMode {
		wfTools, returnDirectlyTools, err = newWorkflowTools(ctx, &workflowConfig{
			wfInfos: conf.Agent.Workflow,
		})
		if err != nil {
			return nil, err
		}
	}

	var dbTools []tool.InvokableTool
	if !workflowCanvasMode && len(conf.Agent.Database) > 0 {
		dbTools, err = newDatabaseTools(ctx, &databaseConfig{
			spaceID:       conf.Agent.SpaceID,
			userID:        conf.UserID,
			agentIdentity: conf.Identity,
			databaseConf:  conf.Agent.Database,
		})
		if err != nil {
			return nil, err
		}
	}

	var strategyTools []tool.InvokableTool
	if len(conf.Agent.Strategies) > 0 {
		strategyTools, err = newStrategyTools(ctx, &strategyConfig{
			spaceID:         conf.Agent.SpaceID,
			userID:          conf.UserID,
			agentIdentity:   conf.Identity,
			strategyIDs:     conf.Agent.Strategies,
			forceToolReturn: conf.Agent.ForceToolReturn != nil && *conf.Agent.ForceToolReturn,
		})
		if err != nil {
			return nil, err
		}
	}

	var avTools []tool.InvokableTool
	// 检查记忆工具配置开关
	// 如果配置为启用(默认)或未配置，则添加记忆工具
	// 如果配置为禁用，则不添加记忆工具
	memoryToolEnabled := true // 默认启用，保持向后兼容
	if conf.Agent.MemoryToolConfig != nil {
		memoryToolEnabled = conf.Agent.MemoryToolConfig.Mode == nil || *conf.Agent.MemoryToolConfig.Mode == 1
	}

	// 超级智能体改用自有的 per-user DB 记忆(memory_save/memory_recall),不再挂老的
	// 关键词记忆工具(getKeywordMemory/setKeywordMemory 等),避免两套记忆并存/不同步。
	// 普通单智能体保持原有关键词记忆,行为不变。
	if memoryToolEnabled && !isSuperAgent(conf) {
		avTools, err = newAgentVariableTools(ctx, avConf)
		if err != nil {
			return nil, err
		}
	}

	// 添加外部知识库工具（如果配置了dataset_ids）
	var externalKnowledgeTools []tool.InvokableTool
	if !workflowCanvasMode && conf.Agent.ExternalKnowledge != nil && len(conf.Agent.ExternalKnowledge.DatasetIds) > 0 {
		externalKnowledgeConfig := &externalKnowledgeConfig{
			spaceID:           conf.Agent.SpaceID,
			userID:            conf.UserID,
			agentIdentity:     conf.Identity,
			botID:             fmt.Sprintf("%d", conf.Agent.AgentID),
			externalKnowledge: conf.Agent.ExternalKnowledge,
			sessionCookie:     "", // 不再使用，改为从数据库获取
		}
		externalKnowledgeTools, err = newExternalKnowledgeTools(ctx, externalKnowledgeConfig)
		if err != nil {
			return nil, err
		}
	}

	containWfTool := false

	if len(wfTools) > 0 {
		containWfTool = true
	}
	agentTools := make([]tool.BaseTool, 0, len(pluginTools)+len(wfTools)+len(dbTools)+len(strategyTools)+len(avTools)+len(externalKnowledgeTools))
	agentTools = append(agentTools, slices.Transform(pluginTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)
	agentTools = append(agentTools, slices.Transform(wfTools, func(a workflow.ToolFromWorkflow) tool.BaseTool { return a.(tool.BaseTool) })...)
	agentTools = append(agentTools, slices.Transform(dbTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)
	agentTools = append(agentTools, slices.Transform(strategyTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)

	agentTools = append(agentTools, slices.Transform(avTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)

	// 添加外部知识库工具
	agentTools = append(agentTools, slices.Transform(externalKnowledgeTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)

	// 沙箱 key（按 connector/agent/user_id 稳定派生），技能工具与沙箱工具共用。
	sandboxKey := sandboxKeyFor(conf.Identity.ConnectorID, conf.Agent.AgentID, conf.UserID)

	// 沙箱服务是否可用：现场未配置沙箱(SANDBOX_ENABLED=false)时启动阶段不会 SetDefaultSVC，
	// 此处 DefaultSVC()==nil。沙箱不可用时对所有 agent 一律进入「无沙箱」降级路径。
	sandboxAvailable := crosssandbox.DefaultSVC() != nil

	// 超级体能力开关(per-agent，默认全开)。
	superToolCfg := conf.Agent.SuperAgentToolConfig
	// 沙箱对该 agent 是否关闭 = 沙箱服务不可用(现场未配沙箱) 或 超级体 per-agent 总开关关闭(纯 MCP 模式)。
	sandboxOff := resolveSandboxOff(sandboxAvailable, isSuperAgent(conf), superToolCfg.SandboxEnabled())

	// 普通体「只读技能」模式：只读取技能内容、不执行脚本。判定 = per-agent 开关
	// (SuperAgentToolConfig.SkillExecution，默认允许) 关闭，或 全局 env SKILL_EXECUTION_DISABLED
	// 兜底关闭。仅对普通体生效；超级体(harness)恒不受影响，保持「super/normal 互不影响」铁律。
	// 此外：沙箱不可用时普通体也强制只读(read_skill 仍返回 SKILL.md，但不注入脚本、不挂 run_bash)。
	skillReadOnly := skillReadOnlyMode(isSuperAgent(conf), superToolCfg.SkillExecutionEnabled())
	if sandboxOff && !isSuperAgent(conf) {
		skillReadOnly = true
	}
	if skillReadOnly {
		logs.CtxInfof(ctx, "[BuildAgent] skill read-only mode ON (per-agent SkillExecution off / SKILL_EXECUTION_DISABLED / no sandbox): no sandbox/script execution for this normal agent")
	}

	// 技能机制(标准 Agent Skills 渐进式披露):
	// - L1 元数据:系统提示注入技能名+简介(见 skillsRenderer)。
	// - L2 说明:read_skill 工具按需返回技能 SKILL.md(显式"打开"技能的工具,所有 agent 都挂)。
	// - L3 脚本/资源:超级智能体额外把技能 eager 落盘到沙箱 /skills/<name>/(SKILL.md + 脚本),
	//   模型按 SKILL.md 指引用 run_bash 执行文件夹里的固定脚本。
	var skillTools []tool.InvokableTool
	if !workflowCanvasMode {
		skillTools = newSkillTools(conf.Agent.SpaceID, sandboxKey, conf.Agent.SkillInfoList, !skillReadOnly && !sandboxOff)
	}
	agentTools = append(agentTools, slices.Transform(skillTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)
	if isSuperAgent(conf) && !workflowCanvasMode && !sandboxOff && len(conf.Agent.SkillInfoList) > 0 {
		// manifest 守卫:技能集合未变时整段跳过,仅首次/变更时才真正落盘。
		syncBoundSkillsToSandbox(ctx, sandboxKey, conf.Agent.SpaceID, conf.Agent.SkillInfoList)
		logs.CtxInfof(ctx, "[BuildAgent] ensured %d skill folder(s) in sandbox /skills (manifest-guarded, key=%s)", len(conf.Agent.SkillInfoList), sandboxKey)
	}

	// 添加沙箱工具 (run_bash/read_file/write_file/list_files 等)。超级体默认挂(需读 /skills/)，
	// 但沙箱总开关关闭时不挂；run_bash 子开关关闭时单独剔除 run_bash。
	//
	// 虚拟员工实例(SourceProductID != 0)：/skills 以只读挂载，防止运行时改写技能模板。
	instanceMode := isInstanceAgent(conf)
	mountSandbox := shouldMountSandbox(isSuperAgent(conf), workflowCanvasMode, sandboxOff, skillReadOnly, len(conf.Agent.SkillInfoList))
	if mountSandbox {
		sandboxTools := newSandboxTools(sandboxKey, instanceMode)
		if isSuperAgent(conf) {
			sandboxTools = gateSuperAgentTools(ctx, sandboxTools, superToolCfg, sandboxOff)
		}
		agentTools = append(agentTools, slices.Transform(sandboxTools, func(a tool.InvokableTool) tool.BaseTool {
			return a
		})...)
		if len(sandboxTools) > 0 {
			logs.CtxInfof(ctx, "[BuildAgent] Mounted %d sandbox tools (key=%s, readonlySkills=%v)", len(sandboxTools), sandboxKey, instanceMode)
		}
		// 虚拟员工实例：调用 EnsureSandboxWithTemplate 确保沙箱处于 running，
		// 仅在冷启动时用模板种子化 /workspace（实例检查点 > 模板 > 空白）。
		// 对已运行的沙箱（即已有积累工作区的轮次）什么都不做，不再覆写。
		if instanceMode {
			templateKey := instanceTemplateObjectKey(conf.Agent.SourceProductID, conf.Agent.SourceProductVersion)
			svc := crosssandbox.DefaultSVC()
			if svc != nil {
				if err := svc.EnsureSandboxWithTemplate(ctx, sandboxKey, templateKey, true); err != nil {
					logs.CtxWarnf(ctx, "[BuildAgent] instance EnsureSandboxWithTemplate failed: %v (continuing)", err)
				} else {
					logs.CtxInfof(ctx, "[BuildAgent] instance sandbox ensured (key=%s, template=%q)", sandboxKey, templateKey)
				}
			}
		}
	} else if sandboxOff {
		logs.CtxInfof(ctx, "[BuildAgent] sandbox disabled by config (pure-MCP mode), key=%s", sandboxKey)
	}

	// 自动绑定技能提示词中引用的工作流
	existingWorkflowIDs := make(map[int64]struct{}, len(conf.Agent.Workflow))
	for _, wf := range conf.Agent.Workflow {
		existingWorkflowIDs[wf.GetWorkflowId()] = struct{}{}
	}
	var skillRes *skillResources
	var skillResErr error
	if !workflowCanvasMode {
		skillRes, skillResErr = resolveSkillResources(ctx, conf.Agent.SkillInfoList, existingWorkflowIDs)
	}
	if skillResErr != nil {
		logs.CtxWarnf(ctx, "[BuildAgent] resolveSkillResources failed: %v", skillResErr)
	} else if skillRes != nil {
		// Workflow references: auto-bind as workflow tools.
		if len(skillRes.WorkflowIDs) > 0 {
			logs.CtxInfof(ctx, "[BuildAgent] Auto-binding %d workflow(s) from skill prompts", len(skillRes.WorkflowIDs))
			skillWfInfos := make([]*bot_common.WorkflowInfo, 0, len(skillRes.WorkflowIDs))
			for _, id := range skillRes.WorkflowIDs {
				skillWfInfos = append(skillWfInfos, &bot_common.WorkflowInfo{
					WorkflowId: ptr.Of(id),
				})
			}
			skillWfTools, skillReturnDirectly, wfErr := newWorkflowTools(ctx, &workflowConfig{wfInfos: skillWfInfos})
			if wfErr == nil {
				agentTools = append(agentTools, slices.Transform(skillWfTools, func(a workflow.ToolFromWorkflow) tool.BaseTool {
					return a.(tool.BaseTool)
				})...)
				for k, v := range skillReturnDirectly {
					returnDirectlyTools[k] = v
				}
				if len(skillWfTools) > 0 {
					containWfTool = true
				}
			} else {
				logs.CtxWarnf(ctx, "[BuildAgent] Failed to create workflow tools from skill prompts: %v", wfErr)
			}
		}

		// Plugin references: NOT auto-mounted yet.
		// A skill {plugin:..|id:..} reference carries a *plugin* id (res_id from
		// library_resource_list), but every cross-domain plugin tool constructor
		// (newPluginTools / GetPluginInvokableTools / GetPluginToolsInfo) requires
		// explicit *tool* ids, and there is no contract method to enumerate a
		// plugin's tools from just its plugin id. Mounting it safely would require
		// a new cross-domain API, which is out of scope here.
		// TODO(skill-resources): add a cross-domain "list tools by plugin id" method,
		// then build invokable tools for skillRes.PluginIDs and append to agentTools.
		if len(skillRes.PluginIDs) > 0 {
			logs.CtxWarnf(ctx, "[BuildAgent] %d plugin reference(s) in skill prompts are not auto-mounted (no contract to list tools by plugin id): %v", len(skillRes.PluginIDs), skillRes.PluginIDs)
		}

		// Knowledge references: NOT auto-mounted yet.
		// Knowledge entities are fetchable via MGetKnowledgeByID, but the only
		// knowledge tool (knowledgeTool) needs a per-request Input message and
		// chat history (GetHistory) that are only available at run time, not at
		// agent-build time. The agent's own knowledge is wired as a graph
		// retriever node (node_retriever.go) for the same reason. Mounting it as a
		// build-time tool would pass empty query/history and is unsafe.
		// TODO(skill-resources): expose knowledge as a retriever-style node (or a
		// tool fed by the runtime request) and mount skillRes.KnowledgeIDs there.
		if len(skillRes.KnowledgeIDs) > 0 {
			logs.CtxWarnf(ctx, "[BuildAgent] %d knowledge reference(s) in skill prompts are not auto-mounted (needs runtime query/history): %v", len(skillRes.KnowledgeIDs), skillRes.KnowledgeIDs)
		}
	}

	// 如果开启 ForceToolReturn，清空 returnDirectlyTools
	if conf.Agent.ForceToolReturn != nil && *conf.Agent.ForceToolReturn {
		returnDirectlyTools = make(map[string]struct{})
		logs.CtxInfof(ctx, "[BuildAgent] ForceToolReturn enabled, all tools return to model")
	}

	// Experimental DeepAgents engine gate (default OFF). When AGENT_ENGINE=deepagents,
	// log that the experimental engine was requested. The DeepAgents engine is not yet
	// wired into the streaming/interrupt flow (adk deep agents have no compose.AnyGraph
	// to inline; see deepagents_engine.go for the blocker + TODO), so we intentionally
	// fall back to the unchanged ReAct path here. This keeps default behavior identical
	// and the change minimal/reversible.
	// TODO(deepagents): once an adk.Runner-based execution path exists, branch to
	// buildDeepAgent(ctx, conf, chatModel, agentTools) instead of building ReAct.
	if deepAgentsEnabled() {
		logs.CtxInfof(ctx, "[BuildAgent] AGENT_ENGINE=deepagents requested; experimental DeepAgents engine not yet wired, falling back to ReAct")
	}

	// 纯按类型区分:deep_task / 沙箱 / 扩展工具只挂给超级 agent。
	// 普通(老)智能体沿用原来那一套,完全不碰沙箱与超级工具。
	if shouldMountDeepTask(isSuperAgent(conf), workflowCanvasMode, sandboxOff, superToolCfg.DeepTaskEnabled()) {
		if dt, derr := newDeepTaskTool(ctx, chatModel, append([]tool.BaseTool(nil), agentTools...)); derr != nil {
			logs.CtxWarnf(ctx, "[BuildAgent] build deep_task tool failed: %v", derr)
		} else if dt != nil {
			agentTools = append(agentTools, dt)
			logs.CtxInfof(ctx, "[BuildAgent] mounted deep_task tool for super agent")
		}
	}

	// 超级 agent 可插拔扩展工具（P5）：只挂给超级 agent，普通 agent 不受影响。
	// 按能力开关门控 web_search/web_fetch/skill_manage（纯 MCP 模式下 web_* 一并剔除）。
	if isSuperAgent(conf) {
		extTools := newSuperAgentExtensionTools(superAgentToolDeps{
			SandboxKey:  sandboxKey,
			UserID:      parseSuperAgentUserID(conf.UserID),
			SpaceID:     conf.Agent.SpaceID,
			AgentID:     conf.Agent.AgentID,
			Ext:         conf.Ext,
			CanvasRelay: canvasauto.SharedWorkflowCommandRelay(),
			Ledger:      newTurnCanvasLedger(),
		})
		extTools = gateSuperAgentTools(ctx, extTools, superToolCfg, sandboxOff)
		for _, et := range extTools {
			agentTools = append(agentTools, et)
		}
		if len(extTools) > 0 {
			logs.CtxInfof(ctx, "[BuildAgent] mounted %d super-agent extension tools", len(extTools))
		}

		// MCP 动态接入：把每个启用的 MCP server 暴露的工具挂进 ReAct 循环（官方 eino-ext/mcp 适配器）。
		if !workflowCanvasMode && superToolCfg != nil && len(superToolCfg.MCPServers) > 0 {
			mcpTools := newMCPTools(ctx, superToolCfg.MCPServers)
			agentTools = append(agentTools, mcpTools...)
			if len(mcpTools) > 0 {
				logs.CtxInfof(ctx, "[BuildAgent] mounted %d MCP tool(s) from %d server(s)", len(mcpTools), len(superToolCfg.MCPServers))
			}
		}
	}

	if workflowCanvasMode {
		before := len(agentTools)
		agentTools = filterWorkflowCanvasTools(ctx, agentTools)
		returnDirectlyTools = make(map[string]struct{})
		containWfTool = false
		logs.CtxInfof(ctx, "[BuildAgent] workflow canvas mode enabled: kept %d/%d canvas tool(s)", len(agentTools), before)
	}

	var isReActAgent bool
	if len(agentTools) > 0 {
		isReActAgent = true
		requireCheckpoint = true
		if modelInfo.Meta.Capability != nil && !modelInfo.Meta.Capability.FunctionCall {
			return nil, fmt.Errorf("model %v does not support function call", modelInfo.Name)
		}
	}

	var agentGraph compose.AnyGraph
	var agentNodeOpts []compose.GraphAddNodeOpt
	var agentNodeName string
	if isReActAgent {
		reactConfig := &react.AgentConfig{
			ToolCallingModel: chatModel,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
				// 强制顺序执行工具，确保正确的 ReAct 流程：
				// 每个工具调用完成后立即返回结果，然后模型基于结果决定下一步
				// 而不是并行执行所有工具后一起返回
				ExecuteSequentially: true,
			},
			ToolReturnDirectly: returnDirectlyTools,
			ModelNodeName:      keyOfReActAgentChatModel,
			ToolsNodeName:      keyOfReActAgentToolsNode,
			// 最大步数：普通单 Agent 默认 30；超级体不设实际上限(对标 Claude Code/Codex 的长程任务)
			MaxStep: agentMaxStep(isSuperAgent(conf)),
		}

		// 根据模型类型自适应选择StreamToolCallChecker
		// 某些模型（如Qwen、Claude）会先输出文本再输出工具调用
		// 需要使用兼容的checker
		needsCompatibleChecker := false

		// 首先检查是否通过环境变量强制使用兼容模式
		if shouldUseCompatibleChecker() {
			needsCompatibleChecker = true
			logs.CtxInfof(ctx, "[AgentBuilder] Force using compatible tool call checker (env: FORCE_COMPATIBLE_TOOL_CHECKER=true)")
		} else if modelInfo != nil && modelInfo.Name != "" {
			modelName := strings.ToLower(modelInfo.Name)

			// Qwen系列模型
			if strings.Contains(modelName, "qwen") {
				needsCompatibleChecker = true
				logs.CtxInfof(ctx, "[AgentBuilder] Detected Qwen model '%s', using compatible tool call checker", modelInfo.Name)
			}
			// Claude系列模型
			if strings.Contains(modelName, "claude") {
				needsCompatibleChecker = true
				logs.CtxInfof(ctx, "[AgentBuilder] Detected Claude model '%s', using compatible tool call checker", modelInfo.Name)
			}
			// 通义千问系列（阿里的模型）
			if strings.Contains(modelName, "tongyi") || strings.Contains(modelName, "qianwen") {
				needsCompatibleChecker = true
				logs.CtxInfof(ctx, "[AgentBuilder] Detected Tongyi/Qianwen model '%s', using compatible tool call checker", modelInfo.Name)
			}
			// Gemini系列模型
			if strings.Contains(modelName, "gemini") {
				needsCompatibleChecker = true
				logs.CtxInfof(ctx, "[AgentBuilder] Detected Gemini model '%s', using compatible tool call checker", modelInfo.Name)
			}
		}

		// 如果需要兼容的checker，使用自定义实现
		if needsCompatibleChecker {
			reactConfig.StreamToolCallChecker = adaptiveToolCallChecker
			logs.CtxInfof(ctx, "[AgentBuilder] Using adaptive tool call checker for better compatibility")
		} else {
			logs.CtxInfof(ctx, "[AgentBuilder] Using default tool call checker")
		}

		agent, err := react.NewAgent(ctx, reactConfig)
		if err != nil {
			return nil, err
		}
		agentGraph, agentNodeOpts = agent.ExportGraph()

		agentNodeName = keyOfReActAgent
	} else {
		agentNodeName = keyOfLLM
	}

	suggestGraph, nsg := newSuggestGraph(ctx, conf, chatModel)

	g := compose.NewGraph[*AgentRequest, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) (state *AgentState) {
			return &AgentState{}
		}))

	_ = g.AddLambdaNode(keyOfPersonRender,
		compose.InvokableLambda[*AgentRequest, string](personaVars.RenderPersona),
		compose.WithStatePreHandler(func(ctx context.Context, ar *AgentRequest, state *AgentState) (*AgentRequest, error) {
			state.UserInput = ar.Input
			return ar, nil
		}),
		compose.WithOutputKey(placeholderOfPersona))

	_ = g.AddLambdaNode(keyOfBoundCardsRender,
		compose.InvokableLambda[*AgentRequest, string](boundCardsRenderer.RenderBoundCards),
		compose.WithOutputKey(placeholderOfBoundCards))

	_ = g.AddLambdaNode(keyOfSkillsRender,
		compose.InvokableLambda[*AgentRequest, string](skillsRenderer.RenderSkills),
		compose.WithOutputKey(placeholderOfAvailableSkills))

	_ = g.AddLambdaNode(keyOfPromptVariables,
		compose.InvokableLambda[*AgentRequest, map[string]any](promptVars.AssemblePromptVariables))

	_ = g.AddLambdaNode(keyOfKnowledgeRetriever,
		compose.InvokableLambda[*AgentRequest, []*schema.Document](kr.Retrieve),
		compose.WithNodeName(keyOfKnowledgeRetriever))

	_ = g.AddLambdaNode(keyOfToolsPreRetriever,
		compose.InvokableLambda[*AgentRequest, []*schema.Message](tr.toolPreRetrieve),
		compose.WithOutputKey(keyOfToolsPreRetriever),
		compose.WithNodeName(keyOfToolsPreRetriever),
	)
	_ = g.AddLambdaNode(keyOfKnowledgeRetrieverPack,
		compose.InvokableLambda[[]*schema.Document, string](kr.PackRetrieveResultInfo),
		compose.WithOutputKey(placeholderOfKnowledge),
	)
	_ = g.AddChatTemplateNode(keyOfPromptTemplate, chatPrompt)

	agentNodeOpts = append(agentNodeOpts, compose.WithNodeName(agentNodeName))

	if isReActAgent {
		_ = g.AddGraphNode(agentNodeName, agentGraph, agentNodeOpts...)
	} else {
		_ = g.AddChatModelNode(agentNodeName, chatModel, agentNodeOpts...)
	}

	if nsg {
		_ = g.AddLambdaNode(keyOfSuggestPreInputParse, compose.ToList[*schema.Message](),
			compose.WithStatePostHandler(func(ctx context.Context, out []*schema.Message, state *AgentState) ([]*schema.Message, error) {
				out = append(out, state.UserInput)
				return out, nil
			}),
		)
		_ = g.AddGraphNode(keyOfSuggestGraph, suggestGraph)
	}

	_ = g.AddEdge(compose.START, keyOfPersonRender)
	_ = g.AddEdge(compose.START, keyOfBoundCardsRender)
	_ = g.AddEdge(compose.START, keyOfSkillsRender)
	_ = g.AddEdge(compose.START, keyOfPromptVariables)
	_ = g.AddEdge(compose.START, keyOfKnowledgeRetriever)
	_ = g.AddEdge(compose.START, keyOfToolsPreRetriever)

	_ = g.AddEdge(keyOfPersonRender, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfBoundCardsRender, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfSkillsRender, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfPromptVariables, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfKnowledgeRetriever, keyOfKnowledgeRetrieverPack)
	_ = g.AddEdge(keyOfKnowledgeRetrieverPack, keyOfPromptTemplate)
	_ = g.AddEdge(keyOfToolsPreRetriever, keyOfPromptTemplate)

	_ = g.AddEdge(keyOfPromptTemplate, agentNodeName)

	if nsg {
		_ = g.AddEdge(agentNodeName, keyOfSuggestPreInputParse)
		_ = g.AddEdge(keyOfSuggestPreInputParse, keyOfSuggestGraph)
		_ = g.AddEdge(keyOfSuggestGraph, compose.END)
	} else {
		_ = g.AddEdge(agentNodeName, compose.END)
	}

	var opts []compose.GraphCompileOption
	if requireCheckpoint {
		opts = append(opts, compose.WithCheckPointStore(conf.CPStore))
	}
	opts = append(opts, compose.WithNodeTriggerMode(compose.AllPredecessor))
	runner, err := g.Compile(ctx, opts...)
	if err != nil {
		return nil, err
	}

	ar := &AgentRunner{
		runner:              runner,
		requireCheckpoint:   requireCheckpoint,
		modelInfo:           modelInfo,
		containWfTool:       containWfTool,
		returnDirectlyTools: returnDirectlyTools,
		superAgent:          isSuperAgent(conf),
		sandboxKey:          sandboxKey,
		chatModel:           chatModel,
	}

	// 实验性 DeepAgents 引擎（AGENT_ENGINE=deepagents）：构建成功则挂上，StreamExecute 会优先走它；
	// 失败或未开启则保持 nil → 走默认 ReAct，零影响。
	if isReActAgent && deepAgentsEnabled() {
		if da, derr := buildDeepAgent(ctx, conf, chatModel, agentTools); derr != nil {
			logs.CtxWarnf(ctx, "[BuildAgent] build deep agent failed, fallback to ReAct: %v", derr)
		} else {
			ar.deepAgent = da
			ar.cpStore = conf.CPStore
			logs.CtxInfof(ctx, "[BuildAgent] DeepAgents engine ENABLED for this agent")
		}
	}

	return ar, nil
}

// isInstanceAgent reports whether conf describes a virtual-employee INSTANCE,
// i.e. an agent whose config was materialized from a published product template
// (SourceProductID != 0). Instance sandboxes are cold-started from the product
// template and have /skills mounted read-only.
func isInstanceAgent(conf *Config) bool {
	if conf == nil || conf.Agent == nil || conf.Agent.SingleAgent == nil {
		return false
	}
	return conf.Agent.SourceProductID != 0
}

// instanceTemplateObjectKey computes the object-store key of the product template
// archive that an instance should restore on cold-start:
//
//	"templates/agent_app/{productID}/{version}.tgz"
//
// Returns "" when productID is 0 (not an instance).
func instanceTemplateObjectKey(productID int64, version string) string {
	if productID == 0 {
		return ""
	}
	return fmt.Sprintf("templates/agent_app/%d/%s.tgz", productID, version)
}

func extractJinja2Placeholder(persona string) (variableNames []string) {
	re := regexp.MustCompile(`{{([^}]*)}}`)
	matches := re.FindAllStringSubmatch(persona, -1)
	variables := make([]string, 0, len(matches))
	for _, match := range matches {
		val := strings.TrimSpace(match[1])
		if val != "" {
			variables = append(variables, match[1])
		}
	}
	return variables
}
