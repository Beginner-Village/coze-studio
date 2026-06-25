# Workflow Canvas Automation Protocol and Node Smoke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Turn workflow canvas AI editing from prompt-dependent behavior into a verified automation layer. Every visible node type must have a machine-readable capability record, a progressive node spec, an explicit edit/configure/binding/test status, and at least one reproducible smoke scenario or a documented resource-bound skip reason.

**Architecture:** Keep the live browser canvas as the first execution adapter for real-time editing, add a shared command protocol for internal agent tools and external MCP clients, and build node smoke scenarios that exercise the same command path used by finmallclaw. Backend MCP and draft editing come after the browser adapter has a trustworthy node matrix.

**Tech Stack:** TypeScript services and Vitest in `frontend/packages/workflow/playground`; existing FlowGram workflow editor services; Go backend agentflow tools; system-provided MCP service in `backend/application/workflow/canvasautomation`, with `backend/cmd/workflowmcpserver` as a thin runtime wrapper.

---

## Current Truth

The afternoon work improved the model guidance, but it does not yet prove complete coverage.

Existing useful pieces:

- `frontend/packages/workflow/playground/src/services/workflow-agent-node-catalog.ts` has user-facing node purposes and configuration hints.
- `frontend/packages/workflow/playground/src/services/workflow-agent-node-capabilities.ts` tracks 41 visible node types and support levels.
- `frontend/packages/workflow/playground/src/services/workflow-agent-semantic-config.ts` converts semantic `configure_node` payloads into real form params for the supported nodes.
- `frontend/packages/workflow/playground/src/services/workflow-agent-canvas-summary.ts` can summarize nodes, bindable variables, and binding diagnostics.
- `frontend/packages/workflow/playground/src/services/workflow-agent-command-service.ts` executes the browser-side canvas commands.
- `backend/domain/agent/singleagent/internal/agentflow/node_tool_workflow_canvas.go` exposes the internal `workflow_canvas_*` tools to the super agent.
- `backend/application/workflow/workflow_agent_test_run.go` and `/api/workflow_api/agent_test_run` provide a stable test-run path that can avoid reloading the chat panel.

Current gap:

- A node can be documented without being actually configurable.
- A node can be configurable by semantic params without being tested on the real canvas.
- A smoke test can pass add/connect while still missing input bindings, outputs, merge groups, or End returns.
- Resource nodes need a truthful `skipped-with-reason`, not a pretend pass.
- The same capability/spec/command schema is not yet exposed to external tools through MCP.

The implementation below makes those gaps visible and testable before expanding the set of supported nodes.

Status update, 2026-06-23:

- The main backend now exposes the system-owned streamable HTTP MCP endpoint at `/api/workflow_mcp/mcp`.
- `backend/application/workflow/canvasautomation` now registers MCP write tools for `workflow.add_node`, `workflow.connect`, `workflow.configure_node`, `workflow.set_node_params`, `workflow.delete_node`, `workflow.delete_line`, `workflow.clear_canvas`, and `workflow.auto_layout`.
- MCP write tools return the same `dispatched_to_canvas` ack shape as the embedded `workflow_canvas_*` tools, so the browser-live bridge can execute them in the visible canvas.
- For external MCP clients, the main backend now has an in-memory command relay and `/api/workflow_mcp/browser_commands` polling route. The workflow page now mounts a browser bridge that polls this route and applies returned commands through `WorkflowAgentCommandService.applyCommandEnvelope`; visual verification in a running browser is still required.
- The browser relay is now bidirectional. Poll responses include per-command `requests[].request_id` and `requires_response`; browser-live read commands post results to `/api/workflow_mcp/browser_command_results`.
- `workflow.get_canvas_context` and `workflow.get_bindable_variables` now use the browser-live relay when the workflow page is open, so external MCP clients can read real canvas context and bindable variables instead of guessing.
- `workflow.test_run` remains a backend execution tool that returns structured test output and should not trigger the frontend test-run drawer.
- End node guidance now includes return-variable mode and return-text mode; return-text mode binds multiple upstream variables into `content/text/template` and can set `streaming_output=true`.

---

## File Map

Create:

- `frontend/packages/workflow/playground/src/services/workflow-agent-command-protocol.ts`
  Shared command envelope, command result, diagnostics, legacy ack normalization, and display-name helpers.
- `frontend/packages/workflow/playground/src/services/__tests__/workflow-agent-command-protocol.test.ts`
  Unit tests for protocol normalization and Chinese display names.
