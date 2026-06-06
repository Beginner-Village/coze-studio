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

import { useParams, useSearchParams } from 'react-router-dom';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import {
  Button,
  Form,
  Layout,
  Modal,
  Space,
  Spin,
  Tag,
  Toast,
} from '@coze-arch/coze-design';

import {
  batchDebugEvaluators,
  getEvaluator,
  runEvaluator,
  type Evaluator,
  type EvaluatorInputData,
  type LoopUser,
} from '../loop-eval-api';

interface RunFormValues {
  input_data: string;
}

interface FormApi {
  validate: () => Promise<RunFormValues>;
}

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
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
  if (
    normalized === '1' ||
    normalized.includes('prompt') ||
    normalized.includes('llm')
  ) {
    return 'LLM';
  }
  if (normalized === '2' || normalized.includes('code')) {
    return 'Code';
  }
  if (normalized === '4' || normalized.includes('agent')) {
    return 'Agent';
  }
  return type ? String(type) : '-';
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

function parseInputData(value: string): EvaluatorInputData {
  const parsed: unknown = JSON.parse(value);
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('请输入 JSON Object');
  }
  return parsed as EvaluatorInputData;
}

function parseBatchInputData(value: string): EvaluatorInputData[] {
  const parsed: unknown = JSON.parse(value);
  if (!Array.isArray(parsed) || parsed.length === 0) {
    throw new Error('请输入非空 JSON 数组，每个元素为一组 input_data');
  }
  return parsed.map((item, index) => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      throw new Error(`第 ${index + 1} 项必须是 JSON Object`);
    }
    return item as EvaluatorInputData;
  });
}

