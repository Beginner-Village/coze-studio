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

package operationlog

import "testing"

func TestMatchExact(t *testing.T) {
	r := Match("POST", "/api/workflow_api/save")
	if r == nil || r.Action != "update" || r.Module != "workflow" {
		t.Fatalf("want workflow update rule, got %+v", r)
	}
}

func TestMatchCaseInsensitiveMethod(t *testing.T) {
	r := Match("post", "/api/workflow_api/create")
	if r == nil || r.Action != "create" {
		t.Fatalf("want create rule for lowercase method, got %+v", r)
	}
}

func TestMatchMiss(t *testing.T) {
	if r := Match("GET", "/api/workflow_api/list"); r != nil {
		t.Fatalf("GET/read should not match, got %+v", r)
	}
	if r := Match("POST", "/api/workflow_api/list"); r != nil {
		t.Fatalf("non-whitelisted POST should not match, got %+v", r)
	}
}

func TestMatchWildcard(t *testing.T) {
	r := Match("DELETE", "/api/space/123/hi-agents/456")
	if r == nil || r.Action != "delete" || r.Module != "agent" {
		t.Fatalf("wildcard should match hi-agent delete, got %+v", r)
	}
	// 段数不一致不应命中
	if r := Match("DELETE", "/api/space/123/hi-agents"); r != nil {
		t.Fatalf("segment count mismatch should not match, got %+v", r)
	}
}

func TestMatchWildcardLength(t *testing.T) {
	// /api/space/:space_id (PUT) vs /api/space/:space_id/members/:user_id/role (PUT)
	r := Match("PUT", "/api/space/9/members/77/role")
	if r == nil || r.Action != "update_role" {
		t.Fatalf("want member update_role, got %+v", r)
	}
	r2 := Match("PUT", "/api/space/9")
	if r2 == nil || r2.Action != "update" || r2.Module != "space" {
		t.Fatalf("want space update, got %+v", r2)
	}
}

func TestMatchPath(t *testing.T) {
	if !matchPath("/api/res/:id", "/api/res/123") {
		t.Fatal("wildcard segment should match")
	}
	if matchPath("/api/res/:id", "/api/res") {
		t.Fatal("length mismatch should not match")
	}
	if matchPath("/api/res/x", "/api/res/y") {
		t.Fatal("literal mismatch should not match")
	}
}

func TestPathSegs(t *testing.T) {
	segs := PathSegs("/api/space/:space_id/hi-agents/:agent_id", "/api/space/11/hi-agents/22")
	if segs["space_id"] != "11" || segs["agent_id"] != "22" {
		t.Fatalf("bad path segs: %+v", segs)
	}
}

func TestExtractSpaceIDFromBody(t *testing.T) {
	got := ExtractSpaceID(nil, nil, map[string]any{"space_id": "123"})
	if got != 123 {
		t.Fatalf("want 123, got %d", got)
	}
	// float64 (json number 默认解码) 也兼容
	got = ExtractSpaceID(nil, nil, map[string]any{"space_id": float64(456)})
	if got != 456 {
		t.Fatalf("want 456 from float64, got %d", got)
	}
}

func TestExtractSpaceIDFromQuery(t *testing.T) {
	got := ExtractSpaceID(map[string]string{"space_id": "789"}, nil, nil)
	if got != 789 {
		t.Fatalf("want 789, got %d", got)
	}
}

func TestExtractSpaceIDMissing(t *testing.T) {
	if got := ExtractSpaceID(nil, nil, map[string]any{"other": "1"}); got != 0 {
		t.Fatalf("want 0 when absent, got %d", got)
	}
}

func TestExtractBySpecBody(t *testing.T) {
	got := ExtractBySpec("body:name", nil, nil, map[string]any{"name": "wf1"}, nil)
	if got != "wf1" {
		t.Fatalf("want wf1, got %q", got)
	}
	// body 数字 id
	got = ExtractBySpec("body:workflow_id", nil, nil, map[string]any{"workflow_id": "55"}, nil)
	if got != "55" {
		t.Fatalf("want 55, got %q", got)
	}
}

func TestExtractBySpecPath(t *testing.T) {
	got := ExtractBySpec("path:agent_id", nil, nil, nil, map[string]string{"agent_id": "22"})
	if got != "22" {
		t.Fatalf("want 22, got %q", got)
	}
}

func TestExtractBySpecEmptySpec(t *testing.T) {
	if got := ExtractBySpec("", nil, nil, nil, nil); got != "" {
		t.Fatalf("empty spec should yield empty, got %q", got)
	}
}

func TestParseInt64Public(t *testing.T) {
	if ParseInt64Public("100") != 100 {
		t.Fatal("ParseInt64Public(100) should be 100")
	}
	if ParseInt64Public("12a3") != 0 {
		t.Fatal("non-numeric should be 0")
	}
}
