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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	sandbox "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
)

// Config 是会话管理器配置。
type Config struct {
	// IdlePauseSec 空闲多久后挂起（pause）。
	IdlePauseSec int64
	// IdleKillSec 空闲多久后回收（kill）。
	IdleKillSec int64
	// MemoryMB / CPUs 新建沙箱的资源上限。
	MemoryMB int
	CPUs     float64
	// Image 基础镜像，空则后端默认。
	Image string
	// WorkspacePrefix MinIO 中 workspace 归档前缀（如 "workspaces/"）。
	WorkspacePrefix string
}

// DefaultConfig 给出可用默认值。
func DefaultConfig() Config {
	return Config{
		IdlePauseSec:    600,  // 10min
		IdleKillSec:     3600, // 1h
		MemoryMB:        1024,
		CPUs:            1,
		WorkspacePrefix: "workspaces/",
	}
}

// Manager 管理 user_id 维度的会话级沙箱。
type Manager struct {
	runner sandbox.Runner
	reg    Registry
	store  Blob
	cfg    Config
	sf     singleflight.Group
	now    func() int64
}

// New 构造一个会话管理器。store 可为 nil（禁用持久化）。
func New(runner sandbox.Runner, reg Registry, store Blob, cfg Config) *Manager {
	return &Manager{
		runner: runner,
		reg:    reg,
		store:  store,
		cfg:    cfg,
		now:    func() int64 { return time.Now().Unix() },
	}
}

func (m *Manager) workspaceKey(sandboxID string) string {
	return m.cfg.WorkspacePrefix + sandboxID + ".tgz"
}

// EnsureSandbox 确保 key 对应的沙箱处于 running，必要时冷启动或 resume。并发调用经 singleflight 去重。
func (m *Manager) EnsureSandbox(ctx context.Context, key string) error {
	_, err, _ := m.sf.Do(key, func() (interface{}, error) {
		return nil, m.ensure(ctx, key)
	})
	return err
}

func (m *Manager) ensure(ctx context.Context, key string) error {
	entry, ok, err := m.reg.Get(ctx, key)
	if err != nil {
		return err
	}
	if ok {
		// 以真实运行时状态为准。
		st, _ := m.runner.State(ctx, key)
		switch st {
		case sandbox.StateRunning:
			return m.reg.Touch(ctx, key, m.now())
		case sandbox.StatePaused:
			if err := m.runner.Resume(ctx, key); err != nil {
				return err
			}
			_ = m.reg.SetState(ctx, key, sandbox.StateRunning)
			return m.reg.Touch(ctx, key, m.now())
		default:
			// 运行时已没了，清理注册表后走冷启动。
			_ = m.reg.Delete(ctx, key)
		}
		_ = entry
	}
	return m.coldStart(ctx, key)
}

func (m *Manager) coldStart(ctx context.Context, key string) error {
	if _, err := m.runner.Create(ctx, &sandbox.CreateRequest{
		SandboxID: key,
		Image:     m.cfg.Image,
		MemoryMB:  m.cfg.MemoryMB,
		CPUs:      m.cfg.CPUs,
	}); err != nil {
		return fmt.Errorf("create sandbox: %w", err)
	}
	if err := m.restore(ctx, key); err != nil {
		return fmt.Errorf("restore workspace: %w", err)
	}
	return m.reg.Put(ctx, &Entry{
		SandboxID:      key,
		State:          sandbox.StateRunning,
		LastActiveUnix: m.now(),
	})
}

// Exec 在 key 沙箱执行命令（先 ensure）。
func (m *Manager) Exec(ctx context.Context, key, cmd string, timeoutSec int) (*sandbox.ExecResponse, error) {
	if err := m.EnsureSandbox(ctx, key); err != nil {
		return nil, err
	}
	res, err := m.runner.Exec(ctx, &sandbox.ExecRequest{SandboxID: key, Cmd: cmd, TimeoutSec: timeoutSec})
	if err != nil {
		return nil, err
	}
	_ = m.reg.Touch(ctx, key, m.now())
	return res, nil
}

