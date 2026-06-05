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

/* eslint-disable @coze-arch/max-line-per-function -- WorkflowCategoryPanel is a scaffold component with inline mock data; will be split when real APIs are wired */

/**
 * WorkflowCategory — 工作流文件夹分类树 + 拖拽归类脚手架
 *
 * 当前实现：纯本地状态的最小可用版本（scaffolding）。
 * TODO（后续迭代）：
 *   1. 接后端接口：GET/POST /api/workflow/categories（创建/重命名/删除分类）
 *   2. 接后端接口：PATCH /api/workflow/:id/category（归类工作流）
 *   3. 用 @dnd-kit/core 替换内置 HTML5 drag-and-drop，支持跨层嵌套拖拽
 *   4. 在 library/workspace-adapter 中挂载侧边栏树，与工作流列表联动过滤
 */

import React, { useState } from 'react';

import { Button, Input, Toast, Tree, Typography } from '@coze-arch/coze-design';

const { Title } = Typography;

export interface WorkflowCategoryNode {
  key: string;
  title: string;
  children?: WorkflowCategoryNode[];
}

interface WorkflowItem {
  id: string;
  name: string;
  categoryKey: string | null;
}

const DEFAULT_CATEGORIES: WorkflowCategoryNode[] = [
  {
    key: 'uncategorized',
    title: '未分类',
  },
  {
    key: 'production',
    title: '生产流程',
    children: [
      { key: 'production-qa', title: 'QA 流程' },
      { key: 'production-etl', title: 'ETL 流程' },
    ],
  },
  {
    key: 'testing',
    title: '测试用例',
  },
];

const MOCK_WORKFLOWS: WorkflowItem[] = [
  { id: 'wf-001', name: '客服自动回复', categoryKey: null },
  { id: 'wf-002', name: '数据处理管线', categoryKey: 'production-etl' },
  { id: 'wf-003', name: '回归测试流', categoryKey: 'testing' },
];

export const WorkflowCategoryPanel: React.FC<{ spaceId?: string }> = ({
  spaceId: _spaceId,
}) => {
  const [categories, setCategories] =
    useState<WorkflowCategoryNode[]>(DEFAULT_CATEGORIES);
  const [workflows, setWorkflows] = useState<WorkflowItem[]>(MOCK_WORKFLOWS);
  const [selectedCategoryKey, setSelectedCategoryKey] = useState<string | null>(
    null,
  );
  const [newCategoryName, setNewCategoryName] = useState('');
  const [draggingWorkflowId, setDraggingWorkflowId] = useState<string | null>(
    null,
  );

  const handleAddCategory = () => {
    const trimmed = newCategoryName.trim();
    if (!trimmed) {
      Toast.error('请输入分类名称');
      return;
    }
    const newNode: WorkflowCategoryNode = {
      key: `cat-${Date.now()}`,
      title: trimmed,
    };
    setCategories(prev => [...prev, newNode]);
    setNewCategoryName('');
    Toast.success(`已添加分类：${trimmed}`);
    // TODO: POST /api/workflow/categories { name: trimmed, space_id: spaceId }
  };

  const visibleWorkflows = selectedCategoryKey
    ? workflows.filter(w => w.categoryKey === selectedCategoryKey)
    : workflows;

  const handleDragStart = (workflowId: string) => {
    setDraggingWorkflowId(workflowId);
  };

  const handleDropOnCategory = (categoryKey: string) => {
    if (!draggingWorkflowId) {
      return;
    }
    setWorkflows(prev =>
      prev.map(w => (w.id === draggingWorkflowId ? { ...w, categoryKey } : w)),
    );
    const cat = findCategory(categories, categoryKey);
    Toast.success(`已移动到"${cat?.title || categoryKey}"`);
    setDraggingWorkflowId(null);
    // TODO: PATCH /api/workflow/:draggingWorkflowId/category { category_key: categoryKey }
  };

  return (
    <div style={{ display: 'flex', height: '100%', gap: 16 }}>
      {/* Left: category tree */}
      <div
        style={{
          width: 220,
          borderRight: '1px solid #e5e7eb',
          paddingRight: 12,
          flexShrink: 0,
        }}
      >
        <Title heading={6} style={{ marginBottom: 8 }}>
          分类
        </Title>
        <Tree
          treeData={categories}
          onSelect={keys => {
            setSelectedCategoryKey(keys.length > 0 ? String(keys[0]) : null);
          }}
          selectedKeys={selectedCategoryKey ? [selectedCategoryKey] : []}
          onDrop={info => {
            // TODO: implement nested drag-and-drop category reordering via @dnd-kit
            void info;
          }}
          draggable
          style={{ marginBottom: 12 }}
        />
        <div style={{ display: 'flex', gap: 4 }}>
          <Input
            value={newCategoryName}
            onChange={v => setNewCategoryName(String(v))}
            placeholder="新分类名"
            size="small"
          />
          <Button size="small" type="primary" onClick={handleAddCategory}>
            添加
          </Button>
        </div>
      </div>

      {/* Right: workflow list with drag handle */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        <Title heading={6} style={{ marginBottom: 8 }}>
          {selectedCategoryKey
            ? `${findCategory(categories, selectedCategoryKey)?.title || selectedCategoryKey} 下的工作流`
            : '全部工作流'}
        </Title>
        {visibleWorkflows.length === 0 ? (
          <div className="text-gray-400 text-sm py-8 text-center">
            暂无工作流
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {visibleWorkflows.map(wf => (
              <div
                key={wf.id}
                draggable
                onDragStart={() => handleDragStart(wf.id)}
                style={{
                  padding: '8px 12px',
                  border: '1px solid #e5e7eb',
                  borderRadius: 6,
                  cursor: 'grab',
                  background: '#fff',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                }}
              >
                <span>{wf.name}</span>
                <span className="text-xs text-gray-400">{wf.id}</span>
              </div>
            ))}
          </div>
        )}

        {/* Drop zones for each category */}
        <div style={{ marginTop: 24 }}>
          <div className="text-xs text-gray-400 mb-2">
            拖拽工作流到以下分类：
          </div>
          {flattenCategories(categories).map(cat => (
            <div
              key={cat.key}
              onDragOver={e => e.preventDefault()}
              onDrop={() => handleDropOnCategory(cat.key)}
              style={{
                padding: '6px 12px',
                border: '1px dashed #d1d5db',
                borderRadius: 4,
                marginBottom: 4,
                background: draggingWorkflowId ? '#f9fafb' : 'transparent',
                cursor: 'copy',
              }}
            >
              {cat.title}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

function findCategory(
  nodes: WorkflowCategoryNode[],
  key: string,
): WorkflowCategoryNode | null {
  for (const node of nodes) {
    if (node.key === key) {
      return node;
    }
    if (node.children) {
      const found = findCategory(node.children, key);
      if (found) {
        return found;
      }
    }
  }
  return null;
}

function flattenCategories(
  nodes: WorkflowCategoryNode[],
): WorkflowCategoryNode[] {
  const result: WorkflowCategoryNode[] = [];
  for (const node of nodes) {
    result.push(node);
    if (node.children) {
      result.push(...flattenCategories(node.children));
    }
  }
  return result;
}

export default WorkflowCategoryPanel;
