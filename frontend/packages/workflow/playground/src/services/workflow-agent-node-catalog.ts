export interface WorkflowAgentNodeTypeLike {
  type: string;
  title: string;
}

interface NodeCatalogInfo {
  capability: string;
  configure: string;
}

const NODE_TYPE = {
  LLM: '3',
  Api: '4',
  Code: '5',
  Dataset: '6',
  If: '8',
  SubWorkflow: '9',
  Variable: '11',
  Database: '12',
  Output: '13',
  Imageflow: '14',
  Text: '15',
  ImageGenerate: '16',
  ImageReference: '17',
  Question: '18',
  Break: '19',
  SetVariable: '20',
  Loop: '21',
  Intent: '22',
  ImageCanvas: '23',
  LTM: '26',
  DatasetWrite: '27',
  Batch: '28',
  Continue: '29',
  Input: '30',
  Comment: '31',
  VariableMerge: '32',
  TriggerUpsert: '34',
  TriggerDelete: '35',
  TriggerRead: '36',
  DatabaseUpdate: '42',
  DatabaseQuery: '43',
  DatabaseDelete: '44',
  Http: '45',
  DatabaseCreate: '46',
  UpdateConversation: '51',
  DeleteConversation: '52',
  QueryConversationList: '53',
  QueryConversationHistory: '54',
  CreateMessage: '55',
  UpdateMessage: '56',
  DeleteMessage: '57',
  JsonStringify: '58',
  JsonParser: '59',
  Mcp: '61',
  CardSelector: '99',
  Agent: '100',
} as const;

export const WORKFLOW_AGENT_NODE_CATALOG: Partial<
  Record<string, NodeCatalogInfo>
