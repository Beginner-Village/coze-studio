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

import { type FC, Suspense } from 'react';

import { useRouteConfig } from '@coze-arch/bot-hooks';

const DEFAULT_WIDTH = 236;

export const SubMenu: FC = () => {
  const config = useRouteConfig();
  const { subMenu: SubMenuComponent } = config;

  if (!SubMenuComponent) {
    return null;
  }

  return (
    <div className="relative flex flex-row flex-none border-0 border-r-[1px] border-solid coz-stroke-primary coz-bg-max">
      <div
        className="overflow-auto flex flex-col box-border px-[8px] py-[12px]"
        style={{ width: `${DEFAULT_WIDTH}px` }}
      >
        <Suspense>
          <SubMenuComponent />
        </Suspense>
      </div>
    </div>
  );
};
