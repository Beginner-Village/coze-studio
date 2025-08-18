/* eslint-disable @coze-arch/max-line-per-function */
/* eslint-disable prettier/prettier */
import { useEffect, useCallback, useState, useRef } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { I18n } from '@coze-arch/i18n';
import {
  Button,
  Breadcrumb,
  Form,
  Spin,
  Toast,
  Space,
  RadioGroup,
  Radio,
  Table,
} from '@coze-arch/coze-design';
import { getParamsFromQuery } from '../../../../../../arch/bot-utils';
import { useParams } from 'react-router-dom';
import {
  IconCozArrowLeft,
  IconCozWarningCircleFill,
} from '@coze-arch/coze-design/icons';
import {
  Content,
  Header,
  SubHeaderSearch,
  HeaderTitle,
  SubHeaderFilters,
  Layout,
  SubHeader,
  HeaderActions,
  type DevelopProps,
} from '@coze-studio/workspace-base/develop';
import { IconCozPlus, IconCozEmpty } from '@coze-arch/coze-design/icons';
import { GridList, GridItem } from './components/gridList';
import placeholderImg from './assets/placeholder.png';

import cls from 'classnames';
import { replaceUrl, parseUrl } from './utils';
import { aopApi } from '@coze-arch/bot-api';

import styles from './index.module.less';

const cardList = [
  {
    bizChannel: '',
    cardClassId: '4',
    cardId: '10000176',
    cardName: '理财产品对比',
    cardPicUrl: '',
    cardShelfStatus: '1',
    cardShelfTime: '',
    code: 'financialProductsComparison',
    createUserId: '',
    createUserName: '',
    picUrl:
      '@filestore/dev-public-cbbiz/20250714174158_截屏2025-07-14 17.39.55.png',
    sassAppId: '100001',
    sassWorkspaceId: '7533521629687578624',
  },
  {
    bizChannel: '',
    cardClassId: '4',
    cardId: '10000173',
    cardName: '产品解读',
    cardPicUrl: '',
    cardShelfStatus: '1',
    cardShelfTime: '',
    code: 'productInterpretation',
    createUserId: '',
    createUserName: '',
    picUrl:
      '@filestore/dev-public-cbbiz/20250714161144_截屏2025-07-14 16.11.36.png',
    sassAppId: '100001',
    sassWorkspaceId: '7533521629687578624',
  },
  {
    bizChannel: '',
    cardClassId: '4',
    cardId: '10000154',
    cardName: '理财持仓收益-弹窗',
    cardPicUrl: '',
    cardShelfStatus: '1',
    cardShelfTime: '',
    code: 'financialProductEarningDialog',
    createUserId: '',
    createUserName: '',
    picUrl: '@filestore/dev-public-cbbiz/20250624104919_12.jpg',
    sassAppId: '100001',
    sassWorkspaceId: '7533521629687578624',
  },
  {
    bizChannel: '',
    cardClassId: '4',
    cardId: '10000143',
    cardName: '产品赎回',
    cardPicUrl: '',
    cardShelfStatus: '1',
    cardShelfTime: '',
    code: 'financialProductRedemption',
    createUserId: '',
    createUserName: '',
    picUrl: '@filestore/dev-public-cbbiz/20250606101850_ic_39566.png',
    sassAppId: '100001',
    sassWorkspaceId: '7533521629687578624',
  },
];

