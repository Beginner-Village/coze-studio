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

// 能力 C:用户给一条 CURL,agent 自动在当前空间建插件并发布,返回真实 plugin_id/api_id,直接可用于 type=4 节点。
// 解析 curl(method/url/headers/JSON body)→ 生成 PluginManifest + OpenAPI(JSON)→ RegisterPlugin → PublishPlugin(v1.0.0)。

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	pluginAPI "github.com/ynet-dev/ynet-studio/backend/api/model/plugin_develop"
	pluginapp "github.com/ynet-dev/ynet-studio/backend/application/plugin"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

func init() {
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasCreatePluginTool{deps: deps}
	})
}

type wfCanvasCreatePluginTool struct{ deps superAgentToolDeps }

func (t *wfCanvasCreatePluginTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_create_plugin_from_curl",
		Desc: "根据一条 CURL 命令在当前空间【创建并发布】一个插件,返回真实 plugin_id 和 api_id,可直接用于 type=4 插件节点。" +
			"自动解析 curl 的 method/url/headers/JSON body 生成 OpenAPI,并发布为 v1.0.0。name 为插件显示名。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"curl": {Type: schema.String, Desc: "完整 curl 命令(支持 -X/-H/-d/--data 等)", Required: true},
			"name": {Type: schema.String, Desc: "插件显示名(中文可)", Required: false},
			"desc": {Type: schema.String, Desc: "插件/接口说明", Required: false},
		}),
	}, nil
}

func (t *wfCanvasCreatePluginTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Curl string `json:"curl"`
		Name string `json:"name"`
		Desc string `json:"desc"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return argParseErrMsg(err), nil
	}
	if strings.TrimSpace(args.Curl) == "" {
		return wfCanvasDBErr("curl 不能为空"), nil
	}
	space := wfCanvasSpaceID(t.deps)
	if space == 0 {
		return wfCanvasDBErr("无法确定 space_id"), nil
	}
	method, rawurl, headers, body := wfParseCurl(args.Curl)
	if rawurl == "" {
		return wfCanvasDBErr("curl 里没解析到 URL"), nil
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return wfCanvasDBErr("URL 解析失败: " + err.Error()), nil
	}
	baseURL := u.Scheme + "://" + u.Host
	path := u.Path
	if path == "" {
		path = "/"
	}
	name := strings.TrimSpace(args.Name)
	if name == "" {
		name = "curl_" + u.Host
	}
	desc := strings.TrimSpace(args.Desc)
	if desc == "" {
		desc = name
	}

	manifest := map[string]any{
		"schema_version":        "v1",
		"name_for_human":        name,
		"name_for_model":        wfSanitizeName(name),
		"description_for_human": desc,
		"description_for_model": desc,
		"auth":                  map[string]any{"type": "none"},
		"api":                   map[string]any{"type": "cloud"},
		"common_params":         wfCurlCommonParams(headers),
	}
	manifestJSON, _ := json.Marshal(manifest)

	op := map[string]any{
		"operationId": wfSanitizeName(method + "_" + path),
		"summary":     desc,
		"responses": map[string]any{
			"200": map[string]any{
				"description": "ok",
				"content": map[string]any{"application/json": map[string]any{
					"schema": map[string]any{"type": "object", "properties": map[string]any{
						"result": map[string]any{"type": "string", "description": "result"},
					}},
				}},
			},
		},
	}
	if qp := wfQueryParams(u); len(qp) > 0 {
		op["parameters"] = qp
	}
	if props := wfBodyProps(body); len(props) > 0 {
		op["requestBody"] = map[string]any{"content": map[string]any{"application/json": map[string]any{
			"schema": map[string]any{"type": "object", "properties": props},
		}}}
	}
	openapi := map[string]any{
		"openapi": "3.0.1",
		"info":    map[string]any{"title": name, "version": "v1", "description": desc},
		"servers": []any{map[string]any{"url": baseURL}},
		"paths":   map[string]any{path: map[string]any{strings.ToLower(method): op}},
	}
	openapiJSON, _ := json.Marshal(openapi)

	reg, err := pluginapp.PluginApplicationSVC.RegisterPlugin(ctx, &pluginAPI.RegisterPluginRequest{
		AiPlugin: string(manifestJSON),
		Openapi:  string(openapiJSON),
		SpaceID:  space,
	})
	if err != nil {
		return wfCanvasDBErr("建插件失败: " + err.Error()), nil
	}
	var pluginID int64
	if reg != nil && reg.Data != nil {
		pluginID = reg.Data.PluginID
	}
	if pluginID == 0 {
		return wfCanvasDBErr("建插件未返回 plugin_id"), nil
	}
	if _, err := pluginapp.PluginApplicationSVC.PublishPlugin(ctx, &pluginAPI.PublishPluginRequest{
		PluginID:    pluginID,
		VersionName: "v1.0.0",
		VersionDesc: "created from curl",
	}); err != nil {
		b, _ := json.Marshal(map[string]any{
			"status": "partial", "plugin_id": strconv.FormatInt(pluginID, 10),
			"message": "已建草稿但发布失败: " + err.Error(),
		})
		return string(b), nil
	}
	list, _ := pluginapp.PluginApplicationSVC.GetPlaygroundPluginList(ctx, &pluginAPI.GetPlaygroundPluginListRequest{
		SpaceID:   ptr.Of(space),
		PluginIds: []string{strconv.FormatInt(pluginID, 10)},
		Page:      ptr.Of(int32(1)),
		Size:      ptr.Of(int32(1)),
	})
	out := map[string]any{"status": "ok", "plugin_id": strconv.FormatInt(pluginID, 10), "plugin_version": "v1.0.0"}
	if list != nil && list.Data != nil {
		out["plugin"] = list.Data.PluginList
		out["instruction"] = "用返回的 plugin_id + 其 api 的 api_id + plugin_version=v1.0.0 配 type=4 插件节点。"
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// ---------- curl 解析 + OpenAPI 构造 helpers ----------

func wfTokenizeShell(s string) []string {
	var toks []string
	var cur strings.Builder
	inS, inD := false, false
	flush := func() {
		if cur.Len() > 0 {
			toks = append(toks, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'' && !inD:
			inS = !inS
		case c == '"' && !inS:
			inD = !inD
		case c == '\\' && inD && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
		case (c == ' ' || c == '\t' || c == '\n' || c == '\r') && !inS && !inD:
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return toks
}

func wfParseCurl(raw string) (method, rawurl string, headers map[string]string, body string) {
	headers = map[string]string{}
	toks := wfTokenizeShell(raw)
	for i := 0; i < len(toks); i++ {
		tk := toks[i]
		switch {
		case tk == "-X" || tk == "--request":
			if i+1 < len(toks) {
				method = strings.ToUpper(toks[i+1])
				i++
			}
		case tk == "-H" || tk == "--header":
			if i+1 < len(toks) {
				h := toks[i+1]
				i++
				if idx := strings.Index(h, ":"); idx > 0 {
					headers[strings.TrimSpace(h[:idx])] = strings.TrimSpace(h[idx+1:])
				}
			}
		case tk == "-d" || tk == "--data" || tk == "--data-raw" || tk == "--data-binary":
			if i+1 < len(toks) {
				body = toks[i+1]
				i++
			}
		case strings.HasPrefix(tk, "http://") || strings.HasPrefix(tk, "https://"):
			rawurl = tk
		}
	}
	if method == "" {
		if body != "" {
			method = "POST"
		} else {
			method = "GET"
		}
	}
	return method, rawurl, headers, body
}

func wfSanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_':
			b.WriteRune(r)
		case r == '/' || r == '-' || r == '.' || r == ' ':
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "api"
	}
	return out
}

func wfCurlCommonParams(headers map[string]string) map[string]any {
	hdr := make([]any, 0, len(headers))
	for k, v := range headers {
		hdr = append(hdr, map[string]any{"name": k, "value": v})
	}
	return map[string]any{"header": hdr, "query": []any{}, "body": []any{}, "path": []any{}}
}

func wfQueryParams(u *url.URL) []any {
	var out []any
	for k := range u.Query() {
		out = append(out, map[string]any{
			"name": k, "in": "query", "required": false,
			"schema": map[string]any{"type": "string"}, "description": k,
		})
	}
	return out
}

func wfBodyProps(body string) map[string]any {
	body = strings.TrimSpace(body)
	if body == "" || body[0] != '{' {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return nil
	}
	props := map[string]any{}
	for k, v := range m {
		typ := "string"
		switch v.(type) {
		case float64:
			typ = "number"
		case bool:
			typ = "boolean"
		}
		props[k] = map[string]any{"type": typ, "description": k}
	}
	return props
}
