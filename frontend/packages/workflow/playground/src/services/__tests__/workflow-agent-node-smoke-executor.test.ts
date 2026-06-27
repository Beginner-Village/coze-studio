import { describe, expect, it } from 'vitest';

import { executeWorkflowAgentNodeSmokePlan } from '../workflow-agent-node-smoke-executor';
import { type WorkflowAgentNodeSmokePlan } from '../workflow-agent-node-smoke-plan';
import { type WorkflowCanvasCommand } from '../workflow-agent-command-protocol';

const executablePlan = (
  overrides: Partial<WorkflowAgentNodeSmokePlan> = {},
): WorkflowAgentNodeSmokePlan => ({
  nodeType: '5',
  nodeName: '代码',
  nodeTag: 'smoke_5_code',
  status: 'verified-full',
  mode: 'execute',
  isolation: 'temporary-workflow',
  ready: true,
  requiredCommands: ['add_node', 'configure_node', 'validate'],
  missingConcreteCommands: [],
  steps: [
    {
      phase: 'exercise',
      command: {
        op: 'add_node',
        target: 'smoke_5_code',
        args: { type: '5', title: '代码 Smoke' },
      },
    },
    {
      phase: 'exercise',
      command: {
        op: 'configure_node',
        target: 'smoke_5_code',
        args: { config: { outputs: [{ name: 'output', type: 'string' }] } },
      },
    },
    { phase: 'exercise', command: { op: 'validate' } },
  ],
  cleanupSteps: [
    {
      phase: 'cleanup',
      command: {
        op: 'delete_node',
        target: 'smoke_5_code',
        args: { node: 'smoke_5_code' },
      },
    },
  ],
  ...overrides,
});

describe('workflow-agent-node-smoke-executor', () => {
  it('executes plan steps and converts them into report execution evidence', async () => {
    const executed: WorkflowCanvasCommand[] = [];
    const result = await executeWorkflowAgentNodeSmokePlan(executablePlan(), {
      executeCommand: async command => {
        executed.push(command);
        return { ok: true };
      },
      captureEvidence: async () => ({
        workflowId: 'wf-smoke-code',
        screenshot: 'reports/code.png',
        testRunStatus: 'success',
      }),
    });

    expect(executed.map(command => command.op)).toEqual([
      'add_node',
      'configure_node',
      'validate',
      'delete_node',
    ]);
    expect(result).toEqual({
      nodeType: '5',
      status: 'verified-full',
      executedCommands: ['add_node', 'configure_node', 'validate'],
      evidence: {
        workflowId: 'wf-smoke-code',
        screenshot: 'reports/code.png',
        testRunStatus: 'success',
      },
    });
  });

  it('runs cleanup and returns a failing result when an exercise command fails', async () => {
    const executed: WorkflowCanvasCommand[] = [];
    const result = await executeWorkflowAgentNodeSmokePlan(executablePlan(), {
      executeCommand: async command => {
        executed.push(command);
        if (command.op === 'configure_node') {
          return {
            ok: false,
            error: '配置输出变量失败',
          };
        }
        return { ok: true };
      },
    });

    expect(executed.map(command => command.op)).toEqual([
      'add_node',
      'configure_node',
      'delete_node',
    ]);
    expect(result).toEqual({
      nodeType: '5',
      status: 'failing',
      executedCommands: ['add_node'],
      failingCommand: 'configure_node',
      error: '配置输出变量失败',
    });
  });

  it('runs cleanup and returns a failing result when the adapter throws', async () => {
    const executed: WorkflowCanvasCommand[] = [];
    const result = await executeWorkflowAgentNodeSmokePlan(executablePlan(), {
      executeCommand: async command => {
        executed.push(command);
        if (command.op === 'configure_node') {
          throw new Error('MCP command timeout');
        }
        return { ok: true };
      },
    });

    expect(executed.map(command => command.op)).toEqual([
      'add_node',
      'configure_node',
      'delete_node',
    ]);
    expect(result).toEqual({
      nodeType: '5',
      status: 'failing',
      executedCommands: ['add_node'],
      failingCommand: 'configure_node',
      error: 'MCP command timeout',
    });
  });

  it('does not execute skipped or not-ready plans', async () => {
    const skipped = await executeWorkflowAgentNodeSmokePlan(
      executablePlan({
        mode: 'skip',
        ready: false,
        steps: [],
        cleanupSteps: [],
        skipReason: '运行依赖空间插件/API资源。',
      }),
      {
        executeCommand: async () => {
          throw new Error('should not execute');
        },
      },
    );
    const notReady = await executeWorkflowAgentNodeSmokePlan(
      executablePlan({
        ready: false,
        missingConcreteCommands: ['connect', 'configure_node'],
      }),
      {
        executeCommand: async () => {
          throw new Error('should not execute');
        },
      },
    );

    expect(skipped).toEqual({
      nodeType: '5',
      status: 'skipped-resource-missing',
      executedCommands: [],
      error: '运行依赖空间插件/API资源。',
    });
    expect(notReady).toEqual({
      nodeType: '5',
      status: 'failing',
      executedCommands: [],
      error: '缺少具体 smoke 步骤 connect, configure_node',
    });
  });

  it('preserves unsupported status for skipped unsupported plans', async () => {
    const result = await executeWorkflowAgentNodeSmokePlan(
      executablePlan({
        status: 'unsupported',
        mode: 'skip',
        ready: false,
        steps: [],
        cleanupSteps: [],
        skipReason: '缺少 MCP 资源发现和语义配置器。',
      }),
      {
        executeCommand: async () => {
          throw new Error('should not execute');
        },
      },
    );

    expect(result).toEqual({
      nodeType: '5',
      status: 'unsupported',
      executedCommands: [],
      error: '缺少 MCP 资源发现和语义配置器。',
    });
  });
});
