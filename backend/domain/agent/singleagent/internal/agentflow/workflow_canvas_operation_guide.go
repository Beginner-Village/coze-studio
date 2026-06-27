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
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type wfCanvasGetOperationGuideTool struct{}

type wfCanvasOperationGuideArgs struct {
	TaskType string `json:"task_type,omitempty"`
}

type wfCanvasOperationGuideStep struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Tools       []string `json:"tools"`
	Description string   `json:"description"`
}

type wfCanvasOperationGuideChecklist struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Items []string `json:"items"`
}

func (t *wfCanvasGetOperationGuideTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_operation_guide",
		Desc: "获取工作流画布渐进式操作规程:先读节点/资源/上下文/可绑定变量,再加节点/连线/配置/布局/试运行/局部修复。" +
			"工作流构建、修改、调试前先调用它,不要只靠提示词或记忆猜工具顺序。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_type": {Type: schema.String, Desc: "可选,如 build_workflow、fix_failure、node_smoke", Required: false},
		}),
	}, nil
}

func (t *wfCanvasGetOperationGuideTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args wfCanvasOperationGuideArgs
	if strings.TrimSpace(argumentsInJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			return argParseErrMsg(err), nil
		}
	}
	taskType := strings.TrimSpace(args.TaskType)
	if taskType == "" {
		taskType = "build_workflow"
	}
	b, _ := json.Marshal(map[string]any{
		"status":         "operation_guide",
		"task_type":      taskType,
		"required_tools": wfCanvasOperationGuideRequiredTools(),
		"steps":          wfCanvasOperationGuideSteps(),
		"guardrails":     wfCanvasOperationGuideGuardrails(),
		"checklists":     wfCanvasOperationGuideChecklists(),
		"recipes":        wfCanvasOperationGuideRecipes(),
		"instruction": "先参考 recipes 选一个匹配的工作流范式作为骨架(尤其对话工作流的槽位填充范式),再按 steps 顺序完成渐进式设计和上下文读取,再修改画布。" +
			" 配置任何变量引用前必须调用 workflow_canvas_get_bindable_variables。" +
			" 每组 add/connect/configure/delete 后调用 workflow_canvas_auto_layout 和 workflow_canvas_get_canvas_context 审计。" +
			" 对话工作流(chatflow)里插件/API/数据库节点的结构化入参,先加 type=3 大模型节点从用户对话抽取参数再绑定,不要写死字面量。" +
			" 失败时优先局部修复对应节点,不要默认 clear_canvas。",
	})
	return string(b), nil
}

func wfCanvasOperationGuideRequiredTools() []string {
	return []string{
		"workflow_canvas_get_operation_guide",
		"workflow_canvas_get_node_catalog",
		"workflow_canvas_get_node_capability_audit",
		"workflow_canvas_get_node_smoke_manifest",
		"workflow_canvas_get_node_smoke_coverage",
		"workflow_canvas_get_resource_catalog",
		"workflow_canvas_get_node_spec",
		"workflow_canvas_get_canvas_context",
		"workflow_canvas_get_bindable_variables",
		"workflow_canvas_add_node",
		"workflow_canvas_connect",
		"workflow_canvas_configure_node",
		"workflow_canvas_set_node_params",
		"workflow_canvas_delete_node",
		"workflow_canvas_delete_line",
		"workflow_canvas_clear_canvas",
		"workflow_canvas_auto_layout",
		"workflow_canvas_test_run",
	}
}

