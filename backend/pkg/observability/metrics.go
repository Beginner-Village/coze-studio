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
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// IsMetricsEnabled reports whether Prometheus exposure is enabled.
// Opt-in: only true when METRICS_ENABLED is set to "true"/"1"/"on"/"yes"
// (case-insensitive). Default (env empty or any other value) is disabled.
func IsMetricsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("METRICS_ENABLED")))
	return v == "true" || v == "1" || v == "on" || v == "yes"
}

// Generic RED: shared by every HTTP handler.
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests total counted by method, normalized path, and status",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method, normalized path, and status",
			Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Currently in-flight HTTP requests",
		},
	)
)

// Studio business metrics.
var (
	StudioLLMTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "studio_llm_tokens_total",
			Help: "Total LLM tokens consumed by model and kind (prompt|completion)",
		},
		[]string{"model", "kind"},
	)

	StudioFileUploadSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "studio_file_upload_size_bytes",
			Help:    "File upload size by file kind (image|doc|other)",
			Buckets: prometheus.ExponentialBuckets(1024, 4, 10), // 1K -> 1G
		},
		[]string{"kind"},
	)

	StudioAgentChatTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "studio_agent_chat_total",
			Help: "Agent chat invocation counts by result",
		},
		[]string{"result"}, // success | error
	)

	// Super-agent run observability (P1 hardening): so production can see run volume,
	// latency and error rate, and whether the per-user sandbox concurrency limit is firing.
	SuperAgentRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "studio_super_agent_runs_total",
			Help: "Super-agent runs by transport (stream|sync) and result (success|error)",
		},
		[]string{"transport", "result"},
	)

	SuperAgentRunDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "studio_super_agent_run_duration_seconds",
			Help:    "Super-agent run latency by transport and result",
			Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300},
		},
		[]string{"transport", "result"},
	)

	SandboxExecRejectedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "studio_sandbox_exec_rejected_total",
			Help: "Sandbox exec calls rejected by the per-user concurrency limit",
		},
	)
)
