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
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/application/workflow/canvasautomation"
)

func TestRegisterExposesWorkflowCanvasMCPOnMainBackend(t *testing.T) {
	h := server.Default()
	Register(h)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"route-test","version":"1.0.0"}}}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/workflow_mcp/mcp",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, string(w.Body.Bytes()), "ynet-workflow-canvas")
}

func TestWorkflowCanvasMCPRejectsInvalidContentTypeInsteadOfFallingThroughTo404(t *testing.T) {
	h := server.Default()
	Register(h)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/workflow_mcp/mcp",
		nil,
	)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, string(w.Body.Bytes()), "Invalid content type")
}

func TestWorkflowCanvasMCPBrowserCommandPolling(t *testing.T) {
	h := server.Default()
	relay := canvasautomation.NewMemoryWorkflowCommandRelay()
	RegisterWithRelay(h, relay)

	require.NoError(t, relay.DispatchWorkflowCommand(context.Background(), canvasautomation.WorkflowCanvasDispatchCommand{
		WorkflowID: "wf-1",
		SpaceID:    "space-1",
		Command: canvasautomation.WorkflowCanvasCommand{
			Op:     "add_node",
			Target: "reply_text",
			Args:   map[string]any{"type": "15"},
		},
	}))

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/workflow_mcp/browser_commands?workflow_id=wf-1&space_id=space-1",
		nil,
	)

	require.Equal(t, http.StatusOK, w.Code)
	var payload struct {
		Status   string `json:"status"`
		Protocol string `json:"protocol"`
		CanvasID string `json:"canvas_id"`
		SpaceID  string `json:"space_id"`
		Cursor   int64  `json:"cursor"`
		Commands []struct {
			Op     string         `json:"op"`
			Target string         `json:"target"`
			Args   map[string]any `json:"args"`
		} `json:"commands"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "ok", payload.Status)
	assert.Equal(t, "canvas_automation.v0", payload.Protocol)
	assert.Equal(t, "wf-1", payload.CanvasID)
	assert.Equal(t, "space-1", payload.SpaceID)
	require.Len(t, payload.Commands, 1)
	assert.Equal(t, "add_node", payload.Commands[0].Op)
	assert.Equal(t, "reply_text", payload.Commands[0].Target)
	assert.Equal(t, "15", payload.Commands[0].Args["type"])
	assert.Greater(t, payload.Cursor, int64(0))
}

func TestWorkflowCanvasMCPBrowserCommandPollingIncludesRequestMetadata(t *testing.T) {
	h := server.Default()
	relay := canvasautomation.NewMemoryWorkflowCommandRelay()
	RegisterWithRelay(h, relay)

	require.NoError(t, relay.DispatchWorkflowCommand(context.Background(), canvasautomation.WorkflowCanvasDispatchCommand{
		WorkflowID:       "wf-1",
		SpaceID:          "space-1",
		RequestID:        "req-context",
		RequiresResponse: true,
		Command: canvasautomation.WorkflowCanvasCommand{
			Op: "get_canvas_context",
		},
	}))

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/workflow_mcp/browser_commands?workflow_id=wf-1&space_id=space-1",
		nil,
	)

	require.Equal(t, http.StatusOK, w.Code)
	var payload struct {
		Status   string `json:"status"`
		Requests []struct {
			ID               int64  `json:"id"`
			RequestID        string `json:"request_id"`
			RequiresResponse bool   `json:"requires_response"`
			Command          struct {
				Op string `json:"op"`
			} `json:"command"`
		} `json:"requests"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, "ok", payload.Status)
	require.Len(t, payload.Requests, 1)
	assert.Equal(t, "req-context", payload.Requests[0].RequestID)
	assert.True(t, payload.Requests[0].RequiresResponse)
	assert.Equal(t, "get_canvas_context", payload.Requests[0].Command.Op)
	assert.Greater(t, payload.Requests[0].ID, int64(0))
}

