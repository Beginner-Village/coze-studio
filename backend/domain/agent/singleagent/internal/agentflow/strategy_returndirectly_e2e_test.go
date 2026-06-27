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

// This file is an end-to-end regression guard for strategy returnDirectly using the
// REAL invokeCapabilityTool ("run") inside the production react recompose topology.
// It asserts the four GOAL behaviours:
//   - 返回文本 (UseAnswerContent) workflow via run, forceToolReturn=false → loop TERMINATES
//     (model called once) and the final message carries StrategyReturnDirectlyMarker so
//     the callback renders the workflow text directly (直出) — like a native bound workflow.
//   - Stream variant (production runs Stream) → same termination.
//   - forceToolReturn=true (off-switch) → loop CONTINUES (model called twice), no marker.
//   - 返回变量 (ReturnVariables) workflow → loop CONTINUES (chaining preserved), no marker.

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
	workflowDomain "github.com/ynet-dev/ynet-studio/backend/domain/workflow"
	workflowEntity "github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
)

// strategyE2EModel emits a run(scene=2,cap=1) tool call first (scenario B = the
// workflow cap in fakeStrategySvc), then a plain answer. Counting model calls
// detects whether returnDirectly terminated the loop (1) or it looped (2+).
type strategyE2EModel struct{ calls *int32 }

func (m *strategyE2EModel) next() *schema.Message {
	if atomic.AddInt32(m.calls, 1) == 1 {
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID:       "call_run_1",
			Function: schema.FunctionCall{Name: "run", Arguments: `{"scene":2,"cap":1}`},
		}})
	}
	return schema.AssistantMessage("MODEL_LOOPED_TEXT", nil)
}
func (m *strategyE2EModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return m.next(), nil
}
func (m *strategyE2EModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Send(m.next(), nil)
	sw.Close()
	return sr, nil
}
func (m *strategyE2EModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

// buildStrategyE2E builds the REAL invokeCapabilityTool inside the production react
// recompose topology, wired to a fake workflow svc that returns a UseAnswerContent
// (返回文本) workflow. forceToolReturn toggles the off-switch.
func buildStrategyE2E(t *testing.T, forceToolReturn bool, wfTerminate bool) (compose.Runnable[*AgentRequest, *schema.Message], *int32) {
	t.Helper()
	ctx := context.Background()

	// scenario 20 (ordinal 2) cap 1 is a workflow cap in fakeStrategySvc.
	svc := &fakeStrategySvc{}

	// Wire a fake workflow service returning a returnDirectly (or not) workflow.
	var wfTool workflowDomain.ToolFromWorkflow
	if wfTerminate {
		wfTool = &fakeInvokableWorkflowReturnDirect{result: "BALANCE_IS_42"} // UseAnswerContent
	} else {
		wfTool = &fakeInvokableWorkflow{result: "BALANCE_IS_42"} // also UseAnswerContent in this file; use stub below
	}
	fakeWF := &crossworkflowStub{tools: []workflowDomain.ToolFromWorkflow{wfTool}}
	orig := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeWF)
	t.Cleanup(func() { crossworkflow.SetDefaultSVC(orig) })

	strategyTools, err := newStrategyTools(ctx, &strategyConfig{
		strategyIDs:     []int64{1},
		svc:             svc,
		forceToolReturn: forceToolReturn,
	})
	if err != nil {
		t.Fatalf("newStrategyTools err=%v", err)
	}

	calls := int32(0)
	agentTools := make([]tool.BaseTool, 0, len(strategyTools))
	for _, st := range strategyTools {
		agentTools = append(agentTools, st)
	}
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &strategyE2EModel{calls: &calls},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               agentTools,
			ExecuteSequentially: true,
		},
		ToolReturnDirectly: map[string]struct{}{},
		ModelNodeName:      keyOfReActAgentChatModel,
		ToolsNodeName:      keyOfReActAgentToolsNode,
		MaxStep:            20,
	})
	if err != nil {
		t.Fatalf("react.NewAgent err=%v", err)
	}
	agentGraph, agentNodeOpts := agent.ExportGraph()

	g := compose.NewGraph[*AgentRequest, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) *AgentState { return &AgentState{} }))
	_ = g.AddLambdaNode("seed",
		compose.InvokableLambda[*AgentRequest, []*schema.Message](func(_ context.Context, _ *AgentRequest) ([]*schema.Message, error) {
			return []*schema.Message{schema.UserMessage("查余额")}, nil
		}))
	agentNodeOpts = append(agentNodeOpts, compose.WithNodeName(keyOfReActAgent))
	_ = g.AddGraphNode(keyOfReActAgent, agentGraph, agentNodeOpts...)
	_ = g.AddEdge(compose.START, "seed")
	_ = g.AddEdge("seed", keyOfReActAgent)
	_ = g.AddEdge(keyOfReActAgent, compose.END)

	runner, err := g.Compile(ctx, compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		t.Fatalf("compile err=%v", err)
	}
	return runner, &calls
}

