package canvasautomation

const (
	SupportSingleton         = "singleton"
	SupportFull              = "full"
	SupportResourceBound     = "resource-bound"
	SupportPartial           = "partial"
	SupportAddOnly           = "add-only"
	SupportDocumentationOnly = "documentation-only"
)

type NodeCapability struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	Registry         string   `json:"registry"`
	SupportLevel     string   `json:"support_level"`
	CanAdd           bool     `json:"can_add"`
	RequiresResource bool     `json:"requires_resource,omitempty"`
	RuntimeSmoke     string   `json:"runtime_smoke,omitempty"`
	Gaps             []string `json:"gaps,omitempty"`
}

type NodeSpec struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Commands     []string `json:"commands"`
	BindingRules []string `json:"binding_rules,omitempty"`
	Gaps         []string `json:"gaps,omitempty"`
}

type NodeSmokeManifest struct {
	Total        int                      `json:"total"`
	CoverageGaps []string                 `json:"coverage_gaps"`
	Summary      NodeSmokeManifestSummary `json:"summary"`
	Nodes        []NodeSmokeManifestNode  `json:"nodes"`
}

type NodeSmokeManifestSummary struct {
	Ready                          int `json:"ready"`
	NotReady                       int `json:"not_ready"`
	Skipped                        int `json:"skipped"`
	Execute                        int `json:"execute"`
	Readonly                       int `json:"readonly"`
	RequiresTemporaryWorkflowOptIn int `json:"requires_temporary_workflow_opt_in"`
	RequiresResourceFixture        int `json:"requires_resource_fixture"`
}

type NodeSmokeManifestNode struct {
	Type                           string   `json:"type"`
	Name                           string   `json:"name"`
	Status                         string   `json:"status"`
	RuntimeSmoke                   string   `json:"runtime_smoke"`
	Mode                           string   `json:"mode"`
	Isolation                      string   `json:"isolation"`
	Ready                          bool     `json:"ready"`
	RequiredCommands               []string `json:"required_commands,omitempty"`
	PlannedCommands                []string `json:"planned_commands,omitempty"`
	CleanupCommands                []string `json:"cleanup_commands,omitempty"`
	MissingConcreteCommands        []string `json:"missing_concrete_commands,omitempty"`
	TemporaryNodeTags              []string `json:"temporary_node_tags,omitempty"`
	RequiresTemporaryWorkflowOptIn bool     `json:"requires_temporary_workflow_opt_in,omitempty"`
	RequiresResourceFixture        bool     `json:"requires_resource_fixture,omitempty"`
	SkipReason                     string   `json:"skip_reason,omitempty"`
	Gaps                           []string `json:"gaps,omitempty"`
	Assertions                     []string `json:"assertions,omitempty"`
	ExpectedBindableVariables      []string `json:"expected_bindable_variables,omitempty"`
}

