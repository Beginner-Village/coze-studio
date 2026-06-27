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
	"strconv"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"

	crosssingleagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/modelmgr"
)

func TestContextCompactThresholdBytes(t *testing.T) {
	// 未配置模型/窗口 → 退回字节兜底(默认 160KB)。
	r0 := &AgentRunner{}
	if got := r0.contextCompactThresholdBytes(); got != defaultContextCompactMaxBytes {
		t.Fatalf("no-model threshold = %d, want byte fallback %d", got, defaultContextCompactMaxBytes)
	}

	// Capability 为 nil → 同样退回字节兜底(不 panic)。
	rNilCap := &AgentRunner{modelInfo: &modelmgr.Model{}}
	if got := rNilCap.contextCompactThresholdBytes(); got != defaultContextCompactMaxBytes {
		t.Fatalf("nil-capability threshold = %d, want byte fallback %d", got, defaultContextCompactMaxBytes)
	}

	// 配置了上下文窗口 → 阈值 = 窗口 × ratio × 每token字节数,且随窗口放大。
	mk := func(window int) *AgentRunner {
		return &AgentRunner{modelInfo: &modelmgr.Model{
			Meta: modelmgr.ModelMeta{Capability: &modelmgr.Capability{InputTokens: window}},
		}}
	}
	want := func(window int) int {
		return int(float64(window) * defaultContextCompactRatio * defaultContextCompactBytesPerToken)
	}
	for _, w := range []int{128000, 200000, 1000000} {
		if got := mk(w).contextCompactThresholdBytes(); got != want(w) {
			t.Fatalf("window=%d threshold = %d, want %d", w, got, want(w))
		}
	}
	// 大窗口模型阈值应显著大于小窗口(自适应)。
	if mk(1000000).contextCompactThresholdBytes() <= mk(128000).contextCompactThresholdBytes() {
		t.Fatal("larger context window should yield larger compaction threshold")
	}
}

func TestHistoryApproxBytesHandlesPartialMultiContent(t *testing.T) {
	history := []*schema.Message{
		{
			Role:    schema.User,
			Content: "base",
			MultiContent: []schema.ChatMessagePart{
				{Type: schema.ChatMessagePartTypeText, Text: "hello"},
				{Type: schema.ChatMessagePartTypeImageURL},
				{
					Type: schema.ChatMessagePartTypeImageURL,
					ImageURL: &schema.ChatMessageImageURL{
						URL: "http://example.com/image.png",
					},
				},
			},
		},
	}

	want := len("base") + len("hello") + len("http://example.com/image.png")
	if got := historyApproxBytes(history); got != want {
		t.Fatalf("historyApproxBytes = %d, want %d", got, want)
	}
}

func TestInferContextWindowByModelName(t *testing.T) {
	cases := map[string]int{
		"GLM-5.2":            1000000,
		"glm5-air":           1000000,
		"claude-opus-4-8":    200000,
		"DeepSeek-V4":        128000,
		"gpt-4o-mini":        128000,
		"qwen2.5-72b":        128000,
		"gemini-2.0-pro":     1000000,
		"some-unknown-model": 0,
		"":                   0,
	}
	for name, want := range cases {
		if got := inferContextWindowByModelName(name); got != want {
			t.Fatalf("inferContextWindowByModelName(%q) = %d, want %d", name, got, want)
		}
	}

	// 未配置 InputTokens 时,modelContextWindowTokens 应回落到按名推断。
	r := &AgentRunner{modelInfo: &modelmgr.Model{Name: "GLM-5.2"}}
	if got := r.modelContextWindowTokens(); got != 1000000 {
		t.Fatalf("window for GLM-5.2 (no InputTokens) = %d, want 1000000", got)
	}
	// 显式配置的 InputTokens 优先于按名推断。
	r2 := &AgentRunner{modelInfo: &modelmgr.Model{Name: "GLM-5.2", Meta: modelmgr.ModelMeta{Capability: &modelmgr.Capability{InputTokens: 256000}}}}
	if got := r2.modelContextWindowTokens(); got != 256000 {
		t.Fatalf("explicit InputTokens should win = %d, want 256000", got)
	}
}

