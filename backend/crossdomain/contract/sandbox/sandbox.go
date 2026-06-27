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

	sbx "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
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
	// CheckpointTo 把 key 沙箱的 /workspace 打包写入调用方指定的对象 key，返回归档内容 hash。
	// 用于模板构建：把构建沙箱固化成模板归档（区别于内部固定的 per-instance 检查点）。
	CheckpointTo(ctx context.Context, key, objectKey string) (contentHash string, err error)
	// RestoreFrom 从调用方指定的对象 key 还原归档到 key 沙箱的 /workspace。用于按模板冷启动。
	RestoreFrom(ctx context.Context, key, objectKey string) error
	// EnsureSandboxWithTemplate 确保沙箱处于 running；仅在冷启动时用 templateObjectKey 种子化
	// /workspace（实例检查点 > 模板 > 空白）。已运行的沙箱不做任何还原，防止覆写积累的工作区。
	EnsureSandboxWithTemplate(ctx context.Context, key, templateObjectKey string, readonlySkills bool) error
}

var defaultSVC Manager

// DefaultSVC 返回全局沙箱管理器（未初始化时为 nil）。
func DefaultSVC() Manager { return defaultSVC }

// SetDefaultSVC 注入全局沙箱管理器。
func SetDefaultSVC(m Manager) { defaultSVC = m }
