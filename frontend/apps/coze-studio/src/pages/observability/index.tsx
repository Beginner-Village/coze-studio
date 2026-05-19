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
/* eslint-disable curly, max-lines, @coze-arch/max-line-per-function */

import React, { useEffect, useState, useCallback, useMemo, useRef } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';

// Eager imports of evaluation sub-pages (chunk splitting via lazy was unreliable
// in our rsbuild setup — see notes in commit feat(observability) follow-up).
import EvaluationSetsPage from './evaluation-sets/index';
import EvaluatorsPage from './evaluators/index';
import ExperimentsPage from './experiments/index';
import {
  Button,
  Modal,
  Select,
  Spin,
  Toast,
} from '@coze-arch/coze-design';
import {
  IconCozRefresh,
  IconCozCopy,
} from '@coze-arch/coze-design/icons';
import copy from 'copy-to-clipboard';

import type { OutputSpan } from '@coze-arch/idl/stone_cozeloop_observability_api';

import { listSpans } from './api';
import {
  exportTracesToDataset,
  listEvaluationSets,
  type EvaluationSet,
} from './loop-eval-api';
import { TraceDetail } from './trace-detail';

// 时间范围预设
const TIME_RANGES = [
  { label: '过去 1 小时', value: 60 * 60 * 1000 },
  { label: '过去 6 小时', value: 6 * 60 * 60 * 1000 },
  { label: '过去 1 天', value: 24 * 60 * 60 * 1000 },
  { label: '过去 3 天', value: 3 * 24 * 60 * 60 * 1000 },
  { label: '过去 7 天', value: 7 * 24 * 60 * 60 * 1000 },
  { label: '过去 30 天', value: 30 * 24 * 60 * 60 * 1000 },
];

const SPAN_LIST_TYPES = [
  { label: 'Root Span', value: 'root' },
  { label: 'All Span', value: 'all' },
];

const SPAN_TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: 'Agent', value: 'Agent' },
  { label: 'Workflow', value: 'Workflow' },
];

const SPAN_TYPE_COLORS: Record<string, string> = {
  Agent: '#165dff',
  Workflow: '#0fc6c2',
};

