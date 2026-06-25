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

package docker

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

func TestBuildCreateArgs(t *testing.T) {
	t.Setenv("SANDBOX_DATA_DIR", "") // 强制走默认根目录,避免环境变量干扰
	got := buildCreateArgs(&sandbox.CreateRequest{
		SandboxID: "sb1", Image: "python:3.11-slim", MemoryMB: 512, CPUs: 1,
		Env: map[string]string{"FOO": "bar"},
	}, "ynet-sb-")
	want := []string{
		"run", "-d", "--name", "ynet-sb-sb1",
		"-v", "/var/lib/ynet-sandboxes/sb1/workspace:/workspace",
		"-v", "/var/lib/ynet-sandboxes/sb1/uploads:/uploads",
		"-v", "/var/lib/ynet-sandboxes/sb1/outputs:/outputs",
		"-v", "/var/lib/ynet-sandboxes/sb1/skills:/skills",
		"--memory", "512m", "--memory-swap", "512m", "--cpus", "1.00",
		"--pids-limit", "512",
		"-e", "FOO=bar",
		"-w", "/workspace",
		"python:3.11-slim", "sleep", "infinity",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildCreateArgsReadonlySkills(t *testing.T) {
	t.Setenv("SANDBOX_DATA_DIR", "")
	got := buildCreateArgs(&sandbox.CreateRequest{
		SandboxID: "sb1", Image: "python:3.11-slim", ReadonlySkills: true,
	}, "ynet-sb-")
	joined := strings.Join(got, " ")
	// /skills must be mounted read-only (and only once).
	if !strings.Contains(joined, "/var/lib/ynet-sandboxes/sb1/skills:/skills:ro") {
		t.Fatalf("expected read-only /skills mount, got %v", got)
	}
	if strings.Contains(joined, "/var/lib/ynet-sandboxes/sb1/skills:/skills ") ||
		strings.HasSuffix(joined, "/var/lib/ynet-sandboxes/sb1/skills:/skills") {
		t.Fatalf("read-write /skills mount must not coexist with the :ro one, got %v", got)
	}
}

func TestBuildExecArgs(t *testing.T) {
	got := buildExecArgs(&sandbox.ExecRequest{SandboxID: "sb1", Cmd: "echo hi", WorkDir: ""}, "ynet-sb-")
	want := []string{"exec", "-w", "/workspace", "ynet-sb-sb1", "bash", "-lc", "echo hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
