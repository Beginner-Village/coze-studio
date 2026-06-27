import {
  createWorkflowCanvasCommandEnvelope,
  type WorkflowCanvasCommand,
  type WorkflowCanvasCommandResult,
  type WorkflowCanvasCommandResultItem,
  type WorkflowCanvasSurface,
} from './workflow-agent-command-protocol';
import {
  type WorkflowAgentNodeSmokeEvidence,
} from './workflow-agent-node-smoke-report';
import {
  type WorkflowAgentNodeSmokeCommandExecutionResult,
  type WorkflowAgentNodeSmokeExecutorAdapter,
} from './workflow-agent-node-smoke-executor';
import { type WorkflowAgentNodeSmokePlan } from './workflow-agent-node-smoke-plan';

export interface WorkflowAgentNodeSmokeCommandServiceLike {
  applyCommandEnvelope: (
    envelope: ReturnType<typeof createWorkflowCanvasCommandEnvelope>,
  ) => Promise<WorkflowCanvasCommandResult>;
}

export interface CreateWorkflowAgentNodeSmokeCommandServiceAdapterOptions {
  surface: WorkflowCanvasSurface;
  service: WorkflowAgentNodeSmokeCommandServiceLike;
  spaceId?: string;
  canvasId?: string;
  allowTemporaryWorkflowPlans?: boolean;
  captureEvidence?: (
    plan: WorkflowAgentNodeSmokePlan,
  ) => Promise<Omit<WorkflowAgentNodeSmokeEvidence, 'workflowId'>>;
}

export const createWorkflowAgentNodeSmokeCommandServiceAdapter = ({
  surface,
  service,
  spaceId,
  canvasId,
  allowTemporaryWorkflowPlans = false,
  captureEvidence,
}: CreateWorkflowAgentNodeSmokeCommandServiceAdapterOptions): WorkflowAgentNodeSmokeExecutorAdapter => ({
  executeCommand: async (command, plan) => {
    if (plan.isolation === 'temporary-workflow' && !allowTemporaryWorkflowPlans) {
      return {
        ok: false,
        error:
          '节点 smoke 计划需要临时 workflow,当前 CommandService adapter 未显式允许执行。',
      };
    }
    const result = await service.applyCommandEnvelope(
      createWorkflowCanvasCommandEnvelope([command], {
        requestId: createSmokeCommandRequestId(plan, command),
        surface,
        mode: 'browser_live',
        spaceId,
        canvasId,
        requiresResponse: true,
      }),
    );
    return toSmokeCommandExecutionResult(result);
  },
  captureEvidence: async plan => ({
    workflowId: canvasId,
    ...(await captureEvidence?.(plan)),
  }),
});

const createSmokeCommandRequestId = (
  plan: WorkflowAgentNodeSmokePlan,
  command: WorkflowCanvasCommand,
): string => `smoke-${plan.nodeType}-${command.op}-${Date.now()}`;

const toSmokeCommandExecutionResult = (
  result: WorkflowCanvasCommandResult,
): WorkflowAgentNodeSmokeCommandExecutionResult => {
  if (result.status === 'ok') {
    return { ok: true };
  }
  const failed = result.results.find(item => !item.ok);
  return {
    ok: false,
    error: failed ? getFailedResultMessage(failed) : result.status,
  };
};

const getFailedResultMessage = (
  item: WorkflowCanvasCommandResultItem,
): string =>
  [
    item.message,
    ...item.diagnostics
      .filter(diagnostic => diagnostic.level === 'error')
      .map(diagnostic => diagnostic.message),
  ]
    .filter(Boolean)
    .join('; ') || `${item.op} 执行失败`;
