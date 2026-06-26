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
// BE-8 implements the list tools (list_scenarios, list_capabilities) and
// the stub for invoke_capability whose dispatch body is filled in by BE-9.
package agentflow

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	pluginModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/plugin"
	knowledgeModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/knowledge"
	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	crossknowledge "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge"
	crossplugin "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/plugin"
	crossstrategy "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/strategy"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	strategyEntity "github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// strategyConfig holds the per-agent configuration for the 3 progressive-disclosure
// strategy tools. svc is injectable so unit tests can pass a fake implementation.
type strategyConfig struct {
	spaceID       int64
	userID        string
	agentIdentity *entity.AgentIdentity
	// strategyIDs are the strategy IDs bound to this agent.
	strategyIDs []int64
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
//   - scenes  (list scenarios)
//   - caps    (list capabilities in a scenario)
//   - run     (invoke a capability)
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
// scenes — list scenarios
// ---------------------------------------------------------------------------

type listScenariosTool struct {
	conf *strategyConfig
	desc string
}

func (t *listScenariosTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "scenes",
		Desc: t.desc,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"strat": {
				Type:     schema.String,
				Desc:     "Strategy id to filter by (optional; omit to list all bound strategies).",
				Required: false,
			},
		}),
	}, nil
}

type listScenariosRequest struct {
	Strat string `json:"strat,omitempty"`
}

type scenarioRow struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	N    int    `json:"n"`
}

func (t *listScenariosTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := t.conf.getSvc()
	if svc == nil {
		return "Error: strategy service is not available", nil
	}

	var req listScenariosRequest
	if argumentsInJSON != "" && argumentsInJSON != "null" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
			return argParseErrMsg(err), nil
		}
	}

	// Determine which strategy IDs to query.
	strategyIDs := t.conf.strategyIDs
	if req.Strat != "" {
		id, err := strconv.ParseInt(req.Strat, 10, 64)
		if err != nil {
			return fmt.Sprintf("Error: invalid strat %q: %v", req.Strat, err), nil
		}
		// R1: verify the requested strategy id is in the agent's bound set.
		if !slices.Contains(t.conf.strategyIDs, id) {
			return "Error: strategy_id is not bound to this agent", nil
		}
		strategyIDs = []int64{id}
	}

	var rows []scenarioRow
	for _, sid := range strategyIDs {
		scenarios, err := svc.ListScenarios(ctx, sid)
		if err != nil {
			return fmt.Sprintf("Error listing scenarios for strategy %d: %v", sid, err), nil
		}
		for _, sc := range scenarios {
			caps, countErr := svc.ListCapabilities(ctx, sc.ID)
			if countErr != nil {
				logs.CtxWarnf(ctx, "scenes: count capabilities for scenario %d failed: %v", sc.ID, countErr)
			}
			rows = append(rows, scenarioRow{
				ID:   sc.ID,
				Name: sc.Name,
				Desc: sc.Description,
				N:    len(caps),
			})
		}
	}

	if rows == nil {
		rows = []scenarioRow{}
	}
	b, _ := json.Marshal(rows)
	return string(b), nil
}

// ---------------------------------------------------------------------------
// caps — list capabilities
// ---------------------------------------------------------------------------

type listCapabilitiesTool struct{ conf *strategyConfig }

func (t *listCapabilitiesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "caps",
		Desc: "List capabilities inside a scenario. Each capability is an invokable action (prompt, knowledge, workflow, or plugin) with its input schema.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"scene": {
				Type:     schema.String,
				Desc:     "Scenario id (from scenes).",
				Required: true,
			},
		}),
	}, nil
}

type listCapabilitiesRequest struct {
	Scene string `json:"scene"`
}

