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

import React from 'react';

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import ReviewQueue from '../review-queue';

const request = vi.fn();
const toastSuccess = vi.fn();
const toastError = vi.fn();

vi.mock('@coze-arch/bot-api', () => ({
  axiosInstance: { request: (...args: unknown[]) => request(...args) },
}));

vi.mock('@coze-arch/coze-design/icons', () => {
  const Icon = () => <span data-testid="icon" />;
  return { IconCozPlugin: Icon, IconCozStore: Icon };
});

vi.mock('@coze-arch/coze-design', () => ({
  Spin: () => <div data-testid="spin" />,
  TextArea: ({
    value,
    onChange,
  }: {
    value?: string;
    onChange?: (value: string) => void;
  }) => (
    <textarea
      value={value || ''}
      onChange={event => onChange?.(event.target.value)}
    />
  ),
  Modal: ({
    children,
    visible,
    onOk,
  }: {
    children?: React.ReactNode;
    visible?: boolean;
    onOk?: () => void;
  }) =>
    visible ? (
      <div>
        {children}
        <button type="button" onClick={onOk}>
          确认驳回
        </button>
      </div>
    ) : null,
  Toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}));

describe('ReviewQueue', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders pending skills and approves one', async () => {
    request.mockImplementation(({ url }: { url: string }) => {
      if (url === '/api/skill/review/pending') {
        return Promise.resolve({
          code: 0,
          data: {
            skill_list: [
              { skill_id: 's1', name: '待审技能A', description: '描述A' },
            ],
            total: 1,
          },
        });
      }
      return Promise.resolve({ code: 0 });
    });

    render(<ReviewQueue spaceId="space-1" />);

    expect(await screen.findByText('待审技能A')).toBeInTheDocument();
    await waitFor(() => {
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/skill/review/pending',
          params: { space_id: 'space-1', page: 1, page_size: 20 },
        }),
      );
    });

    fireEvent.click(screen.getByRole('button', { name: '通过' }));

    await waitFor(() => {
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/skill/review',
          data: { skill_id: 's1', approve: true, note: '' },
        }),
      );
    });
    await waitFor(() =>
      expect(screen.getByText('暂无待审核技能')).toBeInTheDocument(),
    );
    expect(toastSuccess).toHaveBeenCalled();
  });

  it('shows permission error when backend rejects', async () => {
    request.mockResolvedValue({ code: 403, msg: '仅空间管理员可审核' });
    render(<ReviewQueue spaceId="space-1" />);
    expect(
      await screen.findByText('仅空间管理员可审核'),
    ).toBeInTheDocument();
  });

  it('shows empty state without a space', async () => {
    render(<ReviewQueue spaceId="" />);
    expect(
      await screen.findByText('请先进入一个工作空间再审核技能'),
    ).toBeInTheDocument();
    expect(request).not.toHaveBeenCalled();
  });
});
