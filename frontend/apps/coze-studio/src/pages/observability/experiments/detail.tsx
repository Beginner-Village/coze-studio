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
/* eslint-disable @coze-arch/max-line-per-function */
/* eslint-disable complexity */
/* eslint-disable max-lines -- 实验详情单页,拆分收益低 */

import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import {
  Button,
  Layout,
  Spin,
  Table,
  Tabs,
  Tag,
  Toast,
} from '@coze-arch/coze-design';

import {
  getExperiment,
  getExperimentAggrResult,
  listExperimentResults,
  type Experiment,
  type ExperimentColumnEvaluator,
  type ExperimentColumnField,
  type ExperimentResultRow,
  type TrajectoryConfig,
} from '../loop-eval-api';

const ROWS_PAGE_SIZE = 20;

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function formatTime(value?: string | number): string {
  if (!value) {
    return '-';
  }
  const n = Number(value);
  const date = Number.isNaN(n)
    ? new Date(value)
    : new Date(n > 10_000_000_000 ? n : n * 1000);
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString('zh-CN');
}

function statusMeta(status?: string | number): { text: string; color: string } {
  const normalized = String(status || '').toLowerCase();
  if (['11', 'success', 'succeeded'].includes(normalized)) {
    return { text: '成功', color: 'green' };
  }
  if (['12', 'failed', 'fail'].includes(normalized)) {
    return { text: '失败', color: 'red' };
  }
  if (['3', 'processing', 'running'].includes(normalized)) {
    return { text: '执行中', color: 'blue' };
  }
  if (['2', 'pending'].includes(normalized)) {
    return { text: '待执行', color: 'orange' };
  }
  return { text: status ? String(status) : '-', color: 'default' };
}

function isRunningStatus(status?: string | number): boolean {
  const normalized = String(status || '').toLowerCase();
  return ['2', '3', 'pending', 'processing', 'running'].includes(normalized);
}

