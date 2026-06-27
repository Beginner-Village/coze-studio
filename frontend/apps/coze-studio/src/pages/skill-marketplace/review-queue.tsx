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

import React, { useCallback, useEffect, useState } from 'react';

import { axiosInstance } from '@coze-arch/bot-api';
import { Modal, Spin, TextArea, Toast } from '@coze-arch/coze-design';
import { IconCozPlugin, IconCozStore } from '@coze-arch/coze-design/icons';

import styles from './index.module.less';

interface PendingSkill {
  skill_id: string;
  name?: string;
  description?: string;
  icon_uri?: string;
  review_status?: number;
}

interface PendingResponse {
  code?: number;
  msg?: string;
  data?: {
    skill_list?: PendingSkill[];
    total?: number;
  };
}

interface ReviewResponse {
  code?: number;
  msg?: string;
}

const isDisplayableIcon = (uri?: string) =>
  Boolean(
    uri &&
      (/^(https?:)?\/\//.test(uri) ||
        uri.startsWith('/') ||
        uri.startsWith('data:image/')),
  );

const SKILL_TONES = [
  'linear-gradient(150deg, rgb(94,170,255), rgb(53,138,255))',
  'linear-gradient(150deg, rgb(78,217,196), rgb(0,178,120))',
  'linear-gradient(150deg, rgb(186,140,255), rgb(167,0,250))',
  'linear-gradient(150deg, rgb(255,184,122), rgb(255,138,61))',
];

const getSkillTone = (item: PendingSkill) => {
  const source = item.skill_id || item.name || '';
  const hash = Array.from(source).reduce(
    (sum, char) => sum + char.charCodeAt(0),
    0,
  );
  return SKILL_TONES[hash % SKILL_TONES.length];
};

const ReviewCard: React.FC<{
  item: PendingSkill;
  pending: boolean;
  onApprove: (item: PendingSkill) => void;
  onReject: (item: PendingSkill) => void;
}> = ({ item, pending, onApprove, onReject }) => {
  const icon = isDisplayableIcon(item.icon_uri) ? item.icon_uri : '';
  return (
    <article
      className={`${styles.scard} ${styles.reviewCard}`}
      data-testid="skill-review-card"
    >
      <div className={styles.scardHd}>
        <div
          className={styles.scardIcon}
          style={{ background: getSkillTone(item) }}
        >
          <div className={styles.scardIconInner}>
            {icon ? (
              <img
                src={icon}
                alt=""
                draggable={false}
                className={styles.scardCover}
              />
            ) : (
              <IconCozPlugin />
            )}
          </div>
        </div>
        <div className={styles.scardHeaderText}>
          <div className={styles.scardNameRow}>
            <h3 className={styles.scardName} title={item.name}>
              {item.name || '未命名技能'}
            </h3>
            <span className={`${styles.badge} ${styles.badgeGlobal}`}>
              待审核
            </span>
          </div>
          <p className={styles.scardDesc} title={item.description}>
            {item.description || '暂无描述'}
          </p>
        </div>
      </div>

      <div className={styles.scardFooter}>
        <button
          type="button"
          className={styles.detailButton}
          disabled={pending}
          onClick={() => onReject(item)}
        >
          驳回
        </button>
        <div className={styles.spacer} />
        <button
          type="button"
          className={styles.installButton}
          disabled={pending}
          onClick={() => onApprove(item)}
        >
          {pending ? '处理中' : '通过'}
        </button>
      </div>
    </article>
  );
};

const ReviewQueue: React.FC<{
  spaceId: string;
  onCountChange?: (count: number) => void;
}> = ({ spaceId, onCountChange }) => {
  const [items, setItems] = useState<PendingSkill[]>([]);
  const [loading, setLoading] = useState(false);
  const [errorText, setErrorText] = useState('');
  const [processingId, setProcessingId] = useState('');
  const [rejectTarget, setRejectTarget] = useState<PendingSkill | null>(null);
  const [rejectNote, setRejectNote] = useState('');

  useEffect(() => {
    onCountChange?.(items.length);
  }, [items, onCountChange]);

  const fetchPending = useCallback(async () => {
    if (!spaceId) {
      setItems([]);
      setErrorText('请先进入一个工作空间再审核技能');
      return;
    }
    setLoading(true);
    setErrorText('');
    try {
      const response = (await axiosInstance.request({
        url: '/api/skill/review/pending',
        method: 'GET',
        params: { space_id: spaceId, page: 1, page_size: 20 },
        withCredentials: true,
      })) as unknown as PendingResponse;
      if (response.code !== 0) {
        throw new Error(response.msg || '仅空间管理员可审核');
      }
      setItems(response.data?.skill_list || []);
    } catch (error) {
      setItems([]);
      setErrorText(
        error instanceof Error && error.message
          ? error.message
          : '仅空间管理员可审核',
      );
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  useEffect(() => {
    fetchPending();
  }, [fetchPending]);

  const submitReview = useCallback(
    async (item: PendingSkill, approve: boolean, note?: string) => {
      setProcessingId(item.skill_id);
      try {
        const response = (await axiosInstance.request({
          url: '/api/skill/review',
          method: 'POST',
          data: { skill_id: item.skill_id, approve, note: note || '' },
          withCredentials: true,
        })) as unknown as ReviewResponse;
        if (response.code !== 0) {
          throw new Error(response.msg || '操作失败');
        }
        Toast.success(approve ? '已通过，技能进入全局市场' : '已驳回');
        setItems(prev => prev.filter(it => it.skill_id !== item.skill_id));
      } catch (error) {
        Toast.error(error instanceof Error ? error.message : '操作失败');
      } finally {
        setProcessingId('');
      }
    },
    [],
  );

  const handleApprove = useCallback(
    (item: PendingSkill) => {
      submitReview(item, true);
    },
    [submitReview],
  );

  const handleReject = useCallback((item: PendingSkill) => {
    setRejectNote('');
    setRejectTarget(item);
  }, []);

  const confirmReject = useCallback(async () => {
    if (!rejectTarget) {
      return;
    }
    const target = rejectTarget;
    setRejectTarget(null);
    await submitReview(target, false, rejectNote);
  }, [rejectNote, rejectTarget, submitReview]);

  return (
    <div className={styles.marketBody} data-testid="skill-review-queue">
      {loading ? (
        <div className={styles.loading}>
          <Spin size="large" />
        </div>
      ) : errorText ? (
        <div className={styles.empty}>
          <div className={styles.emptyIcon}>
            <IconCozStore />
          </div>
          <div className={styles.emptyText}>{errorText}</div>
        </div>
      ) : items.length === 0 ? (
        <div className={styles.empty}>
          <div className={styles.emptyIcon}>
            <IconCozStore />
          </div>
          <div className={styles.emptyText}>暂无待审核技能</div>
        </div>
      ) : (
        <div className={styles.grid}>
          {items.map(item => (
            <ReviewCard
              key={item.skill_id}
              item={item}
              pending={processingId === item.skill_id}
              onApprove={handleApprove}
              onReject={handleReject}
            />
          ))}
        </div>
      )}

      <Modal
        visible={Boolean(rejectTarget)}
        title={`驳回「${rejectTarget?.name || ''}」`}
        okText="确认驳回"
        cancelText="取消"
        onOk={confirmReject}
        onCancel={() => setRejectTarget(null)}
      >
        <TextArea
          value={rejectNote}
          onChange={value => setRejectNote(value)}
          placeholder="可选：填写驳回原因，作者可见"
          maxCount={200}
          rows={4}
        />
      </Modal>
    </div>
  );
};

export default ReviewQueue;
