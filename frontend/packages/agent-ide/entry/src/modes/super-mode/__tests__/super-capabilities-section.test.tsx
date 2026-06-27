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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { axiosInstance } from '@coze-arch/bot-api';

import { SuperCapabilitiesSection } from '../super-capabilities-section';

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: (selector: (state: unknown) => unknown) =>
    selector({ botId: 'bot-1', space_id: 'space-1' }),
}));

vi.mock('@coze-arch/bot-api', () => ({
  axiosInstance: {
    request: vi.fn(),
  },
}));

vi.mock('@coze-arch/coze-design', () => ({
  Switch: ({ checked, disabled, onChange }: any) => (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange?.(!checked)}
    />
  ),
  Modal: ({ visible, children, onOk, onCancel }: any) =>
    visible ? (
      <div data-testid="mcp-modal">
        {children}
        <button type="button" onClick={onOk}>
          modal-ok
        </button>
        <button type="button" onClick={onCancel}>
          modal-cancel
        </button>
      </div>
    ) : null,
  Input: ({ value, placeholder, onChange }: any) => (
    <input
      placeholder={placeholder}
      value={value}
      onChange={e => onChange?.(e.target.value)}
    />
  ),
  Select: ({ value, optionList, onChange }: any) => (
    <select value={value} onChange={e => onChange?.(e.target.value)}>
      {optionList?.map((o: any) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  ),
  Toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

describe('SuperCapabilitiesSection', () => {
  const request = axiosInstance.request as unknown as ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('defaults all switches on when config is null', async () => {
    request.mockResolvedValueOnce({ code: 0, data: { config: null } });
    render(<SuperCapabilitiesSection />);

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/super-agent/config/get',
          data: { bot_id: 'bot-1', space_id: 'space-1' },
        }),
      ),
    );

    await waitFor(() => {
      const switches = screen.getAllByRole('switch');
      expect(switches).toHaveLength(6);
      switches.forEach(sw =>
        expect(sw.getAttribute('aria-checked')).toBe('true'),
      );
    });
  });

  it('disables sandbox-dependent switches and shows hint when sandbox off', async () => {
    request.mockResolvedValueOnce({
      code: 0,
      data: { config: { sandbox: false } },
    });
    render(<SuperCapabilitiesSection />);

    await waitFor(() =>
      expect(screen.getByText('纯 MCP 模式：仅 MCP 工具与对话可用')).toBeTruthy(),
    );
    // web_search / web_fetch / run_bash disabled; sandbox/deep_task/skill_manage enabled
    const switches = screen.getAllByRole('switch');
    const disabledCount = switches.filter(s =>
      s.hasAttribute('disabled'),
    ).length;
    expect(disabledCount).toBe(3);
  });

  it('saves the full config on switch change', async () => {
    request.mockResolvedValueOnce({ code: 0, data: { config: null } });
    render(<SuperCapabilitiesSection />);

    await waitFor(() => expect(screen.getAllByRole('switch')).toHaveLength(6));

    request.mockResolvedValueOnce({
      code: 0,
      data: { config: { deep_task: false } },
    });
    // deep_task is the 5th switch
    fireEvent.click(screen.getAllByRole('switch')[4]);

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/super-agent/config/update',
          data: expect.objectContaining({
            bot_id: 'bot-1',
            space_id: 'space-1',
            config: expect.objectContaining({
              sandbox: true,
              web_search: true,
              web_fetch: true,
              run_bash: true,
              deep_task: false,
              skill_manage: true,
            }),
          }),
        }),
      ),
    );
  });

  it('shows empty placeholder when no mcp servers', async () => {
    request.mockResolvedValueOnce({ code: 0, data: { config: null } });
    render(<SuperCapabilitiesSection />);

    await waitFor(() =>
      expect(
        screen.getByText('还没有 MCP 服务，点击添加远程 MCP server'),
      ).toBeTruthy(),
    );
  });

  it('adds an mcp server and saves full config with mcp_servers', async () => {
    request.mockResolvedValueOnce({ code: 0, data: { config: null } });
    render(<SuperCapabilitiesSection />);

    await waitFor(() => expect(screen.getAllByRole('switch')).toHaveLength(6));

    fireEvent.click(screen.getByText('添加 MCP 服务'));
    fireEvent.change(screen.getByPlaceholderText('https:// 远程 MCP server 地址'), {
      target: { value: 'https://mcp.example.com/sse' },
    });
    fireEvent.change(
      screen.getByPlaceholderText('便于识别的名称，留空则用 URL'),
      { target: { value: 'demo' } },
    );

    request.mockResolvedValueOnce({ code: 0, data: { config: null } });
    fireEvent.click(screen.getByText('modal-ok'));

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/super-agent/config/update',
          data: expect.objectContaining({
            config: expect.objectContaining({
              sandbox: true,
              skill_manage: true,
              mcp_servers: [
                {
                  name: 'demo',
                  type: 'streamable_http',
                  url: 'https://mcp.example.com/sse',
                  env: {},
                  enabled: true,
                },
              ],
            }),
          }),
        }),
      ),
    );
  });

  it('removes an mcp server and saves remaining list', async () => {
    request.mockResolvedValueOnce({
      code: 0,
      data: {
        config: {
          mcp_servers: [
            { name: 'a', type: 'sse', url: 'https://a.example.com' },
          ],
        },
      },
    });
    render(<SuperCapabilitiesSection />);

    await waitFor(() => expect(screen.getByText('a')).toBeTruthy());

    request.mockResolvedValueOnce({ code: 0, data: { config: null } });
    fireEvent.click(screen.getByLabelText('删除'));

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/super-agent/config/update',
          data: expect.objectContaining({
            config: expect.objectContaining({ mcp_servers: [] }),
          }),
        }),
      ),
    );
  });
});
