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

package singleagent

import (
	"context"
	"fmt"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
)

// GetSuperAgentToolConfig 返回某个超级体的能力开关配置。config 为 nil 表示「默认全开」。
// callerUserID 为登录态调用者，必须是该 agent 的创建者，否则拒绝（防 IDOR 越权读取）。
func (s *SingleAgentApplicationService) GetSuperAgentToolConfig(ctx context.Context, agentID, callerUserID int64) (*crossagent.SuperAgentToolConfig, error) {
	draft, err := s.loadOwnedSuperAgentDraft(ctx, agentID, callerUserID)
	if err != nil {
		return nil, err
	}
	return draft.SuperAgentToolConfig, nil
}

// UpdateSuperAgentToolConfig 持久化某个超级体的能力开关配置（先取完整草稿，仅替换该字段后回写）。
// callerUserID 必须是该 agent 的创建者，否则拒绝（防 IDOR 越权写入）。
func (s *SingleAgentApplicationService) UpdateSuperAgentToolConfig(ctx context.Context, agentID, callerUserID int64, cfg *crossagent.SuperAgentToolConfig) error {
	draft, err := s.loadOwnedSuperAgentDraft(ctx, agentID, callerUserID)
	if err != nil {
		return err
	}
	draft.SuperAgentToolConfig = cfg
	return s.DomainSVC.UpdateSingleAgentDraft(ctx, draft)
}

// loadOwnedSuperAgentDraft 读取草稿并校验调用者是创建者。不存在或非属主一律返回同一个
// 「not found or access denied」错误，避免泄露 agent 是否存在。
func (s *SingleAgentApplicationService) loadOwnedSuperAgentDraft(ctx context.Context, agentID, callerUserID int64) (*entity.SingleAgent, error) {
	if agentID <= 0 {
		return nil, fmt.Errorf("agent_id is required")
	}
	if callerUserID <= 0 {
		return nil, fmt.Errorf("unauthorized")
	}
	draft, err := s.DomainSVC.GetSingleAgentDraft(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if draft == nil || draft.CreatorID != callerUserID {
		return nil, fmt.Errorf("agent %d not found or access denied", agentID)
	}
	return draft, nil
}
