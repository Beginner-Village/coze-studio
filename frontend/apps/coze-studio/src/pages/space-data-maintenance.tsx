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
import { I18n, type I18nKeysNoOptionsType } from '@coze-arch/i18n';

import { DataMaintenanceSection } from './space-management/DataMaintenanceSection';

const { Title } = Typography;

const t = (
  key: string,
  options: Record<string, unknown>,
  fallbackText: string,
) => I18n.t(key as I18nKeysNoOptionsType, options, fallbackText);

const Page = () => {
  const { space_id: spaceId } = useParams<{ space_id: string }>();

  if (!spaceId) {
    return (
      <Layout>
        <Layout.Content>
          <div className="py-16 text-center text-gray-500">
            {t(
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
    <div className="h-full flex flex-col overflow-hidden bg-[rgb(244,246,251)]">
      <div className="flex-none px-[32px] pt-[26px] pb-[18px] bg-[var(--coz-bg-max)] border-0 border-b-[1px] border-solid coz-stroke-primary">
        <Title heading={3} className="!m-0 !text-[24px] !font-[600]">
          {t('space_data_maintenance', {}, '数据维护')}
        </Title>
      </div>
      <div className="flex-1 overflow-y-auto px-[32px] pt-[22px] pb-[40px]">
        <DataMaintenanceSection spaceId={spaceId} />
      </div>
    </div>
  );
};

export { Page as Component };
export default Page;
