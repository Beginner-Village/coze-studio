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
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import { IconCozPlus, IconCozRefresh } from '@coze-arch/coze-design/icons';
import {
  Button,
  Form,
  Layout,
  Modal,
  Space,
  Spin,
  Table,
  Toast,
} from '@coze-arch/coze-design';

import {
  createEvaluationSet,
  listEvaluationSets,
  mGetUserBasicInfo,
  type EvaluationSet,
  type LoopUser,
  type UserBasicInfo,
} from '../loop-eval-api';

const PAGE_SIZE = 20;

interface FormApi {
  validate: () => Promise<{ name: string; description?: string }>;
}

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function getSetId(record: EvaluationSet): string {
  return String(record.evaluation_set_id || record.id || '');
}

function formatCreator(
  creator?: string | LoopUser,
  userMap?: Record<string, UserBasicInfo>,
): string {
  if (!creator) {
    return '-';
  }
  if (typeof creator === 'string') {
    return userMap?.[creator]?.user_name || creator;
  }
  const id = creator.user_id || creator.id;
  if (id && userMap?.[id]?.user_name) {
    return userMap[id].user_name as string;
  }
  return (
    creator.nickname ||
    creator.name ||
    creator.username ||
    id ||
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

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const navigate = useNavigate();
  const [, setSearchParams] = useSearchParams();
  const formApiRef = useRef<FormApi | null>(null);
  const [sets, setSets] = useState<EvaluationSet[]>([]);
  const [loading, setLoading] = useState(false);
  const [userMap, setUserMap] = useState<Record<string, UserBasicInfo>>({});
  const [creating, setCreating] = useState(false);
  const [createVisible, setCreateVisible] = useState(false);

  const fetchSets = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    setLoading(true);
    try {
      const res = await listEvaluationSets({
        workspace_id: spaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      const list = res.evaluation_sets || [];
      setSets(list);
      // Resolve creator user_ids → display names
      const ids = list
        .map(s => {
          const c = s.creator;
          if (typeof c === 'string') {
            return c;
          }
          return (
            c?.user_id ||
            c?.id ||
            s.base_info?.created_by?.user_id ||
            s.base_info?.created_by?.id
          );
        })
        .filter((id): id is string => !!id)
        .map(String);
      if (ids.length > 0) {
        const map = await mGetUserBasicInfo(ids);
        setUserMap(prev => ({ ...prev, ...map }));
      }
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估集失败'));
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  useEffect(() => {
    fetchSets();
  }, [fetchSets]);

  const handleCreate = async () => {
    if (!spaceId || !formApiRef.current) {
      return;
    }
    try {
      const values = await formApiRef.current.validate();
      setCreating(true);
      // Loop accepts workspace_id as string (JSON parses to int64
      // server-side). Send the URL string verbatim so list & create
      // index under the same key.
      await createEvaluationSet({
        workspace_id: spaceId,
        name: values.name,
        description: values.description,
        // Loop requires a non-empty schema. Default to a simple
        // input + reference_output text pair; users can edit later
        // via Loop's native UI.
        evaluation_set_schema: {
          field_schemas: [
            {
              key: 'input',
              name: 'input',
              description: '输入',
              content_type: 'Text',
              default_display_format: 1,
            },
            {
              key: 'reference_output',
              name: 'reference_output',
              description: '参考输出',
              content_type: 'Text',
              default_display_format: 1,
            },
          ],
        },
      });
      Toast.success('创建成功');
      setCreateVisible(false);
      fetchSets();
    } catch (err: unknown) {
      if (!(err instanceof Error)) {
        return;
      }
      Toast.error(err.message || '创建失败');
    } finally {
      setCreating(false);
    }
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
        title: '描述',
        dataIndex: 'description',
        key: 'description',
        render: (description: string) => description || '-',
      },
      {
        title: '版本',
        key: 'version',
        width: 120,
        render: (_: unknown, record: EvaluationSet) =>
          record.latest_version ||
          record.version ||
          record.evaluation_set_version?.version_num ||
          '-',
      },
      {
        title: '条目数',
        key: 'item_count',
        width: 100,
        render: (_: unknown, record: EvaluationSet) =>
          record.item_count ??
          record.evaluation_set_version?.item_count ??
          0,
      },
      {
        title: '创建人',
        key: 'creator',
        width: 160,
        render: (_: unknown, record: EvaluationSet) =>
          formatCreator(
            record.creator || record.base_info?.created_by,
            userMap,
          ),
      },
      {
        title: '创建时间',
        key: 'created_at',
        width: 180,
        render: (_: unknown, record: EvaluationSet) =>
          formatTime(record.created_at || record.base_info?.created_at),
      },
      {
        title: '操作',
        key: 'actions',
        width: 120,
        render: (_: unknown, record: EvaluationSet) => (
          <Button
            size="small"
            theme="borderless"
            onClick={() =>
              setSearchParams({
                tab: 'evaluation-sets',
                id: getSetId(record),
              })
            }
          >
            查看详情
          </Button>
        ),
      },
    ],
    [navigate, spaceId, setSearchParams, userMap],
  );

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">评估集</div>
            <div className="text-sm text-gray-500 mt-1">
              管理 Loop Evaluation Sets
            </div>
          </div>
          <Space>
            <Button
              icon={<IconCozRefresh />}
              onClick={fetchSets}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<IconCozPlus />}
              onClick={() => setCreateVisible(true)}
            >
              新建评估集
            </Button>
          </Space>
        </div>
      </Layout.Header>

      <Layout.Content>
        {loading && sets.length === 0 ? (
          <div className="py-16 text-center">
            <Spin />
          </div>
        ) : (
          <Table
            tableProps={{
              loading,
              dataSource: sets,
              columns,
              rowKey: getSetId,
              pagination: false,
              empty: (
                <div className="text-center py-8 text-gray-500">暂无评估集</div>
              ),
            }}
          />
        )}
      </Layout.Content>

      <Modal
        title="新建评估集"
        visible={createVisible}
        onCancel={() => setCreateVisible(false)}
        footer={null}
        style={{ width: 520 }}
      >
        <Form
          layout="vertical"
          getFormApi={(api: FormApi) => {
            formApiRef.current = api;
          }}
        >
          <Form.Input
            field="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
            placeholder="请输入评估集名称"
          />
          <Form.Input
            field="description"
            label="描述"
            placeholder="请输入描述"
          />
          <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
            <Button onClick={() => setCreateVisible(false)}>取消</Button>
            <Button type="primary" loading={creating} onClick={handleCreate}>
              创建
            </Button>
          </div>
        </Form>
      </Modal>
    </Layout>
  );
};

export { Page as Component };
export default Page;
