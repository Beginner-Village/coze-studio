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
	"testing"

	einoCompose "github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/mock/gomock"

	knowledgeModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/knowledge"
	pluginModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/plugin"
	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	crossknowledge "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge"
	"github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge/knowledgemock"
	crossplugin "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/plugin"
	"github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/plugin/pluginmock"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	agentEntity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	workflowDomain "github.com/ynet-dev/ynet-studio/backend/domain/workflow"
	workflowEntity "github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
)

// fakeStrategySvc implements crossstrategy.StrategyService for tests.
// Bound strategies: [1, 2]. Each strategy has one scenario (ID = strategyID*10).
// Scenarios 10 and 20 are valid; scenario 99 is foreign/unbound.
// Capabilities 100, 101 belong to scenario 10 (strategy 1).
type fakeStrategySvc struct{}

func (f *fakeStrategySvc) GetStrategy(_ context.Context, id int64) (*entity.Strategy, error) {
	return &entity.Strategy{
		ID:          id,
		Name:        fmt.Sprintf("Strategy-%d", id),
		Description: fmt.Sprintf("desc for strategy %d", id),
	}, nil
}

func (f *fakeStrategySvc) ListScenarios(_ context.Context, strategyID int64) ([]*entity.Scenario, error) {
	return []*entity.Scenario{
		{ID: strategyID * 10, StrategyID: strategyID, Name: "ScenarioA", Description: "desc A", SortOrder: 1},
	}, nil
}

func (f *fakeStrategySvc) ListCapabilities(_ context.Context, scenarioID int64) ([]*entity.Capability, error) {
	return []*entity.Capability{
		{
			ID: 100, ScenarioID: scenarioID, Type: entity.CapabilityTypePrompt,
			AliasName: "MyPrompt", AliasDescription: "prompt alias desc",
		},
		{
			ID: 101, ScenarioID: scenarioID, Type: entity.CapabilityTypeKnowledge,
			AliasName: "", AliasDescription: "",
		},
	}, nil
}

func (f *fakeStrategySvc) ResolveCapability(_ context.Context, capID int64) (*entity.Capability, error) {
	return &entity.Capability{
		ID:            capID,
		Type:          entity.CapabilityTypePrompt,
		PromptContent: "Hello, I am the prompt content for cap " + fmt.Sprint(capID),
	}, nil
}

func (f *fakeStrategySvc) ListCapabilityIDsByStrategies(_ context.Context, ids []int64) ([]int64, error) {
	return []int64{100, 101}, nil
}

// TestStrategyTools_Info verifies that newStrategyTools returns 3 tools with the
// exact short names (scenes / caps / run).
func TestStrategyTools_Info(t *testing.T) {
	tools, err := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})
	if err != nil {
		t.Fatalf("newStrategyTools err=%v", err)
	}
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	wantNames := []string{"scenes", "caps", "run"}
	for i, wantName := range wantNames {
		info, err := tools[i].Info(context.Background())
		if err != nil {
			t.Fatalf("tools[%d].Info() err=%v", i, err)
		}
		if info.Name != wantName {
			t.Fatalf("tools[%d].Name = %q, want %q", i, info.Name, wantName)
		}
	}
}

// TestStrategyTools_ScenesDescContainsStrategyName verifies that the "scenes" tool
// description embeds the bound strategy's name, enabling autonomous discovery.
func TestStrategyTools_ScenesDescContainsStrategyName(t *testing.T) {
	tools, err := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})
	if err != nil {
		t.Fatalf("newStrategyTools err=%v", err)
	}

	info, err := tools[0].Info(context.Background())
	if err != nil {
		t.Fatalf("scenes.Info() err=%v", err)
	}
	// fakeStrategySvc.GetStrategy returns "Strategy-1" for id=1.
	if !strings.Contains(info.Desc, "Strategy-1") {
		t.Fatalf("scenes desc should contain bound strategy name %q, got: %s", "Strategy-1", info.Desc)
	}
}

// TestStrategyTools_ListScenariosInvoke verifies scenes returns short JSON keys.
func TestStrategyTools_ListScenariosInvoke(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[0].InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("scenes err=%v", err)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("scenes output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if len(rows) == 0 {
		t.Fatalf("expected at least one scenario, got empty array")
	}

	row := rows[0]
	for _, key := range []string{"id", "name", "desc", "n"} {
		if _, ok := row[key]; !ok {
			t.Fatalf("scenes row missing key %q: %v", key, row)
		}
	}
}

