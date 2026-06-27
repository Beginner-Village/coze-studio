import { type WorkflowCanvasCommand } from './workflow-agent-command-protocol';
import {
  type WorkflowAgentNodeSmokeEvidence,
  type WorkflowAgentNodeSmokeExecutionResult,
} from './workflow-agent-node-smoke-report';
import { type WorkflowAgentNodeSmokePlan } from './workflow-agent-node-smoke-plan';
import { type WorkflowAgentNodeSmokeStatus } from './workflow-agent-node-smoke-scenarios';

export interface WorkflowAgentNodeSmokeCommandExecutionResult {
  ok: boolean;
  error?: string;
}

export interface WorkflowAgentNodeSmokeExecutorAdapter {
  executeCommand: (
    command: WorkflowCanvasCommand,
    plan: WorkflowAgentNodeSmokePlan,
  ) => Promise<WorkflowAgentNodeSmokeCommandExecutionResult>;
  captureEvidence?: (
    plan: WorkflowAgentNodeSmokePlan,
  ) => Promise<WorkflowAgentNodeSmokeEvidence | undefined>;
}

export const executeWorkflowAgentNodeSmokePlan = async (
  plan: WorkflowAgentNodeSmokePlan,
  adapter: WorkflowAgentNodeSmokeExecutorAdapter,
): Promise<WorkflowAgentNodeSmokeExecutionResult> => {
  if (plan.mode === 'skip') {
    return {
      nodeType: plan.nodeType,
      status:
        plan.status === 'unsupported' ? 'unsupported' : 'skipped-resource-missing',
      executedCommands: [],
      error: plan.skipReason ?? '跳过执行',
    };
  }
  if (!plan.ready) {
    return {
      nodeType: plan.nodeType,
      status: 'failing',
      executedCommands: [],
      error: `缺少具体 smoke 步骤 ${plan.missingConcreteCommands.join(', ')}`,
    };
  }

  const executedCommands: WorkflowAgentNodeSmokeExecutionResult['executedCommands'] =
    [];
  let failure:
    | {
        command: WorkflowCanvasCommand;
        error: string;
      }
    | undefined;

  for (const step of plan.steps) {
    const result = await executeSmokeCommand(adapter, step.command, plan);
    if (!result.ok) {
      failure = {
        command: step.command,
        error: result.error ?? `${step.command.op} 执行失败`,
      };
      break;
    }
    executedCommands.push(step.command.op);
  }

  await runCleanupSteps(plan, adapter);

  if (failure) {
    return {
      nodeType: plan.nodeType,
      status: 'failing',
      executedCommands,
      failingCommand: failure.command.op,
      error: failure.error,
    };
  }

  return {
    nodeType: plan.nodeType,
    status: plan.status as WorkflowAgentNodeSmokeStatus,
    executedCommands,
    evidence: await adapter.captureEvidence?.(plan),
  };
};

const runCleanupSteps = async (
  plan: WorkflowAgentNodeSmokePlan,
  adapter: WorkflowAgentNodeSmokeExecutorAdapter,
): Promise<void> => {
  for (const step of plan.cleanupSteps) {
    await executeSmokeCommand(adapter, step.command, plan);
  }
};

const executeSmokeCommand = async (
  adapter: WorkflowAgentNodeSmokeExecutorAdapter,
  command: WorkflowCanvasCommand,
  plan: WorkflowAgentNodeSmokePlan,
): Promise<WorkflowAgentNodeSmokeCommandExecutionResult> => {
  try {
    return await adapter.executeCommand(command, plan);
  } catch (error) {
    return {
      ok: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }
};
