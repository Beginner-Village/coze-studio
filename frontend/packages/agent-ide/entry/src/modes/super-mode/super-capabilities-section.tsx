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

import { useCallback, useEffect, useRef, useState } from 'react';

import classNames from 'classnames';
import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { Switch, Toast, Modal, Input, Select } from '@coze-arch/coze-design';
import { axiosInstance } from '@coze-arch/bot-api';

import { IcSettings, IcPlus, IcTrash } from './icons';

import cs from './super-config-area.module.less';

/** 远程 MCP server 配置项。后端只支持远程 streamable_http / sse，stdio 不支持。 */
interface McpServer {
  name: string;
  type: 'streamable_http' | 'sse';
  url: string;
  env?: Record<string, string>;
  enabled?: boolean;
}

/**
 * super-agent 能力与权限配置。字段全部可选 bool，省略或 true = 开启（默认全开）。
 * sandbox 为沙箱总开关，关闭后 web_search / web_fetch / run_bash 依赖沙箱而失效。
 */
interface SuperAgentCapabilityConfig {
  sandbox?: boolean;
  web_search?: boolean;
  web_fetch?: boolean;
  run_bash?: boolean;
  deep_task?: boolean;
  skill_manage?: boolean;
  mcp_servers?: McpServer[];
}

type CapabilityKey =
  | 'sandbox'
  | 'web_search'
  | 'web_fetch'
  | 'run_bash'
  | 'deep_task'
  | 'skill_manage';

interface CapabilityMeta {
  key: CapabilityKey;
  label: string;
  desc: string;
  /** 是否依赖沙箱：沙箱关闭时置灰禁用 */
  needsSandbox?: boolean;
}

const CAPABILITIES: CapabilityMeta[] = [
  {
    key: 'sandbox',
    label: '沙箱环境',
    desc: '总开关。关闭后进入纯 MCP 模式，仅 MCP 工具与对话可用',
  },
  {
    key: 'web_search',
    label: '网络搜索',
    desc: '联网检索资料（依赖沙箱）',
    needsSandbox: true,
  },
  {
    key: 'web_fetch',
    label: '网络抓取',
    desc: '抓取并解析网页内容（依赖沙箱）',
    needsSandbox: true,
  },
  {
    key: 'run_bash',
    label: '命令执行 run_bash',
    desc: '在沙箱内执行 shell 命令（依赖沙箱）',
    needsSandbox: true,
  },
  {
    key: 'deep_task',
    label: '子任务委托',
    desc: '拆解并委托子任务给子智能体',
  },
  {
    key: 'skill_manage',
    label: '技能管理',
    desc: '创建、编辑与管理技能文件夹',
  },
];

/** config 为 null 或字段缺失时默认开启 */
const resolveSwitch = (
  config: SuperAgentCapabilityConfig | null,
  key: CapabilityKey,
): boolean => config?.[key] !== false;

interface CapabilityApiResponse {
  code?: number;
  msg?: string;
  data?: { config?: SuperAgentCapabilityConfig | null };
}

const postCapabilityApi = (
  url: string,
  body: Record<string, unknown>,
): Promise<CapabilityApiResponse> =>
  axiosInstance.request({
    url,
    method: 'POST',
    data: body,
    withCredentials: true,
  }) as unknown as Promise<CapabilityApiResponse>;

