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

import { useParams, useNavigate } from 'react-router-dom';
import React, { useState } from 'react';

import type { SkillInfo } from '@coze-studio/api-schema/idl/skill/skill';
import {
  IconCozPlus,
  IconCozMore,
  IconCozDelete,
  IconCozEdit,
} from '@coze-arch/coze-design/icons';
import {
  Button,
  Search,
  Spin,
  Empty,
  Modal,
  Dropdown,
  IconButton,
} from '@coze-arch/coze-design';

import { useSkillManagement } from './hooks/use-skill-management';

const SkillCard: React.FC<{
  skill: SkillInfo;
  onEdit: (skill: SkillInfo) => void;
  onDelete: (skill: SkillInfo) => void;
  isHovered: boolean;
  onHover: (id: string | null) => void;
}> = ({ skill, onEdit, onDelete, isHovered, onHover }) => (
  <div
    className="flex-grow h-[158px] min-w-[280px] rounded-[6px] border-solid border-[1px] relative overflow-hidden transition duration-150 ease-out hover:shadow-[0_6px_8px_0_rgba(28,31,35,6%)] coz-stroke-primary coz-mg-card"
    onMouseEnter={() => onHover(skill.skill_id)}
    onMouseLeave={() => onHover(null)}
  >
    <div
      className="h-full w-full cursor-pointer flex flex-col gap-[8px] px-[16px] py-[16px]"
      onClick={() => onEdit(skill)}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-[12px] flex-1 min-w-0">
          <div
            className="w-[40px] h-[40px] flex items-center justify-center rounded-[8px] flex-shrink-0"
            style={{
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            }}
          >
            <span className="text-white text-[16px] font-bold">S</span>
          </div>
          <div className="flex-1 min-w-0">
            <h3
              className="text-[14px] font-medium coz-fg-primary truncate"
              title={skill.name}
            >
              {skill.name}
            </h3>
            <p
              className="text-[12px] coz-fg-secondary line-clamp-2 mt-[2px]"
              title={skill.description}
            >
              {skill.description || '暂无描述'}
            </p>
          </div>
        </div>
      </div>

      {skill.prompt ? (
        <div className="flex-1 min-h-0 overflow-hidden">
          <p
            className="text-[12px] coz-fg-tertiary line-clamp-2 font-mono"
            title={skill.prompt}
          >
            {skill.prompt}
          </p>
        </div>
      ) : null}

      <div className="flex items-center gap-[4px] text-[12px]">
        <span className="coz-fg-tertiary">创建时间</span>
        <span className="coz-fg-secondary">
          {skill.created_at
            ? new Date(skill.created_at).toLocaleDateString('zh-CN')
            : '-'}
        </span>
        {skill.updated_at ? (
          <>
            <span className="coz-fg-tertiary ml-[8px]">更新时间</span>
            <span className="coz-fg-secondary">
              {new Date(skill.updated_at).toLocaleDateString('zh-CN')}
            </span>
          </>
        ) : null}
      </div>

      {isHovered ? (
        <>
          <div
            className="absolute bottom-[16px] right-[16px] w-[100px] h-[16px]"
            style={{
              background:
                'linear-gradient(90deg, rgba(255,255,255,0) 0%, rgba(255,255,255,1) 21.38%)',
            }}
          />
          <div
            className="absolute bottom-[16px] right-[16px] flex gap-[4px]"
            onClick={e => e.stopPropagation()}
          >
            <Dropdown
              trigger="click"
              position="bottomRight"
              render={
                <Dropdown.Menu>
                  <Dropdown.Item
                    icon={<IconCozEdit />}
                    onClick={() => onEdit(skill)}
                  >
                    编辑
                  </Dropdown.Item>
                  <Dropdown.Item
                    icon={<IconCozDelete />}
                    type="danger"
                    onClick={() => onDelete(skill)}
                  >
                    删除
                  </Dropdown.Item>
                </Dropdown.Menu>
              }
            >
              <IconButton icon={<IconCozMore />} />
            </Dropdown>
          </div>
        </>
      ) : null}
    </div>
  </div>
);

const SpaceSkillPage: React.FC = () => {
  const { space_id } = useParams<{ space_id: string }>();
  const navigate = useNavigate();
  const { skillList, loading, total, keyword, setKeyword, deleteSkill } =
    useSkillManagement(space_id || '');

  const [hoveredId, setHoveredId] = useState<string | null>(null);

  const handleCreate = () => {
    navigate(`/space/${space_id}/skill-detail/create`);
  };

  const handleEdit = (skillItem: SkillInfo) => {
    navigate(
      `/space/${space_id}/skill-detail/edit?skill_id=${skillItem.skill_id}`,
    );
  };

  const handleDelete = (skillItem: SkillInfo) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除技能「${skillItem.name}」吗？删除后无法恢复。`,
      okText: '删除',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteSkill(skillItem.skill_id);
        } catch (error) {
          console.error('删除失败:', error);
        }
      },
    });
  };

  return (
    <div className="flex flex-col h-full">
      {/* 顶部标题栏 */}
      <div className="flex items-center justify-between px-[24px] py-[16px] border-b coz-stroke-secondary">
        <div>
          <h1 className="text-[18px] font-medium coz-fg-primary">技能管理</h1>
          <p className="text-[12px] coz-fg-tertiary mt-[4px]">
            场景化能力包，Agent 按需加载完整指令，节省 token 开销
          </p>
        </div>
        <Button type="primary" icon={<IconCozPlus />} onClick={handleCreate}>
          创建技能
        </Button>
      </div>

      {/* 筛选栏 */}
      <div className="flex items-center justify-between px-[24px] py-[12px] border-b coz-stroke-secondary">
        <div className="flex items-center gap-[8px]">
          <span className="text-[14px] coz-fg-secondary">
            共 {total} 个技能
          </span>
        </div>
        <Search
          showClear
          className="w-[200px]"
          placeholder="搜索技能"
          value={keyword}
          onChange={val => setKeyword(val)}
        />
      </div>

      {/* 内容区域 */}
      <div className="flex-1 overflow-y-auto px-[24px] py-[20px]">
        {loading ? (
          <div className="flex justify-center items-center py-[60px]">
            <Spin size="large" />
          </div>
        ) : skillList.length === 0 ? (
          <div className="flex justify-center items-center py-[60px]">
            <Empty
              description={
                keyword
                  ? '没有找到匹配的技能'
                  : '暂无技能，创建你的第一个技能吧'
              }
            />
          </div>
        ) : (
          <div className="grid grid-cols-3 auto-rows-min gap-[20px] [@media(min-width:1600px)]:grid-cols-4">
            {skillList.map(skillItem => (
              <SkillCard
                key={skillItem.skill_id}
                skill={skillItem}
                onEdit={handleEdit}
                onDelete={handleDelete}
                isHovered={hoveredId === skillItem.skill_id}
                onHover={setHoveredId}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default SpaceSkillPage;
