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
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"

	modelworkflow "github.com/ynet-dev/ynet-studio/backend/api/model/workflow"
)

func TestWorkflowCanvasAddNodeRejectsStartAndEnd(t *testing.T) {
	ctx := context.Background()
	addNode := &wfCanvasAddNodeTool{}

	for _, tt := range []struct {
		name string
		args string
	}{
		{name: "start", args: `{"node_tag":"start2","type":"1","title":"Start"}`},
		{name: "end", args: `{"node_tag":"end2","type":"2","title":"End"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := addNode.InvokableRun(ctx, tt.args)
			if err != nil {
				t.Fatalf("InvokableRun returned error: %v", err)
			}
			if !strings.Contains(out, "singleton") && !strings.Contains(out, "单例") {
				t.Fatalf("expected singleton rejection, got %q", out)
			}
			if strings.Contains(out, `"status":"dispatched_to_canvas"`) {
				t.Fatalf("singleton node must not dispatch to canvas, got %q", out)
			}
		})
	}
}

func TestWorkflowCanvasAddNodeAllowsOrdinaryNode(t *testing.T) {
	out, err := (&wfCanvasAddNodeTool{}).InvokableRun(
		context.Background(),
		`{"node_tag":"llm","type":"3","title":"LLM"}`,
	)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	if !strings.Contains(out, `"status":"dispatched_to_canvas"`) {
		t.Fatalf("expected canvas dispatch ack, got %q", out)
	}
}

func TestWorkflowCanvasAckWarnsItIsNotValidationSuccess(t *testing.T) {
	out, err := (&wfCanvasTestRunTool{}).InvokableRun(
		context.Background(),
		`{"input":{"input":"hello"}}`,
	)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"不是执行成功", "不能宣称完成", "当前校验错误为空", "发送前快照"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ack should warn about unverified dispatch %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetContextReturnsNodeCapabilityAudit(t *testing.T) {
	out, err := (&wfCanvasGetContextTool{
		ext: map[string]string{
			workflowCanvasNodeCapabilityExtKey: "节点能力覆盖: total=41, full=8",
			workflowCanvasContextExtKey:        "当前画布摘要",
		},
	}).InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"node_capabilities", "节点能力覆盖", "当前画布摘要"} {
		if !strings.Contains(out, want) {
			t.Fatalf("context should include %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasTestRunReturnsBackendResultWhenAvailable(t *testing.T) {
	tool := &wfCanvasTestRunTool{
		ext: map[string]string{
			workflowCanvasWorkflowIDExtKey: "7654449287564099584",
			workflowCanvasSpaceIDExtKey:    "7652614054615187456",
		},
		backendTestRun: func(ctx context.Context, req workflowCanvasBackendTestRunRequest) (*workflowCanvasBackendTestRunResponse, error) {
			if got := req.Input["input"]; got != "hello" {
				t.Fatalf("expected input to be forwarded, got %q", got)
			}
			return &workflowCanvasBackendTestRunResponse{
				Data: &workflowCanvasBackendTestRunData{
					WorkflowID:        req.WorkflowID,
					ExecuteID:         "exe_1",
					Status:            "failed",
					ExecuteStatus:     modelworkflow.WorkflowExeStatus_Fail,
					ExecuteStatusText: "Fail",
					FailedNodes: []*workflowCanvasBackendTestRunNode{
						{
							NodeID:     "node_bad",
							NodeName:   "输出业务名称",
							NodeStatus: modelworkflow.NodeExeStatus_Fail,
							StatusText: "Fail",
							ErrorInfo:  "引用变量不存在",
						},
					},
				},
			}, nil
		},
	}

	out, err := tool.InvokableRun(context.Background(), `{"input":{"input":"hello"}}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{`"status":"workflow_test_result"`, `"status":"failed"`, "输出业务名称", "引用变量不存在"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected backend result to contain %q, got %q", want, out)
		}
	}
	if strings.Contains(out, `"status":"dispatched_to_canvas"`) {
		t.Fatalf("backend result must not look like a frontend-only ack, got %q", out)
	}
}

func TestWorkflowCanvasAddNodeToolInfoDoesNotAdvertiseStartEndAsAddable(t *testing.T) {
	info, err := (&wfCanvasAddNodeTool{}).Info(context.Background())
	if err != nil {
		t.Fatalf("Info returned error: %v", err)
	}
	if strings.Contains(info.Desc, "'1'=开始") || strings.Contains(info.Desc, "'2'=结束") {
		t.Fatalf("tool description must not advertise start/end as addable: %s", info.Desc)
	}
	if !strings.Contains(info.Desc, "100001") || !strings.Contains(info.Desc, "900001") {
		t.Fatalf("tool description should mention existing start/end ids: %s", info.Desc)
	}
}

func TestWorkflowCanvasConfigureNodeDispatchesSemanticConfig(t *testing.T) {
	out, err := (&wfCanvasConfigureNodeTool{}).InvokableRun(
		context.Background(),
		`{"node_tag":"llm","config":{"input":{"from":"start","output":"input","name":"input"},"prompt":"请处理 {{input}}","outputs":[{"name":"answer","type":"string"}]}}`,
	)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{`"op":"configure_node"`, `"node_tag":"llm"`, `"prompt"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected configure ack to contain %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetContextToolPointsToInjectedCanvasContext(t *testing.T) {
	out, err := (&wfCanvasGetContextTool{ext: map[string]string{
		workflowCanvasContextExtKey:         "节点A outputs: answer:string",
		workflowCanvasNodeCapabilityExtKey:  "节点能力覆盖: full=8",
		workflowCanvasBindableVarsExtKey:    "可绑定变量: node.answer:string",
		workflowCanvasResourceSummaryExtKey: "知识库: dataset_id=kb1",
		workflowCanvasTestRunRequiredExtKey: "true",
		workflowCanvasTestRunInputExtKey:    `{"input":"hello"}`,
	}}).InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"节点A", "answer:string", "节点能力覆盖", "可绑定变量", "dataset_id=kb1", `"test_run_required":"true"`, "hello", "发送前快照"} {
		if !strings.Contains(out, want) {
			t.Fatalf("context tool should return injected context %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasReadOnlyToolsReturnCapabilitiesAndBindableVariables(t *testing.T) {
	ctx := context.Background()
	capabilityOut, err := (&wfCanvasGetNodeCapabilityAuditTool{ext: map[string]string{
		workflowCanvasNodeCapabilityExtKey: "节点能力覆盖: total=41, full=8, partial=22",
	}}).InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("capability tool returned error: %v", err)
	}
	for _, want := range []string{"node_capability_audit", "节点能力覆盖", "partial/add-only"} {
		if !strings.Contains(capabilityOut, want) {
			t.Fatalf("capability tool should mention %q, got %q", want, capabilityOut)
		}
	}

	bindableOut, err := (&wfCanvasGetBindableVariablesTool{ext: map[string]string{
		workflowCanvasBindableVarsExtKey: "可绑定变量:\n- 120001.output:string (文本处理,type=15)",
		workflowCanvasContextExtKey:      "画布摘要",
	}}).InvokableRun(ctx, `{}`)
	if err != nil {
		t.Fatalf("bindable tool returned error: %v", err)
	}
	for _, want := range []string{"bindable_variables", "120001.output", "type=15", "画布摘要"} {
		if !strings.Contains(bindableOut, want) {
			t.Fatalf("bindable tool should mention %q, got %q", want, bindableOut)
		}
	}
}

func TestWorkflowCanvasGetOperationGuideReturnsProgressiveProtocol(t *testing.T) {
	out, err := requireSuperAgentWorkflowCanvasTool(t, "workflow_canvas_get_operation_guide").InvokableRun(context.Background(), `{"task_type":"build_workflow"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{
		`"status":"operation_guide"`,
		`"task_type":"build_workflow"`,
		`"workflow_canvas_get_node_catalog"`,
		`"workflow_canvas_get_node_smoke_coverage"`,
		`"workflow_canvas_get_resource_catalog"`,
		`"workflow_canvas_get_bindable_variables"`,
		`"workflow_canvas_configure_node"`,
		`"workflow_canvas_auto_layout"`,
		`"workflow_canvas_test_run"`,
		"dispatched_to_canvas 不是成功",
		"不要默认 clear_canvas",
		"type=13",
		"failure_repair",
		"binding_audit",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("operation guide should mention %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetNodeSmokeCoverageReportsEveryNodeAndBoundaries(t *testing.T) {
	out, err := requireSuperAgentWorkflowCanvasTool(t, "workflow_canvas_get_node_smoke_coverage").InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{
		`"status":"node_smoke_coverage"`,
		`"total_nodes":41`,
		`"manifest_nodes":41`,
		`"missing_manifest_types":[]`,
		`"missing_spec_types":[]`,
		`"type":"32"`,
		`"assertions":["变量聚合必须先调用 workflow_canvas_get_bindable_variables`,
		`"expected_bindable_variables":["merge.output"]`,
		`"required_tools":["workflow_canvas_add_node","workflow_canvas_connect","workflow_canvas_get_bindable_variables","workflow_canvas_configure_node"`,
		`"planned_tools":["workflow_canvas_add_node","workflow_canvas_connect","workflow_canvas_get_bindable_variables","workflow_canvas_configure_node"`,
		`"workflow_canvas_auto_layout"`,
		`"cleanup_tools":["workflow_canvas_delete_node"`,
		`"temporary_node_tags":["smoke_32_variable_merge","smoke_32_text_primary","smoke_32_text_fallback"]`,
		`"has_expected_bindable_variables":true`,
		`"can_be_downstream_source":true`,
		`"merge.output"`,
		`"type":"13"`,
		`"can_be_downstream_source":false`,
		"display-only",
		`"type":"45"`,
		`"support_level":"partial"`,
		"method/url/headers/query/body/outputs",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("coverage should mention %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetResourceCatalogReturnsResourceFamilies(t *testing.T) {
	out, err := requireSuperAgentWorkflowCanvasTool(t, "workflow_canvas_get_resource_catalog").InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{
		`"status":"resource_catalog"`,
		`"catalog_total":12`,
		`"plugin_api"`,
		`"knowledge_base"`,
		`"mcp_server"`,
		`"http_endpoint"`,
		`"do_not_fabricate_ids":true`,
		`"workflow_canvas_get_operation_guide"`,
		`"workflow_canvas_get_resource_catalog"`,
		`"workflow_canvas_get_node_smoke_coverage"`,
		`"workflow_canvas_get_bindable_variables"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("resource catalog should mention %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetResourceCatalogFiltersByNodeType(t *testing.T) {
	out, err := requireSuperAgentWorkflowCanvasTool(t, "workflow_canvas_get_resource_catalog").InvokableRun(context.Background(), `{"node_type":"4"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{
		`"catalog_total":1`,
		`"family":"plugin_api"`,
		`"applies_to_node_types":["3","4"]`,
		`"required_before_configure":true`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("filtered resource catalog should mention %q, got %q", want, out)
		}
	}
	for _, unwanted := range []string{`"family":"knowledge_base"`, `"family":"mcp_server"`, `"family":"http_endpoint"`} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("node_type=4 catalog should not include %q, got %q", unwanted, out)
		}
	}
}

func TestWorkflowCanvasGetNodeSmokeManifestReturnsUnifiedReadinessPlan(t *testing.T) {
	out, err := (&wfCanvasGetNodeSmokeManifestTool{}).InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{
		`"status":"node_smoke_manifest"`,
		`"total":41`,
		`"type":"32"`,
		`"name":"变量聚合"`,
		`"workflow_canvas_get_bindable_variables"`,
		`"temporary_node_tags":["smoke_32_variable_merge","smoke_32_text_primary","smoke_32_text_fallback"]`,
		`"requires_temporary_workflow_opt_in":true`,
		`"requires_resource_fixture":true`,
		`"type":"4"`,
		`"assertions"`,
		`"expected_bindable_variables":["merge.output"]`,
		`"End 可以返回文本"`,
		`"streaming_output=true"`,
		`"display-only"`,
		"不要聚合 type=13",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("node smoke manifest should contain %q, got %q", want, out)
		}
	}
}

func TestWorkflowCanvasPromptRequiresPostMutationBindingAudit(t *testing.T) {
	for _, want := range []string{"每次完成一组", "workflow_canvas_get_canvas_context", "从 Start 到 End", "输入绑定", "End returns", "发送前快照"} {
		if !strings.Contains(SuperAgentWorkflowCanvasPrompt, want) {
			t.Fatalf("workflow canvas prompt should require post-mutation binding audit %q, got %q", want, SuperAgentWorkflowCanvasPrompt)
		}
	}
}

func TestWorkflowCanvasPromptRequiresInputToOutputConfigurationSequence(t *testing.T) {
	for _, want := range []string{
		"先 workflow_canvas_add_node 创建节点,再 workflow_canvas_connect 连线,然后 workflow_canvas_configure_node 逐个配置",
		"从输入到输出逐节点确认",
		"配置 condition/merge_groups/returns 前必须先调用 workflow_canvas_get_bindable_variables",
		"固定文案没有变量引用时可以删除默认 input",
		"End 支持返回变量和返回文本",
		"streaming_output=true",
	} {
		if !strings.Contains(SuperAgentWorkflowCanvasPrompt, want) {
			t.Fatalf("workflow canvas prompt should require configuration sequence %q, got %q", want, SuperAgentWorkflowCanvasPrompt)
		}
	}
}

func TestWorkflowCanvasPromptForbidsClearingCanvasAsDefaultErrorFix(t *testing.T) {
	for _, want := range []string{"禁止默认 clear_canvas", "试运行失败", "局部修复", "输入绑定"} {
		if !strings.Contains(SuperAgentWorkflowCanvasPrompt, want) {
			t.Fatalf("workflow canvas prompt should forbid blind clear-canvas fixes %q, got %q", want, SuperAgentWorkflowCanvasPrompt)
		}
	}
}

func TestWorkflowCanvasGetNodeCatalogReturnsProgressiveDesignGuide(t *testing.T) {
	out, err := (&wfCanvasGetNodeCatalogTool{ext: map[string]string{
		workflowCanvasNodeCatalogExtKey: "- 77 自定义节点: 当前前端动态节点",
	}}).InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"大模型", "代码", "条件分支", "输出", "输入", "知识库", "智能体", "get_node_spec", "自定义节点"} {
		if !strings.Contains(out, want) {
			t.Fatalf("node catalog should mention %q; got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetNodeSpecReturnsBindableParamsAndExample(t *testing.T) {
	out, err := (&wfCanvasGetNodeSpecTool{}).InvokableRun(context.Background(), `{"type":"3"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"input", "prompt", "outputs", "configure_node", "answer"} {
		if !strings.Contains(out, want) {
			t.Fatalf("LLM node spec should mention %q; got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetOutputNodeSpecDescribesPureOutputNode(t *testing.T) {
	out, err := (&wfCanvasGetNodeSpecTool{}).InvokableRun(context.Background(), `{"type":"13"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"输出", "纯输出", "inputParameters", "content", "streaming_output", "configure_node", "不要让变量聚合", "type=15 文本处理"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Output node spec should mention %q; got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetVariableMergeSpecRejectsOutputNodeAsMergeSource(t *testing.T) {
	out, err := (&wfCanvasGetNodeSpecTool{}).InvokableRun(context.Background(), `{"type":"32"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"不要聚合 type=13", "type=15 文本处理", "真实 output 变量"} {
		if !strings.Contains(out, want) {
			t.Fatalf("VariableMerge spec should mention %q; got %q", want, out)
		}
	}
}

func TestWorkflowCanvasGetCodeNodeSpecUsesRuntimePythonArgs(t *testing.T) {
	out, err := (&wfCanvasGetNodeSpecTool{}).InvokableRun(context.Background(), `{"type":"5"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"async def main(args: Args)", "args.params", "params.get", "不要对 args 直接调用 strip", "language\":\"python"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Code node spec should mention %q; got %q", want, out)
		}
	}
	for _, forbidden := range []string{"language(通常 javascript)", `"language":"javascript"`, "async function main"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Code node spec must not mention %q; got %q", forbidden, out)
		}
	}
}

func TestWorkflowCanvasGetNodeSpecCoversEverySmokeManifestNode(t *testing.T) {
	specTool := &wfCanvasGetNodeSpecTool{}
	for _, cap := range wfCanvasSmokeCapabilities {
		out, err := specTool.InvokableRun(context.Background(), `{"type":"`+cap.Type+`"}`)
		if err != nil {
			t.Fatalf("type %s InvokableRun returned error: %v", cap.Type, err)
		}
		if strings.Contains(out, "未找到该节点的详细规格") {
			t.Fatalf("type %s should expose a concrete or bounded spec, got %q", cap.Type, out)
		}
		for _, want := range []string{"节点规格", cap.Name} {
			if !strings.Contains(out, want) {
				t.Fatalf("type %s spec should mention %q, got %q", cap.Type, want, out)
			}
		}
	}
}

func TestWorkflowCanvasGetNodeSpecFallbackExplainsSupportBoundaries(t *testing.T) {
	specTool := &wfCanvasGetNodeSpecTool{}
	cases := []struct {
		name string
		args string
		want []string
	}{
		{
			name: "singleton start",
			args: `{"type":"1"}`,
			want: []string{"节点规格", "开始", "内置单例", "不能新增", "workflow_canvas_get_canvas_context"},
		},
		{
			name: "partial http",
			args: `{"type":"45"}`,
			want: []string{"HTTP", "method", "url", "outputs", "语义配置器"},
		},
		{
			name: "add only variable",
			args: `{"type":"11"}`,
			want: []string{"变量", "只可靠支持添加", "workflow_canvas_add_node", "workflow_canvas_delete_node"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := specTool.InvokableRun(context.Background(), tt.args)
			if err != nil {
				t.Fatalf("InvokableRun returned error: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Fatalf("node spec should mention %q; got %q", want, out)
				}
			}
		})
	}
}

func TestWorkflowCanvasDeleteAndClearToolsDispatch(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		tool toolRunner
		args string
		want string
	}{
		{name: "delete node", tool: &wfCanvasDeleteNodeTool{}, args: `{"node_tag":"old_llm"}`, want: `"op":"delete_node"`},
		{name: "delete line", tool: &wfCanvasDeleteLineTool{}, args: `{"from":"old_llm","to":"end"}`, want: `"op":"delete_line"`},
		{name: "clear canvas", tool: &wfCanvasClearCanvasTool{}, args: `{}`, want: `"op":"clear_canvas"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.tool.InvokableRun(ctx, tt.args)
			if err != nil {
				t.Fatalf("InvokableRun returned error: %v", err)
			}
			if !strings.Contains(out, tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, out)
			}
		})
	}
}

type toolRunner interface {
	InvokableRun(context.Context, string, ...tool.Option) (string, error)
}

func requireSuperAgentWorkflowCanvasTool(t *testing.T, name string) tool.InvokableTool {
	t.Helper()
	for _, tl := range newSuperAgentExtensionTools(superAgentToolDeps{}) {
		info, err := tl.Info(context.Background())
		if err != nil {
			t.Fatalf("Info returned error: %v", err)
		}
		if info.Name == name {
			return tl
		}
	}
	t.Fatalf("%s not registered", name)
	return nil
}

func TestWorkflowCanvasGetNodeSpecDescribesVariableMergeForBranchJoin(t *testing.T) {
	out, err := (&wfCanvasGetNodeSpecTool{}).InvokableRun(context.Background(), `{"type":"32"}`)
	if err != nil {
		t.Fatalf("InvokableRun returned error: %v", err)
	}
	for _, want := range []string{"变量聚合", "merge_groups", "variables", "End", "output"} {
		if !strings.Contains(out, want) {
			t.Fatalf("VariableMerge node spec should mention %q; got %q", want, out)
		}
	}
}

func TestSuperAgentWorkflowCanvasPromptRequiresContextAndConfigureNode(t *testing.T) {
	for _, want := range []string{
		"workflow_canvas_get_operation_guide",
		"workflow_canvas_get_node_catalog",
		"workflow_canvas_get_node_smoke_manifest",
		"workflow_canvas_get_node_smoke_coverage",
		"workflow_canvas_get_node_capability_audit",
		"workflow_canvas_get_resource_catalog",
		"workflow_canvas_get_node_spec",
		"workflow_canvas_get_canvas_context",
		"workflow_canvas_get_bindable_variables",
		"workflow_canvas_configure_node",
		"Before each workflow_canvas_* tool call",
		"input bindings",
		"outputs",
		"绑定诊断",
	} {
		if !strings.Contains(SuperAgentWorkflowCanvasPrompt, want) {
			t.Fatalf("workflow canvas prompt should mention %q; got %q", want, SuperAgentWorkflowCanvasPrompt)
		}
	}
}
