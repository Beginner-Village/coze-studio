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
	// 脱敏回显：MCP env 里的凭证(Authorization/API key 等)以哨兵替换，绝不明文出后端。
	return draft.SuperAgentToolConfig.RedactedForRead(), nil
}

// UpdateSuperAgentToolConfig 持久化某个超级体的能力开关配置（先取完整草稿，仅替换该字段后回写）。
// callerUserID 必须是该 agent 的创建者，否则拒绝（防 IDOR 越权写入）。
func (s *SingleAgentApplicationService) UpdateSuperAgentToolConfig(ctx context.Context, agentID, callerUserID int64, cfg *crossagent.SuperAgentToolConfig) error {
	draft, err := s.loadOwnedSuperAgentDraft(ctx, agentID, callerUserID)
	if err != nil {
		return err
	}
	// 回填脱敏密钥：客户端把上次读到的(含哨兵 env)整份配置存回时，用已存密钥还原哨兵值，
	// 避免把真实凭证冲成哨兵字面量。draft 此刻仍持有旧配置，正好作为回填来源。
	crossagent.MergeMCPServerSecrets(cfg, draft.SuperAgentToolConfig)
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
	if draft == nil {
		return nil, fmt.Errorf("agent %d not found or access denied", agentID)
	}
	if draft.CreatorID != callerUserID {
		// Allow space members (not only the creator) — space-scoped
		// collaboration. Non-members get the same opaque error, so agent
		// existence is not leaked to unrelated users.
		isMember, _, _, _, mErr := s.appContext.UserDomainSVC.CheckMemberPermission(ctx, draft.SpaceID, callerUserID)
		if mErr != nil {
			return nil, mErr
		}
		if !isMember {
			return nil, fmt.Errorf("agent %d not found or access denied", agentID)
		}
	}
	return draft, nil
}
