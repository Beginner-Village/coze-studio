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

package coze

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	productapp "github.com/ynet-dev/ynet-studio/backend/application/aiproduct"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	productentity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

type superAgentProductListRequest struct {
	SpaceID  int64  `form:"space_id" json:"space_id,string,omitempty"`
	Type     string `form:"type" json:"type,omitempty"`
	Keyword  string `form:"keyword" json:"keyword,omitempty"`
	Page     int32  `form:"page" json:"page,omitempty"`
	PageSize int32  `form:"page_size" json:"page_size,omitempty"`
}

type superAgentProductGetRequest struct {
	ProductID int64 `form:"product_id" json:"product_id,string,omitempty"`
	SpaceID   int64 `form:"space_id" json:"space_id,string,omitempty"`
}

type superAgentProductInstallRequest struct {
	ProductID int64  `form:"product_id" json:"product_id,string,omitempty"`
	SpaceID   int64  `form:"space_id" json:"space_id,string,omitempty"`
	Version   string `form:"version" json:"version,omitempty"`
}

type superAgentRuntimeConfigRequest struct {
	ConversationID  int64          `form:"conversation_id" json:"conversation_id,string,omitempty"`
	AgentID         int64          `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID           int64          `form:"bot_id" json:"bot_id,string,omitempty"`
	SpaceID         int64          `form:"space_id" json:"space_id,string,omitempty"`
	ModelProductID  int64          `form:"model_product_id" json:"model_product_id,string,omitempty"`
	MCPProductIDs   []int64        `form:"mcp_product_ids" json:"mcp_product_ids,omitempty"`
	SkillProductIDs []int64        `form:"skill_product_ids" json:"skill_product_ids,omitempty"`
	ToolPolicy      map[string]any `form:"tool_policy" json:"tool_policy,omitempty"`
	ContextPolicy   map[string]any `form:"context_policy" json:"context_policy,omitempty"`
	User            string         `form:"user_id" json:"user_id,omitempty"`
}

type superAgentProductResponse struct {
	Code int                   `json:"code"`
	Msg  string                `json:"msg"`
	Data superAgentProductData `json:"data"`
}

type superAgentProductListResponse struct {
	Code int                       `json:"code"`
	Msg  string                    `json:"msg"`
	Data superAgentProductListData `json:"data"`
}

type superAgentProductData struct {
	Product       *superAgentProductItem       `json:"product,omitempty"`
	Installation  *superAgentInstallationItem  `json:"installation,omitempty"`
	RuntimeConfig *superAgentRuntimeConfigItem `json:"runtime_config,omitempty"`
}

type superAgentProductListData struct {
	Products []*superAgentProductItem `json:"products"`
	Total    int32                    `json:"total"`
	Page     int32                    `json:"page"`
	PageSize int32                    `json:"page_size"`
}

type superAgentProductItem struct {
	ProductID        string         `json:"product_id"`
	SpaceID          string         `json:"space_id"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Type             string         `json:"type"`
	Status           string         `json:"status"`
	Visibility       string         `json:"visibility"`
	IconURI          string         `json:"icon_uri,omitempty"`
	CoverURI         string         `json:"cover_uri,omitempty"`
	LatestVersion    string         `json:"latest_version,omitempty"`
	PublishedVersion string         `json:"published_version,omitempty"`
	Official         bool           `json:"official"`
	Featured         bool           `json:"featured"`
	InstallCount     int64          `json:"install_count"`
	DownloadCount    int64          `json:"download_count"`
	Feature          map[string]any `json:"feature,omitempty"`
	SourceRefType    string         `json:"source_ref_type,omitempty"`
	SourceRefID      string         `json:"source_ref_id,omitempty"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`
}

type superAgentInstallationItem struct {
	InstallationID string         `json:"installation_id"`
	ProductID      string         `json:"product_id"`
	ProductVersion string         `json:"product_version"`
	TargetSpaceID  string         `json:"target_space_id"`
	TargetUserID   string         `json:"target_user_id,omitempty"`
	InstalledBy    string         `json:"installed_by"`
	Status         string         `json:"status"`
	InstallMode    string         `json:"install_mode"`
	RuntimeConfig  map[string]any `json:"runtime_config,omitempty"`
	CreatedAt      int64          `json:"created_at"`
	UpdatedAt      int64          `json:"updated_at"`
}

