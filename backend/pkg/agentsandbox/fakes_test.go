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

package agentsandbox

import (
	"context"
	"sync"

	sandbox "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

// fakeRunner 是内存版 sandbox.Runner，用于单测生命周期逻辑。
type fakeRunner struct {
	mu          sync.Mutex
	states      map[string]sandbox.State
	files       map[string]map[string][]byte
	CreateCount map[string]int
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		states:      map[string]sandbox.State{},
		files:       map[string]map[string][]byte{},
		CreateCount: map[string]int{},
	}
}

func (f *fakeRunner) Create(_ context.Context, req *sandbox.CreateRequest) (*sandbox.CreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[req.SandboxID] = sandbox.StateRunning
	f.files[req.SandboxID] = map[string][]byte{}
	f.CreateCount[req.SandboxID]++
	return &sandbox.CreateResponse{SandboxID: req.SandboxID, State: sandbox.StateRunning}, nil
}

func (f *fakeRunner) Exec(_ context.Context, req *sandbox.ExecRequest) (*sandbox.ExecResponse, error) {
	return &sandbox.ExecResponse{Stdout: "ok", ExitCode: 0}, nil
}

func (f *fakeRunner) WriteFile(_ context.Context, req *sandbox.WriteFileRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.files[req.SandboxID] == nil {
		f.files[req.SandboxID] = map[string][]byte{}
	}
	f.files[req.SandboxID][req.Path] = req.Content
	return nil
}

func (f *fakeRunner) ReadFile(_ context.Context, req *sandbox.ReadFileRequest) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.files[req.SandboxID][req.Path], nil
}

func (f *fakeRunner) ListFiles(_ context.Context, req *sandbox.ListFilesRequest) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []string{}
	for p := range f.files[req.SandboxID] {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeRunner) Pause(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[id] = sandbox.StatePaused
	return nil
}

func (f *fakeRunner) Resume(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[id] = sandbox.StateRunning
	return nil
}

func (f *fakeRunner) Kill(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[id] = sandbox.StateDead
	delete(f.files, id)
	return nil
}

func (f *fakeRunner) State(_ context.Context, id string) (sandbox.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st, ok := f.states[id]
	if !ok {
		return sandbox.StateDead, nil
	}
	return st, nil
}

// setState 测试辅助：模拟运行时状态漂移（如被外部 kill）。
func (f *fakeRunner) setState(id string, st sandbox.State) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[id] = st
}

var _ sandbox.Runner = (*fakeRunner)(nil)

// fakeStorage 是内存版 storage.Storage。
type fakeStorage struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func newFakeStorage() *fakeStorage { return &fakeStorage{objs: map[string][]byte{}} }

func (s *fakeStorage) PutObject(_ context.Context, key string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]byte, len(content))
	copy(cp, content)
	s.objs[key] = cp
	return nil
}

func (s *fakeStorage) GetObject(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.objs[key], nil
}

var _ Blob = (*fakeStorage)(nil)
