package canvasautomation

type ResourceCatalogEntry struct {
	Family                  string   `json:"family"`
	Name                    string   `json:"name"`
	Status                  string   `json:"status"`
	AppliesToNodeTypes      []string `json:"applies_to_node_types"`
	DiscoveryTool           string   `json:"discovery_tool"`
	BackendAPIs             []string `json:"backend_apis,omitempty"`
	RequiredBeforeConfigure bool     `json:"required_before_configure"`
	DoNotFabricateIDs       bool     `json:"do_not_fabricate_ids"`
	BindingGuidance         []string `json:"binding_guidance,omitempty"`
	Gaps                    []string `json:"gaps,omitempty"`
}

var resourceCatalogEntries = []ResourceCatalogEntry{
	{
		Family:                  "plugin_api",
		Name:                    "插件/API",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"3", "4"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"GetPlaygroundPluginList"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置 type=4 插件/API 节点前必须读取真实 plugin_id/api_id/tool schema。",
			"API 参数按 schema 逐项绑定 workflow.get_bindable_variables 返回的变量或明确字面量。",
			"LLM 技能/工具同样只能选择真实插件/API,不能编造工具名。",
		},
		Gaps: []string{"MCP 尚未返回 live 插件/API 列表;当前只暴露发现契约和已有后端 API 名称。"},
	},
	{
		Family:                  "knowledge_base",
		Name:                    "知识库",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"3", "6", "27"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListKnowledgeDetail"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置知识库检索/写入前必须确认 dataset_id 或 knowledge_id 来自当前 space。",
			"query/input 仍必须从真实可绑定变量或明确字面量配置。",
		},
		Gaps: []string{"MCP 尚未返回 live 知识库列表和字段 schema。"},
	},
	{
		Family:                  "agent",
		Name:                    "智能体",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"3", "100"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListDraftBots", "GetBotInfo"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置 type=100 智能体或 LLM 内部智能体前必须选择真实 bot_id/agent_id。",
			"动态参数必须按智能体输入 schema 和当前可绑定变量配置。",
		},
		Gaps: []string{"MCP 尚未统一返回空间智能体列表和动态参数 schema。"},
	},
	{
		Family:                  "model",
		Name:                    "模型",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"3", "22"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"GetModel"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置 LLM/意图识别前必须选择当前空间允许的 model_id/model_type。",
			"模型只解决推理能力;输出字段仍需在节点 outputs 中显式声明。",
		},
		Gaps: []string{"MCP 尚未返回当前空间可用模型与默认模型策略。"},
	},
	{
		Family:                  "workflow",
		Name:                    "子工作流",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"9"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"OpenAPIGetWorkflowInfo"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置子工作流节点前必须确认 workflow_id、版本和输入输出 schema。",
			"父工作流绑定必须对齐子工作流入参类型。",
		},
		Gaps: []string{"MCP 尚未返回可作为子流程引用的工作流列表和 schema。"},
	},
	{
		Family:                  "database",
		Name:                    "数据库",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"12", "42", "43", "44", "46"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListDatabaseTables", "GetDatabaseTableSchema"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置 SQL/数据库 CRUD 前必须知道表、字段、主键、条件字段和权限。",
			"where/body/update 字段只能绑定真实变量或明确字面量。",
		},
		Gaps: []string{"MCP 尚未封装数据库表和字段 schema 发现。"},
	},
	{
		Family:                  "card",
		Name:                    "卡片",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"99"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListCardTemplates"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置卡片选择前必须确认卡片模板、选项 schema 和返回字段。",
			"卡片选择输出需要声明后续会消费的变量。",
		},
		Gaps: []string{"MCP 尚未封装卡片模板列表和交互 schema。"},
	},
	{
		Family:                  "mcp_server",
		Name:                    "MCP 服务",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"61"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListMCPServers", "ListMCPTools"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置 MCP 节点前必须确认 server_id、tool_name、input schema 和 auth 方式。",
			"工具参数仍按 schema 绑定当前可绑定变量或明确字面量。",
		},
		Gaps: []string{"MCP 尚未封装 live MCP server/tool 列表。"},
	},
	{
		Family:                  "http_endpoint",
		Name:                    "HTTP 端点",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"45"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListHTTPConnectors", "GetHTTPRequestSchema"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置 HTTP 请求节点前必须确认 method、url、headers、query/body schema 和鉴权来源。",
			"请求参数和 body 字段只能绑定真实变量或明确字面量。",
			"响应 outputs 必须声明下游要消费的字段。",
		},
		Gaps: []string{"MCP 尚未封装 HTTP connector/endpoint schema 发现。"},
	},
	{
		Family:                  "image_asset",
		Name:                    "图像/媒体资源",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"14", "16", "17", "23"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListImageAssets", "ListImageWorkflows"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置图像类节点前必须确认图片资源、图像工作流或生成模型能力。",
			"图片输入输出字段必须按节点规格和真实资源 schema 配置。",
		},
		Gaps: []string{"MCP 尚未封装图像资源和图像工作流发现。"},
	},
	{
		Family:                  "trigger",
		Name:                    "触发器",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"34", "35", "36"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"ListTriggerSchemas"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置触发器节点前必须确认触发器类型、schema 和权限。",
			"触发器参数必须绑定真实变量或明确字面量。",
		},
		Gaps: []string{"MCP 尚未封装触发器 schema 发现。"},
	},
	{
		Family:                  "memory",
		Name:                    "记忆/会话资源",
		Status:                  "contract-ready",
		AppliesToNodeTypes:      []string{"26"},
		DiscoveryTool:           "workflow.list_resource_catalog",
		BackendAPIs:             []string{"GetChatFlowRole", "ListMemoryScopes"},
		RequiredBeforeConfigure: true,
		DoNotFabricateIDs:       true,
		BindingGuidance: []string{
			"配置长期记忆前必须确认记忆作用域、读写权限和会话上下文来源。",
			"记忆输入必须来自真实可绑定变量。",
		},
		Gaps: []string{"MCP 尚未封装记忆资源和作用域发现。"},
	},
}

func ListResourceCatalog(filter ResourceCatalogFilter) []ResourceCatalogEntry {
	out := make([]ResourceCatalogEntry, 0, len(resourceCatalogEntries))
	for _, entry := range resourceCatalogEntries {
		if filter.Family != "" && entry.Family != filter.Family {
			continue
		}
		if filter.NodeType != "" && !contains(entry.AppliesToNodeTypes, filter.NodeType) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

type ResourceCatalogFilter struct {
	Family   string
	NodeType string
}

func ResourceCatalogUsageOrder() []string {
	return []string{
		"workflow.get_operation_guide",
		"workflow.list_surfaces",
		"workflow.list_resource_catalog",
		"workflow.get_node_spec",
		"workflow.get_canvas_context",
		"workflow.get_bindable_variables",
		"workflow.configure_node",
		"workflow.test_run",
	}
}

func ResourceCatalogCoverageGaps() []string {
	return []string{
		"当前返回资源族和发现契约,尚未返回 live resource instances。",
		"资源类节点配置仍必须等后续 live list 工具提供真实 ID/schema 后才能宣称完整支持。",
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
