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

import { type NodeData } from '@coze-workflow/base';

import { type FormData } from './types';
import { OUTPUTS, DEFAULT_INPUTS } from './constants';

/**
 * 节点后端数据 -> 前端表单数据转换
 */
export function transformOnInit(data: NodeData): FormData {
  // 如果已经有inputParameters，过滤掉隐藏的MCP配置参数
  // 否则使用默认值
  const allInputParameters =
    data?.inputs?.inputParameters || data?.inputParameters || DEFAULT_INPUTS;

  // 过滤掉以__mcp_开头的隐藏配置参数，只显示工具参数
  const visibleInputParameters = allInputParameters.filter(
    param => !param.name.startsWith('__mcp_'),
  );

  return {
    nodeMeta: data?.nodeMeta, // 保留nodeMeta以支持标题和描述
    inputs: {
      inputParameters: visibleInputParameters,
    },
    outputs: data?.outputs || OUTPUTS,
  };
}

/**
 * 前端表单数据 -> 节点后端数据转换
 */
export function transformOnSubmit(
  data: FormData,
  originalData?: NodeData,
): NodeData {
  // 获取原始的隐藏配置参数
  const originalInputParameters =
    originalData?.inputs?.inputParameters ||
    originalData?.inputParameters ||
    [];
  const hiddenMcpParams = originalInputParameters.filter(param =>
    param.name.startsWith('__mcp_'),
  );

  console.log(
    '🔧 MCP transformOnSubmit - originalInputParameters:',
    originalInputParameters.length,
  );
  console.log(
    '🔧 MCP transformOnSubmit - hiddenMcpParams:',
    hiddenMcpParams.length,
    hiddenMcpParams.map(p => p.name),
  );
  console.log(
    '🔧 MCP transformOnSubmit - user edited params:',
    data.inputs.inputParameters.length,
    data.inputs.inputParameters.map(p => p.name),
  );

  // 合并隐藏参数和用户编辑的参数
  const allInputParameters = [
    ...hiddenMcpParams, // 保留隐藏的MCP配置参数
    ...data.inputs.inputParameters, // 用户编辑的工具参数
  ];

  console.log(
    '🔧 MCP transformOnSubmit - final allInputParameters:',
    allInputParameters.length,
    allInputParameters.map(p => p.name),
  );

  const result = {
    nodeMeta: data.nodeMeta, // 保存nodeMeta信息
    inputParameters: allInputParameters,
    outputs: data.outputs?.length > 0 ? data.outputs : OUTPUTS,
    // 同时保存到inputs结构中确保兼容性
    inputs: {
      inputParameters: allInputParameters,
    },
  };

  console.log('🔧 MCP transformOnSubmit - final result:', result);
  return result;
}
