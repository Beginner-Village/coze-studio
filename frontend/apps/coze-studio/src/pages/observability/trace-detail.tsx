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

import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { Spin, Tag, Tabs } from '@coze-arch/coze-design';
import { IconCozCopy } from '@coze-arch/coze-design/icons';
import copy from 'copy-to-clipboard';

import type { OutputSpan } from '@coze-arch/idl/stone_cozeloop_observability_api';
import {
  TraceTree,
  TraceFlameThread,
  MessagePanel,
  ObservationModules,
  spans2SpanNodes,
  type SpanNode,
  type TraceFrontendSpan,
  type MessagePanelProps,
} from '@coze-workflow/test-run-trace/observation-components';

import { getTrace } from './api';
import { convertOutputSpans } from './converter';

const I18N_MAPPING: MessagePanelProps['i18nMapping'] = {
  [ObservationModules.INPUT]: { title: 'Input' },
  [ObservationModules.OUTPUT]: { title: 'Output' },
  [ObservationModules.RUN_TREE]: { title: 'Run Tree' },
  [ObservationModules.FLAME_THREAD]: { title: 'Flame Graph' },
  [ObservationModules.META_TAGS]: { title: 'Tags' },
  [ObservationModules.SPAN_DETAIL]: { title: 'Detail' },
};

