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

import type { Parameter } from '@coze-workflow/base';
import { workflow } from '@coze-studio/api-schema';

import { useField, withField, useForm } from '@/form';

import type {
  CardSelectorParams,
  FalconCard,
  CardDetail,
  CardParam,
} from '../types';
import { INPUT_PATH } from '../constants';
import { SelectedCardInfo } from './selected-card-info';
import { CardSelector } from './card-selector';

interface CardSelectorFieldProps {
  tooltip?: string;
}

// 将卡片参数类型转换为workflow参数类型
const convertCardParamType = (cardParamType: string): string => {
  switch (cardParamType.toLowerCase()) {
    case 'string':
      return 'str';
    case 'number':
    case 'int':
    case 'integer':
      return 'int';
    case 'boolean':
    case 'bool':
      return 'bool';
    case 'array':
      return 'array';
    case 'object':
      return 'object';
    default:
      return 'str';
  }
};

// 递归转换卡片参数为workflow输入参数
const convertCardParamsToInputs = (cardParams: CardParam[]): Parameter[] => {
  const convertParam = (param: CardParam): Parameter => {
    const baseParam: Parameter = {
      name: param.paramName,
      type: convertCardParamType(param.paramType),
      description: param.paramDesc,
      required: param.isRequired === '1',
    };

    // 如果有子参数（对于array或object类型）
    if (param.children && param.children.length > 0) {
      if (param.paramType.toLowerCase() === 'array') {
        // 对于数组类型，取第一个子参数作为数组元素的结构
        const firstChild = param.children[0];
        if (firstChild.children && firstChild.children.length > 0) {
          // 如果数组元素是对象，生成对象结构描述
          const childFields = firstChild.children.map(child => ({
            name: child.paramName,
            type: convertCardParamType(child.paramType),
            description: child.paramDesc,
            required: child.isRequired === '1',
          }));
          baseParam.schema = {
            type: 'array',
            items: {
              type: 'object',
              properties: childFields.reduce(
                (props: Record<string, unknown>, field) => {
                  props[field.name] = {
                    type: field.type,
                    description: field.description,
                  };
                  return props;
                },
                {},
              ),
              required: childFields.filter(f => f.required).map(f => f.name),
            },
          };
        }
      } else if (param.paramType.toLowerCase() === 'object') {
        // 对于对象类型，递归处理子参数
        const childParams = convertCardParamsToInputs(param.children);
        baseParam.schema = {
          type: 'object',
          properties: childParams.reduce(
            (props: Record<string, unknown>, child) => {
              props[child.name] = {
                type: child.type,
                description: child.description,
              };
              return props;
            },
            {},
          ),
          required: childParams.filter(p => p.required).map(p => p.name),
        };
      }
    }

    return baseParam;
  };

  return cardParams.map(convertParam);
};

export const CardSelectorField = withField(
  ({ tooltip }: CardSelectorFieldProps) => {
    const { value, onChange, errors } = useField<CardSelectorParams>();
    const form = useForm();

    const [cards, setCards] = useState<FalconCard[]>([]);
    const [loading, setLoading] = useState(false);
    const [showDropdown, setShowDropdown] = useState(false);
    const [searchKeyword, setSearchKeyword] = useState('');
    const [selectedCard, setSelectedCard] = useState<FalconCard | null>(null);
    const [cardDetail, setCardDetail] = useState<CardDetail | null>(null);
    const [loadingDetail, setLoadingDetail] = useState(false);

    // Fetch cards from backend API
    const fetchCards = useCallback(
      async (searchValue = '') => {
        setLoading(true);
        try {
          // 使用默认API地址
          const apiEndpoint = 'http://10.10.10.208:8500/aop-web';

          // 使用生成的API客户端调用后端卡片列表 API
          const result = await workflow.GetCardList({
            apiEndpoint,
            searchKeyword: searchValue || '',
            filters: {},
          });

          if (result.code === 0 && result.data) {
            setCards(result.data.cardList || []);
          } else {
            throw new Error(result.message || 'Failed to fetch card list');
          }
        } catch (error) {
          console.error('Failed to fetch card list:', error);
          setCards([]);
        } finally {
          setLoading(false);
        }
      },
      [],
    );

    // Fetch card details by ID from backend API
    const fetchCardDetail = useCallback(
      async (cardId: string) => {
        setLoadingDetail(true);
        try {
          // 使用默认API地址
          const apiEndpoint = 'http://10.10.10.208:8500/aop-web';

          // 使用生成的API客户端调用后端卡片详情 API
          const result = await workflow.GetCardDetail({
            apiEndpoint,
            cardId,
          });

          if (result.code === 0 && result.data) {
            setCardDetail(result.data);

            // 自动更新输入参数配置
            if (
              result.data.paramList &&
              result.data.paramList.length > 0
            ) {
              const convertedParams = convertCardParamsToInputs(
                result.data.paramList,
              );

              // 更新表单中的输入参数
              form.setFieldValue(INPUT_PATH, convertedParams);
            }
          } else {
            throw new Error(result.message || 'Failed to fetch card detail');
          }
        } catch (error) {
          console.error('Failed to fetch card detail:', error);
          setCardDetail(null);
        } finally {
          setLoadingDetail(false);
        }
      },
      [form],
    );

    // Handle card selection
    const handleCardSelect = useCallback(
      async (card: FalconCard) => {
        setSelectedCard(card);
        setShowDropdown(false);
        setSearchKeyword('');

        onChange({
          ...value,
          selectedCardId: card.cardId,
        });

        // Fetch card details by ID
        await fetchCardDetail(card.cardId);
      },
      [value, onChange, fetchCardDetail],
    );


    // Handle toggle dropdown
    const handleToggleDropdown = useCallback(() => {
      setShowDropdown(!showDropdown);
      if (!showDropdown) {
        fetchCards();
      }
    }, [showDropdown, fetchCards]);

    // Handle search in dropdown
    const handleSearchInDropdown = useCallback(
      (searchValue: string) => {
        setSearchKeyword(searchValue);
        fetchCards(searchValue);
      },
      [fetchCards],
    );

    // Initialize card if selectedCardId exists
    useEffect(() => {
      if (value?.selectedCardId && !selectedCard) {
        // Find the card in current cards list
        const found = cards.find(c => c.cardId === value.selectedCardId);
        if (found) {
          setSelectedCard(found);
        }
      }
    }, [value?.selectedCardId, selectedCard, cards]);

    const feedbackText = errors?.[0]?.message || '';

    // Log form validation errors to console for debugging
    if (feedbackText) {
      console.warn('⚠️ CardSelector Form Validation Error:', {
        error: feedbackText,
        fieldValue: value,
        selectedCard,
        cardDetail,
      });
    }

    return (
      <div style={{ width: '100%' }}>
        <CardSelector
          selectedCard={selectedCard}
          showDropdown={showDropdown}
          loading={loading}
          cards={cards}
          searchKeyword={searchKeyword}
          onToggleDropdown={handleToggleDropdown}
          onCardSelect={handleCardSelect}
          onSearchChange={handleSearchInDropdown}
        />

        <SelectedCardInfo
          selectedCard={selectedCard}
          cardDetail={cardDetail}
          loadingDetail={loadingDetail}
        />

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