func wfCanvasOperationGuideSteps() []wfCanvasOperationGuideStep {
	return []wfCanvasOperationGuideStep{
		{
			ID:          "discover_nodes",
			Title:       "读取节点能力",
			Tools:       []string{"workflow_canvas_get_node_catalog", "workflow_canvas_get_node_capability_audit", "workflow_canvas_get_node_smoke_manifest", "workflow_canvas_get_node_smoke_coverage"},
			Description: "先知道有哪些节点、哪些 full/resource-bound/partial/add-only、哪些需要临时工作流或真实资源 fixture,以及每个节点是否有 spec/assertions/expected bindable variables。",
		},
		{
			ID:          "discover_resources",
			Title:       "读取资源目录",
			Tools:       []string{"workflow_canvas_get_resource_catalog"},
			Description: "涉及插件/API、知识库、智能体、模型、数据库、HTTP、MCP、卡片等资源型节点时,先确认资源族、已有 API 和禁止编造 ID 约束。",
		},
		{
			ID:          "read_node_specs",
			Title:       "读取节点规格",
			Tools:       []string{"workflow_canvas_get_node_spec"},
			Description: "对计划使用的每种 node type 读取完整规格,包括必须字段、输入绑定、outputs、End/VariableMerge/type=13 边界。",
		},
		{
			ID:          "read_canvas_context",
			Title:       "读取当前画布",
			Tools:       []string{"workflow_canvas_get_canvas_context"},
			Description: "基于本轮消息发送前的真实画布摘要、节点、连线、outputs、绑定诊断和资源摘要做增量修改。",
		},
		{
			ID:          "mutate_topology",
			Title:       "添加节点和连线",
			Tools:       []string{"workflow_canvas_add_node", "workflow_canvas_connect", "workflow_canvas_delete_node", "workflow_canvas_delete_line"},
			Description: "先完成拓扑,Start/End 只能引用不能新增或删除;重构时优先删除明确目标节点/连线,不要默认 clear_canvas。",
		},
		{
			ID:          "read_bindable_variables",
			Title:       "读取可绑定变量",
			Tools:       []string{"workflow_canvas_get_bindable_variables"},
			Description: "配置 input/inputs、IF condition、变量聚合、End returns/返回文本、prompt 模板、代码/API 参数前必须先读取真实可绑定变量。",
		},
		{
			ID:          "configure_nodes",
			Title:       "逐节点配置",
			Tools:       []string{"workflow_canvas_configure_node", "workflow_canvas_set_node_params"},
			Description: "基于可绑定变量配置每个节点的 input/inputs、prompt、code、condition、merge_groups、outputs、returns、content 或资源参数。",
		},
		{
			ID:          "layout",
			Title:       "优化布局",
			Tools:       []string{"workflow_canvas_auto_layout"},
			Description: "每组 add/connect/configure/delete 后调用,避免节点堆叠。",
		},
		{
			ID:          "validate_and_repair",
			Title:       "试运行和局部修复",
			Tools:       []string{"workflow_canvas_test_run", "workflow_canvas_get_canvas_context", "workflow_canvas_get_bindable_variables", "workflow_canvas_configure_node"},
			Description: "试运行或校验失败时读取最新上下文和失败节点,只修复对应节点绑定/输出/条件/聚合/End 返回,再重跑。",
		},
	}
}

func wfCanvasOperationGuideGuardrails() []string {
	return []string{
		"dispatched_to_canvas 不是成功、不是配置正确、不是测试通过;确认完成前必须读取上下文或拿到真实试运行结果。",
		"不要默认 clear_canvas;只有用户明确要求清空/重建,或上下文证明局部修复不可行时才调用 workflow_canvas_clear_canvas。",
		"一个工作流只能有一个 Start 和一个 End;不能用 workflow_canvas_add_node 新增 type=1/type=2。",
		"type=13 输出/纯输出是 display-only,不要作为变量聚合或 End returns 的真实变量来源;需要下游消费时用 type=15 文本处理产出 output:string。",
		"配置任何变量引用前必须先调用 workflow_canvas_get_bindable_variables,只能绑定返回列表里的真实变量或本轮刚声明 outputs 的变量。",
		"资源型节点必须先调用 workflow_canvas_get_resource_catalog,资源 ID、模型、插件/API、知识库、智能体、数据库、HTTP、MCP 和卡片不能编造。",
		"每组 add/connect/configure/delete 后调用 workflow_canvas_auto_layout,再调用 workflow_canvas_get_canvas_context 做从 Start 到 End 的绑定审计。",
		"对话工作流(chatflow)Start 通常只有用户自然语言输入(USER_INPUT/对话文本),没有结构化字段;插件/API/代码/数据库节点需要的金额、账号、类型、日期等参数不要写死字面量,先加 type=3 大模型节点从对话抽取(outputs 按下游参数逐个声明、类型与接口一致),再把下游入参绑定到大模型输出。",
	}
}

func wfCanvasOperationGuideChecklists() []wfCanvasOperationGuideChecklist {
	return []wfCanvasOperationGuideChecklist{
		{
			ID:    "binding_audit",
			Title: "从输入到输出绑定审计",
			Items: []string{
				"Start 输入或 type=30 输入字段已经声明,下游引用字段名称一致。",
				"每个节点 input/inputs 只绑定真实可用变量;固定文本节点可删除不需要的 input,但若下游消费必须声明 output:string。",
				"每个被下游引用的节点都显式声明 outputs,字段名和类型与后续引用一致。",
				"IF true/false 分支都有连线,condition.left 来自真实变量,right 是明确字面量或真实变量。",
				"变量聚合 merge_groups 只聚合真实上游变量,不要聚合 type=13 输出节点。",
				"End 返回变量时 returns 绑定真实变量;End 返回文本时先配置 input/inputs,再在 content 中引用 {{变量名}},需要流式透传时配置 streaming_output=true。",
			},
		},
		{
			ID:    "failure_repair",
			Title: "失败后局部修复",
			Items: []string{
				"先调用 workflow_canvas_get_canvas_context 读取最新节点、连线、outputs 和当前校验错误。",
				"对失败节点或下游消费节点调用 workflow_canvas_get_bindable_variables,确认真实可绑定变量。",
				"针对 BlockID empty、引用变量不存在、变量值为空、Python Args 规范错误,优先修复 input/inputs、outputs、condition、merge_groups、returns、content 或 code。",
				"不要默认 clear_canvas;只对对应节点做局部修复。",
				"修复后调用 workflow_canvas_auto_layout 和 workflow_canvas_test_run,不要把工具 ack 当成最终成功。",
			},
		},
		{
			ID:    "chatflow_param_extraction",
			Title: "对话工作流大模型提参",
			Items: []string{
				"判断是否对话工作流: get_canvas_context/get_bindable_variables 里 Start 只暴露 USER_INPUT/对话文本而没有结构化字段时即为 chatflow。",
				"插件/API/代码/数据库节点需要的每个结构化参数(金额、账号、类型、日期等)都要有真实来源,不能写死测试字面量,也不要把整段用户输入塞进单个参数。",
				"先 workflow_canvas_add_node 一个 type=3 大模型节点,prompt 要求从用户对话抽取并只输出 JSON,outputs 按下游接口参数逐个声明,类型与 list_plugins 的参数 schema 一致(数字 number、文本 string)。",
				"把大模型节点接在 Start 与插件节点之间,插件节点 inputs 绑定 {from:'大模型node_tag',output:'参数名'}。",
				"试运行用一句包含所需信息的自然语言(例如'给账号12345转100元'),让大模型抽取参数,而不是逐个手填插件入参。",
			},
		},
	}
}

