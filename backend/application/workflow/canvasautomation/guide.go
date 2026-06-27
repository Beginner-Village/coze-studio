package canvasautomation

import "strings"

type OperationGuide struct {
	TaskType      string                    `json:"task_type"`
	RequiredTools []string                  `json:"required_tools"`
	Steps         []OperationGuideStep      `json:"steps"`
	Guardrails    []string                  `json:"guardrails"`
	Checklists    []OperationGuideChecklist `json:"checklists"`
}

type OperationGuideStep struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Tools       []string `json:"tools"`
	Description string   `json:"description"`
}

type OperationGuideChecklist struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Items []string `json:"items"`
}

func GetOperationGuide(taskType string) OperationGuide {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" {
		taskType = "build_workflow"
	}

	guide := OperationGuide{
		TaskType:      taskType,
		RequiredTools: operationGuideRequiredTools(),
		Steps: []OperationGuideStep{
			{
				ID:          "surface_discovery",
				Title:       "确认可编辑界面",
				Tools:       []string{"workflow.list_surfaces"},
				Description: "先确认当前任务操作的是 workflow.canvas、chatflow.canvas 还是设置面。只有 canvas_automation.v0 surface 才能走画布 add/connect/configure。",
			},
			{
				ID:          "resource_discovery",
				Title:       "发现真实资源",
				Tools:       []string{"workflow.list_resource_catalog"},
				Description: "涉及插件/API、知识库、智能体、模型、数据库、卡片、MCP、HTTP 等资源类节点时,先读资源目录和后续 live resource 工具,禁止编造 ID。",
			},
			{
				ID:          "node_coverage",
				Title:       "读取节点覆盖状态",
				Tools:       []string{"workflow.list_node_capabilities", "workflow.node_smoke_coverage"},
				Description: "按需求选择节点前,先读取 41 个节点的能力和 smoke 覆盖状态,确认目标节点是可临时执行、资源 fixture、partial/sub-canvas 还是只读。",
			},
			{
				ID:          "node_planning",
				Title:       "选择节点并读取规格",
				Tools:       []string{"workflow.get_node_spec"},
				Description: "基于覆盖状态设计节点链路,再逐个读取目标 node type 的完整规格。Start/End 是内置单例,只能连接和配置,不能新增。",
			},
			{
				ID:          "canvas_context",
				Title:       "读取当前画布",
				Tools:       []string{"workflow.get_canvas_context"},
				Description: "改动前读取真实画布节点、连线、输出和校验诊断,避免基于旧快照或文字总结做修改。",
			},
			{
				ID:          "mutate_topology",
				Title:       "添加节点和连线",
				Tools:       []string{"workflow.add_node", "workflow.connect", "workflow.delete_node", "workflow.delete_line"},
				Description: "先完成拓扑,普通重构时只删除明确目标节点或连线。除非用户明确要求重建,不要默认 clear_canvas。",
			},
			{
				ID:          "read_bindable_variables",
				Title:       "读取可绑定变量",
				Tools:       []string{"workflow.get_bindable_variables"},
				Description: "配置 IF、变量聚合、End、Prompt、代码输入、HTTP/API 参数前,必须读取目标节点可用的真实可绑定变量。",
			},
			{
				ID:          "configure_nodes",
				Title:       "逐节点配置",
				Tools:       []string{"workflow.configure_node", "workflow.set_node_params"},
				Description: "基于刚读取到的可绑定变量,逐节点配置 input/inputs、outputs、returns、condition、merge_groups、content、prompt、code 或资源参数。",
			},
			{
				ID:          "layout",
				Title:       "优化布局",
				Tools:       []string{"workflow.auto_layout"},
				Description: "每组 add/connect/configure 后调用 workflow.auto_layout,让画布实时可读。",
			},
			{
				ID:          "validate_and_repair",
				Title:       "试运行和局部修复",
				Tools:       []string{"workflow.test_run", "workflow.explain_failure", "workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.configure_node"},
				Description: "试运行失败时读取失败节点和诊断,调用 explain_failure 获取修复方向,再局部修复对应节点绑定或配置。",
			},
		},
		Guardrails: []string{
			"dispatched_to_canvas 不是成功、不是配置正确、不是测试通过;每次变更后必须读取上下文或等待 browser_live 结果。",
			"不要默认 clear_canvas;只有用户明确要求重建,或已证明局部修复不可行时才清空普通节点和连线。",
			"一个工作流只能有一个 Start/End;不能新增 Start/End,只能连接和配置内置单例。",
			"type=13 输出/纯输出是 display-only,不要作为变量聚合或 End returns 的来源;需要真实变量时用 type=15 文本处理产出 output:string。",
			"配置任何变量引用前必须先调用 workflow.get_bindable_variables,只能绑定返回结果里的真实变量或明确字面量。",
			"资源类节点必须先走 workflow.list_resource_catalog 和 live resource discovery,不能编造 plugin_id、knowledge_id、bot_id、model_id、api_id、mcp tool。",
			"每组 add/connect/configure 后调用 workflow.auto_layout,避免节点堆叠。",
			"失败后优先定位失败节点并局部修复 input/inputs、outputs、condition、merge_groups、returns 或 End 返回文本,不要用清空画布掩盖问题。",
		},
		Checklists: []OperationGuideChecklist{
			{
				ID:    "binding_audit",
				Title: "从输入到输出绑定审计",
				Items: []string{
					"Start 输入或 type=30 输入节点字段已经声明,下游引用字段名称一致。",
					"每个节点 input/inputs 只绑定真实可用变量;固定文本节点可删除不需要的输入,但必须声明真实 outputs。",
					"每个会被下游引用的节点都显式声明 outputs,字段名和类型与后续引用一致。",
					"IF true/false 分支都有连线,condition.left 来自真实变量,right 是明确字面量或真实变量。",
					"变量聚合 merge_groups 只聚合真实上游变量,同一业务输出的所有分支放在同一个组里。",
					"End 返回变量时 returns 绑定真实变量;End 返回文本时先配置 input/inputs,再在 content 中引用 {{变量名}},需要流式透传时开启 streaming_output。",
				},
			},
			{
				ID:    "failure_repair",
				Title: "失败后局部修复",
				Items: []string{
					"先调用 workflow.get_canvas_context 读取最新节点、连线、输出和当前校验错误。",
					"对失败节点或下游消费节点调用 workflow.get_bindable_variables,确认真实可绑定变量。",
					"调用 workflow.explain_failure 解释错误,尤其是 BlockID empty、引用变量不存在、变量值为空、Python Args 规范错误。",
					"只对对应节点做局部修复,优先改 input/inputs、outputs、condition、merge_groups、returns、content 或 code。",
					"修复后调用 workflow.auto_layout 和 workflow.test_run,不要把工具 ack 当成最终成功。",
				},
			},
		},
	}
	return guide
}

func operationGuideRequiredTools() []string {
	return []string{
		"workflow.list_surfaces",
		"workflow.list_resource_catalog",
		"workflow.list_node_capabilities",
		"workflow.node_smoke_coverage",
		"workflow.get_node_spec",
		"workflow.get_canvas_context",
		"workflow.get_bindable_variables",
		"workflow.add_node",
		"workflow.connect",
		"workflow.configure_node",
		"workflow.set_node_params",
		"workflow.delete_node",
		"workflow.delete_line",
		"workflow.clear_canvas",
		"workflow.auto_layout",
		"workflow.test_run",
		"workflow.explain_failure",
	}
}
