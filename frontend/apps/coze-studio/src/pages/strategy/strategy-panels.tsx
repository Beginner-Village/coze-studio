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

import { I18n } from '@coze-arch/i18n';
import { Button, Tag, Typography } from '@coze-arch/coze-design';
import type {
  CapabilityInfo,
  ScenarioInfo,
  StrategyInfo,
} from '@coze-arch/bot-api';

import {
  CAPABILITY_TYPE_LABELS,
  DEFAULT_ADD_CAP_FORM,
  TYPE_COLORS,
} from './constants';

import styles from './index.module.less';

const { Text, Title } = Typography;

// ---------- Header ----------

interface StrategyHeaderProps {
  strategy: StrategyInfo | null;
  editName: string;
  editDesc: string;
  saving: boolean;
  publishing: boolean;
  onNameChange: (v: string) => void;
  onDescChange: (v: string) => void;
  onSave: () => void;
  onPublish: () => void;
}

export const StrategyHeader: React.FC<StrategyHeaderProps> = ({
  strategy,
  editName,
  editDesc,
  saving,
  publishing,
  onNameChange,
  onDescChange,
  onSave,
  onPublish,
}) => (
  <div className={styles.header}>
    <div className={styles.headerLeft}>
      <input
        className={styles.nameInput}
        value={editName}
        onChange={e => onNameChange(e.target.value)}
        placeholder="策略名称"
      />
      <input
        className={styles.descInput}
        value={editDesc}
        onChange={e => onDescChange(e.target.value)}
        placeholder={I18n.t('strategy_model_facing_desc')}
      />
      <Button loading={saving} onClick={onSave} style={{ marginLeft: 8 }}>
        保存
      </Button>
      {strategy?.status === 1 && (
        <Tag color="green" style={{ marginLeft: 8 }}>
          已发布
        </Tag>
      )}
    </div>
    <div className={styles.headerRight}>
      <Button type="primary" loading={publishing} onClick={onPublish}>
        {I18n.t('strategy_publish')}
      </Button>
    </div>
  </div>
);

// ---------- Scenario panel ----------

interface ScenarioPanelProps {
  scenarios: ScenarioInfo[] | undefined;
  selectedScenarioId: number | null;
  onSelect: (id: number) => void;
  onAdd: () => void;
  onRename: (scenario: ScenarioInfo) => void;
  onDelete: (scenario: ScenarioInfo) => void;
}

export const ScenarioPanel: React.FC<ScenarioPanelProps> = ({
  scenarios,
  selectedScenarioId,
  onSelect,
  onAdd,
  onRename,
  onDelete,
}) => (
  <div className={styles.scenarioPanel}>
    <div className={styles.scenarioPanelHeader}>
      <Title heading={6} style={{ margin: 0 }}>
        场景
      </Title>
      <Button size="small" onClick={onAdd}>
        {I18n.t('strategy_scenario_add')}
      </Button>
    </div>
    <div className={styles.scenarioList}>
      {scenarios?.map(scenario => (
        <div
          key={scenario.id}
          className={[
            styles.scenarioItem,
            selectedScenarioId === scenario.id
              ? styles.scenarioItemSelected
              : '',
          ].join(' ')}
          onClick={() => onSelect(scenario.id)}
        >
          <Text ellipsis style={{ flex: 1, cursor: 'pointer' }}>
            {scenario.name}
          </Text>
          <div
            className={styles.scenarioActions}
            onClick={e => e.stopPropagation()}
          >
            <Button
              size="mini"
              type="tertiary"
              onClick={() => onRename(scenario)}
            >
              改名
            </Button>
            <Button
              size="mini"
              type="tertiary"
              onClick={() => onDelete(scenario)}
            >
              删除
            </Button>
          </div>
        </div>
      ))}
      {!scenarios?.length && (
        <div className={styles.empty}>
          <Text type="tertiary">暂无场景，请添加</Text>
        </div>
      )}
    </div>
  </div>
);

// ---------- Capability panel ----------

interface CapabilityPanelProps {
  scenario: ScenarioInfo | undefined;
  capabilities: CapabilityInfo[];
  onAdd: () => void;
  onEdit: (cap: CapabilityInfo) => void;
  onDelete: (cap: CapabilityInfo) => void;
}

export const CapabilityPanel: React.FC<CapabilityPanelProps> = ({
  scenario,
  capabilities,
  onAdd,
  onEdit,
  onDelete,
}) => {
  if (!scenario) {
    return (
      <div className={styles.capabilityPanel}>
        <div className={styles.empty}>
          <Text type="tertiary">请从左侧选择一个场景</Text>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.capabilityPanel}>
      <div className={styles.capabilityPanelHeader}>
        <Title heading={6} style={{ margin: 0 }}>
          {scenario.name} — 能力项
        </Title>
        <Button type="primary" size="small" onClick={onAdd}>
          {I18n.t('strategy_capability_add')}
        </Button>
      </div>
      <table className={styles.capTable}>
        <thead>
          <tr>
            <th>类型</th>
            <th>别名</th>
            <th>{I18n.t('strategy_model_facing_desc')}</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {capabilities.map(cap => (
            <tr key={cap.id}>
              <td>
                <Tag color={TYPE_COLORS[cap.type] || 'default'}>
                  {CAPABILITY_TYPE_LABELS[cap.type]?.() ?? cap.type}
                </Tag>
              </td>
              <td>{cap.alias_name || '-'}</td>
              <td>{cap.alias_description || '-'}</td>
              <td>
                <Button size="mini" type="tertiary" onClick={() => onEdit(cap)}>
                  编辑
                </Button>
                <Button
                  size="mini"
                  type="tertiary"
                  onClick={() => onDelete(cap)}
                  style={{ marginLeft: 4 }}
                >
                  删除
                </Button>
              </td>
            </tr>
          ))}
          {capabilities.length === 0 && (
            <tr>
              <td colSpan={4}>
                <Text type="tertiary">暂无能力项</Text>
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
};

// helper used by parent to open add-cap modal with reset state
export { DEFAULT_ADD_CAP_FORM };
