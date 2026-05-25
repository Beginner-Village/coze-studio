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

package space

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	resyncmodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	spaceApp "github.com/ynet-dev/ynet-studio/backend/application/space"
)

// ResyncES drops all ES docs for the given space and replays writes from MySQL.
// Only the space owner can trigger this — non-owners get ErrSpacePermissionCode.
// @router /api/space/resync_es [POST]
func ResyncES(ctx context.Context, c *app.RequestContext) {
	var req resyncmodel.ResyncESRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	resp, err := spaceApp.ResyncSVC.ResyncES(ctx, &req)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, resp)
}
