import { nanoid } from 'nanoid';

type VariableTypeName =
  | 'string'
  | 'integer'
  | 'boolean'
  | 'number'
  | 'object'
  | 'array_string'
  | 'array_object';

const NODE_TYPE = {
  End: '2',
  LLM: '3',
  Api: '4',
  Code: '5',
  Dataset: '6',
  If: '8',
  DatabaseSQL: '12',
  Output: '13',
  Text: '15',
  Question: '18',
  Input: '30',
  VariableMerge: '32',
  DatabaseUpdate: '42',
  DatabaseQuery: '43',
  DatabaseDelete: '44',
  DatabaseCreate: '46',
  JsonStringify: '58',
  CardSelector: '99',
  Agent: '100',
} as const;

export const WORKFLOW_AGENT_SEMANTIC_CONFIG_NODE_TYPES = [
  NODE_TYPE.End,
  NODE_TYPE.LLM,
  NODE_TYPE.Api,
  NODE_TYPE.Code,
  NODE_TYPE.Dataset,
  NODE_TYPE.If,
  NODE_TYPE.Output,
  NODE_TYPE.Text,
  NODE_TYPE.Question,
  NODE_TYPE.Input,
  NODE_TYPE.VariableMerge,
  NODE_TYPE.DatabaseSQL,
  NODE_TYPE.DatabaseUpdate,
  NODE_TYPE.DatabaseQuery,
  NODE_TYPE.DatabaseDelete,
  NODE_TYPE.DatabaseCreate,
  NODE_TYPE.JsonStringify,
  NODE_TYPE.CardSelector,
  NODE_TYPE.Agent,
] as const;

const VALUE_EXPRESSION_TYPE = {
  LITERAL: 'literal',
  REF: 'ref',
} as const;

const VIEW_VARIABLE_TYPE = {
  String: 1,
  Integer: 2,
  Boolean: 3,
  Number: 4,
  Object: 6,
  ArrayString: 99,
  ArrayObject: 103,
} as const;

type ViewVariableTypeValue =
  (typeof VIEW_VARIABLE_TYPE)[keyof typeof VIEW_VARIABLE_TYPE];

interface RefExpression {
  type: 'ref';
  content?: { keyPath: string[] };
  rawMeta?: { type?: ViewVariableTypeValue };
}

interface LiteralExpression {
  type: 'literal';
  content?: string | number | boolean | Array<unknown>;
  rawMeta?: { type?: ViewVariableTypeValue };
}

type ValueExpression = RefExpression | LiteralExpression;

interface InputValueVO {
  name?: string;
  input: ValueExpression;
  children?: InputValueVO[];
  key?: string;
}

interface OutputValueVO {
  key: string;
  name: string;
  type: ViewVariableTypeValue;
  description?: string;
  children?: OutputValueVO[];
}

interface VariableBindingConfig {
  name?: string;
  from?: string;
  node?: string;
  node_tag?: string;
  output?: string;
  variable?: string;
  type?: VariableTypeName | ViewVariableTypeValue;
}

interface OutputConfig {
  name: string;
  type?: VariableTypeName | ViewVariableTypeValue;
  description?: string;
}

interface ConditionConfig {
  left: VariableBindingConfig;
  operator?: string | number;
  right?: string | number | boolean | VariableBindingConfig;
}

interface VariableMergeGroupConfig {
  name?: string;
  variables: VariableBindingConfig[];
}

