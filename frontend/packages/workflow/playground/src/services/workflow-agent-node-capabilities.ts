import { WORKFLOW_AGENT_SEMANTIC_CONFIG_NODE_TYPES } from './workflow-agent-semantic-config';
import { WORKFLOW_AGENT_NODE_CATALOG } from './workflow-agent-node-catalog';

export type WorkflowAgentNodeSupportLevel =
  | 'singleton'
  | 'full'
  | 'resource-bound'
  | 'partial'
  | 'add-only'
  | 'documentation-only';

export interface WorkflowAgentNodeCapability {
  type: string;
  name: string;
  registry: string;
  supportLevel: WorkflowAgentNodeSupportLevel;
  canAdd: boolean;
  canConfigureSemantically: boolean;
  hasCatalogGuidance: boolean;
  hasBackendSpec: boolean;
  requiresResource?: boolean;
  runtimeSmoke?: 'local' | 'resource' | 'sub-canvas' | 'not-executable';
  gaps?: string[];
}

export interface WorkflowAgentNodeCapabilityAudit {
  total: number;
  capabilities: WorkflowAgentNodeCapability[];
  full: string[];
  partial: string[];
  addOnly: string[];
  missingCapabilities: string[];
  missingCatalogGuidance: string[];
  missingSemanticConfig: string[];
  missingBackendSpec: string[];
  registryDiscrepancies: string[];
}

export interface WorkflowAgentNodeTypeLike {
  type: string;
  title?: string;
}

export const WORKFLOW_AGENT_BACKEND_SPEC_NODE_TYPES = [
  '2',
  '3',
  '4',
  '5',
  '6',
  '8',
  '13',
  '15',
  '18',
  '22',
  '30',
  '32',
  '45',
  '58',
  '59',
  '99',
  '100',
] as const;

export const WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES = [
  '1',
  '2',
  '3',
  '4',
  '5',
  '6',
  '8',
  '9',
  '11',
  '12',
  '13',
  '14',
  '15',
  '16',
  '17',
  '18',
  '19',
  '20',
  '21',
  '22',
  '23',
  '26',
  '27',
  '28',
  '29',
  '30',
  '31',
  '32',
  '34',
  '35',
  '36',
  '42',
  '43',
  '44',
  '45',
  '46',
  '58',
  '59',
  '61',
  '99',
  '100',
] as const;

const SEMANTIC_CONFIG_TYPES = new Set<string>(
  WORKFLOW_AGENT_SEMANTIC_CONFIG_NODE_TYPES,
);
const BACKEND_SPEC_TYPES = new Set<string>(WORKFLOW_AGENT_BACKEND_SPEC_NODE_TYPES);

const capability = (
  item: Omit<
    WorkflowAgentNodeCapability,
    'canConfigureSemantically' | 'hasCatalogGuidance' | 'hasBackendSpec'
  >,
): WorkflowAgentNodeCapability => ({
  ...item,
  canConfigureSemantically: SEMANTIC_CONFIG_TYPES.has(item.type),
  hasCatalogGuidance: Boolean(WORKFLOW_AGENT_NODE_CATALOG[item.type]),
  hasBackendSpec: BACKEND_SPEC_TYPES.has(item.type),
});

