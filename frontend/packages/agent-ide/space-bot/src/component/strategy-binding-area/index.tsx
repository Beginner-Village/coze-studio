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

import React, { useState, useCallback, useEffect } from 'react';

import { useShallow } from 'zustand/react/shallow';
import type { StrategyBindItem } from '@coze-studio/bot-detail-store/bot-skill';
import { useBotSkillStore } from '@coze-studio/bot-detail-store/bot-skill';
import { ResType } from '@coze-arch/idl/plugin_develop';
import { I18n } from '@coze-arch/i18n';
import { IconCozCheckMarkCircleFill } from '@coze-arch/coze-design/icons';
import {
  Modal,
  Search,
  Spin,
  Button,
  Empty,
  Typography,
} from '@coze-arch/coze-design';
import { useSpaceStore } from '@coze-arch/bot-studio-store';
import { PluginDevelopApi } from '@coze-arch/bot-api';
import {
  ToolContentBlock,
  AddButton,
  ToolItemActionDelete,
} from '@coze-agent-ide/tool';

const { Text } = Typography;

interface StrategyListItemProps {
  id: string;
  name: string;
  description?: string;
  isAdded: boolean;
  onAdd: () => void;
  onRemove: () => void;
}

const StrategyListItem: React.FC<StrategyListItemProps> = ({
  id,
  name,
  description,
  isAdded,
  onAdd,
  onRemove,
}) => (
  <div
    style={{
      display: 'flex',
      alignItems: 'flex-start',
      gap: 12,
      padding: '12px',
      borderRadius: 10,
      transition: 'background 0.14s ease-out',
    }}
    onMouseEnter={e =>
      ((e.currentTarget as HTMLDivElement).style.background =
        'var(--coz-bg-secondary, rgb(240,240,247))')
    }
    onMouseLeave={e =>
      ((e.currentTarget as HTMLDivElement).style.background = 'transparent')
    }
  >
    <div
      style={{
        display: 'grid',
        placeItems: 'center',
        flexShrink: 0,
        width: 40,
        height: 40,
        marginTop: 2,
        fontSize: 16,
        fontWeight: 700,
        color: '#fff',
        background: 'linear-gradient(135deg, #f5a623 0%, #e8643c 100%)',
        borderRadius: 10,
      }}
    >
      {(name?.trim()?.slice(0, 1) || 'S').toUpperCase()}
    </div>
    <div style={{ flex: 1, minWidth: 0 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <Text
          ellipsis={{ showTooltip: true }}
          style={{
            maxWidth: 360,
            fontSize: 14,
            fontWeight: 600,
            color: 'var(--coz-fg, rgba(15,21,40,82%))',
          }}
        >
          {name || id}
        </Text>
        {isAdded ? (
          <IconCozCheckMarkCircleFill
            style={{ flexShrink: 0, width: 13, height: 13, color: '#00b23c' }}
          />
        ) : null}
      </div>
      {description ? (
        <Text
          ellipsis={{ showTooltip: true }}
          style={{
            display: 'block',
            marginTop: 2,
            fontSize: 12,
            color: 'var(--coz-fg-secondary, rgba(32,41,69,62%))',
          }}
        >
          {description}
        </Text>
      ) : null}
    </div>
    <div style={{ flexShrink: 0, marginTop: 2 }}>
      {isAdded ? (
        <Button
          size="small"
          onClick={e => {
            e.stopPropagation();
            onRemove();
          }}
          style={{ color: 'var(--coz-fg-tertiary)' }}
        >
          {I18n.t('strategy_bound', {}, '已添加')}
        </Button>
      ) : (
        <Button
          size="small"
          type="primary"
          theme="solid"
          onClick={e => {
            e.stopPropagation();
            onAdd();
          }}
        >
          {I18n.t('Add', {}, '添加')}
        </Button>
      )}
    </div>
  </div>
);

interface RemoteStrategyItem {
  res_id?: string;
  name?: string;
  description?: string;
}

interface StrategySelectModalProps {
  visible: boolean;
  loading: boolean;
  search: string;
  strategyList: RemoteStrategyItem[];
  addedIds: Set<string>;
  onClose: () => void;
  onSearchChange: (val: string) => void;
  onAdd: (item: RemoteStrategyItem) => void;
  onRemove: (id: string) => void;
}

const StrategySelectModal: React.FC<StrategySelectModalProps> = ({
  visible,
  loading,
  search,
  strategyList,
  addedIds,
  onClose,
  onSearchChange,
  onAdd,
  onRemove,
}) => (
  <Modal
    title={
      <span>
        {I18n.t('strategy_bind_modal_title', {}, '绑定策略')}{' '}
        <span
          style={{
            fontSize: 12,
            fontWeight: 400,
            color: 'var(--coz-fg-secondary, rgba(32,41,69,62%))',
          }}
        >
          {I18n.t(
            'strategy_bind_modal_subtitle',
            {},
            '从空间策略库中选择并绑定',
          )}
        </span>
      </span>
    }
    visible={visible}
    onCancel={onClose}
    footer={
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '14px 24px',
        }}
      >
        <span
          style={{
            fontSize: 12,
            color: 'var(--coz-fg-dim, rgba(55,67,106,38%))',
          }}
        >
          {I18n.t('strategy_bind_count', {}, '共')}{' '}
          <span
            style={{
              color: 'var(--coz-fg, rgba(15,21,40,82%))',
              fontWeight: 600,
            }}
          >
            {strategyList.length}
          </span>{' '}
          {I18n.t('strategy_bind_unit', {}, '个可用策略')}
        </span>
        <Button color="primary" onClick={onClose}>
          {I18n.t('Done', {}, '完成')}
        </Button>
      </div>
    }
    width={680}
    bodyStyle={{ padding: 0, overflow: 'hidden' }}
  >
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        minHeight: 380,
        maxHeight: 'calc(88vh - 138px)',
      }}
    >
      <div style={{ flexShrink: 0, padding: '0 24px 12px' }}>
        <Search
          placeholder={I18n.t(
            'strategy_bind_search_placeholder',
            {},
            '搜索策略...',
          )}
          value={search}
          onChange={val => onSearchChange(val as string)}
          style={{ width: '100%' }}
        />
      </div>
      <div
        style={{
          flex: '1 1 auto',
          minHeight: 0,
          overflowY: 'auto',
          padding: '0 16px 12px',
          scrollbarWidth: 'thin',
        }}
      >
        {loading ? (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: 240,
            }}
          >
            <Spin />
          </div>
        ) : strategyList.length === 0 ? (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              minHeight: 240,
            }}
          >
            <Empty
              description={
                search
                  ? I18n.t('strategy_bind_no_results', {}, '未找到匹配的策略')
                  : I18n.t(
                      'strategy_bind_empty',
                      {},
                      '暂无策略，请先在「策略管理」页面创建',
                    )
              }
            />
          </div>
        ) : (
          strategyList.map(item => (
            <StrategyListItem
              key={item.res_id}
              id={item.res_id ?? ''}
              name={item.name ?? ''}
              description={item.description}
              isAdded={addedIds.has(item.res_id ?? '')}
              onAdd={() => onAdd(item)}
              onRemove={() => onRemove(item.res_id ?? '')}
            />
          ))
        )}
      </div>
    </div>
  </Modal>
);

