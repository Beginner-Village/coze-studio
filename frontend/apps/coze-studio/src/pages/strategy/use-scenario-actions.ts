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

import { useState } from 'react';

import { Modal, Toast } from '@coze-arch/coze-design';
import { strategyApi, type ScenarioInfo } from '@coze-arch/bot-api';

interface UseScenarioActionsOptions {
  strategyId: string | undefined;
  selectedScenarioId: string | null;
  setSelectedScenarioId: (id: string | null) => void;
  reload: () => Promise<void>;
}

/** Handles create / rename / delete scenario state and API calls. */
export const useScenarioActions = ({
  strategyId,
  selectedScenarioId,
  setSelectedScenarioId,
  reload,
}: UseScenarioActionsOptions) => {
  const [addScenarioVisible, setAddScenarioVisible] = useState(false);
  const [newScenarioName, setNewScenarioName] = useState('');
  const [newScenarioDesc, setNewScenarioDesc] = useState('');
  const [addingScenario, setAddingScenario] = useState(false);

  // renameScenarioId is a string (int64) or null
  const [renameScenarioId, setRenameScenarioId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [renaming, setRenaming] = useState(false);

  const handleAddScenario = async () => {
    if (!strategyId || !newScenarioName.trim()) {
      return;
    }
    setAddingScenario(true);
    try {
      const resp = await strategyApi.createScenario({
        strategy_id: strategyId,
        name: newScenarioName.trim(),
        description: newScenarioDesc.trim() || undefined,
      });
      setAddScenarioVisible(false);
      setNewScenarioName('');
      setNewScenarioDesc('');
      if (resp?.data?.id) {
        setSelectedScenarioId(resp.data.id);
      }
      await reload();
    } catch (err) {
      Toast.error('添加场景失败');
      console.error(err);
    } finally {
      setAddingScenario(false);
    }
  };

  const handleDeleteScenario = (scenario: ScenarioInfo) => {
    Modal.confirm({
      title: '删除场景',
      content: `确定删除场景「${scenario.name}」吗？`,
      okText: '删除',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await strategyApi.deleteScenario({ id: scenario.id });
          if (selectedScenarioId === scenario.id) {
            setSelectedScenarioId(null);
          }
          Toast.success('删除成功');
          await reload();
        } catch (err) {
          Toast.error('删除失败');
          console.error(err);
        }
      },
    });
  };

  const handleRenameScenario = async () => {
    if (renameScenarioId === null || !renameValue.trim()) {
      return;
    }
    setRenaming(true);
    try {
      await strategyApi.updateScenario({
        id: renameScenarioId,
        name: renameValue.trim(),
      });
      setRenameScenarioId(null);
      setRenameValue('');
      await reload();
    } catch (err) {
      Toast.error('重命名失败');
      console.error(err);
    } finally {
      setRenaming(false);
    }
  };

  return {
    addScenarioVisible,
    setAddScenarioVisible,
    newScenarioName,
    setNewScenarioName,
    newScenarioDesc,
    setNewScenarioDesc,
    addingScenario,
    renameScenarioId,
    setRenameScenarioId,
    renameValue,
    setRenameValue,
    renaming,
    handleAddScenario,
    handleDeleteScenario,
    handleRenameScenario,
  };
};
