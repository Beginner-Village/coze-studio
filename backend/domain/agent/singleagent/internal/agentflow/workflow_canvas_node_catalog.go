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

const wfNodeCatalogText = `节点目录(渐进式设计):
- 3 大模型(LLM): 文本理解、生成、分类、抽取、总结。使用前调用 get_node_spec(type=3)。
- 2 结束(End): 内置单例,不能新增;支持返回变量和返回文本,返回文本可流式透传多个上游变量。配置前调用 get_node_spec(type=2)。
- 4 插件/API: 调用空间插件/API/工具。使用前调用 get_node_spec(type=4),并只使用资源清单里的 plugin_id/api_id。
- 5 代码: 确定性计算、格式转换、字段清洗。使用前调用 get_node_spec(type=5)。
- 6 知识库检索: RAG/知识召回。使用前调用 get_node_spec(type=6),并只使用资源清单里的 dataset_id。
- 8 条件分支(IF): 按变量条件分 true/false。使用前调用 get_node_spec(type=8)。
- 9 子工作流: 复用已有工作流。使用前确认 workflow_id 和输入输出 schema。
- 11 变量 / 20 设置变量 / 32 变量聚合: 中间变量、赋值、分支结果合并;分支汇合后 End 应绑定变量聚合输出。
- 13 输出/纯输出: 不调用模型,直接展示固定文本或上游变量给用户;不要作为变量聚合或 End returns 的上游变量来源。使用前调用 get_node_spec(type=13)。
- 15 文本处理: 拼接模板文本或按分隔符拆分字符串,会产出真实 output 变量,适合分支固定回复后接变量聚合。使用前调用 get_node_spec(type=15)。
- 18 问答/追问: 缺少账号、卡号等槽位时向用户追问。使用前调用 get_node_spec(type=18)。
- 21 循环 / 28 批处理 / 19 Break / 29 Continue: 数组遍历、批量处理和循环控制。
- 22 意图识别: 对用户输入或文本做分类路由。
- 27 知识库写入: 将内容写入知识库,需要真实 dataset_id 和写入策略。
- 30 输入: 声明 Start.input 以外的结构化工作流入参。使用前调用 get_node_spec(type=30)。
- 42 更新数据 / 43 查询数据 / 44 删除数据 / 46 新增数据 / 12 SQL自定义: 数据库 CRUD,必须先知道真实数据表和字段。
- 45 HTTP: 直接调用 HTTP 接口。
- 58 JSON序列化 / 59 JSON解析: 对对象和 JSON 字符串做转换。使用前调用 get_node_spec(type=58/59)。
- 61 MCP: 调用 MCP server/tool。
- 99 卡片选择: 输出卡片并等待用户选择,需要真实卡片资源。
- 100 智能体: 调用子智能体/HiAgent/Coze Agent。使用前调用 get_node_spec(type=100),并只使用资源清单里的 agent_id。
设计流程: 先选最小完整节点组合;每个执行节点都要设计 input bindings、核心配置、outputs、下游消费;Start/End 是内置单例只能引用不能新增。试运行失败时优先局部修复输入绑定、变量引用、outputs、merge_groups 或 End returns,不要默认清空画布。`

