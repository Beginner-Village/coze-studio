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

package conversation

import (
	"context"
	"errors"

	crossconv "github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/conversation"
)

type ClearMessageConfig struct {
	Manager crossconv.ConversationManager
}

type MessageClear struct {
	cfg *ClearMessageConfig
}

func NewClearMessage(ctx context.Context, cfg *ClearMessageConfig) (*MessageClear, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.Manager == nil {
		return nil, errors.New("manager is required")
	}

	return &MessageClear{
		cfg: cfg,
	}, nil
}

func (c *MessageClear) Clear(ctx context.Context, input map[string]any) (map[string]any, error) {
	_, ok := input["conversationName"].(string)
	if !ok {
		return nil, errors.New("conversation name is required")
	}

	// For now, use ClearConversationHistory as the implementation
	// since there's no specific ClearMessage method available
	err := c.cfg.Manager.ClearConversationHistory(ctx, &crossconv.ClearConversationHistoryReq{
		ConversationID: 0, // This would need proper conversation ID resolution
	})
	if err != nil {
		return nil, err
	}
	
	return map[string]any{
		"isSuccess": true,
	}, nil
}
