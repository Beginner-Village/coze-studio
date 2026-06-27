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

// workflow_canvas_list_plugins —— 给编排 agent 返回当前空间【真实可用】的插件及其工具(API)。
// 这是把"资源契约"换成 live 数据的第一步(能力 B):配置 type=4 插件/API 节点前,agent 先调本工具
// 拿到真实 plugin_id / api_id(=tool_id) / 参数 schema,从而不再编造、不再因缺 version 而重试。
// 走 application 层单例 PluginApplicationSVC(其鉴权读 ctx session;空间 id 用运行时 Ext/deps)。

import (
	"context"
	"encoding/json"
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
		return &wfCanvasListPluginsTool{deps: deps}
	})
}

type wfCanvasListPluginsTool struct{ deps superAgentToolDeps }

type wfCanvasListPluginsArgs struct {
	Keyword   string `json:"keyword,omitempty"`
	PluginIDs string `json:"plugin_ids,omitempty"` // 逗号分隔的 plugin_id
}

func (t *wfCanvasListPluginsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_list_plugins",
		Desc: "列出当前空间【真实可用】的插件及其工具(API)。配置 type=4 插件/API 节点前【必须】先调本工具," +
			"拿到真实 plugin_id、api_id(=tool_id)、tool 名称/HTTP 方法/路径/参数 schema,禁止编造 ID。" +
			"可用 keyword 模糊过滤插件名,或用 plugin_ids(逗号分隔)精确查。返回 JSON:plugins[] 每项含插件 id/name/desc 及其 api 列表。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"keyword":    {Type: schema.String, Desc: "可选,按插件名/描述模糊过滤", Required: false},
			"plugin_ids": {Type: schema.String, Desc: "可选,逗号分隔的 plugin_id,精确查这些插件", Required: false},
		}),
	}, nil
}

func (t *wfCanvasListPluginsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args wfCanvasListPluginsArgs
	if strings.TrimSpace(argumentsInJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			return argParseErrMsg(err), nil
		}
	}
	spaceID := t.spaceID()
	if spaceID == 0 {
		return wfCanvasPluginsErr("无法确定当前空间(space_id),无法列插件"), nil
	}
	req := &pluginAPI.GetPlaygroundPluginListRequest{
		SpaceID: ptr.Of(spaceID),
		Page:    ptr.Of(int32(1)),
		Size:    ptr.Of(int32(50)),
	}
	if kw := strings.TrimSpace(args.Keyword); kw != "" {
		req.Name = ptr.Of(kw)
	}
	if ids := wfCanvasSplitCSV(args.PluginIDs); len(ids) > 0 {
		req.PluginIds = ids
	}
	resp, err := pluginapp.PluginApplicationSVC.GetPlaygroundPluginList(ctx, req)
	if err != nil {
		return wfCanvasPluginsErr("列插件失败: " + err.Error()), nil
	}
	out := map[string]any{"status": "ok"}
	if resp != nil && resp.Data != nil {
		out["total"] = resp.Data.Total
		out["plugins"] = resp.Data.PluginList
		out["instruction"] = "选定 tool 后,用其真实 plugin_id+api_id 调 workflow_canvas_configure_node 配置 type=4 节点;参数按返回的 schema 绑定真实变量或明确字面量。用 plugin_ids 精确查某插件可额外拿到其 API 的输出字段(response_params),用于配置下游节点引用插件返回值。"
	}
	// 指定 plugin_ids 时,补每个插件 API 的【输出 schema】(response_params),让 agent 知道插件返回啥、好配下游(如用余额结果做 IF)。
	if ids := wfCanvasSplitCSV(args.PluginIDs); len(ids) > 0 {
		details := make([]map[string]any, 0, len(ids))
		for _, idStr := range ids {
			pid, perr := strconv.ParseInt(idStr, 10, 64)
			if perr != nil {
				continue
			}
			apisResp, aerr := pluginapp.PluginApplicationSVC.GetPluginAPIs(ctx, &pluginAPI.GetPluginAPIsRequest{
				PluginID: pid,
				Page:     1,
				Size:     50,
			})
			if aerr != nil || apisResp == nil {
				continue
			}
			for _, api := range apisResp.APIInfo {
				if api == nil {
					continue
				}
				details = append(details, map[string]any{
					"plugin_id":       idStr,
					"api_id":          api.APIID,
					"api_name":        api.Name,
					"request_params":  api.RequestParams,
					"response_params": api.ResponseParams,
				})
			}
		}
		if len(details) > 0 {
			out["api_details"] = details
		}
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

// spaceID 优先取前端通过 Ext 传入的画布空间,否则回退到构造时的 deps.SpaceID。
func (t *wfCanvasListPluginsTool) spaceID() int64 {
	if t.deps.Ext != nil {
		if s := wfExtValue(t.deps.Ext, workflowCanvasSpaceIDExtKey); s != "" {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil && id != 0 {
				return id
			}
		}
	}
	return t.deps.SpaceID
}

func wfCanvasPluginsErr(msg string) string {
	b, _ := json.Marshal(map[string]string{"status": "error", "message": msg})
	return string(b)
}

func wfCanvasSplitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
