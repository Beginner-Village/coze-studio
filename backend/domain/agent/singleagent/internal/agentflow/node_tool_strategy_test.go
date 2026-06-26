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

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

// fakeStrategySvc implements crossstrategy.StrategyService for tests.
// Bound strategies: [1, 2]. Each strategy has one scenario (ID = strategyID*10).
// Scenarios 10 and 20 are valid; scenario 99 is foreign/unbound.
// Capabilities 100, 101 belong to scenario 10 (strategy 1).
type fakeStrategySvc struct{}

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
// exact names required by the progressive-disclosure design.
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

	wantNames := []string{
		"strategy_list_scenarios",
		"strategy_list_capabilities",
		"strategy_invoke_capability",
	}
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

// TestStrategyTools_ListScenariosInvoke verifies list_scenarios returns the
// fake scenarios serialised with the required JSON keys.
func TestStrategyTools_ListScenariosInvoke(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[0].InvokableRun(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("list_scenarios err=%v", err)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("list_scenarios output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if len(rows) == 0 {
		t.Fatalf("expected at least one scenario, got empty array")
	}

	row := rows[0]
	for _, key := range []string{"strategy_id", "scenario_id", "name", "description", "capability_count"} {
		if _, ok := row[key]; !ok {
			t.Fatalf("list_scenarios row missing key %q: %v", key, row)
		}
	}
}

// TestStrategyTools_ListCapabilitiesInvoke verifies list_capabilities maps
// prompt and knowledge capabilities with the correct type and input_schema.
func TestStrategyTools_ListCapabilitiesInvoke(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[1].InvokableRun(context.Background(), `{"scenario_id":"10"}`)
	if err != nil {
		t.Fatalf("list_capabilities err=%v", err)
	}

	// Must contain prompt type
	if !strings.Contains(out, `"type":"prompt"`) {
		t.Fatalf("list_capabilities should contain prompt capability, got: %s", out)
	}
	// Must contain knowledge type
	if !strings.Contains(out, `"type":"knowledge"`) {
		t.Fatalf("list_capabilities should contain knowledge capability, got: %s", out)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("list_capabilities output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	for _, row := range rows {
		for _, key := range []string{"capability_id", "type", "name", "description", "input_schema"} {
			if _, ok := row[key]; !ok {
				t.Fatalf("list_capabilities row missing key %q: %v", key, row)
			}
		}
	}

	// knowledge entry should have query in its input_schema
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
	schemaRaw, _ := json.Marshal(knowledgeRow["input_schema"])
	if !strings.Contains(string(schemaRaw), "query") {
		t.Fatalf("knowledge input_schema should contain query, got: %s", schemaRaw)
	}

	// prompt alias name should be reflected
	if !strings.Contains(out, "MyPrompt") {
		t.Fatalf("list_capabilities prompt entry should use AliasName, got: %s", out)
	}
}

// TestStrategyTools_InvokeSentinel was the BE8 sentinel test. After BE9a, the
// invoke tool returns real results. This test is superseded by the BE9a security
// tests below; it's kept but renamed to avoid confusion.
// (Disabled — sentinel string no longer produced after BE9a implementation.)
func TestStrategyTools_InvokeSentinel_Superseded(t *testing.T) {
	t.Skip("sentinel removed in BE9a — see TestStrategy_InvokePromptSuccess")
}

// TestStrategyTools_ListScenariosFilteredByStrategyID verifies that when
// strategy_id param is provided, only that strategy's scenarios are returned.
// The bound strategies are [1, 2], strategy_id=1 is in-bound → allowed.
func TestStrategyTools_ListScenariosFilteredByStrategyID(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1, 2},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[0].InvokableRun(context.Background(), `{"strategy_id":"1"}`)
	if err != nil {
		t.Fatalf("list_scenarios with strategy_id filter err=%v", err)
	}

	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &rows); jsonErr != nil {
		t.Fatalf("list_scenarios output not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	// should only have scenarios for strategy 1
	for _, row := range rows {
		if sid, _ := row["strategy_id"].(float64); int64(sid) != 1 {
			t.Fatalf("filtered list_scenarios should only return strategy_id=1, got %v", row)
		}
	}
}

// ---------------------------------------------------------------------------
// BE9a Security tests (TDD: written before implementation)
// ---------------------------------------------------------------------------

// TestStrategy_ListScenariosIDORReject verifies that passing a strategy_id that
// is NOT in the agent's bound set is rejected BEFORE any svc.ListScenarios call.
func TestStrategy_ListScenariosIDORReject(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1, 2}, // bound: 1 and 2 only
		svc:         &fakeStrategySvc{},
	})

	// strategy_id=99 is foreign — must be rejected.
	out, err := tools[0].InvokableRun(context.Background(), `{"strategy_id":"99"}`)
	if err != nil {
		t.Fatalf("list_scenarios IDOR reject should not return Go error: %v", err)
	}
	const wantMsg = "Error: strategy_id is not bound to this agent"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("list_scenarios IDOR: expected rejection message %q, got: %s", wantMsg, out)
	}
}

// TestStrategy_ListCapabilitiesIDORReject verifies that passing a scenario_id
// that does NOT belong to any bound strategy is rejected before svc.ListCapabilities.
// Bound strategies [1] → valid scenario IDs = {10}. Scenario 99 is foreign.
func TestStrategy_ListCapabilitiesIDORReject(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1}, // strategy 1 → scenario 10
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[1].InvokableRun(context.Background(), `{"scenario_id":"99"}`)
	if err != nil {
		t.Fatalf("list_capabilities IDOR reject should not return Go error: %v", err)
	}
	const wantMsg = "Error: scenario_id is not accessible to this agent"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("list_capabilities IDOR: expected rejection message %q, got: %s", wantMsg, out)
	}
}

// TestStrategy_InvokePromptSuccess verifies that invoking a bound prompt capability
// returns the PromptContent (the primary BE9a success path).
func TestStrategy_InvokePromptSuccess(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// capID=100 is in the allowed set returned by fakeStrategySvc.ListCapabilityIDsByStrategies.
	out, err := tools[2].InvokableRun(context.Background(), `{"capability_id":"100"}`)
	if err != nil {
		t.Fatalf("invoke prompt capability should not return Go error: %v", err)
	}
	if strings.HasPrefix(out, "Error:") {
		t.Fatalf("invoke prompt capability should succeed, got: %s", out)
	}
	// fakeStrategySvc.ResolveCapability returns PromptContent containing the cap ID.
	if !strings.Contains(out, "100") {
		t.Fatalf("invoke prompt: expected prompt content containing capID, got: %s", out)
	}
}

// TestStrategy_InvokeAuthReject verifies that invoking a capability_id that is
// NOT in the agent's allowed set is rejected BEFORE svc.ResolveCapability is called.
func TestStrategy_InvokeAuthReject(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	// capID=999 is NOT in {100, 101} returned by fakeStrategySvc.ListCapabilityIDsByStrategies.
	out, err := tools[2].InvokableRun(context.Background(), `{"capability_id":"999"}`)
	if err != nil {
		t.Fatalf("invoke auth reject should not return Go error: %v", err)
	}
	const wantMsg = "Error: capability_id is not bound to this agent"
	if !strings.Contains(out, wantMsg) {
		t.Fatalf("invoke auth reject: expected %q, got: %s", wantMsg, out)
	}
}
