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

import { DataNamespace, dataReporter } from '@coze-data/reporter';
import { REPORT_EVENTS } from '@coze-arch/report-events';
import { KnowledgeApi } from '@coze-arch/bot-api';

export interface MergeSliceInput {
  slice_id: string;
  sequence: number;
  content: string;
}

export type MergeResult =
  | { ok: true; target_slice_id: string }
  | {
      ok: false;
      stage: 'validate' | 'update' | 'delete';
      partial?: boolean;
      error?: Error;
    };

export const useMergeSlices = () => {
  const merge = async (slices: MergeSliceInput[]): Promise<MergeResult> => {
    if (slices.length < 2) {
      return { ok: false, stage: 'validate' };
    }

    const sorted = [...slices].sort((a, b) => a.sequence - b.sequence);
    const target = sorted[0];
    const others = sorted.slice(1);
    const mergedContent = sorted.map(s => s.content).join('\n\n');

    try {
      await KnowledgeApi.UpdateSlice({
        slice_id: target.slice_id,
        raw_text: mergedContent,
      });
    } catch (error) {
      dataReporter.errorEvent(DataNamespace.KNOWLEDGE, {
        eventName: REPORT_EVENTS.KnowledgeUpdateSlice,
        error: error as Error,
      });
      return { ok: false, stage: 'update', error: error as Error };
    }

    const otherIds = others.map(s => s.slice_id);
    try {
      await KnowledgeApi.DeleteSlice({ slice_ids: otherIds });
    } catch (firstErr) {
      try {
        await KnowledgeApi.DeleteSlice({ slice_ids: otherIds });
      } catch (secondErr) {
        dataReporter.errorEvent(DataNamespace.KNOWLEDGE, {
          eventName: REPORT_EVENTS.KnowledgeDeleteSlice,
          error: secondErr as Error,
        });
        return {
          ok: false,
          stage: 'delete',
          partial: true,
          error: secondErr as Error,
        };
      }
    }

    return { ok: true, target_slice_id: target.slice_id };
  };

  return { merge };
};