var nodeCapabilities = []NodeCapability{
	{Type: "1", Name: "开始", Registry: "start", SupportLevel: SupportSingleton, CanAdd: false, RuntimeSmoke: "not-executable", Gaps: []string{"只能引用 start/100001,不能新增。"}},
	{Type: "2", Name: "结束", Registry: "end", SupportLevel: SupportSingleton, CanAdd: false, RuntimeSmoke: "not-executable"},
	{Type: "3", Name: "大模型", Registry: "nodes-v2/llm", SupportLevel: SupportFull, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "4", Name: "插件/API", Registry: "plugin", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "5", Name: "代码", Registry: "code", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "6", Name: "知识库检索", Registry: "dataset", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "8", Name: "条件分支", Registry: "if", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "9", Name: "子工作流", Registry: "sub-workflow", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少子工作流 schema 发现与语义配置器。"}},
	{Type: "11", Name: "变量", Registry: "variable", SupportLevel: SupportAddOnly, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"缺少变量节点语义配置器。"}},
	{Type: "12", Name: "SQL自定义", Registry: "database/database-base", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少 databaseInfo/sql/inputParameters 语义配置器。"}},
	{Type: "13", Name: "输出/纯输出", Registry: "output", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "14", Name: "图像流", Registry: "imageflow", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图像工作流资源发现与语义配置器。"}},
	{Type: "15", Name: "文本处理", Registry: "text-process", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "16", Name: "生成图片", Registry: "image-generate", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图片生成参数语义配置器。"}},
	{Type: "17", Name: "图片引用", Registry: "image-reference", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图片资源选择与绑定语义配置器。"}},
	{Type: "18", Name: "问答/追问", Registry: "question", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "19", Name: "跳出循环", Registry: "break", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"只能在循环子画布语义下可靠测试。"}},
	{Type: "20", Name: "变量赋值", Registry: "set-variable", SupportLevel: SupportAddOnly, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"缺少变量赋值语义配置器。"}},
	{Type: "21", Name: "循环", Registry: "loop", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"缺少循环数组、循环变量和子画布节点语义配置器。"}},
	{Type: "22", Name: "意图识别", Registry: "intent", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "resource", Gaps: []string{"原生意图节点语义配置仍需补齐。"}},
	{Type: "23", Name: "图片画布", Registry: "image-canvas", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图像画布配置器。"}},
	{Type: "26", Name: "长期记忆", Registry: "ltm", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少记忆资源发现与配置器。"}},
	{Type: "27", Name: "知识库写入", Registry: "dataset-write", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少写入策略和字段绑定语义配置器。"}},
	{Type: "28", Name: "批处理", Registry: "batch", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"缺少批处理数组、并发和子链路语义配置器。"}},
	{Type: "29", Name: "继续循环", Registry: "continue", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"只能在循环子画布语义下可靠测试。"}},
	{Type: "30", Name: "输入", Registry: "input", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "31", Name: "注释", Registry: "comment", SupportLevel: SupportDocumentationOnly, CanAdd: true, RuntimeSmoke: "not-executable"},
	{Type: "32", Name: "变量聚合", Registry: "variable-merge", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "34", Name: "触发器更新", Registry: "trigger-upsert", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少触发器 schema 配置器。"}},
	{Type: "35", Name: "触发器删除", Registry: "trigger-delete", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少触发器 schema 配置器。"}},
	{Type: "36", Name: "触发器读取", Registry: "trigger-read", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少触发器 schema 配置器。"}},
	{Type: "42", Name: "更新数据", Registry: "database-update", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "43", Name: "查询数据", Registry: "database-query", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "44", Name: "删除数据", Registry: "database-delete", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "45", Name: "HTTP请求", Registry: "http", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "resource", Gaps: []string{"缺少 method/url/headers/query/body/outputs 语义配置器。"}},
	{Type: "46", Name: "新增数据", Registry: "database-create", SupportLevel: SupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "58", Name: "JSON序列化", Registry: "json-stringify", SupportLevel: SupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "59", Name: "JSON解析", Registry: "json-parser", SupportLevel: SupportPartial, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"缺少 schema/paths/outputs 语义配置器。"}},
	{Type: "61", Name: "MCP", Registry: "mcp", SupportLevel: SupportPartial, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少 MCP server/tool 发现和参数绑定配置器。"}},
	{Type: "99", Name: "卡片选择", Registry: "card-selector", SupportLevel: SupportFull, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "100", Name: "智能体", Registry: "agent", SupportLevel: SupportFull, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
}

var nodeSpecs = map[string]NodeSpec{
	"2": {
		Type:        "2",
		Name:        "结束",
		Description: "结束节点是内置单例,不能新增。它支持两种返回方式:返回变量(returnVariables)和返回文本(useAnswerContent)。返回文本可开启流式输出,并可把多个变量通过 input/inputs 绑定后放进 content 模板一次性返回。",
		Commands:    []string{"workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"返回变量时配置 returns,变量必须来自 workflow.get_bindable_variables 返回的真实可绑定变量。",
			"返回文本时配置 input/inputs、content/text/template 和 streaming_output=true;content 只能引用本节点 input/inputs 的 name。",
			"多个输出变量可以全部绑定成 End inputParameters,再在返回文本中用 {{变量名}} 拼接。",
		},
	},
	"3": {
		Type:        "3",
		Name:        "大模型",
		Description: "大模型节点用于文本理解、生成、总结、抽取、分类和路由前判断。必须绑定输入、配置 prompt/system_prompt,并声明 outputs,至少 answer:string 或 output:string。下游只能引用已声明的 outputs。",
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"input/inputs 每项 {from,output,name};prompt 中只能引用本节点 input/inputs 的 name。",
			"prompt 做分类时要求只输出固定类别,并声明 category:string 等结构化输出。",
			"需要下游使用时必须声明 outputs,下游引用 {from:node_tag,output:answer} 或声明的字段名。",
		},
	},
	"4": {
		Type:        "4",
		Name:        "插件/API",
		Description: "插件/API 节点用于调用空间已有插件、API 或工具。配置前必须读取资源清单,plugin_id/api_id/plugin_name/api_name 必须来自真实资源,禁止编造。参数按接口 schema 绑定当前可用变量,并声明下游会消费的 outputs。",
		Commands:    []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.add_node", "workflow.connect", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"先获取空间可用插件/API 资源,再选择 plugin_id/api_id。",
			"inputs 对象的 key 是 API 参数名,value 是 {from,output} 变量绑定或明确字面量。",
			"如果资源清单没有输出 schema,按接口语义先声明 result:string,再由下游绑定。",
		},
		Gaps: []string{"运行 smoke 依赖空间内真实插件/API fixture。"},
	},
	"5": {
		Type:        "5",
		Name:        "代码",
		Description: `代码节点用于确定性逻辑、JSON/字段转换、正则提取、简单计算,不要用它做主观生成。当前运行时只支持 Python,language 写 "python";禁止 JavaScript/async function 语法。必须定义 async def main(args: Args) -> Output,先用 params = args.params or {} 取得全部入参,再用 params.get("入参名") 读取 input/inputs 绑定的字段;不要对 args 直接调用 strip/get/[]。return 字段名必须和 outputs 名称一致。`,
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			`language 写 "python",code 使用 async def main(args: Args) -> Output。`,
			"input/inputs 每项 {from,output,name};name 会成为 args.params 中的字段。",
			"先 params = args.params or {},再 params.get(\"字段名\");不要对 args 直接调用 strip/get。",
			"必须声明 outputs;return dict/Output 的字段名必须和 outputs 对齐,否则下游无法绑定。",
		},
	},
	"6": {
		Type:        "6",
		Name:        "知识库检索",
		Description: "知识库检索节点用于 RAG/知识召回。配置前必须读取空间知识库资源,dataset_id/dataset_ids 必须来自真实资源。输入通常绑定用户 query 或上游改写后的 query,输出供 LLM 或文本处理节点消费。",
		Commands:    []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.add_node", "workflow.connect", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"input 绑定 Query,通常 {from:start,output:input,name:Query} 或来自上游 query。",
			"dataset_ids 必须来自资源清单,top_k/score_threshold 按场景配置。",
			"常见输出为 outputList:array_object 和 output:string,下游使用前读取可绑定变量确认真实字段。",
		},
		Gaps: []string{"运行 smoke 依赖空间内真实知识库 fixture。"},
	},
	"8": {
		Type:        "8",
		Name:        "条件分支",
		Description: "条件分支节点用于按上游变量路由 true/false。条件左值必须绑定真实可绑定变量,right 可为字面量或变量绑定。true 分支连线 from_port=true,false 分支连线 from_port=false。",
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"配置 condition 前先调用 workflow.get_bindable_variables,只使用返回的真实变量。",
			"condition.left 使用 {from,output},operator 使用 equal/not_equal/contains/not_contains/null/not_null/gt/gte/lt/lte 等。",
			"互斥分支汇合时用 type=32 变量聚合,不要只连线不配置 merge_groups。",
		},
	},
	"13": {
		Type:        "13",
		Name:        "输出/纯输出",
		Description: "display-only 消息输出节点,用于展示固定文本或本节点 input/template 渲染结果;不要作为变量聚合或 End returns 的上游变量来源。若分支结果需要下游消费,使用 type=15 文本处理产出真实 output:string。",
		Commands:    []string{"workflow.add_node", "workflow.configure_node", "workflow.validate"},
		BindingRules: []string{
			"content/template 只能引用本节点 input/inputs 定义的变量名。",
			"不要作为变量聚合或 End returns 的变量来源。",
		},
	},
	"15": {
		Type:        "15",
		Name:        "文本处理",
		Description: "文本处理节点可拼接模板或固定文案并产出真实 output:string。固定文案没有变量引用时可以删除默认 input,但仍要声明 output:string 供变量聚合或 End returns 使用。",
		Commands:    []string{"workflow.add_node", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"模板只引用本节点 input/inputs 定义的变量名。",
			"固定文案可以无输入绑定。",
		},
	},
	"18": {
		Type:        "18",
		Name:        "问答/追问",
		Description: "问答/追问节点用于缺槽追问,例如缺少账号、卡号、收款人或确认信息。问题文本可以引用本节点 input/inputs 绑定的上下文变量;需要下游消费用户回答时必须声明 answer 或 USER_RESPONSE 等 outputs。",
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"question/content 中引用变量前,先在 input/inputs 中用同名 name 绑定。",
			"answer_type 可为 text 或 option;option 模式要提供 options。",
			"声明后续要消费的 outputs,下游配置前再读取 workflow.get_bindable_variables。",
		},
	},
	"22": {
		Type:        "22",
		Name:        "意图识别",
		Description: "意图识别节点用于把用户输入或上游文本分类成有限类别,再接 IF 或多分支处理。若当前原生意图节点配置 schema 不完整,可用 type=3 大模型节点模拟意图识别并声明 category:string。",
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"输入绑定待分类文本,类别列表必须明确且互斥。",
			"必须声明 category/intent 等输出,后续 IF 节点基于该真实输出判断。",
			"不确定原生 schema 时,优先用 LLM 节点输出固定类别,避免空选项或变量值为空。",
		},
		Gaps: []string{"原生意图节点语义配置仍需补齐,LLM 模拟分类更稳定。"},
	},
	"30": {
		Type:        "30",
		Name:        "输入",
		Description: "输入节点用于声明 Start.input 以外的结构化工作流入参,例如 user_id、account_id、amount、channel。它不是开始节点,可以新增;Start/End 仍然是内置单例。新增字段后下游应重新读取可绑定变量。",
		Commands:    []string{"workflow.add_node", "workflow.configure_node", "workflow.get_bindable_variables", "workflow.validate"},
		BindingRules: []string{
			"配置 outputs,每项 {name,type,description};字段名要和业务槽位一致。",
			"新增输入字段后调用 workflow.get_bindable_variables,再配置下游绑定。",
			"不要新增 Start 节点;结构化入参用本节点或 Start.input。",
		},
	},
	"32": {
		Type:        "32",
		Name:        "变量聚合",
		Description: "变量聚合节点用于分支汇合。配置前必须调用 workflow.get_bindable_variables,merge_groups.variables 必须全部来自真实可绑定变量;不要聚合 type=13 输出节点,固定文案分支先用 type=15 文本处理产出真实 output 变量。",
		Commands:    []string{"workflow.add_node", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"variables 必须来自 workflow.get_bindable_variables 返回的真实可绑定变量。",
			"所有可能返回分支都放入同一个 output 组。",
			"End returns 绑定聚合节点 output。",
		},
	},
	"58": {
		Type:        "58",
		Name:        "JSON序列化",
		Description: "JSON 序列化节点用于把 object/array/string 变量转换成 JSON 字符串,常用于 HTTP body、日志或返回文本。输入必须绑定真实变量或明确固定对象,需要下游消费时声明 output:string。",
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"input/inputs 绑定要序列化的真实变量,通常 name=input。",
			"声明 outputs 至少 output:string,下游引用 json_node.output。",
			"复杂对象组装可先用 Python Code 节点产出 object,再 JSON 序列化。",
		},
	},
	"59": {
		Type:        "59",
		Name:        "JSON解析",
		Description: "JSON 解析节点用于把 JSON 字符串解析成对象或结构化字段。必须绑定 JSON 字符串输入并声明下游会消费的 outputs;复杂 schema 不确定时优先用 type=5 Python 代码节点显式解析并声明 outputs。",
		Commands:    []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"input 绑定 JSON 字符串变量,不要直接猜旧节点输出。",
			"声明 outputs 为下游要消费的字段,例如 amount:number、payee:string。",
			"如果 schema/paths 语义配置不完整,改用 Code 节点解析并返回对齐 outputs 的 dict。",
		},
		Gaps: []string{"schema/paths 语义配置仍需补齐,复杂解析推荐 Code 节点。"},
	},
	"45": {
		Type:        "45",
		Name:        "HTTP请求",
		Description: "HTTP 请求节点用于调用外部接口。当前只记录为 partial,完整支持需要 method/url/headers/query/body/auth/outputs 语义配置器和运行 smoke。",
		Commands:    []string{"workflow.add_node", "workflow.get_node_spec"},
		Gaps:        []string{"缺少 method/url/headers/query/body/outputs 语义配置器。"},
	},
	"99": {
		Type:        "99",
		Name:        "卡片选择",
		Description: "卡片选择节点用于展示卡片并等待用户选择,例如转账确认、最近收款人选择、套餐选择。配置前必须读取真实卡片资源,card_id/selected_card 必须来自资源清单;选择结果需要声明 outputs 供下游绑定。",
		Commands:    []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.add_node", "workflow.connect", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"card_id 必须来自真实卡片资源,禁止编造。",
			"content/template 引用变量前先在 input/inputs 绑定同名 name。",
			"声明 selected_card/card_id/optionContent 等下游要消费的 outputs。",
		},
		Gaps: []string{"运行 smoke 依赖空间内真实卡片资源 fixture。"},
	},
	"100": {
		Type:        "100",
		Name:        "智能体",
		Description: "智能体节点用于调用已有 HiAgent/Coze 子智能体处理复杂子任务。配置前必须读取空间智能体资源,agent_id/platform/agent_name 必须来自真实资源;输入通常绑定 query 或结构化参数,并声明 answer:string 等 outputs。",
		Commands:    []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.add_node", "workflow.connect", "workflow.configure_node", "workflow.validate", "workflow.test_run"},
		BindingRules: []string{
			"agent_id、platform、agent_name 必须来自资源清单,禁止编造。",
			"query/inputParameters 绑定真实上游变量,例如 Start.input 或意图识别结果。",
			"声明 answer/output 等下游要消费的 outputs,再让变量聚合或 End 绑定这些真实输出。",
		},
		Gaps: []string{"运行 smoke 依赖空间内真实智能体资源 fixture。"},
	},
}

