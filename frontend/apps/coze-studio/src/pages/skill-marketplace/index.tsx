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

import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { skill } from '@coze-studio/api-schema';
import type { SkillInfo } from '@coze-studio/api-schema/idl/skill/skill';
import { Modal, Spin, Toast } from '@coze-arch/coze-design';
import {
  IconCozDownload,
  IconCozEye,
  IconCozFolder,
  IconCozImport,
  IconCozMagnifier,
  IconCozPeople,
  IconCozPlugin,
  IconCozStarFill,
  IconCozStore,
} from '@coze-arch/coze-design/icons';
import { useSpaceStore } from '@coze-foundation/space-ui-adapter';

import {
  agentAppApi,
  type AgentAppItem,
} from '../space-skill/agent-app-api';
import ReviewQueue from './review-queue';
import styles from './index.module.less';

const SKILL_PUBLISH_SCOPE = {
  GLOBAL: 3,
} as const;

interface SkillAssetSummary {
  count?: number;
  image_paths?: string[];
}

type SkillWithAssetSummary = SkillInfo & {
  asset_summary?: SkillAssetSummary;
};

const getSkillFiles = (item: SkillInfo) => item.files ?? {};

const isStandardSkill = (item: SkillInfo) =>
  Boolean(getSkillFiles(item)['SKILL.md']);

const getSkillFileCount = (item: SkillInfo) =>
  Object.keys(getSkillFiles(item)).length;

const getAssetSummary = (item: SkillInfo): SkillAssetSummary => {
  const summary = (item as SkillWithAssetSummary).asset_summary;
  if (summary) {
    return summary;
  }

  const imagePaths = Object.entries(getSkillFiles(item))
    .filter(([path, content]) => {
      if (!path.startsWith('assets/')) {
        return false;
      }
      return (
        content.startsWith('data:image/') ||
        /\.(png|jpe?g|webp|gif|svg)$/i.test(path)
      );
    })
    .map(([path]) => path)
    .sort();

  return {
    count: Object.keys(getSkillFiles(item)).filter(path =>
      path.startsWith('assets/'),
    ).length,
    image_paths: imagePaths,
  };
};

const getSkillCover = (item: SkillInfo) => {
  if (
    item.icon_uri &&
    (/^(https?:)?\/\//.test(item.icon_uri) ||
      item.icon_uri.startsWith('/') ||
      item.icon_uri.startsWith('data:image/'))
  ) {
    return item.icon_uri;
  }

  const firstImagePath = getAssetSummary(item).image_paths?.[0];
  const content = firstImagePath ? getSkillFiles(item)[firstImagePath] : '';
  return content?.startsWith('data:image/') ? content : '';
};

const getSkillMetadataBadges = (item: SkillInfo) => {
  const metadata = item.metadata;
  if (!metadata) {
    return [];
  }

  return [
    metadata.category,
    metadata.version ? `v${metadata.version}` : '',
    metadata.tags?.[0],
    metadata.platforms?.[0],
  ].filter(Boolean) as string[];
};

const STANDARD_FILE_ROOTS = [
  { prefix: 'SKILL.md', label: 'SKILL.md' },
  { prefix: 'scripts/', label: '脚本' },
  { prefix: 'references/', label: '参考' },
  { prefix: 'templates/', label: '模板' },
  { prefix: 'assets/', label: '资产' },
];

const getSkillRootLabels = (item: SkillInfo) => {
  const paths = Object.keys(getSkillFiles(item));
  return STANDARD_FILE_ROOTS.filter(root =>
    paths.some(path =>
      root.prefix.endsWith('/')
        ? path.startsWith(root.prefix)
        : path === root.prefix,
    ),
  ).map(root => root.label);
};

const formatDate = (timestamp?: number) => {
  if (!timestamp) {
    return '-';
  }
  return new Date(timestamp).toLocaleDateString('zh-CN');
};

