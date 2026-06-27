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
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

func TestReaperPausesIdle(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	cfg := DefaultConfig()
	cfg.IdlePauseSec = 100
	cfg.IdleKillSec = 1000
	m := New(fr, NewMemRegistry(), nil, cfg)
	m.now = func() int64 { return 1000 } // active at 1000
	_ = m.EnsureSandbox(ctx, "u1")

	r := NewReaper(m, 0)
	// now=1050: idle=50 < 100 → no change
	if err := r.sweep(ctx, 1050); err != nil {
		t.Fatal(err)
	}
	if st, _ := fr.State(ctx, "u1"); st != sandbox.StateRunning {
		t.Fatalf("should still run, got %v", st)
	}
	// now=1150: idle=150 >= 100 → pause
	if err := r.sweep(ctx, 1150); err != nil {
		t.Fatal(err)
	}
	if st, _ := fr.State(ctx, "u1"); st != sandbox.StatePaused {
		t.Fatalf("should be paused, got %v", st)
	}
	e, ok, _ := m.reg.Get(ctx, "u1")
	if !ok || e.State != sandbox.StatePaused {
		t.Fatalf("registry should record paused, got %+v ok=%v", e, ok)
	}
}

func TestReaperKillsLongIdle(t *testing.T) {
	ctx := context.Background()
	fr := newFakeRunner()
	cfg := DefaultConfig()
	cfg.IdlePauseSec = 100
	cfg.IdleKillSec = 1000
	m := New(fr, NewMemRegistry(), nil, cfg)
	m.now = func() int64 { return 1000 }
	_ = m.EnsureSandbox(ctx, "u1")

	r := NewReaper(m, 0)
	// now=2500: idle=1500 >= 1000 → destroy
	if err := r.sweep(ctx, 2500); err != nil {
		t.Fatal(err)
	}
	if st, _ := fr.State(ctx, "u1"); st != sandbox.StateDead {
		t.Fatalf("should be dead, got %v", st)
	}
	if _, ok, _ := m.reg.Get(ctx, "u1"); ok {
		t.Fatal("registry entry should be deleted after destroy")
	}
}
