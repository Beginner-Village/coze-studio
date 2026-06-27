import { type WorkflowAgentNodeCapability } from './workflow-agent-node-capabilities';
import { type WorkflowCanvasCommandOp } from './workflow-agent-command-protocol';
import {
  type WorkflowAgentNodeSmokeScenario,
  type WorkflowAgentNodeSmokeStatus,
} from './workflow-agent-node-smoke-scenarios';

export type WorkflowAgentNodeSmokeExecutionStatus =
  | 'pending'
  | 'partial'
  | 'executed'
  | 'failed'
  | 'skipped'
  | 'not-required';

export interface WorkflowAgentNodeSmokeEvidence {
  workflowId?: string;
  screenshot?: string;
  testRunStatus?: string;
}

export interface WorkflowAgentNodeSmokeExecutionResult {
  nodeType: string;
  status?: WorkflowAgentNodeSmokeStatus;
  executedCommands?: WorkflowCanvasCommandOp[];
  failingCommand?: string;
  error?: string;
  evidence?: WorkflowAgentNodeSmokeEvidence;
}

export interface WorkflowAgentNodeSmokeReportNode {
  nodeType: string;
  nodeName: string;
  capabilitySupport: string;
  status: WorkflowAgentNodeSmokeStatus;
  runtime: string;
  executionStatus: WorkflowAgentNodeSmokeExecutionStatus;
  executedCommands: WorkflowCanvasCommandOp[];
  missingCommands: WorkflowCanvasCommandOp[];
  assertions: string[];
  expectedBindableVariables: string[];
  evidence?: WorkflowAgentNodeSmokeEvidence;
  notes: string[];
}

export interface WorkflowAgentNodeSmokeReport {
  total: number;
  counts: {
    verifiedFull: number;
    verifiedResourceBound: number;
    addOnly: number;
    notExecutable: number;
    skipped: number;
    unsupported: number;
    failing: number;
  };
  coverageGaps: string[];
  evidenceGaps: string[];
  nextActions: string[];
  nodes: WorkflowAgentNodeSmokeReportNode[];
}

export interface WorkflowAgentNodeSmokeSummary {
  total: number;
  executed: number;
  failed: number;
  skipped: number;
  partial: number;
  pending: number;
  notRequired: number;
  failingNodes: string[];
  skippedNodes: string[];
  partialNodes: string[];
  evidenceGaps: string[];
  nextActions: string[];
}

export const buildWorkflowAgentNodeSmokeReport = ({
  capabilities,
  scenarios,
  results = [],
}: {
  capabilities: WorkflowAgentNodeCapability[];
  scenarios: WorkflowAgentNodeSmokeScenario[];
  results?: WorkflowAgentNodeSmokeExecutionResult[];
}): WorkflowAgentNodeSmokeReport => {
  const scenarioByType = new Map(
    scenarios.map(scenario => [scenario.nodeType, scenario]),
  );
  const resultByType = new Map(results.map(result => [result.nodeType, result]));
  const coverageGaps: string[] = [];
  const evidenceGaps: string[] = [];
  const nextActions: string[] = [];
  const nodes = capabilities.map(capability => {
    const scenario = scenarioByType.get(capability.type);
    const execution = resultByType.get(capability.type);
    if (!scenario) {
      const message = `${capability.type} ${capability.name}: 缺少节点 smoke 场景`;
      coverageGaps.push(message);
      nextActions.push(message);
      return {
        nodeType: capability.type,
        nodeName: capability.name,
        capabilitySupport: capability.supportLevel,
        status: 'failing' as const,
        runtime: capability.runtimeSmoke ?? 'local',
        executionStatus: 'failed' as const,
        executedCommands: [],
        missingCommands: [],
        assertions: [],
        expectedBindableVariables: [],
        notes: [message],
      };
    }

    const status = execution?.status ?? scenario.status;
    const executedCommands = execution?.executedCommands ?? [];
    const missingCommands = getMissingCommands(
      scenario.requiredCommands,
      executedCommands,
    );
    const executionStatus = getExecutionStatus({
      status,
      execution,
      missingCommands,
    });
    if (
      (executionStatus === 'pending' || executionStatus === 'partial') &&
      missingCommands.length
    ) {
      const message = `${capability.type} ${
        capability.name
      }: 缺少 smoke 执行证据 ${missingCommands.join(', ')}`;
      evidenceGaps.push(message);
      nextActions.push(message);
    }

    const notes = [
      ...scenario.assertions,
      ...(scenario.skipReason ? [scenario.skipReason] : []),
      ...(scenario.knownGaps ?? []),
      ...(execution?.error ? [execution.error] : []),
    ];

    if (status === 'skipped-resource-missing' && scenario.skipReason) {
      nextActions.push(
        `${capability.type} ${capability.name}: ${scenario.skipReason}`,
      );
    } else if (status === 'unsupported' && scenario.knownGaps?.length) {
      nextActions.push(
        `${capability.type} ${capability.name}: ${scenario.knownGaps.join('; ')}`,
      );
    } else if (status === 'failing') {
      nextActions.push(
        `${capability.type} ${capability.name}: ${
          execution?.error ?? 'smoke 场景执行失败'
        }`,
      );
    }

    return {
      nodeType: capability.type,
      nodeName: capability.name,
      capabilitySupport: capability.supportLevel,
      status,
      runtime: scenario.runtime,
      executionStatus,
      executedCommands,
      missingCommands,
      assertions: scenario.assertions,
      expectedBindableVariables: scenario.expectedBindableVariables,
      evidence: execution?.evidence,
      notes,
    };
  });

  return {
    total: capabilities.length,
    counts: countNodes(nodes),
    coverageGaps,
    evidenceGaps,
    nextActions,
    nodes,
  };
};