func TestWorkflowCanvasMCPBrowserCommandPollingResetsStaleCursor(t *testing.T) {
	h := server.Default()
	relay := canvasautomation.NewMemoryWorkflowCommandRelay()
	RegisterWithRelay(h, relay)

	require.NoError(t, relay.DispatchWorkflowCommand(context.Background(), canvasautomation.WorkflowCanvasDispatchCommand{
		WorkflowID: "wf-1",
		SpaceID:    "space-1",
		Command: canvasautomation.WorkflowCanvasCommand{
			Op:     "add_node",
			Target: "reply_text",
			Args:   map[string]any{"type": "15"},
		},
	}))

	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/workflow_mcp/browser_commands?workflow_id=wf-1&space_id=space-1&after_id=99",
		nil,
	)

	require.Equal(t, http.StatusOK, w.Code)
	var payload struct {
		Cursor   int64 `json:"cursor"`
		Requests []struct {
			ID int64 `json:"id"`
		} `json:"requests"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Len(t, payload.Requests, 1)
	assert.Equal(t, payload.Requests[0].ID, payload.Cursor)
}

func TestWorkflowCanvasMCPBrowserCommandResultSubmission(t *testing.T) {
	h := server.Default()
	relay := canvasautomation.NewMemoryWorkflowCommandRelay()
	RegisterWithRelay(h, relay)

	body := `{"workflow_id":"wf-1","space_id":"space-1","result":{"request_id":"req-context","status":"ok","canvas_context":"节点: Start -> End"}}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/workflow_mcp/browser_command_results",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)
	result, ok := relay.WaitWorkflowCommandResult(context.Background(), "req-context")
	require.True(t, ok)
	assert.Equal(t, "节点: Start -> End", result.CanvasContext)
}

func TestWorkflowCanvasMCPRunNodeSmokeReturnsBrowserReportThroughRoute(t *testing.T) {
	h := server.Default()
	relay := canvasautomation.NewMemoryWorkflowCommandRelay()
	RegisterWithRelay(h, relay)

	sessionID := initializeWorkflowMCPRoute(t, h)
	callBody := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"workflow.run_node_smoke","arguments":{"workflow_id":"wf-1","space_id":"space-1","node_type":"32","allow_temporary_workflow":true,"include_skipped":false,"timeout_ms":1000}}}`
	responseCh := make(chan *ut.ResponseRecorder, 1)
	go func() {
		responseCh <- ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/workflow_mcp/mcp",
			&ut.Body{Body: bytes.NewBufferString(callBody), Len: len(callBody)},
			ut.Header{Key: "Content-Type", Value: "application/json"},
			ut.Header{Key: "Accept", Value: "application/json, text/event-stream"},
			ut.Header{Key: "Mcp-Session-Id", Value: sessionID},
		)
	}()

	requestID := waitForRouteNodeSmokeRequest(t, h)
	resultBody := `{"workflow_id":"wf-1","space_id":"space-1","result":{"request_id":"` + requestID + `","status":"ok","results":[{"op":"run_node_smoke","ok":true,"target":"32","message":"node smoke completed"}],"node_smoke_report":{"total":1,"nodes":[{"nodeType":"32","executionStatus":"executed"}]}}}`
	resultResp := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/workflow_mcp/browser_command_results",
		&ut.Body{Body: bytes.NewBufferString(resultBody), Len: len(resultBody)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	require.Equal(t, http.StatusOK, resultResp.Code)

	select {
	case w := <-responseCh:
		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, string(w.Body.Bytes()), `"node_smoke_report"`)
		assert.Contains(t, string(w.Body.Bytes()), `"op":"run_node_smoke"`)
		assert.Contains(t, string(w.Body.Bytes()), `"nodeType":"32"`)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for workflow.run_node_smoke MCP response")
	}
}

func initializeWorkflowMCPRoute(t *testing.T, h *server.Hertz) string {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"route-test","version":"1.0.0"}}}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/workflow_mcp/mcp",
		&ut.Body{Body: bytes.NewBufferString(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "Accept", Value: "application/json, text/event-stream"},
	)
	require.Equal(t, http.StatusOK, w.Code)
	sessionID := w.Header().Get("Mcp-Session-Id")
	require.NotEmpty(t, sessionID)
	return sessionID
}

func waitForRouteNodeSmokeRequest(t *testing.T, h *server.Hertz) string {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/workflow_mcp/browser_commands?workflow_id=wf-1&space_id=space-1",
			nil,
		)
		require.Equal(t, http.StatusOK, w.Code)
		var payload struct {
			Requests []struct {
				RequestID        string `json:"request_id"`
				RequiresResponse bool   `json:"requires_response"`
				Command          struct {
					Op     string         `json:"op"`
					Target string         `json:"target"`
					Args   map[string]any `json:"args"`
				} `json:"command"`
			} `json:"requests"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
		if len(payload.Requests) > 0 {
			request := payload.Requests[0]
			require.Equal(t, "run_node_smoke", request.Command.Op)
			require.Equal(t, "32", request.Command.Target)
			require.True(t, request.RequiresResponse)
			require.Equal(t, true, request.Command.Args["allow_temporary_workflow"])
			require.Equal(t, false, request.Command.Args["include_skipped"])
			require.NotEmpty(t, request.RequestID)
			return request.RequestID
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for browser-live node smoke request")
	return ""
}
