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

import { useMemo } from 'react';

import { Button, Modal, Typography } from '@coze-arch/coze-design';

import { CopyButton } from '../copy-button';

// 平台内置的工作流 MCP 端点(streamable-HTTP 标准 MCP 协议)。
const WORKFLOW_MCP_PATH = '/api/workflow_mcp/mcp';

interface ToolGroup {
  title: string;
  desc: string;
  tools: string[];
}

const TOOL_GROUPS: ToolGroup[] = [
  {
    title: '画布编辑类',
    desc: '需在本编辑器打开对应工作流(命令经本页面实时执行)',
    tools: [
      'add_node',
      'connect',
      'configure_node',
      'set_node_params',
      'delete_node',
      'delete_line',
      'clear_canvas',
      'auto_layout',
    ],
  },
  {
    title: '查询类',
    desc: '可独立使用,无需打开编辑器',
    tools: [
      'get_canvas_context',
      'get_bindable_variables',
      'list_node_capabilities',
      'get_node_spec',
      'list_surfaces',
      'list_resource_catalog',
    ],
  },
  {
    title: '执行类',
    desc: '可独立使用,用于调试/排障',
    tools: ['test_run', 'run_node_smoke', 'explain_failure'],
  },
];

const SectionLabel: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <Typography.Text
    strong
    style={{ display: 'block', marginBottom: 6, fontSize: 13 }}
  >
    {children}
  </Typography.Text>
);

const CodeBlock: React.FC<{ value: string }> = ({ value }) => (
  <div
    style={{
      position: 'relative',
      borderRadius: 8,
      border: '1px solid rgba(82,100,154,.16)',
      background: 'rgba(15,21,40,.04)',
    }}
  >
    <div style={{ position: 'absolute', top: 4, right: 4, zIndex: 1 }}>
      <CopyButton value={value} />
    </div>
    <pre
      style={{
        margin: 0,
        padding: '12px 40px 12px 12px',
        fontSize: 12,
        lineHeight: '18px',
        fontFamily:
          'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-all',
        color: 'rgba(15,21,40,.86)',
      }}
    >
      {value}
    </pre>
  </div>
);

export const ExternalMCPGuideModal: React.FC<{
  visible: boolean;
  onClose: () => void;
}> = ({ visible, onClose }) => {
  const endpoint = useMemo(
    () => `${window.location.origin}${WORKFLOW_MCP_PATH}`,
    [],
  );

  const claudeConfig = useMemo(
    () =>
      JSON.stringify(
        {
          mcpServers: {
            'ynet-workflow': {
              type: 'http',
              url: endpoint,
              headers: { Authorization: 'Bearer <你的API Key>' },
            },
          },
        },
        null,
        2,
      ),
    [endpoint],
  );

  return (
    <Modal
      title="接入外部 AI(MCP)"
      closable
      visible={visible}
      width={640}
      onCancel={onClose}
      footer={<Button onClick={onClose}>知道了</Button>}
      zIndex={2000}
    >
      <div style={{ maxHeight: '60vh', overflowY: 'auto', paddingRight: 4 }}>
        <Typography.Paragraph
          type="secondary"
          style={{ marginBottom: 16, fontSize: 13 }}
        >
          本平台内置了工作流 MCP 服务。外部本地 MCP 客户端(如 Claude Code /
          Codex)可连接它,直接用画布工具(共 20
          个)来搭建、修改和调试当前工作流。
        </Typography.Paragraph>

        <div style={{ marginBottom: 16 }}>
          <SectionLabel>MCP 端点</SectionLabel>
          <Typography.Paragraph
            type="tertiary"
            style={{ margin: '0 0 4px', fontSize: 12 }}
          >
            streamable-HTTP 标准 MCP 协议
          </Typography.Paragraph>
          <Typography.Text
            copyable={{ content: endpoint }}
            style={{
              display: 'inline-block',
              padding: '4px 8px',
              borderRadius: 6,
              background: 'rgba(15,21,40,.04)',
              fontSize: 12,
              fontFamily:
                'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
              wordBreak: 'break-all',
            }}
          >
            {endpoint}
          </Typography.Text>
        </div>

        <div style={{ marginBottom: 16 }}>
          <SectionLabel>鉴权</SectionLabel>
          <Typography.Paragraph style={{ margin: 0, fontSize: 13 }}>
            在 HTTP 请求头携带{' '}
            <Typography.Text code>
              Authorization: Bearer &lt;你的 API Key&gt;
            </Typography.Text>
            。API Key 可在平台「API 管理 / 开发者」页生成。
          </Typography.Paragraph>
        </div>

        <div style={{ marginBottom: 16 }}>
          <SectionLabel>Claude Code 配置示例</SectionLabel>
          <Typography.Paragraph
            type="tertiary"
            style={{ margin: '0 0 6px', fontSize: 12 }}
          >
            写入 <Typography.Text code>~/.claude.json</Typography.Text> 或{' '}
            <Typography.Text code>.mcp.json</Typography.Text> 的{' '}
            <Typography.Text code>mcpServers</Typography.Text> 字段:
          </Typography.Paragraph>
          <CodeBlock value={claudeConfig} />
        </div>

        <div style={{ marginBottom: 16 }}>
          <SectionLabel>20 个画布工具</SectionLabel>
          {TOOL_GROUPS.map(group => (
            <div key={group.title} style={{ marginBottom: 10 }}>
              <Typography.Text style={{ fontSize: 13 }} strong>
                {group.title}
              </Typography.Text>
              <Typography.Text
                type="tertiary"
                style={{ marginLeft: 6, fontSize: 12 }}
              >
                {group.desc}
              </Typography.Text>
              <div
                style={{
                  display: 'flex',
                  flexWrap: 'wrap',
                  gap: 6,
                  marginTop: 6,
                }}
              >
                {group.tools.map(tool => (
                  <span
                    key={tool}
                    style={{
                      padding: '2px 8px',
                      borderRadius: 6,
                      background: 'rgba(47,107,255,.08)',
                      color: '#2F6BFF',
                      fontSize: 12,
                      fontFamily:
                        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                    }}
                  >
                    {tool}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div
          style={{
            padding: '10px 12px',
            borderRadius: 8,
            background: 'rgba(255,176,32,.1)',
            border: '1px solid rgba(255,176,32,.28)',
          }}
        >
          <Typography.Text strong style={{ fontSize: 13 }}>
            重要提示
          </Typography.Text>
          <Typography.Paragraph
            style={{ margin: '4px 0 0', fontSize: 12, lineHeight: '18px' }}
          >
            改画布类工具(add_node
            等)需要你在本编辑器中打开对应工作流——命令经本页面实时执行;查询类与
            test_run 类工具可独立使用。
          </Typography.Paragraph>
        </div>
      </div>
    </Modal>
  );
};
