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
/* eslint-disable max-lines -- 单页承载完整批测编排,拆分收益低 */
/**
 * 「批量测试」独立页：选智能体/工作流 → 下载模板 → 上传 CSV → 批量运行 → 逐行结果表。
 * 复用已验证的 loop 评测后端(评测集/版本/实验/逐行结果)编排，但对用户隐藏其复杂度。
 */
import { useParams } from 'react-router-dom';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import {
  IntelligenceType,
  IntelligenceStatus,
} from '@coze-arch/idl/intelligence_api';
import {
  Button,
  Layout,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Toast,
} from '@coze-arch/coze-design';
import { intelligenceApi, workflowApi } from '@coze-arch/bot-api';

import { parseXlsx, buildXlsxBlob } from './observability/xlsx-lite';
import {
  createEvaluationSet,
  createEvaluationSetItems,
  commitEvaluationSetVersion,
  createExperiment,
  getExperiment,
  listExperimentResults,
  type ExperimentResultRow,
} from './observability/loop-eval-api';

type TargetType = 'workflow' | 'bot';
const INPUT_COLUMN = 'input';

interface TargetOption {
  label: string;
  value: string;
}

interface ResultRow {
  index: number;
  input: string;
  output: string;
  latencyMs?: number;
  error?: string;
}

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

// 简易 CSV 解析(支持引号/转义逗号)。首行表头，仅取第一列作为输入。
function parseCsv(text: string): { columns: string[]; rows: string[][] } {
  const lines = text
    .replace(/\r\n/g, '\n')
    .replace(/^\ufeff/, '')
    .split('\n')
    .filter(l => l.trim().length > 0);
  const parseLine = (line: string): string[] => {
    const out: string[] = [];
    let cur = '';
    let inQ = false;
    for (let i = 0; i < line.length; i++) {
      const c = line[i];
      if (inQ) {
        if (c === '"') {
          if (line[i + 1] === '"') {
            cur += '"';
            i++;
          } else {
            inQ = false;
          }
        } else {
          cur += c;
        }
      } else if (c === '"') {
        inQ = true;
      } else if (c === ',') {
        out.push(cur);
        cur = '';
      } else {
        cur += c;
      }
    }
    out.push(cur);
    return out.map(s => s.trim());
  };
  if (lines.length === 0) {
    return { columns: [], rows: [] };
  }
  return {
    columns: parseLine(lines[0]),
    rows: lines.slice(1).map(parseLine),
  };
}