// TestStrategyTools_ListCapabilitiesInvoke verifies caps returns short JSON keys.
func TestStrategyTools_ListCapabilitiesInvoke(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[1].InvokableRun(context.Background(), `{"scene":"10"}`)
	if err != nil {
		t.Fatalf("caps err=%v", err)
	}

	// Must contain prompt type
	if !strings.Contains(out, `"type":"prompt"`) {
		t.Fatalf("caps should contain prompt capability, got: %s", out)
	}
	// Must contain knowledge type
	if !strings.Contains(out, `"type":"knowledge"`) {
		t.Fatalf("caps should contain knowledge capability, got: %s", out)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("caps output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	for _, row := range rows {
		for _, key := range []string{"id", "type", "name", "desc", "schema"} {
			if _, ok := row[key]; !ok {
				t.Fatalf("caps row missing key %q: %v", key, row)
			}
		}
	}

	// knowledge entry should have query in its schema
	var knowledgeRow map[string]any
	for _, row := range rows {
		if row["type"] == "knowledge" {
			knowledgeRow = row
			break
		}
	}
	if knowledgeRow == nil {
		t.Fatalf("no knowledge row found")
	}
	schemaRaw, _ := json.Marshal(knowledgeRow["schema"])
	if !strings.Contains(string(schemaRaw), "query") {
		t.Fatalf("knowledge schema should contain query, got: %s", schemaRaw)
	}

	// prompt alias name should be reflected
	if !strings.Contains(out, "MyPrompt") {
		t.Fatalf("caps prompt entry should use AliasName, got: %s", out)
	}
}

// TestStrategyTools_InvokeSentinel_Superseded is kept for historical context but skipped.
func TestStrategyTools_InvokeSentinel_Superseded(t *testing.T) {
	t.Skip("sentinel removed in BE9a — see TestStrategy_InvokePromptSuccess")
}

// TestStrategyTools_ListScenariosFilteredByStrategyID verifies that when
// strat param is provided, only that strategy's scenarios are returned.
func TestStrategyTools_ListScenariosFilteredByStrategyID(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1, 2},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[0].InvokableRun(context.Background(), `{"strat":"1"}`)
	if err != nil {
		t.Fatalf("scenes with strat filter err=%v", err)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("scenes output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	// All returned scenario ids should be multiples of strategy 1 (i.e. id=10)
	for _, row := range rows {
		if id, _ := row["id"].(float64); int64(id) != 10 {
			t.Fatalf("filtered scenes should only return scenario from strategy 1 (id=10), got %v", row)
		}
	}
}

// ---------------------------------------------------------------------------
// Security tests
// ---------------------------------------------------------------------------

// TestStrategy_ListScenariosIDORReject verifies that passing a strat that is
// NOT in the agent's bound set is rejected BEFORE any svc.ListScenarios call.
func TestStrategy_ListScenariosIDORReject(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1, 2}, // bound: 1 and 2 only
		svc:         &fakeStrategySvc{},
	})

	// strat=99 is foreign — must be rejected.
	out, err := tools[0].InvokableRun(context.Background(), `{"strat":"99"}`)
	if err != nil {
		t.Fatalf("scenes IDOR reject should not return Go error: %v", err)
	}
	const wantMsg = "Error: strategy_id is not bound to this agent"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("scenes IDOR: expected rejection message %q, got: %s", wantMsg, out)
	}
}

// TestStrategy_ListCapabilitiesIDORReject verifies that passing a scene that
// does NOT belong to any bound strategy is rejected before svc.ListCapabilities.
func TestStrategy_ListCapabilitiesIDORReject(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1}, // strategy 1 → scenario 10
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[1].InvokableRun(context.Background(), `{"scene":"99"}`)
	if err != nil {
		t.Fatalf("caps IDOR reject should not return Go error: %v", err)
	}
	const wantMsg = "Error: scenario_id is not accessible to this agent"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("caps IDOR: expected rejection message %q, got: %s", wantMsg, out)
	}
}

