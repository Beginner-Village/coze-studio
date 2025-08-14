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

import React, { useMemo } from 'react';

import { withField, useWatch } from '@/form';
import type { Parameter } from '@coze-workflow/base';

import { CARD_SELECTOR_PATH, INPUT_PATH } from '../constants';
import type { CardSelectorParams } from '../types';

export interface CardSelectorOutputFieldProps {
  title?: string;
  tooltip?: React.ReactNode;
  id?: string;
}

export const CardSelectorOutputField = withField<CardSelectorOutputFieldProps>(
  ({ title = '模板输出预览', tooltip }: CardSelectorOutputFieldProps) => {
    // 监听卡片选择器的值变化
    const cardSelectorValue = useWatch<CardSelectorParams>(CARD_SELECTOR_PATH);
    const inputParameters = useWatch<Parameter[]>(INPUT_PATH);

    // 生成输出结构
    const outputStructure = useMemo(() => {
      // 使用真实的卡片信息生成模板ID和名称
      const selectedCard = cardSelectorValue?.selectedCard;
      const templateId = selectedCard?.code || '{card_code}';
      const templateName = selectedCard?.cardName || '{card_name}';

      // 根据输入参数动态生成dataResponse
      const dataResponse: Record<string, unknown> = {};
      
      if (inputParameters && Array.isArray(inputParameters)) {
        inputParameters.forEach((param: Parameter) => {
          if (param?.name) {
            // 将输入参数名作为dataResponse的键
            dataResponse[param.name] = `{${param.name}}`;
          }
        });
      }

      // 如果没有输入参数，提供一个默认的示例
      if (Object.keys(dataResponse).length === 0) {
        dataResponse.payeeList = '{payeeList}';
      }

      return {
        displayResponseType: 'TEMPLATE',
        rawContent: {},
        templateId,
        templateName,
        kvMap: {},
        dataResponse,
      };
    }, [cardSelectorValue, inputParameters]);

    return (
      <div style={{
        border: '1px solid var(--semi-color-border)',
        borderRadius: '6px',
        padding: '16px',
        marginBottom: '16px',
        background: 'var(--semi-color-bg-0)',
      }}>
        {/* 标题 */}
        <div style={{
          fontSize: '12px',
          fontWeight: 600,
          marginBottom: '12px',
          color: 'var(--semi-color-text-0)',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        }}>
          <span>{title}</span>
          {tooltip && (
            <span 
              title={String(tooltip)}
              style={{ 
                color: 'var(--semi-color-text-2)', 
                cursor: 'help',
                fontSize: '12px',
              }}
            >
              ℹ
            </span>
          )}
        </div>

        {/* 输出结构预览 */}
        <div style={{ 
          backgroundColor: 'var(--semi-color-fill-0)', 
          padding: '12px', 
          borderRadius: '4px',
          fontSize: '12px',
          fontFamily: 'Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
          border: '1px solid var(--semi-color-border)',
        }}>
          <pre style={{ 
            margin: 0, 
            color: 'var(--semi-color-text-0)', 
            overflow: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}>
            {JSON.stringify(outputStructure, null, 2)}
          </pre>
        </div>

        {/* 说明文字 */}
        <div style={{ 
          fontSize: '12px', 
          color: 'var(--semi-color-text-2)',
          marginTop: '8px',
          lineHeight: '1.4',
        }}>
          此输出结构会根据所选卡片和输入参数自动生成。dataResponse 中的变量会根据输入参数动态适配。
        </div>
      </div>
    );
  },
);