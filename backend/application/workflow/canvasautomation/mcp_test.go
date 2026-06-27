package canvasautomation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

type surfacePayload struct {
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	Mode            string   `json:"mode"`
	Adapter         string   `json:"adapter"`
	CommandProtocol string   `json:"command_protocol"`
	RequiresBrowser bool     `json:"requires_browser"`
	Tools           []string `json:"tools"`
	BackendAPIs     []string `json:"backend_apis"`
	Gaps            []string `json:"gaps"`
}

type resourceCatalogPayload struct {
	Family                  string   `json:"family"`
	Name                    string   `json:"name"`
	Status                  string   `json:"status"`
	AppliesToNodeTypes      []string `json:"applies_to_node_types"`
	DiscoveryTool           string   `json:"discovery_tool"`
	BackendAPIs             []string `json:"backend_apis"`
	RequiredBeforeConfigure bool     `json:"required_before_configure"`
	DoNotFabricateIDs       bool     `json:"do_not_fabricate_ids"`
	BindingGuidance         []string `json:"binding_guidance"`
	Gaps                    []string `json:"gaps"`
}

type operationGuidePayload struct {
	TaskType      string                    `json:"task_type"`
	RequiredTools []string                  `json:"required_tools"`
	Steps         []operationGuideStep      `json:"steps"`
	Guardrails    []string                  `json:"guardrails"`
	Checklists    []operationGuideChecklist `json:"checklists"`
}

type operationGuideStep struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Tools       []string `json:"tools"`
	Description string   `json:"description"`
}

type operationGuideChecklist struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Items []string `json:"items"`
}

func TestWorkflowMCPServerRegistersSystemTools(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})
	tools := srv.ListTools()

	for _, name := range []string{
		"workflow.get_operation_guide",
		"workflow.list_surfaces",
		"workflow.list_resource_catalog",
		"workflow.list_node_capabilities",
		"workflow.node_smoke_manifest",
		"workflow.node_smoke_coverage",
		"workflow.run_node_smoke",
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
	} {
		if tools[name] == nil {
			t.Fatalf("expected MCP tool %s to be registered; got %#v", name, tools)
		}
	}
}

