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
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	canvasauto "github.com/ynet-dev/ynet-studio/backend/application/workflow/canvasautomation"
)

// workflowCanvasLiveRelay 是 get_canvas_context 拿"前端实时画布"用的通道(就是前端聊天面板
// 正在 1500ms 轮询的那个 relay)。nil 时工具自动回退到 deps.Ext 里的旧快照,绝不 hang。
type workflowCanvasLiveRelay interface {
	DispatchWorkflowCommand(context.Context, canvasauto.WorkflowCanvasDispatchCommand) error
	WaitWorkflowCommandResult(context.Context, string) (canvasauto.WorkflowCanvasCommandResult, bool)
}

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
		return &wfCanvasGetContextTool{ext: deps.Ext, relay: deps.CanvasRelay, ledger: deps.Ledger}
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
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasAddNodeTool{ledger: deps.Ledger}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasConnectTool{ledger: deps.Ledger}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasDeleteNodeTool{ledger: deps.Ledger}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasDeleteLineTool{ledger: deps.Ledger}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasClearCanvasTool{ledger: deps.Ledger}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasConfigureNodeTool{ledger: deps.Ledger}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasSetParamsTool{ledger: deps.Ledger}
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
	ext    map[string]string
	relay  workflowCanvasLiveRelay
	ledger *turnCanvasLedger
}

func (t *wfCanvasGetContextTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_get_canvas_context",
		Desc: "读取当前工作流上下文:节点、连线、每个节点 outputs、当前可绑定变量、空间资源清单、试运行意图。" +
			"复杂流程修改前先调用它。返回里 canvas_context 是画布读取(因异步应用有延迟,可能还没含你本轮刚加的节点);" +
			"your_turn_operations 是后端记录的你本轮已成功下发的全部画布操作的权威清单,用它确认自己的编辑已生效,不要因 canvas_context 暂时看不到就重做或清空。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasGetContextTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	workflowID := wfExtValue(t.ext, workflowCanvasWorkflowIDExtKey)
	spaceID := wfExtValue(t.ext, workflowCanvasSpaceIDExtKey)
	// 优先取前端正在轮询的 relay 上的【实时画布】(含本轮刚下发的增删改),根治"看不到自己编辑→误判→清空"。
	if t.relay != nil && workflowID != "" && spaceID != "" {
		if live, ok := t.fetchLiveCanvas(ctx, workflowID, spaceID); ok {
			return live, nil
		}
	}
	// 回退:无 relay / 编辑页未打开没人响应 / 超时 → 用消息发送前的旧快照,绝不 hang。
	return t.snapshotResult(workflowID, spaceID), nil
}

func (t *wfCanvasGetContextTool) fetchLiveCanvas(ctx context.Context, workflowID, spaceID string) (string, bool) {
	requestID := fmt.Sprintf("agentflow-get-ctx-%d", time.Now().UnixNano())
	if err := t.relay.DispatchWorkflowCommand(ctx, canvasauto.WorkflowCanvasDispatchCommand{
		WorkflowID:       workflowID,
		SpaceID:          spaceID,
		RequestID:        requestID,
		RequiresResponse: true,
		Command:          canvasauto.WorkflowCanvasCommand{Op: "get_canvas_context"},
	}); err != nil {
		return "", false
	}
	// 命令一直留在 relay 队列里(poll 不删除),前端 bridge 每 ~1.5s 轮询一次。
	// 浏览器↔后端这条链路会间歇性抖动(实测单次断连可达 ~10s),8s 太短会在抖动窗口里
	// 超时回退到旧快照、还把已恢复后到达的回填变成孤儿。给到 25s:只要网络在窗口内恢复
	// 任意一次,bridge 重新 poll 就能取走命令并回填,等待者仍在 → 拿到实时画布。
	waitCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	result, ok := t.relay.WaitWorkflowCommandResult(waitCtx, requestID)
	if !ok || strings.TrimSpace(result.CanvasContext) == "" {
		return "", false
	}
	bindable := strings.TrimSpace(result.BindableVariables)
	if bindable == "" {
		bindable = wfExtValue(t.ext, workflowCanvasBindableVarsExtKey)
	}
	b, _ := json.Marshal(map[string]string{
		"status":               "canvas_context",
		"source":               "browser_live",
		"workflow_id":          workflowID,
		"space_id":             spaceID,
		"canvas_context":       result.CanvasContext,
		"your_turn_operations": t.ledger.renderForAgent(),
		"bindable_variables":   bindable,
		"node_capabilities":    wfExtValue(t.ext, workflowCanvasNodeCapabilityExtKey),
		"resource_summary":     wfExtValue(t.ext, workflowCanvasResourceSummaryExtKey),
		"binding_guide":        wfExtValue(t.ext, workflowCanvasBindingGuideExtKey),
		"test_run_required":    wfExtValue(t.ext, workflowCanvasTestRunRequiredExtKey),
		"instruction":          "canvas_context 是浏览器画布的实时读取,但你本轮刚下发的 add_node/connect/configure 是【异步应用】的,可能还没出现在 canvas_context 里——这是正常现象,不代表失败。your_turn_operations 是后端记录的、你本轮已成功下发的全部画布操作的【权威清单】;请把 canvas_context 与 your_turn_operations 合并起来在脑中重建当前画布,以 your_turn_operations 为准确认自己的编辑已生效。从 Start 到 End 审计输入绑定、变量引用、分支出口、变量聚合、End returns;【严禁】因为 canvas_context 里暂时看不到刚加的节点就重做或 clear_canvas;绑定诊断非 none 时只局部修复对应节点。",
	})
	return string(b), true
}

