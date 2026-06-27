import { isWorkflowAgentSingletonNodeType } from './workflow-agent-node-json';

export interface WorkflowAgentClearableNode {
  flowNodeType?: string | number;
}

export const collectWorkflowAgentClearableNodes = <
  T extends WorkflowAgentClearableNode,
>(
  nodes: readonly T[],
): T[] =>
  nodes.filter(
    node => !isWorkflowAgentSingletonNodeType(String(node.flowNodeType)),
  );
