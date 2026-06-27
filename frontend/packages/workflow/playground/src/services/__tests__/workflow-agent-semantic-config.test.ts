import { describe, expect, it } from 'vitest';

import { createWorkflowAgentSemanticParams } from '../workflow-agent-semantic-config';
import { summarizeWorkflowAgentCanvas } from '../workflow-agent-canvas-summary';

const refExpr = (blockID: string, name: string) => ({
  type: 'ref',
  content: {
    keyPath: [blockID, name],
  },
  rawMeta: { type: 1 },
});

const literalExpr = (content: string) => ({
  type: 'literal',
  content,
  rawMeta: { type: 1 },
});

describe('workflow-agent-semantic-config', () => {
  it('creates schema-aware LLM params with upstream input binding and output definition', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '3',
      config: {
        title: '大模型处理',
        input: { from: 'start', output: 'input', name: 'input' },
        prompt: '请总结 {{input}}',
        system_prompt: '你是严谨的工作流处理节点',
        outputs: [{ name: 'answer', type: 'string' }],
      },
      resolveNodeRef: ref => (ref === 'start' ? '100001' : ref),
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '大模型处理',
      '$$prompt_decorator$$.prompt': '请总结 {{input}}',
      '$$prompt_decorator$$.systemPrompt': '你是严谨的工作流处理节点',
    });
    expect(params['$$input_decorator$$.inputParameters']).toEqual([
      {
        name: 'input',
        input: refExpr('100001', 'input'),
      },
    ]);
    expect(params.outputs).toMatchObject([{ name: 'answer', type: 1 }]);
  });

  it('creates End params that return an upstream variable instead of an undefined output', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '2',
      config: {
        returns: [{ name: 'output', from: 'llm', output: 'answer' }],
      },
      resolveNodeRef: ref => (ref === 'llm' ? '173000' : ref),
    });

    expect(params).toEqual({
      'inputs.terminatePlan': 'returnVariables',
      'inputs.inputParameters': [
        {
          name: 'output',
          input: refExpr('173000', 'answer'),
        },
      ],
    });
  });

  it('creates End params that stream answer text assembled from multiple variables', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '2',
      config: {
        inputs: [
          { name: 'intent', from: 'intent_node', output: 'category' },
          { name: 'answer', from: 'merge', output: 'output' },
        ],
        content: '业务类型: {{intent}}\n处理结果: {{answer}}',
        streaming_output: true,
      },
      resolveNodeRef: ref =>
        ref === 'intent_node' ? '173010' : ref === 'merge' ? '173011' : ref,
    });

    expect(params).toEqual({
      'inputs.terminatePlan': 'useAnswerContent',
      'inputs.inputParameters': [
        {
          name: 'intent',
          input: refExpr('173010', 'category'),
        },
        {
          name: 'answer',
          input: refExpr('173011', 'output'),
        },
      ],
      'inputs.content': '业务类型: {{intent}}\n处理结果: {{answer}}',
      'inputs.streamingOutput': true,
    });
  });

  it('creates Code params with input bindings, executable code, and declared outputs', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '5',
      config: {
        input: { from: 'start', output: 'input', name: 'query' },
        language: 'javascript',
        code: 'async function main({ query }) { return { result: query.trim() }; }',
        outputs: [{ name: 'result', type: 'string' }],
      },
      resolveNodeRef: ref => (ref === 'start' ? '100001' : ref),
    });

    expect(params).toMatchObject({
      'codeParams.language': 'javascript',
      'codeParams.code':
        'async function main({ query }) { return { result: query.trim() }; }',
    });
    expect(params.inputParameters).toEqual([
      {
        name: 'query',
        input: refExpr('100001', 'input'),
      },
    ]);
    expect(params.outputs).toMatchObject([{ name: 'result', type: 1 }]);
  });

  it('creates If params that bind the left side to an upstream variable', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '8',
      config: {
        condition: {
          left: { from: 'intent', output: 'category' },
          operator: 'equal',
          right: '售后',
        },
      },
      resolveNodeRef: ref => (ref === 'intent' ? '173001' : ref),
    });

    expect(params.condition).toEqual([
      {
        condition: {
          logic: 2,
          conditions: [
            {
              left: refExpr('173001', 'category'),
              operator: 1,
              right: literalExpr('售后'),
            },
          ],
        },
      },
    ]);
  });

  it('creates Plugin/API params with resource identifiers and input bindings', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '4',
      config: {
        plugin_id: 'plugin-1',
        api_id: 'api-1',
        plugin_name: '订单插件',
        api_name: '查询订单',
        inputs: {
          order_id: { from: 'start', output: 'input' },
        },
        outputs: [{ name: 'result', type: 'string' }],
      },
      resolveNodeRef: ref => (ref === 'start' ? '100001' : ref),
    });

    expect(params['inputs.apiParam']).toMatchObject([
      { name: 'apiName', input: { value: { content: '查询订单' } } },
      { name: 'pluginID', input: { value: { content: 'plugin-1' } } },
      { name: 'apiID', input: { value: { content: 'api-1' } } },
      { name: 'pluginName', input: { value: { content: '订单插件' } } },
    ]);
    expect(params['inputs.inputParameters']).toEqual({
      order_id: refExpr('100001', 'input'),
    });
    expect(params.outputs).toMatchObject([{ name: 'result', type: 1 }]);
  });

  it('creates Knowledge params with dataset selection, query binding, and outputs', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '6',
      config: {
        dataset_ids: ['dataset-1'],
        input: { from: 'start', output: 'input', name: 'Query' },
      },
      resolveNodeRef: ref => (ref === 'start' ? '100001' : ref),
    });

    expect(params['inputs.inputParameters.Query']).toEqual(
      refExpr('100001', 'input'),
    );
    expect(params['inputs.datasetParameters.datasetParam']).toEqual([
      'dataset-1',
    ]);
    expect(params['inputs.datasetParameters.datasetSetting']).toMatchObject({
      top_k: 5,
      use_rerank: false,
      use_rewrite: false,
      is_personal_only: false,
    });
    expect(params.outputs).toMatchObject([
      {
        name: 'outputList',
        type: 103,
        children: [{ name: 'output', type: 1 }],
      },
    ]);
  });

  it('creates Agent params with agent binding and query variable binding', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '100',
      config: {
        agent_id: 'agent-1',
        agent_name: '客服智能体',
        platform: 'hiagent',
        input: { from: 'start', output: 'input', name: 'query' },
        outputs: [{ name: 'answer', type: 'string' }],
      },
      resolveNodeRef: ref => (ref === 'start' ? '100001' : ref),
    });

    expect(params).toMatchObject({
      'inputs.agent_id': 'agent-1',
      'inputs.agent_name': '客服智能体',
      'inputs.platform': 'hiagent',
      'inputs.query': '',
    });
    expect(params['inputs.inputParameters']).toEqual([
      {
        name: 'query',
        input: refExpr('100001', 'input'),
      },
    ]);
    expect(params.outputs).toMatchObject([{ name: 'answer', type: 1 }]);
  });

  it('creates Variable Merge params with branch variables and merge output', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '32',
      config: {
        title: '分支结果合并',
        merge_groups: [
          {
            name: 'output',
            variables: [
              { from: 'after_sale_code', output: 'result' },
              { from: 'general_agent', output: 'answer' },
            ],
          },
        ],
      },
      resolveNodeRef: ref =>
        ref === 'after_sale_code'
          ? '138999'
          : ref === 'general_agent'
            ? '153481'
            : ref,
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '分支结果合并',
      'inputs.mergeGroups': [
        {
          name: 'output',
          variables: [
            refExpr('138999', 'result'),
            refExpr('153481', 'answer'),
          ],
        },
      ],
    });
    expect(params.outputs).toMatchObject([{ name: 'output', type: 1 }]);
  });

  it('creates Output params with upstream bindings and answer content', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '13',
      config: {
        title: '直接输出',
        input: { from: 'normalize', output: 'result', name: 'result' },
        content: '处理结果：{{result}}',
        streaming_output: true,
      },
      resolveNodeRef: ref => (ref === 'normalize' ? '173010' : ref),
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '直接输出',
      'inputs.content': '处理结果：{{result}}',
      'inputs.streamingOutput': true,
    });
    expect(params['inputs.inputParameters']).toEqual([
      {
        name: 'result',
        input: refExpr('173010', 'result'),
      },
    ]);
  });

  it('does not invent downstream output variables for Output nodes', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '13',
      config: {
        title: '业务回复',
        input: { from: 'llm', output: 'answer', name: 'answer' },
        content: '{{answer}}',
      },
      resolveNodeRef: ref => (ref === 'llm' ? '173030' : ref),
    });

    expect(params.outputs).toBeUndefined();
  });

  it('clears Output node inputs when rendering fixed content without variables', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '13',
      config: {
        title: '固定提示',
        content: '请选择一个业务类型',
      },
      resolveNodeRef: ref => ref,
    });

    expect(params['inputs.inputParameters']).toEqual([]);
    expect(params['inputs.content']).toBe('请选择一个业务类型');
  });

  it('creates Text params using real concat paths and template bindings', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '15',
      config: {
        title: '组装回复',
        method: 'concat',
        input: { from: 'direct_output', output: 'output', name: 'reply' },
        content: '最终回复：{{reply}}',
        outputs: [{ name: 'output', type: 'string' }],
      },
      resolveNodeRef: ref => (ref === 'direct_output' ? '173020' : ref),
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '组装回复',
      method: 'concat',
      concatResult: '最终回复：{{reply}}',
    });
    expect(params.inputParameters).toEqual([
      {
        name: 'reply',
        input: refExpr('173020', 'output'),
      },
    ]);
    expect(params.outputs).toMatchObject([{ name: 'output', type: 1 }]);
  });

  it('clears Text node inputs when fixed text does not need variables', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '15',
      config: {
        title: '固定回复',
        method: 'concat',
        content: '您好，请问有什么可以帮您？',
      },
      resolveNodeRef: ref => ref,
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '固定回复',
      method: 'concat',
      concatResult: '您好，请问有什么可以帮您？',
    });
    expect(params.inputParameters).toEqual([]);
    expect(params.outputs).toMatchObject([{ name: 'output', type: 1 }]);
  });

  it('creates Question params with answer options and declared outputs', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '18',
      config: {
        title: '追问收款卡号',
        question: '请补充收款卡号',
        answer_type: 'option',
        option_type: 'single',
        options: ['最近收款人', '手动输入'],
        limit: 1,
        extra_output: true,
        outputs: [{ name: 'answer', type: 'string' }],
      },
      resolveNodeRef: ref => ref,
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '追问收款卡号',
      'questionParams.question': '请补充收款卡号',
      'questionParams.answer_type': 'option',
      'questionParams.option_type': 'single',
      'questionOutputs.limit': 1,
      'questionOutputs.extra_output': true,
    });
    expect(params['questionParams.options']).toEqual([
      { id: '1', name: '最近收款人', content: '最近收款人' },
      { id: '2', name: '手动输入', content: '手动输入' },
    ]);
    expect(params.outputs).toMatchObject([{ name: 'answer', type: 1 }]);
  });

  it('creates Input params by declaring structured outputs', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '30',
      config: {
        title: '结构化输入',
        outputs: [
          { name: 'account_no', type: 'string' },
          { name: 'amount', type: 'number' },
        ],
      },
      resolveNodeRef: ref => ref,
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '结构化输入',
    });
    expect(params.outputs).toMatchObject([
      { name: 'account_no', type: 1 },
      { name: 'amount', type: 4 },
    ]);
  });

  it('creates JSON stringify params with input binding and default output', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '58',
      config: {
        input: { from: 'payload_node', output: 'payload', name: 'payload' },
      },
      resolveNodeRef: ref => (ref === 'payload_node' ? '173040' : ref),
    });

    expect(params['inputs.inputParameters']).toEqual([
      {
        name: 'payload',
        input: refExpr('173040', 'payload'),
      },
    ]);
    expect(params.outputs).toMatchObject([{ name: 'output', type: 1 }]);
  });

  it('creates Card selector params with selected card and input bindings', () => {
    const params = createWorkflowAgentSemanticParams({
      nodeType: '99',
      config: {
        title: '卡片选择',
        input: { from: 'start', output: 'input', name: 'query' },
        content: '请选择：{{query}}',
        card_id: 'card-1',
        card_name: '转账确认卡片',
        card_code: 'transfer_confirm',
      },
      resolveNodeRef: ref => (ref === 'start' ? '100001' : ref),
    });

    expect(params).toMatchObject({
      'nodeMeta.title': '卡片选择',
      'inputs.content': '请选择：{{query}}',
      'inputs.selectedCard': {
        cardId: 'card-1',
        cardName: '转账确认卡片',
        code: 'transfer_confirm',
      },
    });
    expect(params['inputs.inputParameters']).toEqual([
      {
        name: 'query',
        input: refExpr('100001', 'input'),
      },
    ]);
  });

  it('summarizes canvas outputs and bindable variables for the agent', () => {
    const summary = summarizeWorkflowAgentCanvas({
      nodes: [
        {
          id: '100001',
          type: '1',
          data: {
            nodeMeta: { title: '开始' },
            outputs: [{ name: 'input', type: 'string' }],
          },
        },
        {
          id: '173000',
          type: '3',
          data: {
            nodeMeta: { title: '大模型处理' },
            outputs: [{ name: 'answer', type: 1 }],
            inputs: { inputParameters: [] },
          },
        },
      ],
      edges: [{ sourceNodeID: '100001', targetNodeID: '173000' }],
    });

    expect(summary).toContain('开始(100001,type=1) outputs: input:string');
    expect(summary).toContain(
      '大模型处理(173000,type=3) outputs: answer:string',
    );
    expect(summary).toContain(
      '可绑定变量: 100001.input:string, 173000.answer:string',
    );
  });
});
