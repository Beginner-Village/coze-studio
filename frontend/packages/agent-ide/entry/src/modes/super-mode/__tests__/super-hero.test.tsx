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

import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';

import { axiosInstance } from '@coze-arch/bot-api';

import { SuperAgentHero } from '../super-hero';

vi.mock('@coze-studio/bot-detail-store/bot-info', () => ({
  useBotInfoStore: (selector: (state: unknown) => unknown) =>
    selector({ botId: 'bot-1', space_id: 'space-1' }),
}));

vi.mock('@coze-arch/bot-api', () => ({
  axiosInstance: {
    request: vi.fn(),
  },
}));

const request = axiosInstance.request as unknown as ReturnType<typeof vi.fn>;

/** 找到某个能力卡片的最外层容器（用于断言高亮 class） */
const cardOf = (label: string): HTMLElement =>
  screen.getByText(label).closest('div.flex.items-center') as HTMLElement;

describe('SuperAgentHero', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the settings entry', async () => {
    request.mockResolvedValueOnce({ data: { config: null } });
    const { container } = render(<SuperAgentHero />);

    expect(screen.getByText('人设 · 技能 · MCP')).toBeTruthy();
    expect(container.querySelector('.overflow-x-auto')).toBeTruthy();
  });

  it('reads real capability config and highlights only active cards', async () => {
    request.mockResolvedValueOnce({
      data: { config: { sandbox: false, deep_task: true, skill_manage: true } },
    });
    render(<SuperAgentHero />);

    await waitFor(() =>
      expect(request).toHaveBeenCalledWith(
        expect.objectContaining({
          url: '/api/super-agent/config/get',
          data: { bot_id: 'bot-1', space_id: 'space-1' },
        }),
      ),
    );

    await waitFor(() => {
      // sandbox off -> 独立沙箱 not highlighted
      expect(cardOf('独立沙箱').className).not.toContain('rgba(53,138,255,0.28)');
    });
    // deep_task on -> 自主规划 highlighted; skill_manage on -> 技能 & MCP highlighted
    expect(cardOf('自主规划').className).toContain('rgba(53,138,255,0.28)');
    expect(cardOf('技能 & MCP').className).toContain('rgba(53,138,255,0.28)');
    // 长期记忆 has no backing switch -> never highlighted
    expect(cardOf('长期记忆').className).not.toContain('rgba(53,138,255,0.28)');
  });

  it('keeps long-term-memory card neutral even when config defaults all on', async () => {
    request.mockResolvedValueOnce({ data: { config: null } });
    render(<SuperAgentHero />);

    await waitFor(() => expect(request).toHaveBeenCalled());

    // 默认全开时三项高亮，长期记忆仍保持中性（无假高亮）
    await waitFor(() =>
      expect(cardOf('独立沙箱').className).toContain('rgba(53,138,255,0.28)'),
    );
    expect(cardOf('长期记忆').className).not.toContain('rgba(53,138,255,0.28)');
  });
});
