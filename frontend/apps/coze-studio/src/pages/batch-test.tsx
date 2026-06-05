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
/* eslint-disable @coze-arch/max-line-per-function -- BatchTestPage is a multi-step wizard; each step block is self-contained but co-located for clarity */
/* eslint-disable max-lines -- batch-test page includes helper types, sub-components and main wizard; will be split into a directory when feature stabilises */

import { useParams } from 'react-router-dom';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { I18n } from '@coze-arch/i18n';
import {
  Button,
  Input,
  Layout,
  Select,
  Space,
  Spin,
  Steps,
  Toast,
  Typography,
} from '@coze-arch/coze-design';

import {
  createExperiment,
  getExperiment,
  listEvaluationSets,
  listEvaluators,
  type EvaluationSet,
  type Evaluator,
  type Experiment,
} from './observability/loop-eval-api';

const { Title } = Typography;

const PAGE_SIZE = 50;

const STEP_TARGET = 0;
const STEP_EVAL_SET = 1;
const STEP_EVALUATORS = 2;
const STEP_CONFIRM = 3;

const STEP_TITLES = ['目标配置', '选择数据集', '选择评估器', '确认提交'];

const FieldLabel: React.FC<{
  label: string;
  required?: boolean;
  children: React.ReactNode;
}> = ({ label, required, children }) => (
  <div className="mb-4">
    <div className="text-sm text-gray-700 mb-1">
      {label}
      {required ? <span className="text-red-500 ml-1">*</span> : null}
    </div>
    {children}
  </div>
);

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function getEvaluationSetId(record: EvaluationSet): string {
  return String(record.evaluation_set_id || record.id || '');
}

function getEvaluatorVersionId(record: Evaluator): string {
  return String(record.current_version?.id || record.latest_version || '');
}

function getEvaluatorLabel(record: Evaluator): string {
  const version = record.latest_version ? ` / ${record.latest_version}` : '';
  return `${record.name || record.evaluator_id || record.id || '-'}${version}`;
}

type TargetType = 'bot' | 'workflow';

interface ExperimentResult {
  experiment_id: string;
  status?: string | number;
  progress?:
    | number
    | string
    | { total?: number; finished?: number; success?: number };
}

function validateStep(
  step: number,
  args: {
    name: string;
    targetId: string;
    targetVersionId: string;
    evalSetId: string;
    evaluatorVersionIds: string[];
  },
): string | null {
  const { name, targetId, targetVersionId, evalSetId, evaluatorVersionIds } =
    args;
  if (step === STEP_TARGET) {
    if (!name.trim()) {
      return '请输入批量测试名称';
    }
    if (!targetId.trim() && !targetVersionId.trim()) {
      return '请填写目标 ID 或版本 ID';
    }
  }
  if (step === STEP_EVAL_SET && !evalSetId) {
    return '请选择数据集';
  }
  if (step === STEP_EVALUATORS && evaluatorVersionIds.length === 0) {
    return '请至少选择一个评估器';
  }
  return null;
}

function formatProgress(
  p:
    | number
    | string
    | { total?: number; finished?: number; success?: number }
    | undefined,
): string {
  if (!p) {
    return '-';
  }
  if (typeof p === 'object') {
    const { total = 0, finished = 0, success = 0 } = p;
    return `${finished}/${total} (成功: ${success})`;
  }
  return String(p);
}

interface ConfirmStepProps {
  name: string;
  description: string;
  targetType: TargetType;
  targetId: string;
  targetVersionId: string;
  evalSetLabel: string;
  evaluatorLabels: string[];
}

