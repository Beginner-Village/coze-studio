package canvasautomation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type WorkflowMCPOptions struct {
	TestRunner          WorkflowTestRunner
	CommandDispatcher   WorkflowCanvasCommandDispatcher
	CommandResultWaiter WorkflowCanvasCommandResultWaiter
}

type WorkflowCanvasCommand struct {
	Op     string         `json:"op"`
	Target string         `json:"target,omitempty"`
	Args   map[string]any `json:"args,omitempty"`
}

type WorkflowCanvasDispatchCommand struct {
	WorkflowID       string                `json:"workflow_id"`
	SpaceID          string                `json:"space_id"`
	RequestID        string                `json:"request_id,omitempty"`
	RequiresResponse bool                  `json:"requires_response,omitempty"`
	Command          WorkflowCanvasCommand `json:"command"`
}

type WorkflowCanvasCommandDispatcher interface {
	DispatchWorkflowCommand(context.Context, WorkflowCanvasDispatchCommand) error
}

type WorkflowCanvasCommandResultWaiter interface {
	WaitWorkflowCommandResult(context.Context, string) (WorkflowCanvasCommandResult, bool)
}

type WorkflowCanvasCommandDispatchFunc func(context.Context, WorkflowCanvasDispatchCommand) error

func (fn WorkflowCanvasCommandDispatchFunc) DispatchWorkflowCommand(ctx context.Context, command WorkflowCanvasDispatchCommand) error {
	return fn(ctx, command)
}

type WorkflowTestRunRequest struct {
	WorkflowID string            `json:"workflow_id"`
	SpaceID    string            `json:"space_id"`
	Input      map[string]string `json:"input,omitempty"`
	BotID      string            `json:"bot_id,omitempty"`
	ProjectID  string            `json:"project_id,omitempty"`
	CommitID   string            `json:"commit_id,omitempty"`
	TimeoutMs  int               `json:"timeout_ms,omitempty"`
	IntervalMs int               `json:"interval_ms,omitempty"`
}

type WorkflowTestRunner interface {
	RunWorkflowTest(context.Context, WorkflowTestRunRequest) (any, error)
}

type WorkflowTestRunFunc func(context.Context, WorkflowTestRunRequest) (any, error)

func (fn WorkflowTestRunFunc) RunWorkflowTest(ctx context.Context, req WorkflowTestRunRequest) (any, error) {
	return fn(ctx, req)
}

func NewWorkflowMCPServer(opts WorkflowMCPOptions) *server.MCPServer {
	s := server.NewMCPServer("ynet-workflow-canvas", "1.0.0", server.WithToolCapabilities(true))
	registerWorkflowMCPTools(s, opts)
	return s
}

