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

import { useState, useRef, useEffect } from 'react';

import { Space, Avatar, Typography } from '@coze-arch/coze-design';
import { type BotSpace } from '@coze-arch/bot-api/developer_api';

import { CreateSpaceModal } from '../create-space-modal';
import cls from 'classnames';
import styles from './index.module.less';
import defaultWorkspace from './default-workspace.svg';

// 常量定义
const ROLE_TYPE = {
  OWNER: 1,
  ADMIN: 2,
  MEMBER: 3,
} as const;

const ROLE_NAMES = {
  [ROLE_TYPE.OWNER]: '所有者',
  [ROLE_TYPE.ADMIN]: '管理员',
  [ROLE_TYPE.MEMBER]: '成员',
} as const;

const SPACING = {
  SMALL: 4,
  MEDIUM: 8,
} as const;

interface SpaceSelectorProps {
  currentSpace?: BotSpace;
  spaceList: BotSpace[];
  recentlyUsedSpaceList?: BotSpace[]; // 添加最近使用的空间列表
  loading?: boolean;
  onSpaceChange?: (spaceId: string) => void;
  onCreateSpace?: (data: {
    name: string;
    description: string;
  }) => Promise<void>;
}

// 空间项组件
interface SpaceItemProps {
  space: BotSpace;
  isSelected: boolean;
  onSelect: (spaceId: string) => void;
  keyPrefix?: string;
}

const SpaceItem = ({
  space,
  isSelected,
  onSelect,
  keyPrefix = '',
}: SpaceItemProps) => (
  <div
    key={`${keyPrefix}${space.id}`}
    className="flex items-center px-3 py-2 cursor-pointer hover:bg-gray-50"
    onClick={() => onSelect(space.id || '')}
  >
    <div className="w-[20px] h-[20px] flex items-center justify-center mr-3">
      {isSelected ? (
        <span className="text-blue-500 text-[14px] font-bold">✓</span>
      ) : null}
    </div>
    <Avatar
      className="w-[24px] h-[24px] rounded-[6px] shrink-0 mr-3"
      src={space.icon_url || defaultWorkspace}
    />
    <div className="flex-1">
      <Typography.Text className="text-[14px] block">
        {keyPrefix === 'personal-' ? '个人空间' : space.name || ''}
      </Typography.Text>
    </div>
    <span className="text-xs text-gray-400">
      {space.space_role_type === ROLE_TYPE.OWNER
        ? ROLE_NAMES[ROLE_TYPE.OWNER]
        : space.space_role_type === ROLE_TYPE.ADMIN
          ? ROLE_NAMES[ROLE_TYPE.ADMIN]
          : space.space_role_type === ROLE_TYPE.MEMBER
            ? ROLE_NAMES[ROLE_TYPE.MEMBER]
            : '默认'}
    </span>
  </div>
);

// 最近使用空间列表组件
interface RecentSpacesProps {
  recentlyUsedSpaceList: BotSpace[];
  currentSpace?: BotSpace;
  onSpaceSelect: (spaceId: string) => void;
}

const RecentSpaces = ({
  recentlyUsedSpaceList,
  currentSpace,
  onSpaceSelect,
}: RecentSpacesProps) => {
  if (!recentlyUsedSpaceList || recentlyUsedSpaceList.length === 0) {
    return null;
  }

  return (
    <div className="py-2">
      <div className="px-3 py-2 text-xs text-gray-500 font-medium">最近</div>
      {recentlyUsedSpaceList.map(space => (
        <SpaceItem
          key={`recent-${space.id}`}
          space={space}
          isSelected={space.id === currentSpace?.id}
          onSelect={onSpaceSelect}
          keyPrefix="recent-"
        />
      ))}
    </div>
  );
};

// 空间分组组件
interface SpaceGroupProps {
  title: string;
  icon: string;
  spaces: BotSpace[];
  currentSpace?: BotSpace;
  onSpaceSelect: (spaceId: string) => void;
  keyPrefix: string;
}

const SpaceGroup = ({
  title,
  icon,
  spaces,
  currentSpace,
  onSpaceSelect,
  keyPrefix,
}: SpaceGroupProps) => {
  if (spaces.length === 0) {
    return null;
  }

  return (
    <div className="mb-2">
      <div className="flex items-center px-3 py-1">
        <span
          className={`mr-2 text-sm ${keyPrefix === 'personal-' ? 'text-blue-600' : 'text-orange-500'}`}
        >
          {icon}
        </span>
        <span className="text-xs text-gray-500 font-medium">{title}</span>
      </div>
      {spaces.map(space => (
        <SpaceItem
          key={`${keyPrefix}${space.id}`}
          space={space}
          isSelected={space.id === currentSpace?.id}
          onSelect={onSpaceSelect}
          keyPrefix={keyPrefix}
        />
      ))}
    </div>
  );
};

// 加载状态组件
const LoadingSkeleton = () => (
  <div className="w-full">
    <Space
      className="h-[48px] px-[8px] w-full rounded-[8px]"
      spacing={SPACING.MEDIUM}
    >
      <div className="w-[24px] h-[24px] rounded-[6px] bg-gray-200 animate-pulse" />
      <div className="flex-1 h-[16px] bg-gray-200 rounded animate-pulse" />
      <span className="text-[12px] coz-fg-tertiary">▼</span>
    </Space>
  </div>
);

// 主按钮组件
interface MainButtonProps {
  currentSpace?: BotSpace;
  onClick: () => void;
}