func TestWorkflowMCPNodeSmokeCoverageReportsEveryNodeAndBoundaries(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.node_smoke_coverage"), nil)
	if result.IsError {
		t.Fatalf("node_smoke_coverage should succeed, got error result: %+v", result)
	}

	var payload struct {
		Status   string                  `json:"status"`
		Coverage NodeSmokeCoverageReport `json:"coverage"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" {
		t.Fatalf("unexpected node smoke coverage status: %+v", payload)
	}
	if payload.Coverage.TotalNodes != 41 || payload.Coverage.ManifestNodes != 41 {
		t.Fatalf("coverage should include every known node: %+v", payload.Coverage)
	}
	if len(payload.Coverage.MissingManifestTypes) != 0 || len(payload.Coverage.MissingSpecTypes) != 0 {
		t.Fatalf("coverage should not miss manifest/spec entries: %+v", payload.Coverage)
	}
	if !containsString(payload.Coverage.ExecutableTypes, "32") || !containsString(payload.Coverage.ReadonlyTypes, "1") {
		t.Fatalf("coverage should classify executable and readonly nodes: %+v", payload.Coverage)
	}
	if !containsString(payload.Coverage.ResourceFixtureTypes, "4") || !containsString(payload.Coverage.ResourceFixtureTypes, "61") {
		t.Fatalf("coverage should expose resource fixture nodes: %+v", payload.Coverage)
	}
	if !containsString(payload.Coverage.SubCanvasOrPartialTypes, "21") || !containsString(payload.Coverage.SubCanvasOrPartialTypes, "28") {
		t.Fatalf("coverage should expose sub-canvas/partial nodes: %+v", payload.Coverage)
	}

	merge := findNodeSmokeCoverageItem(t, payload.Coverage.Nodes, "32")
	if !merge.HasSpec || !merge.HasAssertions || !merge.HasExpectedBindableVariables || merge.Mode != "execute" {
		t.Fatalf("coverage should prove variable merge has spec/assertions/bindable expectations: %+v", merge)
	}
	if !containsString(merge.ExpectedBindableVariables, "merge.output") ||
		!containsString(merge.Assertions, "变量聚合必须先调用 workflow.get_bindable_variables,merge_groups.variables 全部来自可绑定变量。") {
		t.Fatalf("coverage should include concrete variable merge assertions and bindable variables: %+v", merge)
	}
	if !containsString(merge.RequiredCommands, "workflow.get_bindable_variables") ||
		!containsString(merge.RequiredCommands, "workflow.configure_node") ||
		!containsString(merge.PlannedCommands, "workflow.auto_layout") ||
		!containsString(merge.CleanupCommands, "workflow.delete_node") ||
		!containsString(merge.TemporaryNodeTags, "smoke_32_variable_merge") {
		t.Fatalf("coverage should include concrete variable merge smoke command plan: %+v", merge)
	}

	output := findNodeSmokeCoverageItem(t, payload.Coverage.Nodes, "13")
	if !output.HasSpec || !output.HasAssertions || output.CanBeDownstreamSource {
		t.Fatalf("coverage should mark output node as display-only, not downstream source: %+v", output)
	}
	if !containsString(output.Assertions, "display-only") {
		t.Fatalf("coverage should include concrete output display-only assertion: %+v", output)
	}

	http := findNodeSmokeCoverageItem(t, payload.Coverage.Nodes, "45")
	if http.SupportLevel != SupportPartial || !containsString(http.Gaps, "缺少 method/url/headers/query/body/outputs 语义配置器。") {
		t.Fatalf("coverage should preserve partial node gaps: %+v", http)
	}
}

func TestWorkflowMCPOperationGuideEnforcesProgressiveEditingProtocol(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.get_operation_guide"), map[string]any{"task_type": "build_workflow"})
	if result.IsError {
		t.Fatalf("get_operation_guide should succeed, got error result: %+v", result)
	}

	var payload struct {
		Status string                `json:"status"`
		Guide  operationGuidePayload `json:"guide"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || payload.Guide.TaskType != "build_workflow" {
		t.Fatalf("unexpected operation guide payload: %+v", payload)
	}
	for _, tool := range []string{
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
		"workflow.auto_layout",
		"workflow.test_run",
		"workflow.explain_failure",
	} {
		if !containsString(payload.Guide.RequiredTools, tool) {
			t.Fatalf("operation guide should require %s: %+v", tool, payload.Guide.RequiredTools)
		}
	}

	if stepIndexWithTool(payload.Guide.Steps, "workflow.list_surfaces") >= stepIndexWithTool(payload.Guide.Steps, "workflow.list_node_capabilities") {
		t.Fatalf("surface discovery must happen before node planning: %+v", payload.Guide.Steps)
	}
	if stepIndexWithTool(payload.Guide.Steps, "workflow.node_smoke_coverage") >= stepIndexWithTool(payload.Guide.Steps, "workflow.get_node_spec") {
		t.Fatalf("node smoke coverage must happen before per-node specs: %+v", payload.Guide.Steps)
	}
	if stepIndexWithTool(payload.Guide.Steps, "workflow.get_bindable_variables") >= stepIndexWithTool(payload.Guide.Steps, "workflow.configure_node") {
		t.Fatalf("bindable variables must be read before configure_node: %+v", payload.Guide.Steps)
	}

	guardrails := strings.Join(payload.Guide.Guardrails, "\n")
	for _, want := range []string{
		"dispatched_to_canvas 不是成功",
		"不要默认 clear_canvas",
		"不能新增 Start/End",
		"type=13",
		"workflow.get_bindable_variables",
		"每组 add/connect/configure 后调用 workflow.auto_layout",
	} {
		if !strings.Contains(guardrails, want) {
			t.Fatalf("operation guide guardrails should mention %q, got %+v", want, payload.Guide.Guardrails)
		}
	}

	bindingAudit := findOperationGuideChecklist(t, payload.Guide.Checklists, "binding_audit")
	for _, want := range []string{"Start 输入", "每个节点 input/inputs", "IF true/false", "变量聚合", "End 返回文本"} {
		if !strings.Contains(strings.Join(bindingAudit.Items, "\n"), want) {
			t.Fatalf("binding audit checklist should mention %q: %+v", want, bindingAudit)
		}
	}

	failureRepair := findOperationGuideChecklist(t, payload.Guide.Checklists, "failure_repair")
	for _, want := range []string{"workflow.get_canvas_context", "workflow.get_bindable_variables", "workflow.explain_failure", "局部修复", "workflow.test_run"} {
		if !strings.Contains(strings.Join(failureRepair.Items, "\n"), want) {
			t.Fatalf("failure repair checklist should mention %q: %+v", want, failureRepair)
		}
	}
}

func TestWorkflowMCPListSurfacesAdvertisesWorkflowAndChatflow(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.list_surfaces"), nil)
	if result.IsError {
		t.Fatalf("list_surfaces should succeed, got error result: %+v", result)
	}

	var payload struct {
		Status   string           `json:"status"`
		Surfaces []surfacePayload `json:"surfaces"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" {
		t.Fatalf("unexpected list_surfaces status: %+v", payload)
	}

	workflowCanvas := findSurfacePayload(t, payload.Surfaces, "workflow.canvas")
	if workflowCanvas.Status != "implemented" || workflowCanvas.Mode != "workflow" || !workflowCanvas.RequiresBrowser {
		t.Fatalf("workflow canvas surface should describe browser-live workflow editing: %+v", workflowCanvas)
	}
	if workflowCanvas.CommandProtocol != "canvas_automation.v0" ||
		!containsString(workflowCanvas.Tools, "workflow.get_canvas_context") ||
		!containsString(workflowCanvas.Tools, "workflow.get_bindable_variables") ||
		!containsString(workflowCanvas.Tools, "workflow.configure_node") ||
		!containsString(workflowCanvas.Tools, "workflow.node_smoke_coverage") ||
		!containsString(workflowCanvas.Tools, "workflow.test_run") {
		t.Fatalf("workflow canvas should expose the shared canvas command protocol and tools: %+v", workflowCanvas)
	}

	chatflowCanvas := findSurfacePayload(t, payload.Surfaces, "chatflow.canvas")
	if chatflowCanvas.Status != "implemented" || chatflowCanvas.Mode != "chatflow" || chatflowCanvas.CommandProtocol != workflowCanvas.CommandProtocol {
		t.Fatalf("chatflow canvas should reuse the workflow canvas automation protocol: %+v", chatflowCanvas)
	}
	if !containsString(chatflowCanvas.Tools, "workflow.run_node_smoke") || !containsString(chatflowCanvas.Gaps, "chatflow 专属测试运行仍需独立后端 MCP 工具。") {
		t.Fatalf("chatflow canvas should declare shared smoke coverage and remaining chatflow gaps: %+v", chatflowCanvas)
	}

	roleSettings := findSurfacePayload(t, payload.Surfaces, "chatflow.role_settings")
	if roleSettings.Status != "planned-backend-api" || roleSettings.RequiresBrowser {
		t.Fatalf("role settings should be a backend-api surface plan: %+v", roleSettings)
	}
	if !containsString(roleSettings.BackendAPIs, "GetChatFlowRole") || !containsString(roleSettings.BackendAPIs, "CreateChatFlowRole") {
		t.Fatalf("role settings should point at existing chatflow role APIs: %+v", roleSettings)
	}

	conversationTemplates := findSurfacePayload(t, payload.Surfaces, "chatflow.conversation_templates")
	if conversationTemplates.Status != "planned-backend-api" ||
		!containsString(conversationTemplates.BackendAPIs, "ListApplicationConversationDef") ||
		!containsString(conversationTemplates.BackendAPIs, "CreateApplicationConversationDef") {
		t.Fatalf("conversation template surface should point at existing application conversation APIs: %+v", conversationTemplates)
	}

	resourceCatalog := findSurfacePayload(t, payload.Surfaces, "space.resource_catalog")
	if resourceCatalog.Status != "implemented-contract" || resourceCatalog.RequiresBrowser {
		t.Fatalf("resource catalog should expose a backend contract without browser dependency: %+v", resourceCatalog)
	}
	if !containsString(resourceCatalog.Tools, "workflow.list_resource_catalog") {
		t.Fatalf("resource catalog surface should point at the MCP catalog tool: %+v", resourceCatalog)
	}
}

func TestWorkflowMCPListResourceCatalogGuidesResourceBoundNodes(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.list_resource_catalog"), nil)
	if result.IsError {
		t.Fatalf("list_resource_catalog should succeed, got error result: %+v", result)
	}

	var payload struct {
		Status       string                   `json:"status"`
		Catalog      []resourceCatalogPayload `json:"catalog"`
		UsageOrder   []string                 `json:"usage_order"`
		CoverageGaps []string                 `json:"coverage_gaps"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || len(payload.Catalog) < 8 {
		t.Fatalf("unexpected resource catalog payload: %+v", payload)
	}
	for _, step := range []string{"workflow.get_operation_guide", "workflow.list_surfaces", "workflow.list_resource_catalog", "workflow.get_node_spec", "workflow.get_bindable_variables"} {
		if !containsString(payload.UsageOrder, step) {
			t.Fatalf("resource catalog should explain progressive usage order %q: %+v", step, payload.UsageOrder)
		}
	}

	pluginAPI := findResourceCatalogPayload(t, payload.Catalog, "plugin_api")
	if pluginAPI.Status != "contract-ready" || !pluginAPI.RequiredBeforeConfigure || !pluginAPI.DoNotFabricateIDs {
		t.Fatalf("plugin/API resources must require discovery and prohibit fabricated IDs: %+v", pluginAPI)
	}
	if !containsString(pluginAPI.AppliesToNodeTypes, "4") || !containsString(pluginAPI.BackendAPIs, "GetPlaygroundPluginList") {
		t.Fatalf("plugin/API catalog should cover plugin node and existing workflow API: %+v", pluginAPI)
	}
	if !strings.Contains(strings.Join(pluginAPI.BindingGuidance, "\n"), "schema") {
		t.Fatalf("plugin/API catalog should guide schema-based parameter binding: %+v", pluginAPI)
	}

	knowledge := findResourceCatalogPayload(t, payload.Catalog, "knowledge_base")
	if !containsString(knowledge.AppliesToNodeTypes, "6") || !containsString(knowledge.AppliesToNodeTypes, "27") {
		t.Fatalf("knowledge catalog should cover retrieval/write nodes: %+v", knowledge)
	}

	mcpServer := findResourceCatalogPayload(t, payload.Catalog, "mcp_server")
	if !containsString(mcpServer.AppliesToNodeTypes, "61") || len(mcpServer.Gaps) == 0 {
		t.Fatalf("MCP resource catalog should cover type=61 and declare live-listing gap: %+v", mcpServer)
	}

	for _, cap := range ListNodeCapabilities() {
		if !cap.RequiresResource && cap.RuntimeSmoke != "resource" {
			continue
		}
		if !resourceCatalogCoversNodeType(payload.Catalog, cap.Type) {
			t.Fatalf("resource-bound node type %s (%s) should be covered by resource catalog", cap.Type, cap.Name)
		}
	}
}

func TestWorkflowMCPListResourceCatalogFiltersByNodeType(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.list_resource_catalog"), map[string]any{"node_type": "4"})
	if result.IsError {
		t.Fatalf("list_resource_catalog node_type filter should succeed, got error result: %+v", result)
	}

	var payload struct {
		Status  string                   `json:"status"`
		Catalog []resourceCatalogPayload `json:"catalog"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || len(payload.Catalog) == 0 {
		t.Fatalf("unexpected filtered resource catalog payload: %+v", payload)
	}
	for _, item := range payload.Catalog {
		if !containsString(item.AppliesToNodeTypes, "4") {
			t.Fatalf("node_type filter returned unrelated catalog item: %+v", item)
		}
	}
}

func TestWorkflowMCPRunNodeSmokeDispatchesToBrowserRunner(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		CommandDispatcher:   relay,
		CommandResultWaiter: relay,
	})

	resultCh := make(chan *mcp.CallToolResult, 1)
	go func() {
		resultCh <- callWorkflowMCPTool(t, srv.GetTool("workflow.run_node_smoke"), map[string]any{
			"workflow_id":              "wf-1",
			"space_id":                 "space-1",
			"node_type":                "32",
			"allow_temporary_workflow": true,
			"include_skipped":          false,
			"timeout_ms":               1000,
		})
	}()

	items := waitForQueuedWorkflowCommand(t, relay, "space-1", "wf-1")
	if items[0].Command.Op != "run_node_smoke" || items[0].Command.Target != "32" || !items[0].RequiresResponse {
		t.Fatalf("expected browser-live node smoke request, got %+v", items[0])
	}
	if items[0].Command.Args["allow_temporary_workflow"] != true || items[0].Command.Args["include_skipped"] != false {
		t.Fatalf("expected node smoke args to be forwarded, got %+v", items[0].Command.Args)
	}
	if err := relay.SubmitWorkflowCommandResult(context.Background(), WorkflowCanvasCommandResult{
		RequestID: items[0].RequestID,
		Status:    "ok",
		Results: []WorkflowCanvasCommandResultItem{{
			Op:      "run_node_smoke",
			OK:      true,
			Target:  "32",
			Message: "node smoke completed",
		}},
		NodeSmokeReport: map[string]any{
			"total": float64(1),
			"nodes": []any{map[string]any{
				"nodeType":        "32",
				"executionStatus": "executed",
			}},
		},
	}); err != nil {
		t.Fatalf("submit browser result failed: %v", err)
	}

	result := <-resultCh
	if result.IsError {
		t.Fatalf("run_node_smoke should return browser execution result, got error result: %+v", result)
	}
	var payload struct {
		Status string                      `json:"status"`
		Mode   string                      `json:"mode"`
		Result WorkflowCanvasCommandResult `json:"result"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || payload.Mode != "browser_live" || payload.Result.NodeSmokeReport == nil {
		t.Fatalf("unexpected node smoke payload: %+v", payload)
	}
}

func TestWorkflowMCPNodeSmokeManifest(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.node_smoke_manifest"), nil)
	if result.IsError {
		t.Fatalf("node_smoke_manifest should succeed, got error result: %+v", result)
	}

	var payload struct {
		Status   string            `json:"status"`
		Manifest NodeSmokeManifest `json:"manifest"`
	}
	unmarshalToolText(t, result, &payload)

	if payload.Status != "ok" || payload.Manifest.Total < 41 {
		t.Fatalf("unexpected manifest payload: %+v", payload)
	}
	if payload.Manifest.Summary.NotReady != 0 {
		t.Fatalf("local full nodes should not be reported as not-ready: %+v", payload.Manifest.Summary)
	}

	merge := findSmokeManifestNode(t, payload.Manifest, "32")
	if merge.Mode != "execute" || merge.Isolation != "temporary-workflow" || !merge.RequiresTemporaryWorkflowOptIn {
		t.Fatalf("variable merge should be a temporary executable smoke plan: %+v", merge)
	}
	if !containsString(merge.PlannedCommands, "workflow.get_bindable_variables") || len(merge.TemporaryNodeTags) < 3 {
		t.Fatalf("variable merge should include bindable-variable read and temporary upstream nodes: %+v", merge)
	}
	if !containsString(merge.Assertions, "不要聚合 type=13 输出节点;固定文案分支先用 type=15 文本处理产出真实 output。") ||
		!containsString(merge.ExpectedBindableVariables, "merge.output") {
		t.Fatalf("variable merge should expose binding assertions and expected output: %+v", merge)
	}

	end := findSmokeManifestNode(t, payload.Manifest, "2")
	if !containsString(end.Assertions, "End 可以返回文本") || !containsString(end.Assertions, "streaming_output=true") {
		t.Fatalf("end node should expose text-return and streaming assertions: %+v", end)
	}

	output := findSmokeManifestNode(t, payload.Manifest, "13")
	if !containsString(output.Assertions, "display-only") {
		t.Fatalf("output node should expose display-only assertion: %+v", output)
	}

	plugin := findSmokeManifestNode(t, payload.Manifest, "4")
	if plugin.Mode != "skip" || plugin.Ready || !plugin.RequiresResourceFixture || plugin.SkipReason == "" {
		t.Fatalf("resource-bound plugin node should be explicit skipped work: %+v", plugin)
	}
}

func TestWorkflowMCPToolsReturnCatalogAndSpecsAsJSON(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	capResult := callWorkflowMCPTool(t, srv.GetTool("workflow.list_node_capabilities"), nil)
	var capPayload struct {
		Status       string           `json:"status"`
		Capabilities []NodeCapability `json:"capabilities"`
	}
	unmarshalToolText(t, capResult, &capPayload)
	if capPayload.Status != "ok" || len(capPayload.Capabilities) < 41 {
		t.Fatalf("unexpected capabilities payload: %+v", capPayload)
	}

	specResult := callWorkflowMCPTool(t, srv.GetTool("workflow.get_node_spec"), map[string]any{"type": "32"})
	var specPayload struct {
		Status string   `json:"status"`
		Spec   NodeSpec `json:"spec"`
	}
	unmarshalToolText(t, specResult, &specPayload)
	if specPayload.Spec.Type != "32" || !strings.Contains(specPayload.Spec.Description, "workflow.get_bindable_variables") {
		t.Fatalf("unexpected spec payload: %+v", specPayload)
	}
}

func findSmokeManifestNode(t *testing.T, manifest NodeSmokeManifest, nodeType string) NodeSmokeManifestNode {
	t.Helper()
	for _, node := range manifest.Nodes {
		if node.Type == nodeType {
			return node
		}
	}
	t.Fatalf("node type %s not found in manifest", nodeType)
	return NodeSmokeManifestNode{}
}

func findNodeSmokeCoverageItem(t *testing.T, nodes []NodeSmokeCoverageItem, nodeType string) NodeSmokeCoverageItem {
	t.Helper()
	for _, node := range nodes {
		if node.Type == nodeType {
			return node
		}
	}
	t.Fatalf("node type %s not found in coverage", nodeType)
	return NodeSmokeCoverageItem{}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func findSurfacePayload(t *testing.T, surfaces []surfacePayload, id string) surfacePayload {
	t.Helper()
	for _, surface := range surfaces {
		if surface.ID == id {
			return surface
		}
	}
	t.Fatalf("surface %s not found in %+v", id, surfaces)
	return surfacePayload{}
}

func findResourceCatalogPayload(t *testing.T, catalog []resourceCatalogPayload, family string) resourceCatalogPayload {
	t.Helper()
	for _, item := range catalog {
		if item.Family == family {
			return item
		}
	}
	t.Fatalf("resource family %s not found in %+v", family, catalog)
	return resourceCatalogPayload{}
}

func resourceCatalogCoversNodeType(catalog []resourceCatalogPayload, nodeType string) bool {
	for _, item := range catalog {
		if containsString(item.AppliesToNodeTypes, nodeType) {
			return true
		}
	}
	return false
}

func stepIndexWithTool(steps []operationGuideStep, tool string) int {
	for i, step := range steps {
		if containsString(step.Tools, tool) {
			return i
		}
	}
	return len(steps)
}

func findOperationGuideChecklist(t *testing.T, checklists []operationGuideChecklist, id string) operationGuideChecklist {
	t.Helper()
	for _, checklist := range checklists {
		if checklist.ID == id {
			return checklist
		}
	}
	t.Fatalf("operation guide checklist %s not found in %+v", id, checklists)
	return operationGuideChecklist{}
}

func TestWorkflowMCPMutationToolReturnsBrowserDispatchAck(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})

	cases := []struct {
		name string
		tool string
		op   string
		args map[string]any
	}{
		{
			name: "add node",
			tool: "workflow.add_node",
			op:   "add_node",
			args: map[string]any{
				"node_tag": "reply_text",
				"type":     "15",
				"title":    "回复文本",
				"x":        120,
				"y":        240,
			},
		},
		{
			name: "connect",
			tool: "workflow.connect",
			op:   "connect",
			args: map[string]any{
				"from":      "start",
				"to":        "reply_text",
				"from_port": "default",
			},
		},
		{
			name: "configure node",
			tool: "workflow.configure_node",
			op:   "configure_node",
			args: map[string]any{
				"node_tag": "reply_text",
				"config": map[string]any{
					"content": "ok",
					"outputs": []any{map[string]any{"name": "output", "type": "string"}},
				},
			},
		},
		{
			name: "auto layout",
			tool: "workflow.auto_layout",
			op:   "auto_layout",
			args: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := callWorkflowMCPTool(t, srv.GetTool(tc.tool), tc.args)
			if result.IsError {
				t.Fatalf("%s should return a dispatch ack, got error result: %+v", tc.tool, result)
			}

			var payload struct {
				Status string         `json:"status"`
				Op     string         `json:"op"`
				Args   map[string]any `json:"args"`
			}
			unmarshalToolText(t, result, &payload)

			if payload.Status != "dispatched_to_canvas" || payload.Op != tc.op {
				t.Fatalf("unexpected dispatch payload: %+v", payload)
			}
		})
	}
}

func TestWorkflowMCPMutationToolDispatchesToConfiguredRelay(t *testing.T) {
	var seen WorkflowCanvasDispatchCommand
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		CommandDispatcher: WorkflowCanvasCommandDispatchFunc(func(_ context.Context, command WorkflowCanvasDispatchCommand) error {
			seen = command
			return nil
		}),
	})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.configure_node"), map[string]any{
		"workflow_id": "wf-1",
		"space_id":    "space-1",
		"node_tag":    "end",
		"config": map[string]any{
			"content":          "结果: {{answer}}",
			"streaming_output": true,
		},
	})
	if result.IsError {
		t.Fatalf("configure_node should dispatch to relay, got error result: %+v", result)
	}

	if seen.WorkflowID != "wf-1" || seen.SpaceID != "space-1" {
		t.Fatalf("dispatcher did not receive workflow identity: %+v", seen)
	}
	if seen.Command.Op != "configure_node" || seen.Command.Target != "end" {
		t.Fatalf("dispatcher did not receive expected command: %+v", seen.Command)
	}
	if seen.Command.Args["config"] == nil {
		t.Fatalf("dispatcher command should include config args: %+v", seen.Command.Args)
	}
}

