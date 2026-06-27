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

interface InputValueVO {
  name?: string;
  input: unknown;
}

interface OutputConfig {
  name: string;
  type?: string | number;
}

export interface WorkflowAgentBindableVariable {
  nodeId: string;
  nodeTitle: string;
  nodeType: string;
  name: string;
  type: string;
  ref: string;
}

interface CanvasNodeLike {
  id?: string;
  type?: string;
  blocks?: CanvasNodeLike[];
  edges?: CanvasEdgeLike[];
  data?: {
    nodeMeta?: { title?: string };
    condition?: unknown;
    inputs?: Record<string, unknown>;
    outputs?: unknown;
  };
}

interface CanvasEdgeLike {
  sourceNodeID?: string;
  targetNodeID?: string;
  source?: string;
  target?: string;
}

interface CanvasLike {
  nodes?: CanvasNodeLike[];
  blocks?: CanvasNodeLike[];
  edges?: CanvasEdgeLike[];
  workflow?: CanvasLike;
}

export interface WorkflowAgentCanvasValidationError {
  nodeId: string;
  targetNodeId?: string;
  errorInfo: string;
  errorLevel?: string;
  errorType?: 'node' | 'line';
}

export interface WorkflowAgentCanvasBindingDiagnostic {
  nodeId: string;
  message: string;
}

export interface WorkflowAgentCanvasSummaryOptions {
  validationErrors?: Record<string, WorkflowAgentCanvasValidationError[]>;
}

const viewTypeByName: Record<string, ViewVariableTypeValue> = {
  string: VIEW_VARIABLE_TYPE.String,
  integer: VIEW_VARIABLE_TYPE.Integer,
  boolean: VIEW_VARIABLE_TYPE.Boolean,
  number: VIEW_VARIABLE_TYPE.Number,
  object: VIEW_VARIABLE_TYPE.Object,
  array_string: VIEW_VARIABLE_TYPE.ArrayString,
  array_object: VIEW_VARIABLE_TYPE.ArrayObject,
};

const typeLabelByViewType: Partial<Record<ViewVariableTypeValue, string>> = {
  [VIEW_VARIABLE_TYPE.String]: 'string',
  [VIEW_VARIABLE_TYPE.Integer]: 'integer',
  [VIEW_VARIABLE_TYPE.Boolean]: 'boolean',
  [VIEW_VARIABLE_TYPE.Number]: 'number',
  [VIEW_VARIABLE_TYPE.Object]: 'object',
  [VIEW_VARIABLE_TYPE.ArrayString]: 'array_string',
  [VIEW_VARIABLE_TYPE.ArrayObject]: 'array_object',
};

const conditionOperatorLabel: Record<string, string> = {
  '1': 'equal',
  '2': 'not_equal',
  '3': 'length_gt',
  '4': 'length_gte',
  '5': 'length_lt',
  '6': 'length_lte',
  '7': 'contains',
  '8': 'not_contains',
  '9': 'null',
  '10': 'not_null',
  '11': 'true',
  '12': 'false',
  '13': 'gt',
  '14': 'gte',
  '15': 'lt',
  '16': 'lte',
};

