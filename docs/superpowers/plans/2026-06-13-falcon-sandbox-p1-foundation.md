# 猎鹰沙箱 P1 · 沙箱底座（抽象层 + Docker 后端 + 最小 demo）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 ynet-studio 后端加一层会话级沙箱运行时抽象（`infra/contract/sandbox`），并实现一个可在任意装了 Docker 的机器上运行的 Docker 后端，跑通"创建沙箱 → exec bash → 读写文件 → 销毁"的最小端到端链路。

**Architecture:** 仿照现有 `infra/contract/coderunner` 的 contract/impl 分层。Contract 定义 `Runner` 接口（Create/Exec/WriteFile/ReadFile/ListFiles/Pause/Resume/Kill）+ 纯数据类型，零第三方依赖。Docker 后端通过 `os/exec` 调本机 `docker` CLI（避免引入重型 docker SDK，交叉编译无 cgo 负担），每个沙箱 = 一个长驻容器，文件/命令通过 `docker exec`/`docker cp` 操作。后续 CubeSandbox 后端实现同一接口即可替换。

**Tech Stack:** Go 1.x、现有 testify/标准库 testing、Docker CLI（`python:3.11-slim` 基础镜像）。

---

### Task 1: 定义沙箱 contract（接口 + 类型）

**Files:**
- Create: `backend/infra/contract/sandbox/sandbox.go`
- Test: `backend/infra/contract/sandbox/sandbox_test.go`

- [ ] **Step 1: Write the failing test**（验证类型与零值常量存在、可编译）

```go
package sandbox

import "testing"

func TestStatesDefined(t *testing.T) {
	if StateRunning == "" || StatePaused == "" || StateDead == "" {
		t.Fatal("sandbox states must be non-empty")
	}
	req := &ExecRequest{Cmd: "echo hi", TimeoutSec: 5}
	if req.Cmd != "echo hi" {
		t.Fatal("ExecRequest.Cmd not wired")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./infra/contract/sandbox/...`
Expected: FAIL（package/类型未定义，编译错误）

- [ ] **Step 3: Write the contract**

```go
package sandbox

import "context"

type State string

const (
	StateRunning State = "running"
	StatePaused  State = "paused"
	StateDead    State = "dead"
)

// CreateRequest 创建一个会话级沙箱。
type CreateRequest struct {
	// SandboxID 调用方指定的稳定 ID（建议 connector/agent_id + user_id 组合 hash）。
	SandboxID string
	// Image 基础镜像，空则后端用默认（python:3.11-slim）。
	Image string
	// Env 注入的环境变量。
	Env map[string]string
	// MemoryMB / CPUs 资源上限，0 表示后端默认。
	MemoryMB int
	CPUs     float64
}

type CreateResponse struct {
	SandboxID string
	State     State
}

type ExecRequest struct {
	SandboxID  string
	Cmd        string
	WorkDir    string // 默认 /workspace
	TimeoutSec int    // 0 表示后端默认 60s
}

type ExecResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type WriteFileRequest struct {
	SandboxID string
	Path      string
	Content   []byte
}

type ReadFileRequest struct {
	SandboxID string
	Path      string
}

type ListFilesRequest struct {
	SandboxID string
	Path      string
}

// Runner 是会话级沙箱运行时的统一抽象。Docker / CubeSandbox / E2B 各实现一份。
type Runner interface {
	Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error)
	Exec(ctx context.Context, req *ExecRequest) (*ExecResponse, error)
	WriteFile(ctx context.Context, req *WriteFileRequest) error
	ReadFile(ctx context.Context, req *ReadFileRequest) ([]byte, error)
	ListFiles(ctx context.Context, req *ListFilesRequest) ([]string, error)
	Pause(ctx context.Context, sandboxID string) error
	Resume(ctx context.Context, sandboxID string) error
	Kill(ctx context.Context, sandboxID string) error
	// State 返回当前状态；沙箱不存在返回 StateDead, nil。
	State(ctx context.Context, sandboxID string) (State, error)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./infra/contract/sandbox/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/infra/contract/sandbox/
git commit -m "feat(sandbox): add session sandbox runner contract"
```

---

### Task 2: Docker 后端 —— 命令构造（纯函数，可单测，不依赖 docker）

**Files:**
- Create: `backend/infra/impl/sandbox/docker/command.go`
- Test: `backend/infra/impl/sandbox/docker/command_test.go`

把"生成 docker CLI 参数"抽成纯函数，便于不依赖真实 docker 做单元测试。

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./infra/impl/sandbox/docker/...`
Expected: FAIL（`buildCreateArgs`/`buildExecArgs` 未定义）

- [ ] **Step 3: Write command builders**

```go
package docker