func TestWorkflowMCPMutationToolWaitsForBrowserResultWhenWaiterConfigured(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		CommandDispatcher:   relay,
		CommandResultWaiter: relay,
	})

	resultCh := make(chan *mcp.CallToolResult, 1)
	go func() {
		resultCh <- callWorkflowMCPTool(t, srv.GetTool("workflow.delete_node"), map[string]any{
			"workflow_id": "wf-1",
			"space_id":    "space-1",
			"node_tag":    "probe",
			"timeout_ms":  1000,
		})
	}()

	items := waitForQueuedWorkflowCommand(t, relay, "space-1", "wf-1")
	if items[0].Command.Op != "delete_node" || !items[0].RequiresResponse || items[0].RequestID == "" {
		t.Fatalf("expected browser-live delete request, got %+v", items[0])
	}
	if err := relay.SubmitWorkflowCommandResult(context.Background(), WorkflowCanvasCommandResult{
		RequestID: items[0].RequestID,
		Status:    "ok",
		Results: []WorkflowCanvasCommandResultItem{{
			Op:     "delete_node",
			OK:     true,
			NodeID: "node-probe",
			Target: "probe",
		}},
	}); err != nil {
		t.Fatalf("submit browser result failed: %v", err)
	}

	result := <-resultCh
	if result.IsError {
		t.Fatalf("delete_node should return browser execution result, got error result: %+v", result)
	}
	var payload struct {
		Status string                      `json:"status"`
		Mode   string                      `json:"mode"`
		Result WorkflowCanvasCommandResult `json:"result"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || payload.Mode != "browser_live" || payload.Result.Results[0].NodeID != "node-probe" {
		t.Fatalf("unexpected browser execution payload: %+v", payload)
	}
}

func TestWorkflowMCPMutationToolRequiresWorkflowIdentityWhenRelayConfigured(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		CommandDispatcher: WorkflowCanvasCommandDispatchFunc(func(context.Context, WorkflowCanvasDispatchCommand) error {
			t.Fatal("dispatcher should not be called without workflow identity")
			return nil
		}),
	})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.add_node"), map[string]any{
		"node_tag": "reply_text",
		"type":     "15",
	})

	if !result.IsError {
		t.Fatalf("missing workflow identity should be a tool error result")
	}
	var payload struct {
		Status string `json:"status"`
		Code   string `json:"code"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "error" || payload.Code != "workflow_identity_required" {
		t.Fatalf("unexpected missing workflow identity payload: %+v", payload)
	}
}

