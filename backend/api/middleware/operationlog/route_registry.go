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

package operationlog

import (
	"encoding/json"
	"strconv"
	"strings"
)

// ResourceType 枚举与 domain/permission/consts.go 对齐。
const (
	resourceTypeApp       int32 = 3
	resourceTypeBot       int32 = 4
	resourceTypePlugin    int32 = 5
	resourceTypeWorkflow  int32 = 6
	resourceTypeKnowledge int32 = 7
	resourceTypePrompt    int32 = 17
	resourceTypeDatabase  int32 = 23
	resourceTypeWorkspace int32 = 2
)

// RouteRule 描述一条被审计的写接口及其语义映射。
type RouteRule struct {
	Method       string // POST/PUT/DELETE/PATCH
	PathPattern  string // 形如 /api/workflow_api/save;段以 ':' 开头为通配
	Module       string
	ResourceType int32
	Action       string
	DescTemplate string // 中文语义描述
	// 抽取来源："body:<field>" / "query:<field>" / "path:<seg>"
	ResourceIDFrom   string
	ResourceNameFrom string
}

// rules 路由白名单。path 与 body 字段名均对照 backend/api/router/* 与 api/model/* 真实注册核对。
// 覆盖模块：workflow / agent(draftbot) / super-agent sandbox / hi-agent / knowledge / plugin / database / variable(memory) / prompt / space member / space。
var rules = []RouteRule{
	// ---- workflow（/api/workflow_api，见 api/router/workflow/workflow_svc.go）----
	{Method: "POST", PathPattern: "/api/workflow_api/create", Module: "workflow", ResourceType: resourceTypeWorkflow, Action: "create", DescTemplate: "创建了工作流", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/workflow_api/save", Module: "workflow", ResourceType: resourceTypeWorkflow, Action: "update", DescTemplate: "更新了工作流", ResourceIDFrom: "body:workflow_id", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/workflow_api/update_meta", Module: "workflow", ResourceType: resourceTypeWorkflow, Action: "update", DescTemplate: "更新了工作流信息", ResourceIDFrom: "body:workflow_id", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/workflow_api/delete", Module: "workflow", ResourceType: resourceTypeWorkflow, Action: "delete", DescTemplate: "删除了工作流", ResourceIDFrom: "body:workflow_id"},
	{Method: "POST", PathPattern: "/api/workflow_api/publish", Module: "workflow", ResourceType: resourceTypeWorkflow, Action: "publish", DescTemplate: "发布了工作流", ResourceIDFrom: "body:workflow_id"},
	{Method: "POST", PathPattern: "/api/workflow_api/copy", Module: "workflow", ResourceType: resourceTypeWorkflow, Action: "copy", DescTemplate: "复制了工作流", ResourceIDFrom: "body:workflow_id"},

	// ---- agent / bot（/api/draftbot，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/draftbot/create", Module: "agent", ResourceType: resourceTypeBot, Action: "create", DescTemplate: "创建了智能体", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/draftbot/update_display_info", Module: "agent", ResourceType: resourceTypeBot, Action: "update", DescTemplate: "更新了智能体", ResourceIDFrom: "body:bot_id"},
	{Method: "POST", PathPattern: "/api/draftbot/delete", Module: "agent", ResourceType: resourceTypeBot, Action: "delete", DescTemplate: "删除了智能体", ResourceIDFrom: "body:bot_id"},
	{Method: "POST", PathPattern: "/api/draftbot/publish", Module: "agent", ResourceType: resourceTypeBot, Action: "publish", DescTemplate: "发布了智能体", ResourceIDFrom: "body:bot_id"},
	{Method: "POST", PathPattern: "/api/draftbot/duplicate", Module: "agent", ResourceType: resourceTypeBot, Action: "copy", DescTemplate: "复制了智能体", ResourceIDFrom: "body:bot_id"},

	// ---- super-agent sandbox / BashTool（/api/super-agent，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/super-agent/runs/create", Module: "super_agent", ResourceType: resourceTypeBot, Action: "create_run", DescTemplate: "创建了超级体运行", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:content"},
	{Method: "POST", PathPattern: "/api/super-agent/runs/reply", Module: "super_agent", ResourceType: resourceTypeBot, Action: "reply_run", DescTemplate: "继续了超级体运行", ResourceIDFrom: "body:run_id", ResourceNameFrom: "body:content"},
	{Method: "POST", PathPattern: "/api/super-agent/runs/cancel", Module: "super_agent", ResourceType: resourceTypeBot, Action: "cancel_run", DescTemplate: "取消了超级体运行", ResourceIDFrom: "body:run_id"},
	{Method: "POST", PathPattern: "/api/super-agent/sessions/create", Module: "super_agent", ResourceType: resourceTypeBot, Action: "create_session", DescTemplate: "创建了超级体会话", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:title"},
	{Method: "POST", PathPattern: "/api/super-agent/sessions/rename", Module: "super_agent", ResourceType: resourceTypeBot, Action: "rename_session", DescTemplate: "重命名了超级体会话", ResourceIDFrom: "body:conversation_id", ResourceNameFrom: "body:title"},
	{Method: "POST", PathPattern: "/api/super-agent/sessions/delete", Module: "super_agent", ResourceType: resourceTypeBot, Action: "delete_session", DescTemplate: "删除了超级体会话", ResourceIDFrom: "body:conversation_id"},
	{Method: "POST", PathPattern: "/api/super-agent/harness/plan", Module: "super_agent", ResourceType: resourceTypeBot, Action: "update_plan", DescTemplate: "更新了超级体执行计划", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:conversation_id"},
	{Method: "POST", PathPattern: "/api/super-agent/harness/context/clear", Module: "super_agent", ResourceType: resourceTypeBot, Action: "clear_context", DescTemplate: "清理了超级体上下文", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:conversation_id"},
	{Method: "POST", PathPattern: "/api/super-agent/harness/cleanup", Module: "super_agent", ResourceType: resourceTypeBot, Action: "cleanup_outputs", DescTemplate: "清理了超级体工具输出", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:conversation_id"},
	{Method: "POST", PathPattern: "/api/super-agent/sandbox/exec", Module: "super_agent", ResourceType: resourceTypeBot, Action: "run_bash", DescTemplate: "执行了超级体沙箱命令", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:command"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/mkdir", Module: "super_agent", ResourceType: resourceTypeBot, Action: "create_dir", DescTemplate: "创建了超级体沙箱目录", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/upload", Module: "super_agent", ResourceType: resourceTypeBot, Action: "upload_file", DescTemplate: "上传了超级体沙箱文件", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/write", Module: "super_agent", ResourceType: resourceTypeBot, Action: "write_file", DescTemplate: "写入了超级体沙箱文件", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/download", Module: "super_agent", ResourceType: resourceTypeBot, Action: "download_file", DescTemplate: "下载了超级体沙箱文件", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/patch", Module: "super_agent", ResourceType: resourceTypeBot, Action: "patch_file", DescTemplate: "应用了超级体沙箱补丁", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:workdir"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/edit", Module: "super_agent", ResourceType: resourceTypeBot, Action: "edit_file", DescTemplate: "编辑了超级体沙箱文件", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/move", Module: "super_agent", ResourceType: resourceTypeBot, Action: "move_file", DescTemplate: "移动了超级体沙箱文件", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:target_path"},
	{Method: "POST", PathPattern: "/api/super-agent/workspace/delete", Module: "super_agent", ResourceType: resourceTypeBot, Action: "delete_file", DescTemplate: "删除了超级体沙箱文件", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/artifacts/download", Module: "super_agent", ResourceType: resourceTypeBot, Action: "download_artifact", DescTemplate: "下载了超级体产物", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},
	{Method: "POST", PathPattern: "/api/super-agent/artifacts/move", Module: "super_agent", ResourceType: resourceTypeBot, Action: "move_artifact", DescTemplate: "移动了超级体产物", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:target_path"},
	{Method: "POST", PathPattern: "/api/super-agent/artifacts/delete", Module: "super_agent", ResourceType: resourceTypeBot, Action: "delete_artifact", DescTemplate: "删除了超级体产物", ResourceIDFrom: "body:agent_id", ResourceNameFrom: "body:path"},

	// ---- hi-agent（/api/space/:space_id/hi-agents，见 api/router/ynet_agent/ynet_agent.go）----
	{Method: "POST", PathPattern: "/api/space/:space_id/hi-agents", Module: "agent", ResourceType: resourceTypeBot, Action: "create", DescTemplate: "创建了智能体", ResourceNameFrom: "body:name"},
	{Method: "PUT", PathPattern: "/api/space/:space_id/hi-agents/:agent_id", Module: "agent", ResourceType: resourceTypeBot, Action: "update", DescTemplate: "更新了智能体", ResourceIDFrom: "path:agent_id", ResourceNameFrom: "body:name"},
	{Method: "DELETE", PathPattern: "/api/space/:space_id/hi-agents/:agent_id", Module: "agent", ResourceType: resourceTypeBot, Action: "delete", DescTemplate: "删除了智能体", ResourceIDFrom: "path:agent_id"},

	// ---- knowledge（/api/knowledge，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/knowledge/create", Module: "knowledge", ResourceType: resourceTypeKnowledge, Action: "create", DescTemplate: "创建了知识库", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/knowledge/update", Module: "knowledge", ResourceType: resourceTypeKnowledge, Action: "update", DescTemplate: "更新了知识库", ResourceIDFrom: "body:dataset_id", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/knowledge/delete", Module: "knowledge", ResourceType: resourceTypeKnowledge, Action: "delete", DescTemplate: "删除了知识库", ResourceIDFrom: "body:dataset_id"},
	{Method: "POST", PathPattern: "/api/knowledge/document/create", Module: "knowledge", ResourceType: resourceTypeKnowledge, Action: "create", DescTemplate: "创建了知识库文档", ResourceIDFrom: "body:dataset_id"},
	{Method: "POST", PathPattern: "/api/knowledge/document/delete", Module: "knowledge", ResourceType: resourceTypeKnowledge, Action: "delete", DescTemplate: "删除了知识库文档"},
	{Method: "POST", PathPattern: "/api/knowledge/document/update", Module: "knowledge", ResourceType: resourceTypeKnowledge, Action: "update", DescTemplate: "更新了知识库文档"},

	// ---- plugin（/api/plugin_api，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/plugin_api/register_plugin_meta", Module: "plugin", ResourceType: resourceTypePlugin, Action: "create", DescTemplate: "创建了插件", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/plugin_api/update", Module: "plugin", ResourceType: resourceTypePlugin, Action: "update", DescTemplate: "更新了插件", ResourceIDFrom: "body:plugin_id"},
	{Method: "POST", PathPattern: "/api/plugin_api/del_plugin", Module: "plugin", ResourceType: resourceTypePlugin, Action: "delete", DescTemplate: "删除了插件", ResourceIDFrom: "body:plugin_id"},
	{Method: "POST", PathPattern: "/api/plugin_api/publish_plugin", Module: "plugin", ResourceType: resourceTypePlugin, Action: "publish", DescTemplate: "发布了插件", ResourceIDFrom: "body:plugin_id"},

	// ---- database（/api/memory/database，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/memory/database/add", Module: "database", ResourceType: resourceTypeDatabase, Action: "create", DescTemplate: "创建了数据库表", ResourceNameFrom: "body:table_name"},
	{Method: "POST", PathPattern: "/api/memory/database/update", Module: "database", ResourceType: resourceTypeDatabase, Action: "update", DescTemplate: "更新了数据库表", ResourceIDFrom: "body:id", ResourceNameFrom: "body:table_name"},
	{Method: "POST", PathPattern: "/api/memory/database/delete", Module: "database", ResourceType: resourceTypeDatabase, Action: "delete", DescTemplate: "删除了数据库表", ResourceIDFrom: "body:id"},
	{Method: "POST", PathPattern: "/api/memory/database/update_records", Module: "database", ResourceType: resourceTypeDatabase, Action: "update", DescTemplate: "更新了数据库表记录"},

	// ---- variable / memory（/api/memory/variable，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/memory/variable/upsert", Module: "variable", ResourceType: resourceTypeApp, Action: "update", DescTemplate: "更新了变量"},
	{Method: "POST", PathPattern: "/api/memory/variable/delete", Module: "variable", ResourceType: resourceTypeApp, Action: "delete", DescTemplate: "删除了变量"},

	// ---- prompt（/api/playground_api，见 api/router/coze/api.go）----
	{Method: "POST", PathPattern: "/api/playground_api/upsert_prompt_resource", Module: "prompt", ResourceType: resourceTypePrompt, Action: "update", DescTemplate: "保存了提示词", ResourceIDFrom: "body:prompt.id", ResourceNameFrom: "body:prompt.name"},
	{Method: "POST", PathPattern: "/api/playground_api/delete_prompt_resource", Module: "prompt", ResourceType: resourceTypePrompt, Action: "delete", DescTemplate: "删除了提示词", ResourceIDFrom: "body:prompt_resource_id"},

	// ---- space member（/api/space/:space_id/members，见 api/router/space_member/space_member.go）----
	{Method: "POST", PathPattern: "/api/space/:space_id/members", Module: "space_member", ResourceType: resourceTypeWorkspace, Action: "invite", DescTemplate: "邀请了空间成员"},
	{Method: "DELETE", PathPattern: "/api/space/:space_id/members/:user_id", Module: "space_member", ResourceType: resourceTypeWorkspace, Action: "remove", DescTemplate: "移除了空间成员", ResourceIDFrom: "path:user_id"},
	{Method: "PUT", PathPattern: "/api/space/:space_id/members/:user_id", Module: "space_member", ResourceType: resourceTypeWorkspace, Action: "update_role", DescTemplate: "修改了空间成员角色", ResourceIDFrom: "path:user_id"},

	// ---- space（/api/space，见 api/router/space/space_management.go）----
	{Method: "POST", PathPattern: "/api/space/create", Module: "space", ResourceType: resourceTypeWorkspace, Action: "create", DescTemplate: "创建了空间", ResourceNameFrom: "body:name"},
	{Method: "PUT", PathPattern: "/api/space/:space_id", Module: "space", ResourceType: resourceTypeWorkspace, Action: "update", DescTemplate: "更新了空间", ResourceIDFrom: "path:space_id", ResourceNameFrom: "body:name"},
	{Method: "DELETE", PathPattern: "/api/space/:space_id", Module: "space", ResourceType: resourceTypeWorkspace, Action: "delete", DescTemplate: "删除了空间", ResourceIDFrom: "path:space_id"},
	{Method: "POST", PathPattern: "/api/space/:space_id/transfer", Module: "space", ResourceType: resourceTypeWorkspace, Action: "transfer", DescTemplate: "转让了空间", ResourceIDFrom: "path:space_id"},
}

// Match 返回命中的规则（method 大写比较），未命中返回 nil。
func Match(method, path string) *RouteRule {
	method = strings.ToUpper(method)
	for i := range rules {
		r := &rules[i]
		if r.Method == method && matchPath(r.PathPattern, path) {
			return r
		}
	}
	return nil
}

// matchPath 支持以 ':' 开头的通配段。
func matchPath(pattern, path string) bool {
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	xs := strings.Split(strings.Trim(path, "/"), "/")
	if len(ps) != len(xs) {
		return false
	}
	for i := range ps {
		if strings.HasPrefix(ps[i], ":") {
			continue
		}
		if ps[i] != xs[i] {
			return false
		}
	}
	return true
}

// PathSegs 按 rule 的 PathPattern 把 path 的通配段抽成 map（key 去掉前导 ':'）。
func PathSegs(pattern, path string) map[string]string {
	out := map[string]string{}
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	xs := strings.Split(strings.Trim(path, "/"), "/")
	if len(ps) != len(xs) {
		return out
	}
	for i := range ps {
		if strings.HasPrefix(ps[i], ":") {
			out[ps[i][1:]] = xs[i]
		}
	}
	return out
}

// ExtractSpaceID 依次从 query / form / body json 中取 space_id（字符串数字均兼容）。
func ExtractSpaceID(query map[string]string, form map[string]string, bodyJSON map[string]any) int64 {
	if v, ok := query["space_id"]; ok {
		if n := parseInt64(v); n > 0 {
			return n
		}
	}
	if v, ok := form["space_id"]; ok {
		if n := parseInt64(v); n > 0 {
			return n
		}
	}
	if v, ok := bodyJSON["space_id"]; ok {
		return anyToInt64(v)
	}
	return 0
}

// ExtractBySpec 按 "body:field"/"query:field"/"form:field"/"path:seg" 取值；返回字符串。
func ExtractBySpec(spec string, query, form map[string]string, bodyJSON map[string]any, pathSegs map[string]string) string {
	if spec == "" {
		return ""
	}
	kind, key, ok := splitSpec(spec)
	if !ok {
		return ""
	}
	switch kind {
	case "query":
		return query[key]
	case "form":
		return form[key]
	case "path":
		return pathSegs[key]
	case "body":
		if v, ok := lookupBody(bodyJSON, key); ok {
			return anyToStr(v)
		}
	}
	return ""
}

// lookupBody 支持点号嵌套路径，如 "prompt.id"：按 "." 分段在嵌套
// map[string]any 中逐层下钻，中间层必须是 map[string]any。
func lookupBody(bodyJSON map[string]any, key string) (any, bool) {
	if !strings.Contains(key, ".") {
		v, ok := bodyJSON[key]
		return v, ok
	}
	segs := strings.Split(key, ".")
	var cur any = bodyJSON
	for _, seg := range segs {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func splitSpec(spec string) (kind, key string, ok bool) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseInt64(s string) int64 {
	var n int64
	if s == "" {
		return 0
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

// ParseInt64Public 是 parseInt64 的导出版，供中间件包使用。
func ParseInt64Public(s string) int64 {
	return parseInt64(s)
}

func anyToInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		return parseInt64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	}
	return 0
}

func anyToStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	}
	return ""
}
