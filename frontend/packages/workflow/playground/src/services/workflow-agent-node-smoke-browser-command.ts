import {
  type WorkflowCanvasCommand,
  type WorkflowCanvasCommandResultItem,
  type WorkflowCanvasSurface,
} from './workflow-agent-command-protocol';
import {
  type WorkflowAgentNodeCapability,
} from './workflow-agent-node-capabilities';
import {
  type WorkflowAgentNodeSmokeReport,
  type WorkflowAgentNodeSmokeSummary,
  summarizeWorkflowAgentNodeSmokeReport,
} from './workflow-agent-node-smoke-report';
import {
  runWorkflowAgentNodeSmokeCommandWithCommandService,
} from './workflow-agent-node-smoke-runner';
import {
  type WorkflowAgentNodeSmokeScenario,
} from './workflow-agent-node-smoke-scenarios';
import {
  type WorkflowAgentNodeSmokeCommandServiceLike,
} from './workflow-agent-node-smoke-command-service-adapter';

export interface ExecuteWorkflowCanvasNodeSmokeCommandOptions {
  surface: WorkflowCanvasSurface;
  service: WorkflowAgentNodeSmokeCommandServiceLike;
  spaceId?: string;
  canvasId?: string;
  capabilities?: WorkflowAgentNodeCapability[];
  scenarios?: WorkflowAgentNodeSmokeScenario[];
}

export interface ExecuteWorkflowCanvasNodeSmokeCommandResult {
  item: WorkflowCanvasCommandResultItem;
  nodeSmokeReport: WorkflowAgentNodeSmokeReport;
  nodeSmokeSummary: WorkflowAgentNodeSmokeSummary;
}

export const executeWorkflowCanvasNodeSmokeCommand = async (
  command: WorkflowCanvasCommand,
  options: ExecuteWorkflowCanvasNodeSmokeCommandOptions,
): Promise<ExecuteWorkflowCanvasNodeSmokeCommandResult | undefined> => {
  if (command.op !== 'run_node_smoke') {
    return undefined;
  }

  const suite = await runWorkflowAgentNodeSmokeCommandWithCommandService({
    surface: options.surface,
    service: options.service,
    spaceId: options.spaceId,
    canvasId: options.canvasId,
    capabilities: options.capabilities,
    scenarios: options.scenarios,
    nodeType: getStringArg(command, 'node_type') ?? command.target,
    includeSkipped: getBooleanArg(command, 'include_skipped', true),
    allowTemporaryWorkflowPlans: getBooleanArg(
      command,
      'allow_temporary_workflow',
      false,
    ),
  });

  const failing = suite.report.counts.failing;
  const summary = summarizeWorkflowAgentNodeSmokeReport(suite.report);
  return {
    item: {
      op: command.op,
      ok: failing === 0,
      target: command.target ?? getStringArg(command, 'node_type'),
      message:
        failing === 0
          ? `节点自测完成: ${suite.report.total} 个节点`
          : `节点自测失败: ${failing} 个节点`,
      diagnostics:
        failing === 0
          ? []
          : suite.report.nextActions.map(message => ({
              level: 'error' as const,
              message,
              op: command.op,
            })),
    },
    nodeSmokeReport: suite.report,
    nodeSmokeSummary: summary,
  };
};

const getStringArg = (
  command: WorkflowCanvasCommand,
  name: string,
): string | undefined => {
  const value = command.args?.[name];
  return typeof value === 'string' && value ? value : undefined;
};

const getBooleanArg = (
  command: WorkflowCanvasCommand,
  name: string,
  fallback: boolean,
): boolean => {
  const value = command.args?.[name];
  return typeof value === 'boolean' ? value : fallback;
};
