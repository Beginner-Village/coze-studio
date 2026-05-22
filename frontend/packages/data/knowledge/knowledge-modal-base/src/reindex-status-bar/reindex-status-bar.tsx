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

import { I18n } from '@coze-arch/i18n';

import { SliceStatusBadge, type SliceBadgeStatus } from '../slice-status-badge';

import styles from './index.module.less';

export interface ReindexStatusBarProps {
  watchedIds: string[];
  statusMap: Record<string, SliceBadgeStatus>;
  /**
   * Optional test id applied to the root element. Lets callers in different
   * workspaces (e.g. text vs. table) target the bar distinctly in e2e tests.
   */
  testId?: string;
}

/**
 * Aggregates the per-slice statusMap into a single bar shown above the editor
 * or table. Returns null when there's nothing to show.
 */
export const ReindexStatusBar: React.FC<ReindexStatusBarProps> = ({
  watchedIds,
  statusMap,
  testId,
}) => {
  const reindexingCount = watchedIds.filter(id => {
    const s = statusMap[id];
    return s === 'Init' || s === 'Processing';
  }).length;
  const failedCount = watchedIds.filter(
    id => statusMap[id] === 'Failed',
  ).length;
  const timeoutCount = watchedIds.filter(
    id => statusMap[id] === 'Timeout',
  ).length;
  if (reindexingCount === 0 && failedCount === 0 && timeoutCount === 0) {
    return null;
  }
  return (
    <div className={styles['reindex-bar']} data-testid={testId}>
      {reindexingCount > 0 ? <SliceStatusBadge status="Processing" /> : null}
      {failedCount > 0 ? (
        <span>
          {I18n.t('knowledge_slice_reindex_failed_count', {
            count: failedCount,
          })}
        </span>
      ) : null}
      {timeoutCount > 0 ? (
        <span>
          {I18n.t('knowledge_slice_reindex_timeout_count', {
            count: timeoutCount,
          })}
        </span>
      ) : null}
    </div>
  );
};
