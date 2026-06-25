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

// 工作流画布操作工具(workflow_canvas_*)。
//
// 设计:这些工具让 finmallclaw 在【工作流编辑器】里自动搭建工作流——加节点、连线、
// 配参数、优化布局、试运行。它们由超级 agent 调用,但【真正的执行在前端浏览器画布】:
// 后端 InvokableRun 只返回一个"已下发"的 ack;前端聊天监听消息流里的 FuncCall 事件,
// 据工具名+参数调用 playground 的指令总线 WorkflowAgentCommandService 在画布上实时执行
// (节点逐个出现、线逐条连上)。
//
// 为什么这样设计:画布状态在浏览器内存里,后端无法直接操作;而消息流里每个 tool_call
// 都会以 FuncCall 事件下发到前端(见 callback_reply_chunk.go),故前端天然能接管执行。
//
// 鉴权:agent 以当前用户身份在其登录会话内运行,前端在同一登录会话里操作画布与保存
// (/api/workflow_api/save 走 session),无需新增授权端点;这些工具本身无服务端副作用。
//
// 节点引用:agent 给每个新节点起一个逻辑名 node_tag,后续 connect/set_params 用 node_tag
// 引用;前端维护 node_tag→真实 nodeId 的映射。故后端无需把真实 id 回传给 agent。

func init() {
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetOperationGuideTool{}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetNodeCatalogTool{ext: deps.Ext}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetNodeSpecTool{}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetContextTool{ext: deps.Ext}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetNodeCapabilityAuditTool{ext: deps.Ext}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetResourceCatalogTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetNodeSmokeManifestTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetNodeSmokeCoverageTool{}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasGetBindableVariablesTool{ext: deps.Ext}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasAddNodeTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasConnectTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasDeleteNodeTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasDeleteLineTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasClearCanvasTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasConfigureNodeTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasSetParamsTool{}
	})
	registerSuperAgentExtension(func(_ superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasAutoLayoutTool{}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasTestRunTool{ext: deps.Ext, backendTestRun: newWorkflowCanvasBackendTestRunFunc(deps)}
	})
}

// 节点类型说明(StandardNodeType),供工具描述里告诉 agent 可新增类型。
const wfNodeTypeHelp = "可新增节点类型(type)取值: '3'=大模型(LLM) " +
	"'4'=插件/API '5'=代码 '6'=知识库检索 '8'=条件分支(If/选择器,出口分 true/false) " +
	"'9'=子工作流 '11'=变量 '12'=数据库 '13'=输出/纯输出 '15'=文本处理 '18'=问答/追问 " +
	"'21'=循环 '22'=意图识别 '27'=知识库写入 '28'=批处理 '30'=输入 '32'=变量聚合 " +
	"'42'=更新数据 '43'=查询数据 '44'=删除数据 '45'=HTTP '46'=新增数据 " +
	"'58'=JSON序列化 '59'=JSON解析 '61'=MCP '99'=卡片选择 '100'=子智能体。" +
	" 开始节点(100001,type=1)和结束节点(900001,type=2)是工作流内置单例,不能新增;需要连线时直接引用 start/end 或 100001/900001。"

const (
	workflowCanvasNodeCatalogExtKey     = "workflow_canvas_node_catalog"
	workflowCanvasNodeCapabilityExtKey  = "workflow_canvas_node_capability_audit"
	workflowCanvasContextExtKey         = "workflow_canvas_context"
	workflowCanvasBindableVarsExtKey    = "workflow_canvas_bindable_variables"
	workflowCanvasResourceSummaryExtKey = "workflow_canvas_resource_summary"
	workflowCanvasBindingGuideExtKey    = "workflow_canvas_binding_guide"
	workflowCanvasTestRunRequiredExtKey = "workflow_canvas_test_run_required"
	workflowCanvasTestRunInputExtKey    = "workflow_canvas_test_run_input"
	workflowCanvasWorkflowIDExtKey      = "workflow_canvas_workflow_id"
	workflowCanvasSpaceIDExtKey         = "workflow_canvas_space_id"
)

func wfExtValue(ext map[string]string, key string) string {
	return strings.TrimSpace(ext[key])
}

