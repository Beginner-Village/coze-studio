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

import React, { useState, useCallback, useMemo, useEffect } from 'react';

import { type InputValueVO } from '@coze-workflow/base';
import { I18n } from '@coze-arch/i18n';
import { AutoComplete, Spin, message } from '@coze-arch/bot-semi';
import type {
  CardParam,
  CardDetail,
} from '@coze-arch/api-schema/idl/workflow/workflow';

import { Section, useField, withField, useForm } from '@/form';

import type { CardItem } from '../types';
import { INPUT_PATH, ANSWER_CONTENT_PATH } from '../constants';
import { fetchCardList, fetchCardDetail } from '../api';

interface CardSelectorCompProps {
  title?: string;
  tooltip?: string;
  sassWorkspaceId?: string;
}

function CardSelectorComp({
  title,
  tooltip,
  sassWorkspaceId,
}: CardSelectorCompProps) {
  const { value, onChange, readonly, name } = useField<CardItem | undefined>();
  const [loading, setLoading] = useState(false);
  const [cardList, setCardList] = useState<CardItem[]>([]);
  const [searchValue, setSearchValue] = useState('');
  const form = useForm();

  const JSON_INDENT = 2;

  // 将卡片参数转换为InputValueVO结构
  const convertParamsToInputValues = useCallback(
    (paramList: CardParam[]): InputValueVO[] => {
      if (!paramList || paramList.length === 0) {
        return [];
      }
      return paramList.map(param => ({
        name: param.paramName,
      }));
    },
    [],
  );

  // 生成输出模板
  const generateAnswerContent = useCallback(
    (cardDetail: CardDetail): string => {
      const dataResponse: Record<string, string> = {};

      // 从paramList中提取参数名
      if (cardDetail.paramList) {
        cardDetail.paramList.forEach((param: CardParam) => {
          dataResponse[param.paramName] = `{{${param.paramName}}}`;
        });
      }

      const template = {
        contentList: [
          {
            displayResponseType: 'TEMPLATE',
            rawContent: {},
            templateId: cardDetail.code,
            templateName: cardDetail.cardName,
            kvMap: {},
            dataResponse,
          },
        ],
      };

      return JSON.stringify(template, null, JSON_INDENT);
    },
    [],
  );

  // 获取卡片列表
  const fetchCards = useCallback(
    async (search = '') => {
      console.log(
        '[CardSelectorComp] fetchCards 被调用，search:',
        search,
        'loading:',
        loading,
        'sassWorkspaceId:',
        sassWorkspaceId,
      );
      if (loading) {
        console.log('[CardSelectorComp] 已在加载中，跳过请求');
        return;
      }

      if (!sassWorkspaceId) {
        console.warn(
          '[CardSelectorComp] sassWorkspaceId 未设置，无法获取卡片列表',
        );
        return;
      }

      setLoading(true);
      try {
        console.log(
          '[CardSelectorComp] 开始调用 fetchCardList，sassWorkspaceId:',
          sassWorkspaceId,
        );
        const response = await fetchCardList({
          sassWorkspaceId,
          pageNo: 1,
          pageSize: 200,
          searchValue: search,
        });

        console.log('[CardSelectorComp] fetchCardList 响应:', response);
        setCardList(response.cardList || []);
      } catch (error) {
        console.error('[CardSelectorComp] 获取卡片列表出错:', error);
        message.error('获取卡片列表失败，请稍后重试');
        setCardList([]);
      } finally {
        setLoading(false);
      }
    },
    [sassWorkspaceId, loading],
  );

  // 处理搜索
  const handleSearch = useCallback(
    (searchText: string) => {
      setSearchValue(searchText);
      if (searchText.trim()) {
        fetchCards(searchText.trim());
      } else {
        // 如果搜索为空，获取默认列表
        fetchCards();
      }
    },
    [fetchCards],
  );

  // 处理焦点事件，首次加载数据
  const handleFocus = useCallback(() => {
    if (cardList.length === 0 && !loading) {
      fetchCards();
    }
  }, [cardList.length, loading, fetchCards]);

  // 在组件挂载时预取卡片列表，确保无论是否触发焦点事件都会请求
  useEffect(() => {
    console.log(
      '[CardSelectorComp] useEffect 挂载，cardList.length:',
      cardList.length,
      'loading:',
      loading,
    );
    // 若已存在列表则不重复请求
    if (cardList.length === 0 && !loading) {
      console.log('[CardSelectorComp] 首次挂载，开始预取卡片列表');
      void fetchCards();
    }
    // 仅在首次挂载时尝试预取
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 处理选择
  const handleSelect = useCallback(
    async (selectedValue: string) => {
      console.log(
        '[CardSelectorComp] handleSelect 被调用，selectedValue:',
        selectedValue,
      );
      console.log('[CardSelectorComp] cardList 长度:', cardList.length);
      console.log('[CardSelectorComp] sassWorkspaceId:', sassWorkspaceId);

      const selectedCard = cardList.find(card => card.cardId === selectedValue);
      console.log('[CardSelectorComp] 找到的卡片:', selectedCard);

      if (!selectedCard) {
        console.warn('[CardSelectorComp] 未找到匹配的卡片');
        return;
      }

      onChange(selectedCard);
      setSearchValue(`${selectedCard.cardName} (${selectedCard.code})`);

      if (!sassWorkspaceId) {
        console.warn(
          '[CardSelectorComp] sassWorkspaceId 为空，无法获取卡片详情',
        );
        message.warning('无法获取卡片详情：工作空间ID未设置');
        return;
      }

      try {
        // 获取卡片详情
        console.log(
          '[CardSelectorComp] 开始获取卡片详情，cardId:',
          selectedCard.cardId,
          'sassWorkspaceId:',
          sassWorkspaceId,
        );
        const { cardDetail } = await fetchCardDetail({
          cardId: selectedCard.cardId,
          sassWorkspaceId,
        });
        console.log('[CardSelectorComp] 获取到卡片详情:', cardDetail);
        console.log('[CardSelectorComp] paramList:', cardDetail.paramList);

        if (cardDetail.paramList && cardDetail.paramList.length > 0) {
          const inputParameters = convertParamsToInputValues(
            cardDetail.paramList,
          );
          console.log('[CardSelectorComp] 生成的输入参数:', inputParameters);
          form.setFieldValue(INPUT_PATH, inputParameters);

          // 自动生成输出模板
          const answerContent = generateAnswerContent(cardDetail);
          console.log('[CardSelectorComp] 生成的输出模板:', answerContent);
          form.setFieldValue(ANSWER_CONTENT_PATH, answerContent);

          message.success('已根据卡片自动生成输入变量和输出模板');
        } else {
          console.log(
            '[CardSelectorComp] paramList 为空，使用卡片信息生成空模板',
          );
          // paramList为空时，设置空的输入变量，但使用选中卡片的信息生成输出模板
          form.setFieldValue(INPUT_PATH, []);

          // 使用选中卡片的 code 和 cardName 生成模板，dataResponse 为空
          const emptyTemplate = JSON.stringify(
            {
              contentList: [
                {
                  displayResponseType: 'TEMPLATE',
                  rawContent: {},
                  templateId: cardDetail.code || selectedCard.code,
                  templateName: cardDetail.cardName || selectedCard.cardName,
                  kvMap: {},
                  dataResponse: {},
                },
              ],
            },
            null,
            JSON_INDENT,
          );
          form.setFieldValue(ANSWER_CONTENT_PATH, emptyTemplate);

          message.success('已根据卡片生成输出模板（该卡片无输入参数）');
        }
      } catch (error) {
        console.error('[CardSelectorComp] 获取卡片详情失败:', error);
        message.error('获取卡片详情失败，请稍后重试');
      }
    },
    [
      cardList,
      onChange,
      sassWorkspaceId,
      form,
      convertParamsToInputValues,
      generateAnswerContent,
    ],
  );

  // 处理输入变化
  const handleChange = useCallback(
    (inputValue: string) => {
      setSearchValue(inputValue);
      // 如果清空了输入，清空选择
      if (!inputValue) {
        onChange(undefined);
      }
    },
    [onChange],
  );

  // 生成选项数据
  const options = useMemo(
    () =>
      cardList.map(card => ({
        value: card.cardId,
        label: `${card.cardName} (${card.code})`,
        card,
      })),
    [cardList],
  );

  // 当前显示的值
  const displayValue = useMemo(() => {
    if (value) {
      return `${value.cardName} (${value.code})`;
    }
    return searchValue;
  }, [value, searchValue]);

  return (
    <Section title={title} tooltip={tooltip}>
      <AutoComplete
        name={name}
        value={displayValue}
        onSearch={handleSearch}
        onSelect={handleSelect}
        onChange={handleChange}
        onFocus={handleFocus}
        // 兼容有些浏览器/组件库不触发 onFocus 的情况
        onClick={() => {
          if (cardList.length === 0 && !loading) {
            void fetchCards();
          }
        }}
        disabled={readonly}
        placeholder={I18n.t('请输入卡片名称或代码进行搜索')}
        style={{ width: '100%' }}
        dropdownMatchSelectWidth={true}
        maxHeight={300}
        loading={loading}
        suffix={loading ? <Spin size="small" /> : undefined}
        data={options.map(option => ({
          value: option.value,
          label: (
            <div style={{ padding: '4px 0' }}>
              <div style={{ fontWeight: 'bold', fontSize: '14px' }}>
                {option.card.cardName}
              </div>
              <div style={{ color: '#666', fontSize: '12px' }}>
                代码: {option.card.code}
              </div>
            </div>
          ),
        }))}
        emptyContent={
          <div style={{ padding: '20px', textAlign: 'center', color: '#999' }}>
            {loading
              ? '加载中...'
              : cardList.length === 0 && searchValue
                ? '未找到匹配的卡片'
                : '请输入关键词搜索卡片'}
          </div>
        }
      />
      {value ? (
        <div
          style={{
            marginTop: '8px',
            padding: '8px',
            backgroundColor: '#f8f9fa',
            borderRadius: '4px',
          }}
        >
          <div style={{ fontSize: '12px', color: '#666' }}>已选择卡片:</div>
          <div style={{ fontSize: '14px', fontWeight: 'bold' }}>
            {value.cardName}
          </div>
          <div style={{ fontSize: '12px', color: '#666' }}>
            代码: {value.code}
          </div>
          <div style={{ fontSize: '12px', color: '#666' }}>
            ID: {value.cardId}
          </div>
        </div>
      ) : null}
    </Section>
  );
}

export const CardSelectorField = withField(CardSelectorComp);