export interface WorkflowAgentSemanticConfig {
  title?: string;
  input?: VariableBindingConfig | VariableBindingConfig[];
  inputs?: VariableBindingConfig[] | Record<string, VariableBindingConfig>;
  variables?: VariableBindingConfig[];
  merge_groups?: VariableMergeGroupConfig[];
  plugin_id?: string;
  api_id?: string;
  plugin_name?: string;
  api_name?: string;
  plugin_version?: string;
  dataset_id?: string;
  dataset_ids?: string[];
  top_k?: number;
  agent_id?: string;
  agent_name?: string;
  platform?: string;
  query?: string;
  prompt?: string;
  user_prompt?: string;
  system_prompt?: string;
  systemPrompt?: string;
  content?: string;
  text?: string;
  template?: string;
  streaming_output?: boolean;
  streamingOutput?: boolean;
  enable_chat_history?: boolean;
  chat_history_round?: number;
  output?: string | OutputConfig;
  outputs?: Array<string | OutputConfig>;
  returns?: VariableBindingConfig | VariableBindingConfig[];
  return?: VariableBindingConfig | VariableBindingConfig[];
  code?: string;
  language?: string;
  condition?: ConditionConfig;
  method?: 'concat' | 'split';
  delimiter?: string | string[];
  concat_char?: string;
  question?: string;
  answer_type?: 'text' | 'option';
  option_type?: string;
  options?: Array<string | { id?: string; name?: string; content?: string }>;
  limit?: number;
  extra_output?: boolean;
  selected_card?: Record<string, unknown>;
  card_id?: string;
  card_name?: string;
  card_code?: string;
  // 绑定到 LLM 节点的 FC(function calling)技能:让大模型在节点内按需调用插件/工作流(节点 agent 化)。
  // 与独立的插件节点(type=4)区别:FC 是模型自主决定何时调用;type=4 是确定性管线调用。
  bind_plugins?: Array<{
    plugin_id: string;
    api_id: string;
    api_name?: string;
    plugin_version?: string;
  }>;
  bind_workflows?: Array<{
    workflow_id: string;
    plugin_id?: string;
    plugin_version?: string;
    workflow_version?: string;
  }>;
  // 数据库节点(type 12/42/43/44/46):table_id/field_id 来自 workflow_canvas_list_databases,禁止编造
  table_id?: string;
  database_id?: string;
  fields?: Array<string | number | { field_id: string | number; name?: string }>;
  conditions?: Array<{
    left: string;
    operator?: string;
    right?: string | number | boolean | VariableBindingConfig;
  }>;
  values?: Array<{
    field_id: string | number;
    value?: string | number | boolean | VariableBindingConfig;
  }>;
  order_by?: Array<{ field_id: string | number; asc?: boolean }>;
  limit?: number;
  sql?: string;
}

export interface CreateWorkflowAgentSemanticParamsOptions {
  nodeType: string;
  config: WorkflowAgentSemanticConfig;
  resolveNodeRef: (ref: string) => string;
}

const viewTypeByName: Record<VariableTypeName, ViewVariableTypeValue> = {
  string: VIEW_VARIABLE_TYPE.String,
  integer: VIEW_VARIABLE_TYPE.Integer,
  boolean: VIEW_VARIABLE_TYPE.Boolean,
  number: VIEW_VARIABLE_TYPE.Number,
  object: VIEW_VARIABLE_TYPE.Object,
  array_string: VIEW_VARIABLE_TYPE.ArrayString,
  array_object: VIEW_VARIABLE_TYPE.ArrayObject,
};

const conditionOperatorByName: Record<string, number> = {
  equal: 1,
  equals: 1,
  not_equal: 2,
  length_gt: 3,
  length_gte: 4,
  length_lt: 5,
  length_lte: 6,
  contains: 7,
  not_contains: 8,
  null: 9,
  not_null: 10,
  true: 11,
  false: 12,
  gt: 13,
  gte: 14,
  lt: 15,
  lte: 16,
};

export function createWorkflowAgentSemanticParams({
  nodeType,
  config,
  resolveNodeRef,
}: CreateWorkflowAgentSemanticParamsOptions): Record<string, unknown> {
  const params: Record<string, unknown> = {};
  if (config.title) {
    params['nodeMeta.title'] = config.title;
  }

  if (nodeType === NODE_TYPE.LLM) {
    assignLLMParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.End) {
    assignEndParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Code) {
    assignCodeParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.If) {
    assignIfParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Output) {
    assignOutputParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Text) {
    assignTextParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Question) {
    assignQuestionParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Input) {
    assignInputParams(params, config);
  } else if (nodeType === NODE_TYPE.Api) {
    assignApiParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Dataset) {
    assignDatasetParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.VariableMerge) {
    assignVariableMergeParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.JsonStringify) {
    assignJsonStringifyParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.CardSelector) {
    assignCardSelectorParams(params, config, resolveNodeRef);
  } else if (nodeType === NODE_TYPE.Agent) {
    assignAgentParams(params, config, resolveNodeRef);
  } else if (
    nodeType === NODE_TYPE.DatabaseQuery ||
    nodeType === NODE_TYPE.DatabaseCreate ||
    nodeType === NODE_TYPE.DatabaseUpdate ||
    nodeType === NODE_TYPE.DatabaseDelete ||
    nodeType === NODE_TYPE.DatabaseSQL
  ) {
    assignDatabaseParams(params, config, resolveNodeRef, nodeType);
  }

  return params;
}

function assignLLMParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const inputParameters = createInputParameters(config, resolveNodeRef);
  if (inputParameters.length) {
    params['$$input_decorator$$.inputParameters'] = inputParameters;
  }
  const prompt = config.prompt ?? config.user_prompt;
  if (prompt !== undefined) {
    params['$$prompt_decorator$$.prompt'] = prompt;
  }
  const systemPrompt = config.system_prompt ?? config.systemPrompt;
  if (systemPrompt !== undefined) {
    params['$$prompt_decorator$$.systemPrompt'] = systemPrompt;
  }
  // 对话工作流:开启对话历史(多轮记忆)。LLM 节点的对话历史在表单字段
  // $$input_decorator$$.chatHistorySetting,形态为 {enableChatHistory, chatHistoryRound}。
  if (
    config.enable_chat_history !== undefined ||
    config.chat_history_round !== undefined
  ) {
    params['$$input_decorator$$.chatHistorySetting'] = {
      enableChatHistory: Boolean(config.enable_chat_history),
      chatHistoryRound: config.chat_history_round ?? 3,
    };
  }
  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
  // FC(function calling):把插件/工作流绑定到大模型节点,让它在节点内按需调用(节点 agent 化)。
  // 表单字段为顶层 fcParam(BoundSkills 结构),提交时落库到 inputs.fcParam。
  const fcParam = buildLLMFcParam(config);
  if (fcParam) {
    params.fcParam = fcParam;
  }
}

function buildLLMFcParam(
  config: WorkflowAgentSemanticConfig,
): Record<string, unknown> | undefined {
  const fcParam: Record<string, unknown> = {};
  const plugins = (config.bind_plugins ?? []).filter(
    p => p && p.plugin_id && p.api_id,
  );
  if (plugins.length) {
    fcParam.pluginFCParam = {
      pluginList: plugins.map(p => ({
        plugin_id: String(p.plugin_id),
        api_id: String(p.api_id),
        api_name: p.api_name ?? '',
        plugin_version: p.plugin_version ?? '',
        is_draft: false,
      })),
    };
  }
  const workflows = (config.bind_workflows ?? []).filter(
    w => w && w.workflow_id,
  );
  if (workflows.length) {
    fcParam.workflowFCParam = {
      workflowList: workflows.map(w => ({
        plugin_id: String(w.plugin_id ?? ''),
        workflow_id: String(w.workflow_id),
        plugin_version: String(w.plugin_version ?? ''),
        workflow_version: String(w.workflow_version ?? ''),
        is_draft: false,
      })),
    };
  }
  return Object.keys(fcParam).length ? fcParam : undefined;
}

function assignEndParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const content = getContent(config);
  if (content !== undefined) {
    params['inputs.terminatePlan'] = 'useAnswerContent';
    const inputParameters = createEndAnswerInputParameters(
      config,
      resolveNodeRef,
    );
    params['inputs.inputParameters'] = inputParameters;
    params['inputs.content'] = content;
    const streamingOutput = config.streaming_output ?? config.streamingOutput;
    if (streamingOutput !== undefined) {
      params['inputs.streamingOutput'] = Boolean(streamingOutput);
    }
    return;
  }

  const returns = normalizeBindings(config.returns ?? config.return);
  if (!returns.length) {
    return;
  }
  params['inputs.terminatePlan'] = 'returnVariables';
  params['inputs.inputParameters'] = returns.map(binding => ({
    name: binding.name || 'output',
    input: createRefExpression(binding, resolveNodeRef),
  }));
}

function createEndAnswerInputParameters(
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): InputValueVO[] {
  const inputParameters = createInputParameters(config, resolveNodeRef);
  if (inputParameters.length) {
    return inputParameters;
  }
  return normalizeBindings(config.returns ?? config.return).map(binding => ({
    name: binding.name || binding.output || 'output',
    input: createRefExpression(binding, resolveNodeRef),
  }));
}

function assignCodeParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const inputParameters = createInputParameters(config, resolveNodeRef);
  if (inputParameters.length) {
    params.inputParameters = inputParameters;
  }
  if (config.code !== undefined) {
    params['codeParams.code'] = config.code;
  }
  if (config.language !== undefined) {
    params['codeParams.language'] = config.language;
  }
  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
}

function assignIfParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  if (!config.condition) {
    return;
  }
  params.condition = [
    {
      condition: {
        logic: 2,
        conditions: [
          {
            left: createRefExpression(config.condition.left, resolveNodeRef),
            operator: normalizeConditionOperator(config.condition.operator),
            right: createConditionRight(config.condition.right, resolveNodeRef),
          },
        ],
      },
    },
  ];
}

function assignOutputParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const inputParameters = createInputParameters(config, resolveNodeRef);
  params['inputs.inputParameters'] = inputParameters;
  const content = getContent(config);
  if (content !== undefined) {
    params['inputs.content'] = content;
  }
  const streamingOutput = config.streaming_output ?? config.streamingOutput;
  if (streamingOutput !== undefined) {
    params['inputs.streamingOutput'] = Boolean(streamingOutput);
  }
}

function assignTextParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const method = config.method === 'split' ? 'split' : 'concat';
  params.method = method;

  const inputParameters = createInputParameters(config, resolveNodeRef);
  params.inputParameters = inputParameters;

  if (method === 'split') {
    const delimiters = normalizeStringList(config.delimiter ?? '\n');
    params.delimiter = {
      value: delimiters,
      options: delimiters.map(value => ({
        label: value,
        value,
        isDefault: false,
      })),
    };
  } else {
    const content = getContent(config);
    const concatChar = config.concat_char ?? '';
    if (content !== undefined) {
      params.concatResult = content;
    }
    params.concatChar = {
      value: concatChar,
      options: [
        {
          label: concatChar || '无',
          value: concatChar,
          isDefault: true,
        },
      ],
    };
  }

  const outputs = createOutputs(config);
  params.outputs = outputs.length ? outputs : createDefaultTextOutputs(method);
}

function assignQuestionParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const inputParameters = createInputParameters(config, resolveNodeRef);
  if (inputParameters.length) {
    params.inputParameters = inputParameters;
  }

  const question = config.question ?? getContent(config);
  if (question !== undefined) {
    params['questionParams.question'] = question;
  }
  if (config.answer_type) {
    params['questionParams.answer_type'] = config.answer_type;
  }
  if (config.option_type) {
    params['questionParams.option_type'] = config.option_type;
  }
  if (config.options?.length) {
    params['questionParams.options'] = normalizeQuestionOptions(config.options);
  }
  if (config.limit !== undefined) {
    params['questionOutputs.limit'] = config.limit;
  }
  if (config.extra_output !== undefined) {
    params['questionOutputs.extra_output'] = Boolean(config.extra_output);
  }

  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
}

function assignInputParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
): void {
  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
}

function assignApiParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const apiParams = [
    createStringBlockInput('apiName', config.api_name),
    createStringBlockInput('pluginID', config.plugin_id),
    createStringBlockInput('apiID', config.api_id),
    createStringBlockInput('pluginName', config.plugin_name),
    createStringBlockInput('pluginVersion', config.plugin_version),
  ].filter(input => input.input.value.content);
  if (apiParams.length) {
    params['inputs.apiParam'] = apiParams;
  }

  const inputMap = createInputParameterMap(config, resolveNodeRef);
  if (Object.keys(inputMap).length) {
    params['inputs.inputParameters'] = inputMap;
  }

  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
}

function assignDatasetParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const queryBinding = normalizeBindings(config.input)[0];
  if (queryBinding) {
    params['inputs.inputParameters.Query'] = createRefExpression(
      queryBinding,
      resolveNodeRef,
    );
  }

  const datasetIds =
    config.dataset_ids ?? (config.dataset_id ? [config.dataset_id] : []);
  if (datasetIds.length) {
    params['inputs.datasetParameters.datasetParam'] = datasetIds;
  }
  params['inputs.datasetParameters.datasetSetting'] = {
    top_k: config.top_k ?? 5,
    min_score: undefined,
    strategy: undefined,
    use_nl2sql: undefined,
    use_rerank: false,
    use_rewrite: false,
    is_personal_only: false,
  };

  const outputs = createOutputs(config);
  params.outputs = outputs.length ? outputs : createDefaultDatasetOutputs();
}

function assignVariableMergeParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const mergeGroups =
    config.merge_groups ??
    (config.variables?.length
      ? [{ name: 'output', variables: config.variables }]
      : []);
  if (!mergeGroups.length) {
    return;
  }

  params['inputs.mergeGroups'] = mergeGroups.map((group, index) => ({
    name: group.name || `Group${index + 1}`,
    variables: group.variables.map(variable =>
      createRefValueExpression(variable, resolveNodeRef),
    ),
  }));

  const outputs = createOutputs(config);
  params.outputs = outputs.length
    ? outputs
    : mergeGroups.map(group => ({
        key: nanoid(),
        name: group.name || 'output',
        type: normalizeViewType(group.variables[0]?.type),
      }));
}

function assignJsonStringifyParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  const inputParameters = createInputParameters(config, resolveNodeRef);
  if (inputParameters.length) {
    params['inputs.inputParameters'] = inputParameters;
  }
  const outputs = createOutputs(config);
  params.outputs = outputs.length
    ? outputs
    : createDefaultJsonStringifyOutputs();
}

function assignCardSelectorParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  assignOutputParams(params, config, resolveNodeRef);
  const selectedCard =
    config.selected_card ??
    (config.card_id
      ? {
          cardId: config.card_id,
          cardName: config.card_name,
          code: config.card_code,
        }
      : undefined);
  if (selectedCard) {
    params['inputs.selectedCard'] = selectedCard;
  }
}

function assignAgentParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): void {
  if (config.agent_id) {
    params['inputs.agent_id'] = config.agent_id;
  }
  if (config.agent_name) {
    params['inputs.agent_name'] = config.agent_name;
  }
  if (config.platform) {
    params['inputs.platform'] = config.platform;
  }
  params['inputs.query'] = config.query ?? '';

  const inputParameters = createInputParameters(config, resolveNodeRef).map(
    input => ({
      ...input,
      name: input.name || 'query',
    }),
  );
  if (inputParameters.length) {
    const queryInput =
      inputParameters.find(input => input.name === 'query') ??
      inputParameters[0];
    params['inputs.inputParameters'] = [
      {
        ...queryInput,
        name: 'query',
      },
    ];
    const dynamicInputs = inputParameters.filter(input => input !== queryInput);
    if (dynamicInputs.length) {
      params['inputs.dynamicInputs'] = dynamicInputs;
    }
  }
  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
}

// 数据库条件操作符必须是后端认的大写串(EQUAL/NOT_EQUAL/...);把智能体可能给的
// 小写/符号/缩写都归一,避免 test_run 报 "not a valid Operation string"。
function normalizeDatabaseOperator(op?: string): string {
  if (!op) {
    return 'EQUAL';
  }
  const u = op.trim().toUpperCase().replace(/\s+/g, '_');
  const map: Record<string, string> = {
    '=': 'EQUAL',
    '==': 'EQUAL',
    EQ: 'EQUAL',
    '!=': 'NOT_EQUAL',
    '<>': 'NOT_EQUAL',
    NE: 'NOT_EQUAL',
    NOTEQUAL: 'NOT_EQUAL',
    '>': 'GREATER_THAN',
    GT: 'GREATER_THAN',
    GREATERTHAN: 'GREATER_THAN',
    '<': 'LESS_THAN',
    LT: 'LESS_THAN',
    LESSTHAN: 'LESS_THAN',
    '>=': 'GREATER_EQUAL',
    GE: 'GREATER_EQUAL',
    GTE: 'GREATER_EQUAL',
    '<=': 'LESS_EQUAL',
    LE: 'LESS_EQUAL',
    LTE: 'LESS_EQUAL',
  };
  return map[u] ?? u;
}

