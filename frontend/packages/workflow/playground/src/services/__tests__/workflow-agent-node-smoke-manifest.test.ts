import { describe, expect, it } from 'vitest';

import {
  buildWorkflowAgentNodeSmokeManifest,
} from '../workflow-agent-node-smoke-manifest';
import {
  WORKFLOW_AGENT_NODE_CAPABILITIES,
  WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES,
} from '../workflow-agent-node-capabilities';
import { WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS } from '../workflow-agent-node-smoke-scenarios';

describe('workflow-agent-node-smoke-manifest', () => {
  it('exports a stable readiness manifest for all visible workflow nodes', () => {
    const manifest = buildWorkflowAgentNodeSmokeManifest({
      capabilities: WORKFLOW_AGENT_NODE_CAPABILITIES,
      scenarios: WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
    });

    expect(manifest.total).toBe(WORKFLOW_AGENT_VISIBLE_NODE_REGISTRY_TYPES.length);
    expect(manifest.coverageGaps).toEqual([]);
    expect(manifest.readiness.notReady).toEqual([]);
    expect(manifest.summary.ready).toBeGreaterThan(0);
    expect(manifest.summary.skipped).toBeGreaterThan(0);
    expect(manifest.summary.requiresTemporaryWorkflowOptIn).toBeGreaterThan(0);

    const variableMerge = manifest.nodes.find(node => node.nodeType === '32');

    expect(variableMerge).toMatchObject({
      nodeName: '变量聚合',
      mode: 'execute',
      isolation: 'temporary-workflow',
      ready: true,
      requiresTemporaryWorkflowOptIn: true,
      plannedCommands: expect.arrayContaining([
        'add_node',
        'connect',
        'get_bindable_variables',
        'configure_node',
        'validate',
        'test_run',
        'auto_layout',
      ]),
      cleanupCommands: ['delete_node', 'delete_node', 'delete_node', 'auto_layout'],
      temporaryNodeTags: [
        'smoke_32_variable_merge',
        'smoke_32_text_primary',
        'smoke_32_text_fallback',
      ],
      assertions: expect.arrayContaining([
        '不能引用 type=13 输出节点。',
      ]),
      expectedBindableVariables: ['merge.output'],
    });
  });

  it('marks resource-bound nodes without fixtures as explicit skipped work', () => {
    const manifest = buildWorkflowAgentNodeSmokeManifest({
      capabilities: WORKFLOW_AGENT_NODE_CAPABILITIES,
      scenarios: WORKFLOW_AGENT_NODE_SMOKE_SCENARIOS,
    });

    expect(manifest.nodes.find(node => node.nodeType === '4')).toMatchObject({
      nodeName: '插件/API',
      mode: 'skip',
      ready: false,
      requiresResourceFixture: true,
      skipReason: '运行依赖空间插件/API资源。',
    });
  });
});
