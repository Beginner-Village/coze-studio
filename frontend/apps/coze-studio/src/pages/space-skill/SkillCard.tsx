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
  IconCozEdit,
  IconCozImport,
  IconCozMore,
  IconCozPeople,
  IconCozRefresh,
  IconCozStarFill,
  IconCozTrashCan,
} from '@coze-arch/coze-design/icons';
import { Button, Dropdown, IconButton } from '@coze-arch/coze-design';

import type { SkillInfo } from './types';
import { SKILL_PUBLISH_SCOPE } from './constants';

import styles from './index.module.less';

interface SkillAssetSummary {
  count?: number;
  image_paths?: string[];
}

type SkillWithAssetSummary = SkillInfo & {
  asset_summary?: SkillAssetSummary;
};

type SkillTone = 'pack' | 'pdf' | 'doc' | 'sheet' | 'slide';

const TONE_CLASS: Record<SkillTone, string> = {
  pack: styles.tonePack,
  pdf: styles.tonePdf,
  doc: styles.toneDoc,
  sheet: styles.toneSheet,
  slide: styles.toneSlide,
};

export const getSkillFiles = (skill: SkillInfo) => skill.files ?? {};

export const getSkillFileCount = (skill: SkillInfo) =>
  Object.keys(getSkillFiles(skill)).length;

export const isStandardSkill = (skill: SkillInfo) =>
  Boolean(getSkillFiles(skill)['SKILL.md']);

export const isPublishedSkill = (skill: SkillInfo) =>
  skill.publish_scope === SKILL_PUBLISH_SCOPE.SPACE ||
  skill.publish_scope === SKILL_PUBLISH_SCOPE.GLOBAL;

export const getAssetSummary = (skill: SkillInfo): SkillAssetSummary => {
  const summary = (skill as SkillWithAssetSummary).asset_summary;
  if (summary) {
    return summary;
  }

  const imagePaths = Object.entries(getSkillFiles(skill))
    .filter(([path, content]) => {
      const fileContent = typeof content === 'string' ? content : '';
      if (!path.startsWith('assets/')) {
        return false;
      }
      return (
        fileContent.startsWith('data:image/') ||
        /\.(png|jpe?g|webp|gif|svg)$/i.test(path)
      );
    })
    .map(([path]) => path)
    .sort();

  return {
    count: Object.keys(getSkillFiles(skill)).filter(path =>
      path.startsWith('assets/'),
    ).length,
    image_paths: imagePaths,
  };
};

export const formatDate = (timestamp?: number) => {
  if (!timestamp) {
    return '-';
  }
  return new Date(timestamp).toLocaleDateString('zh-CN');
};

const getSkillCover = (skill: SkillInfo) => {
  if (
    skill.icon_uri &&
    (/^(https?:)?\/\//.test(skill.icon_uri) ||
      skill.icon_uri.startsWith('/') ||
      skill.icon_uri.startsWith('data:image/'))
  ) {
    return skill.icon_uri;
  }

  const firstImagePath = getAssetSummary(skill).image_paths?.[0];
  const content = firstImagePath ? getSkillFiles(skill)[firstImagePath] : '';
  return typeof content === 'string' && content.startsWith('data:image/')
    ? content
    : '';
};

const getPublishScopeLabel = (skill: SkillInfo) => {
  if (skill.publish_scope === SKILL_PUBLISH_SCOPE.GLOBAL) {
    return '全局商城';
  }
  if (skill.publish_scope === SKILL_PUBLISH_SCOPE.SPACE) {
    return '空间商城';
  }
  return '私有';
};

const getSkillMetadataBadges = (skill: SkillInfo) => {
  const { metadata } = skill;
  if (!metadata) {
    return [];
  }

  return [
    metadata.category,
    metadata.version ? `v${metadata.version}` : '',
    metadata.tags?.[0],
    metadata.platforms?.[0],
  ].filter(Boolean) as string[];
};