func ListNodeCapabilities() []NodeCapability {
	out := make([]NodeCapability, len(nodeCapabilities))
	copy(out, nodeCapabilities)
	return out
}

func BuildNodeSmokeManifest() NodeSmokeManifest {
	nodes := make([]NodeSmokeManifestNode, 0, len(nodeCapabilities))
	for _, cap := range nodeCapabilities {
		nodes = append(nodes, smokeManifestNodeFromCapability(cap))
	}
	return NodeSmokeManifest{
		Total:        len(nodeCapabilities),
		CoverageGaps: []string{},
		Summary:      summarizeSmokeManifestNodes(nodes),
		Nodes:        nodes,
	}
}

func smokeManifestNodeFromCapability(cap NodeCapability) NodeSmokeManifestNode {
	switch {
	case cap.Type == "1":
		return attachSmokeGuidance(NodeSmokeManifestNode{
			Type:             cap.Type,
			Name:             cap.Name,
			Status:           "verified-not-executable",
			RuntimeSmoke:     cap.RuntimeSmoke,
			Mode:             "readonly",
			Isolation:        "current-readonly",
			Ready:            true,
			RequiredCommands: []string{"workflow.get_canvas_context"},
			PlannedCommands:  []string{"workflow.get_canvas_context"},
			Gaps:             append([]string(nil), cap.Gaps...),
		})
	case cap.Type == "2":
		return localExecutableSmokeNode(cap, []string{"smoke_2_text_source"}, []string{"workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate"})
	case isLocalFullSmokeNode(cap):
		return localExecutableSmokeNode(cap, defaultTemporaryNodeTags(cap.Type), defaultRequiredCommands(cap.Type))
	case cap.SupportLevel == SupportAddOnly || cap.SupportLevel == SupportDocumentationOnly:
		return attachSmokeGuidance(NodeSmokeManifestNode{
			Type:                           cap.Type,
			Name:                           cap.Name,
			Status:                         statusFromCapability(cap),
			RuntimeSmoke:                   cap.RuntimeSmoke,
			Mode:                           "execute",
			Isolation:                      "temporary-workflow",
			Ready:                          true,
			RequiredCommands:               []string{"workflow.add_node", "workflow.delete_node"},
			PlannedCommands:                []string{"workflow.add_node", "workflow.delete_node", "workflow.auto_layout"},
			CleanupCommands:                []string{"workflow.delete_node", "workflow.auto_layout"},
			TemporaryNodeTags:              []string{"smoke_" + cap.Type},
			RequiresTemporaryWorkflowOptIn: true,
			Gaps:                           append([]string(nil), cap.Gaps...),
		})
	case cap.RequiresResource || cap.RuntimeSmoke == "resource":
		return attachSmokeGuidance(NodeSmokeManifestNode{
			Type:                    cap.Type,
			Name:                    cap.Name,
			Status:                  statusFromCapability(cap),
			RuntimeSmoke:            cap.RuntimeSmoke,
			Mode:                    "skip",
			Isolation:               "resource-fixture",
			Ready:                   false,
			RequiresResourceFixture: true,
			SkipReason:              resourceSkipReason(cap),
			Gaps:                    append([]string(nil), cap.Gaps...),
		})
	default:
		return attachSmokeGuidance(NodeSmokeManifestNode{
			Type:         cap.Type,
			Name:         cap.Name,
			Status:       "unsupported",
			RuntimeSmoke: cap.RuntimeSmoke,
			Mode:         "skip",
			Isolation:    "sub-canvas",
			Ready:        false,
			SkipReason:   gapSkipReason(cap),
			Gaps:         append([]string(nil), cap.Gaps...),
		})
	}
}

