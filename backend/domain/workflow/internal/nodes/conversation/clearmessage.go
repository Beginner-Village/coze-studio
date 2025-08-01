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
)

type ClearMessageConfig struct {
	// Manager can be nil for now since we don't have a proper implementation
	Manager interface{}
}

type MessageClear struct {
	cfg *ClearMessageConfig
}

func NewClearMessage(ctx context.Context, cfg *ClearMessageConfig) (*MessageClear, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
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

	// TODO: Implement actual message clearing logic when the proper interface is available
	// For now, just return success to allow compilation
	return map[string]any{
		"isSuccess": true,
	}, nil
}