const MainButton = ({ currentSpace, onClick }: MainButtonProps) => (
  <div className="cursor-pointer w-full" onClick={onClick}>
    <Space
      className="h-[48px] px-[8px] w-full hover:coz-mg-secondary-hovered rounded-[8px]"
      spacing={SPACING.MEDIUM}
    >
      <Avatar
        className="w-[24px] h-[24px] rounded-[6px] shrink-0"
        src={currentSpace?.icon_url}
      />
      <Typography.Text
        ellipsis={{ showTooltip: true, rows: 1 }}
        className="flex-1 coz-fg-primary text-[14px] font-[500]"
      >
        {currentSpace?.name || ''}
      </Typography.Text>
      <span
        className={cls('text-[12px]', 'coz-fg-tertiary', styles.dropdown)}
      ></span>
    </Space>
  </div>
);

// 下拉菜单组件
interface DropdownMenuProps {
  isOpen: boolean;
  dropdownRef: React.RefObject<HTMLDivElement>;
  recentlyUsedSpaceList?: BotSpace[];
  personalSpaces: BotSpace[];
  teamSpaces: BotSpace[];
  currentSpace?: BotSpace;
  onSpaceSelect: (spaceId: string) => void;
  onCreateSpace: () => void;
}

const DropdownMenu = ({
  isOpen,
  dropdownRef,
  recentlyUsedSpaceList,
  personalSpaces,
  teamSpaces,
  currentSpace,
  onSpaceSelect,
  onCreateSpace,
}: DropdownMenuProps) => {
  if (!isOpen) {
    return null;
  }

  return (
    <div
      className="fixed bg-white border border-gray-200 rounded-lg shadow-lg max-h-[500px] overflow-y-auto"
      style={{
        minWidth: '320px',
        maxWidth: '400px',
        width: 'max-content',
        zIndex: 999999,
        top: dropdownRef.current
          ? dropdownRef.current.getBoundingClientRect().bottom + SPACING.SMALL
          : 0,
        left: dropdownRef.current
          ? dropdownRef.current.getBoundingClientRect().left
          : 0,
      }}
    >
      {/* 搜索框 */}
      <div className="p-3 border-b border-gray-100">
        <input
          type="text"
          placeholder="搜索工作空间"
          className="w-full px-3 py-2 text-sm border border-gray-200 rounded-md focus:outline-none focus:border-blue-500"
        />
      </div>

      {/* 最近使用 */}
      <RecentSpaces
        recentlyUsedSpaceList={recentlyUsedSpaceList || []}
        currentSpace={currentSpace}
        onSpaceSelect={onSpaceSelect}
      />

      {/* 所有空间 */}
      <div className="py-2 border-t border-gray-100">
        <div className="px-3 py-2 text-xs text-gray-500 font-medium">
          所有空间
        </div>

        {/* 个人空间 */}
        <SpaceGroup
          title="个人空间"
          icon="👤"
          spaces={personalSpaces}
          currentSpace={currentSpace}
          onSpaceSelect={onSpaceSelect}
          keyPrefix="personal-"
        />

        {/* 团队空间 */}
        <SpaceGroup
          title="团队空间"
          icon="👥"
          spaces={teamSpaces}
          currentSpace={currentSpace}
          onSpaceSelect={onSpaceSelect}
          keyPrefix="team-"
        />
      </div>

      {/* 创建新工作空间 - 固定在底部 */}
      <div className="border-t border-gray-100 bg-gray-50">
        <div
          className="flex items-center px-3 py-3 cursor-pointer hover:bg-gray-100 text-blue-600"
          onClick={onCreateSpace}
        >
          <span className="text-[18px] mr-3 font-medium">+</span>
          <Typography.Text className="text-[14px] text-blue-600 font-medium">
            创建新工作空间
          </Typography.Text>
        </div>
      </div>
    </div>
  );
};

export const SpaceSelector = ({
  currentSpace,
  spaceList,
  recentlyUsedSpaceList,
  loading,
  onSpaceChange,
  onCreateSpace,
}: SpaceSelectorProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // 点击外部关闭下拉框
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  const handleToggle = () => {
    setIsOpen(!isOpen);
  };

  const handleSpaceSelect = (spaceId: string) => {
    onSpaceChange?.(spaceId);
    setIsOpen(false);
  };

  const handleCreateSpace = () => {
    setIsOpen(false);
    setShowCreateModal(true);
  };

  const handleCreateSuccess = () => {
    // 创建成功后的回调，可以在这里做一些额外的处理
    console.log('创建空间成功');
  };

  // 分离个人空间和团队空间
  const personalSpaces = spaceList.filter(space =>
    space.name?.includes('Personal'),
  );
  const teamSpaces = spaceList.filter(
    space => !space.name?.includes('Personal'),
  );

  if (loading) {
    return <LoadingSkeleton />;
  }

  return (
    <div className="w-full relative" ref={dropdownRef}>
      {/* 主按钮 */}
      <MainButton currentSpace={currentSpace} onClick={handleToggle} />

      {/* 下拉菜单 */}
      <DropdownMenu
        isOpen={isOpen}
        dropdownRef={dropdownRef}
        recentlyUsedSpaceList={recentlyUsedSpaceList}
        personalSpaces={personalSpaces}
        teamSpaces={teamSpaces}
        currentSpace={currentSpace}
        onSpaceSelect={handleSpaceSelect}
        onCreateSpace={handleCreateSpace}
      />

      {/* 创建空间Modal */}
      <CreateSpaceModal
        visible={showCreateModal}
        onCancel={() => setShowCreateModal(false)}
        onSuccess={handleCreateSuccess}
        onCreateSpace={onCreateSpace}
      />
    </div>
  );
};
