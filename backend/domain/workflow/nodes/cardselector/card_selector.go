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
	"strings"
	"time"

	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/canvas/convert"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/schema"
)

const (
	InputKeySearchKeyword  = "search_keyword"
	InputKeyCardFilters    = "card_filters"
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
	CardID          string `json:"cardId"`
	CardName        string `json:"cardName"`
	Code            string `json:"code"`
	CardClassID     string `json:"cardClassId,omitempty"`
	CardPicURL      string `json:"cardPicUrl,omitempty"`
	CardShelfStatus string `json:"cardShelfStatus,omitempty"`
	CardShelfTime   string `json:"cardShelfTime,omitempty"`
	CreateUserID    string `json:"createUserId,omitempty"`
	CreateUserName  string `json:"createUserName,omitempty"`
	PicURL          string `json:"picUrl,omitempty"`
	SassAppID       string `json:"sassAppId,omitempty"`
	SassWorkspaceID string `json:"sassWorkspaceId,omitempty"`
	BizChannel      string `json:"bizChannel,omitempty"`
}

// FalconAPIRequest represents the API request body
type FalconAPIRequest struct {
	Body FalconRequestBody `json:"body"`
}

type FalconRequestBody struct {
	AgentID             string          `json:"agentId"`
	ApplyScene          string          `json:"applyScene"`
	CardClassID         string          `json:"cardClassId"`
	CardCode            string          `json:"cardCode"`
	CardID              string          `json:"cardId"`
	CardName            string          `json:"cardName"`
	CardPicURL          string          `json:"cardPicUrl"`
	Channel             string          `json:"channel"`
	Code                string          `json:"code"`
	CreateTime          string          `json:"createTime"`
	CreatedBy           bool            `json:"createdBy"`
	GreyConfigInfo      string          `json:"greyConfigInfo"`
	GreyNum             string          `json:"greyNum"`
	ID                  string          `json:"id"`
	IsAdd               string          `json:"isAdd"`
	JSFileURL           string          `json:"jsFileUrl"`
	MainURL             string          `json:"mainUrl"`
	Memo                string          `json:"memo"`
	ModuleName          string          `json:"moduleName"`
	PageNo              string          `json:"pageNo"`
	PageSize            string          `json:"pageSize"`
	PicURL              string          `json:"picUrl"`
	Platform            string          `json:"platform"`
	PlatformStatus      string          `json:"platformStatus"`
	PlatformValue       string          `json:"platformValue"`
	PreviewSchema       string          `json:"previewSchema"`
	PublishMode         string          `json:"publishMode"`
	PublishStatus       string          `json:"publishStatus"`
	PublishType         string          `json:"publishType"`
	RealGreyEndtime     string          `json:"realGreyEndtime"`
	ResourceType        string          `json:"resourceType"`
	SassAppID           string          `json:"sassAppId"`
	SassWorkspaceID     string          `json:"sassWorkspaceId"`
	SchemaValue         string          `json:"schemaValue"`
	SearchValue         string          `json:"searchValue"`
	ServiceModuleID     string          `json:"serviceModuleId"`
	ServiceName         string          `json:"serviceName"`
	SkeletonScreen      string          `json:"skeletonScreen"`
	SoLib               string          `json:"soLib"`
	StaticMemo          string          `json:"staticMemo"`
	StaticType          string          `json:"staticType"`
	StaticVersion       string          `json:"staticVersion"`
	TaskID              string          `json:"taskId"`
	TaskStatus          string          `json:"taskStatus"`
	TemplateID          string          `json:"templateId"`
	TemplateName        string          `json:"templateName"`
	TemplateSchemaValue string          `json:"templateSchemaValue"`
	UnzipPath           string          `json:"unzipPath"`
	UserID              string          `json:"userId"`
	VariableValueList   []VariableValue `json:"variableValueList"`
	Version             string          `json:"version"`
	VersionID           string          `json:"versionId"`
	WhitelistIDs        string          `json:"whitelistIds"`
	WhlBusiness         string          `json:"whlBusiness"`
}

type VariableValue struct {
	BizChannel           string `json:"bizChannel"`
	VariableDefaultValue string `json:"variableDefaultValue"`
	VariableDescribe     string `json:"variableDescribe"`
	VariableKey          string `json:"variableKey"`
	VariableName         string `json:"variableName"`
	VariableStructure    string `json:"variableStructure"`
	VariableType         string `json:"variableType"`
}

