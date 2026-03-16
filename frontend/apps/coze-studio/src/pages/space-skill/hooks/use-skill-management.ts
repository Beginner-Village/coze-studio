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

import { useState, useCallback, useEffect } from 'react';

import type { SkillInfo } from '@coze-studio/api-schema/idl/skill/skill';
import { skill } from '@coze-studio/api-schema';

export function useSkillManagement(spaceId: string) {
  const [skillList, setSkillList] = useState<SkillInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');

  const fetchSkillList = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    try {
      setLoading(true);
      const response = await skill.ListSkills({
        space_id: spaceId,
        page: 1,
        page_size: 50,
        keyword: keyword || undefined,
      });
      if (response.code === 0) {
        setSkillList(response.data?.skill_list || []);
        setTotal(response.data?.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch skills:', error);
    } finally {
      setLoading(false);
    }
  }, [spaceId, keyword]);

  const createSkill = useCallback(
    async (data: {
      name: string;
      description: string;
      prompt: string;
      icon_uri?: string;
    }) => {
      const response = await skill.CreateSkill({
        space_id: spaceId,
        name: data.name,
        description: data.description || undefined,
        prompt: data.prompt || undefined,
        icon_uri: data.icon_uri || undefined,
      });
      if (response.code === 0) {
        await fetchSkillList();
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to create skill');
    },
    [spaceId, fetchSkillList],
  );

  const updateSkill = useCallback(
    async (
      skillId: string,
      data: {
        name?: string;
        description?: string;
        prompt?: string;
        icon_uri?: string;
      },
    ) => {
      const response = await skill.UpdateSkill({
        skill_id: skillId,
        space_id: spaceId,
        ...data,
      });
      if (response.code === 0) {
        await fetchSkillList();
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to update skill');
    },
    [spaceId, fetchSkillList],
  );

  const deleteSkill = useCallback(
    async (skillId: string) => {
      const response = await skill.DeleteSkill({
        skill_id: skillId,
        space_id: spaceId,
      });
      if (response.code === 0) {
        await fetchSkillList();
        return;
      }
      throw new Error(response.msg || 'Failed to delete skill');
    },
    [spaceId, fetchSkillList],
  );

  const getSkill = useCallback(
    async (skillId: string) => {
      const response = await skill.GetSkill({
        skill_id: skillId,
        space_id: spaceId,
      });
      if (response.code === 0) {
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to get skill');
    },
    [spaceId],
  );

  useEffect(() => {
    fetchSkillList();
  }, [fetchSkillList]);

  return {
    skillList,
    loading,
    total,
    keyword,
    setKeyword,
    fetchSkillList,
    createSkill,
    updateSkill,
    deleteSkill,
    getSkill,
  };
}
