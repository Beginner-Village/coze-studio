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
import { useBotSkillStore } from '@coze-studio/bot-detail-store/bot-skill';
import type { SkillInfo } from '@coze-studio/api-schema/idl/skill/skill';
import { skill } from '@coze-studio/api-schema';
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
import {
  ToolContentBlock,
  AddButton,
  ToolItemActionDelete,
} from '@coze-agent-ide/tool';

const { Text } = Typography;

interface AgentSkillAreaProps {
  toolKey?: string;
  title?: string;
}

/** 技能列表项 - 匹配工作流卡片样式 */
const SkillListItem: React.FC<{
  skill: SkillInfo;
  isAdded: boolean;
  onAdd: () => void;
  onRemove: () => void;
}> = ({ skill: skillItem, isAdded, onAdd, onRemove }) => (
  <div
    className="flex items-start gap-[12px] px-[16px] py-[12px] border-b border-solid hover:bg-gray-50 transition-colors"
    style={{ borderColor: 'var(--coz-stroke-secondary, #f0f1f2)' }}
  >
    {/* 图标 */}
    <div
      className="w-[40px] h-[40px] flex items-center justify-center rounded-[8px] flex-shrink-0 mt-[2px]"
      style={{
        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
      }}
    >
      <span className="text-white text-[16px] font-bold">S</span>
    </div>
    {/* 内容 */}
    <div className="flex-1 min-w-0">
      <div className="flex items-center gap-[6px]">
        <Text
          ellipsis={{ showTooltip: true }}
          className="text-[14px] font-medium coz-fg-primary max-w-[300px]"
        >
          {skillItem.name}
        </Text>
        <IconCozCheckMarkCircleFill className="text-xs coz-fg-hglt-green flex-shrink-0" />
      </div>
      {skillItem.description ? (
        <Text
          ellipsis={{ showTooltip: true }}
          className="text-[12px] coz-fg-secondary mt-[2px] block"
        >
          {skillItem.description}
        </Text>
      ) : null}
      <div className="text-[11px] coz-fg-tertiary mt-[4px]">
        {skillItem.created_at
          ? `创建于 ${new Date(skillItem.created_at).toLocaleDateString('zh-CN')}`
          : ''}
      </div>
    </div>
    {/* 添加按钮 */}
    <div className="flex-shrink-0 mt-[2px]">
      {isAdded ? (
        <Button
          size="small"
          onClick={e => {
            e.stopPropagation();
            onRemove();
          }}
          style={{ color: 'var(--coz-fg-tertiary)' }}
        >
          已添加
        </Button>
      ) : (
        <Button
          size="small"
          type="primary"
          theme="borderless"
          onClick={e => {
            e.stopPropagation();
            onAdd();
          }}
        >
          添加
        </Button>
      )}
    </div>
  </div>
);

/** 技能选择弹窗 */
const SkillSelectModal: React.FC<{
  visible: boolean;
  loading: boolean;
  search: string;
  skillList: SkillInfo[];
  addedIds: Set<string>;
  onClose: () => void;
  onSearchChange: (val: string) => void;
  onAdd: (item: SkillInfo) => void;
  onRemove: (id: string) => void;
}> = ({
  visible,
  loading,
  search,
  skillList,
  addedIds,
  onClose,
  onSearchChange,
  onAdd,
  onRemove,
}) => (
  <Modal
    title="添加技能"
    visible={visible}
    onCancel={onClose}
    footer={null}
    width={800}
    bodyStyle={{
      padding: 0,
      maxHeight: '70vh',
      display: 'flex',
      flexDirection: 'column',
    }}
  >
    <div
      className="flex h-full"
      style={{ minHeight: '400px', maxHeight: 'calc(70vh - 56px)' }}
    >
      <div
        className="w-[200px] border-r border-solid flex flex-col flex-shrink-0 p-[12px]"
        style={{ borderColor: 'var(--coz-stroke-secondary, #f0f1f2)' }}
      >
        <Search
          placeholder="搜索"
          value={search}
          onChange={val => onSearchChange(val as string)}
          className="mb-[12px]"
        />
        <div className="text-[12px] coz-fg-tertiary">
          从空间技能库中选择技能添加到当前智能体
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex justify-center items-center py-[60px]">
            <Spin />
          </div>
        ) : skillList.length === 0 ? (
          <div className="flex justify-center items-center py-[60px]">
            <Empty
              description={
                search
                  ? '未找到匹配的技能'
                  : '暂无技能，请先在「技能管理」页面创建'
              }
            />
          </div>
        ) : (
          skillList.map(skillItem => (
            <SkillListItem
              key={skillItem.skill_id}
              skill={skillItem}
              isAdded={addedIds.has(skillItem.skill_id)}
              onAdd={() => onAdd(skillItem)}
              onRemove={() => onRemove(skillItem.skill_id)}
            />
          ))
        )}
      </div>
    </div>
  </Modal>
);

