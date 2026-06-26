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

import { useNavigate } from 'react-router-dom';
import { useState } from 'react';

import { useRequest } from 'ahooks';
import {
  ActionKey,
  type ResourceInfo,
  ResType,
} from '@coze-arch/idl/plugin_develop';
import { I18n } from '@coze-arch/i18n';
import { IconCozLightbulb } from '@coze-arch/coze-design/icons';
import { Menu, Table, Toast, Modal, Input } from '@coze-arch/coze-design';
import { strategyApi } from '@coze-arch/bot-api';

import { type UseEntityConfigHook } from './types';

const { TableAction } = Table;

export const useStrategyConfig: UseEntityConfigHook = ({
  spaceId,
  reloadList,
  getCommonActions,
}) => {
  const navigate = useNavigate();

  const [createVisible, setCreateVisible] = useState(false);
  const [strategyName, setStrategyName] = useState('');
  const [strategyDesc, setStrategyDesc] = useState('');

  const openCreateStrategyModal = () => {
    setStrategyName('');
    setStrategyDesc('');
    setCreateVisible(true);
  };

  const { run: createStrategy, loading: creating } = useRequest(
    () =>
      strategyApi.createStrategy({
        space_id: spaceId,
        name: strategyName,
        description: strategyDesc,
      }),
    {
      manual: true,
      onSuccess: resp => {
        const id = resp?.data?.id;
        setCreateVisible(false);
        if (id) {
          navigate(`/space/${spaceId}/strategy/${id}`);
        }
      },
    },
  );

  // delete action
  const { run: deleteStrategy } = useRequest(
    (strategyId: string) =>
      strategyApi.deleteStrategy({
        id: strategyId,
      }),
    {
      manual: true,
      onSuccess: () => {
        reloadList();
        Toast.success(I18n.t('Delete_success'));
      },
    },
  );

  const createStrategyModal = (
    <Modal
      visible={createVisible}
      title={I18n.t('strategy_create')}
      onOk={() => {
        if (strategyName.trim()) {
          createStrategy();
        }
      }}
      onCancel={() => setCreateVisible(false)}
      okText="确定"
      cancelText="取消"
      okButtonProps={{ loading: creating, disabled: !strategyName.trim() }}
    >
      <div style={{ marginBottom: 12 }}>
        <Input
          value={strategyName}
          onChange={v => setStrategyName(v)}
          placeholder={I18n.t('strategy_scenario_name')}
        />
      </div>
      <div>
        <Input
          value={strategyDesc}
          onChange={v => setStrategyDesc(v)}
          placeholder={I18n.t('strategy_model_facing_desc')}
        />
      </div>
    </Modal>
  );

  return {
    modals: <>{createStrategyModal}</>,
    config: {
      typeFilter: {
        label: I18n.t('library_resource_type_strategy'),
        value: ResType.Strategy,
      },
      onCreate: openCreateStrategyModal,
      renderCreateMenu: () => (
        <Menu.Item
          data-testid="workspace.library.header.create.strategy"
          icon={<IconCozLightbulb />}
          onClick={openCreateStrategyModal}
        >
          {I18n.t('strategy_create')}
        </Menu.Item>
      ),
      target: [ResType.Strategy],
      onItemClick: (item: ResourceInfo) => {
        navigate(`/space/${spaceId}/strategy/${item.res_id}`);
      },
      renderActions: (item: ResourceInfo) => {
        const deleteDisabled = !item.actions?.find(
          action => action.key === ActionKey.Delete,
        )?.enable;

        const deleteProps = {
          disabled: deleteDisabled,
          deleteDesc: I18n.t('library_delete_desc'),
          handler: () => {
            deleteStrategy(item.res_id || '');
          },
        };

        return (
          <TableAction
            deleteProps={deleteProps}
            actionList={getCommonActions?.(item)}
          />
        );
      },
    },
  };
};
