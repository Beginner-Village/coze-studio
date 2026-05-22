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

import { vi, describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import '@testing-library/jest-dom';

vi.mock('@coze-arch/i18n', () => ({
  I18n: {
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  },
}));

vi.mock('@coze-arch/coze-design', () => ({
  // eslint-disable-next-line @typescript-eslint/naming-convention
  Modal: ({ visible, title, children, footer }: any) =>
    visible ? (
      <div data-testid="modal">
        <div data-testid="modal-title">{title}</div>
        <div data-testid="modal-body">{children}</div>
        <div data-testid="modal-footer">{footer}</div>
      </div>
    ) : null,
  // eslint-disable-next-line @typescript-eslint/naming-convention
  Button: ({ children, onClick, disabled, loading, type, ...rest }: any) => (
    <button
      onClick={onClick}
      disabled={disabled}
      data-loading={loading ? 'true' : 'false'}
      data-type={type}
      {...rest}
    >
      {children}
    </button>
  ),
}));

import { MergeSliceConfirmModal } from '../index';

describe('MergeSliceConfirmModal', () => {
  const slices = [
    { slice_id: '1', sequence: 1, content: 'First chunk' },
    { slice_id: '2', sequence: 2, content: 'Second chunk' },
  ];

  it('renders preview with concatenated content', () => {
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByText(/First chunk/)).toBeInTheDocument();
    expect(screen.getByText(/Second chunk/)).toBeInTheDocument();
  });

  it('calls onConfirm when confirm clicked', () => {
    const onConfirm = vi.fn();
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByTestId('merge-confirm-btn'));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('calls onCancel when cancel clicked', () => {
    const onCancel = vi.fn();
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />,
    );
    fireEvent.click(screen.getByTestId('merge-cancel-btn'));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('shows loading state and disables confirm when loading', () => {
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        loading
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    const btn = screen.getByTestId('merge-confirm-btn');
    expect(btn).toBeDisabled();
  });
});
