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

package crossuser

import (
	"context"

	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	"github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/user/service"
)

var defaultSVC crossuser.User

type impl struct {
	DomainSVC service.User
}

func InitDomainService(u service.User) crossuser.User {
	defaultSVC = &impl{
		DomainSVC: u,
	}
	return defaultSVC
}

func (u *impl) GetUserSpaceList(ctx context.Context, userID int64) (spaces []*entity.Space, err error) {
	return u.DomainSVC.GetUserSpaceList(ctx, userID)
}

// CheckSpacePermission checks user's permission in a space
func (u *impl) CheckSpacePermission(ctx context.Context, spaceID, userID int64) (*crossuser.SpacePermission, error) {
	isMember, roleType, canInvite, canManage, err := u.DomainSVC.CheckMemberPermission(ctx, spaceID, userID)
	if err != nil {
		return nil, err
	}

	role := entity.RoleType(roleType)
	return &crossuser.SpacePermission{
		IsMember:  isMember,
		RoleType:  roleType,
		CanInvite: canInvite,
		CanManage: canManage,
		CanEdit:   isMember && role.CanEdit(),
	}, nil
}
