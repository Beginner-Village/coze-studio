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

import { renderHook, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { KnowledgeApi } from '@coze-arch/bot-api';

import { useMergeSlices } from '../index';

vi.mock('@coze-arch/bot-api', () => ({
  KnowledgeApi: {
    UpdateSlice: vi.fn(),
    DeleteSlice: vi.fn(),
  },
}));

describe('useMergeSlices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('merges slices into the smallest-sequence target via UpdateSlice + DeleteSlice', async () => {
    (KnowledgeApi.UpdateSlice as any).mockResolvedValue({});
    (KnowledgeApi.DeleteSlice as any).mockResolvedValue({});

    const { result } = renderHook(() => useMergeSlices());
    const slices = [
      { slice_id: '101', sequence: 2, content: 'B' },
      { slice_id: '100', sequence: 1, content: 'A' },
      { slice_id: '102', sequence: 3, content: 'C' },
    ];

    await act(async () => {
      const res = await result.current.merge(slices);
      expect(res.ok).toBe(true);
      if (res.ok) {
        expect(res.target_slice_id).toBe('100');
      }
    });

    expect(KnowledgeApi.UpdateSlice).toHaveBeenCalledWith({
      slice_id: '100',
      raw_text: 'A\n\nB\n\nC',
    });
    expect(KnowledgeApi.DeleteSlice).toHaveBeenCalledWith({
      slice_ids: ['101', '102'],
    });
  });

  it('returns error and skips DeleteSlice when UpdateSlice fails', async () => {
    (KnowledgeApi.UpdateSlice as any).mockRejectedValue(
      new Error('update failed'),
    );

    const { result } = renderHook(() => useMergeSlices());
    const slices = [
      { slice_id: '100', sequence: 1, content: 'A' },
      { slice_id: '101', sequence: 2, content: 'B' },
    ];

    await act(async () => {
      const res = await result.current.merge(slices);
      expect(res.ok).toBe(false);
      if (!res.ok) {
        expect(res.stage).toBe('update');
      }
    });
    expect(KnowledgeApi.DeleteSlice).not.toHaveBeenCalled();
  });

  it('retries DeleteSlice once on failure, returns partial-ok on persistent failure', async () => {
    (KnowledgeApi.UpdateSlice as any).mockResolvedValue({});
    (KnowledgeApi.DeleteSlice as any)
      .mockRejectedValueOnce(new Error('first fail'))
      .mockRejectedValueOnce(new Error('second fail'));

    const { result } = renderHook(() => useMergeSlices());
    const slices = [
      { slice_id: '100', sequence: 1, content: 'A' },
      { slice_id: '101', sequence: 2, content: 'B' },
    ];

    await act(async () => {
      const res = await result.current.merge(slices);
      expect(res.ok).toBe(false);
      if (!res.ok) {
        expect(res.stage).toBe('delete');
        expect(res.partial).toBe(true);
      }
    });
    expect(KnowledgeApi.DeleteSlice).toHaveBeenCalledTimes(2);
  });

  it('rejects empty or single-item input', async () => {
    const { result } = renderHook(() => useMergeSlices());
    await act(async () => {
      const res = await result.current.merge([]);
      expect(res.ok).toBe(false);
      if (!res.ok) {
        expect(res.stage).toBe('validate');
      }
    });
    await act(async () => {
      const res = await result.current.merge([
        { slice_id: '1', sequence: 1, content: 'A' },
      ]);
      expect(res.ok).toBe(false);
      if (!res.ok) {
        expect(res.stage).toBe('validate');
      }
    });
  });
});
