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

import { useParams, type URLSearchParamsInit } from 'react-router-dom';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { IntelligenceType } from '@coze-arch/idl/intelligence_api';
import {
  Button,
  Input,
  Layout,
  Select,
  Space,
  Spin,
  Steps,
  Toast,
} from '@coze-arch/coze-design';
import { intelligenceApi, workflowApi } from '@coze-arch/bot-api';

import {
  createExperiment,
  listEvaluationSets,
  listEvaluators,
  type EvaluationSet,
  type Evaluator,
} from '../loop-eval-api';

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

const PAGE_SIZE = 50;

const STEP_TITLES = ['基础信息', '选择评估集', '选择评估器'];

// Eval target types. `a2a_agent` / `custom_agent` are the agent-eval types added
// by the coze-loop fusion; `prompt` / `bot` / `workflow` predate it.
// For `workflow` / `bot` we list real candidates via the studio APIs
// (workflowApi.GetWorkFlowList / intelligenceApi.GetDraftIntelligenceList) so the
// user can pick a target_id from a dropdown. For `prompt` / `a2a_agent` /
// `custom_agent` there is no dedicated list endpoint yet, so target_id stays
// free-text. TODO: add a picker for those once a ListTargets endpoint ships.
type TargetType = 'prompt' | 'bot' | 'a2a_agent' | 'custom_agent' | 'workflow';

const TARGET_TYPE_OPTIONS: { label: string; value: TargetType }[] = [
  { label: '工作流 (Workflow)', value: 'workflow' },
  { label: '智能体 (Bot)', value: 'bot' },
  { label: 'Prompt', value: 'prompt' },
  { label: 'A2A Agent', value: 'a2a_agent' },
  { label: 'Custom Agent', value: 'custom_agent' },
];

function targetIdHint(targetType: TargetType): string {
  switch (targetType) {
    case 'workflow':
      return '工作流 ID（target_id）';
    case 'bot':
      return '智能体 ID（target_id）';
    case 'prompt':
      return 'Prompt ID（target_id）';
    case 'a2a_agent':
      return 'A2A Agent ID（target_id）';
    case 'custom_agent':
      return 'Custom Agent ID（target_id）';
    default:
      return 'target_id';
  }
}

interface PageProps {
  workspaceId?: string;
  setSearchParams?: (nextInit: URLSearchParamsInit) => void;
}

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

interface TargetOption {
  label: string;
  value: string;
}

// Whether the given target type has a list endpoint we can drive a picker with.
function isPickableTargetType(targetType: TargetType): boolean {
  return targetType === 'workflow' || targetType === 'bot';
}

