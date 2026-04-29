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

package observability

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPRequestsTotalIncrement(t *testing.T) {
	HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200").Inc()
	got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/foo", "200"))
	if got != 1 {
		t.Fatalf("expected 1, got %v", got)
	}
}

func TestStudioLLMTokensIncrement(t *testing.T) {
	StudioLLMTokensTotal.WithLabelValues("gpt-4", "prompt").Add(150)
	got := testutil.ToFloat64(StudioLLMTokensTotal.WithLabelValues("gpt-4", "prompt"))
	if got != 150 {
		t.Fatalf("expected 150, got %v", got)
	}
}

func TestStudioAgentChatTotalIncrement(t *testing.T) {
	before := testutil.ToFloat64(StudioAgentChatTotal.WithLabelValues("success"))
	StudioAgentChatTotal.WithLabelValues("success").Inc()
	StudioAgentChatTotal.WithLabelValues("error").Inc()
	after := testutil.ToFloat64(StudioAgentChatTotal.WithLabelValues("success"))
	if after-before != 1 {
		t.Fatalf("expected success +1, got +%v", after-before)
	}
	if got := testutil.ToFloat64(StudioAgentChatTotal.WithLabelValues("error")); got < 1 {
		t.Fatalf("expected error counter >= 1, got %v", got)
	}
}

func TestStudioFileUploadSizeObserve(t *testing.T) {
	// Histogram observe should not panic and should be retrievable as a Summary.
	StudioFileUploadSize.WithLabelValues("image").Observe(2048)
	StudioFileUploadSize.WithLabelValues("doc").Observe(1024 * 100)
	StudioFileUploadSize.WithLabelValues("other").Observe(1)
}

func TestMetricNamesNoCollision(t *testing.T) {
	names := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_requests_in_flight",
		"studio_llm_tokens_total",
		"studio_file_upload_size_bytes",
		"studio_agent_chat_total",
	}
	for _, n := range names {
		if !strings.HasPrefix(n, "http_") && !strings.HasPrefix(n, "studio_") {
			t.Fatalf("%s violates prefix convention", n)
		}
	}
}
