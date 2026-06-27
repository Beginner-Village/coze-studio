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

export const SKILL_PUBLISH_SCOPE = {
  PRIVATE: 1,
  SPACE: 2,
  GLOBAL: 3,
} as const;

export type SkillView = 'mine' | 'space-market' | 'global-market' | 'review';

export const VIEW_TABS: Array<{ key: SkillView; label: string }> = [
  { key: 'mine', label: '我的技能' },
  { key: 'space-market', label: '空间商城' },
  { key: 'global-market', label: '全局商城' },
  { key: 'review', label: '待审核' },
];

export const MARKETPLACE_VIEW_CONFIG: Record<
  'space-market' | 'global-market',
  { scope: number; emptyText: string }
> = {
  'space-market': {
    scope: SKILL_PUBLISH_SCOPE.SPACE,
    emptyText: '当前空间还没有发布技能',
  },
  'global-market': {
    scope: SKILL_PUBLISH_SCOPE.GLOBAL,
    emptyText: '全局技能商城还没有发布技能',
  },
};