func wfIsSingletonNodeType(t string) bool {
	switch strings.TrimSpace(t) {
	case "1", "2":
		return true
	default:
		return false
	}
}

func wfAck(op string, payload any) string {
	b, _ := json.Marshal(map[string]any{
		"status": "dispatched_to_canvas",
		"op":     op,
		"args":   payload,
		"note": "已下发到前端画布执行;这不是执行成功、不是配置正确、也不是测试通过。" +
			" 在确认当前校验错误为空或试运行真实结果成功前,不能宣称完成。" +
			" 只有在 workflow_canvas_get_canvas_context 返回当前校验错误为空,或试运行真实结果成功后,才能宣称完成。" +
			" 注意 workflow_canvas_get_canvas_context 是本次用户消息发送前快照;同一轮刚下发的操作要结合本轮 node_tag/outputs/returns 自检,不要把旧快照误判成空画布或最新画布。" +
			" 若仍有错误,必须继续配置/改线/删除重建对应节点。用你起的 node_tag 引用节点进行后续连线/配参。",
	})
	return string(b)
}

// ---- workflow_canvas_get_node_catalog ----

type wfCanvasGetNodeCatalogTool struct {
	ext map[string]string
}

func (t *wfCanvasGetNodeCatalogTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_node_catalog",
		Desc: "获取工作流可添加节点目录和每类节点用途,用于先设计方案。" +
			"第一步先调用它;确定要使用某种节点后,再调用 workflow_canvas_get_node_spec(type) 获取该节点完整配置说明。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetNodeCatalogTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	if dynamic := wfExtValue(t.ext, workflowCanvasNodeCatalogExtKey); dynamic != "" {
		return wfNodeCatalogText + "\n\n当前前端动态可添加节点目录:\n" + dynamic, nil
	}
	return wfNodeCatalogText, nil
}

// ---- workflow_canvas_get_node_spec ----

type wfCanvasGetNodeSpecTool struct{}

type wfNodeSpecArgs struct {
	Type string `json:"type"`
}

func (t *wfCanvasGetNodeSpecTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_node_spec",
		Desc: "按节点 type 获取该节点完整规格:适用场景、必须配置的字段、可绑定参数、输出变量、configure_node 示例。" +
			"准备添加或配置某类节点前调用,不要凭记忆猜节点内部配置。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"type": {Type: schema.String, Desc: "节点类型 id,如 3=大模型,5=代码,8=IF,6=知识库,4=插件/API,100=智能体", Required: true},
		}),
	}, nil
}

func (t *wfCanvasGetNodeSpecTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfNodeSpecArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	nodeType := strings.TrimSpace(a.Type)
	if nodeType == "" {
		return "Error: type 必填", nil
	}
	if spec, ok := wfNodeSpecForType(nodeType); ok {
		return spec, nil
	}
	return "未找到该节点的详细规格。请根据当前可新增节点目录和节点标题判断用途;添加后务必用 workflow_canvas_configure_node 设置 title/input/outputs,不确定资源时在标题标注待确认。", nil
}

// ---- workflow_canvas_get_canvas_context ----

type wfCanvasGetContextTool struct {
	ext map[string]string
}

