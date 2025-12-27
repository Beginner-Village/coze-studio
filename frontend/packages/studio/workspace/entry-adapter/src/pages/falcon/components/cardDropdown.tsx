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
} from '@coze-arch/coze-design';
import { CardExport } from './cardExport';
import { CardImport } from './cardImport';

export const FalconCardDropdown = ({
  spaceId
}) => {
  const [exportVisible, setExportVisible] = useState(false);
  const [importVisible, setImportVisible] = useState(false);
  const showModal = useCallback((value) => {
    console.info('121221========', value)
    if(value === 'export'){
      setExportVisible(true);
    }else if(value === 'import'){
      setImportVisible(true);
    }
  })

  return (
    <div>
      <Dropdown
        trigger="hover" 
        render={
          <Dropdown.SubMenu mode="menu" onSelectionChange={showModal}>
            <Dropdown.Item itemKey="export">导出</Dropdown.Item>
            <Dropdown.Item itemKey="import">导入</Dropdown.Item>
          </Dropdown.SubMenu> 
        }
      >
        <span className="flex items-center cursor-pointer">...</span>
      </Dropdown>
      <CardExport visible={exportVisible} spaceId={spaceId} onVisibleChange={setExportVisible}></CardExport>
      <CardImport visible={importVisible} spaceId={spaceId} onVisibleChange={setImportVisible}></CardImport>
      {/* <Modal
            type="modal"
            title="卡片导出"
            visible={exportVisible}
            onOk={() => setExportVisible(false)}
            onCancel={() => setExportVisible(false)}
            cancelText="取消"
            okText="确定"
            >
            <Modal.SubTitle>副标题</Modal.SubTitle>
            <Modal.Description>这是一段描述文本</Modal.Description>
            <Modal.Content>这是主要内容区域</Modal.Content>
        </Modal> */}
    </div>
  )
}