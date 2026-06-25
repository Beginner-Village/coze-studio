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
	"testing"

	"github.com/cloudwego/eino/schema"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
)

// The review fork is the engine of the closed learning loop, so its tool surface
// is safety-critical: it must be able to maintain the memory files and skills, and
// must NEVER hold execution/network tools (run_bash, web_*).
func TestSuperAgentReviewToolsetIsWhitelistedToMemoryAndSkill(t *testing.T) {
	crosssandbox.SetDefaultSVC(&fakeSandboxMgr{})
	defer crosssandbox.SetDefaultSVC(nil)
	ctx := context.Background()
	tools := buildSuperAgentReviewToolset(ctx, superAgentToolDeps{SandboxKey: "k", UserID: 42, SpaceID: 1, AgentID: 7})

	names := map[string]bool{}
	for _, tl := range tools {
		info, err := tl.Info(ctx)
		if err != nil {
			t.Fatalf("Info err=%v", err)
		}
		names[info.Name] = true
	}

	for _, want := range []string{"read_file", "write_file", "edit_file", "skill_manage"} {
		if !names[want] {
			t.Fatalf("review toolset must include %q; got %v", want, names)
		}
	}
	// File tools to maintain USER.md/MEMORY.md + skill_manage; never execution/network.
	for _, forbidden := range []string{"web_search", "web_fetch", "run_bash", "deep_task", "list_files", "grep"} {
		if names[forbidden] {
			t.Fatalf("review toolset must NOT include %q; got %v", forbidden, names)
		}
	}
	if len(names) != 4 {
		t.Fatalf("review toolset should hold exactly the 4 whitelisted tools; got %v", names)
	}
}

// The closed-learning-loop review MUST NOT run for ordinary single-agents — it is
// strictly a super-agent capability. This proves the guard short-circuits BEFORE any
// model/tool work (a non-super run would otherwise hit the nil ModelMgr and fail),
// so the original single-agent flow is never touched.
func TestRunPostRunReviewNoOpsForNonSuperAgent(t *testing.T) {
	ctx := context.Background()
	conf := &Config{
		Agent:  &entity.SingleAgent{SingleAgent: &crossagent.SingleAgent{AgentID: 1, AgentType: "normal"}},
		UserID: "42",
	}
	transcript := []*schema.Message{
		{Role: schema.User, Content: "hi"},
		{Role: schema.Assistant, Content: "hello"},
	}
	out, err := RunPostRunReview(ctx, conf, transcript)
	if err != nil {
		t.Fatalf("non-super review must be a no-op, not an error: %v", err)
	}
	if out != "" {
		t.Fatalf("non-super review must return empty (no-op); got %q", out)
	}

	// Empty agent_type ("" = normal) must also be a no-op.
	conf.Agent.AgentType = ""
	out, err = RunPostRunReview(ctx, conf, transcript)
	if err != nil || out != "" {
		t.Fatalf("empty-agent-type review must be a no-op; got out=%q err=%v", out, err)
	}
}