var wfNodeSpecByType = map[string]string{
	"3": `节点规格: 3 大模型(LLM)
适用: 文本理解、生成、总结、抽取、分类、路由前判断。
可绑定输入: input/inputs,每项形如 {from:"start或上游node_tag",output:"变量名",name:"本节点入参名"}。
必须配置: prompt 或 user_prompt;建议配置 system_prompt;必须声明 outputs,至少 [{name:"answer",type:"string"}] 或 [{name:"output",type:"string"}]。
下游绑定: 下游引用本节点输出用 {from:"llm_node_tag",output:"answer"}。
configure_node 示例:
{"node_tag":"llm","config":{"title":"大模型处理","input":{"from":"start","output":"input","name":"input"},"system_prompt":"你是严谨的处理节点","prompt":"请处理 {{input}}","outputs":[{"name":"answer","type":"string"}]}}`,

	"4": `节点规格: 4 插件/API
适用: 查询订单、调用外部服务、执行已有插件能力。
可绑定输入: inputs 对象, key 是 API 参数名, value 是变量绑定 {from,output}。
必须配置: plugin_id、api_id、plugin_name/api_name 必须来自当前空间资源清单,禁止编造。
输出: 若资源清单中没有明确 API 输出,先声明 result:string 或按接口语义声明字段。
configure_node 示例:
{"node_tag":"api_order","config":{"title":"查询订单API","plugin_id":"真实plugin_id","api_id":"真实api_id","plugin_name":"订单插件","api_name":"查询订单","inputs":{"order_id":{"from":"start","output":"input"}},"outputs":[{"name":"result","type":"string"}]}}`,

	"5": `节点规格: 5 代码
适用: 确定性逻辑、JSON/字段转换、正则提取、简单计算,不要用它做主观生成。
可绑定输入: input/inputs,每项 {from,output,name};name 会成为代码入参字段。
必须配置: language、code、outputs。当前运行时只支持 Python,language 写 "python";禁止 JavaScript/async function 语法。
代码规范: 必须定义 async def main(args: Args) -> Output,先用 params = args.params or {} 取得全部入参,再用 params.get("入参名") 读取 input 绑定的字段;不要对 args 直接调用 strip/get/[]。
返回规范: return 一个 dict/Output,字段名必须和 outputs 名称一致。字符串处理时先取字段再转字符串,例如 query = str(params.get("query") or "").strip()。
configure_node 示例:
{"node_tag":"normalize","config":{"title":"格式化输入","input":{"from":"start","output":"input","name":"query"},"language":"python","code":"async def main(args: Args) -> Output:\n    params = args.params or {}\n    query = str(params.get('query') or '').strip()\n    return {'result': query}","outputs":[{"name":"result","type":"string"}]}}`,

	"6": `节点规格: 6 知识库检索
适用: 从文档/表格知识库召回内容,供 LLM 基于资料回答。
可绑定输入: input 绑定 Query,通常 {from:"start",output:"input",name:"Query"} 或来自上游改写后的 query。
必须配置: dataset_id 或 dataset_ids,必须来自当前空间资源清单;top_k 可选。
默认输出: outputList:array_object,子字段 output:string。
configure_node 示例:
{"node_tag":"kb","config":{"title":"检索知识库","dataset_ids":["真实dataset_id"],"input":{"from":"start","output":"input","name":"Query"},"top_k":5}}`,

	"8": `节点规格: 8 条件分支(IF)
适用: 基于上游变量做路由,例如意图=售后、分数>阈值、结果为空。
可绑定输入: condition.left 必须绑定变量 {from,output};right 可为字面量或变量绑定。
操作符: equal/not_equal/contains/not_contains/null/not_null/gt/gte/lt/lte/true/false。
连线: true 分支 workflow_canvas_connect from_port="true";false 分支 from_port="false"。
分支汇合: 如果 true/false 分支产出不同节点变量,添加 type=32 变量聚合节点并配置 merge_groups,再让 End returns 绑定聚合节点输出。
configure_node 示例:
{"node_tag":"branch","config":{"title":"判断是否售后","condition":{"left":{"from":"intent","output":"category"},"operator":"equal","right":"售后"}}}`,

	"13": `节点规格: 13 输出/纯输出
适用: 不需要大模型推理时,直接把固定文案或上游变量展示给用户;也适合把“大模型节点改成纯输出节点”作为消息展示。
可绑定输入: input/inputs,每项 {from:"上游node_tag",output:"变量名",name:"模板变量名"};这些 name 可在 content 里用 {{变量名}}。
必须配置: content/text/template 三选一;可选 streaming_output=true/false。内部会写入 inputParameters 和 content;content 只能引用本节点 input/inputs 里定义的 name,例如 input name 是 reply 时只能写 {{reply}},不要写上游原变量名。
重要限制: type=13 是消息展示节点,不是稳定的下游变量来源;不要让变量聚合或 End returns 引用它的 output。如果一个分支要返回固定文案或模板文案,使用 type=15 文本处理节点产出 output:string,再把文本处理 output 接入变量聚合/End。
完成标准: get_canvas_context 中该节点应显示 inputs: reply=上游.output 且 content: ...{{reply}};若后续还要给其他节点消费,改用 type=15 文本处理。
常见错误: BlockID is empty 或引用变量不存在时,不是清空画布,而是重新 configure_node 补 input/inputs,并确认 content 只引用这些 input name。
替换 LLM 的推荐步骤: 如果只是展示消息,add_node type=13 并配置 content;如果替换后还要被聚合/End 返回,请 add_node type=15 文本处理而不是 type=13。
configure_node 示例:
{"node_tag":"direct_output","config":{"title":"直接输出结果","input":{"from":"normalize","output":"result","name":"result"},"content":"处理结果: {{result}}","streaming_output":true}}`,

	"15": `节点规格: 15 文本处理
适用: 确定性文本拼接/模板渲染/字符串拆分,不要为固定文案使用 LLM。分支固定回复、模板回复需要被变量聚合或 End 返回时,优先用本节点,不要用 type=13 输出节点。
可绑定输入: input/inputs,每项 {from,output,name};concat 模式 content/template 中可引用 {{name}};split 模式绑定一个字符串并设置 delimiter。
必须配置: method="concat" 或 "split";concat 设置 content/text/template;split 设置 delimiter;声明 outputs,默认 output:string 或 output:array_string。content/template 只能引用本节点 input/inputs 的 name。
完成标准: get_canvas_context 中该节点应显示 inputs: reply=上游.output、content: ...{{reply}}、outputs: output:string;下游引用 {from:"format_text",output:"output"}。
常见错误: content/template 里用了 {{reply}} 但没有 input name="reply" 时会报引用变量不存在;用 configure_node 补 input,不要清空画布。
configure_node 示例:
{"node_tag":"format_text","config":{"title":"组装回复文案","method":"concat","input":{"from":"api","output":"result","name":"result"},"content":"查询结果: {{result}}","outputs":[{"name":"output","type":"string"}]}}`,

	"18": `节点规格: 18 问答/追问
适用: 缺少必要槽位时向用户追问,例如没有用户账号/缺少卡号/需要确认收款人。
可绑定输入: input/inputs 用于把上下文变量带入问题模板。
必须配置: question 或 content;answer_type="text" 或 "option";option 模式提供 options;limit 可限制次数;需要声明后续要消费的 outputs。
常见输出: USER_RESPONSE:string(用户输入), optionId:string, optionContent:string;也可用 outputs 声明额外抽取字段。
configure_node 示例:
{"node_tag":"ask_card","config":{"title":"追问收款卡号","question":"请提供收款卡号,或回复“最近收款人”。","answer_type":"text","outputs":[{"name":"USER_RESPONSE","type":"string"}]}}`,

	"30": `节点规格: 30 输入
适用: 声明工作流额外结构化入参,例如 user_id、account_id、channel。它不是开始节点,可新增;Start 仍然是内置单例。
必须配置: outputs,每项 {name,type,description};下游可绑定该输入节点的输出变量。
configure_node 示例:
{"node_tag":"user_input_schema","config":{"title":"结构化输入","outputs":[{"name":"user_id","type":"string","description":"登录用户ID"},{"name":"amount","type":"number","description":"转账金额"}]}}`,

	"32": `节点规格: 32 变量聚合
适用: 汇合 IF/循环/并行分支的多个可能输出,生成一个统一输出给 End 或下游节点。
可绑定输入: merge_groups,每组 {name,variables};name 会成为本节点输出变量名;variables 必须全部来自 workflow_canvas_get_canvas_context 的“可绑定变量”,例如 transfer_reply.output、chat_llm.answer。
重要限制: 不要聚合 type=13 输出/消息节点,它不是稳定的下游变量来源。如果分支结果是固定文案或模板文案,先在分支末尾添加 type=15 文本处理节点产出真实 output 变量,再把该 output 放入 merge_groups。
必须配置: 至少一个 merge_groups,每组至少一个变量;建议 name="output",这样 End returns 可绑定 {from:"merge",output:"output"}。如果有多个互斥分支最终都要返回,把每个分支末端节点的输出都放入同一个 output 组,不要只放一个变量。
完成标准: get_canvas_context 中该节点应显示 merge_groups: output=[branch1.output, branch2.output,...] 且 outputs: output:string;End 节点应显示 returns: output=merge.output。若显示 未定义/引用变量不存在,立即重新 get_canvas_context,只用可绑定变量修正 merge_groups。
常见错误: 聚合节点只连了线但 merge_groups 没有引用上游输出;必须 configure_node 写 variables,并让 End returns 绑定 merge.output。
configure_node 示例:
{"node_tag":"merge","config":{"title":"分支结果合并","merge_groups":[{"name":"output","variables":[{"from":"after_sale_code","output":"result"},{"from":"general_agent","output":"answer"}]}]}}`,

	"22": `节点规格: 22 意图识别
适用: 将用户输入分类成有限类别,再接 IF 或多分支处理。
配置建议: 绑定输入文本,定义候选意图/类别和输出 category/intent。若语义配置暂不完整,可用 LLM 节点模拟意图识别并声明 category 输出。
configure_node 示例(使用 LLM 替代更稳): 添加 type=3 LLM, prompt 要求只输出意图类别,outputs=[{name:"category",type:"string"}]。`,

	"45": `节点规格: 45 HTTP请求
适用: 调用 REST API。
支持边界: 当前为 partial,method/url/headers/query/body/auth/outputs 语义配置器仍需补齐。能明确字段时才可配置,不能把它当作已完整支持节点。
必须配置: method、url、headers/body/query 参数、outputs。参数值应绑定当前可用变量。
如果接口已封装成插件/API,优先使用 type=4 插件/API 节点。`,

	"58": `节点规格: 58 JSON序列化
适用: 将对象/数组变量转换为 JSON 字符串,用于 HTTP body、日志或输出。
可绑定输入: input/inputs,通常 name="input",绑定上游 object/array_object/string。
输出: 默认 output:string。
configure_node 示例:
{"node_tag":"json_encode","config":{"title":"序列化请求体","input":{"from":"build_body","output":"body","name":"input","type":"object"},"outputs":[{"name":"output","type":"string"}]}}`,

	"59": `节点规格: 59 JSON解析
适用: 将 JSON 字符串解析为对象或结构化字段。复杂 schema 不确定时优先用 type=5 Python 代码节点显式解析并声明 outputs。
可绑定输入: input/inputs,绑定 JSON 字符串。
配置建议: 声明 outputs 为下游要消费的结构化字段;若当前语义配置不完整,用代码节点替代更稳。`,

	"99": `节点规格: 99 卡片选择
适用: 展示卡片并等待用户选择,用于转账确认、收款人选择、套餐选择等交互。
可绑定输入: input/inputs 绑定卡片模板变量;content/template 可写卡片说明;selected_card/card_id 必须来自真实卡片资源,禁止编造。
configure_node 示例:
{"node_tag":"choose_payee","config":{"title":"选择最近收款人","card_id":"真实card_id","content":"请选择收款人","input":{"from":"recent_payees","output":"list","name":"payees"}}}`,

	"100": `节点规格: 100 智能体
适用: 把复杂子任务交给已有 HiAgent/Coze 子智能体。
可绑定输入: query/inputParameters,通常 {from:"start",output:"input",name:"query"} 或来自上游节点输出。
必须配置: agent_id、platform、agent_name 必须来自当前空间资源清单;声明 answer:string 等 outputs。
configure_node 示例:
{"node_tag":"agent","config":{"title":"调用客服智能体","agent_id":"真实agent_id","agent_name":"客服智能体","platform":"hiagent","input":{"from":"start","output":"input","name":"query"},"outputs":[{"name":"answer","type":"string"}]}}`,

	"2": `节点规格: 2 结束(End)
适用: 工作流最终返回。End 是内置单例,不能新增、不能删除;只能引用 node_tag="end" 或 id=900001 进行连线和配置。
返回方式:
1. 返回变量(returnVariables): config.returns 绑定一个或多个上游输出变量,例如 [{"name":"output","from":"merge","output":"output"}]。
2. 返回文本(useAnswerContent): config.input/config.inputs 先绑定多个上游输出为本节点模板变量,再用 content/text/template 组装最终回答;可设置 streaming_output=true 开启流式透传。
绑定规则: 返回文本 content 只能引用本节点 input/inputs 里定义的 name,例如 input name 是 answer 时写 {{answer}},不要写上游原 output 名。多个输出可以全部绑定到 inputParameters,再统一放进返回文本。
配置前置: 先 workflow_canvas_get_bindable_variables,只使用当前可绑定变量或本轮刚声明 outputs 的节点。若要汇合多个互斥分支,优先让 End 返回文本直接绑定各分支真实输出,或先用 type=32 变量聚合/每个分支 type=15 文本处理产出真实 output 后再返回。
完成标准: get_canvas_context 中 End 应显示 terminatePlan=returnVariables 且 returns/output 已绑定,或 terminatePlan=useAnswerContent 且 inputs/content/streamingOutput 已配置;不能显示变量值为空、未定义、引用变量不存在。
返回变量示例:
{"node_tag":"end","config":{"returns":[{"name":"output","from":"merge","output":"output"}]}}
返回文本流式示例:
{"node_tag":"end","config":{"input":[{"name":"intent","from":"intent","output":"category"},{"name":"answer","from":"merge","output":"output"}],"content":"业务类型: {{intent}}\n处理结果: {{answer}}","streaming_output":true}}`,
}

