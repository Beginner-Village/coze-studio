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

import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { KnowledgeApi } from '@coze-arch/bot-api';

import { useSliceStatusPolling } from '../index';

vi.mock('@coze-arch/bot-api', () => ({
  KnowledgeApi: { ListSlice: vi.fn() },
}));

describe('useSliceStatusPolling', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('polls ListSlice every 5s and reports status', async () => {
    (KnowledgeApi.ListSlice as any).mockResolvedValue({
      slices: [
        { slice_id: '1', status: 0 }, // Init
        { slice_id: '2', status: 1 }, // Done
      ],
    });

    const { result } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1', '2'] }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(1);
    expect(result.current.statusMap['1']).toBe('Init');
    expect(result.current.statusMap['2']).toBe('Done');

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(2);
  });

  it('stops polling when all watched ids are Done', async () => {
    (KnowledgeApi.ListSlice as any)
      .mockResolvedValueOnce({ slices: [{ slice_id: '1', status: 0 }] })
      .mockResolvedValueOnce({ slices: [{ slice_id: '1', status: 1 }] });

    renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1'] }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(2);
  });

  it('marks watched ids as Timeout after 120s', async () => {
    (KnowledgeApi.ListSlice as any).mockResolvedValue({
      slices: [{ slice_id: '1', status: 0 }],
    });
    const { result } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1'] }),
    );

    await act(async () => {
      await vi.advanceTimersByTimeAsync(125000);
    });
    expect(result.current.statusMap['1']).toBe('Timeout');
  });

  it('cleans up timer on unmount', async () => {
    (KnowledgeApi.ListSlice as any).mockResolvedValue({
      slices: [{ slice_id: '1', status: 0 }],
    });
    const { unmount } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1'] }),
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    unmount();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(20000);
    });
    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(1);
  });
});
