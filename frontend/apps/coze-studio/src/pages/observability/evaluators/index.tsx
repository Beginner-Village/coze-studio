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

import { useParams } from 'react-router-dom';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { IconCozPlus, IconCozRefresh } from '@coze-arch/coze-design/icons';
import {
  Button,
  Dropdown,
  Layout,
  Space,
  Spin,
  Table,
  Tag,
  Toast,
} from '@coze-arch/coze-design';

import {
  listEvaluators,
  type Evaluator,
  type LoopUser,
} from '../loop-eval-api';

const PAGE_SIZE = 20;
const LOOP_BASE = 'http://10.10.10.220:8082/console/enterprise/personal';

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function getEvaluatorId(record: Evaluator): string {
  return String(record.evaluator_id || record.id || '');
}

function formatCreator(creator?: string | LoopUser): string {
  if (!creator) {
    return '-';
  }
  if (typeof creator === 'string') {
    return creator;
  }
  return (
    creator.nickname ||
    creator.name ||
    creator.username ||
    creator.user_id ||
    creator.id ||
    '-'
  );
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

function formatEvaluatorType(type?: string | number): string {
  const normalized = String(type || '').toLowerCase();
  if (normalized === '1' || normalized.includes('llm')) {
    return 'LLM';
  }
  if (normalized === '2' || normalized.includes('code')) {
    return 'Code';
  }
  if (normalized === '3' || normalized.includes('agent')) {
    return 'Agent';
  }
  return type ? String(type) : '-';
}

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const [evaluators, setEvaluators] = useState<Evaluator[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchEvaluators = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    setLoading(true);
    try {
      const res = await listEvaluators({
        workspace_id: spaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      setEvaluators(res.evaluators || []);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估器失败'));
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  useEffect(() => {
    fetchEvaluators();
  }, [fetchEvaluators]);

  const openCreate = (type: string) => {
    if (!spaceId) {
      return;
    }
    window.open(
      `${LOOP_BASE}/space/${spaceId}/evaluation/evaluators/create?type=${type}`,
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
        title: '类型',
        dataIndex: 'evaluator_type',
        key: 'evaluator_type',
        width: 120,
        render: (type: string | number) => (
          <Tag>{formatEvaluatorType(type)}</Tag>
        ),
      },
      {
        title: '最新版本',
        dataIndex: 'latest_version',
        key: 'latest_version',
        width: 120,
        render: (version: string) => version || '-',
      },
      {
        title: '创建人',
        dataIndex: 'creator',
        key: 'creator',
        width: 160,
        render: formatCreator,
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

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">评估器</div>
            <div className="text-sm text-gray-500 mt-1">
              管理 Loop Evaluators
            </div>
          </div>
          <Space>
            <Button
              icon={<IconCozRefresh />}
              onClick={fetchEvaluators}
              loading={loading}
            >
              刷新
            </Button>
            <Dropdown
              trigger="click"
              position="bottomRight"
              render={
                <Dropdown.Menu>
                  <Dropdown.Item onClick={() => openCreate('llm')}>
                    LLM 评估器
                  </Dropdown.Item>
                  <Dropdown.Item onClick={() => openCreate('code')}>
                    Code 评估器
                  </Dropdown.Item>
                  <Dropdown.Item onClick={() => openCreate('agent')}>
                    Agent 评估器
                  </Dropdown.Item>
                </Dropdown.Menu>
              }
            >
              <Button type="primary" icon={<IconCozPlus />}>
                新建评估器
              </Button>
            </Dropdown>
          </Space>
        </div>
      </Layout.Header>

      <Layout.Content>
        {loading && evaluators.length === 0 ? (
          <div className="py-16 text-center">
            <Spin />
          </div>
        ) : (
          <Table
            tableProps={{
              loading,
              dataSource: evaluators,
              columns,
              rowKey: getEvaluatorId,
              pagination: false,
              empty: (
                <div className="text-center py-8 text-gray-500">暂无评估器</div>
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