func wfNodeSpecForType(nodeType string) (string, bool) {
	if spec, ok := wfNodeSpecByType[nodeType]; ok {
		return spec, true
	}
	for _, cap := range wfCanvasSmokeCapabilities {
		if cap.Type == nodeType {
			return wfNodeSpecFallbackText(cap), true
		}
	}
	return "", false
}

func wfNodeSpecFallbackText(cap wfCanvasSmokeCapability) string {
	var lines []string
	lines = append(lines, "节点规格: "+cap.Type+" "+cap.Name)
	lines = append(lines, "支持边界: "+wfNodeSpecFallbackBoundary(cap))
	lines = append(lines, "建议工具: "+strings.Join(wfNodeSpecFallbackTools(cap), ", "))
	if rules := wfNodeSpecFallbackRules(cap); len(rules) > 0 {
		lines = append(lines, "绑定/配置规则:")
		for _, rule := range rules {
			lines = append(lines, "- "+rule)
		}
	}
	if len(cap.Gaps) > 0 {
		lines = append(lines, "当前缺口: "+strings.Join(cap.Gaps, "; "))
	}
	return strings.Join(lines, "\n")
}

func wfNodeSpecFallbackBoundary(cap wfCanvasSmokeCapability) string {
	switch cap.SupportLevel {
	case wfCanvasSmokeSupportSingleton:
		return "内置单例节点,只能引用已有节点,不能新增或删除。修改前先读取画布上下文,确认当前内置节点的 node_tag/id 和可绑定变量。"
	case wfCanvasSmokeSupportResourceBound:
		return "resource-bound 节点,配置依赖空间真实资源或资源 fixture。没有资源清单时只能 skipped-with-reason,不能编造 ID 或宣称完整可运行。"
	case wfCanvasSmokeSupportPartial:
		return "partial 节点,已登记能力但语义配置器不完整。可以在用户明确需要时添加节点或保留设计占位,但不能把它当作已完整支持的执行节点。"
	case wfCanvasSmokeSupportAddOnly:
		return "add-only 节点,只可靠支持添加、删除和布局,还没有稳定语义配置器。可以作为占位,不能宣称已完整配置业务逻辑。"
	case wfCanvasSmokeSupportDocumentationOnly:
		return "documentation-only 节点,用于说明画布或辅助人工阅读,不参与运行链路。"
	default:
		return "当前不具备完整自动化支持。使用前必须读取当前缺口并说明不可自动完成的原因。"
	}
}

