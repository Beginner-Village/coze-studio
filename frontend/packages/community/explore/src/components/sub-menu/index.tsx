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

import { useNavigate, useParams } from 'react-router-dom';
import { useState, useEffect, type ReactNode } from 'react';

import { SubMenuItem } from '@coze-community/components';
import { I18n } from '@coze-arch/i18n';
import { IconCozStore } from '@coze-arch/coze-design/icons';
import {
  IconBotDevelop,
  IconBotCard,
} from '../../../../../components/bot-icons';
import { Space } from '@coze-arch/coze-design';
import { aopApi } from '@coze-arch/bot-api';

import { useExploreRoute } from '../../hooks/use-explore-route';
import cls from 'classnames';

import styles from './index.module.less';

const SkillStoreNavButton = ({
  active = false,
  children,
  icon,
  suffix,
  onClick,
}: {
  active?: boolean;
  children: ReactNode;
  icon: ReactNode;
  suffix?: ReactNode;
  onClick?: () => void;
}) => (
  <button
    type="button"
    className={cls(styles.skillStoreNavItem, {
      [styles.skillStoreNavItemActive]: active,
    })}
    onClick={onClick}
  >
    {icon}
    <span className={styles.skillStoreNavText}>{children}</span>
    {suffix ? (
      <span className={styles.skillStoreNavSuffix}>{suffix}</span>
    ) : null}
  </button>
);

const CustomSubMenu = ({ menuConfig }) => {
  const navigate = useNavigate();
  const { type } = useExploreRoute();
  const { sub_route_id } = useParams();
  const firstParentNodeIndex = menuConfig.findIndex(item =>
    Array.isArray(item.children),
  );
  const defaultType =
    firstParentNodeIndex > -1 ? menuConfig[firstParentNodeIndex].type : '';

  const [activeId, setActiveId] = useState(defaultType);

  const toggleActive = id => {
    if (activeId === id) {
      setActiveId('');
      return;
    }
    setActiveId(id);
  };

  return (
    <Space spacing={4} vertical>
      {menuConfig.map(item => [
        <SubMenuItem
          key={item.type}
          {...item}
          isActive={item?.children?.length ? false : item.type === type}
          suffix={
            item?.children?.length ? (
              <div
                className={cls(styles.groupSubMenuArrow, {
                  [styles.groupSubMenuArrowActive]: activeId === item.type,
                })}
              />
            ) : null
          }
          onClick={() => {
            item.path ? navigate(item.path) : toggleActive(item.type);
          }}
        />,
        activeId === item.type &&
          item?.children?.map(child => (
            <SubMenuItem
              key={child.type}
              {...child}
              subNode={true}
              isActive={child.type === sub_route_id}
              onClick={() => {
                navigate(child.path);
              }}
            />
          )),
      ])}
    </Space>
  );
};

export const ExploreSubMenu = () => {
  const navigate = useNavigate();

  return (
    <div className={styles.skillStoreNav} data-testid="skill-store-sub-menu">
      <button
        type="button"
        className={styles.skillStoreWorkspace}
        onClick={() => {
          navigate('/explore/project/latest');
        }}
      >
        <span className={styles.skillStoreWorkspaceIcon}>
          <IconCozStore />
        </span>
        <span className={styles.skillStoreWorkspaceName}>商店</span>
        <span className={styles.skillStoreWorkspaceChevron}>⌄</span>
      </button>
      <div className={styles.skillStoreNavScroll}>
        <div className={styles.skillStoreNavLabel}>浏览</div>
        <SkillStoreNavButton active icon={<IconCozStore />}>
          标准技能包
        </SkillStoreNavButton>
      </div>
    </div>
  );
};

const SubText = ({ children }) => (
  <span className={cls('text-[12px] ml-[4px]')}>{children}</span>
);

export const TemplateSubMenu = () => {
  const [subMenus, setSubMenus] = useState([]);
  useEffect(() => {
    aopApi.GetCardTypeCount().then(res => {
      let total = 0;
      const list = res.body.cardClassList?.map(e => {
        total += Number(e.count || 0);
        return {
          type: `${e.id}`,
          title: e.name,
          subText: <SubText>{e.count}</SubText>,
          isActive: true,
          path: `/template/card/${e.id}`,
        };
      });
      list.unshift({
        type: 'all',
        title: I18n.t('All'),
        subText: <SubText>{total}</SubText>,
        isActive: true,
        path: '/template/card/all',
      });
      setSubMenus(list);
    });
  }, []);

  return (
    <CustomSubMenu
      menuConfig={[
        {
          type: 'project',
          icon: <IconBotDevelop />,
          activeIcon: <IconBotDevelop />,
          title: I18n.t('Template_project'),
          isActive: true,
          path: '/template/project',
        },
        {
          type: 'card',
          icon: <IconBotCard />,
          activeIcon: <IconBotCard />,
          title: I18n.t('Template_card'),
          children: subMenus,
        },
      ]}
    />
  );
};
