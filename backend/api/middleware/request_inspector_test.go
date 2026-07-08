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
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
)

func TestSuperAgentAppServerPathsNeedOpenAPIAuth(t *testing.T) {
	for _, path := range []string{
		"/api/super-agent/runs/create",
		"/api/super-agent/runs/get",
		"/api/super-agent/runs/list",
		"/api/super-agent/runs/reply",
		"/api/super-agent/runs/stream",
		"/api/super-agent/runs/cancel",
		"/api/super-agent/sessions/create",
		"/api/super-agent/sessions/get",
		"/api/super-agent/sessions/list",
		"/api/super-agent/sessions/rename",
		"/api/super-agent/sessions/delete",
		"/api/super-agent/harness/state",
		"/api/super-agent/harness/plan",
		"/api/super-agent/harness/tool-outputs",
		"/api/super-agent/harness/snapshot",
		"/api/super-agent/messages/list",
		"/api/super-agent/sandbox/exec",
		"/api/super-agent/traces/get",
		"/api/super-agent/artifacts/list",
		"/api/super-agent/artifacts/download",
		"/api/super-agent/artifacts/delete",
		"/api/super-agent/artifacts/move",
		"/api/super-agent/workspace/list",
		"/api/super-agent/workspace/read",
		"/api/super-agent/workspace/upload",
		"/api/super-agent/workspace/write",
		"/api/super-agent/workspace/download",
		"/api/super-agent/workspace/delete",
		"/api/super-agent/workspace/move",
		"/api/super-agent/workspace/mkdir",
		"/api/super-agent/workspace/stat",
		"/api/super-agent/workspace/grep",
		"/api/super-agent/workspace/glob",
		"/api/super-agent/workspace/edit",
		"/api/super-agent/workspace/patch",
		"/api/super-agent/skills/create",
		"/api/super-agent/skills/get",
		"/api/super-agent/skills/update",
		"/api/super-agent/skills/delete",
		"/api/super-agent/skills/publish",
		"/api/super-agent/skills/list",
		"/api/super-agent/marketplace/list",
		"/api/super-agent/marketplace/get",
		"/api/super-agent/marketplace/install",
		"/api/super-agent/runtime-config/get",
		"/api/super-agent/runtime-config/update",
		"/api/super-agent/runtime-config/delete",
		"/api/super-agent/approvals/list",
		"/api/super-agent/approvals/resolve",
		"/api/super-agent/products/list",
		"/api/super-agent/products/get",
		"/api/super-agent/products/install",
		"/api/super-agent/products/upgrade",
		"/api/super-agent/products/uninstall",
		"/api/super-agent/config/get",
		"/api/super-agent/config/update",
	} {
		t.Run(path, func(t *testing.T) {
			ctx := &app.RequestContext{Request: protocol.Request{}}
			ctx.Request.SetRequestURI(path)
			if !isNeedOpenapiAuth(ctx) {
				t.Fatalf("path %s should support OpenAPI bearer auth", path)
			}
		})
	}
}
