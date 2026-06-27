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

// This file is a regression guard for the strategy "run" tool's per-invocation
// returnDirectly. It proves, at the eino-graph level, that react.SetReturnDirectly
// called from inside a tool DOES terminate the react loop even though the react agent
// is ExportGraph()'d and recomposed into the outer agent graph (agent_flow_builder.go).
// An earlier commit claimed this "does NOT propagate across the embedded subgraph";
// these tests demonstrate that claim was incorrect (state propagates via compose's
// parent-chain state lookup). It covers Invoke, Stream, and the checkpoint-enabled
// compile options used in production.

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// reproModel emits a single tool call ("run") on the first Generate/Stream call,
// then a plain assistant answer on every subsequent call. We count model calls to
// detect whether the react loop TERMINATED after the tool (1 call) or LOOPED (2+).
type reproModel struct {
	calls *int32
}

func (m *reproModel) nextMessage() *schema.Message {
	n := atomic.AddInt32(m.calls, 1)
	if n == 1 {
		return schema.AssistantMessage("", []schema.ToolCall{
			{
				ID: "call_repro_1",
				Function: schema.FunctionCall{
					Name:      "run",
					Arguments: `{"scene":1,"cap":1}`,
				},
			},
		})
	}
	return schema.AssistantMessage("MODEL_LOOPED_ANSWER", nil)
}

func (m *reproModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return m.nextMessage(), nil
}

func (m *reproModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Send(m.nextMessage(), nil)
	sw.Close()
	return sr, nil
}

func (m *reproModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

// reproRunTool is a stand-in for the strategy "run" tool: it calls
// react.SetReturnDirectly(ctx) and returns a marker-prefixed answer.
type reproRunTool struct {
	setReturnDirectly bool
	rdErr             *error
}

func (t *reproRunTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "run",
		Desc:        "repro run tool",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *reproRunTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	if t.setReturnDirectly {
		err := react.SetReturnDirectly(ctx)
		if t.rdErr != nil {
			*t.rdErr = err
		}
	}
	return "WORKFLOW_DIRECT_TEXT", nil
}

var reproCallsByGraph *int32

// buildReproOuterGraph mirrors agent_flow_builder.go: build a react agent, ExportGraph(),
// then recompose it into an OUTER graph that has its own WithGenLocalState(*AgentState),
// exactly like BuildAgent does. Returns the compiled outer runnable.
func buildReproOuterGraph(t *testing.T, setReturnDirectly bool, staticReturnDirectly bool, rdErr *error) compose.Runnable[*AgentRequest, *schema.Message] {
	t.Helper()
	ctx := context.Background()

	runTool := &reproRunTool{setReturnDirectly: setReturnDirectly, rdErr: rdErr}

	returnDirectly := map[string]struct{}{}
	if staticReturnDirectly {
		returnDirectly["run"] = struct{}{}
	}

	calls := int32(0)
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &reproModel{calls: &calls},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               []tool.BaseTool{runTool},
			ExecuteSequentially: true,
		},
		ToolReturnDirectly: returnDirectly,
		ModelNodeName:      keyOfReActAgentChatModel,
		ToolsNodeName:      keyOfReActAgentToolsNode,
		MaxStep:            20,
	})
	if err != nil {
		t.Fatalf("react.NewAgent err=%v", err)
	}
	agentGraph, agentNodeOpts := agent.ExportGraph()

	reproCallsByGraph = &calls

	// Outer graph: input *AgentRequest, output *schema.Message. Provide its own state.
	g := compose.NewGraph[*AgentRequest, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) *AgentState { return &AgentState{} }))

	// Lambda that turns the *AgentRequest into the []*schema.Message the react agent expects.
	_ = g.AddLambdaNode("seed",
		compose.InvokableLambda[*AgentRequest, []*schema.Message](func(_ context.Context, _ *AgentRequest) ([]*schema.Message, error) {
			return []*schema.Message{schema.UserMessage("hi")}, nil
		}))

	agentNodeOpts = append(agentNodeOpts, compose.WithNodeName(keyOfReActAgent))
	_ = g.AddGraphNode(keyOfReActAgent, agentGraph, agentNodeOpts...)

	_ = g.AddEdge(compose.START, "seed")
	_ = g.AddEdge("seed", keyOfReActAgent)
	_ = g.AddEdge(keyOfReActAgent, compose.END)

	runner, err := g.Compile(ctx, compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		t.Fatalf("outer graph compile err=%v", err)
	}
	return runner
}

