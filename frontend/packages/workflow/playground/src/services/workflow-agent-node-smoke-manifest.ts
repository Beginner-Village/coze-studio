import { type WorkflowAgentNodeCapability } from './workflow-agent-node-capabilities';
import { type WorkflowCanvasCommandOp } from './workflow-agent-command-protocol';
import {
  buildWorkflowAgentNodeSmokePlans,
  getWorkflowAgentNodeSmokePlanReadinessReport,
  type WorkflowAgentNodeSmokePlan,
  type WorkflowAgentNodeSmokePlanIsolation,
  type WorkflowAgentNodeSmokePlanMode,
  type WorkflowAgentNodeSmokePlanReadinessReport,
  type WorkflowAgentNodeSmokeResourceFixture,
} from './workflow-agent-node-smoke-plan';
import {
  type WorkflowAgentNodeSmokeScenario,
  type WorkflowAgentNodeSmokeStatus,
} from './workflow-agent-node-smoke-scenarios';

export interface WorkflowAgentNodeSmokeManifestNode {
  nodeType: string;
  nodeName: string;
  status: WorkflowAgentNodeSmokeStatus;
  mode: WorkflowAgentNodeSmokePlanMode;
  isolation: WorkflowAgentNodeSmokePlanIsolation;
  ready: boolean;
  requiredCommands: WorkflowCanvasCommandOp[];
  plannedCommands: WorkflowCanvasCommandOp[];
  cleanupCommands: WorkflowCanvasCommandOp[];
  missingConcreteCommands: WorkflowCanvasCommandOp[];
  temporaryNodeTags: string[];
  requiresTemporaryWorkflowOptIn: boolean;
  requiresResourceFixture: boolean;
  assertions: string[];
  expectedBindableVariables: string[];
  skipReason?: string;
}

export interface WorkflowAgentNodeSmokeManifest {
  total: number;
  coverageGaps: string[];
  readiness: WorkflowAgentNodeSmokePlanReadinessReport;
  summary: {
    ready: number;
    notReady: number;
    skipped: number;
    execute: number;
    readonly: number;
    requiresTemporaryWorkflowOptIn: number;
    requiresResourceFixture: number;
  };
  nodes: WorkflowAgentNodeSmokeManifestNode[];
}

export const buildWorkflowAgentNodeSmokeManifest = ({
  capabilities,
  scenarios,
  resourceFixtures = {},
}: {
  capabilities: WorkflowAgentNodeCapability[];
  scenarios: WorkflowAgentNodeSmokeScenario[];
  resourceFixtures?: Record<string, WorkflowAgentNodeSmokeResourceFixture>;
}): WorkflowAgentNodeSmokeManifest => {
  const plans = buildWorkflowAgentNodeSmokePlans({
    scenarios,
    resourceFixtures,
  });
  const planByType = new Map(plans.map(plan => [plan.nodeType, plan]));
  const coverageGaps = capabilities
    .filter(capability => !planByType.has(capability.type))
    .map(capability => `${capability.type} ${capability.name}: 缺少节点 smoke 场景`);
  const nodes = plans.map(toManifestNode);

  return {
    total: capabilities.length,
    coverageGaps,
    readiness: getWorkflowAgentNodeSmokePlanReadinessReport(plans),
    summary: summarizeManifestNodes(nodes),
    nodes,
  };
};

const toManifestNode = (
  plan: WorkflowAgentNodeSmokePlan,
): WorkflowAgentNodeSmokeManifestNode => ({
  nodeType: plan.nodeType,
  nodeName: plan.nodeName,
  status: plan.status,
  mode: plan.mode,
  isolation: plan.isolation,
  ready: plan.ready,
  requiredCommands: plan.requiredCommands,
  plannedCommands: plan.steps.map(step => step.command.op),
  cleanupCommands: plan.cleanupSteps.map(step => step.command.op),
  missingConcreteCommands: plan.missingConcreteCommands,
  temporaryNodeTags: collectTemporaryNodeTags(plan),
  requiresTemporaryWorkflowOptIn: plan.isolation === 'temporary-workflow',
  requiresResourceFixture:
    plan.isolation === 'resource-fixture' || isResourceFixtureSkip(plan),
  assertions: plan.assertions,
  expectedBindableVariables: plan.expectedBindableVariables,
  skipReason: plan.skipReason,
});

const collectTemporaryNodeTags = (
  plan: WorkflowAgentNodeSmokePlan,
): string[] => {
  if (plan.isolation !== 'temporary-workflow') {
    return [];
  }
  return uniqueStrings(
    plan.cleanupSteps
      .filter(step => step.command.op === 'delete_node')
      .map(step => step.command.target),
  );
};

const isResourceFixtureSkip = (plan: WorkflowAgentNodeSmokePlan): boolean =>
  plan.mode === 'skip' && plan.status === 'verified-resource-bound';

const summarizeManifestNodes = (
  nodes: WorkflowAgentNodeSmokeManifestNode[],
): WorkflowAgentNodeSmokeManifest['summary'] =>
  nodes.reduce<WorkflowAgentNodeSmokeManifest['summary']>(
    (summary, node) => {
      if (node.ready) {
        summary.ready += 1;
      }
      if (!node.ready && node.mode !== 'skip') {
        summary.notReady += 1;
      }
      if (node.mode === 'skip') {
        summary.skipped += 1;
      }
      if (node.mode === 'execute') {
        summary.execute += 1;
      }
      if (node.mode === 'readonly') {
        summary.readonly += 1;
      }
      if (node.requiresTemporaryWorkflowOptIn) {
        summary.requiresTemporaryWorkflowOptIn += 1;
      }
      if (node.requiresResourceFixture) {
        summary.requiresResourceFixture += 1;
      }
      return summary;
    },
    {
      ready: 0,
      notReady: 0,
      skipped: 0,
      execute: 0,
      readonly: 0,
      requiresTemporaryWorkflowOptIn: 0,
      requiresResourceFixture: 0,
    },
  );

const uniqueStrings = (values: Array<string | undefined>): string[] =>
  Array.from(new Set(values.filter(Boolean) as string[]));