const useCapabilityConfig = (botId: string, spaceId: string) => {
  const [config, setConfig] = useState<SuperAgentCapabilityConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const configRef = useRef<SuperAgentCapabilityConfig | null>(null);
  configRef.current = config;

  useEffect(() => {
    if (!botId) {
      return;
    }
    let cancelled = false;
    postCapabilityApi('/api/super-agent/config/get', {
      bot_id: botId,
      space_id: spaceId,
    })
      .then(resp => {
        if (!cancelled) {
          setConfig(resp?.data?.config ?? null);
        }
      })
      .catch(() => {
        /* 读取失败时保持默认全开，无需打扰 */
      });
    return () => {
      cancelled = true;
    };
  }, [botId, spaceId]);

  // 用一份覆盖项组装完整 config（6 开关 + mcp_servers 一起），乐观更新失败回滚
  const saveConfig = useCallback(
    async (overrides: Partial<SuperAgentCapabilityConfig>) => {
      if (!botId) {
        return;
      }
      const prev = configRef.current;
      const nextConfig: SuperAgentCapabilityConfig = {
        sandbox: resolveSwitch(prev, 'sandbox'),
        web_search: resolveSwitch(prev, 'web_search'),
        web_fetch: resolveSwitch(prev, 'web_fetch'),
        run_bash: resolveSwitch(prev, 'run_bash'),
        deep_task: resolveSwitch(prev, 'deep_task'),
        skill_manage: resolveSwitch(prev, 'skill_manage'),
        mcp_servers: prev?.mcp_servers ?? [],
        ...overrides,
      };
      setConfig(nextConfig);
      setSaving(true);
      try {
        const resp = await postCapabilityApi('/api/super-agent/config/update', {
          bot_id: botId,
          space_id: spaceId,
          config: nextConfig,
        });
        if (resp?.code && resp.code !== 0) {
          throw new Error(resp.msg || 'update failed');
        }
        setConfig(resp?.data?.config ?? nextConfig);
        Toast.success('已保存');
      } catch {
        setConfig(prev);
        Toast.error('保存失败');
      } finally {
        setSaving(false);
      }
    },
    [botId, spaceId],
  );

  const updateSwitch = useCallback(
    (key: CapabilityKey, value: boolean) => saveConfig({ [key]: value }),
    [saveConfig],
  );

  const saveMcpServers = useCallback(
    (mcpServers: McpServer[]) => saveConfig({ mcp_servers: mcpServers }),
    [saveConfig],
  );

  return { config, saving, updateSwitch, saveMcpServers };
};

const MCP_TYPE_OPTIONS = [
  { label: 'Streamable HTTP（推荐）', value: 'streamable_http' },
  { label: 'SSE', value: 'sse' },
];

interface McpFormState {
  name: string;
  type: 'streamable_http' | 'sse';
  url: string;
  authValue: string;
}

const EMPTY_MCP_FORM: McpFormState = {
  name: '',
  type: 'streamable_http',
  url: '',
  authValue: '',
};