function assignDatabaseParams(
  params: Record<string, unknown>,
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
  nodeType: string,
): void {
  const tableId = config.table_id ?? config.database_id;
  if (tableId !== undefined) {
    params['inputs.databaseInfoList'] = [{ databaseInfoID: String(tableId) }];
  }
  const buildConditionList = () =>
    (config.conditions ?? []).map(c => ({
      left: c.left,
      operator: normalizeDatabaseOperator(c.operator),
      right: createConditionRight(c.right, resolveNodeRef),
    }));
  const buildFieldInfo = () =>
    (config.values ?? []).map(v => ({
      fieldID: Number(v.field_id),
      fieldValue: createConditionRight(v.value, resolveNodeRef),
    }));

  if (nodeType === NODE_TYPE.DatabaseQuery) {
    if (config.fields?.length) {
      const fieldList = config.fields
        .map(f => Number(typeof f === 'object' ? f.field_id : f))
        .filter(id => Number.isFinite(id))
        .map(id => ({ fieldID: id, isDistinct: false }));
      if (fieldList.length) {
        params['inputs.selectParam.fieldList'] = fieldList;
      }
    }
    if (config.limit !== undefined) {
      params['inputs.selectParam.limit'] = config.limit;
    }
    if (config.conditions?.length) {
      params['inputs.selectParam.condition.conditionList'] =
        buildConditionList();
    }
    if (config.order_by?.length) {
      params['inputs.selectParam.orderByList'] = config.order_by.map(o => ({
        fieldID: Number(o.field_id),
        isAsc: o.asc ?? true,
      }));
    }
  } else if (nodeType === NODE_TYPE.DatabaseCreate) {
    params['inputs.insertParam.fieldInfo'] = buildFieldInfo();
  } else if (nodeType === NODE_TYPE.DatabaseUpdate) {
    params['inputs.updateParam.fieldInfo'] = buildFieldInfo();
    if (config.conditions?.length) {
      params['inputs.updateParam.condition.conditionList'] =
        buildConditionList();
    }
  } else if (nodeType === NODE_TYPE.DatabaseDelete) {
    if (config.conditions?.length) {
      params['inputs.deleteParam.condition.conditionList'] =
        buildConditionList();
    }
  } else if (nodeType === NODE_TYPE.DatabaseSQL) {
    if (config.sql !== undefined) {
      params['inputs.sql'] = config.sql;
    }
    const inputParameters = createInputParameters(config, resolveNodeRef);
    if (inputParameters.length) {
      params['inputs.inputParameters'] = inputParameters;
    }
  }
  const outputs = createOutputs(config);
  if (outputs.length) {
    params.outputs = outputs;
  }
}

function createInputParameters(
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): InputValueVO[] {
  return normalizeBindings(config.inputs ?? config.input).map(binding => ({
    name: binding.name || 'input',
    input: createRefExpression(binding, resolveNodeRef),
  }));
}

function createInputParameterMap(
  config: WorkflowAgentSemanticConfig,
  resolveNodeRef: (ref: string) => string,
): Record<string, RefExpression> {
  return normalizeBindings(config.inputs ?? config.input).reduce<
    Record<string, RefExpression>
  >((acc, binding) => {
    acc[binding.name || 'input'] = createRefExpression(binding, resolveNodeRef);
    return acc;
  }, {});
}

function normalizeBindings(
  value:
    | VariableBindingConfig
    | VariableBindingConfig[]
    | Record<string, VariableBindingConfig>
    | undefined,
): VariableBindingConfig[] {
  if (!value) {
    return [];
  }
  if (Array.isArray(value)) {
    return value;
  }
  if (isBindingConfig(value)) {
    return [value];
  }
  return Object.entries(value).map(([name, binding]) => ({
    ...binding,
    name: binding.name || name,
  }));
}

function createOutputs(config: WorkflowAgentSemanticConfig): OutputValueVO[] {
  const outputs =
    config.outputs ??
    (typeof config.output === 'string'
      ? [config.output]
      : config.output
        ? [config.output]
        : []);
  return outputs.map(output => {
    const normalized =
      typeof output === 'string' ? { name: output, type: 'string' } : output;
    return {
      key: nanoid(),
      name: normalized.name,
      type: normalizeViewType(normalized.type),
      description: normalized.description,
    };
  });
}

function createDefaultDatasetOutputs(): OutputValueVO[] {
  return [
    {
      key: nanoid(),
      name: 'outputList',
      type: VIEW_VARIABLE_TYPE.ArrayObject,
      children: [
        {
          key: nanoid(),
          name: 'output',
          type: VIEW_VARIABLE_TYPE.String,
        },
      ],
    },
  ];
}

