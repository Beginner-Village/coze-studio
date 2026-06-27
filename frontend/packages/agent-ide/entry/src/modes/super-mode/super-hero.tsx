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

import { useEffect, useState } from 'react';

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { axiosInstance } from '@coze-arch/bot-api';

import {
  IcPlan,
  IcSandbox,
  IcTools,
  IcMemory,
  IcLightning,
  IcSettings,
} from './icons';

/**
 * 与 super-capabilities-section 同源的能力配置。字段缺省视为开启（默认全开），
 * 沙箱为总开关。这里只读用于 Hero 能力卡片的真实高亮，不做写操作。
 */
interface SuperAgentCapabilityConfig {
  sandbox?: boolean;
  deep_task?: boolean;
  skill_manage?: boolean;
  mcp_servers?: { enabled?: boolean }[];
}

const isOn = (
  config: SuperAgentCapabilityConfig | null,
  key: 'sandbox' | 'deep_task' | 'skill_manage',
): boolean => config?.[key] !== false;

/** 读取真实能力配置（只读）。读取失败保持 null，按默认全开渲染。 */
const useHeroCapabilityConfig = (
  botId: string,
  spaceId: string,
): SuperAgentCapabilityConfig | null => {
  const [config, setConfig] = useState<SuperAgentCapabilityConfig | null>(null);

  useEffect(() => {
    if (!botId) {
      return;
    }
    let cancelled = false;
    (
      axiosInstance.request({
        url: '/api/super-agent/config/get',
        method: 'POST',
        data: { bot_id: botId, space_id: spaceId },
        withCredentials: true,
      }) as unknown as Promise<{
        data?: { config?: SuperAgentCapabilityConfig | null };
      }>
    )
      .then(resp => {
        if (!cancelled) {
          setConfig(resp?.data?.config ?? null);
        }
      })
      .catch(() => {
        /* 读取失败保持默认全开，无需打扰 */
      });
    return () => {
      cancelled = true;
    };
  }, [botId, spaceId]);

  return config;
};

export const SuperAgentHero: React.FC<{ onOpenSettings?: () => void }> = ({
  onOpenSettings,
}) => {
  const botId = useBotInfoStore((state: { botId: string }) => state.botId);
  const spaceId = useBotInfoStore(
    (state: { space_id: string }) => state.space_id,
  );
  const config = useHeroCapabilityConfig(botId, spaceId);

  const sandboxOn = isOn(config, 'sandbox');
  // 沙箱关闭时进入纯 MCP 模式，依赖沙箱的能力随之失效
  const skillMcpOn =
    isOn(config, 'skill_manage') ||
    (config?.mcp_servers?.some(s => s.enabled !== false) ?? false);

  const capabilities = [
    {
      Icon: IcPlan,
      label: '自主规划',
      desc: '拆解任务·多步执行',
      // 自主规划对应子任务委托能力
      active: isOn(config, 'deep_task'),
    },
    {
      Icon: IcSandbox,
      label: '独立沙箱',
      desc: '隔离的文件与运行环境',
      active: sandboxOn,
    },
    {
      Icon: IcTools,
      label: '技能 & MCP',
      desc: '可插拔工具与技能',
      active: skillMcpOn,
    },
    // 长期记忆暂无对应能力开关，保持中性展示，不做假高亮
    {
      Icon: IcMemory,
      label: '长期记忆',
      desc: '跨会话记住偏好',
      active: false,
    },
  ];

  return (
    <div className="flex items-center gap-[12px] px-[16px] py-[12px] min-h-[76px] coz-bg-max border-b coz-stroke-primary shrink-0">
      <div className="flex items-center gap-[6px] shrink-0 coz-fg-hglt pr-[4px]">
        <IcLightning size={15} />
        <span className="text-[14px] font-semibold coz-fg-hglt">
          FinMallClaw 能力
        </span>
      </div>
      <div className="flex items-center gap-[10px] flex-1 min-w-0 overflow-x-auto">
        {capabilities.map(c => (
          <div
            key={c.label}
            className={`flex items-center gap-[10px] h-[52px] px-[14px] rounded-[10px] border coz-stroke-primary coz-bg-max shrink-0 min-w-0 transition-all hover:shadow-sm ${
              c.active
                ? '!border-[rgba(53,138,255,0.28)] !bg-[rgba(53,138,255,0.06)]'
                : ''
            }`}
          >
            <div
              className={`w-[30px] h-[30px] rounded-[8px] flex items-center justify-center shrink-0 ${
                c.active
                  ? 'bg-[rgba(53,138,255,0.12)] coz-fg-hglt'
                  : 'coz-mg-primary coz-fg-secondary'
              }`}
            >
              <c.Icon size={16} />
            </div>
            <div className="leading-tight">
              <div className="text-[13px] font-semibold coz-fg-plus whitespace-nowrap">
                {c.label}
              </div>
              <div className="text-[11px] coz-fg-secondary whitespace-nowrap overflow-hidden text-ellipsis">
                {c.desc}
              </div>
            </div>
          </div>
        ))}
      </div>
      <button
        type="button"
        onClick={onOpenSettings}
        className="flex items-center gap-[7px] shrink-0 cursor-pointer h-[36px] px-[14px] rounded-[9px] border border-[rgba(167,0,250,0.3)] bg-[rgba(167,0,250,0.08)] text-[rgb(167,0,250)] hover:bg-[rgba(167,0,250,0.14)] transition-colors"
      >
        <IcSettings size={15} />
        <span className="text-[13px] font-semibold">人设 · 技能 · MCP</span>
      </button>
    </div>
  );
};
