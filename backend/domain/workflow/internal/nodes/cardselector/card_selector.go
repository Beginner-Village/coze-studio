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

package cardselector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/canvas/convert"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/schema"
)

const (
	InputKeySearchKeyword = "search_keyword"
	InputKeyCardFilters   = "card_filters"
	InputKeySelectedCardID = "selected_card_id"
	
	OutputKeySelectedCard = "selected_card"
	OutputKeyCardID       = "card_id"
	OutputKeyCardName     = "card_name"
	OutputKeyCardDesc     = "card_description"
	OutputKeyCards        = "cards"
	OutputKeyCount        = "count"
)

// FalconCard represents a card from the Falcon platform
type FalconCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// FalconAPIResponse represents the API response from Falcon platform
type FalconAPIResponse struct {
	Cards []FalconCard `json:"cards"`
	Total int          `json:"total"`
	Code  int          `json:"code"`
	Msg   string       `json:"msg"`
}

// Config implements NodeAdaptor and NodeBuilder interfaces
type Config struct {
	APIEndpoint string `json:"api_endpoint,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	Timeout     int    `json:"timeout,omitempty"` // timeout in seconds
}

// Adapt implements NodeAdaptor interface
func (c *Config) Adapt(ctx context.Context, n *vo.Node, opts ...nodes.AdaptOption) (*schema.NodeSchema, error) {
	ns := &schema.NodeSchema{
		Key:     vo.NodeKey(n.ID),
		Type:    entity.NodeTypeCardSelector,
		Name:    n.Data.Meta.Title,
		Configs: c,
	}

	// 设置输入字段类型和映射信息
	if err := convert.SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	// 设置输出字段类型信息
	if err := convert.SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

// Build implements NodeBuilder interface
func (c *Config) Build(ctx context.Context, ns *schema.NodeSchema, opts ...schema.BuildOption) (any, error) {
	timeout := 30 // default timeout 30 seconds
	if c.Timeout > 0 {
		timeout = c.Timeout
	}

	return &CardSelector{
		apiEndpoint: c.APIEndpoint,
		apiKey:      c.APIKey,
		timeout:     timeout,
	}, nil
}

// CardSelector is the actual node implementation
type CardSelector struct {
	apiEndpoint string
	apiKey      string
	timeout     int
}

// Invoke implements InvokableNode interface
func (cs *CardSelector) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 检查是否有选定的卡片ID，如果有则直接返回该卡片信息
	if selectedCardID, ok := input[InputKeySelectedCardID].(string); ok && selectedCardID != "" {
		card, err := cs.fetchCardByID(ctx, selectedCardID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch selected card: %w", err)
		}

		return map[string]any{
			OutputKeySelectedCard: map[string]any{
				"id":          card.ID,
				"name":        card.Name,
				"description": card.Description,
				"category":    card.Category,
			},
			OutputKeyCardID:   card.ID,
			OutputKeyCardName: card.Name,
			OutputKeyCardDesc: card.Description,
		}, nil
	}

	// 如果没有选定的卡片ID，则进行搜索
	searchKeyword := ""
	if keyword, ok := input[InputKeySearchKeyword].(string); ok {
		searchKeyword = keyword
	}

	// 处理卡片筛选条件
	filters := make(map[string]any)
	if filterData, ok := input[InputKeyCardFilters]; ok {
		if filterMap, ok := filterData.(map[string]any); ok {
			filters = filterMap
		}
	}

	// 调用猎鹰平台API获取卡片
	cards, err := cs.searchCardsFromAPI(ctx, searchKeyword, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search cards from API: %w", err)
	}

	// 准备输出结果
	cardsOutput := make([]map[string]any, len(cards))
	for i, card := range cards {
		cardsOutput[i] = map[string]any{
			"id":          card.ID,
			"name":        card.Name,
			"description": card.Description,
			"category":    card.Category,
		}
	}

	result := map[string]any{
		OutputKeyCards: cardsOutput,
		OutputKeyCount: len(cards),
	}

	// 如果只有一张卡片，自动选择它
	if len(cards) == 1 {
		card := cards[0]
		result[OutputKeySelectedCard] = cardsOutput[0]
		result[OutputKeyCardID] = card.ID
		result[OutputKeyCardName] = card.Name
		result[OutputKeyCardDesc] = card.Description
	}

	return result, nil
}

// searchCardsFromAPI 调用猎鹰平台API搜索卡片
func (cs *CardSelector) searchCardsFromAPI(ctx context.Context, searchKeyword string, filters map[string]any) ([]FalconCard, error) {
	if cs.apiEndpoint == "" {
		// 如果没有配置API端点，返回模拟数据
		return cs.getMockCards(searchKeyword), nil
	}

	// 构建API请求URL
	apiURL := strings.TrimSuffix(cs.apiEndpoint, "/") + "/api/cards/search"
	
	// 构建查询参数
	params := url.Values{}
	if searchKeyword != "" {
		params.Add("q", searchKeyword)
	}
	
	// 添加筛选条件到查询参数
	for key, value := range filters {
		if strValue, ok := value.(string); ok && strValue != "" {
			params.Add(key, strValue)
		}
	}

	if len(params) > 0 {
		apiURL += "?" + params.Encode()
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置认证头
	if cs.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cs.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: time.Duration(cs.timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析响应
	var apiResp FalconAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查API响应码
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API returned error: %s (code: %d)", apiResp.Msg, apiResp.Code)
	}

	return apiResp.Cards, nil
}

// fetchCardByID 根据ID获取特定卡片信息
func (cs *CardSelector) fetchCardByID(ctx context.Context, cardID string) (*FalconCard, error) {
	if cs.apiEndpoint == "" {
		// 如果没有配置API端点，返回模拟数据
		mockCards := cs.getMockCards("")
		for _, card := range mockCards {
			if card.ID == cardID {
				return &card, nil
			}
		}
		return nil, fmt.Errorf("card not found: %s", cardID)
	}

	// 构建API请求URL
	apiURL := strings.TrimSuffix(cs.apiEndpoint, "/") + "/api/cards/" + cardID

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置认证头
	if cs.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cs.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: time.Duration(cs.timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析响应
	var card FalconCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &card, nil
}

// getMockCards 返回模拟卡片数据，用于开发和测试
func (cs *CardSelector) getMockCards(searchKeyword string) []FalconCard {
	mockCards := []FalconCard{
		{
			ID:          "card_001",
			Name:        "用户注册卡片",
			Description: "处理用户注册相关功能的卡片",
			Category:    "user_management",
		},
		{
			ID:          "card_002", 
			Name:        "数据分析卡片",
			Description: "提供数据分析和报表生成功能",
			Category:    "analytics",
		},
		{
			ID:          "card_003",
			Name:        "消息通知卡片", 
			Description: "发送各种类型的消息通知",
			Category:    "notification",
		},
		{
			ID:          "card_004",
			Name:        "文件处理卡片",
			Description: "处理文件上传、下载和转换功能",
			Category:    "file_management",
		},
		{
			ID:          "card_005",
			Name:        "支付处理卡片",
			Description: "集成支付网关，处理支付流程",
			Category:    "payment",
		},
	}

	// 如果有搜索关键词，进行简单的过滤
	if searchKeyword != "" {
		var filteredCards []FalconCard
		keyword := strings.ToLower(searchKeyword)
		for _, card := range mockCards {
			if strings.Contains(strings.ToLower(card.Name), keyword) ||
				strings.Contains(strings.ToLower(card.Description), keyword) ||
				strings.Contains(strings.ToLower(card.Category), keyword) {
				filteredCards = append(filteredCards, card)
			}
		}
		return filteredCards
	}

	return mockCards
}