// WriteFile 写文件（先 ensure）。
func (m *Manager) WriteFile(ctx context.Context, key, path string, content []byte) error {
	if err := m.EnsureSandbox(ctx, key); err != nil {
		return err
	}
	if err := m.runner.WriteFile(ctx, &sandbox.WriteFileRequest{SandboxID: key, Path: path, Content: content}); err != nil {
		return err
	}
	_ = m.reg.Touch(ctx, key, m.now())
	return nil
}

// ReadFile 读文件（先 ensure）。
func (m *Manager) ReadFile(ctx context.Context, key, path string) ([]byte, error) {
	if err := m.EnsureSandbox(ctx, key); err != nil {
		return nil, err
	}
	b, err := m.runner.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: key, Path: path})
	if err != nil {
		return nil, err
	}
	_ = m.reg.Touch(ctx, key, m.now())
	return b, nil
}

// EditFile 在文件内做精确字符串替换（Claude Code 式 search-replace），返回替换次数。
// replaceAll=false 时：oldStr 出现 0 次报 "not found"、出现多次要求改用 replace_all，确保改动唯一。
// replaceAll=true 时：替换全部出现。
func (m *Manager) EditFile(ctx context.Context, key, path, oldStr, newStr string, replaceAll bool) (int, error) {
	if path == "" {
		return 0, fmt.Errorf("path must not be empty")
	}
	if oldStr == "" {
		return 0, fmt.Errorf("old_string must not be empty")
	}
	if oldStr == newStr {
		return 0, fmt.Errorf("old_string and new_string are identical, nothing to do")
	}
	data, err := m.ReadFile(ctx, key, path)
	if err != nil {
		return 0, err
	}
	content := string(data)
	n := strings.Count(content, oldStr)
	if n == 0 {
		return 0, fmt.Errorf("old_string not found in %s", path)
	}
	if n > 1 && !replaceAll {
		return 0, fmt.Errorf("old_string found %d times in %s; pass replace_all=true or include more surrounding context to make it unique", n, path)
	}
	var out string
	if replaceAll {
		out = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		out = strings.Replace(content, oldStr, newStr, 1)
		n = 1
	}
	if err := m.WriteFile(ctx, key, path, []byte(out)); err != nil {
		return 0, err
	}
	return n, nil
}

// Grep 在沙箱里按正则搜索文件内容（优先 ripgrep，回退 grep -rn）。path 为空时搜 /workspace。
func (m *Manager) Grep(ctx context.Context, key, pattern, path string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("pattern must not be empty")
	}
	if path == "" {
		path = "."
	}
	q := shSingleQuote(pattern)
	p := shSingleQuote(path)
	cmd := fmt.Sprintf(`if command -v rg >/dev/null 2>&1; then rg -n --no-heading -- %s %s; else grep -rn -- %s %s; fi`, q, p, q, p)
	res, err := m.Exec(ctx, key, cmd, 0)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Stdout) == "" {
		return "(no matches)", nil
	}
	return res.Stdout, nil
}

// Glob 在沙箱里按文件名模式查找文件（如 "*.go"）。pattern 是基于文件名的 glob。
func (m *Manager) Glob(ctx context.Context, key, pattern string) (string, error) {
	if pattern == "" {
		return "", fmt.Errorf("pattern must not be empty")
	}
	cmd := fmt.Sprintf(`find . -type f -name %s 2>/dev/null | head -200`, shSingleQuote(pattern))
	res, err := m.Exec(ctx, key, cmd, 0)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Stdout) == "" {
		return "(no files matched)", nil
	}
	return res.Stdout, nil
}

// shSingleQuote 把字符串安全地包成单引号 shell 参数。
func shSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ListFiles 列目录（先 ensure）。
func (m *Manager) ListFiles(ctx context.Context, key, path string) ([]string, error) {
	if err := m.EnsureSandbox(ctx, key); err != nil {
		return nil, err
	}
	files, err := m.runner.ListFiles(ctx, &sandbox.ListFilesRequest{SandboxID: key, Path: path})
	if err != nil {
		return nil, err
	}
	_ = m.reg.Touch(ctx, key, m.now())
	return files, nil
}

