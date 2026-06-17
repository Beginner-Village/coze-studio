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
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow"
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

	persona := conf.Agent.Prompt.GetPrompt()
	if isSuperAgent(conf) {
		persona = persona + "\n\n" + SuperAgentExtraPrompt
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
	skillsRenderer := newSkillsRender(conf.Agent.SkillInfoList)

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

	chatModel, err := newChatModel(ctx, &config{
		modelFactory:      conf.ModelFactory,
		modelInfo:         modelInfo,
		agentModelSetting: conf.Agent.ModelInfo,
	})
	if err != nil {
		return nil, err
	}

	requireCheckpoint := false
	pluginTools, err := newPluginTools(ctx, &toolConfig{
		spaceID:       conf.Agent.SpaceID,
		userID:        conf.UserID,
		agentIdentity: conf.Identity,
		toolConf:      conf.Agent.Plugin,
	})
	if err != nil {
		return nil, err
	}
	tr := newPreToolRetriever(&toolPreCallConf{})

	wfTools, returnDirectlyTools, err := newWorkflowTools(ctx, &workflowConfig{
		wfInfos: conf.Agent.Workflow,
	})
	if err != nil {
		return nil, err
	}

	var dbTools []tool.InvokableTool
	if len(conf.Agent.Database) > 0 {
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

	var avTools []tool.InvokableTool
	// 检查记忆工具配置开关
	// 如果配置为启用(默认)或未配置，则添加记忆工具
	// 如果配置为禁用，则不添加记忆工具
	memoryToolEnabled := true // 默认启用，保持向后兼容
	if conf.Agent.MemoryToolConfig != nil {
		memoryToolEnabled = conf.Agent.MemoryToolConfig.Mode == nil || *conf.Agent.MemoryToolConfig.Mode == 1
	}

	if memoryToolEnabled {
		avTools, err = newAgentVariableTools(ctx, avConf)
		if err != nil {
			return nil, err
		}
	}

	// 添加外部知识库工具（如果配置了dataset_ids）
	var externalKnowledgeTools []tool.InvokableTool
	if conf.Agent.ExternalKnowledge != nil && len(conf.Agent.ExternalKnowledge.DatasetIds) > 0 {
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
	agentTools := make([]tool.BaseTool, 0, len(pluginTools)+len(wfTools)+len(dbTools)+len(avTools)+len(externalKnowledgeTools))
	agentTools = append(agentTools, slices.Transform(pluginTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)
	agentTools = append(agentTools, slices.Transform(wfTools, func(a workflow.ToolFromWorkflow) tool.BaseTool { return a.(tool.BaseTool) })...)
	agentTools = append(agentTools, slices.Transform(dbTools, func(a tool.InvokableTool) tool.BaseTool {
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

	// 添加技能工具 (read_skill)，并把 sandboxKey 传入以便 L3 脚本注入。
	skillTools := newSkillTools(conf.Agent.SpaceID, sandboxKey, conf.Agent.SkillInfoList)
	agentTools = append(agentTools, slices.Transform(skillTools, func(a tool.InvokableTool) tool.BaseTool {
		return a
	})...)

	// 添加沙箱工具 (run_bash/read_file/write_file/list_files)
	if sandboxToolsEnabled(len(conf.Agent.SkillInfoList)) {
		sandboxTools := newSandboxTools(sandboxKey)
		agentTools = append(agentTools, slices.Transform(sandboxTools, func(a tool.InvokableTool) tool.BaseTool {
			return a
		})...)
		if len(sandboxTools) > 0 {
			logs.CtxInfof(ctx, "[BuildAgent] Mounted %d sandbox tools (key=%s)", len(sandboxTools), sandboxKey)
		}
	}

	// 自动绑定技能提示词中引用的工作流
	existingWorkflowIDs := make(map[int64]struct{}, len(conf.Agent.Workflow))
	for _, wf := range conf.Agent.Workflow {
		existingWorkflowIDs[wf.GetWorkflowId()] = struct{}{}
	}
	skillRes, skillResErr := resolveSkillResources(ctx, conf.Agent.SkillInfoList, existingWorkflowIDs)
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
	if isSuperAgent(conf) {
		if dt, derr := newDeepTaskTool(ctx, chatModel, append([]tool.BaseTool(nil), agentTools...)); derr != nil {
			logs.CtxWarnf(ctx, "[BuildAgent] build deep_task tool failed: %v", derr)
		} else if dt != nil {
			agentTools = append(agentTools, dt)
			logs.CtxInfof(ctx, "[BuildAgent] mounted deep_task tool for super agent")
		}
	}

	// 超级 agent 可插拔扩展工具（P5）：只挂给超级 agent，普通 agent 不受影响。
	if isSuperAgent(conf) {
		extTools := newSuperAgentExtensionTools(sandboxKey)
		for _, et := range extTools {
			agentTools = append(agentTools, et)
		}
		if len(extTools) > 0 {
			logs.CtxInfof(ctx, "[BuildAgent] mounted %d super-agent extension tools", len(extTools))
		}
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
			// 最大步数：默认 30(约15轮工具往返)，可经 AGENT_MAX_STEP 环境变量按需放宽以支持更复杂任务
			MaxStep: agentMaxStep(),
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
