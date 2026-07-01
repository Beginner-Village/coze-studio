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
  type Experiment,
  type TrajectoryConfig,
} from '../loop-eval-api';

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

function formatProgress(
  p:
    | number
    | string
    | { total?: number; finished?: number; success?: number }
    | undefined,
): string {
  if (p === undefined || p === null || p === '') {
    return '-';
  }
  if (typeof p === 'object') {
    const { total = 0, finished = 0, success = 0 } = p;
    return `${finished}/${total} (成功: ${success})`;
  }
  return String(p);
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
                <Meta label="ID" value={exptId || '-'} />
                <div>
                  <div className="text-xs text-gray-500 mb-1">状态</div>
                  <Tag color={meta.color}>{meta.text}</Tag>
                </div>
                <Meta
                  label="进度"
                  value={formatProgress(experiment?.progress)}
                />
                <Meta
                  label="评估器数"
                  value={String(experiment?.evaluator_count ?? 0)}
                />
                <Meta
                  label="创建时间"
                  value={formatTime(experiment?.created_at)}
                />
              </div>
            )}

            {activeTab === 'results' &&
              (resultUnavailable || !aggrResult ? (
                <div className="text-center py-16 text-gray-500 border rounded-[6px]">
                  暂无结果
                </div>
              ) : (
                <pre className="rounded-[6px] border p-4 overflow-auto text-sm bg-gray-50">
                  {stringify(aggrResult)}
                </pre>
              ))}

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

export { Page as Component };
export default Page;
