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

import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import React, { useEffect, useMemo, useRef, useState } from 'react';

import { skill } from '@coze-studio/api-schema';
import {
  IconCozArrowLeft,
  IconCozDelete,
  IconCozImport,
  IconCozPlus,
} from '@coze-arch/coze-design/icons';
import {
  Button,
  Input,
  Spin,
  TextArea,
  Typography,
} from '@coze-arch/coze-design';

import {
  buildStandardSkillFiles,
  splitStandardSkillFiles,
  type StandardSkillFileDraft,
} from './standard-skill-files';

const { Text } = Typography;

const STANDARD_FILE_GROUPS = [
  { key: 'root', label: '根文件', root: '', samplePath: 'SKILL.md' },
  { key: 'scripts', label: 'scripts', root: 'scripts/', samplePath: 'scripts/run.py' },
  {
    key: 'references',
    label: 'references',
    root: 'references/',
    samplePath: 'references/guide.md',
  },
  {
    key: 'templates',
    label: 'templates',
    root: 'templates/',
    samplePath: 'templates/example.md',
  },
  { key: 'assets', label: 'assets', root: 'assets/', samplePath: 'assets/example.txt' },
] as const;

const getFileGroup = (path: string) =>
  STANDARD_FILE_GROUPS.find(group => group.root && path.startsWith(group.root))
    ?.key || 'root';

const getFileName = (path: string) => path.split('/').pop() || path;

const normalizeAssetName = (name: string) => {
  const normalized = name
    .trim()
    .replace(/\\/g, '/')
    .split('/')
    .pop()
    ?.replace(/[^a-zA-Z0-9._-]/g, '-')
    .replace(/-+/g, '-');

  return normalized || `asset-${Date.now()}.png`;
};

const ensureUniquePath = (path: string, existingPaths: string[]) => {
  if (!existingPaths.includes(path)) {
    return path;
  }

  const dotIndex = path.lastIndexOf('.');
  const base = dotIndex > -1 ? path.slice(0, dotIndex) : path;
  const ext = dotIndex > -1 ? path.slice(dotIndex) : '';

  for (let index = 2; index < 1000; index++) {
    const nextPath = `${base}-${index}${ext}`;
    if (!existingPaths.includes(nextPath)) {
      return nextPath;
    }
  }

  return `${base}-${Date.now()}${ext}`;
};

const readFileAsDataURL = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });

const isImageAsset = (path: string, content: string) =>
  path.startsWith('assets/') &&
  (content.startsWith('data:image/') || /\.(png|jpe?g|webp|gif|svg)$/i.test(path));

const createFileDraft = (path: string): StandardSkillFileDraft => ({
  path,
  content: path.startsWith('assets/') ? '' : '',
});

