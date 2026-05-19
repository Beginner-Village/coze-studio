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

import { useNavigate, useParams } from 'react-router-dom';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { IconCozPlus, IconCozRefresh } from '@coze-arch/coze-design/icons';
import {
  Button,
  Layout,
  Space,
  Spin,
  Table,
  Tag,
  Toast,
} from '@coze-arch/coze-design';

import {
  listExperiments,
  type EvaluationSet,
  type Experiment,
} from '../loop-eval-api';

const PAGE_SIZE = 20;
const LOOP_BASE = 'http://10.10.10.220:8082/console/enterprise/personal';

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function getExperimentId(record: Experiment): string {
  return String(record.experiment_id || record.expt_id || record.id || '');
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
  if (['13', '14', 'terminated'].includes(normalized)) {
    return { text: '已终止', color: 'default' };
  }
  return { text: status ? String(status) : '-', color: 'default' };
}

function formatEvalSet(value?: string | EvaluationSet): string {
  if (!value) {
    return '-';
  }
  if (typeof value === 'string') {
    return value;
  }
  return value.name || value.evaluation_set_id || value.id || '-';
}

function formatProgress(progress?: Experiment['progress']): string {
  if (progress === undefined || progress === null || progress === '') {
    return '-';
  }
  if (typeof progress === 'number') {
    return `${Math.round(progress * 100)}%`;
  }
  if (typeof progress === 'string') {
    return progress;
  }
  const total = progress.total || 0;
  const finished = progress.finished || progress.success || 0;
  return total > 0 ? `${finished}/${total}` : '-';
}

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const navigate = useNavigate();
  const [experiments, setExperiments] = useState<Experiment[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchExperiments = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    setLoading(true);
    try {
      const res = await listExperiments({
        workspace_id: spaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      setExperiments(res.experiments || []);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载实验失败'));
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  useEffect(() => {
    fetchExperiments();
  }, [fetchExperiments]);

  const openCreate = () => {
    if (!spaceId) {
      return;
    }
    window.open(
      `${LOOP_BASE}/space/${spaceId}/evaluation/experiments/create`,
      '_blank',
      'noopener,noreferrer',
    );
  };

  const columns = useMemo(
    () => [
      {
        title: '名称',
        dataIndex: 'name',
        key: 'name',
        render: (name: string) => name || '-',
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (status: string | number) => {
          const meta = statusMeta(status);
          return <Tag color={meta.color}>{meta.text}</Tag>;
        },
      },
      {
        title: '评估集',
        key: 'eval_set',
        render: (_: unknown, record: Experiment) =>
          record.eval_set_name || formatEvalSet(record.eval_set),
      },
      {
        title: '评估器数',
        dataIndex: 'evaluator_count',
        key: 'evaluator_count',
        width: 100,
        render: (count: number) => count ?? 0,
      },
      {
        title: '进度',
        dataIndex: 'progress',
        key: 'progress',
        width: 120,
        render: formatProgress,
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 180,
        render: formatTime,
      },
      {
        title: '操作',
        key: 'actions',
        width: 120,
        render: (_: unknown, record: Experiment) => (
          <Button
            size="small"
            theme="borderless"
            onClick={() =>
              window.open(
                `http://10.10.10.220:8082/console/enterprise/personal/space/${spaceId}/evaluation/experiments/${getExperimentId(record)}`,
                '_blank',
              )
            }
          >
            查看详情
          </Button>
        ),
      },
    ],
    [navigate, spaceId],
  );

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">实验</div>
            <div className="text-sm text-gray-500 mt-1">
              管理 Loop Experiments
            </div>
          </div>
          <Space>
            <Button
              icon={<IconCozRefresh />}
              onClick={fetchExperiments}
              loading={loading}
            >
              刷新
            </Button>
            <Button type="primary" icon={<IconCozPlus />} onClick={openCreate}>
              新建实验
            </Button>
          </Space>
        </div>
      </Layout.Header>

      <Layout.Content>
        {loading && experiments.length === 0 ? (
          <div className="py-16 text-center">
            <Spin />
          </div>
        ) : (
          <Table
            tableProps={{
              loading,
              dataSource: experiments,
              columns,
              rowKey: getExperimentId,
              pagination: false,
              empty: (
                <div className="text-center py-8 text-gray-500">暂无实验</div>
              ),
            }}
          />
        )}
      </Layout.Content>
    </Layout>
  );
};

export { Page as Component };
export default Page;
