import { describe, expect, it } from 'vitest';

import {
  runWorkflowAgentNodeSmokeCommandWithCommandService,
  runWorkflowAgentNodeSmokeSuite,
  runWorkflowAgentNodeSmokeSuiteWithCommandService,
} from '../workflow-agent-node-smoke-runner';
import { type WorkflowAgentNodeCapability } from '../workflow-agent-node-capabilities';
import { type WorkflowCanvasCommand } from '../workflow-agent-command-protocol';
import { type WorkflowAgentNodeSmokeScenario } from '../workflow-agent-node-smoke-scenarios';

const capabilities: WorkflowAgentNodeCapability[] = [
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
];

const scenarios: WorkflowAgentNodeSmokeScenario[] = [
  {
    nodeType: '5',
    nodeName: '代码',
    status: 'verified-full',
    runtime: 'local',
    requiredCommands: ['add_node', 'configure_node', 'validate'],
    setupCommands: [
      {
        op: 'configure_node',
        target: 'self',
        args: { config: { outputs: [{ name: 'output', type: 'string' }] } },
      },
    ],
    assertions: ['代码节点可本地执行'],
    expectedBindableVariables: ['code.output'],
  },
  {
    nodeType: '4',
    nodeName: '插件/API',
    status: 'verified-resource-bound',
    runtime: 'resource',
    requiredCommands: ['add_node', 'configure_node', 'validate'],
    setupCommands: [],
    assertions: ['需要插件资源'],
    expectedBindableVariables: ['api.output'],
    skipReason: '运行依赖空间插件/API资源。',
  },
];

describe('workflow-agent-node-smoke-runner', () => {
  it('runs smoke plans, collects execution results, and builds a report', async () => {
    const executed: WorkflowCanvasCommand[] = [];
    const suite = await runWorkflowAgentNodeSmokeSuite({
      capabilities,
      scenarios,
      adapter: {
        executeCommand: async command => {
          executed.push(command);
          return { ok: true };
        },
        captureEvidence: async plan => ({
          workflowId: `wf-${plan.nodeType}`,
          screenshot: `reports/${plan.nodeType}.png`,
          testRunStatus: 'success',
        }),
      },
    });

    expect(executed.map(command => command.op)).toEqual([
      'add_node',
      'configure_node',
      'validate',
      'auto_layout',
      'delete_node',
      'auto_layout',
    ]);
    expect(suite.readiness.ready).toEqual(['5 代码']);
    expect(suite.readiness.skipped).toEqual([
      '4 插件/API: 运行依赖空间插件/API资源。',
    ]);
    expect(suite.results).toEqual([
      {
        nodeType: '5',
        status: 'verified-full',
        executedCommands: ['add_node', 'configure_node', 'validate', 'auto_layout'],
        evidence: {
          workflowId: 'wf-5',
          screenshot: 'reports/5.png',
          testRunStatus: 'success',
        },
      },
      {
        nodeType: '4',
        status: 'skipped-resource-missing',
        executedCommands: [],
        error: '运行依赖空间插件/API资源。',
      },
    ]);
    expect(suite.report.nodes).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          nodeType: '5',
          executionStatus: 'executed',
        }),
        expect.objectContaining({
          nodeType: '4',
          executionStatus: 'skipped',
        }),
      ]),
    );
  });

  it('runs the default node suite through a command service adapter', async () => {
    const envelopes: unknown[] = [];
    const suite = await runWorkflowAgentNodeSmokeSuiteWithCommandService({
      surface: 'workflow',
      canvasId: 'wf-default-smoke',
      allowTemporaryWorkflowPlans: true,
      service: {
        applyCommandEnvelope: async envelope => {
          envelopes.push(envelope);
          return {
            protocol: 'canvas_automation.v0',
            request_id: envelope.request_id,
            status: 'ok',
            results: envelope.commands.map(command => ({
              op: command.op,
              ok: true,
              diagnostics: [],
            })),
            binding_diagnostics: [],
          };
        },
      },
    });

    expect(suite.report.total).toBeGreaterThan(30);
    expect(suite.readiness.notReady).toEqual([]);
    expect(suite.readiness.skipped).toEqual(
      expect.arrayContaining([
        '4 插件/API: 运行依赖空间插件/API资源。',
      ]),
    );
    expect(envelopes.length).toBeGreaterThan(0);
    for (const nodeType of ['2', '5', '8', '13', '15', '18', '30', '32', '58']) {
      expect(suite.results.find(result => result.nodeType === nodeType)).toMatchObject(
        {
          evidence: {
            workflowId: 'wf-default-smoke',
          },
        },
      );
    }
  });

  it('runs one requested node type from a browser-live MCP smoke command', async () => {
    const envelopes: unknown[] = [];
    const suite = await runWorkflowAgentNodeSmokeCommandWithCommandService({
      surface: 'workflow',
      canvasId: 'wf-node-smoke',
      allowTemporaryWorkflowPlans: true,
      nodeType: '5',
      includeSkipped: false,
      capabilities,
      scenarios,
      service: {
        applyCommandEnvelope: async envelope => {
          envelopes.push(envelope);
          return {
            protocol: 'canvas_automation.v0',
            request_id: envelope.request_id,
            status: 'ok',
            results: envelope.commands.map(command => ({
              op: command.op,
              ok: true,
              diagnostics: [],
            })),
            binding_diagnostics: [],
          };
        },
      },
    });

    expect(suite.report.total).toBe(1);
    expect(suite.results).toHaveLength(1);
    expect(suite.results[0]).toMatchObject({
      nodeType: '5',
      status: 'verified-full',
      evidence: { workflowId: 'wf-node-smoke' },
    });
    expect(envelopes.length).toBeGreaterThan(0);
  });
});
