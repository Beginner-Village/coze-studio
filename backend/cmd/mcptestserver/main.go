// Command mcptestserver is a real (non-mock) MCP server over streamable-http used to
// verify the super-agent's MCP runtime integration end-to-end against a genuine external
// MCP server. It exposes one real "add" tool. Run: mcptestserver :9900  (URL: /mcp).
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	addr := ":9900"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	s := server.NewMCPServer("ynet-mcp-demo", "1.0.0", server.WithToolCapabilities(true))
	s.AddTool(
		mcp.NewTool("add",
			mcp.WithDescription("Add two integers and return their sum. Use this for any addition."),
			mcp.WithNumber("a", mcp.Required(), mcp.Description("first integer")),
			mcp.WithNumber("b", mcp.Required(), mcp.Description("second integer")),
		),
		func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return mcp.NewToolResultText(strconv.FormatFloat(a+b, 'f', -1, 64)), nil
		},
	)

	fmt.Printf("[mcptestserver] real MCP server listening on %s/mcp\n", addr)
	if err := server.NewStreamableHTTPServer(s).Start(addr); err != nil {
		panic(err)
	}
}
