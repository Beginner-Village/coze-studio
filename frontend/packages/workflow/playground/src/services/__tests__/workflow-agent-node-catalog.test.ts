import { formatWorkflowAgentNodeCatalog } from '../workflow-agent-node-catalog';

describe('workflow-agent-node-catalog', () => {
  it('formats available nodes with capability and configuration guidance', () => {
    const catalog = formatWorkflowAgentNodeCatalog([
      { type: '3', title: '大模型' },
      { type: '5', title: '代码' },
      { type: '8', title: '条件分支' },
      { type: '13', title: '输出' },
      { type: '30', title: '输入' },
      { type: '6', title: '知识库' },
      { type: '100', title: '智能体' },
    ]);

    expect(catalog).toContain('3 大模型');
    expect(catalog).toContain('文本理解、生成、分类、抽取');
    expect(catalog).toContain('5 代码');
    expect(catalog).toContain('声明 outputs');
    expect(catalog).toContain('async def main(args: Args)');
    expect(catalog).toContain('args.params.get');
    expect(catalog).toContain('8 条件分支');
    expect(catalog).toContain('true/false');
    expect(catalog).toContain('13 输出');
    expect(catalog).toContain('纯输出');
    expect(catalog).toContain('不是稳定的下游变量来源');
    expect(catalog).toContain('不要接变量聚合');
    expect(catalog).toContain('30 输入');
    expect(catalog).toContain('声明工作流入参');
    expect(catalog).toContain('6 知识库');
    expect(catalog).toContain('dataset_id');
    expect(catalog).toContain('100 智能体');
    expect(catalog).toContain('agent_id');
  });

  it('guides branch replies through Text before VariableMerge', () => {
    const catalog = formatWorkflowAgentNodeCatalog([
      { type: '15', title: '文本处理' },
      { type: '32', title: '变量聚合' },
    ]);

    expect(catalog).toContain('真实 output 变量');
    expect(catalog).toContain('固定文案不需要变量时可不绑定输入');
    expect(catalog).toContain('不要聚合 type=13');
    expect(catalog).toContain('type=15 文本处理产出 output');
  });

  it('keeps unknown loaded node types visible instead of dropping them', () => {
    const catalog = formatWorkflowAgentNodeCatalog([
      { type: '777', title: '自定义节点' },
    ]);

    expect(catalog).toContain('777 自定义节点');
    expect(catalog).toContain('按节点标题判断用途');
  });
});
