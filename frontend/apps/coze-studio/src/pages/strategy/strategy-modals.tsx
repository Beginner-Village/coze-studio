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

import React, { useEffect, useRef } from 'react';

import { I18n } from '@coze-arch/i18n';
import {
  Input,
  Modal,
  Select,
  TextArea,
  Typography,
} from '@coze-arch/coze-design';
import { strategyApi } from '@coze-arch/bot-api';
import type { CapabilitySchema } from '@coze-arch/bot-api';

import type { AddCapabilityForm, EditCapabilityForm } from './types';
import { CAPABILITY_TYPES } from './constants';
import {
  CapabilityRefPicker,
  useResourceOptions,
} from './capability-ref-picker';
import { CapabilityParamList } from './capability-param-list';

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

// ---------- Auto-preview schema when a resource is picked ----------

interface UseSchemaPreviewOptions {
  spaceId: string;
  type: string;
  refId: string;
  refSubId: string;
  onSchema: (schema: CapabilitySchema | undefined) => void;
}

function useSchemaPreview({
  spaceId,
  type,
  refId,
  refSubId,
  onSchema,
}: UseSchemaPreviewOptions) {
  const seqRef = useRef(0);

  useEffect(() => {
    // Only fetch for non-prompt types with a ref_id present
    if (!spaceId || !refId || type === 'prompt') {
      onSchema(undefined);
      return;
    }
    const seq = ++seqRef.current;
    strategyApi
      .previewCapabilitySchema({
        space_id: spaceId,
        type,
        ref_id: refId,
        ref_sub_id: refSubId || undefined,
      })
      .then(resp => {
        if (seq !== seqRef.current) {
          return; // stale — discard
        }
        if ((resp.code === 0 || resp.code === null) && resp.schema) {
          onSchema(resp.schema);
        } else {
          onSchema(undefined);
        }
      })
      .catch(() => {
        if (seq === seqRef.current) {
          onSchema(undefined);
        }
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onSchema is a stable callback, intentionally omitted
  }, [spaceId, type, refId, refSubId]);
}

// ---------- AddCapabilityModal ----------

interface AddCapabilityModalProps {
  visible: boolean;
  form: AddCapabilityForm;
  loading: boolean;
  spaceId: string;
  onOk: () => void;
  onCancel: () => void;
  onFormChange: (patch: Partial<AddCapabilityForm>) => void;
}

export const AddCapabilityModal: React.FC<AddCapabilityModalProps> = ({
  visible,
  form,
  loading,
  spaceId,
  onOk,
  onCancel,
  onFormChange,
}) => {
  const { options, loading: resourceLoading } = useResourceOptions(
    spaceId,
    form.type,
    visible,
  );

  useSchemaPreview({
    spaceId,
    type: form.type,
    refId: form.ref_id,
    refSubId: form.ref_sub_id,
    onSchema: schema => onFormChange({ schema }),
  });

  const isOkDisabled = form.type !== 'prompt' && !form.ref_id.trim();

  return (
    <Modal
      visible={visible}
      title={I18n.t('strategy_capability_add')}
      okText="确定"
      cancelText="取消"
      onOk={onOk}
      onCancel={onCancel}
      okButtonProps={{ loading, disabled: isOkDisabled }}
      style={{ width: 520 }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>类型</Text>
          <Select
            value={form.type}
            onChange={v =>
              onFormChange({ type: String(v), ref_id: '', ref_sub_id: '' })
            }
            style={{ width: '100%' }}
            optionList={CAPABILITY_TYPES.map(t => ({
              label: t.label(),
              value: t.value,
            }))}
          />
        </div>

        {form.type !== 'prompt' && (
          <CapabilityRefPicker
            type={form.type}
            refId={form.ref_id}
            refSubId={form.ref_sub_id}
            spaceId={spaceId}
            visible={visible}
            options={options}
            resourceLoading={resourceLoading}
            onFormChange={onFormChange}
          />
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

        {form.schema ? <CapabilityParamList schema={form.schema} /> : null}

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
};

// ---------- Edit Capability ----------

interface EditCapabilityModalProps {
  visible: boolean;
  form: EditCapabilityForm;
  loading: boolean;
  spaceId: string;
  onOk: () => void;
  onCancel: () => void;
  onFormChange: (patch: Partial<EditCapabilityForm>) => void;
}

export const EditCapabilityModal: React.FC<EditCapabilityModalProps> = ({
  visible,
  form,
  loading,
  spaceId,
  onOk,
  onCancel,
  onFormChange,
}) => {
  const { options: refOptions, loading: refLoading } = useResourceOptions(
    spaceId,
    form.type,
    visible,
  );

  useSchemaPreview({
    spaceId,
    type: form.type,
    refId: form.ref_id,
    refSubId: form.ref_sub_id,
    onSchema: schema => onFormChange({ schema }),
  });

  return (
    <Modal
      visible={visible}
      title="编辑能力项"
      okText="确定"
      cancelText="取消"
      onOk={onOk}
      onCancel={onCancel}
      okButtonProps={{ loading }}
      style={{ width: 520 }}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {/* Type — editable select */}
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>类型</Text>
          <Select
            value={form.type}
            onChange={v =>
              onFormChange({
                type: String(v),
                ref_id: '',
                ref_sub_id: '',
                schema: undefined,
              })
            }
            style={{ width: '100%' }}
            optionList={CAPABILITY_TYPES.map(t => ({
              label: t.label(),
              value: t.value,
            }))}
          />
        </div>

        {/* Resource picker — same as Add modal */}
        {form.type !== 'prompt' && (
          <CapabilityRefPicker
            type={form.type}
            refId={form.ref_id}
            refSubId={form.ref_sub_id}
            spaceId={spaceId}
            visible={visible}
            options={refOptions}
            resourceLoading={refLoading}
            onFormChange={patch =>
              onFormChange(patch as Partial<EditCapabilityForm>)
            }
          />
        )}

        {/* prompt_content — only for prompt type */}
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

        {/* Schema / input params */}
        {form.schema ? <CapabilityParamList schema={form.schema} /> : null}

        {/* alias_name */}
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>别名</Text>
          <Input
            value={form.alias_name}
            onChange={v => onFormChange({ alias_name: v })}
            placeholder="面向模型的能力名称"
          />
        </div>

        {/* alias_description */}
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
};
