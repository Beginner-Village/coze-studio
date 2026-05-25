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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { SpaceApi } from '@coze-arch/bot-api';

import { DataMaintenanceSection } from '../DataMaintenanceSection';

vi.mock('@coze-arch/bot-api', () => ({
  SpaceApi: { resyncES: vi.fn() },
}));

vi.mock('@coze-arch/i18n', () => ({
  I18n: {
    t: vi.fn(
      (_key: string, _opts?: unknown, fallback?: string) => fallback ?? _key,
    ),
  },
}));

vi.mock('@coze-arch/coze-design', () => ({
  // eslint-disable-next-line @typescript-eslint/naming-convention -- mock a React component re-exported from third-party lib
  Button: ({ children, onClick, loading, color, ...rest }: any) => (
    <button
      onClick={onClick}
      disabled={loading}
      data-color={color}
      data-testid={rest['data-testid']}
    >
      {children}
    </button>
  ),
  // eslint-disable-next-line @typescript-eslint/naming-convention -- mock a React component re-exported from third-party lib
  Modal: ({
    visible,
    title,
    children,
    onOk,
    onCancel,
    okText,
    cancelText,
    confirmLoading,
  }: any) =>
    visible ? (
      <div role="dialog" data-testid="resync-modal">
        <div data-testid="modal-title">{title}</div>
        <div data-testid="modal-body">{children}</div>
        <button onClick={onOk} disabled={confirmLoading} data-testid="modal-ok">
          {okText || 'OK'}
        </button>
        <button onClick={onCancel} data-testid="modal-cancel">
          {cancelText || 'Cancel'}
        </button>
      </div>
    ) : null,
  Toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

describe('DataMaintenanceSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders title + resync button', () => {
    render(<DataMaintenanceSection spaceId="100" />);
    expect(screen.getByText(/数据维护|Data Maintenance/i)).toBeInTheDocument();
    expect(screen.getByTestId('space-resync-es-button')).toBeInTheDocument();
  });

  it('opens confirm modal on button click', () => {
    render(<DataMaintenanceSection spaceId="100" />);
    expect(screen.queryByTestId('resync-modal')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    expect(screen.getByTestId('resync-modal')).toBeInTheDocument();
  });

  it('calls SpaceApi.resyncES with the right spaceId on confirm', async () => {
    (SpaceApi.resyncES as any).mockResolvedValue({
      code: 0,
      msg: 'success',
      counts: {
        project_draft: 4,
        coze_resource: 70,
        kb_entries: 6,
        slice_reindex_jobs: 231,
      },
    });
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    fireEvent.click(screen.getByTestId('modal-ok'));
    await waitFor(() => {
      expect(SpaceApi.resyncES).toHaveBeenCalledWith({ space_id: '100' });
    });
  });

  it('closes the modal after a successful resync', async () => {
    (SpaceApi.resyncES as any).mockResolvedValue({
      code: 0,
      msg: 'success',
      counts: {
        project_draft: 0,
        coze_resource: 0,
        kb_entries: 0,
        slice_reindex_jobs: 0,
      },
    });
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    fireEvent.click(screen.getByTestId('modal-ok'));
    await waitFor(() => {
      expect(screen.queryByTestId('resync-modal')).not.toBeInTheDocument();
    });
  });
});