> = {
  [NODE_TYPE.LLM]: {
    capability: '文本理解、生成、分类、抽取、改写、总结、多轮推理。',
    configure:
      '必须绑定 input/inputs, 写 prompt/system_prompt, 声明 outputs; prompt 可用 {{变量名}}。',
  },
  [NODE_TYPE.Api]: {
    capability: '调用插件/API/工具完成外部查询或动作。',
    configure:
      '必须使用资源清单里的 plugin_id/api_id, 绑定 API 入参, 声明或继承 outputs。',
  },
  [NODE_TYPE.Code]: {
    capability: '执行确定性数据处理、格式转换、计算、字段清洗。',
    configure:
      '必须绑定 input/inputs, 只写 Python: async def main(args: Args), 通过 args.params.get("入参名") 取值, return 字段要和 outputs 对齐。',
  },
  [NODE_TYPE.Dataset]: {
    capability: '知识库检索/RAG, 从文档或表格知识中召回内容。',
    configure:
      '必须使用资源清单里的 dataset_id/dataset_ids, 绑定 Query, 设置 top_k, 声明 outputList。',
  },
  [NODE_TYPE.If]: {
    capability: '按变量条件分流。',
    configure:
      '必须设置 condition.left/operator/right; 连线时 true/false 分支使用 from_port=true/false。',
  },
  [NODE_TYPE.SubWorkflow]: {
    capability: '调用已有工作流作为子流程复用能力。',
    configure: '选择真实 workflow_id, 按子工作流 schema 绑定输入并接收输出。',
  },
  [NODE_TYPE.Variable]: {
    capability: '声明或组装中间变量。',
    configure: '定义变量名、类型和值; 下游按输出变量引用。',
  },
  [NODE_TYPE.Database]: {
    capability: '数据库类能力入口, 用于数据查询或结构化存取。',
    configure: '选择数据表/字段, 绑定查询条件和输出字段。',
  },
  [NODE_TYPE.Output]: {
    capability:
      '纯输出/消息输出节点, 不调用模型, 直接把固定文本或上游变量展示给用户。',
    configure:
      '绑定上游变量作为 input/inputs, content/template 只引用本节点输入名;它不是稳定的下游变量来源,不要接变量聚合或 End returns。',
  },
  [NODE_TYPE.Text]: {
    capability:
      '文本处理, 拼接模板文本或按分隔符拆分字符串,会产出真实 output 变量。',
    configure:
      '绑定 input/inputs;固定文案不需要变量时可不绑定输入并清空默认输入;concat 模式设置 content/template 且只引用输入名, split 模式设置 delimiter;声明 output。分支结果要被聚合时优先用它。',
  },
  [NODE_TYPE.Question]: {
    capability: '向用户追问缺失信息。',
    configure:
      '设置 question/content、answer_type(text/option)、options/limit, 绑定必要上下文。',
  },
  [NODE_TYPE.SetVariable]: {
    capability: '给变量赋值, 适合流程中更新上下文状态。',
    configure: '选择变量名并绑定新值。',
  },
  [NODE_TYPE.VariableMerge]: {
    capability: '合并多个分支或多个变量。',
    configure:
      '分支汇合时使用; merge_groups.variables 必须来自 get_canvas_context 的可绑定变量,覆盖所有可能返回分支;不要聚合 type=13 输出/消息节点,固定文案分支先用 type=15 文本处理产出 output;End 绑定合并输出。',
  },
  [NODE_TYPE.Loop]: {
    capability: '遍历数组逐项处理。',
    configure: '绑定数组输入, 配置循环体和循环输出。',
  },
  [NODE_TYPE.Batch]: {
    capability: '批量处理多条输入, 并行或分批执行子链路。',
    configure: '绑定批量数组, 设置批大小/并发, 声明批量输出。',
  },
  [NODE_TYPE.Input]: {
    capability: '声明工作流入参, 用于补充 Start.input 以外的结构化输入。',
    configure:
      'configure_node 设置 outputs=[{name,type,description}], 下游可按该节点输出变量绑定。',
  },
  [NODE_TYPE.Break]: {
    capability: '在循环中提前终止。',
    configure: '通常放在循环分支内, 条件由上游 If 控制。',
  },
  [NODE_TYPE.Continue]: {
    capability: '在循环中跳过当前项继续下一项。',
    configure: '通常放在循环分支内, 条件由上游 If 控制。',
  },
  [NODE_TYPE.Intent]: {
    capability: '意图识别/分类, 用于把用户输入路由到不同分支。',
    configure: '绑定用户输入, 定义候选意图/类别, 声明分类输出。',
  },
  [NODE_TYPE.Http]: {
    capability: '直接调用 HTTP 接口。',
    configure: '配置 method/url/headers/body, 绑定变量, 声明响应 outputs。',
  },
  [NODE_TYPE.Mcp]: {
    capability: '调用 MCP 工具。',
    configure: '选择 MCP server/tool, 绑定工具参数, 声明工具返回 outputs。',
  },
  [NODE_TYPE.Agent]: {
    capability: '调用子智能体/HiAgent/Coze Agent 完成独立任务。',
    configure:
      '必须使用资源清单里的 agent_id/platform, 绑定 query/inputParameters, 声明 answer 等 outputs。',
  },
  [NODE_TYPE.DatasetWrite]: {
    capability: '写入知识库。',
    configure: '选择 dataset_id, 绑定待写入知识内容和写入策略。',
  },
  [NODE_TYPE.DatabaseQuery]: {
    capability: '查询数据库。',
    configure: '选择表和字段, 绑定过滤条件, 声明查询结果输出。',
  },
  [NODE_TYPE.DatabaseCreate]: {
    capability: '新增数据库记录。',
    configure: '选择表, 绑定新增字段值, 声明创建结果。',
  },
  [NODE_TYPE.DatabaseUpdate]: {
    capability: '更新数据库记录。',
    configure: '选择表, 绑定过滤条件和更新字段。',
  },
  [NODE_TYPE.DatabaseDelete]: {
    capability: '删除数据库记录。',
    configure: '选择表, 绑定删除条件。',
  },
  [NODE_TYPE.JsonStringify]: {
    capability: '把对象转 JSON 字符串。',
    configure: '绑定对象输入, 声明 string 输出。',
  },
  [NODE_TYPE.JsonParser]: {
    capability: '解析 JSON 字符串为结构化对象。',
    configure: '绑定 JSON 字符串, 定义解析 schema/outputs。',
  },
  [NODE_TYPE.QueryMessageList]: {
    capability: '查询会话消息列表。',
    configure: '绑定会话标识和分页参数。',
  },
  [NODE_TYPE.CreateConversation]: {
    capability: '创建会话。',
    configure: '绑定用户/业务标识并声明 conversation_id 输出。',
  },
  [NODE_TYPE.UpdateConversation]: {
    capability: '更新会话。',
    configure: '绑定 conversation_id 和更新字段。',
  },
  [NODE_TYPE.DeleteConversation]: {
    capability: '删除会话。',
    configure: '绑定 conversation_id。',
  },
  [NODE_TYPE.QueryConversationList]: {
    capability: '查询会话列表。',
    configure: '绑定查询条件和分页参数。',
  },
  [NODE_TYPE.QueryConversationHistory]: {
    capability: '查询会话历史。',
    configure: '绑定 conversation_id 和分页参数。',
  },
  [NODE_TYPE.CreateMessage]: {
    capability: '创建消息。',
    configure: '绑定 conversation_id、role、content。',
  },
  [NODE_TYPE.UpdateMessage]: {
    capability: '更新消息。',
    configure: '绑定 message_id 和更新内容。',
  },
  [NODE_TYPE.DeleteMessage]: {
    capability: '删除消息。',
    configure: '绑定 message_id。',
  },
  [NODE_TYPE.CardSelector]: {
    capability: '卡片选择/交互分支。',
    configure:
      '设置 selected_card/card_id 和 content, 绑定卡片模板变量, 将用户选择作为输出变量。',
  },
  [NODE_TYPE.Comment]: {
    capability: '画布注释说明, 不参与运行。',
    configure: '写清说明即可, 不要当成执行节点。',
  },
  [NODE_TYPE.ImageGenerate]: {
    capability: '生成图片。',
    configure: '绑定图片提示词、尺寸/风格参数, 声明图片输出。',
  },
  [NODE_TYPE.ImageReference]: {
    capability: '引用图片作为后续图像处理输入。',
    configure: '绑定图片变量或上传资源。',
  },
  [NODE_TYPE.ImageCanvas]: {
    capability: '图片画布编辑/组合, 用于视觉内容生成链路中的画布操作。',
    configure:
      '需要图片资源或上游图片变量;当前只可作为占位添加,完整画布参数需补语义配置器。',
  },
  [NODE_TYPE.LTM]: {
    capability: '长期记忆读写, 用于跨会话保存或检索用户偏好/事实。',
    configure:
      '需要明确记忆资源、读写模式和输入绑定;当前需补资源发现与语义配置器。',
  },
  [NODE_TYPE.Imageflow]: {
    capability: '调用图像工作流。',
    configure: '选择 imageflow, 绑定图片/文本输入, 接收图片输出。',
  },
  [NODE_TYPE.TriggerUpsert]: {
    capability: '新增或更新定时/事件触发器。',
    configure:
      '需要触发器资源、触发条件和输入参数;当前需补触发器语义配置器。',
  },
  [NODE_TYPE.TriggerDelete]: {
    capability: '删除定时/事件触发器。',
    configure: '需要绑定触发器标识;当前需补触发器删除语义配置器。',
  },
  [NODE_TYPE.TriggerRead]: {
    capability: '查询定时/事件触发器。',
    configure: '需要绑定查询条件并声明 outputs;当前需补触发器查询语义配置器。',
  },
};

export function formatWorkflowAgentNodeCatalog(
  nodeTypes: WorkflowAgentNodeTypeLike[],
): string {
  if (!nodeTypes.length) {
    return '未加载到可新增节点模板。';
  }

  const lines = [
    '节点设计原则: 先根据需求选择最小但完整的节点组合; 每个执行节点都要考虑输入绑定、核心配置、outputs、下游消费; Start/End 是内置单例只能引用不能新增。',
    '可新增节点目录:',
  ];

  nodeTypes.forEach(node => {
    const info = getWorkflowAgentNodeCatalogInfo(node.type);
    lines.push(
      `- ${node.type} ${node.title}: ${info.capability} 配置要点: ${info.configure}`,
    );
  });

  return lines.join('\n');
}

export function getWorkflowAgentNodeCatalogInfo(
  type: string,
): NodeCatalogInfo {
  return (
    WORKFLOW_AGENT_NODE_CATALOG[type] ?? {
      capability:
        '按节点标题判断用途; 适合需求明确但当前目录暂无详细说明的扩展节点。',
      configure:
        '先添加占位, 再用 configure_node 写标题、输入绑定和 outputs; 不确定资源时标题标注待确认。',
    }
  );
}