type superAgentRuntimeConfigItem struct {
	ConversationID   string         `json:"conversation_id"`
	AgentID          string         `json:"agent_id"`
	SpaceID          string         `json:"space_id"`
	ModelProductID   string         `json:"model_product_id,omitempty"`
	MCPProductIDs    []string       `json:"mcp_product_ids,omitempty"`
	SkillProductIDs  []string       `json:"skill_product_ids,omitempty"`
	ToolPolicy       map[string]any `json:"tool_policy,omitempty"`
	ContextPolicy    map[string]any `json:"context_policy,omitempty"`
	ResolvedSnapshot map[string]any `json:"resolved_snapshot,omitempty"`
	CreatedBy        string         `json:"created_by"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`
}

func SuperAgentListProducts(ctx context.Context, c *app.RequestContext) {
	SuperAgentListMarketplaceProducts(ctx, c)
}

func SuperAgentGetProduct(ctx context.Context, c *app.RequestContext) {
	SuperAgentGetMarketplaceProduct(ctx, c)
}

func SuperAgentGetSessionRuntimeConfig(ctx context.Context, c *app.RequestContext) {
	var req superAgentRuntimeConfigRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if _, err := getSuperAgentRuntimeConversation(ctx, req.ConversationID, userID); err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	config, err := svc.GetSessionRuntimeConfig(ctx, req.ConversationID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{RuntimeConfig: runtimeConfigToResponse(config)},
	})
}

func SuperAgentUpdateSessionRuntimeConfig(ctx context.Context, c *app.RequestContext) {
	var req superAgentRuntimeConfigRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	if req.SpaceID <= 0 {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	currentConversation, err := getSuperAgentRuntimeConversation(ctx, req.ConversationID, userID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	agentID := req.AgentID
	if agentID == 0 {
		agentID = req.BotID
	}
	if agentID == 0 {
		agentID = currentConversation.AgentID
	}
	if agentID <= 0 {
		invalidParamRequestResponse(c, "agent_id or bot_id is required")
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	config, err := svc.UpsertSessionRuntimeConfig(ctx, &productentity.SessionRuntimeConfig{
		ConversationID:  req.ConversationID,
		AgentID:         agentID,
		SpaceID:         req.SpaceID,
		ModelProductID:  req.ModelProductID,
		MCPProductIDs:   req.MCPProductIDs,
		SkillProductIDs: req.SkillProductIDs,
		ToolPolicy:      req.ToolPolicy,
		ContextPolicy:   req.ContextPolicy,
	})
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{RuntimeConfig: runtimeConfigToResponse(config)},
	})
}

func SuperAgentDeleteSessionRuntimeConfig(ctx context.Context, c *app.RequestContext) {
	var req superAgentRuntimeConfigRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	userID, err := resolveSuperAgentSessionUserID(ctx, req.User)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if _, err := getSuperAgentRuntimeConversation(ctx, req.ConversationID, userID); err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	if err := svc.DeleteSessionRuntimeConfig(ctx, req.ConversationID); err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{},
	})
}

func SuperAgentInstallProduct(ctx context.Context, c *app.RequestContext) {
	var req superAgentProductInstallRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ProductID <= 0 {
		invalidParamRequestResponse(c, "product_id is required")
		return
	}
	if req.SpaceID <= 0 {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	installation, err := svc.InstallProduct(ctx, req.ProductID, req.SpaceID, req.Version)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{Installation: installationToResponse(installation)},
	})
}

func SuperAgentUninstallProduct(ctx context.Context, c *app.RequestContext) {
	var req superAgentProductInstallRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ProductID <= 0 {
		invalidParamRequestResponse(c, "product_id is required")
		return
	}
	if req.SpaceID <= 0 {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	if err := svc.UninstallProduct(ctx, req.ProductID, req.SpaceID); err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{},
	})
}

func SuperAgentUpgradeProduct(ctx context.Context, c *app.RequestContext) {
	var req superAgentProductInstallRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ProductID <= 0 {
		invalidParamRequestResponse(c, "product_id is required")
		return
	}
	if req.SpaceID <= 0 {
		invalidParamRequestResponse(c, "space_id is required")
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	installation, err := svc.UpgradeProduct(ctx, req.ProductID, req.SpaceID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{Installation: installationToResponse(installation)},
	})
}

