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

/* eslint-disable max-lines -- AgentSkillArea owns create, select, and attached skill list modals in this package. */
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
  Input,
  TextArea,
  Toast,
} from '@coze-arch/coze-design';
import { useSpaceStore } from '@coze-arch/bot-studio-store';
import {
  ToolContentBlock,
  AddButton,
  ToolItemActionDelete,
} from '@coze-agent-ide/tool';

import styles from './index.module.less';

const { Text } = Typography;

interface AgentSkillAreaProps {
  toolKey?: string;
  title?: string;
}

const SkillListItem: React.FC<{
  skill: SkillInfo;
  isAdded: boolean;
  onAdd: () => void;
  onRemove: () => void;
}> = ({ skill: skillItem, isAdded, onAdd, onRemove }) => (
  <div className={styles.skillListItem}>
    <div className={styles.skillAvatar}>
      {(skillItem.name?.trim()?.slice(0, 1) || 'S').toUpperCase()}
    </div>
    <div className={styles.skillInfo}>
      <div className={styles.skillNameRow}>
        <Text ellipsis={{ showTooltip: true }} className={styles.skillName}>
          {skillItem.name}
        </Text>
        {isAdded ? (
          <IconCozCheckMarkCircleFill className={styles.skillCheck} />
        ) : null}
      </div>
      {skillItem.description ? (
        <Text ellipsis={{ showTooltip: true }} className={styles.skillDesc}>
          {skillItem.description}
        </Text>
      ) : null}
      <div className={styles.skillDate}>
        {skillItem.created_at
          ? `创建于 ${new Date(skillItem.created_at).toLocaleDateString('zh-CN')}`
          : ''}
      </div>
    </div>
    <div className={styles.skillAction}>
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
          theme="solid"
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
  onCreateClick: () => void;
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
  onCreateClick,
}) => (
  <Modal
    className={styles.skillSelectModal}
    title={
      <span className={styles.skillModalTitle}>
        添加技能 <span>从空间技能库中选择</span>
      </span>
    }
    visible={visible}
    onCancel={onClose}
    footer={
      <div className={styles.skillModalFooter}>
        <div className={styles.skillModalCount}>
          共 <span>{skillList.length}</span> 个可用技能
        </div>
        <div className={styles.skillModalActions}>
          <Button color="secondary" onClick={onCreateClick}>
            新建技能
          </Button>
          <Button color="primary" onClick={onClose}>
            完成
          </Button>
        </div>
      </div>
    }
    width={720}
    bodyStyle={{
      padding: 0,
      overflow: 'hidden',
    }}
  >
    <div className={styles.skillSelectBody}>
      <div className={styles.skillSearchWrap}>
        <Search
          placeholder="搜索技能..."
          value={search}
          onChange={val => onSearchChange(val as string)}
          className={styles.skillSearch}
        />
      </div>
      <div className={styles.skillFilters}>
        <button className={styles.skillFilterActive} type="button">
          全部
        </button>
        <button type="button">文档</button>
        <button type="button">数据</button>
        <button type="button">自动化</button>
      </div>
      <div className={styles.skillList}>
        {loading ? (
          <div className={styles.skillLoading}>
            <Spin />
          </div>
        ) : skillList.length === 0 ? (
          <div className={styles.skillLoading}>
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

interface ScriptFile {
  path: string;
  content: string;
}

interface BuildSkillFilesInput {
  name: string;
  description: string;
  skillMd: string;
  scripts: ScriptFile[];
}

const normalizeSkillFilePath = (path: string) => {
  const normalized = path
    .trim()
    .replace(/\\/g, '/')
    .replace(/^\/+/, '')
    .replace(/\/+/g, '/');
  if (
    !normalized ||
    normalized === 'SKILL.md' ||
    normalized.split('/').some(part => part === '..' || part === '.')
  ) {
    return '';
  }
  return normalized;
};

const buildSkillMd = (name: string, description: string, skillMd: string) => {
  const content = skillMd.trim()
    ? skillMd.replace(/\r\n/g, '\n')
    : `# ${name.trim()}\n\n用途说明...\n\n## 工作流\n1. ...`;
  if (content.trimStart().startsWith('---')) {
    return content;
  }
  return `---\nname: ${name.trim()}\ndescription: ${description.trim()}\n---\n${content}`;
};

const buildSkillFiles = ({
  name,
  description,
  skillMd,
  scripts,
}: BuildSkillFilesInput) => {
  const files: Record<string, string> = {
    'SKILL.md': buildSkillMd(name, description, skillMd),
  };
  scripts.forEach(script => {
    const path = normalizeSkillFilePath(script.path);
    if (path && script.content.trim()) {
      files[path] = script.content.replace(/\r\n/g, '\n');
    }
  });
  return files;
};

/** 新建技能弹窗:写 SKILL.md + 可选脚本,存进空间技能库 */
const CreateSkillModal: React.FC<{
  visible: boolean;
  spaceId: string;
  onClose: () => void;
  onCreated: () => void;
}> = ({ visible, spaceId, onClose, onCreated }) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [skillMd, setSkillMd] = useState('');
  const [scripts, setScripts] = useState<ScriptFile[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const reset = () => {
    setName('');
    setDescription('');
    setSkillMd('');
    setScripts([]);
  };

  const handleSubmit = async () => {
    if (!name.trim()) {
      Toast.warning('请填写技能名称');
      return;
    }
    setSubmitting(true);
    try {
      const resp = await skill.CreateSkill({
        space_id: spaceId,
        name: name.trim(),
        description: description.trim(),
        files: buildSkillFiles({ name, description, skillMd, scripts }),
        icon_uri: '',
      });
      if (resp.code === 0) {
        Toast.success('技能已创建');
        reset();
        onCreated();
        onClose();
      } else {
        Toast.error(resp.msg || '创建失败');
      }
    } catch (e) {
      Toast.error('创建失败(技能名可能已存在)');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="新建技能"
      visible={visible}
      onCancel={onClose}
      onOk={handleSubmit}
      okText="创建"
      cancelText="取消"
      okButtonProps={{ loading: submitting }}
      width={640}
    >
      <div className="flex flex-col gap-[12px] max-h-[60vh] overflow-y-auto pr-[4px]">
        <div>
          <div className="text-[13px] coz-fg-secondary mb-[4px]">
            技能名称 *(英文/数字,作为 /skills/&lt;名&gt;/ 目录名)
          </div>
          <Input
            value={name}
            onChange={setName}
            placeholder="例如 pdf-extract"
          />
        </div>
        <div>
          <div className="text-[13px] coz-fg-secondary mb-[4px]">
            一句话描述
          </div>
          <Input
            value={description}
            onChange={setDescription}
            placeholder="这个技能干什么用的"
          />
        </div>
        <div>
          <div className="text-[13px] coz-fg-secondary mb-[4px]">
            SKILL.md(技能说明:用途、工作流、脚本用法)
          </div>
          <TextArea
            value={skillMd}
            onChange={setSkillMd}
            autosize={{ minRows: 6, maxRows: 14 }}
            placeholder={
              '# 技能名\n\n用途说明...\n\n## 工作流\n1. ...\n2. 运行 run_bash: python /skills/<名>/run.py ...'
            }
          />
        </div>
        <div>
          <div className="flex items-center justify-between mb-[4px]">
            <span className="text-[13px] coz-fg-secondary">
              脚本文件(可选,落进技能文件夹供 agent 执行)
            </span>
            <Button
              size="mini"
              color="primary"
              onClick={() =>
                setScripts([...scripts, { path: '', content: '' }])
              }
            >
              + 添加脚本
            </Button>
          </div>
          {scripts.map((s, i) => (
            <div
              key={i}
              className="border coz-stroke-primary rounded-[8px] p-[8px] mb-[8px]"
            >
              <div className="flex items-center gap-[8px] mb-[6px]">
                <Input
                  size="small"
                  value={s.path}
                  onChange={v =>
                    setScripts(
                      scripts.map((x, j) => (j === i ? { ...x, path: v } : x)),
                    )
                  }
                  placeholder="文件名,如 run.py"
                />
                <Button
                  size="mini"
                  color="secondary"
                  onClick={() => setScripts(scripts.filter((_, j) => j !== i))}
                >
                  删除
                </Button>
              </div>
              <TextArea
                value={s.content}
                onChange={v =>
                  setScripts(
                    scripts.map((x, j) => (j === i ? { ...x, content: v } : x)),
                  )
                }
                autosize={{ minRows: 3, maxRows: 10 }}
                placeholder="脚本内容(Python/Shell 等)"
              />
            </div>
          ))}
        </div>
      </div>
    </Modal>
  );
};

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
  const [isCreateVisible, setIsCreateVisible] = useState(false);
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
        onCreateClick={() => setIsCreateVisible(true)}
      />

      <CreateSkillModal
        visible={isCreateVisible}
        spaceId={spaceId}
        onClose={() => setIsCreateVisible(false)}
        onCreated={loadSkillList}
      />
    </>
  );
};

export default AgentSkillArea;