function createDefaultTextOutputs(method: 'concat' | 'split'): OutputValueVO[] {
  return [
    {
      key: nanoid(),
      name: 'output',
      type:
        method === 'split'
          ? VIEW_VARIABLE_TYPE.ArrayString
          : VIEW_VARIABLE_TYPE.String,
    },
  ];
}

function createDefaultJsonStringifyOutputs(): OutputValueVO[] {
  return [
    {
      key: nanoid(),
      name: 'output',
      type: VIEW_VARIABLE_TYPE.String,
    },
  ];
}

function createStringBlockInput(name: string, value = '') {
  return {
    name,
    input: {
      type: 'string',
      value: {
        type: 'literal',
        content: value,
        rawMeta: { type: VIEW_VARIABLE_TYPE.String },
      },
    },
  };
}

function createRefExpression(
  binding: VariableBindingConfig,
  resolveNodeRef: (ref: string) => string,
): RefExpression {
  return createRefValueExpression(binding, resolveNodeRef);
}

function createRefValueExpression(
  binding: VariableBindingConfig,
  resolveNodeRef: (ref: string) => string,
): RefExpression {
  const source = binding.from ?? binding.node ?? binding.node_tag ?? 'start';
  const output = binding.output ?? binding.variable ?? 'output';
  const viewType = normalizeViewType(binding.type);
  return {
    type: VALUE_EXPRESSION_TYPE.REF,
    content: {
      keyPath: [resolveNodeRef(source), output],
    },
    rawMeta: { type: viewType },
  };
}

function createLiteralExpression(
  value: string | number | boolean | Array<unknown>,
): LiteralExpression {
  const viewType = normalizeLiteralType(value);
  return {
    type: VALUE_EXPRESSION_TYPE.LITERAL,
    content: value,
    rawMeta: { type: viewType },
  };
}

function createConditionRight(
  value: ConditionConfig['right'],
  resolveNodeRef: (ref: string) => string,
): ValueExpression | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (isBindingConfig(value)) {
    return createRefExpression(value, resolveNodeRef);
  }
  return createLiteralExpression(value);
}

function normalizeConditionOperator(
  operator: string | number | undefined,
): number {
  if (typeof operator === 'number') {
    return operator;
  }
  if (!operator) {
    return conditionOperatorByName.equal;
  }
  return conditionOperatorByName[operator] ?? conditionOperatorByName.equal;
}

function normalizeViewType(
  type?: VariableTypeName | ViewVariableTypeValue,
): ViewVariableTypeValue {
  if (typeof type === 'number') {
    return type;
  }
  if (type && type in viewTypeByName) {
    return viewTypeByName[type];
  }
  return VIEW_VARIABLE_TYPE.String;
}

function normalizeLiteralType(
  value: string | number | boolean | Array<unknown>,
): ViewVariableTypeValue {
  if (Array.isArray(value)) {
    return VIEW_VARIABLE_TYPE.ArrayObject;
  }
  if (typeof value === 'boolean') {
    return VIEW_VARIABLE_TYPE.Boolean;
  }
  if (typeof value === 'number') {
    return Number.isInteger(value)
      ? VIEW_VARIABLE_TYPE.Integer
      : VIEW_VARIABLE_TYPE.Number;
  }
  return VIEW_VARIABLE_TYPE.String;
}

function isBindingConfig(value: unknown): value is VariableBindingConfig {
  return (
    isRecord(value) &&
    ('from' in value ||
      'node' in value ||
      'node_tag' in value ||
      'output' in value)
  );
}

function getContent(config: WorkflowAgentSemanticConfig): string | undefined {
  return config.content ?? config.text ?? config.template ?? config.prompt;
}

function normalizeStringList(value: string | string[]): string[] {
  return Array.isArray(value) ? value : [value];
}

function normalizeQuestionOptions(
  options: NonNullable<WorkflowAgentSemanticConfig['options']>,
): Array<Record<string, string>> {
  return options.map((option, index) => {
    if (typeof option === 'string') {
      return {
        id: String(index + 1),
        name: option,
        content: option,
      };
    }
    const name = option.name ?? option.content ?? String(index + 1);
    return {
      id: option.id ?? String(index + 1),
      name,
      content: option.content ?? name,
    };
  });
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}
