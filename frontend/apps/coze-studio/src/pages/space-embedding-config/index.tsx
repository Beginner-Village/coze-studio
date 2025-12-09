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

import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import {
  Layout,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  Typography,
  Popconfirm,
  Checkbox,
} from '@coze-arch/coze-design';
import { IconCozPlus } from '@coze-arch/coze-design/icons';
import { space_embedding } from '@coze-studio/api-schema';

const { Title } = Typography;

// Types
type SpaceEmbeddingConfig = space_embedding.SpaceEmbeddingConfig;
type EmbeddingType = space_embedding.EmbeddingType;

const EMBEDDING_TYPES = [
  { value: 'openai', label: 'OpenAI', description: 'OpenAI Embeddings API' },
  { value: 'ark', label: 'ARK', description: '火山引擎 ARK Embedding' },
  { value: 'ollama', label: 'Ollama', description: '本地 Ollama 服务' },
  { value: 'http', label: 'HTTP', description: '自定义 HTTP 接口' },
];

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();

  const [embeddings, setEmbeddings] = useState<SpaceEmbeddingConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingEmbedding, setEditingEmbedding] =
    useState<SpaceEmbeddingConfig | null>(null);
  const [selectedType, setSelectedType] = useState<EmbeddingType>('openai');

  // Fetch embeddings list
  const fetchEmbeddings = useCallback(async () => {
    if (!spaceId) return;

    try {
      setLoading(true);
      const response = await space_embedding.ListSpaceEmbeddings({
        space_id: spaceId,
      });

      if (response.code === 0 && response.data) {
        setEmbeddings(response.data);
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        const data = error.data || error.response?.data?.data;
        if (data) {
          setEmbeddings(data);
        }
      }
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  useEffect(() => {
    fetchEmbeddings();
  }, [fetchEmbeddings]);

  // Delete embedding
  const handleDelete = async (embeddingId: string) => {
    if (!spaceId) return;

    try {
      const response = await space_embedding.DeleteSpaceEmbedding({
        space_id: spaceId,
        embedding_id: embeddingId,
      });

      if (response.code === 0) {
        fetchEmbeddings();
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        fetchEmbeddings();
      }
    }
  };

  // Set as default
  const handleSetDefault = async (embeddingId: string) => {
    if (!spaceId) return;

    try {
      const response = await space_embedding.SetDefaultSpaceEmbedding({
        space_id: spaceId,
        embedding_id: embeddingId,
      });

      if (response.code === 0) {
        fetchEmbeddings();
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        fetchEmbeddings();
      }
    }
  };

  // Toggle enable/disable
  const handleToggleStatus = async (embedding: SpaceEmbeddingConfig) => {
    if (!spaceId) return;

    try {
      const api =
        embedding.status === 1
          ? space_embedding.DisableSpaceEmbedding
          : space_embedding.EnableSpaceEmbedding;

      const response = await api({
        space_id: spaceId,
        embedding_id: embedding.id,
      });

      if (response.code === 0) {
        fetchEmbeddings();
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        fetchEmbeddings();
      }
    }
  };

  // Open modal for editing
  const openEditModal = (embedding: SpaceEmbeddingConfig) => {
    setEditingEmbedding(embedding);
    setSelectedType(embedding.embedding_type);
    setShowAddModal(true);
  };

  // Close modal
  const closeModal = () => {
    setShowAddModal(false);
    setEditingEmbedding(null);
    setSelectedType('openai');
  };

  // Render config fields based on type
  const renderConfigFields = () => {
    switch (selectedType) {
      case 'openai':
        return (
          <>
            <Form.Input
              field="base_url"
              label="Base URL"
              placeholder="https://api.openai.com/v1"
              initValue={editingEmbedding?.config?.openai_config?.base_url}
            />
            <Form.Input
              field="api_key"
              label="API Key"
              type="password"
              rules={[{ required: !editingEmbedding, message: '请输入 API Key' }]}
              placeholder="sk-..."
            />
            <Form.Input
              field="model"
              label="Model"
              rules={[{ required: true, message: '请输入模型名称' }]}
              placeholder="text-embedding-3-small"
              initValue={editingEmbedding?.config?.openai_config?.model}
            />
            <Form.Input
              field="dims"
              label="Dimensions"
              rules={[{ required: true, message: '请输入向量维度' }]}
              placeholder="1536"
              initValue={editingEmbedding?.config?.openai_config?.dims?.toString()}
            />
            <Form.Checkbox field="by_azure" initValue={editingEmbedding?.config?.openai_config?.by_azure}>
              Use Azure OpenAI
            </Form.Checkbox>
          </>
        );

      case 'ark':
        return (
          <>
            <Form.Input
              field="base_url"
              label="Base URL"
              placeholder="https://ark.cn-beijing.volces.com/api/v3"
              initValue={editingEmbedding?.config?.ark_config?.base_url}
            />
            <Form.Input
              field="api_key"
              label="API Key"
              type="password"
              rules={[{ required: !editingEmbedding, message: '请输入 API Key' }]}
            />
            <Form.Input
              field="model"
              label="Model (Endpoint ID)"
              rules={[{ required: true, message: '请输入模型端点ID' }]}
              placeholder="ep-xxx"
              initValue={editingEmbedding?.config?.ark_config?.model}
            />
            <Form.Input
              field="dims"
              label="Dimensions"
              rules={[{ required: true, message: '请输入向量维度' }]}
              placeholder="2048"
              initValue={editingEmbedding?.config?.ark_config?.dims?.toString()}
            />
            <Form.Select
              field="api_type"
              label="API Type"
              initValue={editingEmbedding?.config?.ark_config?.api_type || 'text'}
            >
              <Select.Option value="text">Text</Select.Option>
              <Select.Option value="multimodal">Multimodal</Select.Option>
            </Form.Select>
          </>
        );

      case 'ollama':
        return (
          <>
            <Form.Input
              field="base_url"
              label="Base URL"
              rules={[{ required: true, message: '请输入 Ollama 服务地址' }]}
              placeholder="http://localhost:11434"
              initValue={editingEmbedding?.config?.ollama_config?.base_url}
            />
            <Form.Input
              field="model"
              label="Model"
              rules={[{ required: true, message: '请输入模型名称' }]}
              placeholder="nomic-embed-text"
              initValue={editingEmbedding?.config?.ollama_config?.model}
            />
            <Form.Input
              field="dims"
              label="Dimensions"
              rules={[{ required: true, message: '请输入向量维度' }]}
              placeholder="768"
              initValue={editingEmbedding?.config?.ollama_config?.dims?.toString()}
            />
          </>
        );

      case 'http':
        return (
          <>
            <Form.Input
              field="addr"
              label="HTTP Address"
              rules={[{ required: true, message: '请输入 HTTP 地址' }]}
              placeholder="http://localhost:8080/embed"
              initValue={editingEmbedding?.config?.http_config?.addr}
            />
            <Form.Input
              field="dims"
              label="Dimensions"
              rules={[{ required: true, message: '请输入向量维度' }]}
              placeholder="768"
              initValue={editingEmbedding?.config?.http_config?.dims?.toString()}
            />
          </>
        );

      default:
        return null;
    }
  };

  // Handle form submit
  const handleSubmit = async (values: any) => {
    if (!spaceId) return;

    const config: Record<string, any> = {
      dims: parseInt(values.dims) || 0,
    };

    // Add type-specific config
    if (selectedType === 'openai') {
      config.base_url = values.base_url;
      config.api_key = values.api_key;
      config.model = values.model;
      config.by_azure = values.by_azure || false;
    } else if (selectedType === 'ark') {
      config.base_url = values.base_url;
      config.api_key = values.api_key;
      config.model = values.model;
      config.api_type = values.api_type || 'text';
    } else if (selectedType === 'ollama') {
      config.base_url = values.base_url;
      config.model = values.model;
    } else if (selectedType === 'http') {
      config.addr = values.addr;
    }

    try {
      if (editingEmbedding) {
        // Update
        const response = await space_embedding.UpdateSpaceEmbedding({
          space_id: spaceId,
          embedding_id: editingEmbedding.id,
          name: values.name,
          description: values.description,
          type: selectedType,
          config,
          max_batch_size: parseInt(values.max_batch_size) || 100,
        });

        if (response.code === 0) {
          closeModal();
          fetchEmbeddings();
        }
      } else {
        // Create
        const response = await space_embedding.CreateSpaceEmbedding({
          space_id: spaceId,
          name: values.name,
          description: values.description,
          type: selectedType,
          config,
          max_batch_size: parseInt(values.max_batch_size) || 100,
          set_as_default: values.set_as_default || false,
        });

        if (response.code === 0) {
          closeModal();
          fetchEmbeddings();
        }
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        closeModal();
        fetchEmbeddings();
      }
    }
  };

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="w-full flex items-center justify-between">
          <Title heading={4}>Embedding 配置</Title>
          <Button
            type="primary"
            icon={<IconCozPlus />}
            onClick={() => {
              setEditingEmbedding(null);
              setSelectedType('openai');
              setShowAddModal(true);
            }}
          >
            添加配置
          </Button>
        </div>
      </Layout.Header>

      <Layout.Content>
        {loading ? (
          <div className="py-16 text-center text-gray-500">加载中...</div>
        ) : embeddings.length === 0 ? (
          <div className="py-16 text-center text-gray-500">
            暂无 Embedding 配置，请点击"添加配置"按钮创建
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {embeddings.map(embedding => (
              <div
                key={embedding.id}
                className="bg-white border border-gray-200 rounded-lg shadow-sm p-4 flex flex-col gap-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-base font-medium text-gray-900">
                      {embedding.name}
                    </div>
                    <div className="text-xs text-gray-500 mt-1">
                      ID：{embedding.id}
                    </div>
                  </div>
                  <Space size="small">
                    {embedding.is_default && (
                      <Tag color="green">默认</Tag>
                    )}
                    <Tag color={embedding.status === 1 ? 'blue' : 'gray'}>
                      {embedding.status === 1 ? '启用' : '禁用'}
                    </Tag>
                    <Tag color="purple">
                      {embedding.embedding_type.toUpperCase()}
                    </Tag>
                  </Space>
                </div>

                <div className="text-sm text-gray-600">
                  {embedding.description || '暂无描述'}
                </div>

                <div className="text-xs text-gray-500 space-y-1">
                  <div>
                    <span className="font-medium text-gray-600">维度：</span>
                    {embedding.dimensions}
                  </div>
                  <div>
                    <span className="font-medium text-gray-600">创建：</span>
                    {new Date(embedding.created_at).toLocaleString()}
                  </div>
                </div>

                <div className="flex items-center justify-end mt-auto pt-2 border-t">
                  <Space size="small">
                    <Button
                      size="small"
                      onClick={() => handleToggleStatus(embedding)}
                    >
                      {embedding.status === 1 ? '禁用' : '启用'}
                    </Button>
                    {!embedding.is_default && embedding.status === 1 && (
                      <Button
                        size="small"
                        onClick={() => handleSetDefault(embedding.id)}
                      >
                        设为默认
                      </Button>
                    )}
                    <Button size="small" onClick={() => openEditModal(embedding)}>
                      编辑
                    </Button>
                    <Popconfirm
                      title="确认删除"
                      content="删除后不可恢复，确定要删除吗？"
                      okText="删除"
                      cancelText="取消"
                      onConfirm={() => handleDelete(embedding.id)}
                      position="topRight"
                    >
                      <Button size="small" type="danger">
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                </div>
              </div>
            ))}
          </div>
        )}
      </Layout.Content>

      {/* Add/Edit Modal */}
      <Modal
        title={editingEmbedding ? '编辑 Embedding 配置' : '添加 Embedding 配置'}
        visible={showAddModal}
        onCancel={closeModal}
        footer={null}
        style={{ width: 560 }}
      >
        <Form onSubmit={handleSubmit}>
          <Form.Input
            field="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
            placeholder="My Embedding"
            initValue={editingEmbedding?.name}
          />
          <Form.Input
            field="description"
            label="描述"
            placeholder="配置描述..."
            initValue={editingEmbedding?.description}
          />
          <Form.Select
            field="type"
            label="类型"
            rules={[{ required: true, message: '请选择类型' }]}
            initValue={editingEmbedding?.embedding_type || 'openai'}
            onChange={value => setSelectedType(value as EmbeddingType)}
          >
            {EMBEDDING_TYPES.map(type => (
              <Select.Option key={type.value} value={type.value}>
                {type.label} - {type.description}
              </Select.Option>
            ))}
          </Form.Select>
          <Form.Input
            field="max_batch_size"
            label="最大批处理大小"
            placeholder="100"
            initValue={editingEmbedding?.config?.max_batch_size?.toString() || '100'}
          />

          <div className="border-t pt-4 mt-4">
            <div className="text-sm font-medium text-gray-700 mb-3">配置参数</div>
            {renderConfigFields()}
          </div>

          {!editingEmbedding && (
            <Form.Checkbox field="set_as_default" className="mt-4">
              设为默认配置
            </Form.Checkbox>
          )}

          <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
            <Button onClick={closeModal}>取消</Button>
            <Button htmlType="submit" type="primary">
              {editingEmbedding ? '保存' : '创建'}
            </Button>
          </div>
        </Form>
      </Modal>
    </Layout>
  );
};

export { Page as Component };
export default Page;