function stringify(value: unknown): string {
  if (value === undefined || value === null || value === '') {
    return '-';
  }
  if (typeof value === 'string') {
    return value;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const [searchParams] = useSearchParams();
  const exptId = searchParams.get('id') || '';
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState('overview');
  const [experiment, setExperiment] = useState<Experiment | null>(null);
  const [aggrResult, setAggrResult] = useState<Record<string, unknown> | null>(
    null,
  );
  const [rows, setRows] = useState<ExperimentResultRow[]>([]);
  const [rowCols, setRowCols] = useState<{
    fields: ExperimentColumnField[];
    evaluators: ExperimentColumnEvaluator[];
  }>({ fields: [], evaluators: [] });
  const [rowsTotal, setRowsTotal] = useState(0);
  const [rowsPage, setRowsPage] = useState(1);
  const [rowsLoading, setRowsLoading] = useState(false);
  const [trajectoryConfigs, setTrajectoryConfigs] = useState<
    TrajectoryConfig[]
  >([]);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [resultUnavailable, setResultUnavailable] = useState(false);
  const [trajectoryUnavailable] = useState(false);

  // Lightweight status/progress refresh, reused by the manual refresh button and
  // the auto-poll loop while the experiment is still running.
  const refreshStatus = useCallback(async () => {
    if (!spaceId || !exptId) {
      return;
    }
    setRefreshing(true);
    try {
      const detail = await getExperiment({
        workspace_id: spaceId,
        experiment_id: exptId,
      });
      setExperiment(detail.experiment || null);
      if (!isRunningStatus(detail.experiment?.status)) {
        try {
          const result = await getExperimentAggrResult({
            workspace_id: spaceId,
            experiment_id: exptId,
          });
          setAggrResult(result);
          setResultUnavailable(false);
        } catch (err) {
          // Aggregate result may not be ready yet; leave previous state.
          console.error('[ExperimentDetail] aggr result not ready:', err);
        }
      }
    } catch (err) {
      console.error('[ExperimentDetail] Failed to refresh status:', err);
    } finally {
      setRefreshing(false);
    }
  }, [spaceId, exptId]);

  const fetchDetail = useCallback(async () => {
    if (!spaceId || !exptId) {
      return;
    }
    setLoading(true);
    try {
      const detail = await getExperiment({
        workspace_id: spaceId,
        experiment_id: exptId,
      });
      setExperiment(detail.experiment || null);

      try {
        const result = await getExperimentAggrResult({
          workspace_id: spaceId,
          experiment_id: exptId,
        });
        setAggrResult(result);
        setResultUnavailable(false);
      } catch (err) {
        console.error(
          '[ExperimentDetail] Failed to fetch aggregate result:',
          err,
        );
        setAggrResult(null);
        setResultUnavailable(true);
      }

      // Loop's trajectory endpoint requires concrete trace_ids. Experiment
      // detail responses do not expose them yet, so avoid a failing probe here.
      setTrajectoryConfigs([]);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载实验详情失败'));
    } finally {
      setLoading(false);
    }
  }, [spaceId, exptId]);

  useEffect(() => {
    fetchDetail();
  }, [fetchDetail]);

  // 逐行结果：每条评测项的目标输出 + 各评估器打分（batch test 的核心产出）。
  useEffect(() => {
    if (!spaceId || !exptId) {
      return;
    }
    let cancelled = false;
    setRowsLoading(true);
    listExperimentResults({
      workspace_id: spaceId,
      experiment_id: exptId,
      page_number: rowsPage,
      page_size: ROWS_PAGE_SIZE,
    })
      .then(res => {
        if (cancelled) {
          return;
        }
        setRows(res.item_results || []);
        setRowCols({
          fields: res.column_eval_set_fields || [],
          evaluators: res.expt_column_evaluators || res.column_evaluators || [],
        });
        setRowsTotal(Number(res.total) || 0);
      })
      .catch(err => {
        console.error('[ExperimentDetail] fetch rows failed:', err);
      })
      .finally(() => {
        if (!cancelled) {
          setRowsLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [spaceId, exptId, rowsPage]);

  // Auto-poll status/progress while the experiment is still running.
  useEffect(() => {
    if (!isRunningStatus(experiment?.status)) {
      return;
    }
    const timer = setInterval(() => {
      refreshStatus();
    }, 5000);
    return () => clearInterval(timer);
  }, [experiment?.status, refreshStatus]);

  const trajectoryColumns = useMemo(
    () => [
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
        render: (name: string) => name || '-',
      },
      {
        title: '描述',
        dataIndex: 'description',
        key: 'description',
        render: (description: string) => description || '-',
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 180,
        render: formatTime,
      },
    ],
    [],
  );

  const meta = statusMeta(experiment?.status);

  const stats = experiment?.expt_stats;
  const totalTurns = stats
    ? (stats.success_turn_cnt || 0) +
      (stats.fail_turn_cnt || 0) +
      (stats.pending_turn_cnt || 0) +
      (stats.processing_turn_cnt || 0) +
      (stats.terminated_turn_cnt || 0)
    : 0;
  const successTurns = stats?.success_turn_cnt || 0;
  const failTurns = stats?.fail_turn_cnt || 0;
  const successRate =
    totalTurns > 0 ? `${Math.round((successTurns / totalTurns) * 100)}%` : '-';
  const targetType = experiment?.eval_target?.eval_target_type;
  const targetTypeLabel =
    targetType === 4
      ? '工作流'
      : targetType === 1
        ? '智能体'
        : targetType === 2
          ? 'Prompt'
          : '评测对象';
  const targetSourceId =
    experiment?.eval_target?.source_target_id || experiment?.target_id;
  const evalSet = experiment?.eval_set;
  const evalSetName =
    typeof evalSet === 'object' && evalSet
      ? (evalSet as { name?: string }).name
      : experiment?.eval_set_name;
  const evaluatorCount =
    experiment?.evaluator_count ??
    experiment?.evaluator_version_ids?.length ??
    0;
  const startSec = Number(experiment?.start_time || 0);
  const endSec = Number(experiment?.end_time || 0);
  const durationText =
    startSec > 0 && endSec >= startSec ? `${endSec - startSec} s` : '-';

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">
              {experiment?.name || '实验详情'}
            </div>
            <div className="text-sm text-gray-500 mt-1">
              {experiment?.description || `ID: ${exptId || '-'}`}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button loading={refreshing} onClick={refreshStatus}>
              刷新
            </Button>
            <Button
              onClick={() =>
                navigate(`/space/${spaceId}/observability?tab=experiments`)
              }
            >
              返回列表
            </Button>
          </div>
        </div>
      </Layout.Header>

      <Layout.Content>
        {loading ? (
          <div className="py-16 text-center">
            <Spin />
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <Tabs activeKey={activeTab} onChange={setActiveTab}>
              <Tabs.TabPane tab="概览" itemKey="overview" />
              <Tabs.TabPane tab="结果" itemKey="results" />
              <Tabs.TabPane tab="轨迹分析" itemKey="trajectory" />
            </Tabs>

            {activeTab === 'overview' && (
              <div className="grid grid-cols-4 gap-4 rounded-[6px] border p-4">
                <div>
                  <div className="text-xs text-gray-500 mb-1">状态</div>
                  <Tag color={meta.color}>{meta.text}</Tag>
                </div>
                <Meta
                  label="评测对象"
                  value={
                    targetSourceId
                      ? `${targetTypeLabel} #${targetSourceId}`
                      : targetTypeLabel
                  }
                />
                <Meta label="评测集" value={evalSetName || '-'} />
                <Meta
                  label="成功 / 总数"
                  value={
                    totalTurns > 0 ? `${successTurns} / ${totalTurns}` : '-'
                  }
                />
                <Meta label="成功率" value={successRate} />
                <Meta label="失败数" value={String(failTurns)} />
                <Meta label="评估器数" value={String(evaluatorCount)} />
                <Meta label="耗时" value={durationText} />
                <Meta
                  label="创建时间"
                  value={formatTime(
                    experiment?.created_at ||
                      experiment?.base_info?.created_at ||
                      experiment?.start_time,
                  )}
                />
                <Meta label="ID" value={exptId || '-'} />
              </div>
            )}

            {activeTab === 'results' && (
              <ResultsPanel
                rows={rows}
                cols={rowCols}
                total={rowsTotal}
                page={rowsPage}
                loading={rowsLoading}
                onPrev={() => setRowsPage(p => Math.max(1, p - 1))}
                onNext={() => setRowsPage(p => p + 1)}
                aggrResult={resultUnavailable ? null : aggrResult}
              />
            )}

            {activeTab === 'trajectory' &&
              (trajectoryUnavailable ? (
                <div className="text-center py-16 text-gray-500 border rounded-[6px]">
                  暂无轨迹分析配置
                </div>
              ) : (
                <Table
                  tableProps={{
                    dataSource: trajectoryConfigs,
                    columns: trajectoryColumns,
                    rowKey: (record: TrajectoryConfig) =>
                      record.config_id || record.id || '',
                    pagination: false,
                    empty: (
                      <div className="text-center py-8 text-gray-500">
                        暂无轨迹分析数据
                      </div>
                    ),
                  }}
                />
              ))}
          </div>
        )}
      </Layout.Content>
    </Layout>
  );
};

const Meta: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div>
    <div className="text-xs text-gray-500 mb-1">{label}</div>
    <div className="text-sm text-gray-900 break-all">{value}</div>
  </div>
);

interface ExtractedResultRow {
  key: number;
  inputs: Record<string, string>;
  output: string;
  latency?: number;
  error?: string;
  scores: Record<string, string>;
}

// 从逐行结果里提取输入/目标输出/评估器分/耗时（结构较深，全程防御式取值）。
function extractResultRow(
  item: ExperimentResultRow,
  seq: number,
): ExtractedResultRow {
  const it = item as Record<string, unknown>;
  const payload = (
    (it.turn_results as Array<Record<string, unknown>> | undefined)?.[0]
      ?.experiment_results as Array<Record<string, unknown>> | undefined
  )?.[0]?.payload as Record<string, unknown> | undefined;
  const rec = (payload?.target_output as Record<string, unknown> | undefined)
    ?.eval_target_record as Record<string, unknown> | undefined;
  const inputFields = (
    rec?.eval_target_input_data as Record<string, unknown> | undefined
  )?.input_fields as Record<string, { text?: string }> | undefined;
  const inputs: Record<string, string> = {};
  Object.entries(inputFields || {}).forEach(([k, v]) => {
    inputs[k] = v?.text ?? '';
  });
  const outData = rec?.eval_target_output_data as
    | Record<string, unknown>
    | undefined;
  const output =
    (outData?.output_fields as Record<string, { text?: string }> | undefined)
      ?.actual_output?.text ?? '';
  const latency = outData?.time_consuming_ms as number | undefined;
  const errObj = outData?.eval_target_run_error as
    | { code?: number; message?: string }
    | undefined;
  const error =
    errObj && (errObj.code || errObj.message)
      ? errObj.message || `执行失败（错误码 ${errObj.code}）`
      : undefined;
  const scores: Record<string, string> = {};
  const evalRecords = (
    payload?.evaluator_output as
      | {
          evaluator_records?: Record<
            string,
            { evaluator_result?: { score?: number }; score?: number }
          >;
        }
      | undefined
  )?.evaluator_records;
  Object.entries(evalRecords || {}).forEach(([k, v]) => {
    const s = v?.evaluator_result?.score ?? v?.score;
    scores[k] = s === undefined || s === null ? '-' : String(s);
  });
  return { key: seq, inputs, output, latency, error, scores };
}

const Cell: React.FC<{ text: string }> = ({ text }) => (
  <div className="whitespace-pre-wrap break-words text-sm max-w-[440px]">
    {text || '-'}
  </div>
);

// 逐行结果面板：表格化展示（输入列 / 目标输出 / 各评估器分 / 耗时），列随数据集与
// 评估器动态生成；聚合结果收进可折叠区。
const ResultsPanel: React.FC<{
  rows: ExperimentResultRow[];
  cols: {
    fields: ExperimentColumnField[];
    evaluators: ExperimentColumnEvaluator[];
  };
  total: number;
  page: number;
  loading: boolean;
  onPrev: () => void;
  onNext: () => void;
  aggrResult: Record<string, unknown> | null;
}> = ({ rows, cols, total, page, loading, onPrev, onNext, aggrResult }) => {
  const data = rows.map((r, i) =>
    extractResultRow(r, (page - 1) * ROWS_PAGE_SIZE + i + 1),
  );
  const inputCols = cols.fields.map(f => {
    const k = f.key || f.name || '';
    return {
      title: f.name || f.key || '输入',
      key: `in_${k}`,
      render: (_: unknown, r: ExtractedResultRow) => (
        <Cell text={r.inputs[k]} />
      ),
    };
  });
  const evalCols = cols.evaluators.map((e, i) => {
    const k = e.evaluator_version_id || e.evaluator_id || String(i);
    return {
      title: e.name || '评估器',
      key: `ev_${k}`,
      width: 110,
      render: (_: unknown, r: ExtractedResultRow) => r.scores[k] ?? '-',
    };
  });
  const columns = [
    { title: '#', dataIndex: 'key', key: 'seq', width: 56 },
    ...inputCols,
    {
      title: '目标输出',
      key: 'output',
      render: (_: unknown, r: ExtractedResultRow) =>
        r.error ? <Tag color="red">{r.error}</Tag> : <Cell text={r.output} />,
    },
    ...evalCols,
    {
      title: '耗时',
      key: 'latency',
      width: 90,
      render: (_: unknown, r: ExtractedResultRow) =>
        r.latency ? `${r.latency} ms` : '-',
    },
  ];
  return (
    <div className="flex flex-col gap-4">
      {loading && rows.length === 0 ? (
        <div className="py-16 text-center">
          <Spin />
        </div>
      ) : rows.length === 0 ? (
        <div className="text-center py-16 text-gray-500 border rounded-[6px]">
          暂无逐行结果
        </div>
      ) : (
        <div className="rounded-[6px] border">
          <Table
            tableProps={{
              dataSource: data,
              columns,
              rowKey: 'key',
              pagination: false,
            }}
          />
        </div>
      )}
      <div className="flex items-center justify-end gap-3">
        <span className="text-sm text-gray-500">
          第 {page} 页{total ? ` / 共 ${total} 条` : ''}
        </span>
        <Button size="small" disabled={page <= 1 || loading} onClick={onPrev}>
          上一页
        </Button>
        <Button
          size="small"
          disabled={rows.length < ROWS_PAGE_SIZE || loading}
          onClick={onNext}
        >
          下一页
        </Button>
      </div>
      {aggrResult ? (
        <details className="rounded-[6px] border p-3">
          <summary className="text-sm text-gray-600 cursor-pointer">
            汇总结果 (aggregate)
          </summary>
          <pre className="mt-2 overflow-auto text-xs bg-gray-50 p-2 rounded">
            {stringify(aggrResult)}
          </pre>
        </details>
      ) : null}
    </div>
  );
};

export { Page as Component };
export default Page;
