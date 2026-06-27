import { describe, expect, it } from 'vitest';

import { createWorkflowAgentNodeSmokeCommandServiceAdapter } from '../workflow-agent-node-smoke-command-service-adapter';
import { type WorkflowAgentNodeSmokePlan } from '../workflow-agent-node-smoke-plan';
import { type WorkflowCanvasCommandEnvelope } from '../workflow-agent-command-protocol';

const plan: WorkflowAgentNodeSmokePlan = {
  nodeType: '5',
  nodeName: '代码',
  nodeTag: 'smoke_5_code',
  status: 'verified-full',
  mode: 'execute',
  isolation: 'temporary-workflow',
  ready: true,
  requiredCommands: ['add_node'],
  missingConcreteCommands: [],
  steps: [],
  cleanupSteps: [],
};

describe('workflow-agent-node-smoke-command-service-adapter', () => {
  it('executes smoke commands through WorkflowAgentCommandService envelopes', async () => {
    const envelopes: WorkflowCanvasCommandEnvelope[] = [];
    const adapter = createWorkflowAgentNodeSmokeCommandServiceAdapter({
      surface: 'workflow',
      spaceId: 'space-1',
      canvasId: 'wf-1',
      allowTemporaryWorkflowPlans: true,
      service: {
        applyCommandEnvelope: async envelope => {
          envelopes.push(envelope);
          return {
            protocol: 'canvas_automation.v0',
            request_id: envelope.request_id,
            status: 'ok',
            results: [
              {
                op: envelope.commands[0].op,
                ok: true,
                diagnostics: [],
              },
            ],
            binding_diagnostics: [],
          };
        },
      },
    });

    const result = await adapter.executeCommand(
      {
        op: 'add_node',
        target: 'smoke_5_code',
        args: { type: '5' },
      },
      plan,
    );

    expect(result).toEqual({ ok: true });
    expect(envelopes).toHaveLength(1);
    expect(envelopes[0]).toMatchObject({
      protocol: 'canvas_automation.v0',
      surface: 'workflow',
      space_id: 'space-1',
      canvas_id: 'wf-1',
      mode: 'browser_live',
      requires_response: true,
      commands: [
        {
          op: 'add_node',
          target: 'smoke_5_code',
          args: { type: '5' },
        },
      ],
    });
    expect(envelopes[0].request_id).toContain('smoke-5-add_node');
  });

  it('turns failed envelope results into command execution errors', async () => {
    const adapter = createWorkflowAgentNodeSmokeCommandServiceAdapter({
      surface: 'workflow',
      allowTemporaryWorkflowPlans: true,
      service: {
        applyCommandEnvelope: async envelope => ({
          protocol: 'canvas_automation.v0',
          request_id: envelope.request_id,
          status: 'failed',
          results: [
            {
              op: 'configure_node',
              ok: false,
              message: '变量绑定不存在',
              diagnostics: [
                {
                  level: 'error',
                  message: 'BlockID is empty',
                },
              ],
            },
          ],
          binding_diagnostics: [],
        }),
      },
    });

    const result = await adapter.executeCommand(
      {
        op: 'configure_node',
        target: 'smoke_5_code',
        args: { config: {} },
      },
      plan,
    );

    expect(result).toEqual({
      ok: false,
      error: '变量绑定不存在; BlockID is empty',
    });
  });

  it('refuses temporary-workflow plans unless the caller explicitly opts in', async () => {
    let called = false;
    const adapter = createWorkflowAgentNodeSmokeCommandServiceAdapter({
      surface: 'workflow',
      service: {
        applyCommandEnvelope: async envelope => {
          called = true;
          return {
            protocol: 'canvas_automation.v0',
            request_id: envelope.request_id,
            status: 'ok',
            results: [],
            binding_diagnostics: [],
          };
        },
      },
    });

    await expect(
      adapter.executeCommand(
        {
          op: 'add_node',
          target: 'smoke_5_code',
          args: { type: '5' },
        },
        plan,
      ),
    ).resolves.toEqual({
      ok: false,
      error:
        '节点 smoke 计划需要临时 workflow,当前 CommandService adapter 未显式允许执行。',
    });
    expect(called).toBe(false);
  });

  it('captures workflow evidence from the adapter options', async () => {
    const adapter = createWorkflowAgentNodeSmokeCommandServiceAdapter({
      surface: 'workflow',
      canvasId: 'wf-1',
      service: {
        applyCommandEnvelope: async envelope => ({
          protocol: 'canvas_automation.v0',
          request_id: envelope.request_id,
          status: 'ok',
          results: [],
          binding_diagnostics: [],
        }),
      },
      captureEvidence: async () => ({
        screenshot: 'reports/code.png',
        testRunStatus: 'success',
      }),
    });

    await expect(adapter.captureEvidence?.(plan)).resolves.toEqual({
      workflowId: 'wf-1',
      screenshot: 'reports/code.png',
      testRunStatus: 'success',
    });
  });
});
