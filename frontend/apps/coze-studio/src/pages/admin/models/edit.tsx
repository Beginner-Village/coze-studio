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
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Input,
  Select,
  Switch,
  Toast,
  Spin,
} from '@coze-arch/coze-design';
import { admin } from '@coze-studio/api-schema';

export const AdminModelEditPage: React.FC = () => {
  const navigate = useNavigate();
  const { model_id } = useParams<{ model_id: string }>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    model_type: 'llm',
    protocol: 'openai',
    base_url: '',
    api_key: '',
    model: '',
    input_tokens: 128000,
    output_tokens: 4096,
    function_call: true,
    json_mode: false,
    reasoning: false,
    enable_thinking: false,
  });

  useEffect(() => {
    if (model_id) {
      fetchModel();
    }
  }, [model_id]);

  const fetchModel = async () => {
    try {
      const response = await admin.GetPublicModel({ model_id: model_id! });
      if (response.code === 0 && response.data) {
        const data = response.data;
        setFormData({
          name: data.name || '',
          description: data.description || '',
          model_type: data.model_type || 'llm',
          protocol: data.protocol || 'openai',
          base_url: data.base_url || '',
          api_key: '', // API Key 不返回，需要重新输入
          model: data.model || '',
          input_tokens: data.input_tokens || 128000,
          output_tokens: data.output_tokens || 4096,
          function_call: data.function_call ?? true,
          json_mode: data.json_mode ?? false,
          reasoning: data.reasoning ?? false,
          enable_thinking: data.enable_thinking ?? false,
        });
      }
    } catch (error: any) {
      // 处理API客户端特殊情况
      if (error.code === '0' || error.code === 0) {
        const data = error.response?.data?.data || error.data;
        if (data) {
          setFormData({
            name: data.name || '',
            description: data.description || '',
            model_type: data.model_type || 'llm',
            protocol: data.protocol || 'openai',
            base_url: data.base_url || '',
            api_key: '',
            model: data.model || '',
            input_tokens: data.input_tokens || 128000,
            output_tokens: data.output_tokens || 4096,
            function_call: data.function_call ?? true,
            json_mode: data.json_mode ?? false,
            reasoning: data.reasoning ?? false,
            enable_thinking: data.enable_thinking ?? false,
          });
        }
      } else {
        console.error('获取模型详情失败:', error);
        Toast.error('获取模型详情失败');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async () => {
    if (!formData.name || !formData.base_url || !formData.model) {
      Toast.error('请填写必填字段');
      return;
    }

    setSaving(true);
    try {
      const response = await admin.UpdatePublicModel({
        model_id: model_id!,
        name: formData.name,
        description: formData.description,
        model_type: formData.model_type,
        protocol: formData.protocol,
        base_url: formData.base_url,
        api_key: formData.api_key || undefined, // 如果为空则不更新
        model: formData.model,
        input_tokens: formData.input_tokens,
        output_tokens: formData.output_tokens,
        function_call: formData.function_call,
        json_mode: formData.json_mode,
        reasoning: formData.reasoning,
        enable_thinking: formData.enable_thinking,
      });

      if (response.code === 0) {
        Toast.success('更新成功');
        navigate('/admin/models');
      } else {
        Toast.error(response.msg || '更新失败');
      }
    } catch (error: any) {
      // 处理API客户端特殊情况
      if (error.code === '0' || error.code === 0) {
        Toast.success('更新成功');
        navigate('/admin/models');
      } else {
        console.error('更新失败:', error);
        Toast.error('更新失败');
      }
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-semibold">编辑公共模型</h2>
        <Button onClick={() => navigate('/admin/models')}>返回</Button>
      </div>

      <div className="bg-white rounded-lg border p-6 space-y-6">
        {/* 基本信息 */}
        <div>
          <h3 className="font-medium mb-4">基本信息</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">
                模型名称 <span className="text-red-500">*</span>
              </label>
              <Input
                value={formData.name}
                onChange={(val) => setFormData({ ...formData, name: val })}
                placeholder="例如: GPT-4o"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">描述</label>
              <Input
                value={formData.description}
                onChange={(val) => setFormData({ ...formData, description: val })}
                placeholder="模型描述"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1">模型类型</label>
                <Select
                  value={formData.model_type}
                  onChange={(val) => setFormData({ ...formData, model_type: val as string })}
                >
                  <Select.Option value="llm">LLM</Select.Option>
                  <Select.Option value="embedding">Embedding</Select.Option>
                  <Select.Option value="rerank">Rerank</Select.Option>
                  <Select.Option value="tts">TTS</Select.Option>
                </Select>
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">协议</label>
                <Select
                  value={formData.protocol}
                  onChange={(val) => setFormData({ ...formData, protocol: val as string })}
                >
                  <Select.Option value="openai">OpenAI</Select.Option>
                  <Select.Option value="claude">Claude</Select.Option>
                  <Select.Option value="gemini">Gemini</Select.Option>
                  <Select.Option value="qwen">Qwen</Select.Option>
                  <Select.Option value="deepseek">DeepSeek</Select.Option>
                  <Select.Option value="ark">Volcengine Ark</Select.Option>
                </Select>
              </div>
            </div>
          </div>
        </div>

        {/* 连接配置 */}
        <div>
          <h3 className="font-medium mb-4">连接配置</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">
                Base URL <span className="text-red-500">*</span>
              </label>
              <Input
                value={formData.base_url}
                onChange={(val) => setFormData({ ...formData, base_url: val })}
                placeholder="例如: https://api.openai.com/v1"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">
                API Key
              </label>
              <Input
                type="password"
                value={formData.api_key}
                onChange={(val) => setFormData({ ...formData, api_key: val })}
                placeholder="留空则不更新 API Key"
              />
              <p className="text-xs text-gray-400 mt-1">
                出于安全考虑，API Key 不会显示。如需修改请重新输入。
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">
                模型标识 <span className="text-red-500">*</span>
              </label>
              <Input
                value={formData.model}
                onChange={(val) => setFormData({ ...formData, model: val })}
                placeholder="例如: gpt-4o"
              />
            </div>
          </div>
        </div>

        {/* 能力配置 */}
        <div>
          <h3 className="font-medium mb-4">能力配置</h3>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1">输入 Tokens</label>
                <Input
                  type="number"
                  value={String(formData.input_tokens)}
                  onChange={(val) => setFormData({ ...formData, input_tokens: parseInt(val) || 0 })}
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1">输出 Tokens</label>
                <Input
                  type="number"
                  value={String(formData.output_tokens)}
                  onChange={(val) => setFormData({ ...formData, output_tokens: parseInt(val) || 0 })}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="flex items-center justify-between p-3 bg-gray-50 rounded">
                <span className="text-sm">Function Call</span>
                <Switch
                  checked={formData.function_call}
                  onChange={(checked) => setFormData({ ...formData, function_call: checked })}
                />
              </div>

              <div className="flex items-center justify-between p-3 bg-gray-50 rounded">
                <span className="text-sm">JSON Mode</span>
                <Switch
                  checked={formData.json_mode}
                  onChange={(checked) => setFormData({ ...formData, json_mode: checked })}
                />
              </div>

              <div className="flex items-center justify-between p-3 bg-gray-50 rounded">
                <span className="text-sm">Reasoning</span>
                <Switch
                  checked={formData.reasoning}
                  onChange={(checked) => setFormData({ ...formData, reasoning: checked })}
                />
              </div>

              <div className="flex items-center justify-between p-3 bg-gray-50 rounded">
                <span className="text-sm">Enable Thinking</span>
                <Switch
                  checked={formData.enable_thinking}
                  onChange={(checked) => setFormData({ ...formData, enable_thinking: checked })}
                />
              </div>
            </div>
          </div>
        </div>

        {/* 提交按钮 */}
        <div className="flex justify-end gap-3 pt-4 border-t">
          <Button onClick={() => navigate('/admin/models')}>取消</Button>
          <Button type="primary" loading={saving} onClick={handleSubmit}>
            保存
          </Button>
        </div>
      </div>
    </div>
  );
};

export { AdminModelEditPage as Component };
export default AdminModelEditPage;
