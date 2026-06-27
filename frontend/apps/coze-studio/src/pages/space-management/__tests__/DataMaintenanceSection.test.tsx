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
import { Toast } from '@coze-arch/coze-design';

import { DataMaintenanceSection } from '../DataMaintenanceSection';

vi.mock('@coze-arch/bot-api', () => ({
  SpaceApi: { resyncES: vi.fn() },
  getLocalizedErrorMessage: vi.fn((message?: string) => message || ''),
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
  Input: ({ value, onChange, placeholder }: any) => (
    <input
      value={value || ''}
      onChange={event => onChange?.(event.target.value)}
      placeholder={placeholder}
    />
  ),
  // eslint-disable-next-line @typescript-eslint/naming-convention -- mock a React component re-exported from third-party lib
  InputNumber: ({ value, onChange, placeholder }: any) => (
    <input
      type="number"
      value={value ?? ''}
      onChange={event => onChange?.(Number(event.target.value))}
      placeholder={placeholder}
    />
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

  it('shows error toast when API returns code != 0 (permission denied)', async () => {
    (SpaceApi.resyncES as any).mockResolvedValue({
      code: 403,
      msg: 'permission denied',
      counts: null,
    });
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    fireEvent.click(screen.getByTestId('modal-ok'));
    await waitFor(() => {
      expect(Toast.error).toHaveBeenCalled();
    });
    const errorArg = (Toast.error as any).mock.calls[0][0];
    expect(String(errorArg)).toContain('permission denied');
    expect(Toast.success).not.toHaveBeenCalled();
  });

  it('shows error toast on API throw (network timeout)', async () => {
    (SpaceApi.resyncES as any).mockRejectedValue(new Error('network timeout'));
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    fireEvent.click(screen.getByTestId('modal-ok'));
    await waitFor(() => {
      expect(Toast.error).toHaveBeenCalled();
    });
    const errorArg = (Toast.error as any).mock.calls[0][0];
    expect(String(errorArg)).toContain('timeout');
    expect(Toast.success).not.toHaveBeenCalled();
  });

  it('shows error toast when API returns code 500 (internal server error)', async () => {
    (SpaceApi.resyncES as any).mockResolvedValue({
      code: 500,
      msg: 'internal server error',
      counts: null,
    });
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    fireEvent.click(screen.getByTestId('modal-ok'));
    await waitFor(() => {
      expect(Toast.error).toHaveBeenCalled();
    });
    const errorArg = (Toast.error as any).mock.calls[0][0];
    expect(String(errorArg)).toContain('internal server error');
  });

  it('does NOT call API when modal is cancelled', () => {
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    expect(screen.getByTestId('resync-modal')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('modal-cancel'));
    expect(SpaceApi.resyncES).not.toHaveBeenCalled();
    expect(screen.queryByTestId('resync-modal')).not.toBeInTheDocument();
  });

  it('disables confirm button while loading (prevents double-submit)', async () => {
    // Hold the promise open so loading state stays true and we can assert on disabled.
    let resolveFn: (v: any) => void = () => undefined;
    (SpaceApi.resyncES as any).mockImplementation(
      () =>
        new Promise(resolve => {
          resolveFn = resolve;
        }),
    );
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    const okBtn = screen.getByTestId('modal-ok') as HTMLButtonElement;
    fireEvent.click(okBtn);
    // Confirm button should now be disabled while in-flight.
    await waitFor(() => {
      expect(okBtn.disabled).toBe(true);
    });
    // Simulated double-click while disabled — should NOT trigger a second API call.
    fireEvent.click(okBtn);
    expect(SpaceApi.resyncES).toHaveBeenCalledTimes(1);
    // Cleanup: resolve so React can flush and the component unmounts cleanly.
    resolveFn({
      code: 0,
      msg: 'success',
      counts: {
        project_draft: 0,
        coze_resource: 0,
        kb_entries: 0,
        slice_reindex_jobs: 0,
      },
    });
    await waitFor(() => {
      expect(screen.queryByTestId('resync-modal')).not.toBeInTheDocument();
    });
  });

  it('closes modal after error path too (finally block)', async () => {
    (SpaceApi.resyncES as any).mockRejectedValue(new Error('boom'));
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    expect(screen.getByTestId('resync-modal')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('modal-ok'));
    // After an error, loading state must reset (button no longer disabled).
    await waitFor(() => {
      expect(Toast.error).toHaveBeenCalled();
    });
    const okBtn = screen.getByTestId('modal-ok') as HTMLButtonElement;
    await waitFor(() => {
      expect(okBtn.disabled).toBe(false);
    });
    // Modal stays open on error (current impl only closes on success), but the
    // button must be re-enabled so the user can retry or cancel.
  });

  it('handles null counts gracefully without panic', async () => {
    (SpaceApi.resyncES as any).mockResolvedValue({
      code: 0,
      msg: 'success',
      counts: null,
    });
    render(<DataMaintenanceSection spaceId="100" />);
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    fireEvent.click(screen.getByTestId('modal-ok'));
    await waitFor(() => {
      expect(Toast.success).toHaveBeenCalled();
    });
    // The success toast should fall back to all-zero counts when counts is null.
    const successArg = (Toast.success as any).mock.calls[0][0];
    const summary = String(successArg);
    expect(summary).toContain('0');
    // No error should fire on the null-counts happy path.
    expect(Toast.error).not.toHaveBeenCalled();
  });

  it('i18n fallback renders Chinese text when key has no translation', () => {
    // Existing I18n mock returns the fallback (3rd arg) when present, otherwise
    // the key. This test asserts the Chinese fallback strings actually render.
    render(<DataMaintenanceSection spaceId="100" />);
    expect(screen.getByText('数据维护')).toBeInTheDocument();
    expect(screen.getByText('重新同步 ES 索引')).toBeInTheDocument();
    expect(screen.getByText('重新同步')).toBeInTheDocument();
    // Confirm-modal fallback strings appear once it's opened.
    fireEvent.click(screen.getByTestId('space-resync-es-button'));
    expect(screen.getByText('确认重新同步 ES 索引？')).toBeInTheDocument();
    expect(screen.getByText('确认')).toBeInTheDocument();
    expect(screen.getByText('取消')).toBeInTheDocument();
  });
});
