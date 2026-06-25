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

import React from 'react';

import {
  IconCozFolder,
  IconCozImport,
  IconCozPlus,
  IconCozPlugin,
  IconCozStarFill,
  IconCozStore,
} from '@coze-arch/coze-design/icons';
import { Button, Search, Spin } from '@coze-arch/coze-design';

import ReviewQueue from '../skill-marketplace/review-queue';
import type { SkillInfo } from './types';
import {
  getAssetSummary,
  isPublishedSkill,
  isStandardSkill,
  SkillCard,
} from './SkillCard';
import {
  MARKETPLACE_VIEW_CONFIG,
  VIEW_TABS,
  type SkillView,
} from './constants';

import styles from './index.module.less';

interface SkillPageViewProps {
  spaceId: string;
  skillList: SkillInfo[];
  currentList: SkillInfo[];
  currentLoading: boolean;
  currentTotal: number;
  activeView: SkillView;
  keyword: string;
  importing: boolean;
  importError: string;
  exportingSkillId: string;
  zipInputRef: React.RefObject<HTMLInputElement>;
  setActiveView: (view: SkillView) => void;
  setKeyword: (keyword: string) => void;
  onCreate: () => void;
  onImportClick: () => void;
  onZipInputChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onEdit: (skill: SkillInfo) => void;
  onDelete: (skill: SkillInfo) => void;
  onPublish: (skill: SkillInfo, scope: number) => void;
  onInstall: (skill: SkillInfo) => void;
  onExport: (skill: SkillInfo) => void;
  onRecruit?: (skill: SkillInfo) => void;
}

const SkillHero: React.FC<
  Pick<
    SkillPageViewProps,
    | 'skillList'
    | 'importing'
    | 'importError'
    | 'zipInputRef'
    | 'onCreate'
    | 'onImportClick'
    | 'onZipInputChange'
  > & { total: number }
> = ({
  skillList,
  total,
  importing,
  importError,
  zipInputRef,
  onCreate,
  onImportClick,
  onZipInputChange,
}) => {
  const standardCount = skillList.filter(isStandardSkill).length;
  const publishedCount = skillList.filter(isPublishedSkill).length;
  const assetCount = skillList.reduce(
    (sum, item) => sum + (getAssetSummary(item).count ?? 0),
    0,
  );
  const pageStats = [
    {
      label: '我的技能',
      value: total,
      icon: <IconCozPlugin />,
      iconClassName: styles.statIconBrand,
    },
    {
      label: '标准技能包',
      value: standardCount,
      icon: <IconCozImport />,
      iconClassName: styles.statIconAi,
    },
    {
      label: '已发布',
      value: publishedCount,
      icon: <IconCozStarFill />,
      iconClassName: styles.statIconSuccess,
    },
    {
      label: '资产文件',
      value: assetCount,
      icon: <IconCozFolder />,
      iconClassName: styles.statIconAsset,
    },
  ];

  return (
    <div className={styles.hero}>
      <div className={styles.heroTop}>
        <div className={styles.heroHeading}>
          <h1 className={styles.heroTitle}>技能中心</h1>
          <div className={styles.heroDesc}>
            标准技能包、空间技能与全局技能统一管理
          </div>
          {importError ? (
            <div className={styles.importError}>{importError}</div>
          ) : null}
        </div>
        <div className={styles.heroActions}>
          <input
            ref={zipInputRef}
            type="file"
            accept=".zip,application/zip"
            className="hidden"
            onChange={onZipInputChange}
          />
          <Button
            color="secondary"
            className={styles.heroButton}
            icon={<IconCozImport />}
            loading={importing}
            disabled={importing}
            onClick={onImportClick}
          >
            上传 ZIP
          </Button>
          <Button
            type="primary"
            className={styles.createButton}
            icon={<IconCozPlus />}
            onClick={onCreate}
          >
            创建标准技能
          </Button>
        </div>
      </div>

      <div className={styles.stats}>
        {pageStats.map(item => (
          <div key={item.label} className={styles.stat}>
            <div className={styles.statTop}>
              <span className={styles.statLabel}>{item.label}</span>
              <span className={`${styles.statIcon} ${item.iconClassName}`}>
                {item.icon}
              </span>
            </div>
            <div className={styles.statNum}>{item.value}</div>
          </div>
        ))}
      </div>
    </div>
  );
};

