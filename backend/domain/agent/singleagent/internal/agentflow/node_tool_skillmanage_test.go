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

func TestSkillManageTool(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	ctx := context.Background()
	sm := &skillManageTool{key: "k"}

	// create
	out, err := sm.InvokableRun(ctx, `{"action":"create","name":"demo","content":"x"}`)
	if err != nil || !strings.Contains(out, "Created skill demo") {
		t.Fatalf("create out=%q err=%v", out, err)
	}
	if string(fm.files["/skills/demo/SKILL.md"]) != "x" {
		t.Fatalf("skill file not written: %v", fm.files)
	}

	// invalid name
	out, err = sm.InvokableRun(ctx, `{"action":"create","name":"Bad Name","content":"y"}`)
	if err != nil || !strings.Contains(out, "Error") {
		t.Fatalf("invalid name should error: out=%q err=%v", out, err)
	}

	// read
	out, err = sm.InvokableRun(ctx, `{"action":"read","name":"demo"}`)
	if err != nil || out != "x" {
		t.Fatalf("read out=%q err=%v", out, err)
	}

	// list (fake Exec returns "hello\n")
	out, err = sm.InvokableRun(ctx, `{"action":"list"}`)
	if err != nil || !strings.Contains(out, "hello") {
		t.Fatalf("list out=%q err=%v", out, err)
	}
}