const Page: React.FC<PageProps> = ({ workspaceId, setSearchParams }) => {
  const { space_id: spaceIdFromParams } = useParams<{ space_id: string }>();
  const effectiveWorkspaceId = workspaceId || spaceIdFromParams || '';
  const [step, setStep] = useState(0);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [targetType, setTargetType] = useState<TargetType>('workflow');
  const [targetId, setTargetId] = useState('');
  const [targetVersionId, setTargetVersionId] = useState('');
  const [evalSetId, setEvalSetId] = useState('');
  const [evaluatorVersionIds, setEvaluatorVersionIds] = useState<string[]>([]);
  const [evalSets, setEvalSets] = useState<EvaluationSet[]>([]);
  const [evaluators, setEvaluators] = useState<Evaluator[]>([]);
  const [evalSetsLoading, setEvalSetsLoading] = useState(false);
  const [evaluatorsLoading, setEvaluatorsLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [targetOptions, setTargetOptions] = useState<TargetOption[]>([]);
  const [targetOptionsLoading, setTargetOptionsLoading] = useState(false);

  const fetchTargetOptions = useCallback(async () => {
    if (!effectiveWorkspaceId || !isPickableTargetType(targetType)) {
      setTargetOptions([]);
      return;
    }
    setTargetOptionsLoading(true);
    try {
      if (targetType === 'workflow') {
        const res = await workflowApi.GetWorkFlowList({
          space_id: effectiveWorkspaceId,
          page: 1,
          size: PAGE_SIZE,
        });
        setTargetOptions(
          (res.data?.workflow_list || [])
            .map(wf => ({
              label: wf.name || wf.workflow_id || '-',
              value: String(wf.workflow_id || ''),
            }))
            .filter(option => option.value),
        );
      } else {
        const res = await intelligenceApi.GetDraftIntelligenceList({
          space_id: effectiveWorkspaceId,
          types: [IntelligenceType.Bot],
          size: PAGE_SIZE,
        });
        setTargetOptions(
          (res.data?.intelligences || [])
            .map(item => ({
              label: item.basic_info?.name || item.basic_info?.id || '-',
              value: String(item.basic_info?.id || ''),
            }))
            .filter(option => option.value),
        );
      }
    } catch (err: unknown) {
      // Graceful degradation: keep the free-text input usable if listing fails.
      setTargetOptions([]);
      Toast.error(getErrorMessage(err, '加载评测对象列表失败'));
    } finally {
      setTargetOptionsLoading(false);
    }
  }, [effectiveWorkspaceId, targetType]);

  useEffect(() => {
    fetchTargetOptions();
  }, [fetchTargetOptions]);

  const fetchEvalSets = useCallback(async () => {
    if (!effectiveWorkspaceId || evalSets.length > 0 || evalSetsLoading) {
      return;
    }
    setEvalSetsLoading(true);
    try {
      const res = await listEvaluationSets({
        workspace_id: effectiveWorkspaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      setEvalSets(res.evaluation_sets || []);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估集失败'));
    } finally {
      setEvalSetsLoading(false);
    }
  }, [effectiveWorkspaceId, evalSets.length, evalSetsLoading]);

  const fetchEvaluators = useCallback(async () => {
    if (!effectiveWorkspaceId || evaluators.length > 0 || evaluatorsLoading) {
      return;
    }
    setEvaluatorsLoading(true);
    try {
      const res = await listEvaluators({
        workspace_id: effectiveWorkspaceId,
        page_size: PAGE_SIZE,
        page_number: 1,
      });
      setEvaluators(res.evaluators || []);
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '加载评估器失败'));
    } finally {
      setEvaluatorsLoading(false);
    }
  }, [effectiveWorkspaceId, evaluators.length, evaluatorsLoading]);

  useEffect(() => {
    if (step === 1) {
      fetchEvalSets();
    }
    if (step === 2) {
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

  const goNext = () => {
    if (step === 0 && !name.trim()) {
      Toast.error('请输入实验名称');
      return;
    }
    if (step === 1 && !evalSetId) {
      Toast.error('请选择评估集');
      return;
    }
    setStep(prev => Math.min(prev + 1, STEP_TITLES.length - 1));
  };

  const handleSubmit = async () => {
    if (!effectiveWorkspaceId) {
      Toast.error('缺少 workspace_id');
      return;
    }
    if (evaluatorVersionIds.length === 0) {
      Toast.error('请至少选择一个评估器');
      return;
    }
    setSubmitting(true);
    try {
      await createExperiment({
        workspace_id: effectiveWorkspaceId,
        name: name.trim(),
        description: description.trim() || undefined,
        eval_set_id: evalSetId,
        evaluator_version_ids: evaluatorVersionIds,
        target_id: targetId.trim() || undefined,
        target_version_id: targetVersionId.trim() || undefined,
      });
      Toast.success('创建成功');
      setSearchParams?.({ tab: 'experiments' });
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '创建实验失败'));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">新建实验</div>
            <div className="text-sm text-gray-500 mt-1">
              配置基础信息、评估集和评估器
            </div>
          </div>
          <Button onClick={() => setSearchParams?.({ tab: 'experiments' })}>
            返回列表
          </Button>
        </div>
      </Layout.Header>

      <Layout.Content>
        <div style={{ maxWidth: 760, padding: '24px 0' }}>
          <Steps current={step}>
            {STEP_TITLES.map(title => (
              <Steps.Step key={title} title={title} />
            ))}
          </Steps>

          <div className="mt-8">
            {step === 0 ? (
              <div>
                <FieldLabel label="实验名称" required>
                  <Input
                    value={name}
                    onChange={value => setName(String(value))}
                    placeholder="请输入实验名称"
                  />
                </FieldLabel>
                <FieldLabel label="描述">
                  <Input
                    value={description}
                    onChange={value => setDescription(String(value))}
                    placeholder="请输入描述"
                  />
                </FieldLabel>
                <FieldLabel label="评测对象类型">
                  <Select
                    value={targetType}
                    onChange={value => {
                      setTargetType(value as TargetType);
                      setTargetId('');
                    }}
                    optionList={TARGET_TYPE_OPTIONS}
                    style={{ width: '100%' }}
                  />
                </FieldLabel>
                <FieldLabel label="评测对象 ID (target_id)">
                  {isPickableTargetType(targetType) ? (
                    <Select
                      filter
                      value={targetId || undefined}
                      onChange={value => setTargetId(String(value ?? ''))}
                      optionList={targetOptions}
                      loading={targetOptionsLoading}
                      placeholder={targetIdHint(targetType)}
                      style={{ width: '100%' }}
                    />
                  ) : (
                    <Input
                      value={targetId}
                      onChange={value => setTargetId(String(value))}
                      placeholder={targetIdHint(targetType)}
                    />
                  )}
                </FieldLabel>
                <FieldLabel label="评测对象版本 ID (target_version_id)">
                  <Input
                    value={targetVersionId}
                    onChange={value => setTargetVersionId(String(value))}
                    placeholder="可选: 现有 eval_target_version 的 ID"
                  />
                </FieldLabel>
              </div>
            ) : null}

            {step === 1 ? (
              <Spin spinning={evalSetsLoading}>
                <FieldLabel label="评估集" required>
                  <Select
                    value={evalSetId}
                    onChange={value => setEvalSetId(String(value))}
                    optionList={evalSetOptions}
                    placeholder="请选择评估集"
                    style={{ width: '100%' }}
                  />
                </FieldLabel>
              </Spin>
            ) : null}

            {step === 2 ? (
              <Spin spinning={evaluatorsLoading}>
                <FieldLabel label="评估器" required>
                  <Select
                    multiple
                    value={evaluatorVersionIds}
                    onChange={value =>
                      setEvaluatorVersionIds(
                        Array.isArray(value) ? value.map(String) : [],
                      )
                    }
                    optionList={evaluatorOptions}
                    placeholder="请选择评估器"
                    style={{ width: '100%' }}
                  />
                </FieldLabel>
              </Spin>
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
      </Layout.Content>
    </Layout>
  );
};

export { Page as Component };
export default Page;
