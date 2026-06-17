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

func TestMemoryTools(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	ctx := context.Background()

	recall := &memoryRecallTool{key: "k"}
	out, err := recall.InvokableRun(ctx, "")
	if err != nil || !strings.Contains(out, "(no memories") {
		t.Fatalf("empty recall out=%q err=%v", out, err)
	}

	save := &memorySaveTool{key: "k"}
	out, err = save.InvokableRun(ctx, `{"content":"likes Go"}`)
	if err != nil || !strings.Contains(out, "Now 1 memories") {
		t.Fatalf("save1 out=%q err=%v", out, err)
	}
	out, err = save.InvokableRun(ctx, `{"content":"timezone UTC+8"}`)
	if err != nil || !strings.Contains(out, "Now 2 memories") {
		t.Fatalf("save2 out=%q err=%v", out, err)
	}

	out, err = recall.InvokableRun(ctx, "")
	if err != nil {
		t.Fatalf("recall err=%v", err)
	}
	if !strings.Contains(out, "likes Go") || !strings.Contains(out, "timezone UTC+8") {
		t.Fatalf("recall missing entries: %q", out)
	}
}

func TestSuperAgentExtensionsRegistered(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	tools := newSuperAgentExtensionTools("k")
	names := map[string]bool{}
	for _, tl := range tools {
		info, err := tl.Info(context.Background())
		if err != nil {
			t.Fatalf("Info err=%v", err)
		}
		names[info.Name] = true
	}
	for _, want := range []string{"memory_save", "memory_recall", "skill_manage"} {
		if !names[want] {
			t.Fatalf("extension %q not registered; got %v", want, names)
		}
	}
}
