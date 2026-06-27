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

import React, { type HTMLAttributes, forwardRef } from 'react';

import classNames from 'classnames';

export type LayoutBaseProps = HTMLAttributes<HTMLDivElement>;

export const Layout = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'h-full min-h-[100%]',
        'flex flex-col',
        'overflow-hidden',
        'bg-[rgb(244,246,251)]',
      )}
    >
      {children}
    </div>
  ),
);

export const Header = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'flex-shrink-0',
        'w-full',
        'flex items-start justify-between gap-[16px]',
        'px-[32px] pt-[26px] pb-[18px]',
        'bg-[var(--coz-bg-max)]',
      )}
    >
      {children}
    </div>
  ),
);

export const HeaderTitle = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'text-[24px] font-[600] leading-[32px]',
        'coz-fg-plus',
        'flex items-center gap-[8px]',
      )}
    >
      {children}
    </div>
  ),
);

export const HeaderActions = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'flex items-center gap-[12px] ml-[32px] pt-[2px]',
      )}
    >
      {children}
    </div>
  ),
);

export const SubHeader = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'flex-shrink-0',
        'w-full',
        'flex items-center justify-between gap-[12px]',
        'px-[32px] py-[18px]',
        'border-0 border-t-[1px] border-b-[1px] border-solid coz-stroke-primary',
        'bg-[rgb(244,246,251)]',
      )}
    >
      {children}
    </div>
  ),
);

export const SubHeaderFilters = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'flex items-center gap-[12px] flex-wrap',
      )}
    >
      {children}
    </div>
  ),
);

export const SubHeaderSearch = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(restProps.className, 'flex-shrink-0')}
    >
      {children}
    </div>
  ),
);

export const Content = forwardRef<HTMLDivElement, LayoutBaseProps>(
  ({ children, ...restProps }, ref) => (
    <div
      {...restProps}
      ref={ref}
      className={classNames(
        restProps.className,
        'flex-grow',
        'overflow-x-hidden overflow-y-auto',
        'px-[32px] pt-[6px] pb-[40px]',
      )}
    >
      {children}
    </div>
  ),
);
