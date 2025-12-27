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
  Upload,
  Tooltip,
} from '@coze-arch/coze-design';

export const CardImport = ({
    spaceId,
    visible,
    onVisibleChange
}) => {
    const [importVisible, setImportVisible] = useState(false);

    useEffect(() => {
        setImportVisible(visible);
    }, [visible]);

    const setVisible = useCallback((visible) => {
        setImportVisible(visible)
        onVisibleChange(visible);
    })
    
    return (
        <Modal
            type="modal"
            title="卡片导入"
            width={800}
            visible={importVisible}
            onOk={() => setVisible(false)}
            onCancel={() => setVisible(false)}
            cancelText="取消"
            okText="确定"
            >
            <Modal.Content>
                ===

            </Modal.Content>
            {/* <Upload
                action="https://www.mocky.io/v2/5cc8019d300000980a055e76">
                <Button type="primary">上传文件</Button>
            </Upload> */}
        </Modal>
    )
}