func localExecutableSmokeNode(cap NodeCapability, temporaryNodeTags []string, requiredCommands []string) NodeSmokeManifestNode {
	return attachSmokeGuidance(NodeSmokeManifestNode{
		Type:                           cap.Type,
		Name:                           cap.Name,
		Status:                         statusFromCapability(cap),
		RuntimeSmoke:                   cap.RuntimeSmoke,
		Mode:                           "execute",
		Isolation:                      "temporary-workflow",
		Ready:                          true,
		RequiredCommands:               requiredCommands,
		PlannedCommands:                plannedCommandsForRequired(requiredCommands),
		CleanupCommands:                cleanupCommandsForTemporaryNodes(len(temporaryNodeTags)),
		TemporaryNodeTags:              temporaryNodeTags,
		RequiresTemporaryWorkflowOptIn: true,
		Gaps:                           append([]string(nil), cap.Gaps...),
	})
}

func attachSmokeGuidance(node NodeSmokeManifestNode) NodeSmokeManifestNode {
	node.Assertions = smokeAssertions(node.Type)
	node.ExpectedBindableVariables = smokeExpectedBindableVariables(node.Type)
	return node
}

func smokeAssertions(nodeType string) []string {
	switch nodeType {
	case "2":
		return []string{
			"End 可以返回变量: returns 必须绑定真实上游输出,不能为空。",
			"End 可以返回文本",
			"End 可以返回文本: input/inputs 绑定多个上游变量,content 用 {{变量名}} 拼接,streaming_output=true 时流式透传。",
			"streaming_output=true",
			"配置 End 前先调用 workflow.get_bindable_variables,不要绑定 display-only 输出节点。",
		}
	case "5":
		return []string{
			"Python 代码节点使用 async def main(args: Args),输入从 args.params 读取。",
			"不要对 args 直接调用 strip/get 等字符串或 dict 方法。",
			"需要下游绑定时必须声明 outputs。",
		}
	case "8":
		return []string{
			"条件左值必须来自 workflow.get_bindable_variables 返回的可绑定变量。",
			"true/false 分支端口必须明确连线,修改条件后重新读取画布上下文确认。",
		}
	case "13":
		return []string{
			"display-only",
			"输出/纯输出节点是 display-only 消息展示节点。",
			"不要作为 VariableMerge 或 End returns 的稳定变量来源。",
			"如果固定文案需要被下游消费,先用 type=15 文本处理产出真实 output。",
		}
	case "15":
		return []string{
			"固定文案没有变量引用时可以删除默认 input。",
			"需要被下游消费时必须声明 output:string。",
			"content 里使用 {{变量名}} 时,必须先在本节点 input/inputs 中绑定同名变量。",
		}
	case "18":
		return []string{
			"问答节点用于缺槽追问,问题文本可以引用本节点已绑定 input。",
			"需要继续向下游传递时声明 answer 或 output。",
		}
	case "30":
		return []string{
			"输入节点用于显式收集用户补充信息,字段名必须和后续绑定变量一致。",
			"新增输入字段后重新读取可绑定变量再配置下游。",
		}
	case "32":
		return []string{
			"变量聚合必须先调用 workflow.get_bindable_variables,merge_groups.variables 全部来自可绑定变量。",
			"不要聚合 type=13 输出节点;固定文案分支先用 type=15 文本处理产出真实 output。",
			"End returns 应绑定 VariableMerge 的真实 output,不要直接猜旧节点变量。",
		}
	case "58":
		return []string{
			"JSON 序列化输入必须绑定真实变量或固定对象字段。",
			"需要下游消费时声明 output:string。",
		}
	default:
		return nil
	}
}

