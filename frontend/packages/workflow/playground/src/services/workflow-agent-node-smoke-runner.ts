import {
  WORKFLOW_AGENT_NODE_CAPABILITIES,
  type WorkflowAgentNodeCapability,
} from './workflow-agent-node-capabilities';
import {
  createWorkflowAgentNodeSmokeCommandServiceAdapter,
  type CreateWorkflowAgentNodeSmokeCommandServiceAdapterOptions,
} from './workflow-agent-node-smoke-command-service-adapter';
import {
  executeWorkflowAgentNodeSmokePlan,
  type WorkflowAgentNodeSmokeExecutorAdapter,
} from './workflow-agent-node-smoke-executor';
import {
  buildWorkflowAgentNodeSmokePlans,
  getWorkflowAgentNodeSmokePlanReadinessReport,
  type WorkflowAgentNodeSmokePlan,
  type WorkflowAgentNodeSmokePlanReadinessReport,
  type WorkflowAgentNodeSmokeResourceFixture,
} from './workflow-agent-node-smoke-plan';
import {
  buildWorkflowAgentNodeSmokeReport,
  type WorkflowAgentNodeSmokeExecutionResult,
  type WorkflowAgentNodeSmokeReport,
} from './workflow-agent-node-smoke-report';
import {
  WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
  type WorkflowAgentNodeSmokeScenario,
} from './workflow-agent-node-smoke-scenarios';

export interface WorkflowAgentNodeSmokeSuiteResult {
  plans: WorkflowAgentNodeSmokePlan[];
  readiness: WorkflowAgentNodeSmokePlanReadinessReport;
  results: WorkflowAgentNodeSmokeExecutionResult[];
  report: WorkflowAgentNodeSmokeReport;
}

export const runWorkflowAgentNodeSmokeSuite = async ({
  capabilities,
  scenarios,
  adapter,
  resourceFixtures,
}: {
  capabilities: WorkflowAgentNodeCapability[];
  scenarios: WorkflowAgentNodeSmokeScenario[];
  adapter: WorkflowAgentNodeSmokeExecutorAdapter;
  resourceFixtures?: Record<string, WorkflowAgentNodeSmokeResourceFixture>;
}): Promise<WorkflowAgentNodeSmokeSuiteResult> => {
  const plans = buildWorkflowAgentNodeSmokePlans({
    scenarios,
    resourceFixtures,
  });
  const readiness = getWorkflowAgentNodeSmokePlanReadinessReport(plans);
  const results: WorkflowAgentNodeSmokeExecutionResult[] = [];

  for (const plan of plans) {
    results.push(await executeWorkflowAgentNodeSmokePlan(plan, adapter));
  }

  return {
    plans,
    readiness,
    results,
    report: buildWorkflowAgentNodeSmokeReport({
      capabilities,
      scenarios,
      results,
    }),
  };
};

export const runWorkflowAgentNodeSmokeSuiteWithCommandService = async (
  options: CreateWorkflowAgentNodeSmokeCommandServiceAdapterOptions & {
    capabilities?: WorkflowAgentNodeCapability[];
    scenarios?: WorkflowAgentNodeSmokeScenario[];
    resourceFixtures?: Record<string, WorkflowAgentNodeSmokeResourceFixture>;
  },
): Promise<WorkflowAgentNodeSmokeSuiteResult> =>
  await runWorkflowAgentNodeSmokeSuite({
    capabilities: options.capabilities ?? WORKFLOW_AGENT_NODE_CAPABILITIES,
    scenarios: options.scenarios ?? WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
    resourceFixtures: options.resourceFixtures,
    adapter: createWorkflowAgentNodeSmokeCommandServiceAdapter(options),
  });

export const runWorkflowAgentNodeSmokeCommandWithCommandService = async (
  options: CreateWorkflowAgentNodeSmokeCommandServiceAdapterOptions & {
    nodeType?: string;
    includeSkipped?: boolean;
    capabilities?: WorkflowAgentNodeCapability[];
    scenarios?: WorkflowAgentNodeSmokeScenario[];
    resourceFixtures?: Record<string, WorkflowAgentNodeSmokeResourceFixture>;
  },
): Promise<WorkflowAgentNodeSmokeSuiteResult> => {
  const scenarios = selectWorkflowAgentNodeSmokeScenarios({
    scenarios: options.scenarios ?? WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
    nodeType: options.nodeType,
    includeSkipped: options.includeSkipped ?? true,
  });
  const selectedTypes = new Set(scenarios.map(scenario => scenario.nodeType));
  const capabilities = (options.capabilities ?? WORKFLOW_AGENT_NODE_CAPABILITIES)
    .filter(capability => selectedTypes.has(capability.type));

  return await runWorkflowAgentNodeSmokeSuite({
    capabilities,
    scenarios,
    resourceFixtures: options.resourceFixtures,
    adapter: createWorkflowAgentNodeSmokeCommandServiceAdapter(options),
  });
};

const selectWorkflowAgentNodeSmokeScenarios = ({
  scenarios,
  nodeType,
  includeSkipped,
}: {
  scenarios: WorkflowAgentNodeSmokeScenario[];
  nodeType?: string;
  includeSkipped: boolean;
}): WorkflowAgentNodeSmokeScenario[] =>
  scenarios.filter(scenario => {
    if (nodeType && scenario.nodeType !== nodeType) {
      return false;
    }
    if (includeSkipped) {
      return true;
    }
    return scenario.runtime !== 'resource' &&
      scenario.runtime !== 'sub-canvas' &&
      scenario.status !== 'unsupported';
  });
