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

import React, { useState, useCallback, useRef, useEffect, useMemo } from 'react';
import { useShallow } from 'zustand/react/shallow';

import {
  Modal,
  Search,
  Spin,
  Button,
  Empty,
  Tooltip,
  Select,
} from '@coze-arch/coze-design';
import {
  IconCozCard,
  IconCozCheckMarkFill,
  IconCozSetting,
} from '@coze-arch/coze-design/icons';
import { useSpaceStore } from '@coze-arch/bot-studio-store';
import {
  ToolContentBlock,
  AddButton,
  ToolItemActionDelete,
} from '@coze-agent-ide/tool';
import { useBotSkillStore } from '@coze-studio/bot-detail-store/bot-skill';

// API 函数 - 直接调用外部卡片服务
async function fetchCardList(params: {
  sassWorkspaceId: string;
  pageNo?: number;
  pageSize?: number;
  searchValue?: string;
}): Promise<{ cardList: CardInfo[]; totalNums: string }> {
  const { sassWorkspaceId, pageNo = 1, pageSize = 30, searchValue } = params;

  const requestBody = {
    body: {
      sassWorkspaceId,
      pageNo: String(pageNo),
      pageSize: String(pageSize),
      searchValue: searchValue || '',
      cardName: '',
      cardCode: '',
      createdBy: true,
      variableValueList: [{}],
    },
  };

  const response = await fetch('/aop-web/IDC10001.do', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: '*/*',
    },
    body: JSON.stringify(requestBody),
  });

  if (!response.ok) {
    throw new Error(`HTTP Error: ${response.status}`);
  }

  const data = await response.json();

  if (data.header?.errorCode !== '0') {
    throw new Error(`API Error: ${data.header?.errorMsg}`);
  }

  const cardList: CardInfo[] = (data.body?.cardList || []).map(
    (card: { cardId?: string; cardName?: string; code?: string; cardPicUrl?: string }) => ({
      cardId: card.cardId || '',
      cardName: card.cardName || '',
      code: card.code || '',
      cardPicUrl: card.cardPicUrl,
    }),
  );

  return { cardList, totalNums: data.body?.totalNums || '0' };
}

async function fetchCardDetail(params: {
  cardId: string;
  sassWorkspaceId: string;
}): Promise<{ cardDetail: CardInfo }> {
  const { cardId, sassWorkspaceId } = params;

  const requestBody = {
    body: {
      cardId,
      sassWorkspaceId,
    },
  };

  const response = await fetch('/aop-web/IDC10025.do', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: '*/*',
    },
    body: JSON.stringify(requestBody),
  });

  if (!response.ok) {
    throw new Error(`HTTP Error: ${response.status}`);
  }

  const data = await response.json();

  if (data.header?.errorCode !== '0') {
    throw new Error(`API Error: ${data.header?.errorMsg}`);
  }

  const body = data.body || {};
  const cardDetail: CardInfo = {
    cardId: body.cardId || '',
    cardName: body.cardName || '',
    code: body.code || '',
    cardPicUrl: body.cardPicUrl,
    paramList: body.paramList?.map(
      (param: { paramName?: string; paramType?: string; isRequired?: string; paramDesc?: string; children?: unknown[] }) => ({
        paramName: param.paramName || '',
        paramType: param.paramType || 'string',
        required: param.isRequired === '1',
        desc: param.paramDesc,
        children: param.children,
      }),
    ),
  };

  return { cardDetail };
}

// 卡片信息接口
interface CardInfo {
  cardId: string;
  cardName: string;
  code: string;
  cardPicUrl?: string;
  paramList?: Array<{
    paramName: string;
    paramType: string;
    required: boolean;
    desc?: string;
    children?: unknown[];
  }>;
}

// 参数映射接口
interface ParamMapping {
  paramName: string;
  variableName: string;
}

// 绑定的卡片信息（包含参数映射）
interface BoundCardInfo extends CardInfo {
  paramMapping?: ParamMapping[];
}