func smokeExpectedBindableVariables(nodeType string) []string {
	switch nodeType {
	case "2":
		return []string{"end.text"}
	case "3":
		return []string{"llm.answer"}
	case "5":
		return []string{"code.output"}
	case "8":
		return []string{"branch.condition"}
	case "13":
		return []string{}
	case "15":
		return []string{"text.output"}
	case "18":
		return []string{"question.answer"}
	case "30":
		return []string{"input.value"}
	case "32":
		return []string{"merge.output"}
	case "58":
		return []string{"json.output"}
	default:
		return nil
	}
}

func isLocalFullSmokeNode(cap NodeCapability) bool {
	return cap.RuntimeSmoke == "local" && cap.SupportLevel == SupportFull
}

func defaultRequiredCommands(nodeType string) []string {
	switch nodeType {
	case "13":
		return []string{"workflow.add_node", "workflow.connect", "workflow.configure_node", "workflow.validate"}
	case "30":
		return []string{"workflow.add_node", "workflow.configure_node", "workflow.validate"}
	default:
		return []string{"workflow.add_node", "workflow.connect", "workflow.get_bindable_variables", "workflow.configure_node", "workflow.validate", "workflow.test_run"}
	}
}

func plannedCommandsForRequired(required []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(required)+1)
	for _, command := range required {
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		out = append(out, command)
	}
	if _, ok := seen["workflow.auto_layout"]; !ok {
		out = append(out, "workflow.auto_layout")
	}
	return out
}

