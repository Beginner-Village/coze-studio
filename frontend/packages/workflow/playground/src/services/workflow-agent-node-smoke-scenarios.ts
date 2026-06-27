import {
  getWorkflowAgentNodeCapability,
  WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES,
} from './workflow-agent-node-capabilities';
import { type WorkflowCanvasCommandOp } from './workflow-agent-command-protocol';

export type WorkflowAgentNodeSmokeStatus =
  | 'verified-full'
  | 'verified-resource-bound'
  | 'verified-add-only'
  | 'verified-not-executable'
  | 'skipped-resource-missing'
  | 'unsupported'
  | 'failing';

export interface WorkflowAgentNodeSmokeScenario {
  nodeType: string;
  nodeName: string;
  status: WorkflowAgentNodeSmokeStatus;
  runtime: 'local' | 'resource' | 'sub-canvas' | 'not-executable';
  requiredCommands: WorkflowCanvasCommandOp[];
  setupCommands: Array<{
    op: WorkflowCanvasCommandOp;
    target?: string;
    args?: Record<string, unknown>;
  }>;
  assertions: string[];
  expectedBindableVariables: string[];
  skipReason?: string;
  knownGaps?: string[];
  permitsNoInputBinding?: boolean;
}

type ScenarioInput = Omit<WorkflowAgentNodeSmokeScenario, 'nodeName'> & {
  nodeName?: string;
};

const scenario = (input: ScenarioInput): WorkflowAgentNodeSmokeScenario => ({
  ...input,
  nodeName:
    input.nodeName ??
    getWorkflowAgentNodeCapability(input.nodeType)?.name ??
    input.nodeType,
});