// TestStrategyE2E_ReturnDirectly_Terminates: the REAL run tool executing a 返回文本
// workflow with forceToolReturn=false must TERMINATE the react loop (model called once)
// and the final message must carry the marker (rendering half).
func TestStrategyE2E_ReturnDirectly_Terminates(t *testing.T) {
	runner, calls := buildStrategyE2E(t, false /*forceToolReturn*/, true /*wfTerminate*/)
	out, err := runner.Invoke(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Invoke err=%v", err)
	}
	got := atomic.LoadInt32(calls)
	t.Logf("[e2e-returndirect] model calls=%d, final=%q", got, out.Content)
	if got != 1 {
		t.Fatalf("[e2e-returndirect] EXPECTED 1 model call (terminate) but got %d; final=%q", got, out.Content)
	}
	if !strings.Contains(out.Content, "BALANCE_IS_42") {
		t.Fatalf("[e2e-returndirect] final should contain workflow text, got %q", out.Content)
	}
	if !strings.HasPrefix(out.Content, StrategyReturnDirectlyMarker) {
		t.Fatalf("[e2e-returndirect] final should carry the marker for the callback to render, got %q", out.Content)
	}
}

// Stream variant (production uses Stream).
func TestStrategyE2E_ReturnDirectly_Terminates_Stream(t *testing.T) {
	runner, calls := buildStrategyE2E(t, false, true)
	sr, err := runner.Stream(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Stream err=%v", err)
	}
	drainStream(sr)
	got := atomic.LoadInt32(calls)
	t.Logf("[e2e-returndirect-stream] model calls=%d", got)
	if got != 1 {
		t.Fatalf("[e2e-returndirect-stream] EXPECTED 1 model call (terminate) but got %d", got)
	}
}

// TestStrategyE2E_ForceToolReturn_Loops: with forceToolReturn=true the off-switch
// must keep the loop going (no marker, no termination) — model called TWICE.
func TestStrategyE2E_ForceToolReturn_Loops(t *testing.T) {
	runner, calls := buildStrategyE2E(t, true /*forceToolReturn*/, true)
	out, err := runner.Invoke(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Invoke err=%v", err)
	}
	got := atomic.LoadInt32(calls)
	t.Logf("[e2e-force] model calls=%d, final=%q", got, out.Content)
	if got != 2 {
		t.Fatalf("[e2e-force] EXPECTED 2 model calls (loop, off-switch) but got %d; final=%q", got, out.Content)
	}
	if strings.Contains(out.Content, StrategyReturnDirectlyMarker) {
		t.Fatalf("[e2e-force] final must NOT carry marker under forceToolReturn, got %q", out.Content)
	}
}

// capabilityType compile-time anchor to avoid unused import of entity in some builds.
var _ = entity.CapabilityTypeWorkflow

