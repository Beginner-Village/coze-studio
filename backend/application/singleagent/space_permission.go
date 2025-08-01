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

package singleagent

import (
	"context"

	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	"github.com/coze-dev/coze-studio/backend/domain/agent/singleagent/entity"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/types/consts"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

// SpaceRole 定义空间角色类型
type SpaceRole int32

const (
	SpaceRoleOwner  SpaceRole = 1 // 所有者
	SpaceRoleAdmin  SpaceRole = 2 // 管理员  
	SpaceRoleMember SpaceRole = 3 // 成员
)

// AgentPermission 定义智能体权限类型
type AgentPermission struct {
	CanRead          bool
	CanEdit          bool
	CanDelete        bool
	CanDebug         bool
	CanPublish       bool
	CanReadChangelog bool
}

// GetAgentPermissionByRole 根据角色获取权限
func GetAgentPermissionByRole(role SpaceRole) *AgentPermission {
	switch role {
	case SpaceRoleOwner:
		return &AgentPermission{
			CanRead:          true,
			CanEdit:          true,
			CanDelete:        true,
			CanDebug:         true,
			CanPublish:       true,
			CanReadChangelog: true,
		}
	case SpaceRoleAdmin:
		return &AgentPermission{
			CanRead:          true,
			CanEdit:          true,
			CanDelete:        true,
			CanDebug:         true,
			CanPublish:       true,
			CanReadChangelog: true,
		}
	case SpaceRoleMember:
		return &AgentPermission{
			CanRead:          true,
			CanEdit:          false,
			CanDelete:        false,
			CanDebug:         true,
			CanPublish:       false,
			CanReadChangelog: true,
		}
	default:
		// 无权限
		return &AgentPermission{
			CanRead:          false,
			CanEdit:          false,
			CanDelete:        false,
			CanDebug:         false,
			CanPublish:       false,
			CanReadChangelog: false,
		}
	}
}

// checkSpacePermission 检查用户在空间中的权限
func (s *SingleAgentApplicationService) checkSpacePermission(ctx context.Context, userID int64, spaceID int64) (SpaceRole, error) {
	// 获取用户在空间中的角色
	spaces, err := s.appContext.UserDomainSVC.GetUserSpaceList(ctx, userID)
	if err != nil {
		return 0, errorx.Wrapf(err, "GetUserSpaceList failed, userID=%d", userID)
	}

	for _, space := range spaces {
		if space.ID == spaceID {
			return SpaceRole(space.RoleType), nil
		}
	}

	// 用户不在该空间中
	return 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "user not in space"))
}

// validateAgentPermission 验证智能体权限（支持不同的权限要求）
func (s *SingleAgentApplicationService) validateAgentPermission(ctx context.Context, agentID int64, requiredPermission string) (*entity.SingleAgent, SpaceRole, error) {
	uid := ctxutil.GetUIDFromCtx(ctx)
	if uid == nil {
		return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "session is required"))
	}

	agent, err := s.DomainSVC.GetSingleAgentDraft(ctx, agentID)
	if err != nil {
		return nil, 0, errorx.Wrapf(err, "GetSingleAgentDraft failed, agentID=%d", agentID)
	}

	if agent == nil {
		return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KVf("msg", "No agent draft(%d) found for the given agent ID", agentID))
	}

	// 模板空间特殊处理，不需要检查权限
	if agent.SpaceID == consts.TemplateSpaceID {
		return agent, SpaceRoleOwner, nil
	}

	// 1. 检查是否为智能体原始创建者（向后兼容）
	if agent.CreatorID == *uid {
		return agent, SpaceRoleOwner, nil // 原始创建者视为Owner
	}

	// 2. 检查用户在空间中的角色
	role, err := s.checkSpacePermission(ctx, *uid, agent.SpaceID)
	if err != nil {
		return nil, 0, err
	}

	// 3. 根据角色检查权限
	permission := GetAgentPermissionByRole(role)
	
	switch requiredPermission {
	case "read":
		if !permission.CanRead {
			return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "no read permission"))
		}
	case "edit":
		if !permission.CanEdit {
			return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "no edit permission"))
		}
	case "delete":
		if !permission.CanDelete {
			return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "no delete permission"))
		}
	case "debug":
		if !permission.CanDebug {
			return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "no debug permission"))
		}
	case "publish":
		if !permission.CanPublish {
			return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "no publish permission"))
		}
	default:
		// 默认需要读取权限
		if !permission.CanRead {
			return nil, 0, errorx.New(errno.ErrAgentPermissionCode, errorx.KV("msg", "no permission"))
		}
	}

	return agent, role, nil
}