const getSkillMdPreview = (item: SkillInfo) => {
  const content = getSkillFiles(item)['SKILL.md'] || item.prompt || '';
  if (content.length <= 1600) {
    return content;
  }
  return `${content.slice(0, 1600)}\n...`;
};

const getCategories = (items: SkillInfo[]) => {
  const categories = items
    .map(item => item.metadata?.category || '通用')
    .filter(Boolean);
  return Array.from(new Set(categories)).sort();
};

const SKILL_TONES = [
  'linear-gradient(150deg, rgb(94,170,255), rgb(53,138,255))',
  'linear-gradient(150deg, rgb(78,217,196), rgb(0,178,120))',
  'linear-gradient(150deg, rgb(186,140,255), rgb(167,0,250))',
  'linear-gradient(150deg, rgb(255,184,122), rgb(255,138,61))',
];

const getSkillTone = (item: SkillInfo) => {
  const source = item.skill_id || item.name || '';
  const hash = Array.from(source).reduce(
    (sum, char) => sum + char.charCodeAt(0),
    0,
  );
  return SKILL_TONES[hash % SKILL_TONES.length];
};

const SkillCard: React.FC<{
  item: SkillInfo;
  installing: boolean;
  onOpen: (item: SkillInfo) => void;
  onInstall: (item: SkillInfo) => void;
}> = ({ item, installing, onOpen, onInstall }) => {
  const cover = getSkillCover(item);
  const fileCount = getSkillFileCount(item);
  const assetSummary = getAssetSummary(item);
  const metadataBadges = getSkillMetadataBadges(item);
  const rootLabels = getSkillRootLabels(item);

  return (
    <article className={styles.scard} data-testid="skill-marketplace-card">
      <div className={styles.scardHd}>
        <div
          className={styles.scardIcon}
          style={{ background: getSkillTone(item) }}
        >
          <div className={styles.scardIconInner}>
            {cover ? (
              <img
                src={cover}
                alt=""
                draggable={false}
                className={styles.scardCover}
              />
            ) : (
              <IconCozPlugin />
            )}
          </div>
        </div>
        <div className={styles.scardHeaderText}>
          <div className={styles.scardNameRow}>
            <h3 className={styles.scardName} title={item.name}>
              {item.name || '未命名技能'}
            </h3>
            <span className={`${styles.badge} ${styles.badgeGlobal}`}>
              全局
            </span>
          </div>
          <p className={styles.scardDesc} title={item.description}>
            {item.description || '暂无描述'}
          </p>
        </div>
      </div>

      <div className={styles.scardTags}>
        <span className={`${styles.tag} ${styles.tagPackage}`}>
          <IconCozPlugin />
          标准技能包
        </span>
        {rootLabels.map(label => (
          <span key={label} className={styles.tag}>
            <IconCozFolder />
            {label}
          </span>
        ))}
        {metadataBadges.map(badge => (
          <span key={badge} className={styles.tag} title={badge}>
            {badge}
          </span>
        ))}
      </div>

      <div className={styles.scardMeta}>
        {[
          ['文件', fileCount],
          ['资产', assetSummary.count ?? 0],
          [
            '版本',
            item.published_version ? `v${item.published_version}` : '-',
            true,
          ],
        ].map(([label, value, highlight]) => (
          <div key={label as string}>
            <div className={styles.metaKey}>{label}</div>
            <div
              className={`${styles.metaValue} ${
                highlight ? styles.metaValueVersion : ''
              }`}
            >
              {value}
            </div>
          </div>
        ))}
      </div>

      <div className={styles.scardFooter}>
        <button
          type="button"
          className={styles.detailButton}
          onClick={() => onOpen(item)}
        >
          <IconCozEye />
          查看详情
        </button>
        <div className={styles.spacer} />
        <span className={styles.publishDate}>
          发布于 {formatDate(item.published_at)}
        </span>
        <button
          type="button"
          className={styles.installButton}
          disabled={installing}
          onClick={() => onInstall(item)}
        >
          <IconCozImport />
          {installing ? '安装中' : '安装'}
        </button>
      </div>
    </article>
  );
};

