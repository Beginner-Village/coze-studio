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
import { Button, Toast } from '@coze-arch/coze-design';
import {
  SpaceApi,
  type DiagnoseData,
  type DiagnoseESIndex,
  type DiagnoseMilvusCollection,
  type DiagnoseModelProbe,
  type DiagnoseMySQLTable,
} from '@coze-arch/bot-api';

import styles from './DiagnoseCard.module.less';

interface Props {
  spaceId: string;
}

// formatTimestamp renders Date to a HH:mm:ss / YYYY-MM-DD HH:mm:ss string
// without pulling in a date library — we only need a "last run at" stamp.
const formatTimestamp = (d: Date): string => {
  const pad = (n: number) => n.toString().padStart(2, '0');
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  );
};

// ProbeMiniCard renders one of chat / embedder / rerank. Three states:
//   - configured && reachable  → green dot, latency badge
//   - configured && !reachable → red dot, error message
//   - !configured              → muted dot, "未配置" + reason
const ProbeMiniCard: FC<{ label: string; probe: DiagnoseModelProbe }> = ({
  label,
  probe,
}) => {
  let dotClass = styles.statusDotMuted;
  let statusText: string;
  if (!probe.configured) {
    statusText = I18n.t('space_diagnose_probe_not_configured', {}, '未配置');
  } else if (probe.reachable) {
    dotClass = styles.statusDotOk;
    statusText = I18n.t(
      'space_diagnose_probe_ok',
      { latency: probe.latency_ms },
      `正常 (${probe.latency_ms}ms)`,
    );
  } else {
    dotClass = styles.statusDotBad;
    statusText = I18n.t(
      'space_diagnose_probe_unreachable',
      { status: probe.http_status },
      `异常${probe.http_status ? ` (HTTP ${probe.http_status})` : ''}`,
    );
  }

  return (
    <div className={styles.probeMiniCard}>
      <div className={styles.probeMiniHeader}>
        <span className={`${styles.statusDot} ${dotClass}`} />
        <span>{label}</span>
      </div>
      <div className={styles.probeMiniBody}>
        <div>{statusText}</div>
        {probe.endpoint ? (
          <div title={probe.endpoint}>
            endpoint: <code>{probe.endpoint}</code>
          </div>
        ) : null}
        {probe.model ? (
          <div>
            model: <code>{probe.model}</code>
          </div>
        ) : null}
      </div>
      {probe.error ? (
        <div className={styles.probeMiniError}>{probe.error}</div>
      ) : null}
    </div>
  );
};

const ModelProbesSection: FC<{ data: DiagnoseData['model_probes'] }> = ({
  data,
}) => (
  <div className={styles.section}>
    <div className={styles.sectionTitle}>
      {I18n.t('space_diagnose_section_probes', {}, '模型探活')}
    </div>
    <div className={styles.probesGrid}>
      <ProbeMiniCard label="Chat" probe={data.chat} />
      <ProbeMiniCard label="Embedder" probe={data.embedder} />
      <ProbeMiniCard label="Rerank" probe={data.rerank} />
    </div>
  </div>
);

