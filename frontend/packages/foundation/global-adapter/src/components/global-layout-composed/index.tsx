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

import { useParams } from 'react-router-dom';
import {
  type FC,
  type PropsWithChildren,
  type ReactNode,
  useEffect,
} from 'react';

import { GlobalLayout } from '@coze-foundation/layout';
import { useCreateBotAction } from '@coze-foundation/global';
import { RequireAuthContainer } from '@coze-foundation/account-ui-adapter';
import { logout } from '@coze-foundation/account-adapter';
import { I18n } from '@coze-arch/i18n';
import { IconCozPlusCircle } from '@coze-arch/coze-design/icons';
import { useRouteConfig } from '@coze-arch/bot-hooks';

import { useHasSider } from './hooks/use-has-sider';
import { AccountDropdown } from '../account-dropdown';

type RailIconName = 'workspace' | 'store' | 'template';

const railIconPaths: Record<RailIconName, ReactNode> = {
  workspace: (
    <>
      <path d="M12 2L3 7l9 5 9-5-9-5z" />
      <path d="M3 12l9 5 9-5" />
      <path d="M3 17l9 5 9-5" />
    </>
  ),
  store: (
    <>
      <path d="M3 9l1.5-5h15L21 9" />
      <path d="M3 9v10a1 1 0 0 0 1 1h16a1 1 0 0 0 1-1V9" />
      <path d="M3 9a2.5 2.5 0 0 0 5 0 2.5 2.5 0 0 0 5 0 2.5 2.5 0 0 0 5 0 2.5 2.5 0 0 0 3 0" />
    </>
  ),
  template: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M9 3v18M3 9h6" />
    </>
  ),
};

const RailIcon = ({ name }: { name: RailIconName }) => (
  <svg
    className="block h-[21px] w-[21px]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth={2}
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden
    focusable="false"
  >
    {railIconPaths[name]}
  </svg>
);

export const GlobalLayoutComposed: FC<PropsWithChildren> = ({ children }) => {
  const config = useRouteConfig();
  const hasSider = useHasSider();
  const { space_id } = useParams();

  useEffect(() => {
    const handler = () => logout();
    window.document.addEventListener('exit', handler, false);
    return () => {
      window.document.removeEventListener('exit', handler, false);
    };
  }, []);

  const { createBot, createBotModal } = useCreateBotAction({
    currentSpaceId: space_id,
  });

  return (
    <RequireAuthContainer
      needLogin={!!config.requireAuth}
      loginOptional={!!config.requireAuthOptional}
    >
      <GlobalLayout
        hasSider={hasSider}
        banner={null}
        actions={[
          {
            tooltip: I18n.t('creat_tooltip_create'),
            icon: <IconCozPlusCircle />,
            onClick: createBot,
            dataTestId: 'layout_create-agent-button',
          },
        ]}
        menus={[
          {
            title: I18n.t('navigation_workspace'),
            icon: <RailIcon name="workspace" />,
            activeIcon: <RailIcon name="workspace" />,
            path: '/space',
            dataTestId: 'layout_workspace-button',
          },
          {
            title: I18n.t('menu_title_store'),
            icon: <RailIcon name="store" />,
            activeIcon: <RailIcon name="store" />,
            path: '/explore',
            dataTestId: 'layout_explore-button',
          },
          {
            title: I18n.t('menu_title_template'),
            icon: <RailIcon name="template" />,
            activeIcon: <RailIcon name="template" />,
            path: '/template',
            dataTestId: 'layout_explore-template-button',
          },
        ]}
        extras={[]}
        footer={<AccountDropdown />}
      >
        {children}
        {createBotModal}
      </GlobalLayout>
    </RequireAuthContainer>
  );
};