func (t *wfCanvasGetContextTool) snapshotResult(workflowID, spaceID string) string {
	b, _ := json.Marshal(map[string]string{
		"status":               "canvas_context",
		"source":               "pre_turn_snapshot",
		"workflow_id":          workflowID,
		"space_id":             spaceID,
		"canvas_context":       wfExtValue(t.ext, workflowCanvasContextExtKey),
		"your_turn_operations": t.ledger.renderForAgent(),
		"node_capabilities":    wfExtValue(t.ext, workflowCanvasNodeCapabilityExtKey),
		"bindable_variables":   wfExtValue(t.ext, workflowCanvasBindableVarsExtKey),
		"resource_summary":     wfExtValue(t.ext, workflowCanvasResourceSummaryExtKey),
		"binding_guide":        wfExtValue(t.ext, workflowCanvasBindingGuideExtKey),
		"test_run_required":    wfExtValue(t.ext, workflowCanvasTestRunRequiredExtKey),
		"test_run_input_hint":  wfExtValue(t.ext, workflowCanvasTestRunInputExtKey),
		"instruction":          "canvas_context 是本次用户消息发送前的快照(实时画布本次没取到,通常是编辑页未打开或浏览器↔后端网络瞬时抖动,与你的操作无关);你本轮刚下发的画布操作【一定还没反映在这份快照里】。your_turn_operations 是后端记录的、你本轮已成功下发的全部画布操作的【权威清单】;请把快照与 your_turn_operations 合并起来在脑中重建当前画布,以 your_turn_operations 为准确认自己的编辑已生效,【绝不能把旧快照误判成空画布或最新画布】。只要工具返回成功就继续往下做,【严禁因为快照里看不到刚加的节点就 clear_canvas 或推倒重做】;配置前先确认可绑定变量;绑定诊断非 none 时只局部修复对应节点。",
	})
	return string(b)
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

type wfCanvasAddNodeTool struct{ ledger *turnCanvasLedger }

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
	t.ledger.record(turnCanvasLedgerOp{Op: "add_node", NodeTag: a.NodeTag, Type: a.Type, Title: a.Title})
	return wfAck("add_node", a), nil
}

// ---- workflow_canvas_connect ----

type wfCanvasConnectTool struct{ ledger *turnCanvasLedger }

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
	t.ledger.record(turnCanvasLedgerOp{Op: "connect", From: a.From, To: a.To})
	return wfAck("connect", a), nil
}

// ---- workflow_canvas_delete_node ----

type wfCanvasDeleteNodeTool struct{ ledger *turnCanvasLedger }

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
	delTag := a.NodeTag
	if delTag == "" {
		delTag = a.Node
	}
	t.ledger.record(turnCanvasLedgerOp{Op: "delete_node", NodeTag: delTag})
	return wfAck("delete_node", a), nil
}

// ---- workflow_canvas_delete_line ----

type wfCanvasDeleteLineTool struct{ ledger *turnCanvasLedger }

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
	t.ledger.record(turnCanvasLedgerOp{Op: "delete_line", From: a.From, To: a.To})
	return wfAck("delete_line", a), nil
}

// ---- workflow_canvas_clear_canvas ----

type wfCanvasClearCanvasTool struct{ ledger *turnCanvasLedger }

func (t *wfCanvasClearCanvasTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_clear_canvas",
		Desc: "清空当前画布中除内置 Start/End 外的所有普通节点和连线,用于复杂需求推倒重来。" +
			" 只有当用户明确要求重新设计、清空、重构,或当前流程与需求明显冲突时使用。清空后必须重新 add_node/connect/configure_node/auto_layout。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *wfCanvasClearCanvasTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	t.ledger.record(turnCanvasLedgerOp{Op: "clear_canvas"})
	return wfAck("clear_canvas", struct{}{}), nil
}

// ---- workflow_canvas_configure_node ----

type wfCanvasConfigureNodeTool struct{ ledger *turnCanvasLedger }

type wfConfigureNodeArgs struct {
	NodeTag string          `json:"node_tag"`
	Config  json.RawMessage `json:"config"`
}

func (t *wfCanvasConfigureNodeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_configure_node",
		Desc: "用语义化 JSON 配置节点内部表单,由前端翻译成真实画布 schema。优先使用它,不要猜内部表单 path。" +
			" 常用字段: title; input/inputs 绑定上游变量,形如 {from:'start',output:'input',name:'input'};" +
			" prompt/user_prompt/system_prompt 配置 LLM; bind_plugins/bind_workflows 给 LLM 节点绑 FC 工具让模型按需调用,如 bind_plugins:[{plugin_id,api_id,api_name,plugin_version}]; outputs 声明输出变量,如 [{name:'answer',type:'string'}];" +
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
	t.ledger.record(turnCanvasLedgerOp{Op: "configure_node", NodeTag: a.NodeTag, Detail: wfTruncate(string(a.Config), 160)})
	return wfAck("configure_node", a), nil
}

// ---- workflow_canvas_set_node_params ----

type wfCanvasSetParamsTool struct{ ledger *turnCanvasLedger }

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
	t.ledger.record(turnCanvasLedgerOp{Op: "set_node_params", NodeTag: a.NodeTag, Detail: wfTruncate(string(a.Params), 160)})
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
