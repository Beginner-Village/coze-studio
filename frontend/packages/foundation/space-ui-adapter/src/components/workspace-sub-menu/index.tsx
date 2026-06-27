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

/* eslint-disable @coze-arch/max-line-per-function --
 * WorkspaceSubMenu keeps the full product menu model in one place.
 */
import { useNavigate, useLocation } from 'react-router-dom';
import { type ReactNode } from 'react';

import {
  WorkspaceSubMenu as BaseWorkspaceSubMenu,
  SpaceSelector,
} from '@coze-foundation/space-ui-base';
import { useSpaceStore } from '@coze-foundation/space-store';
import { ResType } from '@coze-arch/idl/plugin_develop';
import { I18n } from '@coze-arch/i18n';
import { useRouteConfig } from '@coze-arch/bot-hooks';

import { SpaceSubModuleEnum } from '@/const';

// Feature flags - 这些值通过 rsbuild source.define 从 GLOBAL_ENVS 注入
// 在 features.ts 中定义，构建时替换为实际值
declare const FEATURE_SHOW_MCP: boolean;
declare const FEATURE_SHOW_EXTERNAL_KNOWLEDGE: boolean;

type NavIconName =
  | 'project'
  | 'mcp'
  | 'card'
  | 'workflow'
  | 'plugin'
  | 'knowledge'
  | 'skill'
  | 'prompt'
  | 'database'
  | 'model'
  | 'embedding'
  | 'rerank'
  | 'agent'
  | 'members'
  | 'transfer'
  | 'maintenance'
  | 'log'
  | 'trace'
  | 'evalSet'
  | 'evaluator'
  | 'experiment';

const iconPaths: Record<NavIconName, ReactNode> = {
  project: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M3 9h18M9 21V9" />
    </>
  ),
  mcp: (
    <>
      <path d="M4 7h16M4 12h16M4 17h10" />
      <path d="M17 17l3 3M20 17l-3 3" />
    </>
  ),
  card: (
    <>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="M3 10h18" />
    </>
  ),
  workflow: (
    <>
      <rect x="3" y="3" width="6" height="6" rx="1" />
      <rect x="15" y="15" width="6" height="6" rx="1" />
      <path d="M6 9v4a2 2 0 0 0 2 2h7" />
    </>
  ),
  plugin: (
    <>
      <path d="M10 3v5M14 3v5" />
      <path d="M8 8h8v3a4 4 0 0 1-8 0z" />
      <path d="M12 15v6" />
    </>
  ),
  knowledge: (
    <>
      <path d="M4 4h11a2 2 0 0 1 2 2v14a2 2 0 0 0-2-2H4z" />
      <path d="M17 18a2 2 0 0 1 2-2h1V4h-3" />
    </>
  ),
  skill: (
    <>
      <circle cx="12" cy="12" r="9" />
      <circle cx="12" cy="12" r="4.5" />
      <circle cx="12" cy="12" r="1" />
    </>
  ),
  prompt: (
    <>
      <circle cx="12" cy="9" r="7" />
      <path d="M9.5 9h5M12 6.5v5" />
      <path d="M12 16v6M9 19l3 3 3-3" />
    </>
  ),
  database: (
    <>
      <ellipse cx="12" cy="5" rx="8" ry="3" />
      <path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5" />
      <path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3" />
    </>
  ),
  model: (
    <>
      <path d="M12 2l9 5v10l-9 5-9-5V7z" />
      <path d="M12 12l9-5M12 12v10M12 12L3 7" />
    </>
  ),
  embedding: (
    <>
      <path d="M12 3l9 5-9 5-9-5z" />
      <path d="M3 13l9 5 9-5" />
    </>
  ),
  rerank: (
    <>
      <path d="M7 4v16M7 4L3 8M7 4l4 4" />
      <path d="M17 20V4M17 20l4-4M17 20l-4-4" />
    </>
  ),
  agent: (
    <>
      <rect x="4" y="8" width="16" height="12" rx="2" />
      <path d="M12 8V5M9 2h6" />
      <circle cx="9" cy="14" r="1" />
      <circle cx="15" cy="14" r="1" />
    </>
  ),
  members: (
    <>
      <circle cx="9" cy="8" r="3" />
      <path d="M3 20a6 6 0 0 1 12 0" />
      <circle cx="17" cy="9" r="2.4" />
      <path d="M15.5 20a5 5 0 0 1 5.5-4.6" />
    </>
  ),
  transfer: (
    <>
      <path d="M7 4v16M7 4L3 8M7 4l4 4" />
      <path d="M17 20V4M17 20l4-4M17 20l-4-4" />
    </>
  ),
  maintenance: (
    <>
      <rect x="3" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="3" width="7" height="7" rx="1" />
      <rect x="3" y="14" width="7" height="7" rx="1" />
      <rect x="14" y="14" width="7" height="7" rx="1" />
    </>
  ),
  log: (
    <>
      <path d="M8 6h12M8 12h12M8 18h12" />
      <path d="M3.5 6h.01M3.5 12h.01M3.5 18h.01" />
    </>
  ),
  trace: <path d="M3 12h4l3 8 4-16 3 8h4" />,
  evalSet: (
    <>
      <rect x="5" y="3" width="14" height="18" rx="2" />
      <path d="M9 7h6M9 11h6M9 15h4" />
    </>
  ),
  evaluator: (
    <>
      <path d="M12 14a8 8 0 1 1 8-8" />
      <path d="M12 14l4-4" />
      <path d="M4 20h16" />
    </>
  ),
  experiment: (
    <>
      <path d="M9 3h6M10 3v6L5 19a1.5 1.5 0 0 0 1.4 2h11.2A1.5 1.5 0 0 0 19 19l-5-10V3" />
      <path d="M8 14h8" />
    </>
  ),
};

