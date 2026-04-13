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

import React, { useState, useEffect, useCallback, useRef } from 'react';
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
  Toast,
} from '@coze-arch/coze-design';
import { IconCozPlus } from '@coze-arch/coze-design/icons';
import { space_rerank } from '@coze-studio/api-schema';

const { Title } = Typography;

// Types
type SpaceRerankConfig = space_rerank.SpaceRerankConfig;
type RerankType = space_rerank.RerankType;

const RERANK_TYPES = [
  { value: 'openai', label: 'OpenAI Compatible', description: '兼容 OpenAI/Jina/Cohere/vLLM 等 Rerank API' },
];

const Page: React.FC = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();

  const [reranks, setReranks] = useState<SpaceRerankConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingRerank, setEditingRerank] =
    useState<SpaceRerankConfig | null>(null);
  const [selectedType, setSelectedType] = useState<RerankType>('openai');
  const formApiRef = useRef<any>(null);

  // Fetch reranks list
  const fetchReranks = useCallback(async () => {
    if (!spaceId) return;

    try {
      setLoading(true);
      const response = await space_rerank.ListSpaceReranks({
        space_id: spaceId,
      });

      if (response.code === 0 && response.data) {
        setReranks(response.data);
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        const data = error.data || error.response?.data?.data;
        if (data) {
          setReranks(data);
        }
      }
    } finally {
      setLoading(false);
    }
  }, [spaceId]);

  useEffect(() => {
    fetchReranks();
  }, [fetchReranks]);

  // Delete rerank
  const handleDelete = async (rerankId: string) => {
    if (!spaceId) return;

    try {
      const response = await space_rerank.DeleteSpaceRerank({
        space_id: spaceId,
        rerank_id: rerankId,
      });

      if (response.code === 0) {
        fetchReranks();
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        fetchReranks();
      }
    }
  };

  // Set as default
  const handleSetDefault = async (rerankId: string) => {
    if (!spaceId) return;

    try {
      const response = await space_rerank.SetDefaultSpaceRerank({
        space_id: spaceId,
        rerank_id: rerankId,
      });

      if (response.code === 0) {
        fetchReranks();
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        fetchReranks();
      }
    }
  };

  // Toggle enable/disable
  const handleToggleStatus = async (rerank: SpaceRerankConfig) => {
    if (!spaceId) return;

    try {
      const api =
        rerank.status === 1
          ? space_rerank.DisableSpaceRerank
          : space_rerank.EnableSpaceRerank;

      const response = await api({
        space_id: spaceId,
        rerank_id: rerank.id,
      });

      if (response.code === 0) {
        fetchReranks();
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        fetchReranks();
      }
    }
  };

  // Open modal for editing
  const openEditModal = (rerank: SpaceRerankConfig) => {
    setEditingRerank(rerank);
    setSelectedType(rerank.rerank_type);
    setShowAddModal(true);
  };

  // Close modal
  const closeModal = () => {
    setShowAddModal(false);
    setEditingRerank(null);
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
              rules={[{ required: true, message: '请输入 API 地址' }]}
              placeholder="https://api.jina.ai/v1"
              initValue={editingRerank?.config?.openai_config?.base_url}
            />
            <Form.Input
              field="api_key"
              label="API Key"
              type="password"
              rules={[{ required: !editingRerank, message: '请输入 API Key' }]}
              placeholder="Bearer token..."
            />
            <Form.Input
              field="model"
              label="Model"
              rules={[{ required: true, message: '请输入模型名称' }]}
              placeholder="jina-reranker-v2-base-multilingual"
              initValue={editingRerank?.config?.openai_config?.model}
            />
          </>
        );

      default:
        return null;
    }
  };

  // Handle form submit via manual validation
  const handleSubmit = async () => {
    if (!spaceId) return;

    const formApi = formApiRef.current;
    if (!formApi) return;

    try {
      const values = await formApi.validate();

      const config: Record<string, any> = {};

      if (selectedType === 'openai') {
        config.base_url = values.base_url;
        config.api_key = values.api_key;
        config.model = values.model;
      }

      if (editingRerank) {
        // Update
        const response = await space_rerank.UpdateSpaceRerank({
          space_id: spaceId,
          rerank_id: editingRerank.id,
          name: values.name,
          description: values.description,
          type: selectedType,
          config,
        });

        if (response.code === 0) {
          Toast.success('更新成功');
          closeModal();
          fetchReranks();
        }
      } else {
        // Create
        const response = await space_rerank.CreateSpaceRerank({
          space_id: spaceId,
          name: values.name,
          description: values.description,
          type: selectedType,
          config,
          set_as_default: values.set_as_default || false,
        });

        if (response.code === 0) {
          Toast.success('创建成功');
          closeModal();
          fetchReranks();
        }
      }
    } catch (error: any) {
      if (error.code === 0 || error.code === '0') {
        Toast.success(editingRerank ? '更新成功' : '创建成功');
        closeModal();
        fetchReranks();
      } else if (error instanceof Object && !error.code) {
        // Form validation error - shown inline
      } else {
        Toast.error('操作失败: ' + (error.message || '未知错误'));
      }
    }
  };

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <div className="w-full flex items-center justify-between">
          <div>
            <Title heading={4}>Rerank 模型配置</Title>
            <div className="text-sm text-gray-500 mt-1">
              配置重排序模型后，知识库检索开启 Rerank 时将使用模型进行语义重排序，未配置则使用 RRF 算法
            </div>
          </div>
          <Button
            type="primary"
            icon={<IconCozPlus />}
            onClick={() => {
              setEditingRerank(null);
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
        ) : reranks.length === 0 ? (
          <div className="py-16 text-center text-gray-500">
            暂无 Rerank 模型配置。未配置时，知识库检索将使用 RRF（Reciprocal Rank Fusion）算法进行结果排序。
            <br />
            点击"添加配置"按钮配置 Rerank 模型以获得更好的语义重排序效果。
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {reranks.map(rerank => (
              <div
                key={rerank.id}
                className="bg-white border border-gray-200 rounded-lg shadow-sm p-4 flex flex-col gap-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-base font-medium text-gray-900">
                      {rerank.name}
                    </div>
                    <div className="text-xs text-gray-500 mt-1">
                      ID: {rerank.id}
                    </div>
                  </div>
                  <Space size="small">
                    {rerank.is_default && (
                      <Tag color="green">默认</Tag>
                    )}
                    <Tag color={rerank.status === 1 ? 'blue' : 'gray'}>
                      {rerank.status === 1 ? '启用' : '禁用'}
                    </Tag>
                    <Tag color="purple">
                      {rerank.rerank_type.toUpperCase()}
                    </Tag>
                  </Space>
                </div>

                <div className="text-sm text-gray-600">
                  {rerank.description || '暂无描述'}
                </div>

                <div className="text-xs text-gray-500 space-y-1">
                  {rerank.config?.openai_config?.model && (
                    <div>
                      <span className="font-medium text-gray-600">模型: </span>
                      {rerank.config.openai_config.model}
                    </div>
                  )}
                  <div>
                    <span className="font-medium text-gray-600">创建: </span>
                    {new Date(rerank.created_at).toLocaleString()}
                  </div>
                </div>

                <div className="flex items-center justify-end mt-auto pt-2 border-t">
                  <Space size="small">
                    <Button
                      size="small"
                      onClick={() => handleToggleStatus(rerank)}
                    >
                      {rerank.status === 1 ? '禁用' : '启用'}
                    </Button>
                    {!rerank.is_default && rerank.status === 1 && (
                      <Button
                        size="small"
                        onClick={() => handleSetDefault(rerank.id)}
                      >
                        设为默认
                      </Button>
                    )}
                    <Button size="small" onClick={() => openEditModal(rerank)}>
                      编辑
                    </Button>
                    <Popconfirm
                      title="确认删除"
                      content="删除后不可恢复，确定要删除吗？"
                      okText="删除"
                      cancelText="取消"
                      onConfirm={() => handleDelete(rerank.id)}
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
        title={editingRerank ? '编辑 Rerank 配置' : '添加 Rerank 配置'}
        visible={showAddModal}
        onCancel={closeModal}
        footer={null}
        style={{ width: 560 }}
      >
        <Form
          layout="vertical"
          autoComplete="off"
          getFormApi={(api: any) => { formApiRef.current = api; }}
        >
          <Form.Input
            field="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
            placeholder="My Rerank Model"
            initValue={editingRerank?.name}
          />
          <Form.Input
            field="description"
            label="描述"
            placeholder="配置描述..."
            initValue={editingRerank?.description}
          />
          <Form.Select
            field="type"
            label="类型"
            rules={[{ required: true, message: '请选择类型' }]}
            initValue={editingRerank?.rerank_type || 'openai'}
            onChange={value => setSelectedType(value as RerankType)}
          >
            {RERANK_TYPES.map(type => (
              <Select.Option key={type.value} value={type.value}>
                {type.label} - {type.description}
              </Select.Option>
            ))}
          </Form.Select>

          <div className="border-t pt-4 mt-4">
            <div className="text-sm font-medium text-gray-700 mb-3">配置参数</div>
            {renderConfigFields()}
          </div>

          {!editingRerank && (
            <Form.Checkbox field="set_as_default" className="mt-4">
              设为默认配置
            </Form.Checkbox>
          )}

          <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
            <Button onClick={closeModal}>取消</Button>
            <Button type="primary" onClick={handleSubmit}>
              {editingRerank ? '保存' : '创建'}
            </Button>
          </div>
        </Form>
      </Modal>
    </Layout>
  );
};

export { Page as Component };
export default Page;
