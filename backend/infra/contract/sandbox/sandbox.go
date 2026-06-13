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
