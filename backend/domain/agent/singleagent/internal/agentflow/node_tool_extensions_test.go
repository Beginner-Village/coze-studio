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
	"strings"
	"testing"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

func TestSuperAgentExtensionRegistry(t *testing.T) {
	tools := newSuperAgentExtensionTools("ukey")
	if len(tools) == 0 {
		t.Fatal("expected at least one registered extension tool")
	}
	// web_fetch 应在其中
	var found bool
	for _, tl := range tools {
		info, err := tl.Info(context.Background())
		if err != nil {
			t.Fatalf("Info err: %v", err)
		}
		if info.Name == "web_fetch" {
			found = true
		}
	}
	if !found {
		t.Fatal("web_fetch not registered as super-agent extension tool")
	}
}

func TestWebFetchTool(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)
	ctx := context.Background()

	wf := &webFetchTool{key: "ukey"}

	// 非 http(s) URL 被拒
	out, err := wf.InvokableRun(ctx, `{"url":"ftp://x"}`)
	if err != nil || !strings.Contains(out, "must start with http") {
		t.Fatalf("want url validation error, got out=%q err=%v", out, err)
	}

	// 合法 URL → 走沙箱 curl（fakeSandboxMgr.Exec 返回 stdout "hello"）
	out, err = wf.InvokableRun(ctx, `{"url":"http://example.com"}`)
	if err != nil || !strings.Contains(out, "hello") {
		t.Fatalf("web_fetch out=%q err=%v", out, err)
	}
	if !strings.Contains(fm.lastCmd, "python3") || !strings.Contains(fm.lastCmd, "http://example.com") {
		t.Fatalf("web_fetch did not run python fetch in sandbox: %q", fm.lastCmd)
	}
}
