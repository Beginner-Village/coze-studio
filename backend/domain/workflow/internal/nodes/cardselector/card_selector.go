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
	"github.com/coze-dev/coze-studio/backend/pkg/sonic"
)

const (
	InputKeySearchKeyword    = "search_keyword"
	InputKeyCardFilters      = "card_filters"
	InputKeySelectedCardID   = "selected_card_id"
	InputKeySelectedCard     = "selected_card"
	InputKeyInputParameters  = "input_parameters"

	OutputKeySelectedCard = "selected_card"
	OutputKeyCardID       = "card_id"
	OutputKeyCardName     = "card_name"
	OutputKeyCardDesc     = "card_description"
	OutputKeyCards        = "cards"
	OutputKeyCount        = "count"
	// 新增的模板输出键
	OutputKeyTemplateResponse = "template_response"
)

// 辅助函数：获取map的所有键
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

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
	APIEndpoint    string                 `json:"api_endpoint,omitempty"`
	Timeout        int                    `json:"timeout,omitempty"` // timeout in seconds
	SelectedCard   map[string]interface{} `json:"selected_card,omitempty"`   // 前端选择的卡片信息
	SelectedCardID string                 `json:"selected_card_id,omitempty"` // 选择的卡片ID
}

// Adapt implements NodeAdaptor interface
func (c *Config) Adapt(ctx context.Context, n *vo.Node, opts ...nodes.AdaptOption) (*schema.NodeSchema, error) {
	if n == nil {
		return nil, fmt.Errorf("node cannot be nil")
	}

	var name string
	if n.Data != nil && n.Data.Meta != nil {
		name = n.Data.Meta.Title
	}

	// 🚨 调试：打印前端传来的节点数据
	fmt.Printf("🔍 [CardSelector Adapt] Processing node: %s\n", n.ID)
	if n.Data != nil {
		dataBytes, _ := sonic.Marshal(n.Data)
		fmt.Printf("🔍 [CardSelector Adapt] Node data: %s\n", string(dataBytes))
	}

	// Parse configuration data from node data (not just inputs)
	if n.Data != nil {
		// Try to extract from the raw data structure
		dataBytes, err := sonic.Marshal(n.Data)
		if err == nil {
			var dataMap map[string]interface{}
			if err := sonic.Unmarshal(dataBytes, &dataMap); err == nil {
				fmt.Printf("🔍 [CardSelector Adapt] Parsed dataMap: %+v\n", dataMap)
				
				// Look for snake_case or camelCase variants in the top level
				if val, ok := dataMap["selected_card"]; ok {
					if cardMap, ok := val.(map[string]interface{}); ok {
						c.SelectedCard = cardMap
						fmt.Printf("✅ [CardSelector Adapt] Found selected_card: %+v\n", cardMap)
					}
				}
				if val, ok := dataMap["selected_card_id"]; ok {
					if cardID, ok := val.(string); ok {
						c.SelectedCardID = cardID
						fmt.Printf("✅ [CardSelector Adapt] Found selected_card_id: %s\n", cardID)
					}
				}
				
				// 检查 cardSelectorParams
				if val, ok := dataMap["cardSelectorParams"]; ok {
					if params, ok := val.(map[string]interface{}); ok {
						fmt.Printf("✅ [CardSelector Adapt] Found cardSelectorParams: %+v\n", params)
						
						if card, ok := params["selectedCard"]; ok {
							if cardMap, ok := card.(map[string]interface{}); ok {
								c.SelectedCard = cardMap
								fmt.Printf("✅ [CardSelector Adapt] Set selectedCard from params: %+v\n", cardMap)
							}
						}
						if cardId, ok := params["selectedCardId"]; ok {
							if cardID, ok := cardId.(string); ok {
								c.SelectedCardID = cardID
								fmt.Printf("✅ [CardSelector Adapt] Set selectedCardId from params: %s\n", cardID)
							}
						}
					}
				}
			}
		}
	}
	
	fmt.Printf("🔍 [CardSelector Adapt] Final config - SelectedCard: %+v, SelectedCardID: %s\n", c.SelectedCard, c.SelectedCardID)

	ns := &schema.NodeSchema{
		Key:     vo.NodeKey(n.ID),
		Type:    entity.NodeTypeCardSelector,
		Name:    name,
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

	fmt.Printf("🔍 [CardSelector Build] Building node with config - SelectedCard: %+v, SelectedCardID: %s\n", c.SelectedCard, c.SelectedCardID)

	cardSelector := &CardSelector{
		apiEndpoint:    c.APIEndpoint,
		timeout:        timeout,
		selectedCard:   c.SelectedCard,
		selectedCardID: c.SelectedCardID,
	}

	fmt.Printf("🔍 [CardSelector Build] Created CardSelector instance: %+v\n", cardSelector)
	
	return cardSelector, nil
}

// CardSelector is the actual node implementation
type CardSelector struct {
	apiEndpoint    string
	timeout        int
	selectedCard   map[string]interface{} // 配置的选择卡片信息
	selectedCardID string                 // 配置的选择卡片ID
}

// Invoke implements InvokableNode interface
func (cs *CardSelector) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 🚨 调试信息：打印接收到的输入数据
	fmt.Printf("🔍 [CardSelector Debug] Received input keys: %v\n", getMapKeys(input))
	fmt.Printf("🔍 [CardSelector Debug] Full input: %+v\n", input)
	fmt.Printf("🔍 [CardSelector Debug] cs.selectedCard: %+v\n", cs.selectedCard)
	fmt.Printf("🔍 [CardSelector Debug] cs.selectedCardID: %s\n", cs.selectedCardID)
	
	// 检查运行时输入中的关键字段
	if selectedCardID, ok := input[InputKeySelectedCardID]; ok {
		fmt.Printf("✅ [CardSelector Debug] Found selected_card_id in input: %v\n", selectedCardID)
	} else {
		fmt.Printf("❌ [CardSelector Debug] selected_card_id NOT found in input\n")
	}
	
	if selectedCard, ok := input[InputKeySelectedCard]; ok {
		fmt.Printf("✅ [CardSelector Debug] Found selected_card in input: %v\n", selectedCard)
	} else {
		fmt.Printf("❌ [CardSelector Debug] selected_card NOT found in input\n")
	}

	// 最高优先级：使用配置中的选择卡片信息（前端已配置）
	if cs.selectedCard != nil && len(cs.selectedCard) > 0 {
		card := &FalconCard{}
		if cardId, exists := cs.selectedCard["cardId"]; exists {
			if id, ok := cardId.(string); ok {
				card.CardID = id
			}
		}
		if cardName, exists := cs.selectedCard["cardName"]; exists {
			if name, ok := cardName.(string); ok {
				card.CardName = name
			}
		}
		if code, exists := cs.selectedCard["code"]; exists {
			if c, ok := code.(string); ok {
				card.Code = c
			}
		}

		if card.CardID != "" && card.CardName != "" {
			templateResponse := cs.generateTemplateResponse(card, input)
			return map[string]any{
				OutputKeySelectedCard: map[string]any{
					"id":          card.CardID,
					"name":        card.CardName,
					"description": card.Code,
					"category":    card.CardClassID,
				},
				OutputKeyCardID:           card.CardID,
				OutputKeyCardName:         card.CardName,
				OutputKeyCardDesc:         card.Code,
				OutputKeyTemplateResponse: templateResponse,
			}, nil
		}
	}

	// 次优先级：使用配置中的选择卡片ID
	if cs.selectedCardID != "" {
		card, err := cs.fetchCardByID(ctx, cs.selectedCardID)
		if err == nil {
			templateResponse := cs.generateTemplateResponse(card, input)
			return map[string]any{
				OutputKeySelectedCard: map[string]any{
					"id":          card.CardID,
					"name":        card.CardName,
					"description": card.Code,
					"category":    card.CardClassID,
				},
				OutputKeyCardID:           card.CardID,
				OutputKeyCardName:         card.CardName,
				OutputKeyCardDesc:         card.Code,
				OutputKeyTemplateResponse: templateResponse,
			}, nil
		}
	}

	// 第三优先级：检查运行时传递的完整卡片信息
	if selectedCardData, ok := input[InputKeySelectedCard]; ok {
		if cardMap, ok := selectedCardData.(map[string]interface{}); ok {
			// 从前端传递的卡片信息构建FalconCard
			card := &FalconCard{}
			if cardId, exists := cardMap["cardId"]; exists {
				if id, ok := cardId.(string); ok {
					card.CardID = id
				}
			}
			if cardName, exists := cardMap["cardName"]; exists {
				if name, ok := cardName.(string); ok {
					card.CardName = name
				}
			}
			if code, exists := cardMap["code"]; exists {
				if c, ok := code.(string); ok {
					card.Code = c
				}
			}

			// 如果卡片信息完整，直接使用
			if card.CardID != "" && card.CardName != "" && card.Code != "" {
				// 生成模板输出结构
				templateResponse := cs.generateTemplateResponse(card, input)

				return map[string]any{
					OutputKeySelectedCard: map[string]any{
						"id":          card.CardID,
						"name":        card.CardName,
						"description": card.Code,
						"category":    card.CardClassID,
					},
					OutputKeyCardID:           card.CardID,
					OutputKeyCardName:         card.CardName,
					OutputKeyCardDesc:         card.Code,
					OutputKeyTemplateResponse: templateResponse,
				}, nil
			}
		}
	}

	// 如果没有完整的卡片信息，检查是否有选定的卡片ID，通过API获取
	if selectedCardID, ok := input[InputKeySelectedCardID].(string); ok && selectedCardID != "" {
		card, err := cs.fetchCardByID(ctx, selectedCardID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch selected card: %w", err)
		}

		// 生成模板输出结构
		templateResponse := cs.generateTemplateResponse(card, input)

		return map[string]any{
			OutputKeySelectedCard: map[string]any{
				"id":          card.CardID,
				"name":        card.CardName,
				"description": card.Code,
				"category":    card.CardClassID,
			},
			OutputKeyCardID:           card.CardID,
			OutputKeyCardName:         card.CardName,
			OutputKeyCardDesc:         card.Code,
			OutputKeyTemplateResponse: templateResponse,
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

	// 如果没有搜索条件且API端点未配置，返回空结果
	if searchKeyword == "" && len(filters) == 0 && cs.apiEndpoint == "" {
		fmt.Printf("📍 [CardSelector Debug] Returning empty search results (no conditions)\n")
		return map[string]any{
			OutputKeyCards: []map[string]any{},
			OutputKeyCount: 0,
		}, nil
	}

	fmt.Printf("📍 [CardSelector Debug] Calling searchCardsFromAPI with keyword='%s', filters=%v\n", searchKeyword, filters)
	
	// 调用猎鹰平台API获取卡片
	cards, err := cs.searchCardsFromAPI(ctx, searchKeyword, filters)
	if err != nil {
		fmt.Printf("❌ [CardSelector Debug] searchCardsFromAPI failed: %v\n", err)
		return nil, fmt.Errorf("failed to search cards from API: %w", err)
	}

	fmt.Printf("📍 [CardSelector Debug] searchCardsFromAPI returned %d cards\n", len(cards))

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
		
		// 生成模板输出结构
		templateResponse := cs.generateTemplateResponse(&card, input)
		result[OutputKeyTemplateResponse] = templateResponse
	}

	return result, nil
}

// searchCardsFromAPI 调用猎鹰平台API搜索卡片
func (cs *CardSelector) searchCardsFromAPI(ctx context.Context, searchKeyword string, filters map[string]any) ([]FalconCard, error) {
	if cs.apiEndpoint == "" {
		return nil, fmt.Errorf("API endpoint not configured")
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
	listAPIURL := cs.apiEndpoint
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
	if apiResp.Header.ErrorCode != "0" {
		return nil, fmt.Errorf("API returned error: %s (code: %s)", apiResp.Header.ErrorMsg, apiResp.Header.ErrorCode)
	}

	return apiResp.Body.CardList, nil
}

// fetchCardByID 根据ID获取特定卡片信息
func (cs *CardSelector) fetchCardByID(ctx context.Context, cardID string) (*FalconCard, error) {
	if cs.apiEndpoint == "" {
		return nil, fmt.Errorf("API endpoint not configured")
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
			SassWorkspaceID:     "7533521629687578624", // 写死spaceId
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
	detailAPIURL := cs.apiEndpoint
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
	if cs.apiEndpoint == "" {
		return nil, fmt.Errorf("API endpoint not configured")
	}

	// 构建卡片详情API请求体
	requestBody := FalconAPIRequest{
		Body: FalconRequestBody{
			CardID:          cardID,
			CreatedBy:       true,
			SassWorkspaceID: "7533521629687578624", // 写死spaceId
			PageNo:          "",
			PageSize:        "",
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
	detailAPIURL := cs.apiEndpoint
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


// SearchCards 公开方法：搜索卡片
func (cs *CardSelector) SearchCards(ctx context.Context, searchKeyword string, filters map[string]any) ([]FalconCard, error) {
	return cs.searchCardsFromAPI(ctx, searchKeyword, filters)
}

// GetCardDetail 公开方法：获取卡片详情
func (cs *CardSelector) GetCardDetail(ctx context.Context, cardID string) (*CardDetailBody, error) {
	return cs.fetchCardDetailWithParams(ctx, cardID)
}

// ToCallbackInput 实现 CallbackInputConverted 接口，用于格式化试运行输入
func (cs *CardSelector) ToCallbackInput(ctx context.Context, input map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	
	// 显示API配置信息
	if cs.apiEndpoint != "" {
		result["apiEndpoint"] = cs.apiEndpoint
	} else {
		result["apiEndpoint"] = "http://10.10.10.208:8500/aop-web" // 默认端点
	}
	
	// 显示超时配置
	result["timeout"] = cs.timeout
	
	// 获取并显示完整的卡片详情信息
	var cardDetail *FalconCard
	var cardDetailInfo *CardDetailBody
	
	// 优先使用配置中的选择卡片信息
	if cs.selectedCard != nil && len(cs.selectedCard) > 0 {
		cardDetail = &FalconCard{}
		if cardId, exists := cs.selectedCard["cardId"]; exists {
			if id, ok := cardId.(string); ok {
				cardDetail.CardID = id
			}
		}
		if cardName, exists := cs.selectedCard["cardName"]; exists {
			if name, ok := cardName.(string); ok {
				cardDetail.CardName = name
			}
		}
		if code, exists := cs.selectedCard["code"]; exists {
			if c, ok := code.(string); ok {
				cardDetail.Code = c
			}
		}
		
		// 如果有卡片ID，获取完整的卡片详情（包括参数列表）
		if cardDetail.CardID != "" {
			if detail, err := cs.fetchCardDetailWithParams(ctx, cardDetail.CardID); err == nil {
				cardDetailInfo = detail
			}
		}
	}
	
	// 次优先级：使用配置中的选择卡片ID
	if cardDetail == nil && cs.selectedCardID != "" {
		if card, err := cs.fetchCardByID(ctx, cs.selectedCardID); err == nil {
			cardDetail = card
			if detail, err := cs.fetchCardDetailWithParams(ctx, cs.selectedCardID); err == nil {
				cardDetailInfo = detail
			}
		}
	}
	
	// 第三优先级：检查运行时传递的完整卡片信息（来自测试表单）
	if cardDetail == nil {
		if selectedCardData, ok := input["selected_card"]; ok {
			if cardMap, ok := selectedCardData.(map[string]interface{}); ok {
				cardDetail = &FalconCard{}
				if cardId, exists := cardMap["cardId"]; exists {
					if id, ok := cardId.(string); ok {
						cardDetail.CardID = id
					}
				}
				if cardName, exists := cardMap["cardName"]; exists {
					if name, ok := cardName.(string); ok {
						cardDetail.CardName = name
					}
				}
				if code, exists := cardMap["code"]; exists {
					if c, ok := code.(string); ok {
						cardDetail.Code = c
					}
				}
				
				// 获取完整的卡片详情
				if cardDetail.CardID != "" {
					if detail, err := cs.fetchCardDetailWithParams(ctx, cardDetail.CardID); err == nil {
						cardDetailInfo = detail
					}
				}
			}
		}
	}
	
	// 第四优先级：检查运行时传递的完整卡片信息（来自节点配置）
	if cardDetail == nil {
		if selectedCardData, ok := input[InputKeySelectedCard]; ok {
			if cardMap, ok := selectedCardData.(map[string]interface{}); ok {
				cardDetail = &FalconCard{}
				if cardId, exists := cardMap["cardId"]; exists {
					if id, ok := cardId.(string); ok {
						cardDetail.CardID = id
					}
				}
				if cardName, exists := cardMap["cardName"]; exists {
					if name, ok := cardName.(string); ok {
						cardDetail.CardName = name
					}
				}
				if code, exists := cardMap["code"]; exists {
					if c, ok := code.(string); ok {
						cardDetail.Code = c
					}
				}
				
				// 获取完整的卡片详情
				if cardDetail.CardID != "" {
					if detail, err := cs.fetchCardDetailWithParams(ctx, cardDetail.CardID); err == nil {
						cardDetailInfo = detail
					}
				}
			}
		}
	}
	
	// 最后：检查运行时传递的卡片ID
	if cardDetail == nil {
		if selectedCardID, ok := input[InputKeySelectedCardID].(string); ok && selectedCardID != "" {
			if card, err := cs.fetchCardByID(ctx, selectedCardID); err == nil {
				cardDetail = card
				if detail, err := cs.fetchCardDetailWithParams(ctx, selectedCardID); err == nil {
					cardDetailInfo = detail
				}
			}
		}
	}
	
	// 如果找到了卡片详情，显示完整信息
	if cardDetail != nil {
		result["selectedCardDetail"] = map[string]any{
			"cardId":      cardDetail.CardID,
			"cardName":    cardDetail.CardName,
			"code":        cardDetail.Code,
			"cardPicURL":  cardDetail.CardPicURL,
		}
		
		// 如果有详细参数信息，也包含进去
		if cardDetailInfo != nil {
			result["cardParameters"] = cardDetailInfo.ParamList
			result["cardMainURL"] = cardDetailInfo.MainURL
			result["cardVersion"] = cardDetailInfo.Version
		}
	}
	
	// 显示搜索关键词
	if searchKeyword, ok := input[InputKeySearchKeyword].(string); ok && searchKeyword != "" {
		result["searchKeyword"] = searchKeyword
	}
	
	// 显示卡片筛选条件
	if cardFilters, ok := input[InputKeyCardFilters]; ok {
		if filterMap, ok := cardFilters.(map[string]any); ok && len(filterMap) > 0 {
			result["cardFilters"] = filterMap
		}
	}
	
	// 显示输入参数列表（工作流节点的输入参数定义）
	if inputParameters, ok := input[InputKeyInputParameters]; ok {
		result["inputParameters"] = inputParameters
	}
	
	// 显示其他运行时输入参数（用户在试运行表单中输入的变量值）
	variableInputs := make(map[string]any)
	for key, value := range input {
		// 跳过特殊的内置键，只显示用户定义的变量参数
		if key != "selected_card" && key != InputKeySearchKeyword && key != InputKeyCardFilters && 
		   key != InputKeySelectedCardID && key != InputKeySelectedCard && 
		   key != InputKeyInputParameters {
			variableInputs[key] = value
		}
	}
	if len(variableInputs) > 0 {
		result["variableInputs"] = variableInputs
	}
	
	return result, nil
}

// generateTemplateResponse 生成模板输出结构，按照用户要求的模板格式
func (cs *CardSelector) generateTemplateResponse(card *FalconCard, input map[string]any) map[string]any {
	// 生成dataResponse，根据输入参数和实际的变量值自动适配
	dataResponse := make(map[string]any)

	// 首先尝试从输入参数列表获取参数定义
	hasParameters := false
	if inputParameters, ok := input[InputKeyInputParameters]; ok {
		if paramList, ok := inputParameters.([]interface{}); ok {
			for _, param := range paramList {
				if paramMap, ok := param.(map[string]interface{}); ok {
					if paramName, exists := paramMap["name"]; exists {
						if name, ok := paramName.(string); ok && name != "" {
							hasParameters = true
							// 检查是否有实际的输入值
							if actualValue, hasValue := input[name]; hasValue {
								dataResponse[name] = actualValue
							} else {
								// 使用变量占位符格式
								dataResponse[name] = fmt.Sprintf("{%s}", name)
							}
						}
					}
				}
			}
		}
	}

	// 如果没有定义的输入参数，检查所有非特殊键的输入作为变量
	if !hasParameters {
		for key, value := range input {
			// 跳过特殊的内置键，将其他键作为变量处理
			if key != InputKeySearchKeyword && key != InputKeyCardFilters && 
			   key != InputKeySelectedCardID && key != InputKeySelectedCard && 
			   key != InputKeyInputParameters {
				dataResponse[key] = value
			}
		}
	}

	// 如果还是没有任何参数，提供默认的示例结构
	if len(dataResponse) == 0 {
		dataResponse["payeeList"] = "{payeeList}"
	}

	// 构建符合要求的模板输出结构
	templateResponse := map[string]any{
		"displayResponseType": "TEMPLATE",
		"rawContent":          map[string]any{},  // 原始内容，通常为空
		"templateId":          card.Code,         // 使用卡片的Code作为模板ID
		"templateName":        card.CardName,     // 使用卡片名称作为模板名称  
		"kvMap":               map[string]any{},  // 键值映射，通常为空
		"dataResponse":        dataResponse,      // 根据输入变量自动适配的数据响应
	}

	return templateResponse
}

// ToCallbackOutput 实现 CallbackOutputConverted 接口，用于格式化试运行输出
func (cs *CardSelector) ToCallbackOutput(ctx context.Context, out map[string]any) (*nodes.StructuredCallbackOutput, error) {
	// 检查是否有模板响应数据
	if templateResponse, ok := out[OutputKeyTemplateResponse]; ok {
		if templateMap, ok := templateResponse.(map[string]any); ok {
			// 试运行时直接返回模板响应格式，这样前端可以正确显示
			return &nodes.StructuredCallbackOutput{
				RawOutput: out,         // 保留原始输出
				Output:    templateMap, // 显示的输出为模板响应格式
			}, nil
		}
	}

	// 如果没有模板响应，返回原始输出
	return &nodes.StructuredCallbackOutput{
		RawOutput: out,
		Output:    out,
	}, nil
}