export function summarizeWorkflowAgentCanvas(
  canvas: unknown,
  options: WorkflowAgentCanvasSummaryOptions = {},
): string {
  const graph = isRecord(canvas) ? (canvas as CanvasLike) : {};
  const nodes = collectCanvasNodes(graph);
  const edges = collectCanvasEdges(graph);
  const nodeLabelById = new Map(
    nodes.map(node => {
      const id = node.id || 'unknown';
      return [id, formatNodeLabel(node)] as const;
    }),
  );

  const nodeLines = nodes.map(node => {
    const outputs = getOutputConfigs(node.data?.outputs)
      .map(output => `${output.name}:${formatViewType(output.type)}`)
      .join(', ');
    const inputs = extractInputParameters(node.data?.inputs)
      .map(input => `${input.name || 'input'}=${formatExpression(input.input)}`)
      .join(', ');
    const returns = String(node.type) === '2' ? inputs : '';
    const mergeGroups = extractMergeGroupSummaries(node.data?.inputs).join('; ');
    const content = extractNodeContentSummary(node.data?.inputs);
    const conditions = extractConditionSummaries(node.data).join('; ');
    return [
      formatNodeLabel(node),
      `outputs: ${outputs || 'none'}`,
      inputs && !returns ? `inputs: ${inputs}` : undefined,
      returns ? `returns: ${returns}` : undefined,
      content ? `content: ${content}` : undefined,
      mergeGroups ? `merge_groups: ${mergeGroups}` : undefined,
      conditions ? `conditions: ${conditions}` : undefined,
    ]
      .filter(Boolean)
      .join(' ');
  });

  const edgeLines = edges
    .map(edge => {
      const source = edge.sourceNodeID || edge.source;
      const target = edge.targetNodeID || edge.target;
      const sourcePort =
        (edge as { sourcePortID?: string; sourcePort?: string; fromPort?: string })
          .sourcePortID ||
        (edge as { sourcePortID?: string; sourcePort?: string; fromPort?: string })
          .sourcePort ||
        (edge as { sourcePortID?: string; sourcePort?: string; fromPort?: string })
          .fromPort;
      return source && target
        ? `${source}${sourcePort ? `(${sourcePort})` : ''}->${target}`
        : '';
    })
    .filter(Boolean);

  const bindable = collectWorkflowAgentBindableVariables(canvas)
    .map(variable => `${variable.ref}:${variable.type}`)
    .join(', ');
  const bindingDiagnostics = collectWorkflowAgentBindingDiagnostics(canvas);
  const nonBindableOutputNodes = nodes
    .filter(
      node =>
        String(node.type) === '13' &&
        !getOutputConfigs(node.data?.outputs).length,
    )
    .map(formatNodeLabel);

  return [
    nodeLines.length ? `节点:\n${nodeLines.join('\n')}` : '节点: none',
    edgeLines.length ? `连线: ${edgeLines.join(', ')}` : '连线: none',
    `可绑定变量: ${bindable || 'none'}`,
    formatBindingDiagnosticSection(bindingDiagnostics),
    nonBindableOutputNodes.length
      ? `不可作为聚合变量的输出节点: ${nonBindableOutputNodes.join(', ')}; type=13 是消息展示节点,若分支结果需要被变量聚合/End 返回,请在分支末尾使用 type=15 文本处理节点产出 output:string。`
      : undefined,
    formatValidationErrorSection(options.validationErrors, nodeLabelById),
  ]
    .filter(Boolean)
    .join('\n');
}

export function collectWorkflowAgentBindableVariables(
  canvas: unknown,
): WorkflowAgentBindableVariable[] {
  const graph = isRecord(canvas) ? (canvas as CanvasLike) : {};
  return collectCanvasNodes(graph).flatMap(node =>
    getOutputConfigs(node.data?.outputs).map(output => ({
      nodeId: node.id || 'unknown',
      nodeTitle: node.data?.nodeMeta?.title || node.id || 'unknown',
      nodeType: node.type || 'unknown',
      name: output.name,
      type: formatViewType(output.type),
      ref: `${node.id || 'unknown'}.${output.name}`,
    })),
  );
}

export function formatWorkflowAgentBindableVariables(canvas: unknown): string {
  const variables = collectWorkflowAgentBindableVariables(canvas);
  if (!variables.length) {
    return '可绑定变量: none';
  }
  return [
    '可绑定变量:',
    ...variables.map(
      variable =>
        `- ${variable.ref}:${variable.type} (${variable.nodeTitle}, type=${variable.nodeType})`,
    ),
    '绑定格式示例: {"from":"节点node_tag或真实nodeId","output":"变量名","name":"本节点入参名"}。只能绑定这里列出的变量或本轮已 add_node 并声明 outputs 的变量;type=13 输出/消息节点若没有 outputs 不会出现在此列表,需要下游消费时改用 type=15 文本处理节点 output。',
  ].join('\n');
}