func TestWorkflowMCPGetCanvasContextWaitsForBrowserLiveResult(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		CommandDispatcher:   relay,
		CommandResultWaiter: relay,
	})

	resultCh := make(chan *mcp.CallToolResult, 1)
	go func() {
		resultCh <- callWorkflowMCPTool(t, srv.GetTool("workflow.get_canvas_context"), map[string]any{
			"workflow_id": "wf-1",
			"space_id":    "space-1",
			"timeout_ms":  1000,
		})
	}()

	items := waitForQueuedWorkflowCommand(t, relay, "space-1", "wf-1")
	if items[0].Command.Op != "get_canvas_context" || !items[0].RequiresResponse || items[0].RequestID == "" {
		t.Fatalf("expected browser-live context request, got %+v", items[0])
	}
	if err := relay.SubmitWorkflowCommandResult(context.Background(), WorkflowCanvasCommandResult{
		RequestID:     items[0].RequestID,
		Status:        "ok",
		CanvasContext: "节点: Start -> Text -> End\n可绑定变量: text.output",
	}); err != nil {
		t.Fatalf("submit browser result failed: %v", err)
	}

	result := <-resultCh
	if result.IsError {
		t.Fatalf("get_canvas_context should return browser result, got error result: %+v", result)
	}
	var payload struct {
		Status        string `json:"status"`
		Mode          string `json:"mode"`
		CanvasContext string `json:"canvas_context"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || payload.Mode != "browser_live" || !strings.Contains(payload.CanvasContext, "text.output") {
		t.Fatalf("unexpected canvas context payload: %+v", payload)
	}
}

func TestWorkflowMCPGetBindableVariablesWaitsForBrowserLiveResult(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		CommandDispatcher:   relay,
		CommandResultWaiter: relay,
	})

	resultCh := make(chan *mcp.CallToolResult, 1)
	go func() {
		resultCh <- callWorkflowMCPTool(t, srv.GetTool("workflow.get_bindable_variables"), map[string]any{
			"workflow_id": "wf-1",
			"space_id":    "space-1",
			"target_node": "end",
			"timeout_ms":  1000,
		})
	}()

	items := waitForQueuedWorkflowCommand(t, relay, "space-1", "wf-1")
	if items[0].Command.Op != "get_bindable_variables" || items[0].Command.Target != "end" || !items[0].RequiresResponse {
		t.Fatalf("expected browser-live bindable variables request, got %+v", items[0])
	}
	if err := relay.SubmitWorkflowCommandResult(context.Background(), WorkflowCanvasCommandResult{
		RequestID:         items[0].RequestID,
		Status:            "ok",
		BindableVariables: "start.input\ntext.output",
	}); err != nil {
		t.Fatalf("submit browser result failed: %v", err)
	}

	result := <-resultCh
	if result.IsError {
		t.Fatalf("get_bindable_variables should return browser result, got error result: %+v", result)
	}
	var payload struct {
		Status            string `json:"status"`
		Mode              string `json:"mode"`
		BindableVariables string `json:"bindable_variables"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "ok" || payload.Mode != "browser_live" || !strings.Contains(payload.BindableVariables, "text.output") {
		t.Fatalf("unexpected bindable variables payload: %+v", payload)
	}
}

