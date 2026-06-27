import { describe, expect, it } from 'vitest';

import {
  getWorkflowAgentNodeSmokeScenario,
  WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
} from '../workflow-agent-node-smoke-scenarios';
import { WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES } from '../workflow-agent-node-capabilities';

describe('workflow-agent-node-smoke-scenarios', () => {
  it('declares exactly one smoke scenario for every visible workflow node type', () => {
    const scenarioTypes = WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS.map(
      scenario => scenario.nodeType,
    );

    expect([...new Set(scenarioTypes)].sort()).toEqual(
      [...WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES].sort(),
    );
    expect(scenarioTypes).toHaveLength(
      WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES.length,
    );
  });

  it('does not mark nodes verified-full without a semantic configure step', () => {
    const invalid = WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS.filter(
      scenario =>
        scenario.status === 'verified-full' &&
        scenario.runtime !== 'not-executable' &&
        !scenario.requiredCommands.includes('configure_node'),
    );

    expect(invalid).toEqual([]);
  });

  it('documents the VariableMerge rule that only real upstream outputs are mergeable', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('32');

    expect(scenario).toMatchObject({
      status: 'verified-full',
      nodeName: '变量聚合',
    });
    expect(scenario.requiredCommands).toEqual(
      expect.arrayContaining([
        'get_bindable_variables',
        'configure_node',
        'validate',
      ]),
    );
    expect(scenario.assertions.join('\n')).toContain(
      '不能引用 type=13 输出节点',
    );
  });

  it('provides executable VariableMerge setup with real Text upstream outputs', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('32');

    expect(scenario.setupCommands.map(command => command.op)).toEqual([
      'add_node',
      'configure_node',
      'connect',
      'add_node',
      'configure_node',
      'connect',
      'connect',
      'connect',
      'get_bindable_variables',
      'configure_node',
    ]);
    expect(scenario.setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          op: 'add_node',
          target: 'smoke_32_text_primary',
          args: expect.objectContaining({ type: '15' }),
        }),
        expect.objectContaining({
          op: 'add_node',
          target: 'smoke_32_text_fallback',
          args: expect.objectContaining({ type: '15' }),
        }),
        expect.objectContaining({
          op: 'configure_node',
          target: 'self',
        }),
      ]),
    );
    expect(JSON.stringify(scenario.setupCommands)).toContain(
      'smoke_32_text_primary',
    );
    expect(JSON.stringify(scenario.setupCommands)).not.toContain('"type":"13"');
  });

  it('documents that fixed-text Text nodes can remove default inputs', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('15');

    expect(scenario).toMatchObject({
      status: 'verified-full',
      permitsNoInputBinding: true,
    });
    expect(scenario.assertions.join('\n')).toContain(
      '固定文案允许删除默认 input',
    );
  });

  it('provides concrete Code smoke setup so the default suite can execute it', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('5');

    expect(scenario.setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          op: 'connect',
          args: { from: 'start', to: 'self' },
        }),
        expect.objectContaining({
          op: 'configure_node',
          target: 'self',
        }),
      ]),
    );
  });

  it('uses the platform Python Args runtime for Code smoke setup', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('5');
    const configure = scenario.setupCommands.find(
      command => command.op === 'configure_node' && command.target === 'self',
    );

    expect(configure?.args?.config).toMatchObject({
      language: 'python',
    });
    expect(JSON.stringify(configure?.args?.config)).toContain(
      'async def main(args: Args)',
    );
    expect(JSON.stringify(configure?.args?.config)).toContain('args.params');
    expect(JSON.stringify(configure?.args?.config)).not.toContain(
      'async function main',
    );
    expect(scenario.assertions.join('\n')).toContain('Python');
  });

  it('provides concrete local smoke setup for Output, Text, and Input nodes', () => {
    expect(getWorkflowAgentNodeSmokeScenario('13').setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ op: 'connect' }),
        expect.objectContaining({ op: 'configure_node', target: 'self' }),
      ]),
    );
    expect(getWorkflowAgentNodeSmokeScenario('15').setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ op: 'connect' }),
        expect.objectContaining({ op: 'configure_node', target: 'self' }),
      ]),
    );
    expect(getWorkflowAgentNodeSmokeScenario('30').setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ op: 'configure_node', target: 'self' }),
      ]),
    );
  });

  it('provides executable IF setup that binds its condition to a real variable', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('8');

    expect(scenario.setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          op: 'connect',
          args: { from: 'start', to: 'self' },
        }),
        expect.objectContaining({
          op: 'get_bindable_variables',
          target: 'self',
        }),
        expect.objectContaining({
          op: 'configure_node',
          target: 'self',
          args: expect.objectContaining({
            config: expect.objectContaining({
              condition: expect.objectContaining({
                left: { from: 'start', output: 'input', type: 'string' },
              }),
            }),
          }),
        }),
      ]),
    );
  });

  it('provides executable JSON stringify setup with a real Text upstream output', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('58');

    expect(scenario.setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          op: 'add_node',
          target: 'smoke_58_text_source',
          args: expect.objectContaining({ type: '15' }),
        }),
        expect.objectContaining({
          op: 'connect',
          args: { from: 'smoke_58_text_source', to: 'self' },
        }),
        expect.objectContaining({
          op: 'configure_node',
          target: 'self',
          args: expect.objectContaining({
            config: expect.objectContaining({
              inputs: [
                {
                  name: 'input',
                  from: 'smoke_58_text_source',
                  output: 'output',
                  type: 'string',
                },
              ],
            }),
          }),
        }),
      ]),
    );
  });

  it('provides executable Question setup with context input and answer output', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('18');

    expect(scenario.setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          op: 'connect',
          args: { from: 'start', to: 'self' },
        }),
        expect.objectContaining({
          op: 'configure_node',
          target: 'self',
          args: expect.objectContaining({
            config: expect.objectContaining({
              input: { name: 'context', from: 'start', output: 'input' },
              question: '请补充必要信息: {{context}}',
              answer_type: 'text',
              outputs: [{ name: 'answer', type: 'string' }],
            }),
          }),
        }),
      ]),
    );
  });

  it('documents End text-return mode with streaming templates', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('2');

    expect(scenario).toMatchObject({
      status: 'verified-not-executable',
      nodeName: '结束',
    });
    expect(scenario.assertions.join('\n')).toContain('返回文本');
    expect(scenario.assertions.join('\n')).toContain('streaming_output=true');
    expect(scenario.assertions.join('\n')).toContain('多个上游变量');
  });

  it('provides End text-return setup without adding another End node', () => {
    const scenario = getWorkflowAgentNodeSmokeScenario('2');

    expect(scenario.setupCommands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          op: 'add_node',
          target: 'smoke_2_text_source',
          args: expect.objectContaining({ type: '15' }),
        }),
        expect.objectContaining({
          op: 'connect',
          args: { from: 'smoke_2_text_source', to: 'end' },
        }),
        expect.objectContaining({
          op: 'configure_node',
          target: 'end',
          args: expect.objectContaining({
            config: expect.objectContaining({
              content: '最终结果: {{answer}}',
              streaming_output: true,
            }),
          }),
        }),
      ]),
    );
    expect(
      scenario.setupCommands.some(
        command => command.op === 'add_node' && command.args?.type === '2',
      ),
    ).toBe(false);
  });
});
