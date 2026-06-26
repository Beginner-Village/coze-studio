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
import { strategyApi, type CapabilityInfo } from '@coze-arch/bot-api';

import type { AddCapabilityForm, EditCapabilityForm } from './types';
import { DEFAULT_ADD_CAP_FORM } from './constants';

interface UseCapabilityActionsOptions {
  strategyId: string | undefined;
  selectedScenarioId: number | null;
  reload: () => Promise<void>;
}

/** Handles add / edit / delete capability state and API calls. */
export const useCapabilityActions = ({
  strategyId,
  selectedScenarioId,
  reload,
}: UseCapabilityActionsOptions) => {
  const [addCapabilityVisible, setAddCapabilityVisible] = useState(false);
  const [addCapForm, setAddCapForm] =
    useState<AddCapabilityForm>(DEFAULT_ADD_CAP_FORM);
  const [addingCap, setAddingCap] = useState(false);

  const [editCapId, setEditCapId] = useState<number | null>(null);
  const [editCapForm, setEditCapForm] = useState<EditCapabilityForm>({
    alias_name: '',
    alias_description: '',
    prompt_content: '',
  });
  const [editingCap, setEditingCap] = useState(false);

  const handleAddCapability = async () => {
    if (!selectedScenarioId || !strategyId) {
      return;
    }
    const isPrompt = addCapForm.type === 'prompt';
    if (!isPrompt && !addCapForm.ref_id.trim()) {
      Toast.warning('请填写引用 ID');
      return;
    }
    if (isPrompt && !addCapForm.prompt_content.trim()) {
      Toast.warning('请填写提示词内容');
      return;
    }
    setAddingCap(true);
    try {
      await strategyApi.addCapability({
        scenario_id: selectedScenarioId,
        strategy_id: Number(strategyId),
        type: addCapForm.type,
        ref_id: isPrompt ? 0 : Number(addCapForm.ref_id),
        ref_sub_id:
          addCapForm.type === 'plugin' && addCapForm.ref_sub_id
            ? Number(addCapForm.ref_sub_id)
            : undefined,
        prompt_content: isPrompt ? addCapForm.prompt_content : undefined,
        alias_name: addCapForm.alias_name || undefined,
        alias_description: addCapForm.alias_description || undefined,
      });
      setAddCapabilityVisible(false);
      setAddCapForm(DEFAULT_ADD_CAP_FORM);
      await reload();
    } catch (err) {
      Toast.error('添加能力项失败');
      console.error(err);
    } finally {
      setAddingCap(false);
    }
  };

  const handleDeleteCapability = (cap: CapabilityInfo) => {
    Modal.confirm({
      title: '删除能力项',
      content: `确定删除能力项「${cap.alias_name || cap.type}」吗？`,
      okText: '删除',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await strategyApi.deleteCapability({ id: cap.id });
          Toast.success('删除成功');
          await reload();
        } catch (err) {
          Toast.error('删除失败');
          console.error(err);
        }
      },
    });
  };

  const handleUpdateCapability = async () => {
    if (editCapId === null) {
      return;
    }
    setEditingCap(true);
    try {
      await strategyApi.updateCapability({
        id: editCapId,
        alias_name: editCapForm.alias_name || undefined,
        alias_description: editCapForm.alias_description || undefined,
        prompt_content: editCapForm.prompt_content || undefined,
      });
      setEditCapId(null);
      await reload();
    } catch (err) {
      Toast.error('更新失败');
      console.error(err);
    } finally {
      setEditingCap(false);
    }
  };

  const openEditCapability = (cap: CapabilityInfo) => {
    setEditCapId(cap.id);
    setEditCapForm({
      alias_name: cap.alias_name || '',
      alias_description: cap.alias_description || '',
      prompt_content: cap.prompt_content || '',
    });
  };

  return {
    addCapabilityVisible,
    setAddCapabilityVisible,
    addCapForm,
    setAddCapForm,
    addingCap,
    editCapId,
    setEditCapId,
    editCapForm,
    setEditCapForm,
    editingCap,
    handleAddCapability,
    handleDeleteCapability,
    handleUpdateCapability,
    openEditCapability,
  };
};
