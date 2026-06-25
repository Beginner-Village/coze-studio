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
	"strings"
	"testing"
)

func TestSuperAgentExtensionsRegistered(t *testing.T) {
	tools := newSuperAgentExtensionTools(superAgentToolDeps{SandboxKey: "k", UserID: 1})
	names := map[string]bool{}
	for _, tl := range tools {
		info, err := tl.Info(context.Background())
		if err != nil {
			t.Fatalf("Info err=%v", err)
		}
		names[info.Name] = true
	}
	// skill_manage is a super-agent extension; the (file-based) long-term memory is
	// NOT a tool — it lives in USER.md/MEMORY.md maintained via the sandbox file tools.
	if !names["skill_manage"] {
		t.Fatalf("skill_manage not registered as super-agent extension; got %v", names)
	}
}

func TestSuperAgentExtraPromptMentionsStandardSkillFiles(t *testing.T) {
	for _, want := range []string{"action=list", "action=write_file", "action=edit", "action=remove_file", "action=delete", "action=diff", "scripts/", "references/", "templates/", "assets/"} {
		if !strings.Contains(SuperAgentExtraPrompt, want) {
			t.Fatalf("SuperAgentExtraPrompt should mention %q; got %q", want, SuperAgentExtraPrompt)
		}
	}
}

func TestSuperAgentReviewPromptMaintainsMemoryFilesAndSkills(t *testing.T) {
	// The post-run review fork (closed learning loop) must maintain the markdown
	// memory files and capture reusable CLASS-LEVEL skills, confined to file/skill tools.
	for _, want := range []string{"USER.md", "MEMORY.md", "skill_manage", "CLASS-LEVEL", "Nothing to save"} {
		if !strings.Contains(SuperAgentReviewPrompt, want) {
			t.Fatalf("SuperAgentReviewPrompt should mention %q; got %q", want, SuperAgentReviewPrompt)
		}
	}
	lower := strings.ToLower(SuperAgentReviewPrompt)
	if !strings.Contains(lower, "do not run commands") && !strings.Contains(lower, "not run commands") {
		t.Fatalf("SuperAgentReviewPrompt must forbid running commands; got %q", SuperAgentReviewPrompt)
	}
	for _, guard := range []string{"environment-dependent", "transient"} {
		if !strings.Contains(lower, guard) {
			t.Fatalf("SuperAgentReviewPrompt should warn against capturing %q failures; got %q", guard, SuperAgentReviewPrompt)
		}
	}
}

func TestSuperAgentExtraPromptMandatesPlanFirst(t *testing.T) {
	for _, want := range []string{"update_plan", "in_progress", "completed"} {
		if !strings.Contains(SuperAgentExtraPrompt, want) {
			t.Fatalf("SuperAgentExtraPrompt should drive plan-first execution mentioning %q; got %q", want, SuperAgentExtraPrompt)
		}
	}
	if !strings.Contains(strings.ToLower(SuperAgentExtraPrompt), "plan") {
		t.Fatalf("SuperAgentExtraPrompt should instruct the agent to plan multi-step work; got %q", SuperAgentExtraPrompt)
	}
}

func TestSuperAgentExtraPromptUsesMarkdownMemoryFiles(t *testing.T) {
	// Memory is Hermes-style markdown files (USER.md / MEMORY.md), auto-loaded and
	// maintained via the agent's file tools — not a database or KV tool.
	for _, want := range []string{"USER.md", "MEMORY.md", "read_file", "write_file", "edit_file"} {
		if !strings.Contains(SuperAgentExtraPrompt, want) {
			t.Fatalf("SuperAgentExtraPrompt should describe markdown memory files mentioning %q; got %q", want, SuperAgentExtraPrompt)
		}
	}
	lower := strings.ToLower(SuperAgentExtraPrompt)
	if !strings.Contains(lower, "auto-loaded") && !strings.Contains(lower, "auto-load") {
		t.Fatalf("SuperAgentExtraPrompt should say the memory files are auto-loaded; got %q", SuperAgentExtraPrompt)
	}
}
