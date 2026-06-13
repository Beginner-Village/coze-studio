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

import (
	"context"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// Reaper 周期扫描注册表，按空闲阈值挂起/回收沙箱。
type Reaper struct {
	mgr      *Manager
	interval time.Duration
}

// NewReaper 构造空闲回收器。
func NewReaper(mgr *Manager, interval time.Duration) *Reaper {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Reaper{mgr: mgr, interval: interval}
}

// Start 启动后台回收循环，直到 ctx 取消。
func (r *Reaper) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.sweep(ctx, r.mgr.now()); err != nil {
					logs.CtxWarnf(ctx, "sandbox reaper sweep error: %v", err)
				}
			}
		}
	}()
}

// sweep 执行一次回收判定。idle>IdleKillSec→回收；idle>IdlePauseSec 且仍 running→挂起。
func (r *Reaper) sweep(ctx context.Context, now int64) error {
	entries, err := r.mgr.reg.List(ctx)
	if err != nil {
		return err
	}
	cfg := r.mgr.cfg
	for _, e := range entries {
		idle := now - e.LastActiveUnix
		switch {
		case cfg.IdleKillSec > 0 && idle >= cfg.IdleKillSec:
			if err := r.mgr.Destroy(ctx, e.SandboxID); err != nil {
				logs.CtxWarnf(ctx, "reaper destroy %s: %v", e.SandboxID, err)
			}
		case cfg.IdlePauseSec > 0 && idle >= cfg.IdlePauseSec && e.State == sandbox.StateRunning:
			if err := r.mgr.Pause(ctx, e.SandboxID); err != nil {
				logs.CtxWarnf(ctx, "reaper pause %s: %v", e.SandboxID, err)
			}
		}
	}
	return nil
}