const McpServersSubsection: React.FC<{
  servers: McpServer[];
  saving: boolean;
  onSave: (servers: McpServer[]) => void;
}> = ({ servers, saving, onSave }) => {
  const [modalOpen, setModalOpen] = useState(false);
  const [form, setForm] = useState<McpFormState>(EMPTY_MCP_FORM);

  const openModal = useCallback(() => {
    setForm(EMPTY_MCP_FORM);
    setModalOpen(true);
  }, []);

  const handleRemove = useCallback(
    (index: number) => {
      onSave(servers.filter((_, i) => i !== index));
    },
    [servers, onSave],
  );

  const handleConfirm = useCallback(() => {
    const url = form.url.trim();
    if (!url) {
      Toast.error('请填写 URL');
      return;
    }
    if (!/^https?:\/\//i.test(url)) {
      Toast.error('URL 需为 http(s) 远程地址');
      return;
    }
    const name = form.name.trim() || url;
    const env = form.authValue.trim()
      ? { Authorization: form.authValue.trim() }
      : {};
    const next: McpServer = {
      name,
      type: form.type,
      url,
      env,
      enabled: true,
    };
    onSave([...servers, next]);
    setModalOpen(false);
  }, [form, servers, onSave]);

  return (
    <div className={cs.mcpBlock}>
      <div className={cs.mcpHead}>
        <div className={cs.mcpHeadInfo}>
          <div className={cs.capName}>MCP 服务</div>
          <div className={cs.capDesc}>
            接入远程 MCP server（streamable_http / sse）扩展工具
          </div>
        </div>
        <button
          type="button"
          className={cs.mcpAddBtn}
          disabled={saving}
          onClick={openModal}
        >
          <IcPlus size={14} />
          添加 MCP 服务
        </button>
      </div>
      {servers.length === 0 ? (
        <div className={cs.mcpEmpty}>
          还没有 MCP 服务，点击添加远程 MCP server
        </div>
      ) : (
        <div className={cs.mcpList}>
          {servers.map((srv, index) => (
            <div className={cs.mcpRow} key={`${srv.name}-${index}`}>
              <div className={cs.mcpRowInfo}>
                <div className={cs.mcpRowTop}>
                  <span className={cs.mcpName}>{srv.name}</span>
                  <span className={cs.mcpType}>{srv.type}</span>
                </div>
                <div className={cs.mcpUrl}>{srv.url}</div>
              </div>
              <button
                type="button"
                className={cs.mcpDelBtn}
                disabled={saving}
                aria-label="删除"
                onClick={() => handleRemove(index)}
              >
                <IcTrash size={15} />
              </button>
            </div>
          ))}
        </div>
      )}
      <Modal
        visible={modalOpen}
        title="添加 MCP 服务"
        onCancel={() => setModalOpen(false)}
        onOk={handleConfirm}
        okText="添加"
        cancelText="取消"
      >
        <div className={cs.mcpForm}>
          <div className={cs.mcpField}>
            <div className={cs.mcpFieldLabel}>名称</div>
            <Input
              value={form.name}
              placeholder="便于识别的名称，留空则用 URL"
              onChange={(value: string) =>
                setForm(prev => ({ ...prev, name: value }))
              }
            />
          </div>
          <div className={cs.mcpField}>
            <div className={cs.mcpFieldLabel}>类型</div>
            <Select
              style={{ width: '100%' }}
              value={form.type}
              optionList={MCP_TYPE_OPTIONS}
              onChange={(value: unknown) =>
                setForm(prev => ({
                  ...prev,
                  type: value as 'streamable_http' | 'sse',
                }))
              }
            />
          </div>
          <div className={cs.mcpField}>
            <div className={cs.mcpFieldLabel}>
              URL<span className={cs.required}> *</span>
            </div>
            <Input
              value={form.url}
              placeholder="https:// 远程 MCP server 地址"
              onChange={(value: string) =>
                setForm(prev => ({ ...prev, url: value }))
              }
            />
          </div>
          <div className={cs.mcpField}>
            <div className={cs.mcpFieldLabel}>认证头（可选）</div>
            <Input
              value={form.authValue}
              placeholder="Authorization 值，如 Bearer xxx"
              onChange={(value: string) =>
                setForm(prev => ({ ...prev, authValue: value }))
              }
            />
          </div>
        </div>
      </Modal>
    </div>
  );
};

export const SuperCapabilitiesSection: React.FC = () => {
  const botId = useBotInfoStore((state: { botId: string }) => state.botId);
  const spaceId = useBotInfoStore(
    (state: { space_id: string }) => state.space_id,
  );
  const { config, saving, updateSwitch, saveMcpServers } = useCapabilityConfig(
    botId,
    spaceId,
  );

  const sandboxOn = resolveSwitch(config, 'sandbox');
  const mcpServers = config?.mcp_servers ?? [];

  return (
    <section className={cs.capsSection}>
      <div className={cs.capsHeader}>
        <div className={cs.capsTitle}>
          <IcSettings size={16} />
          能力与权限
        </div>
      </div>
      <div className={cs.capsBody}>
        {CAPABILITIES.map(item => {
          const disabled = Boolean(item.needsSandbox) && !sandboxOn;
          const checked = disabled ? false : resolveSwitch(config, item.key);
          return (
            <div
              key={item.key}
              className={classNames(cs.capRow, disabled && cs.capRowDisabled)}
            >
              <div className={cs.capInfo}>
                <div className={cs.capName}>{item.label}</div>
                <div className={cs.capDesc}>{item.desc}</div>
              </div>
              <Switch
                size="small"
                checked={checked}
                disabled={disabled || saving}
                onChange={(value: boolean) => updateSwitch(item.key, value)}
              />
            </div>
          );
        })}
        {!sandboxOn ? (
          <div className={cs.capsHint}>纯 MCP 模式：仅 MCP 工具与对话可用</div>
        ) : null}
        <McpServersSubsection
          servers={mcpServers}
          saving={saving}
          onSave={saveMcpServers}
        />
      </div>
    </section>
  );
};
