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

	"github.com/ynet-dev/ynet-studio/backend/api/internal/httputil"
	resyncmodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	spaceApp "github.com/ynet-dev/ynet-studio/backend/application/space"
)

// resyncESInvoker is the minimal contract the handler needs from the
// application-layer ResyncSVC. Pulled out as an interface so tests can swap
// in a fake without booting the full application.Init wiring.
type resyncESInvoker interface {
	ResyncES(ctx context.Context, req *resyncmodel.ResyncESRequest) (*resyncmodel.ResyncESResponse, error)
	ResyncAllES(ctx context.Context, req *resyncmodel.ResyncAllESRequest) (*resyncmodel.ResyncAllESResponse, error)
}

// resyncSvcGetter returns the active invoker. Production reads the global
// spaceApp.ResyncSVC at call-time; tests override this to inject a fake.
var resyncSvcGetter = func() resyncESInvoker {
	if spaceApp.ResyncSVC == nil {
		return nil
	}
	return spaceApp.ResyncSVC
}

// ResyncES drops all ES docs for the given space and replays writes from MySQL.
// Only the space owner can trigger this — non-owners get ErrSpacePermissionCode.
// @router /api/space/resync_es [POST]
func ResyncES(ctx context.Context, c *app.RequestContext) {
	var req resyncmodel.ResyncESRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	svc := resyncSvcGetter()
	if svc == nil {
		c.String(consts.StatusInternalServerError, "resync service not initialized")
		return
	}

	resp, err := svc.ResyncES(ctx, &req)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(consts.StatusOK, resp)
}

// ResyncAllES purges the three list indices and rebuilds every space in the
// request body, bypassing the per-space owner gate. Admin/maintenance use
// after a DB-level data sync. Requires a logged-in caller.
// @router /api/space/resync_all_es [POST]
func ResyncAllES(ctx context.Context, c *app.RequestContext) {
	var req resyncmodel.ResyncAllESRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	svc := resyncSvcGetter()
	if svc == nil {
		c.String(consts.StatusInternalServerError, "resync service not initialized")
		return
	}

	resp, err := svc.ResyncAllES(ctx, &req)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(consts.StatusOK, resp)
}
