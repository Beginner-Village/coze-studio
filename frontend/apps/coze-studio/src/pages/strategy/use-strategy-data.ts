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

import { useCallback, useEffect, useState } from 'react';

import { I18n } from '@coze-arch/i18n';
import { Toast } from '@coze-arch/coze-design';
import { strategyApi, type StrategyInfo } from '@coze-arch/bot-api';

/** Loads strategy detail and handles save/publish. */
export const useStrategyData = (strategyId: string | undefined) => {
  const [strategy, setStrategy] = useState<StrategyInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [editName, setEditName] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [saving, setSaving] = useState(false);
  const [selectedScenarioId, setSelectedScenarioId] = useState<number | null>(
    null,
  );

  const loadDetail = useCallback(
    async (currentSelectedId: number | null) => {
      if (!strategyId) {
        return;
      }
      setLoading(true);
      try {
        const resp = await strategyApi.getStrategyDetail({
          id: Number(strategyId),
        });
        if (resp?.data) {
          setStrategy(resp.data);
          setEditName(resp.data.name || '');
          setEditDesc(resp.data.description || '');
          if (resp.data.scenarios?.length && currentSelectedId === null) {
            setSelectedScenarioId(resp.data.scenarios[0].id);
          }
        }
      } catch (err) {
        Toast.error('加载策略失败');
        console.error(err);
      } finally {
        setLoading(false);
      }
    },
    [strategyId],
  );

  useEffect(() => {
    void loadDetail(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only run on strategyId mount
  }, [strategyId]);

  const reload = useCallback(
    () => loadDetail(selectedScenarioId),
    [loadDetail, selectedScenarioId],
  );

  const handleSaveStrategy = async () => {
    if (!strategyId || !editName.trim()) {
      return;
    }
    setSaving(true);
    try {
      await strategyApi.updateStrategy({
        id: Number(strategyId),
        name: editName.trim(),
        description: editDesc.trim() || undefined,
      });
      Toast.success('保存成功');
      await reload();
    } catch (err) {
      Toast.error('保存失败');
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  const handlePublish = async () => {
    if (!strategyId) {
      return;
    }
    setPublishing(true);
    try {
      await strategyApi.publishStrategy({ id: Number(strategyId) });
      Toast.success(`${I18n.t('strategy_publish')} 成功`);
      await reload();
    } catch (err) {
      Toast.error('发布失败');
      console.error(err);
    } finally {
      setPublishing(false);
    }
  };

  return {
    strategy,
    loading,
    publishing,
    editName,
    setEditName,
    editDesc,
    setEditDesc,
    saving,
    selectedScenarioId,
    setSelectedScenarioId,
    reload,
    handleSaveStrategy,
    handlePublish,
  };
};