import (
	"fmt"
	"sort"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

const (
	defaultImage   = "python:3.11-slim"
	defaultWorkDir = "/workspace"
)

func containerName(prefix, sandboxID string) string { return prefix + sandboxID }

func buildCreateArgs(req *sandbox.CreateRequest, prefix string) []string {
	image := req.Image
	if image == "" {
		image = defaultImage
	}
	args := []string{"run", "-d", "--name", containerName(prefix, req.SandboxID)}
	if req.MemoryMB > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", req.MemoryMB))
	}
	if req.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.2f", req.CPUs))
	}
	keys := make([]string, 0, len(req.Env))
	for k := range req.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, req.Env[k]))
	}
	args = append(args, "-w", defaultWorkDir, image, "sleep", "infinity")
	return args
}

func buildExecArgs(req *sandbox.ExecRequest, prefix string) []string {
	wd := req.WorkDir
	if wd == "" {
		wd = defaultWorkDir
	}
	return []string{"exec", "-w", wd, containerName(prefix, req.SandboxID), "bash", "-lc", req.Cmd}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./infra/impl/sandbox/docker/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/infra/impl/sandbox/docker/command.go backend/infra/impl/sandbox/docker/command_test.go
git commit -m "feat(sandbox): docker backend command builders with unit tests"
```

---

### Task 3: Docker 后端 —— Runner 实现（exec docker CLI）

**Files:**
- Create: `backend/infra/impl/sandbox/docker/runner.go`

实现 `sandbox.Runner`。命令执行走 `os/exec`，文件读写用 `docker exec sh -c 'cat > path'` / `cat path`，列目录用 `ls -1`。

- [ ] **Step 1: Write the implementation**

```go
package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
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
	if _, _, err := r.runDocker(ctx, nil, buildCreateArgs(req, r.prefix)...); err != nil {
		return nil, fmt.Errorf("docker run: %w", err)
	}
	// 确保 workspace 存在。
	if _, _, err := r.runDocker(ctx, nil, "exec", containerName(r.prefix, req.SandboxID), "mkdir", "-p", defaultWorkDir); err != nil {
		return nil, fmt.Errorf("mkdir workspace: %w", err)
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
	dir := req.Path[:strings.LastIndex(req.Path, "/")+1]
	if dir != "" {
		if _, _, err := r.runDocker(ctx, nil, "exec", containerName(r.prefix, req.SandboxID), "mkdir", "-p", dir); err != nil {
			return fmt.Errorf("mkdir for write: %w", err)
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
	_, _, err := r.runDocker(ctx, nil, "pause", containerName(r.prefix, sandboxID))
	return err
}

func (r *Runner) Resume(ctx context.Context, sandboxID string) error {
	_, _, err := r.runDocker(ctx, nil, "unpause", containerName(r.prefix, sandboxID))
	return err
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
```

- [ ] **Step 2: Verify it compiles**

Run: `cd backend && go build ./infra/impl/sandbox/...`
Expected: 编译通过（`var _ sandbox.Runner` 断言接口完整实现）

- [ ] **Step 3: Commit**

```bash
git add backend/infra/impl/sandbox/docker/runner.go
git commit -m "feat(sandbox): docker backend runner implementing sandbox.Runner"
```

---

### Task 4: 端到端集成测试（gated on docker）

**Files:**
- Create: `backend/infra/impl/sandbox/docker/runner_integration_test.go`

集成测试默认在 `docker` 不可用时 `t.Skip`，本机有 docker 时跑真实容器，验证整条链路。

- [ ] **Step 1: Write the integration test**

```go
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
```

- [ ] **Step 2: Run the integration test (with docker)**

Run: `cd backend && go test ./infra/impl/sandbox/docker/ -run TestDockerRunnerEndToEnd -v`
Expected: PASS（若本机无 docker → SKIP；本机有 docker → 真实跑通 create/exec/write/read/list/python/state）

- [ ] **Step 3: Commit**

```bash
git add backend/infra/impl/sandbox/docker/runner_integration_test.go
git commit -m "test(sandbox): docker backend end-to-end integration test"
```

---

## P1 验收标准

- `go build ./...` 全绿，`go test ./infra/contract/sandbox/... ./infra/impl/sandbox/...` 全绿。
- 本机有 docker 时，集成测试真实跑通：创建容器 → exec `echo` → 写文件 → 读文件 → 列目录 → 写并运行 python 脚本 → 查状态。
- 抽象层 `sandbox.Runner` 稳定，后续 CubeSandbox 后端只需实现同一接口。

## Self-Review notes
- Spec 覆盖：对应报告 §6 新增落点 `infra/contract/sandbox` + `impl/.../docker`、报告 P1「Go 调后端在沙箱跑 echo/读写文件」验收项。
- 无 placeholder：每步含真实代码与命令。
- 类型一致性：`Runner` 接口方法签名在 contract 定义，Docker impl 用 `var _ sandbox.Runner` 编译期校验一致。