// TestRepro_NativeStaticReturnDirectly_Terminates is the CONTROL: a tool in the
// static ToolReturnDirectly map must terminate the recomposed loop (model called once).
func TestRepro_NativeStaticReturnDirectly_Terminates(t *testing.T) {
	runner := buildReproOuterGraph(t, false /*setReturnDirectly*/, true /*static*/, nil)
	out, err := runner.Invoke(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Invoke err=%v", err)
	}
	calls := atomic.LoadInt32(reproCallsByGraph)
	t.Logf("[native-static] model calls=%d, final content=%q", calls, out.Content)
	if calls != 1 {
		t.Fatalf("[native-static] expected model called ONCE (terminated), got %d calls; final=%q", calls, out.Content)
	}
	if out.Content != "WORKFLOW_DIRECT_TEXT" {
		t.Fatalf("[native-static] expected direct tool text, got %q", out.Content)
	}
}

// TestRepro_ToolInternalSetReturnDirectly_Terminates is the EXPERIMENT: a tool NOT
// in the static map that calls react.SetReturnDirectly(ctx). If the ExportGraph
// recompose breaks state propagation, the loop will NOT terminate (model called 2x).
func TestRepro_ToolInternalSetReturnDirectly_Terminates(t *testing.T) {
	var rdErr error
	runner := buildReproOuterGraph(t, true /*setReturnDirectly*/, false /*static*/, &rdErr)
	out, err := runner.Invoke(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Invoke err=%v", err)
	}
	calls := atomic.LoadInt32(reproCallsByGraph)
	t.Logf("[tool-internal] SetReturnDirectly err=%v, model calls=%d, final content=%q", rdErr, calls, out.Content)
	if calls != 1 {
		t.Fatalf("[tool-internal] EXPECTED terminate (1 model call) but got %d; SetReturnDirectly err=%v, final=%q", calls, rdErr, out.Content)
	}
}

// Stream variants mirror the above via Stream, since production runs Stream.
func TestRepro_NativeStaticReturnDirectly_Terminates_Stream(t *testing.T) {
	runner := buildReproOuterGraph(t, false, true, nil)
	sr, err := runner.Stream(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Stream err=%v", err)
	}
	drainStream(sr)
	calls := atomic.LoadInt32(reproCallsByGraph)
	t.Logf("[native-static-stream] model calls=%d", calls)
	if calls != 1 {
		t.Fatalf("[native-static-stream] expected 1 model call, got %d", calls)
	}
}

func TestRepro_ToolInternalSetReturnDirectly_Terminates_Stream(t *testing.T) {
	var rdErr error
	runner := buildReproOuterGraph(t, true, false, &rdErr)
	sr, err := runner.Stream(context.Background(), &AgentRequest{})
	if err != nil {
		t.Fatalf("Stream err=%v", err)
	}
	drainStream(sr)
	calls := atomic.LoadInt32(reproCallsByGraph)
	t.Logf("[tool-internal-stream] SetReturnDirectly err=%v, model calls=%d", rdErr, calls)
	if calls != 1 {
		t.Fatalf("[tool-internal-stream] EXPECTED terminate (1 model call) but got %d; err=%v", calls, rdErr)
	}
}

func drainStream(sr *schema.StreamReader[*schema.Message]) {
	defer sr.Close()
	for {
		_, err := sr.Recv()
		if err != nil {
			return
		}
	}
}

// memCPStore is an in-memory CheckPointStore mirroring production (which enables
// checkpointing when tools are present). This is the most faithful repro of the
// production graph compile options.
type memCPStore struct {
	m map[string][]byte
}

