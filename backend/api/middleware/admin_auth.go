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

package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	adminapp "github.com/ynet-dev/ynet-studio/backend/application/admin"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
)

// AdminAuthMiddleware 管理员权限验证中间件
func AdminAuthMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userIDPtr := ctxutil.GetUIDFromCtx(ctx)
		if userIDPtr == nil || *userIDPtr == 0 {
			c.JSON(consts.StatusUnauthorized, utils.H{"code": 401, "msg": "请先登录"})
			c.Abort()
			return
		}

		isAdmin, err := adminapp.AdminApplicationSVC.IsAdmin(ctx, uint64(*userIDPtr))
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "msg": "权限验证失败"})
			c.Abort()
			return
		}

		if !isAdmin {
			c.JSON(consts.StatusForbidden, utils.H{"code": 403, "msg": "无管理员权限"})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// SuperAdminAuthMiddleware 超级管理员权限验证中间件
func SuperAdminAuthMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userIDPtr := ctxutil.GetUIDFromCtx(ctx)
		if userIDPtr == nil || *userIDPtr == 0 {
			c.JSON(consts.StatusUnauthorized, utils.H{"code": 401, "msg": "请先登录"})
			c.Abort()
			return
		}

		isSuperAdmin, err := adminapp.AdminApplicationSVC.IsSuperAdmin(ctx, uint64(*userIDPtr))
		if err != nil {
			c.JSON(consts.StatusInternalServerError, utils.H{"code": 500, "msg": "权限验证失败"})
			c.Abort()
			return
		}

		if !isSuperAdmin {
			c.JSON(consts.StatusForbidden, utils.H{"code": 403, "msg": "需要超级管理员权限"})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}
