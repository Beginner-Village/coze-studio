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

// deepagents_engine.go scaffolds an experimental, feature-flagged DeepAgents
// engine for the single-agent flow. It is GATED OFF by default: unless the
// AGENT_ENGINE environment variable is set to "deepagents", nothing in this
// file affects the running agent — the existing ReAct path is used verbatim.
//
// Why a separate engine (and not a replacement for ReAct):
// eino v0.9.6 ships github.com/cloudwego/eino/adk/prebuilt/deep, a "DeepAgents"
// implementation (task sub-agents + todos + filesystem) that is a better long
// term home for autonomous planning. The review conclusion was to introduce it
// as an independent gray-release engine while keeping ReAct as the fallback,
// rather than rewriting the current flow.
//
// IMPORTANT INTEGRATION NOTE (the blocker):
// The current builder (agent_flow_builder.go) inlines the ReAct agent as a
// compose.AnyGraph node via react.Agent.ExportGraph() and wires it into a
// larger compose.Graph (persona render, knowledge retriever, prompt template,
// suggest graph, checkpoint store, streaming + interrupt handling in
// callback_reply_chunk.go / agent_flow_runner.go).
//
// The adk deep agent does NOT expose ExportGraph() / a compose.AnyGraph. It is
// an adk.ResumableAgent that is meant to be driven by an adk.Runner
// (adk.NewRunner), with its own event/stream/interrupt model. It therefore
// cannot be dropped into the existing compose.Graph as a node without an
// adk<->compose bridge, and wiring its streaming/interrupt semantics into the
// existing callback_reply_chunk.go / agent_flow_runner.go path would require
// changing that path — which is explicitly out of scope for this scaffolding
// step.
//
// So this file only provides:
//   1. deepAgentsEnabled() — the feature switch.
//   2. buildDeepAgent(...) — a minimal, compile-safe constructor that builds a
//      deep.ResumableAgent from the already-aggregated agentTools. It returns
//      the agent instance (NOT a compose graph), because deep has no
//      ExportGraph equivalent. The caller (builder) does not yet consume it.
//
// TODO(deepagents): to actually run this engine, add an adk.Runner-based
// execution path (parallel to AgentRunner) that:
//   - drives the deep agent via adk.NewRunner / Runner.Run|Resume,
//   - adapts adk AgentEvent streams to the existing reply-chunk callback
//     contract (callback_reply_chunk.go) WITHOUT modifying that file,
//   - maps adk interrupt/resume onto the existing checkpoint/interrupt flow.
// Until then, buildDeepAgent is constructed-but-unused and the builder falls
// back to ReAct (see agent_flow_builder.go), guaranteeing zero behavior change.

import (
	"context"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

// agentEngineEnvKey is the environment variable that selects the agent engine.
// Unset (or any value other than "deepagents") => ReAct (default, unchanged).
const agentEngineEnvKey = "AGENT_ENGINE"

// deepAgentsValue is the AGENT_ENGINE value that opts into the experimental
// DeepAgents engine.
const deepAgentsValue = "deepagents"

// deepAgentsEnabled reports whether the experimental DeepAgents engine is
// selected via the AGENT_ENGINE environment variable. It returns false (ReAct)
// for any value other than exactly "deepagents" (case-insensitive).
func deepAgentsEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv(agentEngineEnvKey)), deepAgentsValue)
}

// buildDeepAgent constructs an experimental DeepAgents agent from the
// already-aggregated agentTools, reusing the same chat model as the ReAct path.
//
// It returns an adk.ResumableAgent (NOT a compose.AnyGraph) because the eino
// adk deep package does not expose an ExportGraph()-style API; deep agents run
// through an adk.Runner instead. The signature is intentionally kept close to
// the inputs the ReAct sub-graph is built from (ctx, conf, chatModel,
// agentTools) so a future builder branch can choose between the two.
//
// NOTE: the returned agent is not yet wired into the running flow. See the
// package-level TODO(deepagents) for what remains before it can execute without
// touching the existing streaming/interrupt code.
func buildDeepAgent(
	ctx context.Context,
	conf *Config,
	chatModel chatmodel.ToolCallingChatModel,
	agentTools []tool.BaseTool,
) (adk.ResumableAgent, error) {
	name := "single_agent_deep"

	cfg := &deep.Config{
		Name:        name,
		Description: "Experimental DeepAgents engine for single-agent flow (AGENT_ENGINE=deepagents).",
		// ToolCallingChatModel embeds model.BaseChatModel == model.BaseModel[*schema.Message],
		// which satisfies deep.Config.ChatModel.
		ChatModel: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
			},
		},
	}

	return deep.New(ctx, cfg)
}
