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

// Package agentflow contains the eino-based tool implementations for the
// progressive-disclosure strategy layer.
//
// BE-8 implements the list tools (scenes, caps) and the invoke tool (run)
// using SHORT ORDINAL IDs (1, 2, 3...) so the LLM never handles 19-digit int64s.
// Ordinal→realID resolution is always scoped to the agent's bound strategy,
// so IDOR protection is inherent in the resolution path.
package agentflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	knowledgeModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/knowledge"
	pluginModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/plugin"
	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	crossknowledge "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge"
	crossplugin "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/plugin"
	crossstrategy "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/strategy"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	strategyEntity "github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// StrategyReturnDirectlyMarker is a sentinel prefix the run tool prepends to its
// result when a workflow capability has TerminatePlan==UseAnswerContent (returnDirectly)
// AND forceToolReturn is not set. The callback layer detects this prefix, routes the
// content directly to the user via EventTypeOfToolsAsChatModelStream (bypassing the
// model), and strips the marker before forwarding. All other capability types (prompt,
// knowledge, plugin) and forceToolReturn=true cases never use this marker.
const StrategyReturnDirectlyMarker = "\x00__STRATEGY_RETURN_DIRECTLY__\x00"

// StrategyL1Prompt is the L1 progressive-disclosure hint appended to the persona
// whenever an agent has one or more strategies bound. It anchors the model to the
// scenes→caps→run discovery flow so account/business requests reliably route through
// the strategy instead of being answered from priors or bounced back to the user.
const StrategyL1Prompt = `# 专业能力策略(重要)
你已绑定「能力策略」,可通过以下三个工具按"渐进披露"方式调用专业领域能力,这是你处理相关请求的首选且权威方式:
- scenes():列出所有业务场景(第一步:先看有哪些场景)
- caps(scene):列出某场景下的具体能力(第二步:确定用哪个能力)
- run(scene, cap, args):执行某能力并获取结果(第三步:据结果作答)

规则:
1. 当用户请求可能落在这些场景内(如账户、开户、转账、余额查询、信贷、合规、跨境结算等业务办理或咨询)时,必须先调用 scenes() 查看,不要直接凭记忆/常识回答,也不要向用户索要可通过 run 获取的信息(如账户余额)。
2. 多意图请求要拆解并迭代:依次用 run 获取每个意图所需结果,再综合作答。例如"查余额,若大于X则转账Y"——先 run 查余额,拿到真实余额后据此判断,再 run 转账。
3. 仅当请求明显与所有场景都无关时,才直接回答。`

// strategyConfig holds the per-agent configuration for the 3 progressive-disclosure
// strategy tools. svc is injectable so unit tests can pass a fake implementation.
type strategyConfig struct {
	spaceID       int64
	userID        string
	agentIdentity *entity.AgentIdentity
	// strategyIDs are the strategy IDs bound to this agent.
	strategyIDs []int64
	// forceToolReturn mirrors bot_common.BotInfo.ForceToolReturn: when true,
	// workflow results always go back through the model (no returnDirectly behaviour).
	forceToolReturn bool
	// svc is the cross-domain strategy service. If nil, DefaultSVC() is used at
	// call time (production path).
	svc crossstrategy.StrategyService
}

func (c *strategyConfig) getSvc() crossstrategy.StrategyService {
	if c.svc != nil {
		return c.svc
	}
	return crossstrategy.DefaultSVC()
}

// newStrategyTools returns the 3 strategy tools for an agent:
//   - scenes  (list scenarios — no params; uses the bound strategy)
//   - caps    (list capabilities in a scenario by 1-based ordinal)
//   - run     (invoke a capability by scene+cap ordinals)
//
// The "scenes" tool description embeds each bound strategy's name+desc so the
// model can autonomously decide when to invoke it without a persona instruction.
func newStrategyTools(ctx context.Context, conf *strategyConfig) ([]tool.InvokableTool, error) {
	scenesDesc := buildScenesDesc(ctx, conf)
	return []tool.InvokableTool{
		&listScenariosTool{conf: conf, desc: scenesDesc},
		&listCapabilitiesTool{conf: conf},
		&invokeCapabilityTool{conf: conf},
	}, nil
}