const SpaceSkillDetail: React.FC = () => {
  const { space_id, page_type } = useParams<{
    space_id: string;
    page_type: string;
  }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const uploadInputRef = useRef<HTMLInputElement>(null);

  const skillId = searchParams.get('skill_id') || '';
  const isEditing = page_type === 'edit';

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [skillMd, setSkillMd] = useState('');
  const [extraFiles, setExtraFiles] = useState<StandardSkillFileDraft[]>([]);
  const [activePath, setActivePath] = useState('SKILL.md');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [loadingSkill, setLoadingSkill] = useState(false);
  const [errorText, setErrorText] = useState('');

  useEffect(() => {
    if (isEditing && skillId && space_id) {
      setLoadingSkill(true);
      skill
        .GetSkill({ skill_id: skillId, space_id })
        .then(response => {
          if (response.code === 0 && response.data?.skill_info) {
            const info = response.data.skill_info;
            setName(info.name || '');
            setDescription(info.description || '');
            const standardForm = splitStandardSkillFiles({
              name: info.name || '',
              description: info.description || '',
              prompt: info.prompt || '',
              files: info.files,
            });
            setSkillMd(standardForm.skillMd);
            setExtraFiles(standardForm.extraFiles);
            setActivePath('SKILL.md');
          }
        })
        .catch(error => {
          console.error('加载技能失败:', error);
          setErrorText('加载技能失败');
        })
        .finally(() => {
          setLoadingSkill(false);
        });
    }
  }, [isEditing, skillId, space_id]);

  const allFiles = useMemo(
    () => [{ path: 'SKILL.md', content: skillMd }, ...extraFiles],
    [extraFiles, skillMd],
  );

  const activeFile = allFiles.find(file => file.path === activePath) || allFiles[0];

  const groupedFiles = useMemo(
    () =>
      STANDARD_FILE_GROUPS.map(group => ({
        ...group,
        files: allFiles
          .filter(file => getFileGroup(file.path) === group.key)
          .sort((left, right) => left.path.localeCompare(right.path)),
      })),
    [allFiles],
  );

  const imageAssets = useMemo(
    () =>
      extraFiles
        .filter(file => isImageAsset(file.path, file.content))
        .sort((left, right) => left.path.localeCompare(right.path)),
    [extraFiles],
  );

  useEffect(() => {
    if (!allFiles.some(file => file.path === activePath)) {
      setActivePath('SKILL.md');
    }
  }, [activePath, allFiles]);

  const updateExtraFile = (
    currentPath: string,
    nextFile: Partial<StandardSkillFileDraft>,
  ) => {
    setExtraFiles(files =>
      files.map(file =>
        file.path === currentPath ? { ...file, ...nextFile } : file,
      ),
    );
    if (nextFile.path && activePath === currentPath) {
      setActivePath(nextFile.path);
    }
  };

  const upsertExtraFile = (path: string, content: string) => {
    setExtraFiles(files => {
      const existing = files.find(file => file.path === path);
      if (existing) {
        return files.map(file =>
          file.path === path ? { ...file, content } : file,
        );
      }
      return [...files, { path, content }];
    });
    setActivePath(path);
  };

  const handleActiveContentChange = (content: string) => {
    if (activePath === 'SKILL.md') {
      setSkillMd(content);
      return;
    }
    updateExtraFile(activePath, { content });
  };

  const handleAddFile = (samplePath: string) => {
    if (samplePath === 'SKILL.md') {
      setActivePath('SKILL.md');
      return;
    }

    const nextPath = ensureUniquePath(
      samplePath,
      allFiles.map(file => file.path),
    );
    setExtraFiles(files => [...files, createFileDraft(nextPath)]);
    setActivePath(nextPath);
  };

  const handleRemoveFile = (path: string) => {
    if (path === 'SKILL.md') {
      return;
    }
    setExtraFiles(files => files.filter(file => file.path !== path));
    setActivePath('SKILL.md');
  };

  const handleUploadAsset = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const files = Array.from(event.target.files || []);
    if (files.length === 0) {
      return;
    }

    try {
      for (const file of files) {
        const content = await readFileAsDataURL(file);
        const assetPath = ensureUniquePath(
          `assets/${normalizeAssetName(file.name)}`,
          allFiles.map(item => item.path),
        );
        upsertExtraFile(assetPath, content);
      }
      setErrorText('');
    } catch (error) {
      console.error('上传资产失败:', error);
      setErrorText('上传资产失败');
    } finally {
      event.target.value = '';
    }
  };

  const handleSave = async () => {
    if (!name.trim() || isSubmitting || !space_id) {
      return;
    }
    setIsSubmitting(true);
    setErrorText('');
    try {
      const files = buildStandardSkillFiles({
        name,
        description,
        skillMd,
        extraFiles,
      });

      const response = isEditing
        ? await skill.UpdateSkill({
            skill_id: skillId,
            space_id,
            name,
            description: description || undefined,
            files,
          })
        : await skill.CreateSkill({
            space_id,
            name,
            description: description || undefined,
            files,
          });

      if (response.code !== 0) {
        throw new Error(response.msg || '保存失败');
      }
      navigate(-1);
    } catch (error) {
      console.error('保存失败:', error);
      setErrorText(error instanceof Error ? error.message : '保存失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loadingSkill) {
    return (
      <div className="flex justify-center items-center h-full">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full min-w-[900px] coz-mg-plus">
      <div className="flex items-center justify-between px-[24px] py-[14px] border-b border-solid coz-stroke-primary">
        <div className="flex items-center gap-[12px] min-w-0">
          <Button
            icon={<IconCozArrowLeft />}
            theme="borderless"
            onClick={() => navigate(-1)}
          />
          <div className="min-w-0">
            <h1 className="m-0 text-[18px] leading-[26px] font-semibold coz-fg-primary">
              {isEditing ? '编辑标准技能' : '创建标准技能'}
            </h1>
            <div className="text-[12px] leading-[18px] coz-fg-secondary truncate">
              {name || '未命名技能'}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-[8px]">
          {errorText ? (
            <Text className="text-[12px] text-red-500">{errorText}</Text>
          ) : null}
          <Button onClick={() => navigate(-1)}>取消</Button>
          <Button
            type="primary"
            onClick={handleSave}
            disabled={!name.trim()}
            loading={isSubmitting}
          >
            保存
          </Button>
        </div>
      </div>

      <div
        className="flex-1 min-h-0 grid"
        style={{ gridTemplateColumns: '240px minmax(360px, 1fr) 280px' }}
      >
        <aside className="border-r border-solid coz-stroke-primary flex flex-col min-h-0">
          <div className="px-[16px] py-[14px] border-b border-solid coz-stroke-primary">
            <div className="text-[13px] leading-[20px] font-medium coz-fg-primary">
              技能文件
            </div>
            <div className="mt-[4px] text-[12px] leading-[18px] coz-fg-secondary">
              {allFiles.length} 个文件
            </div>
          </div>
          <div className="flex-1 overflow-y-auto px-[10px] py-[10px]">
            {groupedFiles.map(group => (
              <div key={group.key} className="mb-[14px]">
                <div className="flex items-center justify-between px-[6px] mb-[6px]">
                  <span className="text-[11px] leading-[16px] font-medium coz-fg-tertiary">
                    {group.label}
                  </span>
                  {group.key !== 'root' ? (
                    <button
                      type="button"
                      className="border-0 bg-transparent p-0 text-[12px] cursor-pointer coz-fg-hglt"
                      onClick={() => handleAddFile(group.samplePath)}
                    >
                      添加
                    </button>
                  ) : null}
                </div>
                <div className="flex flex-col gap-[4px]">
                  {group.files.length ? (
                    group.files.map(file => {
                      const selected = file.path === activePath;
                      return (
                        <button
                          key={file.path}
                          type="button"
                          className={`w-full min-h-[32px] px-[8px] py-[6px] rounded-[6px] border-0 text-left cursor-pointer flex items-center justify-between gap-[8px] ${
                            selected
                              ? 'coz-mg-hglt coz-fg-hglt'
                              : 'bg-transparent coz-fg-primary hover:coz-mg-secondary'
                          }`}
                          onClick={() => setActivePath(file.path)}
                        >
                          <span className="truncate text-[12px]">
                            {getFileName(file.path)}
                          </span>
                          {file.path === 'SKILL.md' ? (
                            <span className="text-[10px] coz-fg-tertiary">
                              必需
                            </span>
                          ) : null}
                        </button>
                      );
                    })
                  ) : (
                    <div className="px-[8px] py-[6px] text-[12px] coz-fg-tertiary">
                      暂无文件
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </aside>

        <main className="flex flex-col min-h-0">
          <div className="px-[20px] py-[14px] border-b border-solid coz-stroke-primary">
            <div className="flex items-center justify-between gap-[12px]">
              <div className="min-w-0">
                <div className="text-[13px] leading-[20px] font-medium coz-fg-primary truncate">
                  {activeFile.path}
                </div>
                <div className="mt-[2px] text-[12px] leading-[18px] coz-fg-secondary">
                  {activePath === 'SKILL.md'
                    ? '技能入口文件'
                    : getFileGroup(activePath)}
                </div>
              </div>
              {activePath !== 'SKILL.md' ? (
                <Button
                  size="small"
                  color="secondary"
                  icon={<IconCozDelete />}
                  onClick={() => handleRemoveFile(activePath)}
                >
                  删除文件
                </Button>
              ) : null}
            </div>
          </div>
          <div className="flex-1 min-h-0 overflow-y-auto p-[20px]">
            {activePath !== 'SKILL.md' ? (
              <div className="mb-[12px]">
                <Text className="text-[12px] coz-fg-secondary block mb-[6px]">
                  文件路径
                </Text>
                <Input
                  value={activePath}
                  onChange={path => updateExtraFile(activePath, { path })}
                  placeholder="scripts/run.py 或 assets/logo.png"
                />
              </div>
            ) : null}

            {isImageAsset(activeFile.path, activeFile.content) &&
            activeFile.content.startsWith('data:image/') ? (
              <div className="mb-[12px] rounded-[8px] border border-solid coz-stroke-primary p-[12px] coz-mg-card">
                <img
                  src={activeFile.content}
                  alt=""
                  className="max-h-[180px] max-w-full object-contain rounded-[6px]"
                  draggable={false}
                />
              </div>
            ) : null}

            <TextArea
              value={activeFile.content}
              onChange={handleActiveContentChange}
              autosize={{ minRows: 22, maxRows: 36 }}
              placeholder={
                activePath === 'SKILL.md'
                  ? '---\nname: pdf-tools\ndescription: Handle PDF files\n---\n# PDF Tools\n\n## Workflow\n1. Read inputs\n2. Run scripts/run.py\n3. Return output artifacts'
                  : '文件内容'
              }
            />
          </div>
        </main>

        <aside className="border-l border-solid coz-stroke-primary flex flex-col min-h-0">
          <div className="px-[16px] py-[14px] border-b border-solid coz-stroke-primary">
            <div className="text-[13px] leading-[20px] font-medium coz-fg-primary">
              技能信息
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-[16px] flex flex-col gap-[18px]">
            <div>
              <Text className="text-[12px] font-medium coz-fg-primary block mb-[6px]">
                技能名称
              </Text>
              <Input
                value={name}
                onChange={val => setName(val)}
                placeholder="例如：财务票据核验"
                maxLength={50}
              />
            </div>

            <div>
              <Text className="text-[12px] font-medium coz-fg-primary block mb-[6px]">
                简短描述
              </Text>
              <Input
                value={description}
                onChange={val => setDescription(val)}
                placeholder="一句话描述触发场景"
              />
            </div>

            <div className="grid grid-cols-2 gap-[8px]">
              <div className="rounded-[8px] border border-solid coz-stroke-primary coz-mg-card px-[12px] py-[10px]">
                <div className="text-[11px] coz-fg-tertiary">文件</div>
                <div className="mt-[3px] text-[18px] leading-[24px] font-semibold coz-fg-primary">
                  {allFiles.length}
                </div>
              </div>
              <div className="rounded-[8px] border border-solid coz-stroke-primary coz-mg-card px-[12px] py-[10px]">
                <div className="text-[11px] coz-fg-tertiary">图片资产</div>
                <div className="mt-[3px] text-[18px] leading-[24px] font-semibold coz-fg-primary">
                  {imageAssets.length}
                </div>
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-[8px]">
                <Text className="text-[12px] font-medium coz-fg-primary">
                  assets
                </Text>
                <Button
                  size="small"
                  color="secondary"
                  icon={<IconCozImport />}
                  onClick={() => uploadInputRef.current?.click()}
                >
                  上传图片
                </Button>
              </div>
              <input
                ref={uploadInputRef}
                type="file"
                accept="image/*"
                multiple
                className="hidden"
                onChange={handleUploadAsset}
              />
              <div className="flex flex-col gap-[8px]">
                {imageAssets.length ? (
                  imageAssets.map(asset => (
                    <button
                      key={asset.path}
                      type="button"
                      className="w-full border border-solid coz-stroke-primary rounded-[8px] coz-mg-card p-[8px] flex items-center gap-[8px] cursor-pointer text-left"
                      onClick={() => setActivePath(asset.path)}
                    >
                      {asset.content.startsWith('data:image/') ? (
                        <img
                          src={asset.content}
                          alt=""
                          className="w-[36px] h-[36px] object-cover rounded-[6px] flex-shrink-0"
                          draggable={false}
                        />
                      ) : (
                        <div className="w-[36px] h-[36px] rounded-[6px] coz-mg-secondary flex-shrink-0" />
                      )}
                      <span className="min-w-0 flex-1 text-[12px] coz-fg-primary truncate">
                        {asset.path}
                      </span>
                    </button>
                  ))
                ) : (
                  <div className="rounded-[8px] border border-dashed coz-stroke-primary px-[12px] py-[18px] text-[12px] coz-fg-tertiary text-center">
                    暂无图片资产
                  </div>
                )}
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-[8px]">
                <Text className="text-[12px] font-medium coz-fg-primary">
                  快速添加
                </Text>
              </div>
              <div className="grid grid-cols-2 gap-[8px]">
                {STANDARD_FILE_GROUPS.filter(group => group.key !== 'root').map(
                  group => (
                    <Button
                      key={group.key}
                      size="small"
                      color="secondary"
                      icon={<IconCozPlus />}
                      onClick={() => handleAddFile(group.samplePath)}
                    >
                      {group.label}
                    </Button>
                  ),
                )}
              </div>
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
};

export default SpaceSkillDetail;
