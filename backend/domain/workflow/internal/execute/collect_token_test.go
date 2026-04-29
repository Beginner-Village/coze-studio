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

package execute

import (
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ynet-dev/ynet-studio/backend/pkg/observability"
)

func TestRecordLLMTokenMetrics(t *testing.T) {
	const modelName = "test-record-llm-token-metrics-model"
	usage := &model.TokenUsage{
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}

	before := testutil.ToFloat64(observability.StudioLLMTokensTotal.WithLabelValues(modelName, "prompt"))
	recordLLMTokenMetrics(modelName, usage)
	after := testutil.ToFloat64(observability.StudioLLMTokensTotal.WithLabelValues(modelName, "prompt"))
	if after-before != 100 {
		t.Fatalf("prompt tokens: expected +100, got +%v", after-before)
	}

	got := testutil.ToFloat64(observability.StudioLLMTokensTotal.WithLabelValues(modelName, "completion"))
	if got != 50 {
		t.Fatalf("completion tokens: expected 50, got %v", got)
	}
}

func TestRecordLLMTokenMetrics_NilUsageIsNoOp(t *testing.T) {
	// Should not panic.
	recordLLMTokenMetrics("any", nil)
}

func TestRecordLLMTokenMetrics_EmptyModelDefaultsToUnknown(t *testing.T) {
	usage := &model.TokenUsage{PromptTokens: 7}
	before := testutil.ToFloat64(observability.StudioLLMTokensTotal.WithLabelValues("unknown", "prompt"))
	recordLLMTokenMetrics("", usage)
	after := testutil.ToFloat64(observability.StudioLLMTokensTotal.WithLabelValues("unknown", "prompt"))
	if after-before != 7 {
		t.Fatalf("expected +7 unknown prompt tokens, got +%v", after-before)
	}
}
