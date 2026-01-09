/*
 * Copyright 2025 coze-dev Authors
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
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

// 可以通过环境变量控制是否强制使用兼容模式
func shouldUseCompatibleChecker() bool {
	// 可以通过环境变量来控制
	if os.Getenv("FORCE_COMPATIBLE_TOOL_CHECKER") == "true" {
		return true
	}
	return false
}

// qwenCompatibleToolCallChecker 是一个兼容 Qwen/Claude 等模型的 tool call checker
// 这些模型可能会先输出文本内容，然后才输出工具调用
// 与默认的 firstChunkStreamToolCallChecker 不同，这个 checker 会读取更多的 chunks
// 来判断是否包含工具调用，而不是看到文本就立即返回 false
func qwenCompatibleToolCallChecker(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()

	const maxChunksToCheck = 10 // 最多检查前10个chunks
	chunksChecked := 0
	hasContent := false
	contentBuilder := strings.Builder{}

	logs.CtxInfof(ctx, "[QwenChecker] Starting to check stream for tool calls")

	for chunksChecked < maxChunksToCheck {
		msg, err := sr.Recv()
		if err == io.EOF {
			logs.CtxInfof(ctx, "[QwenChecker] Stream ended, chunks checked: %d, hasContent: %v", 
				chunksChecked, hasContent)
			break
		}
		if err != nil {
			logs.CtxErrorf(ctx, "[QwenChecker] Error reading stream: %v", err)
			return false, err
		}

		chunksChecked++

		// 如果发现工具调用，立即返回 true
		if len(msg.ToolCalls) > 0 {
			logs.CtxInfof(ctx, "[QwenChecker] Found tool calls in chunk %d: %+v", 
				chunksChecked, msg.ToolCalls)
			return true, nil
		}

		// 收集文本内容
		if len(msg.Content) > 0 {
			hasContent = true
			contentBuilder.WriteString(msg.Content)
			logs.CtxDebugf(ctx, "[QwenChecker] Chunk %d has content: %s", 
				chunksChecked, msg.Content)
		}

		// 如果已经有足够的内容，检查是否像是最终答案
		if hasContent && chunksChecked >= 3 {
			content := contentBuilder.String()
			
			// 如果内容包含明确的工具调用意图词，继续等待
			toolIntentKeywords := []string{
				"让我", "我将", "我来", "正在", "开始",
				"Let me", "I will", "I'll", "Starting", "Now",
				"查询", "获取", "搜索", "调用",
				"query", "fetch", "search", "calling",
			}
			
			hasToolIntent := false
			for _, keyword := range toolIntentKeywords {
				if strings.Contains(content, keyword) {
					hasToolIntent = true
					break
				}
			}
			
			if hasToolIntent {
				logs.CtxInfof(ctx, "[QwenChecker] Content suggests tool intent, continue checking. Content: %s", 
					content)
				continue // 继续检查更多chunks
			}
			
			// 如果内容看起来像是直接的答案（没有工具调用意图），可以提前返回
			answerKeywords := []string{
				"根据", "您的", "以下是", "信息如下", "结果是",
				"Based on", "Your", "Here is", "The result", "According to",
			}
			
			for _, keyword := range answerKeywords {
				if strings.Contains(content, keyword) {
					logs.CtxInfof(ctx, "[QwenChecker] Content appears to be final answer, no tools needed. Content: %s", 
						content)
					return false, nil
				}
			}
		}
	}

	// 检查完指定数量的chunks后，做最终判断
	if !hasContent {
		// 没有内容也没有工具调用，可能是空响应
		logs.CtxInfof(ctx, "[QwenChecker] No content or tool calls found")
		return false, nil
	}

	// 有内容但没有工具调用
	// 对于Qwen模型，如果前10个chunks都没有工具调用，那很可能就是没有
	logs.CtxInfof(ctx, "[QwenChecker] Checked %d chunks, found content but no tool calls. Content preview: %s", 
		chunksChecked, contentBuilder.String())
	
	return false, nil
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

	// 如果没有检测到 tool_calls，打印内容预览用于调试
	contentPreview := finalContent
	if len(contentPreview) > 500 {
		contentPreview = contentPreview[:500] + "..."
	}
	logs.CtxInfof(ctx, "[AdaptiveChecker] ❌ No tool calls found. Content preview: %s", contentPreview)

	return false, nil
}