export const WORKFLOW_AGENT_NODE_CAPABILITIES: WorkflowAgentNodeCapability[] = [
  capability({
    type: '1',
    name: '开始',
    registry: 'start',
    supportLevel: 'singleton',
    canAdd: false,
    runtimeSmoke: 'local',
    gaps: ['只能引用 start/100001,不能新增。'],
  }),
  capability({
    type: '2',
    name: '结束',
    registry: 'end',
    supportLevel: 'singleton',
    canAdd: false,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '3',
    name: '大模型',
    registry: 'nodes-v2/llm',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
  }),
  capability({
    type: '4',
    name: '插件/API',
    registry: 'plugin',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
  }),
  capability({
    type: '5',
    name: '代码',
    registry: 'code',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '6',
    name: '知识库检索',
    registry: 'dataset',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
  }),
  capability({
    type: '8',
    name: '条件分支',
    registry: 'if',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '9',
    name: '子工作流',
    registry: 'sub-workflow',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少语义化 configure_node 适配,需要读取子工作流 schema 后绑定。'],
  }),
  capability({
    type: '11',
    name: '变量',
    registry: 'variable',
    supportLevel: 'add-only',
    canAdd: true,
    runtimeSmoke: 'local',
    gaps: ['缺少变量节点语义配置器。'],
  }),
  capability({
    type: '12',
    name: 'SQL自定义',
    registry: 'database/database-base',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少 databaseInfo/sql/inputParameters 语义配置器。'],
  }),
  capability({
    type: '13',
    name: '输出/纯输出',
    registry: 'output',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '14',
    name: '图像流',
    registry: 'imageflow',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少图像工作流资源发现与语义配置器。'],
  }),
  capability({
    type: '15',
    name: '文本处理',
    registry: 'text-process',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '16',
    name: '生成图片',
    registry: 'image-generate',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少图片生成参数语义配置器。'],
  }),
  capability({
    type: '17',
    name: '图片引用',
    registry: 'image-reference',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少图片资源选择与绑定语义配置器。'],
  }),
  capability({
    type: '18',
    name: '问答/追问',
    registry: 'question',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '19',
    name: '跳出循环',
    registry: 'break',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'sub-canvas',
    gaps: ['只能在循环子画布语义下可靠测试。'],
  }),
  capability({
    type: '20',
    name: '变量赋值',
    registry: 'set-variable',
    supportLevel: 'add-only',
    canAdd: true,
    runtimeSmoke: 'local',
    gaps: ['缺少变量赋值语义配置器。'],
  }),
  capability({
    type: '21',
    name: '循环',
    registry: 'loop',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'sub-canvas',
    gaps: ['缺少循环数组、循环变量和子画布节点语义配置器。'],
  }),
  capability({
    type: '22',
    name: '意图识别',
    registry: 'intent',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'resource',
    gaps: ['当前规格建议用 LLM 替代,原生意图节点语义配置仍需补齐。'],
  }),
  capability({
    type: '23',
    name: '图片画布',
    registry: 'image-canvas',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少图像画布配置器。'],
  }),
  capability({
    type: '26',
    name: '长期记忆',
    registry: 'ltm',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少记忆资源发现与配置器。'],
  }),
  capability({
    type: '27',
    name: '知识库写入',
    registry: 'dataset-write',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少写入策略和字段绑定语义配置器。'],
  }),
  capability({
    type: '28',
    name: '批处理',
    registry: 'batch',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'sub-canvas',
    gaps: ['缺少批处理数组、并发和子画布语义配置器。'],
  }),
  capability({
    type: '29',
    name: '继续循环',
    registry: 'continue',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'sub-canvas',
    gaps: ['只能在循环子画布语义下可靠测试。'],
  }),
  capability({
    type: '30',
    name: '输入',
    registry: 'input',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '31',
    name: '注释',
    registry: 'comment',
    supportLevel: 'documentation-only',
    canAdd: true,
    runtimeSmoke: 'not-executable',
    gaps: ['注释节点不参与运行,不应被当作工作流执行节点。'],
  }),
  capability({
    type: '32',
    name: '变量聚合',
    registry: 'nodes-v2/variable-merge',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '34',
    name: '新增/更新触发器',
    registry: 'trigger-upsert',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少触发器资源配置器。'],
  }),
  capability({
    type: '35',
    name: '删除触发器',
    registry: 'trigger-delete',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少触发器删除配置器。'],
  }),
  capability({
    type: '36',
    name: '查询触发器',
    registry: 'trigger-read',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少触发器查询配置器。'],
  }),
  capability({
    type: '42',
    name: '更新数据',
    registry: 'database/database-update',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少数据库字段、条件和更新值语义配置器。'],
  }),
  capability({
    type: '43',
    name: '查询数据',
    registry: 'database/database-query',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少数据库字段、条件、排序和 limit 语义配置器。'],
  }),
  capability({
    type: '44',
    name: '删除数据',
    registry: 'database/database-delete',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少数据库条件语义配置器。'],
  }),
  capability({
    type: '45',
    name: 'HTTP请求',
    registry: 'http',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'local',
    gaps: ['后端有节点规格,前端缺少 method/url/headers/body/outputs 语义配置器。'],
  }),
  capability({
    type: '46',
    name: '新增数据',
    registry: 'database/database-create',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少数据库字段和值语义配置器。'],
  }),
  capability({
    type: '58',
    name: 'JSON序列化',
    registry: 'json-stringify',
    supportLevel: 'full',
    canAdd: true,
    runtimeSmoke: 'local',
  }),
  capability({
    type: '59',
    name: 'JSON解析',
    registry: 'json-parser',
    supportLevel: 'partial',
    canAdd: true,
    runtimeSmoke: 'local',
    gaps: ['后端有节点规格,前端缺少 JSON parser schema/outputs 语义配置器。'],
  }),
  capability({
    type: '61',
    name: 'MCP',
    registry: 'mcp',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
    gaps: ['缺少 MCP server/tool 资源发现和参数配置器。'],
  }),
  capability({
    type: '99',
    name: '卡片选择',
    registry: 'card-selector',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
  }),
  capability({
    type: '100',
    name: '智能体',
    registry: 'agent',
    supportLevel: 'resource-bound',
    canAdd: true,
    runtimeSmoke: 'resource',
    requiresResource: true,
  }),
];