func TestPreHandlerReqAutoCompactsSuperAgentHistory(t *testing.T) {
	t.Setenv("AGENT_CONTEXT_COMPACT_MAX_BYTES", "300")
	t.Setenv("AGENT_CONTEXT_COMPACT_RECENT_MESSAGES", "4")

	fm := &fakeSandboxMgr{files: map[string][]byte{}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	r := &AgentRunner{superAgent: true, sandboxKey: "super-key"}

	history := longContextCompactionHistory()
	req := &AgentRequest{
		UserID:  "u1",
		Input:   schema.UserMessage("继续完成剩余实现"),
		History: history,
		Identity: &crosssingleagent.AgentIdentity{
			AgentID:     123,
			ConnectorID: 10000010,
		},
	}

	got := r.PreHandlerReq(context.Background(), req)

	if len(got.History) >= len(history) {
		t.Fatalf("history should be compacted: before=%d after=%d", len(history), len(got.History))
	}
	if got.History[0].Role != schema.System {
		t.Fatalf("first compacted message should be system summary, got %s", got.History[0].Role)
	}
	if !strings.Contains(got.History[0].Content, "Context summary") {
		t.Fatalf("summary message should identify context compaction, got %q", got.History[0].Content)
	}
	if !strings.Contains(got.History[0].Content, "/workspace/.agent/context-summary.json") {
		t.Fatalf("summary message should reference persisted summary path, got %q", got.History[0].Content)
	}
	if !strings.Contains(got.History[len(got.History)-1].Content, "recent requirement 11") {
		t.Fatalf("recent tail should be preserved, got tail %q", got.History[len(got.History)-1].Content)
	}

	raw := fm.files["/workspace/.agent/context-summary.json"]
	if len(raw) == 0 {
		t.Fatal("context summary was not written to sandbox")
	}
	var summary struct {
		Version           string   `json:"version"`
		Summary           string   `json:"summary"`
		UpdatedAt         int64    `json:"updated_at"`
		Trigger           string   `json:"trigger"`
		OriginalMessages  int      `json:"original_messages"`
		CompactedMessages int      `json:"compacted_messages"`
		RetainedMessages  int      `json:"retained_messages"`
		OriginalBytes     int      `json:"original_bytes"`
		MaxBytes          int      `json:"max_bytes"`
		SummaryPath       string   `json:"summary_path"`
		KeyFiles          []string `json:"key_files"`
		Artifacts         []string `json:"artifacts"`
		NextActions       []string `json:"next_actions"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("summary should be valid JSON: %v\n%s", err, raw)
	}
	if summary.Version != "v1" {
		t.Fatalf("summary version = %q, want v1", summary.Version)
	}
	if !strings.Contains(summary.Summary, "legacy requirement 0") {
		t.Fatalf("summary should include compacted older context, got %q", summary.Summary)
	}
	if summary.UpdatedAt == 0 {
		t.Fatal("summary should include updated_at")
	}
	// 该测试的模型未配置上下文窗口,走字节兜底阈值,trigger 带 no_window_configured 标记。
	if !strings.HasPrefix(summary.Trigger, "history_bytes_exceeded") {
		t.Fatalf("summary trigger = %q, want history_bytes_exceeded*", summary.Trigger)
	}
	if summary.OriginalMessages != len(history) {
		t.Fatalf("summary original_messages = %d, want %d", summary.OriginalMessages, len(history))
	}
	if summary.CompactedMessages != len(history)-4 {
		t.Fatalf("summary compacted_messages = %d, want %d", summary.CompactedMessages, len(history)-4)
	}
	if summary.RetainedMessages != 4 {
		t.Fatalf("summary retained_messages = %d, want 4", summary.RetainedMessages)
	}
	if summary.OriginalBytes <= 300 {
		t.Fatalf("summary original_bytes = %d, want greater than threshold", summary.OriginalBytes)
	}
	if summary.MaxBytes != 300 {
		t.Fatalf("summary max_bytes = %d, want 300", summary.MaxBytes)
	}
	if summary.SummaryPath != "/workspace/.agent/context-summary.json" {
		t.Fatalf("summary summary_path = %q", summary.SummaryPath)
	}
	if !containsString(summary.KeyFiles, "/workspace/main.go") {
		t.Fatalf("summary should track key workspace files, got %#v", summary.KeyFiles)
	}
	if !containsString(summary.Artifacts, "/outputs/report.html") {
		t.Fatalf("summary should track artifact paths, got %#v", summary.Artifacts)
	}
	if !containsString(summary.NextActions, "继续完成剩余实现") {
		t.Fatalf("summary should carry next action from current input, got %#v", summary.NextActions)
	}
}

func TestPreHandlerReqAutoCompactsSuperAgentHistoryByConversation(t *testing.T) {
	t.Setenv("AGENT_CONTEXT_COMPACT_MAX_BYTES", "300")
	t.Setenv("AGENT_CONTEXT_COMPACT_RECENT_MESSAGES", "4")

	fm := &fakeSandboxMgr{files: map[string][]byte{}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	r := &AgentRunner{superAgent: true, sandboxKey: "super-key"}

	req := &AgentRequest{
		UserID:         "u1",
		ConversationID: 456,
		Input:          schema.UserMessage("继续这个会话的实现"),
		History:        longContextCompactionHistory(),
		Identity: &crosssingleagent.AgentIdentity{
			AgentID:     123,
			ConnectorID: 10000010,
		},
	}

	got := r.PreHandlerReq(context.Background(), req)

	wantPath := "/workspace/.agent/sessions/456/context-summary.json"
	if !strings.Contains(got.History[0].Content, wantPath) {
		t.Fatalf("summary message should reference session-scoped summary path %q, got %q", wantPath, got.History[0].Content)
	}
	if _, ok := fm.files["/workspace/.agent/context-summary.json"]; ok {
		t.Fatal("conversation-scoped compaction must not overwrite global context summary")
	}
	raw := fm.files[wantPath]
	if len(raw) == 0 {
		t.Fatalf("context summary was not written to %s", wantPath)
	}
	var summary struct {
		SummaryPath string `json:"summary_path"`
	}
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("summary should be valid JSON: %v\n%s", err, raw)
	}
	if summary.SummaryPath != wantPath {
		t.Fatalf("summary_path = %q, want %q", summary.SummaryPath, wantPath)
	}
}

func TestPreHandlerReqDoesNotCompactNormalAgentHistory(t *testing.T) {
	t.Setenv("AGENT_CONTEXT_COMPACT_MAX_BYTES", "300")

	fm := &fakeSandboxMgr{files: map[string][]byte{}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	r := &AgentRunner{}
	req := &AgentRequest{
		UserID:  "u1",
		Input:   schema.UserMessage("普通智能体继续"),
		History: longContextCompactionHistory(),
		Identity: &crosssingleagent.AgentIdentity{
			AgentID:     123,
			ConnectorID: 10000010,
		},
	}

	got := r.PreHandlerReq(context.Background(), req)

	if len(got.History) != len(req.History) {
		t.Fatalf("normal agent history should not be compacted: before=%d after=%d", len(req.History), len(got.History))
	}
	if _, ok := fm.files["/workspace/.agent/context-summary.json"]; ok {
		t.Fatal("normal agent should not write context summary")
	}
}

func longContextCompactionHistory() []*schema.Message {
	history := make([]*schema.Message, 0, 24)
	for i := 0; i < 12; i++ {
		userText := strings.Repeat("用户上下文 ", 20) + "legacy requirement " + strconv.Itoa(i)
		if i == 0 {
			userText += " key file /workspace/main.go"
		}
		assistantText := strings.Repeat("助手分析 ", 20) + "recent requirement " + strconv.Itoa(i)
		if i == 1 {
			assistantText += " artifact /outputs/report.html"
		}
		history = append(history, schema.UserMessage(userText), schema.AssistantMessage(assistantText, nil))
	}
	return history
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