// TestStrategy_InvokePromptSuccess verifies that invoking a bound prompt capability
// returns the PromptContent via the short "cap" param.
func TestStrategy_InvokePromptSuccess(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// cap=100 is in the allowed set returned by fakeStrategySvc.ListCapabilityIDsByStrategies.
	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("run prompt capability should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error:") {
		t.Fatalf("run prompt capability should succeed, got: %s", out)
	}
	// fakeStrategySvc.ResolveCapability returns PromptContent containing the cap ID.
	if !strings.Contains(out, "100") {
		t.Fatalf("run prompt: expected prompt content containing capID, got: %s", out)
	}
}

// TestStrategy_InvokeAuthReject verifies that invoking a cap that is NOT in the
// agent's allowed set is rejected BEFORE svc.ResolveCapability is called.
func TestStrategy_InvokeAuthReject(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// cap=999 is NOT in {100, 101} returned by fakeStrategySvc.ListCapabilityIDsByStrategies.
	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"999"}`)
	if err != nil {
		t.Fatalf("run auth reject should not return Go error: %v", err)
	}
	const wantMsg = "Error: capability_id is not bound to this agent"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("run auth reject: expected %q, got: %s", wantMsg, out)
	}
}

// ---------------------------------------------------------------------------
// BE9b: execution routing tests (plugin / workflow / knowledge)
// ---------------------------------------------------------------------------

// fakeStrategySvcWithCap extends fakeStrategySvc to return a custom capability
// from ResolveCapability, allowing BE9b tests to control the dispatched type.
type fakeStrategySvcWithCap struct {
	fakeStrategySvc
	cap *entity.Capability
}

func (f *fakeStrategySvcWithCap) ResolveCapability(_ context.Context, _ int64) (*entity.Capability, error) {
	return f.cap, nil
}

// ---------------------------------------------------------------------------
// Plugin branch
// ---------------------------------------------------------------------------

// TestStrategy_InvokePlugin_RoutesCorrectly verifies that a plugin capability
// routes to crossplugin.DefaultSVC().ExecuteTool with PluginID=RefSubID, ToolID=RefID.
func TestStrategy_InvokePlugin_RoutesCorrectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const (
		wantPluginID int64 = 42
		wantToolID   int64 = 77
		wantResp           = "plugin result text"
	)

	mockPlugin := pluginmock.NewMockPluginService(ctrl)
	mockPlugin.EXPECT().
		ExecuteTool(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *pluginModel.ExecuteToolRequest, _ ...pluginModel.ExecuteToolOpt) (*pluginModel.ExecuteToolResponse, error) {
			if req.PluginID != wantPluginID {
				t.Errorf("PluginID = %d, want %d", req.PluginID, wantPluginID)
			}
			if req.ToolID != wantToolID {
				t.Errorf("ToolID = %d, want %d", req.ToolID, wantToolID)
			}
			return &pluginModel.ExecuteToolResponse{TrimmedResp: wantResp}, nil
		})

	// Wire mock as default SVC; restore original after test.
	origPlugin := crossplugin.DefaultSVC()
	crossplugin.SetDefaultSVC(mockPlugin)
	defer crossplugin.SetDefaultSVC(origPlugin)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{
			ID:       100,
			Type:     entity.CapabilityTypePlugin,
			RefSubID: wantPluginID, // plugin_id
			RefID:    wantToolID,   // tool_id
		},
	}

	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("run plugin should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error") {
		t.Fatalf("run plugin should succeed, got: %s", out)
	}
	if out != wantResp {
		t.Fatalf("run plugin: expected %q, got %q", wantResp, out)
	}
}

// TestStrategy_InvokePlugin_ErrorIsString verifies plugin errors are returned as
// model-visible strings, not Go errors.
func TestStrategy_InvokePlugin_ErrorIsString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPlugin := pluginmock.NewMockPluginService(ctrl)
	mockPlugin.EXPECT().
		ExecuteTool(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("upstream failure"))

	origPlugin := crossplugin.DefaultSVC()
	crossplugin.SetDefaultSVC(mockPlugin)
	defer crossplugin.SetDefaultSVC(origPlugin)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypePlugin},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("plugin error must be returned as string, not Go error: %v", err)
	}
	if !strings.Contains(out, "Error executing plugin") {
		t.Fatalf("expected friendly plugin error string, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Knowledge branch
// ---------------------------------------------------------------------------

// TestStrategy_InvokeKnowledge_RoutesCorrectly verifies that a knowledge capability
// parses the query from args, passes the capability's RefID as KnowledgeID,
// and returns the retrieved slice content.
func TestStrategy_InvokeKnowledge_RoutesCorrectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const (
		wantKnowledgeID int64 = 55
		wantQuery             = "what is the meaning of life"
		sliceText             = "42"
	)

	txt := sliceText
	mockKnowledge := knowledgemock.NewMockKnowledge(ctrl)
	mockKnowledge.EXPECT().
		Retrieve(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *knowledgeModel.RetrieveRequest) (*knowledgeModel.RetrieveResponse, error) {
			if req.Query != wantQuery {
				t.Errorf("query = %q, want %q", req.Query, wantQuery)
			}
			if len(req.KnowledgeIDs) != 1 || req.KnowledgeIDs[0] != wantKnowledgeID {
				t.Errorf("KnowledgeIDs = %v, want [%d]", req.KnowledgeIDs, wantKnowledgeID)
			}
			return &knowledgeModel.RetrieveResponse{
				RetrieveSlices: []*knowledgeModel.RetrieveSlice{
					{
						Slice: &knowledgeModel.Slice{
							RawContent: []*knowledgeModel.SliceContent{
								{Type: knowledgeModel.SliceContentTypeText, Text: &txt},
							},
						},
						Score: 0.9,
					},
				},
			}, nil
		})

	origKnowledge := crossknowledge.DefaultSVC()
	crossknowledge.SetDefaultSVC(mockKnowledge)
	defer crossknowledge.SetDefaultSVC(origKnowledge)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{
			ID:    100,
			Type:  entity.CapabilityTypeKnowledge,
			RefID: wantKnowledgeID,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	argsJSON, _ := json.Marshal(map[string]any{"cap": "100", "args": map[string]any{"query": wantQuery}})
	out, err := tools[2].InvokableRun(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("run knowledge should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error") {
		t.Fatalf("run knowledge should succeed, got: %s", out)
	}
	if !strings.Contains(out, sliceText) {
		t.Fatalf("run knowledge: expected slice content %q in output, got: %s", sliceText, out)
	}
}

// TestStrategy_InvokeKnowledge_MissingQuery verifies that omitting the query
// argument returns a friendly error string (not a Go error).
func TestStrategy_InvokeKnowledge_MissingQuery(t *testing.T) {
	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypeKnowledge, RefID: 55},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	// args has no "query" key.
	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100","args":{}}`)
	if err != nil {
		t.Fatalf("missing query must not return Go error: %v", err)
	}
	const wantMsg = "Error: knowledge capability requires a 'query' argument"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("missing query: expected %q, got: %s", wantMsg, out)
	}
}

