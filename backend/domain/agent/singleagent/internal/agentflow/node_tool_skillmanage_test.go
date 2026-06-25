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

func TestSkillManageToolWritesStandardSkillFile(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"write_file","name":"demo","path":"scripts/run.py","content":"print('ok')\n"}`)
	if err != nil {
		t.Fatalf("write_file err=%v", err)
	}
	if !strings.Contains(out, "Wrote skill file scripts/run.py") {
		t.Fatalf("write_file out=%q", out)
	}
	if string(fm.files["/skills/demo/scripts/run.py"]) != "print('ok')\n" {
		t.Fatalf("standard skill file not written: %v", fm.files)
	}
}

func TestSkillManageToolListsStandardSkillFiles(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/SKILL.md":             []byte("# Demo\n"),
		"/skills/demo/scripts/run.py":       []byte("print('ok')\n"),
		"/skills/demo/references/guide.md":  []byte("# Guide\n"),
		"/skills/other/templates/report.md": []byte("# Other\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"list","name":"demo"}`)
	if err != nil {
		t.Fatalf("list skill files err=%v", err)
	}
	for _, want := range []string{"SKILL.md", "scripts/run.py", "references/guide.md"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list skill files should contain %q, got %q", want, out)
		}
	}
	if strings.Contains(out, "templates/report.md") {
		t.Fatalf("list skill files should not include another skill's files: %q", out)
	}
}

func TestSkillManageToolReadsStandardSkillFile(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/references/guide.md": []byte("# Guide\nUse carefully.\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"read","name":"demo","path":"references/guide.md"}`)
	if err != nil {
		t.Fatalf("read standard file err=%v", err)
	}
	if out != "# Guide\nUse carefully.\n" {
		t.Fatalf("read standard file out=%q", out)
	}
}

func TestSkillManageToolDiffsStandardSkillFile(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/templates/report.md": []byte("old title\nsame line\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"diff","name":"demo","path":"templates/report.md","content":"new title\nsame line\n"}`)
	if err != nil {
		t.Fatalf("diff err=%v", err)
	}
	for _, want := range []string{"--- current/templates/report.md", "+++ proposed/templates/report.md", "-old title", "+new title"} {
		if !strings.Contains(out, want) {
			t.Fatalf("diff should contain %q, got %q", want, out)
		}
	}
}

func TestSkillManageToolEditsStandardSkillFile(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/SKILL.md": []byte("description: old\n\nUse old flow.\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"edit","name":"demo","path":"SKILL.md","old_string":"old","new_string":"new","replace_all":true}`)
	if err != nil {
		t.Fatalf("edit err=%v", err)
	}
	if !strings.Contains(out, "Edited skill file SKILL.md") || !strings.Contains(out, "2 replacement") {
		t.Fatalf("edit out=%q", out)
	}
	if string(fm.files["/skills/demo/SKILL.md"]) != "description: new\n\nUse new flow.\n" {
		t.Fatalf("standard skill file not edited: %q", string(fm.files["/skills/demo/SKILL.md"]))
	}
}

func TestSkillManageToolRemovesStandardSkillFile(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/scripts/run.py": []byte("print('ok')\n"),
		"/skills/demo/SKILL.md":       []byte("# Demo\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"remove_file","name":"demo","path":"scripts/run.py"}`)
	if err != nil {
		t.Fatalf("remove_file err=%v", err)
	}
	if !strings.Contains(out, "Removed skill file scripts/run.py") {
		t.Fatalf("remove_file out=%q", out)
	}
	if _, ok := fm.files["/skills/demo/scripts/run.py"]; ok {
		t.Fatalf("standard skill file was not removed: %v", fm.files)
	}
	if string(fm.files["/skills/demo/SKILL.md"]) != "# Demo\n" {
		t.Fatalf("remove_file should not delete other skill files: %v", fm.files)
	}
}

func TestSkillManageToolRejectsRemovingNonStandardSkillPath(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/lib/helper.py": []byte("x"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"remove_file","name":"demo","path":"lib/helper.py"}`)
	if err != nil {
		t.Fatalf("remove_file invalid path err=%v", err)
	}
	if !strings.Contains(out, "Error:") || !strings.Contains(out, "scripts/") {
		t.Fatalf("remove_file should reject non-standard paths, got %q", out)
	}
	if string(fm.files["/skills/demo/lib/helper.py"]) != "x" {
		t.Fatalf("invalid remove_file should not mutate files: %v", fm.files)
	}
}

func TestSkillManageToolRejectsRemovingSkillMarkdownEntry(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/SKILL.md": []byte("# Demo\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"remove_file","name":"demo","path":"SKILL.md"}`)
	if err != nil {
		t.Fatalf("remove_file SKILL.md err=%v", err)
	}
	if !strings.Contains(out, "Error:") || !strings.Contains(out, "SKILL.md") {
		t.Fatalf("remove_file should reject deleting SKILL.md directly, got %q", out)
	}
	if string(fm.files["/skills/demo/SKILL.md"]) != "# Demo\n" {
		t.Fatalf("remove_file should not delete SKILL.md: %v", fm.files)
	}
}

func TestSkillManageToolDeletesSkillFolder(t *testing.T) {
	fm := &fakeSandboxMgr{files: map[string][]byte{
		"/skills/demo/SKILL.md":         []byte("# Demo\n"),
		"/skills/demo/assets/data.json": []byte("{}\n"),
		"/skills/other/SKILL.md":        []byte("# Other\n"),
	}}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	sm := &skillManageTool{key: "k"}
	out, err := sm.InvokableRun(context.Background(), `{"action":"delete","name":"demo"}`)
	if err != nil {
		t.Fatalf("delete err=%v", err)
	}
	if !strings.Contains(out, "Deleted skill demo") {
		t.Fatalf("delete out=%q", out)
	}
	for p := range fm.files {
		if strings.HasPrefix(p, "/skills/demo/") {
			t.Fatalf("delete should remove all demo files, still have %s in %v", p, fm.files)
		}
	}
	if string(fm.files["/skills/other/SKILL.md"]) != "# Other\n" {
		t.Fatalf("delete should not remove other skills: %v", fm.files)
	}
}