- `frontend/packages/workflow/playground/src/services/workflow-agent-node-smoke-scenarios.ts`
  Node-by-node smoke scenario catalog. This is the source of truth for what is actually verified.
- `frontend/packages/workflow/playground/src/services/__tests__/workflow-agent-node-smoke-scenarios.test.ts`
  Tests that every visible node has a scenario, skip reason, or explicit unsupported status.
- `frontend/packages/workflow/playground/src/services/workflow-agent-node-smoke-report.ts`
  Converts scenario results into a compact JSON and markdown report.
- `frontend/packages/workflow/playground/src/services/__tests__/workflow-agent-node-smoke-report.test.ts`
  Tests report status rollups, missing coverage failures, and resource-bound skip rendering.
- `backend/application/workflow/canvasautomation/catalog.go`
  Shared backend node capability and node spec registry that can later be consumed by agentflow and MCP.
- `backend/application/workflow/canvasautomation/catalog_test.go`
  Tests backend catalog coverage for the nodes exposed to the agent.
- `backend/application/workflow/canvasautomation/mcp.go`
  System-owned MCP tool registration for readonly capabilities/specs/context placeholders and `workflow.test_run`.
- `backend/application/workflow/canvasautomation/mcp_test.go`
  Tests system MCP tool registration and JSON/error contracts.
- `backend/cmd/workflowmcpserver/main.go`
  Thin streamable HTTP startup wrapper around the system MCP package.
- `backend/cmd/workflowmcpserver/main_test.go`
  Startup/tool registration tests for the MCP command package.

Modify:

- `frontend/packages/workflow/playground/src/services/index.ts`
  Export protocol, smoke scenario, and smoke report helpers.
- `frontend/packages/workflow/playground/src/services/workflow-agent-command-service.ts`
  Keep existing commands working, add envelope-level `applyCommandEnvelope` and normalize command results through the protocol.
- `frontend/packages/workflow/playground/src/services/workflow-agent-node-capabilities.ts`
  Add smoke status fields derived from the scenario catalog only if this does not create an import cycle. Otherwise keep the fields in the report builder.
- `backend/domain/agent/singleagent/internal/agentflow/workflow_canvas_node_catalog.go`
  Either consume the new backend catalog or keep it as a temporary mirror with a test that detects drift.
- `backend/domain/agent/singleagent/internal/agentflow/node_tool_workflow_canvas.go`
  Keep internal tool names stable. Optionally normalize read-tool responses to match the protocol once the backend catalog exists.

Do not modify unrelated super-agent, marketplace, memory, or model-selector files in this workstream.

---

## Data Contracts

### Command Protocol

`workflow-agent-command-protocol.ts` must define:

```ts
export type WorkflowCanvasSurface = 'workflow' | 'conversation';

export type WorkflowCanvasCommandOp =
  | 'get_node_catalog'
  | 'get_node_spec'
  | 'get_node_capability_audit'
  | 'get_canvas_context'
  | 'get_bindable_variables'
  | 'add_node'
  | 'delete_node'
  | 'delete_line'
  | 'connect'
  | 'configure_node'
  | 'set_node_params'
  | 'clear_canvas'
  | 'auto_layout'
  | 'validate'
  | 'test_run'
  | 'explain_failure';

export interface WorkflowCanvasCommandEnvelope {
  protocol: 'canvas_automation.v0';
  surface: WorkflowCanvasSurface;
  space_id?: string;
  canvas_id?: string;
  mode: 'browser_live' | 'backend_draft' | 'readonly';
  request_id: string;
  commands: WorkflowCanvasCommand[];
}

export interface WorkflowCanvasCommand {
  op: WorkflowCanvasCommandOp;
  target?: string;
  args?: Record<string, unknown>;
}

export interface WorkflowCanvasCommandResultItem {
  op: WorkflowCanvasCommandOp;
  ok: boolean;
  node_id?: string;
  line_id?: string;
  target?: string;
  message?: string;
  diagnostics: WorkflowCanvasDiagnostic[];
}

export interface WorkflowCanvasCommandResult {
  protocol: 'canvas_automation.v0';
  request_id: string;
  status: 'ok' | 'partial' | 'failed';
  results: WorkflowCanvasCommandResultItem[];
  canvas_context?: string;
  binding_diagnostics: WorkflowCanvasDiagnostic[];
}
```