const CAPABILITY_BY_TYPE = new Map(
  WORKFLOW_AGENT_NODE_CAPABILITIES.map(item => [item.type, item]),
);

export function getWorkflowAgentNodeCapability(
  type: string,
): WorkflowAgentNodeCapability | undefined {
  return CAPABILITY_BY_TYPE.get(type);
}

export function auditWorkflowAgentNodeCapabilities(
  availableNodes: WorkflowAgentNodeTypeLike[] = WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES.map(
    type => ({ type }),
  ),
): WorkflowAgentNodeCapabilityAudit {
  const availableTypes = availableNodes.map(node => String(node.type));
  const missingCapabilities = availableTypes.filter(
    type => !CAPABILITY_BY_TYPE.has(type),
  );
  const registeredTypes = new Set(WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES);
  const registryDiscrepancies = availableTypes.filter(
    type => !registeredTypes.has(type as never),
  );
  const capabilities = availableTypes
    .map(type => CAPABILITY_BY_TYPE.get(type))
    .filter(Boolean) as WorkflowAgentNodeCapability[];

  return {
    total: availableTypes.length,
    capabilities,
    full: capabilities
      .filter(item => item.supportLevel === 'full')
      .map(formatNodeName),
    partial: capabilities
      .filter(item =>
        ['partial', 'resource-bound'].includes(item.supportLevel),
      )
      .map(formatNodeName),
    addOnly: capabilities
      .filter(item =>
        ['add-only', 'documentation-only'].includes(item.supportLevel),
      )
      .map(formatNodeName),
    missingCapabilities,
    missingCatalogGuidance: capabilities
      .filter(item => !item.hasCatalogGuidance)
      .map(formatNodeName),
    missingSemanticConfig: capabilities
      .filter(
        item =>
          item.canAdd &&
          !item.canConfigureSemantically &&
          item.supportLevel !== 'documentation-only',
      )
      .map(formatNodeName),
    missingBackendSpec: capabilities
      .filter(
        item =>
          item.canAdd &&
          item.supportLevel !== 'documentation-only' &&
          !item.hasBackendSpec,
      )
      .map(formatNodeName),
    registryDiscrepancies,
  };
}

export function formatWorkflowAgentNodeCapabilityAudit(
  audit: WorkflowAgentNodeCapabilityAudit,
): string {
  return [
    `节点能力覆盖: total=${audit.total}, full=${audit.full.length}, partial=${audit.partial.length}, add_only=${audit.addOnly.length}`,
    audit.full.length
      ? `完整可配置节点: ${audit.full.join(', ')}`
      : '完整可配置节点: none',
    formatDetailedCapabilityGroup(
      '资源/部分支持节点',
      audit.capabilities.filter(item =>
        ['resource-bound', 'partial'].includes(item.supportLevel),
      ),
    ),
    formatDetailedCapabilityGroup(
      '只添加/文档节点',
      audit.capabilities.filter(item =>
        ['add-only', 'documentation-only'].includes(item.supportLevel),
      ),
    ),
    audit.missingCapabilities.length
      ? `未登记节点: ${audit.missingCapabilities.join(', ')}`
      : '未登记节点: none',
    audit.missingCatalogGuidance.length
      ? `缺少节点目录说明: ${audit.missingCatalogGuidance.join(', ')}`
      : '缺少节点目录说明: none',
    audit.missingSemanticConfig.length
      ? `缺少语义配置器: ${audit.missingSemanticConfig.join(', ')}`
      : '缺少语义配置器: none',
    audit.missingBackendSpec.length
      ? `缺少渐进式节点规格: ${audit.missingBackendSpec.join(', ')}`
      : '缺少渐进式节点规格: none',
    audit.registryDiscrepancies.length
      ? `运行时出现但静态注册表未登记: ${audit.registryDiscrepancies.join(', ')}`
      : '运行时出现但静态注册表未登记: none',
  ].join('\n');
}

function formatNodeName(item: WorkflowAgentNodeCapability): string {
  return `${item.type} ${item.name}`;
}

function formatDetailedCapabilityGroup(
  title: string,
  items: WorkflowAgentNodeCapability[],
): string {
  if (!items.length) {
    return `${title}: none`;
  }
  return [
    `${title}:`,
    ...items.map(item => {
      const gaps = item.gaps?.length ? item.gaps.join('; ') : '需要真实资源或运行时上下文验证。';
      return `- ${formatNodeName(item)}(${item.supportLevel}): ${gaps}`;
    }),
  ].join('\n');
}
