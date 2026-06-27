---
name: finmall-workflow-builder
description: 在工作流编辑器(FlowGram)里像人一样自动搭建工作流——一个个加节点、一条条连线、跑调试。finmallclaw 产出画布指令脚本(AgentCanvasCommand[]),经内嵌面板的指令总线 WorkflowAgentCommandService 逐条动态执行,用户实时看到节点出现、连线生成。当用户要求"用智能体搭一个工作流/加节点/连节点/自动编排"时使用。
metadata:
  type: ui-automation
  target: workflow playground (FlowGram free-layout)
  bridge: window.__finmallclawWorkflow / WorkflowAgentCommandService
---

# finmall 工作流自动搭建技能（渐进式）

finmallclaw 不手画 canvas JSON、不手点界面,而是产出一段**画布指令脚本**,由编辑器内嵌的
指令总线逐条执行:节点一个个冒出来、线一条条连上(动态可见,不刷新)。最后触发 test_run 完整调试。

## 执行通道
指令脚本 = `AgentCanvasCommand[]`,通过以下之一进入画布执行:
- 编辑器右下角 finmallclaw 面板的"执行指令"框;
- `window.__finmallclawWorkflow.runScript(commands)` 桥(内嵌对话驱动用)。
底层服务:`WorkflowAgentCommandService`(playground/src/services/workflow-agent-command-service.ts)。

## 指令格式(AgentCanvasCommand)
```jsonc
[
  { "op": "addNode", "tag": "start", "args": { "type": "1" } },          // 加节点,tag=逻辑名
  { "op": "addNode", "tag": "llm",   "args": { "type": "3",
      "nodeJson": { "data": { "nodeMeta": { "title": "意图理解" } } } } },// 可带初始数据
  { "op": "connect", "args": { "from": "start", "to": "llm" } },         // 连线(用 tag 引用)
  { "op": "addNode", "tag": "end",   "args": { "type": "2" } },
  { "op": "connect", "args": { "from": "llm", "to": "end" } },
  { "op": "testRun", "args": { "input": {} } }                          // 完整调试
]
```
- `op: addNode` → `args.type`(StandardNodeType,见下表)、可选 `position{x,y}`(缺省自动避让)、可选 `nodeJson`。`tag` 是逻辑名,供后续 connect 引用(自动映射到真实 nodeId)。
  > ⚠️ **nodeJson 要么不传、要么完整**:只在含 `data.inputs` 时才被透传给引擎;只给 `data.nodeMeta.title` 这种半成品会被忽略(否则会覆盖掉节点必需的默认 inputs,导致节点非法、无法调试)。最稳做法:**先不传 nodeJson 用默认创建**(节点合法可调试),需要改标题/参数再到节点表单上设置,或传据节点模板默认补全后的完整 nodeJson。
- `op: connect` → `args.from/to`(节点 tag 或真实 id)、可选 `fromPort/toPort`(缺省用默认端口;If 节点出口端口区分 true/false 分支)。
- `op: testRun` → 触发整图试运行,结果由 WorkflowRunService 轮询 get_process 展示。

## 节点类型 StandardNodeType（@coze-workflow/base）
| type | 节点 | 用途 |
|---|---|---|
| '1' | Start 开始 | 工作流入口(每图一个) |
| '2' | End 结束 | 出口 |
| '3' | LLM 大模型 | 调模型生成 |
| '4' | Api 插件 | 调插件/工具 |
| '5' | Code 代码 | 自定义代码 |
| '6' | Dataset 知识库 | 检索知识库 |
| '8' | If 条件分支 | 条件判断(出口分 true/false) |
| '9' | SubWorkflow 子工作流 | 调另一个工作流 |
| '11' | Variable 变量 | 读写变量 |
| '12' | Database 数据库 | 库操作 |
| '21' | Loop 循环 | 循环体(内含子节点) |
| '28' | Batch 批处理 | 批量(内含子节点) |
| '61' | Mcp | MCP 工具 |
| '100' | Agent 智能体 | 子智能体 |
（完整清单运行时由 `listNodeTypes()` / 后端 `/api/workflow_api/node_template_list` 给出。）

## 渐进式搭建规程
1. **读现状**:`getCanvasJSON()` 看已有节点(空图通常自带 Start/End)。
2. **规划**:把需求拆成 节点序列 + 连边(线性 / 分支 / 循环)。
3. **逐步下发**:按"先加节点(带 tag)→再 connect"的顺序产出 `AgentCanvasCommand[]`;
   指令总线每条之间有停顿(可 `setPace(ms)` 调速),用户看到现场搭建过程。
4. **分支**:If 节点('8')两个出口,connect 时分别指定 `fromPort`(true/false 分支端口)。
5. **循环/批处理**:Loop('21')/Batch('28')是容器,子节点 addNode 时通过 nodeJson 指定父级(或后续在容器内创建)。
6. **填参**:节点关键参数放 addNode 的 `nodeJson.data.inputs`(如 LLM 的 prompt、Api 的入参)。
7. **调试收尾**:全部连好后下发 `{op:"testRun"}`,观察节点执行结果;失败则据 get_process 的 reason 修正。

## 常见配方
- **最简问答流**:Start → LLM → End。
- **带检索**:Start → Dataset(知识库) → LLM → End。
- **条件分支**:… → If → (true)分支A / (false)分支B → End。
- **工具调用**:Start → LLM(决定调用) → Api(插件) → LLM(总结) → End。

## 注意
- 先加节点拿到 tag,再用 tag 连线;不要在节点还没创建时 connect。
- 一次脚本不宜过长,可分多段下发,边搭边看。
- 这是驱动**内存画布**,不自动存盘;搭完调试 OK 后由用户/agent 触发保存(`/api/workflow_api/save`)。