func cleanupCommandsForTemporaryNodes(count int) []string {
	out := make([]string, 0, count+1)
	for i := 0; i < count; i++ {
		out = append(out, "workflow.delete_node")
	}
	return append(out, "workflow.auto_layout")
}

func defaultTemporaryNodeTags(nodeType string) []string {
	switch nodeType {
	case "32":
		return []string{"smoke_32_variable_merge", "smoke_32_text_primary", "smoke_32_text_fallback"}
	case "58":
		return []string{"smoke_58_json_stringify", "smoke_58_text_source"}
	default:
		return []string{"smoke_" + nodeType}
	}
}

func statusFromCapability(cap NodeCapability) string {
	switch cap.SupportLevel {
	case SupportFull:
		return "verified-full"
	case SupportResourceBound:
		return "verified-resource-bound"
	case SupportAddOnly:
		return "verified-add-only"
	case SupportSingleton, SupportDocumentationOnly:
		return "verified-not-executable"
	default:
		return "unsupported"
	}
}

func resourceSkipReason(cap NodeCapability) string {
	if len(cap.Gaps) > 0 {
		return cap.Gaps[0]
	}
	return "运行依赖空间资源 fixture。"
}

func gapSkipReason(cap NodeCapability) string {
	if len(cap.Gaps) > 0 {
		return cap.Gaps[0]
	}
	return "当前节点 smoke 仍需补齐场景或子画布环境。"
}

