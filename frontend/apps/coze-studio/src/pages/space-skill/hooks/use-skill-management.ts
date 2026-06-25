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

import type {
  SkillInfo,
  SkillPackageValidation,
} from '@coze-studio/api-schema/idl/skill/skill';
import { skill } from '@coze-studio/api-schema';

export function useSkillManagement(spaceId: string) {
  const [skillList, setSkillList] = useState<SkillInfo[]>([]);
  const [marketplaceList, setMarketplaceList] = useState<SkillInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [marketplaceLoading, setMarketplaceLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [marketplaceTotal, setMarketplaceTotal] = useState(0);
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

  const fetchMarketplaceSkills = useCallback(
    async (scope?: number) => {
      try {
        setMarketplaceLoading(true);
        const response = await skill.MarketplaceListSkills({
          space_id: spaceId || undefined,
          scope,
          page: 1,
          page_size: 50,
          keyword: keyword || undefined,
        });
        if (response.code === 0) {
          setMarketplaceList(response.data?.skill_list || []);
          setMarketplaceTotal(response.data?.total || 0);
        }
      } catch (error) {
        console.error('Failed to fetch marketplace skills:', error);
      } finally {
        setMarketplaceLoading(false);
      }
    },
    [spaceId, keyword],
  );

  const createSkill = useCallback(
    async (data: {
      name: string;
      description: string;
      prompt: string;
      files?: Record<string, string>;
      icon_uri?: string;
    }) => {
      const response = await skill.CreateSkill({
        space_id: spaceId,
        name: data.name,
        description: data.description || undefined,
        prompt: data.prompt || undefined,
        files: data.files,
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

  const importSkillPackage = useCallback(
    async (data: { filename?: string; content: string; icon_uri?: string }) => {
      const response = await skill.SuperAgentImportSkillPackage({
        space_id: spaceId,
        filename: data.filename,
        content: data.content,
        icon_uri: data.icon_uri || undefined,
      });
      if (response.code === 0) {
        await fetchSkillList();
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to import skill package');
    },
    [spaceId, fetchSkillList],
  );

  const validateSkillPackage = useCallback(
    async (data: {
      filename?: string;
      content: string;
    }): Promise<SkillPackageValidation | undefined> => {
      const response = await skill.SuperAgentValidateSkillPackage({
        filename: data.filename,
        content: data.content,
      });
      if (response.code === 0) {
        return response.data?.validation;
      }
      throw new Error(response.msg || 'Failed to validate skill package');
    },
    [],
  );

  const exportSkillPackage = useCallback(
    async (skillId: string) => {
      const response = await skill.SuperAgentExportSkillPackage({
        space_id: spaceId,
        skill_id: skillId,
      });
      if (response.code === 0) {
        return response.data?.package;
      }
      throw new Error(response.msg || 'Failed to export skill package');
    },
    [spaceId],
  );

  const listSkillAssets = useCallback(
    async (skillId: string) => {
      const response = await skill.SuperAgentListSkillAssets({
        space_id: spaceId,
        skill_id: skillId,
      });
      if (response.code === 0) {
        return response.data?.assets || [];
      }
      throw new Error(response.msg || 'Failed to list skill assets');
    },
    [spaceId],
  );

  const getSkillAsset = useCallback(
    async (skillId: string, path: string) => {
      const response = await skill.SuperAgentGetSkillAsset({
        space_id: spaceId,
        skill_id: skillId,
        path,
      });
      if (response.code === 0) {
        return response.data?.asset;
      }
      throw new Error(response.msg || 'Failed to get skill asset');
    },
    [spaceId],
  );

  const deleteSkillAsset = useCallback(
    async (skillId: string, path: string) => {
      const response = await skill.SuperAgentDeleteSkillAsset({
        space_id: spaceId,
        skill_id: skillId,
        path,
      });
      if (response.code === 0) {
        await fetchSkillList();
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to delete skill asset');
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
        files?: Record<string, string>;
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

  const publishSkill = useCallback(
    async (skillId: string, scope: number) => {
      const response = await skill.PublishSkill({
        skill_id: skillId,
        space_id: spaceId,
        scope,
      });
      if (response.code === 0) {
        await fetchSkillList();
        await fetchMarketplaceSkills();
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to publish skill');
    },
    [spaceId, fetchSkillList, fetchMarketplaceSkills],
  );

  const installMarketplaceSkill = useCallback(
    async (skillId: string) => {
      const response = await skill.InstallMarketplaceSkill({
        skill_id: skillId,
        space_id: spaceId,
      });
      if (response.code === 0) {
        await fetchSkillList();
        await fetchMarketplaceSkills();
        return response.data?.skill_info;
      }
      throw new Error(response.msg || 'Failed to install skill');
    },
    [spaceId, fetchSkillList, fetchMarketplaceSkills],
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
    marketplaceList,
    loading,
    marketplaceLoading,
    total,
    marketplaceTotal,
    keyword,
    setKeyword,
    fetchSkillList,
    fetchMarketplaceSkills,
    createSkill,
    importSkillPackage,
    validateSkillPackage,
    exportSkillPackage,
    listSkillAssets,
    getSkillAsset,
    deleteSkillAsset,
    updateSkill,
    deleteSkill,
    publishSkill,
    installMarketplaceSkill,
    getSkill,
  };
}