// buildScenesDesc constructs the autonomous-discovery description for the "scenes" tool
// by fetching each bound strategy's name+desc. Failures are silently skipped.
func buildScenesDesc(ctx context.Context, conf *strategyConfig) string {
	svc := conf.getSvc()
	base := "发现并使用以下策略下的能力。当用户请求与之相关时，先调用本工具列出场景，再用 caps 查看能力，用 run 调用。"
	if svc == nil || len(conf.strategyIDs) == 0 {
		return base
	}

	var parts []string
	for _, id := range conf.strategyIDs {
		strat, err := svc.GetStrategy(ctx, id)
		if err != nil || strat == nil {
			logs.CtxWarnf(ctx, "buildScenesDesc: failed to fetch strategy %d: %v", id, err)
			continue
		}
		entry := strat.Name
		if strat.Description != "" {
			entry += "（" + strat.Description + "）"
		}
		parts = append(parts, entry)
	}
	if len(parts) == 0 {
		return base
	}
	return base + "可用策略：" + strings.Join(parts, "；") + "。"
}

// ---------------------------------------------------------------------------
// Stable-ordering helpers — CRITICAL: scenes, caps, and run MUST use the same
// ordering so an ordinal means the same thing across all 3 calls.
// ---------------------------------------------------------------------------

// sortedScenarios returns the bound strategy's scenarios in stable order
// (SortOrder ASC, then ID ASC). All three tools call this to guarantee
// ordinal consistency within a conversation.
func sortedScenarios(ctx context.Context, svc crossstrategy.StrategyService, strategyID int64) ([]*strategyEntity.Scenario, error) {
	scenarios, err := svc.ListScenarios(ctx, strategyID)
	if err != nil {
		return nil, err
	}
	sort.Slice(scenarios, func(i, j int) bool {
		if scenarios[i].SortOrder != scenarios[j].SortOrder {
			return scenarios[i].SortOrder < scenarios[j].SortOrder
		}
		return scenarios[i].ID < scenarios[j].ID
	})
	return scenarios, nil
}

// sortedCapabilities returns a scenario's capabilities in stable order
// (SortOrder ASC, then ID ASC).
func sortedCapabilities(ctx context.Context, svc crossstrategy.StrategyService, scenarioID int64) ([]*strategyEntity.Capability, error) {
	caps, err := svc.ListCapabilities(ctx, scenarioID)
	if err != nil {
		return nil, err
	}
	sort.Slice(caps, func(i, j int) bool {
		if caps[i].SortOrder != caps[j].SortOrder {
			return caps[i].SortOrder < caps[j].SortOrder
		}
		return caps[i].ID < caps[j].ID
	})
	return caps, nil
}

// boundStrategyID returns conf.strategyIDs[0], the single bound strategy.
// The agent product decision is "one bound strategy per agent".
func boundStrategyID(conf *strategyConfig) (int64, error) {
	if len(conf.strategyIDs) == 0 {
		return 0, fmt.Errorf("no strategy bound to this agent")
	}
	return conf.strategyIDs[0], nil
}

// ---------------------------------------------------------------------------
// scenes — list scenarios (NO params; uses the single bound strategy)
// ---------------------------------------------------------------------------

type listScenariosTool struct {
	conf *strategyConfig
	desc string
}