const Page: React.FC = () => {
  const { space_id: routeSpaceId } = useParams<{ space_id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const evaluatorId = searchParams.get('id') || '';
  const spaceId = searchParams.get('space_id') || routeSpaceId || '';
  const formApiRef = useRef<FormApi | null>(null);
  const batchFormApiRef = useRef<FormApi | null>(null);
  const [evaluator, setEvaluator] = useState<Evaluator | null>(null);
  const [loading, setLoading] = useState(false);
  const [runVisible, setRunVisible] = useState(false);
  const [running, setRunning] = useState(false);
  const [runResult, setRunResult] = useState<unknown>(null);
  const [batchVisible, setBatchVisible] = useState(false);
  const [batchRunning, setBatchRunning] = useState(false);
  const [batchResult, setBatchResult] = useState<unknown>(null);

  const fetchDetail = useCallback(async () => {
    if (!spaceId || !evaluatorId) {
      return;
    }
    setLoading(true);
    try {
      const res = await getEvaluator({
        space_id: spaceId,
        evaluator_id: evaluatorId,
      });
      setEvaluator(res.evaluator || null);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估器详情失败'));
    } finally {
      setLoading(false);
    }
  }, [evaluatorId, spaceId]);

  useEffect(() => {
    fetchDetail();
  }, [fetchDetail]);

  const backToList = () => {
    setSearchParams({ tab: 'evaluators' });
  };

  const handleRun = async () => {
    const evaluatorVersionId = evaluator?.current_version?.id;
    if (!spaceId || !evaluatorVersionId || !formApiRef.current) {
      Toast.error('缺少评估器版本 ID');
      return;
    }
    try {
      const values = await formApiRef.current.validate();
      setRunning(true);
      const res = await runEvaluator({
        workspace_id: spaceId,
        evaluator_version_id: evaluatorVersionId,
        input_data: parseInputData(values.input_data),
      });
      setRunResult(res.record || res);
      Toast.success('运行完成');
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '运行测试失败'));
    } finally {
      setRunning(false);
    }
  };

  const handleBatchRun = async () => {
    const evaluatorVersionId = evaluator?.current_version?.id;
    if (!spaceId || !evaluatorVersionId || !batchFormApiRef.current) {
      Toast.error('缺少评估器版本 ID');
      return;
    }
    try {
      const values = await batchFormApiRef.current.validate();
      const inputs = parseBatchInputData(values.input_data);
      setBatchRunning(true);
      const res = await batchDebugEvaluators({
        workspace_id: spaceId,
        evaluator_version_id: evaluatorVersionId,
        items: inputs.map((inputData, index) => ({
          id: String(index),
          input_data: inputData,
        })),
      });
      setBatchResult(res.results || res.records || res);
      Toast.success('批量调试完成');
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '批量调试失败'));
    } finally {
      setBatchRunning(false);
    }
  };

  const version =
    evaluator?.current_version?.version ||
    evaluator?.latest_version ||
    evaluator?.version ||
    '-';
  const creator = evaluator?.creator || evaluator?.base_info?.created_by;
  const createdAt = evaluator?.created_at || evaluator?.base_info?.created_at;

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">
              {evaluator?.name || '评估器详情'}
            </div>
            <div className="text-sm text-gray-500 mt-1">
              {evaluator?.description || `ID: ${evaluatorId || '-'}`}
            </div>
          </div>
          <Space>
            <Button disabled={!evaluator} onClick={() => setRunVisible(true)}>
              运行测试
            </Button>
            <Button disabled={!evaluator} onClick={() => setBatchVisible(true)}>
              批量调试
            </Button>
            <Button onClick={backToList}>返回列表</Button>
          </Space>
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
              <Meta label="ID" value={evaluatorId || '-'} />
              <div>
                <div className="text-xs text-gray-500 mb-1">类型</div>
                <Tag>{formatEvaluatorType(evaluator?.evaluator_type)}</Tag>
              </div>
              <Meta label="版本" value={version} />
              <Meta label="创建人" value={formatCreator(creator)} />
              <Meta label="创建时间" value={formatTime(createdAt)} />
            </div>
            <pre className="rounded-[6px] border p-4 overflow-auto text-sm bg-gray-50">
              {stringify(evaluator?.current_version?.evaluator_content)}
            </pre>
          </div>
        )}
      </Layout.Content>

      <Modal
        title="运行测试"
        visible={runVisible}
        onCancel={() => setRunVisible(false)}
        footer={null}
        style={{ width: 720 }}
      >
        <Form
          layout="vertical"
          initValues={{
            input_data: JSON.stringify(
              {
                input_fields: {},
                evaluate_target_output_fields: {},
              },
              null,
              2,
            ),
          }}
          getFormApi={(api: FormApi) => {
            formApiRef.current = api;
          }}
        >
          <Form.TextArea
            field="input_data"
            label="Input Data"
            rules={[{ required: true, message: '请输入 JSON' }]}
            autosize={{ minRows: 8, maxRows: 14 }}
          />
        </Form>
        {runResult ? (
          <pre className="rounded-[6px] border p-4 overflow-auto text-sm bg-gray-50">
            {stringify(runResult)}
          </pre>
        ) : null}
        <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
          <Button onClick={() => setRunVisible(false)}>取消</Button>
          <Button type="primary" loading={running} onClick={handleRun}>
            运行
          </Button>
        </div>
      </Modal>

      <Modal
        title="批量调试"
        visible={batchVisible}
        onCancel={() => setBatchVisible(false)}
        footer={null}
        style={{ width: 720 }}
      >
        <div className="text-xs text-gray-500 mb-2">
          输入 JSON 数组，每个元素为一组 input_data，将批量调用评估器。
        </div>
        <Form
          layout="vertical"
          initValues={{
            input_data: JSON.stringify(
              [
                {
                  input_fields: {},
                  evaluate_target_output_fields: {},
                },
              ],
              null,
              2,
            ),
          }}
          getFormApi={(api: FormApi) => {
            batchFormApiRef.current = api;
          }}
        >
          <Form.TextArea
            field="input_data"
            label="Sample Inputs (JSON Array)"
            rules={[{ required: true, message: '请输入 JSON 数组' }]}
            autosize={{ minRows: 8, maxRows: 16 }}
          />
        </Form>
        {batchResult ? (
          <pre className="rounded-[6px] border p-4 overflow-auto text-sm bg-gray-50">
            {stringify(batchResult)}
          </pre>
        ) : null}
        <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
          <Button onClick={() => setBatchVisible(false)}>取消</Button>
          <Button
            type="primary"
            loading={batchRunning}
            onClick={handleBatchRun}
          >
            批量运行
          </Button>
        </div>
      </Modal>
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
