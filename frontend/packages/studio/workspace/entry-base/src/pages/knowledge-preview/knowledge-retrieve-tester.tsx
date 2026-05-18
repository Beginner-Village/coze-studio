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

import { useMemo, useState } from 'react';

type RetrieveTestHit = {
  slice_id: string;
  knowledge_id: string;
  knowledge_name?: string;
  document_id: string;
  document_name: string;
  document_uri?: string;
  document_url?: string;
  content: string;
  answer?: string;
  score: number;
};

type RetrieveTestResponse = {
  code: number;
  msg: string;
  data?: {
    total: number;
    hits: RetrieveTestHit[];
  };
};

export const KnowledgeRetrieveTester = ({
  datasetID,
}: {
  datasetID: string;
}) => {
  const [query, setQuery] = useState('');
  const [topK, setTopK] = useState('5');
  const [searchType, setSearchType] = useState<'semantic' | 'hybrid' | 'fulltext'>('semantic');
  const [collapsed, setCollapsed] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [hits, setHits] = useState<RetrieveTestHit[]>([]);

  const canSubmit = useMemo(
    () => query.trim().length > 0 && datasetID.trim().length > 0,
    [datasetID, query],
  );

  const runTest = async () => {
    if (!canSubmit || loading) {
      return;
    }
    setLoading(true);
    setError('');
    try {
      const parsedTopK = Number.parseInt(topK, 10);
      const body = {
        dataset_id: datasetID,
        query: query.trim(),
        top_k: Number.isFinite(parsedTopK) && parsedTopK > 0 ? parsedTopK : 5,
        search_type: searchType,
      };

      const resp = await fetch('/api/knowledge/retrieve_test', {
        method: 'POST',
        credentials: 'include',
        headers: {
          'content-type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      const json = (await resp.json()) as RetrieveTestResponse;
      if (!resp.ok || json.code !== 0) {
        throw new Error(json.msg || '检索测试失败');
      }
      const sortedHits = [...(json.data?.hits ?? [])].sort(
        (a, b) => b.score - a.score,
      );
      setHits(sortedHits);
    } catch (e) {
      setHits([]);
      setError(e instanceof Error ? e.message : '检索测试失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        position: 'fixed',
        right: 16,
        bottom: 16,
        width: collapsed ? 220 : 420,
        maxHeight: '70vh',
        overflow: 'hidden',
        zIndex: 30,
        background: '#fff',
        border: '1px solid #e5e6eb',
        borderRadius: 12,
        boxShadow: '0 8px 20px rgba(15, 23, 42, 0.12)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 8,
          padding: '10px 12px',
          borderBottom: collapsed ? 'none' : '1px solid #f2f3f5',
        }}
      >
        <strong
          style={{
            fontSize: 13,
            lineHeight: '20px',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          知识库检索测试
        </strong>
        <button
          onClick={() => setCollapsed(v => !v)}
          style={{
            border: 'none',
            background: '#f2f3f5',
            borderRadius: 6,
            padding: '4px 8px',
            cursor: 'pointer',
            fontSize: 12,
            lineHeight: '18px',
            whiteSpace: 'nowrap',
            flexShrink: 0,
          }}
        >
          {collapsed ? '展开' : '收起'}
        </button>
      </div>
      {!collapsed ? (
        <div style={{ padding: 12 }}>
          <div style={{ marginBottom: 8, fontSize: 12, color: '#4e5969' }}>
            Dataset: {datasetID || '-'}
          </div>
          <textarea
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="输入要测试的 query"
            rows={3}
            style={{
              width: '100%',
              resize: 'vertical',
              border: '1px solid #d9dce1',
              borderRadius: 8,
              padding: 8,
              outline: 'none',
              fontSize: 13,
              marginBottom: 8,
            }}
          />
          <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
            <input
              value={topK}
              onChange={e => setTopK(e.target.value)}
              placeholder="TopK"
              style={{
                width: 86,
                border: '1px solid #d9dce1',
                borderRadius: 8,
                padding: '6px 8px',
                fontSize: 12,
              }}
            />
            <select
              value={searchType}
              onChange={e =>
                setSearchType(e.target.value as 'semantic' | 'hybrid' | 'fulltext')
              }
              style={{
                flex: 1,
                border: '1px solid #d9dce1',
                borderRadius: 8,
                padding: '6px 8px',
                fontSize: 12,
              }}
            >
              <option value="semantic">semantic</option>
              <option value="hybrid">hybrid</option>
              <option value="fulltext">fulltext</option>
            </select>
            <button
              disabled={!canSubmit || loading}
              onClick={runTest}
              style={{
                border: 'none',
                borderRadius: 8,
                padding: '0 12px',
                fontSize: 12,
                cursor: canSubmit && !loading ? 'pointer' : 'not-allowed',
                background: canSubmit && !loading ? '#1d4ed8' : '#9ca3af',
                color: '#fff',
              }}
            >
              {loading ? '测试中...' : '开始测试'}
            </button>
          </div>

          {error ? (
            <div
              style={{
                marginBottom: 8,
                background: '#fef2f2',
                color: '#b91c1c',
                borderRadius: 8,
                padding: '8px 10px',
                fontSize: 12,
              }}
            >
              {error}
            </div>
          ) : null}

          <div
            style={{
              maxHeight: '42vh',
              overflowY: 'auto',
              borderTop: '1px solid #f2f3f5',
              paddingTop: 8,
            }}
          >
            {hits.length === 0 ? (
              <div style={{ color: '#86909c', fontSize: 12 }}>暂无命中结果</div>
            ) : (
              hits.map(item => (
                <div
                  key={item.slice_id}
                  style={{
                    border: '1px solid #eef0f3',
                    borderRadius: 8,
                    padding: 10,
                    marginBottom: 8,
                    background: '#fcfcfd',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      alignItems: 'flex-start',
                      justifyContent: 'space-between',
                      gap: 8,
                      marginBottom: 6,
                    }}
                  >
                    <div
                      style={{
                        fontSize: 13,
                        fontWeight: 600,
                        color: '#0f172a',
                        wordBreak: 'break-all',
                        lineHeight: '18px',
                      }}
                      title={item.document_name || '未命名文档'}
                    >
                      📄 {item.document_name || '未命名文档'}
                    </div>
                    <div
                      style={{
                        flexShrink: 0,
                        fontSize: 11,
                        fontWeight: 600,
                        color: '#1d4ed8',
                        background: '#dbeafe',
                        borderRadius: 6,
                        padding: '2px 6px',
                        lineHeight: '16px',
                      }}
                    >
                      {item.score.toFixed(4)}
                    </div>
                  </div>
                  <div style={{ fontSize: 11, color: '#6b7280', marginBottom: 4, wordBreak: 'break-all' }}>
                    {item.knowledge_name ? <span>知识库: {item.knowledge_name} · </span> : null}
                    doc_id: {item.document_id} · slice_id: {item.slice_id}
                  </div>
                  {item.document_uri ? (
                    <div
                      style={{
                        fontSize: 11,
                        color: '#475569',
                        background: '#f1f5f9',
                        borderRadius: 4,
                        padding: '2px 6px',
                        marginBottom: 6,
                        wordBreak: 'break-all',
                        fontFamily:
                          'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                      }}
                      title={item.document_uri}
                    >
                      {item.document_uri}
                    </div>
                  ) : null}
                  <div
                    style={{
                      fontSize: 13,
                      color: '#1f2937',
                      whiteSpace: 'pre-wrap',
                      borderTop: '1px dashed #e5e7eb',
                      paddingTop: 6,
                    }}
                  >
                    {item.content || '-'}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
};
