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
	configmodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	spaceApp "github.com/ynet-dev/ynet-studio/backend/application/space"
)

// configureModelsInvoker is the minimal contract the handler needs from the
// application-layer ConfigureModelsSVC. Pulled out as an interface so tests
// can swap in a fake without booting the full application.Init wiring.
type configureModelsInvoker interface {
	ConfigureModels(ctx context.Context, req *configmodel.ConfigureModelsRequest) (*configmodel.ConfigureModelsResponse, error)
}

// configureModelsSvcGetter returns the active invoker. Production reads the
// global spaceApp.ConfigureModelsSVC at call-time; tests override this.
var configureModelsSvcGetter = func() configureModelsInvoker {
	if spaceApp.ConfigureModelsSVC == nil {
		return nil
	}
	return spaceApp.ConfigureModelsSVC
}

// ConfigureModels one-shot rewrites chat / embedder / rerank configs for a
// space and clears the per-space model cache. Only the space owner may call.
// @router /api/space/configure_models [POST]
func ConfigureModels(ctx context.Context, c *app.RequestContext) {
	var req configmodel.ConfigureModelsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	svc := configureModelsSvcGetter()
	if svc == nil {
		c.String(consts.StatusInternalServerError, "configure_models service not initialized")
		return
	}

	resp, err := svc.ConfigureModels(ctx, &req)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(consts.StatusOK, resp)
}
