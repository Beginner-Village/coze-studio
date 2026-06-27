import {
  createWorkflowAgentPlaceholderNodeJson,
  isWorkflowAgentSingletonNodeType,
  resolveWorkflowAgentSingletonNodeRef,
} from '../workflow-agent-node-json';

describe('workflow-agent-node-json', () => {
  it('creates a renderable API placeholder node when no plugin resource is bound', () => {
    const nodeJson = createWorkflowAgentPlaceholderNodeJson(
      '4',
      '订单查询插件-需绑定资源',
    );

    expect(nodeJson?.data?.nodeMeta?.title).toBe('订单查询插件-需绑定资源');
    expect(nodeJson?.data?.inputs?.apiParam?.map(item => item.name)).toEqual([
      'apiID',
      'apiName',
      'pluginID',
      'pluginName',
      'pluginVersion',
      'tips',
      'outDocLink',
    ]);
    expect(nodeJson?.data?.outputs).toEqual([]);
  });

  it('does not create placeholders for ordinary nodes', () => {
    expect(createWorkflowAgentPlaceholderNodeJson('3', 'LLM')).toBeUndefined();
  });

  it('marks start and end as singleton node types', () => {
    expect(isWorkflowAgentSingletonNodeType('1')).toBe(true);
    expect(isWorkflowAgentSingletonNodeType('2')).toBe(true);
    expect(isWorkflowAgentSingletonNodeType('3')).toBe(false);
  });

  it('resolves stable singleton node aliases', () => {
    expect(resolveWorkflowAgentSingletonNodeRef('start')).toBe('100001');
    expect(resolveWorkflowAgentSingletonNodeRef('Start')).toBe('100001');
    expect(resolveWorkflowAgentSingletonNodeRef('end')).toBe('900001');
    expect(resolveWorkflowAgentSingletonNodeRef('llm')).toBeUndefined();
  });
});
