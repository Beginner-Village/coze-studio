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
import { Spin, Button, Tooltip } from '@coze-arch/coze-design';

import styles from './index.module.less';

export type SliceBadgeStatus =
  | 'Init'
  | 'Processing'
  | 'Done'
  | 'Failed'
  | 'Timeout';

export interface SliceStatusBadgeProps {
  status: SliceBadgeStatus;
  reason?: string;
  onRetry?: () => void;
}

export const SliceStatusBadge: React.FC<SliceStatusBadgeProps> = ({
  status,
  reason,
  onRetry,
}) => {
  if (status === 'Done') {
    return null;
  }

  if (status === 'Init' || status === 'Processing') {
    return (
      <span
        data-testid="slice-status-spinner"
        className={`${styles.badge} ${styles.spinner}`}
      >
        <Spin size="small" />
        <span>{I18n.t('knowledge_slice_status_reindexing')}</span>
      </span>
    );
  }

  if (status === 'Failed') {
    const label = I18n.t('knowledge_slice_status_failed');
    return (
      <span
        data-testid="slice-status-failed"
        className={`${styles.badge} ${styles.failed}`}
      >
        <Tooltip content={reason ?? label}>
          <span>{label}</span>
        </Tooltip>
        {reason ? <span className={styles.reason}>{reason}</span> : null}
        {onRetry ? (
          <Button
            data-testid="slice-status-retry-btn"
            size="small"
            onClick={onRetry}
          >
            {I18n.t('knowledge_slice_status_retry')}
          </Button>
        ) : null}
      </span>
    );
  }

  if (status === 'Timeout') {
    return (
      <span
        data-testid="slice-status-timeout"
        className={`${styles.badge} ${styles.timeout}`}
      >
        {I18n.t('knowledge_slice_status_timeout')}
      </span>
    );
  }

  return null;
};