type capabilityRow struct {
	ID     int64          `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Desc   string         `json:"desc"`
	Schema map[string]any `json:"schema"`
}

// capabilityInputSchema returns a minimal JSON schema for each capability type.
func capabilityInputSchema(cap *strategyEntity.Capability) map[string]any {
	switch cap.Type {
	case strategyEntity.CapabilityTypePrompt:
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	case strategyEntity.CapabilityTypeKnowledge:
		return map[string]any{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Search query"},
				"top_k": map[string]any{"type": "integer", "description": "Number of results to return"},
			},
		}
	case strategyEntity.CapabilityTypeWorkflow, strategyEntity.CapabilityTypePlugin:
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	default:
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
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
	if req.Scene == "" {
		return "Error: scene is required", nil
	}

	scenarioID, err := strconv.ParseInt(req.Scene, 10, 64)
	if err != nil {
		return fmt.Sprintf("Error: invalid scene %q: %v", req.Scene, err), nil
	}

	// R2: verify scenario_id belongs to one of the agent's bound strategies.
	validScenarioIDs := make(map[int64]struct{})
	for _, sid := range t.conf.strategyIDs {
		scenarios, sErr := svc.ListScenarios(ctx, sid)
		if sErr != nil {
			logs.CtxWarnf(ctx, "caps: failed to list scenarios for strategy %d: %v", sid, sErr)
			continue
		}
		for _, sc := range scenarios {
			validScenarioIDs[sc.ID] = struct{}{}
		}
	}
	if _, ok := validScenarioIDs[scenarioID]; !ok {
		return "Error: scenario_id is not accessible to this agent", nil
	}

	caps, err := svc.ListCapabilities(ctx, scenarioID)
	if err != nil {
		return fmt.Sprintf("Error listing capabilities for scenario %d: %v", scenarioID, err), nil
	}

	rows := make([]capabilityRow, 0, len(caps))
	for _, cap := range caps {
		name := cap.AliasName
		if name == "" {
			name = fmt.Sprintf("%s#%d", cap.Type, cap.ID)
		}
		desc := cap.AliasDescription
		if desc == "" {
			desc = fmt.Sprintf("%s capability (id=%d)", cap.Type, cap.ID)
		}
		rows = append(rows, capabilityRow{
			ID:     cap.ID,
			Type:   cap.Type,
			Name:   name,
			Desc:   desc,
			Schema: capabilityInputSchema(cap),
		})
	}

	b, _ := json.Marshal(rows)
	return string(b), nil
}

// ---------------------------------------------------------------------------
// run — invoke a capability
// ---------------------------------------------------------------------------

type invokeCapabilityTool struct{ conf *strategyConfig }

func (t *invokeCapabilityTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run",
		Desc: "Invoke a capability by its id. Returns the result (prompt text, retrieved knowledge, workflow output, or plugin response).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"cap": {
				Type:     schema.String,
				Desc:     "Capability id (from caps).",
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
	Cap  string         `json:"cap"`
	Args map[string]any `json:"args,omitempty"`
}

// InvokableRun implements BE9a: auth gate + prompt dispatch.
// workflow / plugin / knowledge routing is implemented in BE9b.
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
	if req.Cap == "" {
		return "Error: cap is required", nil
	}
	capID, err := strconv.ParseInt(req.Cap, 10, 64)
	if err != nil {
		return fmt.Sprintf("Error: invalid cap %q: %v", req.Cap, err), nil
	}

	// 2. AUTH: verify capability_id is in the agent's bound set.
	allowed, err := svc.ListCapabilityIDsByStrategies(ctx, t.conf.strategyIDs)
	if err != nil {
		return fmt.Sprintf("Error: failed to validate capability authorization: %v", err), nil
	}
	if !slices.Contains(allowed, capID) {
		return "Error: capability_id is not bound to this agent", nil
	}

	// 3. Resolve and dispatch.
	cap, err := svc.ResolveCapability(ctx, capID)
	if err != nil {
		return fmt.Sprintf("Error: failed to resolve capability %d: %v", capID, err), nil
	}

	// Re-marshal arguments for downstream callers that expect a JSON string.
	var argumentsJSON string
	if req.Args != nil {
		b, marshalErr := json.Marshal(req.Args)
		if marshalErr != nil {
			return fmt.Sprintf("Error: failed to marshal args: %v", marshalErr), nil
		}
		argumentsJSON = string(b)
	}

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