func (t *listScenariosTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "scenes",
		Desc:        t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

type scenarioRow struct {
	ID   int `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	N    int    `json:"n"`
}

func (t *listScenariosTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	svc := t.conf.getSvc()
	if svc == nil {
		return "Error: strategy service is not available", nil
	}

	strategyID, err := boundStrategyID(t.conf)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	scenarios, err := sortedScenarios(ctx, svc, strategyID)
	if err != nil {
		return fmt.Sprintf("Error listing scenarios for strategy %d: %v", strategyID, err), nil
	}

	rows := make([]scenarioRow, 0, len(scenarios))
	for i, sc := range scenarios {
		caps, countErr := sortedCapabilities(ctx, svc, sc.ID)
		if countErr != nil {
			logs.CtxWarnf(ctx, "scenes: count capabilities for scenario %d failed: %v", sc.ID, countErr)
		}
		rows = append(rows, scenarioRow{
			ID:   i + 1, // 1-based ordinal
			Name: sc.Name,
			Desc: sc.Description,
			N:    len(caps),
		})
	}

	b, _ := json.Marshal(rows)
	return string(b), nil
}

// ---------------------------------------------------------------------------
// caps — list capabilities (param: scene = 1-based scenario ordinal)
// ---------------------------------------------------------------------------

type listCapabilitiesTool struct{ conf *strategyConfig }

func (t *listCapabilitiesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "caps",
		Desc: "List capabilities inside a scenario. Each capability is an invokable action (prompt, knowledge, workflow, or plugin) with its input schema.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"scene": {
				Type:     schema.Integer,
				Desc:     "Scenario ordinal (1-based, from scenes).",
				Required: true,
			},
		}),
	}, nil
}

type listCapabilitiesRequest struct {
	Scene int `json:"scene"`
}

type capabilityRow struct {
	ID     int            `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Desc   string         `json:"desc"`
	Schema map[string]any `json:"schema"`
}

// emptyInputSchema is the fallback returned when a capability takes no inputs
// or when the real schema cannot be derived (deleted / unpublished resource).
func emptyInputSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// toolInfoToInputSchema converts a tool's ParamsOneOf into a JSON-schema
// map[string]any of shape {"type":"object","properties":{...},"required":[...]}.
// It uses eino's ParamsOneOf.ToJSONSchema() (schema/tool.go) and coerces the
// resulting *jsonschema.Schema to a map via json round-trip. Returns the empty
// fallback when the tool has no parameters or conversion fails.
func toolInfoToInputSchema(ctx context.Context, info *schema.ToolInfo) map[string]any {
	if info == nil || info.ParamsOneOf == nil {
		return emptyInputSchema()
	}
	js, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil || js == nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: ToJSONSchema failed: %v", err)
		return emptyInputSchema()
	}
	b, err := json.Marshal(js)
	if err != nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: marshal json schema failed: %v", err)
		return emptyInputSchema()
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: unmarshal json schema failed: %v", err)
		return emptyInputSchema()
	}
	// Guarantee the "type":"object" envelope even if the underlying schema omitted it.
	if _, ok := m["type"]; !ok {
		m["type"] = "object"
	}
	if _, ok := m["properties"]; !ok {
		m["properties"] = map[string]any{}
	}
	return m
}

// capabilityInputSchema returns the REAL input JSON schema for each capability
// type so the model knows exactly which params run(args) expects:
//   - prompt:    no inputs (empty object).
//   - knowledge: {query (required), top_k} — matches the run tool's args parsing.
//   - workflow:  derived from the workflow-as-model-tool's declared parameters,
//     loaded the same way the run tool loads it (WorkflowAsModelTool by RefID/RefVersion).
//   - plugin:    derived from the plugin tool's declared parameters via
//     GetPluginInvokableTools (RefSubID=plugin_id, RefID=tool_id).
//
// Loading the workflow/plugin tool is best-effort: any failure (deleted /
// unpublished / service unavailable) degrades to the empty fallback and is
// logged at debug — it never errors the whole caps call.
func capabilityInputSchema(ctx context.Context, cap *strategyEntity.Capability, conf *strategyConfig) map[string]any {
	switch cap.Type {
	case strategyEntity.CapabilityTypePrompt:
		return emptyInputSchema()
	case strategyEntity.CapabilityTypeKnowledge:
		return map[string]any{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Search query"},
				"top_k": map[string]any{"type": "integer", "description": "Number of results to return"},
			},
		}
	case strategyEntity.CapabilityTypeWorkflow:
		return workflowCapabilityInputSchema(ctx, cap)
	case strategyEntity.CapabilityTypePlugin:
		return pluginCapabilityInputSchema(ctx, cap, conf)
	default:
		return emptyInputSchema()
	}
}