const SkillMarketplacePage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [items, setItems] = useState<SkillInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [errorText, setErrorText] = useState('');
  const [keyword, setKeyword] = useState('');
  const [activeCategory, setActiveCategory] = useState('全部');
  const [detailItem, setDetailItem] = useState<SkillInfo | null>(null);
  const [installingSkillId, setInstallingSkillId] = useState('');
  const [view, setView] = useState<'market' | 'review' | 'employee'>('market');
  const [employees, setEmployees] = useState<AgentAppItem[]>([]);
  const [employeesLoading, setEmployeesLoading] = useState(false);
  const [employeesError, setEmployeesError] = useState('');
  const [recruitingId, setRecruitingId] = useState('');

  useEffect(() => {
    let canceled = false;
    const fetchSkills = async () => {
      try {
        setLoading(true);
        setErrorText('');
        const response = await skill.SuperAgentMarketplaceListSkills({
          scope: SKILL_PUBLISH_SCOPE.GLOBAL,
          page: 1,
          page_size: 200,
        });
        if (canceled) {
          return;
        }
        if (response.code !== 0) {
          throw new Error(response.msg || '全局技能商城加载失败');
        }
        setItems((response.data?.skill_list || []).filter(isStandardSkill));
      } catch (error) {
        if (!canceled) {
          setErrorText(
            error instanceof Error ? error.message : '全局技能商城加载失败',
          );
        }
      } finally {
        if (!canceled) {
          setLoading(false);
        }
      }
    };

    fetchSkills();
    return () => {
      canceled = true;
    };
  }, []);

  // 入口（如 store-chat-page）跳转到 /explore 时不带 space_id，
  // 这会让审核队列拿不到空间上下文。兜底取用户当前所在空间。
  const fallbackSpaceId = useSpaceStore(s => s.space?.id);
  const targetSpaceId =
    searchParams.get('space_id') || fallbackSpaceId || '';

  // 虚拟员工商城：进入「虚拟员工」分页时拉取本空间已发布的 agent_app 产品。
  useEffect(() => {
    if (view !== 'employee') {
      return;
    }
    // 全局虚拟员工对所有人可见，列表不要求先进入空间（招聘时才需要落地空间）。
    let canceled = false;
    const fetchEmployees = async () => {
      try {
        setEmployeesLoading(true);
        setEmployeesError('');
        const res = await agentAppApi.listAgentApps({
          space_id: targetSpaceId,
          page: 1,
          page_size: 100,
        });
        if (canceled) {
          return;
        }
        if (res.code !== 0) {
          throw new Error(res.msg || '虚拟员工列表加载失败');
        }
        setEmployees(res.data?.products || []);
      } catch (error) {
        if (!canceled) {
          setEmployeesError(
            error instanceof Error ? error.message : '虚拟员工列表加载失败',
          );
        }
      } finally {
        if (!canceled) {
          setEmployeesLoading(false);
        }
      }
    };
    fetchEmployees();
    return () => {
      canceled = true;
    };
  }, [view, targetSpaceId]);

  // 招聘并进入独立聊天界面：招聘生成只读影子智能体，再跳转到 /employee-chat。
  const handleChatEmployee = async (employee: AgentAppItem) => {
    if (!targetSpaceId) {
      Toast.warning('请先进入一个工作空间，再招聘虚拟员工');
      navigate('/space');
      return;
    }
    try {
      setRecruitingId(employee.product_id);
      const res = await agentAppApi.recruit({
        product_id: employee.product_id,
        space_id: targetSpaceId,
      });
      if (res.code !== 0 || !res.data?.shadow_agent_id) {
        throw new Error(res.msg || '招聘失败');
      }
      // 进入独立员工聊天界面：复用超级体运行管线（标准 agent-run），对话对象为
      // 招聘生成的只读实例体。带 employeeChat=1 以渲染纯聊天（隐藏编辑面板）。
      navigate(
        `/space/${targetSpaceId}/bot/${res.data.shadow_agent_id}/arrange?employeeChat=1&name=${encodeURIComponent(
          employee.name,
        )}`,
      );
    } catch (error) {
      Toast.error(error instanceof Error ? error.message : '招聘失败，请重试');
    } finally {
      setRecruitingId('');
    }
  };
  const categories = useMemo(() => getCategories(items), [items]);
  const filteredItems = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    return items.filter(item => {
      const category = item.metadata?.category || '通用';
      const categoryMatched =
        activeCategory === '全部' || category === activeCategory;
      if (!categoryMatched) {
        return false;
      }
      if (!normalizedKeyword) {
        return true;
      }
      return [
        item.name,
        item.description,
        category,
        item.metadata?.tags?.join(' '),
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
        .includes(normalizedKeyword);
    });
  }, [activeCategory, items, keyword]);

  const stats = useMemo(
    () => [
      {
        label: '全局标准技能',
        value: items.length,
        icon: <IconCozStore />,
        tone: 'rgba(46, 115, 255, 0.10)',
        color: 'rgb(46, 115, 255)',
      },
      {
        label: '技能分类',
        value: categories.length,
        icon: <IconCozFolder />,
        tone: 'rgba(139, 92, 246, 0.10)',
        color: 'rgb(124, 58, 237)',
      },
      {
        label: '公司可安装',
        value: items.length,
        icon: <IconCozPeople />,
        tone: 'rgba(16, 185, 129, 0.12)',
        color: 'rgb(5, 150, 105)',
      },
      {
        label: '包含资产',
        value: items.reduce(
          (sum, item) => sum + (getAssetSummary(item).count ?? 0),
          0,
        ),
        icon: <IconCozDownload />,
        tone: 'rgba(6, 182, 212, 0.12)',
        color: 'rgb(8, 145, 178)',
      },
    ],
    [categories.length, items],
  );

  const handleInstall = (item: SkillInfo) => {
    if (!targetSpaceId) {
      Toast.warning('请先进入一个工作空间，再安装全局技能');
      navigate('/space');
      return;
    }

    Modal.confirm({
      title: '安装全局技能',
      content: `将「${item.name}」安装到当前空间，系统会生成一份可编辑的私有副本。`,
      okText: '安装',
      cancelText: '取消',
      onOk: async () => {
        try {
          setInstallingSkillId(item.skill_id);
          const response = await skill.SuperAgentInstallMarketplaceSkill({
            skill_id: item.skill_id,
            space_id: targetSpaceId,
          });
          if (response.code !== 0) {
            throw new Error(response.msg || '安装失败');
          }
          Toast.success('技能已安装到当前空间');
        } catch (error) {
          Toast.error(error instanceof Error ? error.message : '安装失败');
        } finally {
          setInstallingSkillId('');
        }
      },
    });
  };

  const emptyDescription = keyword
    ? '没有找到匹配的标准技能'
    : '全局技能商城还没有发布标准技能';
  const resultCountText = `${filteredItems.length} 个技能包`;

  return (
    <div className={styles.page}>
      <div className={styles.scroll}>
        <div className={styles.wrap} data-testid="skill-marketplace-wrap">
          <section className={styles.hero} data-testid="skill-marketplace-hero">
            <div className={styles.heroInner}>
              <span className={styles.heroPill}>
                <IconCozStarFill />
                全公司级资源
              </span>
              <h1>商店</h1>
              <p>
                统一浏览与获取全公司级资源：技能商城提供标准技能包，虚拟员工可一键招聘对话。
              </p>
              <div className={styles.heroActions}>
                <button
                  type="button"
                  className={styles.primaryButton}
                  onClick={() => navigate('/space')}
                >
                  <IconCozFolder />
                  进入空间管理
                </button>
                <div className={styles.segmented}>
                  <button
                    type="button"
                    className={
                      view === 'market' ? styles.segmentedActive : undefined
                    }
                    onClick={() => setView('market')}
                  >
                    技能商城
                  </button>
                  <button
                    type="button"
                    className={
                      view === 'employee' ? styles.segmentedActive : undefined
                    }
                    onClick={() => setView('employee')}
                  >
                    员工商城
                  </button>
                  <button
                    type="button"
                    className={
                      view === 'review' ? styles.segmentedActive : undefined
                    }
                    onClick={() => setView('review')}
                  >
                    待审核技能
                  </button>
                </div>
              </div>
            </div>
            <div className={styles.heroDecor}>
              {stats.map((item, index) => (
                <div
                  key={item.label}
                  className={`${styles.decorTile} ${
                    styles[`decorTile${index + 1}` as keyof typeof styles]
                  }`}
                >
                  {item.icon}
                </div>
              ))}
            </div>
          </section>

          <section className={styles.stats}>
            {stats.map(item => (
              <div key={item.label} className={styles.stat}>
                <div
                  className={styles.statIcon}
                  style={{ background: item.tone, color: item.color }}
                >
                  {item.icon}
                </div>
                <div className={styles.statText}>
                  <span className={styles.statLabel}>{item.label}</span>
                  <span className={styles.statNumber}>{item.value}</span>
                </div>
              </div>
            ))}
          </section>

          {view === 'review' ? (
            <ReviewQueue spaceId={targetSpaceId} />
          ) : view === 'employee' ? (
            <div className={styles.marketBody}>
              {employeesError && !employeesLoading ? (
                <div className={styles.empty}>
                  <div className={styles.emptyIcon}>
                    <IconCozPeople />
                  </div>
                  <div className={styles.emptyText}>{employeesError}</div>
                </div>
              ) : employeesLoading ? (
                <div className={styles.loading}>
                  <Spin size="large" />
                </div>
              ) : employees.length === 0 ? (
                <div className={styles.empty}>
                  <div className={styles.emptyIcon}>
                    <IconCozPeople />
                  </div>
                  <div className={styles.emptyText}>
                    本空间还没有虚拟员工。在超级智能体里「发布为虚拟员工」后即可在此招聘。
                  </div>
                </div>
              ) : (
                <div className={styles.grid}>
                  {employees.map(emp => (
                    <div
                      key={emp.product_id}
                      style={{
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 10,
                        padding: 18,
                        borderRadius: 14,
                        border: '1px solid rgba(28, 31, 35, 0.08)',
                        background: '#fff',
                        boxShadow: '0 1px 2px rgba(28, 31, 35, 0.04)',
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                        <div
                          style={{
                            width: 44,
                            height: 44,
                            borderRadius: 12,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: 18,
                            fontWeight: 600,
                            color: '#fff',
                            background:
                              'linear-gradient(135deg, #7c5cff 0%, #4e8bff 100%)',
                            flex: '0 0 44px',
                          }}
                        >
                          {(emp.name || '员').slice(0, 1)}
                        </div>
                        <div style={{ minWidth: 0 }}>
                          <div
                            style={{
                              fontSize: 15,
                              fontWeight: 600,
                              color: '#1d2129',
                              whiteSpace: 'nowrap',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                            }}
                          >
                            {emp.name}
                          </div>
                          <div style={{ fontSize: 12, color: '#86909c' }}>
                            虚拟员工
                          </div>
                        </div>
                      </div>
                      <div
                        style={{
                          fontSize: 13,
                          lineHeight: '20px',
                          color: '#4e5969',
                          minHeight: 40,
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}
                      >
                        {emp.description || '暂无描述'}
                      </div>
                      <button
                        type="button"
                        className={styles.primaryButton}
                        style={{ alignSelf: 'flex-start' }}
                        disabled={recruitingId === emp.product_id}
                        onClick={() => handleChatEmployee(emp)}
                      >
                        {recruitingId === emp.product_id ? '进入中…' : '对话'}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <>
          <section className={styles.filterBar}>
            <div className={styles.chips}>
              {['全部', ...categories].map(category => (
                <button
                  key={category}
                  type="button"
                  className={`${styles.chip} ${
                    category === activeCategory ? styles.chipActive : ''
                  }`}
                  onClick={() => setActiveCategory(category)}
                >
                  {category}
                </button>
              ))}
            </div>
            <span className={styles.resultCount}>{resultCountText}</span>
            <div className={styles.spacer} />
            <label className={styles.searchBox}>
              <IconCozMagnifier />
              <input
                placeholder="搜索技能名称或描述"
                value={keyword}
                onChange={event => setKeyword(event.target.value)}
              />
            </label>
          </section>

          <div className={styles.marketBody}>
            {errorText && !loading ? (
              <div className={styles.empty}>
                <div className={styles.emptyIcon}>
                  <IconCozStore />
                </div>
                <div className={styles.emptyText}>{errorText}</div>
              </div>
            ) : loading ? (
              <div className={styles.loading}>
                <Spin size="large" />
              </div>
            ) : filteredItems.length === 0 ? (
              <div className={styles.empty}>
                <div className={styles.emptyIcon}>
                  <IconCozStore />
                </div>
                <div className={styles.emptyText}>{emptyDescription}</div>
              </div>
            ) : (
              <div className={styles.grid}>
                {filteredItems.map(item => (
                  <SkillCard
                    key={item.skill_id}
                    item={item}
                    installing={installingSkillId === item.skill_id}
                    onOpen={setDetailItem}
                    onInstall={handleInstall}
                  />
                ))}
              </div>
            )}
          </div>
            </>
          )}
        </div>
      </div>

      <Modal
        visible={Boolean(detailItem)}
        title={detailItem?.name || '技能详情'}
        width={760}
        onCancel={() => setDetailItem(null)}
        footer={null}
      >
        {detailItem ? (
          <div className="max-h-[70vh] overflow-y-auto pr-[4px]">
            <p className="mt-0 text-[13px] leading-[22px] coz-fg-secondary">
              {detailItem.description || '暂无描述'}
            </p>
            <div className="grid grid-cols-4 gap-[10px] my-[16px]">
              {[
                ['分类', detailItem.metadata?.category || '通用'],
                [
                  '版本',
                  detailItem.metadata?.version ||
                    String(
                      detailItem.published_version || detailItem.version || '-',
                    ),
                ],
                ['文件', String(getSkillFileCount(detailItem))],
                ['资产', String(getAssetSummary(detailItem).count ?? 0)],
              ].map(([label, value]) => (
                <div
                  key={label}
                  className="rounded-[8px] border border-solid coz-stroke-primary px-[12px] py-[10px]"
                >
                  <div className="text-[12px] coz-fg-tertiary">{label}</div>
                  <div className="mt-[4px] text-[13px] coz-fg-primary truncate">
                    {value}
                  </div>
                </div>
              ))}
            </div>
            <div className="mb-[16px]">
              <div className="mb-[8px] text-[13px] font-medium coz-fg-primary">
                标准文件
              </div>
              <div className="flex flex-wrap gap-[6px]">
                {Object.keys(getSkillFiles(detailItem))
                  .sort()
                  .map(path => (
                    <span
                      key={path}
                      className="px-[8px] py-[4px] rounded-[4px] text-[12px] coz-mg-secondary coz-fg-secondary"
                    >
                      {path}
                    </span>
                  ))}
              </div>
            </div>
            <div>
              <div className="mb-[8px] text-[13px] font-medium coz-fg-primary">
                SKILL.md 预览
              </div>
              <pre className="m-0 max-h-[320px] overflow-auto rounded-[8px] border border-solid coz-stroke-primary coz-mg-secondary p-[12px] text-[12px] leading-[18px] whitespace-pre-wrap coz-fg-primary">
                {getSkillMdPreview(detailItem) || '暂无 SKILL.md 内容'}
              </pre>
            </div>
          </div>
        ) : null}
      </Modal>
    </div>
  );
};

export default SkillMarketplacePage;