const ConfirmStep: React.FC<ConfirmStepProps> = ({
  name,
  description,
  targetType,
  targetId,
  targetVersionId,
  evalSetLabel,
  evaluatorLabels,
}) => (
  <div className="space-y-3 text-sm">
    <div className="font-medium text-base mb-4">确认配置</div>
    <div className="grid grid-cols-[120px_1fr] gap-y-2">
      <span className="text-gray-500">测试名称：</span>
      <span>{name}</span>
      {description ? (
        <>
          <span className="text-gray-500">描述：</span>
          <span>{description}</span>
        </>
      ) : null}
      <span className="text-gray-500">目标类型：</span>
      <span>{targetType === 'workflow' ? '工作流' : '智能体'}</span>
      {targetId ? (
        <>
          <span className="text-gray-500">目标 ID：</span>
          <span className="font-mono text-xs">{targetId}</span>
        </>
      ) : null}
      {targetVersionId ? (
        <>
          <span className="text-gray-500">版本 ID：</span>
          <span className="font-mono text-xs">{targetVersionId}</span>
        </>
      ) : null}
      <span className="text-gray-500">数据集：</span>
      <span>{evalSetLabel}</span>
      <span className="text-gray-500">评估器：</span>
      <span>{evaluatorLabels.join('、') || '-'}</span>
    </div>
  </div>
);

interface ResultViewProps {
  result: ExperimentResult;
  pollingLoading: boolean;
  onRefresh: () => void;
  onReset: () => void;
}

const ResultView: React.FC<ResultViewProps> = ({
  result,
  pollingLoading,
  onRefresh,
  onReset,
}) => (
  <div style={{ maxWidth: 760 }}>
    <div className="mb-6 p-4 border rounded-lg bg-green-50">
      <div className="font-medium text-green-700 mb-2">批量测试已提交</div>
      <div className="text-sm text-gray-600 space-y-1">
        <div>
          <span className="font-medium">实验 ID：</span>
          {result.experiment_id || '-'}
        </div>
        <div>
          <span className="font-medium">状态：</span>
          {String(result.status || '-')}
        </div>
        <div>
          <span className="font-medium">进度：</span>
          {formatProgress(result.progress)}
        </div>
      </div>
    </div>
    <Space>
      <Button type="primary" loading={pollingLoading} onClick={onRefresh}>
        刷新结果
      </Button>
      <Button onClick={onReset}>新建测试</Button>
    </Space>
  </div>
);

