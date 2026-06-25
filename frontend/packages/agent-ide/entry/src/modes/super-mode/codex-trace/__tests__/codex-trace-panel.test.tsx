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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import { useTraceStore } from '../trace-store';
import { CodexTracePanel } from '../codex-trace-panel';

const { lazyCozeMdBoxExport, markdownSpy, sandboxFilenameRegExp } = vi.hoisted(
  () => {
    const sandboxFilenamePattern =
      '(^|[\\s,，、])' +
      '([A-Za-z0-9_-]+\\.(?:py|js|ts|tsx|jsx|json|md|txt|csv|xlsx|xls|' +
      'docx|doc|pptx|ppt|pdf|sh|go|yaml|yml|toml|lock))(?![A-Za-z0-9_/.-])';

    return {
      lazyCozeMdBoxExport: 'LazyCozeMdBox',
      markdownSpy: vi.fn(),
      sandboxFilenameRegExp: new RegExp(sandboxFilenamePattern, 'g'),
    };
  },
);

vi.mock('@coze-arch/coze-design', () => ({
  Toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@coze-common/chat-uikit', () => ({
  [lazyCozeMdBoxExport]: ({ markDown }: { markDown: string }) => {
    markdownSpy(markDown);

    return <div>{markDown}</div>;
  },
  protectSandboxFilenames: (text: string) =>
    text.replace(sandboxFilenameRegExp, '$1`$2`'),
}));

