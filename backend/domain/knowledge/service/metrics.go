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

package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ParseFileSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "knowledge_parse_file_size_bytes",
			Help:    "File size in bytes processed by knowledge parser.",
			Buckets: prometheus.ExponentialBuckets(1024, 4, 12),
		},
		[]string{"file_type"},
	)

	ParseDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "knowledge_parse_duration_seconds",
			Help:    "Knowledge parse duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"file_type", "outcome"},
	)

	ParseFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "knowledge_parse_failed_total",
			Help: "Total number of failed knowledge parses, by reason.",
		},
		[]string{"file_type", "reason"},
	)

	LargeFileWorkerActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "knowledge_large_file_worker_active",
			Help: "Number of currently active large-file parse workers.",
		},
	)

	LargeFileWorkerQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "knowledge_large_file_worker_queue_depth",
			Help: "Number of large-file parse tasks waiting in queue.",
		},
	)

	DocumentReaperCleanedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "knowledge_reaper_cleaned_total",
			Help: "Total documents cleaned up by the document reaper.",
		},
	)
)

// FileTypeForLabel 把 filename 转为 metric label，限制 cardinality。
func FileTypeForLabel(filename string) string {
	idx := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "unknown"
	}
	ext := filename[idx+1:]
	if ext == "" {
		return "unknown"
	}
	switch ext {
	case "txt", "md", "json", "csv", "pdf", "docx", "doc", "ppt", "pptx",
		"jpg", "jpeg", "png", "gif", "webp", "bmp":
		return ext
	}
	return "other"
}
