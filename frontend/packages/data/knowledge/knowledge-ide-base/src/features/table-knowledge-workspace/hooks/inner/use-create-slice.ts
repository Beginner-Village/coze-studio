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

import { useKnowledgeStore } from '@coze-data/knowledge-stores';
import { FormatType } from '@coze-arch/bot-api/knowledge';

import { type ISliceInfo } from '@/types/slice';
import { useCreateSlice as useCreateSliceService } from '@/service/slice';

import { useTableData } from '../../context/table-data-context';
import { useTableActions } from '../../context/table-actions-context';

export const useCreateSlice = () => {
  const dataSetDetail = useKnowledgeStore(state => state.dataSetDetail);
  const setDataSetDetail = useKnowledgeStore(state => state.setDataSetDetail);
  const { sliceListData } = useTableData();
  const { mutateSliceListData } = useTableActions();
  const documentList = useKnowledgeStore(state => state.documentList);
  const curDoc = documentList?.[0];

  // Create slice
  const { createSlice } = useCreateSliceService({
    onReload: (createItem: ISliceInfo) => {
      const list =
        (sliceListData?.list ?? []).filter(item => !item.addId) ?? [];
      const isQAFormat = dataSetDetail?.format_type === FormatType.QA;

      if (isQAFormat) {
        let question = createItem.content ?? '';
        let answer = createItem.answer ?? '';
        try {
          const parsed = JSON.parse(createItem.content ?? '{}') as {
            question?: string;
            answer?: string;
            content?: string;
          };
          question = parsed.question ?? parsed.content ?? question;
          answer = parsed.answer ?? answer;
        } catch {
          // keep original values
        }
        list.push({
          ...createItem,
          content: question,
          answer,
        });
      } else {
        const createSliceContent = JSON.parse(createItem.content ?? '{}');
        const itemContent = (curDoc?.table_meta ?? []).reduce(
          (
            prev: { column_name: string; column_id: string; value: string }[],
            cur,
          ) => {
            prev.push({
              column_name: cur?.column_name ?? '',
              column_id: cur?.id ?? '',
              value: cur.id ? createSliceContent[cur.id] : '',
            });
            return prev;
          },
          [],
        );
        list.push({
          ...createItem,
          content: JSON.stringify(itemContent),
        });
      }

      mutateSliceListData({
        ...sliceListData,
        total: Number(sliceListData?.total ?? '0'),
        list,
      });

      if (dataSetDetail) {
        setDataSetDetail({
          ...dataSetDetail,
          slice_count: list?.length ?? 0,
        });
      }
    },
  });

  return {
    createSlice,
  };
};
