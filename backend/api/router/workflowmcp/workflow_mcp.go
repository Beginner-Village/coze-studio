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

package workflowmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	mcpserver "github.com/mark3labs/mcp-go/server"

	appworkflow "github.com/ynet-dev/ynet-studio/backend/application/workflow"
	"github.com/ynet-dev/ynet-studio/backend/application/workflow/canvasautomation"
)

// Register exposes the system-provided workflow canvas MCP endpoint from the
// main backend service, so internal agents and external MCP clients use the
// same catalog/spec/test-run contracts.
func Register(r *server.Hertz) {
	RegisterWithRelay(r, canvasautomation.SharedWorkflowCommandRelay())
}

func RegisterWithRelay(r *server.Hertz, relay *canvasautomation.MemoryWorkflowCommandRelay) {
	if relay == nil {
		relay = canvasautomation.NewMemoryWorkflowCommandRelay()
	}
	mcpHandler := mcpserver.NewStreamableHTTPServer(
		canvasautomation.NewWorkflowMCPServer(canvasautomation.WorkflowMCPOptions{
			TestRunner:          applicationWorkflowTestRunner{},
			CommandDispatcher:   relay,
			CommandResultWaiter: relay,
		}),
		mcpserver.WithEndpointPath("/api/workflow_mcp/mcp"),
		mcpserver.WithDisableStreaming(true),
	)

	handler := bufferedHTTPHandler(mcpHandler)
	r.POST("/api/workflow_mcp/mcp", handler)
	r.GET("/api/workflow_mcp/mcp", handler)
	r.DELETE("/api/workflow_mcp/mcp", handler)
	r.GET("/api/workflow_mcp/browser_commands", browserCommandPollHandler(relay))
	r.POST("/api/workflow_mcp/browser_command_results", browserCommandResultHandler(relay))
}

type applicationWorkflowTestRunner struct{}

func (applicationWorkflowTestRunner) RunWorkflowTest(ctx context.Context, req canvasautomation.WorkflowTestRunRequest) (any, error) {
	return appworkflow.SVC.AgentTestRun(ctx, &appworkflow.AgentWorkflowTestRunRequest{
		WorkflowID: req.WorkflowID,
		SpaceID:    req.SpaceID,
		Input:      req.Input,
		BotID:      req.BotID,
		ProjectID:  req.ProjectID,
		CommitID:   req.CommitID,
		TimeoutMs:  req.TimeoutMs,
		IntervalMs: req.IntervalMs,
	})
}

func bufferedHTTPHandler(handler http.Handler) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		req, err := http.NewRequestWithContext(
			ctx,
			string(c.Method()),
			"http://workflow-mcp.local"+string(c.Request.URI().RequestURI()),
			bytes.NewReader(c.Request.Body()),
		)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		c.Request.Header.VisitAll(func(key, value []byte) {
			req.Header.Add(string(key), string(value))
		})

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		for key, values := range recorder.Header() {
			for _, value := range values {
				c.Response.Header.Add(key, value)
			}
		}
		c.SetStatusCode(recorder.Code)
		c.Response.SetBody(recorder.Body.Bytes())
	}
}

func browserCommandPollHandler(relay *canvasautomation.MemoryWorkflowCommandRelay) app.HandlerFunc {
	return func(_ context.Context, c *app.RequestContext) {
		workflowID := strings.TrimSpace(c.Query("workflow_id"))
		spaceID := strings.TrimSpace(c.Query("space_id"))
		if workflowID == "" || spaceID == "" {
			c.JSON(http.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  "workflow_id and space_id are required",
			})
			return
		}
		afterID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("after_id")), 10, 64)
		limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
		items := relay.PollWorkflowCommands(spaceID, workflowID, afterID, limit)
		commands := make([]canvasautomation.WorkflowCanvasCommand, 0, len(items))
		requests := make([]map[string]any, 0, len(items))
		cursor := afterID
		if len(items) > 0 && afterID > items[len(items)-1].ID {
			cursor = 0
		}
		for _, item := range items {
			commands = append(commands, item.Command)
			requests = append(requests, map[string]any{
				"id":                item.ID,
				"request_id":        item.RequestID,
				"requires_response": item.RequiresResponse,
				"command":           item.Command,
			})
			if item.ID > cursor {
				cursor = item.ID
			}
		}
		c.JSON(http.StatusOK, map[string]any{
			"status":    "ok",
			"protocol":  "canvas_automation.v0",
			"mode":      "browser_live",
			"space_id":  spaceID,
			"canvas_id": workflowID,
			"cursor":    cursor,
			"commands":  commands,
			"requests":  requests,
		})
	}
}

func browserCommandResultHandler(relay *canvasautomation.MemoryWorkflowCommandRelay) app.HandlerFunc {
	type browserCommandResultPayload struct {
		WorkflowID string                                       `json:"workflow_id"`
		SpaceID    string                                       `json:"space_id"`
		Result     canvasautomation.WorkflowCanvasCommandResult `json:"result"`
	}

	return func(ctx context.Context, c *app.RequestContext) {
		var payload browserCommandResultPayload
		if err := json.Unmarshal(c.Request.Body(), &payload); err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  "invalid JSON body",
			})
			return
		}
		if strings.TrimSpace(payload.WorkflowID) == "" || strings.TrimSpace(payload.SpaceID) == "" {
			c.JSON(http.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  "workflow_id and space_id are required",
			})
			return
		}
		if err := relay.SubmitWorkflowCommandResult(ctx, payload.Result); err != nil {
			c.JSON(http.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	}
}
