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

import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  fireEvent,
  render,
  screen,
  waitFor,
  act,
} from '@testing-library/react';
import { axiosInstance } from '@coze-arch/bot-api';

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: (selector: (state: unknown) => unknown) =>
    selector({ botId: 'bot-1', space_id: 'space-1' }),
}));

vi.mock('@coze-arch/bot-api', () => ({
  axiosInstance: {
    request: vi.fn(),
  },
}));

const makeInput = ({
  value,
  placeholder,
  onChange,
  disabled,
  'data-testid': testId,
}: any) => (
  <input
    data-testid={testId}
    placeholder={placeholder}
    value={value}
    disabled={disabled}
    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
      onChange?.(e.target.value)
    }
  />
);

const makeModal = ({
  visible,
  title,
  children,
  onOk,
  onCancel,
  okText,
  cancelText,
  okButtonProps,
  cancelButtonProps,
}: any) =>
  visible ? (
    <div data-testid="publish-modal">
      <div data-testid="modal-title">{title}</div>
      {children}
      <button
        type="button"
        data-testid="modal-ok"
        disabled={okButtonProps?.disabled}
        onClick={onOk}
      >
        {okText}
      </button>
      <button
        type="button"
        data-testid="modal-cancel"
        disabled={cancelButtonProps?.disabled}
        onClick={onCancel}
      >
        {cancelText}
      </button>
    </div>
  ) : null;

/* eslint-disable @typescript-eslint/naming-convention */
vi.mock('@coze-arch/coze-design', () => ({
  Input: (props: any) => makeInput(props),
  Modal: (props: any) => makeModal(props),
  Toast: {
    error: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
  },
}));
/* eslint-enable @typescript-eslint/naming-convention */

import {
  PublishVirtualEmployeeModal,
  agentAppApi,
} from '../publish-virtual-employee-modal';

const request = axiosInstance.request as unknown as ReturnType<typeof vi.fn>;

describe('PublishVirtualEmployeeModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders name and version fields', () => {
    render(<PublishVirtualEmployeeModal visible onClose={vi.fn()} />);

    expect(screen.getByTestId('publish-modal')).toBeTruthy();
    expect(screen.getByTestId('modal-title').textContent).toBe(
      '发布为虚拟员工',
    );
    expect(screen.getByTestId('name-input')).toBeTruthy();
    expect(screen.getByTestId('version-input')).toBeTruthy();
  });

  it('does not render when not visible', () => {
    render(<PublishVirtualEmployeeModal visible={false} onClose={vi.fn()} />);
    expect(screen.queryByTestId('publish-modal')).toBeNull();
  });

  it('calls publish api on submit with correct params', async () => {
    request.mockResolvedValueOnce({
      code: 0,
      data: { product_id: 'prod-1', version: 'v1.0.0' },
    });
    request.mockImplementation(() => new Promise(() => {})); // poll never resolves

    render(<PublishVirtualEmployeeModal visible onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('name-input'), {
      target: { value: '销售助手' },
    });

    await act(() => {
      fireEvent.click(screen.getByTestId('modal-ok'));
    });

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/super-agent/agent-app/publish',
          data: expect.objectContaining({
            agent_id: 'bot-1',
            space_id: 'space-1',
            name: '销售助手',
            version: 'v1.0.0',
          }),
        }),
      ),
    );
  });

  it('shows 构建中 status label after publish succeeds', async () => {
    request.mockResolvedValueOnce({
      code: 0,
      data: { product_id: 'prod-1', version: 'v1.0.0' },
    });
    request.mockImplementation(() => new Promise(() => {})); // poll never resolves

    render(<PublishVirtualEmployeeModal visible onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('name-input'), {
      target: { value: '销售助手' },
    });

    await act(() => {
      fireEvent.click(screen.getByTestId('modal-ok'));
    });

    await waitFor(() =>
      expect(screen.getByTestId('build-status').textContent).toBe('构建中'),
    );
  });

  it('shows 已就绪 when buildStatus returns ready', async () => {
    let resolveStatus!: (v: unknown) => void;
    const statusPromise = new Promise(r => {
      resolveStatus = r;
    });

    request.mockResolvedValueOnce({
      code: 0,
      data: { product_id: 'prod-1', version: 'v1.0.0' },
    });
    request.mockReturnValueOnce(statusPromise);

    render(<PublishVirtualEmployeeModal visible onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('name-input'), {
      target: { value: '销售助手' },
    });

    await act(() => {
      fireEvent.click(screen.getByTestId('modal-ok'));
    });

    await waitFor(() =>
      expect(screen.getByTestId('build-status').textContent).toBe('构建中'),
    );

    await act(() => {
      resolveStatus({
        code: 0,
        data: {
          product_id: 'prod-1',
          version: 'v1.0.0',
          build_status: 'ready',
        },
      });
      return new Promise(r => setTimeout(r, 2500));
    });

    await waitFor(() =>
      expect(screen.getByTestId('build-status').textContent).toBe('已就绪'),
    );
  });

  it('shows 失败 when buildStatus returns failed', async () => {
    let resolveStatus!: (v: unknown) => void;
    const statusPromise = new Promise(r => {
      resolveStatus = r;
    });

    request.mockResolvedValueOnce({
      code: 0,
      data: { product_id: 'prod-2', version: 'v1.0.0' },
    });
    request.mockReturnValueOnce(statusPromise);

    render(<PublishVirtualEmployeeModal visible onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('name-input'), {
      target: { value: '失败测试' },
    });

    await act(() => {
      fireEvent.click(screen.getByTestId('modal-ok'));
    });

    await waitFor(() =>
      expect(screen.getByTestId('build-status').textContent).toBe('构建中'),
    );

    await act(() => {
      resolveStatus({
        code: 0,
        data: {
          product_id: 'prod-2',
          version: 'v1.0.0',
          build_status: 'failed',
        },
      });
      return new Promise(r => setTimeout(r, 2500));
    });

    await waitFor(() =>
      expect(screen.getByTestId('build-status').textContent).toBe('失败'),
    );
  });

  it('calls onClose when cancel is clicked', () => {
    const onClose = vi.fn();
    render(<PublishVirtualEmployeeModal visible onClose={onClose} />);
    fireEvent.click(screen.getByTestId('modal-cancel'));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('agentAppApi.publish posts to the correct endpoint', async () => {
    request.mockResolvedValueOnce({
      code: 0,
      data: { product_id: 'p1', version: 'v2.0.0' },
    });

    const result = await agentAppApi.publish({
      bot_id: 'bot-42',
      space_id: 'sp-1',
      name: 'Test',
      version: 'v2.0.0',
    });

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/api/super-agent/agent-app/publish',
        data: {
          agent_id: 'bot-42',
          space_id: 'sp-1',
          name: 'Test',
          version: 'v2.0.0',
        },
      }),
    );
    expect(result.data.product_id).toBe('p1');
  });

  it('agentAppApi.buildStatus posts to the correct endpoint', async () => {
    request.mockResolvedValueOnce({
      code: 0,
      data: { product_id: 'p1', version: 'v1.0.0', build_status: 'ready' },
    });

    const result = await agentAppApi.buildStatus({
      product_id: 'p1',
      space_id: 'sp-1',
      version: 'v1.0.0',
    });

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        url: '/api/super-agent/agent-app/build-status',
        data: { product_id: 'p1', space_id: 'sp-1', version: 'v1.0.0' },
      }),
    );
    expect(result.data.build_status).toBe('ready');
  });
});
