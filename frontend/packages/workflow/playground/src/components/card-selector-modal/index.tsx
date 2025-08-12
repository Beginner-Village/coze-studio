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

import { useState, useEffect } from 'react';

import { Button, Space } from '@coze-arch/coze-design';

export interface CardItem {
  id: string;
  name: string;
  description: string;
  icon?: string;
  category?: string;
}

export interface CardSelectorModalProps {
  visible: boolean;
  onCancel: () => void;
  onConfirm: (selectedCards: CardItem[]) => void;
  title?: string;
}

// Mock卡片数据 - 实际项目中应该从API获取
const MOCK_CARDS: CardItem[] = [
  {
    id: 'card_001',
    name: '用户信息卡片',
    description: '显示用户基本信息，包括头像、姓名、联系方式等',
    category: '用户管理',
  },
  {
    id: 'card_002',
    name: '订单详情卡片',
    description: '展示订单的详细信息，包括商品、价格、状态等',
    category: '订单管理',
  },
  {
    id: 'card_003',
    name: '数据统计卡片',
    description: '显示关键业务指标和数据图表',
    category: '数据分析',
  },
  {
    id: 'card_004',
    name: '任务进度卡片',
    description: '展示任务执行进度和状态信息',
    category: '任务管理',
  },
  {
    id: 'card_005',
    name: '消息通知卡片',
    description: '显示系统消息和用户通知',
    category: '消息中心',
  },
];

export const CardSelectorModal: React.FC<CardSelectorModalProps> = ({
  visible,
  onCancel,
  onConfirm,
  title = '选择卡片',
}) => {
  const [cards] = useState<CardItem[]>(MOCK_CARDS);
  const [selectedCardIds, setSelectedCardIds] = useState<string[]>([]);
  const [searchKeyword, setSearchKeyword] = useState('');

  // 过滤卡片
  const filteredCards = cards.filter(
    card =>
      card.name.toLowerCase().includes(searchKeyword.toLowerCase()) ||
      card.description.toLowerCase().includes(searchKeyword.toLowerCase()) ||
      (card.category &&
        card.category.toLowerCase().includes(searchKeyword.toLowerCase())),
  );

  // 处理卡片选择
  const handleCardSelect = (cardId: string, checked: boolean) => {
    if (checked) {
      setSelectedCardIds([...selectedCardIds, cardId]);
    } else {
      setSelectedCardIds(selectedCardIds.filter(id => id !== cardId));
    }
  };

  // 全选/取消全选
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedCardIds(filteredCards.map(card => card.id));
    } else {
      setSelectedCardIds([]);
    }
  };

  // 确认选择
  const handleConfirm = () => {
    const selectedCards = cards.filter(card =>
      selectedCardIds.includes(card.id),
    );
    onConfirm(selectedCards);
  };

  // 重置状态
  useEffect(() => {
    if (visible) {
      setSelectedCardIds([]);
      setSearchKeyword('');
    }
  }, [visible]);

  const isAllSelected =
    filteredCards.length > 0 &&
    filteredCards.every(card => selectedCardIds.includes(card.id));

  if (!visible) {
    return null;
  }

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        style={{
          backgroundColor: 'white',
          borderRadius: '8px',
          width: '800px',
          maxHeight: '80vh',
          overflow: 'hidden',
          boxShadow: '0 4px 20px rgba(0, 0, 0, 0.15)',
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: '20px',
            borderBottom: '1px solid #e8e8e8',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <h3 style={{ margin: 0, fontSize: '18px', fontWeight: 'bold' }}>
            {title}
          </h3>
          <button
            onClick={onCancel}
            style={{
              background: 'none',
              border: 'none',
              fontSize: '20px',
              cursor: 'pointer',
              padding: '4px',
            }}
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div style={{ padding: '20px' }}>
          <Space direction="vertical" style={{ width: '100%' }} size={16}>
            {/* 搜索框 */}
            <input
              type="text"
              placeholder="搜索卡片名称、描述或分类"
              value={searchKeyword}
              onChange={e => setSearchKeyword(e.target.value)}
              style={{
                width: '100%',
                padding: '8px 12px',
                border: '1px solid #d9d9d9',
                borderRadius: '4px',
                fontSize: '14px',
              }}
            />

            {/* 全选选项 */}
            <div>
              <label
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  cursor: 'pointer',
                }}
              >
                <input
                  type="checkbox"
                  checked={isAllSelected}
                  onChange={e => handleSelectAll(e.target.checked)}
                  style={{ marginRight: '8px' }}
                />
                全选 ({filteredCards.length} 个卡片)
              </label>
            </div>

            {/* 卡片列表 */}
            <div
              style={{
                maxHeight: '400px',
                overflow: 'auto',
                display: 'grid',
                gridTemplateColumns: 'repeat(2, 1fr)',
                gap: '12px',
              }}
            >
              {filteredCards.map(card => (
                <div
                  key={card.id}
                  onClick={() =>
                    handleCardSelect(
                      card.id,
                      !selectedCardIds.includes(card.id),
                    )
                  }
                  style={{
                    border: selectedCardIds.includes(card.id)
                      ? '2px solid #1890ff'
                      : '1px solid #d9d9d9',
                    borderRadius: '6px',
                    padding: '12px',
                    cursor: 'pointer',
                    backgroundColor: selectedCardIds.includes(card.id)
                      ? '#f6ffed'
                      : 'white',
                    transition: 'all 0.2s',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'flex-start' }}>
                    <input
                      type="checkbox"
                      checked={selectedCardIds.includes(card.id)}
                      onChange={e => {
                        e.stopPropagation();
                        handleCardSelect(card.id, e.target.checked);
                      }}
                      style={{ marginRight: '12px', marginTop: '2px' }}
                    />
                    <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: 500, marginBottom: '4px' }}>
                        {card.name}
                      </div>
                      <div
                        style={{
                          fontSize: '12px',
                          color: '#666',
                          marginBottom: '8px',
                        }}
                      >
                        {card.description}
                      </div>
                      {card.category ? (
                        <div>
                          <span
                            style={{
                              fontSize: '11px',
                              color: '#1890ff',
                              backgroundColor: '#e6f7ff',
                              padding: '2px 6px',
                              borderRadius: '4px',
                            }}
                          >
                            {card.category}
                          </span>
                        </div>
                      ) : null}
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* 无搜索结果提示 */}
            {filteredCards.length === 0 && searchKeyword ? (
              <div
                style={{
                  textAlign: 'center',
                  padding: '40px 0',
                  color: '#999',
                }}
              >
                未找到匹配的卡片
              </div>
            ) : null}
          </Space>
        </div>

        {/* Footer */}
        <div
          style={{
            padding: '20px',
            borderTop: '1px solid #e8e8e8',
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '12px',
          }}
        >
          <Button onClick={onCancel}>取消</Button>
          <Button
            type="primary"
            disabled={selectedCardIds.length === 0}
            onClick={handleConfirm}
          >
            确认选择 ({selectedCardIds.length})
          </Button>
        </div>
      </div>
    </div>
  );
};
