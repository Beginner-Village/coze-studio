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

// ✅ 正确：使用从IDL生成的类型
type FalconCard = workflow.FalconCard;

import { useField, withField, useForm } from '@/form';

import type { CardSelectorParams, CardDetail, CardParam } from '../types';
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
    const [hasAttemptedLoad, setHasAttemptedLoad] = useState(false);

    // Fetch cards from backend API
    const fetchCards = useCallback(async (searchValue = '') => {
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
    }, []);

    // Fetch card details by ID from backend API
    const fetchCardDetail = useCallback(
      async (cardId: string) => {
        setLoadingDetail(true);
        setHasAttemptedLoad(true);
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

            // 🔧 标准做法：如果没有selectedCard，从详情数据创建UI状态
            if (!selectedCard) {
              console.log('🔄 从卡片详情恢复selectedCard状态:', result.data);
              const selectedCardData: FalconCard = {
                cardId: result.data.cardId,
                cardName: result.data.cardName,
                code: result.data.code,
              };
              setSelectedCard(selectedCardData);
            }

            // 🔧 标准做法：增量更新输入参数，保留用户已配置的值
            if (result.data.paramList && result.data.paramList.length > 0) {
              const newParams = convertCardParamsToInputs(
                result.data.paramList,
              );
              
              // 获取用户当前已配置的输入参数
              const currentParams = form.getFieldValue(INPUT_PATH) || [];
              
              // 合并参数：保留已配置的，添加新的
              const mergedParams = newParams.map(newParam => {
                const existingParam = currentParams.find(
                  (p: Parameter) => p.name === newParam.name,
                );
                
                if (existingParam) {
                  console.log('🔄 保留用户已配置的参数:', existingParam.name, existingParam);
                  return existingParam;  // 保留用户配置
                }
                
                console.log('🆕 添加新参数:', newParam.name);
                return newParam;  // 使用新参数默认值
              });

              // 更新表单中的输入参数
              form.setFieldValue(INPUT_PATH, mergedParams);
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
      [form, selectedCard],
    );

    // Handle card selection
    const handleCardSelect = useCallback(
      async (card: FalconCard) => {
        setSelectedCard(card);
        setShowDropdown(false);
        setSearchKeyword('');
        setHasAttemptedLoad(false); // Reset attempted load state for new card

        // 🔧 标准做法：只保存核心业务数据
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

    // 🔧 标准做法：基于业务数据进行状态恢复
    useEffect(() => {
      if (value?.selectedCardId && !selectedCard) {
        console.log('🔄 基于selectedCardId恢复UI状态:', value.selectedCardId);
        
        // 1. 首先尝试从当前cards列表中查找
        const found = cards.find(c => c.cardId === value.selectedCardId);
        if (found) {
          console.log('✅ 从cards列表中找到卡片:', found);
          setSelectedCard(found);
          // 同时获取详情以更新输入参数
          fetchCardDetail(found.cardId);
        } else {
          // 2. 如果列表中没有，主动获取详情（这会同时恢复selectedCard和cardDetail）
          console.log('🔍 从API获取卡片详情:', value.selectedCardId);
          fetchCardDetail(value.selectedCardId);
        }
      }
    }, [value?.selectedCardId, selectedCard, cards, fetchCardDetail]);

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
          hasAttemptedLoad={hasAttemptedLoad}
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
