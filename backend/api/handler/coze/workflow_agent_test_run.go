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

package coze

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	appworkflow "github.com/ynet-dev/ynet-studio/backend/application/workflow"
)

// WorkFlowAgentTestRun waits for a workflow test run to finish and returns a compact result for agents.
// @router /api/workflow_api/agent_test_run [POST]
func WorkFlowAgentTestRun(ctx context.Context, c *app.RequestContext) {
	var req appworkflow.AgentWorkflowTestRunRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if strings.TrimSpace(req.WorkflowID) == "" {
		invalidParamRequestResponse(c, "workflow_id is required")
		return
	}
	if strings.TrimSpace(req.SpaceID) == "" {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}

	resp, err := appworkflow.SVC.AgentTestRun(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}

	c.JSON(consts.StatusOK, resp)
}
