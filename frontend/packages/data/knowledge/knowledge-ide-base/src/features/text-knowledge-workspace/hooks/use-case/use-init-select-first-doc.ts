/*
 * Copyright 2025 coze-dev Authors
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

import { useEffect } from 'react';

import { useKnowledgeStore } from '@coze-data/knowledge-stores';
import { DocumentStatus } from '@coze-arch/bot-api/knowledge';

import { useDocumentManagement } from './use-document-management';

export const useInitSelectFirstDoc = () => {
  const documentList = useKnowledgeStore(state => state.documentList);
  const { handleSelectDocument } = useDocumentManagement();
  useEffect(() => {
    if (documentList?.length) {
      // Prefer enabled documents (status = 1) to show documents with slices
      const enabledDoc = documentList.find(
        doc => doc.status === DocumentStatus.Enable,
      );
      const docId =
        enabledDoc?.document_id ?? documentList[0]?.document_id ?? '';
      handleSelectDocument(docId);
    }
  }, [documentList]);
};
