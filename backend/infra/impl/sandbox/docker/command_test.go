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
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

func TestBuildCreateArgs(t *testing.T) {
	got := buildCreateArgs(&sandbox.CreateRequest{
		SandboxID: "sb1", Image: "", MemoryMB: 512, CPUs: 1,
		Env: map[string]string{"FOO": "bar"},
	}, "ynet-sb-")
	want := []string{
		"run", "-d", "--name", "ynet-sb-sb1",
		"--memory", "512m", "--cpus", "1.00",
		"-e", "FOO=bar",
		"-w", "/workspace",
		"python:3.11-slim", "sleep", "infinity",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildExecArgs(t *testing.T) {
	got := buildExecArgs(&sandbox.ExecRequest{SandboxID: "sb1", Cmd: "echo hi", WorkDir: ""}, "ynet-sb-")
	want := []string{"exec", "-w", "/workspace", "ynet-sb-sb1", "bash", "-lc", "echo hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
