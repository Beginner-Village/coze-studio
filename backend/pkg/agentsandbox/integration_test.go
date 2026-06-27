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
	"os/exec"
	"testing"

	dockerimpl "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/docker"
)

func dockerAvailable() bool {
	return exec.Command("docker", "version").Run() == nil &&
		exec.Command("docker", "image", "inspect", "ynet-sandbox:rich").Run() == nil
}

// TestCheckpointRestoreRoundTrip 用真实 docker 验证 workspace 持久化往返。
func TestCheckpointRestoreRoundTrip(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker sandbox image ynet-sandbox:rich not available")
	}
	ctx := context.Background()
	runner := dockerimpl.NewRunner()
	store := newFakeStorage()
	cfg := DefaultConfig()
	cfg.MemoryMB = 256
	m := New(runner, NewMemRegistry(), store, cfg)

	const key = "p2-roundtrip"
	_ = runner.Kill(ctx, key)
	defer runner.Kill(ctx, key)

	// 冷启动 + 写文件。
	if err := m.WriteFile(ctx, key, "/workspace/keep/data.txt", []byte("persist-me")); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Checkpoint 到 MinIO(fake)。
	if err := m.Checkpoint(ctx, key); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	// 模拟回收：kill + 删注册表。
	if err := m.runner.Kill(ctx, key); err != nil {
		t.Fatalf("kill: %v", err)
	}
	_ = m.reg.Delete(ctx, key)

	// 再次访问 → 冷启动从 MinIO 还原。
	b, err := m.ReadFile(ctx, key, "/workspace/keep/data.txt")
	if err != nil {
		t.Fatalf("read after restore: %v", err)
	}
	if string(b) != "persist-me" {
		t.Fatalf("restored content = %q, want persist-me", b)
	}
}
