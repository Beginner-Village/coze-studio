import { describe, expect, it, vi } from 'vitest';

import { executeWorkflowCanvasReadonlyCommand } from '../workflow-agent-command-readonly';

describe('executeWorkflowCanvasReadonlyCommand', () => {
  it('returns live canvas context for get_canvas_context', async () => {
    const result = await executeWorkflowCanvasReadonlyCommand(
      { op: 'get_canvas_context' },
      {
        getCanvasSummary: vi.fn().mockResolvedValue('节点: Start -> End'),
        getBindableVariablesSummary: vi.fn(),
      },
    );

    expect(result).toEqual({
      item: {
        op: 'get_canvas_context',
        ok: true,
        diagnostics: [],
      },
      canvasContext: '节点: Start -> End',
    });
  });

  it('returns bindable variables for get_bindable_variables', async () => {
    const result = await executeWorkflowCanvasReadonlyCommand(
      { op: 'get_bindable_variables', target: 'end' },
      {
        getCanvasSummary: vi.fn(),
        getBindableVariablesSummary: vi
          .fn()
          .mockResolvedValue('start.input\ntext.output'),
      },
    );

    expect(result).toEqual({
      item: {
        op: 'get_bindable_variables',
        ok: true,
        target: 'end',
        diagnostics: [],
      },
      bindableVariables: 'start.input\ntext.output',
    });
  });

  it('returns undefined for write commands', async () => {
    const result = await executeWorkflowCanvasReadonlyCommand(
      { op: 'add_node', target: 'text', args: { type: '15' } },
      {
        getCanvasSummary: vi.fn(),
        getBindableVariablesSummary: vi.fn(),
      },
    );

    expect(result).toBeUndefined();
  });
});