const NavIcon = ({ name }: { name: NavIconName }) => (
  <svg
    className="block h-[17px] w-[17px] flex-none"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={2}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden
    focusable="false"
  >
    {iconPaths[name]}
  </svg>
);

const menuIcon = (name: NavIconName) => <NavIcon name={name} />;

const createSubMenuConfig = () => {
  const menuItems = [
    {
      icon: menuIcon('project'),
      activeIcon: menuIcon('project'),
      title: () => I18n.t('navigation_workspace_develop', {}, 'Develop'),
      path: SpaceSubModuleEnum.DEVELOP,
      dataTestId: 'navigation_workspace_develop',
    },
    {
      type: 'title',
      title: () =>
        I18n.t('navigation_workspace_library_title', {}, 'Resource Library'),
    },
    // MCP 菜单 - 受 FEATURE_SHOW_MCP 控制
    FEATURE_SHOW_MCP && {
      icon: menuIcon('mcp'),
      activeIcon: menuIcon('mcp'),
      title: () => I18n.t('navigation_workspace_library_mcp', {}, 'Mcp'),
      path: `${SpaceSubModuleEnum.MCP}`,
      dataTestId: 'navigation_workspace_library_mcp',
    },
    // 卡片菜单
    {
      icon: menuIcon('card'),
      activeIcon: menuIcon('card'),
      title: () => I18n.t('navigation_workspace_library_card', {}, 'Card'),
      path: `${SpaceSubModuleEnum.CARD}`,
      dataTestId: 'navigation_workspace_library_card',
    },
    {
      icon: menuIcon('workflow'),
      activeIcon: menuIcon('workflow'),
      title: () =>
        I18n.t('navigation_workspace_library_workflow', {}, 'Workflow'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Workflow}`,
      dataTestId: 'navigation_workspace_library_workflow',
    },
    {
      icon: menuIcon('plugin'),
      activeIcon: menuIcon('plugin'),
      title: () =>
        I18n.t('navigation_workspace_library_plugins', {}, 'Plugins'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Plugin}`,
      dataTestId: 'navigation_workspace_library_plugins',
    },
    {
      icon: menuIcon('knowledge'),
      activeIcon: menuIcon('knowledge'),
      title: () =>
        I18n.t('navigation_workspace_library_knowledge', {}, 'Knowledge'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Knowledge}`,
      dataTestId: 'navigation_workspace_library_knowledge',
    },
    // 外部知识库菜单 - 受 FEATURE_SHOW_EXTERNAL_KNOWLEDGE 控制
    FEATURE_SHOW_EXTERNAL_KNOWLEDGE && {
      icon: menuIcon('knowledge'),
      activeIcon: menuIcon('knowledge'),
      title: () => '外部知识库',
      path: `${SpaceSubModuleEnum.LIBRARY}/external-knowledge`,
      dataTestId: 'navigation_workspace_library_external_knowledge',
    },
    {
      icon: menuIcon('skill'),
      activeIcon: menuIcon('skill'),
      title: () => '技能',
      path: SpaceSubModuleEnum.SKILLS,
      dataTestId: 'navigation_workspace_library_skills',
    },
    {
      icon: menuIcon('prompt'),
      activeIcon: menuIcon('prompt'),
      title: () => I18n.t('navigation_workspace_library_prompt', {}, 'Prompt'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Prompt}`,
      dataTestId: 'navigation_workspace_library_prompt',
    },
    {
      icon: menuIcon('database'),
      activeIcon: menuIcon('database'),
      title: () =>
        I18n.t('navigation_workspace_library_database', {}, 'Database'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Database}`,
      dataTestId: 'navigation_workspace_library_database',
    },
    {
      icon: menuIcon('skill'),
      activeIcon: menuIcon('skill'),
      title: () => I18n.t('navigation_workspace_library_strategy', {}, '策略'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Strategy}`,
      dataTestId: 'navigation_workspace_library_strategy',
    },
    {
      type: 'title',
      title: () => I18n.t('navigation_workspace_manage_title', {}, 'Manage'),
    },
    {
      icon: menuIcon('model'),
      activeIcon: menuIcon('model'),
      title: () => I18n.t('navigation_workspace_manage_models', {}, 'Models'),
      path: SpaceSubModuleEnum.MODELS,
      dataTestId: 'navigation_workspace_models',
    },
    {
      icon: menuIcon('embedding'),
      activeIcon: menuIcon('embedding'),
      title: () =>
        I18n.t('navigation_workspace_manage_embedding', {}, 'Embedding'),
      path: SpaceSubModuleEnum.EMBEDDING,
      dataTestId: 'navigation_workspace_embedding',
    },
    {
      icon: menuIcon('rerank'),
      activeIcon: menuIcon('rerank'),
      title: () => 'Rerank',
      path: SpaceSubModuleEnum.RERANK,
      dataTestId: 'navigation_workspace_rerank',
    },
    {
      icon: menuIcon('agent'),
      activeIcon: menuIcon('agent'),
      title: () =>
        I18n.t('navigation_workspace_manage_external_agents', {}, '外部智能体'),
      path: SpaceSubModuleEnum.HIAGENTS,
      dataTestId: 'navigation_workspace_external_agents',
    },
    {
      icon: menuIcon('members'),
      activeIcon: menuIcon('members'),
      title: () => I18n.t('navigation_workspace_members', {}, 'Members'),
      path: SpaceSubModuleEnum.MEMBERS,
      dataTestId: 'navigation_workspace_members',
    },
    {
      icon: menuIcon('transfer'),
      activeIcon: menuIcon('transfer'),
      title: () => '导出/导入',
      path: SpaceSubModuleEnum.EXPORT_IMPORT,
      dataTestId: 'navigation_workspace_export_import',
    },
    {
      icon: menuIcon('maintenance'),
      activeIcon: menuIcon('maintenance'),
      title: () =>
        I18n.t('navigation_workspace_data_maintenance', {}, '数据维护'),
      path: SpaceSubModuleEnum.DATA_MAINTENANCE,
      dataTestId: 'navigation_workspace_data_maintenance',
    },
    {
      icon: menuIcon('log'),
      activeIcon: menuIcon('log'),
      title: () => I18n.t('navigation_workspace_operation_log', {}, '操作日志'),
      path: SpaceSubModuleEnum.OPERATION_LOG,
      dataTestId: 'navigation_workspace_operation_log',
    },
    {
      icon: menuIcon('trace'),
      activeIcon: menuIcon('trace'),
      title: () => '可观测性',
      path: SpaceSubModuleEnum.OBSERVABILITY,
      dataTestId: 'navigation_workspace_observability',
    },
    {
      icon: menuIcon('evalSet'),
      activeIcon: menuIcon('evalSet'),
      title: () => '评估集',
      path: 'observability?tab=evaluation-sets',
      dataTestId: 'navigation_workspace_observability_evaluation_sets',
    },
    {
      icon: menuIcon('evaluator'),
      activeIcon: menuIcon('evaluator'),
      title: () => '评估器',
      path: 'observability?tab=evaluators',
      dataTestId: 'navigation_workspace_observability_evaluators',
    },
    {
      icon: menuIcon('experiment'),
      activeIcon: menuIcon('experiment'),
      title: () => '实验',
      path: 'observability?tab=experiments',
      dataTestId: 'navigation_workspace_observability_experiments',
    },
  ];

  // 过滤掉 false 值（被 feature flag 隐藏的菜单项）
  return menuItems.filter(Boolean);
};

