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
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

// fakeStrategySvc implements crossstrategy.StrategyService for tests.
type fakeStrategySvc struct{}

func (f *fakeStrategySvc) ListScenarios(_ context.Context, strategyID int64) ([]*entity.Scenario, error) {
	return []*entity.Scenario{
		{ID: 10, StrategyID: strategyID, Name: "ScenarioA", Description: "desc A", SortOrder: 1},
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
	return &entity.Capability{ID: capID, Type: entity.CapabilityTypePrompt}, nil
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
	if !contains(out, `"type":"prompt"`) {
		t.Fatalf("list_capabilities should contain prompt capability, got: %s", out)
	}
	// Must contain knowledge type
	if !contains(out, `"type":"knowledge"`) {
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
	if !contains(string(schemaRaw), "query") {
		t.Fatalf("knowledge input_schema should contain query, got: %s", schemaRaw)
	}

	// prompt alias name should be reflected
	if !contains(out, "MyPrompt") {
		t.Fatalf("list_capabilities prompt entry should use AliasName, got: %s", out)
	}
}

// TestStrategyTools_InvokeSentinel verifies that strategy_invoke_capability
// returns the BE9 sentinel string and no error (sentinel signals deferred wiring).
func TestStrategyTools_InvokeSentinel(t *testing.T) {
	tools, _ := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs: []int64{1},
		svc:         &fakeStrategySvc{},
	})

	out, err := tools[2].InvokableRun(context.Background(), `{"capability_id":"100"}`)
	if err != nil {
		t.Fatalf("invoke_capability should not return error (sentinel): %v", err)
	}
	if !contains(out, "BE9") && !contains(out, "not yet") {
		t.Fatalf("invoke_capability sentinel not found in output: %s", out)
	}
}

// TestStrategyTools_ListScenariosFilteredByStrategyID verifies that when
// strategy_id param is provided, only that strategy's scenarios are returned.
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

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
