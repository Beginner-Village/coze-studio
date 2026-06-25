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
	// 沙箱里确实写了结构化工具输出，harness 才能展示工具、返回和摘要。
	var storedPath string
	var storedContent []byte
	for p, c := range fm.files {
		if strings.HasPrefix(p, "/workspace/.agent/tooloutputs/") && strings.HasSuffix(p, ".json") {
			storedPath = p
			storedContent = c
		}
	}
	if storedPath == "" {
		t.Fatal("structured tool output was not written to sandbox file")
	}
	if !strings.Contains(got, storedPath) {
		t.Fatalf("model reference should point to structured output path %q, got %q", storedPath, got)
	}

	var payload struct {
		Tool    string `json:"tool"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
		Result  struct {
			Content   string `json:"content"`
			Bytes     int    `json:"bytes"`
			Truncated bool   `json:"truncated"`
		} `json:"result"`
	}
	if err := json.Unmarshal(storedContent, &payload); err != nil {
		t.Fatalf("structured tool output should be valid JSON: %v", err)
	}
	if payload.Tool != "tool_output" {
		t.Fatalf("tool = %q, want tool_output", payload.Tool)
	}
	if payload.Status != "completed" {
		t.Fatalf("status = %q, want completed", payload.Status)
	}
	if payload.Result.Content != big {
		t.Fatalf("stored content length = %d, want %d", len(payload.Result.Content), len(big))
	}
	if payload.Result.Bytes != len(big) || !payload.Result.Truncated {
		t.Fatalf("result metadata = %+v, want bytes=%d truncated=true", payload.Result, len(big))
	}
	if !strings.Contains(payload.Summary, "500 bytes") {
		t.Fatalf("summary should mention full size, got %q", payload.Summary)
	}
}

func TestOffloadToolResultOrTruncateWritesToolArguments(t *testing.T) {
	t.Setenv("AGENT_TOOL_OUTPUT_MAX_BYTES", "100")
	ctx := context.Background()
	fm := &fakeSandboxMgr{files: map[string][]byte{}}

	big := strings.Repeat("B", 500)
	got := offloadToolResultOrTruncate(ctx, fm, "k", toolOutputOffloadMeta{
		Tool:      "run_bash",
		Arguments: json.RawMessage(`{"command":"python3 - <<'PY'\nprint('x')\nPY","timeout_sec":3}`),
	}, big)
	if !strings.Contains(got, "/workspace/.agent/tooloutputs/") || !strings.Contains(got, ".json") {
		t.Fatalf("offload result missing JSON reference: %q", got)
	}

	var storedContent []byte
	for p, c := range fm.files {
		if strings.HasPrefix(p, "/workspace/.agent/tooloutputs/") && strings.HasSuffix(p, ".json") {
			storedContent = c
		}
	}
	if storedContent == nil {
		t.Fatal("structured tool output was not written")
	}

	var payload struct {
		Tool      string         `json:"tool"`
		Status    string         `json:"status"`
		Arguments map[string]any `json:"arguments"`
		Result    struct {
			Content string `json:"content"`
			Bytes   int    `json:"bytes"`
		} `json:"result"`
	}
	if err := json.Unmarshal(storedContent, &payload); err != nil {
		t.Fatalf("structured tool output should be valid JSON: %v", err)
	}
	if payload.Tool != "run_bash" {
		t.Fatalf("tool = %q, want run_bash", payload.Tool)
	}
	if payload.Status != "completed" {
		t.Fatalf("status = %q, want completed", payload.Status)
	}
	if payload.Arguments["timeout_sec"] != float64(3) {
		t.Fatalf("arguments = %#v, want timeout_sec=3", payload.Arguments)
	}
	if payload.Result.Content != big || payload.Result.Bytes != len(big) {
		t.Fatalf("result = %+v, want full content and bytes=%d", payload.Result, len(big))
	}
}

func TestOffloadToolResultOrTruncateScopesOutputToConversation(t *testing.T) {
	t.Setenv("AGENT_TOOL_OUTPUT_MAX_BYTES", "100")
	ctx := withToolOutputConversationID(context.Background(), 456)
	fm := &fakeSandboxMgr{files: map[string][]byte{}}

	big := strings.Repeat("C", 500)
	got := offloadToolResultOrTruncate(ctx, fm, "k", toolOutputOffloadMeta{
		Tool: "grep",
	}, big)

	wantPrefix := "/workspace/.agent/tooloutputs/sessions/456/"
	if !strings.Contains(got, wantPrefix) {
		t.Fatalf("model reference should use session-scoped tool output prefix %q, got %q", wantPrefix, got)
	}

	var storedPath string
	var storedContent []byte
	for p, c := range fm.files {
		if strings.HasPrefix(p, wantPrefix) && strings.HasSuffix(p, ".json") {
			storedPath = p
			storedContent = c
		}
	}
	if storedPath == "" {
		t.Fatalf("structured tool output was not written under %s; files=%v", wantPrefix, fm.files)
	}

	var payload struct {
		ConversationID string `json:"conversation_id"`
		Tool           string `json:"tool"`
	}
	if err := json.Unmarshal(storedContent, &payload); err != nil {
		t.Fatalf("structured tool output should be valid JSON: %v", err)
	}
	if payload.ConversationID != "456" {
		t.Fatalf("conversation_id = %q, want 456", payload.ConversationID)
	}
	if payload.Tool != "grep" {
		t.Fatalf("tool = %q, want grep", payload.Tool)
	}
}
