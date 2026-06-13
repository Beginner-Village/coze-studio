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
	sbx "github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

type fakeSandboxMgr struct {
	lastKey   string
	lastCmd   string
	lastPath  string
	lastWrite []byte
	files     map[string][]byte
}

func (m *fakeSandboxMgr) Exec(_ context.Context, key, cmd string, _ int) (*sbx.ExecResponse, error) {
	m.lastKey, m.lastCmd = key, cmd
	return &sbx.ExecResponse{Stdout: "hello\n", Stderr: "", ExitCode: 0}, nil
}
func (m *fakeSandboxMgr) ReadFile(_ context.Context, key, path string) ([]byte, error) {
	m.lastKey, m.lastPath = key, path
	return m.files[path], nil
}
func (m *fakeSandboxMgr) WriteFile(_ context.Context, key, path string, content []byte) error {
	m.lastKey, m.lastPath, m.lastWrite = key, path, content
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[path] = content
	return nil
}
func (m *fakeSandboxMgr) ListFiles(_ context.Context, key, path string) ([]string, error) {
	m.lastKey, m.lastPath = key, path
	return []string{"a.txt", "b.py"}, nil
}
func (m *fakeSandboxMgr) SyncSkill(_ context.Context, key, name string, files map[string][]byte) error {
	m.lastKey = key
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	for rel, c := range files {
		m.files["/skills/"+name+"/"+rel] = c
	}
	return nil
}

func TestSandboxKeyStableAndSafe(t *testing.T) {
	k1 := sandboxKeyFor(1, 2, "user@x")
	k2 := sandboxKeyFor(1, 2, "user@x")
	k3 := sandboxKeyFor(1, 2, "other")
	if k1 != k2 {
		t.Fatal("key must be stable for same inputs")
	}
	if k1 == k3 {
		t.Fatal("key must differ for different user")
	}
	if !strings.HasPrefix(k1, "u") || strings.ContainsAny(k1, "@/ ") {
		t.Fatalf("key not container-safe: %q", k1)
	}
}

func TestResolvePath(t *testing.T) {
	if resolvePath("") != "/workspace" {
		t.Fatal("empty -> /workspace")
	}
	if resolvePath("a/b.txt") != "/workspace/a/b.txt" {
		t.Fatal("relative -> under /workspace")
	}
	if resolvePath("/skills/x/run.sh") != "/skills/x/run.sh" {
		t.Fatal("absolute preserved")
	}
}

func TestSandboxToolsInvoke(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	ctx := context.Background()
	tools := newSandboxTools("ukey")
	if len(tools) != 5 {
		t.Fatalf("want 5 tools, got %d", len(tools))
	}

	// run_bash
	rb := &runBashTool{key: "ukey"}
	out, err := rb.InvokableRun(ctx, `{"command":"echo hello"}`)
	if err != nil || !strings.Contains(out, "exit_code: 0") || !strings.Contains(out, "hello") {
		t.Fatalf("run_bash out=%q err=%v", out, err)
	}
	if fm.lastCmd != "echo hello" || fm.lastKey != "ukey" {
		t.Fatalf("run_bash not forwarded: %+v", fm)
	}

	// write_file
	wf := &writeFileTool{key: "ukey"}
	out, err = wf.InvokableRun(ctx, `{"path":"out.txt","content":"data"}`)
	if err != nil || !strings.Contains(out, "Wrote 4 bytes") {
		t.Fatalf("write_file out=%q err=%v", out, err)
	}
	if fm.lastPath != "/workspace/out.txt" || string(fm.lastWrite) != "data" {
		t.Fatalf("write_file not forwarded: %+v", fm)
	}

	// read_file
	rf := &readFileTool{key: "ukey"}
	out, err = rf.InvokableRun(ctx, `{"path":"out.txt"}`)
	if err != nil || out != "data" {
		t.Fatalf("read_file out=%q err=%v", out, err)
	}

	// list_files
	lf := &listFilesTool{key: "ukey"}
	out, err = lf.InvokableRun(ctx, `{"path":""}`)
	if err != nil || !strings.Contains(out, "a.txt") || !strings.Contains(out, "b.py") {
		t.Fatalf("list_files out=%q err=%v", out, err)
	}
}

func TestUpdatePlanTool(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	up := &updatePlanTool{key: "ukey"}
	out, err := up.InvokableRun(context.Background(),
		`{"plan":[{"content":"read input","status":"done"},{"content":"process","status":"in_progress"},{"content":"write output","status":"pending"}]}`)
	if err != nil {
		t.Fatalf("update_plan err=%v", err)
	}
	if !strings.Contains(out, "1/3 done") {
		t.Fatalf("progress wrong: %q", out)
	}
	if !strings.Contains(out, "[x] read input") || !strings.Contains(out, "[~] process") || !strings.Contains(out, "[ ] write output") {
		t.Fatalf("render wrong: %q", out)
	}
	// persisted to sandbox plan file
	if _, ok := fm.files[planFilePath]; !ok {
		t.Fatalf("plan not persisted; files=%v", fm.files)
	}
}

func TestSandboxToolsNilWhenNoSVC(t *testing.T) {
	crosssandbox.SetDefaultSVC(nil)
	if newSandboxTools("k") != nil {
		t.Fatal("expected nil tools when sandbox svc not set")
	}
}

func TestSandboxToolsEnabled(t *testing.T) {
	t.Setenv("SANDBOX_TOOLS_ENABLED", "")
	if sandboxToolsEnabled(0) {
		t.Fatal("no skills + no env -> disabled")
	}
	if !sandboxToolsEnabled(2) {
		t.Fatal("has skills -> enabled")
	}
	t.Setenv("SANDBOX_TOOLS_ENABLED", "true")
	if !sandboxToolsEnabled(0) {
		t.Fatal("env force -> enabled")
	}
}
