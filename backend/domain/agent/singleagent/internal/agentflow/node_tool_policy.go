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

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

// ---- 两轴沙箱安全模型（借鉴 Codex）----
//
// sandbox mode（物理允许什么）由环境变量 SANDBOX_MODE 控制：
//   - read-only      ：只读，禁止任何写/删/改（含写类 bash、write_file、edit_file）。用于 plan/审查。
//   - workspace-write ：默认。允许在沙箱工作区内读写执行（容器即隔离边界）。
//   - full            ：放开全部。
//
// 当一个变更类工具/命令在 read-only 模式下被拒，返回里会告诉模型「需要更高权限/审批」，
// 即 approval 轴的体现（由上层/用户决定是否提升 SANDBOX_MODE）。

const (
	modeReadOnly       = "read-only"
	modeWorkspaceWrite = "workspace-write"
	modeFull           = "full"
)

// sandboxMode 返回当前沙箱模式（默认 workspace-write）。
func sandboxMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SANDBOX_MODE"))) {
	case modeReadOnly:
		return modeReadOnly
	case modeFull:
		return modeFull
	default:
		return modeWorkspaceWrite
	}
}

// mutatingCmdRe 粗粒度识别会写/删/改的 shell 命令（best-effort）。
var mutatingCmdRe = regexp.MustCompile(`(^|\s|;|&|\|)(rm|mv|cp|mkdir|rmdir|touch|tee|dd|truncate|chmod|chown|ln|install|sed\s+-i|patch)\b|>>|[^>]>`)

// isMutatingCommand 判断一条 bash 命令是否可能产生写/删除副作用。
func isMutatingCommand(cmd string) bool {
	return mutatingCmdRe.MatchString(cmd)
}

// checkMutationAllowed 在 read-only 模式下拒绝变更类操作，返回 (允许, 拒绝原因)。
func checkMutationAllowed(action string) (bool, string) {
	if sandboxMode() == modeReadOnly {
		return false, fmt.Sprintf("Blocked by sandbox policy: mode is read-only, so %s is not permitted. Use read-only tools (read_file/list_files/grep/glob) to investigate, or ask the user to raise SANDBOX_MODE to workspace-write.", action)
	}
	return true, ""
}

// ---- context 压缩：DeepAgent 式结构化截断 + 落盘引用 ----

const toolOutputsDir = "/workspace/.agent/tooloutputs"

// offloadOrTruncate 处理超长工具输出：写进沙箱文件，只回灌「头尾摘要 + 文件引用」，不调额外 LLM。
// 写盘失败时退回纯截断。
func offloadOrTruncate(ctx context.Context, svc crosssandbox.Manager, key, s string) string {
	max := maxToolOutputBytes()
	if len(s) <= max {
		return s
	}
	sum := sha256.Sum256([]byte(s))
	path := toolOutputsDir + "/" + hex.EncodeToString(sum[:])[:16] + ".txt"
	if svc == nil || svc.WriteFile(ctx, key, path, []byte(s)) != nil {
		return truncateForModel(s)
	}
	head := s[:max*7/10]
	tail := s[len(s)-max*3/10:]
	return head + fmt.Sprintf("\n\n...[output truncated; full %d bytes saved to %s — use read_file/grep on that path to view more]...\n\n", len(s), path) + tail
}
