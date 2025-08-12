import { type FC, useEffect } from 'react';
import { useSpaceStore } from '@coze-foundation/space-store-adapter';
import { SpaceType } from '@coze-arch/bot-api/developer_api';
import { I18n, type I18nKeysNoOptionsType } from '@coze-arch/i18n';
import classNames from 'classnames';
import {
  highlightFilterStyle,
  WorkspaceEmpty,
  DevelopCustomPublishStatus,
  isPublishStatus,
  isRecentOpen,
  isSearchScopeEnum,
  getPublishRequestParam,
  getTypeRequestParams,
  isEqualDefaultFilterParams,
  isFilterHighlight,
  CREATOR_FILTER_OPTIONS,
  FILTER_PARAMS_DEFAULT,
  STATUS_FILTER_OPTIONS,
  TYPE_FILTER_OPTIONS,
  BotCard,
  Content,
  Header,
  HeaderActions,
  HeaderTitle,
  Layout,
  SubHeader,
  SubHeaderFilters,
  SubHeaderSearch,
  useIntelligenceList,
  useIntelligenceActions,
  useCachedQueryParams,
  useGlobalEventListeners,
  type DevelopProps,
  useProjectCopyPolling,
  useCardActions,
} from '@coze-studio/workspace-base/develop';
import { IconCozLoading, IconCozPlus } from '@coze-arch/coze-design/icons';
import {
  Button,
  IconButton,
  Search,
  Select,
  Spin,
} from '@coze-arch/coze-design';
import { EVENT_NAMES, sendTeaEvent } from '@coze-arch/bot-tea';

export const FalconCard: FC<DevelopProps> = ({ spaceId }) => {

    // const isPersonal = useSpaceStore(
    //     state => state.space.space_type === SpaceType.Personal,
    // );

  //   useGlobalEventListeners({ reload: true, spaceId });

  //   /**
  //    * report tea event
  //    */
  //   useEffect(() => {
  //     sendTeaEvent(EVENT_NAMES.view_bot, { tab: 'my_bots' });
  //   }, []);

  //   useProjectCopyPolling({
  //     listData: [],
  //     spaceId,
  //     mutate: [],
  //   });

  //   /**
  //  * Create project
  //  */
  // const { contextHolder, actions } = useIntelligenceActions({
  //   spaceId,
  //   mutateList: [],
  //   reloadList: [],
  // });

  // return (
  //   <div>
  //     ================================mcp===
  //   </div>
  // )

    return (
        <>
      <Layout>
        <Header>
          <HeaderTitle>
            <span>{I18n.t('workspace_develop')}</span>
          </HeaderTitle>
          <HeaderActions>
            <Button icon={<IconCozPlus />}>
              {I18n.t('workspace_create')}
            </Button>
          </HeaderActions>
        </Header>
        <SubHeader>
  
        </SubHeader>
        <Content>
          =============card==
        </Content>
      </Layout>
    </>
    )
}