// workflowCapabilityInputSchema loads the workflow as a model tool (mirroring the
// run tool's WorkflowAsModelTool call) and converts its declared parameters into a
// JSON schema. Degrades to the empty fallback on any failure.
func workflowCapabilityInputSchema(ctx context.Context, cap *strategyEntity.Capability) map[string]any {
	wfSVC := crossworkflow.DefaultSVC()
	if wfSVC == nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: workflow service unavailable for cap %d", cap.ID)
		return emptyInputSchema()
	}
	policy := &vo.GetPolicy{
		ID:    cap.RefID,
		QType: workflowModel.FromLatestVersion,
	}
	if cap.RefVersion != "" {
		policy.QType = workflowModel.FromSpecificVersion
		policy.Version = cap.RefVersion
	}
	wfTools, err := wfSVC.WorkflowAsModelTool(ctx, []*vo.GetPolicy{policy})
	if err != nil || len(wfTools) == 0 {
		logs.CtxDebugf(ctx, "capabilityInputSchema: load workflow tool for cap %d failed: %v", cap.ID, err)
		return emptyInputSchema()
	}
	info, err := wfTools[0].Info(ctx)
	if err != nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: workflow tool Info for cap %d failed: %v", cap.ID, err)
		return emptyInputSchema()
	}
	return toolInfoToInputSchema(ctx, info)
}

// pluginCapabilityInputSchema loads the plugin tool via GetPluginInvokableTools
// (RefSubID=plugin_id, RefID=tool_id) and converts its declared parameters into a
// JSON schema. The InvokableTool's Info() carries the real ParamsOneOf. Degrades
// to the empty fallback on any failure.
func pluginCapabilityInputSchema(ctx context.Context, cap *strategyEntity.Capability, conf *strategyConfig) map[string]any {
	pluginSVC := crossplugin.DefaultSVC()
	if pluginSVC == nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: plugin service unavailable for cap %d", cap.ID)
		return emptyInputSchema()
	}
	isDraft := conf != nil && conf.agentIdentity != nil && conf.agentIdentity.IsDraft
	// PluginVersion "" => latest/online; specific version is honoured when pinned.
	// (nil/"0" would mean draft, but draft is signalled via IsDraft instead.)
	req := &pluginModel.ToolsInvokableRequest{
		PluginEntity: pluginModel.PluginEntity{
			PluginID:      cap.RefSubID,
			PluginVersion: ptr.Of(cap.RefVersion),
		},
		ToolsInvokableInfo: map[int64]*pluginModel.ToolsInvokableInfo{
			cap.RefID: {ToolID: cap.RefID},
		},
		IsDraft: isDraft,
	}
	toolMap, err := pluginSVC.GetPluginInvokableTools(ctx, req)
	if err != nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: load plugin tool for cap %d failed: %v", cap.ID, err)
		return emptyInputSchema()
	}
	pt, ok := toolMap[cap.RefID]
	if !ok || pt == nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: plugin tool %d not found for cap %d", cap.RefID, cap.ID)
		return emptyInputSchema()
	}
	info, err := pt.Info(ctx)
	if err != nil {
		logs.CtxDebugf(ctx, "capabilityInputSchema: plugin tool Info for cap %d failed: %v", cap.ID, err)
		return emptyInputSchema()
	}
	return toolInfoToInputSchema(ctx, info)
}

