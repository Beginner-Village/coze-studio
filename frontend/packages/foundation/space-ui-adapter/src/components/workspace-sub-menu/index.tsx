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

import { useNavigate } from 'react-router-dom';

import {
  WorkspaceSubMenu as BaseWorkspaceSubMenu,
  SpaceSelector,
} from '@coze-foundation/space-ui-base';
import { useSpaceStore } from '@coze-foundation/space-store';
import { ResType } from '@coze-arch/idl/plugin_develop';
import { I18n } from '@coze-arch/i18n';
import { IconCozAnalytics } from '@coze-arch/coze-design/icons';
import {
  IconBotDevelop,
  IconBotDevelopActive,
  IconBotPlugin,
  IconBotPluginActive,
  IconBotKnowledge,
  IconBotKnowledgeActive,
  IconBotPrompt,
  IconBotPromptActive,
  IconBotDatabaseDefault,
  IconBotDatabaseActive,
  IconBotModel,
  IconBotModelActive,
  IconBotWorkflow,
  IconBotWorkflowActive,
  IconBotMember,
  IconBotMemberActive,
  IconMcp,
  IconMcpActive,
  IconCard,
  IconCardActive,
} from '@coze-arch/bot-icons';
import { useRouteConfig } from '@coze-arch/bot-hooks';

import { SpaceSubModuleEnum } from '@/const';

// Feature flags - 这些值通过 rsbuild source.define 从 GLOBAL_ENVS 注入
// 在 features.ts 中定义，构建时替换为实际值
declare const FEATURE_SHOW_MCP: boolean;
declare const FEATURE_SHOW_EXTERNAL_KNOWLEDGE: boolean;

