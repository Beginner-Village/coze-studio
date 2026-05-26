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

	diagmodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
	spaceApp "github.com/ynet-dev/ynet-studio/backend/application/space"
)

// diagnoseInvoker is the minimal contract the handler needs from the
// application-layer DiagnoseSVC. Pulled out as an interface so tests can
// swap in a fake without booting the full application.Init wiring.
type diagnoseInvoker interface {
	Diagnose(ctx context.Context, req *diagmodel.DiagnoseRequest) (*diagmodel.DiagnoseResponse, error)
}

// diagnoseSvcGetter returns the active invoker. Production reads the
// global spaceApp.DiagnoseSVC at call-time; tests override this.
var diagnoseSvcGetter = func() diagnoseInvoker {
	if spaceApp.DiagnoseSVC == nil {
		return nil
	}
	return spaceApp.DiagnoseSVC
}

// Diagnose runs a read-only health-check for the given space — chat /
// embedder / rerank live probes, MySQL counts, ES doc counts, Milvus
// collection existence. Only the space owner may call.
// @router /api/space/diagnose [POST]
func Diagnose(ctx context.Context, c *app.RequestContext) {
	var req diagmodel.DiagnoseRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	svc := diagnoseSvcGetter()
	if svc == nil {
		c.String(consts.StatusInternalServerError, "diagnose service not initialized")
		return
	}

	resp, err := svc.Diagnose(ctx, &req)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, resp)
}