export const WorkspaceSubMenu = () => {
  const { subMenuKey } = useRouteConfig();
  const navigate = useNavigate();
  const location = useLocation();

  // Highlight active sub-menu by URL. Routes that share a path but
  // differ by ?tab= (e.g. observability/evaluation-sets/evaluators/
  // experiments) need the tab query to disambiguate. Falls back to
  // subMenuKey supplied by the route loader for everything else.
  const lastSegment = location.pathname.split('/').filter(Boolean).pop() || '';
  const tab = new URLSearchParams(location.search).get('tab');
  const activeSubMenu = tab ? `${lastSegment}?tab=${tab}` : subMenuKey;

  const currentSpace = useSpaceStore(state => state.space);
  const spaceList = useSpaceStore(state => state.spaceList);
  const recentlyUsedSpaceList = useSpaceStore(
    state => state.recentlyUsedSpaceList,
  );
  const loading = useSpaceStore(state => !!state.loading || !state.inited);
  const createSpace = useSpaceStore(state => state.createSpace);
  const fetchSpaces = useSpaceStore(state => state.fetchSpaces);

  const subMenu = createSubMenuConfig();

  const handleSpaceChange = (spaceId: string) => {
    // 更新空间store中的当前空间
    useSpaceStore.getState().setSpace(spaceId);
    // 导航到新空间的develop页面
    navigate(`/space/${spaceId}/develop`);
  };

  const handleCreateSpace = async (data: {
    name: string;
    description: string;
  }) => {
    // 调用store的createSpace方法
    // 使用数字常量1表示Team空间类型 (SpaceType.Team = 1)
    const result = await createSpace({
      name: data.name,
      description: data.description,
      icon_uri: '',
      space_type: 1, // Team空间
    });

    if (result?.id) {
      // 刷新空间列表
      await fetchSpaces(true);
      // 切换到新创建的空间
      handleSpaceChange(result.id);
    }
  };

  const headerNode = (
    <SpaceSelector
      currentSpace={currentSpace}
      spaceList={spaceList}
      recentlyUsedSpaceList={recentlyUsedSpaceList}
      loading={loading}
      onSpaceChange={handleSpaceChange}
      onCreateSpace={handleCreateSpace}
    />
  );

  return (
    <BaseWorkspaceSubMenu
      header={headerNode}
      menus={subMenu}
      currentSubMenu={activeSubMenu}
    />
  );
};