function formatDurationMs(durationStr: string): string {
  const ms = Number(durationStr);
  if (isNaN(ms)) return '-';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}min`;
}

function latencyColor(durationStr: string): string {
  const ms = Number(durationStr);
  if (isNaN(ms)) return '#86909c';
  if (ms < 500) return '#00b365';
  if (ms < 2000) return '#ff7d00';
  return '#f53f3f';
}

function truncateContent(text: string, maxLen = 120): string {
  if (!text) return '';
  try {
    const parsed = JSON.parse(text);
    const str = typeof parsed === 'string' ? parsed : JSON.stringify(parsed);
    return str.length <= maxLen ? str : str.slice(0, maxLen) + '...';
  } catch {
    return text.length <= maxLen ? text : text.slice(0, maxLen) + '...';
  }
}

function formatStartTime(startedAt: string): string {
  if (!startedAt) return '-';
  const ms = Number(startedAt);
  if (isNaN(ms)) return '-';
  const d = new Date(ms);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

// ── 小组件 ──

const StatusIcon: React.FC<{ code: number }> = ({ code }) => (
  <span
    style={{
      display: 'inline-block',
      width: 8,
      height: 8,
      borderRadius: '50%',
      backgroundColor: code === 0 ? '#00b365' : '#f53f3f',
    }}
  />
);

const CopyableId: React.FC<{ id: string }> = ({ id }) => {
  const display = id ? id.slice(0, 10) + '...' : '-';
  return (
    <span className="flex items-center gap-1" style={{ fontFamily: 'monospace', fontSize: 13 }}>
      {display}
      <span
        style={{ cursor: 'pointer', opacity: 0.5, fontSize: 12 }}
        onClick={e => {
          e.stopPropagation();
          copy(id);
        }}
      >
        <IconCozCopy style={{ width: 14, height: 14 }} />
      </span>
    </span>
  );
};

const SpanTypeTag: React.FC<{ type: string }> = ({ type }) => {
  if (!type) return <span style={{ color: '#c9cdd4' }}>-</span>;
  const color = SPAN_TYPE_COLORS[type] || '#86909c';
  return (
    <span style={{
      display: 'inline-block',
      padding: '1px 6px',
      borderRadius: 3,
      fontSize: 12,
      lineHeight: '18px',
      color,
      background: `${color}14`,
      border: `1px solid ${color}33`,
      fontWeight: 500,
    }}>
      {type}
    </span>
  );
};

const ContentCell: React.FC<{ text: string }> = ({ text }) => {
  if (!text) return <span style={{ color: '#c9cdd4' }}>-</span>;
  return (
    <div
      style={{
        background: 'var(--coz-bg-tag-info-normal, #e8f3ff)',
        borderRadius: 4,
        padding: '2px 6px',
        fontSize: 13,
        lineHeight: '20px',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        maxWidth: '100%',
        color: '#1d2129',
      }}
      title={text}
    >
      {truncateContent(text)}
    </div>
  );
};

const LatencyCell: React.FC<{ duration: string }> = ({ duration }) => {
  const color = latencyColor(duration);
  return (
    <span className="flex items-center gap-1">
      <span
        style={{
          display: 'inline-block',
          width: 6,
          height: 6,
          borderRadius: '50%',
          backgroundColor: color,
        }}
      />
      <span style={{ color, fontSize: 13, fontWeight: 500 }}>
        {formatDurationMs(duration)}
      </span>
    </span>
  );
};

const ROW_HEIGHT = 42;

// ── 主页面 ──

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();

  const [spans, setSpans] = useState<OutputSpan[]>([]);
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState(7 * 24 * 60 * 60 * 1000);
  const [spanListType, setSpanListType] = useState('root');
  const [spanTypeFilter, setSpanTypeFilter] = useState('');
  const [pageToken, setPageToken] = useState<string>('');
  const [hasMore, setHasMore] = useState(false);
  const [selectedSpan, setSelectedSpan] = useState<OutputSpan | null>(null);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [exportVisible, setExportVisible] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [evalSets, setEvalSets] = useState<EvaluationSet[]>([]);
  const [selectedEvalSetId, setSelectedEvalSetId] = useState('');
  const tableRef = useRef<HTMLDivElement>(null);

  const fetchSpans = useCallback(
    async (append = false) => {
      if (!spaceId) return;
      setLoading(true);
      try {
        const now = Date.now();

        // 构建筛选条件
        const filters = spanTypeFilter
          ? {
              filter_fields: [
                { field_name: 'span_type', values: [spanTypeFilter], query_type: 'eq' },
              ],
            }
          : undefined;

        const res = await listSpans({
          workspace_id: spaceId,
          start_time: String(now - timeRange),
          end_time: String(now),
          page_size: 50,
          page_token: append ? pageToken : undefined,
          span_list_type: spanListType === 'root' ? undefined : spanListType,
          filters,
          order_bys: [{ field: 'started_at', is_asc: false }],
        });
        setSpans(prev => (append ? [...prev, ...(res.spans || [])] : (res.spans || [])));
        setPageToken(res.next_page_token);
        setHasMore(res.has_more);
      } catch (err) {
        console.error('[Observability] Failed to fetch spans:', err);
      } finally {
        setLoading(false);
      }
    },
    [spaceId, timeRange, spanListType, spanTypeFilter, pageToken],
  );

  useEffect(() => {
    setPageToken('');
    setSelectedSpan(null);
    fetchSpans(false);
  }, [spaceId, timeRange, spanListType, spanTypeFilter]);

  const handleRefresh = useCallback(() => {
    setPageToken('');
    fetchSpans(false);
  }, [fetchSpans]);

  const displayedTraceIds = useMemo(
    () => Array.from(new Set(spans.map(span => span.trace_id).filter(Boolean))),
    [spans],
  );

  const openExportModal = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    setExportVisible(true);
    try {
      const res = await listEvaluationSets({
        workspace_id: Number(spaceId) as unknown as string,
        page_size: 20,
        page_number: 1,
      });
      const sets = res.evaluation_sets || [];
      setEvalSets(sets);
      setSelectedEvalSetId(String(sets[0]?.evaluation_set_id || sets[0]?.id || ''));
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估集失败'));
    }
  }, [spaceId]);

  const handleExport = useCallback(async () => {
    if (!spaceId || displayedTraceIds.length === 0) {
      return;
    }
    setExporting(true);
    try {
      await exportTracesToDataset({
        workspace_id: Number(spaceId) as unknown as string,
        trace_ids: displayedTraceIds,
        evaluation_set_id: selectedEvalSetId || undefined,
      });
      Toast.success('导出成功');
      setExportVisible(false);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '导出失败'));
    } finally {
      setExporting(false);
    }
  }, [displayedTraceIds, selectedEvalSetId, spaceId]);

  // 滚动加载更多
  const handleScroll = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      if (!hasMore || loading) return;
      const el = e.currentTarget;
      if (el.scrollHeight - el.scrollTop - el.clientHeight < 150) {
        fetchSpans(true);
      }
    },
    [hasMore, loading, fetchSpans],
  );

  const handleRowClick = useCallback(
    (record: OutputSpan, index: number) => {
      setSelectedSpan(record);
      setSelectedIndex(index);
    },
    [],
  );

  // 上一条/下一条导航
  const handlePrev = useCallback(() => {
    if (selectedIndex > 0) {
      setSelectedIndex(selectedIndex - 1);
      setSelectedSpan(spans[selectedIndex - 1]);
    }
  }, [selectedIndex, spans]);

  const handleNext = useCallback(() => {
    if (selectedIndex < spans.length - 1) {
      setSelectedIndex(selectedIndex + 1);
      setSelectedSpan(spans[selectedIndex + 1]);
    }
  }, [selectedIndex, spans]);

  return (
    <div className="h-full w-full flex flex-col overflow-hidden" style={{ minWidth: 980 }}>
      {/* 页面头部 */}
      <div style={{ padding: '20px 24px 0' }}>
        <div style={{ fontSize: 20, fontWeight: 600, marginBottom: 16 }}>Trace</div>

        {/* 筛选栏 */}
        <div className="flex items-center gap-2" style={{ marginBottom: 16 }}>
          <Select
            value={timeRange}
            onChange={v => {
              setTimeRange(v as number);
              setPageToken('');
            }}
            style={{ width: 144 }}
            optionList={TIME_RANGES.map(r => ({ label: r.label, value: r.value }))}
            size="small"
          />
          <Select
            value={spanListType}
            onChange={v => {
              setSpanListType(v as string);
              setPageToken('');
            }}
            style={{ width: 128 }}
            optionList={SPAN_LIST_TYPES.map(r => ({ label: r.label, value: r.value }))}
            size="small"
          />
          <Select
            value={spanTypeFilter}
            onChange={v => {
              setSpanTypeFilter(v as string);
              setPageToken('');
            }}
            style={{ width: 128 }}
            optionList={SPAN_TYPE_OPTIONS.map(r => ({ label: r.label, value: r.value }))}
            size="small"
          />

          <div className="flex-1" />

          <span style={{ fontSize: 13, color: '#86909c' }}>
            {spans.length} 条{hasMore ? '+' : ''}
          </span>

          <Button
            size="small"
            disabled={displayedTraceIds.length === 0}
            onClick={openExportModal}
          >
            导出到评估集
          </Button>

          <Button
            theme="borderless"
            icon={<IconCozRefresh />}
            onClick={handleRefresh}
            loading={loading}
            size="small"
          />
        </div>
      </div>

      {/* 表格 */}
      <div
        ref={tableRef}
        className="flex-1 overflow-auto"
        style={{ padding: '0 24px 16px' }}
        onScroll={handleScroll}
      >
        {/* 表头 */}
        <div
          className="flex items-center"
          style={{
            height: 40,
            borderBottom: '1px solid var(--coz-stroke-default, #e5e6e8)',
            fontSize: 12,
            color: '#86909c',
            fontWeight: 500,
            position: 'sticky',
            top: 0,
            background: 'var(--coz-bg-body, #fff)',
            zIndex: 10,
          }}
        >
          <div style={{ width: 48, flexShrink: 0, textAlign: 'center' }} />
          <div style={{ width: 150, flexShrink: 0 }}>Name</div>
          <div style={{ width: 80, flexShrink: 0 }}>Type</div>
          <div style={{ width: 140, flexShrink: 0 }}>TraceID</div>
          <div style={{ flex: 1, minWidth: 160 }}>Input</div>
          <div style={{ flex: 1, minWidth: 160 }}>Output</div>
          <div style={{ width: 100, flexShrink: 0, paddingLeft: 12 }}>Latency</div>
          <div style={{ width: 140, flexShrink: 0, paddingLeft: 12 }}>Start Time</div>
        </div>

        {/* 数据行 */}
        {loading && spans.length === 0 ? (
          <div className="flex items-center justify-center" style={{ height: 200 }}>
            <Spin />
          </div>
        ) : spans.length === 0 ? (
          <div
            className="flex items-center justify-center"
            style={{ height: 200, color: '#86909c', fontSize: 14 }}
          >
            暂无 Trace 数据
          </div>
        ) : (
          spans.map((span, index) => (
            <div
              key={span.span_id || index}
              className="flex items-center"
              onClick={() => handleRowClick(span, index)}
              style={{
                height: ROW_HEIGHT,
                cursor: 'pointer',
                borderBottom: '1px solid var(--coz-stroke-default, #f2f3f5)',
                background:
                  selectedSpan?.span_id === span.span_id
                    ? 'var(--coz-bg-tag-info-normal, #f2f8ff)'
                    : undefined,
                transition: 'background 0.15s',
              }}
              onMouseEnter={e => {
                if (selectedSpan?.span_id !== span.span_id) {
                  e.currentTarget.style.background = 'var(--coz-bg-hover, #f7f8fa)';
                }
              }}
              onMouseLeave={e => {
                if (selectedSpan?.span_id !== span.span_id) {
                  e.currentTarget.style.background = '';
                }
              }}
            >
              <div style={{ width: 48, flexShrink: 0, textAlign: 'center' }}>
                <StatusIcon code={span.status_code} />
              </div>
              <div style={{ width: 150, flexShrink: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 13, fontWeight: 500, color: '#1d2129' }}>
                {span.span_name || '-'}
              </div>
              <div style={{ width: 80, flexShrink: 0 }}>
                <SpanTypeTag type={span.span_type} />
              </div>
              <div style={{ width: 140, flexShrink: 0 }}>
                <CopyableId id={span.trace_id} />
              </div>
              <div style={{ flex: 1, minWidth: 160, paddingRight: 8, overflow: 'hidden' }}>
                <ContentCell text={span.input} />
              </div>
              <div style={{ flex: 1, minWidth: 160, paddingRight: 8, overflow: 'hidden' }}>
                <ContentCell text={span.output} />
              </div>
              <div style={{ width: 100, flexShrink: 0, paddingLeft: 12 }}>
                <LatencyCell duration={span.duration} />
              </div>
              <div style={{ width: 140, flexShrink: 0, paddingLeft: 12, fontSize: 13, color: '#4e5969' }}>
                {formatStartTime(span.started_at)}
              </div>
            </div>
          ))
        )}

        {/* 加载更多指示器 */}
        {loading && spans.length > 0 && (
          <div className="flex items-center justify-center" style={{ height: 48 }}>
            <Spin size="small" />
          </div>
        )}

        {/* 无更多数据 */}
        {!loading && !hasMore && spans.length > 0 && (
          <div style={{ textAlign: 'center', padding: '12px 0', color: '#c9cdd4', fontSize: 12 }}>
            — 没有更多数据 —
          </div>
        )}
      </div>

      {/* 详情面板 */}
      {selectedSpan && spaceId && (
        <TraceDetail
          span={selectedSpan}
          spaceId={spaceId}
          timeRange={timeRange}
          onClose={() => {
            setSelectedSpan(null);
            setSelectedIndex(-1);
          }}
          onPrev={selectedIndex > 0 ? handlePrev : undefined}
          onNext={selectedIndex < spans.length - 1 ? handleNext : undefined}
        />
      )}

      <Modal
        title="导出到评估集"
        visible={exportVisible}
        onCancel={() => setExportVisible(false)}
        footer={null}
        style={{ width: 480 }}
      >
        <div className="flex flex-col gap-4">
          <div>
            <div className="text-sm text-gray-700 mb-2">评估集</div>
            <Select
              value={selectedEvalSetId}
              onChange={v => setSelectedEvalSetId(v as string)}
              style={{ width: '100%' }}
              optionList={evalSets.map(set => ({
                label: set.name || set.evaluation_set_id || set.id || '-',
                value: String(set.evaluation_set_id || set.id || ''),
              }))}
              placeholder="请选择评估集"
            />
          </div>
          <div className="text-sm text-gray-500">
            将导出当前列表中的 {displayedTraceIds.length} 个 Trace。
          </div>
          <div
            className="rounded-[4px] border p-2 text-xs text-gray-600"
            style={{ maxHeight: 120, overflow: 'auto', fontFamily: 'monospace' }}
          >
            {displayedTraceIds.map(traceId => (
              <div key={traceId}>{traceId}</div>
            ))}
          </div>
          <div className="flex justify-end gap-3 pt-4 border-t">
            <Button onClick={() => setExportVisible(false)}>取消</Button>
            <Button
              type="primary"
              loading={exporting}
              disabled={!selectedEvalSetId || displayedTraceIds.length === 0}
              onClick={handleExport}
            >
              导出
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

// Tab dispatcher: render the right sub-page based on ?tab query.
// This avoids the (unreliable in our rsbuild) lazy-chunk route registration
// and keeps everything in the existing /observability chunk.
const ObservabilityRoute: React.FC = () => {
  const [params] = useSearchParams();
  const tab = params.get('tab');
  if (tab === 'evaluation-sets') return <EvaluationSetsPage />;
  if (tab === 'evaluators') return <EvaluatorsPage />;
  if (tab === 'experiments') return <ExperimentsPage />;
  return <Page />;
};

export { ObservabilityRoute as Component };
export default ObservabilityRoute;
