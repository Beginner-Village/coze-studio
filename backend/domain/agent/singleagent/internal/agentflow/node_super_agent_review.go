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

// node_super_agent_review.go 实现 Hermes 式「闭环学习」的复盘 fork:每个 super-agent
// run 结束后,一个**受限**子代理(白名单仅 memory + skill_manage)重放本轮轨迹,把用户
// 事实沉淀进 per-user 记忆、把可复用做法写成/patch 类级技能。它是「grows with you」的引擎。
// 安全边界:复盘 agent 绝不挂 run_bash/web 等工具——只能动 memory 和 skill。

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

// superAgentReviewToolNames is the strict whitelist of tools the post-run review
// fork may use: file tools to maintain the USER.md/MEMORY.md memory files, plus
// skill_manage. It must NEVER include execution/network tools (run_bash, web_*).
var superAgentReviewToolNames = map[string]bool{
	"read_file":    true,
	"write_file":   true,
	"edit_file":    true,
	"skill_manage": true,
}

// superAgentReviewMaxStep caps the review fork's tool-call rounds. Reviewing a
// finished trajectory needs only a handful of memory/skill writes.
const superAgentReviewMaxStep = 16

// buildSuperAgentReviewToolset builds the review fork's tools — the sandbox file
// tools (to maintain USER.md/MEMORY.md) plus skill_manage — keeping ONLY the
// whitelist. It stays confined even if new tools are registered later.
func buildSuperAgentReviewToolset(ctx context.Context, deps superAgentToolDeps) []tool.BaseTool {
	candidates := make([]tool.InvokableTool, 0, 12)
	candidates = append(candidates, newSandboxTools(deps.SandboxKey, false)...)
	candidates = append(candidates, newSuperAgentExtensionTools(deps)...)
	out := make([]tool.BaseTool, 0, len(superAgentReviewToolNames))
	for _, t := range candidates {
		info, err := t.Info(ctx)
		if err != nil || info == nil {
			continue
		}
		if superAgentReviewToolNames[info.Name] {
			out = append(out, t)
		}
	}
	return out
}

// buildSuperAgentReviewAgent builds the restricted ReAct agent that performs the
// post-run review. It reuses the main run's chat model, exposes only the
// whitelisted memory/skill tools, and pins SuperAgentReviewPrompt as the system
// prompt so the caller only needs to feed the finished conversation transcript.
func buildSuperAgentReviewAgent(ctx context.Context, chatModel chatmodel.ToolCallingChatModel, deps superAgentToolDeps) (*react.Agent, error) {
	tools := buildSuperAgentReviewToolset(ctx, deps)
	cfg := &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               tools,
			ExecuteSequentially: true,
		},
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(SuperAgentReviewPrompt)}, input...)
		},
		MaxStep: superAgentReviewMaxStep,
	}
	return react.NewAgent(ctx, cfg)
}

// RunPostRunReview runs the closed-learning-loop review fork for a just-finished
// super-agent run: it replays the transcript through the restricted review agent,
// which distills per-user memory and authors/patches class-level skills.
//
// It is a NO-OP for non-super agents (guarding the original single-agent flow) and
// for empty transcripts. It returns the reviewer's one-line summary. Callers should
// invoke it asynchronously — it makes its own LLM call and must never block or alter
// the user-facing run.
func RunPostRunReview(ctx context.Context, conf *Config, transcript []*schema.Message) (string, error) {
	if conf == nil || !isSuperAgent(conf) {
		return "", nil
	}
	if len(transcript) == 0 || conf.Agent == nil || conf.Agent.ModelInfo == nil {
		return "", nil
	}

	modelInfo, err := loadModelInfo(ctx, conf.ModelMgr, ptr.From(conf.Agent.ModelInfo.ModelId), conf.Agent.SpaceID)
	if err != nil {
		return "", err
	}
	chatModel, err := newChatModel(ctx, &config{
		modelFactory:      conf.ModelFactory,
		modelInfo:         modelInfo,
		agentModelSetting: conf.Agent.ModelInfo,
	})
	if err != nil {
		return "", err
	}

	deps := superAgentToolDeps{
		SandboxKey: sandboxKeyFor(conf.Identity.ConnectorID, conf.Agent.AgentID, conf.UserID),
		UserID:     parseSuperAgentUserID(conf.UserID),
		SpaceID:    conf.Agent.SpaceID,
		AgentID:    conf.Agent.AgentID,
	}
	reviewer, err := buildSuperAgentReviewAgent(ctx, chatModel, deps)
	if err != nil {
		return "", err
	}
	out, err := reviewer.Generate(ctx, transcript)
	if err != nil {
		return "", err
	}
	if out == nil {
		return "", nil
	}
	return out.Content, nil
}
