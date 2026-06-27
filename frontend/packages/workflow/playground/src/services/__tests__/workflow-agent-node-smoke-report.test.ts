import { describe, expect, it } from 'vitest';

import {
  buildWorkflowAgentNodeSmokeReport,
  renderWorkflowAgentNodeSmokeReportMarkdown,
  summarizeWorkflowAgentNodeSmokeReport,
} from '../workflow-agent-node-smoke-report';
import { WORKFLOW_AGENT_NODE_CAPABILITIES } from '../workflow-agent-node-capabilities';
import { WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS } from '../workflow-agent-node-smoke-scenarios';

describe('workflow-agent-node-smoke-report', () => {
  it('builds a report from the node capability matrix and smoke scenarios', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: WORKFLOW_AGENT_NODE_CAPABILITIES,
      scenarios: WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
    });

    expect(report.total).toBe(WORKFLOW_AGENT_NODE_CAPABILITIES.length);
    expect(report.coverageGaps).toEqual([]);
    expect(report.counts.verifiedFull).toBeGreaterThan(0);
    expect(report.nodes.find(node => node.nodeType === '32')).toMatchObject({
      nodeName: '变量聚合',
      status: 'verified-full',
      assertions: expect.arrayContaining(['不能引用 type=13 输出节点。']),
      expectedBindableVariables: ['merge.output'],
    });
  });

  it('treats resource-bound skipped nodes as explicit gaps rather than passes', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: [
        {
          type: '4',
          name: '插件/API',
          registry: 'plugin',
          supportLevel: 'resource-bound',
          canAdd: true,
          canConfigureSemantically: true,
          hasCatalogGuidance: true,
          hasBackendSpec: true,
          requiresResource: true,
          runtimeSmoke: 'resource',
        },
      ],
      scenarios: [
        {
          nodeType: '4',
          nodeName: '插件/API',
          status: 'skipped-resource-missing',
          runtime: 'resource',
          requiredCommands: ['add_node', 'configure_node'],
          setupCommands: [],
          assertions: ['缺少真实 plugin_id 时必须跳过'],
          expectedBindableVariables: [],
          skipReason: '当前空间没有可用插件资源',
        },
      ],
    });

    expect(report.counts.verifiedFull).toBe(0);
    expect(report.counts.skipped).toBe(1);
    expect(report.nextActions).toContain('4 插件/API: 当前空间没有可用插件资源');
  });

  it('reports missing scenario coverage as a coverage gap', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: [WORKFLOW_AGENT_NODE_CAPABILITIES[0]],
      scenarios: [],
    });

    expect(report.coverageGaps).toEqual(['1 开始: 缺少节点 smoke 场景']);
    expect(report.counts.failing).toBe(1);
  });

  it('separates planned node support from actual execution evidence', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: [
        {
          type: '5',
          name: '代码',
          registry: 'code',
          supportLevel: 'full',
          canAdd: true,
          canConfigureSemantically: true,
          hasCatalogGuidance: true,
          hasBackendSpec: true,
          runtimeSmoke: 'local',
        },
      ],
      scenarios: [
        {
          nodeType: '5',
          nodeName: '代码',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: ['add_node', 'connect', 'configure_node', 'validate'],
          setupCommands: [],
          assertions: ['代码节点需要真实执行证据才能算跑过'],
          expectedBindableVariables: ['code.output'],
        },
      ],
    });

    expect(report.nodes[0]).toMatchObject({
      status: 'verified-full',
      executionStatus: 'pending',
    });
    expect(report.evidenceGaps).toEqual([
      '5 代码: 缺少 smoke 执行证据 add_node, connect, configure_node, validate',
    ]);
    expect(report.nextActions).toContain(
      '5 代码: 缺少 smoke 执行证据 add_node, connect, configure_node, validate',
    );
  });

  it('marks executable nodes as covered only when all required commands have evidence', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: [
        {
          type: '5',
          name: '代码',
          registry: 'code',
          supportLevel: 'full',
          canAdd: true,
          canConfigureSemantically: true,
          hasCatalogGuidance: true,
          hasBackendSpec: true,
          runtimeSmoke: 'local',
        },
      ],
      scenarios: [
        {
          nodeType: '5',
          nodeName: '代码',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: ['add_node', 'connect', 'configure_node', 'validate'],
          setupCommands: [],
          assertions: ['代码节点需要真实执行证据才能算跑过'],
          expectedBindableVariables: ['code.output'],
        },
      ],
      results: [
        {
          nodeType: '5',
          status: 'verified-full',
          executedCommands: [
            'add_node',
            'connect',
            'configure_node',
            'validate',
          ],
          evidence: {
            workflowId: 'wf-smoke-1',
            screenshot: 'reports/code-node.png',
            testRunStatus: 'success',
          },
        },
      ],
    });

    expect(report.nodes[0]).toMatchObject({
      executionStatus: 'executed',
      evidence: {
        workflowId: 'wf-smoke-1',
        screenshot: 'reports/code-node.png',
        testRunStatus: 'success',
      },
    });
    expect(report.evidenceGaps).toEqual([]);
  });

  it('summarizes failing, skipped, and partial nodes for agent repair loops', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: [
        {
          type: '5',
          name: '代码',
          registry: 'code',
          supportLevel: 'full',
          canAdd: true,
          canConfigureSemantically: true,
          hasCatalogGuidance: true,
          hasBackendSpec: true,
          runtimeSmoke: 'local',
        },
        {
          type: '4',
          name: '插件/API',
          registry: 'plugin',
          supportLevel: 'resource-bound',
          canAdd: true,
          canConfigureSemantically: true,
          hasCatalogGuidance: true,
          hasBackendSpec: true,
          requiresResource: true,
          runtimeSmoke: 'resource',
        },
      ],
      scenarios: [
        {
          nodeType: '5',
          nodeName: '代码',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: ['add_node', 'connect', 'configure_node', 'validate'],
          setupCommands: [],
          assertions: ['代码节点需要真实执行证据才能算跑过'],
          expectedBindableVariables: ['code.output'],
        },
        {
          nodeType: '4',
          nodeName: '插件/API',
          status: 'skipped-resource-missing',
          runtime: 'resource',
          requiredCommands: ['add_node', 'configure_node'],
          setupCommands: [],
          assertions: ['缺少真实 plugin_id 时必须跳过'],
          expectedBindableVariables: [],
          skipReason: '当前空间没有可用插件资源',
        },
      ],
      results: [
        {
          nodeType: '5',
          status: 'failing',
          executedCommands: ['add_node'],
          failingCommand: 'configure_node',
          error: '输出变量未声明',
        },
      ],
    });

    expect(summarizeWorkflowAgentNodeSmokeReport(report)).toMatchObject({
      total: 2,
      failed: 1,
      skipped: 1,
      failingNodes: ['5 代码: 输出变量未声明'],
      skippedNodes: ['4 插件/API: 当前空间没有可用插件资源'],
      nextActions: [
        '5 代码: 输出变量未声明',
        '4 插件/API: 当前空间没有可用插件资源',
      ],
    });
  });

  it('renders a compact markdown report for human review', () => {
    const report = buildWorkflowAgentNodeSmokeReport({
      capabilities: WORKFLOW_AGENT_NODE_CAPABILITIES.slice(0, 2),
      scenarios: WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS.slice(0, 2),
    });

    const markdown = renderWorkflowAgentNodeSmokeReportMarkdown(report);

    expect(markdown).toContain('# Workflow Agent Node Smoke Report');
    expect(markdown).toContain('| 1 | 开始 | verified-not-executable |');
    expect(markdown).toContain('| 2 | 结束 | verified-not-executable |');
  });
});