const getSkillTone = (skill: SkillInfo): SkillTone => {
  const haystack = [
    skill.name,
    skill.description,
    skill.metadata?.category,
    ...(skill.metadata?.tags ?? []),
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();

  if (/\b(pdf)\b/.test(haystack)) {
    return 'pdf';
  }
  if (/\b(ppt|pptx|slide|presentation)\b/.test(haystack)) {
    return 'slide';
  }
  if (/\b(doc|docx|word)\b/.test(haystack)) {
    return 'doc';
  }
  if (/\b(xls|xlsx|excel|sheet|spreadsheet)\b/.test(haystack)) {
    return 'sheet';
  }
  return 'pack';
};

const getSkillInitial = (skill: SkillInfo) =>
  (skill.name?.trim()?.slice(0, 1) || 'S').toUpperCase();

const SkillCardActions: React.FC<{
  skill: SkillInfo;
  exporting?: boolean;
  readonly?: boolean;
  onEdit: (skill: SkillInfo) => void;
  onDelete: (skill: SkillInfo) => void;
  onPublish: (skill: SkillInfo, scope: number) => void;
  onInstall: (skill: SkillInfo) => void;
  onExport: (skill: SkillInfo) => void;
  onRecruit?: (skill: SkillInfo) => void;
}> = ({
  skill,
  exporting,
  readonly,
  onEdit,
  onDelete,
  onPublish,
  onInstall,
  onExport,
  onRecruit,
}) => {
  if (readonly && skill.type === 'agent_app') {
    return (
      <div className={styles.cardActions}>
        <Button
          size="small"
          color="primary"
          className={`${styles.marketButton} ${styles.installButton}`}
          icon={<IconCozPeople />}
          onClick={() => onRecruit?.(skill)}
        >
          招聘
        </Button>
      </div>
    );
  }

  return readonly ? (
    <div className={styles.cardActions}>
      <Button
        size="small"
        color="secondary"
        className={styles.marketButton}
        icon={<IconCozImport />}
        loading={exporting}
        disabled={exporting}
        onClick={() => onExport(skill)}
      >
        下载 ZIP
      </Button>
      <Button
        size="small"
        color="primary"
        className={`${styles.marketButton} ${styles.installButton}`}
        icon={<IconCozImport />}
        onClick={() => onInstall(skill)}
      >
        安装
      </Button>
    </div>
  ) : (
    <div className={styles.cardActions}>
      <Button
        size="small"
        color="secondary"
        className={styles.cardButton}
        icon={<IconCozEdit />}
        onClick={() => onEdit(skill)}
      >
        编辑
      </Button>
      <Dropdown
        trigger="click"
        position="bottomRight"
        render={
          <Dropdown.Menu>
            <Dropdown.Item
              icon={<IconCozPeople />}
              onClick={() => onPublish(skill, SKILL_PUBLISH_SCOPE.SPACE)}
            >
              发布到空间商城
            </Dropdown.Item>
            <Dropdown.Item
              icon={<IconCozStarFill />}
              onClick={() => onPublish(skill, SKILL_PUBLISH_SCOPE.GLOBAL)}
            >
              发布到全局商城
            </Dropdown.Item>
            {isPublishedSkill(skill) ? (
              <Dropdown.Item
                icon={<IconCozRefresh />}
                onClick={() => onPublish(skill, SKILL_PUBLISH_SCOPE.PRIVATE)}
              >
                下架为私有
              </Dropdown.Item>
            ) : null}
            <Dropdown.Item
              icon={<IconCozImport />}
              onClick={() => onExport(skill)}
            >
              下载 ZIP
            </Dropdown.Item>
            <Dropdown.Item
              icon={<IconCozTrashCan />}
              type="danger"
              onClick={() => onDelete(skill)}
            >
              删除
            </Dropdown.Item>
          </Dropdown.Menu>
        }
      >
        <IconButton className={styles.moreButton} icon={<IconCozMore />} />
      </Dropdown>
    </div>
  );
};

export const SkillCard: React.FC<{
  skill: SkillInfo;
  readonly?: boolean;
  exporting?: boolean;
  onEdit: (skill: SkillInfo) => void;
  onDelete: (skill: SkillInfo) => void;
  onPublish: (skill: SkillInfo, scope: number) => void;
  onInstall: (skill: SkillInfo) => void;
  onExport: (skill: SkillInfo) => void;
  onRecruit?: (skill: SkillInfo) => void;
}> = props => {
  const { skill } = props;
  const cover = getSkillCover(skill);
  const metadataBadges = getSkillMetadataBadges(skill);
  const assetSummary = getAssetSummary(skill);
  const fileCount = getSkillFileCount(skill);
  const publishScopeLabel = getPublishScopeLabel(skill);
  const publishedVersion = skill.published_version
    ? `v${skill.published_version}`
    : '–';

  return (
    <article className={styles.card}>
      <div className={styles.cardHeader}>
        <div className={`${styles.avatar} ${TONE_CLASS[getSkillTone(skill)]}`}>
          {cover ? (
            <img src={cover} alt="" draggable={false} />
          ) : (
            getSkillInitial(skill)
          )}
        </div>
        <div className={styles.cardText}>
          <div className={styles.nameRow}>
            <h3 className={styles.cardName} title={skill.name}>
              {skill.name || '未命名技能'}
            </h3>
            <span
              className={`${styles.badge} ${
                publishScopeLabel !== '私有' ? styles.badgeGlobal : ''
              }`}
            >
              {publishScopeLabel}
            </span>
          </div>
          <p className={styles.cardDesc} title={skill.description}>
            {skill.description || '暂无描述'}
          </p>
        </div>
      </div>

      <div className={styles.tags}>
        <span className={styles.tag}>
          {isStandardSkill(skill) ? '标准技能包' : '兼容技能'}
        </span>
        <span className={styles.tag}>{fileCount} 文件</span>
        <span className={styles.tag}>{assetSummary.count ?? 0} 资产</span>
        {metadataBadges.map(badge => (
          <span key={badge} className={styles.tag} title={badge}>
            {badge}
          </span>
        ))}
      </div>

      <div className={styles.meta}>
        <div>
          <div className={styles.metaKey}>当前版本</div>
          <div className={styles.metaValue}>v{skill.version ?? 1}</div>
        </div>
        <div>
          <div className={styles.metaKey}>发布版本</div>
          <div
            className={`${styles.metaValue} ${
              skill.published_version
                ? styles.metaValueBrand
                : styles.metaValueDim
            }`}
          >
            {publishedVersion}
          </div>
        </div>
        <div>
          <div className={styles.metaKey}>更新</div>
          <div className={styles.metaValue}>{formatDate(skill.updated_at)}</div>
        </div>
      </div>

      <div className={styles.cardFooter}>
        <span className={styles.createdAt}>
          创建 {formatDate(skill.created_at)}
        </span>
        <SkillCardActions {...props} />
      </div>
    </article>
  );
};
