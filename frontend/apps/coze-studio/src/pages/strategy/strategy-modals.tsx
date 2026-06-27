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

import React, { useEffect, useState } from 'react';

import { ResType } from '@coze-arch/idl/plugin_develop';
import { I18n } from '@coze-arch/i18n';
import {
  Input,
  Modal,
  Select,
  Spin,
  Tag,
  TextArea,
  Typography,
} from '@coze-arch/coze-design';
import { PluginDevelopApi } from '@coze-arch/bot-api';

import type { AddCapabilityForm, EditCapabilityForm } from './types';
import {
  CAPABILITY_TYPE_LABELS,
  CAPABILITY_TYPES,
  TYPE_COLORS,
} from './constants';
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

// ---------- Add Capability ----------

interface ResourceOption {
  res_id: string;
  name: string;
  description?: string;
}

function useResourceOptions(
  spaceId: string,
  type: string,
  visible: boolean,
): { options: ResourceOption[]; loading: boolean } {
  const [options, setOptions] = useState<ResourceOption[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!visible || !spaceId || type === 'prompt' || type === 'plugin') {
      setOptions([]);
      return;
    }
    const resType = type === 'workflow' ? ResType.Workflow : ResType.Knowledge;
    setLoading(true);
    PluginDevelopApi.LibraryResourceList({
      space_id: spaceId,
      res_type_filter: [resType],
      size: 200,
    })
      .then(resp => {
        if (resp.code === 0 || resp.code === null || resp.code === undefined) {
          setOptions(
            (resp.resource_list ?? []).map(r => ({
              res_id: r.res_id ?? '',
              name: r.name ?? r.res_id ?? '',
              description: r.description,
            })),
          );
        }
      })
      .catch(() => setOptions([]))
      .finally(() => setLoading(false));
  }, [spaceId, type, visible]);

  return { options, loading };
}

// ---------- Resource picker (workflow / knowledge) ----------

interface CapabilityRefPickerProps {
  type: string;
  refId: string;
  refSubId: string;
  options: ResourceOption[];
  resourceLoading: boolean;
  onFormChange: (patch: Partial<AddCapabilityForm>) => void;
}

const CapabilityRefPicker: React.FC<CapabilityRefPickerProps> = ({
  type,
  refId,
  refSubId,
  options,
  resourceLoading,
  onFormChange,
}) => {
  const emptyLabel =
    type === 'workflow' ? '该空间暂无可用工作流' : '该空间暂无可用知识库';
  const kindLabel = type === 'workflow' ? '工作流' : '知识库';

  if (type === 'workflow' || type === 'knowledge') {
    return (
      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>
          {type === 'workflow' ? '选择工作流' : '选择知识库'}
        </Text>
        {resourceLoading ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Spin size="small" />
            <Text type="tertiary">加载中...</Text>
          </div>
        ) : (
          <Select
            value={refId || undefined}
            onChange={v => {
              const selected = options.find(o => o.res_id === String(v));
              onFormChange({
                ref_id: String(v),
                alias_name: selected?.name ?? '',
                alias_description: selected?.description ?? '',
              });
            }}
            style={{ width: '100%' }}
            filter
            showClear
            placeholder={
              options.length === 0 ? emptyLabel : `搜索并选择${kindLabel}...`
            }
            emptyContent={emptyLabel}
            optionList={options.map(o => ({
              label: o.name,
              value: o.res_id,
              showTick: true,
            }))}
          />
        )}
      </div>
    );
  }

  if (type === 'plugin') {
    return (
      <>
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>
            Tool ID (ref_id)
          </Text>
          <Input
            value={refId}
            onChange={v => onFormChange({ ref_id: v })}
            placeholder="工具 ID（数字）"
            type="number"
          />
        </div>
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>
            Plugin ID (ref_sub_id)
          </Text>
          <Input
            value={refSubId}
            onChange={v => onFormChange({ ref_sub_id: v })}
            placeholder="插件 ID（数字）"
            type="number"
          />
          <Text
            type="tertiary"
            style={{ display: 'block', marginTop: 4, fontSize: 12 }}
          >
            提示：可在插件管理页面查看对应的 Tool ID 和 Plugin ID
          </Text>
        </div>
      </>
    );
  }

  return null;
};

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

  // Resolve the display name for the bound resource (workflow / knowledge).
  // Falls back to the raw ref_id if the resource list hasn't loaded yet or
  // the id isn't found.
  const boundName =
    form.type !== 'prompt' && form.ref_id
      ? (refOptions.find(o => o.res_id === form.ref_id)?.name ?? form.ref_id)
      : null;

  const typeLabel = CAPABILITY_TYPE_LABELS[form.type]?.() ?? form.type;
  const typeColor = TYPE_COLORS[form.type] ?? 'default';

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
        {/* Type (read-only) */}
        <div>
          <Text style={{ display: 'block', marginBottom: 4 }}>类型</Text>
          <Tag color={typeColor}>{typeLabel}</Tag>
        </div>

        {/* Bound resource name (workflow / knowledge / plugin) */}
        {form.type !== 'prompt' && form.ref_id ? (
          <div>
            <Text style={{ display: 'block', marginBottom: 4 }}>
              {form.type === 'workflow'
                ? '绑定工作流'
                : form.type === 'knowledge'
                  ? '绑定知识库'
                  : '绑定插件工具'}
            </Text>
            {refLoading ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Spin size="small" />
                <Text type="tertiary">加载中...</Text>
              </div>
            ) : (
              <Text type="secondary">{boundName}</Text>
            )}
          </div>
        ) : null}

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
              rows={3}
            />
          </div>
        )}
      </div>
    </Modal>
  );
};
