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

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

import SpaceSkillPage from '../index';

const { recruitMock } = vi.hoisted(() => ({
  recruitMock: vi.fn(),
}));

vi.mock('@coze-arch/bot-api', () => ({
  axiosInstance: { request: vi.fn() },
}));

vi.mock('../agent-app-api', () => ({
  agentAppApi: {
    recruit: recruitMock,
  },
}));

const fetchMarketplaceSkills = vi.fn();
const importSkillPackage = vi.fn();
const exportSkillPackage = vi.fn();
const validateSkillPackage = vi.fn();
const setKeyword = vi.fn();
const navigate = vi.fn();
const mockSkillManagementState = {
  skillList: [] as any[],
  marketplaceList: [] as any[],
  loading: false,
  marketplaceLoading: false,
  total: 0,
  marketplaceTotal: 0,
  keyword: '',
};

vi.mock('react-router-dom', () => ({
  useParams: () => ({ space_id: 'space-1' }),
  useNavigate: () => navigate,
}));

vi.mock('@coze-arch/coze-design/icons', () => {
  const Icon = () => <span data-testid="icon" />;
  return {
    IconCozPlus: Icon,
    IconCozMore: Icon,
    IconCozDelete: Icon,
    IconCozEdit: Icon,
    IconCozPeople: Icon,
    IconCozRefresh: Icon,
    IconCozStarFill: Icon,
    IconCozImport: Icon,
    IconCozFolder: Icon,
    IconCozPlugin: Icon,
    IconCozStore: Icon,
    IconCozTrashCan: Icon,
  };
});