func summarizeSmokeManifestNodes(nodes []NodeSmokeManifestNode) NodeSmokeManifestSummary {
	var summary NodeSmokeManifestSummary
	for _, node := range nodes {
		if node.Ready {
			summary.Ready++
		}
		if !node.Ready && node.Mode != "skip" {
			summary.NotReady++
		}
		if node.Mode == "skip" {
			summary.Skipped++
		}
		if node.Mode == "execute" {
			summary.Execute++
		}
		if node.Mode == "readonly" {
			summary.Readonly++
		}
		if node.RequiresTemporaryWorkflowOptIn {
			summary.RequiresTemporaryWorkflowOptIn++
		}
		if node.RequiresResourceFixture {
			summary.RequiresResourceFixture++
		}
	}
	return summary
}

func GetNodeCapability(typ string) (NodeCapability, bool) {
	for _, cap := range nodeCapabilities {
		if cap.Type == typ {
			return cap, true
		}
	}
	return NodeCapability{}, false
}

func GetNodeSpec(typ string) (NodeSpec, bool) {
	if spec, ok := nodeSpecs[typ]; ok {
		return spec, true
	}
	if cap, ok := GetNodeCapability(typ); ok {
		return fallbackNodeSpec(cap), true
	}
	return NodeSpec{}, false
}