describe('CodexTracePanel', () => {
  beforeEach(() => {
    markdownSpy.mockClear();
    HTMLElement.prototype.scrollIntoView = vi.fn();
    useTraceStore.getState().setPendingReply(false);
  });

  it('renders the empty state with a real icon instead of an emoji', () => {
    useTraceStore.getState().setMessages([]);

    const { container } = render(<CodexTracePanel />);

    expect(screen.getByText('FinMallClaw 执行台')).toBeTruthy();
    expect(container.querySelector('svg')).toBeTruthy();
    expect(container.textContent).not.toContain('⚡');
  });

  it('protects sandbox filenames before rendering assistant markdown', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'assistant-1',
        role: 'assistant',
        type: 'answer',
        content:
          'Created create_excel.py, package-lock.json and https://example.com/hello.txt',
        is_finish: true,
        extra_info: {},
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: 'write files',
        extra_info: {},
      },
    ]);

    render(<CodexTracePanel />);

    expect(markdownSpy).toHaveBeenCalledWith(
      'Created `create_excel.py`, `package-lock.json` and https://example.com/hello.txt',
    );
  });

  it('does not render empty placeholder tool events as timeline dots', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'assistant-empty-answer',
        role: 'assistant',
        type: 'answer',
        content: '   ',
        is_finish: false,
        extra_info: {},
      },
      {
        message_id: 'assistant-empty-tool',
        role: 'assistant',
        type: 'function_call',
        content: '',
        extra_info: {
          call_id: 'empty-tool',
          plugin_request: '{}',
        },
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '介绍技能',
        extra_info: {},
      },
    ]);

    const { container } = render(<CodexTracePanel />);

    expect(screen.getByText('介绍技能')).toBeTruthy();
    expect(container.querySelector('span[data-status]')).toBeNull();
    expect(container.querySelector('[class*="timeline"]')).toBeNull();
    expect(container.textContent).not.toContain('tool');
  });

  it('renders common tool names in the design-system tool card', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'assistant-search',
        role: 'assistant',
        type: 'function_call',
        content: '',
        extra_info: {
          call_id: 'search-tool',
          tool_name: 'web_search',
          plugin_request: JSON.stringify({ query: 'glm5.2' }),
        },
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '搜索模型',
        extra_info: {},
      },
    ]);

    render(<CodexTracePanel />);

    expect(screen.getByText(/已调用/)).toBeTruthy();
    expect(screen.getByText('web_search')).toBeTruthy();
    expect(screen.queryByText(/运行过程/)).toBeNull();
    expect(screen.queryByText('Search')).toBeNull();
  });

  it('renders tool calls inline with surrounding assistant text', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'u-1',
        role: 'user',
        type: 'question',
        content: '重构当前工作流',
        created_at: 1000,
        extra_info: {},
      },
      {
        message_id: 'a-1',
        role: 'assistant',
        type: 'answer',
        content: '我先清空旧画布。',
        created_at: 2000,
        is_finish: false,
        extra_info: {},
      },
      {
        message_id: 'fc-1',
        role: 'assistant',
        type: 'function_call',
        content: '',
        created_at: 3000,
        extra_info: {
          call_id: 'clear-canvas',
          tool_name: 'workflow_canvas_clear_canvas',
          plugin_request: '{}',
        },
      },
      {
        message_id: 'tr-1',
        role: 'assistant',
        type: 'tool_response',
        content: '{"status":"dispatched_to_canvas","op":"clear_canvas"}',
        created_at: 4000,
        extra_info: {
          call_id: 'clear-canvas',
          plugin_status: '0',
        },
      },
      {
        message_id: 'a-2',
        role: 'assistant',
        type: 'answer',
        content: '然后重新添加节点。',
        created_at: 5000,
        is_finish: true,
        extra_info: {},
      },
    ]);

    const { container } = render(<CodexTracePanel />);
    const text = container.textContent ?? '';
    const firstTextIndex = text.indexOf('我先清空旧画布。');
    const toolIndex = text.indexOf('清空画布');
    const secondTextIndex = text.indexOf('然后重新添加节点。');

    expect(firstTextIndex).toBeGreaterThanOrEqual(0);
    expect(toolIndex).toBeGreaterThan(firstTextIndex);
    expect(secondTextIndex).toBeGreaterThan(toolIndex);
    expect(screen.queryByText(/运行过程/)).toBeNull();
  });

  it('renders workflow canvas tool calls as readable execution steps', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'u-1',
        role: 'user',
        type: 'question',
        content: '加一个大模型节点',
        extra_info: {},
      },
      {
        message_id: 'fc-1',
        role: 'assistant',
        type: 'function_call',
        content: '',
        extra_info: {
          call_id: 'add-llm',
          tool_name: 'workflow_canvas_add_node',
          plugin_request: JSON.stringify({
            node_tag: 'llm',
            type: '3',
            title: '大模型处理',
          }),
        },
      },
      {
        message_id: 'tr-1',
        role: 'assistant',
        type: 'tool_response',
        content: JSON.stringify({
          status: 'dispatched_to_canvas',
          op: 'add_node',
          args: {
            node_tag: 'llm',
            type: '3',
            title: '大模型处理',
          },
        }),
        extra_info: {
          call_id: 'add-llm',
          plugin_status: '0',
        },
      },
    ]);

    render(<CodexTracePanel />);

    expect(screen.getByText('添加节点')).toBeTruthy();
    expect(screen.getByText('大模型处理')).toBeTruthy();
    expect(screen.queryByText('workflow_canvas_add_node')).toBeNull();
  });

  it('shows an assistant typing placeholder immediately while waiting', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '生成一份计划',
        extra_info: {},
      },
    ]);
    useTraceStore.getState().setPendingReply(true);

    const { container } = render(<CodexTracePanel />);

    expect(screen.getByText('生成一份计划')).toBeTruthy();
    expect(screen.getByLabelText('正在生成回复')).toBeTruthy();
    expect(container.querySelectorAll('[aria-label="正在生成回复"] span')).toHaveLength(
      3,
    );
  });

  it('shows an assistant typing placeholder even before the sent message is mirrored', () => {
    useTraceStore.getState().setMessages([]);
    useTraceStore.getState().setPendingReply(true);

    const { container } = render(<CodexTracePanel />);

    expect(screen.queryByText('FinMallClaw 执行台')).toBeNull();
    expect(screen.getByLabelText('正在生成回复')).toBeTruthy();
    expect(container.querySelectorAll('[aria-label="正在生成回复"] span')).toHaveLength(
      3,
    );
  });

  it('hides the typing placeholder after answer text starts streaming', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'assistant-1',
        role: 'assistant',
        type: 'answer',
        content: '已经开始生成',
        is_finish: false,
        extra_info: {},
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '生成一份计划',
        extra_info: {},
      },
    ]);
    useTraceStore.getState().setPendingReply(true);

    render(<CodexTracePanel />);

    expect(screen.getByText('已经开始生成')).toBeTruthy();
    expect(screen.queryByLabelText('正在生成回复')).toBeNull();
  });

  it('keeps tool payloads out of the compact trace card', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'tool-response',
        role: 'assistant',
        type: 'tool_response',
        content: JSON.stringify({ ok: true, title: 'GLM-5.2' }),
        extra_info: {
          call_id: 'fetch-tool',
          plugin_status: '0',
        },
      },
      {
        message_id: 'assistant-fetch',
        role: 'assistant',
        type: 'function_call',
        content: '',
        extra_info: {
          call_id: 'fetch-tool',
          tool_name: 'web_fetch',
          plugin_request: JSON.stringify({
            url: 'https://example.com/glm',
            timeout_ms: 10000,
          }),
        },
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '抓取资料',
        extra_info: {},
      },
    ]);

    render(<CodexTracePanel />);

    expect(screen.getByText('web_fetch')).toBeTruthy();
    expect(screen.queryByText('参数')).toBeNull();
    expect(screen.queryByText('返回')).toBeNull();
    expect(screen.queryByText(/GLM-5\.2/)).toBeNull();
  });

  it('reads tool arguments from function_call content when metadata omits plugin_request', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'tool-response',
        role: 'assistant',
        type: 'tool_response',
        content: 'hello.txt',
        extra_info: {
          call_id: 'list-tool',
          plugin_status: '0',
        },
      },
      {
        message_id: 'assistant-list',
        role: 'assistant',
        type: 'function_call',
        content: JSON.stringify({
          function: {
            name: 'list_files',
            arguments: JSON.stringify({ path: '/workspace', depth: 1 }),
          },
        }),
        extra_info: {
          call_id: 'list-tool',
          tool_name: 'list_files',
        },
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '列出文件',
        extra_info: {},
      },
    ]);

    const { container } = render(<CodexTracePanel />);

    expect(screen.getByText('list_files')).toBeTruthy();
    expect(container.querySelector('[title="/workspace"]')).toBeTruthy();
  });

  it('expands a tool call to reveal its arguments and result', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'tr-1',
        role: 'assistant',
        type: 'tool_response',
        content: 'hi-there-output',
        extra_info: { call_id: 'bash-1', plugin_status: '0' },
      },
      {
        message_id: 'fc-1',
        role: 'assistant',
        type: 'function_call',
        content: JSON.stringify({
          function: {
            name: 'run_bash',
            arguments: JSON.stringify({ cmd: 'echo hi' }),
          },
        }),
        extra_info: { call_id: 'bash-1', tool_name: 'run_bash' },
      },
      {
        message_id: 'u-1',
        role: 'user',
        type: 'question',
        content: '跑一下',
        extra_info: {},
      },
    ]);

    const { container } = render(<CodexTracePanel />);
    // collapsed: detail not shown yet
    expect(screen.queryByText('入参')).toBeNull();
    // click the tool row to expand
    fireEvent.click(screen.getByText('run_bash').closest('[data-expandable]'));
    // now arguments + result blocks are visible with real content
    expect(screen.getByText('入参')).toBeTruthy();
    expect(screen.getByText('结果')).toBeTruthy();
    expect(container.textContent).toContain('echo hi');
    expect(container.textContent).toContain('hi-there-output');
  });

  it('renders context compaction as a harness event card', () => {
    useTraceStore.getState().setMessages([
      {
        message_id: 'context-compact',
        role: 'assistant',
        type: 'context_compacted',
        content:
          'Older conversation context was summarized for the next model call.',
        created_at: 3000,
        extra_info: {
          event: 'context.compacted',
          trigger: 'history_bytes_exceeded',
          original_messages: '42',
          compacted_messages: '26',
          retained_messages: '16',
          original_bytes: '262144',
          max_bytes: '163840',
          summary_path: '/workspace/.agent/context-summary.json',
        },
      },
      {
        message_id: 'user-1',
        role: 'user',
        type: 'question',
        content: '继续做 harness',
        created_at: 1000,
        extra_info: {},
      },
    ]);

    render(<CodexTracePanel />);

    expect(screen.getByText('上下文已自动压缩')).toBeTruthy();
    expect(screen.getByText('42 条历史消息')).toBeTruthy();
    expect(screen.getByText('压缩 26 条')).toBeTruthy();
    expect(screen.getByText('保留 16 条')).toBeTruthy();
    expect(screen.getByText('/workspace/.agent/context-summary.json')).toBeTruthy();
    expect(screen.queryByText('运行过程 · 1 个工具')).toBeNull();
  });
});