const MySQLTablesSection: FC<{ rows: DiagnoseMySQLTable[] }> = ({ rows }) => (
  <div className={styles.section}>
    <div className={styles.sectionTitle}>
      {I18n.t('space_diagnose_section_mysql', {}, 'MySQL 数据')}
    </div>
    {rows.length === 0 ? (
      <div className={styles.emptyHint}>
        {I18n.t('space_diagnose_empty', {}, '无数据')}
      </div>
    ) : (
      <table className={styles.table}>
        <thead>
          <tr>
            <th>{I18n.t('space_diagnose_th_table', {}, '表名')}</th>
            <th>{I18n.t('space_diagnose_th_rows', {}, '行数')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(r => (
            <tr key={r.name} className={r.rows === 0 ? styles.rowZero : ''}>
              <td>
                <code>{r.name}</code>
              </td>
              <td>
                {r.error ? (
                  <span className={styles.tableInlineError}>{r.error}</span>
                ) : (
                  r.rows
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    )}
  </div>
);

const ESIndicesSection: FC<{ rows: DiagnoseESIndex[] }> = ({ rows }) => (
  <div className={styles.section}>
    <div className={styles.sectionTitle}>
      {I18n.t('space_diagnose_section_es', {}, 'ES 索引')}
    </div>
    {rows.length === 0 ? (
      <div className={styles.emptyHint}>
        {I18n.t('space_diagnose_empty', {}, '无数据')}
      </div>
    ) : (
      <table className={styles.table}>
        <thead>
          <tr>
            <th>{I18n.t('space_diagnose_th_index', {}, '索引名')}</th>
            <th>{I18n.t('space_diagnose_th_doc_count', {}, '文档数')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(r => (
            <tr key={r.name} className={!r.exists ? styles.rowMissing : ''}>
              <td>
                <code>{r.name}</code>
              </td>
              <td>
                {r.exists ? (
                  r.doc_count
                ) : (
                  <span>
                    {I18n.t('space_diagnose_missing', {}, '缺失')}
                    {r.error ? (
                      <span className={styles.tableInlineError}>
                        {' '}
                        ({r.error})
                      </span>
                    ) : null}
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    )}
  </div>
);

const MilvusSection: FC<{ rows: DiagnoseMilvusCollection[] }> = ({ rows }) => (
  <div className={styles.section}>
    <div className={styles.sectionTitle}>
      {I18n.t('space_diagnose_section_milvus', {}, 'Milvus Collection')}
    </div>
    {rows.length === 0 ? (
      <div className={styles.emptyHint}>
        {I18n.t(
          'space_diagnose_no_kbs',
          {},
          '本空间暂无知识库（无需检查向量库 collection）',
        )}
      </div>
    ) : (
      <table className={styles.table}>
        <thead>
          <tr>
            <th>{I18n.t('space_diagnose_th_kb', {}, '知识库')}</th>
            <th>{I18n.t('space_diagnose_th_collection', {}, 'Collection')}</th>
            <th>{I18n.t('space_diagnose_th_status', {}, '状态')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(r => (
            <tr
              key={r.collection_name}
              className={!r.exists ? styles.rowMissing : ''}
            >
              <td>{r.kb_name || `KB#${r.kb_id}`}</td>
              <td>
                <code>{r.collection_name}</code>
              </td>
              <td>
                {r.exists
                  ? I18n.t('space_diagnose_milvus_ok', {}, '✓ 已存在')
                  : I18n.t('space_diagnose_milvus_missing', {}, '✗ 缺失')}
                {r.error ? (
                  <span className={styles.tableInlineError}> {r.error}</span>
                ) : null}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    )}
  </div>
);

export const DiagnoseCard: FC<Props> = ({ spaceId }) => {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<DiagnoseData | null>(null);
  const [ranAt, setRanAt] = useState<Date | null>(null);

  const handleRun = async () => {
    setLoading(true);
    try {
      const resp = await SpaceApi.diagnose({ space_id: spaceId });
      if (resp.code !== 0) {
        throw new Error(resp.msg || 'unknown error');
      }
      setData(resp.data);
      setRanAt(new Date());
      Toast.success(I18n.t('space_diagnose_done', {}, '诊断完成'));
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'unknown error';
      Toast.error(I18n.t('space_diagnose_failed', { msg }, `诊断失败: ${msg}`));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.card}>
      <div className={styles.header}>
        <div className={styles.headerText}>
          <div className={styles.title}>
            {I18n.t('space_diagnose_title', {}, '数据完整性诊断')}
          </div>
          <div className={styles.desc}>
            {I18n.t(
              'space_diagnose_desc',
              {},
              '一次性检查本空间的模型可达性、MySQL 关键表行数、ES 索引文档数、Milvus collection 状态。只读，不写入任何数据。仅 space owner 可执行。',
            )}
            {ranAt ? (
              <span className={styles.timestamp}>
                {I18n.t(
                  'space_diagnose_last_run',
                  { ts: formatTimestamp(ranAt) },
                  `上次运行：${formatTimestamp(ranAt)}`,
                )}
              </span>
            ) : null}
          </div>
        </div>
        <Button
          color="primary"
          loading={loading}
          onClick={handleRun}
          data-testid="space-diagnose-button"
        >
          {I18n.t('space_diagnose_run', {}, '开始诊断')}
        </Button>
      </div>

      {data ? (
        <div className={styles.sections}>
          <ModelProbesSection data={data.model_probes} />
          <MySQLTablesSection rows={data.mysql_tables ?? []} />
          <ESIndicesSection rows={data.es_indices ?? []} />
          <MilvusSection rows={data.milvus_collections ?? []} />
        </div>
      ) : null}
    </div>
  );
};

export default DiagnoseCard;