// TestStrategy_InvokeKnowledge_ErrorIsString verifies Retrieve errors become friendly strings.
func TestStrategy_InvokeKnowledge_ErrorIsString(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockKnowledge := knowledgemock.NewMockKnowledge(ctrl)
	mockKnowledge.EXPECT().
		Retrieve(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("db timeout"))

	origKnowledge := crossknowledge.DefaultSVC()
	crossknowledge.SetDefaultSVC(mockKnowledge)
	defer crossknowledge.SetDefaultSVC(origKnowledge)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypeKnowledge, RefID: 55},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	argsJSON := `{"cap":"100","args":{"query":"test"}}`
	out, err := tools[2].InvokableRun(context.Background(), argsJSON)
	if err != nil {
		t.Fatalf("knowledge error must be returned as string, not Go error: %v", err)
	}
	if !strings.Contains(out, "Error retrieving knowledge") {
		t.Fatalf("expected friendly knowledge error string, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// Workflow branch
// ---------------------------------------------------------------------------

// fakeInvokableWorkflow is a minimal tool.InvokableTool used as a stand-in
// for the real workflow tool returned by WorkflowAsModelTool.
type fakeInvokableWorkflow struct {
	result string
	err    error
}

func (f *fakeInvokableWorkflow) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "fake_wf"}, nil
}
func (f *fakeInvokableWorkflow) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return f.result, f.err
}
func (f *fakeInvokableWorkflow) TerminatePlan() vo.TerminatePlan { return vo.UseAnswerContent }
func (f *fakeInvokableWorkflow) GetWorkflow() *workflowEntity.Workflow {
	return &workflowEntity.Workflow{}
}

// Verify fakeInvokableWorkflow satisfies workflowDomain.ToolFromWorkflow at compile time.
var _ workflowDomain.ToolFromWorkflow = (*fakeInvokableWorkflow)(nil)

