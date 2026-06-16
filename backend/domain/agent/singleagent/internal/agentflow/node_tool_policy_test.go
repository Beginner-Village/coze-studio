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

func TestIsMutatingCommand(t *testing.T) {
	mut := []string{"rm -rf x", "mv a b", "mkdir d", "echo hi > f", "echo hi >> f", "sed -i s/a/b/ f", "touch z", "cp a b", "tee f"}
	for _, c := range mut {
		if !isMutatingCommand(c) {
			t.Errorf("want mutating: %q", c)
		}
	}
	safe := []string{"ls -la", "cat f", "grep foo .", "python script.py", "echo hi", "find . -name '*.go'", "go test ./..."}
	for _, c := range safe {
		if isMutatingCommand(c) {
			t.Errorf("want non-mutating: %q", c)
		}
	}
}

func TestCheckMutationAllowed_Modes(t *testing.T) {
	// 默认（未设 env）= workspace-write，允许变更。
	if ok, _ := checkMutationAllowed("write_file"); !ok {
		t.Fatal("default mode should allow mutation")
	}

	t.Setenv("SANDBOX_MODE", "read-only")
	ok, reason := checkMutationAllowed("write_file")
	if ok {
		t.Fatal("read-only mode should block mutation")
	}
	if !strings.Contains(reason, "read-only") {
		t.Fatalf("reason should mention read-only, got %q", reason)
	}

	t.Setenv("SANDBOX_MODE", "full")
	if ok, _ := checkMutationAllowed("write_file"); !ok {
		t.Fatal("full mode should allow mutation")
	}
}

func TestOffloadOrTruncate(t *testing.T) {
	t.Setenv("AGENT_TOOL_OUTPUT_MAX_BYTES", "100")
	ctx := context.Background()
	fm := &fakeSandboxMgr{files: map[string][]byte{}}

	// 短输出原样返回。
	if got := offloadOrTruncate(ctx, fm, "k", "short"); got != "short" {
		t.Fatalf("short output changed: %q", got)
	}

	// 超长输出 → 落盘 + 回灌引用。
	big := strings.Repeat("A", 500)
	got := offloadOrTruncate(ctx, fm, "k", big)
	if !strings.Contains(got, "/workspace/.agent/tooloutputs/") || !strings.Contains(got, "truncated") {
		t.Fatalf("offload result missing reference: %q", got)
	}
	if len(got) >= len(big) {
		t.Fatalf("offloaded output not shorter: %d", len(got))
	}
	// 沙箱里确实写了完整内容。
	var stored bool
	for p, c := range fm.files {
		if strings.HasPrefix(p, "/workspace/.agent/tooloutputs/") && len(c) == 500 {
			stored = true
		}
	}
	if !stored {
		t.Fatal("full output was not written to sandbox file")
	}
}
