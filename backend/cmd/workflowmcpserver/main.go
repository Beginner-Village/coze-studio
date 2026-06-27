// Command workflowmcpserver exposes the system-provided workflow canvas MCP tools over
// streamable-http. It is a thin runtime wrapper around application/workflow/canvasautomation.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	appworkflow "github.com/ynet-dev/ynet-studio/backend/application/workflow"
	"github.com/ynet-dev/ynet-studio/backend/application/workflow/canvasautomation"
)

func main() {
	addr := ":9901"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	mcpServer := newWorkflowMCPServer(applicationWorkflowTestRunner{})
	fmt.Printf("[workflowmcpserver] workflow canvas MCP server listening on %s/mcp\n", addr)
	if err := server.NewStreamableHTTPServer(mcpServer).Start(addr); err != nil {
		panic(err)
	}
}

func newWorkflowMCPServer(runner canvasautomation.WorkflowTestRunner) *server.MCPServer {
	return canvasautomation.NewWorkflowMCPServer(canvasautomation.WorkflowMCPOptions{
		TestRunner: runner,
	})
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