// crossworkflowStub is a hand-written stub implementing crossworkflow.Workflow.
type crossworkflowStub struct {
	tools          []workflowDomain.ToolFromWorkflow
	wfErr          error
	wantWorkflowID int64
	wantQType      workflowModel.Locator
	wantVersion    string
	t              *testing.T
}

func (s *crossworkflowStub) WorkflowAsModelTool(_ context.Context, policies []*vo.GetPolicy) ([]workflowDomain.ToolFromWorkflow, error) {
	if s.t != nil && len(policies) > 0 {
		p := policies[0]
		if s.wantWorkflowID != 0 && p.ID != s.wantWorkflowID {
			s.t.Errorf("WorkflowAsModelTool policy.ID = %d, want %d", p.ID, s.wantWorkflowID)
		}
		if s.wantQType != 0 && p.QType != s.wantQType {
			s.t.Errorf("WorkflowAsModelTool policy.QType = %v, want %v", p.QType, s.wantQType)
		}
		if s.wantVersion != "" && p.Version != s.wantVersion {
			s.t.Errorf("WorkflowAsModelTool policy.Version = %q, want %q", p.Version, s.wantVersion)
		}
	}
	return s.tools, s.wfErr
}

func (s *crossworkflowStub) WithResumeToolWorkflow(_ *workflowEntity.ToolInterruptEvent, _ string, _ map[string]*workflowEntity.ToolInterruptEvent) einoCompose.Option {
	panic("crossworkflowStub.WithResumeToolWorkflow not implemented")
}
func (s *crossworkflowStub) ReleaseApplicationWorkflows(_ context.Context, _ int64, _ *vo.ReleaseWorkflowConfig) ([]*vo.ValidateIssue, error) {
	panic("crossworkflowStub.ReleaseApplicationWorkflows not implemented")
}
func (s *crossworkflowStub) GetWorkflowIDsByAppID(_ context.Context, _ int64) ([]int64, error) {
	panic("crossworkflowStub.GetWorkflowIDsByAppID not implemented")
}
func (s *crossworkflowStub) SyncExecuteWorkflow(_ context.Context, _ crossworkflow.ExecuteConfig, _ map[string]any) (*workflowEntity.WorkflowExecution, vo.TerminatePlan, error) {
	panic("crossworkflowStub.SyncExecuteWorkflow not implemented")
}
func (s *crossworkflowStub) AsyncExecute(_ context.Context, _ crossworkflow.ExecuteConfig, _ map[string]any) (int64, error) {
	panic("crossworkflowStub.AsyncExecute not implemented")
}
func (s *crossworkflowStub) GetExecution(_ context.Context, _ *workflowEntity.WorkflowExecution, _ bool) (*workflowEntity.WorkflowExecution, error) {
	panic("crossworkflowStub.GetExecution not implemented")
}
func (s *crossworkflowStub) StreamExecute(_ context.Context, _ crossworkflow.ExecuteConfig, _ map[string]any) (*schema.StreamReader[*workflowEntity.Message], error) {
	panic("crossworkflowStub.StreamExecute not implemented")
}
func (s *crossworkflowStub) WithExecuteConfig(_ crossworkflow.ExecuteConfig) einoCompose.Option {
	panic("crossworkflowStub.WithExecuteConfig not implemented")
}
func (s *crossworkflowStub) StreamResume(_ context.Context, _ *workflowEntity.ResumeRequest, _ crossworkflow.ExecuteConfig) (*schema.StreamReader[*workflowEntity.Message], error) {
	panic("crossworkflowStub.StreamResume not implemented")
}
func (s *crossworkflowStub) WithMessagePipe() (einoCompose.Option, *schema.StreamReader[*workflowEntity.Message], *schema.StreamWriter[*workflowEntity.Message]) {
	panic("crossworkflowStub.WithMessagePipe not implemented")
}
func (s *crossworkflowStub) InitApplicationDefaultConversationTemplate(_ context.Context, _ int64, _ int64, _ int64) error {
	panic("crossworkflowStub.InitApplicationDefaultConversationTemplate not implemented")
}