The actual implementation can use existing local naming conventions, but the public shape must stay stable. The command service can continue accepting old `AgentCanvasCommand` values during migration.

### Smoke Scenario Model

`workflow-agent-node-smoke-scenarios.ts` must define:

```ts
export type WorkflowAgentNodeSmokeStatus =
  | 'verified-full'
  | 'verified-resource-bound'
  | 'verified-add-only'
  | 'verified-not-executable'
  | 'skipped-resource-missing'
  | 'unsupported'
  | 'failing';

export interface WorkflowAgentNodeSmokeScenario {
  nodeType: string;
  nodeName: string;
  status: WorkflowAgentNodeSmokeStatus;
  runtime: 'local' | 'resource' | 'sub-canvas' | 'not-executable';
  requiredCommands: WorkflowCanvasCommandOp[];
  setupCommands: WorkflowCanvasCommand[];
  assertions: string[];
  expectedBindableVariables: string[];
  skipReason?: string;
  knownGaps?: string[];
}
```

The scenario file is not allowed to say a node is `verified-full` unless it has:

- an add or singleton reference strategy,
- a semantic configure step or a not-executable reason,
- at least one binding assertion,
- a validate or test-run assertion when runtime is `local`.

---

## Implementation Tasks

### Task 1: Protocol Types and Legacy Ack Normalization

- [ ] Create `workflow-agent-command-protocol.ts`.
- [ ] Add Chinese display labels for every command:
  - `add_node` -> `添加节点`
  - `configure_node` -> `配置节点`
  - `get_bindable_variables` -> `读取可绑定变量`
  - `test_run` -> `试运行工作流`
  - keep the full mapping for all command ops.
- [ ] Add `normalizeWorkflowCanvasLegacyAck(input)` for existing backend ack:
  - Accepts `{status:"dispatched_to_canvas", op, args}`.
  - Returns a `WorkflowCanvasCommandResultItem` with `ok=true` for dispatched writes.
  - Does not claim workflow success.
- [ ] Add `createWorkflowCanvasCommandEnvelope(commands, identity)` to stamp protocol, mode, request id, and commands.
- [ ] Add unit tests:
  - legacy ack maps to one result item,
  - unknown op gets a safe display label,
  - batched envelope keeps command order,
  - `test_run` result can carry binding diagnostics.

### Task 2: Command Service Envelope Entry Point

- [ ] Modify `workflow-agent-command-service.ts` to import protocol types.
- [ ] Add `applyCommandEnvelope(envelope)` that loops through commands and calls existing `execCommand`.
- [ ] Preserve existing `execCommand` behavior for the chat panel.
- [ ] Return protocol results from `applyCommandEnvelope`.
- [ ] After any write command batch, trigger the existing auto-layout policy once, not once per command, when the batch contains add/connect/delete/clear.
- [ ] Do not change the UI panel rendering in this task.
- [ ] Add tests or extend existing command-service tests:
  - envelope with add + connect keeps tag mapping,
  - batch auto-layout is requested once,
  - failed command returns `status:"partial"` and does not hide the failure.

### Task 3: Node Smoke Scenario Catalog

- [ ] Create `workflow-agent-node-smoke-scenarios.ts`.
- [ ] Define scenarios for currently local-verifiable nodes first:
  - `1` Start: singleton, not addable, exposes `input:string`.
  - `2` End: singleton, bind returns from a real upstream output.
  - `5` Code: bind `start.input`, return `output:string`.
  - `8` If: bind `start.input`, true/false ports, condition uses a real variable.
  - `13` Output: message node, can display text or local input, cannot be a stable source for merge or End.
  - `15` Text: fixed text without input, text with bound input, produces `output:string`.
  - `18` Question: configure prompt/question and answer variable.
  - `30` Input: declares structured workflow input output.
  - `32` VariableMerge: merge only real upstream outputs, never type 13 output nodes.
  - `58` JsonStringify: bind object-like input and produce string output.
- [ ] Define resource-bound scenarios with skip reasons:
  - `3` LLM: requires model resource, semantic config exists, runtime smoke needs model.
  - `4` Plugin/API: requires plugin/api resource.
  - `6` Knowledge: requires dataset resource.
  - `99` CardSelector: requires card resource.
  - `100` Agent: requires child agent resource.