export function collectWorkflowAgentBindingDiagnostics(
  canvas: unknown,
): WorkflowAgentCanvasBindingDiagnostic[] {
  const graph = isRecord(canvas) ? (canvas as CanvasLike) : {};
  const nodes = collectCanvasNodes(graph);
  const bindableRefs = new Set(
    collectWorkflowAgentBindableVariables(canvas).map(variable => variable.ref),
  );
  const nodeById = new Map(nodes.map(node => [node.id || '', node]));
  const diagnostics: WorkflowAgentCanvasBindingDiagnostic[] = [];

  const addInvalidRef = (
    node: CanvasNodeLike,
    location: string,
    ref: string,
  ) => {
    if (bindableRefs.has(ref)) {
      return;
    }
    diagnostics.push({
      nodeId: node.id || 'unknown',
      message: `${formatNodeLabel(node)} ${location} 引用 ${ref} 不可绑定; ${formatInvalidRefAdvice(ref, nodeById)}`,
    });
  };

  nodes.forEach(node => {
    extractInputParameters(node.data?.inputs).forEach(input => {
      const ref = extractRefPath(input.input);
      if (ref) {
        addInvalidRef(
          node,
          String(node.type) === '2'
            ? `returns.${input.name || 'output'}`
            : `inputs.${input.name || 'input'}`,
          ref,
        );
      }
    });

    extractMergeGroupRefPaths(node.data?.inputs).forEach(({ groupName, ref }) => {
      addInvalidRef(node, `merge_groups.${groupName}`, ref);
    });

    extractConditionRefPaths(node.data).forEach(ref => {
      addInvalidRef(node, 'condition', ref);
    });

    extractMissingContentBindings(node.data?.inputs).forEach(name => {
      diagnostics.push({
        nodeId: node.id || 'unknown',
        message: `${formatNodeLabel(node)} content 引用了 {{${name}}},但本节点 input/inputs 未声明 ${name}; 请先配置 inputParameters 再引用模板变量,固定文案不要使用 {{}}。`,
      });
    });
  });

  return diagnostics;
}

function collectCanvasNodes(graph: CanvasLike): CanvasNodeLike[] {
  const roots =
    firstArray<CanvasNodeLike>(graph.nodes, graph.blocks, graph.workflow?.nodes, graph.workflow?.blocks) ??
    [];
  const result: CanvasNodeLike[] = [];
  const visit = (node: CanvasNodeLike) => {
    result.push(node);
    node.blocks?.forEach(visit);
  };
  roots.forEach(visit);
  return result;
}

function collectCanvasEdges(graph: CanvasLike): CanvasEdgeLike[] {
  const edges = [
    ...(firstArray<CanvasEdgeLike>(graph.edges, graph.workflow?.edges) ?? []),
  ];
  const nodes =
    firstArray<CanvasNodeLike>(graph.nodes, graph.blocks, graph.workflow?.nodes, graph.workflow?.blocks) ??
    [];
  nodes.forEach(node => {
    if (Array.isArray(node.edges)) {
      edges.push(...node.edges);
    }
  });
  return edges;
}

function firstArray<T>(...values: Array<unknown>): T[] | undefined {
  return values.find(Array.isArray) as T[] | undefined;
}

function formatNodeLabel(node: CanvasNodeLike): string {
  const id = node.id || 'unknown';
  const title = node.data?.nodeMeta?.title || id;
  const type = node.type || 'unknown';
  return `${title}(${id},type=${type})`;
}

function formatValidationErrorSection(
  errors: WorkflowAgentCanvasSummaryOptions['validationErrors'],
  nodeLabelById: Map<string, string>,
): string {
  if (!errors) {
    return '';
  }
  const lines = Object.entries(errors)
    .flatMap(([nodeId, nodeErrors]) =>
      (nodeErrors ?? []).map(error => {
        const label =
          nodeLabelById.get(error.nodeId || nodeId) ??
          `${error.nodeId || nodeId}(type=unknown)`;
        const target = error.targetNodeId
          ? ` -> ${nodeLabelById.get(error.targetNodeId) ?? error.targetNodeId}`
          : '';
        const level = error.errorLevel || 'error';
        const kind = error.errorType === 'line' ? 'line ' : '';
        return `[${level}] ${kind}${label}${target}: ${error.errorInfo}`;
      }),
    )
    .filter(Boolean)
    .slice(0, 20);

  return lines.length
    ? `当前校验错误:\n${lines.join('\n')}`
    : '当前校验错误: none';
}

function formatBindingDiagnosticSection(
  diagnostics: WorkflowAgentCanvasBindingDiagnostic[],
): string {
  return diagnostics.length
    ? `绑定诊断:\n${diagnostics.map(item => item.message).join('\n')}`
    : '绑定诊断: none';
}

function formatViewType(type?: unknown): string {
  const viewType =
    typeof type === 'number'
      ? type
      : typeof type === 'string' && type in viewTypeByName
        ? viewTypeByName[type]
        : VIEW_VARIABLE_TYPE.String;
  return typeLabelByViewType[viewType] || String(type || 'string');
}

