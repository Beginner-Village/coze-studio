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
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

func testManager(runner sandbox.Runner) *Manager {
	cfg := DefaultConfig()
	m := New(runner, NewMemRegistry(), nil, cfg)
	var clk int64 = 1000
	m.now = func() int64 { return clk }
	return m
}

func TestEnsureColdStartRegisters(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	m := testManager(fr)

	if err := m.EnsureSandbox(ctx, "u1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if fr.CreateCount["u1"] != 1 {
		t.Fatalf("create count = %d, want 1", fr.CreateCount["u1"])
	}
	e, ok, _ := m.reg.Get(ctx, "u1")
	if !ok || e.State != sandbox.StateRunning {
		t.Fatalf("registry entry = %+v ok=%v", e, ok)
	}
}

func TestEnsureReuseDoesNotRecreate(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	m := testManager(fr)
	_ = m.EnsureSandbox(ctx, "u1")
	_ = m.EnsureSandbox(ctx, "u1")
	_ = m.EnsureSandbox(ctx, "u1")
	if fr.CreateCount["u1"] != 1 {
		t.Fatalf("create count = %d, want 1 (reuse)", fr.CreateCount["u1"])
	}
}

func TestEnsureResumesPaused(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	m := testManager(fr)
	_ = m.EnsureSandbox(ctx, "u1")
	// 模拟被挂起。
	_ = m.Pause(ctx, "u1")
	if st, _ := fr.State(ctx, "u1"); st != sandbox.StatePaused {
		t.Fatalf("expected paused, got %v", st)
	}
	// 再次 ensure 应 resume，而不是新建。
	if err := m.EnsureSandbox(ctx, "u1"); err != nil {
		t.Fatalf("ensure resume: %v", err)
	}
	if st, _ := fr.State(ctx, "u1"); st != sandbox.StateRunning {
		t.Fatalf("expected running after resume, got %v", st)
	}
	if fr.CreateCount["u1"] != 1 {
		t.Fatalf("create count = %d, want 1 (resume not recreate)", fr.CreateCount["u1"])
	}
}

func TestEnsureRecreatesIfRuntimeDead(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	m := testManager(fr)
	_ = m.EnsureSandbox(ctx, "u1")
	// 运行时被外部杀掉，但注册表还在。
	fr.setState("u1", sandbox.StateDead)
	if err := m.EnsureSandbox(ctx, "u1"); err != nil {
		t.Fatalf("ensure recreate: %v", err)
	}
	if fr.CreateCount["u1"] != 2 {
		t.Fatalf("create count = %d, want 2 (recreate after dead)", fr.CreateCount["u1"])
	}
}

func TestEnsureConcurrentSingleFlight(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	m := testManager(fr)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.EnsureSandbox(ctx, "u1")
		}()
	}
	wg.Wait()
	if fr.CreateCount["u1"] != 1 {
		t.Fatalf("create count = %d, want 1 (singleflight dedup)", fr.CreateCount["u1"])
	}
}

func TestToolMethodsEnsureAndTouch(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	m := testManager(fr)

	if err := m.WriteFile(ctx, "u1", "/workspace/a.txt", []byte("x")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if fr.CreateCount["u1"] != 1 {
		t.Fatalf("write should cold-start once, got %d", fr.CreateCount["u1"])
	}
	b, err := m.ReadFile(ctx, "u1", "/workspace/a.txt")
	if err != nil || string(b) != "x" {
		t.Fatalf("read: err=%v b=%q", err, b)
	}
	res, err := m.Exec(ctx, "u1", "echo hi", 5)
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("exec: err=%v res=%+v", err, res)
	}
}
