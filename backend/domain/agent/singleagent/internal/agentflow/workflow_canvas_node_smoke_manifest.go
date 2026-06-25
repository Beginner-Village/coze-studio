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

import "strings"

const (
	wfCanvasSmokeSupportSingleton         = "singleton"
	wfCanvasSmokeSupportFull              = "full"
	wfCanvasSmokeSupportResourceBound     = "resource-bound"
	wfCanvasSmokeSupportPartial           = "partial"
	wfCanvasSmokeSupportAddOnly           = "add-only"
	wfCanvasSmokeSupportDocumentationOnly = "documentation-only"
)

type wfCanvasSmokeCapability struct {
	Type             string
	Name             string
	SupportLevel     string
	CanAdd           bool
	RequiresResource bool
	RuntimeSmoke     string
	Gaps             []string
}

type wfCanvasNodeSmokeManifest struct {
	Status      string                           `json:"status"`
	Total       int                              `json:"total"`
	Summary     wfCanvasNodeSmokeManifestSummary `json:"summary"`
	Nodes       []wfCanvasNodeSmokeManifestNode  `json:"nodes"`
	Instruction string                           `json:"instruction"`
}

type wfCanvasNodeSmokeManifestSummary struct {
	Ready                          int `json:"ready"`
	NotReady                       int `json:"not_ready"`
	Skipped                        int `json:"skipped"`
	Execute                        int `json:"execute"`
	Readonly                       int `json:"readonly"`
	RequiresTemporaryWorkflowOptIn int `json:"requires_temporary_workflow_opt_in"`
	RequiresResourceFixture        int `json:"requires_resource_fixture"`
}

type wfCanvasNodeSmokeManifestNode struct {
	Type                           string   `json:"type"`
	Name                           string   `json:"name"`
	Status                         string   `json:"status"`
	RuntimeSmoke                   string   `json:"runtime_smoke"`
	Mode                           string   `json:"mode"`
	Isolation                      string   `json:"isolation"`
	Ready                          bool     `json:"ready"`
	RequiredTools                  []string `json:"required_tools,omitempty"`
	PlannedTools                   []string `json:"planned_tools,omitempty"`
	CleanupTools                   []string `json:"cleanup_tools,omitempty"`
	TemporaryNodeTags              []string `json:"temporary_node_tags,omitempty"`
	RequiresTemporaryWorkflowOptIn bool     `json:"requires_temporary_workflow_opt_in,omitempty"`
	RequiresResourceFixture        bool     `json:"requires_resource_fixture,omitempty"`
	SkipReason                     string   `json:"skip_reason,omitempty"`
	Gaps                           []string `json:"gaps,omitempty"`
	Assertions                     []string `json:"assertions,omitempty"`
	ExpectedBindableVariables      []string `json:"expected_bindable_variables,omitempty"`
}

