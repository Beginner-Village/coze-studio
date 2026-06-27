package canvasautomation

type SurfaceCapability struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	SurfaceType     string   `json:"surface_type"`
	Status          string   `json:"status"`
	Mode            string   `json:"mode,omitempty"`
	Adapter         string   `json:"adapter"`
	CommandProtocol string   `json:"command_protocol,omitempty"`
	RequiresBrowser bool     `json:"requires_browser"`
	Description     string   `json:"description"`
	Tools           []string `json:"tools,omitempty"`
	BackendAPIs     []string `json:"backend_apis,omitempty"`
	Gaps            []string `json:"gaps,omitempty"`
}

var surfaceCapabilities = []SurfaceCapability{
	{
		ID:              "workflow.canvas",
		Name:            "工作流画布",
		SurfaceType:     "canvas",
		Status:          "implemented",
		Mode:            "workflow",
		Adapter:         "browser-live",
		CommandProtocol: "canvas_automation.v0",
		RequiresBrowser: true,
		Description:     "普通 Workflow 画布编辑面。用于读取上下文、添加/删除节点、连线、配置节点、读取可绑定变量、布局、试运行和失败解释。",
		Tools:           sharedCanvasAutomationTools(),
	},
	{
		ID:              "chatflow.canvas",
		Name:            "ChatFlow 画布",
		SurfaceType:     "canvas",
		Status:          "implemented",
		Mode:            "chatflow",
		Adapter:         "browser-live",
		CommandProtocol: "canvas_automation.v0",
		RequiresBrowser: true,
		Description:     "ChatFlow 画布复用同一套 Workflow 画布自动化协议,节点编辑、变量绑定和 smoke 计划与普通 Workflow 保持一致。",
		Tools:           sharedCanvasAutomationTools(),
		Gaps:            []string{"chatflow 专属测试运行仍需独立后端 MCP 工具。"},
	},
	{
		ID:              "chatflow.role_settings",
		Name:            "ChatFlow 角色配置",
		SurfaceType:     "settings",
		Status:          "planned-backend-api",
		Mode:            "chatflow",
		Adapter:         "backend-api",
		CommandProtocol: "surface_settings.v0",
		RequiresBrowser: false,
		Description:     "ChatFlow 右侧角色/欢迎语/建议回复/用户输入等配置面。后端已有读写 API,下一步应封装为 MCP 读写工具。",
		BackendAPIs:     []string{"GetChatFlowRole", "CreateChatFlowRole", "DeleteChatFlowRole"},
		Gaps:            []string{"缺少 MCP 工具封装字段级读写、校验和 smoke 用例。"},
	},
	{
		ID:              "chatflow.conversation_templates",
		Name:            "ChatFlow 会话模板",
		SurfaceType:     "settings",
		Status:          "planned-backend-api",
		Mode:            "chatflow",
		Adapter:         "backend-api",
		CommandProtocol: "surface_settings.v0",
		RequiresBrowser: false,
		Description:     "ChatFlow/Project 会话模板管理面。后端已有创建、更新、删除和列表 API,适合封装成无需重载前端的 MCP 工具。",
		BackendAPIs: []string{
			"ListApplicationConversationDef",
			"CreateApplicationConversationDef",
			"UpdateApplicationConversationDef",
			"DeleteApplicationConversationDef",
		},
		Gaps: []string{"缺少 MCP 工具封装 conversation template CRUD 与引用影响检查。"},
	},
	{
		ID:              "space.resource_catalog",
		Name:            "空间资源目录",
		SurfaceType:     "resource_catalog",
		Status:          "implemented-contract",
		Adapter:         "backend-api",
		CommandProtocol: "resource_catalog.v0",
		RequiresBrowser: false,
		Description:     "统一暴露空间可用插件/API、知识库、智能体、模型、卡片和子工作流,供节点配置前先查真实资源,避免编造 ID。",
		Tools:           []string{"workflow.list_resource_catalog"},
		Gaps:            []string{"缺少 MCP 工具聚合插件/API、知识库、智能体、模型、卡片、子工作流资源。"},
	},
}

func ListSurfaceCapabilities() []SurfaceCapability {
	out := make([]SurfaceCapability, len(surfaceCapabilities))
	copy(out, surfaceCapabilities)
	return out
}

func sharedCanvasAutomationTools() []string {
	return []string{
		"workflow.get_operation_guide",
		"workflow.get_canvas_context",
		"workflow.get_bindable_variables",
		"workflow.list_node_capabilities",
		"workflow.get_node_spec",
		"workflow.node_smoke_manifest",
		"workflow.node_smoke_coverage",
		"workflow.run_node_smoke",
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