func (t *listCapabilitiesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := t.conf.getSvc()
	if svc == nil {
		return "Error: strategy service is not available", nil
	}

	var req listCapabilitiesRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	if req.Scene <= 0 {
		return "Error: scene must be a positive integer (1-based ordinal from scenes)", nil
	}

	strategyID, err := boundStrategyID(t.conf)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	// Resolve scene ordinal → real scenario using the same stable order.
	scenarios, err := sortedScenarios(ctx, svc, strategyID)
	if err != nil {
		return fmt.Sprintf("Error listing scenarios: %v", err), nil
	}
	if req.Scene > len(scenarios) {
		return fmt.Sprintf("Error: scene %d is out of range (there are %d scenarios)", req.Scene, len(scenarios)), nil
	}
	sc := scenarios[req.Scene-1] // 1-based → 0-based

	caps, err := sortedCapabilities(ctx, svc, sc.ID)
	if err != nil {
		return fmt.Sprintf("Error listing capabilities for scenario %d: %v", sc.ID, err), nil
	}

	rows := make([]capabilityRow, 0, len(caps))
	for i, cap := range caps {
		name := cap.AliasName
		if name == "" {
			name = fmt.Sprintf("%s#%d", cap.Type, cap.ID)
		}
		desc := cap.AliasDescription
		if desc == "" {
			desc = fmt.Sprintf("%s capability", cap.Type)
		}
		rows = append(rows, capabilityRow{
			ID:     i + 1, // 1-based ordinal within this scenario
			Type:   cap.Type,
			Name:   name,
			Desc:   desc,
			Schema: capabilityInputSchema(ctx, cap, t.conf),
		})
	}

	b, _ := json.Marshal(rows)
	return string(b), nil
}

// ---------------------------------------------------------------------------
// run — invoke a capability (params: scene + cap ordinals + optional args)
// ---------------------------------------------------------------------------

type invokeCapabilityTool struct{ conf *strategyConfig }

func (t *invokeCapabilityTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run",
		Desc: "Invoke a capability by its scene and cap ordinals. Returns the result (prompt text, retrieved knowledge, workflow output, or plugin response).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"scene": {
				Type:     schema.Integer,
				Desc:     "Scenario ordinal (1-based, from scenes).",
				Required: true,
			},
			"cap": {
				Type:     schema.Integer,
				Desc:     "Capability ordinal (1-based, from caps for that scene).",
				Required: true,
			},
			"args": {
				Type:     schema.Object,
				Desc:     "Arguments for the capability (shape depends on type).",
				Required: false,
			},
		}),
	}, nil
}

type invokeCapabilityRequest struct {
	Scene int            `json:"scene"`
	Cap   int            `json:"cap"`
	Args  map[string]any `json:"args,omitempty"`
}

