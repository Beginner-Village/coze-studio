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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
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
	name := containerName(r.prefix, req.SandboxID)

	// 沙箱容器按 (connector/agent/user) 确定性命名,需幂等:同名容器若已存在则复用,
	// 避免再次 docker run 撞名报 125 Conflict(并防止误删用户已有的沙箱文件)。
	status, _, _ := r.runDocker(ctx, nil, "inspect", "-f", "{{.State.Status}}", name)
	switch strings.TrimSpace(status) {
	case "running":
		// 已就绪,直接复用
		_, _, _ = r.runDocker(ctx, nil, "exec", name, "mkdir", "-p", defaultWorkDir)
		return &sandbox.CreateResponse{SandboxID: req.SandboxID, State: sandbox.StateRunning}, nil
	case "paused":
		// 空闲被暂停过,恢复后复用
		if _, stderr, err := r.runDocker(ctx, nil, "unpause", name); err != nil {
			return nil, fmt.Errorf("docker unpause: %w (%s)", err, stderr)
		}
		_, _, _ = r.runDocker(ctx, nil, "exec", name, "mkdir", "-p", defaultWorkDir)
		return &sandbox.CreateResponse{SandboxID: req.SandboxID, State: sandbox.StateRunning}, nil
	case "exited", "created":
		// 已停止,启动后复用(保留其中的文件)
		if _, stderr, err := r.runDocker(ctx, nil, "start", name); err != nil {
			return nil, fmt.Errorf("docker start: %w (%s)", err, stderr)
		}
		_, _, _ = r.runDocker(ctx, nil, "exec", name, "mkdir", "-p", defaultWorkDir)
		return &sandbox.CreateResponse{SandboxID: req.SandboxID, State: sandbox.StateRunning}, nil
	}

	// 不存在,正常创建。
	if _, stderr, err := r.runDocker(ctx, nil, buildCreateArgs(req, r.prefix)...); err != nil {
		return nil, fmt.Errorf("docker run: %w (%s)", err, stderr)
	}
	// 确保 workspace 存在。
	if _, stderr, err := r.runDocker(ctx, nil, "exec", name, "mkdir", "-p", defaultWorkDir); err != nil {
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

// PurgeAgent 删除某智能体名下「所有用户/连接器」的沙箱容器及其宿主机持久化目录。
// 删智能体时调用,避免容器与数据目录成为孤儿。仅此处会删数据,reaper 的回收不删数据。
func (r *Runner) PurgeAgent(ctx context.Context, agentID int64) error {
	idStr := strconv.FormatInt(agentID, 10)
	// 1) 按容器名前缀枚举并强删该 agent 的所有沙箱容器(key 形如 a<id>-u<hash>)。
	namePrefix := r.prefix + "a" + idStr + "-"
	out, _, _ := r.runDocker(ctx, nil, "ps", "-aq", "--filter", "name="+namePrefix)
	for _, id := range strings.Fields(out) {
		_, _, _ = r.runDocker(ctx, nil, "rm", "-f", id)
	}
	// 2) 删宿主机数据目录(后端在容器内无法直接操作宿主机路径,用辅助容器 rm)。
	//    匹配 <root>/a<id>-* 覆盖该 agent 全部用户的目录(含已被 reaper 回收只剩目录的)。
	root := sandboxDataRoot()
	if _, stderr, err := r.runDocker(ctx, nil,
		"run", "--rm", "-v", root+":/data", resolveImage(""),
		"sh", "-c", "rm -rf /data/a"+idStr+"-*",
	); err != nil {
		return fmt.Errorf("purge agent %d data dirs: %w (%s)", agentID, err, stderr)
	}
	return nil
}

// WriteFilesTar 把多个文件打成 tar 通过单次 `docker exec -i tar x` 灌入,
// 避免逐文件 docker exec(技能文件夹常有上百个文件,逐个写会慢到几十秒)。
// files 的 key 为容器内绝对路径。
func (r *Runner) WriteFilesTar(ctx context.Context, sandboxID string, files map[string][]byte) error {
	if len(files) == 0 {
		return nil
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for p, content := range files {
		name := strings.TrimPrefix(p, "/")
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}); err != nil {
			return fmt.Errorf("tar header %s: %w", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			return fmt.Errorf("tar write %s: %w", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar close: %w", err)
	}
	name := containerName(r.prefix, sandboxID)
	if _, stderr, err := r.runDocker(ctx, buf.Bytes(), "exec", "-i", name, "tar", "-xpf", "-", "-C", "/"); err != nil {
		return fmt.Errorf("tar extract: %w (%s)", err, stderr)
	}
	return nil
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