interface StrategyBindingAreaProps {
  toolKey?: string;
  title?: string;
}

// eslint-disable-next-line @coze-arch/max-line-per-function -- strategy area includes modal and card list rendering
export const StrategyBindingArea: React.FC<StrategyBindingAreaProps> = ({
  title,
}) => {
  const { strategies, updateSkillStrategies } = useBotSkillStore(
    useShallow(state => ({
      strategies: state.strategies,
      updateSkillStrategies: state.updateSkillStrategies,
    })),
  );

  const [isModalVisible, setIsModalVisible] = useState(false);
  const [strategyList, setStrategyList] = useState<RemoteStrategyItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  const spaceId = useSpaceStore(state => state.space?.id || '');

  const loadStrategyList = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    setLoading(true);
    try {
      const response = await PluginDevelopApi.LibraryResourceList({
        space_id: spaceId,
        res_type_filter: [ResType.Strategy],
        name: search || undefined,
        size: 100,
      });
      if (
        response.code === 0 ||
        response.code === undefined ||
        response.code === null
      ) {
        setStrategyList(
          (response.resource_list ?? []).map(r => ({
            res_id: r.res_id,
            name: r.name,
            description: r.description,
          })),
        );
      }
    } catch (error) {
      console.error('加载策略列表失败:', error);
    } finally {
      setLoading(false);
    }
  }, [spaceId, search]);

  const handleOpenModal = useCallback(
    (e?: React.MouseEvent) => {
      e?.preventDefault();
      e?.stopPropagation();
      setIsModalVisible(true);
      loadStrategyList();
    },
    [loadStrategyList],
  );

  const handleCloseModal = useCallback(() => {
    setIsModalVisible(false);
    setSearch('');
    setStrategyList([]);
  }, []);

  const handleAddStrategy = useCallback(
    (item: RemoteStrategyItem) => {
      const newStrategy: StrategyBindItem = {
        strategy_id: item.res_id ?? '',
        strategy_name: item.name ?? '',
        strategy_desc: item.description,
      };
      updateSkillStrategies([...strategies, newStrategy]);
    },
    [strategies, updateSkillStrategies],
  );

  const handleRemoveStrategy = useCallback(
    (id: string) => {
      updateSkillStrategies(strategies.filter(s => s.strategy_id !== id));
    },
    [strategies, updateSkillStrategies],
  );

  useEffect(() => {
    if (isModalVisible) {
      loadStrategyList();
    }
  }, [search, isModalVisible, loadStrategyList]);

  const addedIds = new Set(strategies.map(s => s.strategy_id));

  const displayTitle =
    title ?? I18n.t('navigation_workspace_library_strategy', {}, '策略');

  return (
    <>
      <ToolContentBlock
        header={displayTitle}
        showBottomBorder
        defaultExpand={true}
        actionButton={<AddButton onClick={handleOpenModal} enableAutoHidden />}
      >
        {strategies.length > 0 ? (
          <div className="space-y-2">
            {strategies.map(item => (
              <div
                key={item.strategy_id}
                className="p-3 border rounded-lg hover:bg-gray-50 transition-colors group relative bg-white cursor-pointer"
              >
                <div className="flex items-start">
                  <div
                    className="w-8 h-8 flex items-center justify-center rounded flex-shrink-0 mr-3"
                    style={{
                      background:
                        'linear-gradient(135deg, #f5a623 0%, #e8643c 100%)',
                    }}
                  >
                    <span className="text-white text-xs font-bold">
                      {(
                        item.strategy_name?.trim()?.slice(0, 1) || 'S'
                      ).toUpperCase()}
                    </span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-gray-900 truncate text-[13px]">
                      {item.strategy_name || item.strategy_id}
                    </div>
                    <div className="text-xs text-gray-400 mt-1 line-clamp-2">
                      {item.strategy_desc ||
                        I18n.t('strategy_no_desc', {}, '暂无描述')}
                    </div>
                  </div>
                  <div
                    className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={e => e.stopPropagation()}
                  >
                    <ToolItemActionDelete
                      onClick={() => handleRemoveStrategy(item.strategy_id)}
                      tooltips={I18n.t('strategy_remove', {}, '移除策略')}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </ToolContentBlock>

      <StrategySelectModal
        visible={isModalVisible}
        loading={loading}
        search={search}
        strategyList={strategyList}
        addedIds={addedIds}
        onClose={handleCloseModal}
        onSearchChange={setSearch}
        onAdd={handleAddStrategy}
        onRemove={handleRemoveStrategy}
      />
    </>
  );
};

export default StrategyBindingArea;
