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

import { Checkbox } from '@coze-arch/coze-design';

import { type MergeCandidate } from './use-merge-state';

const PREVIEW_MAX_LEN = 60;

const previewText = (text: string): string => {
  const trimmed = (text ?? '').replace(/\s+/g, ' ').trim();
  return trimmed.length > PREVIEW_MAX_LEN
    ? `${trimmed.slice(0, PREVIEW_MAX_LEN)}...`
    : trimmed;
};

export interface MergeCandidateListProps {
  candidates: MergeCandidate[];
  selectedSliceIds: string[];
  onToggle: (sliceId: string) => void;
}

export const MergeCandidateList: React.FC<MergeCandidateListProps> = ({
  candidates,
  selectedSliceIds,
  onToggle,
}) => (
  <div className="flex flex-col gap-1 border border-solid coz-stroke-primary rounded-[6px] p-2 max-h-[200px] overflow-auto">
    {candidates.map(c => (
      <label
        key={c.slice_id}
        className="flex items-start gap-2 px-1 py-1 cursor-pointer hover:coz-mg-secondary-hovered rounded"
      >
        <Checkbox
          checked={selectedSliceIds.includes(c.slice_id)}
          onChange={() => onToggle(c.slice_id)}
          data-testid={`merge-checkbox-${c.slice_id}`}
        />
        <span className="text-xs coz-fg-secondary shrink-0 w-10">
          #{c.sequence}
        </span>
        <span className="text-sm coz-fg-primary truncate">
          {previewText(c.content)}
        </span>
      </label>
    ))}
  </div>
);