// InvokableRun resolves scene+cap ordinals to real capability via the bound
// strategy's stable-sorted lists, then dispatches by capability type.
func (t *invokeCapabilityTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := t.conf.getSvc()
	if svc == nil {
		return "Error: strategy service is not available", nil
	}

	// 1. Parse arguments.
	var req invokeCapabilityRequest
	if argumentsInJSON != "" && argumentsInJSON != "null" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
			return argParseErrMsg(err), nil
		}
	}
	if req.Scene <= 0 {
		return "Error: scene must be a positive integer (1-based ordinal from scenes)", nil
	}
	if req.Cap <= 0 {
		return "Error: cap must be a positive integer (1-based ordinal from caps)", nil
	}

	strategyID, err := boundStrategyID(t.conf)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	// 2. Resolve scene ordinal → real scenario (auth is inherent: only bound strategy's scenarios).
	scenarios, err := sortedScenarios(ctx, svc, strategyID)
	if err != nil {
		return fmt.Sprintf("Error listing scenarios: %v", err), nil
	}
	if req.Scene > len(scenarios) {
		return fmt.Sprintf("Error: scene %d is out of range (there are %d scenarios)", req.Scene, len(scenarios)), nil
	}
	sc := scenarios[req.Scene-1]

	// 3. Resolve cap ordinal → real capability.
	caps, err := sortedCapabilities(ctx, svc, sc.ID)
	if err != nil {
		return fmt.Sprintf("Error listing capabilities for scenario: %v", err), nil
	}
	if req.Cap > len(caps) {
		return fmt.Sprintf("Error: cap %d is out of range (there are %d capabilities in scene %d)", req.Cap, len(caps), req.Scene), nil
	}
	cap := caps[req.Cap-1]

	// 4. Re-marshal arguments for downstream callers that expect a JSON string.
	var argumentsJSON string
	if req.Args != nil {
		b, marshalErr := json.Marshal(req.Args)
		if marshalErr != nil {
			return fmt.Sprintf("Error: failed to marshal args: %v", marshalErr), nil
		}
		argumentsJSON = string(b)
	}

	// 5. Dispatch by capability type (identical logic to before; only id resolution changed).
	switch cap.Type {
	case strategyEntity.CapabilityTypePrompt:
		return cap.PromptContent, nil

	case strategyEntity.CapabilityTypePlugin:
		execScene := pluginModel.ExecSceneOfOnlineAgent
		if t.conf.agentIdentity != nil && t.conf.agentIdentity.IsDraft {
			execScene = pluginModel.ExecSceneOfDraftAgent
		}
		pluginReq := &pluginModel.ExecuteToolRequest{
			UserID:          t.conf.userID,
			PluginID:        cap.RefSubID,
			ToolID:          cap.RefID,
			ExecDraftTool:   false,
			ArgumentsInJson: argumentsJSON,
			ExecScene:       execScene,
		}
		opts := []pluginModel.ExecuteToolOpt{
			pluginModel.WithToolVersion(cap.RefVersion),
		}
		pluginSVC := crossplugin.DefaultSVC()
		if pluginSVC == nil {
			return "Error: plugin service is not available", nil
		}
		pluginResp, pluginErr := pluginSVC.ExecuteTool(ctx, pluginReq, opts...)
		if pluginErr != nil {
			return fmt.Sprintf("Error executing plugin: %v", pluginErr), nil
		}
		return pluginResp.TrimmedResp, nil

	case strategyEntity.CapabilityTypeWorkflow:
		policy := &vo.GetPolicy{
			ID:    cap.RefID,
			QType: workflowModel.FromLatestVersion,
		}
		if cap.RefVersion != "" {
			policy.QType = workflowModel.FromSpecificVersion
			policy.Version = cap.RefVersion
		}
		policies := []*vo.GetPolicy{policy}
		wfSVC := crossworkflow.DefaultSVC()
		if wfSVC == nil {
			return "Error: workflow service is not available", nil
		}
		wfTools, wfErr := wfSVC.WorkflowAsModelTool(ctx, policies)
		if wfErr != nil {
			return fmt.Sprintf("Error executing workflow: %v", wfErr), nil
		}
		if len(wfTools) == 0 {
			return fmt.Sprintf("Error executing workflow: workflow %d not found", cap.RefID), nil
		}
		invokable, ok := wfTools[0].(tool.InvokableTool)
		if !ok {
			return "Error executing workflow: workflow tool is not invokable", nil
		}
		wfResult, wfRunErr := invokable.InvokableRun(ctx, argumentsJSON)
		if wfRunErr != nil {
			return fmt.Sprintf("Error executing workflow: %v", wfRunErr), nil
		}
		// Mirror native workflow returnDirectly: if the workflow returns answer content
		// (i.e. NOT ReturnVariables — this matches workflow_tool.InvokableRun's own
		// text-vs-JSON branch, and correctly covers the empty/default TerminatePlan that
		// WorkflowAsModelTool leaves on "return text" End nodes) AND forceToolReturn is
		// not set, prefix the result with the sentinel so the callback can route it
		// directly to the user (bypass the model).
		if !t.conf.forceToolReturn && wfTools[0].TerminatePlan() != vo.ReturnVariables {
			// Native eino dynamic returnDirectly: signal the react loop to TERMINATE
			// after this run call so the workflow's answer-content becomes the final
			// reply (mirrors how a directly-bound "return text" workflow behaves).
			// buildReturnDirectly is wired unconditionally, so this works even when the
			// agent has no statically-configured ToolReturnDirectly tools.
			// "return variable" workflows (ReturnVariables) skip this and keep looping,
			// which is what lets multi-intent chains (查余额→转账) continue.
			// react.SetReturnDirectly is the native eino dynamic returnDirectly hook: it
			// records THIS run call's id in the react state. eino state lookup is lexical
			// with a parent chain, and the ExportGraph()-recomposed agent installs the
			// react *state once at entry, so the write from inside the tool IS visible to
			// the post-tools buildReturnDirectly branch — the loop terminates and this
			// workflow's text becomes the final reply (直出), mirroring a directly-bound
			// "返回文本" workflow tool. ReturnVariables workflows skip this and keep
			// looping (so multi-intent chains like 查余额[返回变量]→转账[返回文本] work),
			// and forceToolReturn=true also skips it (global off-switch → relay to model).
			// Verified end-to-end on 226 + by graph-level tests (strategy_returndirectly_*).
			if rdErr := react.SetReturnDirectly(ctx); rdErr != nil {
				logs.CtxWarnf(ctx, "strategy run: SetReturnDirectly failed: %v", rdErr)
			}
			// Keep the marker so the callback layer renders this result directly to the
			// user via EventTypeOfToolsAsChatModelStream (the rendering half of returnDirectly).
			return StrategyReturnDirectlyMarker + wfResult, nil
		}
		return wfResult, nil

	case strategyEntity.CapabilityTypeKnowledge:
		// Parse query from args (required).
		query, _ := req.Args["query"].(string)
		if query == "" {
			return "Error: knowledge capability requires a 'query' argument", nil
		}

		// Parse optional top_k from args, falling back to RetrieveConfig.
		var topK *int64
		if tkRaw, ok := req.Args["top_k"]; ok {
			switch v := tkRaw.(type) {
			case float64:
				n := int64(v)
				topK = &n
			case int64:
				topK = &v
			case int:
				n := int64(v)
				topK = &n
			}
		}
		if topK == nil && cap.RetrieveConfig != "" {
			var rc struct {
				TopK *int64 `json:"top_k"`
			}
			if jsonErr := json.Unmarshal([]byte(cap.RetrieveConfig), &rc); jsonErr == nil && rc.TopK != nil {
				topK = rc.TopK
			}
		}

		knowledgeSVC := crossknowledge.DefaultSVC()
		if knowledgeSVC == nil {
			return "Error: knowledge service is not available", nil
		}
		retrieveReq := &knowledgeModel.RetrieveRequest{
			Query:        query,
			KnowledgeIDs: []int64{cap.RefID},
		}
		if topK != nil {
			retrieveReq.Strategy = &knowledgeModel.RetrievalStrategy{TopK: topK}
		}
		retrieveResp, retrieveErr := knowledgeSVC.Retrieve(ctx, retrieveReq)
		if retrieveErr != nil {
			return fmt.Sprintf("Error retrieving knowledge: %v", retrieveErr), nil
		}
		if retrieveResp == nil || len(retrieveResp.RetrieveSlices) == 0 {
			return "No results found.", nil
		}
		var sb strings.Builder
		for i, rs := range retrieveResp.RetrieveSlices {
			if rs.Slice == nil {
				continue
			}
			if i > 0 {
				sb.WriteString("\n\n---\n\n")
			}
			sb.WriteString(rs.Slice.GetSliceContent())
		}
		return sb.String(), nil

	default:
		return fmt.Sprintf("Error: unknown capability type %q", cap.Type), nil
	}
}