export const summarizeWorkflowAgentNodeSmokeReport = (
  report: WorkflowAgentNodeSmokeReport,
): WorkflowAgentNodeSmokeSummary => {
  const summary: WorkflowAgentNodeSmokeSummary = {
    total: report.total,
    executed: 0,
    failed: 0,
    skipped: 0,
    partial: 0,
    pending: 0,
    notRequired: 0,
    failingNodes: [],
    skippedNodes: [],
    partialNodes: [],
    evidenceGaps: [...report.evidenceGaps],
    nextActions: [...report.nextActions],
  };

  for (const node of report.nodes) {
    const label = `${node.nodeType} ${node.nodeName}`;
    const lastNote = node.notes[node.notes.length - 1];
    if (node.executionStatus === 'executed') {
      summary.executed += 1;
    } else if (node.executionStatus === 'failed') {
      summary.failed += 1;
      summary.failingNodes.push(`${label}: ${lastNote ?? '执行失败'}`);
    } else if (node.executionStatus === 'skipped') {
      summary.skipped += 1;
      summary.skippedNodes.push(`${label}: ${lastNote ?? '跳过执行'}`);
    } else if (node.executionStatus === 'partial') {
      summary.partial += 1;
      summary.partialNodes.push(
        `${label}: 缺少 ${node.missingCommands.join(', ')}`,
      );
    } else if (node.executionStatus === 'pending') {
      summary.pending += 1;
    } else if (node.executionStatus === 'not-required') {
      summary.notRequired += 1;
    }
  }

  return summary;
};

export const renderWorkflowAgentNodeSmokeReportMarkdown = (
  report: WorkflowAgentNodeSmokeReport,
): string => {
  const lines = [
    '# Workflow Agent Node Smoke Report',
    '',
    `Total: ${report.total}`,
    '',
    '| Type | Node | Status | Runtime | Execution | Notes |',
    '| --- | --- | --- | --- | --- | --- |',
    ...report.nodes.map(node =>
      [
        '|',
        node.nodeType,
        '|',
        node.nodeName,
        '|',
        node.status,
        '|',
        node.runtime,
        '|',
        node.executionStatus,
        '|',
        node.notes.slice(0, 2).join('; '),
        '|',
      ].join(' '),
    ),
  ];

  if (report.nextActions.length) {
    lines.push('', '## Next Actions', ...report.nextActions.map(item => `- ${item}`));
  }

  return lines.join('\n');
};

const countNodes = (
  nodes: WorkflowAgentNodeSmokeReportNode[],
): WorkflowAgentNodeSmokeReport['counts'] =>
  nodes.reduce<WorkflowAgentNodeSmokeReport['counts']>(
    (counts, node) => {
      if (node.status === 'verified-full') {
        counts.verifiedFull += 1;
      } else if (node.status === 'verified-resource-bound') {
        counts.verifiedResourceBound += 1;
      } else if (node.status === 'verified-add-only') {
        counts.addOnly += 1;
      } else if (node.status === 'verified-not-executable') {
        counts.notExecutable += 1;
      } else if (node.status === 'skipped-resource-missing') {
        counts.skipped += 1;
      } else if (node.status === 'unsupported') {
        counts.unsupported += 1;
      } else if (node.status === 'failing') {
        counts.failing += 1;
      }
      return counts;
    },
    {
      verifiedFull: 0,
      verifiedResourceBound: 0,
      addOnly: 0,
      notExecutable: 0,
      skipped: 0,
      unsupported: 0,
      failing: 0,
    },
  );

const getMissingCommands = (
  requiredCommands: WorkflowCanvasCommandOp[],
  executedCommands: WorkflowCanvasCommandOp[],
): WorkflowCanvasCommandOp[] => {
  const executed = new Set(executedCommands);
  return requiredCommands.filter(command => !executed.has(command));
};

const getExecutionStatus = ({
  status,
  execution,
  missingCommands,
}: {
  status: WorkflowAgentNodeSmokeStatus;
  execution?: WorkflowAgentNodeSmokeExecutionResult;
  missingCommands: WorkflowCanvasCommandOp[];
}): WorkflowAgentNodeSmokeExecutionStatus => {
  if (status === 'skipped-resource-missing') {
    return 'skipped';
  }
  if (status === 'unsupported') {
    return 'not-required';
  }
  if (status === 'failing' || execution?.error) {
    return 'failed';
  }
  if (!execution) {
    return missingCommands.length ? 'pending' : 'not-required';
  }
  return missingCommands.length ? 'partial' : 'executed';
};