const workspaceArchivePath = "/tmp/ynet-workspace.tgz"

// Checkpoint 把 /workspace 打包回 MinIO。store 为 nil 时为 no-op。
func (m *Manager) Checkpoint(ctx context.Context, key string) error {
	if m.store == nil {
		return nil
	}
	res, err := m.runner.Exec(ctx, &sandbox.ExecRequest{
		SandboxID: key,
		Cmd:       "tar czf " + workspaceArchivePath + " -C /workspace . 2>/dev/null || true",
	})
	if err != nil {
		return fmt.Errorf("tar workspace: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("tar workspace exit %d: %s", res.ExitCode, res.Stderr)
	}
	data, err := m.runner.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: key, Path: workspaceArchivePath})
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}
	if err := m.store.PutObject(ctx, m.workspaceKey(key), data); err != nil {
		return fmt.Errorf("put workspace: %w", err)
	}
	return nil
}

// restore 冷启动时从 MinIO 还原 workspace。store 为 nil 或无归档时为 no-op。
func (m *Manager) restore(ctx context.Context, key string) error {
	if m.store == nil {
		return nil
	}
	data, err := m.store.GetObject(ctx, m.workspaceKey(key))
	if err != nil || len(data) == 0 {
		// 首次没有归档，正常。
		return nil
	}
	if err := m.runner.WriteFile(ctx, &sandbox.WriteFileRequest{SandboxID: key, Path: workspaceArchivePath, Content: data}); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}
	res, err := m.runner.Exec(ctx, &sandbox.ExecRequest{
		SandboxID: key,
		Cmd:       "mkdir -p /workspace && tar xzf " + workspaceArchivePath + " -C /workspace",
	})
	if err != nil {
		return fmt.Errorf("untar workspace: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("untar workspace exit %d: %s", res.ExitCode, res.Stderr)
	}
	return nil
}

// SyncSkill 把技能脚本注入沙箱 /skills/<name>/，按内容 hash 去重（同版本跳过）。
func (m *Manager) SyncSkill(ctx context.Context, key, name string, files map[string][]byte) error {
	if err := m.EnsureSandbox(ctx, key); err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	base := "/skills/" + name
	hash := hashFiles(files)
	hashPath := base + "/.skillhash"
	// dedup：已是同版本则跳过。
	if cur, err := m.runner.ReadFile(ctx, &sandbox.ReadFileRequest{SandboxID: key, Path: hashPath}); err == nil {
		if string(cur) == hash {
			return nil
		}
	}
	for rel, content := range files {
		p := base + "/" + rel
		if err := m.runner.WriteFile(ctx, &sandbox.WriteFileRequest{SandboxID: key, Path: p, Content: content}); err != nil {
			return fmt.Errorf("inject skill file %s: %w", rel, err)
		}
	}
	if err := m.runner.WriteFile(ctx, &sandbox.WriteFileRequest{SandboxID: key, Path: hashPath, Content: []byte(hash)}); err != nil {
		return fmt.Errorf("write skill hash: %w", err)
	}
	_ = m.reg.Touch(ctx, key, m.now())
	return nil
}

func hashFiles(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		h.Write([]byte(n))
		h.Write([]byte{0})
		h.Write(files[n])
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Pause 挂起沙箱并更新注册表。
func (m *Manager) Pause(ctx context.Context, key string) error {
	if err := m.runner.Pause(ctx, key); err != nil {
		return err
	}
	return m.reg.SetState(ctx, key, sandbox.StatePaused)
}

// Destroy 回收沙箱：先 Checkpoint，再 kill，再删注册表。
func (m *Manager) Destroy(ctx context.Context, key string) error {
	_ = m.Checkpoint(ctx, key)
	if err := m.runner.Kill(ctx, key); err != nil {
		return err
	}
	return m.reg.Delete(ctx, key)
}
