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

import { useParams } from 'react-router-dom';
import { Layout, Typography } from '@coze-arch/coze-design';
import { I18n } from '@coze-arch/i18n';

import { DataMaintenanceSection } from './space-management/DataMaintenanceSection';

const { Title } = Typography;

const Page = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();

  if (!spaceId) {
    return (
      <Layout>
        <Layout.Content>
          <div className="py-16 text-center text-gray-500">
            {I18n.t(
              'space_data_maintenance_missing_id',
              {},
              '缺少 space_id 参数',
            )}
          </div>
        </Layout.Content>
      </Layout>
    );
  }

  return (
    <Layout>
      <Layout.Header className="pb-0">
        <Title heading={4}>
          {I18n.t('space_data_maintenance', {}, '数据维护')}
        </Title>
      </Layout.Header>
      <Layout.Content style={{ overflowY: 'auto', padding: '0 24px 24px' }}>
        <DataMaintenanceSection spaceId={spaceId} />
      </Layout.Content>
    </Layout>
  );
};

export { Page as Component };
export default Page;
