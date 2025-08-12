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

import React, { useState, useEffect, useCallback } from 'react';

import { useField, withField } from '@/form';

import type { CardSelectorParams, FalconCard } from '../types';

interface CardSelectorFieldProps {
  tooltip?: string;
}

export const CardSelectorField = withField(
  ({ tooltip }: CardSelectorFieldProps) => {
    const { value, onChange, errors } = useField<CardSelectorParams>();

    const [cards, setCards] = useState<FalconCard[]>([]);
    const [loading, setLoading] = useState(false);
    const [searchKeyword, setSearchKeyword] = useState(
      value?.searchKeyword || '',
    );

    // Fetch cards from API
    const fetchCards = useCallback((keyword = '') => {
      setLoading(true);
      try {
        // Mock API call - replace with actual API endpoint
        const mockCards: FalconCard[] = [
          {
            id: 'card_001',
            name: '用户注册卡片',
            description: '处理用户注册相关功能的卡片',
            category: 'user_management',
          },
          {
            id: 'card_002',
            name: '数据分析卡片',
            description: '提供数据分析和报表生成功能',
            category: 'analytics',
          },
          {
            id: 'card_003',
            name: '消息通知卡片',
            description: '发送各种类型的消息通知',
            category: 'notification',
          },
          {
            id: 'card_004',
            name: '文件处理卡片',
            description: '处理文件上传、下载和转换功能',
            category: 'file_management',
          },
          {
            id: 'card_005',
            name: '支付处理卡片',
            description: '集成支付网关，处理支付流程',
            category: 'payment',
          },
        ];

        // Simple keyword filtering
        let filteredCards = mockCards;
        if (keyword.trim()) {
          const lowerKeyword = keyword.toLowerCase();
          filteredCards = mockCards.filter(
            card =>
              card.name.toLowerCase().includes(lowerKeyword) ||
              card.description.toLowerCase().includes(lowerKeyword) ||
              (card.category &&
                card.category.toLowerCase().includes(lowerKeyword)),
          );
        }

        setCards(filteredCards);
      } catch (error) {
        console.error('Failed to fetch cards:', error);
        setCards([]);
      } finally {
        setLoading(false);
      }
    }, []);

    // Handle search keyword change
    const handleSearchChange = useCallback(
      (searchValue: string) => {
        setSearchKeyword(searchValue);
        onChange({
          ...value,
          searchKeyword: searchValue,
        });
      },
      [value, onChange],
    );

    // Handle card selection
    const handleCardSelect = useCallback(
      (cardId: string) => {
        onChange({
          ...value,
          selectedCardId: cardId,
          searchKeyword,
        });
      },
      [value, onChange, searchKeyword],
    );

    // Handle API configuration
    const handleApiConfigChange = useCallback(
      (field: keyof CardSelectorParams) => (fieldValue: string) => {
        onChange({
          ...value,
          [field]: fieldValue,
        });
      },
      [value, onChange],
    );

    // Initial load
    useEffect(() => {
      fetchCards(searchKeyword);
    }, [fetchCards, searchKeyword]);

    const feedbackText = errors?.[0]?.message || '';

    return (
      <div style={{ width: '100%' }}>
        {/* API Configuration */}
        <div style={{ marginBottom: 16 }}>
          <div
            style={{
              fontSize: '12px',
              fontWeight: 600,
              marginBottom: 8,
              color: 'var(--semi-color-text-0)',
            }}
          >
            API配置
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <input
              placeholder="API端点 (可选)"
              value={value?.apiEndpoint || ''}
              onChange={e =>
                handleApiConfigChange('apiEndpoint')(e.target.value)
              }
              style={{
                padding: '8px 12px',
                border: '1px solid var(--semi-color-border)',
                borderRadius: '6px',
                fontSize: '14px',
              }}
            />
            <input
              type="password"
              placeholder="API Key (可选)"
              value={value?.apiKey || ''}
              onChange={e => handleApiConfigChange('apiKey')(e.target.value)}
              style={{
                padding: '8px 12px',
                border: '1px solid var(--semi-color-border)',
                borderRadius: '6px',
                fontSize: '14px',
              }}
            />
          </div>
        </div>

        {/* Card Search */}
        <div style={{ marginBottom: 16 }}>
          <div
            style={{
              fontSize: '12px',
              fontWeight: 600,
              marginBottom: 8,
              color: 'var(--semi-color-text-0)',
            }}
          >
            搜索卡片
          </div>
          <input
            placeholder="输入关键词搜索卡片..."
            value={searchKeyword}
            onChange={e => handleSearchChange(e.target.value)}
            style={{
              width: '100%',
              padding: '8px 12px',
              border: '1px solid var(--semi-color-border)',
              borderRadius: '6px',
              fontSize: '14px',
            }}
          />
        </div>

        {/* Card Selection */}
        <div style={{ marginBottom: 16 }}>
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

          {loading ? (
            <div
              style={{
                padding: '20px',
                textAlign: 'center',
                color: 'var(--semi-color-text-2)',
                border: '1px solid var(--semi-color-border)',
                borderRadius: '6px',
              }}
            >
              加载中...
            </div>
          ) : (
            <select
              value={value?.selectedCardId || ''}
              onChange={e => handleCardSelect(e.target.value)}
              style={{
                width: '100%',
                padding: '8px 12px',
                border: '1px solid var(--semi-color-border)',
                borderRadius: '6px',
                fontSize: '14px',
                background: 'var(--semi-color-bg-0)',
              }}
            >
              <option value="">选择一个卡片...</option>
              {cards.map(card => (
                <option key={card.id} value={card.id}>
                  {card.name} - {card.description}
                </option>
              ))}
            </select>
          )}
        </div>

        {/* Selected Card Info */}
        {value?.selectedCardId ? (
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
            {(() => {
              const selectedCard = cards.find(
                c => c.id === value.selectedCardId,
              );
              if (selectedCard) {
                return (
                  <div>
                    <div style={{ fontSize: '14px', fontWeight: 600 }}>
                      {selectedCard.name}
                    </div>
                    <div
                      style={{
                        fontSize: '12px',
                        color: 'var(--semi-color-text-2)',
                        marginTop: 4,
                      }}
                    >
                      ID: {selectedCard.id}
                    </div>
                    <div
                      style={{
                        fontSize: '12px',
                        color: 'var(--semi-color-text-1)',
                        marginTop: 4,
                      }}
                    >
                      {selectedCard.description}
                    </div>
                  </div>
                );
              }
              return null;
            })()}
          </div>
        ) : null}

        {/* Error display */}
        {feedbackText ? (
          <div
            style={{
              color: 'var(--semi-color-danger)',
              fontSize: '12px',
              marginTop: 8,
            }}
          >
            {feedbackText}
          </div>
        ) : null}
      </div>
    );
  },
);
