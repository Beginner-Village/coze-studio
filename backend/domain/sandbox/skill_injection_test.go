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
	"testing"

	dockerimpl "github.com/ynet-dev/ynet-studio/backend/infra/impl/sandbox/docker"
)

// TestSyncSkillInjectAndRun 用真实 docker 验证：注入技能脚本→run_bash 跑通；同版本去重跳过。
func TestSyncSkillInjectAndRun(t *testing.T) {
	if !dockerAvailable() {
		t.Skip("docker not available")
	}
	ctx := context.Background()
	runner := dockerimpl.NewRunner()
	cfg := DefaultConfig()
	cfg.MemoryMB = 256
	m := New(runner, NewMemRegistry(), nil, cfg)

	const key = "p4-skill"
	_ = runner.Kill(ctx, key)
	defer runner.Kill(ctx, key)

	files := map[string][]byte{
		"scripts/run.py": []byte("print('skill-ran')"),
	}
	if err := m.SyncSkill(ctx, key, "demo", files); err != nil {
		t.Fatalf("sync: %v", err)
	}
	res, err := m.Exec(ctx, key, "python /skills/demo/scripts/run.py", 30)
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("exec: err=%v res=%+v", err, res)
	}
	if res.Stdout != "skill-ran\n" {
		t.Fatalf("stdout=%q", res.Stdout)
	}

	// 同版本再次注入应跳过（hash 命中）。删除文件后再 sync 同内容，应因 hash 命中而不重写 → 文件仍缺失。
	if _, err := m.Exec(ctx, key, "rm -rf /skills/demo/scripts", 10); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if err := m.SyncSkill(ctx, key, "demo", files); err != nil {
		t.Fatalf("sync2: %v", err)
	}
	r2, _ := m.Exec(ctx, key, "test -f /skills/demo/scripts/run.py && echo yes || echo no", 10)
	if r2.Stdout != "no\n" {
		t.Fatalf("expected dedup to skip rewrite (no), got %q", r2.Stdout)
	}
}
