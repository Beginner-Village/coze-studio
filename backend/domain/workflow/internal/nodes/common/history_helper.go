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

package common

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/execute"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

// HistoryConfig 通用的历史记录配置
type HistoryConfig struct {
	EnableHistory      bool  // 是否启用历史记录
	HistoryRounds      int   // 历史记录轮数
	IncludeCurrentTurn bool  // 是否包含当前轮对话
}

// ConversationHistory 通用的历史记录处理器
type ConversationHistory struct {
	config *HistoryConfig
}

// NewConversationHistory 创建历史记录处理器
func NewConversationHistory(config *HistoryConfig) *ConversationHistory {
	return &ConversationHistory{
		config: config,
	}
}

// GetHistoryMessages 获取格式化的历史消息数组
func (h *ConversationHistory) GetHistoryMessages(ctx context.Context) ([]*schema.Message, error) {
	if h.config == nil || !h.config.EnableHistory || h.config.HistoryRounds <= 0 {
		logs.CtxInfof(ctx, "ConversationHistory: disabled or no rounds configured (enable=%v, rounds=%d)", 
			h.config != nil && h.config.EnableHistory, 
			func() int { if h.config == nil { return 0 } else { return h.config.HistoryRounds } }())
		return nil, nil
	}

	execCtx := execute.GetExeCtx(ctx)
	if execCtx == nil || execCtx.RootCtx.ConversationHistory == nil {
		logs.CtxInfof(ctx, "ConversationHistory: no execution context or conversation history available")
		return nil, nil
	}

	historyData, ok := execCtx.RootCtx.ConversationHistory["fullHistory"]
	if !ok {
		logs.CtxInfof(ctx, "ConversationHistory: no fullHistory found in conversation history")
		return nil, nil
	}

	// Try different types for the history data
	var historyList []interface{}
	
	if list, ok := historyData.([]interface{}); ok {
		historyList = list
	} else if mapList, ok := historyData.([]map[string]interface{}); ok {
		// Convert []map[string]interface{} to []interface{}
		historyList = make([]interface{}, len(mapList))
		for i, item := range mapList {
			historyList[i] = item
		}
	} else {
		logs.CtxInfof(ctx, "ConversationHistory: fullHistory is not a supported array type, type=%T", historyData)
		return nil, nil
	}

	logs.CtxInfof(ctx, "ConversationHistory: processing %d total messages, configured for %d rounds", 
		len(historyList), h.config.HistoryRounds)

	// Build messages array from the configured number of rounds
	var messages []*schema.Message
	rounds := 0
	
	// Process history in reverse order (most recent first) and limit by rounds
	// Collect messages first, then reverse to maintain chronological order
	var tempMessages []*schema.Message
	for i := len(historyList) - 1; i >= 0; i-- {
		msgData, ok := historyList[i].(map[string]interface{})
		if !ok {
			continue
		}

		role, ok := msgData["role"].(string)
		if !ok {
			continue
		}

		content, ok := msgData["content"].(string)
		if !ok {
			continue
		}

		// Check if we should stop before processing this message
		if role == "user" {
			if rounds >= h.config.HistoryRounds {
				break // Stop processing if we've reached the limit
			}
			rounds++
		}

		// Convert role to schema.RoleType
		var messageRole schema.RoleType
		switch role {
		case "user":
			messageRole = schema.User
		case "assistant":
			messageRole = schema.Assistant
		case "system":
			messageRole = schema.System
		default:
			continue // Skip unknown roles
		}

		// Add message to temp array
		tempMessages = append(tempMessages, &schema.Message{
			Role:    messageRole,
			Content: content,
		})
	}

	// Reverse the messages to maintain chronological order
	for i := len(tempMessages) - 1; i >= 0; i-- {
		messages = append(messages, tempMessages[i])
	}

	logs.CtxInfof(ctx, "ConversationHistory: prepared %d messages (%d rounds)", len(messages), rounds)
	for i, msg := range messages {
		logs.CtxInfof(ctx, "ConversationHistory[%d]: %s: %s", i, msg.Role, msg.Content)
	}

	return messages, nil
}

// GetHistoryText 获取文本格式的历史记录（适用于自定义prompt的节点）
func (h *ConversationHistory) GetHistoryText(ctx context.Context) (string, error) {
	messages, err := h.GetHistoryMessages(ctx)
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "", nil
	}

	var historyText string
	for _, msg := range messages {
		var roleText string
		switch msg.Role {
		case schema.User:
			roleText = "用户"
		case schema.Assistant:
			roleText = "助手"
		case schema.System:
			roleText = "系统"
		default:
			roleText = "未知"
		}
		historyText += roleText + ": " + msg.Content + "\n"
	}

	return historyText, nil
}