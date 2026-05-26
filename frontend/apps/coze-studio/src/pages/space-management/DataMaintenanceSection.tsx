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

import { useState, type FC } from 'react';

import { I18n } from '@coze-arch/i18n';
import { Button, Modal, Toast } from '@coze-arch/coze-design';
import { SpaceApi } from '@coze-arch/bot-api';

import styles from './DataMaintenanceSection.module.less';
import { DiagnoseCard } from './DiagnoseCard';
import { ConfigureModelsCard } from './ConfigureModelsCard';

interface Props {
  spaceId: string;
}

export const DataMaintenanceSection: FC<Props> = ({ spaceId }) => {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleResync = async () => {
    setLoading(true);
    try {
      const resp = await SpaceApi.resyncES({ space_id: spaceId });
      if (resp.code !== 0) {
        throw new Error(resp.msg || 'unknown error');
      }
      const c = resp.counts ?? {
        project_draft: 0,
        coze_resource: 0,
        kb_entries: 0,
        slice_reindex_jobs: 0,
      };
      const summary =
        `同步完成: 智能体 ${c.project_draft} / 资源 ${c.coze_resource} / ` +
        `知识库 ${c.kb_entries} / 切片重新索引中 ${c.slice_reindex_jobs}`;
      Toast.success(
        I18n.t(
          'space_resync_success',
          {
            agents: c.project_draft,
            resources: c.coze_resource,
            kbs: c.kb_entries,
            slices: c.slice_reindex_jobs,
          },
          summary,
        ),
      );
      setConfirmOpen(false);
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'unknown error';
      Toast.error(I18n.t('space_resync_failed', { msg }, `同步失败: ${msg}`));
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className={styles.section}>
      <h3 className={styles.title}>
        {I18n.t('space_data_maintenance', {}, '数据维护')}
      </h3>
      <DiagnoseCard spaceId={spaceId} />
      <div className={styles.card} style={{ marginTop: 16 }}>
        <div className={styles.cardTitle}>
          {I18n.t('space_resync_es_title', {}, '重新同步 ES 索引')}
        </div>
        <div className={styles.cardDesc}>
          {I18n.t(
            'space_resync_es_desc',
            {},
            '清空本空间所有 ES 索引并从 MySQL 重写。适用于：列表数据展示不全、检索结果跟实际不符、从备份导入数据后等场景。',
          )}
        </div>
        <Button
          color="red"
          loading={loading}
          onClick={() => setConfirmOpen(true)}
          data-testid="space-resync-es-button"
        >
          {I18n.t('space_resync_es_button', {}, '重新同步')}
        </Button>
      </div>
      <Modal
        visible={confirmOpen}
        title={I18n.t(
          'space_resync_es_confirm_title',
          {},
          '确认重新同步 ES 索引？',
        )}
        onOk={handleResync}
        okText={I18n.t('Confirm', {}, '确认')}
        cancelText={I18n.t('Cancel', {}, '取消')}
        confirmLoading={loading}
        onCancel={() => setConfirmOpen(false)}
      >
        <p>
          {I18n.t(
            'space_resync_es_confirm_desc',
            {},
            '将清空本空间所有 ES 索引并从 MySQL 重写。期间 1-5 分钟内列表查询可能为空，知识库检索结果可能不完整。仅 space owner 可执行。',
          )}
        </p>
      </Modal>

      <ConfigureModelsCard spaceId={spaceId} />
    </section>
  );
};

export default DataMaintenanceSection;
