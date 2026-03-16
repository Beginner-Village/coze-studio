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

import { useRef, useState } from 'react';

import { useDataModalWithCoze } from '@coze-data/utils';
import { useDataNavigate } from '@coze-data/knowledge-stores';
import { UnitType } from '@coze-data/knowledge-resource-processor-core';
import {
  CozeKnowledgeAddTypeContent,
  type CozeKnowledgeAddTypeContentFormData,
} from '@coze-data/knowledge-modal-base/create-knowledge-modal-v2';
import { KnowledgeE2e } from '@coze-data/e2e';
import { I18n } from '@coze-arch/i18n';
import { Button, Form, LoadingButton } from '@coze-arch/coze-design';
import { useSpaceStore } from '@coze-arch/bot-studio-store';
import { FormatType } from '@coze-arch/bot-api/memory';
import { KnowledgeApi } from '@coze-arch/bot-api';

import styles from './index.module.less';

export interface UseCreateKnowledgeModalParams {
  projectID?: string;
  onFinish?: (datasetId: string, type: UnitType, shouldUpload: boolean) => void;
  beforeCreate?: (shouldUpload: boolean) => void;
}
export const useCreateKnowledgeModalV2 = (
  params: UseCreateKnowledgeModalParams = {},
) => {
  const { onFinish, beforeCreate, projectID } = params;
  const formRef = useRef<Form<CozeKnowledgeAddTypeContentFormData>>(null);
  // Use useState to ensure re-rendering
  const [currentFormatType, setCurrentFormatType] = useState(FormatType.Text);
  const spaceId = useSpaceStore(store => store.getSpaceId());

  const resourceNavigate = useDataNavigate();

  const [unitType, setUnitType] = useState<UnitType>(UnitType.TEXT_DOC);
  // Use ref to store latest unitType value to avoid closure issues in onClick callbacks
  const unitTypeRef = useRef<UnitType>(UnitType.TEXT_DOC);
  const handleUnitTypeChange = (type: UnitType) => {
    setUnitType(type);
    unitTypeRef.current = type;
  };

  const createDataset = async () => {
    await formRef.current?.formApi.validate();
    const { dataset_id: datasetId } = await KnowledgeApi.CreateDataset({
      project_id: projectID || undefined,
      name: formRef.current?.formApi.getValue('name'),
      format_type: currentFormatType,
      description: formRef.current?.formApi.getValue('description'),
      icon_uri: formRef.current?.formApi.getValue('icon_uri')?.[0].uid,
      space_id: spaceId || undefined,
    });
    return datasetId;
  };

  const { open, close, modal } = useDataModalWithCoze({
    title: (
      <div data-testid={KnowledgeE2e.CreateKnowledgeModalTitle}>
        {I18n.t('datasets_model_create_title')}
      </div>
    ),
    centered: true,
    className: styles['create-knowledge-modal'],
    onCancel: () => {
      close();
    },
    footer: (
      <div className="flex w-full justify-end">
        <Button
          color="primary"
          onClick={() => {
            close();
          }}
        >
          {I18n.t('cancel')}
        </Button>
        <LoadingButton
          color="primary"
          onClick={async () => {
            beforeCreate?.(false);
            const datasetId = await createDataset();
            if (onFinish) {
              // Use unitTypeRef.current to get the latest value (avoid closure issues)
              onFinish(datasetId || '', unitTypeRef.current, false);
            } else {
              resourceNavigate.toResource?.('knowledge', datasetId);
            }
          }}
        >
          {I18n.t('kl_write_108')}
        </LoadingButton>
        <LoadingButton
          data-testid={KnowledgeE2e.CreateKnowledgeModalSubmitAndImportButton}
          color="primary"
          onClick={async () => {
            beforeCreate?.(true);
            const datasetId = await createDataset();
            if (onFinish) {
              // Use unitTypeRef.current to get the latest value (avoid closure issues)
              onFinish(datasetId || '', unitTypeRef.current, true);
            } else {
              resourceNavigate.upload?.({ type: unitTypeRef.current });
            }
          }}
        >
          {I18n.t('kl_write_109')}
        </LoadingButton>
      </div>
    ),
  });

  return {
    modal: modal(
      <Form<CozeKnowledgeAddTypeContentFormData>
        ref={formRef}
        showValidateIcon={false}
        className={styles['create-form']}
      >
        <CozeKnowledgeAddTypeContent
          onImportKnowledgeTypeChange={handleUnitTypeChange}
          onSelectFormatTypeChange={setCurrentFormatType}
        />
      </Form>,
    ),
    open: () => {
      setCurrentFormatType(FormatType.Text);
      setUnitType(UnitType.TEXT_DOC);
      unitTypeRef.current = UnitType.TEXT_DOC;
      open();
    },
    close,
  };
};