const SkillToolbar: React.FC<
  Pick<
    SkillPageViewProps,
    'activeView' | 'currentTotal' | 'keyword' | 'setActiveView' | 'setKeyword'
  >
> = ({ activeView, currentTotal, keyword, setActiveView, setKeyword }) => (
  <div className={styles.bar}>
    <div className={styles.tabs}>
      {VIEW_TABS.map(tab => (
        <button
          key={tab.key}
          type="button"
          className={`${styles.tab} ${
            activeView === tab.key ? styles.tabActive : ''
          }`}
          onClick={() => setActiveView(tab.key)}
        >
          {tab.label}
        </button>
      ))}
    </div>
    <span className={styles.tabCount}>{currentTotal} 个</span>
    <div className={styles.barSpacer} />
    <Search
      showClear
      className={styles.search}
      placeholder="搜索技能名称或描述"
      value={keyword}
      onChange={val => setKeyword(val)}
    />
  </div>
);

const SkillContent: React.FC<
  Pick<
    SkillPageViewProps,
    | 'activeView'
    | 'spaceId'
    | 'currentList'
    | 'currentLoading'
    | 'keyword'
    | 'exportingSkillId'
    | 'onEdit'
    | 'onDelete'
    | 'onPublish'
    | 'onInstall'
    | 'onExport'
    | 'onRecruit'
  >
> = ({
  activeView,
  spaceId,
  currentList,
  currentLoading,
  keyword,
  exportingSkillId,
  onEdit,
  onDelete,
  onPublish,
  onInstall,
  onExport,
  onRecruit,
}) => {
  if (activeView === 'review') {
    return (
      <div className={styles.content}>
        <ReviewQueue spaceId={spaceId} />
      </div>
    );
  }

  const emptyDescription = keyword
    ? '没有找到匹配的技能'
    : activeView === 'mine'
      ? '暂无技能'
      : MARKETPLACE_VIEW_CONFIG[activeView].emptyText;

  if (currentLoading) {
    return (
      <div className={styles.content}>
        <div className={styles.loading}>
          <Spin size="large" />
        </div>
      </div>
    );
  }

  if (currentList.length === 0) {
    return (
      <div className={styles.content}>
        <div className={styles.empty}>
          <div className={styles.emptyIcon}>
            <IconCozStore />
          </div>
          <div className={styles.emptyText}>{emptyDescription}</div>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.content}>
      <div className={styles.grid}>
        {currentList.map(skillItem => (
          <SkillCard
            key={skillItem.skill_id}
            skill={skillItem}
            readonly={activeView !== 'mine'}
            exporting={exportingSkillId === skillItem.skill_id}
            onEdit={onEdit}
            onDelete={onDelete}
            onPublish={onPublish}
            onInstall={onInstall}
            onExport={onExport}
            onRecruit={onRecruit}
          />
        ))}
      </div>
    </div>
  );
};

export const SkillPageView: React.FC<SkillPageViewProps> = props => (
  <div className={styles.page}>
    <SkillHero
      skillList={props.skillList}
      total={props.currentTotal}
      importing={props.importing}
      importError={props.importError}
      zipInputRef={props.zipInputRef}
      onCreate={props.onCreate}
      onImportClick={props.onImportClick}
      onZipInputChange={props.onZipInputChange}
    />
    <SkillToolbar
      activeView={props.activeView}
      currentTotal={props.currentTotal}
      keyword={props.keyword}
      setActiveView={props.setActiveView}
      setKeyword={props.setKeyword}
    />
    <SkillContent
      activeView={props.activeView}
      spaceId={props.spaceId}
      currentList={props.currentList}
      currentLoading={props.currentLoading}
      keyword={props.keyword}
      exportingSkillId={props.exportingSkillId}
      onEdit={props.onEdit}
      onDelete={props.onDelete}
      onPublish={props.onPublish}
      onInstall={props.onInstall}
      onExport={props.onExport}
      onRecruit={props.onRecruit}
    />
  </div>
);
