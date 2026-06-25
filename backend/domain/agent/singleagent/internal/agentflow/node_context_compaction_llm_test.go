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
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// erroringChatModel always fails Generate, to prove the compaction LLM path falls
// back to the rule-based summary on error.
type erroringChatModel struct{}

func (erroringChatModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	return nil, errors.New("model unavailable")
}
func (erroringChatModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("model unavailable")
}
func (e erroringChatModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return e, nil
}

func TestLLMCompactionSummary(t *testing.T) {
	ctx := context.Background()
	old := []*schema.Message{
		{Role: schema.User, Content: "build a parser"},
		{Role: schema.Assistant, Content: "wrote parser.go and ran tests"},
	}

	// With a working model, the LLM summary is used.
	if got := llmCompactionSummary(ctx, &fakeToolCallingModel{}, old, ""); got != "ok" {
		t.Fatalf("expected LLM summary %q, got %q", "ok", got)
	}
	// nil model => empty so the caller falls back to the rule-based summary.
	if got := llmCompactionSummary(ctx, nil, old, ""); got != "" {
		t.Fatalf("nil model must yield empty (rule-based fallback), got %q", got)
	}
	// Erroring model => empty (graceful fallback, never blocks the run).
	if got := llmCompactionSummary(ctx, erroringChatModel{}, old, ""); got != "" {
		t.Fatalf("erroring model must yield empty (rule-based fallback), got %q", got)
	}
}

func TestBuildContextCompactionSummaryPrefersLLMWhenAvailable(t *testing.T) {
	ctx := context.Background()
	old := []*schema.Message{
		{Role: schema.User, Content: "legacy requirement 0: keep records"},
		{Role: schema.Assistant, Content: "acknowledged"},
	}

	// With a model: the LLM summary ("ok") is used.
	withModel := buildContextCompactionSummary(ctx, nil, "", &AgentRequest{}, old, contextSummaryPath, &fakeToolCallingModel{})
	if withModel == nil || strings.TrimSpace(withModel.Summary) != "ok" {
		t.Fatalf("expected LLM summary 'ok', got %+v", withModel)
	}
	if withModel.Version != "v1" {
		t.Fatalf("summary version must stay v1 for contract stability, got %q", withModel.Version)
	}

	// Without a model: the rule-based summary (containing the old content) is used.
	ruleBased := buildContextCompactionSummary(ctx, nil, "", &AgentRequest{}, old, contextSummaryPath, nil)
	if ruleBased == nil || !strings.Contains(ruleBased.Summary, "legacy requirement 0") {
		t.Fatalf("rule-based fallback should contain old content, got %+v", ruleBased)
	}
}