// fakeReturnVariablesWorkflow is a 返回变量 (ReturnVariables) workflow tool: it must
// NOT trigger returnDirectly, so the react loop keeps going (chaining).
type fakeReturnVariablesWorkflow struct{ result string }

func (f *fakeReturnVariablesWorkflow) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "fake_wf_vars"}, nil
}
func (f *fakeReturnVariablesWorkflow) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return f.result, nil
}
func (f *fakeReturnVariablesWorkflow) TerminatePlan() vo.TerminatePlan { return vo.ReturnVariables }
func (f *fakeReturnVariablesWorkflow) GetWorkflow() *workflowEntity.Workflow {
	return &workflowEntity.Workflow{}
}

var _ workflowDomain.ToolFromWorkflow = (*fakeReturnVariablesWorkflow)(nil)

// TestStrategyE2E_ReturnVariables_Loops: a 返回变量 workflow executed via run must NOT
// terminate (no marker) — the loop continues so multi-intent chains keep working.
// This is the chaining-preservation guarantee.
func TestStrategyE2E_ReturnVariables_Loops(t *testing.T) {
	svc := &fakeStrategySvc{}
	fakeWF := &crossworkflowStub{tools: []workflowDomain.ToolFromWorkflow{
		&fakeReturnVariablesWorkflow{result: `{"balance":42}`},
	}}
	orig := crossworkflow.DefaultSVC()
	crossworkflow.SetDefaultSVC(fakeWF)
	t.Cleanup(func() { crossworkflow.SetDefaultSVC(orig) })

	strategyTools, err := newStrategyTools(context.Background(), &strategyConfig{
		strategyIDs:     []int64{1},
		svc:             svc,
		forceToolReturn: false,
	})
	if err != nil {
		t.Fatalf("newStrategyTools err=%v", err)
	}
	calls := int32(0)
	agentTools := make([]tool.BaseTool, 0, len(strategyTools))
	for _, st := range strategyTools {
		agentTools = append(agentTools, st)
	}
	agent, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: &strategyE2EModel{calls: &calls},
		ToolsConfig:      compose.ToolsNodeConfig{Tools: agentTools, ExecuteSequentially: true},
		ToolReturnDirectly: map[string]struct{}{},
		ModelNodeName:    keyOfReActAgentChatModel,
		ToolsNodeName:    keyOfReActAgentToolsNode,
		MaxStep:          20,
	})
	if err != nil {
		t.Fatalf("react.NewAgent err=%v", err)
	}
	agentGraph, agentNodeOpts := agent.ExportGraph()
	g := compose.NewGraph[*AgentRequest, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) *AgentState { return &AgentState{} }))
	_ = g.AddLambdaNode("seed",
		compose.InvokableLambda[*AgentRequest, []*schema.Message](func(_ context.Context, _ *AgentRequest) ([]*schema.Message, error) {
			return []*schema.Message{schema.UserMessage("查余额")}, nil
		}))
	agentNodeOpts = append(agentNodeOpts, compose.WithNodeName(keyOfReActAgent))
	_ = g.AddGraphNode(keyOfReActAgent, agentGraph, agentNodeOpts...)
	_ = g.AddEdge(compose.START, "seed")
	_ = g.AddEdge("seed", keyOfReActAgent)
	_ = g.AddEdge(keyOfReActAgent, compose.END)
	runner, err := g.Compile(context.Background(), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		t.Fatalf("compile err=%v", err)
	}

	out, err := runner.Invoke(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Invoke err=%v", err)
	}
	got := atomic.LoadInt32(&calls)
	t.Logf("[e2e-returnvariables] model calls=%d, final=%q", got, out.Content)
	if got != 2 {
		t.Fatalf("[e2e-returnvariables] EXPECTED 2 model calls (loop/chain) but got %d; final=%q", got, out.Content)
	}
	if strings.Contains(out.Content, StrategyReturnDirectlyMarker) {
		t.Fatalf("[e2e-returnvariables] 返回变量 result must NOT carry marker, got %q", out.Content)
	}
}