func (t *wfCanvasGetContextTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_canvas_context",
		Desc: "读取当前工作流上下文:节点、连线、每个节点 outputs、当前可绑定变量、空间资源清单、试运行意图。" +
			"复杂流程修改前先调用它,后续 configure_node 必须只绑定这些已存在变量或你本轮 add_node 后声明的 outputs。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetContextTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	b, _ := json.Marshal(map[string]string{
		"status":              "canvas_context",
		"workflow_id":         wfExtValue(t.ext, workflowCanvasWorkflowIDExtKey),
		"space_id":            wfExtValue(t.ext, workflowCanvasSpaceIDExtKey),
		"canvas_context":      wfExtValue(t.ext, workflowCanvasContextExtKey),
		"node_capabilities":   wfExtValue(t.ext, workflowCanvasNodeCapabilityExtKey),
		"bindable_variables":  wfExtValue(t.ext, workflowCanvasBindableVarsExtKey),
		"resource_summary":    wfExtValue(t.ext, workflowCanvasResourceSummaryExtKey),
		"binding_guide":       wfExtValue(t.ext, workflowCanvasBindingGuideExtKey),
		"test_run_required":   wfExtValue(t.ext, workflowCanvasTestRunRequiredExtKey),
		"test_run_input_hint": wfExtValue(t.ext, workflowCanvasTestRunInputExtKey),
		"instruction":         "基于 canvas_context 增量编辑;canvas_context 是本次用户消息发送前快照,同一轮刚下发的画布操作可能尚未反映,不要把旧快照误判成空画布或最新画布;配置节点前先确认可绑定变量;完成一组增删改连线后从 Start 到 End 审计输入绑定、变量引用、分支出口、变量聚合和 End returns;如果绑定诊断不是 none,禁止说完成,必须局部修复对应节点;禁止默认 clear_canvas,除非用户明确要求整体重做或上下文证明局部修复不可行;试运行失败后读取失败节点并局部修复对应节点的输入绑定/变量引用/outputs/merge_groups/End returns。",
	})
	return string(b), nil
}

// ---- workflow_canvas_get_node_capability_audit ----

type wfCanvasGetNodeCapabilityAuditTool struct {
	ext map[string]string
}

func (t *wfCanvasGetNodeCapabilityAuditTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_node_capability_audit",
		Desc: "读取当前前端节点能力覆盖审计:哪些节点 full support,哪些只是 resource-bound/partial/add-only,哪些缺少语义配置器或节点规格。" +
			"设计复杂工作流前调用它,不要把 partial/add-only 节点当作已经完整可配置。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetNodeCapabilityAuditTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	b, _ := json.Marshal(map[string]string{
		"status":            "node_capability_audit",
		"node_capabilities": wfExtValue(t.ext, workflowCanvasNodeCapabilityExtKey),
		"instruction":       "full 节点可优先直接使用 configure_node; resource-bound 节点必须先确认真实资源; partial/add-only 节点不要猜内部表单 path,优先使用已支持节点替代或明确告诉用户当前能力缺口。",
	})
	return string(b), nil
}

// ---- workflow_canvas_get_node_smoke_manifest ----

type wfCanvasGetNodeSmokeManifestTool struct{}

func (t *wfCanvasGetNodeSmokeManifestTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_node_smoke_manifest",
		Desc: "获取工作流节点 smoke/readiness 清单:每种节点是否可执行自测、需要哪些 workflow_canvas_* 工具、是否必须临时 workflow 隔离、是否需要真实资源 fixture。" +
			"复杂建图、节点巡检或调试变量绑定前调用它,再按清单逐节点读取 spec/context/bindable variables 并配置。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetNodeSmokeManifestTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	b, _ := json.Marshal(buildWFCanvasNodeSmokeManifest())
	return string(b), nil
}

// ---- workflow_canvas_get_node_smoke_coverage ----

type wfCanvasGetNodeSmokeCoverageTool struct{}

func (t *wfCanvasGetNodeSmokeCoverageTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_node_smoke_coverage",
		Desc: "获取 41 个工作流节点的覆盖总表: manifest/spec 是否完整、是否可执行、是否 resource-bound/partial、是否能作为下游变量来源。" +
			"复杂建图和节点巡检前调用它,尤其用于避免把 type=13 输出节点误当成变量来源,并确认 type=32 变量聚合需要哪些绑定断言。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetNodeSmokeCoverageTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	b, _ := json.Marshal(buildWFCanvasNodeSmokeCoverageReport())
	return string(b), nil
}

// ---- workflow_canvas_get_bindable_variables ----

type wfCanvasGetBindableVariablesTool struct {
	ext map[string]string
}

func (t *wfCanvasGetBindableVariablesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_bindable_variables",
		Desc: "读取当前画布可绑定变量候选,用于配置节点 input/inputs、IF condition、变量聚合 merge_groups、End returns。" +
			"配置变量前调用它;只能绑定返回列表里的变量,或本轮刚 add_node 且已声明 outputs 的变量。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetBindableVariablesTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	b, _ := json.Marshal(map[string]string{
		"status":             "bindable_variables",
		"bindable_variables": wfExtValue(t.ext, workflowCanvasBindableVarsExtKey),
		"canvas_context":     wfExtValue(t.ext, workflowCanvasContextExtKey),
		"instruction":        "配置 input/inputs/condition/merge_groups/returns 时,from 使用节点 node_tag 或真实 nodeId,output 使用候选变量名;type=13 输出/消息节点如果不在候选列表里,不能被聚合或 End 返回,需要用 type=15 文本处理产出 output。",
	})
	return string(b), nil
}

// ---- workflow_canvas_add_node ----

type wfCanvasAddNodeTool struct{}

type wfAddNodeArgs struct {
	NodeTag string `json:"node_tag"`
	Type    string `json:"type"`
	Title   string `json:"title,omitempty"`
	X       *int   `json:"x,omitempty"`
	Y       *int   `json:"y,omitempty"`
}

func (t *wfCanvasAddNodeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_add_node",
		Desc: "在工作流画布上新增一个节点(实时出现)。" + wfNodeTypeHelp +
			" 必须给该节点起一个唯一的 node_tag(逻辑名,如 'intent'、'branch1'),后续 connect/set_params 用它引用。" +
			" 位置 x/y 可不传(自动避让排布)。新建空节点采用该类型默认配置;要让工作流可运行,之后用 workflow_canvas_configure_node 配置输入绑定、prompt/代码/条件和 outputs。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"node_tag": {Type: schema.String, Desc: "该节点的唯一逻辑名,用于后续引用", Required: true},
			"type":     {Type: schema.String, Desc: "节点类型 id,见说明", Required: true},
			"title":    {Type: schema.String, Desc: "节点标题(可选)", Required: false},
			"x":        {Type: schema.Integer, Desc: "画布 x 坐标(可选)", Required: false},
			"y":        {Type: schema.Integer, Desc: "画布 y 坐标(可选)", Required: false},
		}),
	}, nil
}

func (t *wfCanvasAddNodeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfAddNodeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	if a.NodeTag == "" || a.Type == "" {
		return "Error: node_tag 和 type 必填", nil
	}
	if wfIsSingletonNodeType(a.Type) {
		return "Error: Start/End 是工作流内置 singleton 单例节点,不能通过 workflow_canvas_add_node 新增;请直接引用 start(100001) 或 end(900001) 进行连线/配置。", nil
	}
	return wfAck("add_node", a), nil
}

// ---- workflow_canvas_connect ----

type wfCanvasConnectTool struct{}

type wfConnectArgs struct {
	From     string `json:"from"`
	To       string `json:"to"`
	FromPort string `json:"from_port,omitempty"`
	ToPort   string `json:"to_port,omitempty"`
}

func (t *wfCanvasConnectTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_connect",
		Desc: "在两个已存在的节点间连一条线(实时出现),数据从 from 流向 to。from/to 用 add_node 时起的 node_tag。" +
			" 条件分支(type='8')有两个出口,用 from_port 区分: 'true'(如果) / 'false'(否则)。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"from":      {Type: schema.String, Desc: "源节点 node_tag", Required: true},
			"to":        {Type: schema.String, Desc: "目标节点 node_tag", Required: true},
			"from_port": {Type: schema.String, Desc: "源出口端口(可选;条件分支用 true/false)", Required: false},
			"to_port":   {Type: schema.String, Desc: "目标入口端口(可选)", Required: false},
		}),
	}, nil
}

func (t *wfCanvasConnectTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfConnectArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	if a.From == "" || a.To == "" {
		return "Error: from 和 to 必填", nil
	}
	return wfAck("connect", a), nil
}

// ---- workflow_canvas_delete_node ----

type wfCanvasDeleteNodeTool struct{}

type wfDeleteNodeArgs struct {
	NodeTag string `json:"node_tag"`
	Node    string `json:"node,omitempty"`
}

func (t *wfCanvasDeleteNodeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_delete_node",
		Desc: "删除画布上的一个普通节点,用于重构或移除错误节点。Start/End 是内置单例,不能删除。" +
			" 删除节点会同时移除它相关的连线;删除后如需继续使用该能力请重新 add_node 并 configure_node。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"node_tag": {Type: schema.String, Desc: "要删除的节点 node_tag 或真实 node id", Required: true},
		}),
	}, nil
}

func (t *wfCanvasDeleteNodeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfDeleteNodeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	if a.NodeTag == "" && a.Node == "" {
		return "Error: node_tag 必填", nil
	}
	return wfAck("delete_node", a), nil
}

// ---- workflow_canvas_delete_line ----

type wfCanvasDeleteLineTool struct{}

func (t *wfCanvasDeleteLineTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_delete_line",
		Desc: "删除两个节点之间的连线,用于改线、重构分支或修复错误连接。from/to 用 node_tag;条件分支可用 from_port=true/false 精确指定出口。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"from":      {Type: schema.String, Desc: "源节点 node_tag", Required: true},
			"to":        {Type: schema.String, Desc: "目标节点 node_tag", Required: true},
			"from_port": {Type: schema.String, Desc: "源出口端口(可选;条件分支用 true/false)", Required: false},
			"to_port":   {Type: schema.String, Desc: "目标入口端口(可选)", Required: false},
		}),
	}, nil
}

func (t *wfCanvasDeleteLineTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfConnectArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	if a.From == "" || a.To == "" {
		return "Error: from 和 to 必填", nil
	}
	return wfAck("delete_line", a), nil
}

// ---- workflow_canvas_clear_canvas ----

type wfCanvasClearCanvasTool struct{}

func (t *wfCanvasClearCanvasTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_clear_canvas",
		Desc: "清空当前画布中除内置 Start/End 外的所有普通节点和连线,用于复杂需求推倒重来。" +
			" 只有当用户明确要求重新设计、清空、重构,或当前流程与需求明显冲突时使用。清空后必须重新 add_node/connect/configure_node/auto_layout。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasClearCanvasTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return wfAck("clear_canvas", struct{}{}), nil
}

// ---- workflow_canvas_configure_node ----

type wfCanvasConfigureNodeTool struct{}

type wfConfigureNodeArgs struct {
	NodeTag string          `json:"node_tag"`
	Config  json.RawMessage `json:"config"`
}

func (t *wfCanvasConfigureNodeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_configure_node",
		Desc: "用语义化 JSON 配置节点内部表单,由前端翻译成真实画布 schema。优先使用它,不要猜内部表单 path。" +
			" 常用字段: title; input/inputs 绑定上游变量,形如 {from:'start',output:'input',name:'input'};" +
			" prompt/user_prompt/system_prompt 配置 LLM; outputs 声明输出变量,如 [{name:'answer',type:'string'}];" +
			" returns 配置 End 返回变量,如 [{name:'output',from:'llm',output:'answer'}];" +
			" End 返回文本时用 input/inputs 绑定多个上游输出,content/text/template 拼最终回答,streaming_output=true 开启流式透传;" +
			" code/language 配置 Code; condition 配置 If,如 {left:{from:'intent',output:'category'},operator:'equal',right:'售后'}。" +
			" merge_groups 配置变量聚合(type=32),如 [{name:'output',variables:[{from:'code',output:'result'},{from:'agent',output:'answer'}]}]。" +
			" 配置前先根据 workflow_canvas_get_canvas_context/当前画布上下文确认可绑定变量。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"node_tag": {Type: schema.String, Desc: "要配置的节点 node_tag;内置结束节点可用 end", Required: true},
			"config":   {Type: schema.Object, Desc: "语义化配置对象(input/outputs/prompt/returns/code/condition 等)", Required: true},
		}),
	}, nil
}

func (t *wfCanvasConfigureNodeTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfConfigureNodeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	if a.NodeTag == "" {
		return "Error: node_tag 必填", nil
	}
	if len(a.Config) == 0 || string(a.Config) == "null" {
		return "Error: config 必填", nil
	}
	return wfAck("configure_node", a), nil
}

// ---- workflow_canvas_set_node_params ----

type wfCanvasSetParamsTool struct{}

type wfSetParamsArgs struct {
	NodeTag string          `json:"node_tag"`
	Params  json.RawMessage `json:"params"`
}