const createSubMenuConfig = () => {
  const menuItems = [
    {
      icon: <IconBotDevelop />,
      activeIcon: <IconBotDevelopActive />,
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
      icon: <IconMcp />,
      activeIcon: <IconMcpActive />,
      title: () => I18n.t('navigation_workspace_library_mcp', {}, 'Mcp'),
      path: `${SpaceSubModuleEnum.MCP}`,
      dataTestId: 'navigation_workspace_library_mcp',
    },
    // 卡片菜单
    {
      icon: <IconCard />,
      activeIcon: <IconCardActive />,
      title: () => I18n.t('navigation_workspace_library_card', {}, 'Card'),
      path: `${SpaceSubModuleEnum.CARD}`,
      dataTestId: 'navigation_workspace_library_card',
    },
    {
      icon: <IconBotWorkflow />,
      activeIcon: <IconBotWorkflowActive />,
      title: () =>
        I18n.t('navigation_workspace_library_workflow', {}, 'Workflow'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Workflow}`,
      dataTestId: 'navigation_workspace_library_workflow',
    },
    {
      icon: <IconBotPlugin />,
      activeIcon: <IconBotPluginActive />,
      title: () =>
        I18n.t('navigation_workspace_library_plugins', {}, 'Plugins'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Plugin}`,
      dataTestId: 'navigation_workspace_library_plugins',
    },
    {
      icon: <IconBotKnowledge />,
      activeIcon: <IconBotKnowledgeActive />,
      title: () =>
        I18n.t('navigation_workspace_library_knowledge', {}, 'Knowledge'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Knowledge}`,
      dataTestId: 'navigation_workspace_library_knowledge',
    },
    // 外部知识库菜单 - 受 FEATURE_SHOW_EXTERNAL_KNOWLEDGE 控制
    FEATURE_SHOW_EXTERNAL_KNOWLEDGE && {
      icon: <IconBotKnowledge />,
      activeIcon: <IconBotKnowledgeActive />,
      title: () => '外部知识库',
      path: `${SpaceSubModuleEnum.LIBRARY}/external-knowledge`,
      dataTestId: 'navigation_workspace_library_external_knowledge',
    },
    {
      icon: <IconBotPrompt />,
      activeIcon: <IconBotPromptActive />,
      title: () => '技能',
      path: SpaceSubModuleEnum.SKILLS,
      dataTestId: 'navigation_workspace_library_skills',
    },
    {
      icon: <IconBotPrompt />,
      activeIcon: <IconBotPromptActive />,
      title: () => I18n.t('navigation_workspace_library_prompt', {}, 'Prompt'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Prompt}`,
      dataTestId: 'navigation_workspace_library_prompt',
    },
    {
      icon: <IconBotDatabaseDefault />,
      activeIcon: <IconBotDatabaseActive />,
      title: () =>
        I18n.t('navigation_workspace_library_database', {}, 'Database'),
      path: `${SpaceSubModuleEnum.LIBRARY}/${ResType.Database}`,
      dataTestId: 'navigation_workspace_library_database',
    },
    {
      type: 'title',
      title: () => I18n.t('navigation_workspace_manage_title', {}, 'Manage'),
    },
    {
      icon: <IconBotModel />,
      activeIcon: <IconBotModelActive />,
      title: () => I18n.t('navigation_workspace_manage_models', {}, 'Models'),
      path: SpaceSubModuleEnum.MODELS,
      dataTestId: 'navigation_workspace_models',
    },
    {
      icon: <IconBotKnowledge />,
      activeIcon: <IconBotKnowledgeActive />,
      title: () =>
        I18n.t('navigation_workspace_manage_embedding', {}, 'Embedding'),
      path: SpaceSubModuleEnum.EMBEDDING,
      dataTestId: 'navigation_workspace_embedding',
    },
    {
      icon: <IconBotKnowledge />,
      activeIcon: <IconBotKnowledgeActive />,
      title: () => 'Rerank',
      path: SpaceSubModuleEnum.RERANK,
      dataTestId: 'navigation_workspace_rerank',
    },
    {
      icon: <IconBotPlugin />,
      activeIcon: <IconBotPluginActive />,
      title: () =>
        I18n.t('navigation_workspace_manage_external_agents', {}, '外部智能体'),
      path: SpaceSubModuleEnum.HIAGENTS,
      dataTestId: 'navigation_workspace_external_agents',
    },
    {
      icon: <IconBotMember />,
      activeIcon: <IconBotMemberActive />,
      title: () => I18n.t('navigation_workspace_members', {}, 'Members'),
      path: SpaceSubModuleEnum.MEMBERS,
      dataTestId: 'navigation_workspace_members',
    },
    {
      icon: <IconBotWorkflow />,
      activeIcon: <IconBotWorkflowActive />,
      title: () => '导出/导入',
      path: SpaceSubModuleEnum.EXPORT_IMPORT,
      dataTestId: 'navigation_workspace_export_import',
    },
    {
      icon: <IconCozAnalytics />,
      activeIcon: <IconCozAnalytics />,
      title: () => '可观测性',
      path: SpaceSubModuleEnum.OBSERVABILITY,
      dataTestId: 'navigation_workspace_observability',
    },
    { icon: <IconCozAnalytics />, activeIcon: <IconCozAnalytics />, title: () => '评估集', path: 'observability?tab=evaluation-sets', dataTestId: 'navigation_workspace_observability_evaluation_sets' },
    { icon: <IconCozAnalytics />, activeIcon: <IconCozAnalytics />, title: () => '评估器', path: 'observability?tab=evaluators', dataTestId: 'navigation_workspace_observability_evaluators' },
    { icon: <IconCozAnalytics />, activeIcon: <IconCozAnalytics />, title: () => '实验', path: 'observability?tab=experiments', dataTestId: 'navigation_workspace_observability_experiments' },
  ];

  // 过滤掉 false 值（被 feature flag 隐藏的菜单项）
  return menuItems.filter(Boolean);
};

export const WorkspaceSubMenu = () => {
  const { subMenuKey } = useRouteConfig();
  const navigate = useNavigate();

  const {
    space: currentSpace,
    spaceList,
    recentlyUsedSpaceList,
    loading,
    createSpace,
    fetchSpaces,
  } = useSpaceStore(state => ({
    space: state.space,
    spaceList: state.spaceList,
    recentlyUsedSpaceList: state.recentlyUsedSpaceList,
    loading: !!state.loading || !state.inited,
    createSpace: state.createSpace,
    fetchSpaces: state.fetchSpaces,
  }));

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
      currentSubMenu={subMenuKey}
    />
  );
};
