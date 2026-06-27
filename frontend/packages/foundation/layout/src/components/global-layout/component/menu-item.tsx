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

import { NavLink, useLocation } from 'react-router-dom';
import { type FC } from 'react';

import classNames from 'classnames';

import { reportNavClick } from '../utils';
import { type LayoutMenuItem } from '../types';

const menuStyle = classNames(
  'w-[56px] py-[8px] pb-[6px]',
  'flex flex-col items-center justify-center',
  'rounded-[12px]',
  'transition-all',
  'hover:bg-[var(--coz-bg-secondary)] hover:coz-fg-plus',
);

export const GLobalLayoutMenuItem: FC<LayoutMenuItem> = ({
  title,
  icon,
  activeIcon,
  path,
  dataTestId,
}) => {
  const location = useLocation();

  let isActive = false;
  let newPath = '';
  // If path is an array, take the first matching path
  if (Array.isArray(path)) {
    isActive = path.some(p => location.pathname.startsWith(p));
    newPath = path.find(p => location.pathname.startsWith(p)) || path[0];
  } else {
    isActive = location.pathname.startsWith(path);
    newPath = path;
  }

  // cp-disable-next-line
  const isLink = newPath.startsWith('https://');

  const navId = `primary-menu-${
    newPath.startsWith('/') ? newPath.slice(1) : newPath
  }`;
  return (
    <NavLink
      to={newPath}
      target={isLink ? '_blank' : undefined}
      className="no-underline"
      onClick={() => {
        reportNavClick(title);
      }}
      data-testid={dataTestId}
    >
      <div
        className={classNames(
          menuStyle,
          isActive
            ? 'bg-[rgba(53,138,255,0.1)] text-[rgb(53,138,255)]'
            : 'coz-bg-max coz-fg-secondary',
        )}
        id={navId}
      >
        <div className="h-[21px] w-[21px] leading-none [&_svg]:h-[21px] [&_svg]:w-[21px] [&_svg]:stroke-current">
          {isActive ? activeIcon : icon}
        </div>
        <div className="mt-[5px] h-[11px] font-[500] flex items-center justify-center overflow-hidden leading-none w-full text-[11px]">
          <span className="whitespace-nowrap">{title}</span>
        </div>
      </div>
    </NavLink>
  );
};