export const FalconMarketCardDetail = () => {
  const cardId = getParamsFromQuery({ key: 'card_id' });
  const creator = getParamsFromQuery({ key: 'creator' });
  const createTime = getParamsFromQuery({ key: 'createTime' });
  const previewImg = getParamsFromQuery({ key: 'preview_img' });
  const [cardDetail, setCardDetail] = useState({});
  const [showType, setShowType] = useState('preview');
  const [addCardId, setAddCardId] = useState('');
  const navigate = useNavigate();

  const addToMe = useCallback(() => {
    aopApi
      .CardMarketAddToMe({
        cardId: cardId,
      })
      .then(res => {
        Toast.success(I18n.t('Added'));
        setAddCardId(cardId);
      })
      .catch(err => {
        Toast.error(err.message);
      });
  }, [cardId]);

  useEffect(() => {
    aopApi
      .GetCardMarketDetail({
        cardId: cardId,
      })
      .then(res => {
        setCardDetail(res.body);
      })
      .catch(err => {
        Toast.error(err.message);
      });
  }, [cardId]);

  return (
    <div className="mt-[16px] mx-[24px]">
      <Header>
        <HeaderTitle>
          <Button
            color="secondary"
            icon={<IconCozArrowLeft />}
            onClick={() => navigate(-1)}
          >
            {I18n.t('back') + I18n.t('workspace_card_library')}
          </Button>
        </HeaderTitle>
      </Header>
      <div className={styles.marketCardDetailContent}>
        <div
          className="py-[24px] mx-[20px] flex items-start"
          style={{
            borderBottom:
              '1px solid rgba(var(--coze-stroke-5), var(--coze-stroke-5-alpha))',
          }}
        >
          <div className="flex-col flex-1">
            <div className="text-[24px] font-bold mb-[12px]">
              {cardDetail.cardName}
            </div>
            <Space spacing={12} className="text-[12px] coz-fg-secondary">
              <div>
                {I18n.t('Publisher')}：{creator || '暂无'}
              </div>
              <div>
                {I18n.t('PublishedTime')}：{createTime || '暂无'}
              </div>
            </Space>
            <div className="text-[16px] mt-[16px] coz-fg-secondary">
              {cardDetail.code}
            </div>
          </div>
          <Button
            size="large"
            type="primary"
            icon={<IconCozPlus />}
            onClick={addToMe}
            disabled={addCardId === cardId}
          >
            {I18n.t('workspace_card_add_my_workstation')}
          </Button>
        </div>
        <div className="mt-[24px] mx-[20px] flex gap-[24px]">
          <div className="flex-1">
            <RadioGroup
              type="button"
              value={showType}
              onChange={e => {
                setShowType(e.target.value);
              }}
            >
              <Radio value="preview">概览</Radio>
              <Radio value="version">版本</Radio>
            </RadioGroup>
            {showType === 'preview' && (
              <div className="mt-[16px]">
                <div className="text-[20px] font-[600] mb-[12px]">
                  {I18n.t('workspace_card_preview')}
                </div>
                <div className="w-full py-[54px] bg-[#EFF0F4] rounded-[6px]">
                  <div className="w-full h-[300px]">
                    <img
                      src={previewImg}
                      alt=""
                      className="block h-[100%] mx-[auto]"
                    />
                  </div>
                </div>
                <div className="text-[20px] font-[600] mb-[12px] mt-[24px]">
                  {I18n.t('workspace_card_params')}
                </div>
                <div
                  className="w-full px-[24px] py-[24px] bg-[#fff] rounded-[6px]"
                  style={{
                    border:
                      '1px solid rgba(var(--coze-stroke-5), var(--coze-stroke-5-alpha))',
                  }}
                >
                  <Table
                    tableProps={{
                      columns: [
                        {
                          key: '1',
                          title: '参数',
                          dataIndex: 'paramName',
                        },
                        {
                          key: '2',
                          title: '名称',
                          dataIndex: 'paramDesc',
                        },
                        {
                          key: '3',
                          title: '类型',
                          dataIndex: 'paramType',
                        },
                        {
                          key: '4',
                          title: '是否必填',
                          dataIndex: 'isRequired',
                          align: 'left',
                          render: (text, record) =>
                            record.isRequired === '1' ? '是' : '否',
                        },
                      ],
                      className: 'bg-[#fff]',
                      rowKey: 'paramId',
                      dataSource: cardDetail.paramList || [],
                      pagination: false,
                    }}
                  />
                  {(cardDetail.paramList || []).length === 0 && (
                    <div className="w-full h-full flex flex-col items-center pt-[20px]">
                      <IconCozEmpty className="w-[48px] h-[48px] coz-fg-dim" />
                      <div className="text-[16px] font-[500] leading-[22px] mt-[8px] mb-[16px] coz-fg-primary">
                        {I18n.t('analytic_query_blank_context')}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
          <div className="w-[276px]">
            <div className="text-[18px] font-[600] mb-[20px]">
              {I18n.t('workspace_card_hot_recommend')}
            </div>
            <GridList averageItemWidth={276}>
              {cardList.map(item => (
                <GridItem key={item.cardId}>
                  <div
                    className={cls(
                      'px-[16px] h-full flex flex-col justify-between',
                    )}
                    onClick={e => {
                      navigate(
                        `/template/market-card-detail?card_id=${
                          item.cardId
                        }&preview_img=${replaceUrl(item.picUrl)}&creator=${
                          item.createUserName
                        }&createTime=${item.cardShelfTime}`,
                        { replace: true },
                      );
                    }}
                  >
                    <div className="py-[12px]">
                      <div className="flex flex-col gap-[8px]">
                        <div
                          className="w-full h-[180px] px-[12px] py-[12px] bg-[#EFF0F4] rounded-[6px]"
                          style={{
                            background: `#EFF0F4 url("${placeholderImg}") no-repeat center center / 108px auto`,
                          }}
                        >
                          <div
                            className="w-full h-full"
                            style={{
                              background: `url("${replaceUrl(item.picUrl)}") no-repeat center center / contain`,
                              cursor: 'pointer',
                            }}
                          />
                        </div>
                        <div>
                          <div className="flex gap-[6px] mb-[4px] items-center">
                            <div className="text-[18px] font-medium">
                              {item.cardName}
                            </div>
                          </div>
                          <div className={styles.cardTag}>{item.code}</div>
                        </div>
                      </div>
                    </div>
                  </div>
                </GridItem>
              ))}
            </GridList>
          </div>
        </div>
      </div>
    </div>
  );
};
