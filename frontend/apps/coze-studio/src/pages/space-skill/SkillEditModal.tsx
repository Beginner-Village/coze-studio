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

import React, { useState, useEffect } from 'react';

import type { SkillInfo } from '@coze-studio/api-schema/idl/skill/skill';
import { Modal, Button, Input, Typography } from '@coze-arch/coze-design';

import { SkillPromptEditor } from './SkillPromptEditor';
import { useSpaceResources } from './hooks/use-space-resources';

const { Text } = Typography;

interface SkillEditModalProps {
  isOpen: boolean;
  skill?: SkillInfo | null;
  spaceId: string;
  onClose: () => void;
  onSubmit: (data: {
    name: string;
    description: string;
    prompt: string;
    icon_uri?: string;
  }) => Promise<void>;
}

export const SkillEditModal: React.FC<SkillEditModalProps> = ({
  isOpen,
  skill,
  spaceId,
  onClose,
  onSubmit,
}) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [prompt, setPrompt] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const isEditing = !!skill;
  const {
    resources,
    loading: resourcesLoading,
    fetchResources,
  } = useSpaceResources(spaceId);

  useEffect(() => {
    if (skill) {
      setName(skill.name || '');
      setDescription(skill.description || '');
      setPrompt(skill.prompt || '');
    } else {
      setName('');
      setDescription('');
      setPrompt('');
    }
  }, [skill, isOpen]);

  const handleSubmit = async () => {
    if (!name.trim() || isSubmitting) {
      return;
    }
    setIsSubmitting(true);
    try {
      await onSubmit({ name, description, prompt });
      onClose();
    } catch (error) {
      console.error('保存失败:', error);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      title={isEditing ? '编辑技能' : '创建技能'}
      visible={isOpen}
      onCancel={onClose}
      width={720}
      closable
      maskClosable={false}
      footer={
        <div className="flex justify-end gap-[8px]">
          <Button onClick={onClose}>取消</Button>
          <Button
            type="primary"
            onClick={handleSubmit}
            disabled={!name.trim()}
            loading={isSubmitting}
          >
            {isEditing ? '保存' : '创建'}
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-[16px]">
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
    </Modal>
  );
};
