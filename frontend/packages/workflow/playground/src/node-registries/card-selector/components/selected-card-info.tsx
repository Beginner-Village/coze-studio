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

import React from 'react';

import type { FalconCard, CardDetail } from '../types';

interface SelectedCardInfoProps {
  selectedCard: FalconCard | null;
  cardDetail: CardDetail | null;
  loadingDetail: boolean;
}

export function SelectedCardInfo({
  selectedCard,
  cardDetail,
  loadingDetail,
}: SelectedCardInfoProps) {
  if (!selectedCard) {
    return null;
  }

  return (
    <div
      style={{
        padding: '12px',
        background: 'var(--semi-color-fill-0)',
        borderRadius: '6px',
        border: '1px solid var(--semi-color-border)',
      }}
    >
      <div
        style={{
          fontSize: '12px',
          fontWeight: 600,
          marginBottom: 8,
          color: 'var(--semi-color-text-0)',
        }}
      >
        已选择的卡片
      </div>
      <div>
        <div style={{ fontSize: '14px', fontWeight: 600 }}>
          {selectedCard.cardName}
        </div>
        <div
          style={{
            fontSize: '12px',
            color: 'var(--semi-color-text-2)',
            marginTop: 4,
          }}
        >
          ID: {selectedCard.cardId}
        </div>
        <div
          style={{
            fontSize: '12px',
            color: 'var(--semi-color-text-2)',
            marginTop: 2,
          }}
        >
          代码: {selectedCard.code}
        </div>

        {/* Card Detail Info */}
        {loadingDetail ? (
          <div
            style={{
              fontSize: '12px',
              color: 'var(--semi-color-text-2)',
              marginTop: 8,
              fontStyle: 'italic',
            }}
          >
            正在加载卡片详情...
          </div>
        ) : cardDetail ? (
          <div
            style={{
              marginTop: 8,
              paddingTop: 8,
              borderTop: '1px solid var(--semi-color-border)',
            }}
          >
            <div
              style={{
                fontSize: '12px',
                fontWeight: 600,
                marginBottom: 4,
                color: 'var(--semi-color-text-0)',
              }}
            >
              卡片详情
            </div>
            {cardDetail.version ? (
              <div
                style={{
                  fontSize: '12px',
                  color: 'var(--semi-color-text-2)',
                  marginTop: 2,
                }}
              >
                版本: {cardDetail.version}
              </div>
            ) : null}
            {cardDetail.mainUrl ? (
              <div
                style={{
                  fontSize: '12px',
                  color: 'var(--semi-color-text-2)',
                  marginTop: 2,
                }}
              >
                主URL: {cardDetail.mainUrl}
              </div>
            ) : null}
            {cardDetail.paramList && cardDetail.paramList.length > 0 ? (
              <div
                style={{
                  fontSize: '12px',
                  color: 'var(--semi-color-text-2)',
                  marginTop: 2,
                }}
              >
                参数数量: {cardDetail.paramList.length}
              </div>
            ) : null}
          </div>
        ) : (
          <div
            style={{
              fontSize: '12px',
              color: 'var(--semi-color-warning)',
              marginTop: 8,
              fontStyle: 'italic',
            }}
          >
            无法加载卡片详情
          </div>
        )}
      </div>
    </div>
  );
}
