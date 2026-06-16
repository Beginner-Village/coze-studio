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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

type Runner struct {
	prefix string
}

func NewRunner() *Runner { return &Runner{prefix: "ynet-sb-"} }

func (r *Runner) runDocker(ctx context.Context, stdin []byte, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func (r *Runner) Create(ctx context.Context, req *sandbox.CreateRequest) (*sandbox.CreateResponse, error) {
	if _, stderr, err := r.runDocker(ctx, nil, buildCreateArgs(req, r.prefix)...); err != nil {
		return nil, fmt.Errorf("docker run: %w (%s)", err, stderr)
	}
	// 确保 workspace 存在。
	if _, stderr, err := r.runDocker(ctx, nil, "exec", containerName(r.prefix, req.SandboxID), "mkdir", "-p", defaultWorkDir); err != nil {
		return nil, fmt.Errorf("mkdir workspace: %w (%s)", err, stderr)
	}
	return &sandbox.CreateResponse{SandboxID: req.SandboxID, State: sandbox.StateRunning}, nil
}

func (r *Runner) Exec(ctx context.Context, req *sandbox.ExecRequest) (*sandbox.ExecResponse, error) {
	timeout := time.Duration(req.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stdout, stderr, err := r.runDocker(cctx, nil, buildExecArgs(req, r.prefix)...)
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return nil, fmt.Errorf("docker exec: %w (%s)", err, stderr)
		}
	}
	return &sandbox.ExecResponse{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}, nil
}

func (r *Runner) WriteFile(ctx context.Context, req *sandbox.WriteFileRequest) error {
	if i := strings.LastIndex(req.Path, "/"); i > 0 {
		dir := req.Path[:i]
		if _, stderr, err := r.runDocker(ctx, nil, "exec", containerName(r.prefix, req.SandboxID), "mkdir", "-p", dir); err != nil {
			return fmt.Errorf("mkdir for write: %w (%s)", err, stderr)
		}
	}
	args := []string{"exec", "-i", containerName(r.prefix, req.SandboxID), "sh", "-c", "cat > " + shellQuote(req.Path)}
	if _, stderr, err := r.runDocker(ctx, req.Content, args...); err != nil {
		return fmt.Errorf("write file: %w (%s)", err, stderr)
	}
	return nil
}

func (r *Runner) ReadFile(ctx context.Context, req *sandbox.ReadFileRequest) ([]byte, error) {
	stdout, stderr, err := r.runDocker(ctx, nil, "exec", containerName(r.prefix, req.SandboxID), "cat", req.Path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w (%s)", err, stderr)
	}
	return []byte(stdout), nil
}

func (r *Runner) ListFiles(ctx context.Context, req *sandbox.ListFilesRequest) ([]string, error) {
	stdout, stderr, err := r.runDocker(ctx, nil, "exec", containerName(r.prefix, req.SandboxID), "ls", "-1", req.Path)
	if err != nil {
		return nil, fmt.Errorf("list files: %w (%s)", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

func (r *Runner) Pause(ctx context.Context, sandboxID string) error {
	_, stderr, err := r.runDocker(ctx, nil, "pause", containerName(r.prefix, sandboxID))
	if err != nil {
		return fmt.Errorf("pause: %w (%s)", err, stderr)
	}
	return nil
}

func (r *Runner) Resume(ctx context.Context, sandboxID string) error {
	_, stderr, err := r.runDocker(ctx, nil, "unpause", containerName(r.prefix, sandboxID))
	if err != nil {
		return fmt.Errorf("resume: %w (%s)", err, stderr)
	}
	return nil
}

func (r *Runner) Kill(ctx context.Context, sandboxID string) error {
	_, _, _ = r.runDocker(ctx, nil, "rm", "-f", containerName(r.prefix, sandboxID))
	return nil
}

func (r *Runner) State(ctx context.Context, sandboxID string) (sandbox.State, error) {
	stdout, _, err := r.runDocker(ctx, nil, "inspect", "-f", "{{.State.Status}}", containerName(r.prefix, sandboxID))
	if err != nil {
		return sandbox.StateDead, nil
	}
	switch strings.TrimSpace(stdout) {
	case "running":
		return sandbox.StateRunning, nil
	case "paused":
		return sandbox.StatePaused, nil
	default:
		return sandbox.StateDead, nil
	}
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

var _ sandbox.Runner = (*Runner)(nil)
