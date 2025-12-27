import { useEffect, useCallback, useState, useRef, useMemo } from 'react';
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
  Image,
  Modal,
  Upload,
  Table,
  EmptyState,
  Tooltip,
  Toast
} from '@coze-arch/coze-design';
import { aopApi } from '@coze-arch/bot-api';
import { replaceUrl } from '../utils';

export const CardImport = ({
    spaceId,
    visible,
    onVisibleChange
}) => {
    const [importVisible, setImportVisible] = useState(false);
    const uploadRef = useRef(null);
    const uploadUrl = aopApi.GetImportCardUploadUrl()
    const [detailVisible, setDetailVisible] = useState(false);
    const [uploadFileName, setUploadFileName] = useState('')
    const [uploadStatus, setUploadStatus] = useState('')
    const [taskInfo, setTaskInfo] = useState({} as any)
    const [cardClassMap, setCardClassMap] = useState({})
    const [buttonLoading, setButtonLoading] = useState(false)
    const [timer, setTimer] = useState(null as any)
    const cardStatusMap = {
        '0': '等待',
        '1': '成功',
        '2': '失败'
    }
    const columns = [
        {
            title: '卡片名称',
            dataIndex: 'name',
            width: '20%',
            align: 'left',
        },
        {
            title: '卡片编码',
            dataIndex: 'code',
            width: '20%',
            align: 'left',
        },
        {
            title: '卡片缩略图',
            dataIndex: 'picUrl',
            width: '20%',
            align: 'left',
            render: (text, record) => (<Image src={replaceUrl(record.picUrl)} width={40} height={40} />)
        },
        {
            title: '卡片分类',
            dataIndex: 'cardClassId',
            width: '20%',
            align: 'left',
            render: (text, record) => (<span>{cardClassMap[record.cardClassId] || '--'}</span>)
        },
        {
            title: '状态',
            dataIndex: 'status',
            width: '20%',
            align: 'left',
            render: (text, record) => (<span>{cardStatusMap[record.status] || '--'}</span>)
        }
    ];
    const statInfo = useRef({
        total: 5,
        success: 0,
        fail: 0,
        import: 0
    })

    useEffect(() => {
        setImportVisible(visible);
    }, [visible]);

    const setVisible = useCallback((visible) => {
        setImportVisible(visible)
        onVisibleChange(visible);
    })

    const getCardClassList = useCallback(() => {
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

    const exchangeUpload = useCallback(() => {
        timer && clearTimeout(timer)
        setDetailVisible(false)
    })

    const exchangeList = useCallback((taskInfo) => {
        setTaskInfo(taskInfo)
        setDetailVisible(true)
    })

    const updateStatInfo = useCallback(() => {
        let cards = taskInfo.cards || []
        statInfo.current = {
            total: cards.length,
            success: cards.filter(item => item.status == '1').length,
            fail: cards.filter(item => item.status == '2').length,
            import: cards.filter(item => !item.status || item.status == '0').length
        }
    })

    const startTimer = useCallback(() => {
        timer && clearTimeout(timer)
        setTimer(setTimeout(() => {
            let importCards = (taskInfo as any).cards.filter(item => !item.status || item.status == '0')
            if(importCards.length){
                getImportCardList(importCards)
            }else{
                timer && clearTimeout(timer)
                Toast.info({
                    content: '卡片导入完成',
                    duration: 2
                })
                setButtonLoading(false)
            }
        }, 1000))
    }, [taskInfo])

    const getImportCardList = useCallback(async (importCards) => {
        let res = await aopApi.GetImportCardResutList({taskId: taskInfo.taskId, sassAppId: '100001', sassWorkspaceId: spaceId})
        let list = res.body.importCardList || []
        let cardMap = list.reduce((retObj, item) => {
            retObj[item.cardCode] = item
            return retObj
        }, {})
        let cards = taskInfo.cards || []
        cards.forEach(info => {
            let cardInfo = cardMap[info.code]
            if(cardInfo) {
                info.status = cardInfo.status
            }
        })
        updateStatInfo()
        startTimer()
    }, [spaceId, taskInfo])

    const importCard = useCallback(async () => {
        setButtonLoading(true)
        try{
            await aopApi.ImportCard({taskId: taskInfo.taskId, zipUrl: taskInfo.zipUrl, sassAppId: '100001', sassWorkspaceId: spaceId})
            startTimer()
        }catch(err){
            setButtonLoading(false)
        }
    }, [spaceId, taskInfo])

    useEffect(() => {
        timer && clearTimeout(timer)
        if (!visible) return
        setDetailVisible(false)
        getCardClassList()
    }, [visible])
    
    return (
        <Modal
            type="modal"
            title="卡片导入"
            width={800}
            style={{minHeight: '600px'}}
            visible={importVisible}
            onCancel={() => setVisible(false)}
            footer={
                detailVisible ?
                <div className="flex justify-center items-center relative sticky">
                    <div className="absolute left-0">
                        <span>进度{statInfo.success + statInfo.fail}/{statInfo.total}, 成功：{ statInfo.success },</span>
                        <span>失败：{ statInfo.fail }</span>
                    </div>
                    <Button color="primary" disabled={buttonLoading} onClick={() => exchangeUpload()}>返回上一步</Button>
                    <Button color="brand" loading={buttonLoading} onClick={importCard}>导入</Button>
                </div>
                : null
            }
        >
            { !detailVisible ?
            <Modal.Content className="w-full h-full flex justify-center items-center p-20">
                <Upload 
                    ref={uploadRef.current}
                    action={uploadUrl}
                    name="file"
                    multiple={false}
                    limit={1}
                    draggable={true}
                    data={{
                        applyScene: '1',
                        sassWorkspaceId: spaceId,
                        sassAppId: '100001'
                    }}
                    onChange={(file) => {
                        setUploadFileName(file.currentFile.name)
                        console.info('卡片变化==========', file);
                    }}
                    onSuccess={(res) => {
                        console.info('卡片导入成功==========', res);
                        if(res.header.errorCode == '0'){
                            exchangeList(res.body)
                            setUploadStatus('1')
                        }else{
                            setUploadStatus('2')
                        }
                    }}
                    onError={(res) =>{
                        console.info('卡片导入失败==========', res);
                        setUploadStatus('2')
                    }}
                >
                    <div>
                        <Button type="primary">上传文件</Button>
                    </div>
                </Upload>
            </Modal.Content>
            :
            <Modal.Content className={((taskInfo as any).cards || []).length ? 'p-[8px]' : 'w-full h-full flex justify-center items-center' }>
                <Table
                    tableProps={{
                        columns,
                        dataSource: (taskInfo as any).cards || [],
                        rowKey: 'id',
                    }}
                    empty={
                        <EmptyState
                            title="暂无数据"
                            description="当前没有任何数据，请稍后再试"
                        />
                    }
                />
            </Modal.Content>
            }
        </Modal>
    )
}