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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import SkillMarketplacePage from '../index';

const navigate = vi.fn();
const listSkills = vi.fn();
const installSkill = vi.fn();

vi.mock('react-router-dom', () => ({
  useNavigate: () => navigate,
  useSearchParams: () => [new URLSearchParams('space_id=space-1')],
}));

vi.mock('@coze-arch/bot-api', () => ({
  axiosInstance: { request: vi.fn().mockResolvedValue({ code: 0, data: {} }) },
}));

vi.mock('@coze-studio/api-schema', () => ({
  skill: {
    SuperAgentMarketplaceListSkills: (...args: unknown[]) =>
      listSkills(...args),
    SuperAgentInstallMarketplaceSkill: (...args: unknown[]) =>
      installSkill(...args),
  },
}));

vi.mock('@coze-arch/coze-design/icons', () => {
  const Icon = () => <span data-testid="icon" />;
  return {
    IconCozDownload: Icon,
    IconCozEye: Icon,
    IconCozFolder: Icon,
    IconCozImport: Icon,
    IconCozMagnifier: Icon,
    IconCozPeople: Icon,
    IconCozPlugin: Icon,
    IconCozStarFill: Icon,
    IconCozStore: Icon,
  };
});

vi.mock('@coze-arch/coze-design', () => ({
  Button: ({
    children,
    onClick,
    disabled,
  }: {
    children?: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button type="button" disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
  Empty: ({ description }: { description?: React.ReactNode }) => (
    <div>{description}</div>
  ),
  Modal: Object.assign(
    ({
      children,
      visible,
    }: {
      children?: React.ReactNode;
      visible?: boolean;
    }) => (visible ? <div>{children}</div> : null),
    { confirm: vi.fn() },
  ),
  Search: ({
    value,
    onChange,
    placeholder,
  }: {
    value?: string;
    onChange?: (value: string) => void;
    placeholder?: string;
  }) => (
    <input
      value={value || ''}
      placeholder={placeholder}
      onChange={event => onChange?.(event.target.value)}
    />
  ),
  Spin: () => <div data-testid="spin" />,
  Toast: {
    error: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
  },
}));

vi.mock('../../../assets/skill-marketplace-hero.webp', () => ({
  default: 'skill-marketplace-hero.webp',
}));

describe('SkillMarketplacePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    installSkill.mockResolvedValue({ code: 0 });
    listSkills.mockResolvedValue({
      code: 0,
      data: {
        skill_list: [
          {
            skill_id: 'skill-doc',
            name: '文档处理',
            description: '处理 docx、pdf 和表格文件',
            files: {
              'SKILL.md': '# 文档处理',
              'scripts/run.py': 'print("ok")',
              'assets/logo.png': 'data:image/png;base64,iVBORw0KGgo=',
            },
            asset_summary: {
              count: 1,
              image_paths: ['assets/logo.png'],
            },
            metadata: {
              category: '文档处理',
              tags: ['office'],
              version: '1.0.0',
            },
            publish_scope: 3,
            published_version: 1,
          },
          {
            skill_id: 'skill-general',
            name: '通用助手',
            description: '常用脚本和参考模板',
            files: {
              'SKILL.md': '# 通用助手',
              'references/readme.md': 'Reference',
            },
            metadata: {
              category: '通用',
              tags: ['general'],
            },
            publish_scope: 3,
            published_version: 2,
          },
          {
            skill_id: 'prompt-only',
            name: '提示词技能',
            description: '非标准技能',
            files: {},
            publish_scope: 3,
          },
        ],
      },
    });
  });

  it('renders the redesigned standard skill marketplace shell', async () => {
    render(<SkillMarketplacePage />);

    await waitFor(() => {
      expect(listSkills).toHaveBeenCalledWith({
        scope: 3,
        page: 1,
        page_size: 200,
      });
    });

    expect(await screen.findByText('技能商店')).toBeInTheDocument();
    expect(screen.getByTestId('skill-marketplace-wrap')).toBeInTheDocument();
    expect(screen.getByTestId('skill-marketplace-hero')).toBeInTheDocument();
    expect(screen.getAllByTestId('skill-marketplace-card')).toHaveLength(2);
    expect(screen.getByText('2 个技能包')).toBeInTheDocument();
    expect(screen.getByText('全局标准技能')).toBeInTheDocument();
    expect(screen.getByText('包含资产')).toBeInTheDocument();
    expect(screen.getByText('技能商城')).toBeInTheDocument();
    expect(screen.getByText('待审核技能')).toBeInTheDocument();
    expect(screen.queryByText('全部资源')).not.toBeInTheDocument();
    expect(screen.getAllByText('文档处理').length).toBeGreaterThan(0);
    expect(screen.getByText('通用助手')).toBeInTheDocument();
    expect(screen.queryByText('提示词技能')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '文档处理' }));

    expect(screen.getByText('1 个技能包')).toBeInTheDocument();
    expect(screen.getAllByText('文档处理').length).toBeGreaterThan(0);
    expect(screen.queryByText('通用助手')).not.toBeInTheDocument();
  });
});
