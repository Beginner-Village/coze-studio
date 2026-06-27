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

// node_tool_extensions.go 实现「路线 P5」：超级智能体的**可插拔工具机制**。
//
// 核心是一个注册表：任何「能力提供方」（未来的浏览器工具、doc 工具、MCP server
// 适配器等）只要 registerSuperAgentExtension(factory)，其工具就会**只挂给超级 agent**
// （isSuperAgent），普通 agent 完全不受影响。这样以后往里加新工具/技能 = 注册一个
// factory，核心运行时零改动 —— 这就是用户要的「以后能对接不同功能、自由组合」。
//
// 本文件同时内置一个示范工具 web_fetch（在沙箱里 curl 拉取 URL 文本），用来验证机制
// 跑通。完整 MCP server 发现可作为本注册表后面的一个 provider 接入（不在本期）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
)

// parseSuperAgentUserID parses the runtime user id (a decimal string in Config)
// to int64 for per-user capabilities like memory; returns 0 when absent/invalid.
func parseSuperAgentUserID(s string) int64 {
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// superAgentToolDeps 是构造一个超级 agent 扩展工具所需的运行时依赖。
// SandboxKey 给沙箱类工具用;UserID/SpaceID/AgentID 给 per-user 能力(如记忆)用。
type superAgentToolDeps struct {
	SandboxKey  string
	UserID      int64
	SpaceID     int64
	AgentID     int64
	Ext         map[string]string
	CanvasRelay workflowCanvasLiveRelay
	// Ledger 记录本轮 agent 已下发的画布编辑;BuildAgent per-message 构造,故天然 per-turn。
	Ledger *turnCanvasLedger
}

// superAgentExtensionFactory 按运行时依赖造一个工具。
type superAgentExtensionFactory func(deps superAgentToolDeps) tool.InvokableTool

// superAgentExtensions 是已注册的超级 agent 扩展工具工厂。
var superAgentExtensions []superAgentExtensionFactory

// registerSuperAgentExtension 注册一个只挂给超级 agent 的扩展工具工厂。
// 由各能力提供方在 init() 里调用。
func registerSuperAgentExtension(f superAgentExtensionFactory) {
	superAgentExtensions = append(superAgentExtensions, f)
}

// newSuperAgentExtensionTools 为超级 agent 构造全部已注册的扩展工具。
func newSuperAgentExtensionTools(deps superAgentToolDeps) []tool.InvokableTool {
	out := make([]tool.InvokableTool, 0, len(superAgentExtensions))
	for _, f := range superAgentExtensions {
		if t := f(deps); t != nil {
			out = append(out, t)
		}
	}
	return out
}

func init() {
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &webFetchTool{key: deps.SandboxKey}
	})
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &webSearchTool{key: deps.SandboxKey}
	})
}

// ---- 内置工具：web_search（沙箱内查 Bing,免费、无需 API key）----
// 后端服务器没有通用外网,但沙箱可达 cn.bing.com,故在沙箱里用 python 抓取并解析。

type webSearchTool struct{ key string }

type webSearchRequest struct {
	Query string `json:"query" jsonschema:"description=The search query keywords"`
}

// bingSearchPy 抓取 cn.bing.com 搜索结果页并解析出 标题/链接/摘要,输出 JSON 数组。
const bingSearchPy = "import sys,urllib.request,urllib.parse,re,json,html\n" +
	"q=sys.argv[1]\n" +
	"url='https://cn.bing.com/search?q='+urllib.parse.quote(q)\n" +
	"req=urllib.request.Request(url,headers={'User-Agent':'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120 Safari/537.36'})\n" +
	"d=urllib.request.urlopen(req,timeout=15).read().decode('utf-8','replace')\n" +
	"r=[]\n" +
	"for b in d.split('class=\"b_algo\"')[1:9]:\n" +
	"    m=re.search(r'<h2[^>]*><a[^>]*href=\"([^\"]+)\"[^>]*>(.*?)</a>',b,re.S)\n" +
	"    if not m: continue\n" +
	"    t=html.unescape(re.sub(r'<[^>]+>','',m.group(2))).strip()\n" +
	"    sn=re.search(r'<p[^>]*>(.*?)</p>',b,re.S)\n" +
	"    s=html.unescape(re.sub(r'<[^>]+>','',sn.group(1))).strip()[:240] if sn else ''\n" +
	"    r.append({'title':t,'url':m.group(1),'snippet':s})\n" +
	"print(json.dumps(r,ensure_ascii=False))"

func (t *webSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "Search the web for up-to-date information by keywords (uses Bing). Returns a JSON array of results, each with title, url and snippet. Use this to find current facts, news or pages; then optionally use web_fetch to read a specific url.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "The search query keywords", Required: true},
		}),
	}, nil
}

func (t *webSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req webSearchRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		return "Error: query is required", nil
	}
	cmd := fmt.Sprintf("python3 -c %s %s", shQuote(bingSearchPy), shQuote(q))
	res, err := svc.Exec(ctx, t.key, cmd, 30)
	if err != nil {
		return fmt.Sprintf("Error searching: %v", err), nil
	}
	out := strings.TrimSpace(res.Stdout)
	if out == "" || out == "[]" {
		return fmt.Sprintf("(no results; exit_code=%d stderr=%s)", res.ExitCode, res.Stderr), nil
	}
	return offloadToolResultOrTruncate(ctx, svc, t.key, toolOutputOffloadMeta{
		Tool:      "web_search",
		Arguments: req,
	}, out), nil
}

// ---- 示范工具：web_fetch（沙箱内 curl 拉取 URL）----

type webFetchTool struct{ key string }

type webFetchRequest struct {
	URL string `json:"url" jsonschema:"description=The http(s) URL to fetch"`
}

func (t *webFetchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_fetch",
		Desc: "Fetch the text content of an http(s) URL (runs curl inside the sandbox). Use to read a web page or call an HTTP API. Returns the response body (truncated if large).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {Type: schema.String, Desc: "The http(s) URL to fetch", Required: true},
		}),
	}, nil
}

func (t *webFetchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req webFetchRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	u := strings.TrimSpace(req.URL)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "Error: url must start with http:// or https://", nil
	}
	// 用沙箱内一定存在的 python3 urllib 拉取（slim 镜像默认无 curl/wget）。
	const fetchPy = "import sys,urllib.request\n" +
		"req=urllib.request.Request(sys.argv[1],headers={'User-Agent':'ynet-agent'})\n" +
		"sys.stdout.write(urllib.request.urlopen(req,timeout=20).read().decode('utf-8','replace'))"
	cmd := fmt.Sprintf("python3 -c %s %s", shQuote(fetchPy), shQuote(u))
	res, err := svc.Exec(ctx, t.key, cmd, 30)
	if err != nil {
		return fmt.Sprintf("Error fetching url: %v", err), nil
	}
	if strings.TrimSpace(res.Stdout) == "" {
		return fmt.Sprintf("(empty response; exit_code=%d stderr=%s)", res.ExitCode, res.Stderr), nil
	}
	return offloadToolResultOrTruncate(ctx, svc, t.key, toolOutputOffloadMeta{
		Tool:      "web_fetch",
		Arguments: req,
	}, res.Stdout), nil
}

// shQuote 把字符串安全包成单引号 shell 参数。
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