export const WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS: WorkflowAgentNodeSmokeScenario[] =
  [
    scenario({
      nodeType: '1',
      status: 'verified-not-executable',
      runtime: 'not-executable',
      requiredCommands: ['get_canvas_context'],
      setupCommands: [],
      assertions: ['Start 是内置单例,不能新增,默认暴露 input:string。'],
      expectedBindableVariables: ['start.input'],
    }),
    scenario({
      nodeType: '2',
      status: 'verified-not-executable',
      runtime: 'not-executable',
      requiredCommands: [
        'connect',
        'get_bindable_variables',
        'configure_node',
        'validate',
      ],
      setupCommands: [
        {
          op: 'add_node',
          target: 'smoke_2_text_source',
          args: { type: '15', title: '结束返回文本来源' },
        },
        {
          op: 'configure_node',
          target: 'smoke_2_text_source',
          args: {
            config: {
              title: '结束返回文本来源',
              method: 'concat',
              content: 'End 文本返回 smoke',
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
        {
          op: 'connect',
          args: { from: 'start', to: 'smoke_2_text_source' },
        },
        {
          op: 'connect',
          args: { from: 'smoke_2_text_source', to: 'end' },
        },
        {
          op: 'get_bindable_variables',
          target: 'end',
          args: { target_node: 'end' },
        },
        {
          op: 'configure_node',
          target: 'end',
          args: {
            config: {
              inputs: [
                {
                  name: 'answer',
                  from: 'smoke_2_text_source',
                  output: 'output',
                  type: 'string',
                },
              ],
              content: '最终结果: {{answer}}',
              streaming_output: true,
            },
          },
        },
      ],
      assertions: [
        'End 可以返回变量: returns 必须绑定真实上游输出,不能为空。',
        'End 可以返回文本: input/inputs 绑定多个上游变量,content 用 {{变量名}} 拼接,streaming_output=true 时流式透传。',
      ],
      expectedBindableVariables: ['text.output'],
    }),
    scenario({
      nodeType: '3',
      status: 'verified-resource-bound',
      runtime: 'resource',
      requiredCommands: ['add_node', 'configure_node', 'validate', 'test_run'],
      setupCommands: [],
      assertions: ['需要真实模型资源;prompt 输入变量必须来自本节点 input。'],
      expectedBindableVariables: ['llm.answer'],
      skipReason: '运行依赖真实模型资源。',
    }),
    scenario({
      nodeType: '4',
      status: 'verified-resource-bound',
      runtime: 'resource',
      requiredCommands: ['add_node', 'configure_node', 'validate', 'test_run'],
      setupCommands: [],
      assertions: ['需要真实 plugin_id/api_id;入参必须按资源 schema 绑定。'],
      expectedBindableVariables: ['api.output'],
      skipReason: '运行依赖空间插件/API资源。',
    }),
    scenario({
      nodeType: '5',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: [
        'add_node',
        'connect',
        'configure_node',
        'get_bindable_variables',
        'validate',
        'test_run',
      ],
      setupCommands: [
        {
          op: 'connect',
          args: { from: 'start', to: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              input: { from: 'start', output: 'input', name: 'query' },
              language: 'python',
              code: [
                'async def main(args: Args):',
                '    params = args.params',
                '    query = str(params.get("query", "")).strip()',
                '    return {"result": query}',
              ].join('\n'),
              outputs: [{ name: 'result', type: 'string' }],
            },
          },
        },
      ],
      assertions: [
        '代码 smoke 使用平台支持的 Python Args main 入口。',
        '输入必须从 args.params 读取,不要对 args 直接调用 strip。',
        'return 字段必须和 outputs 对齐。',
      ],
      expectedBindableVariables: ['code.output'],
    }),
    scenario({
      nodeType: '6',
      status: 'verified-resource-bound',
      runtime: 'resource',
      requiredCommands: ['add_node', 'configure_node', 'validate'],
      setupCommands: [],
      assertions: ['需要真实 dataset_id;query 绑定真实输入变量。'],
      expectedBindableVariables: ['dataset.outputList'],
      skipReason: '运行依赖空间知识库资源。',
    }),
    scenario({
      nodeType: '8',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: [
        'add_node',
        'connect',
        'configure_node',
        'get_bindable_variables',
        'validate',
      ],
      setupCommands: [
        {
          op: 'connect',
          args: { from: 'start', to: 'self' },
        },
        {
          op: 'get_bindable_variables',
          target: 'self',
          args: { target_node: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: '条件分支 Smoke',
              condition: {
                left: { from: 'start', output: 'input', type: 'string' },
                operator: 'equal',
                right: '1',
              },
            },
          },
        },
      ],
      assertions: ['条件左值必须来自可绑定变量;true/false 分支端口必须明确。'],
      expectedBindableVariables: [],
    }),
    scenario({
      nodeType: '9',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要读取子工作流 schema 后才能绑定输入输出。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少子工作流 schema 发现和语义配置器。'],
    }),
    scenario({
      nodeType: '11',
      status: 'verified-add-only',
      runtime: 'local',
      requiredCommands: ['add_node', 'delete_node'],
      setupCommands: [],
      assertions: ['当前只验证可添加/删除,不可宣称完整执行节点。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少变量节点语义配置器。'],
    }),
    scenario({
      nodeType: '12',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要数据库资源和 SQL/inputParameters 语义配置器。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少数据库资源发现和 SQL 配置器。'],
    }),
    scenario({
      nodeType: '13',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: ['add_node', 'connect', 'configure_node', 'validate'],
      setupCommands: [
        {
          op: 'connect',
          args: { from: 'start', to: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: '展示输出 Smoke',
              content: '节点 smoke 展示输出',
            },
          },
        },
      ],
      assertions: [
        '输出节点用于展示消息。',
        '不能作为 VariableMerge 或 End returns 的稳定变量来源。',
      ],
      expectedBindableVariables: [],
    }),
    scenario({
      nodeType: '14',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要图像流资源发现。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少图像流语义配置器。'],
    }),
    scenario({
      nodeType: '15',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: [
        'add_node',
        'connect',
        'configure_node',
        'get_bindable_variables',
        'validate',
        'test_run',
      ],
      setupCommands: [
        {
          op: 'connect',
          args: { from: 'start', to: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: '固定文本 Smoke',
              method: 'concat',
              content: '固定文本 smoke 输出',
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
      ],
      assertions: [
        '固定文案允许删除默认 input。',
        '需要被下游消费时必须声明 output:string。',
      ],
      expectedBindableVariables: ['text.output'],
      permitsNoInputBinding: true,
    }),
    scenario({
      nodeType: '16',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要图片生成参数语义配置器。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少图片生成语义配置器。'],
    }),
    scenario({
      nodeType: '17',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要图片资源选择和绑定。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少图片引用配置器。'],
    }),
    scenario({
      nodeType: '18',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: ['add_node', 'connect', 'configure_node', 'validate'],
      setupCommands: [
        {
          op: 'connect',
          args: { from: 'start', to: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: '追问 Smoke',
              input: { name: 'context', from: 'start', output: 'input' },
              question: '请补充必要信息: {{context}}',
              answer_type: 'text',
              outputs: [{ name: 'answer', type: 'string' }],
            },
          },
        },
      ],
      assertions: ['追问节点必须声明问题和答案变量。'],
      expectedBindableVariables: ['question.answer'],
    }),
    scenario({
      nodeType: '19',
      status: 'unsupported',
      runtime: 'sub-canvas',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['只能在循环子画布中验证。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少循环子画布 smoke。'],
    }),
    scenario({
      nodeType: '20',
      status: 'verified-add-only',
      runtime: 'local',
      requiredCommands: ['add_node', 'delete_node'],
      setupCommands: [],
      assertions: ['当前只验证可添加/删除。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少变量赋值语义配置器。'],
    }),
    scenario({
      nodeType: '21',
      status: 'unsupported',
      runtime: 'sub-canvas',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['循环需要数组输入、循环变量和子画布验证。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少循环语义配置器。'],
    }),
    scenario({
      nodeType: '22',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['建议先用 LLM 分类替代,原生意图节点待补。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少候选意图配置和分类输出验证。'],
    }),
    scenario({
      nodeType: '23',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要图像画布配置器。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少图像画布配置器。'],
    }),
    scenario({
      nodeType: '26',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要长期记忆资源发现。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少长期记忆配置器。'],
    }),
    scenario({
      nodeType: '27',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['需要知识库写入资源和写入策略。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少知识库写入配置器。'],
    }),
    scenario({
      nodeType: '28',
      status: 'unsupported',
      runtime: 'sub-canvas',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['批处理需要数组输入和子链路验证。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少批处理语义配置器。'],
    }),
    scenario({
      nodeType: '29',
      status: 'unsupported',
      runtime: 'sub-canvas',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['只能在循环子画布中验证。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少循环子画布 smoke。'],
    }),
    scenario({
      nodeType: '30',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: ['add_node', 'configure_node', 'validate'],
      setupCommands: [
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: '结构化输入 Smoke',
              outputs: [{ name: 'user_input', type: 'string' }],
            },
          },
        },
      ],
      assertions: ['输入节点必须声明 outputs,下游按输出变量绑定。'],
      expectedBindableVariables: ['input.output'],
    }),
    scenario({
      nodeType: '31',
      status: 'verified-not-executable',
      runtime: 'not-executable',
      requiredCommands: ['add_node', 'delete_node'],
      setupCommands: [],
      assertions: ['注释节点不参与执行和变量绑定。'],
      expectedBindableVariables: [],
    }),
    scenario({
      nodeType: '32',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: [
        'add_node',
        'connect',
        'get_bindable_variables',
        'configure_node',
        'validate',
        'test_run',
      ],
      setupCommands: [
        {
          op: 'add_node',
          target: 'smoke_32_text_primary',
          args: { type: '15', title: '聚合输入 A' },
        },
        {
          op: 'configure_node',
          target: 'smoke_32_text_primary',
          args: {
            config: {
              title: '聚合输入 A',
              method: 'concat',
              content: '主要分支输出',
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
        {
          op: 'connect',
          args: { from: 'start', to: 'smoke_32_text_primary' },
        },
        {
          op: 'add_node',
          target: 'smoke_32_text_fallback',
          args: { type: '15', title: '聚合输入 B' },
        },
        {
          op: 'configure_node',
          target: 'smoke_32_text_fallback',
          args: {
            config: {
              title: '聚合输入 B',
              method: 'concat',
              content: '备用分支输出',
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
        {
          op: 'connect',
          args: { from: 'start', to: 'smoke_32_text_fallback' },
        },
        {
          op: 'connect',
          args: { from: 'smoke_32_text_primary', to: 'self' },
        },
        {
          op: 'connect',
          args: { from: 'smoke_32_text_fallback', to: 'self' },
        },
        {
          op: 'get_bindable_variables',
          target: 'self',
          args: { target_node: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: '变量聚合 Smoke',
              merge_groups: [
                {
                  name: 'output',
                  variables: [
                    {
                      from: 'smoke_32_text_primary',
                      output: 'output',
                      type: 'string',
                    },
                    {
                      from: 'smoke_32_text_fallback',
                      output: 'output',
                      type: 'string',
                    },
                  ],
                },
              ],
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
      ],
      assertions: [
        'merge_groups.variables 必须全部来自可绑定变量。',
        '不能引用 type=13 输出节点。',
        'End returns 应绑定 VariableMerge 的真实 output。',
      ],
      expectedBindableVariables: ['merge.output'],
    }),
    ...['34', '35', '36'].map(nodeType =>
      scenario({
        nodeType,
        status: 'unsupported',
        runtime: 'resource',
        requiredCommands: ['add_node', 'get_node_spec'],
        setupCommands: [],
        assertions: ['触发器类节点需要资源和触发 schema。'],
        expectedBindableVariables: [],
        knownGaps: ['缺少触发器配置器。'],
      }),
    ),
    ...['42', '43', '44', '46'].map(nodeType =>
      scenario({
        nodeType,
        status: 'unsupported',
        runtime: 'resource',
        requiredCommands: ['add_node', 'get_node_spec'],
        setupCommands: [],
        assertions: ['数据库 CRUD 节点需要表、字段和条件绑定。'],
        expectedBindableVariables: [],
        knownGaps: ['缺少数据库 CRUD 语义配置器。'],
      }),
    ),
    scenario({
      nodeType: '45',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['HTTP 需要 method/url/headers/query/body/outputs 配置器。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少 HTTP 语义配置器。'],
    }),
    scenario({
      nodeType: '58',
      status: 'verified-full',
      runtime: 'local',
      requiredCommands: [
        'add_node',
        'connect',
        'configure_node',
        'get_bindable_variables',
        'validate',
        'test_run',
      ],
      setupCommands: [
        {
          op: 'add_node',
          target: 'smoke_58_text_source',
          args: { type: '15', title: 'JSON 输入来源' },
        },
        {
          op: 'configure_node',
          target: 'smoke_58_text_source',
          args: {
            config: {
              title: 'JSON 输入来源',
              method: 'concat',
              content: 'JSON 序列化 smoke',
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
        {
          op: 'connect',
          args: { from: 'start', to: 'smoke_58_text_source' },
        },
        {
          op: 'connect',
          args: { from: 'smoke_58_text_source', to: 'self' },
        },
        {
          op: 'get_bindable_variables',
          target: 'self',
          args: { target_node: 'self' },
        },
        {
          op: 'configure_node',
          target: 'self',
          args: {
            config: {
              title: 'JSON 序列化 Smoke',
              inputs: [
                {
                  name: 'input',
                  from: 'smoke_58_text_source',
                  output: 'output',
                  type: 'string',
                },
              ],
              outputs: [{ name: 'output', type: 'string' }],
            },
          },
        },
      ],
      assertions: ['JSON 序列化输入必须来自真实变量,输出 string。'],
      expectedBindableVariables: ['json.output'],
    }),
    scenario({
      nodeType: '59',
      status: 'unsupported',
      runtime: 'local',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['JSON 解析需要 schema/paths 和 outputs。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少 JSON 解析语义配置器。'],
    }),
    scenario({
      nodeType: '61',
      status: 'unsupported',
      runtime: 'resource',
      requiredCommands: ['add_node', 'get_node_spec'],
      setupCommands: [],
      assertions: ['MCP 节点需要 server/tool 发现和参数绑定。'],
      expectedBindableVariables: [],
      knownGaps: ['缺少 MCP 资源发现和语义配置器。'],
    }),
    scenario({
      nodeType: '99',
      status: 'verified-resource-bound',
      runtime: 'resource',
      requiredCommands: ['add_node', 'configure_node', 'validate'],
      setupCommands: [],
      assertions: ['需要真实卡片资源,输入变量必须绑定。'],
      expectedBindableVariables: ['card.output'],
      skipReason: '运行依赖卡片资源。',
    }),
    scenario({
      nodeType: '100',
      status: 'verified-resource-bound',
      runtime: 'resource',
      requiredCommands: ['add_node', 'configure_node', 'validate', 'test_run'],
      setupCommands: [],
      assertions: ['需要真实子智能体资源,query 绑定真实输入变量。'],
      expectedBindableVariables: ['agent.answer'],
      skipReason: '运行依赖子智能体资源。',
    }),
  ].sort(
    (a, b) =>
      WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES.indexOf(
        a.nodeType as (typeof WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES)[number],
      ) -
      WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES.indexOf(
        b.nodeType as (typeof WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES)[number],
      ),
  );

export const getWorkflowAgentNodeSmokeScenario = (
  nodeType: string,
): WorkflowAgentNodeSmokeScenario | undefined =>
  WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS.find(
    scenario => scenario.nodeType === nodeType,
  );
