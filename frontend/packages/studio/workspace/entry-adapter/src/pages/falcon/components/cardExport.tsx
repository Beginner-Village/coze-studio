import { useEffect, useCallback, useState, useRef } from 'react';
import { useParams } from 'react-router-dom';
import {
  Button,
  IconButton,
  Search,
  Select,
  Spin,
  Menu,
  MenuItem,
  Popconfirm,
  RadioGroup,
  Radio,
  SideSheet,
  Dropdown,
  Modal,
  Tooltip,
  Toast,
  Table,
  Image,
  EmptyState,
  CozPagination
} from '@coze-arch/coze-design';
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
import { aopApi } from '@coze-arch/bot-api';
import { replaceUrl } from '../utils';

export const CardExport = ({
    spaceId,
    visible,
    onVisibleChange
}) => {
    const [loading, setLoading] = useState(false);
    const [exportVisible, setExportVisible] = useState(false);
    const [filterType, setFilterType] = useState('');
    const [cardList, setCardList] = useState([]);
    const [pageNo, setPageNo] = useState(1);
    const [pageSize, setPageSize] = useState(5);
    const [totalNum, setTotalNum] = useState(0);
    const [selectCards, setSelectCards] = useState([] as any[]);
    const [cardClassMap, setCardClassMap] = useState({});
    const [buttonLoading, setButtonLoading] = useState(false);
    const [recordVisible, setRecordVisible] = useState(false);
    const [recordPageNo, setRecordPageNo] = useState(1);
    const [recordPageSize, setRecordPageSize] = useState(5);
    const [recordTotalNum, setRecordTotalNum] = useState(0);
    const [recordList, setRecordList] = useState([]);
    const [recordLoading, setRecordLoading] = useState(false);
    const [typeList, setTypeList] = useState([
        {
            label: '全部',
            value: '',
            count: -1,
        },
    ]);
    const columns = [
        {
            title: '卡片名称',
            dataIndex: 'cardName',
            width: '22%',
            align: 'left',
        },
        {
            title: '卡片编码',
            dataIndex: 'code',
            width: '22%',
            align: 'left',
        },
        {
            title: '卡片缩略图',
            dataIndex: 'picUrl',
            width: '22%',
            align: 'left',
            render: (text, record) => (<Image src={replaceUrl(record.picUrl)} width={40} height={40} />)
        },
        {
            title: '卡片分类',
            dataIndex: 'cardClassId',
            width: '22%',
            align: 'left',
            render: (text, record) => (<span>{cardClassMap[record.cardClassId] || '--'}</span>)
        },
    ];
    const recordColumns = [
        {
            title: '导出任务',
            dataIndex: 'cardCount',
            width: '25%',
            align: 'left',
            render: (text, record) => (<span>{record.cardCount}张卡片</span>)
        },
        {
            title: '导出时间',
            dataIndex: 'createTime',
            width: '25%',
            align: 'left',
        },
        {
            title: '导出文件',
            dataIndex: 'downUrl',
            width: '25%',
            align: 'left',
            render: (text, record) => (record.status == '1' ? <a href={replaceUrl(record.downUrl)} target='_blank'>下载</a> : '--')
        },
        {
            title: '导出状态',
            dataIndex: 'status',
            width: '25%',
            align: 'left',
            render: (text, record) => (record.status == '1' ? <span>成功</span> : record.status == '2' ? <span>失败</span> : record.status == '0' ? <span>{record.progress || '等待'}</span> : '--')
        },
    ];

    useEffect(() => {
        setExportVisible(visible);
    }, [visible]);

    const setVisible = useCallback((visible) => {
        setExportVisible(visible)
        onVisibleChange(visible);
    })

    const getCardClassList = useCallback(() => {
        aopApi
        .GetCardTypeCount({
            sassAppId: '100001',
            sassWorkspaceId: spaceId,
        })
        .then(res => {
            const listData = res.body.cardClassList;
            let allCount = 0
            
            listData.forEach(item => {
                allCount += Number(item.count);
            });
            const list = [
                {
                    label: '全部',
                    value: '',
                    count: allCount || -1,
                },
                ...listData.map(item => ({
                    label: item.name,
                    value: item.id,
                    count: Number(item.count),
                })),
            ];
            setTypeList(list);
        });
    })

    const getCardClassMap = useCallback(() => {
            aopApi
            .GetCardTypes({
                sassAppId: '100001',
                sassWorkspaceId: spaceId,
            })
            .then(res => {
                const listData = res.body.cardClassList;
                let classMap = {}
                listData.forEach(item => {
                    classMap[item.id] = item.name
                });
                setCardClassMap(classMap);
            });
        })

    const getCardListData = useCallback(
        () => {
            setLoading(true);
            console.log('[FalconCard] Calling GetCardResourceList API...');
            aopApi
            .GetCardResourceList({
                createdBy: true,
                sassAppId: '100001',
                sassWorkspaceId: spaceId,
                cardClassId: filterType,
                pageNo: pageNo,
                pageSize: pageSize,
            })
            .then(res => {
                console.log('[FalconCard] GetCardResourceList response:', res);
                const newList = res.body.cardList || [];
                setTotalNum(Number(res.body.totalNums));
                setCardList(newList);
            })
            .catch(err => {
                console.error('[FalconCard] GetCardResourceList error:', err);
            })
            .finally(() => {
                setLoading(false);
            });
        },
        // [spaceId, filterType, pageNo, pageSize]
    );

    const getCardRecordListData = useCallback(
        () => {
            setRecordLoading(true);
            aopApi
            .GetExportCardRecordList({
                applyScene: "1",
                sassAppId: '100001',
                sassWorkspaceId: spaceId,
                pageNo: recordPageNo,
                pageSize: recordPageSize,
            })
            .then(res => {
                setRecordTotalNum(Number(res.body.totalNums));
                setRecordList(res.body.exportTaskLists || []);
            })
            .catch(err => {
                console.error('[FalconCard] GetCardResourceList error:', err);
            })
            .finally(() => {
                setRecordLoading(false);
            });
        }, 
        // [spaceId, recordPageNo, recordPageSize]
    )

    const exportCard = useCallback(async () => {
        if(selectCards.length === 0) return
        Modal.confirm({
            title: '卡片导出',
            content: '确定要导出选中的卡片吗？',
            okText: '确定',
            cancelText: '取消',
            onOk: () => {
                (async ()=>{
                    try{
                        setButtonLoading(true)
                        // 调用导出卡片接口
                        await aopApi.ExportCard({
                            applyScene: "1",
                            cards: selectCards.map((item) => (item as any).cardId),
                            sassAppId: '100001',
                            sassWorkspaceId: spaceId,
                        })
                        Toast.info({
                            content: '卡片导出任务已加入队列，请查看“导出记录列表”',
                            duration: 3,
                        })
                    }finally{
                        setButtonLoading(false)
                    }
                })();
            },
        })
    })

    useEffect(() => {
        getCardListData();
    }, [spaceId, filterType, pageNo, pageSize]);

    useEffect(() => {
        getCardRecordListData();
    }, [spaceId, recordPageNo, recordPageSize]);

    useEffect(() => {
        if(!recordVisible) return
        setRecordPageNo(1)
        setRecordPageSize(5)
        setRecordTotalNum(0)
        getCardRecordListData()
    }, [recordVisible]);

    useEffect(() => {
        if(!visible) return
        setFilterType('')
        setPageNo(1)
        setPageSize(5)
        setTotalNum(0)
        setSelectCards([])
        getCardClassList()
        getCardClassMap()
        getCardListData()
    }, [visible]);

    return (
        <Modal
            type="modal"
            title="卡片导出"
            width={800}
            visible={exportVisible}
            onOk={() => setVisible(false)}
            onCancel={() => setVisible(false)}
            cancelText="取消"
            okText="确定"
            footer={
                <div className="flex justify-center items-center relative sticky">
                    <div className="absolute left-0">已选{selectCards.length}个</div>
                    <Button color="primary" disabled={buttonLoading} onClick={() => setVisible(false)}>取消</Button>
                    <Button color="brand" loading={buttonLoading} onClick={exportCard}>导出</Button>
                    <Button color="secondary" className="absolute right-0" onClick={() => setRecordVisible(true)}>导出记录列表</Button>
                </div>
            }
            >
            <Modal.Content >
                <SubHeader className="sticky">
                    <SubHeaderFilters>
                        <Select
                            className="min-w-[128px]"
                            value={filterType}
                            onChange={(val: string | number) => {
                                setPageNo(1);
                                setFilterType(val as string);
                            }}
                        >
                        {typeList.map(opt => (
                            <Select.Option key={opt.value} value={opt.value}>
                                <span>{opt.label}</span>
                                <span className="text-[12px] ml-[4px] coz-fg-secondary">
                                {opt.count > -1 ? opt.count : ''}
                                </span>
                            </Select.Option>
                        ))}
                        </Select>
                    </SubHeaderFilters>
                </SubHeader>
                <div className="mt-6">
                    <Table
                        tableProps={{
                            columns,
                            dataSource: cardList,
                            rowKey: 'cardId',
                            loading,
                            rowSelection: {
                                selectedRowKeys: selectCards.map((item) => (item as any).cardId),
                                onChange: (selectedRowKeys, selectedRows) => {
                                    console.log(
                                    `selectedRowKeys: ${selectedRowKeys}`,
                                    'selectedRows: ',
                                    selectedRows,
                                    );
                                    let selects: any[] = [].concat(selectCards as any)
                                    let selectIds = selects.map((item) => {
                                        return item.cardId
                                    });
                                    let list = (cardList as any)
                                    list.forEach((item) => {
                                        if((selectedRowKeys || []).includes(item.cardId)){
                                            if(!selectIds.includes(item.cardId)){
                                                selects.push(item)
                                            }
                                        }else{
                                            selects = selects.filter((i) => i.cardId !== item.cardId)
                                        }
                                    })
                                    setSelectCards(selects);
                                },
                            },
                        }}
                        empty={
                            <EmptyState
                                title="暂无数据"
                                description="当前没有任何数据，请稍后再试"
                            />
                        }
                    />
                    <div className="flex justify-end mt-6">
                        <CozPagination
                            currentPage={pageNo}
                            pageSize={pageSize}
                            pageSizeOpts={[5, 10, 20, 30, 40, 50, 100]}
                            total={totalNum}
                            showTotal
                            showSizeChanger
                            onChange={(page, pageSize) => {
                                setPageNo(page);
                                setPageSize(pageSize);
                                console.info('asdf===========', page, pageSize)
                            }}
                        />
                    </div>
                </div>
                <Modal
                    type="modal"
                    title="导出记录"
                    width={600}
                    visible={recordVisible}
                    onCancel={() => setRecordVisible(false)}
                    >
                    <Modal.Content >
                        <div className="mt-6">
                            <Table
                                tableProps={{
                                    columns: recordColumns,
                                    dataSource: recordList,
                                    rowKey: 'taskId',
                                    loading: recordLoading,
                                }}
                                empty={
                                    <EmptyState
                                        title="暂无数据"
                                        description="当前没有任何数据，请稍后再试"
                                    />
                                }
                            />
                            <div className="flex justify-end mt-6">
                                <CozPagination
                                    currentPage={recordPageNo}
                                    pageSize={recordPageSize}
                                    pageSizeOpts={[5, 10, 20, 30, 40]}
                                    total={recordTotalNum}
                                    showTotal
                                    showSizeChanger
                                    onChange={(page, pageSize) => {
                                        setRecordPageNo(page);
                                        setRecordPageSize(pageSize);
                                    }}
                                />
                            </div>
                        </div>
                    </Modal.Content>
                </Modal>
            </Modal.Content>
        </Modal>
    )
}