// TestStrategy_InvokeWorkflow_RoutesCorrectly verifies that a workflow capability
// routes to crossworkflow.DefaultSVC().WorkflowAsModelTool with the policy ID=RefID.
func TestStrategy_InvokeWorkflow_RoutesCorrectly(t *testing.T) {
	const (
		wantWorkflowID int64  = 88
		wantResult            = "workflow output"
	)

	fakeTool := &fakeInvokableWorkflow{result: wantResult}
	fakeSVC := &crossworkflowStub{
		tools:          []workflowDomain.ToolFromWorkflow{fakeTool},
		wantWorkflowID: wantWorkflowID,
		t:              t,
	}

	origWorkflow := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeSVC)
	defer crossworkflow.SetDefaultSVC(origWorkflow)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{
			ID:    100,
			Type:  entity.CapabilityTypeWorkflow,
			RefID: wantWorkflowID,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("run workflow should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error") {
		t.Fatalf("run workflow should succeed, got: %s", out)
	}
	if out != wantResult {
		t.Fatalf("run workflow: expected %q, got %q", wantResult, out)
	}
}

// TestStrategy_InvokeWorkflow_ErrorIsString verifies workflow errors become friendly strings.
func TestStrategy_InvokeWorkflow_ErrorIsString(t *testing.T) {
	fakeSVC := &crossworkflowStub{
		wfErr: fmt.Errorf("workflow not found"),
		t:     t,
	}

	origWorkflow := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeSVC)
	defer crossworkflow.SetDefaultSVC(origWorkflow)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypeWorkflow, RefID: 88},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("workflow error must be returned as string, not Go error: %v", err)
	}
	if !strings.Contains(out, "Error executing workflow") {
		t.Fatalf("expected friendly workflow error string, got: %s", out)
	}
}

// TestStrategy_InvokeWorkflow_PinnedVersion verifies that when cap.RefVersion != "",
// the GetPolicy uses FromSpecificVersion and passes the version string through.
func TestStrategy_InvokeWorkflow_PinnedVersion(t *testing.T) {
	const (
		wantWorkflowID int64 = 88
		wantVersion          = "v1.2.3"
		wantResult           = "pinned workflow output"
	)

	fakeTool := &fakeInvokableWorkflow{result: wantResult}
	fakeSVC := &crossworkflowStub{
		tools:          []workflowDomain.ToolFromWorkflow{fakeTool},
		wantWorkflowID: wantWorkflowID,
		wantQType:      workflowModel.FromSpecificVersion,
		wantVersion:    wantVersion,
		t:              t,
	}

	origWorkflow := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeSVC)
	defer crossworkflow.SetDefaultSVC(origWorkflow)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{
			ID:         100,
			Type:       entity.CapabilityTypeWorkflow,
			RefID:      wantWorkflowID,
			RefVersion: wantVersion,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("run pinned-version workflow should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error") {
		t.Fatalf("run pinned-version workflow should succeed, got: %s", out)
	}
	if out != wantResult {
		t.Fatalf("run pinned-version workflow: expected %q, got %q", wantResult, out)
	}
}

// TestStrategy_InvokePlugin_DraftExecScene verifies that when agentIdentity.IsDraft=true,
// the ExecScene is ExecSceneOfDraftAgent.
func TestStrategy_InvokePlugin_DraftExecScene(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const wantResp = "draft plugin result"

	mockPlugin := pluginmock.NewMockPluginService(ctrl)
	mockPlugin.EXPECT().
		ExecuteTool(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *pluginModel.ExecuteToolRequest, _ ...pluginModel.ExecuteToolOpt) (*pluginModel.ExecuteToolResponse, error) {
			if req.ExecScene != pluginModel.ExecSceneOfDraftAgent {
				t.Errorf("ExecScene = %v, want ExecSceneOfDraftAgent", req.ExecScene)
			}
			return &pluginModel.ExecuteToolResponse{TrimmedResp: wantResp}, nil
		})

	origPlugin := crossplugin.DefaultSVC()
	crossplugin.SetDefaultSVC(mockPlugin)
	defer crossplugin.SetDefaultSVC(origPlugin)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypePlugin, RefSubID: 42, RefID: 77},
	}

	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:   []int64{1},
		svc:           svc,
		agentIdentity: &agentEntity.AgentIdentity{IsDraft: true},
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"cap":"100"}`)
	if err != nil {
		t.Fatalf("run draft plugin should not return Go error: %v", err)
	}
	if out != wantResp {
		t.Fatalf("run draft plugin: expected %q, got %q", wantResp, out)
	}
}
