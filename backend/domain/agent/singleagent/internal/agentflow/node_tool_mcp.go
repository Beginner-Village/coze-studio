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

// node_tool_mcp.go：超级智能体的 MCP（Model Context Protocol）动态接入。
//
// 架构（不手搓协议，用官方栈）：
//   - 协议/传输层：mark3labs/mcp-go（MCP 官方生态成熟实现，支持 SSE / streamable-http）
//   - eino 适配层：cloudwego/eino-ext/components/tool/mcp 的 GetTools，把 MCP server 暴露的
//     工具直接包成 eino tool.BaseTool，无缝进入我们的 ReAct 循环。
//
// 安全：只接受**远程 URL**（sse / streamable_http）。stdio 传输会在后端进程内 fork 任意
// 子进程命令，对平台是 RCE 风险，故显式禁用。MCP server 列表与认证头来自 per-agent 的
// SuperAgentToolConfig.MCPServers（用户在设置弹窗里配），仅超级体生效。

import (
	"context"
	"fmt"
	"time"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

const (
	mcpDefaultTimeoutSec = 20
	mcpClientName        = "ynet-super-agent"
	mcpClientVersion     = "1.0.0"
)

// newMCPTools 连接每个启用的 MCP server，返回它们暴露的全部工具（eino BaseTool）。
// 单个 server 连接/握手失败不阻塞其它 server，也不阻塞整个 agent 构建 —— 仅记录告警并跳过。
func newMCPTools(ctx context.Context, servers []*crossagent.MCPServerConfig) []tool.BaseTool {
	if len(servers) == 0 {
		return nil
	}
	var all []tool.BaseTool
	for _, s := range servers {
		if s == nil {
			continue
		}
		if s.Enabled != nil && !*s.Enabled {
			continue
		}
		tools, err := connectMCPServer(ctx, s)
		if err != nil {
			logs.CtxWarnf(ctx, "[MCP] server %q (%s) unavailable, skipped: %v", s.Name, s.Type, err)
			continue
		}
		all = append(all, tools...)
		logs.CtxInfof(ctx, "[MCP] server %q exposed %d tool(s)", s.Name, len(tools))
	}
	return all
}

// connectMCPServer 建链 + 握手 + 拉取工具。仅支持远程 sse / streamable_http。
func connectMCPServer(ctx context.Context, s *crossagent.MCPServerConfig) ([]tool.BaseTool, error) {
	if s.URL == "" {
		return nil, fmt.Errorf("mcp server url is required")
	}
	timeout := time.Duration(s.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = mcpDefaultTimeoutSec * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cli *mcpclient.Client
	var err error
	switch s.Type {
	case "sse":
		cli, err = mcpclient.NewSSEMCPClient(s.URL, mcptransport.WithHeaders(s.Env))
	case "streamable_http", "streamable-http", "http":
		cli, err = mcpclient.NewStreamableHttpClient(s.URL, mcptransport.WithHTTPHeaders(s.Env))
	case "stdio":
		// 后端进程内 fork 任意命令 = 平台 RCE 风险，禁用。
		return nil, fmt.Errorf("stdio transport is disabled for platform safety; use a remote sse/streamable_http MCP server")
	default:
		return nil, fmt.Errorf("unsupported MCP transport %q (allowed: sse, streamable_http)", s.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	if err = cli.Start(dialCtx); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: mcpClientName, Version: mcpClientVersion}
	if _, err = cli.Initialize(dialCtx, initReq); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli, CustomHeaders: s.Env})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	return tools, nil
}
