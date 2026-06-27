/* eslint-disable max-lines-per-function */
/* eslint-disable complexity */
/* eslint-disable max-lines -- large library page: folder sections + batch-add UI layered on top of the existing filters/grid */
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

import {
  forwardRef,
  useImperativeHandle,
  useState,
  useRef,
  useCallback,
  useEffect,
  useMemo,
} from 'react';

import classNames from 'classnames';
import { useInfiniteScroll } from 'ahooks';
import {
  GridList,
  GridItem,
} from '../../../../entry-adapter/src/pages/falcon/components/gridList';
import { I18n } from '@coze-arch/i18n';
import {
  Table,
  Select,
  Search,
  Layout,
  Space,
  Spin,
  Toast,
  Checkbox,
} from '@coze-arch/coze-design';
import { renderHtmlTitle } from '@coze-arch/bot-utils';
import { EVENT_NAMES, sendTeaEvent } from '@coze-arch/bot-tea';
import {
  ResType,
  // type ResType,
  type LibraryResourceListRequest,
  type ResourceInfo,
} from '@coze-arch/bot-api/plugin_develop';
import { PluginDevelopApi } from '@coze-arch/bot-api';

import { highlightFilterStyle } from '@/constants/filter-style';
import { WorkspaceEmpty } from '@/components/workspace-empty';

import { type ListData, type BaseLibraryPageProps } from './types';
import {
  useFolderManagement,
  type FolderInfo,
} from './hooks/use-folder-management';
import { useGetColumns } from './hooks/use-columns';
import { useCachedQueryParams } from './hooks/use-cached-query-params';
import {
  eventLibraryType,
  getScopeOptions,
  getStatusOptions,
  LIBRARY_PAGE_SIZE,
} from './consts';
import { LibraryHeader } from './components/library-header';
import { FolderCard } from './components/folder-card';
import { FolderBatchBar } from './components/folder-batch-bar';

import s from './index.module.less';

// Resource type used by the folder backend to scope workflow statistics
const FOLDER_WORKFLOW_RESOURCE_TYPE = 2;

export { useDatabaseConfig } from './hooks/use-entity-configs/use-database-config';
export { usePluginConfig } from './hooks/use-entity-configs/use-plugin-config';
export { useWorkflowConfig } from './hooks/use-entity-configs/use-workflow-config';
export { usePromptConfig } from './hooks/use-entity-configs/use-prompt-config';
export { useKnowledgeConfig } from './hooks/use-entity-configs/use-knowledge-config';
export { useStrategyConfig } from './hooks/use-entity-configs/use-strategy-config';
export { type LibraryEntityConfig } from './types';
export { type UseEntityConfigHook } from './hooks/use-entity-configs/types';
import { GridLibraryItem } from './components/grid-library-item';

export const BaseLibraryPage = forwardRef<
  { reloadList: () => void },
  BaseLibraryPageProps
