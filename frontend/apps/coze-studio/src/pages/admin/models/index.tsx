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

import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Avatar,
  IconButton,
  Dropdown,
  Select,
  Search,
  Spin,
  Empty,
  Toast,
} from '@coze-arch/coze-design';
import {
  IconCozPlus,
  IconCozMore,
} from '@coze-arch/coze-design/icons';
import { admin } from '@coze-studio/api-schema';

interface PublicModel {
  id: string;
  name: string;
  description?: string;
  model_type: string;
  protocol: string;
  icon_url?: string;
  icon_uri?: string;
  status: number;
  context_length?: number;
  created_at: number;
}

export const AdminModelsPage: React.FC = () => {
  const navigate = useNavigate();
  const [models, setModels] = useState<PublicModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [typeFilter, setTypeFilter] = useState('all');
  const [searchValue, setSearchValue] = useState('');

  useEffect(() => {
    fetchModels();
  }, []);

  const fetchModels = async () => {
    setLoading(true);
    try {
      const response = await admin.ListPublicModels({});
      if (response.code === 0 && response.items) {
        setModels(response.items as PublicModel[]);
      }
    } catch (error: any) {
      // 处理API客户端特殊情况
      if (error.code === '0' || error.code === 0) {
        const responseData = error.response?.data || error;
        if (responseData.items) {
          setModels(responseData.items as PublicModel[]);
        }
      } else {
        console.error('获取公共模型列表失败:', error);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (modelId: string) => {
    if (!confirm('确定要删除这个公共模型吗？删除后所有空间将无法使用此模型。')) {
      return;
    }

    try {
      const response = await admin.DeletePublicModel({ model_id: modelId });
      if (response.code === 0) {
        Toast.success('删除成功');
        fetchModels();
      }
    } catch (error: any) {
      if (error.code === '0' || error.code === 0) {
        Toast.success('删除成功');
        fetchModels();
      } else {
        console.error('删除失败:', error);
        Toast.error('删除失败');
      }
    }
  };

  const handleToggleStatus = async (modelId: string, currentStatus: number) => {
    try {
      if (currentStatus === 1) {
        await admin.DisablePublicModel({ model_id: modelId });
      } else {
        await admin.EnablePublicModel({ model_id: modelId });
      }
      fetchModels();
    } catch (error: any) {
      if (error.code === '0' || error.code === 0) {
        fetchModels();
      } else {
        console.error('状态切换失败:', error);
        Toast.error('状态切换失败');
      }
    }
  };

  const filteredModels = models.filter(model => {
    if (typeFilter !== 'all' && model.model_type !== typeFilter) {
      return false;
    }
    if (searchValue && !model.name.toLowerCase().includes(searchValue.toLowerCase())) {
      return false;
    }
    return true;
  });

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      {/* 头部操作栏 */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <Select
            value={typeFilter}
            onChange={(val) => setTypeFilter(val as string)}
            style={{ width: 150 }}
          >
            <Select.Option value="all">全部类型</Select.Option>
            <Select.Option value="llm">LLM</Select.Option>
            <Select.Option value="embedding">Embedding</Select.Option>
            <Select.Option value="rerank">Rerank</Select.Option>
            <Select.Option value="tts">TTS</Select.Option>
          </Select>

          <Search
            placeholder="搜索模型"
            value={searchValue}
            onChange={(val) => setSearchValue(val)}
            style={{ width: 200 }}
          />
        </div>

        <Button
          type="primary"
          icon={<IconCozPlus />}
          onClick={() => navigate('/admin/models/create')}
        >
          添加公共模型
        </Button>
      </div>

      {/* 模型列表 */}
      {filteredModels.length === 0 ? (
        <Empty description="暂无公共模型" />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredModels.map((model) => (
            <ModelCard
              key={model.id}
              model={model}
              onEdit={() => navigate(`/admin/models/${model.id}/edit`)}
              onDelete={() => handleDelete(model.id)}
              onToggleStatus={() => handleToggleStatus(model.id, model.status)}
            />
          ))}
        </div>
      )}
    </div>
  );
};

interface ModelCardProps {
  model: PublicModel;
  onEdit: () => void;
  onDelete: () => void;
  onToggleStatus: () => void;
}

const ModelCard: React.FC<ModelCardProps> = ({
  model,
  onEdit,
  onDelete,
  onToggleStatus,
}) => {
  const isEnabled = model.status === 1;

  return (
    <div className="bg-white rounded-lg border p-4 hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <Avatar shape="square" style={{ width: 40, height: 40 }}>
            {model.icon_url ? (
              <img src={model.icon_url} alt={model.name} className="w-full h-full object-cover" />
            ) : (
              <span style={{ fontSize: '20px' }}>🤖</span>
            )}
          </Avatar>
          <div>
            <h3 className="font-medium">{model.name}</h3>
            <p className="text-xs text-gray-500">{model.protocol}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span
            className={`px-2 py-1 rounded text-xs font-medium ${
              isEnabled
                ? 'bg-green-100 text-green-700'
                : 'bg-gray-100 text-gray-500'
            }`}
          >
            {isEnabled ? '已启用' : '已禁用'}
          </span>

          <Dropdown
            trigger="click"
            position="bottomRight"
            render={
              <Dropdown.Menu>
                <Dropdown.Item onClick={onEdit}>编辑</Dropdown.Item>
                <Dropdown.Item onClick={onToggleStatus}>
                  {isEnabled ? '禁用' : '启用'}
                </Dropdown.Item>
                <Dropdown.Item type="danger" onClick={onDelete}>
                  删除
                </Dropdown.Item>
              </Dropdown.Menu>
            }
          >
            <IconButton icon={<IconCozMore />} />
          </Dropdown>
        </div>
      </div>

      <p className="text-sm text-gray-600 line-clamp-2 mb-3">
        {model.description || '暂无描述'}
      </p>

      <div className="flex items-center gap-4 text-xs text-gray-500">
        <span>类型: {model.model_type?.toUpperCase()}</span>
        {model.context_length && (
          <span>上下文: {Math.floor(model.context_length / 1000)}K</span>
        )}
      </div>
    </div>
  );
};

export { AdminModelsPage as Component };
export default AdminModelsPage;
