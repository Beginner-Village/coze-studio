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

import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { Button, Layout, Spin, Table, Toast } from '@coze-arch/coze-design';

import {
  getEvaluationSet,
  listEvaluationSetItems,
  type EvaluationSet,
  type EvaluationSetItem,
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

function stringify(value: unknown): string {
  if (value === undefined || value === null || value === '') {
    return '-';
  }
  if (typeof value === 'string') {
    return value;
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const [searchParams] = useSearchParams();
  const setId = searchParams.get('id') || '';
  const navigate = useNavigate();
  const [evaluationSet, setEvaluationSet] = useState<EvaluationSet | null>(
    null,
  );
  const [items, setItems] = useState<EvaluationSetItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [itemsUnavailable, setItemsUnavailable] = useState(false);

  const fetchDetail = useCallback(async () => {
    if (!spaceId || !setId) {
      return;
    }
    setLoading(true);
    try {
      const detail = await getEvaluationSet({
        workspace_id: spaceId,
        evaluation_set_id: setId,
      });
      setEvaluationSet(detail.evaluation_set || null);

      try {
        const itemRes = await listEvaluationSetItems({
          workspace_id: spaceId,
          evaluation_set_id: setId,
          page_size: 20,
          page_number: 1,
        });
        setItems(itemRes.items || []);
        setItemsUnavailable(false);
      } catch (err) {
        console.error('[EvaluationSetDetail] Failed to fetch items:', err);
        setItems([]);
        setItemsUnavailable(true);
      }
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估集详情失败'));
    } finally {
      setLoading(false);
    }
  }, [spaceId, setId]);

  useEffect(() => {
    fetchDetail();
  }, [fetchDetail]);

  const columns = useMemo(
    () => [
      {
        title: 'ID',
        key: 'id',
        width: 180,
        render: (_: unknown, record: EvaluationSetItem) =>
          record.item_id || record.id || '-',
      },
      {
        title: 'Input',
        dataIndex: 'input',
        key: 'input',
        render: stringify,
      },
      {
        title: 'Output',
        dataIndex: 'output',
        key: 'output',
        render: stringify,
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
            <div className="font-[500] text-[20px]">
              {evaluationSet?.name || '评估集详情'}
            </div>
            <div className="text-sm text-gray-500 mt-1">
              {evaluationSet?.description || '暂无描述'}
            </div>
          </div>
          <Button
            onClick={() =>
              navigate(`/space/${spaceId}/observability?tab=evaluation-sets`)
            }
          >
            返回列表
          </Button>
        </div>
      </Layout.Header>

      <Layout.Content>
        {loading ? (
          <div className="py-16 text-center">
            <Spin />
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-4 gap-4 rounded-[6px] border p-4">
              <Meta label="ID" value={setId || '-'} />
              <Meta
                label="版本"
                value={
                  evaluationSet?.latest_version || evaluationSet?.version || '-'
                }
              />
              <Meta
                label="条目数"
                value={String(evaluationSet?.item_count ?? 0)}
              />
              <Meta
                label="创建时间"
                value={formatTime(evaluationSet?.created_at)}
              />
            </div>

            <div>
              <div className="font-[500] text-[16px] mb-3">评估集条目</div>
              {itemsUnavailable ? (
                <div className="text-center py-16 text-gray-500 border rounded-[6px]">
                  敬请期待
                </div>
              ) : (
                <Table
                  tableProps={{
                    dataSource: items,
                    columns,
                    rowKey: (record: EvaluationSetItem) =>
                      record.item_id || record.id || '',
                    pagination: false,
                    empty: (
                      <div className="text-center py-8 text-gray-500">
                        暂无条目
                      </div>
                    ),
                  }}
                />
              )}
            </div>
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
