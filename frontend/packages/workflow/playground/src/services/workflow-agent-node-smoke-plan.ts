import {
  type WorkflowCanvasCommand,
  type WorkflowCanvasCommandOp,
} from './workflow-agent-command-protocol';
import {
  type WorkflowAgentNodeSmokeScenario,
  type WorkflowAgentNodeSmokeStatus,
} from './workflow-agent-node-smoke-scenarios';

export type WorkflowAgentNodeSmokePlanMode = 'execute' | 'readonly' | 'skip';

export type WorkflowAgentNodeSmokePlanIsolation =
  | 'temporary-workflow'
  | 'current-readonly'
  | 'resource-fixture'
  | 'sub-canvas';

export interface WorkflowAgentNodeSmokePlanStep {
  phase: 'exercise' | 'cleanup';
  command: WorkflowCanvasCommand;
}

export interface WorkflowAgentNodeSmokePlan {
  nodeType: string;
  nodeName: string;
  nodeTag: string;
  status: WorkflowAgentNodeSmokeStatus;
  mode: WorkflowAgentNodeSmokePlanMode;
  isolation: WorkflowAgentNodeSmokePlanIsolation;
  ready: boolean;
  requiredCommands: WorkflowCanvasCommandOp[];
  assertions: string[];
  expectedBindableVariables: string[];
  missingConcreteCommands: WorkflowCanvasCommandOp[];
  steps: WorkflowAgentNodeSmokePlanStep[];
  cleanupSteps: WorkflowAgentNodeSmokePlanStep[];
  skipReason?: string;
}

export interface WorkflowAgentNodeSmokeResourceFixture {
  label: string;
}

export interface WorkflowAgentNodeSmokePlanReadinessReport {
  ready: string[];
  notReady: string[];
  skipped: string[];
}

const CONCRETE_COMMANDS = new Set<WorkflowCanvasCommandOp>([
  'connect',
  'configure_node',
  'set_node_params',
  'delete_line',
]);

const GENERIC_EXERCISE_COMMANDS = new Set<WorkflowCanvasCommandOp>([
  'get_node_catalog',
  'get_node_spec',
  'get_node_capability_audit',
  'get_canvas_context',
  'get_bindable_variables',
  'add_node',
  'delete_node',
  'clear_canvas',
  'auto_layout',
  'validate',
  'test_run',
  'explain_failure',
]);

const WRITE_COMMANDS = new Set<WorkflowCanvasCommandOp>([
  'add_node',
  'delete_node',
  'delete_line',
  'connect',
  'configure_node',
  'set_node_params',
  'clear_canvas',
  'auto_layout',
]);

const NODE_TAG_SLUGS: Record<string, string> = {
  '5': 'code',
  '8': 'if',
  '13': 'output',
  '15': 'text',
  '18': 'question',
  '30': 'input',
  '32': 'variable_merge',
  '58': 'json_stringify',
};

export const buildWorkflowAgentNodeSmokePlans = ({
  scenarios,
  resourceFixtures = {},
}: {
  scenarios: WorkflowAgentNodeSmokeScenario[];
  resourceFixtures?: Record<string, WorkflowAgentNodeSmokeResourceFixture>;
}): WorkflowAgentNodeSmokePlan[] =>
  scenarios.map(scenario => buildPlanForScenario(scenario, resourceFixtures));

export const getWorkflowAgentNodeSmokePlanReadinessReport = (
  plans: WorkflowAgentNodeSmokePlan[],
): WorkflowAgentNodeSmokePlanReadinessReport =>
  plans.reduce<WorkflowAgentNodeSmokePlanReadinessReport>(
    (report, plan) => {
      const label = `${plan.nodeType} ${plan.nodeName}`;
      if (plan.mode === 'skip') {
        report.skipped.push(`${label}: ${plan.skipReason ?? '跳过执行'}`);
      } else if (!plan.ready) {
        report.notReady.push(
          `${label}: 缺少具体 smoke 步骤 ${plan.missingConcreteCommands.join(
            ', ',
          )}`,
        );
      } else {
        report.ready.push(label);
      }
      return report;
    },
    { ready: [], notReady: [], skipped: [] },
  );

