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

package middleware

import (
	"strings"
	"testing"

	oplogmw "github.com/ynet-dev/ynet-studio/backend/api/middleware/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

func TestBuildEventFields(t *testing.T) {
	rule := &oplogmw.RouteRule{
		Module: "workflow", ResourceType: 6, Action: "update",
		DescTemplate: "更新了工作流", ResourceIDFrom: "body:workflow_id", ResourceNameFrom: "body:name",
	}
	ev := buildEvent(rule, "POST", "/api/workflow_api/save", 999,
		nil, map[string]any{"space_id": "7", "workflow_id": "55", "name": "wf1"}, nil,
		`{"space_id":"7","workflow_id":"55","name":"wf1"}`, 200, "1.2.3.4", 12, "lg1", 1000)

	if ev.SpaceID != 7 {
		t.Fatalf("space_id want 7, got %d", ev.SpaceID)
	}
	if ev.ResourceID != 55 {
		t.Fatalf("resource_id want 55, got %d", ev.ResourceID)
	}
	if ev.ResourceName != "wf1" {
		t.Fatalf("resource_name want wf1, got %q", ev.ResourceName)
	}
	if ev.OperatorID != 999 {
		t.Fatalf("operator_id want 999, got %d", ev.OperatorID)
	}
	if ev.Status != entity.StatusSuccess {
		t.Fatalf("status want success, got %d", ev.Status)
	}
	if ev.Module != "workflow" || ev.Action != "update" || ev.ResourceType != 6 {
		t.Fatalf("bad semantic fields: %+v", ev)
	}
	if ev.Method != "POST" || ev.Path != "/api/workflow_api/save" {
		t.Fatalf("bad method/path: %+v", ev)
	}
	if ev.LogID != "lg1" || ev.ClientIP != "1.2.3.4" || ev.DurationMs != 12 || ev.CreatedAt != 1000 {
		t.Fatalf("bad io fields: %+v", ev)
	}
}

func TestBuildEventSpaceIDFromPath(t *testing.T) {
	rule := &oplogmw.RouteRule{
		Module: "agent", ResourceType: 4, Action: "delete",
		DescTemplate: "删除了智能体", ResourceIDFrom: "path:agent_id",
	}
	pattern := "/api/space/:space_id/hi-agents/:agent_id"
	path := "/api/space/88/hi-agents/22"
	segs := oplogmw.PathSegs(pattern, path)
	ev := buildEvent(rule, "DELETE", path, 1,
		nil, nil, segs, "", 200, "0.0.0.0", 1, "", 1)

	if ev.SpaceID != 88 {
		t.Fatalf("space_id from path want 88, got %d", ev.SpaceID)
	}
	if ev.ResourceID != 22 {
		t.Fatalf("resource_id from path want 22, got %d", ev.ResourceID)
	}
}

func TestBuildEventFailStatus(t *testing.T) {
	rule := &oplogmw.RouteRule{Module: "workflow", ResourceType: 6, Action: "create", DescTemplate: "创建了工作流"}
	ev := buildEvent(rule, "POST", "/api/workflow_api/create", 5,
		nil, map[string]any{}, nil, "", 500, "1.1.1.1", 3, "", 2)

	if ev.Status != entity.StatusFail {
		t.Fatalf("status want fail for 500, got %d", ev.Status)
	}
	if ev.SpaceID != 0 {
		t.Fatalf("space_id want 0 when absent, got %d", ev.SpaceID)
	}
}

func TestBuildEventSummaryTruncated(t *testing.T) {
	rule := &oplogmw.RouteRule{Module: "workflow", ResourceType: 6, Action: "create", DescTemplate: "创建了工作流"}
	raw := strings.Repeat("x", 1000)
	ev := buildEvent(rule, "POST", "/api/workflow_api/create", 5,
		nil, map[string]any{}, nil, raw, 200, "1.1.1.1", 3, "", 2)

	if len(ev.RequestSummary) != opLogMaxSummary {
		t.Fatalf("summary want truncated to %d, got %d", opLogMaxSummary, len(ev.RequestSummary))
	}
}

func TestRedactSummarySensitive(t *testing.T) {
	body := `{"name":"u1","password":"hunter2","api_key":"abc","Authorization":"Bearer x","keep":"ok"}`
	got := redactSummary([]byte(body))
	if strings.Contains(got, "hunter2") {
		t.Fatalf("password value should be redacted, got %q", got)
	}
	if strings.Contains(got, "abc") {
		t.Fatalf("api_key value should be redacted, got %q", got)
	}
	if strings.Contains(got, "Bearer x") {
		t.Fatalf("authorization value should be redacted, got %q", got)
	}
	if !strings.Contains(got, `"***"`) {
		t.Fatalf("redacted marker missing, got %q", got)
	}
	if !strings.Contains(got, `"u1"`) || !strings.Contains(got, `"ok"`) {
		t.Fatalf("non-sensitive values should remain, got %q", got)
	}
}

func TestRedactSummaryNonJSONFallback(t *testing.T) {
	raw := strings.Repeat("x", 1000)
	got := redactSummary([]byte(raw))
	if len(got) != opLogMaxSummary {
		t.Fatalf("non-json body want truncated to %d, got %d", opLogMaxSummary, len(got))
	}
}

func TestRedactSummaryPlainJSONUnchanged(t *testing.T) {
	body := `{"name":"wf1","workflow_id":"55"}`
	got := redactSummary([]byte(body))
	if !strings.Contains(got, `"wf1"`) || !strings.Contains(got, `"55"`) {
		t.Fatalf("plain body should be preserved, got %q", got)
	}
	if strings.Contains(got, "***") {
		t.Fatalf("plain body should not be redacted, got %q", got)
	}
}

func TestStripQuery(t *testing.T) {
	if got := stripQuery("/api/x?a=1&b=2"); got != "/api/x" {
		t.Fatalf("want /api/x, got %q", got)
	}
	if got := stripQuery("/api/x"); got != "/api/x" {
		t.Fatalf("want /api/x, got %q", got)
	}
}
