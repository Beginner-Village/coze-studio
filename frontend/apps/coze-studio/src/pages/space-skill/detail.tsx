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

import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import React, { useState, useEffect } from 'react';

import { skill } from '@coze-studio/api-schema';
import { IconCozArrowLeft } from '@coze-arch/coze-design/icons';
import { Button, Input, Spin, Typography } from '@coze-arch/coze-design';

import { SkillPromptEditor } from './SkillPromptEditor';
import { useSpaceResources } from './hooks/use-space-resources';

const { Text } = Typography;

// eslint-disable-next-line @coze-arch/max-line-per-function -- full-page form with multiple fields and prompt editor
const SpaceSkillDetail: React.FC = () => {
  const { space_id, page_type } = useParams<{
    space_id: string;
    page_type: string;
  }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  const skillId = searchParams.get('skill_id') || '';
  const isEditing = page_type === 'edit';

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [prompt, setPrompt] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [loadingSkill, setLoadingSkill] = useState(false);

  const {
    resources,
    loading: resourcesLoading,
    fetchResources,
  } = useSpaceResources(space_id || '');

  useEffect(() => {
    if (isEditing && skillId && space_id) {
      setLoadingSkill(true);
      skill
        .GetSkill({ skill_id: skillId, space_id })
        .then(response => {
          if (response.code === 0 && response.data?.skill_info) {
            const info = response.data.skill_info;
            setName(info.name || '');
            setDescription(info.description || '');
            setPrompt(info.prompt || '');
          }
        })
        .catch(error => {
          console.error('加载技能失败:', error);
        })
        .finally(() => {
          setLoadingSkill(false);
        });
    }
  }, [isEditing, skillId, space_id]);

  const handleSave = async () => {
    if (!name.trim() || isSubmitting || !space_id) {
      return;
    }
    setIsSubmitting(true);
    try {
      if (isEditing) {
        await skill.UpdateSkill({
          skill_id: skillId,
          space_id,
          name,
          description: description || undefined,
          prompt: prompt || undefined,
        });
      } else {
        await skill.CreateSkill({
          space_id,
          name,
          description: description || undefined,
          prompt: prompt || undefined,
        });
      }
      navigate(-1);
    } catch (error) {
      console.error('保存失败:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loadingSkill) {
    return (
      <div className="flex justify-center items-center h-full">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* 顶部栏 */}
      <div className="flex items-center justify-between px-[24px] py-[16px] border-b coz-stroke-secondary">
        <div className="flex items-center gap-[12px]">
          <Button
            icon={<IconCozArrowLeft />}
            theme="borderless"
            onClick={() => navigate(-1)}
          />
          <h1 className="text-[18px] font-medium coz-fg-primary">
            {isEditing ? '编辑技能' : '创建技能'}
          </h1>
        </div>
        <Button
          type="primary"
          onClick={handleSave}
          disabled={!name.trim()}
          loading={isSubmitting}
        >
          保存
        </Button>
      </div>

      {/* 表单内容 */}
      <div className="flex-1 overflow-y-auto px-[24px] py-[20px]">
        <div className="max-w-[720px] mx-auto flex flex-col gap-[20px]">
          {/* 技能名称 */}
          <div>
            <div className="flex items-center mb-[6px]">
              <Text className="text-[13px] font-medium coz-fg-primary">
                技能名称
              </Text>
              <span className="text-red-500 ml-[2px]">*</span>
            </div>
            <Input
              value={name}
              onChange={val => setName(val)}
              placeholder="例如：客户退款处理、数据分析报告"
              maxLength={50}
            />
          </div>

          {/* 技能描述 */}
          <div>
            <div className="flex items-center mb-[6px]">
              <Text className="text-[13px] font-medium coz-fg-primary">
                简短描述
              </Text>
            </div>
            <Input
              value={description}
              onChange={val => setDescription(val)}
              placeholder="简要描述技能用途，将注入 Agent 的 system prompt 中"
            />
            <Text className="text-[11px] coz-fg-tertiary mt-[4px] block">
              此描述会注入到 Agent 的系统提示词中（约 50 tokens），Agent
              根据描述判断是否需要加载此技能
            </Text>
          </div>

          {/* 完整指令（Prompt） */}
          <div>
            <div className="flex items-center justify-between mb-[6px]">
              <Text className="text-[13px] font-medium coz-fg-primary">
                完整指令
              </Text>
              <Text className="text-[11px] coz-fg-tertiary">
                输入{' '}
                <code className="px-[4px] py-[1px] rounded bg-gray-100 text-[11px]">
                  {'{'}
                </code>{' '}
                可引用工作流、插件、知识库
              </Text>
            </div>
            <SkillPromptEditor
              value={prompt}
              onChange={setPrompt}
              resources={resources}
              resourcesLoading={resourcesLoading}
              onRequestResources={fetchResources}
              placeholder={
                '编写技能的完整指令，Agent 触发该技能时会按需加载这些指令。\n\n' +
                '输入 { 可引用资源：\n' +
                '  工作流 → {workflow:退款流程|id:123}\n' +
                '  插件   → {plugin:CRM系统|id:456}\n' +
                '  知识库 → {knowledge:FAQ文档|id:789}\n\n' +
                '示例：\n' +
                '1. 首先查询 {knowledge:退款政策|id:xxx} 了解退款规则\n' +
                '2. 调用 {plugin:订单查询|id:xxx} 获取订单详情\n' +
                '3. 执行 {workflow:退款处理|id:xxx} 完成退款'
              }
            />
            <Text className="text-[11px] coz-fg-tertiary mt-[4px] block">
              完整指令仅在 Agent 调用 read_skill 时按需加载，不会占用常驻 token
            </Text>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SpaceSkillDetail;