const BatchTestPage: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();
  const workspaceId = spaceId || '';

  const [step, setStep] = useState(0);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [targetType, setTargetType] = useState<TargetType>('workflow');
  const [targetId, setTargetId] = useState('');
  const [targetVersionId, setTargetVersionId] = useState('');
  const [evalSetId, setEvalSetId] = useState('');
  const [evalSets, setEvalSets] = useState<EvaluationSet[]>([]);
  const [evalSetsLoading, setEvalSetsLoading] = useState(false);
  const [evaluatorVersionIds, setEvaluatorVersionIds] = useState<string[]>([]);
  const [evaluators, setEvaluators] = useState<Evaluator[]>([]);
  const [evaluatorsLoading, setEvaluatorsLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<ExperimentResult | null>(null);
  const [pollingLoading, setPollingLoading] = useState(false);

  const fetchEvalSets = useCallback(async () => {
    if (!workspaceId || evalSets.length > 0 || evalSetsLoading) {
      return;
    }
    setEvalSetsLoading(true);
    try {
      const res = await listEvaluationSets({
        workspace_id: workspaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      setEvalSets(res.evaluation_sets || []);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载数据集失败'));
    } finally {
      setEvalSetsLoading(false);
    }
  }, [workspaceId, evalSets.length, evalSetsLoading]);

  const fetchEvaluators = useCallback(async () => {
    if (!workspaceId || evaluators.length > 0 || evaluatorsLoading) {
      return;
    }
    setEvaluatorsLoading(true);
    try {
      const res = await listEvaluators({
        workspace_id: workspaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      setEvaluators(res.evaluators || []);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估器失败'));
    } finally {
      setEvaluatorsLoading(false);
    }
  }, [workspaceId, evaluators.length, evaluatorsLoading]);

  useEffect(() => {
    if (step === STEP_EVAL_SET) {
      fetchEvalSets();
    }
    if (step === STEP_EVALUATORS) {
      fetchEvaluators();
    }
  }, [fetchEvalSets, fetchEvaluators, step]);

  const evalSetOptions = useMemo(
    () =>
      evalSets.map(set => ({
        label: set.name || set.evaluation_set_id || set.id || '-',
        value: getEvaluationSetId(set),
      })),
    [evalSets],
  );

  const evaluatorOptions = useMemo(
    () =>
      evaluators
        .map(evaluator => ({
          label: getEvaluatorLabel(evaluator),
          value: getEvaluatorVersionId(evaluator),
        }))
        .filter(option => option.value),
    [evaluators],
  );

  const selectedEvalSetLabel = useMemo(() => {
    const found = evalSets.find(s => getEvaluationSetId(s) === evalSetId);
    return found?.name || evalSetId || '-';
  }, [evalSets, evalSetId]);

  const selectedEvaluatorLabels = useMemo(
    () =>
      evaluatorVersionIds.map(vid => {
        const found = evaluators.find(e => getEvaluatorVersionId(e) === vid);
        return found ? getEvaluatorLabel(found) : vid;
      }),
    [evaluators, evaluatorVersionIds],
  );

  const goNext = () => {
    const error = validateStep(step, {
      name,
      targetId,
      targetVersionId,
      evalSetId,
      evaluatorVersionIds,
    });
    if (error) {
      Toast.error(error);
      return;
    }
    setStep(prev => Math.min(prev + 1, STEP_TITLES.length - 1));
  };

  const handleSubmit = async () => {
    if (!workspaceId) {
      Toast.error('缺少 workspace_id');
      return;
    }
    setSubmitting(true);
    try {
      const resp = await createExperiment({
        workspace_id: workspaceId,
        name: name.trim(),
        description: description.trim() || undefined,
        eval_set_id: evalSetId,
        evaluator_version_ids: evaluatorVersionIds,
        target_id: targetId.trim() || undefined,
        target_version_id: targetVersionId.trim() || undefined,
      });
      const experimentId = resp.experiment_id || resp.expt_id || '';
      Toast.success('批量测试已提交');
      setResult({ experiment_id: experimentId, status: 'running' });
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '提交失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleRefreshResult = async () => {
    if (!result?.experiment_id || !workspaceId) {
      return;
    }
    setPollingLoading(true);
    try {
      const resp = await getExperiment({
        workspace_id: workspaceId,
        experiment_id: result.experiment_id,
      });
      const expt: Experiment = resp.experiment || {};
      setResult(prev =>
        prev ? { ...prev, status: expt.status, progress: expt.progress } : prev,
      );
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '刷新结果失败'));
    } finally {
      setPollingLoading(false);
    }
  };

  const handleReset = () => {
    setStep(0);
    setName('');
    setDescription('');
    setTargetType('workflow');
    setTargetId('');
    setTargetVersionId('');
    setEvalSetId('');
    setEvaluatorVersionIds([]);
    setResult(null);
  };

  if (!spaceId) {
    return (
      <Layout>
        <Layout.Content>
          <div className="py-16 text-center text-gray-500">
            {I18n.t('batch_test_missing_space_id', {}, '缺少 space_id 参数')}
          </div>
        </Layout.Content>
      </Layout>
    );
  }

  if (result) {
    return (
      <div
        style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}
      >
        <div style={{ padding: '16px 24px 0' }}>
          <Title heading={4}>
            {I18n.t('batch_test_title', {}, '猎鹰批量测试')}
          </Title>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
          <ResultView
            result={result}
            pollingLoading={pollingLoading}
            onRefresh={handleRefreshResult}
            onReset={handleReset}
          />
        </div>
      </div>
    );
  }

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '16px 24px 0' }}>
        <Title heading={4}>
          {I18n.t('batch_test_title', {}, '猎鹰批量测试')}
        </Title>
        <div className="text-sm text-gray-500 mt-1">
          {I18n.t(
            'batch_test_subtitle',
            {},
            '选择智能体/工作流目标，配置数据集和评估器，提交批量评测',
          )}
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }}>
        <div style={{ maxWidth: 760 }}>
          <Steps current={step} className="mb-8">
            {STEP_TITLES.map(title => (
              <Steps.Step key={title} title={title} />
            ))}
          </Steps>

          <div className="mt-8">
            {step === STEP_TARGET ? (
              <div>
                <FieldLabel
                  label={I18n.t('batch_test_name', {}, '测试名称')}
                  required
                >
                  <Input
                    value={name}
                    onChange={value => setName(String(value))}
                    placeholder="请输入批量测试名称"
                  />
                </FieldLabel>
                <FieldLabel label={I18n.t('batch_test_desc', {}, '描述')}>
                  <Input
                    value={description}
                    onChange={value => setDescription(String(value))}
                    placeholder="可选"
                  />
                </FieldLabel>
                <FieldLabel
                  label={I18n.t('batch_test_target_type', {}, '目标类型')}
                  required
                >
                  <Select
                    value={targetType}
                    onChange={value => setTargetType(value as TargetType)}
                    optionList={[
                      { label: '工作流 (Workflow)', value: 'workflow' },
                      { label: '智能体 (Bot)', value: 'bot' },
                    ]}
                    style={{ width: '100%' }}
                  />
                </FieldLabel>
                <FieldLabel
                  label={I18n.t('batch_test_target_id', {}, '目标 ID')}
                >
                  <Input
                    value={targetId}
                    onChange={value => setTargetId(String(value))}
                    placeholder={
                      targetType === 'workflow'
                        ? '工作流 ID（target_id）'
                        : '智能体 ID（target_id）'
                    }
                  />
                </FieldLabel>
                <FieldLabel
                  label={I18n.t(
                    'batch_test_target_version_id',
                    {},
                    '目标版本 ID（可选）',
                  )}
                >
                  <Input
                    value={targetVersionId}
                    onChange={value => setTargetVersionId(String(value))}
                    placeholder="已有 eval_target_version 的 ID（可选）"
                  />
                </FieldLabel>
              </div>
            ) : null}

            {step === STEP_EVAL_SET ? (
              <Spin spinning={evalSetsLoading}>
                <FieldLabel
                  label={I18n.t('batch_test_eval_set', {}, '评估数据集')}
                  required
                >
                  <Select
                    value={evalSetId}
                    onChange={value => setEvalSetId(String(value))}
                    optionList={evalSetOptions}
                    placeholder="请选择数据集"
                    style={{ width: '100%' }}
                  />
                </FieldLabel>
              </Spin>
            ) : null}

            {step === STEP_EVALUATORS ? (
              <Spin spinning={evaluatorsLoading}>
                <FieldLabel
                  label={I18n.t('batch_test_evaluators', {}, '评估器')}
                  required
                >
                  <Select
                    multiple
                    value={evaluatorVersionIds}
                    onChange={value =>
                      setEvaluatorVersionIds(
                        Array.isArray(value) ? value.map(String) : [],
                      )
                    }
                    optionList={evaluatorOptions}
                    placeholder="请选择评估器（可多选）"
                    style={{ width: '100%' }}
                  />
                </FieldLabel>
              </Spin>
            ) : null}

            {step === STEP_CONFIRM ? (
              <ConfirmStep
                name={name}
                description={description}
                targetType={targetType}
                targetId={targetId}
                targetVersionId={targetVersionId}
                evalSetLabel={selectedEvalSetLabel}
                evaluatorLabels={selectedEvaluatorLabels}
              />
            ) : null}
          </div>

          <div className="flex justify-end mt-8 pt-4 border-t">
            <Space>
              <Button disabled={step === 0} onClick={() => setStep(step - 1)}>
                上一步
              </Button>
              {step < STEP_TITLES.length - 1 ? (
                <Button type="primary" onClick={goNext}>
                  下一步
                </Button>
              ) : (
                <Button
                  type="primary"
                  loading={submitting}
                  onClick={handleSubmit}
                >
                  提交
                </Button>
              )}
            </Space>
          </div>
        </div>
      </div>
    </div>
  );
};

export { BatchTestPage as Component };
export default BatchTestPage;
