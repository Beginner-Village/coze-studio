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

import type { OutputTreeMeta, Parameter } from '@coze-workflow/base';

export interface FalconCard {
  cardId: string;
  cardName: string;
  code: string;
  cardClassId?: string;
  cardPicUrl?: string;
  cardShelfStatus?: string;
  cardShelfTime?: string;
  createUserId?: string;
  createUserName?: string;
  picUrl?: string;
  sassAppId?: string;
  sassWorkspaceId?: string;
  bizChannel?: string;
}

export interface CardListResponse {
  header: {
    errorCode: string;
    errorMsg: string;
  };
  body: {
    cardList: FalconCard[];
    pageNo: string;
    pageSize: string;
    totalNums: string;
    totalPages: string;
  };
}

export interface CardParam {
  paramId: string;
  paramName: string;
  paramType: string;
  paramDesc: string;
  isRequired: string;
  bizChannel?: string;
  sassAppId?: string;
  sassWorkspaceId?: string;
  children?: CardParam[];
}

export interface CardDetail {
  cardId: string;
  cardName: string;
  cardPicUrl: string;
  code: string;
  mainUrl: string;
  paramList: CardParam[];
  skeletonScreen: string;
  version: string;
}

export interface CardDetailResponse {
  header: {
    errorCode: string;
    errorMsg: string;
  };
  body: CardDetail;
}

export interface CardSelectorParams {
  selectedCardId?: string;
}

export interface FormData {
  inputParameters: Parameter[];
  cardSelectorParams: CardSelectorParams;
  outputs: OutputTreeMeta[];
}
