/*
 * Copyright 2025 coze-dev Authors
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

import { useState, useCallback } from 'react';

export interface ResourceItem {
  type: 'workflow' | 'plugin' | 'knowledge';
  id: string;
  name: string;
  description: string;
  icon?: string;
}

export interface ResourceGroup {
  type: 'workflow' | 'plugin' | 'knowledge';
  label: string;
  items: ResourceItem[];
}

// ResType 枚举，匹配 resource_common.ResType
const RES_TYPE_PLUGIN = 1;
const RES_TYPE_WORKFLOW = 2;
const RES_TYPE_KNOWLEDGE = 4;

const RES_TYPE_MAP: Record<number, 'workflow' | 'plugin' | 'knowledge'> = {
  [RES_TYPE_WORKFLOW]: 'workflow',
  [RES_TYPE_PLUGIN]: 'plugin',
  [RES_TYPE_KNOWLEDGE]: 'knowledge',
};

const RES_TYPE_LABEL: Record<string, string> = {
  workflow: '工作流',
  plugin: '插件',
  knowledge: '知识库',
};

export function useSpaceResources(spaceId: string) {
  const [resources, setResources] = useState<ResourceGroup[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchResources = useCallback(async () => {
    if (!spaceId || loading) {
      return;
    }
    setLoading(true);
    try {
      // 使用统一的 library_resource_list API，获取工作流+插件+知识库
      const res = await fetch('/api/plugin_api/library_resource_list', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          space_id: spaceId,
          res_type_filter: [
            RES_TYPE_WORKFLOW,
            RES_TYPE_PLUGIN,
            RES_TYPE_KNOWLEDGE,
          ],
          size: 100,
        }),
      });
      const data = await res.json();

      if (data.code === 0 && data.resource_list) {
        // 按 res_type 分组
        const groups: Record<string, ResourceItem[]> = {};

        for (const item of data.resource_list) {
          const type = RES_TYPE_MAP[item.res_type];
          if (!type) {
            continue;
          }

          if (!groups[type]) {
            groups[type] = [];
          }

          groups[type].push({
            type,
            id: String(item.res_id),
            name: item.name || '',
            description: item.desc || '',
            icon: item.icon || '',
          });
        }

        // 按 工作流 → 插件 → 知识库 排序
        const order: Array<'workflow' | 'plugin' | 'knowledge'> = [
          'workflow',
          'plugin',
          'knowledge',
        ];
        const result: ResourceGroup[] = order
          .filter(t => groups[t]?.length)
          .map(t => ({
            type: t,
            label: RES_TYPE_LABEL[t],
            items: groups[t],
          }));

        setResources(result);
      } else {
        setResources([]);
      }
    } catch (error) {
      console.error('加载资源失败:', error);
      setResources([]);
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  return { resources, loading, fetchResources };
}