func registerWorkflowMCPTools(s *server.MCPServer, opts WorkflowMCPOptions) {
	s.AddTool(
		mcp.NewTool("workflow.get_operation_guide",
			mcp.WithDescription("Return the progressive workflow editing protocol that agents must follow before mutating canvas nodes, bindings, resources, layout, or test repair."),
			mcp.WithString("task_type", mcp.Description("optional guide type, such as build_workflow, fix_failure, or node_smoke")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonToolResult(false, map[string]any{
				"status": "ok",
				"guide":  GetOperationGuide(req.GetString("task_type", "")),
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.list_surfaces",
			mcp.WithDescription("List editable workflow-related surfaces exposed by the system MCP service, including workflow canvas, chatflow canvas, chatflow settings, and resource catalogs."),
		),
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonToolResult(false, map[string]any{
				"status":   "ok",
				"surfaces": ListSurfaceCapabilities(),
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.list_resource_catalog",
			mcp.WithDescription("List workflow resource families required by resource-bound nodes, including plugin/API, knowledge, agent, model, database, card, MCP, image, trigger, and memory resources. This returns the discovery contract and known backend APIs; live resource instances require follow-up tools."),
			mcp.WithString("node_type", mcp.Description("optional workflow node type filter, such as 4, 6, 61, or 100")),
			mcp.WithString("family", mcp.Description("optional resource family filter, such as plugin_api or knowledge_base")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filter := ResourceCatalogFilter{
				NodeType: strings.TrimSpace(req.GetString("node_type", "")),
				Family:   strings.TrimSpace(req.GetString("family", "")),
			}
			return jsonToolResult(false, map[string]any{
				"status":        "ok",
				"catalog":       ListResourceCatalog(filter),
				"usage_order":   ResourceCatalogUsageOrder(),
				"coverage_gaps": ResourceCatalogCoverageGaps(),
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.list_node_capabilities",
			mcp.WithDescription("List workflow canvas node capabilities, support levels, resource requirements, and current gaps."),
		),
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonToolResult(false, map[string]any{
				"status":       "ok",
				"capabilities": ListNodeCapabilities(),
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.node_smoke_manifest",
			mcp.WithDescription("Return the unified workflow-node smoke readiness manifest used by internal agents, external MCP clients, Codex, and visual smoke runners."),
		),
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonToolResult(false, map[string]any{
				"status":   "ok",
				"manifest": BuildNodeSmokeManifest(),
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.node_smoke_coverage",
			mcp.WithDescription("Return a machine-readable coverage report for every workflow node type: manifest/spec coverage, executable vs skipped/resource/sub-canvas status, assertions, and binding expectations."),
		),
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonToolResult(false, map[string]any{
				"status":   "ok",
				"coverage": BuildNodeSmokeCoverageReport(),
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.run_node_smoke",
			mcp.WithDescription("Ask the opened browser workflow canvas to execute node smoke plan(s) through the visual command runner and return the smoke report. Use a temporary workflow and set allow_temporary_workflow=true for executable node plans."),
			mcp.WithString("workflow_id", mcp.Required(), mcp.Description("workflow id for the opened browser canvas")),
			mcp.WithString("space_id", mcp.Required(), mcp.Description("space id")),
			mcp.WithString("node_type", mcp.Description("optional node type to smoke, such as 32. Omit to run the suite plan.")),
			mcp.WithBoolean("allow_temporary_workflow", mcp.Description("explicit opt-in that this request is running against an isolated temporary workflow")),
			mcp.WithBoolean("include_skipped", mcp.Description("include skipped/resource-bound nodes in the returned report")),
			mcp.WithNumber("timeout_ms", mcp.Description("optional browser response timeout in milliseconds")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeType := strings.TrimSpace(req.GetString("node_type", ""))
			args := map[string]any{
				"allow_temporary_workflow": req.GetBool("allow_temporary_workflow", false),
				"include_skipped":          req.GetBool("include_skipped", true),
			}
			if nodeType != "" {
				args["node_type"] = nodeType
			}
			return dispatchToolResult(ctx, opts, req, "run_node_smoke", nodeType, args)
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.get_node_spec",
			mcp.WithDescription("Get the progressive configuration and binding spec for one workflow node type."),
			mcp.WithString("type", mcp.Required(), mcp.Description("workflow node type, such as 15 or 32")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeType := strings.TrimSpace(req.GetString("type", ""))
			spec, ok := GetNodeSpec(nodeType)
			if !ok {
				return jsonToolResult(true, errorPayload("node_type_not_found", fmt.Sprintf("node type %q not found", nodeType)))
			}
			return jsonToolResult(false, map[string]any{
				"status": "ok",
				"spec":   spec,
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.get_canvas_context",
			mcp.WithDescription("Read live workflow canvas context from the opened browser canvas, including nodes, edges, outputs, and validation diagnostics."),
			mcp.WithString("workflow_id", mcp.Required(), mcp.Description("workflow id")),
			mcp.WithString("space_id", mcp.Required(), mcp.Description("space id")),
			mcp.WithNumber("timeout_ms", mcp.Description("optional browser response timeout in milliseconds")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := browserLiveReadResult(ctx, opts, req, "get_canvas_context", "", nil)
			if err != nil {
				return jsonToolResult(true, errorPayload("browser_live_context_failed", err.Error()))
			}
			return jsonToolResult(false, map[string]any{
				"status":         "ok",
				"mode":           "browser_live",
				"canvas_context": result.CanvasContext,
				"diagnostics":    result.BindingDiagnostics,
				"result":         result,
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.get_bindable_variables",
			mcp.WithDescription("Read live bindable variables from the opened workflow canvas before configuring IF, VariableMerge, End returns, or prompt bindings."),
			mcp.WithString("workflow_id", mcp.Required(), mcp.Description("workflow id")),
			mcp.WithString("space_id", mcp.Required(), mcp.Description("space id")),
			mcp.WithString("target_node", mcp.Description("optional target node tag or id")),
			mcp.WithNumber("timeout_ms", mcp.Description("optional browser response timeout in milliseconds")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			target := strings.TrimSpace(req.GetString("target_node", ""))
			args := map[string]any{}
			if target != "" {
				args["target_node"] = target
			}
			result, err := browserLiveReadResult(ctx, opts, req, "get_bindable_variables", target, args)
			if err != nil {
				return jsonToolResult(true, errorPayload("browser_live_bindable_variables_failed", err.Error()))
			}
			return jsonToolResult(false, map[string]any{
				"status":             "ok",
				"mode":               "browser_live",
				"bindable_variables": result.BindableVariables,
				"diagnostics":        result.BindingDiagnostics,
				"result":             result,
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.add_node",
			mcp.WithDescription("Dispatch a live browser-canvas command to add one workflow node. Start/End are built-in singletons and cannot be added."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
			mcp.WithString("node_tag", mcp.Required(), mcp.Description("stable logical node tag used by later tools")),
			mcp.WithString("type", mcp.Required(), mcp.Description("workflow node type, such as 3, 8, 15, 32, 100")),
			mcp.WithString("title", mcp.Description("optional node title")),
			mcp.WithNumber("x", mcp.Description("optional canvas x position")),
			mcp.WithNumber("y", mcp.Description("optional canvas y position")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeTag := strings.TrimSpace(req.GetString("node_tag", ""))
			nodeType := strings.TrimSpace(req.GetString("type", ""))
			if nodeTag == "" {
				return jsonToolResult(true, errorPayload("invalid_request", "node_tag is required"))
			}
			if nodeType == "" {
				return jsonToolResult(true, errorPayload("invalid_request", "type is required"))
			}
			if nodeType == "1" || nodeType == "2" {
				return jsonToolResult(true, errorPayload("singleton_node", "Start and End are built-in singleton nodes; connect/configure existing start/end instead of adding them"))
			}

			args := map[string]any{
				"node_tag": nodeTag,
				"type":     nodeType,
			}
			if title := strings.TrimSpace(req.GetString("title", "")); title != "" {
				args["title"] = title
			}
			copyNumberArg(args, req.GetArguments(), "x")
			copyNumberArg(args, req.GetArguments(), "y")
			return dispatchToolResult(ctx, opts, req, "add_node", nodeTag, args)
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.connect",
			mcp.WithDescription("Dispatch a live browser-canvas command to connect two nodes. Use node_tag or start/end singleton refs."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
			mcp.WithString("from", mcp.Required(), mcp.Description("source node tag/id, or start")),
			mcp.WithString("to", mcp.Required(), mcp.Description("target node tag/id, or end")),
			mcp.WithString("from_port", mcp.Description("optional source port, such as true/false for IF")),
			mcp.WithString("to_port", mcp.Description("optional target port")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, err := requiredStringArgs(req, "from", "to")
			if err != nil {
				return jsonToolResult(true, errorPayload("invalid_request", err.Error()))
			}
			copyStringArg(args, req, "from_port")
			copyStringArg(args, req, "to_port")
			return dispatchToolResult(ctx, opts, req, "connect", "", args)
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.configure_node",
			mcp.WithDescription("Dispatch a live browser-canvas semantic configuration command for one node. Use input/inputs, outputs, returns, prompt, code, condition, merge_groups, content, streaming_output."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
			mcp.WithString("node_tag", mcp.Required(), mcp.Description("node tag/id to configure; End can be end")),
			mcp.WithObject("config", mcp.Required(), mcp.Description("semantic node config")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeTag := strings.TrimSpace(req.GetString("node_tag", ""))
			config := objectArg(req.GetArguments(), "config")
			if nodeTag == "" {
				return jsonToolResult(true, errorPayload("invalid_request", "node_tag is required"))
			}
			if len(config) == 0 {
				return jsonToolResult(true, errorPayload("invalid_request", "config is required"))
			}
			return dispatchToolResult(ctx, opts, req, "configure_node", nodeTag, map[string]any{
				"node_tag": nodeTag,
				"config":   config,
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.set_node_params",
			mcp.WithDescription("Expert fallback: dispatch direct form-path params to one live browser-canvas node. Prefer workflow.configure_node first."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
			mcp.WithString("node_tag", mcp.Required(), mcp.Description("node tag/id to configure")),
			mcp.WithObject("params", mcp.Required(), mcp.Description("direct node form params")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeTag := strings.TrimSpace(req.GetString("node_tag", ""))
			params := objectArg(req.GetArguments(), "params")
			if nodeTag == "" {
				return jsonToolResult(true, errorPayload("invalid_request", "node_tag is required"))
			}
			if len(params) == 0 {
				return jsonToolResult(true, errorPayload("invalid_request", "params is required"))
			}
			return dispatchToolResult(ctx, opts, req, "set_node_params", nodeTag, map[string]any{
				"node_tag": nodeTag,
				"params":   params,
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.delete_node",
			mcp.WithDescription("Dispatch a live browser-canvas command to delete one ordinary node. Start/End cannot be deleted."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
			mcp.WithString("node_tag", mcp.Required(), mcp.Description("node tag/id to delete")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			nodeTag := strings.TrimSpace(req.GetString("node_tag", ""))
			if nodeTag == "" {
				return jsonToolResult(true, errorPayload("invalid_request", "node_tag is required"))
			}
			return dispatchToolResult(ctx, opts, req, "delete_node", nodeTag, map[string]any{"node_tag": nodeTag})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.delete_line",
			mcp.WithDescription("Dispatch a live browser-canvas command to delete one edge."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
			mcp.WithString("from", mcp.Required(), mcp.Description("source node tag/id, or start")),
			mcp.WithString("to", mcp.Required(), mcp.Description("target node tag/id, or end")),
			mcp.WithString("from_port", mcp.Description("optional source port")),
			mcp.WithString("to_port", mcp.Description("optional target port")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args, err := requiredStringArgs(req, "from", "to")
			if err != nil {
				return jsonToolResult(true, errorPayload("invalid_request", err.Error()))
			}
			copyStringArg(args, req, "from_port")
			copyStringArg(args, req, "to_port")
			return dispatchToolResult(ctx, opts, req, "delete_line", "", args)
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.clear_canvas",
			mcp.WithDescription("Dispatch a live browser-canvas command to delete all ordinary nodes and lines while keeping Start/End. Use only for explicit rebuilds."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return dispatchToolResult(ctx, opts, req, "clear_canvas", "", map[string]any{})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.auto_layout",
			mcp.WithDescription("Dispatch a live browser-canvas command to optimize layout after mutations."),
			mcp.WithString("workflow_id", mcp.Description("workflow id, required when routing to an opened browser canvas")),
			mcp.WithString("space_id", mcp.Description("space id, required when routing to an opened browser canvas")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return dispatchToolResult(ctx, opts, req, "auto_layout", "", map[string]any{})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.test_run",
			mcp.WithDescription("Run a workflow test through the system backend and return compact node-level results."),
			mcp.WithString("workflow_id", mcp.Required(), mcp.Description("workflow id")),
			mcp.WithString("space_id", mcp.Required(), mcp.Description("space id")),
			mcp.WithObject("input", mcp.Description("test input map")),
			mcp.WithString("bot_id", mcp.Description("optional bot id")),
			mcp.WithString("project_id", mcp.Description("optional project id")),
			mcp.WithString("commit_id", mcp.Description("optional workflow commit id")),
			mcp.WithNumber("timeout_ms", mcp.Description("optional timeout in milliseconds")),
			mcp.WithNumber("interval_ms", mcp.Description("optional polling interval in milliseconds")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			testReq := workflowTestRunRequestFromMCP(req)
			if err := validateWorkflowTestRunRequest(testReq); err != nil {
				return jsonToolResult(true, errorPayload("invalid_request", err.Error()))
			}
			if opts.TestRunner == nil {
				return jsonToolResult(true, errorPayload("test_runner_not_configured", "workflow test runner is not configured"))
			}
			result, err := opts.TestRunner.RunWorkflowTest(ctx, testReq)
			if err != nil {
				return jsonToolResult(true, errorPayload("workflow_test_run_failed", err.Error()))
			}
			return jsonToolResult(false, map[string]any{
				"status": "workflow_test_result",
				"result": result,
			})
		},
	)

	s.AddTool(
		mcp.NewTool("workflow.explain_failure",
			mcp.WithDescription("Explain a workflow validation or test_run failure and suggest local fixes."),
			mcp.WithString("error", mcp.Description("error message or failed node output")),
			mcp.WithString("node_type", mcp.Description("optional failed node type")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonToolResult(false, map[string]any{
				"status": "ok",
				"advice": explainWorkflowFailure(req.GetString("error", ""), req.GetString("node_type", "")),
			})
		},
	)
}

func dispatchToolResult(ctx context.Context, opts WorkflowMCPOptions, req mcp.CallToolRequest, op string, target string, args map[string]any) (*mcp.CallToolResult, error) {
	if opts.CommandDispatcher != nil {
		workflowID := strings.TrimSpace(req.GetString("workflow_id", ""))
		spaceID := strings.TrimSpace(req.GetString("space_id", ""))
		if workflowID == "" || spaceID == "" {
			return jsonToolResult(true, errorPayload("workflow_identity_required", "workflow_id and space_id are required when routing MCP commands to an opened browser canvas"))
		}
		requestID := ""
		requiresResponse := opts.CommandResultWaiter != nil
		if requiresResponse {
			requestID = newWorkflowMCPRequestID(op)
		}
		if err := opts.CommandDispatcher.DispatchWorkflowCommand(ctx, WorkflowCanvasDispatchCommand{
			WorkflowID:       workflowID,
			SpaceID:          spaceID,
			RequestID:        requestID,
			RequiresResponse: requiresResponse,
			Command: WorkflowCanvasCommand{
				Op:     op,
				Target: target,
				Args:   args,
			},
		}); err != nil {
			return jsonToolResult(true, errorPayload("workflow_command_dispatch_failed", err.Error()))
		}
		if requiresResponse {
			result, err := waitForBrowserWorkflowCommandResult(ctx, opts, req, requestID, op)
			if err != nil {
				return jsonToolResult(true, errorPayload("workflow_command_result_timeout", err.Error()))
			}
			return jsonToolResult(result.Status != "ok", map[string]any{
				"status": result.Status,
				"mode":   "browser_live",
				"op":     op,
				"args":   args,
				"result": result,
			})
		}
	}
	return jsonToolResult(false, dispatchPayload(op, args))
}

func browserLiveReadResult(ctx context.Context, opts WorkflowMCPOptions, req mcp.CallToolRequest, op string, target string, args map[string]any) (WorkflowCanvasCommandResult, error) {
	if opts.CommandDispatcher == nil || opts.CommandResultWaiter == nil {
		return WorkflowCanvasCommandResult{}, errors.New("browser-live command dispatcher/result waiter is not configured")
	}
	workflowID := strings.TrimSpace(req.GetString("workflow_id", ""))
	spaceID := strings.TrimSpace(req.GetString("space_id", ""))
	if workflowID == "" || spaceID == "" {
		return WorkflowCanvasCommandResult{}, errors.New("workflow_id and space_id are required")
	}

	requestID := newWorkflowMCPRequestID(op)
	if err := opts.CommandDispatcher.DispatchWorkflowCommand(ctx, WorkflowCanvasDispatchCommand{
		WorkflowID:       workflowID,
		SpaceID:          spaceID,
		RequestID:        requestID,
		RequiresResponse: true,
		Command: WorkflowCanvasCommand{
			Op:     op,
			Target: target,
			Args:   args,
		},
	}); err != nil {
		return WorkflowCanvasCommandResult{}, err
	}

	result, ok := waitForWorkflowCommandResult(ctx, opts, req, requestID)
	if !ok {
		return WorkflowCanvasCommandResult{}, fmt.Errorf("timed out waiting for opened browser canvas to return %s", op)
	}
	if result.Status == "failed" {
		return result, fmt.Errorf("browser canvas returned failed result for %s", op)
	}
	return result, nil
}

func waitForBrowserWorkflowCommandResult(ctx context.Context, opts WorkflowMCPOptions, req mcp.CallToolRequest, requestID string, op string) (WorkflowCanvasCommandResult, error) {
	result, ok := waitForWorkflowCommandResult(ctx, opts, req, requestID)
	if !ok {
		return WorkflowCanvasCommandResult{}, fmt.Errorf("timed out waiting for opened browser canvas to execute %s", op)
	}
	return result, nil
}

func waitForWorkflowCommandResult(ctx context.Context, opts WorkflowMCPOptions, req mcp.CallToolRequest, requestID string) (WorkflowCanvasCommandResult, bool) {
	timeoutMs := intFromAny(req.GetArguments()["timeout_ms"])
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	return opts.CommandResultWaiter.WaitWorkflowCommandResult(waitCtx, requestID)
}

func newWorkflowMCPRequestID(op string) string {
	return fmt.Sprintf("workflow-mcp-%s-%d", strings.ReplaceAll(op, "_", "-"), time.Now().UnixNano())
}

func dispatchPayload(op string, args map[string]any) map[string]any {
	return map[string]any{
		"status": "dispatched_to_canvas",
		"op":     op,
		"args":   args,
		"note": "已下发到当前浏览器画布执行;这不是执行成功、不是配置正确、也不是测试通过。" +
			"后续必须读取画布上下文/可绑定变量并运行后端 workflow.test_run 确认真正结果。",
	}
}

func requiredStringArgs(req mcp.CallToolRequest, names ...string) (map[string]any, error) {
	out := make(map[string]any, len(names))
	for _, name := range names {
		value := strings.TrimSpace(req.GetString(name, ""))
		if value == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
		out[name] = value
	}
	return out, nil
}

func copyStringArg(out map[string]any, req mcp.CallToolRequest, name string) {
	if value := strings.TrimSpace(req.GetString(name, "")); value != "" {
		out[name] = value
	}
}

func copyNumberArg(out map[string]any, args map[string]any, name string) {
	switch value := args[name].(type) {
	case int:
		out[name] = value
	case int64:
		out[name] = value
	case float64:
		out[name] = value
	case json.Number:
		out[name] = value
	}
}

func objectArg(args map[string]any, name string) map[string]any {
	value, ok := args[name].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func workflowTestRunRequestFromMCP(req mcp.CallToolRequest) WorkflowTestRunRequest {
	args := req.GetArguments()
	return WorkflowTestRunRequest{
		WorkflowID: strings.TrimSpace(req.GetString("workflow_id", "")),
		SpaceID:    strings.TrimSpace(req.GetString("space_id", "")),
		Input:      stringMap(args["input"]),
		BotID:      strings.TrimSpace(req.GetString("bot_id", "")),
		ProjectID:  strings.TrimSpace(req.GetString("project_id", "")),
		CommitID:   strings.TrimSpace(req.GetString("commit_id", "")),
		TimeoutMs:  intFromAny(args["timeout_ms"]),
		IntervalMs: intFromAny(args["interval_ms"]),
	}
}

func validateWorkflowTestRunRequest(req WorkflowTestRunRequest) error {
	if strings.TrimSpace(req.WorkflowID) == "" {
		return errors.New("workflow_id is required")
	}
	if strings.TrimSpace(req.SpaceID) == "" {
		return errors.New("space_id is required")
	}
	return nil
}

func explainWorkflowFailure(message, nodeType string) string {
	combined := strings.ToLower(message + " " + nodeType)
	switch {
	case strings.Contains(combined, "blockid") || strings.Contains(combined, "引用变量"):
		return "优先调用 workflow.get_bindable_variables,然后局部修复该节点 input/inputs、merge_groups 或 End returns,不要清空画布。"
	case strings.Contains(combined, "32") || strings.Contains(combined, "merge"):
		return "变量聚合节点只能聚合真实可绑定变量;不要聚合 type=13 输出节点,固定文案先用 type=15 文本处理产出 output:string。"
	case strings.Contains(combined, "13"):
		return "type=13 是 display-only 输出节点,不要作为 VariableMerge 或 End returns 的变量来源。"
	default:
		return "先读取 canvas context 和 binding diagnostics,定位失败节点后局部修复输入绑定、outputs、条件、merge_groups 或 End returns。"
	}
}

func jsonToolResult(isError bool, payload any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	result := mcp.NewToolResultText(string(data))
	result.IsError = isError
	result.StructuredContent = payload
	return result, nil
}

func errorPayload(code, message string) map[string]any {
	return map[string]any{
		"status": "error",
		"code":   code,
		"error":  message,
	}
}

func stringMap(value any) map[string]string {
	if value == nil {
		return nil
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if str, ok := val.(string); ok {
			out[key] = str
			continue
		}
		if val != nil {
			out[key] = fmt.Sprint(val)
		}
	}
	return out
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}