type wfCanvasOperationGuideRecipe struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	When  string   `json:"when"`
	Flow  []string `json:"flow"`
	Tips  []string `json:"tips,omitempty"`
}

// wfCanvasOperationGuideRecipes 是常见工作流范式的"经验骨架",让智能体面对一个需求时
// 心里先有套路,而不是从零乱拼节点。重点是对话工作流的槽位填充范式。
func wfCanvasOperationGuideRecipes() []wfCanvasOperationGuideRecipe {
	return []wfCanvasOperationGuideRecipe{
		{
			ID:    "slot_filling_chatflow",
			Title: "槽位填充对话工作流(参数不全就反问补齐,齐了才调插件)",
			When:  "对话里要调用一个需要多个参数的插件/接口,但用户不一定一次说全。这是对话工作流最常见的范式。",
			Flow: []string{
				"Start: 用户自然语言输入(USER_INPUT)",
				"type=3 大模型(提参+查缺): 从对话提取目标接口需要的参数,outputs 按接口参数逐个声明(类型与接口一致),并额外声明 missing:string 列出仍缺失的必填参数(都齐全则为空字符串);建议 enable_chat_history=true 支持多轮补齐",
				"type=8 条件IF: 判断 missing 是否为空(left=大模型.missing, operator=为空/null)。true=参数齐全分支, false=有缺失分支",
				"齐全分支(true): type=4 插件节点,inputs 绑定大模型提取的各参数; 或用 type=3 大模型 bind_plugins 让模型自己按需调",
				"缺失分支(false): type=18 问答节点,question 用 {{missing}} 反问用户补齐缺失参数(例如'还需要您提供: {{missing}}')",
				"汇合: 两分支末端各用 type=15 文本处理产出 output:string, 再 type=32 变量聚合成一个 output, End 返回该 output",
			},
			Tips: []string{
				"判断参数是否齐全,优先让大模型在提参时直接输出 missing 字段,IF 只判断 missing 是否为空——比给每个参数单独 IF 判断简单可靠得多。",
				"大模型节点开 enable_chat_history=true 后,用户补齐参数时模型能结合上一轮上下文继续,实现多轮补全。",
				"齐全分支可用独立 type=4 插件节点(确定性调用)或大模型 bind_plugins(模型自主调用)——前者更可控,后者更灵活。",
			},
		},
		{
			ID:    "intent_routing",
			Title: "意图路由(先分类再分流处理)",
			When:  "用户意图有多种(如查询/办理/投诉),需要按意图走不同处理分支。",
			Flow: []string{
				"Start → type=3 大模型(意图分类: prompt 要求只输出一个类别词, outputs=[{name:category,type:string}]) → type=8 条件IF 按 category 分流 → 各分支处理 → type=32 变量聚合 → End",
			},
			Tips: []string{"用大模型节点本身做分类即可,通常无需单独的 type=22 意图节点(它当前语义配置未完整)。"},
		},
		{
			ID:    "llm_fc_agent",
			Title: "节点内智能体(大模型 FC 绑插件/工作流, 自主多轮调用)",
			When:  "希望大模型按需、可多轮地调用一个或多个工具并整合结果,而不是写死的固定管线。",
			Flow: []string{
				"Start → type=3 大模型(bind_plugins/bind_workflows 绑工具 + system_prompt 说明何时调用哪个工具 + enable_chat_history 多轮) → End",
			},
			Tips: []string{"绑了工具的大模型节点会进入 ReAct 模式自动调用工具; plugin_id/api_id 来自 workflow_canvas_list_plugins,禁止编造。"},
		},
		{
			ID:    "rag_qa",
			Title: "知识库问答(RAG)",
			When:  "要基于资料库内容回答用户问题。",
			Flow: []string{
				"Start → type=6 知识库检索(dataset_ids 必须真实) → type=3 大模型(prompt 基于检索到的资料回答) → End",
			},
		},
	}
}
