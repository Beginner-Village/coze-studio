/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {
  WORKFLOW_AGENT_NODE_BINDING_GUIDE,
  discoverWorkflowAgentResources,
  formatWorkflowAgentResourceSummary,
} from '../workflow-agent-resource-service';

describe('workflow-agent-resource-service', () => {
  it('formats plugin api, knowledge, and agent ids for the workflow copilot prompt', async () => {
    const seenSpaceIds: string[] = [];
    const resources = await discoverWorkflowAgentResources('space-1', {
      getPlugins: spaceId => {
        seenSpaceIds.push(spaceId);
        return Promise.resolve([
          {
            id: 'plugin-1',
            name: '订单插件',
            desc: '订单查询与售后',
            plugin_type: 1,
            plugin_apis: [
              {
                api_id: 'api-1',
                name: 'query_order',
                desc: '按订单号查询订单',
              },
            ],
          },
        ]);
      },
      getKnowledgeBases: spaceId => {
        seenSpaceIds.push(spaceId);
        return Promise.resolve([
          {
            dataset_id: 'dataset-1',
            name: '商品知识库',
            description: '商品资料和 FAQ',
          },
        ]);
      },
      getCozeAgents: spaceId => {
        seenSpaceIds.push(spaceId);
        return Promise.resolve([
          {
            id: 'agent-1',
            name: '客服助手',
            description: '处理售前咨询',
          },
        ]);
      },
      getHiAgents: spaceId => {
        seenSpaceIds.push(spaceId);
        return Promise.resolve([
          {
            id: 'hiagent-1',
            name: '外部质检',
            description: '质检流程',
          },
        ]);
      },
    });

    const summary = formatWorkflowAgentResourceSummary(resources);

    expect(seenSpaceIds).toEqual(['space-1', 'space-1', 'space-1', 'space-1']);
    expect(summary).toContain('插件/API');
    expect(summary).toContain('plugin-1');
    expect(summary).toContain('api-1');
    expect(summary).toContain('query_order');
    expect(summary).toContain('知识库');
    expect(summary).toContain('dataset-1');
    expect(summary).toContain('智能体');
    expect(summary).toContain('agent-1');
    expect(summary).toContain('hiagent-1');
  });

  it('keeps a failed resource category as a warning instead of rejecting discovery', async () => {
    const resources = await discoverWorkflowAgentResources('space-1', {
      getPlugins: () => Promise.reject(new Error('plugin api failed')),
      getKnowledgeBases: () => Promise.resolve([]),
      getCozeAgents: () => Promise.resolve([]),
      getHiAgents: () => Promise.resolve([]),
    });

    expect(resources.plugins).toEqual([]);
    expect(resources.warnings).toContain('插件/API: plugin api failed');
    expect(formatWorkflowAgentResourceSummary(resources)).toContain(
      '资源发现警告',
    );
  });

  it('describes Python code node runtime args in the binding guide', () => {
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('language="python"');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      'async def main(args: Args)',
    );
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('args.params');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      '不要对 args 直接 strip/get/[]',
    );
  });

  it('requires a post-mutation binding audit before claiming completion', () => {
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('每次完成一组');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      'workflow_canvas_get_canvas_context',
    );
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('从 Start 到 End');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('输入绑定');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('End returns');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('发送前快照');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('绑定诊断');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('不是 none');
  });

  it('tells the agent to fix bindings locally instead of clearing on errors', () => {
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      '禁止默认 clear_canvas',
    );
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('局部修复输入绑定');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('BlockID is empty');
  });

  it('requires configure and bindable-variable checks from inputs to End before test run', () => {
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      '先 add_node 创建节点,再 connect 连线,然后 configure_node 逐个配置',
    );
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      '配置 condition/merge_groups/returns 前必须先调用 workflow_canvas_get_bindable_variables',
    );
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      '从输入到输出逐节点确认',
    );
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain(
      '没有变量引用时可以删除默认 input',
    );
  });

  it('documents End text-return mode with streaming output and variable templates', () => {
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('End节点');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('返回文本');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('streaming_output=true');
    expect(WORKFLOW_AGENT_NODE_BINDING_GUIDE).toContain('多个输出变量');
  });
});
