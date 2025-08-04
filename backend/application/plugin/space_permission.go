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

package plugin

import (
	"context"

	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	"github.com/coze-dev/coze-studio/backend/domain/plugin/entity"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

// SpaceRole 定义空间角色类型
type SpaceRole int32

const (
	SpaceRoleOwner  SpaceRole = 1 // 所有者
	SpaceRoleAdmin  SpaceRole = 2 // 管理员  
	SpaceRoleMember SpaceRole = 3 // 成员
)

// PluginPermission 定义插件权限类型
type PluginPermission struct {
	CanRead          bool
	CanEdit          bool
	CanDelete        bool
	CanDebug         bool
	CanPublish       bool
	CanReadChangelog bool
}

// GetPluginPermissionByRole 根据角色获取权限
func GetPluginPermissionByRole(role SpaceRole) *PluginPermission {
	switch role {
	case SpaceRoleOwner:
		return &PluginPermission{
			CanRead:          true,
			CanEdit:          true,
			CanDelete:        true,
			CanDebug:         true,
			CanPublish:       true,
			CanReadChangelog: true,
		}
	case SpaceRoleAdmin:
		return &PluginPermission{
			CanRead:          true,
			CanEdit:          true,
			CanDelete:        true,
			CanDebug:         true,
			CanPublish:       true,
			CanReadChangelog: true,
		}
	case SpaceRoleMember:
		return &PluginPermission{
			CanRead:          true,
			CanEdit:          false,
			CanDelete:        false,
			CanDebug:         true,
			CanPublish:       false,
			CanReadChangelog: true,
		}
	default:
		// 无权限
		return &PluginPermission{
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
func (p *PluginApplicationService) checkSpacePermission(ctx context.Context, userID int64, spaceID int64) (SpaceRole, error) {
	// 获取用户在空间中的角色
	spaces, err := p.userSVC.GetUserSpaceList(ctx, userID)
	if err != nil {
		return 0, errorx.Wrapf(err, "GetUserSpaceList failed, userID=%d", userID)
	}

	for _, space := range spaces {
		if space.ID == spaceID {
			return SpaceRole(space.RoleType), nil
		}
	}

	// 用户不在该空间中
	return 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "user not in space"))
}

// validatePluginPermission 验证插件权限（支持不同的权限要求）
func (p *PluginApplicationService) validatePluginPermission(ctx context.Context, pluginID int64, requiredPermission string) (*entity.PluginInfo, SpaceRole, error) {
	uid := ctxutil.GetUIDFromCtx(ctx)
	if uid == nil {
		return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "session is required"))
	}

	// 先尝试获取草稿插件，如果失败则尝试获取已发布插件
	plugin, err := p.DomainSVC.GetDraftPlugin(ctx, pluginID)
	if err != nil {
		// 如果草稿插件不存在，尝试获取已发布的插件
		plugin, err = p.DomainSVC.GetOnlinePlugin(ctx, pluginID)
		if err != nil {
			return nil, 0, errorx.Wrapf(err, "Plugin not found (tried both draft and published), pluginID=%d", pluginID)
		}
	}

	// 1. 检查是否为插件原始创建者（向后兼容）
	if plugin.DeveloperID == *uid {
		return plugin, SpaceRoleOwner, nil // 原始创建者视为Owner
	}

	// 2. 检查用户在空间中的角色
	role, err := p.checkSpacePermission(ctx, *uid, plugin.SpaceID)
	if err != nil {
		return nil, 0, err
	}

	// 3. 根据角色检查权限
	permission := GetPluginPermissionByRole(role)
	
	switch requiredPermission {
	case "read":
		if !permission.CanRead {
			return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "no read permission"))
		}
	case "edit":
		if !permission.CanEdit {
			return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "no edit permission"))
		}
	case "delete":
		if !permission.CanDelete {
			return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "no delete permission"))
		}
	case "debug":
		if !permission.CanDebug {
			return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "no debug permission"))
		}
	case "publish":
		if !permission.CanPublish {
			return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "no publish permission"))
		}
	default:
		// 默认需要读取权限
		if !permission.CanRead {
			return nil, 0, errorx.New(errno.ErrPluginPermissionCode, errorx.KV(errno.PluginMsgKey, "no permission"))
		}
	}

	return plugin, role, nil
}