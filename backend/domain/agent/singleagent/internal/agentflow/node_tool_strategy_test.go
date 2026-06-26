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
// Bound strategy: 1, with two scenarios (IDs 10 and 20, in that sort order).
// Scenario 10 has capabilities 100 (prompt) and 101 (knowledge).
// Scenario 20 has capability 200 (workflow).
type fakeStrategySvc struct{}

func (f *fakeStrategySvc) GetStrategy(_ context.Context, id int64) (*entity.Strategy, error) {
	return &entity.Strategy{
		ID:          id,
		Name:        fmt.Sprintf("Strategy-%d", id),
		Description: fmt.Sprintf("desc for strategy %d", id),
	}, nil
}

func (f *fakeStrategySvc) ListScenarios(_ context.Context, strategyID int64) ([]*entity.Scenario, error) {
	// Two scenarios — returned in unsorted order to prove stable sort works.
	return []*entity.Scenario{
		{ID: 20, StrategyID: strategyID, Name: "ScenarioB", Description: "desc B", SortOrder: 2},
		{ID: 10, StrategyID: strategyID, Name: "ScenarioA", Description: "desc A", SortOrder: 1},
	}, nil
}

func (f *fakeStrategySvc) ListCapabilities(_ context.Context, scenarioID int64) ([]*entity.Capability, error) {
	switch scenarioID {
	case 10:
		return []*entity.Capability{
			{
				ID: 100, ScenarioID: 10, Type: entity.CapabilityTypePrompt,
				AliasName: "MyPrompt", AliasDescription: "prompt alias desc",
				PromptContent: "Hello from prompt cap 100", SortOrder: 1,
			},
			{
				ID: 101, ScenarioID: 10, Type: entity.CapabilityTypeKnowledge,
				AliasName: "", AliasDescription: "", RefID: 101, SortOrder: 2,
			},
		}, nil
	case 20:
		return []*entity.Capability{
			{
				ID: 200, ScenarioID: 20, Type: entity.CapabilityTypeWorkflow,
				AliasName: "MyWorkflow", AliasDescription: "workflow cap", SortOrder: 1,
			},
		}, nil
	default:
		return nil, nil
	}
}

func (f *fakeStrategySvc) ResolveCapability(_ context.Context, capID int64) (*entity.Capability, error) {
	return &entity.Capability{
		ID:            capID,
		Type:          entity.CapabilityTypePrompt,
		PromptContent: "Hello, I am the prompt content for cap " + fmt.Sprint(capID),
	}, nil
}

func (f *fakeStrategySvc) ListCapabilityIDsByStrategies(_ context.Context, ids []int64) ([]int64, error) {
	return []int64{100, 101, 200}, nil
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

// TestStrategyTools_ScenesNoParams verifies that scenes takes no parameters
// (the bound strategy provides the filter automatically).
func TestStrategyTools_ScenesNoParams(t *testing.T) {
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
	// Verify no param named "strat" exists by serializing the schema to JSON.
	js, schErr := info.ParamsOneOf.ToJSONSchema()
	if schErr != nil {
		t.Fatalf("scenes.Info().ParamsOneOf.ToJSONSchema() err=%v", schErr)
	}
	jsBytes, _ := json.Marshal(js)
	if strings.Contains(string(jsBytes), `"strat"`) {
		t.Fatalf("scenes must NOT have a 'strat' parameter in the new ordinal design, schema: %s", jsBytes)
	}
}

// TestStrategyTools_ListScenariosInvoke verifies scenes returns ordinal ids
// and stable ordering (SortOrder ASC → ScenarioA=1, ScenarioB=2).
func TestStrategyTools_ListScenariosInvoke(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scenes takes no params — pass empty JSON.
	out, err := tools[0].InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("scenes err=%v", err)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("scenes output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 scenarios, got %d: %s", len(rows), out)
	}

	// First row must be ScenarioA (SortOrder=1, id=10 < 20)
	if rows[0]["name"] != "ScenarioA" {
		t.Fatalf("expected first scene to be ScenarioA (SortOrder=1), got: %v", rows[0]["name"])
	}
	// Ordinal ids must be 1 and 2.
	if id, _ := rows[0]["id"].(float64); int(id) != 1 {
		t.Fatalf("first scenario ordinal id must be 1, got: %v", rows[0]["id"])
	}
	if id, _ := rows[1]["id"].(float64); int(id) != 2 {
		t.Fatalf("second scenario ordinal id must be 2, got: %v", rows[1]["id"])
	}

	for _, key := range []string{"id", "name", "desc", "n"} {
		if _, ok := rows[0][key]; !ok {
			t.Fatalf("scenes row missing key %q: %v", key, rows[0])
		}
	}
}

