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

import { useBotInfoStore } from '@coze-studio/bot-detail-store/bot-info';
import { Avatar } from '@coze-arch/coze-design';

const CAPABILITIES = [
  { icon: '🧠', label: '自主规划', desc: '拆解任务·多步执行' },
  { icon: '📦', label: '独立沙箱', desc: '隔离的文件与运行环境' },
  { icon: '🔧', label: '技能 & MCP', desc: '可插拔工具与技能' },
  { icon: '💾', label: '长期记忆', desc: '跨会话记住偏好' },
];

export const SuperAgentHero: React.FC = () => {
  const name = useBotInfoStore(state => state.name);
  const iconUrl = useBotInfoStore(state => state.icon_url);

  return (
    <div
      className="relative flex items-center gap-[20px] px-[24px] py-[16px] overflow-hidden"
      style={{
        background:
          'linear-gradient(108deg, #2B1C66 0%, #4332A8 38%, #6D4AE0 72%, #8A63F4 100%)',
      }}
    >
      {/* 背景装饰光斑 */}
      <div
        className="absolute pointer-events-none"
        style={{
          right: '-40px',
          top: '-60px',
          width: '260px',
          height: '260px',
          borderRadius: '50%',
          background:
            'radial-gradient(circle, rgba(180,150,255,0.35) 0%, rgba(180,150,255,0) 70%)',
        }}
      />

      {/* 身份 */}
      <Avatar
        src={iconUrl}
        size="large"
        className="shrink-0 ring-2 ring-white/30"
      />
      <div className="shrink-0 min-w-[180px]">
        <div className="flex items-center gap-[8px]">
          <span className="text-white text-[18px] font-semibold leading-[24px]">
            {name || '超级智能体'}
          </span>
          <span className="text-[11px] px-[8px] py-[2px] rounded-[6px] bg-white/20 text-white font-medium">
            超级智能体
          </span>
        </div>
        <div className="text-[12px] text-white/70 mt-[4px]">
          自主规划任务 · 独立沙箱空间 · 技能与 MCP 工具
        </div>
      </div>

      {/* 能力胶囊 */}
      <div className="flex items-center gap-[10px] flex-1 justify-end flex-wrap">
        {CAPABILITIES.map(c => (
          <div
            key={c.label}
            className="flex items-center gap-[10px] px-[14px] py-[8px] rounded-[12px] backdrop-blur-sm"
            style={{
              background: 'rgba(255,255,255,0.12)',
              border: '1px solid rgba(255,255,255,0.16)',
            }}
          >
            <span className="text-[18px] leading-none">{c.icon}</span>
            <div className="leading-tight">
              <div className="text-[13px] text-white font-medium">
                {c.label}
              </div>
              <div className="text-[11px] text-white/60">{c.desc}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
