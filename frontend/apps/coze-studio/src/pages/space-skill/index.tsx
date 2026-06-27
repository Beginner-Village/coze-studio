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

/* eslint-disable @coze-arch/max-line-per-function */
import { useParams, useNavigate } from 'react-router-dom';
import React, { useEffect, useRef, useState } from 'react';

import { Modal } from '@coze-arch/coze-design';

import type { SkillInfo } from './types';
import { SkillPageView } from './SkillPageView';
import { useSkillManagement } from './hooks/use-skill-management';
import {
  MARKETPLACE_VIEW_CONFIG,
  SKILL_PUBLISH_SCOPE,
  type SkillView,
} from './constants';
import { agentAppApi } from './agent-app-api';

const readFileAsDataURL = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });

const downloadDataURL = (filename: string, content: string) => {
  const link = document.createElement('a');
  link.href = content;
  link.download = filename || 'skill-package.zip';
  document.body.appendChild(link);
  link.click();
  link.remove();
};

const SpaceSkillPage: React.FC = () => {
  const { space_id } = useParams<{ space_id: string }>();
  const navigate = useNavigate();
  const {
    skillList,
    marketplaceList,
    loading,
    marketplaceLoading,
    total,
    marketplaceTotal,
    keyword,
    setKeyword,
    fetchMarketplaceSkills,
    deleteSkill,
    publishSkill,
    installMarketplaceSkill,
    importSkillPackage,
    validateSkillPackage,
    exportSkillPackage,
  } = useSkillManagement(space_id || '');

  const [activeView, setActiveView] = useState<SkillView>('mine');
  const [reviewCount, setReviewCount] = useState(0);
  const [importing, setImporting] = useState(false);
  const [exportingSkillId, setExportingSkillId] = useState('');
  const [importError, setImportError] = useState('');
  const zipInputRef = useRef<HTMLInputElement | null>(null);

  const marketplaceScope =
    activeView === 'space-market' || activeView === 'global-market'
      ? MARKETPLACE_VIEW_CONFIG[activeView].scope
      : undefined;

  useEffect(() => {
    if (activeView === 'space-market' || activeView === 'global-market') {
      fetchMarketplaceSkills(marketplaceScope);
    }
  }, [activeView, fetchMarketplaceSkills, marketplaceScope]);

  const currentList = activeView === 'mine' ? skillList : marketplaceList;
  const currentLoading = activeView === 'mine' ? loading : marketplaceLoading;
  const currentTotal =
    activeView === 'mine'
      ? total
      : activeView === 'review'
        ? reviewCount
        : marketplaceTotal;

  const handleCreate = () => {
    navigate(`/space/${space_id}/skill-detail/create`);
  };

  const handleImportClick = () => {
    setImportError('');
    zipInputRef.current?.click();
  };

  const handleZipInputChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) {
      return;
    }
    if (!file.name.toLowerCase().endsWith('.zip')) {
      setImportError('请上传标准技能 ZIP 包');
      return;
    }
    try {
      setImporting(true);
      setImportError('');
      const content = await readFileAsDataURL(file);
      const validation = await validateSkillPackage({
        filename: file.name,
        content,
      });
      if (validation && !validation.valid) {
        setImportError(validation.error || '标准技能 ZIP 包校验失败');
        return;
      }
      await importSkillPackage({ filename: file.name, content });
      setActiveView('mine');
    } catch (error) {
      setImportError(
        error instanceof Error ? error.message : '标准技能 ZIP 包上传失败',
      );
    } finally {
      setImporting(false);
    }
  };

  const handleEdit = (skillItem: SkillInfo) => {
    navigate(
      `/space/${space_id}/skill-detail/edit?skill_id=${skillItem.skill_id}`,
    );
  };

  const handleDelete = (skillItem: SkillInfo) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除技能「${skillItem.name}」吗？删除后无法恢复。`,
      okText: '删除',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteSkill(skillItem.skill_id);
        } catch (error) {
          console.error('删除失败:', error);
        }
      },
    });
  };

  const handlePublish = (skillItem: SkillInfo, scope: number) => {
    const scopeLabel =
      scope === SKILL_PUBLISH_SCOPE.GLOBAL
        ? '全局商城'
        : scope === SKILL_PUBLISH_SCOPE.SPACE
          ? '空间商城'
          : '私有';
    Modal.confirm({
      title: scope === SKILL_PUBLISH_SCOPE.PRIVATE ? '确认下架' : '确认发布',
      content:
        scope === SKILL_PUBLISH_SCOPE.PRIVATE
          ? `确定将技能「${skillItem.name}」下架为私有吗？`
          : `确定将技能「${skillItem.name}」发布到${scopeLabel}吗？`,
      okText: scope === SKILL_PUBLISH_SCOPE.PRIVATE ? '下架' : '发布',
      cancelText: '取消',
      onOk: async () => {
        try {
          await publishSkill(skillItem.skill_id, scope);
        } catch (error) {
          console.error('发布失败:', error);
        }
      },
    });
  };

  const handleInstall = (skillItem: SkillInfo) => {
    Modal.confirm({
      title: '安装技能',
      content: `将技能「${skillItem.name}」安装到当前空间。安装后会生成一份私有副本，可继续编辑。`,
      okText: '安装',
      cancelText: '取消',
      onOk: async () => {
        try {
          await installMarketplaceSkill(skillItem.skill_id);
          setActiveView('mine');
        } catch (error) {
          console.error('安装失败:', error);
        }
      },
    });
  };

  const handleRecruit = async (skillItem: SkillInfo) => {
    try {
      const result = await agentAppApi.recruit({
        product_id: skillItem.product_id || skillItem.skill_id,
        space_id: space_id || '',
      });
      const shadowAgentId = result?.data?.shadow_agent_id;
      if (shadowAgentId) {
        navigate(`/space/${space_id}/bot/${shadowAgentId}/arrange`);
      }
    } catch (error) {
      console.error('招聘失败:', error);
    }
  };

  const handleExport = async (skillItem: SkillInfo) => {
    try {
      setImportError('');
      setExportingSkillId(skillItem.skill_id);
      const pkg = await exportSkillPackage(skillItem.skill_id);
      if (!pkg?.content) {
        throw new Error('标准技能 ZIP 包下载失败');
      }
      downloadDataURL(
        pkg.filename || `${skillItem.name || 'skill'}.zip`,
        pkg.content,
      );
    } catch (error) {
      setImportError(
        error instanceof Error ? error.message : '标准技能 ZIP 包下载失败',
      );
    } finally {
      setExportingSkillId('');
    }
  };

  return (
    <SkillPageView
      spaceId={space_id || ''}
      skillList={skillList}
      currentList={currentList}
      currentLoading={currentLoading}
      currentTotal={currentTotal}
      activeView={activeView}
      keyword={keyword}
      importing={importing}
      importError={importError}
      exportingSkillId={exportingSkillId}
      zipInputRef={zipInputRef}
      setActiveView={setActiveView}
      setKeyword={setKeyword}
      onCreate={handleCreate}
      onImportClick={handleImportClick}
      onZipInputChange={handleZipInputChange}
      onEdit={handleEdit}
      onDelete={handleDelete}
      onPublish={handlePublish}
      onInstall={handleInstall}
      onExport={handleExport}
      onRecruit={handleRecruit}
      onReviewCountChange={setReviewCount}
    />
  );
};

export default SpaceSkillPage;
