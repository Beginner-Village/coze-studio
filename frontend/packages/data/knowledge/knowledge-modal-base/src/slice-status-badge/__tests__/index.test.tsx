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
  Spin: ({ size }: any) => <span data-testid="spin" data-size={size} />,
  // eslint-disable-next-line @typescript-eslint/naming-convention
  Button: ({ children, onClick, disabled, size, ...rest }: any) => (
    <button onClick={onClick} disabled={disabled} data-size={size} {...rest}>
      {children}
    </button>
  ),
  // eslint-disable-next-line @typescript-eslint/naming-convention
  Tooltip: ({ children, content }: any) => (
    <span data-testid="tooltip" data-content={String(content ?? '')}>
      {children}
    </span>
  ),
}));

import { SliceStatusBadge } from '../index';

describe('SliceStatusBadge', () => {
  it('renders nothing when status is Done', () => {
    const { container } = render(<SliceStatusBadge status="Done" />);
    expect(container.firstChild).toBeNull();
  });

  it('renders spinner when status is Init or Processing', () => {
    const { rerender } = render(<SliceStatusBadge status="Init" />);
    expect(screen.getByTestId('slice-status-spinner')).toBeInTheDocument();
    expect(
      screen.getByText('knowledge_slice_status_reindexing'),
    ).toBeInTheDocument();
    rerender(<SliceStatusBadge status="Processing" />);
    expect(screen.getByTestId('slice-status-spinner')).toBeInTheDocument();
    expect(
      screen.getByText('knowledge_slice_status_reindexing'),
    ).toBeInTheDocument();
  });

  it('renders retry button + reason when status is Failed', () => {
    const onRetry = vi.fn();
    render(
      <SliceStatusBadge
        status="Failed"
        reason="embedding timeout"
        onRetry={onRetry}
      />,
    );
    expect(screen.getByTestId('slice-status-failed')).toBeInTheDocument();
    expect(screen.getByText(/embedding timeout/)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('slice-status-retry-btn'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders timeout state when status is Timeout', () => {
    render(<SliceStatusBadge status="Timeout" />);
    expect(screen.getByTestId('slice-status-timeout')).toBeInTheDocument();
    expect(
      screen.getByText('knowledge_slice_status_timeout'),
    ).toBeInTheDocument();
  });
});