func TestWorkflowMCPAddNodeRejectsStartAndEndSingletons(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})
	result := callWorkflowMCPTool(t, srv.GetTool("workflow.add_node"), map[string]any{
		"node_tag": "end2",
		"type":     "2",
	})

	if !result.IsError {
		t.Fatalf("adding End through MCP should be rejected")
	}
	var payload struct {
		Status string `json:"status"`
		Code   string `json:"code"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "error" || payload.Code != "singleton_node" {
		t.Fatalf("unexpected singleton error payload: %+v", payload)
	}
}

func TestWorkflowMCPTestRunRequiresWorkflowIdentity(t *testing.T) {
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{})
	result := callWorkflowMCPTool(t, srv.GetTool("workflow.test_run"), map[string]any{
		"workflow_id": "",
		"space_id":    "space-1",
	})

	if !result.IsError {
		t.Fatalf("missing workflow_id should be a tool error result")
	}
	var payload struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "error" || !strings.Contains(payload.Error, "workflow_id") {
		t.Fatalf("unexpected missing identity payload: %+v", payload)
	}
}

func TestWorkflowMCPTestRunCallsConfiguredRunner(t *testing.T) {
	var seen WorkflowTestRunRequest
	srv := NewWorkflowMCPServer(WorkflowMCPOptions{
		TestRunner: WorkflowTestRunFunc(func(_ context.Context, req WorkflowTestRunRequest) (any, error) {
			seen = req
			return map[string]any{"status": "success", "execute_id": "exe-1"}, nil
		}),
	})

	result := callWorkflowMCPTool(t, srv.GetTool("workflow.test_run"), map[string]any{
		"workflow_id": "wf-1",
		"space_id":    "space-1",
		"input":       map[string]any{"input": "hello"},
	})
	if result.IsError {
		t.Fatalf("test_run should succeed, got error result: %+v", result)
	}
	if seen.WorkflowID != "wf-1" || seen.SpaceID != "space-1" || seen.Input["input"] != "hello" {
		t.Fatalf("runner did not receive expected request: %+v", seen)
	}

	var payload struct {
		Status string         `json:"status"`
		Result map[string]any `json:"result"`
	}
	unmarshalToolText(t, result, &payload)
	if payload.Status != "workflow_test_result" || payload.Result["execute_id"] != "exe-1" {
		t.Fatalf("unexpected test result payload: %+v", payload)
	}
}

func callWorkflowMCPTool(t *testing.T, tool *mcpserver.ServerTool, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	if tool == nil {
		t.Fatal("tool is nil")
	}
	result, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: args},
	})
	if err != nil {
		t.Fatalf("tool call failed: %v", err)
	}
	return result
}

func waitForQueuedWorkflowCommand(t *testing.T, relay *MemoryWorkflowCommandRelay, spaceID, workflowID string) []WorkflowCanvasQueuedCommand {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		items := relay.PollWorkflowCommands(spaceID, workflowID, 0, 10)
		if len(items) > 0 {
			return items
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for queued workflow command")
	return nil
}

func unmarshalToolText(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatalf("empty tool result: %+v", result)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	if err := json.Unmarshal([]byte(text.Text), target); err != nil {
		t.Fatalf("invalid JSON tool result %q: %v", text.Text, err)
	}
}
