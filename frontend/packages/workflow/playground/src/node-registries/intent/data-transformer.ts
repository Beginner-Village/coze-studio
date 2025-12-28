/*
 * Copyright 2025 coze-dev Authors
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

import { get, isNil, omit, camelCase } from 'lodash-es';
import { type NodeFormContext } from '@flowgram-adapter/free-layout-editor';
import { variableUtils } from '@coze-workflow/variable';
import { getDefaultLLMParams, INTENT_NODE_MODE } from '@coze-workflow/nodes';
import {
  type NodeDataDTO,
  type ValueExpression,
  BlockInput,
} from '@coze-workflow/base';
import { ModelParamType, type Model } from '@coze-arch/bot-api/developer_api';

import { type FormData } from './types';
import { getDefaultOutputs } from './constants';

/**
 * 将 model 对象转换为 BlockInput 数组格式
 * 类似于 LLM 节点的 modelItemToBlockInput
 */
const modelToBlockInputArray = (
  model: Record<string, unknown>,
  modelMeta: Model | undefined,
): BlockInput[] =>
  Object.keys(model).map(k => {
    const type = modelMeta?.model_params?.find(
      p => camelCase(p.name) === k,
    )?.type;
    if (ModelParamType.Float === type) {
      return BlockInput.createFloat(k, model[k] as string);
    } else if (['modelType'].includes(k)) {
      // modelType 使用 String 类型避免大整数精度丢失
      return BlockInput.createString(k, `${model[k]}`);
    } else if (ModelParamType.Int === type) {
      return BlockInput.createInteger(k, model[k] as string);
    }
    return BlockInput.createString(k, model[k] as string);
  });

/**
 * 从 BlockInput 数组中提取值对
 */
const reviseLLMParamPair = (d: BlockInput): [string, unknown] => {
  let k = d?.name || '';
  if (k === 'modleName') {
    k = 'modelName';
  }
  let v = d.input?.value?.content;
  // modelType 保持原始字符串类型，避免大整数精度丢失
  if (['modelType'].includes(k)) {
    v = `${v}`;
  } else if (['float', 'integer'].includes(d.input?.type as string)) {
    v = Number(v);
  } else if (d.input?.type === 'boolean') {
    v = v === 'true' || v === true;
  }
  return [k, v];
};

/**
 * Node Backend Data - > Frontend Form Data
 */
export const transformOnInit = (
  value: NodeDataDTO,
  context: NodeFormContext,
) => {
  const { playgroundContext } = context || {};
  const { inputs, nodeMeta, outputs } = value || {};

  let llmParamRaw = get(inputs, 'llmParam');

  const { models } = playgroundContext;

  // When first dragged into the canvas: Parse out the default value from the backend return value.
  if (!llmParamRaw) {
    llmParamRaw = getDefaultLLMParams(models);
  }

  // 将 BlockInput 数组转换为对象格式
  const model: { [k: string]: unknown } = {};
  if (Array.isArray(llmParamRaw)) {
    // BlockInput 数组格式
    llmParamRaw.forEach((d: BlockInput) => {
      const [k, v] = reviseLLMParamPair(d);
      model[k] = v;
    });
  } else {
    // 旧格式：普通对象
    Object.assign(model, llmParamRaw);
  }

  const isNewCreateInInit = isNil(inputs);
  const inputParameters = get(inputs, 'inputParameters', []);

  // - If it is a new node, the default is fast mode, otherwise it is determined according to the backend return value (if there is no backend mode field, it means it is historical data, then it is standard mode)
  // - will support soon
  const intentModeInInit =
    isNewCreateInInit && !IS_OPEN_SOURCE
      ? INTENT_NODE_MODE.MINIMAL
      : (get(inputs, 'mode') as string) || INTENT_NODE_MODE.STANDARD;
  const isMinimalMode = intentModeInInit === INTENT_NODE_MODE.MINIMAL;
  const emptyIntent = [{ name: '' }];

  const intentsValue = get(inputs, 'intents', emptyIntent);

  return {
    nodeMeta,
    outputs: outputs || getDefaultOutputs(intentModeInInit),

    model: omit(model, [
      'enableChatHistory',
      'systemPrompt',
      'chatHistoryRound',
    ]) as { [k: string]: unknown },

    // The open-source version only supports standard mode
    intentMode: intentModeInInit,

    intents: isMinimalMode ? emptyIntent : intentsValue,

    quickIntents: isMinimalMode ? intentsValue : emptyIntent,

    inputs: {
      chatHistorySetting: {
        enableChatHistory: model.enableChatHistory || false,
        chatHistoryRound: model.chatHistoryRound || 3,
      },
      inputParameters:
        inputParameters.length === 0 ? [{ name: 'query' }] : inputParameters,
    },
    systemPrompt:
      (model?.systemPrompt as Record<string, Record<string, unknown>>)?.value
        ?.content ??
      (model?.systemPrompt as string) ??
      '',
  };
};

/**
 * Front-end form data - > node back-end data
 * @param value
 * @returns
 */
export const transformOnSubmit = (
  value: FormData,
  context: NodeFormContext,
): NodeDataDTO => {
  const { playgroundContext } = context || {};
  const {
    model,
    inputs,
    intents,
    quickIntents,
    intentMode,
    systemPrompt,
    nodeMeta,
    outputs,
  } = value || {};
  const { chatHistorySetting, inputParameters } = inputs || {};
  const { enableChatHistory, chatHistoryRound } = chatHistorySetting || {};

  const { models, globalState, variableService, node } =
    playgroundContext || {};

  const { isChatflow } = globalState || {};

  // 使用字符串比较避免大整数精度丢失问题
  const modelMeta = models.find(
    m => `${m.model_type}` === `${model.modelType}`,
  );

  const promptItem = {
    type: 'literal',
    content: '{{query}}',
  };
  const systemPromptItem = {
    type: 'literal',
    content: intentMode === INTENT_NODE_MODE.MINIMAL ? '' : systemPrompt,
  };

  // 构建 llmParam 数组，使用 BlockInput 格式（与 LLM 节点保持一致）
  const llmParam = modelToBlockInputArray(model, modelMeta);

  // 添加其他 llmParam 字段
  llmParam.push(BlockInput.createString('modelName', modelMeta?.name ?? ''));
  llmParam.push(
    variableUtils.valueExpressionToDTO(
      promptItem as ValueExpression,
      variableService,
      { node },
    ) as BlockInput,
  );
  llmParam.push(
    variableUtils.valueExpressionToDTO(
      systemPromptItem as ValueExpression,
      variableService,
      { node },
    ) as BlockInput,
  );
  // If it is a workflow, the history is closed by default when submitting, and the dialog flow is submitted according to the actual value of the user.
  llmParam.push(
    BlockInput.createBoolean(
      'enableChatHistory',
      isChatflow ? Boolean(enableChatHistory) : false,
    ),
  );
  // History rounds
  llmParam.push(
    BlockInput.createInteger('chatHistoryRound', String(chatHistoryRound ?? 3)),
  );

  const formattedValue: Record<string, unknown> = {
    nodeMeta,
    outputs,
    inputs: {
      ...(inputs || {}),
      inputParameters,
      llmParam,
      intents: intentMode === INTENT_NODE_MODE.MINIMAL ? quickIntents : intents,
      mode: intentMode,
    },
  };

  return formattedValue as unknown as NodeDataDTO;
};