// FalconAPIResponse represents the API response from Falcon platform
type FalconAPIResponse struct {
	Header APIHeader `json:"header"`
	Body   APIBody   `json:"body"`
}

type APIHeader struct {
	ICIFID      interface{} `json:"iCIFID"`
	ECIFID      interface{} `json:"eCIFID"`
	ErrorCode   string      `json:"errorCode"`
	ErrorMsg    string      `json:"errorMsg"`
	Encry       interface{} `json:"encry"`
	TransCode   interface{} `json:"transCode"`
	Channel     interface{} `json:"channel"`
	ChannelDate interface{} `json:"channelDate"`
	ChannelTime interface{} `json:"channelTime"`
	ChannelFlow interface{} `json:"channelFlow"`
	Type        interface{} `json:"type"`
	TransID     interface{} `json:"transId"`
}

type APIBody struct {
	CardList   []FalconCard `json:"cardList"`
	ErrorCode  string       `json:"errorCode"`
	ErrorMsg   string       `json:"errorMsg"`
	PageNo     string       `json:"pageNo"`
	PageSize   string       `json:"pageSize"`
	TotalNums  string       `json:"totalNums"`
	TotalPages string       `json:"totalPages"`
}

// CardParam represents card parameter information
type CardParam struct {
	ParamID         string      `json:"paramId"`
	ParamName       string      `json:"paramName"`
	ParamType       string      `json:"paramType"`
	ParamDesc       string      `json:"paramDesc"`
	IsRequired      string      `json:"isRequired"`
	BizChannel      string      `json:"bizChannel,omitempty"`
	SassAppID       string      `json:"sassAppId,omitempty"`
	SassWorkspaceID string      `json:"sassWorkspaceId,omitempty"`
	Children        []CardParam `json:"children,omitempty"`
}

// CardDetailResponse represents the card detail API response
type CardDetailResponse struct {
	Header APIHeader      `json:"header"`
	Body   CardDetailBody `json:"body"`
}

type CardDetailBody struct {
	CardID         string      `json:"cardId"`
	CardName       string      `json:"cardName"`
	CardPicURL     string      `json:"cardPicUrl"`
	Code           string      `json:"code"`
	ErrorCode      string      `json:"errorCode"`
	ErrorMsg       string      `json:"errorMsg"`
	MainURL        string      `json:"mainUrl"`
	ParamList      []CardParam `json:"paramList"`
	SkeletonScreen string      `json:"skeletonScreen"`
	Version        string      `json:"version"`
}

// Config implements NodeAdaptor and NodeBuilder interfaces
type Config struct {
	APIEndpoint string `json:"api_endpoint,omitempty"`
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
		ApiEndpoint: c.APIEndpoint,
		Timeout:     timeout,
	}, nil
}

// CardSelector is the actual node implementation
type CardSelector struct {
	ApiEndpoint string
	Timeout     int
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
				"id":          card.CardID,
				"name":        card.CardName,
				"description": card.Code,
				"category":    card.CardClassID,
			},
			OutputKeyCardID:   card.CardID,
			OutputKeyCardName: card.CardName,
			OutputKeyCardDesc: card.Code,
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
			"id":          card.CardID,
			"name":        card.CardName,
			"description": card.Code,
			"category":    card.CardClassID,
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
		result[OutputKeyCardID] = card.CardID
		result[OutputKeyCardName] = card.CardName
		result[OutputKeyCardDesc] = card.Code
	}

	return result, nil
}