- [ ] Define priority partial scenarios:
  - `45` HTTP: unsupported until semantic config covers method, url, headers, query, body, outputs.
  - `59` JsonParser: unsupported until semantic config covers schema and output extraction.
  - `22` Intent: unsupported until candidate list and classification output are verified.
  - `21` Loop, `28` Batch, `19` Break, `29` Continue`: sub-canvas runtime required.
- [ ] Define all remaining visible node types with a skip or unsupported reason:
  - `9`, `11`, `12`, `14`, `16`, `17`, `20`, `23`, `26`, `27`, `31`, `34`, `35`, `36`, `42`, `43`, `44`, `46`, `61`.
- [ ] Unit test that every type in `WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES` has exactly one smoke scenario.
- [ ] Unit test that any `verified-full` scenario has `configure_node` unless it is a singleton or not-executable.
- [ ] Unit test that type `32` expected variables never come from type `13` output nodes.
- [ ] Unit test that type `15` fixed-text scenario explicitly permits no input binding.

### Task 4: Smoke Report Builder

- [ ] Create `workflow-agent-node-smoke-report.ts`.
- [ ] Add `buildWorkflowAgentNodeSmokeReport({capabilities, scenarios, results})`.
- [ ] Report fields:
  - total visible nodes,
  - verified full count,
  - resource-bound count,
  - skipped count,
  - unsupported count,
  - failing count,
  - per-node status,
  - required next implementation tasks.
- [ ] Add markdown renderer for human review.
- [ ] Add JSON renderer for machine use by future MCP/CI.
- [ ] Unit tests:
  - missing scenario fails the report,
  - resource-bound skip is not counted as pass,
  - failing scenario lists failing command and node type,
  - local verified scenario lists required bindable variables.

### Task 5: Binding Diagnostics Contract Tightening

- [ ] Extend tests in `workflow-agent-canvas-summary.test.ts` for the specific failures seen today:
  - VariableMerge references an undefined upstream variable.
  - VariableMerge references a type `13` output node.
  - End return is empty.
  - Text node fixed content has default input removed and should not be diagnosed as missing input.
  - Output node with fixed content is valid as display only but invalid as a merge source.
- [ ] Update `workflow-agent-canvas-summary.ts` only if the tests expose false positives or missing diagnostics.
- [ ] Add a short diagnostic recommendation string for each failure:
  - use `workflow_canvas_get_bindable_variables`,
  - configure upstream `outputs`,
  - replace display Output with Text when downstream needs a variable,
  - bind End returns to a real upstream output.
- [ ] Run the frontend service test subset after this task.

### Task 6: Progressive Skill Prompt Gate

- [ ] Update frontend binding guide in `workflow-agent-resource-service.ts` so the model is told this sequence:
  - get node catalog,
  - get node spec for planned types,
  - add nodes,
  - connect nodes,
  - configure every node from Start to End,
  - get bindable variables before configuring conditions, merge groups, and End returns,
  - get canvas context,
  - fix binding diagnostics locally,
  - only then run test.
- [ ] Update backend prompt in `system_prompt.go` with the same sequence.
- [ ] Add tests that the prompt forbids clearing the canvas when the user only asks to fix binding errors.
- [ ] Add tests that the prompt mentions fixed-text Text nodes can remove default inputs.

### Task 7: Backend Shared Catalog for MCP

- [ ] Create `backend/application/workflow/canvasautomation/catalog.go`.
- [ ] Define Go structs for node capability and node spec:
  - `NodeCapability`
  - `NodeSpec`
  - `CommandSpec`
  - `BindingRule`
- [ ] Seed the backend catalog with the same full/local and resource-bound nodes as the frontend.
- [ ] Add tests that:
  - all backend tool documented types have catalog entries,
  - type `13` says it is display-only and not a merge source,
  - type `32` says variables must come from bindable variables,
  - type `15` says fixed text may remove default input.
- [ ] Decide in code comments or package doc that the frontend remains the source for browser-specific semantic params until backend draft editing is implemented.
- [ ] Do not import from `backend/domain/agent/singleagent/internal/agentflow` into `backend/cmd/workflowmcpserver`; that would violate Go internal package boundaries.

### Task 8: MCP Server Skeleton

- [ ] Create `backend/cmd/workflowmcpserver/main.go` using the same library style as `backend/cmd/mcptestserver/main.go`.
- [ ] Register readonly tools:
  - `workflow.list_node_capabilities`
  - `workflow.get_node_spec`
  - `workflow.get_canvas_context`
  - `workflow.get_bindable_variables`
- [ ] Register testing tools:
  - `workflow.test_run`
  - `workflow.explain_failure`
- [ ] First version behavior:
  - capabilities/spec read from `canvasautomation`.
  - `test_run` calls the existing application service or returns a typed error if required identity is missing.
  - context/bindable variables can return a typed `not_implemented_backend_draft_context` error until backend draft reader exists.
- [ ] Add tests that all tools are registered and missing identity errors are stable JSON.
- [ ] Document that write tools are intentionally deferred until the backend draft adapter exists or a browser bridge is explicitly connected.

### Task 9: Browser Visual Smoke Runner Design

- [ ] Add a small design note under `docs/workflow-agent-unified-canvas-automation-plan-20260623.md` or a new `docs/workflow-agent-node-smoke-runner.md` describing the local runner.
- [ ] Runner will use local dev by default:
  - backend: `make server` with remote-compatible config,
  - frontend: local `rsbuild dev` with `API_PROXY_TARGET=http://10.10.10.226:8896`,
  - browser: visible Playwright or Browser MCP, not headless.
