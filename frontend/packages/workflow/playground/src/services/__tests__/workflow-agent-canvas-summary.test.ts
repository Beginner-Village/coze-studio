import { describe, expect, it } from 'vitest';

import {
  collectWorkflowAgentBindableVariables,
  formatWorkflowAgentBindableVariables,
  summarizeWorkflowAgentCanvas,
} from '../workflow-agent-canvas-summary';

describe('workflow-agent-canvas-summary', () => {
  it('summarizes DTO-style bindings, IF conditions, object inputs, and End returns', () => {
    const summary = summarizeWorkflowAgentCanvas({
      nodes: [
        {
          id: '100001',
          type: '1',
          data: {
            nodeMeta: { title: '开始' },
            outputs: [{ name: 'input', type: 1 }],
          },
        },
        {
          id: '184844',
          type: '3',
          data: {
            nodeMeta: { title: '意图分类LLM' },
            inputs: {
              inputParameters: [
                {
                  name: 'input',
                  input: {
                    type: 'string',
                    value: {
                      type: 'ref',
                      content: {
                        source: 'block-output',
                        blockID: '100001',
                        name: 'input',
                      },
                      rawMeta: { type: 1 },
                    },
                  },
                },
              ],
            },
            outputs: [
              { name: 'category', type: 1 },
              { name: 'answer', type: 1 },
            ],
          },
        },
        {
          id: '144621',
          type: '8',
          data: {
            nodeMeta: { title: '售后判断' },
            inputs: {
              branches: [
                {
                  condition: {
                    logic: 2,
                    conditions: [
                      {
                        operator: 1,
                        left: {
                          input: {
                            type: 'string',
                            value: {
                              type: 'ref',
                              content: {
                                source: 'block-output',
                                blockID: '184844',
                                name: 'category',
                              },
                              rawMeta: { type: 1 },
                            },
                          },
                        },
                        right: {
                          input: {
                            type: 'string',
                            value: {
                              type: 'literal',
                              content: '售后',
                              rawMeta: { type: 1 },
                            },
                          },
                        },
                      },
                    ],
                  },
                },
              ],
            },
          },
        },
        {
          id: '199001',
          type: '4',
          data: {
            nodeMeta: { title: '查询订单API' },
            inputs: {
              inputParameters: {
                order_id: {
                  type: 'ref',
                  content: { keyPath: ['100001', 'input'] },
                  rawMeta: { type: 1 },
                },
              },
            },
            outputs: [{ name: 'result', type: 1 }],
          },
        },
        {
          id: '900001',
          type: '2',
          data: {
            nodeMeta: { title: '结束' },
            inputs: {
              inputParameters: [
                {
                  name: 'output',
                  input: {
                    type: 'string',
                    value: {
                      type: 'ref',
                      content: {
                        source: 'block-output',
                        blockID: '199001',
                        name: 'result',
                      },
                      rawMeta: { type: 1 },
                    },
                  },
                },
              ],
            },
          },
        },
      ],
      edges: [
        { sourceNodeID: '100001', targetNodeID: '184844' },
        { sourceNodeID: '184844', targetNodeID: '144621' },
      ],
    });

    expect(summary).toContain('意图分类LLM(184844,type=3)');
    expect(summary).toContain('inputs: input=100001.input');
    expect(summary).toContain('售后判断(144621,type=8)');
    expect(summary).toContain('conditions: IF 184844.category equal 售后');
    expect(summary).toContain('查询订单API(199001,type=4)');
    expect(summary).toContain('inputs: order_id=100001.input');
    expect(summary).toContain('结束(900001,type=2)');
    expect(summary).toContain('returns: output=199001.result');
  });

  it('includes current validation errors with node titles and ids', () => {
    const summary = summarizeWorkflowAgentCanvas(
      {
        nodes: [
          {
            id: '124149',
            type: '22',
            data: { nodeMeta: { title: '意图识别' } },
          },
          {
            id: '161803',
            type: '13',
            data: { nodeMeta: { title: '输出业务名称' } },
          },
        ],
        edges: [],
      },
      {
        validationErrors: {
          '124149': [
            {
              nodeId: '124149',
              errorInfo: '变量值不可为空;选项内容不能为空',
              errorLevel: 'error',
              errorType: 'node',
            },
          ],
          '161803': [
            {
              nodeId: '161803',
              errorInfo: '引用变量不存在',
              errorLevel: 'error',
              errorType: 'node',
            },
          ],
        },
      },
    );

    expect(summary).toContain('当前校验错误:');
    expect(summary).toContain(
      '[error] 意图识别(124149,type=22): 变量值不可为空;选项内容不能为空',
    );
    expect(summary).toContain(
      '[error] 输出业务名称(161803,type=13): 引用变量不存在',
    );
  });

  it('diagnoses invalid merge and end bindings against real bindable outputs', () => {
    const summary = summarizeWorkflowAgentCanvas({
      nodes: [
        {
          id: '100001',
          type: '1',
          data: {
            nodeMeta: { title: '开始' },
            outputs: [{ name: 'input', type: 1 }],
          },
        },
        {
          id: '110001',
          type: '13',
          data: {
            nodeMeta: { title: '纯输出回复' },
            inputs: { content: '固定回复' },
          },
        },
        {
          id: '120001',
          type: '15',
          data: {
            nodeMeta: { title: '文本处理' },
            inputs: { concatResult: '固定回复' },
            outputs: [{ name: 'output', type: 1 }],
          },
        },
        {
          id: '130001',
          type: '32',
          data: {
            nodeMeta: { title: '聚合' },
            inputs: {
              mergeGroups: [
                {
                  name: 'output',
                  variables: [
                    {
                      type: 'ref',
                      content: { keyPath: ['110001', 'output'] },
                      rawMeta: { type: 1 },
                    },
                    {
                      type: 'ref',
                      content: { keyPath: ['120001', 'missing'] },
                      rawMeta: { type: 1 },
                    },
                  ],
                },
              ],
            },
            outputs: [{ name: 'output', type: 1 }],
          },
        },
        {
          id: '900001',
          type: '2',
          data: {
            nodeMeta: { title: '结束' },
            inputs: {
              inputParameters: [
                {
                  name: 'output',
                  input: {
                    type: 'ref',
                    content: { keyPath: ['130001', 'missing'] },
                    rawMeta: { type: 1 },
                  },
                },
              ],
            },
          },
        },
      ],
    });

    expect(summary).toContain('绑定诊断:');
    expect(summary).toContain(
      '聚合(130001,type=32) merge_groups.output 引用 110001.output 不可绑定',
    );
    expect(summary).toContain(
      '纯输出回复(110001,type=13) 是消息展示节点且没有真实 outputs',
    );
    expect(summary).toContain(
      '聚合(130001,type=32) merge_groups.output 引用 120001.missing 不可绑定',
    );
    expect(summary).toContain(
      '结束(900001,type=2) returns.output 引用 130001.missing 不可绑定',
    );
  });

  it('summarizes block-style workflow nodes, merge groups, text content, and returns', () => {
    const canvas = {
      blocks: [
        {
          id: '100001',
          type: '1',
          data: {
            nodeMeta: { title: '开始' },
            outputs: [{ name: 'input', type: 1 }],
          },
        },
        {
          id: '110001',
          type: '13',
          data: {
            nodeMeta: { title: '转账业务回复' },
            inputs: {
              inputParameters: [
                {
                  name: 'query',
                  input: {
                    type: 'string',
                    value: {
                      type: 'ref',
                      content: {
                        source: 'block-output',
                        blockID: '100001',
                        name: 'input',
                      },
                      rawMeta: { type: 1 },
                    },
                  },
                },
              ],
              content: '这是转账业务: {{query}}',
            },
          },
        },
        {
          id: '120001',
          type: '15',
          data: {
            nodeMeta: { title: '文本处理' },
            inputs: {
              inputParameters: [
                {
                  name: 'reply',
                  input: {
                    type: 'ref',
                    content: { keyPath: ['100001', 'input'] },
                    rawMeta: { type: 1 },
                  },
                },
              ],
              concatParams: [
                {
                  name: 'concatResult',
                  input: {
                    type: 'string',
                    value: {
                      type: 'literal',
                      content: '最终回复: {{reply}}',
                    },
                  },
                },
              ],
            },
            outputs: [{ name: 'output', type: 1 }],
          },
        },
        {
          id: '130001',
          type: '32',
          data: {
            nodeMeta: { title: '输出聚合' },
            inputs: {
              mergeGroups: [
                {
                  name: 'output',
                  variables: [
                    {
                      type: 'ref',
                      content: { keyPath: ['120001', 'output'] },
                      rawMeta: { type: 1 },
                    },
                  ],
                },
              ],
            },
            outputs: [{ name: 'output', type: 1 }],
          },
        },
        {
          id: '900001',
          type: '2',
          data: {
            nodeMeta: { title: '结束' },
            inputs: {
              terminatePlan: 'returnVariables',
              inputParameters: [
                {
                  name: 'output',
                  input: {
                    type: 'ref',
                    content: { keyPath: ['130001', 'output'] },
                    rawMeta: { type: 1 },
                  },
                },
              ],
            },
          },
        },
      ],
      edges: [
        { sourceNodeID: '100001', targetNodeID: '110001' },
        { sourceNodeID: '110001', targetNodeID: '130001' },
        { sourceNodeID: '130001', targetNodeID: '900001' },
      ],
    };
    const summary = summarizeWorkflowAgentCanvas(canvas);

    expect(summary).toContain('转账业务回复(110001,type=13)');
    expect(summary).toContain('inputs: query=100001.input');
    expect(summary).toContain('content: 这是转账业务: {{query}}');
    expect(summary).toContain('不可作为聚合变量的输出节点');
    expect(summary).toContain('type=15 文本处理节点产出 output:string');
    expect(summary).toContain('文本处理(120001,type=15)');
    expect(summary).toContain('content: 最终回复: {{reply}}');
    expect(summary).toContain('merge_groups: output=[120001.output]');
    expect(summary).toContain('returns: output=130001.output');
    expect(summary).toContain(
      '可绑定变量: 100001.input:string, 120001.output:string, 130001.output:string',
    );

    expect(collectWorkflowAgentBindableVariables(canvas)).toEqual([
      {
        nodeId: '100001',
        nodeTitle: '开始',
        nodeType: '1',
        name: 'input',
        type: 'string',
        ref: '100001.input',
      },
      {
        nodeId: '120001',
        nodeTitle: '文本处理',
        nodeType: '15',
        name: 'output',
        type: 'string',
        ref: '120001.output',
      },
      {
        nodeId: '130001',
        nodeTitle: '输出聚合',
        nodeType: '32',
        name: 'output',
        type: 'string',
        ref: '130001.output',
      },
    ]);
    expect(formatWorkflowAgentBindableVariables(canvas)).toContain(
      '- 120001.output:string (文本处理, type=15)',
    );
    expect(formatWorkflowAgentBindableVariables(canvas)).toContain(
      '绑定格式示例',
    );
    expect(formatWorkflowAgentBindableVariables(canvas)).not.toContain(
      '110001.output',
    );
  });

  it('does not diagnose fixed-text Text nodes that intentionally remove default inputs', () => {
    const summary = summarizeWorkflowAgentCanvas({
      nodes: [
        {
          id: '100001',
          type: '1',
          data: {
            nodeMeta: { title: '开始' },
            outputs: [{ name: 'input', type: 1 }],
          },
        },
        {
          id: '120001',
          type: '15',
          data: {
            nodeMeta: { title: '固定回复' },
            inputs: {
              concatParams: [
                {
                  name: 'concatResult',
                  input: {
                    type: 'string',
                    value: {
                      type: 'literal',
                      content: '好的,已收到。',
                    },
                  },
                },
              ],
            },
            outputs: [{ name: 'output', type: 1 }],
          },
        },
        {
          id: '900001',
          type: '2',
          data: {
            nodeMeta: { title: '结束' },
            inputs: {
              inputParameters: [
                {
                  name: 'output',
                  input: {
                    type: 'ref',
                    content: { keyPath: ['120001', 'output'] },
                    rawMeta: { type: 1 },
                  },
                },
              ],
            },
          },
        },
      ],
      edges: [
        { sourceNodeID: '100001', targetNodeID: '120001' },
        { sourceNodeID: '120001', targetNodeID: '900001' },
      ],
    });

    expect(summary).toContain('固定回复(120001,type=15)');
    expect(summary).toContain('content: 好的,已收到。');
    expect(summary).toContain('可绑定变量: 100001.input:string, 120001.output:string');
    expect(summary).toContain('绑定诊断: none');
  });
});
