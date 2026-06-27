import { describe, expect, it } from 'vitest';

import { collectWorkflowAgentClearableNodes } from '../workflow-agent-clear-canvas';

describe('workflow-agent-clear-canvas', () => {
  it('keeps Start/End and returns all live ordinary canvas nodes', () => {
    const start = { id: '100001', flowNodeType: '1' };
    const end = { id: '900001', flowNodeType: '2' };
    const llm = { id: 'llm_1', flowNodeType: '3' };
    const ifNode = { id: 'if_1', flowNodeType: '8' };

    expect(
      collectWorkflowAgentClearableNodes([start, llm, ifNode, end]),
    ).toEqual([llm, ifNode]);
  });
});
