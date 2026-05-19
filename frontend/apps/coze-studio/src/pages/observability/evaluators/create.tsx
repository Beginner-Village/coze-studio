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
import React, { useMemo, useRef, useState } from 'react';

import {
  Button,
  Form,
  Layout,
  Select,
  Space,
  Toast,
} from '@coze-arch/coze-design';

import { createEvaluator, type CreateEvaluatorReq } from '../loop-eval-api';

const DEFAULT_MODEL = 'doubao-1-5-pro-32k-character-250228';

const EVALUATOR_TYPE = {
  Prompt: 1,
  Code: 2,
};

const ROLE = {
  System: 1,
};

const PROMPT_SOURCE_TYPE = {
  Custom: 3,
};

interface EvaluatorFormValues {
  name: string;
  description?: string;
  prompt_template?: string;
  model_name?: string;
  input_vars?: string;
  code?: string;
  language?: string;
}

interface FormApi {
  validate: () => Promise<EvaluatorFormValues>;
}

function getErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

function normalizeType(type: string | null): 'llm' | 'code' | 'agent' {
  if (type === 'code' || type === 'agent') {
    return type;
  }
  return 'llm';
}

function parseInputVars(value?: string): string[] {
  return (value || '')
    .split(',')
    .map(item => item.trim())
    .filter(Boolean);
}

const Page: React.FC = () => {
  const { space_id: routeSpaceId } = useParams<{ space_id: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const formApiRef = useRef<FormApi | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const evaluatorType = normalizeType(searchParams.get('type'));
  const spaceId = searchParams.get('space_id') || routeSpaceId || '';

  const initialValues = useMemo(
    () => ({
      model_name: DEFAULT_MODEL,
      language: 'Python',
    }),
    [],
  );

  const backToList = () => {
    setSearchParams({ tab: 'evaluators' });
  };

  const buildCreateReq = (values: EvaluatorFormValues): CreateEvaluatorReq => {
    const common = {
      name: values.name,
      description: values.description,
      workspace_id: spaceId,
    };

    if (evaluatorType === 'code') {
      return {
        cid: String(Date.now()),
        evaluator: {
          ...common,
          evaluator_type: EVALUATOR_TYPE.Code,
          current_version: {
            version: '0.0.1',
            evaluator_content: {
              code_evaluator: {
                language_type: values.language || 'Python',
                code_content: values.code || '',
              },
            },
          },
        },
      };
    }

    const inputVars = parseInputVars(values.input_vars);
    return {
      cid: String(Date.now()),
      evaluator: {
        ...common,
        evaluator_type: EVALUATOR_TYPE.Prompt,
        current_version: {
          version: '0.0.1',
          evaluator_content: {
            input_schemas: inputVars.map(key => ({
              key,
              support_content_types: ['Text'],
              json_schema: JSON.stringify({ type: 'string' }),
            })),
            prompt_evaluator: {
              prompt_source_type: PROMPT_SOURCE_TYPE.Custom,
              model_config: {
                model_id: values.model_name || DEFAULT_MODEL,
                model_name: values.model_name || DEFAULT_MODEL,
              },
              message_list: [
                {
                  role: ROLE.System,
                  content: {
                    content_type: 'Text',
                    text: values.prompt_template || '',
                  },
                },
              ],
            },
          },
        },
      },
    };
  };

  const handleSubmit = async () => {
    if (!spaceId || !formApiRef.current) {
      Toast.error('缺少空间 ID');
      return;
    }
    try {
      const values = await formApiRef.current.validate();
      setSubmitting(true);
      await createEvaluator(buildCreateReq(values));
      Toast.success('创建成功');
      backToList();
    } catch (err: unknown) {
      Toast.error(getErrorMessage(err, '创建评估器失败'));
    } finally {
      setSubmitting(false);
    }
  };

  if (evaluatorType === 'agent') {
    return (
      <Layout>
        <Layout.Header className="pb-0">
          <div className="flex items-center justify-between w-full">
            <div>
              <div className="font-[500] text-[20px]">新建 Agent 评估器</div>
              <div className="text-sm text-gray-500 mt-1">敬请期待</div>
            </div>
            <Button onClick={backToList}>取消</Button>
          </div>
        </Layout.Header>
        <Layout.Content>
          <div className="text-center py-16 text-gray-500 border rounded-[6px]">
            敬请期待
          </div>
        </Layout.Content>
      </Layout>
    );
  }

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="flex items-center justify-between w-full">
          <div>
            <div className="font-[500] text-[20px]">
              新建{evaluatorType === 'code' ? ' Code ' : ' LLM '}评估器
            </div>
            <div className="text-sm text-gray-500 mt-1">
              创建 Studio 原生评估器
            </div>
          </div>
          <Space>
            <Button onClick={backToList}>取消</Button>
            <Button type="primary" loading={submitting} onClick={handleSubmit}>
              创建
            </Button>
          </Space>
        </div>
      </Layout.Header>

      <Layout.Content>
        <div className="rounded-[6px] border p-4">
          <Form
            layout="vertical"
            initValues={initialValues}
            getFormApi={(api: FormApi) => {
              formApiRef.current = api;
            }}
          >
            <Form.Input
              field="name"
              label="名称"
              rules={[{ required: true, message: '请输入名称' }]}
              placeholder="请输入评估器名称"
            />
            <Form.Input
              field="description"
              label="描述"
              placeholder="请输入描述"
            />
            {evaluatorType === 'code' ? (
              <>
                <Form.TextArea
                  field="code"
                  label="代码"
                  rules={[{ required: true, message: '请输入代码' }]}
                  placeholder="请输入评估器代码"
                  autosize={{ minRows: 12, maxRows: 20 }}
                />
                <Form.Select field="language" label="语言">
                  <Select.Option value="Python">Python</Select.Option>
                  <Select.Option value="JS">JavaScript</Select.Option>
                </Form.Select>
              </>
            ) : (
              <>
                <Form.TextArea
                  field="prompt_template"
                  label="Prompt Template"
                  rules={[{ required: true, message: '请输入 Prompt' }]}
                  placeholder="请输入 Prompt 模板"
                  autosize={{ minRows: 8, maxRows: 16 }}
                />
                <Form.Select field="model_name" label="模型">
                  <Select.Option value={DEFAULT_MODEL}>
                    {DEFAULT_MODEL}
                  </Select.Option>
                </Form.Select>
                <Form.Input
                  field="input_vars"
                  label="输入变量"
                  placeholder="多个变量用英文逗号分隔，例如 input, actual_output"
                />
              </>
            )}
          </Form>
        </div>
      </Layout.Content>
    </Layout>
  );
};

export { Page as Component };
export default Page;
