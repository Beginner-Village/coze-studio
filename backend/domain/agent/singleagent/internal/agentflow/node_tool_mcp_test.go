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

package agentflow

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
)

// startTestMCPServer spins up a real in-memory MCP server over streamable-http with one
// "echo" tool, so the test exercises the genuine MCP protocol end-to-end (not a mock).
func startTestMCPServer(t *testing.T) string {
	t.Helper()
	s := mcpserver.NewMCPServer("test-mcp", "1.0.0", mcpserver.WithToolCapabilities(true))
	s.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("Echo back the provided message"),
			mcp.WithString("message", mcp.Required(), mcp.Description("text to echo")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			msg, _ := req.GetArguments()["message"].(string)
			return mcp.NewToolResultText("echo: " + msg), nil
		},
	)
	ts := mcpserver.NewTestStreamableHTTPServer(s)
	t.Cleanup(ts.Close)
	return ts.URL
}

// The MCP adapter must connect to a real MCP server, list its tools, and the tools must be
// invokable through the eino tool interface — proving dynamic MCP 接入 works end-to-end.
func TestNewMCPTools_ConnectsAndInvokes(t *testing.T) {
	url := startTestMCPServer(t)
	ctx := context.Background()

	tools := newMCPTools(ctx, []*crossagent.MCPServerConfig{
		{Name: "test", Type: "streamable_http", URL: url},
	})
	if len(tools) == 0 {
		t.Fatalf("expected MCP tools from server, got none")
	}

	var echo tool.InvokableTool
	for _, tl := range tools {
		info, err := tl.Info(ctx)
		if err != nil {
			t.Fatalf("Info err: %v", err)
		}
		if info.Name == "echo" {
			inv, ok := tl.(tool.InvokableTool)
			if !ok {
				t.Fatalf("echo tool is not invokable")
			}
			echo = inv
		}
	}
	if echo == nil {
		t.Fatalf("MCP server's 'echo' tool was not exposed; got %d tools", len(tools))
	}

	out, err := echo.InvokableRun(ctx, `{"message":"hello"}`)
	if err != nil {
		t.Fatalf("invoke echo err: %v", err)
	}
	if !strings.Contains(out, "echo: hello") {
		t.Fatalf("echo tool returned %q, want it to contain %q", out, "echo: hello")
	}
}

// Unsupported / unsafe transports must be rejected (stdio = backend RCE risk).
func TestConnectMCPServer_RejectsUnsafeTransports(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ typ, wantSub string }{
		{"stdio", "disabled"},
		{"websocket", "unsupported"},
	} {
		_, err := connectMCPServer(ctx, &crossagent.MCPServerConfig{Name: "x", Type: tc.typ, URL: "http://x"})
		if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
			t.Fatalf("transport %q: want error containing %q, got %v", tc.typ, tc.wantSub, err)
		}
	}
	// Disabled server must be skipped (no tools, no error).
	disabled := false
	tools := newMCPTools(ctx, []*crossagent.MCPServerConfig{{Name: "off", Type: "sse", URL: "http://x", Enabled: &disabled}})
	if len(tools) != 0 {
		t.Fatalf("disabled MCP server must yield no tools; got %d", len(tools))
	}
}
