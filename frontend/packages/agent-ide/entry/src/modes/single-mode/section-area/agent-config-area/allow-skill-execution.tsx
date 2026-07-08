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

import { type FC, useCallback, useEffect, useRef, useState } from 'react';

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { useBotDetailIsReadonly } from '@coze-studio/bot-detail-store';
import { I18n } from '@coze-arch/i18n';
import { Switch, Toast, Typography } from '@coze-arch/coze-design';
import { axiosInstance } from '@coze-arch/bot-api';

/**
 * 普通(非超级)智能体的「允许执行技能脚本(沙箱)」per-agent 开关。
 *
 * 关闭后，该智能体的技能退化为只读文档：只读取技能说明(SKILL.md)，不在沙箱执行脚本、
 * 也不挂载 run_bash 等沙箱工具。默认允许（字段缺失 / 非 false = 允许，向后兼容，
 * 与后端 SkillExecution *bool 的 nil=允许 语义一致）。
 *
 * 持久化复用 super-agent 能力配置端点，落到 single_agent_draft.super_agent_tool_config
 * 列的 skill_execution 字段。该端点只做登录态属主校验、不限 super 类型，普通体 botId 可用。
 */
interface AgentToolConfig {
  skill_execution?: boolean;
  [key: string]: unknown;
}

interface ToolConfigApiResponse {
  code?: number;
  msg?: string;
  data?: { config?: AgentToolConfig | null };
}

const postToolConfigApi = (
  url: string,
  body: Record<string, unknown>,
): Promise<ToolConfigApiResponse> =>
  axiosInstance.request({
    url,
    method: 'POST',
    data: body,
    withCredentials: true,
  }) as unknown as Promise<ToolConfigApiResponse>;

export const AllowSkillExecution: FC = () => {
  const botId = useBotInfoStore(state => state.botId);
  const spaceId = useBotInfoStore(state => state.space_id);
  const isReadonly = useBotDetailIsReadonly();

  const [config, setConfig] = useState<AgentToolConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const configRef = useRef<AgentToolConfig | null>(null);
  configRef.current = config;

  useEffect(() => {
    if (!botId) {
      return;
    }
    let cancelled = false;
    postToolConfigApi('/api/super-agent/config/get', {
      bot_id: botId,
      space_id: spaceId,
    })
      .then(resp => {
        if (!cancelled) {
          setConfig(resp?.data?.config ?? null);
        }
      })
      .catch(() => {
        /* 读取失败时保持默认(允许)，不打扰用户 */
      });
    return () => {
      cancelled = true;
    };
  }, [botId, spaceId]);

  // 字段缺失 / 非 false 视为「允许」，与后端 nil=允许 一致。切勿写成 === true。
  const allowed = config?.skill_execution !== false;

  const onChange = useCallback(
    async (checked: boolean) => {
      if (!botId) {
        return;
      }
      const prev = configRef.current;
      // 保留已有其它字段，只覆盖 skill_execution，避免清掉别的配置。
      const nextConfig: AgentToolConfig = {
        ...(prev ?? {}),
        skill_execution: checked,
      };
      setConfig(nextConfig);
      setSaving(true);
      try {
        const resp = await postToolConfigApi('/api/super-agent/config/update', {
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

  return (
    <div
      style={{
        padding: '12px 16px',
        borderBottom: '1px solid var(--semi-color-border)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ flex: 1 }}>
          <Typography.Text strong>{I18n.t('允许执行技能脚本')}</Typography.Text>
          <Typography.Paragraph
            type="tertiary"
            style={{ margin: '4px 0 0 0', fontSize: '12px' }}
          >
            {I18n.t(
              '关闭后仅读取技能说明(SKILL.md)，不在沙箱执行脚本，也不挂载 run_bash 等沙箱工具',
            )}
          </Typography.Paragraph>
        </div>
        <Switch
          checked={allowed}
          disabled={isReadonly || saving || !botId}
          onChange={onChange}
          data-testid="bot.editor.tool.allow-skill-execution.switch"
        />
      </div>
    </div>
  );
};