function formatDurationMs(durationStr: string): string {
  const ms = Number(durationStr);
  if (isNaN(ms)) return '-';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}min`;
}

function formatDurationFromMs(ms: number | string): string {
  const v = Number(ms);
  if (isNaN(v) || v === 0) return '-';
  if (v < 1000) return `${Math.round(v)}ms`;
  if (v < 60000) return `${(v / 1000).toFixed(1)}s`;
  return `${(v / 60000).toFixed(1)}min`;
}

function formatFullTimestamp(ms: number): string {
  if (!ms) return '-';
  const d = new Date(ms);
  if (isNaN(d.getTime())) return '-';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function latencyColor(durationStr: string): string {
  const ms = Number(durationStr);
  if (isNaN(ms)) return '#86909c';
  if (ms < 500) return '#00b365';
  if (ms < 2000) return '#ff7d00';
  return '#f53f3f';
}

// ── Props ──

interface TraceDetailProps {
  span: OutputSpan;
  spaceId: string;
  timeRange: number;
  onClose: () => void;
  onPrev?: () => void;
  onNext?: () => void;
}

// ── Main Component ──

export const TraceDetail: React.FC<TraceDetailProps> = ({
  span,
  spaceId,
  timeRange,
  onClose,
  onPrev,
  onNext,
}) => {
  const [loading, setLoading] = useState(false);
  const [frontendSpans, setFrontendSpans] = useState<TraceFrontendSpan[]>([]);
  const [selectedSpan, setSelectedSpan] = useState<SpanNode | null>(null);
  const [graphTab, setGraphTab] = useState('tree');

  const fetchTraceDetail = useCallback(async () => {
    if (!span.trace_id) return;
    setLoading(true);
    try {
      const now = Date.now();
      const res = await getTrace({
        workspace_id: spaceId,
        trace_id: span.trace_id,
        start_time: String(now - timeRange),
        end_time: String(now),
      });
      const converted = convertOutputSpans(res.spans || []);
      setFrontendSpans(converted);
    } catch (err) {
      console.error('[TraceDetail] Failed to fetch trace:', err);
    } finally {
      setLoading(false);
    }
  }, [span.trace_id, spaceId, timeRange]);

  useEffect(() => {
    fetchTraceDetail();
  }, [fetchTraceDetail]);

  // Escape 键关闭
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const { roots } = useMemo(
    () => spans2SpanNodes(frontendSpans),
    [frontendSpans],
  );

  // 自动选中第一个 root span
  useEffect(() => {
    if (roots.length > 0 && !selectedSpan) {
      setSelectedSpan(roots[0]);
    }
  }, [roots, selectedSpan]);

  const handleTreeSelect = useCallback(
    (v: { node: { extra?: { spanNode?: SpanNode } } }) => {
      if (v.node.extra?.spanNode) {
        setSelectedSpan(v.node.extra.spanNode);
      }
    },
    [],
  );

  const handleFlameClick = useCallback(
    (v: { extra?: { span?: SpanNode } }) => {
      if (v.extra?.span) {
        setSelectedSpan(v.extra.span as SpanNode);
      }
    },
    [],
  );

  const lColor = latencyColor(span.duration);

  return (
    <div style={overlayStyle}>
      {/* ── Header ── */}
      <div style={headerStyle}>
        {/* 左：Trace 信息 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, minWidth: 0 }}>
          <span style={traceNameStyle}>
            {span.span_name || 'Unknown Trace'}
          </span>
          <Tag
            color={span.status_code === 0 ? 'green' : 'red'}
            size="small"
          >
            {span.status_code === 0 ? 'Success' : 'Error'}
          </Tag>
          <span style={metricsTagStyle}>
            <span style={{ color: lColor, fontWeight: 500 }}>
              Latency: {formatDurationMs(span.duration)}
            </span>
            <span style={metricsDivider} />
            <span>Tokens: 0</span>
          </span>
        </div>

        {/* 右：操作按钮 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <IconBtn
            title="Copy Trace ID"
            onClick={() => copy(span.trace_id)}
          >
            <IconCozCopy style={{ width: 16, height: 16 }} />
          </IconBtn>

          {onPrev && (
            <IconBtn title="Previous Trace" onClick={onPrev}>
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                <path d="M10 3l-5 5 5 5" stroke="currentColor" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </IconBtn>
          )}
          {onNext && (
            <IconBtn title="Next Trace" onClick={onNext}>
              <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                <path d="M6 3l5 5-5 5" stroke="currentColor" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </IconBtn>
          )}

          <IconBtn title="Close (Esc)" onClick={onClose} style={{ marginLeft: 8 }}>
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </IconBtn>
        </div>
      </div>

      {/* ── Content: 水平分割 ── */}
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {/* 左侧：Span 树 / 火焰图 */}
        <div style={leftPanelStyle}>
          <div style={{ flexShrink: 0, borderBottom: '1px solid var(--coz-stroke-default, #e5e6e8)' }}>
            <Tabs
              activeKey={graphTab}
              onChange={setGraphTab}
              size="small"
              style={{ padding: '0 12px' }}
            >
              <Tabs.TabPane tab="Tree" itemKey="tree" />
              <Tabs.TabPane tab="Flame Graph" itemKey="flame" />
            </Tabs>
          </div>

          <div style={{ flex: 1, overflow: 'auto', padding: '12px 0 0' }}>
            {loading ? (
              <div style={emptyCenter}><Spin /></div>
            ) : frontendSpans.length === 0 ? (
              <div style={{ ...emptyCenter, color: '#86909c', fontSize: 14 }}>
                暂无 Span 数据
              </div>
            ) : (
              <>
                {graphTab === 'tree' && (
                  <TraceTree
                    spans={roots}
                    selectedSpanId={selectedSpan?.span_id}
                    onSelect={handleTreeSelect}
                  />
                )}
                {graphTab === 'flame' && (
                  <TraceFlameThread
                    spans={roots}
                    selectedSpanId={selectedSpan?.span_id}
                    onClick={handleFlameClick}
                  />
                )}
              </>
            )}
          </div>
        </div>

        {/* 右侧：Span 详情 */}
        <div style={rightPanelStyle}>
          {selectedSpan ? (
            <SpanDetailPanel span={selectedSpan} />
          ) : (
            <div style={{ ...emptyCenter, color: '#86909c', fontSize: 14 }}>
              点击左侧 Span 查看详情
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// ── IconBtn 组件 ──

const IconBtn: React.FC<{
  title?: string;
  onClick?: () => void;
  style?: React.CSSProperties;
  children: React.ReactNode;
}> = ({ title, onClick, style, children }) => {
  const [hovered, setHovered] = useState(false);
  return (
    <button
      title={title}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        ...iconBtnBase,
        background: hovered ? 'var(--coz-bg-hover, #f2f3f5)' : 'none',
        ...style,
      }}
    >
      {children}
    </button>
  );
};

// ── Span Detail Panel ──

const SpanDetailPanel: React.FC<{ span: SpanNode }> = ({ span }) => (
  <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
    {/* Span 标题头 */}
    <div style={spanHeaderStyle}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
        <span style={{ fontSize: 16, fontWeight: 600, color: '#1d2129' }}>
          {span.alias_name || span.name || '-'}
        </span>
        {span.type && (
          <Tag color="primary" size="small">{span.type}</Tag>
        )}
        <Tag
          color={span.status_code === 0 ? 'green' : span.status_code === 1 ? 'red' : undefined}
          size="small"
        >
          {span.status_code === 0 ? 'OK' : span.status_code === 1 ? 'Error' : String(span.status_code ?? '-')}
        </Tag>
        {span.duration !== undefined && (
          <span style={{ fontSize: 12, color: '#86909c' }}>
            {formatDurationFromMs(span.duration)}
          </span>
        )}
      </div>
    </div>

    {/* 内容区域：主内容 + 字段栏 */}
    <div style={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 0 }}>
      {/* 主内容：Input / Output */}
      <div style={mainContentStyle}>
        {span.input?.content && (
          <div style={{ marginBottom: 16 }}>
            <MessagePanel
              content={span.input.content}
              category={ObservationModules.INPUT}
              i18nMapping={I18N_MAPPING}
              jsonViewerProps={{ displayDataTypes: false }}
            />
          </div>
        )}
        {span.output?.content && (
          <div style={{ marginBottom: 16 }}>
            <MessagePanel
              content={span.output.content}
              category={ObservationModules.OUTPUT}
              i18nMapping={I18N_MAPPING}
              jsonViewerProps={{ displayDataTypes: false }}
            />
          </div>
        )}
        {!span.input?.content && !span.output?.content && (
          <div style={{ color: '#86909c', fontSize: 14, padding: '60px 0', textAlign: 'center' }}>
            无输入/输出数据
          </div>
        )}
      </div>

      {/* 右侧字段列表 */}
      <div style={fieldSidebarStyle}>
        <FieldItem label="Status">
          <span style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
          }}>
            <span style={{
              display: 'inline-block',
              width: 6,
              height: 6,
              borderRadius: '50%',
              backgroundColor: span.status_code === 0 ? '#00b365' : '#f53f3f',
            }} />
            {span.status_code === 0 ? 'OK' : span.status_code === 1 ? 'Error' : String(span.status_code ?? '-')}
          </span>
        </FieldItem>
        <FieldItem label="Duration">
          {formatDurationFromMs(span.duration || 0)}
        </FieldItem>
        <FieldItem label="Started At">
          {formatFullTimestamp(Number(span.start_time) || 0)}
        </FieldItem>
        <FieldItem label="Ended At">
          {formatFullTimestamp((Number(span.start_time) || 0) + (Number(span.duration) || 0))}
        </FieldItem>
        <CopyableField label="Trace ID" value={span.trace_id || '-'} />
        <CopyableField label="Span ID" value={span.span_id || '-'} />
        {span.parent_id && (
          <CopyableField label="Parent ID" value={span.parent_id} />
        )}

        {/* Custom Tags */}
        {span.tags && span.tags.length > 0 && (
          <div style={{ marginTop: 16, paddingTop: 12, borderTop: '1px solid var(--coz-stroke-default, #e5e6e8)' }}>
            <div style={{ color: '#86909c', fontSize: 12, marginBottom: 8, fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.5px' }}>
              Tags
            </div>
            {span.tags.map((tag, i) => (
              <div key={i} style={{ marginBottom: 8, fontSize: 13 }}>
                <div style={{ color: '#86909c', fontSize: 12, marginBottom: 2 }}>{tag.key}</div>
                <div style={{ color: '#1d2129', wordBreak: 'break-all', lineHeight: '18px' }}>
                  {tag.value?.v_str || '-'}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  </div>
);

// ── FieldItem 组件 ──

const FieldItem: React.FC<{
  label: string;
  children: React.ReactNode;
}> = ({ label, children }) => (
  <div style={{ marginBottom: 12 }}>
    <div style={{ color: '#86909c', fontSize: 12, marginBottom: 2, lineHeight: '18px' }}>{label}</div>
    <div style={{ color: '#1d2129', fontSize: 13, lineHeight: '20px' }}>{children}</div>
  </div>
);

// ── CopyableField 组件 ──

const CopyableField: React.FC<{
  label: string;
  value: string;
}> = ({ label, value }) => (
  <div style={{ marginBottom: 12 }}>
    <div style={{ color: '#86909c', fontSize: 12, marginBottom: 2, lineHeight: '18px' }}>{label}</div>
    <div style={{
      display: 'flex',
      alignItems: 'center',
      gap: 4,
      color: '#1d2129',
      fontSize: 12,
      fontFamily: 'monospace',
      lineHeight: '20px',
    }}>
      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {value}
      </span>
      {value !== '-' && (
        <span
          style={{ cursor: 'pointer', opacity: 0.4, flexShrink: 0 }}
          onClick={() => copy(value)}
          title={`Copy ${label}`}
        >
          <IconCozCopy style={{ width: 12, height: 12 }} />
        </span>
      )}
    </div>
  </div>
);

// ── Styles ──

const overlayStyle: React.CSSProperties = {
  position: 'fixed',
  top: 0,
  right: 0,
  bottom: 0,
  left: 0,
  zIndex: 1000,
  display: 'flex',
  flexDirection: 'column',
  background: 'var(--coz-bg-body, #fff)',
};

const headerStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  height: 56,
  padding: '0 24px',
  borderBottom: '1px solid var(--coz-stroke-default, #e5e6e8)',
  flexShrink: 0,
};

const traceNameStyle: React.CSSProperties = {
  fontSize: 16,
  fontWeight: 600,
  maxWidth: 300,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  color: '#1d2129',
};

const metricsTagStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 8,
  padding: '2px 10px',
  borderRadius: 4,
  fontSize: 12,
  color: 'var(--coz-fg-secondary, #4e5969)',
  background: 'var(--coz-bg-tag-primary-normal, #f2f3f5)',
  lineHeight: '20px',
};

const metricsDivider: React.CSSProperties = {
  display: 'inline-block',
  width: 1,
  height: 12,
  background: 'var(--coz-fg-dim, #c9cdd4)',
};

const iconBtnBase: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 28,
  height: 28,
  cursor: 'pointer',
  borderRadius: 6,
  border: 'none',
  color: '#4e5969',
  padding: 0,
  transition: 'background 0.15s',
};

const leftPanelStyle: React.CSSProperties = {
  width: '26%',
  minWidth: 300,
  borderRight: '1px solid var(--coz-stroke-default, #e5e6e8)',
  display: 'flex',
  flexDirection: 'column',
  flexShrink: 0,
};

const rightPanelStyle: React.CSSProperties = {
  flex: 1,
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
  minWidth: 0,
};

const emptyCenter: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  height: '100%',
  width: '100%',
};

const spanHeaderStyle: React.CSSProperties = {
  padding: '16px 20px 12px',
  borderBottom: '1px solid var(--coz-stroke-default, #e5e6e8)',
  flexShrink: 0,
  position: 'sticky',
  top: 0,
  background: 'var(--coz-bg-body, #fff)',
  zIndex: 10,
};

const mainContentStyle: React.CSSProperties = {
  flex: 1,
  overflow: 'auto',
  padding: '16px 20px',
  minWidth: 0,
};

const fieldSidebarStyle: React.CSSProperties = {
  width: 220,
  flexShrink: 0,
  borderLeft: '1px solid var(--coz-stroke-default, #e5e6e8)',
  padding: '16px 16px',
  overflow: 'auto',
};

export default TraceDetail;
