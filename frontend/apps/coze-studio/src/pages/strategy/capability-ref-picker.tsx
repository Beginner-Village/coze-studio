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

import { PublishStatus, ResType } from '@coze-arch/idl/plugin_develop';
import type { PluginAPIInfo } from '@coze-arch/idl/plugin_develop';
import { Select, Spin, Typography } from '@coze-arch/coze-design';
import { PluginDevelopApi } from '@coze-arch/bot-api';

import type { AddCapabilityForm } from './types';

const { Text } = Typography;

// ---------- Shared resource option ----------

export interface ResourceOption {
  res_id: string;
  name: string;
  description?: string;
  // workflow 未发布时为 true:不可被引用(run 用 FromLatestVersion 需已发布版本),下拉中标灰禁选
  disabled?: boolean;
}

// ---------- Hook: workflow / knowledge list ----------

export function useResourceOptions(
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
      size: 100,
    })
      .then(resp => {
        if (resp.code === 0 || resp.code === null || resp.code === undefined) {
          const mapped = (resp.resource_list ?? []).map(r => ({
            res_id: r.res_id ?? '',
            name: r.name ?? r.res_id ?? '',
            description: r.description,
            // 只有工作流要求"已发布"才能被引用;知识库始终可用
            disabled:
              type === 'workflow' &&
              r.publish_status !== PublishStatus.Published,
          }));
          // 已发布(可选)的排在前面,未发布(禁选)沉底
          mapped.sort((a, b) => Number(a.disabled) - Number(b.disabled));
          setOptions(mapped);
        }
      })
      .catch(() => setOptions([]))
      .finally(() => setLoading(false));
  }, [spaceId, type, visible]);

  return { options, loading };
}

// ---------- Hook: plugin list ----------

function usePluginList(
  spaceId: string,
  visible: boolean,
): { plugins: ResourceOption[]; loading: boolean } {
  const [plugins, setPlugins] = useState<ResourceOption[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!visible || !spaceId) {
      setPlugins([]);
      return;
    }
    setLoading(true);
    PluginDevelopApi.LibraryResourceList({
      space_id: spaceId,
      res_type_filter: [ResType.Plugin],
      size: 100,
    })
      .then(resp => {
        if (resp.code === 0 || resp.code === null || resp.code === undefined) {
          setPlugins(
            (resp.resource_list ?? []).map(r => ({
              res_id: r.res_id ?? '',
              name: r.name ?? r.res_id ?? '',
            })),
          );
        }
      })
      .catch(() => setPlugins([]))
      .finally(() => setLoading(false));
  }, [spaceId, visible]);

  return { plugins, loading };
}

// ---------- Hook: tools within a plugin ----------

function usePluginTools(
  pluginId: string,
  visible: boolean,
): { tools: PluginAPIInfo[]; loading: boolean } {
  const [tools, setTools] = useState<PluginAPIInfo[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!visible || !pluginId) {
      setTools([]);
      return;
    }
    setLoading(true);
    PluginDevelopApi.GetPluginAPIs({
      plugin_id: pluginId,
      page: 1,
      size: 20,
    })
      .then(resp => {
        if (resp.code === 0 || resp.code === null || resp.code === undefined) {
          setTools(resp.api_info ?? []);
        }
      })
      .catch(() => setTools([]))
      .finally(() => setLoading(false));
  }, [pluginId, visible]);

  return { tools, loading };
}

// ---------- Plugin cascading picker (level-1: plugin, level-2: tool) ----------

interface PluginPickerProps {
  refId: string;
  refSubId: string;
  spaceId: string;
  visible: boolean;
  onFormChange: (patch: Partial<AddCapabilityForm>) => void;
}

const PluginPicker: React.FC<PluginPickerProps> = ({
  refId,
  refSubId,
  spaceId,
  visible,
  onFormChange,
}) => {
  const { plugins, loading: pluginsLoading } = usePluginList(spaceId, visible);
  const { tools, loading: toolsLoading } = usePluginTools(refSubId, visible);

  return (
    <>
      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>选择插件</Text>
        {pluginsLoading ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Spin size="small" />
            <Text type="tertiary">加载中...</Text>
          </div>
        ) : (
          <Select
            value={refSubId || undefined}
            onChange={v => {
              const selected = plugins.find(p => p.res_id === String(v));
              onFormChange({
                ref_sub_id: String(v),
                ref_id: '',
                alias_name: selected?.name ?? '',
                schema: undefined,
              });
            }}
            style={{ width: '100%' }}
            filter
            showClear
            placeholder={
              plugins.length === 0 ? '该空间暂无可用插件' : '搜索并选择插件...'
            }
            emptyContent="该空间暂无可用插件"
            optionList={plugins.map(p => ({
              label: p.name,
              value: p.res_id,
              showTick: true,
            }))}
          />
        )}
      </div>

      <div>
        <Text style={{ display: 'block', marginBottom: 4 }}>选择工具</Text>
        {toolsLoading ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Spin size="small" />
            <Text type="tertiary">加载中...</Text>
          </div>
        ) : (
          <Select
            value={refId || undefined}
            disabled={!refSubId}
            onChange={v => {
              const selected = tools.find(t => t.api_id === String(v));
              onFormChange({
                ref_id: String(v),
                alias_name: selected?.name ?? '',
                alias_description: selected?.desc ?? '',
              });
            }}
            style={{ width: '100%' }}
            filter
            showClear
            placeholder={
              !refSubId
                ? '请先选择插件'
                : tools.length === 0
                  ? '该插件暂无可用工具'
                  : '搜索并选择工具...'
            }
            emptyContent="该插件暂无可用工具"
            optionList={tools.map(t => ({
              label: t.name ?? t.api_id ?? '',
              value: t.api_id ?? '',
              showTick: true,
            }))}
          />
        )}
      </div>
    </>
  );
};

// ---------- CapabilityRefPicker: routes to workflow/knowledge/plugin picker ----------

export interface CapabilityRefPickerProps {
  type: string;
  refId: string;
  refSubId: string;
  spaceId: string;
  visible: boolean;
  options: ResourceOption[];
  resourceLoading: boolean;
  onFormChange: (patch: Partial<AddCapabilityForm>) => void;
}

export const CapabilityRefPicker: React.FC<CapabilityRefPickerProps> = ({
  type,
  refId,
  refSubId,
  spaceId,
  visible,
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
              label: o.disabled ? `${o.name}(未发布,不可用)` : o.name,
              value: o.res_id,
              disabled: o.disabled,
              showTick: true,
            }))}
          />
        )}
      </div>
    );
  }

  if (type === 'plugin') {
    return (
      <PluginPicker
        refId={refId}
        refSubId={refSubId}
        spaceId={spaceId}
        visible={visible}
        onFormChange={onFormChange}
      />
    );
  }

  return null;
};
