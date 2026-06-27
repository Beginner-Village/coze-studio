import {
  auditWorkflowAgentNodeCapabilities,
  formatWorkflowAgentNodeCapabilityAudit,
  getWorkflowAgentNodeCapability,
  WORKFLOW_AGENT_NODE_CAPABILITIES,
  WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES,
} from '../workflow-agent-node-capabilities';

describe('workflow-agent-node-capabilities', () => {
  it('registers every visible workflow node type in the agent capability matrix', () => {
    const audit = auditWorkflowAgentNodeCapabilities();

    expect(audit.total).toBe(WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES.length);
    expect(audit.missingCapabilities).toEqual([]);
    expect(WORKFLOW_AGENT_NODE_CAPABILITIES.map(item => item.type).sort()).toEqual(
      [...WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES].sort(),
    );
  });

  it('keeps current partial support explicit instead of silently claiming all nodes are complete', () => {
    const audit = auditWorkflowAgentNodeCapabilities();

    expect(audit.full).toEqual(
      expect.arrayContaining([
        '3 大模型',
        '5 代码',
        '8 条件分支',
        '13 输出/纯输出',
        '15 文本处理',
        '30 输入',
        '32 变量聚合',
        '58 JSON序列化',
      ]),
    );
    expect(audit.missingSemanticConfig).toEqual(
      expect.arrayContaining([
        '9 子工作流',
        '12 SQL自定义',
        '21 循环',
        '22 意图识别',
        '45 HTTP请求',
        '59 JSON解析',
        '61 MCP',
      ]),
    );
    expect(formatWorkflowAgentNodeCapabilityAudit(audit)).toContain(
      '缺少语义配置器',
    );
    expect(formatWorkflowAgentNodeCapabilityAudit(audit)).toContain(
      '完整可配置节点:',
    );
    expect(formatWorkflowAgentNodeCapabilityAudit(audit)).toContain(
      '资源/部分支持节点:',
    );
    expect(formatWorkflowAgentNodeCapabilityAudit(audit)).toContain(
      '45 HTTP请求(partial): 后端有节点规格,前端缺少 method/url/headers/body/outputs 语义配置器。',
    );
  });

  it('marks display-only and singleton nodes so agents do not misuse them as normal execution nodes', () => {
    expect(getWorkflowAgentNodeCapability('1')).toMatchObject({
      supportLevel: 'singleton',
      canAdd: false,
    });
    expect(getWorkflowAgentNodeCapability('31')).toMatchObject({
      supportLevel: 'documentation-only',
      runtimeSmoke: 'not-executable',
    });
  });

  it('reports runtime node types that are not yet part of the static registry contract', () => {
    const audit = auditWorkflowAgentNodeCapabilities([
      { type: '3', title: '大模型' },
      { type: '777', title: '实验节点' },
    ]);

    expect(audit.missingCapabilities).toEqual(['777']);
    expect(audit.registryDiscrepancies).toEqual(['777']);
  });
});