// searchCardsFromAPI 调用猎鹰平台API搜索卡片
func (cs *CardSelector) searchCardsFromAPI(ctx context.Context, searchKeyword string, filters map[string]any) ([]FalconCard, error) {
	if cs.ApiEndpoint == "" {
		// 如果没有配置API端点，返回模拟数据
		return cs.getMockCards(searchKeyword), nil
	}

	// 构建API请求体
	requestBody := FalconAPIRequest{
		Body: FalconRequestBody{
			AgentID:             "",
			ApplyScene:          "",
			CardClassID:         "",
			CardCode:            "",
			CardID:              "",
			CardName:            "",
			CardPicURL:          "",
			Channel:             "",
			Code:                "",
			CreateTime:          "",
			CreatedBy:           true,
			GreyConfigInfo:      "",
			GreyNum:             "",
			ID:                  "",
			IsAdd:               "",
			JSFileURL:           "",
			MainURL:             "",
			Memo:                "",
			ModuleName:          "",
			PageNo:              "1",
			PageSize:            "50",
			PicURL:              "",
			Platform:            "",
			PlatformStatus:      "",
			PlatformValue:       "",
			PreviewSchema:       "",
			PublishMode:         "",
			PublishStatus:       "",
			PublishType:         "",
			RealGreyEndtime:     "",
			ResourceType:        "",
			SassAppID:           "",
			SassWorkspaceID:     "",
			SchemaValue:         "",
			SearchValue:         searchKeyword,
			ServiceModuleID:     "",
			ServiceName:         "",
			SkeletonScreen:      "",
			SoLib:               "",
			StaticMemo:          "",
			StaticType:          "",
			StaticVersion:       "",
			TaskID:              "",
			TaskStatus:          "",
			TemplateID:          "",
			TemplateName:        "",
			TemplateSchemaValue: "",
			UnzipPath:           "",
			UserID:              "",
			VariableValueList: []VariableValue{
				{
					BizChannel:           "",
					VariableDefaultValue: "",
					VariableDescribe:     "",
					VariableKey:          "",
					VariableName:         "",
					VariableStructure:    "",
					VariableType:         "",
				},
			},
			Version:      "",
			VersionID:    "",
			WhitelistIDs: "",
			WhlBusiness:  "",
		},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 自动拼接卡片列表接口路径
	listAPIURL := cs.ApiEndpoint
	if listAPIURL == "" {
		listAPIURL = "http://10.10.10.208:8500/aop-web"
	}
	if !strings.HasSuffix(listAPIURL, "/IDC10030.do") {
		listAPIURL = strings.TrimSuffix(listAPIURL, "/") + "/IDC10030.do"
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", listAPIURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Request-Origin", "SwaggerBootstrapUi")
	req.Header.Set("Accept", "*/*")

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: time.Duration(cs.Timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		// 连接失败时优雅降级到mock数据
		if strings.Contains(err.Error(), "connection refused") || 
		   strings.Contains(err.Error(), "no such host") || 
		   strings.Contains(err.Error(), "timeout") {
			// 记录警告但不中断服务
			fmt.Printf("⚠️ Falcon API connection failed (%s), falling back to mock data\n", err.Error())
			return cs.getMockCards(searchKeyword), nil
		}
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
	if apiResp.Header.ErrorCode != "0" {
		return nil, fmt.Errorf("API returned error: %s (code: %s)", apiResp.Header.ErrorMsg, apiResp.Header.ErrorCode)
	}

	return apiResp.Body.CardList, nil
}

// fetchCardByID 根据ID获取特定卡片信息
func (cs *CardSelector) fetchCardByID(ctx context.Context, cardID string) (*FalconCard, error) {
	if cs.ApiEndpoint == "" {
		// 如果没有配置API端点，返回模拟数据
		mockCards := cs.getMockCards("")
		for _, card := range mockCards {
			if card.CardID == cardID {
				return &card, nil
			}
		}
		return nil, fmt.Errorf("card not found: %s", cardID)
	}

	// 构建卡片详情API请求体
	requestBody := FalconAPIRequest{
		Body: FalconRequestBody{
			AgentID:             "",
			ApplyScene:          "",
			CardClassID:         "",
			CardCode:            "",
			CardID:              cardID, // 设置要查询的卡片ID
			CardName:            "",
			CardPicURL:          "",
			Channel:             "",
			Code:                "",
			CreateTime:          "",
			CreatedBy:           true,
			GreyConfigInfo:      "",
			GreyNum:             "",
			ID:                  "",
			IsAdd:               "",
			JSFileURL:           "",
			MainURL:             "",
			Memo:                "",
			ModuleName:          "",
			PageNo:              "",
			PageSize:            "",
			PicURL:              "",
			Platform:            "",
			PlatformStatus:      "",
			PlatformValue:       "",
			PreviewSchema:       "",
			PublishMode:         "",
			PublishStatus:       "",
			PublishType:         "",
			RealGreyEndtime:     "",
			ResourceType:        "",
			SassAppID:           "",
			SassWorkspaceID:     "",
			SchemaValue:         "",
			SearchValue:         "",
			ServiceModuleID:     "",
			ServiceName:         "",
			SkeletonScreen:      "",
			SoLib:               "",
			StaticMemo:          "",
			StaticType:          "",
			StaticVersion:       "",
			TaskID:              "",
			TaskStatus:          "",
			TemplateID:          "",
			TemplateName:        "",
			TemplateSchemaValue: "",
			UnzipPath:           "",
			UserID:              "",
			VariableValueList: []VariableValue{
				{
					BizChannel:           "",
					VariableDefaultValue: "",
					VariableDescribe:     "",
					VariableKey:          "",
					VariableName:         "",
					VariableStructure:    "",
					VariableType:         "",
				},
			},
			Version:      "",
			VersionID:    "",
			WhitelistIDs: "",
			WhlBusiness:  "",
		},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 构建卡片详情API URL
	detailAPIURL := cs.ApiEndpoint
	if detailAPIURL == "" {
		detailAPIURL = "http://10.10.10.208:8500/aop-web"
	}
	if !strings.HasSuffix(detailAPIURL, "/IDC10025.do") {
		detailAPIURL = strings.TrimSuffix(detailAPIURL, "/") + "/IDC10025.do"
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", detailAPIURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Request-Origin", "SwaggerBootstrapUi")
	req.Header.Set("Accept", "*/*")

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: time.Duration(cs.Timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		// 连接失败时优雅降级到mock数据
		if strings.Contains(err.Error(), "connection refused") || 
		   strings.Contains(err.Error(), "no such host") || 
		   strings.Contains(err.Error(), "timeout") {
			// 记录警告但不中断服务
			fmt.Printf("⚠️ Falcon API connection failed (%s), falling back to mock data\n", err.Error())
			mockCards := cs.getMockCards("")
			for _, card := range mockCards {
				if card.CardID == cardID {
					return &card, nil
				}
			}
			return nil, fmt.Errorf("card not found in mock data: %s", cardID)
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析响应
	var apiResp CardDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查API响应码
	if apiResp.Header.ErrorCode != "0" {
		return nil, fmt.Errorf("API returned error: %s (code: %s)", apiResp.Header.ErrorMsg, apiResp.Header.ErrorCode)
	}

	// 构建返回的卡片信息
	card := &FalconCard{
		CardID:     apiResp.Body.CardID,
		CardName:   apiResp.Body.CardName,
		Code:       apiResp.Body.Code,
		CardPicURL: apiResp.Body.CardPicURL,
	}

	return card, nil
}

// fetchCardDetailWithParams 获取卡片详情包括参数信息
func (cs *CardSelector) fetchCardDetailWithParams(ctx context.Context, cardID string) (*CardDetailBody, error) {
	if cs.ApiEndpoint == "" {
		// 如果没有配置API端点，返回模拟数据
		return &CardDetailBody{
			CardID:   cardID,
			CardName: "模拟卡片",
			Code:     "mock_card",
			ParamList: []CardParam{
				{
					ParamID:    "mock_param_1",
					ParamName:  "input",
					ParamType:  "string",
					ParamDesc:  "输入参数",
					IsRequired: "1",
				},
			},
		}, nil
	}

	// 构建卡片详情API请求体
	requestBody := FalconAPIRequest{
		Body: FalconRequestBody{
			CardID:    cardID,
			CreatedBy: true,
			PageNo:    "",
			PageSize:  "",
			VariableValueList: []VariableValue{
				{
					BizChannel:           "",
					VariableDefaultValue: "",
					VariableDescribe:     "",
					VariableKey:          "",
					VariableName:         "",
					VariableStructure:    "",
					VariableType:         "",
				},
			},
		},
	}

	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 构建卡片详情API URL
	detailAPIURL := cs.ApiEndpoint
	if detailAPIURL == "" {
		detailAPIURL = "http://10.10.10.208:8500/aop-web"
	}
	if !strings.HasSuffix(detailAPIURL, "/IDC10025.do") {
		detailAPIURL = strings.TrimSuffix(detailAPIURL, "/") + "/IDC10025.do"
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", detailAPIURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Request-Origin", "SwaggerBootstrapUi")
	req.Header.Set("Accept", "*/*")

	// 创建HTTP客户端并设置超时
	client := &http.Client{
		Timeout: time.Duration(cs.Timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		// 连接失败时优雅降级到mock数据
		if strings.Contains(err.Error(), "connection refused") || 
		   strings.Contains(err.Error(), "no such host") || 
		   strings.Contains(err.Error(), "timeout") {
			// 记录警告但不中断服务
			fmt.Printf("⚠️ Falcon API connection failed (%s), falling back to mock data\n", err.Error())
			return cs.getMockCardDetail(cardID), nil
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析响应
	var apiResp CardDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查API响应码
	if apiResp.Header.ErrorCode != "0" {
		return nil, fmt.Errorf("API returned error: %s (code: %s)", apiResp.Header.ErrorMsg, apiResp.Header.ErrorCode)
	}

	return &apiResp.Body, nil
}

// getMockCards 返回模拟卡片数据，用于开发和测试
func (cs *CardSelector) getMockCards(searchKeyword string) []FalconCard {
	mockCards := []FalconCard{
		{
			CardID:          "10000084",
			CardName:        "转账步骤-选择联系人_",
			Code:            "accountCreditedlist",
			CardClassID:     "2",
			CardShelfStatus: "1",
			PicURL:          "@filestore/sit-public-cbbiz/20250507160448_ic_95782.png",
			SassAppID:       "100001",
			SassWorkspaceID: "dev",
		},
		{
			CardID:          "card_002",
			CardName:        "数据分析卡片",
			Code:            "data_analytics",
			CardClassID:     "1",
			CardShelfStatus: "1",
			SassAppID:       "100001",
			SassWorkspaceID: "dev",
		},
		{
			CardID:          "card_003",
			CardName:        "消息通知卡片",
			Code:            "notification",
			CardClassID:     "3",
			CardShelfStatus: "1",
			SassAppID:       "100001",
			SassWorkspaceID: "dev",
		},
		{
			CardID:          "card_004",
			CardName:        "文件处理卡片",
			Code:            "file_management",
			CardClassID:     "4",
			CardShelfStatus: "1",
			SassAppID:       "100001",
			SassWorkspaceID: "dev",
		},
		{
			CardID:          "card_005",
			CardName:        "支付处理卡片",
			Code:            "payment",
			CardClassID:     "5",
			CardShelfStatus: "1",
			SassAppID:       "100001",
			SassWorkspaceID: "dev",
		},
	}

	// 如果有搜索关键词，进行简单的过滤
	if searchKeyword != "" {
		var filteredCards []FalconCard
		keyword := strings.ToLower(searchKeyword)
		for _, card := range mockCards {
			if strings.Contains(strings.ToLower(card.CardName), keyword) ||
				strings.Contains(strings.ToLower(card.Code), keyword) ||
				strings.Contains(strings.ToLower(card.CardID), keyword) {
				filteredCards = append(filteredCards, card)
			}
		}
		return filteredCards
	}

	return mockCards
}

// SearchCards 公开方法：搜索卡片
func (cs *CardSelector) SearchCards(ctx context.Context, searchKeyword string, filters map[string]any) ([]FalconCard, error) {
	return cs.searchCardsFromAPI(ctx, searchKeyword, filters)
}

// GetCardDetail 公开方法：获取卡片详情
func (cs *CardSelector) GetCardDetail(ctx context.Context, cardID string) (*CardDetailBody, error) {
	return cs.fetchCardDetailWithParams(ctx, cardID)
}

// getMockCardDetail 返回模拟卡片详情数据，用于开发和测试
func (cs *CardSelector) getMockCardDetail(cardID string) *CardDetailBody {
	return &CardDetailBody{
		CardID:   cardID,
		CardName: "模拟卡片详情",
		Code:     "mock_card_detail",
		ParamList: []CardParam{
			{
				ParamID:    "mock_param_1",
				ParamName:  "输入文本",
				ParamType:  "string",
				ParamDesc:  "用户输入的文本内容",
				IsRequired: "1",
			},
			{
				ParamID:    "mock_param_2", 
				ParamName:  "选项配置",
				ParamType:  "object",
				ParamDesc:  "配置选项对象",
				IsRequired: "0",
			},
		},
	}
}
