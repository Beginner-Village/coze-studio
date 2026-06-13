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

/* eslint-disable @typescript-eslint/no-magic-numbers -- status codes / column widths are self-evident literals */
import { useParams } from 'react-router-dom';
import { useCallback, useEffect, useMemo, useState, type FC } from 'react';

import {
  Table,
  Input,
  Select,
  DatePicker,
  Tag,
  Typography,
  Pagination,
  Empty,
} from '@coze-arch/coze-design';
import { SpaceApi, type OperationLogItem } from '@coze-arch/bot-api';

const { Title, Text } = Typography;

// 仅 Owner/Admin 可查；后端对普通成员返回该业务码。
const ERR_NO_PERMISSION = 112100001;
const NO_PERMISSION_MSG = '无权限查看：仅空间所有者/管理员可访问操作日志';
const DEFAULT_PAGE_SIZE = 20;
const STATUS_FAIL = 2;

// 资源类型枚举 → 中文(与后端 resource_type 对齐)
const RESOURCE_TYPE_MAP: Record<number, string> = {
  2: '空间',
  3: '应用/变量',
  4: '智能体',
  5: '插件',
  6: '工作流',
  7: '知识库',
  17: '提示词',
  23: '数据库',
};

// 动作枚举 → 中文
const ACTION_MAP: Record<string, string> = {
  create: '创建',
  update: '更新',
  delete: '删除',
  publish: '发布',
  copy: '复制',
  invite: '邀请',
  remove: '移除',
  update_role: '修改角色',
  transfer: '转让',
};

const RESOURCE_TYPE_OPTIONS = Object.entries(RESOURCE_TYPE_MAP).map(
  ([value, label]) => ({ value: Number(value), label }),
);

const ACTION_OPTIONS = Object.entries(ACTION_MAP).map(([value, label]) => ({
  value,
  label,
}));

const formatTime = (ms: number): string => {
  if (!ms) {
    return '-';
  }
  const d = new Date(ms);
  const pad = (n: number) => `${n}`.padStart(2, '0');
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  );
};

interface Filters {
  operatorId?: string;
  resourceType?: number;
  action?: string;
  startTime?: number;
  endTime?: number;
  keyword?: string;
}

interface ApiLikeError {
  code?: number;
  msg?: string;
  message?: string;
  response?: { data?: { code?: number; msg?: string } };
}

const resolveError = (e: ApiLikeError): string => {
  const code = e?.code ?? e?.response?.data?.code;
  const msg = e?.msg ?? e?.response?.data?.msg ?? e?.message;
  return code === ERR_NO_PERMISSION ? NO_PERMISSION_MSG : msg || '查询失败';
};

const columns = [
  {
    title: '时间',
    dataIndex: 'created_at',
    width: 180,
    render: (v: number) => formatTime(v),
  },
  {
    title: '操作人',
    dataIndex: 'operator_name',
    width: 160,
    render: (_: string, record: OperationLogItem) =>
      record.operator_name || record.operator_id || '-',
  },
  {
    title: '模块',
    dataIndex: 'resource_type',
    width: 100,
    render: (_: number, record: OperationLogItem) =>
      RESOURCE_TYPE_MAP[record.resource_type] || record.module || '-',
  },
  {
    title: '资源',
    dataIndex: 'resource_name',
    width: 180,
    render: (v: string) => v || '-',
  },
  {
    title: '动作',
    dataIndex: 'action',
    render: (_: string, record: OperationLogItem) => (
      <div>
        <Tag color="primary" size="small">
          {ACTION_MAP[record.action] || record.action || '-'}
        </Tag>
        {record.description ? (
          <Text style={{ marginLeft: 8 }} type="tertiary">
            {record.description}
          </Text>
        ) : null}
      </div>
    ),
  },
  {
    title: '状态',
    dataIndex: 'status',
    width: 90,
    render: (v: number) =>
      v === STATUS_FAIL ? (
        <Tag color="red">失败</Tag>
      ) : (
        <Tag color="green">成功</Tag>
      ),
  },
];

interface FilterBarProps {
  draft: Filters;
  setDraft: (updater: (prev: Filters) => Filters) => void;
  onSearch: () => void;
  onReset: () => void;
}