func SuperAgentListMarketplaceProducts(ctx context.Context, c *app.RequestContext) {
	var req superAgentProductListRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	result, err := svc.ListMarketplaceProducts(ctx, &productentity.ListProductsRequest{
		SpaceID:  req.SpaceID,
		Type:     productentity.AIProductType(req.Type),
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	products := make([]*superAgentProductItem, 0, len(result.Products))
	for _, product := range result.Products {
		products = append(products, productToResponse(product))
	}
	c.JSON(http.StatusOK, &superAgentProductListResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductListData{
			Products: products,
			Total:    result.Total,
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	})
}

func SuperAgentGetMarketplaceProduct(ctx context.Context, c *app.RequestContext) {
	var req superAgentProductGetRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ProductID <= 0 {
		invalidParamRequestResponse(c, "product_id is required")
		return
	}
	svc := productApplication()
	if svc == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("ai product service is not initialized"))
		return
	}
	product, err := svc.GetVisibleProduct(ctx, req.ProductID, req.SpaceID)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	if product == nil {
		internalServerErrorResponse(ctx, c, fmt.Errorf("product not found"))
		return
	}
	c.JSON(http.StatusOK, &superAgentProductResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentProductData{Product: productToResponse(product)},
	})
}

func productApplication() *productapp.ProductApplicationService {
	return productapp.ProductApplicationSVC
}

func productToResponse(product *productentity.Product) *superAgentProductItem {
	if product == nil {
		return nil
	}
	return &superAgentProductItem{
		ProductID:        fmt.Sprintf("%d", product.ProductID),
		SpaceID:          fmt.Sprintf("%d", product.SpaceID),
		Name:             product.Name,
		Description:      product.Description,
		Type:             string(product.Type),
		Status:           string(product.Status),
		Visibility:       string(product.Visibility),
		IconURI:          product.IconURI,
		CoverURI:         product.CoverURI,
		LatestVersion:    product.LatestVersion,
		PublishedVersion: product.PublishedVersion,
		Official:         product.Official,
		Featured:         product.Featured,
		InstallCount:     product.InstallCount,
		DownloadCount:    product.DownloadCount,
		Feature:          product.Feature,
		SourceRefType:    product.SourceRefType,
		SourceRefID:      fmt.Sprintf("%d", product.SourceRefID),
		CreatedAt:        product.CreatedAt,
		UpdatedAt:        product.UpdatedAt,
	}
}

func installationToResponse(installation *productentity.ProductInstallation) *superAgentInstallationItem {
	if installation == nil {
		return nil
	}
	return &superAgentInstallationItem{
		InstallationID: fmt.Sprintf("%d", installation.InstallationID),
		ProductID:      fmt.Sprintf("%d", installation.ProductID),
		ProductVersion: installation.ProductVersion,
		TargetSpaceID:  fmt.Sprintf("%d", installation.TargetSpaceID),
		TargetUserID:   fmt.Sprintf("%d", installation.TargetUserID),
		InstalledBy:    fmt.Sprintf("%d", installation.InstalledBy),
		Status:         string(installation.Status),
		InstallMode:    installation.InstallMode,
		RuntimeConfig:  installation.RuntimeConfig,
		CreatedAt:      installation.CreatedAt,
		UpdatedAt:      installation.UpdatedAt,
	}
}

func runtimeConfigToResponse(config *productentity.SessionRuntimeConfig) *superAgentRuntimeConfigItem {
	if config == nil {
		return nil
	}
	return &superAgentRuntimeConfigItem{
		ConversationID:   fmt.Sprintf("%d", config.ConversationID),
		AgentID:          fmt.Sprintf("%d", config.AgentID),
		SpaceID:          fmt.Sprintf("%d", config.SpaceID),
		ModelProductID:   formatOptionalInt64(config.ModelProductID),
		MCPProductIDs:    formatInt64Slice(config.MCPProductIDs),
		SkillProductIDs:  formatInt64Slice(config.SkillProductIDs),
		ToolPolicy:       config.ToolPolicy,
		ContextPolicy:    config.ContextPolicy,
		ResolvedSnapshot: config.ResolvedSnapshot,
		CreatedBy:        fmt.Sprintf("%d", config.CreatedBy),
		CreatedAt:        config.CreatedAt,
		UpdatedAt:        config.UpdatedAt,
	}
}

func getSuperAgentRuntimeConversation(ctx context.Context, conversationID, userID int64) (*convEntity.Conversation, error) {
	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if currentConversation == nil {
		return nil, errorx.New(errno.ErrConversationNotFound)
	}
	if currentConversation.CreatorID != userID {
		return nil, errorx.New(errno.ErrConversationPermissionCode, errorx.KV("msg", "permission denied"))
	}
	return currentConversation, nil
}

func formatOptionalInt64(v int64) string {
	if v <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", v)
}

func formatInt64Slice(values []int64) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		out = append(out, fmt.Sprintf("%d", value))
	}
	return out
}
