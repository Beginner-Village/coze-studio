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

import type { FalconCard } from '../types';

interface CardSelectorProps {
  selectedCard: FalconCard | null;
  showDropdown: boolean;
  loading: boolean;
  cards: FalconCard[];
  searchKeyword: string;
  onToggleDropdown: () => void;
  onCardSelect: (card: FalconCard) => void;
  onSearchChange: (keyword: string) => void;
}

export function CardSelector({
  selectedCard,
  showDropdown,
  loading,
  cards,
  searchKeyword,
  onToggleDropdown,
  onCardSelect,
  onSearchChange,
}: CardSelectorProps) {
  return (
    <div style={{ marginBottom: 16, position: 'relative' }}>
      <div
        style={{
          fontSize: '12px',
          fontWeight: 600,
          marginBottom: 8,
          color: 'var(--semi-color-text-0)',
        }}
      >
        选择卡片
      </div>

      {/* Selection Input */}
      <div
        onClick={onToggleDropdown}
        style={{
          width: '100%',
          padding: '8px 12px',
          border: '1px solid var(--semi-color-border)',
          borderRadius: '6px',
          fontSize: '14px',
          background: 'var(--semi-color-bg-0)',
          cursor: 'pointer',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <span
          style={{
            color: selectedCard
              ? 'var(--semi-color-text-0)'
              : 'var(--semi-color-text-2)',
          }}
        >
          {selectedCard
            ? `${selectedCard.cardName} (${selectedCard.code})`
            : '点击选择卡片...'}
        </span>
        <span
          style={{
            transform: showDropdown ? 'rotate(180deg)' : 'rotate(0deg)',
            transition: 'transform 0.2s',
          }}
        >
          ▼
        </span>
      </div>

      {/* Dropdown */}
      {showDropdown ? (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            background: 'var(--semi-color-bg-0)',
            border: '1px solid var(--semi-color-border)',
            borderRadius: '6px',
            marginTop: '4px',
            zIndex: 1000,
            maxHeight: '300px',
            overflowY: 'auto',
            boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)',
          }}
        >
          {/* Search in dropdown */}
          <div style={{ padding: '8px' }}>
            <input
              placeholder="搜索卡片名称或代码..."
              value={searchKeyword}
              onChange={e => onSearchChange(e.target.value)}
              onClick={e => e.stopPropagation()}
              style={{
                width: '100%',
                padding: '6px 8px',
                border: '1px solid var(--semi-color-border)',
                borderRadius: '4px',
                fontSize: '12px',
              }}
            />
          </div>

          {/* Card list */}
          {loading ? (
            <div
              style={{
                padding: '20px',
                textAlign: 'center',
                color: 'var(--semi-color-text-2)',
              }}
            >
              加载中...
            </div>
          ) : cards.length > 0 ? (
            cards.map(card => (
              <div
                key={card.cardId}
                onClick={() => onCardSelect(card)}
                style={{
                  padding: '12px',
                  borderBottom: '1px solid var(--semi-color-border)',
                  cursor: 'pointer',
                  ':hover': {
                    background: 'var(--semi-color-fill-0)',
                  },
                }}
                onMouseEnter={e => {
                  e.currentTarget.style.background = 'var(--semi-color-fill-0)';
                }}
                onMouseLeave={e => {
                  e.currentTarget.style.background = 'transparent';
                }}
              >
                <div style={{ fontSize: '14px', fontWeight: 600 }}>
                  {card.cardName}
                </div>
                <div
                  style={{
                    fontSize: '12px',
                    color: 'var(--semi-color-text-2)',
                    marginTop: 2,
                  }}
                >
                  代码: {card.code}
                </div>
              </div>
            ))
          ) : (
            <div
              style={{
                padding: '20px',
                textAlign: 'center',
                color: 'var(--semi-color-text-2)',
              }}
            >
              {searchKeyword ? '未找到匹配的卡片' : '暂无卡片数据'}
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}