// eslint-disable-next-line @coze-arch/max-line-per-function -- skill area includes modal and card list rendering
export const AgentSkillArea: React.FC<AgentSkillAreaProps> = ({
  title = '技能',
}) => {
  const { agentSkills, updateAgentSkills } = useBotSkillStore(
    useShallow(state => ({
      agentSkills: state.agentSkills,
      updateAgentSkills: state.updateAgentSkills,
    })),
  );

  const [isModalVisible, setIsModalVisible] = useState(false);
  const [skillList, setSkillList] = useState<SkillInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  const spaceId = useSpaceStore(state => state.space?.id || '');

  const loadSkillList = useCallback(async () => {
    if (!spaceId) {
      return;
    }
    setLoading(true);
    try {
      const response = await skill.ListSkills({
        space_id: spaceId,
        page: 1,
        page_size: 100,
        keyword: search || undefined,
      });
      if (response.code === 0) {
        setSkillList(response.data?.skill_list || []);
      }
    } catch (error) {
      console.error('加载技能列表失败:', error);
    } finally {
      setLoading(false);
    }
  }, [spaceId, search]);

  const handleOpenModal = useCallback(
    (e?: React.MouseEvent) => {
      e?.preventDefault();
      e?.stopPropagation();
      setIsModalVisible(true);
      loadSkillList();
    },
    [loadSkillList],
  );

  const handleCloseModal = useCallback(() => {
    setIsModalVisible(false);
    setSearch('');
    setSkillList([]);
  }, []);

  const handleAddSkill = useCallback(
    (skillItem: SkillInfo) => {
      const newSkill = {
        skill_id: skillItem.skill_id,
        skill_name: skillItem.name,
        skill_description: skillItem.description,
      };
      updateAgentSkills([...agentSkills, newSkill]);
    },
    [agentSkills, updateAgentSkills],
  );

  const handleRemoveSkill = useCallback(
    (skillId: string) => {
      updateAgentSkills(agentSkills.filter(s => s.skill_id !== skillId));
    },
    [agentSkills, updateAgentSkills],
  );

  useEffect(() => {
    if (isModalVisible) {
      loadSkillList();
    }
  }, [search, isModalVisible, loadSkillList]);

  const addedIds = new Set(agentSkills.map(s => s.skill_id));

  return (
    <>
      <ToolContentBlock
        header={title}
        showBottomBorder
        defaultExpand={true}
        actionButton={<AddButton onClick={handleOpenModal} enableAutoHidden />}
      >
        {agentSkills.length > 0 ? (
          <div className="space-y-2">
            {agentSkills.map(skillItem => (
              <div
                key={skillItem.skill_id}
                className="p-3 border rounded-lg hover:bg-gray-50 transition-colors group relative bg-white cursor-pointer"
                onClick={() =>
                  window.open(
                    `/space/${spaceId}/skill-detail/edit?skill_id=${skillItem.skill_id}`,
                    '_blank',
                  )
                }
              >
                <div className="flex items-start">
                  <div
                    className="w-8 h-8 flex items-center justify-center rounded flex-shrink-0 mr-3"
                    style={{
                      background:
                        'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    }}
                  >
                    <span className="text-white text-xs font-bold">S</span>
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-gray-900 truncate text-[13px]">
                      {skillItem.skill_name}
                    </div>
                    <div className="text-xs text-gray-400 mt-1 line-clamp-2">
                      {skillItem.skill_description || '暂无描述'}
                    </div>
                  </div>
                  <div
                    className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={e => e.stopPropagation()}
                  >
                    <ToolItemActionDelete
                      onClick={() => handleRemoveSkill(skillItem.skill_id)}
                      tooltips="移除技能"
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </ToolContentBlock>

      <SkillSelectModal
        visible={isModalVisible}
        loading={loading}
        search={search}
        skillList={skillList}
        addedIds={addedIds}
        onClose={handleCloseModal}
        onSearchChange={setSearch}
        onAdd={handleAddSkill}
        onRemove={handleRemoveSkill}
      />
    </>
  );
};

export default AgentSkillArea;
