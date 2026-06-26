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
import {
  Input,
  Modal,
  Select,
  TextArea,
  Typography,
} from '@coze-arch/coze-design';

import type { AddCapabilityForm, EditCapabilityForm } from './types';
import { CAPABILITY_TYPES } from './constants';

const { Text } = Typography;

// ---------- Add Scenario ----------

interface AddScenarioModalProps {
  visible: boolean;
  name: string;
  desc: string;
  loading: boolean;
  onOk: () => void;
  onCancel: () => void;
  onNameChange: (v: string) => void;
  onDescChange: (v: string) => void;
}

export const AddScenarioModal: React.FC<AddScenarioModalProps> = ({
  visible,
  name,
  desc,
  loading,
  onOk,
  onCancel,
  onNameChange,
  onDescChange,
}) => (
  <Modal
    visible={visible}
    title={I18n.t('strategy_scenario_add')}
    okText="确定"
    cancelText="取消"
    onOk={onOk}
    onCancel={onCancel}
    okButtonProps={{ loading, disabled: !name.trim() }}
  >
    <div style={{ marginBottom: 12 }}>
      <Input
        value={name}
        onChange={onNameChange}
        placeholder={I18n.t('strategy_scenario_name')}
      />
    </div>
    <div>
      <Input
        value={desc}
        onChange={onDescChange}
        placeholder={I18n.t('strategy_model_facing_desc')}
      />
    </div>
  </Modal>
);

// ---------- Rename Scenario ----------

interface RenameScenarioModalProps {
  visible: boolean;
  value: string;
  loading: boolean;
  onOk: () => void;
  onCancel: () => void;
  onChange: (v: string) => void;
}

export const RenameScenarioModal: React.FC<RenameScenarioModalProps> = ({
  visible,
  value,
  loading,
  onOk,
  onCancel,
  onChange,
}) => (
  <Modal
    visible={visible}
    title="重命名场景"
    okText="确定"
    cancelText="取消"
    onOk={onOk}
    onCancel={onCancel}
    okButtonProps={{ loading, disabled: !value.trim() }}
  >
    <Input
      value={value}
      onChange={onChange}
      placeholder={I18n.t('strategy_scenario_name')}
    />
  </Modal>
);

// ---------- Add Capability ----------

interface AddCapabilityModalProps {
  visible: boolean;
  form: AddCapabilityForm;
  loading: boolean;
  onOk: () => void;
  onCancel: () => void;
  onFormChange: (patch: Partial<AddCapabilityForm>) => void;
}

export const AddCapabilityModal: React.FC<AddCapabilityModalProps> = ({
  visible,
  form,
  loading,
  onOk,
  onCancel,
  onFormChange,
}) => (
  <Modal
    visible={visible}
    title={I18n.t('strategy_capability_add')}
    okText="确定"
    cancelText="取消"
    onOk={onOk}
    onCancel={onCancel}
    okButtonProps={{ loading }}
    style={{ width: 520 }}
  >
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>类型</Text>
        <Select
          value={form.type}
          onChange={v => onFormChange({ type: String(v) })}
          style={{ width: '100%' }}
          optionList={CAPABILITY_TYPES.map(t => ({
            label: t.label(),
            value: t.value,
          }))}
        />
      </div>

      {form.type !== 'prompt' && (
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>
            {form.type === 'workflow'
              ? 'Workflow ID'
              : form.type === 'plugin'
                ? 'Tool ID (ref_id)'
                : 'Knowledge ID'}
          </Text>
          <Input
            value={form.ref_id}
            onChange={v => onFormChange({ ref_id: v })}
            placeholder="数字 ID"
            type="number"
          />
        </div>
      )}

      {form.type === 'plugin' && (
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>
            Plugin ID (ref_sub_id)
          </Text>
          <Input
            value={form.ref_sub_id}
            onChange={v => onFormChange({ ref_sub_id: v })}
            placeholder="插件 ID（数字）"
            type="number"
          />
        </div>
      )}

      {form.type === 'prompt' && (
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>
            {I18n.t('strategy_prompt_content')}
          </Text>
          <TextArea
            value={form.prompt_content}
            onChange={v => onFormChange({ prompt_content: v })}
            placeholder={I18n.t('strategy_prompt_content')}
            rows={4}
          />
        </div>
      )}

      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>别名</Text>
        <Input
          value={form.alias_name}
          onChange={v => onFormChange({ alias_name: v })}
          placeholder="面向模型的能力名称"
        />
      </div>

      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>
          {I18n.t('strategy_model_facing_desc')}
        </Text>
        <TextArea
          value={form.alias_description}
          onChange={v => onFormChange({ alias_description: v })}
          placeholder={I18n.t('strategy_model_facing_desc')}
          rows={2}
        />
      </div>
    </div>
  </Modal>
);

// ---------- Edit Capability ----------

interface EditCapabilityModalProps {
  visible: boolean;
  form: EditCapabilityForm;
  loading: boolean;
  onOk: () => void;
  onCancel: () => void;
  onFormChange: (patch: Partial<EditCapabilityForm>) => void;
}

export const EditCapabilityModal: React.FC<EditCapabilityModalProps> = ({
  visible,
  form,
  loading,
  onOk,
  onCancel,
  onFormChange,
}) => (
  <Modal
    visible={visible}
    title="编辑能力项"
    okText="确定"
    cancelText="取消"
    onOk={onOk}
    onCancel={onCancel}
    okButtonProps={{ loading }}
    style={{ width: 480 }}
  >
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>别名</Text>
        <Input
          value={form.alias_name}
          onChange={v => onFormChange({ alias_name: v })}
          placeholder="面向模型的能力名称"
        />
      </div>
      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>
          {I18n.t('strategy_model_facing_desc')}
        </Text>
        <TextArea
          value={form.alias_description}
          onChange={v => onFormChange({ alias_description: v })}
          placeholder={I18n.t('strategy_model_facing_desc')}
          rows={2}
        />
      </div>
      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>
          {I18n.t('strategy_prompt_content')}
        </Text>
        <TextArea
          value={form.prompt_content}
          onChange={v => onFormChange({ prompt_content: v })}
          placeholder={I18n.t('strategy_prompt_content')}
          rows={3}
        />
      </div>
    </div>
  </Modal>
);
