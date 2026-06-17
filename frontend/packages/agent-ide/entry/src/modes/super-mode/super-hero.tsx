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

import {
  IcPlan,
  IcSandbox,
  IcTools,
  IcMemory,
  IcLightning,
  IcSettings,
} from './icons';

const CAPABILITIES = [
  { Icon: IcPlan, label: '自主规划', desc: '拆解任务·多步执行' },
  { Icon: IcSandbox, label: '独立沙箱', desc: '隔离的文件与运行环境' },
  { Icon: IcTools, label: '技能 & MCP', desc: '可插拔工具与技能' },
  { Icon: IcMemory, label: '长期记忆', desc: '跨会话记住偏好' },
];

/**
 * 超级智能体能力条:纤细、浅色、与系统配色一致。
 * 右侧「⚙ 设置」打开人设/技能/MCP 弹框 —— 主页面留给「左工作区 + 右聊天」。
 */
export const SuperAgentHero: React.FC<{ onOpenSettings?: () => void }> = ({
  onOpenSettings,
}) => (
  <div className="flex items-center gap-[16px] px-[20px] h-[52px] coz-bg-plus border-b coz-stroke-primary shrink-0">
    <div className="flex items-center gap-[7px] shrink-0 coz-fg-hglt">
      <IcLightning size={15} />
      <span className="text-[14px] font-semibold coz-fg-plus">
        超级智能体能力
      </span>
    </div>
    <div className="h-[16px] w-px coz-stroke-primary" />
    <div className="flex items-center gap-[8px] flex-1 overflow-x-auto">
      {CAPABILITIES.map(c => (
        <div
          key={c.label}
          className="flex items-center gap-[8px] px-[12px] py-[6px] rounded-[10px] coz-mg-secondary shrink-0"
        >
          <c.Icon size={16} className="coz-fg-secondary" />
          <div className="leading-tight">
            <div className="text-[12px] font-medium coz-fg-primary">
              {c.label}
            </div>
            <div className="text-[11px] coz-fg-dim">{c.desc}</div>
          </div>
        </div>
      ))}
    </div>
    <div
      onClick={onOpenSettings}
      className="flex items-center gap-[6px] shrink-0 cursor-pointer px-[12px] py-[6px] rounded-[10px] coz-mg-hglt coz-fg-hglt hover:opacity-80 transition-opacity"
    >
      <IcSettings size={15} />
      <span className="text-[13px] font-medium">人设 · 技能 · MCP</span>
    </div>
  </div>
);
