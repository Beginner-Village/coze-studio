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

import React, { useState } from 'react';
import type { SpaceModelItem } from '@coze-arch/bot-space-api';
import { ModelCard } from '../ModelCard';
import { useSpaceModels, useSpaceModelsByProtocol } from '../../hooks/useSpaceModels';

export interface ModelListProps {
  className?: string;
  onModelClick?: (model: SpaceModelItem) => void;
}

type ViewMode = 'grid' | 'protocol';

export const ModelList: React.FC<ModelListProps> = ({
  className = '',
  onModelClick,
}) => {
  const [viewMode, setViewMode] = useState<ViewMode>('grid');
  const [searchKeyword, setSearchKeyword] = useState('');
  
  const { data: models, loading, error, refresh } = useSpaceModels();
  const { data: modelsByProtocol } = useSpaceModelsByProtocol();

  // 搜索过滤
  const filteredModels = models?.filter(model => {
    if (!searchKeyword) return true;
    const keyword = searchKeyword.toLowerCase();
    return (
      model.name.toLowerCase().includes(keyword) ||
      model.description.toLowerCase().includes(keyword) ||
      model.protocol.toLowerCase().includes(keyword)
    );
  }) || [];

  // 加载状态
  if (loading) {
    return (
      <div className={`space-model-list ${className}`}>
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-3 text-gray-600">正在加载模型列表...</span>
        </div>
      </div>
    );
  }

  // 错误状态
  if (error) {
    return (
      <div className={`space-model-list ${className}`}>
        <div className="flex flex-col items-center justify-center py-12">
          <div className="text-red-500 mb-2">⚠️ 加载失败</div>
          <p className="text-gray-600 mb-4">无法获取模型列表，请稍后重试</p>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            重新加载
          </button>
        </div>
      </div>
    );
  }

  // 空状态
  if (!models || models.length === 0) {
    return (
      <div className={`space-model-list ${className}`}>
        <div className="text-center py-12">
          <div className="text-6xl mb-4">🤖</div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">暂无可用模型</h3>
          <p className="text-gray-600">当前空间下还没有配置任何 AI 模型</p>
        </div>
      </div>
    );
  }

  // 渲染网格视图
  const renderGridView = () => (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {filteredModels.map((model) => (
        <ModelCard
          key={model.id}
          model={model}
          onClick={onModelClick}
        />
      ))}
    </div>
  );

  // 渲染协议分组视图
  const renderProtocolView = () => {
    const filteredProtocolGroups = Object.entries(modelsByProtocol).filter(
      ([, models]) => models.some(model => filteredModels.includes(model))
    );

    return (
      <div className="space-y-8">
        {filteredProtocolGroups.map(([protocol, protocolModels]) => {
          const visibleModels = protocolModels.filter(model => 
            filteredModels.includes(model)
          );
          
          if (visibleModels.length === 0) return null;

          return (
            <div key={protocol}>
              <div className="flex items-center mb-4">
                <h3 className="text-lg font-semibold text-gray-900 capitalize">
                  {protocol}
                </h3>
                <span className="ml-2 text-sm text-gray-500 bg-gray-100 px-2 py-1 rounded">
                  {visibleModels.length} 个模型
                </span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {visibleModels.map((model) => (
                  <ModelCard
                    key={model.id}
                    model={model}
                    onClick={onModelClick}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className={`space-model-list ${className}`}>
      {/* 页面头部 */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">可用模型</h2>
            <p className="text-gray-600 mt-1">
              当前空间下可使用的 AI 模型 ({models.length} 个)
            </p>
          </div>
          
          {/* 视图切换和添加按钮 */}
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setViewMode('grid')}
                className={`px-3 py-2 text-sm rounded transition-colors ${
                  viewMode === 'grid'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                网格视图
              </button>
              <button
                onClick={() => setViewMode('protocol')}
                className={`px-3 py-2 text-sm rounded transition-colors ${
                  viewMode === 'protocol'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                协议分组
              </button>
            </div>
            
            {/* 添加模型按钮 */}
            <button
              onClick={() => {
                // TODO: 实现添加模型功能
                console.log('添加模型按钮被点击');
              }}
              className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded hover:bg-blue-700 transition-colors flex items-center space-x-2"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              <span>添加模型</span>
            </button>
          </div>
        </div>

        {/* 搜索框 */}
        <div className="max-w-md">
          <input
            type="text"
            placeholder="搜索模型名称、描述或协议..."
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>

      {/* 搜索结果提示 */}
      {searchKeyword && (
        <div className="mb-4 text-sm text-gray-600">
          {filteredModels.length > 0 
            ? `找到 ${filteredModels.length} 个匹配的模型`
            : '未找到匹配的模型'
          }
        </div>
      )}

      {/* 模型列表 */}
      {filteredModels.length > 0 ? (
        viewMode === 'grid' ? renderGridView() : renderProtocolView()
      ) : searchKeyword ? (
        <div className="text-center py-12">
          <div className="text-4xl mb-4">🔍</div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">未找到匹配结果</h3>
          <p className="text-gray-600">请尝试其他关键词进行搜索</p>
        </div>
      ) : null}
    </div>
  );
};