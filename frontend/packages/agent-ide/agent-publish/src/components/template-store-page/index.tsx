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

import React, { useState, useEffect, type FC } from 'react';
import { useLoaderData, useNavigate } from 'react-router-dom';
import { templateApi, type StoreTemplateInfo } from '@coze-arch/bot-api';
import { Button, Toast } from '@coze-arch/bot-semi';
import { I18n } from '@coze-arch/i18n';

interface LoaderData {
  type: string;
  showCopyButton?: boolean;
}

// 模板卡片组件
const StoreTemplateCard: FC<{
  template: StoreTemplateInfo;
  onExperience: () => void;
  showCopyButton?: boolean;
}> = ({ template, onExperience, showCopyButton = false }) => {
  return (
    <div
      style={{
        border: '1px solid #e6e6e6',
        borderRadius: '8px',
        padding: '16px',
        backgroundColor: '#fff',
        boxShadow: '0 2px 4px rgba(0,0,0,0.05)',
        transition: 'all 0.2s ease',
        cursor: 'pointer',
        height: '278px', // 与模板页面卡片高度一致
        display: 'flex',
        flexDirection: 'column',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = '0 4px 8px rgba(0,0,0,0.1)';
        e.currentTarget.style.transform = 'translateY(-2px)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = '0 2px 4px rgba(0,0,0,0.05)';
        e.currentTarget.style.transform = 'translateY(0)';
      }}
    >
      {/* 顶部图片区域 */}
      <div 
        style={{
          position: 'relative',
          width: '100%',
          height: '140px',
          borderRadius: '8px',
          overflow: 'hidden',
          marginBottom: '16px',
          backgroundColor: '#f5f5f5',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {template.cover_url ? (
          <img 
            src={template.cover_url} 
            alt={template.title}
            style={{ 
              width: '100%', 
              height: '100%', 
              objectFit: 'cover',
              objectPosition: 'center'
            }}
          />
        ) : (
          <span style={{ fontSize: '48px', color: '#999' }}>🤖</span>
        )}
      </div>
      
      {/* 内容区域 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* 标题 */}
        <h3 style={{ 
          margin: '0 0 8px 0', 
          fontSize: '16px', 
          fontWeight: 600,
          color: '#333',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          lineHeight: '22px'
        }}>
          {template.title}
        </h3>
        
        {/* 描述 */}
        <p style={{
          margin: '0 0 12px 0',
          fontSize: '14px',
          color: '#666',
          lineHeight: '20px',
          overflow: 'hidden',
          display: '-webkit-box',
          WebkitLineClamp: 2,
          WebkitBoxOrient: 'vertical',
          flex: 1
        }}>
          {template.description || '暂无描述'}
        </p>

        {/* 底部信息和按钮 */}
        <div style={{ marginTop: 'auto' }}>
          {/* 作者信息 */}
          <div style={{ 
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center',
            fontSize: '12px',
            color: '#999',
            marginBottom: '12px'
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <div style={{
                width: '16px',
                height: '16px',
                borderRadius: '50%',
                backgroundColor: '#f0f0f0',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                overflow: 'hidden',
              }}>
                {template.author_avatar ? (
                  <img 
                    src={template.author_avatar} 
                    alt={template.author_name}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  <span style={{ fontSize: '8px' }}>👤</span>
                )}
              </div>
              <span>{template.author_name || 'Anonymous'}</span>
            </div>
            
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              {template.heat && template.heat > 0 && (
                <span>🔥 {template.heat}</span>
              )}
            </div>
          </div>
          
          {/* 操作按钮 */}
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button
              onClick={(e) => {
                e.stopPropagation();
                onExperience();
              }}
              type="primary"
              size="small"
              style={{ flex: 1 }}
            >
              立即体验
            </Button>
            
            {showCopyButton && (
              <Button
                onClick={(e) => {
                  e.stopPropagation();
                  Toast.info('复制功能待实现');
                }}
                size="small"
                style={{ flex: 1 }}
              >
                复制模板
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

// 骨架屏组件
const StoreTemplateCardSkeleton: FC = () => {
  return (
    <div
      style={{
        border: '1px solid #e6e6e6',
        borderRadius: '8px',
        padding: '16px',
        backgroundColor: '#fff',
        height: '278px',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* 图片骨架屏 */}
      <div 
        style={{
          width: '100%',
          height: '140px',
          borderRadius: '8px',
          backgroundColor: '#f0f0f0',
          marginBottom: '16px',
          animation: 'pulse 1.5s ease-in-out infinite alternate',
        }}
      />
      
      {/* 内容骨架屏 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* 标题骨架屏 */}
        <div 
          style={{
            height: '22px',
            backgroundColor: '#f0f0f0',
            borderRadius: '4px',
            marginBottom: '8px',
            animation: 'pulse 1.5s ease-in-out infinite alternate',
          }}
        />
        
        {/* 描述骨架屏 */}
        <div 
          style={{
            height: '40px',
            backgroundColor: '#f0f0f0',
            borderRadius: '4px',
            marginBottom: '12px',
            animation: 'pulse 1.5s ease-in-out infinite alternate',
          }}
        />
        
        {/* 底部骨架屏 */}
        <div style={{ marginTop: 'auto' }}>
          <div 
            style={{
              height: '16px',
              backgroundColor: '#f0f0f0',
              borderRadius: '4px',
              marginBottom: '12px',
              animation: 'pulse 1.5s ease-in-out infinite alternate',
            }}
          />
          <div 
            style={{
              height: '32px',
              backgroundColor: '#f0f0f0',
              borderRadius: '4px',
              animation: 'pulse 1.5s ease-in-out infinite alternate',
            }}
          />
        </div>
      </div>
      
      <style>
        {`
          @keyframes pulse {
            0% {
              opacity: 1;
            }
            100% {
              opacity: 0.4;
            }
          }
        `}
      </style>
    </div>
  );
};

export const TemplateStorePage: FC = () => {
  const loaderData = useLoaderData() as LoaderData;
  const navigate = useNavigate();
  
  // 根据路由类型判断显示模式
  const isProjectStore = loaderData?.type === 'project-latest' || loaderData?.type === 'project-store';
  
  const [templates, setTemplates] = useState<StoreTemplateInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 获取商店模板数据
  const loadTemplates = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await templateApi.getStoreTemplateList({
        page_num: 0,
        page_size: 1000,
      });

      if (response.code === 0) {
        setTemplates(response.templates || []);
      } else {
        setError(`加载失败: ${response.msg || '未知错误'}`);
      }
    } catch (error: any) {
      console.error('Load store templates error:', error);
      
      // 处理特殊的成功响应错误处理
      if (error.code === '200' || error.code === 200) {
        const responseData = error.response?.data;
        if (responseData && responseData.templates) {
          setTemplates(responseData.templates || []);
          return;
        }
      }
      
      setError(`加载失败: ${error.message || '网络错误'}`);
    } finally {
      setLoading(false);
    }
  };

  // 处理体验操作
  const handleExperience = (template: StoreTemplateInfo) => {
    if (template.agent_id) {
      // 商店智能体位于特殊的空间ID 888888 中
      navigate(`/space/888888/bot/${template.agent_id}`);
    } else {
      Toast.error('智能体ID不存在');
    }
  };

  // 刷新数据
  const refresh = () => {
    loadTemplates();
  };

  useEffect(() => {
    loadTemplates();
  }, []);

  // 错误状态
  if (error && !loading) {
    return (
      <div style={{ padding: '24px' }}>
        <h2 style={{ 
          lineHeight: '72px', 
          fontSize: '20px', 
          margin: '0', 
          paddingLeft: '24px', 
          paddingRight: '24px' 
        }}>
          {isProjectStore ? '最新智能体' : '模板商店'}
        </h2>
        
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '80px',
          textAlign: 'center'
        }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>⚠️</div>
          <div style={{ fontSize: '18px', fontWeight: 500, marginBottom: '8px' }}>
            加载失败
          </div>
          <div style={{ color: '#666', marginBottom: '24px' }}>
            {error}
          </div>
          <Button onClick={refresh} type="primary">
            重试
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div style={{ backgroundColor: 'transparent' }}>
      <h2 style={{ 
        lineHeight: '72px', 
        fontSize: '20px', 
        margin: '0', 
        paddingLeft: '24px', 
        paddingRight: '24px' 
      }}>
        {isProjectStore ? '最新智能体' : '模板商店'}
      </h2>

      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, 1fr)',
        gap: '20px',
        padding: '0 24px 24px 24px',
      }}
      className="[@media(min-width:1600px)]:grid-cols-4"
      >
        {loading ? (
          // 显示骨架屏
          Array.from({ length: 20 }, (_, index) => (
            <StoreTemplateCardSkeleton key={index} />
          ))
        ) : (
          // 显示模板卡片
          templates.map((template) => (
            <StoreTemplateCard
              key={template.template_id}
              template={template}
              onExperience={() => handleExperience(template)}
              showCopyButton={!isProjectStore}
            />
          ))
        )}
      </div>

      {/* 空状态 */}
      {!loading && templates.length === 0 && !error && (
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '80px',
          textAlign: 'center'
        }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>📦</div>
          <div style={{ fontSize: '18px', fontWeight: 500, marginBottom: '8px' }}>
            暂无模板
          </div>
          <div style={{ color: '#666' }}>
            快来发布第一个模板吧！
          </div>
        </div>
      )}
    </div>
  );
};