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
	"context"
	"os/exec"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

func dockerAvailable() bool {
	return exec.Command("docker", "version").Run() == nil
}

func TestDockerRunnerEndToEnd(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	ctx := context.Background()
	r := NewRunner()
	const id = "p1-e2e"
	_ = r.Kill(ctx, id)

	if _, err := r.Create(ctx, &sandbox.CreateRequest{SandboxID: id, MemoryMB: 256, CPUs: 1}); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer r.Kill(ctx, id)

	// exec bash
	res, err := r.Exec(ctx, &sandbox.ExecRequest{SandboxID: id, Cmd: "echo hello-sandbox"})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("exec: err=%v res=%+v", err, res)
	}
	if got := res.Stdout; got != "hello-sandbox\n" {
		t.Fatalf("stdout=%q", got)
	}

	// write + read file
	if err := r.WriteFile(ctx, &sandbox.WriteFileRequest{SandboxID: id, Path: "/workspace/sub/a.txt", Content: []byte("data123")}); err != nil {
		t.Fatalf("write: %v", err)
	}
	content, err := r.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: id, Path: "/workspace/sub/a.txt"})
	if err != nil || string(content) != "data123" {
		t.Fatalf("read: err=%v content=%q", err, content)
	}

	// list
	files, err := r.ListFiles(ctx, &sandbox.ListFilesRequest{SandboxID: id, Path: "/workspace/sub"})
	if err != nil || len(files) != 1 || files[0] != "a.txt" {
		t.Fatalf("list: err=%v files=%v", err, files)
	}

	// run a python script we wrote
	if err := r.WriteFile(ctx, &sandbox.WriteFileRequest{SandboxID: id, Path: "/workspace/hi.py", Content: []byte("print('py-ok')")}); err != nil {
		t.Fatalf("write py: %v", err)
	}
	res2, err := r.Exec(ctx, &sandbox.ExecRequest{SandboxID: id, Cmd: "python /workspace/hi.py"})
	if err != nil || res2.ExitCode != 0 || res2.Stdout != "py-ok\n" {
		t.Fatalf("python exec: err=%v res=%+v", err, res2)
	}

	// state
	st, _ := r.State(ctx, id)
	if st != sandbox.StateRunning {
		t.Fatalf("state=%v", st)
	}
}