export const CardBindingArea: React.FC = () => {
  // 连接到全局 store
  const { boundCards, updateBoundCards, variables } = useBotSkillStore(
    useShallow(state => ({
      boundCards: state.boundCards,
      updateBoundCards: state.updateBoundCards,
      variables: state.variables,
    })),
  );

  // 本地状态
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [cardList, setCardList] = useState<CardInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [search, setSearch] = useState('');
  const [tempSelectedIds, setTempSelectedIds] = useState<Set<string>>(
    new Set(),
  );
  const [mappingModalVisible, setMappingModalVisible] = useState(false);
  const [editingCard, setEditingCard] = useState<BoundCardInfo | null>(null);
  const [tempMapping, setTempMapping] = useState<Record<string, string>>({});

  const pageNoRef = useRef(1);
  const spaceId = useSpaceStore(state => state.space?.id || '');
  const sassWorkspaceId = spaceId;

  // 可用变量列表
  const availableVariables = useMemo(
    () =>
      variables
        ?.filter(v => v.key && !v.is_disabled)
        .map(v => ({
          label: v.key || '',
          value: v.key || '',
          description: v.description,
        })) || [],
    [variables],
  );

  // 加载卡片列表
  const loadCardList = useCallback(
    async (isLoadMore = false) => {
      if (!sassWorkspaceId) {
        console.warn('sassWorkspaceId is not available');
        return;
      }

      setLoading(true);
      try {
        const { cardList: newCards } = await fetchCardList({
          sassWorkspaceId,
          pageNo: pageNoRef.current,
          pageSize: 30,
          searchValue: search,
        });

        if (isLoadMore) {
          setCardList(prev => [...prev, ...newCards]);
        } else {
          setCardList(newCards);
        }

        setHasMore(newCards.length >= 30);
      } catch (error) {
        console.error('Failed to load card list:', error);
      } finally {
        setLoading(false);
      }
    },
    [sassWorkspaceId, search],
  );

  // 打开弹窗
  const handleOpenModal = useCallback(
    (e?: React.MouseEvent) => {
      e?.preventDefault();
      e?.stopPropagation();
      setIsModalVisible(true);
      setTempSelectedIds(new Set((boundCards || []).map(c => c.cardId)));
      pageNoRef.current = 1;
      loadCardList();
    },
    [boundCards, loadCardList],
  );

  // 关闭弹窗
  const handleCloseModal = useCallback(() => {
    setIsModalVisible(false);
    setSearch('');
    setCardList([]);
  }, []);

  // 搜索
  const handleSearch = useCallback(
    (value: string) => {
      setSearch(value);
      pageNoRef.current = 1;
      loadCardList();
    },
    [loadCardList],
  );

  // 加载更多
  const handleLoadMore = useCallback(() => {
    if (!loading && hasMore) {
      pageNoRef.current += 1;
      loadCardList(true);
    }
  }, [loading, hasMore, loadCardList]);

  // 切换卡片选择
  const toggleCardSelection = useCallback((card: CardInfo) => {
    setTempSelectedIds(prev => {
      const newSet = new Set(prev);
      if (newSet.has(card.cardId)) {
        newSet.delete(card.cardId);
      } else {
        newSet.add(card.cardId);
      }
      return newSet;
    });
  }, []);

  // 确认选择
  const handleConfirm = useCallback(async () => {
    const selectedCardInfos: BoundCardInfo[] = [];

    for (const cardId of tempSelectedIds) {
      // 先从已有列表中查找
      const existingCard =
        cardList.find(c => c.cardId === cardId) ||
        (boundCards || []).find(c => c.cardId === cardId);

      if (existingCard) {
        // 获取完整的卡片详情（包含参数列表）
        try {
          const { cardDetail } = await fetchCardDetail({
            cardId,
            sassWorkspaceId,
          });

          // 保留已有的参数映射
          const existingBoundCard = (boundCards || []).find(c => c.cardId === cardId);

          selectedCardInfos.push({
            cardId: existingCard.cardId,
            cardName: existingCard.cardName,
            code: existingCard.code,
            cardPicUrl: existingCard.cardPicUrl,
            paramList: cardDetail.paramList,
            paramMapping: existingBoundCard?.paramMapping || [],
          });
        } catch {
          // 如果获取详情失败，使用基本信息
          const existingBoundCard = (boundCards || []).find(c => c.cardId === cardId);
          selectedCardInfos.push({
            ...existingCard,
            paramMapping: existingBoundCard?.paramMapping || [],
          });
        }
      }
    }

    // 更新到 store
    updateBoundCards(selectedCardInfos);
    handleCloseModal();
  }, [
    tempSelectedIds,
    cardList,
    boundCards,
    sassWorkspaceId,
    updateBoundCards,
    handleCloseModal,
  ]);

  // 移除卡片
  const handleRemoveCard = useCallback(
    (cardId: string) => {
      const newCards = (boundCards || []).filter(c => c.cardId !== cardId);
      updateBoundCards(newCards);
    },
    [boundCards, updateBoundCards],
  );

  // 打开参数映射弹窗
  const handleOpenMappingModal = useCallback(
    (card: BoundCardInfo, e?: React.MouseEvent) => {
      e?.preventDefault();
      e?.stopPropagation();
      setEditingCard(card);
      // 初始化临时映射
      const mapping: Record<string, string> = {};
      card.paramMapping?.forEach(m => {
        mapping[m.paramName] = m.variableName;
      });
      setTempMapping(mapping);
      setMappingModalVisible(true);
    },
    [],
  );

  // 关闭参数映射弹窗
  const handleCloseMappingModal = useCallback(() => {
    setMappingModalVisible(false);
    setEditingCard(null);
    setTempMapping({});
  }, []);

  // 更新参数映射
  const handleMappingChange = useCallback(
    (paramName: string, variableName: string) => {
      setTempMapping(prev => ({
        ...prev,
        [paramName]: variableName,
      }));
    },
    [],
  );

  // 保存参数映射
  const handleSaveMapping = useCallback(() => {
    if (!editingCard) return;

    const newMapping: ParamMapping[] = Object.entries(tempMapping)
      .filter(([, variableName]) => variableName)
      .map(([paramName, variableName]) => ({
        paramName,
        variableName,
      }));

    const newCards = (boundCards || []).map(c =>
      c.cardId === editingCard.cardId ? { ...c, paramMapping: newMapping } : c,
    );

    updateBoundCards(newCards);
    handleCloseMappingModal();
  }, [
    editingCard,
    tempMapping,
    boundCards,
    updateBoundCards,
    handleCloseMappingModal,
  ]);

  // 滚动加载更多
  const handleScroll = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.target as HTMLDivElement;
      if (
        target.scrollHeight - target.scrollTop <= target.clientHeight + 100 &&
        !loading &&
        hasMore
      ) {
        handleLoadMore();
      }
    },
    [loading, hasMore, handleLoadMore],
  );

  // 当搜索变化时重新加载
  useEffect(() => {
    if (isModalVisible) {
      pageNoRef.current = 1;
      loadCardList();
    }
  }, [search, isModalVisible, loadCardList]);

  // 生成提示词预览（流式标签格式）
  const promptPreview = useMemo(() => {
    const cards = boundCards || [];
    if (cards.length === 0) return '';

    let prompt = '**可用卡片**\n当需要以结构化卡片形式展示内容时，请使用以下标记格式输出：\n\n';

    cards.forEach((card, index) => {
      prompt += `### ${index + 1}. ${card.cardName}\n`;
      prompt += `卡片代码: \`${card.code}\`\n`;

      if (card.paramList && card.paramList.length > 0) {
        prompt += '参数说明：\n';
        card.paramList.forEach(param => {
          const requiredMark = param.required ? ' (必填)' : '';
          prompt += `- \`${param.paramName}\` (${param.paramType}): ${param.desc || ''}${requiredMark}\n`;
        });
      }

      // Generate streaming format example
      prompt += '\n输出格式示例：\n```\n';
      prompt += `<<CARD:${card.code}:${card.cardName}>>\n`;

      if (card.paramList && card.paramList.length > 0) {
        card.paramList.forEach(param => {
          const mappedVar = card.paramMapping?.find(
            m => m.paramName === param.paramName,
          );
          const exampleValue = mappedVar
            ? `{{${mappedVar.variableName}}}`
            : `<${param.paramName}的值>`;
          prompt += `<<${param.paramName}>>${exampleValue}\n`;
        });
      }

      prompt += '<</CARD>>\n```\n\n';
    });

    // Add GROUP layout documentation if multiple cards
    if (cards.length >= 2) {
      prompt += '**卡片组布局**（多卡片并排显示）：\n';
      prompt += '```\n<<GROUP:horizontal:2>>\n';
      cards.slice(0, 2).forEach((card, i) => {
        prompt += `<<CARD:${card.code}:${card.cardName}>>\n`;
        if (card.paramList && card.paramList.length > 0) {
          card.paramList.forEach(param => {
            prompt += `<<${param.paramName}>>示例值${i + 1}\n`;
          });
        }
        prompt += '<</CARD>>\n';
      });
      prompt += '<</GROUP>>\n```\n\n';
    }

    prompt += '**⚠️ 重要提示**：\n';
    prompt += '1. 当需要使用卡片展示内容时，请严格按照上述标记格式输出\n';
    prompt += '2. GROUP 标记内**必须**包含完整的 CARD 内容，不能为空\n';
    prompt += '3. 每个 CARD 内**必须**包含所有必填字段';

    return prompt;
  }, [boundCards]);

  return (
    <>
      <ToolContentBlock
        header="卡片绑定"
        showBottomBorder
        defaultExpand={true}
        actionButton={<AddButton onClick={handleOpenModal} enableAutoHidden />}
      >
        {(boundCards || []).length > 0 ? (
          <div className="space-y-2">
            {(boundCards || []).map(card => (
              <div
                key={card.cardId}
                className="p-3 border rounded-lg hover:bg-gray-50 transition-colors group relative bg-white"
              >
                <div className="flex items-start">
                  {card.cardPicUrl ? (
                    <img
                      src={card.cardPicUrl}
                      alt={card.cardName}
                      className="w-8 h-8 rounded object-cover mr-3 flex-shrink-0"
                    />
                  ) : (
                    <div className="w-8 h-8 flex items-center justify-center bg-blue-500 rounded mr-3 flex-shrink-0">
                      <IconCozCard className="text-[16px] text-white" />
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-gray-900 truncate">
                      {card.cardName}
                    </div>
                    <div className="text-xs text-gray-400 truncate mt-1">
                      {card.code}
                    </div>
                    {card.paramMapping && card.paramMapping.length > 0 && (
                      <div className="text-xs text-blue-500 mt-1">
                        已映射 {card.paramMapping.length} 个参数
                      </div>
                    )}
                  </div>
                  <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
                    {card.paramList && card.paramList.length > 0 && (
                      <Tooltip content="配置参数映射">
                        <Button
                          type="text"
                          size="small"
                          icon={<IconCozSetting />}
                          onClick={e => handleOpenMappingModal(card, e)}
                        />
                      </Tooltip>
                    )}
                    <ToolItemActionDelete
                      onClick={() => handleRemoveCard(card.cardId)}
                      tooltips="删除卡片"
                    />
                  </div>
                </div>
              </div>
            ))}

            {/* 提示词预览 */}
            {promptPreview && (
              <div className="mt-4 p-3 bg-gray-50 rounded-lg">
                <div className="text-xs font-medium text-gray-500 mb-2">
                  生成的提示词预览：
                </div>
                <pre className="text-xs text-gray-600 whitespace-pre-wrap font-mono">
                  {promptPreview}
                </pre>
              </div>
            )}
          </div>
        ) : (
          <div className="flex flex-col items-center py-8">
            <IconCozCard className="text-[48px] text-gray-300 mb-3" />
            <span className="text-gray-400">暂未绑定卡片</span>
            <Button
              type="text"
              onClick={handleOpenModal}
              className="mt-2 text-primary"
            >
              选择卡片
            </Button>
          </div>
        )}
      </ToolContentBlock>

      {/* 卡片选择弹窗 */}
      <Modal
        title="选择卡片"
        visible={isModalVisible}
        onCancel={handleCloseModal}
        footer={
          <div className="flex justify-end gap-2">
            <Button onClick={handleCloseModal}>取消</Button>
            <Button type="primary" onClick={handleConfirm}>
              确认 ({tempSelectedIds.size})
            </Button>
          </div>
        }
        width={800}
        style={{ maxHeight: '80vh' }}
      >
        <div className="mb-4">
          <Search
            placeholder="搜索卡片"
            value={search}
            onChange={value => setSearch(value as string)}
            onSearch={handleSearch}
          />
        </div>

        <div
          className="overflow-y-auto"
          style={{ maxHeight: '400px' }}
          onScroll={handleScroll}
        >
          {loading && cardList.length === 0 ? (
            <div className="flex justify-center items-center py-8">
              <Spin />
            </div>
          ) : cardList.length === 0 ? (
            <Empty description="暂无卡片" />
          ) : (
            <div className="grid grid-cols-3 gap-4">
              {cardList.map(card => {
                const isSelected = tempSelectedIds.has(card.cardId);
                return (
                  <div
                    key={card.cardId}
                    className={`relative cursor-pointer rounded-lg border-2 p-3 transition-all ${
                      isSelected
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                    onClick={() => toggleCardSelection(card)}
                  >
                    {isSelected && (
                      <div className="absolute top-2 right-2">
                        <IconCozCheckMarkFill className="text-blue-500 text-lg" />
                      </div>
                    )}
                    {card.cardPicUrl ? (
                      <img
                        src={card.cardPicUrl}
                        alt={card.cardName}
                        className="w-full h-24 object-cover rounded mb-2"
                      />
                    ) : (
                      <div className="w-full h-24 bg-gray-100 rounded mb-2 flex items-center justify-center">
                        <IconCozCard className="text-3xl text-gray-400" />
                      </div>
                    )}
                    <Tooltip content={card.cardName}>
                      <div className="text-sm font-medium truncate">
                        {card.cardName}
                      </div>
                    </Tooltip>
                    <div className="text-xs text-gray-400 truncate mt-1">
                      {card.code}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {loading && cardList.length > 0 && (
            <div className="flex justify-center py-4">
              <Spin size="small" />
            </div>
          )}
        </div>
      </Modal>

      {/* 参数映射弹窗 */}
      <Modal
        title={`配置参数映射 - ${editingCard?.cardName || ''}`}
        visible={mappingModalVisible}
        onCancel={handleCloseMappingModal}
        footer={
          <div className="flex justify-end gap-2">
            <Button onClick={handleCloseMappingModal}>取消</Button>
            <Button type="primary" onClick={handleSaveMapping}>
              保存
            </Button>
          </div>
        }
        width={600}
      >
        <div className="mb-4 text-sm text-gray-500">
          将卡片参数映射到智能体变量，运行时会自动填充变量值。
        </div>

        {editingCard?.paramList && editingCard.paramList.length > 0 ? (
          <div className="space-y-4">
            {editingCard.paramList.map(param => (
              <div key={param.paramName} className="flex items-center gap-4">
                <div className="w-1/3">
                  <div className="font-medium text-sm">{param.paramName}</div>
                  <div className="text-xs text-gray-400">
                    {param.paramType}
                    {param.required && (
                      <span className="text-red-500 ml-1">*必填</span>
                    )}
                  </div>
                  {param.desc && (
                    <div className="text-xs text-gray-400 mt-1">
                      {param.desc}
                    </div>
                  )}
                </div>
                <div className="text-gray-400">→</div>
                <div className="flex-1">
                  <Select
                    placeholder="选择映射的变量"
                    value={tempMapping[param.paramName] || undefined}
                    onChange={value =>
                      handleMappingChange(param.paramName, value as string)
                    }
                    allowClear
                    style={{ width: '100%' }}
                  >
                    {availableVariables.map(v => (
                      <Select.Option key={v.value} value={v.value}>
                        <div>
                          <div>{v.label}</div>
                          {v.description && (
                            <div className="text-xs text-gray-400">
                              {v.description}
                            </div>
                          )}
                        </div>
                      </Select.Option>
                    ))}
                  </Select>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <Empty description="该卡片没有可配置的参数" />
        )}

        {availableVariables.length === 0 && (
          <div className="mt-4 p-3 bg-yellow-50 text-yellow-700 rounded text-sm">
            提示：当前智能体没有可用的变量。请先在"变量"区域添加变量，然后再配置参数映射。
          </div>
        )}
      </Modal>
    </>
  );
};

export default CardBindingArea;
