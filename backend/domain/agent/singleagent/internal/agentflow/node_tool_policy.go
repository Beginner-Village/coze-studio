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
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
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

// argParseErrMsg 把「工具参数解析失败」转成可恢复的提示文本(以 nil error 返回)。
// 模型生成超大工具调用(如把整个文件内容塞进参数)超 max_tokens 时,流式 JSON 会被
// 截断成残缺串;若直接返回 error 会让整轮 run 崩溃。改为把错误反馈给模型,让它重试
// (分块写),run 不中断。
func argParseErrMsg(err error) string {
	return fmt.Sprintf("Error: could not parse tool arguments (%v). The arguments were likely truncated because the payload is too large for a single tool call. Retry with a smaller payload — for large files, write them in several smaller steps (e.g. multiple `run_bash` calls appending chunks with `cat >> file`, or write_file in parts) instead of one huge call.", err)
}

// ---- context 压缩：DeepAgent 式结构化截断 + 落盘引用 ----

const toolOutputsDir = "/workspace/.agent/tooloutputs"

type toolOutputConversationIDKey struct{}

type toolOutputOffloadRecord struct {
	Version        string                  `json:"version"`
	ConversationID string                  `json:"conversation_id,omitempty"`
	ToolCallID     string                  `json:"tool_call_id,omitempty"`
	Tool           string                  `json:"tool"`
	Status         string                  `json:"status"`
	Arguments      any                     `json:"arguments,omitempty"`
	Summary        string                  `json:"summary"`
	Result         toolOutputOffloadResult `json:"result"`
}

type toolOutputOffloadResult struct {
	Content   string `json:"content"`
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated"`
}

type toolOutputOffloadMeta struct {
	ToolCallID string
	Tool       string
	Status     string
	Arguments  any
}

func withToolOutputConversationID(ctx context.Context, conversationID int64) context.Context {
	if conversationID <= 0 {
		return ctx
	}
	return context.WithValue(ctx, toolOutputConversationIDKey{}, conversationID)
}

func toolOutputConversationID(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if conversationID, ok := ctx.Value(toolOutputConversationIDKey{}).(int64); ok && conversationID > 0 {
		return conversationID
	}
	return 0
}

// offloadOrTruncate 处理超长工具输出：写进沙箱文件，只回灌「头尾摘要 + 文件引用」，不调额外 LLM。
// 写盘失败时退回纯截断。
func offloadOrTruncate(ctx context.Context, svc crosssandbox.Manager, key, s string) string {
	return offloadToolResultOrTruncate(ctx, svc, key, toolOutputOffloadMeta{Tool: "tool_output"}, s)
}

func offloadToolResultOrTruncate(ctx context.Context, svc crosssandbox.Manager, key string, meta toolOutputOffloadMeta, s string) string {
	max := maxToolOutputBytes()
	if len(s) <= max {
		// 工具输出(尤其网页内容)可能含非法 UTF-8;入库前必须清洗,否则 message 表 INSERT
		// 会报 MySQL 1366 Incorrect string value 导致整轮失败。
		return strings.ToValidUTF8(s, "")
	}
	sum := sha256.Sum256([]byte(s))
	conversationID := toolOutputConversationID(ctx)
	path := toolOutputPath(conversationID, hex.EncodeToString(sum[:])[:16])
	content := strings.ToValidUTF8(s, "")
	toolName := strings.TrimSpace(meta.Tool)
	if toolName == "" {
		toolName = "tool_output"
	}
	status := strings.TrimSpace(meta.Status)
	if status == "" {
		status = "completed"
	}
	record := toolOutputOffloadRecord{
		Version:        "v1",
		ConversationID: formatToolOutputConversationID(conversationID),
		ToolCallID:     strings.TrimSpace(meta.ToolCallID),
		Tool:           toolName,
		Status:         status,
		Arguments:      meta.Arguments,
		Summary:        fmt.Sprintf("%s output offloaded: %d bytes", toolName, len(s)),
		Result: toolOutputOffloadResult{
			Content:   content,
			Bytes:     len(s),
			Truncated: true,
		},
	}
	raw, err := json.MarshalIndent(record, "", "  ")
	if err != nil || svc == nil || svc.WriteFile(ctx, key, path, raw) != nil {
		return truncateForModel(s)
	}
	// 按字节切会把多字节字符(中文/emoji)切成两半 → 非法 UTF-8,ToValidUTF8 去掉碎片。
	head := strings.ToValidUTF8(s[:max*7/10], "")
	tail := strings.ToValidUTF8(s[len(s)-max*3/10:], "")
	return head + fmt.Sprintf("\n\n...[output truncated; full %d bytes saved to %s — use read_file/grep on that path to view more]...\n\n", len(s), path) + tail
}

func toolOutputPath(conversationID int64, digest string) string {
	if conversationID > 0 {
		return fmt.Sprintf("%s/sessions/%d/%s.json", toolOutputsDir, conversationID, digest)
	}
	return toolOutputsDir + "/" + digest + ".json"
}

func formatToolOutputConversationID(conversationID int64) string {
	if conversationID <= 0 {
		return ""
	}
	return strconv.FormatInt(conversationID, 10)
}
