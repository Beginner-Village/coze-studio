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

import React, { type FC } from 'react';

import { useBotSkillStore } from '@coze-studio/bot-detail-store/bot-skill';
import { useBotDetailIsReadonly } from '@coze-studio/bot-detail-store';
import { I18n } from '@coze-arch/i18n';
import { Switch, Typography } from '@coze-arch/coze-design';

export const ForceToolReturn: FC = () => {
  const forceToolReturn = useBotSkillStore($store => $store.forceToolReturn);
  const updateForceToolReturn = useBotSkillStore(
    $store => $store.updateForceToolReturn,
  );
  const isReadonly = useBotDetailIsReadonly();

  return (
    <div
      style={{
        padding: '12px 16px',
        borderBottom: '1px solid var(--semi-color-border)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ flex: 1 }}>
          <Typography.Text strong>{I18n.t('工具结果回传模型')}</Typography.Text>
          <Typography.Paragraph
            type="tertiary"
            style={{ margin: '4px 0 0 0', fontSize: '12px' }}
          >
            {I18n.t(
              '开启后，工作流的"文本回复"不会直接发送给用户，而是返回给模型继续推理',
            )}
          </Typography.Paragraph>
        </div>
        <Switch
          checked={forceToolReturn}
          disabled={isReadonly}
          onChange={checked => {
            updateForceToolReturn(checked);
          }}
          data-testid="bot.editor.tool.force-tool-return.switch"
        />
      </div>
    </div>
  );
};