const buildPlanForScenario = (
  scenario: WorkflowAgentNodeSmokeScenario,
  resourceFixtures: Record<string, WorkflowAgentNodeSmokeResourceFixture>,
): WorkflowAgentNodeSmokePlan => {
  const nodeTag = createNodeSmokeTag(scenario);
  const mode = getPlanMode(scenario, resourceFixtures);
  const isolation = getPlanIsolation(scenario, mode);
  const missingConcreteCommands =
    mode === 'execute' ? getMissingConcreteCommands(scenario) : [];

  if (mode === 'skip') {
    return {
      nodeType: scenario.nodeType,
      nodeName: scenario.nodeName,
      nodeTag,
      status: scenario.status,
      mode,
      isolation,
      ready: false,
      requiredCommands: scenario.requiredCommands,
      assertions: scenario.assertions,
      expectedBindableVariables: scenario.expectedBindableVariables,
      missingConcreteCommands,
      steps: [],
      cleanupSteps: [],
      skipReason: getSkipReason(scenario),
    };
  }

  const steps = buildExerciseSteps(scenario, nodeTag, missingConcreteCommands);
  const cleanupSteps = buildCleanupSteps(scenario, nodeTag);
  const ready = missingConcreteCommands.length === 0;

  return {
    nodeType: scenario.nodeType,
    nodeName: scenario.nodeName,
    nodeTag,
    status: scenario.status,
    mode,
    isolation,
    ready,
    requiredCommands: scenario.requiredCommands,
    assertions: scenario.assertions,
    expectedBindableVariables: scenario.expectedBindableVariables,
    missingConcreteCommands,
    steps,
    cleanupSteps,
  };
};

const getPlanMode = (
  scenario: WorkflowAgentNodeSmokeScenario,
  resourceFixtures: Record<string, WorkflowAgentNodeSmokeResourceFixture>,
): WorkflowAgentNodeSmokePlanMode => {
  if (scenario.status === 'unsupported') {
    return 'skip';
  }
  if (scenario.runtime === 'resource' && !resourceFixtures[scenario.nodeType]) {
    return 'skip';
  }
  if (scenario.runtime === 'sub-canvas') {
    return 'skip';
  }
  if (
    scenario.runtime === 'not-executable' &&
    !scenario.requiredCommands.some(command => WRITE_COMMANDS.has(command))
  ) {
    return 'readonly';
  }
  return 'execute';
};

const getPlanIsolation = (
  scenario: WorkflowAgentNodeSmokeScenario,
  mode: WorkflowAgentNodeSmokePlanMode,
): WorkflowAgentNodeSmokePlanIsolation => {
  if (mode === 'readonly') {
    return 'current-readonly';
  }
  if (scenario.runtime === 'resource') {
    return 'resource-fixture';
  }
  if (scenario.runtime === 'sub-canvas') {
    return 'sub-canvas';
  }
  return 'temporary-workflow';
};

const getSkipReason = (scenario: WorkflowAgentNodeSmokeScenario): string =>
  scenario.skipReason ??
  scenario.knownGaps?.join('; ') ??
  '当前节点 smoke 仍需补齐场景或资源。';

const getMissingConcreteCommands = (
  scenario: WorkflowAgentNodeSmokeScenario,
): WorkflowCanvasCommandOp[] => {
  const concreteSetupCommands = new Set(
    scenario.setupCommands.map(command => command.op),
  );
  return scenario.requiredCommands.filter(
    command => CONCRETE_COMMANDS.has(command) && !concreteSetupCommands.has(command),
  );
};