// 模板行:首行表头(单列 input),后跟两条示例。CSV 与 Excel 共用同一数据。
function templateRows(targetType: TargetType): string[][] {
  const example =
    targetType === 'bot'
      ? ['你好，请用一句话自我介绍', '今天天气怎么样']
      : ['我有哪些银行卡', '查询我的余额'];
  return [[INPUT_COLUMN], ...example.map(v => [v])];
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function downloadCsvTemplate(targetType: TargetType): void {
  const csv = `${templateRows(targetType)
    .map(r => r.join(','))
    .join('\n')}\n`;
  downloadBlob(
    new Blob([`\ufeff${csv}`], { type: 'text/csv;charset=utf-8' }),
    `批量测试模板-${targetType}.csv`,
  );
}

function downloadExcelTemplate(targetType: TargetType): void {
  downloadBlob(
    buildXlsxBlob(templateRows(targetType)),
    `批量测试模板-${targetType}.xlsx`,
  );
}

// 首行为表头,其余为数据;仅取第一列非空值作为输入。
function extractInputs(matrix: string[][]): string[] {
  if (matrix.length === 0) {
    return [];
  }
  return matrix
    .slice(1)
    .map(r => (r[0] ?? '').trim())
    .filter(v => v.length > 0);
}

// 从逐行结果里提取输入/目标输出/耗时(结构较深，防御式取值)。
function extractResultRow(item: ExperimentResultRow, idx: number): ResultRow {
  const it = item as Record<string, unknown>;
  const turn = (
    it.turn_results as Array<Record<string, unknown>> | undefined
  )?.[0];
  const payload = (
    turn?.experiment_results as Array<Record<string, unknown>> | undefined
  )?.[0]?.payload as Record<string, unknown> | undefined;
  const rec = (payload?.target_output as Record<string, unknown> | undefined)
    ?.eval_target_record as Record<string, unknown> | undefined;
  const inputFields = (
    rec?.eval_target_input_data as Record<string, unknown> | undefined
  )?.input_fields as Record<string, { text?: string }> | undefined;
  const input =
    inputFields?.[INPUT_COLUMN]?.text ??
    Object.values(inputFields || {})[0]?.text ??
    '';
  const outData = rec?.eval_target_output_data as
    | Record<string, unknown>
    | undefined;
  const output =
    (outData?.output_fields as Record<string, { text?: string }> | undefined)
      ?.actual_output?.text ?? '';
  const latencyMs = outData?.time_consuming_ms as number | undefined;
  const errObj = outData?.eval_target_run_error as
    | { code?: number; message?: string }
    | undefined;
  const error =
    errObj && (errObj.code || errObj.message)
      ? errObj.message || `执行失败（错误码 ${errObj.code}）`
      : undefined;
  return { index: idx + 1, input, output, latencyMs, error };
}

const sleep = (ms: number) => new Promise<void>(r => setTimeout(r, ms));

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const workspaceId = spaceId || '';

  const [targetType, setTargetType] = useState<TargetType>('workflow');
  const [targetId, setTargetId] = useState('');
  const [targetOptions, setTargetOptions] = useState<TargetOption[]>([]);
  const [targetLoading, setTargetLoading] = useState(false);
  const [rows, setRows] = useState<string[]>([]);
  const [fileName, setFileName] = useState('');
  const [running, setRunning] = useState(false);
  const [progress, setProgress] = useState('');
  const [results, setResults] = useState<ResultRow[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const fetchTargets = useCallback(async () => {
    if (!workspaceId) {
      return;
    }
    setTargetLoading(true);
    setTargetOptions([]);
    setTargetId('');
    try {
      if (targetType === 'workflow') {
        const res = await workflowApi.GetWorkFlowList({
          space_id: workspaceId,
          page: 1,
          size: 100,
        });
        setTargetOptions(
          (res.data?.workflow_list || [])
            .map(wf => ({
              label: wf.name || wf.workflow_id || '-',
              value: String(wf.workflow_id || ''),
            }))
            .filter(o => o.value),
        );
      } else {
        const res = await intelligenceApi.GetDraftIntelligenceList({
          space_id: workspaceId,
          types: [IntelligenceType.Bot],
          size: 100,
          // status 必传:缺省时接口返回空。取「使用中/封禁/迁移失败」与 develop 页一致。
          status: [
            IntelligenceStatus.Using,
            IntelligenceStatus.Banned,
            IntelligenceStatus.MoveFailed,
          ],
        });
        setTargetOptions(
          (res.data?.intelligences || [])
            .map(item => ({
              label: item.basic_info?.name || item.basic_info?.id || '-',
              value: String(item.basic_info?.id || ''),
            }))
            .filter(o => o.value),
        );
      }
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评测对象失败'));
    } finally {
      setTargetLoading(false);
    }
  }, [workspaceId, targetType]);

  useEffect(() => {
    fetchTargets();
  }, [fetchTargets]);

  const onUpload = async (file: File): Promise<void> => {
    try {
      let matrix: string[][];
      if (/\.xlsx$/i.test(file.name)) {
        matrix = await parseXlsx(await file.arrayBuffer());
      } else {
        const parsed = parseCsv(await file.text());
        matrix = [parsed.columns, ...parsed.rows];
      }
      const inputs = extractInputs(matrix);
      if (inputs.length === 0) {
        Toast.error(
          '没有解析到有效输入行（首行为表头 input，从第二行开始为数据）',
        );
        return;
      }
      setRows(inputs);
      setFileName(file.name);
      setResults([]);
      Toast.success(`已解析 ${inputs.length} 条输入`);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '解析文件失败'));
    }
  };

  const runBatch = async () => {
    if (!workspaceId) {
      Toast.error('缺少 workspace_id');
      return;
    }
    if (!targetId) {
      Toast.error('请选择评测对象');
      return;
    }
    if (rows.length === 0) {
      Toast.error('请先上传输入 CSV');
      return;
    }
    setRunning(true);
    setResults([]);
    const ts = Date.now();
    try {
      setProgress('创建数据集…');
      const setResp = await createEvaluationSet({
        workspace_id: workspaceId,
        name: `批量测试-${ts}`,
        description: 'batch test',
        evaluation_set_schema: {
          field_schemas: [
            {
              key: INPUT_COLUMN,
              name: INPUT_COLUMN,
              content_type: 'Text',
              default_display_format: 1,
            },
          ],
        },
      });
      const evalSetRec = setResp.evaluation_set as
        | Record<string, unknown>
        | undefined;
      const evalSetId = String(
        evalSetRec?.id ||
          evalSetRec?.evaluation_set_id ||
          (setResp as Record<string, unknown>).evaluation_set_id ||
          '',
      );
      if (!evalSetId) {
        throw new Error('创建数据集失败：无 id');
      }

      setProgress(`写入 ${rows.length} 条输入…`);
      await createEvaluationSetItems({
        workspace_id: workspaceId,
        evaluation_set_id: evalSetId,
        rows: rows.map(v => ({ [INPUT_COLUMN]: v })),
      });

      setProgress('提交数据集版本…');
      const verResp = await commitEvaluationSetVersion({
        workspace_id: workspaceId,
        evaluation_set_id: evalSetId,
        version: '1.0.0',
        description: 'batch test',
      });
      const versionId = String(verResp.id || '');
      if (!versionId) {
        throw new Error('提交版本失败：无 id');
      }

      setProgress('创建并运行批量测试…');
      const exp = await createExperiment({
        workspace_id: workspaceId,
        name: `批量测试-${targetType}-${ts}`,
        eval_set_id: evalSetId,
        eval_set_version_id: versionId,
        evaluator_version_ids: [],
        create_eval_target_param: {
          eval_target_type: targetType === 'bot' ? 1 : 4,
          source_target_id: targetId,
        },
        target_field_mapping: {
          from_eval_set: [
            { field_name: INPUT_COLUMN, from_field_name: INPUT_COLUMN },
          ],
        },
        expt_type: 1,
      });
      const expResp = exp as Record<string, unknown>;
      const exptId = String(
        (expResp.experiment as Record<string, unknown> | undefined)?.id ||
          expResp.experiment_id ||
          expResp.expt_id ||
          '',
      );
      if (!exptId) {
        throw new Error('创建实验失败：无 id');
      }

      // 轮询直到完成(status 11 成功 / 12 失败)
      for (let i = 0; i < 60; i++) {
        await sleep(3000);
        const detail = await getExperiment({
          workspace_id: workspaceId,
          experiment_id: exptId,
        });
        const st = String(detail.experiment?.status ?? '');
        setProgress(`运行中…(${i * 3}s)`);
        if (st === '11' || st === '12' || st === '13' || st === '14') {
          break;
        }
      }

      setProgress('拉取逐行结果…');
      const res = await listExperimentResults({
        workspace_id: workspaceId,
        experiment_id: exptId,
        page_number: 1,
        page_size: 200,
      });
      const table = (res.item_results || []).map(extractResultRow);
      if (table.length === 0) {
        Toast.error('运行完成，但未取到逐行结果（可能仍在计算，可稍后刷新）');
      }
      setResults(table);
      setProgress('');
      Toast.success(`批量测试完成，共 ${table.length} 行`);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '批量测试失败'));
      setProgress('');
    } finally {
      setRunning(false);
    }
  };

  const columns = [
    { title: '#', dataIndex: 'index', key: 'index', width: 60 },
    {
      title: '输入',
      dataIndex: 'input',
      key: 'input',
      width: 320,
      render: (v: string) => (
        <div className="whitespace-pre-wrap break-words text-sm">
          {v || '-'}
        </div>
      ),
    },
    {
      title: '目标输出',
      dataIndex: 'output',
      key: 'output',
      render: (v: string, r: ResultRow) =>
        r.error ? (
          <Tag color="red">{r.error}</Tag>
        ) : (
          <div className="whitespace-pre-wrap break-words text-sm">
            {v || '-'}
          </div>
        ),
    },
    {
      title: '耗时',
      dataIndex: 'latencyMs',
      key: 'latencyMs',
      width: 100,
      render: (v?: number) => (v ? `${v} ms` : '-'),
    },
  ];

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div>
          <div className="font-[500] text-[20px]">批量测试</div>
          <div className="text-sm text-gray-500 mt-1">
            选择智能体 / 工作流，上传一批输入，批量运行并查看逐行输出
          </div>
        </div>
      </Layout.Header>

      <Layout.Content>
        <div className="flex flex-col gap-5" style={{ maxWidth: 1100 }}>
          <div className="grid grid-cols-2 gap-4 rounded-[8px] border p-4">
            <div>
              <div className="text-sm text-gray-700 mb-1">评测对象类型</div>
              <Select
                value={targetType}
                onChange={v => setTargetType(v as TargetType)}
                optionList={[
                  { label: '工作流 (Workflow)', value: 'workflow' },
                  { label: '智能体 (Bot)', value: 'bot' },
                ]}
                style={{ width: '100%' }}
              />
            </div>
            <div>
              <div className="text-sm text-gray-700 mb-1">选择评测对象</div>
              <Select
                filter
                value={targetId || undefined}
                onChange={v => setTargetId(String(v ?? ''))}
                optionList={targetOptions}
                loading={targetLoading}
                placeholder="搜索并选择"
                style={{ width: '100%' }}
              />
            </div>
          </div>

          <div className="rounded-[8px] border p-4">
            <div className="text-sm text-gray-700 mb-2">
              上传输入数据（CSV / Excel）
            </div>
            <Space>
              <Button onClick={() => downloadCsvTemplate(targetType)}>
                下载 CSV 模板
              </Button>
              <Button onClick={() => downloadExcelTemplate(targetType)}>
                下载 Excel 模板
              </Button>
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,.xlsx"
                style={{ display: 'none' }}
                onChange={e => {
                  const f = e.target.files?.[0];
                  if (f) {
                    void onUpload(f);
                  }
                  e.target.value = '';
                }}
              />
              <Button
                type="primary"
                onClick={() => fileInputRef.current?.click()}
              >
                上传文件
              </Button>
              {fileName ? (
                <span className="text-sm text-gray-500">
                  {fileName}（{rows.length} 条）
                </span>
              ) : (
                <span className="text-sm text-gray-400">
                  模板仅一列 input，每行一条输入
                </span>
              )}
            </Space>
          </div>

          <div className="flex items-center gap-3">
            <Button
              type="primary"
              size="large"
              loading={running}
              disabled={!targetId || rows.length === 0}
              onClick={runBatch}
            >
              开始批量测试
            </Button>
            {progress ? (
              <span className="text-sm text-gray-500">
                <Spin size="small" /> {progress}
              </span>
            ) : null}
          </div>

          {results.length > 0 ? (
            <div className="rounded-[8px] border">
              <Table
                tableProps={{
                  dataSource: results,
                  columns,
                  rowKey: 'index',
                  pagination: false,
                }}
              />
            </div>
          ) : null}
        </div>
      </Layout.Content>
    </Layout>
  );
};

export { Page as Component };
export default Page;