var wfCanvasSmokeCapabilities = []wfCanvasSmokeCapability{
	{Type: "1", Name: "开始", SupportLevel: wfCanvasSmokeSupportSingleton, CanAdd: false, RuntimeSmoke: "not-executable", Gaps: []string{"只能引用 start/100001,不能新增。"}},
	{Type: "2", Name: "结束", SupportLevel: wfCanvasSmokeSupportSingleton, CanAdd: false, RuntimeSmoke: "not-executable"},
	{Type: "3", Name: "大模型", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "4", Name: "插件/API", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "5", Name: "代码", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "6", Name: "知识库检索", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "8", Name: "条件分支", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "9", Name: "子工作流", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少子工作流 schema 发现与语义配置器。"}},
	{Type: "11", Name: "变量", SupportLevel: wfCanvasSmokeSupportAddOnly, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"缺少变量节点语义配置器。"}},
	{Type: "12", Name: "SQL自定义", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少 databaseInfo/sql/inputParameters 语义配置器。"}},
	{Type: "13", Name: "输出/纯输出", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "14", Name: "图像流", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图像工作流资源发现与语义配置器。"}},
	{Type: "15", Name: "文本处理", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "16", Name: "生成图片", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图片生成参数语义配置器。"}},
	{Type: "17", Name: "图片引用", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图片资源选择与绑定语义配置器。"}},
	{Type: "18", Name: "问答/追问", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "19", Name: "跳出循环", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"只能在循环子画布语义下可靠测试。"}},
	{Type: "20", Name: "变量赋值", SupportLevel: wfCanvasSmokeSupportAddOnly, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"缺少变量赋值语义配置器。"}},
	{Type: "21", Name: "循环", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"缺少循环数组、循环变量和子画布节点语义配置器。"}},
	{Type: "22", Name: "意图识别", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "resource", Gaps: []string{"原生意图节点语义配置仍需补齐。"}},
	{Type: "23", Name: "图片画布", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少图像画布配置器。"}},
	{Type: "26", Name: "长期记忆", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少记忆资源发现与配置器。"}},
	{Type: "27", Name: "知识库写入", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少写入策略和字段绑定语义配置器。"}},
	{Type: "28", Name: "批处理", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"缺少批处理数组、并发和子链路语义配置器。"}},
	{Type: "29", Name: "继续循环", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "sub-canvas", Gaps: []string{"只能在循环子画布语义下可靠测试。"}},
	{Type: "30", Name: "输入", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "31", Name: "注释", SupportLevel: wfCanvasSmokeSupportDocumentationOnly, CanAdd: true, RuntimeSmoke: "not-executable"},
	{Type: "32", Name: "变量聚合", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"配置前必须调用 workflow_canvas_get_bindable_variables;不要聚合 type=13 输出节点,固定文案分支先用 type=15 文本处理产出真实 output。"}},
	{Type: "34", Name: "触发器更新", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少触发器 schema 配置器。"}},
	{Type: "35", Name: "触发器删除", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少触发器 schema 配置器。"}},
	{Type: "36", Name: "触发器读取", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少触发器 schema 配置器。"}},
	{Type: "42", Name: "更新数据", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "43", Name: "查询数据", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "44", Name: "删除数据", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "45", Name: "HTTP请求", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "resource", Gaps: []string{"缺少 method/url/headers/query/body/outputs 语义配置器。"}},
	{Type: "46", Name: "新增数据", SupportLevel: wfCanvasSmokeSupportResourceBound, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少数据库表字段和条件绑定配置器。"}},
	{Type: "58", Name: "JSON序列化", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RuntimeSmoke: "local"},
	{Type: "59", Name: "JSON解析", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RuntimeSmoke: "local", Gaps: []string{"缺少 schema/paths/outputs 语义配置器。"}},
	{Type: "61", Name: "MCP", SupportLevel: wfCanvasSmokeSupportPartial, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource", Gaps: []string{"缺少 MCP server/tool 发现和参数绑定配置器。"}},
	{Type: "99", Name: "卡片选择", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
	{Type: "100", Name: "智能体", SupportLevel: wfCanvasSmokeSupportFull, CanAdd: true, RequiresResource: true, RuntimeSmoke: "resource"},
}

func buildWFCanvasNodeSmokeManifest() wfCanvasNodeSmokeManifest {
	nodes := make([]wfCanvasNodeSmokeManifestNode, 0, len(wfCanvasSmokeCapabilities))
	for _, cap := range wfCanvasSmokeCapabilities {
		nodes = append(nodes, wfCanvasSmokeNodeFromCapability(cap))
	}
	return wfCanvasNodeSmokeManifest{
		Status:  "node_smoke_manifest",
		Total:   len(wfCanvasSmokeCapabilities),
		Summary: summarizeWFCanvasSmokeManifestNodes(nodes),
		Nodes:   nodes,
		Instruction: "复杂建图或节点巡检前先读本清单;execute 节点只应在临时 workflow 或用户明确允许的隔离环境中执行。" +
			" 资源型节点必须先准备真实资源 fixture;变量绑定必须通过 workflow_canvas_get_bindable_variables 确认,不要猜变量。完成 add/connect/configure 后执行 auto_layout 和上下文审计。",
	}
}

func wfCanvasSmokeNodeFromCapability(cap wfCanvasSmokeCapability) wfCanvasNodeSmokeManifestNode {
	switch {
	case cap.Type == "1":
		return wfCanvasAttachSmokeGuidance(wfCanvasNodeSmokeManifestNode{
			Type:          cap.Type,
			Name:          cap.Name,
			Status:        "verified-not-executable",
			RuntimeSmoke:  cap.RuntimeSmoke,
			Mode:          "readonly",
			Isolation:     "current-readonly",
			Ready:         true,
			RequiredTools: []string{"workflow_canvas_get_canvas_context"},
			PlannedTools:  []string{"workflow_canvas_get_canvas_context"},
			Gaps:          append([]string(nil), cap.Gaps...),
		})
	case cap.Type == "2":
		return wfCanvasLocalExecutableSmokeNode(cap, []string{"smoke_2_text_source"}, []string{"workflow_canvas_connect", "workflow_canvas_get_bindable_variables", "workflow_canvas_configure_node", "workflow_canvas_get_canvas_context", "workflow_canvas_test_run"})
	case wfCanvasIsLocalFullSmokeNode(cap):
		return wfCanvasLocalExecutableSmokeNode(cap, wfCanvasDefaultTemporaryNodeTags(cap.Type), wfCanvasDefaultRequiredTools(cap.Type))
	case cap.SupportLevel == wfCanvasSmokeSupportAddOnly || cap.SupportLevel == wfCanvasSmokeSupportDocumentationOnly:
		return wfCanvasAttachSmokeGuidance(wfCanvasNodeSmokeManifestNode{
			Type:                           cap.Type,
			Name:                           cap.Name,
			Status:                         wfCanvasSmokeStatusFromCapability(cap),
			RuntimeSmoke:                   cap.RuntimeSmoke,
			Mode:                           "execute",
			Isolation:                      "temporary-workflow",
			Ready:                          true,
			RequiredTools:                  []string{"workflow_canvas_add_node", "workflow_canvas_delete_node"},
			PlannedTools:                   []string{"workflow_canvas_add_node", "workflow_canvas_delete_node", "workflow_canvas_auto_layout"},
			CleanupTools:                   []string{"workflow_canvas_delete_node", "workflow_canvas_auto_layout"},
			TemporaryNodeTags:              []string{"smoke_" + cap.Type},
			RequiresTemporaryWorkflowOptIn: true,
			Gaps:                           append([]string(nil), cap.Gaps...),
		})
	case cap.RequiresResource || cap.RuntimeSmoke == "resource":
		return wfCanvasAttachSmokeGuidance(wfCanvasNodeSmokeManifestNode{
			Type:                    cap.Type,
			Name:                    cap.Name,
			Status:                  wfCanvasSmokeStatusFromCapability(cap),
			RuntimeSmoke:            cap.RuntimeSmoke,
			Mode:                    "skip",
			Isolation:               "resource-fixture",
			Ready:                   false,
			RequiresResourceFixture: true,
			SkipReason:              wfCanvasSmokeResourceSkipReason(cap),
			Gaps:                    append([]string(nil), cap.Gaps...),
		})
	default:
		return wfCanvasAttachSmokeGuidance(wfCanvasNodeSmokeManifestNode{
			Type:         cap.Type,
			Name:         cap.Name,
			Status:       "unsupported",
			RuntimeSmoke: cap.RuntimeSmoke,
			Mode:         "skip",
			Isolation:    "sub-canvas",
			Ready:        false,
			SkipReason:   wfCanvasSmokeGapSkipReason(cap),
			Gaps:         append([]string(nil), cap.Gaps...),
		})
	}
}

func wfCanvasLocalExecutableSmokeNode(cap wfCanvasSmokeCapability, temporaryNodeTags []string, requiredTools []string) wfCanvasNodeSmokeManifestNode {
	return wfCanvasAttachSmokeGuidance(wfCanvasNodeSmokeManifestNode{
		Type:                           cap.Type,
		Name:                           cap.Name,
		Status:                         wfCanvasSmokeStatusFromCapability(cap),
		RuntimeSmoke:                   cap.RuntimeSmoke,
		Mode:                           "execute",
		Isolation:                      "temporary-workflow",
		Ready:                          true,
		RequiredTools:                  requiredTools,
		PlannedTools:                   wfCanvasPlannedToolsForRequired(requiredTools),
		CleanupTools:                   wfCanvasCleanupToolsForTemporaryNodes(len(temporaryNodeTags)),
		TemporaryNodeTags:              temporaryNodeTags,
		RequiresTemporaryWorkflowOptIn: true,
		Gaps:                           append([]string(nil), cap.Gaps...),
	})
}

func wfCanvasAttachSmokeGuidance(node wfCanvasNodeSmokeManifestNode) wfCanvasNodeSmokeManifestNode {
	node.Assertions = wfCanvasSmokeAssertions(node.Type)
	node.ExpectedBindableVariables = wfCanvasSmokeExpectedBindableVariables(node.Type)
	return node
}

func wfCanvasSmokeAssertions(nodeType string) []string {
	switch nodeType {
	case "2":
		return []string{
			"End 可以返回变量: returns 必须绑定真实上游输出,不能为空。",
			"End 可以返回文本",
			"End 可以返回文本: input/inputs 绑定多个上游变量,content 用 {{变量名}} 拼接,streaming_output=true 时流式透传。",
			"streaming_output=true",
			"配置 End 前先调用 workflow_canvas_get_bindable_variables,不要绑定 display-only 输出节点。",
		}
	case "5":
		return []string{
			"Python 代码节点使用 async def main(args: Args),输入从 args.params 读取。",
			"不要对 args 直接调用 strip/get 等字符串或 dict 方法。",
			"需要下游绑定时必须声明 outputs。",
		}
	case "8":
		return []string{
			"条件左值必须来自 workflow_canvas_get_bindable_variables 返回的可绑定变量。",
			"true/false 分支端口必须明确连线,修改条件后重新读取画布上下文确认。",
		}
	case "13":
		return []string{
			"display-only",
			"输出/纯输出节点是 display-only 消息展示节点。",
			"不要作为 VariableMerge 或 End returns 的稳定变量来源。",
			"如果固定文案需要被下游消费,先用 type=15 文本处理产出真实 output。",
		}
	case "15":
		return []string{
			"固定文案没有变量引用时可以删除默认 input。",
			"需要被下游消费时必须声明 output:string。",
			"content 里使用 {{变量名}} 时,必须先在本节点 input/inputs 中绑定同名变量。",
		}
	case "18":
		return []string{
			"问答节点用于缺槽追问,问题文本可以引用本节点已绑定 input。",
			"需要继续向下游传递时声明 answer 或 output。",
		}
	case "30":
		return []string{
			"输入节点用于显式收集用户补充信息,字段名必须和后续绑定变量一致。",
			"新增输入字段后重新读取可绑定变量再配置下游。",
		}
	case "32":
		return []string{
			"变量聚合必须先调用 workflow_canvas_get_bindable_variables,merge_groups.variables 全部来自可绑定变量。",
			"不要聚合 type=13 输出节点;固定文案分支先用 type=15 文本处理产出真实 output。",
			"End returns 应绑定 VariableMerge 的真实 output,不要直接猜旧节点变量。",
		}
	case "58":
		return []string{
			"JSON 序列化输入必须绑定真实变量或固定对象字段。",
			"需要下游消费时声明 output:string。",
		}
	default:
		return nil
	}
}

func wfCanvasSmokeExpectedBindableVariables(nodeType string) []string {
	switch nodeType {
	case "2":
		return []string{"end.text"}
	case "3":
		return []string{"llm.answer"}
	case "5":
		return []string{"code.output"}
	case "8":
		return []string{"branch.condition"}
	case "13":
		return []string{}
	case "15":
		return []string{"text.output"}
	case "18":
		return []string{"question.answer"}
	case "30":
		return []string{"input.value"}
	case "32":
		return []string{"merge.output"}
	case "58":
		return []string{"json.output"}
	default:
		return nil
	}
}

func wfCanvasIsLocalFullSmokeNode(cap wfCanvasSmokeCapability) bool {
	return cap.RuntimeSmoke == "local" && cap.SupportLevel == wfCanvasSmokeSupportFull
}

func wfCanvasDefaultRequiredTools(nodeType string) []string {
	switch nodeType {
	case "13":
		return []string{"workflow_canvas_add_node", "workflow_canvas_connect", "workflow_canvas_configure_node", "workflow_canvas_get_canvas_context"}
	case "30":
		return []string{"workflow_canvas_add_node", "workflow_canvas_configure_node", "workflow_canvas_get_canvas_context"}
	default:
		return []string{"workflow_canvas_add_node", "workflow_canvas_connect", "workflow_canvas_get_bindable_variables", "workflow_canvas_configure_node", "workflow_canvas_get_canvas_context", "workflow_canvas_test_run"}
	}
}

func wfCanvasPlannedToolsForRequired(required []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(required)+1)
	for _, command := range required {
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		out = append(out, command)
	}
	if _, ok := seen["workflow_canvas_auto_layout"]; !ok {
		out = append(out, "workflow_canvas_auto_layout")
	}
	return out
}

func wfCanvasCleanupToolsForTemporaryNodes(count int) []string {
	out := make([]string, 0, count+1)
	for i := 0; i < count; i++ {
		out = append(out, "workflow_canvas_delete_node")
	}
	return append(out, "workflow_canvas_auto_layout")
}

func wfCanvasDefaultTemporaryNodeTags(nodeType string) []string {
	switch nodeType {
	case "32":
		return []string{"smoke_32_variable_merge", "smoke_32_text_primary", "smoke_32_text_fallback"}
	case "58":
		return []string{"smoke_58_json_stringify", "smoke_58_text_source"}
	default:
		return []string{"smoke_" + nodeType}
	}
}

func wfCanvasSmokeStatusFromCapability(cap wfCanvasSmokeCapability) string {
	switch cap.SupportLevel {
	case wfCanvasSmokeSupportFull:
		return "verified-full"
	case wfCanvasSmokeSupportResourceBound:
		return "verified-resource-bound"
	case wfCanvasSmokeSupportAddOnly:
		return "verified-add-only"
	case wfCanvasSmokeSupportSingleton, wfCanvasSmokeSupportDocumentationOnly:
		return "verified-not-executable"
	default:
		return "unsupported"
	}
}

func wfCanvasSmokeResourceSkipReason(cap wfCanvasSmokeCapability) string {
	if len(cap.Gaps) > 0 {
		return cap.Gaps[0]
	}
	return "运行依赖空间资源 fixture。"
}

func wfCanvasSmokeGapSkipReason(cap wfCanvasSmokeCapability) string {
	if len(cap.Gaps) > 0 {
		return cap.Gaps[0]
	}
	return "当前节点 smoke 仍需补齐场景或子画布环境。"
}

func summarizeWFCanvasSmokeManifestNodes(nodes []wfCanvasNodeSmokeManifestNode) wfCanvasNodeSmokeManifestSummary {
	var summary wfCanvasNodeSmokeManifestSummary
	for _, node := range nodes {
		if node.Ready {
			summary.Ready++
		} else {
			summary.NotReady++
		}
		switch strings.TrimSpace(node.Mode) {
		case "execute":
			summary.Execute++
		case "readonly":
			summary.Readonly++
		case "skip":
			summary.Skipped++
		}
		if node.RequiresTemporaryWorkflowOptIn {
			summary.RequiresTemporaryWorkflowOptIn++
		}
		if node.RequiresResourceFixture {
			summary.RequiresResourceFixture++
		}
	}
	return summary
}