>(
  // eslint-disable-next-line @coze-arch/max-line-per-function -- Complex library page component
  ({ spaceId, sourceType, isPersonalSpace = true, entityConfigs }, ref) => {
    const { params, setParams, resetParams, hasFilter, ready } =
      useCachedQueryParams({
        spaceId,
      });

    const [layoutType, setLayoutType] = useState('grid');
    const scrollRef = useRef<HTMLDivElement>(null);
    const defaultGridItemWidth = 318;
    const [gridItemWidth, setGridItemWidth] = useState(defaultGridItemWidth);

    const resType = Number(sourceType);
    // const restTypeFilter =
    //   resType === ResType.Knowledge
    //     ? [resType, params.res_type_filter?.[1]]
    //     : [resType];

    const listResp = useInfiniteScroll<ListData>(
      async prev => {
        if (!ready) {
          return {
            list: [],
            nextCursorId: undefined,
            hasMore: false,
          };
        }
        const typeFilter = Number(sourceType);
        // Allow business to customize request parameters
        const resp = await PluginDevelopApi.LibraryResourceList(
          entityConfigs.reduce<LibraryResourceListRequest>(
            (res, config) => config.parseParams?.(res) ?? res,
            {
              ...params,
              res_type_filter:
                typeFilter === ResType.Knowledge
                  ? [typeFilter, params.res_type_filter?.[1] ?? -1]
                  : [typeFilter],
              cursor: prev?.nextCursorId,
              space_id: spaceId,
              size: LIBRARY_PAGE_SIZE,
            },
          ),
        );
        return {
          list: resp?.resource_list || [],
          nextCursorId: resp?.cursor,
          hasMore: !!resp?.has_more,
        };
      },
      {
        reloadDeps: [params, spaceId, sourceType],
      },
    );

    useImperativeHandle(ref, () => ({
      reloadList: listResp.reload,
    }));

    // Workflow tab folder category (display-only grouping, single level)
    const folderEnabled = resType === ResType.Workflow;
    const {
      folders,
      refreshFolders,
      createFolder,
      renameFolder,
      deleteFolder,
      moveResourcesToFolder,
    } = useFolderManagement({
      spaceId,
      resourceType: FOLDER_WORKFLOW_RESOURCE_TYPE,
      onSuccess: () => {
        listResp.reload();
      },
    });
    const [currentFolderId, setCurrentFolderId] = useState<string | null>(null);
    const [batchFolderId, setBatchFolderId] = useState<string | null>(null);
    const [selectedResIds, setSelectedResIds] = useState<Set<string>>(
      new Set(),
    );
    const [moving, setMoving] = useState(false);
    const batchMode = batchFolderId !== null;

    const exitBatch = useCallback(() => {
      setBatchFolderId(null);
      setSelectedResIds(new Set());
    }, []);

    const startBatch = useCallback((folderId: string) => {
      setBatchFolderId(folderId);
      setSelectedResIds(new Set());
    }, []);

    const toggleSelect = useCallback((resId?: string) => {
      if (!resId) {
        return;
      }
      setSelectedResIds(prev => {
        const next = new Set(prev);
        if (next.has(resId)) {
          next.delete(resId);
        } else {
          next.add(resId);
        }
        return next;
      });
    }, []);

    // Reset drill-in + batch state when switching tabs
    useEffect(() => {
      setCurrentFolderId(null);
      exitBatch();
    }, [sourceType, exitBatch]);

    // Drilling into a folder cancels any in-progress batch add
    useEffect(() => {
      if (currentFolderId) {
        exitBatch();
      }
    }, [currentFolderId, exitBatch]);

    // Keep folder resource grouping in sync after any list reload (e.g. after a move)
    useEffect(() => {
      if (folderEnabled) {
        refreshFolders();
      }
      // eslint-disable-next-line react-hooks/exhaustive-deps -- intentionally resync on list change
    }, [folderEnabled, listResp.data]);

    const currentFolder = useMemo(
      () => folders.find(f => f.id === currentFolderId) ?? null,
      [folders, currentFolderId],
    );

    const batchFolder = useMemo(
      () => folders.find(f => f.id === batchFolderId) ?? null,
      [folders, batchFolderId],
    );

    const handleConfirmBatch = useCallback(async () => {
      if (!batchFolderId || selectedResIds.size === 0) {
        return;
      }
      setMoving(true);
      try {
        await moveResourcesToFolder(
          batchFolderId,
          Array.from(selectedResIds),
          FOLDER_WORKFLOW_RESOURCE_TYPE,
        );
        Toast.success('已加入分类');
        exitBatch();
      } catch (error) {
        Toast.error('加入分类失败');
      } finally {
        setMoving(false);
      }
    }, [batchFolderId, selectedResIds, moveResourcesToFolder, exitBatch]);

    // Map of resource id -> owning folder, used for grouping / filtering
    const resourceFolderMap = useMemo(() => {
      const map = new Map<string, FolderInfo>();
      folders.forEach(folder => {
        (folder.resource_ids ?? []).forEach(resId => {
          map.set(resId as string, folder);
        });
      });
      return map;
    }, [folders]);

    const rawList = listResp.data?.list ?? [];
    const visibleList = useMemo(() => {
      if (!folderEnabled) {
        return rawList;
      }
      if (currentFolderId) {
        // Second level: only workflows belonging to the current folder
        return rawList.filter(
          item =>
            item.res_id !== undefined &&
            resourceFolderMap.get(item.res_id)?.id === currentFolderId,
        );
      }
      // First level: only uncategorized workflows (folder cards rendered above)
      return rawList.filter(
        item =>
          item.res_id === undefined || !resourceFolderMap.has(item.res_id),
      );
    }, [folderEnabled, currentFolderId, rawList, resourceFolderMap]);

    // Show folder cards only at the first level when the feature is healthy
    const showFolderCards =
      folderEnabled && !currentFolderId && folders.length > 0;

    const columns = useGetColumns({
      entityConfigs,
      reloadList: listResp.reload,
      isPersonalSpace,
    });

    // const typeFilterData = [
    //   { label: I18n.t('library_filter_tags_all_types'), value: -1 },
    //   ...entityConfigs.map(item => item.typeFilter).filter(filter => !!filter),
    // ];
    const knowledgeFilterData =
      entityConfigs.find(item => item?.typeFilter?.value === ResType.Knowledge)
        ?.typeFilter?.children || [];
    const scopeOptions = getScopeOptions();
    const statusOptions = getStatusOptions();

    const handleScroll = useCallback(() => {
      if (scrollRef.current && !listResp.loading) {
        const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
        const threshold = 100;
        if (scrollTop + clientHeight >= scrollHeight - threshold) {
          if (listResp.data?.hasMore) {
            listResp.loadMore();
          }
        }
      }
    }, [listResp]);

    // Grid view only loads more on scroll. If the first page doesn't fill the
    // viewport the user can never trigger a scroll, so keep loading until the
    // container overflows or there's nothing more to fetch.
    useEffect(() => {
      if (
        layoutType !== 'grid' ||
        listResp.loading ||
        !listResp.data?.hasMore
      ) {
        return;
      }
      const el = scrollRef.current;
      if (el && el.scrollHeight <= el.clientHeight) {
        listResp.loadMore();
      }
      // eslint-disable-next-line react-hooks/exhaustive-deps -- re-check fill after each load
    }, [layoutType, listResp.loading, listResp.data]);

    const onRowClick = (record?: ResourceInfo) => {
      if (!record || record.res_type === undefined || record.detail_disable) {
        return {};
      }
      return {
        onClick: () => {
          sendTeaEvent(EVENT_NAMES.workspace_action_front, {
            space_id: spaceId,
            space_type: isPersonalSpace ? 'personal' : 'teamspace',
            tab_name: 'library',
            action: 'click',
            id: record.res_id,
            name: record.name,
            type: record.res_type && eventLibraryType[record.res_type],
          });
          entityConfigs
            .find(c => c.target.includes(record.res_type as ResType))
            ?.onItemClick(record);
        },
      };
    };

    const renderWorkflowItem = (record: ResourceInfo) => {
      const resId = record.res_id;
      const selected = resId !== undefined && selectedResIds.has(resId);
      return (
        <GridItem key={resId} className={s['resource-grid-item']}>
          <div
            className={classNames(s['resource-grid-inner'], 'relative')}
            onClick={
              batchMode ? undefined : () => onRowClick(record)?.onClick?.()
            }
          >
            <GridLibraryItem
              resourceInfo={record}
              entityConfigs={entityConfigs}
              reloadList={listResp.reload}
              gridItemWidth={gridItemWidth}
            />
            {batchMode ? (
              <>
                <div
                  className="absolute inset-0 z-[5] cursor-pointer"
                  onClick={() => toggleSelect(resId)}
                />
                <div
                  className="absolute top-[10px] left-[10px] z-[6]"
                  onClick={e => e.stopPropagation()}
                >
                  <Checkbox
                    checked={selected}
                    onChange={() => toggleSelect(resId)}
                  />
                </div>
              </>
            ) : null}
          </div>
        </GridItem>
      );
    };

    return (
      <Layout
        className={classNames(s['layout-content'], {
          'flex-col': layoutType === 'grid',
        })}
        title={renderHtmlTitle(I18n.t('navigation_workspace_library'))}
      >
        <Layout.Header className={s['layout-header']}>
          <div className="w-full">
            <LibraryHeader
              entityConfigs={entityConfigs}
              spaceId={spaceId}
              sourceType={resType}
              onRefresh={listResp.reload}
              folderEnabled={folderEnabled}
              folderNames={folders.map(f => f.name)}
              onCreateFolder={async name => {
                await createFolder(name);
              }}
            />
            <div className={s['filter-bar']}>
              <Space>
                {/* <Cascader
                  data-testid="workspace.library.filter.type"
                  className={s.cascader}
                  style={restTypeFilter?.[0] !== -1 ? highlightFilterStyle : {}}
                  dropdownClassName="[&_.semi-cascader-option-lists]:h-fit"
                  showClear={false}
                  value={restTypeFilter}
                  treeData={typeFilterData}
                  onChange={v => {
                    const typeFilter = typeFilterData.find(
                      item =>
                        item.value === ((v as Array<number>)?.[0] as number),
                    );
                    sendTeaEvent(EVENT_NAMES.workspace_action_front, {
                      space_id: spaceId,
                      space_type: isPersonalSpace ? 'personal' : 'teamspace',
                      tab_name: 'library',
                      action: 'filter',
                      filter_type: 'types',
                      filter_name: typeFilter?.filterName ?? typeFilter?.label,
                    });
                    setParams(prev => ({
                      ...prev,
                      res_type_filter: v as Array<number>,
                    }));
                  }}
                /> */}
                {resType === ResType.Knowledge && (
                  <Select
                    data-testid="workspace.library.filter.type"
                    className={s.select}
                    value={params.res_type_filter?.[1] ?? -1}
                    optionList={knowledgeFilterData}
                    onChange={v => {
                      setParams(prev => ({
                        ...prev,
                        res_type_filter: [ResType.Knowledge, v],
                      }));
                    }}
                  />
                )}
                {!isPersonalSpace ? (
                  <Select
                    data-testid="workspace.library.filter.user"
                    className={classNames(s.select)}
                    style={
                      params?.user_filter !== 0 ? highlightFilterStyle : {}
                    }
                    showClear={false}
                    value={params.user_filter}
                    optionList={scopeOptions}
                    onChange={v => {
                      sendTeaEvent(EVENT_NAMES.workspace_action_front, {
                        space_id: spaceId,
                        space_type: isPersonalSpace ? 'personal' : 'teamspace',
                        tab_name: 'library',
                        action: 'filter',
                        filter_type: 'creators',
                        filter_name: scopeOptions.find(
                          item =>
                            item.value ===
                            ((v as Array<number>)?.[0] as number),
                        )?.label,
                      });
                      setParams(prev => ({
                        ...prev,
                        user_filter: v as number,
                      }));
                    }}
                  />
                ) : null}
                <Select
                  data-testid="workspace.library.filter.status"
                  className={s.select}
                  style={
                    params?.publish_status_filter !== 0
                      ? highlightFilterStyle
                      : {}
                  }
                  showClear={false}
                  value={params.publish_status_filter}
                  optionList={statusOptions}
                  onChange={v => {
                    sendTeaEvent(EVENT_NAMES.workspace_action_front, {
                      space_id: spaceId,
                      space_type: isPersonalSpace ? 'personal' : 'teamspace',
                      tab_name: 'library',
                      action: 'filter',
                      filter_type: 'status',
                      filter_name: statusOptions.find(
                        item =>
                          item.value === ((v as Array<number>)?.[0] as number),
                      )?.label,
                    });
                    setParams(prev => ({
                      ...prev,
                      publish_status_filter: v as number,
                    }));
                  }}
                />
              </Space>
              <Space>
                <div className={s.filterSwitch}>
                  {['list', 'grid'].map(item => (
                    <div
                      key={item}
                      className={classNames(s.filterItem, s[item], {
                        [s.active]: layoutType === item,
                      })}
                      onClick={() => {
                        setLayoutType(item);
                      }}
                    />
                  ))}
                </div>
                <Search
                  data-testid="workspace.library.filter.name"
                  className={s.search}
                  style={params.name ? highlightFilterStyle : {}}
                  showClear={true}
                  width={300}
                  loading={listResp.loading}
                  placeholder={I18n.t('workspace_library_search')}
                  value={params.name}
                  onSearch={v => {
                    sendTeaEvent(EVENT_NAMES.search_front, {
                      full_url: window.location.href,
                      source: 'library',
                      search_word: v,
                    });
                    setParams(prev => ({
                      ...prev,
                      name: v,
                    }));
                  }}
                />
              </Space>
            </div>
          </div>
        </Layout.Header>
        {layoutType === 'list' ? (
          <Layout.Content>
            <Table
              data-testid="workspace.library.table"
              offsetY={178}
              tableProps={{
                loading: listResp.loading,
                dataSource: visibleList,
                columns,
                onRow: onRowClick,
              }}
              empty={
                <WorkspaceEmpty onClear={resetParams} hasFilter={hasFilter} />
              }
              enableLoad
              loadMode="cursor"
              strictDataSourceProp
              hasMore={listResp.data?.hasMore}
              onLoad={listResp.loadMore}
            />
          </Layout.Content>
        ) : (
          <>
            <div
              ref={scrollRef}
              onScroll={handleScroll}
              className={s['grid-scroll']}
            >
              {folderEnabled && currentFolder ? (
                <div
                  data-testid="workspace.library.folder.breadcrumb"
                  className="flex items-center gap-[4px] text-[14px] coz-fg-secondary mb-[12px] mt-[4px]"
                >
                  <span
                    className="cursor-pointer hover:coz-fg-primary"
                    onClick={() => setCurrentFolderId(null)}
                  >
                    {`← ${I18n.t('workspace_library_folder_all') || '全部'}`}
                  </span>
                  <span>/</span>
                  <span className="coz-fg-primary">{currentFolder.name}</span>
                </div>
              ) : null}
              {showFolderCards ? (
                <>
                  <div className="text-[16px] font-[600] coz-fg-primary mb-[12px]">
                    分类
                  </div>
                  <GridList
                    averageItemWidth={defaultGridItemWidth}
                    gap={18}
                    className={s['resource-grid']}
                    onResize={(width: number) => {
                      setGridItemWidth(width);
                    }}
                  >
                    {folders.map(folder => (
                      <GridItem
                        key={`folder-${folder.id}`}
                        className={s['resource-grid-item']}
                      >
                        <div className={s['resource-grid-inner']}>
                          <FolderCard
                            folder={folder}
                            gridItemWidth={gridItemWidth}
                            active={batchFolderId === folder.id}
                            disabled={batchMode}
                            onClick={f => setCurrentFolderId(f.id)}
                            onBatchAdd={f => startBatch(f.id)}
                            onRename={(f, name) => renameFolder(f.id, name)}
                            onDelete={f => deleteFolder(f.id)}
                          />
                        </div>
                      </GridItem>
                    ))}
                  </GridList>
                  {visibleList.length > 0 ? (
                    <>
                      <div className="text-[16px] font-[600] coz-fg-primary mb-[12px] mt-[20px]">
                        未分类工作流
                      </div>
                      <GridList
                        averageItemWidth={defaultGridItemWidth}
                        gap={18}
                        className={s['resource-grid']}
                        onResize={(width: number) => {
                          setGridItemWidth(width);
                        }}
                      >
                        {visibleList.map(renderWorkflowItem)}
                      </GridList>
                    </>
                  ) : (
                    <div className="text-[13px] coz-fg-secondary py-[24px]">
                      暂无未分类工作流
                    </div>
                  )}
                </>
              ) : (
                <GridList
                  averageItemWidth={defaultGridItemWidth}
                  gap={18}
                  className={s['resource-grid']}
                  onResize={(width: number) => {
                    setGridItemWidth(width);
                  }}
                >
                  {visibleList.map(renderWorkflowItem)}
                </GridList>
              )}
              {listResp.loading ? (
                <Spin>
                  <div className="w-full h-[100px] flex items-center justify-center" />
                </Spin>
              ) : null}
              {!visibleList.length && !showFolderCards ? (
                <div className="w-full h-full flex items-center justify-center">
                  <WorkspaceEmpty onClear={resetParams} hasFilter={hasFilter} />
                </div>
              ) : null}
            </div>
            {batchMode && batchFolder ? (
              <FolderBatchBar
                selectedCount={selectedResIds.size}
                folderName={batchFolder.name}
                loading={moving}
                onCancel={exitBatch}
                onConfirm={handleConfirmBatch}
              />
            ) : null}
          </>
        )}
      </Layout>
    );
  },
);