vi.mock('@coze-arch/coze-design', () => {
  const Tabs = ({
    activeKey,
    onChange,
    children,
  }: {
    activeKey?: string;
    onChange?: (key: string) => void;
    children?: React.ReactNode;
  }) => (
    <div data-active-key={activeKey}>
      {React.Children.map(children, child => {
        if (!React.isValidElement(child)) {
          return child;
        }
        const props = child.props as {
          tab?: React.ReactNode;
          itemKey?: string;
        };
        return (
          <button type="button" onClick={() => onChange?.(props.itemKey || '')}>
            {props.tab}
          </button>
        );
      })}
    </div>
  );
  Tabs.TabPane = (_props: { tab?: React.ReactNode; itemKey?: string }) => null;

  const Button = ({
    children,
    onClick,
  }: {
    children?: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  );
  const Search = ({
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
      onChange={event => onChange?.(event.target.value)}
      placeholder={placeholder}
    />
  );
  const Spin = () => <div data-testid="spin" />;
  const Empty = ({ description }: { description?: React.ReactNode }) => (
    <div>{description}</div>
  );
  const Dropdown = ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  );
  const IconButton = ({ icon }: { icon?: React.ReactNode }) => (
    <button>{icon}</button>
  );
  return {
    Button,
    Search,
    Spin,
    Empty,
    Modal: {
      confirm: vi.fn(),
    },
    Dropdown,
    IconButton,
    Tabs,
  };
});

vi.mock('../hooks/use-skill-management', () => ({
  useSkillManagement: () => ({
    ...mockSkillManagementState,
    setKeyword,
    fetchMarketplaceSkills,
    deleteSkill: vi.fn(),
    publishSkill: vi.fn(),
    installMarketplaceSkill: vi.fn(),
    importSkillPackage,
    exportSkillPackage,
    validateSkillPackage,
  }),
}));

describe('SpaceSkillPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    importSkillPackage.mockResolvedValue({});
    validateSkillPackage.mockResolvedValue({ valid: true });
    recruitMock.mockResolvedValue({
      code: 0,
      msg: 'ok',
      data: { shadow_agent_id: 'shadow-42' },
    });
    mockSkillManagementState.skillList = [];
    mockSkillManagementState.marketplaceList = [];
    mockSkillManagementState.loading = false;
    mockSkillManagementState.marketplaceLoading = false;
    mockSkillManagementState.total = 0;
    mockSkillManagementState.marketplaceTotal = 0;
    mockSkillManagementState.keyword = '';
  });

  it('lets users open the global marketplace view', async () => {
    render(<SpaceSkillPage />);

    expect(screen.getByText('上传 ZIP')).toBeInTheDocument();
    expect(screen.getByText('空间商城')).toBeInTheDocument();
    expect(screen.getByText('全局商城')).toBeInTheDocument();

    fireEvent.click(screen.getByText('全局商城'));

    await waitFor(() => {
      expect(fetchMarketplaceSkills).toHaveBeenCalledWith(3);
    });
  });

  it('shows standard skill metadata on marketplace cards', async () => {
    mockSkillManagementState.marketplaceList = [
      {
        skill_id: 'skill-1',
        space_id: 'space-2',
        name: 'doc-tools',
        description: 'Create and inspect docs',
        files: {
          'SKILL.md': '# Doc Tools',
          'scripts/export.sh': 'echo ok',
          'assets/logo.png': 'data:image/png;base64,iVBORw0KGgo=',
        },
        asset_summary: {
          count: 1,
          image_paths: ['assets/logo.png'],
        },
        metadata: {
          category: 'productivity',
          tags: ['documents', 'office'],
          platforms: ['linux', 'macos'],
          version: '1.2.3',
        },
        version: 2,
        publish_scope: 3,
        published_version: 2,
        created_at: 1781800000000,
        updated_at: 1781800000000,
      },
    ];
    mockSkillManagementState.marketplaceTotal = 1;

    render(<SpaceSkillPage />);

    fireEvent.click(screen.getByText('全局商城'));

    expect(await screen.findByText('productivity')).toBeInTheDocument();
    expect(screen.getByText('documents')).toBeInTheDocument();
    expect(screen.getByText('linux')).toBeInTheDocument();
    expect(screen.getByText('v1.2.3')).toBeInTheDocument();
    expect(screen.getByText('1 资产')).toBeInTheDocument();
    expect(screen.getByText('下载 ZIP')).toBeInTheDocument();
    expect(screen.getByText('安装')).toBeInTheDocument();
  });

  it('validates zip packages before importing them', async () => {
    const { container } = render(<SpaceSkillPage />);
    const input = container.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement;
    const file = new File(['zip-bytes'], 'doc-tools.zip', {
      type: 'application/zip',
    });

    fireEvent.change(input, { target: { files: [file] } });

    await waitFor(() => {
      expect(validateSkillPackage).toHaveBeenCalledWith({
        filename: 'doc-tools.zip',
        content: expect.stringMatching(/^data:application\/zip/),
      });
    });
    await waitFor(() => {
      expect(importSkillPackage).toHaveBeenCalledWith({
        filename: 'doc-tools.zip',
        content: expect.stringMatching(/^data:application\/zip/),
      });
    });
    expect(validateSkillPackage.mock.invocationCallOrder[0]).toBeLessThan(
      importSkillPackage.mock.invocationCallOrder[0],
    );
  });

  it('shows validation errors and does not import invalid zip packages', async () => {
    validateSkillPackage.mockResolvedValue({
      valid: false,
      error: 'SKILL.md is required',
    });

    const { container } = render(<SpaceSkillPage />);
    const input = container.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement;
    const file = new File(['zip-bytes'], 'broken.zip', {
      type: 'application/zip',
    });

    fireEvent.change(input, { target: { files: [file] } });

    expect(await screen.findByText('SKILL.md is required')).toBeInTheDocument();
    expect(importSkillPackage).not.toHaveBeenCalled();
  });

  it('renders 招聘 button for agent_app cards and navigates on click', async () => {
    mockSkillManagementState.marketplaceList = [
      {
        skill_id: 'agent-app-1',
        product_id: 'prod-1',
        name: '招聘助手',
        description: '虚拟员工',
        type: 'agent_app',
        publish_scope: 3,
        version: 1,
        created_at: 1781800000000,
        updated_at: 1781800000000,
      },
    ];
    mockSkillManagementState.marketplaceTotal = 1;

    render(<SpaceSkillPage />);

    fireEvent.click(screen.getByText('全局商城'));

    expect(await screen.findByText('招聘')).toBeInTheDocument();
    expect(screen.queryByText('安装')).not.toBeInTheDocument();

    fireEvent.click(screen.getByText('招聘'));

    await waitFor(() => {
      expect(recruitMock).toHaveBeenCalledWith({
        product_id: 'prod-1',
        space_id: 'space-1',
      });
    });
    await waitFor(() => {
      expect(navigate).toHaveBeenCalledWith(
        '/space/space-1/bot/shadow-42/arrange',
      );
    });
  });
});