- [ ] Runner must create or open a dedicated test workflow, not mutate the user's active workflow.
- [ ] Runner must execute command envelopes, not direct UI clicks, except for final visual confirmation screenshots.
- [ ] Runner must output:
  - JSON smoke report,
  - screenshots for failing nodes,
  - console/network error summary.
- [ ] This task is design-only unless the user explicitly asks to build the runner in the same pass.

### Task 10: Complex Workflow Regression Set

- [ ] Define four complex workflow fixtures in the smoke scenario file or a sibling fixture file:
  - banking transfer multi-intent with missing account/card/cardholder/confirmation paths,
  - branch merge with Text nodes and VariableMerge,
  - external resource flow with Plugin or HTTP once supported,
  - loop/batch flow once sub-canvas support exists.
- [ ] For the first pass, implement only the branch merge fixture using local nodes.
- [ ] The banking transfer fixture can be specified but should be marked resource-bound until mock API and plugin registration are ready.
- [ ] Each fixture must list:
  - required nodes,
  - expected bindable variables after each stage,
  - expected validation state,
  - test input,
  - expected output or expected skipped reason.

---

## Verification Commands

Run frontend service tests after Tasks 1 to 6:

```bash
cd frontend/packages/workflow/playground
pnpm vitest run \
  src/services/__tests__/workflow-agent-command-protocol.test.ts \
  src/services/__tests__/workflow-agent-node-smoke-scenarios.test.ts \
  src/services/__tests__/workflow-agent-node-smoke-report.test.ts \
  src/services/__tests__/workflow-agent-canvas-summary.test.ts \
  src/services/__tests__/workflow-agent-resource-service.test.ts
pnpm tsc --noEmit -p tsconfig.json
```

Run backend tests after Tasks 7 and 8:

```bash
cd backend
SESSION_HMAC_SECRET=test-secret go test ./application/workflow/canvasautomation ./cmd/workflowmcpserver -count=1
```

Run existing agentflow prompt/tool tests after prompt changes:

```bash
cd backend
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestWorkflowCanvas|TestSuperAgentWorkflowCanvasPromptRequiresContextAndConfigureNode' -count=1
```

Do not deploy to `10.10.10.226` until all local unit/typecheck commands pass.

---

## Acceptance Criteria

- Every visible workflow node type has one explicit smoke scenario entry.
- The report can answer: full support, resource-bound, add-only, unsupported, failing.
- The agent can no longer honestly claim “配置正确” while binding diagnostics contain errors.
- VariableMerge has dedicated tests and guidance for real bindable outputs.
- Type `13` Output is clearly treated as display-only, not as a source for merge or End returns.
- Type `15` Text can be configured as fixed text with no input when appropriate.
- The same command protocol can be used by finmallclaw browser execution and future MCP entry points.
- Backend MCP has a first readonly/test-run skeleton without pretending backend draft write support is complete.

---

## Execution Order

1. Implement Tasks 1 to 4 first. This gives a protocol and a node coverage report without touching risky canvas internals.
2. Implement Tasks 5 and 6 next. This fixes the current VariableMerge/Text/Output guidance failures and prevents the model from clearing the canvas on local binding errors.
3. Implement Tasks 7 and 8 after the frontend matrix is stable. This exposes the same capability/spec surface to external MCP clients.
4. Implement Tasks 9 and 10 after the unit matrix is green. This turns the matrix into visible browser evidence and complex workflow regressions.

This order is deliberate: first prove what the system knows, then improve node-specific behavior, then expose it externally.