func (s *memCPStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	if s.m == nil {
		return nil, false, nil
	}
	b, ok := s.m[id]
	return b, ok, nil
}

func (s *memCPStore) Set(_ context.Context, id string, b []byte) error {
	if s.m == nil {
		s.m = map[string][]byte{}
	}
	s.m[id] = b
	return nil
}

// buildReproOuterGraphWithCheckpoint mirrors agent_flow_builder.go MORE faithfully:
// it adds WithCheckPointStore at compile time and runs with WithCheckPointID, exactly
// like production when tools are present (requireCheckpoint=true). The "run" tool
// returns a marker-prefixed result to match production's StrategyReturnDirectlyMarker.
func buildReproOuterGraphWithCheckpoint(t *testing.T, markerReturn bool) (compose.Runnable[*AgentRequest, *schema.Message], *int32) {
	t.Helper()
	ctx := context.Background()

	runTool := &reproRunToolMarker{marker: markerReturn}

	calls := int32(0)
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &reproModel{calls: &calls},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               []tool.BaseTool{runTool},
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
			return []*schema.Message{schema.UserMessage("hi")}, nil
		}))

	agentNodeOpts = append(agentNodeOpts, compose.WithNodeName(keyOfReActAgent))
	_ = g.AddGraphNode(keyOfReActAgent, agentGraph, agentNodeOpts...)

	_ = g.AddEdge(compose.START, "seed")
	_ = g.AddEdge("seed", keyOfReActAgent)
	_ = g.AddEdge(keyOfReActAgent, compose.END)

	_ = compose.RegisterSerializableType[*AgentState]("agent_state_repro")
	runner, err := g.Compile(ctx,
		compose.WithCheckPointStore(&memCPStore{}),
		compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		t.Fatalf("outer graph compile err=%v", err)
	}
	return runner, &calls
}

type reproRunToolMarker struct{ marker bool }

func (t *reproRunToolMarker) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "run",
		Desc:        "repro run tool marker",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *reproRunToolMarker) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	_ = react.SetReturnDirectly(ctx)
	if t.marker {
		return StrategyReturnDirectlyMarker + "WORKFLOW_DIRECT_TEXT", nil
	}
	return "WORKFLOW_DIRECT_TEXT", nil
}

// TestRepro_Checkpoint_ToolInternalSetReturnDirectly_Stream is the MOST FAITHFUL repro:
// checkpointing enabled + WithCheckPointID + Stream + marker-prefixed return, like prod.
func TestRepro_Checkpoint_ToolInternalSetReturnDirectly_Stream(t *testing.T) {
	runner, calls := buildReproOuterGraphWithCheckpoint(t, true /*markerReturn*/)
	sr, err := runner.Stream(context.Background(), &AgentRequest{},
		compose.WithCheckPointID("repro-cp-1"))
	if err != nil {
		t.Fatalf("Stream err=%v", err)
	}
	drainStream(sr)
	got := atomic.LoadInt32(calls)
	t.Logf("[checkpoint-stream-marker] model calls=%d", got)
	if got != 1 {
		t.Fatalf("[checkpoint-stream-marker] EXPECTED terminate (1 model call) but got %d", got)
	}
}

// TestRepro_Checkpoint_ToolInternalSetReturnDirectly_Invoke same but Invoke.
func TestRepro_Checkpoint_ToolInternalSetReturnDirectly_Invoke(t *testing.T) {
	runner, calls := buildReproOuterGraphWithCheckpoint(t, true)
	out, err := runner.Invoke(context.Background(), &AgentRequest{},
		compose.WithCheckPointID("repro-cp-2"))
	if err != nil {
		t.Fatalf("Invoke err=%v", err)
	}
	got := atomic.LoadInt32(calls)
	t.Logf("[checkpoint-invoke-marker] model calls=%d, final=%q", got, out.Content)
	if got != 1 {
		t.Fatalf("[checkpoint-invoke-marker] EXPECTED terminate (1 model call) but got %d; final=%q", got, out.Content)
	}
}