const buildExerciseSteps = (
  scenario: WorkflowAgentNodeSmokeScenario,
  nodeTag: string,
  missingConcreteCommands: WorkflowCanvasCommandOp[],
): WorkflowAgentNodeSmokePlanStep[] => {
  const missing = new Set(missingConcreteCommands);
  const setupSteps = scenario.setupCommands.map(command => ({
    phase: 'exercise' as const,
    command: normalizeScenarioCommand(command, nodeTag),
  }));
  const setupOps = new Set(setupSteps.map(step => step.command.op));
  const hasSelfAddNodeStep = setupSteps.some(
    step => step.command.op === 'add_node' && step.command.target === nodeTag,
  );

  const steps: WorkflowAgentNodeSmokePlanStep[] = [];
  if (
    scenario.requiredCommands.includes('add_node') &&
    !missing.has('add_node') &&
    !hasSelfAddNodeStep
  ) {
    steps.push({
      phase: 'exercise',
      command: genericCommand('add_node', scenario, nodeTag),
    });
  }

  steps.push(...setupSteps.filter(step => !missing.has(step.command.op)));

  scenario.requiredCommands.forEach(command => {
    if (missing.has(command)) {
      return;
    }
    if (command === 'add_node' || setupOps.has(command)) {
      return;
    }
    if (!GENERIC_EXERCISE_COMMANDS.has(command)) {
      return;
    }
    steps.push({
      phase: 'exercise',
      command: genericCommand(command, scenario, nodeTag),
    });
  });

  if (
    steps.some(step => WRITE_COMMANDS.has(step.command.op)) &&
    !steps.some(step => step.command.op === 'auto_layout')
  ) {
    steps.push({ phase: 'exercise', command: { op: 'auto_layout' } });
  }

  return steps;
};

const buildCleanupSteps = (
  scenario: WorkflowAgentNodeSmokeScenario,
  nodeTag: string,
): WorkflowAgentNodeSmokePlanStep[] => {
  const setupAddedNodeTags = scenario.setupCommands
    .filter(command => command.op === 'add_node')
    .map(command => (command.target === 'self' ? nodeTag : command.target));
  if (
    !scenario.requiredCommands.includes('add_node') &&
    !setupAddedNodeTags.length
  ) {
    return [];
  }
  const addedNodeTags = uniqueStrings([
    ...(scenario.requiredCommands.includes('add_node') ? [nodeTag] : []),
    ...setupAddedNodeTags,
  ]);

  return [
    ...addedNodeTags.map<WorkflowAgentNodeSmokePlanStep>(target => ({
      phase: 'cleanup',
      command: {
        op: 'delete_node',
        target,
        args: { node: target },
      },
    })),
    { phase: 'cleanup', command: { op: 'auto_layout' } },
  ];
};

const genericCommand = (
  op: WorkflowCanvasCommandOp,
  scenario: WorkflowAgentNodeSmokeScenario,
  nodeTag: string,
): WorkflowCanvasCommand => {
  if (op === 'add_node') {
    return {
      op,
      target: nodeTag,
      args: {
        type: scenario.nodeType,
        title: `${scenario.nodeName} Smoke`,
      },
    };
  }
  if (op === 'delete_node') {
    return { op, target: nodeTag, args: { node: nodeTag } };
  }
  if (op === 'get_node_spec') {
    return { op, args: { type: scenario.nodeType } };
  }
  if (op === 'get_bindable_variables') {
    return { op, target: nodeTag, args: { target_node: nodeTag } };
  }
  return { op };
};

const normalizeScenarioCommand = (
  command: WorkflowAgentNodeSmokeScenario['setupCommands'][number],
  nodeTag: string,
): WorkflowCanvasCommand => ({
  op: command.op,
  target: command.target === 'self' ? nodeTag : command.target,
  args: normalizeSelfReferences(command.args, nodeTag),
});

const normalizeSelfReferences = (
  value: unknown,
  nodeTag: string,
): Record<string, unknown> | undefined => {
  if (value === undefined) {
    return undefined;
  }
  if (!isPlainRecord(value)) {
    return value === 'self' ? { value: nodeTag } : undefined;
  }
  return Object.fromEntries(
    Object.entries(value).map(([key, child]) => [
      key,
      child === 'self'
        ? nodeTag
        : isPlainRecord(child)
          ? normalizeSelfReferences(child, nodeTag)
          : child,
    ]),
  );
};

const isPlainRecord = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === 'object' && !Array.isArray(value);

const uniqueStrings = (values: Array<string | undefined>): string[] =>
  Array.from(new Set(values.filter(Boolean) as string[]));

const createNodeSmokeTag = (scenario: WorkflowAgentNodeSmokeScenario): string =>
  `smoke_${scenario.nodeType}_${NODE_TAG_SLUGS[scenario.nodeType] ?? 'node'}`;