func fallbackNodeSpec(cap NodeCapability) NodeSpec {
	spec := NodeSpec{
		Type:     cap.Type,
		Name:     cap.Name,
		Commands: fallbackSpecCommands(cap),
		Gaps:     append([]string(nil), cap.Gaps...),
	}

	switch cap.SupportLevel {
	case SupportSingleton:
		spec.Description = cap.Name + " 是内置单例节点,只能引用已有节点,不能新增或删除。修改前先读取画布上下文,确认当前内置节点的 node_tag/id 和可绑定变量。"
		spec.BindingRules = []string{
			"不要调用 workflow.add_node 新增单例节点。",
			"连线或配置前先调用 workflow.get_canvas_context 确认现有节点。",
		}
	case SupportResourceBound:
		spec.Description = cap.Name + " 是资源绑定节点,配置依赖空间内真实资源或外部资源 fixture。没有资源清单时只能登记为 skipped-with-reason,不能编造 ID 或宣称完整可运行。"
		spec.BindingRules = []string{
			"先读取画布上下文和可绑定变量,再选择真实资源。",
			"资源 ID、名称、schema 必须来自系统返回的资源清单或用户明确提供。",
			"缺少资源 fixture 时不要执行 test_run,应报告 requires_resource_fixture。",
		}
	case SupportPartial:
		spec.Description = cap.Name + " 已登记能力但语义配置器仍不完整。可以在用户明确需要时添加节点或保留设计占位,但不能把它当作已完整支持的执行节点。"
		spec.BindingRules = []string{
			"优先使用已 full support 的替代节点完成同等语义。",
			"如果必须使用该节点,只能在明确字段和资源都已知时配置。",
			"validate/test_run 失败时优先局部修复,不要默认清空画布。",
		}
	case SupportAddOnly:
		spec.Description = cap.Name + " 目前只可靠支持添加、删除和布局,还没有稳定语义配置器。它可以作为占位或简单变量结构使用,但不能宣称已完整配置业务逻辑。"
		spec.BindingRules = []string{
			"可以调用 workflow.add_node 创建,用 workflow.delete_node 清理。",
			"需要下游消费变量时,先确认该节点是否已经在画布中暴露真实 outputs。",
		}
	case SupportDocumentationOnly:
		spec.Description = cap.Name + " 是文档/辅助类节点,用于说明画布或辅助人工阅读,不参与运行链路。"
		spec.BindingRules = []string{
			"不要把该节点作为业务执行节点。",
			"不要把它的内容作为 End returns 或变量聚合来源。",
		}
	default:
		spec.Description = cap.Name + " 已登记但当前不具备完整自动化支持。使用前必须读取 gaps 并说明不可自动完成的原因。"
		spec.BindingRules = []string{
			"不能编造配置字段或资源 ID。",
			"需要人工或后续语义配置器补齐后才能升级支持等级。",
		}
	}

	if len(spec.Gaps) > 0 {
		spec.Description += " 当前缺口: " + spec.Gaps[0]
	}
	return spec
}

func fallbackSpecCommands(cap NodeCapability) []string {
	if !cap.CanAdd {
		return []string{"workflow.get_canvas_context", "workflow.get_bindable_variables"}
	}
	switch cap.SupportLevel {
	case SupportAddOnly, SupportDocumentationOnly:
		return []string{"workflow.add_node", "workflow.delete_node", "workflow.auto_layout"}
	case SupportResourceBound:
		return []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.add_node", "workflow.configure_node", "workflow.validate"}
	case SupportPartial:
		return []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.add_node", "workflow.configure_node", "workflow.validate"}
	default:
		return []string{"workflow.get_canvas_context", "workflow.add_node", "workflow.configure_node", "workflow.validate"}
	}
}
