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
import { useParams } from 'react-router-dom';
import { templateApi } from '@coze-arch/bot-api';
import { TextArea, Button, Upload, Toast } from '@coze-arch/bot-semi';
import { type DynamicParams } from '@coze-arch/bot-typings/teamspace';

interface TemplatePublishSectionProps {
  disabled?: boolean;
  className?: string;
}

export const TemplatePublishSection: React.FC<TemplatePublishSectionProps> = ({
  disabled = false,
  className = '',
}) => {
  const params = useParams<DynamicParams>();
  const { bot_id } = params;
  
  // 共享的表单状态
  const [templateTitle, setTemplateTitle] = useState('');
  const [templateDescription, setTemplateDescription] = useState('');
  const [coverImage, setCoverImage] = useState<string>('');
  const [coverImagePreview, setCoverImagePreview] = useState<string>('');
  const [uploadingImage, setUploadingImage] = useState(false);
  
  // 个人模板状态
  const [enablePersonalTemplate, setEnablePersonalTemplate] = useState(false);
  const [personalPublishing, setPersonalPublishing] = useState(false);
  const [personalUnpublishing, setPersonalUnpublishing] = useState(false);
  const [isPersonalPublished, setIsPersonalPublished] = useState(false);
  const [personalTemplate, setPersonalTemplate] = useState<any>(null);
  
  // 商店模板状态
  const [enableStoreTemplate, setEnableStoreTemplate] = useState(false);
  const [storePublishing, setStorePublishing] = useState(false);
  const [storeUnpublishing, setStoreUnpublishing] = useState(false);
  const [isStorePublished, setIsStorePublished] = useState(false);
  const [storeTemplate, setStoreTemplate] = useState<any>(null);
  const [storeTags, setStoreTags] = useState<string[]>([]);
  const [newTag, setNewTag] = useState('');
  
  const [loading, setLoading] = useState(false);

  // 初始化时检查发布状态
  useEffect(() => {
    if (bot_id) {
      checkPublishStatus();
    }
  }, [bot_id]);

  const checkPublishStatus = async () => {
    if (!bot_id) return;
    
    try {
      setLoading(true);
      
      // 并行检查个人模板和商店模板状态
      const [personalResponse, storeResponse] = await Promise.all([
        templateApi.checkPublishStatus({ agent_id: bot_id }),
        templateApi.checkStorePublishStatus({ agent_id: bot_id })
      ]);

      // 处理个人模板状态
      if (personalResponse.code === 0) {
        setIsPersonalPublished(personalResponse.is_published);
        if (personalResponse.is_published && personalResponse.template_info) {
          setPersonalTemplate(personalResponse.template_info);
          setTemplateTitle(personalResponse.template_info.title);
          setTemplateDescription(personalResponse.template_info.description || '');
          setEnablePersonalTemplate(true);
          
          // 优先使用可访问的URL，如果没有则使用存储路径
          if (personalResponse.template_info.cover_url) {
            setCoverImage(personalResponse.template_info.cover_url);
          } else if (personalResponse.template_info.cover_uri) {
            setCoverImage(personalResponse.template_info.cover_uri);
          }
        }
      }

      // 处理商店模板状态
      if (storeResponse.code === 0) {
        setIsStorePublished(storeResponse.is_published);
        if (storeResponse.is_published && storeResponse.template_info) {
          setStoreTemplate(storeResponse.template_info);
          setEnableStoreTemplate(true);
          
          // 如果个人模板还没有设置标题，使用商店模板的信息
          if (!templateTitle && storeResponse.template_info.title) {
            setTemplateTitle(storeResponse.template_info.title);
            setTemplateDescription(storeResponse.template_info.description || '');
            
            // 设置封面图片
            if (storeResponse.template_info.cover_url) {
              setCoverImage(storeResponse.template_info.cover_url);
            } else if (storeResponse.template_info.cover_uri) {
              setCoverImage(storeResponse.template_info.cover_uri);
            }
          }
          
          if (storeResponse.template_info.tags) {
            setStoreTags(storeResponse.template_info.tags);
          }
        }
      }
    } catch (error: any) {
      console.error('Check publish status failed:', error);
      // 处理特殊的成功响应格式
      if (error.code === '200' || error.code === 200) {
        const responseData = error.response?.data;
        if (responseData) {
          // 处理可能的成功响应...
        }
      }
    } finally {
      setLoading(false);
    }
  };

  // 个人模板发布
  const handlePublishPersonalTemplate = async () => {
    if (!bot_id || !templateTitle.trim()) {
      Toast.error('请输入模板标题');
      return;
    }

    try {
      setPersonalPublishing(true);
      
      const response = await templateApi.publishAsTemplate({
        agent_id: bot_id,
        title: templateTitle.trim(),
        description: templateDescription.trim() || undefined,
        is_public: true,
        cover_uri: coverImage || undefined,
      });

      if (response.code === 0) {
        Toast.success(response.status === 'updated' ? '个人模板更新成功！' : '个人模板发布成功！');
        setIsPersonalPublished(true);
        await checkPublishStatus();
      } else {
        Toast.error('个人模板操作失败: ' + (response.msg || '未知错误'));
      }
    } catch (error: any) {
      console.error('Personal template publish error:', error);
      Toast.error('个人模板发布失败: ' + error.message);
    } finally {
      setPersonalPublishing(false);
    }
  };

  // 个人模板取消发布
  const handleUnpublishPersonalTemplate = async () => {
    if (!bot_id) return;

    try {
      setPersonalUnpublishing(true);
      
      const response = await templateApi.unpublishTemplate({
        agent_id: bot_id,
      });

      if (response.code === 0) {
        Toast.success('个人模板取消发布成功！');
        setIsPersonalPublished(false);
        setPersonalTemplate(null);
        setEnablePersonalTemplate(false);
      } else {
        Toast.error('个人模板取消发布失败: ' + (response.msg || '未知错误'));
      }
    } catch (error: any) {
      console.error('Personal template unpublish error:', error);
      Toast.error('个人模板取消发布失败: ' + error.message);
    } finally {
      setPersonalUnpublishing(false);
    }
  };

  // 商店模板发布
  const handlePublishStoreTemplate = async () => {
    if (!bot_id || !templateTitle.trim()) {
      Toast.error('请输入模板标题');
      return;
    }

    try {
      setStorePublishing(true);
      
      const response = await templateApi.publishToStore({
        agent_id: bot_id,
        title: templateTitle.trim(),
        description: templateDescription.trim() || undefined,
        tags: storeTags.length > 0 ? storeTags : undefined,
        cover_uri: coverImage || undefined,
      });

      if (response.code === 0) {
        Toast.success(response.status === 'updated' ? '商店模板更新成功！' : '商店模板发布成功！');
        setIsStorePublished(true);
        await checkPublishStatus();
      } else {
        Toast.error('商店模板操作失败: ' + (response.msg || '未知错误'));
      }
    } catch (error: any) {
      console.error('Store template publish error:', error);
      
      // 处理特殊的成功响应
      if (error.code === '200' || error.code === 200) {
        const responseData = error.response?.data;
        if (responseData && responseData.store_template_id) {
          Toast.success('商店模板发布成功！');
          setIsStorePublished(true);
          await checkPublishStatus();
          return;
        }
      }
      
      Toast.error('商店模板发布失败: ' + error.message);
    } finally {
      setStorePublishing(false);
    }
  };

  // 商店模板取消发布
  const handleUnpublishStoreTemplate = async () => {
    if (!bot_id) return;

    try {
      setStoreUnpublishing(true);
      
      const response = await templateApi.unpublishFromStore({
        agent_id: bot_id,
      });

      if (response.code === 0) {
        Toast.success('商店模板取消发布成功！');
        setIsStorePublished(false);
        setStoreTemplate(null);
        setEnableStoreTemplate(false);
        setStoreTags([]);
      } else {
        Toast.error('商店模板取消发布失败: ' + (response.msg || '未知错误'));
      }
    } catch (error: any) {
      console.error('Store template unpublish error:', error);
      Toast.error('商店模板取消发布失败: ' + error.message);
    } finally {
      setStoreUnpublishing(false);
    }
  };

  // 标签管理
  const handleAddTag = () => {
    if (newTag.trim() && !storeTags.includes(newTag.trim())) {
      setStoreTags([...storeTags, newTag.trim()]);
      setNewTag('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setStoreTags(storeTags.filter(tag => tag !== tagToRemove));
  };

  // 图片上传
  const fileToBase64 = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.readAsDataURL(file);
      reader.onload = () => {
        const result = reader.result as string;
        const base64 = result.split(',')[1];
        resolve(base64);
      };
      reader.onerror = error => reject(error);
    });
  };

  const getFileExtension = (fileName: string): string => {
    return fileName.split('.').pop()?.toLowerCase() || '';
  };

  const handleImageUpload = async (fileInfo: any) => {
    const { file } = fileInfo;
    
    try {
      setUploadingImage(true);
      
      if (file && file.fileInstance) {
        const fileInstance = file.fileInstance as File;
        const fileExtension = getFileExtension(fileInstance.name);
        const base64Data = await fileToBase64(fileInstance);
        
        // 设置预览
        const reader = new FileReader();
        reader.onload = (e) => {
          setCoverImagePreview(e.target?.result as string);
        };
        reader.readAsDataURL(fileInstance);

        // 上传图片
        const response = await templateApi.uploadTemplateIcon({
          file_head: {
            file_type: fileExtension,
            biz_type: 11,
          },
          data: base64Data,
        });

        if (response.code === 0) {
          setCoverImage(response.data.upload_uri);
          Toast.success('图片上传成功');
        } else {
          Toast.error('图片上传失败: ' + response.msg);
        }
      } else {
        Toast.error('请选择有效的图片文件');
      }
    } catch (error: any) {
      console.error('Image upload error:', error);
      
      if (error.code === '200' || error.code === 200) {
        const responseData = error.response?.data;
        if (responseData && responseData.data) {
          setCoverImage(responseData.data.upload_uri);
          Toast.success('图片上传成功');
          return;
        }
      }
      
      Toast.error('图片上传失败: ' + error.message);
    } finally {
      setUploadingImage(false);
    }
  };

  if (loading) {
    return (
      <div style={{ border: '1px solid #e6e6e6', padding: '16px', borderRadius: '8px', marginTop: '16px' }}>
        <div style={{ textAlign: 'center', padding: '20px' }}>
          检查发布状态中...
        </div>
      </div>
    );
  }

  return (
    <div style={{ border: '1px solid #e6e6e6', padding: '16px', borderRadius: '8px', marginTop: '16px' }}>
      <div style={{ marginBottom: '24px' }}>
        <h3 style={{ margin: '0 0 8px 0', fontSize: '16px', fontWeight: 600 }}>模板发布</h3>
        <p style={{ margin: '0', fontSize: '14px', color: '#666' }}>将您的智能体发布为模板供他人使用</p>
      </div>

      {/* 共享的模板信息表单 */}
      {(enablePersonalTemplate || enableStoreTemplate) && (
        <div style={{ marginBottom: '32px' }}>
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontSize: '14px', fontWeight: 500 }}>
              模板标题 *
            </label>
            <TextArea
              value={templateTitle}
              onChange={setTemplateTitle}
              placeholder="请输入模板标题"
              maxLength={100}
              rows={1}
              showClear
              disabled={disabled}
              style={{ fontSize: '14px' }}
            />
          </div>

          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontSize: '14px', fontWeight: 500 }}>
              模板描述
            </label>
            <TextArea
              value={templateDescription}
              onChange={setTemplateDescription}
              placeholder="请输入模板描述（可选）"
              rows={3}
              maxLength={500}
              showClear
              disabled={disabled}
              style={{ fontSize: '14px' }}
            />
          </div>

          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontSize: '14px', fontWeight: 500 }}>
              模板封面
            </label>
            <Upload
              action=""
              accept="image/*"
              maxCount={1}
              customRequest={handleImageUpload}
              disabled={disabled || uploadingImage}
              style={{ width: '100%' }}
              showUploadList={false}
            >
              <Button disabled={disabled || uploadingImage} loading={uploadingImage} style={{ width: '100%' }}>
                {uploadingImage ? '上传中...' : (coverImage ? '更换图片' : '上传封面图片（可选）')}
              </Button>
            </Upload>
            {(coverImagePreview || coverImage) && (
              <div style={{ marginTop: '8px', textAlign: 'center' }}>
                <img 
                  src={coverImagePreview || coverImage} 
                  alt="模板封面预览" 
                  style={{ 
                    maxWidth: '200px', 
                    maxHeight: '150px', 
                    objectFit: 'cover',
                    borderRadius: '4px',
                    border: '1px solid #e6e6e6'
                  }} 
                />
                <div style={{ fontSize: '12px', color: '#999', marginTop: '4px' }}>
                  当前封面图片
                </div>
              </div>
            )}
            <div style={{ fontSize: '12px', color: '#999', marginTop: '4px' }}>
              如不上传，将使用智能体的默认图标作为模板封面
            </div>
          </div>
        </div>
      )}

      {/* 个人模板区域 */}
      <div style={{ marginBottom: '32px', padding: '16px', backgroundColor: '#fafafa', borderRadius: '8px' }}>
        <div style={{ marginBottom: '16px' }}>
          <h4 style={{ margin: '0 0 4px 0', fontSize: '14px', fontWeight: 600 }}>个人模板</h4>
          <p style={{ margin: '0', fontSize: '12px', color: '#666' }}>发布后所有用户都可以复制的模板</p>
        </div>

        {isPersonalPublished && personalTemplate && (
          <div style={{ 
            backgroundColor: '#f6ffed', 
            border: '1px solid #b7eb8f', 
            borderRadius: '6px', 
            padding: '12px', 
            marginBottom: '16px' 
          }}>
            <div style={{ color: '#52c41a', fontWeight: 500, marginBottom: '4px' }}>
              ✅ 已发布为个人模板
            </div>
            <div style={{ fontSize: '12px', color: '#666' }}>
              {personalTemplate.title}
            </div>
          </div>
        )}

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={enablePersonalTemplate}
              onChange={e => setEnablePersonalTemplate(e.target.checked)}
              disabled={disabled}
              style={{ marginRight: '8px' }}
            />
            <span style={{ fontSize: '14px' }}>
              {isPersonalPublished ? '更新个人模板' : '启用个人模板'}
            </span>
          </label>
        </div>

        {enablePersonalTemplate && (
          <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
            <Button
              onClick={handlePublishPersonalTemplate}
              disabled={disabled || !templateTitle.trim() || personalPublishing}
              loading={personalPublishing}
              type="primary"
              size="small"
            >
              {isPersonalPublished ? '更新个人模板' : '发布个人模板'}
            </Button>

            {isPersonalPublished && (
              <Button
                onClick={handleUnpublishPersonalTemplate}
                disabled={disabled || personalUnpublishing}
                loading={personalUnpublishing}
                type="secondary"
                size="small"
              >
                取消发布
              </Button>
            )}
          </div>
        )}
      </div>

      {/* 商店模板区域 */}
      <div style={{ padding: '16px', backgroundColor: '#f0f8ff', borderRadius: '8px' }}>
        <div style={{ marginBottom: '16px' }}>
          <h4 style={{ margin: '0 0 4px 0', fontSize: '14px', fontWeight: 600 }}>模板商店</h4>
          <p style={{ margin: '0', fontSize: '12px', color: '#666' }}>发布到全局商店，用户可以发现和立即体验</p>
        </div>

        {isStorePublished && storeTemplate && (
          <div style={{ 
            backgroundColor: '#e6f4ff', 
            border: '1px solid #91caff', 
            borderRadius: '6px', 
            padding: '12px', 
            marginBottom: '16px' 
          }}>
            <div style={{ color: '#1677ff', fontWeight: 500, marginBottom: '4px' }}>
              🌟 已发布到商店
            </div>
            <div style={{ fontSize: '12px', color: '#666' }}>
              {storeTemplate.title}
              {storeTemplate.tags && storeTemplate.tags.length > 0 && (
                <span style={{ marginLeft: '8px' }}>
                  标签: {storeTemplate.tags.join(', ')}
                </span>
              )}
            </div>
          </div>
        )}

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={enableStoreTemplate}
              onChange={e => setEnableStoreTemplate(e.target.checked)}
              disabled={disabled}
              style={{ marginRight: '8px' }}
            />
            <span style={{ fontSize: '14px' }}>
              {isStorePublished ? '更新商店模板' : '发布到商店'}
            </span>
          </label>
        </div>

        {enableStoreTemplate && (
          <div>
            {/* 标签管理 */}
            <div style={{ marginBottom: '16px' }}>
              <label style={{ display: 'block', marginBottom: '8px', fontSize: '14px', fontWeight: 500 }}>
                模板标签
              </label>
              <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
                <TextArea
                  value={newTag}
                  onChange={setNewTag}
                  placeholder="输入标签名称"
                  rows={1}
                  disabled={disabled}
                  style={{ fontSize: '14px', flex: 1 }}
                  onKeyPress={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      handleAddTag();
                    }
                  }}
                />
                <Button
                  onClick={handleAddTag}
                  disabled={disabled || !newTag.trim()}
                  size="small"
                >
                  添加
                </Button>
              </div>
              
              {storeTags.length > 0 && (
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px', marginBottom: '8px' }}>
                  {storeTags.map((tag, index) => (
                    <span
                      key={index}
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        backgroundColor: '#e6f4ff',
                        padding: '4px 8px',
                        borderRadius: '4px',
                        fontSize: '12px',
                        gap: '4px'
                      }}
                    >
                      {tag}
                      <button
                        onClick={() => handleRemoveTag(tag)}
                        disabled={disabled}
                        style={{
                          border: 'none',
                          background: 'none',
                          cursor: 'pointer',
                          padding: '0',
                          color: '#999',
                          fontSize: '14px'
                        }}
                      >
                        ×
                      </button>
                    </span>
                  ))}
                </div>
              )}
              <div style={{ fontSize: '12px', color: '#999' }}>
                标签可以帮助用户更好地发现您的模板
              </div>
            </div>

            <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
              <Button
                onClick={handlePublishStoreTemplate}
                disabled={disabled || !templateTitle.trim() || storePublishing}
                loading={storePublishing}
                type="primary"
                size="small"
              >
                {isStorePublished ? '更新商店模板' : '发布到商店'}
              </Button>

              {isStorePublished && (
                <Button
                  onClick={handleUnpublishStoreTemplate}
                  disabled={disabled || storeUnpublishing}
                  loading={storeUnpublishing}
                  type="secondary"
                  size="small"
                >
                  从商店下架
                </Button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};