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

package agentflow

// node_super_agent_memory.go 实现 Hermes 式的 markdown 文件长期记忆(USER.md + MEMORY.md):
// 记忆不再进数据库,而是以两份 markdown 文件存在沙箱里,由超级体用文件工具
// (read_file/write_file/edit_file)自己维护;每次会话开场自动注入系统提示。
//   - /workspace/.agent/USER.md   —— 关于用户是谁的整体叙事画像(prose)
//   - /workspace/.agent/MEMORY.md  —— 自由格式的长期笔记/事实
// 只作用于超级体;普通单智能体不受影响。

import (
	"context"
	"strings"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

const (
	superAgentUserProfilePath = "/workspace/.agent/USER.md"
	superAgentMemoryNotesPath = "/workspace/.agent/MEMORY.md"
	// superAgentMemoryDocMaxBytes caps how much of each doc is injected into the
	// system prompt, so a runaway file never blows up the context.
	superAgentMemoryDocMaxBytes = 12000
)

// loadSuperAgentMemoryDocs reads the user's USER.md + MEMORY.md from the sandbox and
// renders them as a system-prompt injection block (Hermes "volatile" memory layer),
// or "" when neither exists. Best-effort: any read error is treated as "absent".
func loadSuperAgentMemoryDocs(ctx context.Context, sandboxKey string) string {
	svc := crosssandbox.DefaultSVC()
	if svc == nil || sandboxKey == "" {
		return ""
	}
	profile := readSandboxMemoryDoc(ctx, svc, sandboxKey, superAgentUserProfilePath)
	notes := readSandboxMemoryDoc(ctx, svc, sandboxKey, superAgentMemoryNotesPath)
	if profile == "" && notes == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("----- Long-term memory (auto-loaded from your memory files; keep it in mind, and update the files as you learn more) -----\n")
	if profile != "" {
		sb.WriteString("## USER.md — who this user is\n")
		sb.WriteString(profile)
		sb.WriteString("\n\n")
	}
	if notes != "" {
		sb.WriteString("## MEMORY.md — durable notes\n")
		sb.WriteString(notes)
		sb.WriteString("\n")
	}
	sb.WriteString("----- End of long-term memory -----")
	return strings.TrimRight(sb.String(), "\n")
}

func readSandboxMemoryDoc(ctx context.Context, svc crosssandbox.Manager, key, path string) string {
	b, err := svc.ReadFile(ctx, key, path)
	if err != nil || len(b) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if len(s) > superAgentMemoryDocMaxBytes {
		s = s[:superAgentMemoryDocMaxBytes] + "\n…(truncated)"
	}
	return s
}
