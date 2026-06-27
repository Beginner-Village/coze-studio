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

import React from 'react';

import { Typography } from '@coze-arch/coze-design';
import type { CapabilitySchema } from '@coze-arch/bot-api';

const { Text } = Typography;

interface CapabilityParamListProps {
  schema?: CapabilitySchema;
}

export const CapabilityParamList: React.FC<CapabilityParamListProps> = ({
  schema,
}) => {
  const properties = schema?.properties;
  const required = schema?.required ?? [];

  if (!properties || Object.keys(properties).length === 0) {
    return null;
  }

  return (
    <div>
      <Text style={{ display: 'block', marginBottom: 6, fontWeight: 500 }}>
        入参（模型将自动按此传参）
      </Text>
      <div
        style={{
          border: '1px solid var(--semi-color-border)',
          borderRadius: 6,
          overflow: 'hidden',
        }}
      >
        <table
          style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}
        >
          <thead>
            <tr style={{ background: 'var(--semi-color-fill-0)' }}>
              {(['参数名', '类型', '必填', '说明'] as const).map(h => (
                <th
                  key={h}
                  style={{
                    padding: '6px 10px',
                    textAlign: 'left',
                    fontWeight: 500,
                    borderBottom: '1px solid var(--semi-color-border)',
                  }}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Object.entries(properties).map(([name, def], idx) => (
              <tr
                key={name}
                style={{
                  borderBottom:
                    idx < Object.keys(properties).length - 1
                      ? '1px solid var(--semi-color-border)'
                      : undefined,
                }}
              >
                <td style={{ padding: '5px 10px', fontFamily: 'monospace' }}>
                  {name}
                </td>
                <td style={{ padding: '5px 10px' }}>
                  <Text type="tertiary">{def.type ?? 'string'}</Text>
                </td>
                <td style={{ padding: '5px 10px' }}>
                  {required.includes(name) ? (
                    <Text style={{ color: 'var(--semi-color-danger)' }}>
                      是
                    </Text>
                  ) : (
                    <Text type="tertiary">否</Text>
                  )}
                </td>
                <td style={{ padding: '5px 10px' }}>
                  <Text type="secondary">{def.description ?? ''}</Text>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
