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

	sbx "github.com/ynet-dev/ynet-studio/backend/infra/contract/sandbox"
)

// Manager 是给 agent 运行时用的跨域沙箱接口。
// domain/sandbox.Manager 直接满足它。
type Manager interface {
	Exec(ctx context.Context, key, cmd string, timeoutSec int) (*sbx.ExecResponse, error)
	ReadFile(ctx context.Context, key, path string) ([]byte, error)
	WriteFile(ctx context.Context, key, path string, content []byte) error
	ListFiles(ctx context.Context, key, path string) ([]string, error)
	// EditFile 精确字符串替换（Claude Code 式 search-replace），返回替换次数。
	EditFile(ctx context.Context, key, path, oldStr, newStr string, replaceAll bool) (int, error)
	// Grep 按正则搜索文件内容（优先 ripgrep，回退 grep）。
	Grep(ctx context.Context, key, pattern, path string) (string, error)
	// Glob 按文件名模式查找文件（如 "*.go"）。
	Glob(ctx context.Context, key, pattern string) (string, error)
	// SyncSkill 把技能的脚本文件注入沙箱 /skills/<name>/，按内容 hash 去重。
	SyncSkill(ctx context.Context, key, name string, files map[string][]byte) error
}

var defaultSVC Manager

// DefaultSVC 返回全局沙箱管理器（未初始化时为 nil）。
func DefaultSVC() Manager { return defaultSVC }

// SetDefaultSVC 注入全局沙箱管理器。
func SetDefaultSVC(m Manager) { defaultSVC = m }
