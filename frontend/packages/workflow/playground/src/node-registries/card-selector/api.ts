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

import type { CardItem } from './types';

console.log('[Card Selector API] 使用直接调用 /aop-web/ 的新版本');

// 外部卡片API响应类型
interface ExternalCardListResponse {
  header: {
    errorCode: string;
    errorMsg: string;
  };
  body: {
    cardList: Array<{
      cardId: string;
      cardName: string;
      code: string;
      cardPicUrl: string;
      picUrl: string;
      cardShelfStatus: string;
      cardShelfTime: string;
      createUserId: string;
      createUserName: string;
      sassAppId: string;
      sassWorkspaceId: string;
      bizChannel: string;
      cardClassId: string;
    }>;
    pageNo: string;
    pageSize: string;
    totalNums: string;
    totalPages: string;
  };
}

interface ExternalCardDetailResponse {
  header: {
    errorCode: string;
    errorMsg: string;
  };
  body: {
    cardId: string;
    cardName: string;
    code: string;
    cardPicUrl: string;
    picUrl: string;
    cardShelfStatus: string;
    cardShelfTime: string;
    createUserId: string;
    createUserName: string;
    sassAppId: string;
    sassWorkspaceId: string;
    bizChannel: string;
    cardClassId: string;
    paramList: Array<{
      paramName: string;
      paramType: string;
      isRequired: string; // "0" 或 "1"
      paramDesc: string;
      children?: Array<{
        paramName: string;
        paramType: string;
        isRequired: string;
        paramDesc: string;
      }>;
    }>;
  };
}

/**
 * 获取卡片列表 - 直接调用外部API
 * @param params 请求参数
 * @returns 卡片列表响应
 */
export async function fetchCardList(params: {
  sassWorkspaceId: string;
  pageNo?: number;
  pageSize?: number;
  searchValue?: string;
}): Promise<{ cardList: CardItem[]; totalNums: string; totalPages: string }> {
  const { sassWorkspaceId, pageNo = 1, pageSize = 200, searchValue } = params;

  console.log('[Card Selector API] fetchCardList 被调用，参数:', {
    sassWorkspaceId,
    pageNo,
    pageSize,
    searchValue,
  });

  try {
    // 构造外部API请求体
    const requestBody = {
      body: {
        sassWorkspaceId,
        pageNo: String(pageNo),
        pageSize: String(pageSize),
        searchValue: searchValue || '',
        cardName: '',
        cardCode: '',
        createdBy: true,
        variableValueList: [{}],
      },
    };

    console.log(
      '[Card Selector API] 准备发送请求到 /aop-web/IDC10001.do，请求体:',
      requestBody,
    );

    const response = await fetch('/aop-web/IDC10001.do', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Request-Origion': 'SwaggerBootstrapUi',
        Accept: '*/*',
      },
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      throw new Error(`HTTP Error: ${response.status}`);
    }

    const data: ExternalCardListResponse = await response.json();

    if (data.header.errorCode !== '0') {
      throw new Error(`API Error: ${data.header.errorMsg}`);
    }

    // 转换为本地类型格式
    const cardList: CardItem[] = data.body.cardList.map(card => ({
      cardId: card.cardId,
      cardName: card.cardName,
      code: card.code,
      cardPicUrl: card.cardPicUrl,
      picUrl: card.picUrl,
      cardShelfStatus: card.cardShelfStatus,
      cardShelfTime: card.cardShelfTime,
      createUserId: card.createUserId,
      createUserName: card.createUserName,
      sassAppId: card.sassAppId,
      sassWorkspaceId: card.sassWorkspaceId,
      bizChannel: card.bizChannel,
      cardClassId: card.cardClassId,
    }));

    return {
      cardList,
      totalNums: data.body.totalNums,
      totalPages: data.body.totalPages,
    };
  } catch (error) {
    console.error('Failed to fetch card list:', error);
    throw error;
  }
}

/**
 * 获取卡片详情 - 直接调用外部API
 * @param params 请求参数
 * @returns 卡片详情响应
 */
export async function fetchCardDetail(params: {
  cardId: string;
  sassWorkspaceId: string;
}): Promise<{
  cardDetail: CardItem & {
    paramList?: Array<{
      paramName: string;
      paramType: string;
      required: boolean;
      desc?: string;
      children?: Array<{
        paramName: string;
        paramType: string;
        required: boolean;
        desc?: string;
      }>;
    }>;
  };
}> {
  const { cardId, sassWorkspaceId } = params;

  try {
    // 构造外部API请求体
    const requestBody = {
      body: {
        cardId,
        sassWorkspaceId,
      },
    };

    const response = await fetch('/aop-web/IDC10025.do', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Request-Origion': 'SwaggerBootstrapUi',
        Accept: '*/*',
      },
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      throw new Error(`HTTP Error: ${response.status}`);
    }

    const data: ExternalCardDetailResponse = await response.json();

    if (data.header.errorCode !== '0') {
      throw new Error(`API Error: ${data.header.errorMsg}`);
    }

    // 转换为本地类型格式
    const cardDetail = {
      cardId: data.body.cardId,
      cardName: data.body.cardName,
      code: data.body.code,
      cardPicUrl: data.body.cardPicUrl,
      picUrl: data.body.picUrl,
      cardShelfStatus: data.body.cardShelfStatus,
      cardShelfTime: data.body.cardShelfTime,
      createUserId: data.body.createUserId,
      createUserName: data.body.createUserName,
      sassAppId: data.body.sassAppId,
      sassWorkspaceId: data.body.sassWorkspaceId,
      bizChannel: data.body.bizChannel,
      cardClassId: data.body.cardClassId,
      paramList: data.body.paramList?.map(param => ({
        paramName: param.paramName,
        paramType: param.paramType,
        required: param.isRequired === '1', // "1" 表示必需
        desc: param.paramDesc,
        children: param.children?.map(child => ({
          paramName: child.paramName,
          paramType: child.paramType,
          required: child.isRequired === '1',
          desc: child.paramDesc,
        })),
      })),
    };

    return { cardDetail };
  } catch (error) {
    console.error('Failed to fetch card detail:', error);
    throw error;
  }
}
