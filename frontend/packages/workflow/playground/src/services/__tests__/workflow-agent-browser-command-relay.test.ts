import { describe, expect, it, vi } from 'vitest';

import {
  pollWorkflowCanvasBrowserCommands,
  postWorkflowCanvasBrowserCommandResult,
} from '../workflow-agent-browser-command-relay';

describe('workflow-agent-browser-command-relay', () => {
  it('polls browser commands and converts them into a command envelope', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          status: 'ok',
          protocol: 'canvas_automation.v0',
          mode: 'browser_live',
          space_id: 'space-1',
          canvas_id: 'wf-1',
          cursor: 12,
          commands: [
            {
              op: 'add_node',
              target: 'reply_text',
              args: { type: '15', title: '回复文本' },
            },
          ],
        }),
    });

    const result = await pollWorkflowCanvasBrowserCommands({
      workflowId: 'wf-1',
      spaceId: 'space-1',
      afterId: 7,
      limit: 20,
      fetcher,
    });

    expect(fetcher).toHaveBeenCalledWith(
      '/api/workflow_mcp/browser_commands?workflow_id=wf-1&space_id=space-1&after_id=7&limit=20',
      { method: 'GET' },
    );
    expect(result.cursor).toBe(12);
    expect(result.envelope).toEqual({
      protocol: 'canvas_automation.v0',
      surface: 'workflow',
      space_id: 'space-1',
      canvas_id: 'wf-1',
      mode: 'browser_live',
      request_id: 'workflow-mcp-relay-12',
      commands: [
        {
          op: 'add_node',
          target: 'reply_text',
          args: { type: '15', title: '回复文本' },
        },
      ],
    });
  });

  it('returns an empty envelope when there are no new commands', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          status: 'ok',
          protocol: 'canvas_automation.v0',
          mode: 'browser_live',
          space_id: 'space-1',
          canvas_id: 'wf-1',
          cursor: 7,
          commands: [],
        }),
    });

    const result = await pollWorkflowCanvasBrowserCommands({
      workflowId: 'wf-1',
      spaceId: 'space-1',
      afterId: 7,
      fetcher,
    });

    expect(result.cursor).toBe(7);
    expect(result.envelope.commands).toEqual([]);
  });

  it('converts browser request metadata into per-request envelopes', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          status: 'ok',
          protocol: 'canvas_automation.v0',
          mode: 'browser_live',
          space_id: 'space-1',
          canvas_id: 'wf-1',
          cursor: 13,
          requests: [
            {
              id: 12,
              request_id: 'req-context',
              requires_response: true,
              command: { op: 'get_canvas_context' },
            },
            {
              id: 13,
              request_id: 'req-layout',
              requires_response: false,
              command: { op: 'auto_layout' },
            },
          ],
        }),
    });

    const result = await pollWorkflowCanvasBrowserCommands({
      workflowId: 'wf-1',
      spaceId: 'space-1',
      afterId: 7,
      fetcher,
    });

    expect(result.cursor).toBe(13);
    expect(result.envelopes).toEqual([
      {
        protocol: 'canvas_automation.v0',
        surface: 'workflow',
        space_id: 'space-1',
        canvas_id: 'wf-1',
        mode: 'browser_live',
        request_id: 'req-context',
        requires_response: true,
        commands: [{ op: 'get_canvas_context' }],
      },
      {
        protocol: 'canvas_automation.v0',
        surface: 'workflow',
        space_id: 'space-1',
        canvas_id: 'wf-1',
        mode: 'browser_live',
        request_id: 'req-layout',
        requires_response: false,
        commands: [{ op: 'auto_layout' }],
      },
    ]);
  });

  it('posts browser command results back to the system relay', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });

    await postWorkflowCanvasBrowserCommandResult({
      workflowId: 'wf-1',
      spaceId: 'space-1',
      result: {
        protocol: 'canvas_automation.v0',
        request_id: 'req-context',
        status: 'ok',
        results: [],
        canvas_context: '节点: Start -> End',
        binding_diagnostics: [],
      },
      fetcher,
    });

    expect(fetcher).toHaveBeenCalledWith(
      '/api/workflow_mcp/browser_command_results',
      {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          workflow_id: 'wf-1',
          space_id: 'space-1',
          result: {
            protocol: 'canvas_automation.v0',
            request_id: 'req-context',
            status: 'ok',
            results: [],
            canvas_context: '节点: Start -> End',
            binding_diagnostics: [],
          },
        }),
      },
    );
  });

  it('throws on backend errors so the bridge can retry later without advancing cursor', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ status: 'error', error: 'relay failed' }),
    });

    await expect(
      pollWorkflowCanvasBrowserCommands({
        workflowId: 'wf-1',
        spaceId: 'space-1',
        afterId: 7,
        fetcher,
      }),
    ).rejects.toThrow('relay failed');
  });
});
