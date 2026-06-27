import { describe, expect, it } from 'vitest';

import {
  buildWorkflowAgentNodeSmokePlans,
  getWorkflowAgentNodeSmokePlanReadinessReport,
} from '../workflow-agent-node-smoke-plan';
import { WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES } from '../workflow-agent-node-capabilities';
import { WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS } from '../workflow-agent-node-smoke-scenarios';

describe('workflow-agent-node-smoke-plan', () => {
  it('builds one executable or skipped plan for every declared scenario', () => {
    const plans = buildWorkflowAgentNodeSmokePlans({
      scenarios: [
        {
          nodeType: '1',
          nodeName: '开始',
          status: 'verified-not-executable',
          runtime: 'not-executable',
          requiredCommands: ['get_canvas_context'],
          setupCommands: [],
          assertions: ['Start 只读'],
          expectedBindableVariables: ['start.input'],
        },
        {
          nodeType: '5',
          nodeName: '代码',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: ['add_node', 'configure_node', 'validate'],
          setupCommands: [
            {
              op: 'configure_node',
              target: 'self',
              args: {
                config: {
                  code: 'async def main(args: Args):\n    return {"output": "ok"}',
                  outputs: [{ name: 'output', type: 'string' }],
                },
              },
            },
          ],
          assertions: ['代码节点可本地执行'],
          expectedBindableVariables: ['code.output'],
        },
      ],
    });

    expect(plans).toHaveLength(2);
    expect(plans[0]).toMatchObject({
      nodeType: '1',
      mode: 'readonly',
      isolation: 'current-readonly',
      ready: true,
    });
    expect(plans[0].steps.map(step => step.command.op)).toEqual([
      'get_canvas_context',
    ]);
    expect(plans[1]).toMatchObject({
      nodeType: '5',
      mode: 'execute',
      isolation: 'temporary-workflow',
      ready: true,
      nodeTag: 'smoke_5_code',
    });
    expect(plans[1].steps.map(step => step.command.op)).toEqual([
      'add_node',
      'configure_node',
      'validate',
      'auto_layout',
    ]);
    expect(plans[1].cleanupSteps.map(step => step.command.op)).toEqual([
      'delete_node',
      'auto_layout',
    ]);
  });

  it('does not execute resource-bound nodes without a matching resource fixture', () => {
    const [plan] = buildWorkflowAgentNodeSmokePlans({
      scenarios: [
        {
          nodeType: '4',
          nodeName: '插件/API',
          status: 'verified-resource-bound',
          runtime: 'resource',
          requiredCommands: ['add_node', 'configure_node', 'validate'],
          setupCommands: [],
          assertions: ['需要插件资源'],
          expectedBindableVariables: ['api.output'],
          skipReason: '运行依赖空间插件/API资源。',
        },
      ],
      resourceFixtures: {},
    });

    expect(plan).toMatchObject({
      mode: 'skip',
      ready: false,
      skipReason: '运行依赖空间插件/API资源。',
    });
    expect(plan.steps).toEqual([]);
    expect(plan.cleanupSteps).toEqual([]);
  });

  it('surfaces missing concrete connect/configure steps instead of guessing VariableMerge wiring', () => {
    const [plan] = buildWorkflowAgentNodeSmokePlans({
      scenarios: [
        {
          nodeType: '32',
          nodeName: '变量聚合',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: [
            'add_node',
            'connect',
            'get_bindable_variables',
            'configure_node',
            'validate',
          ],
          setupCommands: [],
          assertions: ['必须绑定真实变量'],
          expectedBindableVariables: ['merge.output'],
        },
      ],
    });

    expect(plan).toMatchObject({
      mode: 'execute',
      ready: false,
      missingConcreteCommands: ['connect', 'configure_node'],
    });
    expect(plan.steps.map(step => step.command.op)).toEqual([
      'add_node',
      'get_bindable_variables',
      'validate',
      'auto_layout',
    ]);

    const report = getWorkflowAgentNodeSmokePlanReadinessReport([plan]);

    expect(report.notReady).toEqual([
      '32 变量聚合: 缺少具体 smoke 步骤 connect, configure_node',
    ]);
  });

  it('creates a plan for every visible node scenario and reports current readiness gaps', () => {
    const plans = buildWorkflowAgentNodeSmokePlans({
      scenarios: WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
    });

    expect(plans.map(plan => plan.nodeType).sort()).toEqual(
      [...WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES].sort(),
    );

    const report = getWorkflowAgentNodeSmokePlanReadinessReport(plans);

    expect(report.notReady).toEqual([]);
    expect(report.ready).toEqual(
      expect.arrayContaining([
        '8 条件分支',
        '18 问答/追问',
        '32 变量聚合',
        '58 JSON序列化',
      ]),
    );
    expect(report.skipped).toEqual(
      expect.arrayContaining([
        '4 插件/API: 运行依赖空间插件/API资源。',
        '21 循环: 缺少循环语义配置器。',
      ]),
    );
  });

  it('normalizes self references inside concrete command args', () => {
    const [plan] = buildWorkflowAgentNodeSmokePlans({
      scenarios: [
        {
          nodeType: '5',
          nodeName: '代码',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: ['add_node', 'connect'],
          setupCommands: [
            {
              op: 'connect',
              args: { from: 'start', to: 'self' },
            },
          ],
          assertions: ['连接 start 到当前 smoke 节点'],
          expectedBindableVariables: ['code.output'],
        },
      ],
    });

    expect(plan.steps[1].command).toEqual({
      op: 'connect',
      target: undefined,
      args: { from: 'start', to: 'smoke_5_code' },
    });
  });

  it('preserves ordered multi-node setup commands for nodes that need real upstream variables', () => {
    const [plan] = buildWorkflowAgentNodeSmokePlans({
      scenarios: [
        {
          nodeType: '32',
          nodeName: '变量聚合',
          status: 'verified-full',
          runtime: 'local',
          requiredCommands: [
            'add_node',
            'connect',
            'get_bindable_variables',
            'configure_node',
            'validate',
          ],
          setupCommands: [
            {
              op: 'add_node',
              target: 'smoke_32_text_a',
              args: { type: '15', title: '聚合输入 A' },
            },
            {
              op: 'configure_node',
              target: 'smoke_32_text_a',
              args: {
                config: {
                  content: 'A',
                  outputs: [{ name: 'output', type: 'string' }],
                },
              },
            },
            {
              op: 'connect',
              args: { from: 'start', to: 'smoke_32_text_a' },
            },
            {
              op: 'connect',
              args: { from: 'smoke_32_text_a', to: 'self' },
            },
            {
              op: 'get_bindable_variables',
              target: 'self',
              args: { target_node: 'self' },
            },
            {
              op: 'configure_node',
              target: 'self',
              args: {
                config: {
                  merge_groups: [
                    {
                      name: 'output',
                      variables: [
                        {
                          from: 'smoke_32_text_a',
                          output: 'output',
                          type: 'string',
                        },
                      ],
                    },
                  ],
                },
              },
            },
          ],
          assertions: ['必须绑定真实变量'],
          expectedBindableVariables: ['merge.output'],
        },
      ],
    });

    expect(plan.ready).toBe(true);
    expect(
      plan.steps.map(step => ({
        op: step.command.op,
        target: step.command.target,
        args: step.command.args,
      })),
    ).toMatchObject([
      { op: 'add_node', target: 'smoke_32_variable_merge' },
      { op: 'add_node', target: 'smoke_32_text_a' },
      { op: 'configure_node', target: 'smoke_32_text_a' },
      { op: 'connect', args: { from: 'start', to: 'smoke_32_text_a' } },
      {
        op: 'connect',
        args: {
          from: 'smoke_32_text_a',
          to: 'smoke_32_variable_merge',
        },
      },
      {
        op: 'get_bindable_variables',
        target: 'smoke_32_variable_merge',
        args: { target_node: 'smoke_32_variable_merge' },
      },
      { op: 'configure_node', target: 'smoke_32_variable_merge' },
      { op: 'validate' },
      { op: 'auto_layout' },
    ]);
    expect(plan.cleanupSteps.map(step => step.command.target)).toEqual([
      'smoke_32_variable_merge',
      'smoke_32_text_a',
      undefined,
    ]);
  });

  it('cleans temporary upstream nodes for singleton targets that are not added by smoke', () => {
    const [plan] = buildWorkflowAgentNodeSmokePlans({
      scenarios: [
        {
          nodeType: '2',
          nodeName: '结束',
          status: 'verified-not-executable',
          runtime: 'not-executable',
          requiredCommands: ['get_bindable_variables', 'configure_node', 'validate'],
          setupCommands: [
            {
              op: 'add_node',
              target: 'smoke_2_text_source',
              args: { type: '15', title: '结束返回文本来源' },
            },
            {
              op: 'configure_node',
              target: 'end',
              args: {
                config: {
                  inputs: [
                    {
                      name: 'answer',
                      from: 'smoke_2_text_source',
                      output: 'output',
                    },
                  ],
                  content: '{{answer}}',
                  streaming_output: true,
                },
              },
            },
          ],
          assertions: ['End 只配置内置节点'],
          expectedBindableVariables: ['text.output'],
        },
      ],
    });

    expect(plan.steps.map(step => step.command.op)).toEqual([
      'add_node',
      'configure_node',
      'get_bindable_variables',
      'validate',
      'auto_layout',
    ]);
    expect(plan.cleanupSteps.map(step => step.command)).toEqual([
      {
        op: 'delete_node',
        target: 'smoke_2_text_source',
        args: { node: 'smoke_2_text_source' },
      },
      { op: 'auto_layout' },
    ]);
  });
});
