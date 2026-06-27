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

import { useParams, useNavigate } from 'react-router-dom';
import React from 'react';

import { Typography } from '@coze-arch/coze-design';

import { useStrategyEditor } from './use-strategy-editor';
import {
  CapabilityPanel,
  ScenarioPanel,
  StrategyHeader,
} from './strategy-panels';
import {
  AddCapabilityModal,
  AddScenarioModal,
  EditCapabilityModal,
  RenameScenarioModal,
} from './strategy-modals';
import { DEFAULT_ADD_CAP_FORM } from './constants';

import styles from './index.module.less';

const { Text } = Typography;

const StrategyEditorPage: React.FC = () => {
  const { strategy_id, space_id } = useParams<{
    strategy_id: string;
    space_id: string;
  }>();
  const navigate = useNavigate();
  const editor = useStrategyEditor(strategy_id);

  const selectedScenario = editor.strategy?.scenarios?.find(
    s => s.id === editor.selectedScenarioId,
  );
  const capabilities = selectedScenario?.capabilities || [];

  const handleBack = () => {
    if (space_id) {
      navigate(`/space/${space_id}/library/10`);
    } else {
      navigate(-1);
    }
  };

  if (editor.loading && !editor.strategy) {
    return (
      <div className={styles.loading}>
        <Text>加载中...</Text>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      {/* Top nav bar with back button */}
      <div className={styles.topNav}>
        <button type="button" className={styles.backBtn} onClick={handleBack}>
          ← 返回
        </button>
        <span className={styles.topNavTitle}>
          {editor.strategy?.name || '策略编辑器'}
        </span>
      </div>

      <StrategyHeader
        strategy={editor.strategy}
        editName={editor.editName}
        editDesc={editor.editDesc}
        saving={editor.saving}
        publishing={editor.publishing}
        onNameChange={editor.setEditName}
        onDescChange={editor.setEditDesc}
        onSave={editor.handleSaveStrategy}
        onPublish={editor.handlePublish}
      />

      <div className={styles.body}>
        <ScenarioPanel
          scenarios={editor.strategy?.scenarios}
          selectedScenarioId={editor.selectedScenarioId}
          onSelect={editor.setSelectedScenarioId}
          onAdd={() => {
            editor.setNewScenarioName('');
            editor.setNewScenarioDesc('');
            editor.setAddScenarioVisible(true);
          }}
          onRename={scenario => {
            editor.setRenameScenarioId(scenario.id);
            editor.setRenameValue(scenario.name);
          }}
          onDelete={editor.handleDeleteScenario}
        />

        <CapabilityPanel
          scenario={selectedScenario}
          capabilities={capabilities}
          onAdd={() => {
            editor.setAddCapForm(DEFAULT_ADD_CAP_FORM);
            editor.setAddCapabilityVisible(true);
          }}
          onEdit={editor.openEditCapability}
          onDelete={editor.handleDeleteCapability}
        />
      </div>

      <AddScenarioModal
        visible={editor.addScenarioVisible}
        name={editor.newScenarioName}
        desc={editor.newScenarioDesc}
        loading={editor.addingScenario}
        onOk={editor.handleAddScenario}
        onCancel={() => editor.setAddScenarioVisible(false)}
        onNameChange={editor.setNewScenarioName}
        onDescChange={editor.setNewScenarioDesc}
      />
      <RenameScenarioModal
        visible={editor.renameScenarioId !== null}
        value={editor.renameValue}
        loading={editor.renaming}
        onOk={editor.handleRenameScenario}
        onCancel={() => editor.setRenameScenarioId(null)}
        onChange={editor.setRenameValue}
      />
      <AddCapabilityModal
        visible={editor.addCapabilityVisible}
        form={editor.addCapForm}
        loading={editor.addingCap}
        spaceId={space_id ?? ''}
        onOk={editor.handleAddCapability}
        onCancel={() => editor.setAddCapabilityVisible(false)}
        onFormChange={patch => editor.setAddCapForm(f => ({ ...f, ...patch }))}
      />
      <EditCapabilityModal
        visible={editor.editCapId !== null}
        form={editor.editCapForm}
        loading={editor.editingCap}
        spaceId={space_id ?? ''}
        onOk={editor.handleUpdateCapability}
        onCancel={() => editor.setEditCapId(null)}
        onFormChange={patch => editor.setEditCapForm(f => ({ ...f, ...patch }))}
      />
    </div>
  );
};

export default StrategyEditorPage;