function formatExpression(input: unknown): string {
  if (!isRecord(input)) {
    return input === undefined || input === null ? 'unset' : String(input);
  }
  if ('input' in input) {
    return formatExpression(input.input);
  }
  if (isRecord(input.value)) {
    return formatExpression(input.value);
  }
  if (input.type === VALUE_EXPRESSION_TYPE.REF && isRecord(input.content)) {
    const { keyPath, blockID, name } = input.content;
    return Array.isArray(keyPath) && keyPath.length
      ? keyPath.join('.')
      : blockID && name
        ? `${blockID}.${name}`
        : 'unset-ref';
  }
  if (input.type === VALUE_EXPRESSION_TYPE.LITERAL) {
    return String(input.content ?? '');
  }
  if (input.type === 'object_ref') {
    return stringifyBrief(input.content);
  }
  return String(input.type || 'unknown');
}

function extractInputParameters(inputs: unknown): InputValueVO[] {
  if (!isRecord(inputs)) {
    return [];
  }
  const { inputParameters } = inputs;
  const result: InputValueVO[] = [];
  if (Array.isArray(inputParameters)) {
    result.push(
      ...(inputParameters.filter(isRecord).map(input => ({
        name: typeof input.name === 'string' ? input.name : undefined,
        input: 'input' in input ? input.input : input,
      })) as InputValueVO[]),
    );
  } else if (isRecord(inputParameters)) {
    Object.entries(inputParameters).forEach(([name, input]) => {
      result.push({ name, input });
    });
  }

  if (inputs.query !== undefined) {
    result.push({ name: 'query', input: inputs.query });
  }

  return result;
}

function extractMergeGroupSummaries(inputs: unknown): string[] {
  if (!isRecord(inputs) || !Array.isArray(inputs.mergeGroups)) {
    return [];
  }
  return inputs.mergeGroups
    .filter(isRecord)
    .map((group, index) => {
      const name =
        typeof group.name === 'string' && group.name
          ? group.name
          : `Group${index + 1}`;
      const variables = Array.isArray(group.variables)
        ? group.variables.map(formatExpression).filter(Boolean)
        : [];
      return `${name}=[${variables.join(', ')}]`;
    });
}

function extractMergeGroupRefPaths(
  inputs: unknown,
): Array<{ groupName: string; ref: string }> {
  if (!isRecord(inputs) || !Array.isArray(inputs.mergeGroups)) {
    return [];
  }
  return inputs.mergeGroups.filter(isRecord).flatMap((group, index) => {
    const groupName =
      typeof group.name === 'string' && group.name
        ? group.name
        : `Group${index + 1}`;
    return Array.isArray(group.variables)
      ? group.variables
          .map(variable => extractRefPath(variable))
          .filter((ref): ref is string => Boolean(ref))
          .map(ref => ({ groupName, ref }))
      : [];
  });
}

function extractConditionRefPaths(
  data: CanvasNodeLike['data'] | undefined,
): string[] {
  const branches = Array.isArray(data?.condition)
    ? data?.condition
    : isRecord(data?.inputs) && Array.isArray(data.inputs.branches)
      ? data.inputs.branches
      : [];

  return branches
    .filter(isRecord)
    .flatMap(branch => {
      const condition = isRecord(branch.condition) ? branch.condition : {};
      const conditions = Array.isArray(condition.conditions)
        ? condition.conditions
        : [];
      return conditions.filter(isRecord).flatMap(item =>
        [extractRefPath(item.left), extractRefPath(item.right)].filter(
          (ref): ref is string => Boolean(ref),
        ),
      );
    });
}

function extractNodeContentSummary(inputs: unknown): string {
  if (!isRecord(inputs)) {
    return '';
  }
  const directContent = formatContentValue(inputs.content);
  if (directContent) {
    return directContent;
  }
  const directConcat = formatContentValue(inputs.concatResult);
  if (directConcat) {
    return directConcat;
  }
  if (Array.isArray(inputs.concatParams)) {
    const concat = inputs.concatParams
      .filter(isRecord)
      .find(param => param.name === 'concatResult');
    const content = formatContentValue(concat?.input);
    if (content) {
      return content;
    }
  }
  return '';
}