// TestStrategyTools_ListCapabilitiesInvoke verifies caps returns ordinal ids
// when given scene=1 (ScenarioA, ordinal 1).
func TestStrategyTools_ListCapabilitiesInvoke(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scene=1 → ScenarioA (real ID 10).
	out, err := tools[1].InvokableRun(context.Background(), `{"scene":1}`)
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
	if len(rows) != 2 {
		t.Fatalf("expected 2 capabilities for scene 1, got %d: %s", len(rows), out)
	}

	// Ordinal ids must be 1 and 2.
	if id, _ := rows[0]["id"].(float64); int(id) != 1 {
		t.Fatalf("first cap ordinal id must be 1, got: %v", rows[0]["id"])
	}
	if id, _ := rows[1]["id"].(float64); int(id) != 2 {
		t.Fatalf("second cap ordinal id must be 2, got: %v", rows[1]["id"])
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

// TestStrategyTools_OrdinalResolution_Scene2 verifies that scene=2 resolves to
// ScenarioB (ordinal 2 in stable sort), not ScenarioA.
func TestStrategyTools_OrdinalResolution_Scene2(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scene=2 → ScenarioB (real ID 20), which has one workflow cap.
	out, err := tools[1].InvokableRun(context.Background(), `{"scene":2}`)
	if err != nil {
		t.Fatalf("caps scene=2 err=%v", err)
	}
	if !strings.Contains(out, `"type":"workflow"`) {
		t.Fatalf("scene=2 (ScenarioB) should return workflow cap, got: %s", out)
	}
	if strings.Contains(out, `"type":"prompt"`) {
		t.Fatalf("scene=2 (ScenarioB) must NOT contain prompt cap, got: %s", out)
	}
}

// TestStrategyTools_InvokeSentinel_Superseded is kept for historical context but skipped.
func TestStrategyTools_InvokeSentinel_Superseded(t *testing.T) {
	t.Skip("sentinel removed in BE9a — see TestStrategy_InvokePromptSuccess")
}

// ---------------------------------------------------------------------------
// Security / bounds tests
// ---------------------------------------------------------------------------

// TestStrategy_CapsOutOfRange verifies that scene out of range returns a friendly error.
func TestStrategy_CapsOutOfRange(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scene=99 is beyond 2 scenarios.
	out, err := tools[1].InvokableRun(context.Background(), `{"scene":99}`)
	if err != nil {
		t.Fatalf("caps out-of-range should not return Go error: %v", err)
	}
	if !strings.Contains(out, "out of range") {
		t.Fatalf("caps out-of-range: expected 'out of range' message, got: %s", out)
	}
}

// TestStrategy_RunOutOfRangeScene verifies scene out of range returns friendly error.
func TestStrategy_RunOutOfRangeScene(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":99,"cap":1}`)
	if err != nil {
		t.Fatalf("run out-of-range scene should not return Go error: %v", err)
	}
	if !strings.Contains(out, "out of range") {
		t.Fatalf("run out-of-range scene: expected 'out of range' message, got: %s", out)
	}
}

// TestStrategy_RunOutOfRangeCap verifies cap out of range returns friendly error.
func TestStrategy_RunOutOfRangeCap(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scene=1 is valid; cap=99 is beyond the 2 capabilities in ScenarioA.
	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":99}`)
	if err != nil {
		t.Fatalf("run out-of-range cap should not return Go error: %v", err)
	}
	if !strings.Contains(out, "out of range") {
		t.Fatalf("run out-of-range cap: expected 'out of range' message, got: %s", out)
	}
}

// TestStrategy_InvokePromptSuccess verifies that run with scene=1,cap=1 resolves to
// the first capability in ScenarioA (MyPrompt / CapabilityTypePrompt) and returns
// its PromptContent directly — no long IDs involved.
func TestStrategy_InvokePromptSuccess(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scene=1 → ScenarioA; cap=1 → capability 100 (MyPrompt, prompt type).
	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
	if err != nil {
		t.Fatalf("run prompt capability should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error:") {
		t.Fatalf("run prompt capability should succeed, got: %s", out)
	}
	// fakeStrategySvc capability 100 has PromptContent "Hello from prompt cap 100".
	if !strings.Contains(out, "Hello from prompt cap 100") {
		t.Fatalf("run prompt: expected PromptContent, got: %s", out)
	}
}

// TestStrategy_OrdinalResolution_Cap2 verifies that scene=1,cap=2 resolves to
// the second capability in ScenarioA (capability 101, knowledge type).
func TestStrategy_OrdinalResolution_Cap2(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const wantKnowledgeID int64 = 101
	const wantQuery = "test query"
	txt := "result text"

	mockKnowledge := knowledgemock.NewMockKnowledge(ctrl)
	mockKnowledge.EXPECT().
		Retrieve(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *knowledgeModel.RetrieveRequest) (*knowledgeModel.RetrieveResponse, error) {
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

	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// scene=1 → ScenarioA; cap=2 → capability 101 (knowledge).
	argsJSON, _ := json.Marshal(map[string]any{
		"scene": 1, "cap": 2,
		"args": map[string]any{"query": wantQuery},
	})
	out, err := tools[2].InvokableRun(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("run scene=1,cap=2 knowledge should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error") {
		t.Fatalf("run knowledge should succeed, got: %s", out)
	}
	if !strings.Contains(out, "result text") {
		t.Fatalf("run knowledge: expected slice content in output, got: %s", out)
	}
}

// ---------------------------------------------------------------------------
// BE9b: execution routing tests (plugin / workflow / knowledge)
// ---------------------------------------------------------------------------

// fakeStrategySvcWithCap extends fakeStrategySvc to allow overriding the
// capability returned for scenario 10 cap 1, enabling per-type dispatch tests.
type fakeStrategySvcWithCap struct {
	fakeStrategySvc
	cap *entity.Capability
}

// ListCapabilities overrides to return the test cap for scenario 10.
func (f *fakeStrategySvcWithCap) ListCapabilities(_ context.Context, scenarioID int64) ([]*entity.Capability, error) {
	if scenarioID == 10 && f.cap != nil {
		return []*entity.Capability{f.cap}, nil
	}
	return f.fakeStrategySvc.ListCapabilities(nil, scenarioID) //nolint:staticcheck
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
			SortOrder: 1,
		},
	}

	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	// scene=1 → ScenarioA; cap=1 → the plugin capability.
	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
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
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypePlugin, SortOrder: 1},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
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
			ID:       100,
			Type:     entity.CapabilityTypeKnowledge,
			RefID:    wantKnowledgeID,
			SortOrder: 1,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	argsJSON, _ := json.Marshal(map[string]any{"scene": 1, "cap": 1, "args": map[string]any{"query": wantQuery}})
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
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypeKnowledge, RefID: 55, SortOrder: 1},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	// args has no "query" key.
	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1,"args":{}}`)
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
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypeKnowledge, RefID: 55, SortOrder: 1},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	argsJSON := `{"scene":1,"cap":1,"args":{"query":"test"}}`
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
			SortOrder: 1,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:     []int64{1},
		svc:             svc,
		forceToolReturn: true, // suppress marker so test can assert plain output
	})

	// scene=1, cap=1 → the workflow capability.
	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
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
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypeWorkflow, RefID: 88, SortOrder: 1},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         svc,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
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
			SortOrder:  1,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:     []int64{1},
		svc:             svc,
		forceToolReturn: true, // suppress marker so test can assert plain output
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
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

// ---------------------------------------------------------------------------
// ReturnDirectly sentinel tests (BE9c)
// ---------------------------------------------------------------------------

// fakeInvokableWorkflowReturnDirect is like fakeInvokableWorkflow but with
// TerminatePlan==UseAnswerContent (returnDirectly).
type fakeInvokableWorkflowReturnDirect struct {
	result string
}

func (f *fakeInvokableWorkflowReturnDirect) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "fake_wf_direct"}, nil
}
func (f *fakeInvokableWorkflowReturnDirect) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return f.result, nil
}
func (f *fakeInvokableWorkflowReturnDirect) TerminatePlan() vo.TerminatePlan {
	return vo.UseAnswerContent
}
func (f *fakeInvokableWorkflowReturnDirect) GetWorkflow() *workflowEntity.Workflow {
	return &workflowEntity.Workflow{}
}

var _ workflowDomain.ToolFromWorkflow = (*fakeInvokableWorkflowReturnDirect)(nil)

// TestStrategy_WorkflowReturnDirectly verifies that when a workflow capability has
// TerminatePlan==UseAnswerContent and forceToolReturn=false, run prefixes the result
// with StrategyReturnDirectlyMarker so the callback can route it directly to the user.
func TestStrategy_WorkflowReturnDirectly(t *testing.T) {
	const wantResult = "direct answer from workflow"

	fakeTool := &fakeInvokableWorkflowReturnDirect{result: wantResult}
	fakeSVC := &crossworkflowStub{tools: []workflowDomain.ToolFromWorkflow{fakeTool}}

	origWorkflow := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeSVC)
	defer crossworkflow.SetDefaultSVC(origWorkflow)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{
			ID:       100,
			Type:     entity.CapabilityTypeWorkflow,
			RefID:    88,
			SortOrder: 1,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:     []int64{1},
		svc:             svc,
		forceToolReturn: false,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
	if err != nil {
		t.Fatalf("run returnDirectly workflow should not return Go error: %v", err)
	}
	if !strings.HasPrefix(out, StrategyReturnDirectlyMarker) {
		t.Fatalf("run: expected result prefixed with StrategyReturnDirectlyMarker, got: %q", out)
	}
	withoutMarker := strings.TrimPrefix(out, StrategyReturnDirectlyMarker)
	if withoutMarker != wantResult {
		t.Fatalf("run: content after marker = %q, want %q", withoutMarker, wantResult)
	}
}

// TestStrategy_WorkflowReturnDirectly_ForceToolReturn verifies that when
// forceToolReturn=true, a returnDirectly workflow is NOT marked — its result is
// returned to the model as a normal tool response.
func TestStrategy_WorkflowReturnDirectly_ForceToolReturn(t *testing.T) {
	const wantResult = "answer for model"

	fakeTool := &fakeInvokableWorkflowReturnDirect{result: wantResult}
	fakeSVC := &crossworkflowStub{tools: []workflowDomain.ToolFromWorkflow{fakeTool}}

	origWorkflow := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeSVC)
	defer crossworkflow.SetDefaultSVC(origWorkflow)

	svc := &fakeStrategySvcWithCap{
		cap: &entity.Capability{
			ID:       100,
			Type:     entity.CapabilityTypeWorkflow,
			RefID:    88,
			SortOrder: 1,
		},
	}
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:     []int64{1},
		svc:             svc,
		forceToolReturn: true,
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
	if err != nil {
		t.Fatalf("run forceToolReturn workflow should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, StrategyReturnDirectlyMarker) {
		t.Fatalf("run with forceToolReturn=true: result must NOT be marked, got: %q", out)
	}
	if out != wantResult {
		t.Fatalf("run forceToolReturn: expected %q, got %q", wantResult, out)
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
		cap: &entity.Capability{ID: 100, Type: entity.CapabilityTypePlugin, RefSubID: 42, RefID: 77, SortOrder: 1},
	}

	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:   []int64{1},
		svc:           svc,
		agentIdentity: &agentEntity.AgentIdentity{IsDraft: true},
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"scene":1,"cap":1}`)
	if err != nil {
		t.Fatalf("run draft plugin should not return Go error: %v", err)
	}
	if out != wantResp {
		t.Fatalf("run draft plugin: expected %q, got %q", wantResp, out)
	}
}