func wfNodeSpecFallbackTools(cap wfCanvasSmokeCapability) []string {
	if !cap.CanAdd {
		return []string{"workflow_canvas_get_canvas_context", "workflow_canvas_get_bindable_variables"}
	}
	switch cap.SupportLevel {
	case wfCanvasSmokeSupportAddOnly, wfCanvasSmokeSupportDocumentationOnly:
		return []string{"workflow_canvas_add_node", "workflow_canvas_delete_node", "workflow_canvas_auto_layout"}
	case wfCanvasSmokeSupportResourceBound, wfCanvasSmokeSupportPartial:
		return []string{"workflow_canvas_get_canvas_context", "workflow_canvas_get_bindable_variables", "workflow_canvas_add_node", "workflow_canvas_configure_node", "workflow_canvas_get_canvas_context"}
	default:
		return []string{"workflow_canvas_get_canvas_context", "workflow_canvas_add_node", "workflow_canvas_configure_node", "workflow_canvas_get_canvas_context"}
	}
}

func wfNodeSpecFallbackRules(cap wfCanvasSmokeCapability) []string {
	switch cap.SupportLevel {
	case wfCanvasSmokeSupportSingleton:
		return []string{
			"不要调用 workflow_canvas_add_node 新增单例节点。",
			"连线或配置前先调用 workflow_canvas_get_canvas_context 确认现有节点。",
		}
	case wfCanvasSmokeSupportResourceBound:
		return []string{
			"先读取画布上下文、可绑定变量和资源清单,再选择真实资源。",
			"资源 ID、名称、schema 必须来自系统返回的资源清单或用户明确提供。",
			"缺少资源 fixture 时不要执行真实 test_run,应说明 requires_resource_fixture。",
		}
	case wfCanvasSmokeSupportPartial:
		return []string{
			"优先使用 full support 节点完成同等语义。",
			"如果必须使用该节点,只能在字段、资源和绑定变量都明确时配置。",
			"validate/test_run 失败时优先局部修复,不要默认清空画布。",
		}
	case wfCanvasSmokeSupportAddOnly:
		return []string{
			"可以调用 workflow_canvas_add_node 创建,用 workflow_canvas_delete_node 清理。",
			"需要下游消费变量时,先确认该节点是否已经在画布中暴露真实 outputs。",
		}
	case wfCanvasSmokeSupportDocumentationOnly:
		return []string{
			"不要把该节点作为业务执行节点。",
			"不要把它的内容作为 End returns 或变量聚合来源。",
		}
	default:
		return []string{
			"不能编造配置字段或资源 ID。",
			"需要人工或后续语义配置器补齐后才能升级支持等级。",
		}
	}
}
