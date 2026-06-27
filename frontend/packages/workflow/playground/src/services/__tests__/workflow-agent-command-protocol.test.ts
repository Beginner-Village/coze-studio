import { describe, expect, it } from 'vitest';

import {
  createWorkflowCanvasCommandEnvelope,
  getWorkflowCanvasCommandDisplayName,
  isWorkflowCanvasWriteOp,
  normalizeWorkflowCanvasLegacyAck,
  workflowCanvasCommandToAgentCanvasCommand,
} from '../workflow-agent-command-protocol';

describe('workflow-agent-command-protocol', () => {
  it('normalizes legacy dispatched canvas ack without claiming workflow success', () => {
    const result = normalizeWorkflowCanvasLegacyAck({
      status: 'dispatched_to_canvas',
      op: 'configure_node',
      args: { node: 'merge', config: { merge_groups: [] } },
    });

    expect(result).toMatchObject({
      op: 'configure_node',
      ok: true,
      target: 'merge',
      message: '已下发到画布执行',
      diagnostics: [],
    });
  });

  it('creates ordered command envelopes for browser live execution', () => {
    const envelope = createWorkflowCanvasCommandEnvelope(
      [
        { op: 'add_node', target: 'text_ok', args: { type: '15' } },
        { op: 'connect', args: { from: 'start', to: 'text_ok' } },
      ],
      {
        requestId: 'req-1',
        surface: 'workflow',
        mode: 'browser_live',
        spaceId: 'space-1',
        canvasId: 'workflow-1',
      },
    );

    expect(envelope).toEqual({
      protocol: 'canvas_automation.v0',
      surface: 'workflow',
      space_id: 'space-1',
      canvas_id: 'workflow-1',
      mode: 'browser_live',
      request_id: 'req-1',
      commands: [
        { op: 'add_node', target: 'text_ok', args: { type: '15' } },
        { op: 'connect', args: { from: 'start', to: 'text_ok' } },
      ],
    });
  });

  it('maps command names to compact Chinese labels for chat rendering', () => {
    expect(getWorkflowCanvasCommandDisplayName('add_node')).toBe('添加节点');
    expect(getWorkflowCanvasCommandDisplayName('get_bindable_variables')).toBe(
      '读取可绑定变量',
    );
    expect(getWorkflowCanvasCommandDisplayName('test_run')).toBe('试运行工作流');
    expect(getWorkflowCanvasCommandDisplayName('run_node_smoke')).toBe(
      '节点自测',
    );
    expect(getWorkflowCanvasCommandDisplayName('unknown_command')).toBe(
      '执行工具',
    );
  });

  it('converts protocol commands to existing browser command service commands', () => {
    expect(
      workflowCanvasCommandToAgentCanvasCommand({
        op: 'add_node',
        target: 'text_ok',
        args: { type: '15', title: '成功回复' },
      }),
    ).toEqual({
      op: 'addNode',
      tag: 'text_ok',
      args: { type: '15', title: '成功回复' },
    });

    expect(
      workflowCanvasCommandToAgentCanvasCommand({
        op: 'configure_node',
        target: 'text_ok',
        args: { config: { outputs: [{ name: 'output', type: 'string' }] } },
      }),
    ).toEqual({
      op: 'configureNode',
      args: {
        node: 'text_ok',
        config: { outputs: [{ name: 'output', type: 'string' }] },
      },
    });

    expect(
      workflowCanvasCommandToAgentCanvasCommand({
        op: 'delete_node',
        target: 'text_ok',
        args: { node_tag: 'text_ok' },
      }),
    ).toEqual({
      op: 'deleteNode',
      args: { node: 'text_ok' },
    });
  });

  it('identifies write operations that should trigger a batch layout after success', () => {
    expect(isWorkflowCanvasWriteOp('add_node')).toBe(true);
    expect(isWorkflowCanvasWriteOp('configure_node')).toBe(true);
    expect(isWorkflowCanvasWriteOp('run_node_smoke')).toBe(true);
    expect(isWorkflowCanvasWriteOp('get_canvas_context')).toBe(false);
    expect(isWorkflowCanvasWriteOp('test_run')).toBe(false);
  });
});