function extractMissingContentBindings(inputs: unknown): string[] {
  const content = extractNodeContentSummary(inputs);
  if (!content) {
    return [];
  }
  const inputNames = new Set(
    extractInputParameters(inputs)
      .map(input => input.name)
      .filter((name): name is string => Boolean(name)),
  );
  const placeholders = Array.from(content.matchAll(/\{\{\s*([\w.-]+)\s*\}\}/g))
    .map(match => match[1]?.split('.')[0])
    .filter((name): name is string => Boolean(name));
  return Array.from(new Set(placeholders)).filter(name => !inputNames.has(name));
}

function extractRefPath(value: unknown): string | undefined {
  if (!isRecord(value)) {
    return undefined;
  }
  if ('input' in value) {
    return extractRefPath(value.input);
  }
  if (isRecord(value.value)) {
    return extractRefPath(value.value);
  }
  if (value.type === VALUE_EXPRESSION_TYPE.REF && isRecord(value.content)) {
    const { keyPath, blockID, name } = value.content;
    if (Array.isArray(keyPath) && keyPath.length >= 2) {
      return `${keyPath[0]}.${keyPath[1]}`;
    }
    if (typeof blockID === 'string' && typeof name === 'string') {
      return `${blockID}.${name}`;
    }
  }
  return undefined;
}

function formatInvalidRefAdvice(
  ref: string,
  nodeById: Map<string, CanvasNodeLike>,
): string {
  const [sourceNodeId] = ref.split('.');
  const sourceNode = sourceNodeId ? nodeById.get(sourceNodeId) : undefined;
  if (
    sourceNode &&
    String(sourceNode.type) === '13' &&
    !getOutputConfigs(sourceNode.data?.outputs).length
  ) {
    return `${formatNodeLabel(sourceNode)} 是消息展示节点且没有真实 outputs; 固定文案请改用 type=15 文本处理节点产出 output:string。`;
  }
  return '请先确认上游节点声明 outputs,并使用 workflow_canvas_get_bindable_variables 返回的变量名。';
}

function formatContentValue(value: unknown): string {
  if (typeof value === 'string') {
    return value.length > 180 ? `${value.slice(0, 180)}...` : value;
  }
  if (!isRecord(value)) {
    return '';
  }
  if (isRecord(value.value)) {
    return formatContentValue(value.value);
  }
  if (value.type === VALUE_EXPRESSION_TYPE.LITERAL) {
    return formatContentValue(value.content);
  }
  if ('content' in value && typeof value.content === 'string') {
    return formatContentValue(value.content);
  }
  return '';
}

function getOutputConfigs(outputs: unknown): OutputConfig[] {
  if (!Array.isArray(outputs)) {
    return [];
  }
  return outputs.filter(isRecord).map(output => ({
    name: typeof output.name === 'string' ? output.name : 'output',
    type:
      typeof output.type === 'number' || typeof output.type === 'string'
        ? output.type
        : 'string',
  }));
}

function extractConditionSummaries(
  data: CanvasNodeLike['data'] | undefined,
): string[] {
  const branches = Array.isArray(data?.condition)
    ? data?.condition
    : isRecord(data?.inputs) && Array.isArray(data.inputs.branches)
      ? data.inputs.branches
      : [];

  return branches
    .filter(isRecord)
    .map((branch, branchIndex) => {
      const condition = isRecord(branch.condition) ? branch.condition : {};
      const conditions = Array.isArray(condition.conditions)
        ? condition.conditions
        : [];
      const logic = condition.logic === 1 ? ' OR ' : ' AND ';
      const parts = conditions
        .filter(isRecord)
        .map(item => {
          const left = formatExpression(item.left);
          const operator = formatConditionOperator(item.operator);
          const right = 'right' in item ? formatExpression(item.right) : '';
          return [left, operator, right].filter(Boolean).join(' ');
        })
        .filter(Boolean);
      if (!parts.length) {
        return '';
      }
      return `${branchIndex === 0 ? 'IF' : 'ELSE IF'} ${parts.join(logic)}`;
    })
    .filter(Boolean);
}

function formatConditionOperator(operator: unknown): string {
  if (operator === undefined || operator === null) {
    return 'unset_operator';
  }
  const key = String(operator);
  return conditionOperatorLabel[key] ?? key;
}

function stringifyBrief(value: unknown): string {
  try {
    const text = JSON.stringify(value);
    return text.length > 120 ? `${text.slice(0, 120)}...` : text;
  } catch {
    return 'object';
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}