const FilterBar: FC<FilterBarProps> = ({
  draft,
  setDraft,
  onSearch,
  onReset,
}) => (
  <div
    style={{
      padding: '16px 24px 8px',
      display: 'flex',
      flexWrap: 'wrap',
      gap: 12,
      alignItems: 'center',
    }}
  >
    <Input
      style={{ width: 180 }}
      placeholder="操作人 ID"
      value={draft.operatorId}
      onChange={v => setDraft(prev => ({ ...prev, operatorId: v }))}
    />
    <Select
      style={{ width: 160 }}
      placeholder="资源类型"
      showClear
      optionList={RESOURCE_TYPE_OPTIONS}
      value={draft.resourceType}
      onChange={v => setDraft(prev => ({ ...prev, resourceType: v as number }))}
    />
    <Select
      style={{ width: 140 }}
      placeholder="动作"
      showClear
      optionList={ACTION_OPTIONS}
      value={draft.action}
      onChange={v => setDraft(prev => ({ ...prev, action: v as string }))}
    />
    <DatePicker
      type="dateTimeRange"
      style={{ width: 360 }}
      placeholder={['开始时间', '结束时间']}
      onChange={date => {
        const range = date as Date[] | null;
        const valid = Array.isArray(range) && range.length === 2;
        setDraft(prev => ({
          ...prev,
          startTime: valid ? range[0]?.getTime() : undefined,
          endTime: valid ? range[1]?.getTime() : undefined,
        }));
      }}
    />
    <Input
      style={{ width: 200 }}
      placeholder="关键词(描述/资源名)"
      value={draft.keyword}
      onChange={v => setDraft(prev => ({ ...prev, keyword: v }))}
    />
    <span style={{ display: 'inline-flex', gap: 8 }}>
      <button
        type="button"
        onClick={onSearch}
        style={{
          padding: '6px 16px',
          borderRadius: 6,
          border: 'none',
          cursor: 'pointer',
          background: 'var(--coz-mg-hglt, #4e40e5)',
          color: '#fff',
        }}
      >
        查询
      </button>
      <button
        type="button"
        onClick={onReset}
        style={{
          padding: '6px 16px',
          borderRadius: 6,
          border: '1px solid var(--coz-stroke-primary, #d0d3d6)',
          cursor: 'pointer',
          background: 'transparent',
        }}
      >
        重置
      </button>
    </span>
  </div>
);

const Page = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();

  const [logs, setLogs] = useState<OperationLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<Filters>({});
  // 受控的筛选输入(点击查询/分页时才真正应用)
  const [draft, setDraft] = useState<Filters>({});

  const fetchLogs = useCallback(
    async (nextPage: number, applied: Filters) => {
      if (!spaceId) {
        return;
      }
      setLoading(true);
      setErrorMsg('');
      try {
        const resp = await SpaceApi.listOperationLog({
          space_id: spaceId,
          operator_id: applied.operatorId || undefined,
          resource_type: applied.resourceType,
          action: applied.action || undefined,
          start_time: applied.startTime,
          end_time: applied.endTime,
          keyword: applied.keyword || undefined,
          page: nextPage,
          page_size: DEFAULT_PAGE_SIZE,
        });
        if (resp.code !== 0) {
          setLogs([]);
          setTotal(0);
          setErrorMsg(resolveError(resp));
          return;
        }
        setLogs(resp.logs || []);
        setTotal(resp.total || 0);
      } catch (e) {
        setLogs([]);
        setTotal(0);
        setErrorMsg(resolveError(e as ApiLikeError));
      } finally {
        setLoading(false);
      }
    },
    [spaceId],
  );

  useEffect(() => {
    fetchLogs(1, {});
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅在 spaceId 变化时初始化
  }, [spaceId]);

  const handleSearch = () => {
    setFilters(draft);
    setPage(1);
    fetchLogs(1, draft);
  };

  const handleReset = () => {
    setDraft({});
    setFilters({});
    setPage(1);
    fetchLogs(1, {});
  };

  const handlePageChange = (nextPage: number) => {
    setPage(nextPage);
    fetchLogs(nextPage, filters);
  };

  const tableProps = useMemo(
    () => ({
      loading,
      columns,
      dataSource: logs,
      rowKey: 'id',
      pagination: false as const,
      empty: <div className="text-center py-8 text-gray-500">暂无操作日志</div>,
    }),
    [loading, logs],
  );

  if (!spaceId) {
    return (
      <div className="py-16 text-center text-gray-500">缺少 space_id 参数</div>
    );
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '16px 24px 0' }}>
        <Title heading={4}>操作日志</Title>
      </div>

      <FilterBar
        draft={draft}
        setDraft={setDraft}
        onSearch={handleSearch}
        onReset={handleReset}
      />

      <div style={{ flex: 1, overflowY: 'auto', padding: '8px 24px 24px' }}>
        {errorMsg ? (
          <Empty
            title="无法加载操作日志"
            description={errorMsg}
            style={{ paddingTop: 80 }}
          />
        ) : (
          <>
            <Table tableProps={tableProps} />
            <div
              style={{
                display: 'flex',
                justifyContent: 'flex-end',
                marginTop: 16,
              }}
            >
              <Pagination
                total={total}
                currentPage={page}
                pageSize={DEFAULT_PAGE_SIZE}
                onPageChange={handlePageChange}
              />
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export { Page as Component };
export default Page;
