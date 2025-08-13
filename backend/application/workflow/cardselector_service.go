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

package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/nodes/cardselector"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

// CardSelectorService 卡片选择器服务
type CardSelectorService struct{}

// NewCardSelectorService 创建新的卡片选择器服务
func NewCardSelectorService() *CardSelectorService {
	return &CardSelectorService{}
}

// SearchCards 搜索卡片
func (s *CardSelectorService) SearchCards(ctx context.Context, apiEndpoint, searchKeyword string, filters map[string]interface{}) ([]interface{}, error) {
	logs.CtxInfof(ctx, "CardSelectorService.SearchCards called with apiEndpoint=%s, searchKeyword=%s", apiEndpoint, searchKeyword)

	// 处理API端点
	if apiEndpoint == "" {
		// 使用默认API端点
		apiEndpoint = "http://10.10.10.208:8500/aop-web"
		logs.CtxInfof(ctx, "Using default apiEndpoint: %s", apiEndpoint)
	}
	// 自动拼接卡片列表接口路径
	if !strings.HasSuffix(apiEndpoint, "/IDC10030.do") {
		apiEndpoint = strings.TrimSuffix(apiEndpoint, "/") + "/IDC10030.do"
		logs.CtxInfof(ctx, "Auto-appended IDC10030.do, final apiEndpoint: %s", apiEndpoint)
	}

	// 直接创建CardSelector实例
	timeout := 30
	cardSelector := &cardselector.CardSelector{
		ApiEndpoint: apiEndpoint,
		Timeout:     timeout,
	}
	logs.CtxInfof(ctx, "Created CardSelector with ApiEndpoint=%s, Timeout=%d", apiEndpoint, timeout)

	// 调用搜索方法
	logs.CtxInfof(ctx, "Calling cardSelector.SearchCards")
	cardList, err := cardSelector.SearchCards(ctx, searchKeyword, filters)
	if err != nil {
		logs.CtxErrorf(ctx, "cardSelector.SearchCards failed: %v", err)
		return nil, fmt.Errorf("failed to search cards: %w", err)
	}

	logs.CtxInfof(ctx, "cardSelector.SearchCards success, found %d cards", len(cardList))

	// 转换为interface{}数组
	result := make([]interface{}, len(cardList))
	for i, card := range cardList {
		result[i] = card
	}

	logs.CtxInfof(ctx, "CardSelectorService.SearchCards completed successfully")
	return result, nil
}

// GetCardDetail 获取卡片详情
func (s *CardSelectorService) GetCardDetail(ctx context.Context, apiEndpoint, cardID string) (interface{}, error) {
	logs.CtxInfof(ctx, "CardSelectorService.GetCardDetail called with apiEndpoint=%s, cardID=%s", apiEndpoint, cardID)

	// 处理API端点
	if apiEndpoint == "" {
		// 使用默认API端点
		apiEndpoint = "http://10.10.10.208:8500/aop-web"
		logs.CtxInfof(ctx, "Using default apiEndpoint: %s", apiEndpoint)
	}

	// 直接创建CardSelector实例
	timeout := 30
	cardSelector := &cardselector.CardSelector{
		ApiEndpoint: apiEndpoint,
		Timeout:     timeout,
	}
	logs.CtxInfof(ctx, "Created CardSelector with ApiEndpoint=%s, Timeout=%d", apiEndpoint, timeout)

	// 调用获取详情方法
	logs.CtxInfof(ctx, "Calling cardSelector.GetCardDetail")
	cardDetail, err := cardSelector.GetCardDetail(ctx, cardID)
	if err != nil {
		logs.CtxErrorf(ctx, "cardSelector.GetCardDetail failed: %v", err)
		return nil, fmt.Errorf("failed to get card detail: %w", err)
	}

	logs.CtxInfof(ctx, "CardSelectorService.GetCardDetail completed successfully")
	return cardDetail, nil
}
