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
	"fmt"
	"strings"
	"testing"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	sbx "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
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
	if strings.HasPrefix(cmd, "cd '/skills/") && strings.Contains(cmd, "' 2>/dev/null && find . -type f") {
		skillPath := strings.TrimSuffix(strings.TrimPrefix(strings.Split(cmd, " 2>/dev/null && find . -type f")[0], "cd '"), "'")
		prefix := skillPath + "/"
		var files []string
		for p := range m.files {
			if strings.HasPrefix(p, prefix) {
				files = append(files, strings.TrimPrefix(p, prefix))
			}
		}
		return &sbx.ExecResponse{Stdout: strings.Join(files, "\n") + "\n", Stderr: "", ExitCode: 0}, nil
	}
	if strings.HasPrefix(cmd, "rm -f -- '") && strings.HasSuffix(cmd, "'") {
		delete(m.files, strings.TrimSuffix(strings.TrimPrefix(cmd, "rm -f -- '"), "'"))
		return &sbx.ExecResponse{Stdout: "", Stderr: "", ExitCode: 0}, nil
	}
	if strings.HasPrefix(cmd, "rm -rf -- '") && strings.HasSuffix(cmd, "'") {
		prefix := strings.TrimSuffix(strings.TrimPrefix(cmd, "rm -rf -- '"), "'") + "/"
		for p := range m.files {
			if strings.HasPrefix(p, prefix) {
				delete(m.files, p)
			}
		}
		return &sbx.ExecResponse{Stdout: "", Stderr: "", ExitCode: 0}, nil
	}
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
func (m *fakeSandboxMgr) EditFile(_ context.Context, key, path, oldStr, newStr string, replaceAll bool) (int, error) {
	m.lastKey, m.lastPath = key, path
	cur := string(m.files[path])
	n := strings.Count(cur, oldStr)
	if n == 0 {
		return 0, fmt.Errorf("old_string not found")
	}
	if replaceAll {
		m.files[path] = []byte(strings.ReplaceAll(cur, oldStr, newStr))
		return n, nil
	}
	m.files[path] = []byte(strings.Replace(cur, oldStr, newStr, 1))
	return 1, nil
}
func (m *fakeSandboxMgr) Grep(_ context.Context, key, pattern, path string) (string, error) {
	m.lastKey = key
	return "(no matches)", nil
}
func (m *fakeSandboxMgr) Glob(_ context.Context, key, pattern string) (string, error) {
	m.lastKey = key
	return "(no files matched)", nil
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

func (m *fakeSandboxMgr) CheckpointTo(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (m *fakeSandboxMgr) RestoreFrom(_ context.Context, _, _ string) error {
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
	if !strings.HasPrefix(k1, "a2-u") || strings.ContainsAny(k1, "@/ ") {
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
	tools := newSandboxTools("ukey", false)
	if len(tools) != 8 {
		t.Fatalf("want 8 tools, got %d", len(tools))
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

func TestUpdatePlanToolAcceptsCompletedStatus(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	up := &updatePlanTool{key: "ukey"}
	out, err := up.InvokableRun(context.Background(),
		`{"plan":[{"content":"read input","status":"completed"},{"content":"write output","status":"pending"}]}`)
	if err != nil {
		t.Fatalf("update_plan err=%v", err)
	}
	if !strings.Contains(out, "1/2 done") {
		t.Fatalf("completed status should count as done: %q", out)
	}
	if !strings.Contains(out, "[x] read input") {
		t.Fatalf("completed status should render as checked: %q", out)
	}
}

func TestSandboxToolsNilWhenNoSVC(t *testing.T) {
	crosssandbox.SetDefaultSVC(nil)
	if newSandboxTools("k", false) != nil {
		t.Fatal("expected nil tools when sandbox svc not set")
	}
}

func TestTruncateForModel(t *testing.T) {
	short := "hello world"
	if got := truncateForModel(short); got != short {
		t.Fatalf("short input should be returned unchanged, got %q", got)
	}

	long := strings.Repeat("a", defaultMaxToolOutputBytes*2)
	got := truncateForModel(long)
	if len(got) >= len(long) {
		t.Fatalf("long input should be truncated, len(got)=%d len(long)=%d", len(got), len(long))
	}
	if !strings.Contains(got, "[truncated") {
		t.Fatalf("truncated output should contain marker, got prefix %q", got[:64])
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

func TestSkillsPathGuardBlocksInstanceWrites(t *testing.T) {
	if !pathIsUnderSkills("/skills/pdf/SKILL.md") {
		t.Fatal("should detect /skills path")
	}
	if pathIsUnderSkills("/workspace/out.txt") {
		t.Fatal("should not flag /workspace path")
	}
	if err := guardSkillWrite(true, "/skills/pdf/x.py"); err == nil {
		t.Fatal("write under /skills must be rejected for instances")
	}
	if err := guardSkillWrite(false, "/skills/pdf/x.py"); err != nil {
		t.Fatal("non-instance writes must be allowed")
	}
}

func TestSkillsPathGuardResistsTraversalBypass(t *testing.T) {
	// Traversal paths that land inside /skills must be detected.
	if !pathIsUnderSkills("/workspace/../skills/evil.py") {
		t.Fatal("path /workspace/../skills/evil.py resolves to /skills/evil.py — must be blocked")
	}
	if !pathIsUnderSkills("../skills/evil.py") {
		t.Fatal("path ../skills/evil.py resolves to /skills/evil.py — must be blocked")
	}

	// Traversal path that escapes /skills must NOT be flagged as skills.
	if pathIsUnderSkills("/skills/../etc/passwd") {
		t.Fatal("path /skills/../etc/passwd resolves to /etc/passwd — must not be blocked as /skills")
	}

	// guardSkillWrite must reject the traversal bypass for instances.
	if err := guardSkillWrite(true, "/workspace/../skills/evil.py"); err == nil {
		t.Fatal("guardSkillWrite must block /workspace/../skills/evil.py for readonly instance")
	}
}

// TestInstanceReadonlySkillsFlag verifies that newSandboxTools(key, true) wires
// readonlySkills=true into the write tools, so /skills writes are rejected.
func TestInstanceReadonlySkillsFlag(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	ctx := context.Background()

	// readonlySkills=true: write to /skills must fail.
	tools := newSandboxTools("k", true)
	if len(tools) != 8 {
		t.Fatalf("want 8 tools, got %d", len(tools))
	}

	wf := &writeFileTool{key: "k", readonlySkills: true}
	out, _ := wf.InvokableRun(ctx, `{"path":"/skills/foo/SKILL.md","content":"x"}`)
	if !strings.Contains(out, "permission denied") && !strings.Contains(out, "read-only") {
		t.Fatalf("instance write to /skills must be denied, got: %q", out)
	}

	// readonlySkills=false (default agent): same path must succeed.
	wfNormal := &writeFileTool{key: "k", readonlySkills: false}
	out2, err := wfNormal.InvokableRun(ctx, `{"path":"/skills/foo/SKILL.md","content":"x"}`)
	if err != nil || strings.Contains(out2, "permission denied") {
		t.Fatalf("normal agent write to /skills must be allowed, got: %q err=%v", out2, err)
	}
}

// TestNewSandboxToolsReadonlyFlag verifies that the readonlySkills bool param
// is threaded into the tool structs returned by newSandboxTools.
func TestNewSandboxToolsReadonlyFlag(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	toolsNormal := newSandboxTools("k", false)
	toolsInstance := newSandboxTools("k", true)
	if len(toolsNormal) != 8 || len(toolsInstance) != 8 {
		t.Fatalf("want 8 tools each, got normal=%d instance=%d", len(toolsNormal), len(toolsInstance))
	}

	// The run_bash, write_file, edit_file tools from the instance set must carry
	// readonlySkills=true. We can verify indirectly: bash write to /skills should
	// produce an error for instance but succeed for normal agent.
	ctx := context.Background()
	for _, tt := range toolsInstance {
		info, _ := tt.Info(ctx)
		if info.Name == "run_bash" {
			out, _ := tt.InvokableRun(ctx, `{"command":"cp /workspace/a.txt /skills/a.txt"}`)
			if !strings.Contains(out, "permission denied") && !strings.Contains(out, "read-only") {
				t.Fatalf("instance run_bash writing to /skills must be denied: %q", out)
			}
		}
	}
	for _, tt := range toolsNormal {
		info, _ := tt.Info(ctx)
		if info.Name == "run_bash" {
			out, err := tt.InvokableRun(ctx, `{"command":"cp /workspace/a.txt /skills/a.txt"}`)
			if err != nil || strings.Contains(out, "permission denied") {
				t.Fatalf("normal agent run_bash writing to /skills must be allowed: %q err=%v", out, err)
			}
		}
	}
}
