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
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// 可以通过环境变量控制是否强制使用兼容模式
func shouldUseCompatibleChecker() bool {
	// 可以通过环境变量来控制
	if os.Getenv("FORCE_COMPATIBLE_TOOL_CHECKER") == "true" {
		return true
	}
	return false
}

// adaptiveToolCallChecker 自适应的工具调用检查器
// 关键改进：读取整个流直到 EOF，确保不会错过任何 tool_calls
// eino 框架要求 checker 必须关闭流，所以我们需要完整消费流
func adaptiveToolCallChecker(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()

	logs.CtxInfof(ctx, "[AdaptiveChecker] Starting adaptive tool call check (reading entire stream)")

	chunkCount := 0
	totalContent := strings.Builder{}
	hasToolCalls := false

	// 🔥 关键改进：读取整个流直到 EOF，不再限制 chunks 数量
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			logs.CtxInfof(ctx, "[AdaptiveChecker] Stream EOF after %d chunks", chunkCount)
			break
		}
		if err != nil {
			logs.CtxErrorf(ctx, "[AdaptiveChecker] Error reading stream at chunk %d: %v", chunkCount, err)
			return false, err
		}

		chunkCount++

		// 记录每个 chunk 的内容用于调试
		if len(msg.Content) > 0 {
			totalContent.WriteString(msg.Content)
			// 每 10 个 chunks 打印一次进度
			if chunkCount%10 == 0 {
				logs.CtxInfof(ctx, "[AdaptiveChecker] Progress: chunk %d, content length: %d", chunkCount, totalContent.Len())
			}
		}

		// 检查是否有工具调用
		if len(msg.ToolCalls) > 0 {
			logs.CtxInfof(ctx, "[AdaptiveChecker] ✅ Found tool calls in chunk %d!", chunkCount)
			for i, tc := range msg.ToolCalls {
				logs.CtxInfof(ctx, "[AdaptiveChecker] ToolCall[%d]: ID=%s, Name=%s", i, tc.ID, tc.Function.Name)
			}
			hasToolCalls = true
			// 继续读取完整个流，确保流被完全消费
			// 不能提前返回，因为流需要被完全消费
		}
	}

	// 流读取完成后，做最终判断
	finalContent := totalContent.String()
	logs.CtxInfof(ctx, "[AdaptiveChecker] Stream completed: %d chunks, content length: %d, hasToolCalls: %v",
		chunkCount, len(finalContent), hasToolCalls)

	if hasToolCalls {
		logs.CtxInfof(ctx, "[AdaptiveChecker] ✅ Returning true - tool calls detected")
		return true, nil
	}

	// 不打印内容预览（含模型输出原文），仅记录长度，避免对话内容落日志。
	logs.CtxInfof(ctx, "[AdaptiveChecker] ❌ No tool calls found. content len: %d", len(finalContent))

	return false, nil
}
