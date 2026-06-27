import { describe, expect, it } from 'vitest';

import { executeWorkflowCanvasNodeSmokeCommand } from '../workflow-agent-node-smoke-browser-command';
import { type WorkflowAgentNodeCapability } from '../workflow-agent-node-capabilities';
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

describe('workflow-agent-node-smoke-browser-command', () => {
  it('executes one browser-live node smoke command and returns a report result', async () => {
    const result = await executeWorkflowCanvasNodeSmokeCommand(
      {
        op: 'run_node_smoke',
        target: '5',
        args: {
          node_type: '5',
          allow_temporary_workflow: true,
          include_skipped: false,
        },
      },
      {
        surface: 'workflow',
        canvasId: 'wf-smoke',
        capabilities,
        scenarios,
        service: {
          applyCommandEnvelope: async envelope => ({
            protocol: 'canvas_automation.v0',
            request_id: envelope.request_id,
            status: 'ok',
            results: envelope.commands.map(command => ({
              op: command.op,
              ok: true,
              diagnostics: [],
            })),
            binding_diagnostics: [],
          }),
        },
      },
    );

    expect(result).toMatchObject({
      item: {
        op: 'run_node_smoke',
        ok: true,
        target: '5',
      },
      nodeSmokeReport: {
        total: 1,
        nodes: [expect.objectContaining({ nodeType: '5' })],
      },
    });
  });

  it('returns a compact smoke summary for agent repair loops', async () => {
    const result = await executeWorkflowCanvasNodeSmokeCommand(
      {
        op: 'run_node_smoke',
        args: {
          allow_temporary_workflow: true,
          include_skipped: true,
        },
      },
      {
        surface: 'workflow',
        canvasId: 'wf-smoke',
        capabilities,
        scenarios,
        service: {
          applyCommandEnvelope: async envelope => {
            const results = envelope.commands.map(command => ({
              op: command.op,
              ok: command.op !== 'configure_node',
              diagnostics:
                command.op === 'configure_node'
                  ? [
                      {
                        level: 'error' as const,
                        message: '输出变量未声明',
                        op: command.op,
                      },
                    ]
                  : [],
            }));
            return {
              protocol: 'canvas_automation.v0',
              request_id: envelope.request_id,
              status: results.some(result => !result.ok) ? 'failed' : 'ok',
              results,
              binding_diagnostics: [],
            };
          },
        },
      },
    );

    expect(result).toMatchObject({
      item: {
        ok: false,
      },
      nodeSmokeSummary: {
        total: 2,
        failed: 1,
        skipped: 1,
        failingNodes: ['5 代码: 输出变量未声明'],
        skippedNodes: ['4 插件/API: 运行依赖空间插件/API资源。'],
      },
    });
    expect(result?.nodeSmokeSummary.nextActions).toEqual(
      expect.arrayContaining([
        '5 代码: 输出变量未声明',
        '4 插件/API: 运行依赖空间插件/API资源。',
      ]),
    );
  });

  it('ignores non-smoke commands so normal command handling can continue', async () => {
    await expect(
      executeWorkflowCanvasNodeSmokeCommand(
        { op: 'add_node', target: 'text', args: { type: '15' } },
        {
          surface: 'workflow',
          service: {
            applyCommandEnvelope: async () => {
              throw new Error('should not run');
            },
          },
        },
      ),
    ).resolves.toBeUndefined();
  });
});