func (t *wfCanvasSetParamsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_set_node_params",
		Desc: "高级兜底:直接配置某个节点的内部表单 path。一般应优先使用 workflow_canvas_configure_node。" +
			" params 是该节点表单 path 到 value 的 JSON。" +
			" 例如给大模型节点设 prompt、给输入参数绑定上游节点的输出(引用上游 node_tag 的 output)。" +
			" 只连线不配参数的节点是'未配置输入'状态、无法试运行,所以搭完拓扑后要给关键节点配参。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"node_tag": {Type: schema.String, Desc: "要配置的节点 node_tag", Required: true},
			"params":   {Type: schema.Object, Desc: "节点表单数据(JSON 对象),如 {inputs, llmParam, condition ...}", Required: true},
		}),
	}, nil
}

func (t *wfCanvasSetParamsTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfSetParamsArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &a); err != nil {
		return argParseErrMsg(err), nil
	}
	if a.NodeTag == "" {
		return "Error: node_tag 必填", nil
	}
	return wfAck("set_node_params", a), nil
}

// ---- workflow_canvas_auto_layout ----

type wfCanvasAutoLayoutTool struct{}

func (t *wfCanvasAutoLayoutTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "workflow_canvas_auto_layout",
		Desc:        "一键优化/整理画布布局,让节点排布整齐、连线清晰。每次新增若干节点和连线后建议调用一次。无参数。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasAutoLayoutTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return wfAck("auto_layout", struct{}{}), nil
}

// ---- workflow_canvas_test_run ----

type wfCanvasTestRunTool struct {
	ext            map[string]string
	backendTestRun workflowCanvasBackendTestRunFunc
}

type wfTestRunArgs struct {
	Input map[string]string `json:"input,omitempty"`
}

func (t *wfCanvasTestRunTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_test_run",
		Desc: "对当前工作流发起试运行(完整调试),验证整图能否跑通。input 是开始节点的输入参数(键值对,可选)。" +
			" 运行前应确保关键节点已配置输入,否则会因'未配置输入'失败。" +
			" 当用户要求试运行、调试、运行、验证、测试或'完成后跑一下'时,必须在 auto_layout 后调用本工具。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"input": {Type: schema.Object, Desc: "开始节点输入参数(键值对,可选)", Required: false},
		}),
	}, nil
}

func (t *wfCanvasTestRunTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var a wfTestRunArgs
	if argumentsInJSON != "" {
		_ = json.Unmarshal([]byte(argumentsInJSON), &a)
	}
	workflowID := wfExtValue(t.ext, workflowCanvasWorkflowIDExtKey)
	spaceID := wfExtValue(t.ext, workflowCanvasSpaceIDExtKey)
	if workflowID == "" || spaceID == "" {
		return wfAck("test_run", a), nil
	}

	if t.backendTestRun == nil {
		return wfAck("test_run", a), nil
	}

	resp, err := t.backendTestRun(ctx, workflowCanvasBackendTestRunRequest{
		WorkflowID: workflowID,
		SpaceID:    spaceID,
		Input:      a.Input,
	})
	if err != nil {
		b, _ := json.Marshal(map[string]any{
			"status":      "workflow_test_error",
			"op":          "test_run",
			"args":        a,
			"workflow_id": workflowID,
			"space_id":    spaceID,
			"error":       err.Error(),
			"note":        "后端同步试运行启动或轮询失败;不能宣称测试通过。请先根据错误修复或稍后重试。",
		})
		return string(b), nil
	}
	b, _ := json.Marshal(map[string]any{
		"status":      "workflow_test_result",
		"op":          "test_run",
		"args":        a,
		"workflow_id": workflowID,
		"space_id":    spaceID,
		"result":      resp.Data,
		"note":        "这是后端对已保存草稿发起的真实试运行结果。status=success 才能宣称通过;status=failed/timeout/canceled 时必须根据 failed_nodes、reason、nodes 继续修复。若你刚在前端画布下发了增删改节点操作,需等待自动保存完成后再运行,否则后端可能测到旧草稿。",
	})
	return string(b), nil
}
