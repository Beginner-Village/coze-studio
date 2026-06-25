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

package aiproduct

import (
	"context"
	"strings"
	"testing"
)

type fakeSandbox struct {
	synced     []string
	execCmds   []string
	checkpoint string
	failOnCmd  string
	destroyed  bool
}

func (f *fakeSandbox) EnsureSandbox(_ context.Context, _ string) error { return nil }
func (f *fakeSandbox) SyncSkill(_ context.Context, _, name string, _ map[string]string) error {
	f.synced = append(f.synced, name)
	return nil
}
func (f *fakeSandbox) Exec(_ context.Context, _, cmd string, _ int) (string, string, int, error) {
	f.execCmds = append(f.execCmds, cmd)
	if f.failOnCmd != "" && strings.Contains(cmd, f.failOnCmd) {
		return "", "boom", 1, nil
	}
	return "ok", "", 0, nil
}
func (f *fakeSandbox) Checkpoint(_ context.Context, _, objectKey string) (string, error) {
	f.checkpoint = objectKey
	return "sha256:deadbeef", nil
}
func (f *fakeSandbox) Destroy(_ context.Context, _ string) error { f.destroyed = true; return nil }

func TestBuildTemplateInjectsSkillsInstallsDepsCheckpoints(t *testing.T) {
	fb := &fakeSandbox{}
	res, err := BuildTemplate(context.Background(), fb, BuildTemplateRequest{
		BuildKey:  "build-1",
		ObjectKey: "templates/agent_app/100/1.tgz",
		Skills: []BuildSkill{
			{Name: "pdf-tools", Files: map[string]string{"SKILL.md": "x"}, PipDeps: []string{"pypdf"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Status != "ready" || res.ContentHash != "sha256:deadbeef" {
		t.Fatalf("bad result: %#v", res)
	}
	if len(fb.synced) != 1 || fb.synced[0] != "pdf-tools" {
		t.Fatalf("skill not injected: %#v", fb.synced)
	}
	if fb.checkpoint != "templates/agent_app/100/1.tgz" {
		t.Fatalf("checkpoint key wrong: %s", fb.checkpoint)
	}
	if !fb.destroyed {
		t.Fatalf("build sandbox not destroyed")
	}
	joined := strings.Join(fb.execCmds, " | ")
	if !strings.Contains(joined, "pip install") || !strings.Contains(joined, "pypdf") {
		t.Fatalf("pip deps not installed: %s", joined)
	}
}

func TestBuildTemplateFailsWhenDependencyInstallFails(t *testing.T) {
	fb := &fakeSandbox{failOnCmd: "pip install"}
	res, err := BuildTemplate(context.Background(), fb, BuildTemplateRequest{
		BuildKey:  "build-2",
		ObjectKey: "templates/agent_app/100/2.tgz",
		Skills:    []BuildSkill{{Name: "x", Files: map[string]string{"SKILL.md": "x"}, PipDeps: []string{"bad"}}},
	})
	if err != nil {
		t.Fatalf("builder must not return a Go error on business failure: %v", err)
	}
	if res.Status != "failed" || !strings.Contains(res.Detail, "boom") {
		t.Fatalf("expected failed result with detail, got %#v", res)
	}
	if !fb.destroyed {
		t.Fatalf("build sandbox must be destroyed even on failure")
	}
}
