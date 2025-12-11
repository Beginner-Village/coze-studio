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

package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	adminapi "github.com/coze-dev/coze-studio/backend/api/model/admin"
	adminentity "github.com/coze-dev/coze-studio/backend/domain/admin/entity"
	adminrepo "github.com/coze-dev/coze-studio/backend/domain/admin/repository"
	modelentity "github.com/coze-dev/coze-studio/backend/domain/model/entity"
	modelrepo "github.com/coze-dev/coze-studio/backend/domain/model/repository"
	"github.com/coze-dev/coze-studio/backend/infra/impl/storage"
	"github.com/coze-dev/coze-studio/backend/pkg/idgen"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

type AdminApplicationService struct {
	AdminRepo     adminrepo.AdminRepository
	ModelRepo     modelrepo.ModelRepository
	StorageClient storage.Storage
}

var AdminApplicationSVC = &AdminApplicationService{}

// CheckAdminInit 检查系统是否已初始化
func (s *AdminApplicationService) CheckAdminInit(ctx context.Context, userID uint64) (*adminapi.CheckAdminInitResponse, error) {
	initialized, err := s.AdminRepo.IsInitialized(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check initialization: %w", err)
	}

	resp := &adminapi.CheckAdminInitResponse{
		Initialized: initialized,
		Code:        0,
		Msg:         "success",
	}

	if initialized && userID > 0 {
		admin, err := s.AdminRepo.GetByUserID(ctx, userID)
		if err != nil {
			logs.CtxErrorf(ctx, "failed to get admin by user id: %v", err)
		}
		if admin != nil {
			isAdmin := true
			resp.IsAdmin = &isAdmin
			resp.Role = &admin.Role
		} else {
			isAdmin := false
			resp.IsAdmin = &isAdmin
		}
	}

	return resp, nil
}

// InitAdmin 初始化超级管理员
func (s *AdminApplicationService) InitAdmin(ctx context.Context, req *adminapi.InitAdminRequest) (*adminapi.InitAdminResponse, error) {
	// 检查是否已初始化
	initialized, err := s.AdminRepo.IsInitialized(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check initialization: %w", err)
	}
	if initialized {
		return &adminapi.InitAdminResponse{
			Code: 40002,
			Msg:  "系统已初始化，不能重复初始化",
		}, nil
	}

	// 解析用户ID
	userID, err := strconv.ParseUint(req.GetUserID(), 10, 64)
	if err != nil {
		return &adminapi.InitAdminResponse{
			Code: 40001,
			Msg:  "无效的用户ID",
		}, nil
	}

	// 创建超级管理员
	admin := &adminentity.AdminUser{
		UserID:    userID,
		Role:      adminentity.RoleSuperAdmin,
		CreatedBy: userID,
	}

	if err := s.AdminRepo.Create(ctx, admin); err != nil {
		return nil, fmt.Errorf("failed to create super admin: %w", err)
	}

	return &adminapi.InitAdminResponse{
		Code: 0,
		Msg:  "初始化成功",
	}, nil
}

// ListAdminUsers 获取管理员列表
func (s *AdminApplicationService) ListAdminUsers(ctx context.Context) (*adminapi.ListAdminUsersResponse, error) {
	admins, err := s.AdminRepo.ListAdmins(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}

	items := make([]*adminapi.AdminUserInfo, 0, len(admins))
	for _, admin := range admins {
		items = append(items, &adminapi.AdminUserInfo{
			UserID:    fmt.Sprintf("%d", admin.UserID),
			Username:  fmt.Sprintf("用户%d", admin.UserID), // TODO: 从用户服务获取用户名
			AvatarURL: "",
			Role:      admin.Role,
			CreatedAt: int64(admin.CreatedAt),
		})
	}

	return &adminapi.ListAdminUsersResponse{
		Items: items,
		Code:  0,
		Msg:   "success",
	}, nil
}

// AddAdmin 添加管理员
func (s *AdminApplicationService) AddAdmin(ctx context.Context, req *adminapi.AddAdminRequest, operatorID uint64) (*adminapi.AddAdminResponse, error) {
	userID, err := strconv.ParseUint(req.GetUserID(), 10, 64)
	if err != nil {
		return &adminapi.AddAdminResponse{
			Code: 40001,
			Msg:  "无效的用户ID",
		}, nil
	}

	// 检查是否已是管理员
	exists, err := s.AdminRepo.IsAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin: %w", err)
	}
	if exists {
		return &adminapi.AddAdminResponse{
			Code: 40006,
			Msg:  "该用户已是管理员",
		}, nil
	}

	role := adminentity.RoleAdmin
	if req.Role != nil && *req.Role == adminentity.RoleSuperAdmin {
		role = adminentity.RoleSuperAdmin
	}

	admin := &adminentity.AdminUser{
		UserID:    userID,
		Role:      role,
		CreatedBy: operatorID,
	}

	if err := s.AdminRepo.Create(ctx, admin); err != nil {
		return nil, fmt.Errorf("failed to create admin: %w", err)
	}

	return &adminapi.AddAdminResponse{
		Code: 0,
		Msg:  "添加成功",
	}, nil
}

// RemoveAdmin 移除管理员
func (s *AdminApplicationService) RemoveAdmin(ctx context.Context, req *adminapi.RemoveAdminRequest, operatorID uint64) (*adminapi.RemoveAdminResponse, error) {
	userID, err := strconv.ParseUint(req.GetUserID(), 10, 64)
	if err != nil {
		return &adminapi.RemoveAdminResponse{
			Code: 40001,
			Msg:  "无效的用户ID",
		}, nil
	}

	// 不能移除自己
	if userID == operatorID {
		return &adminapi.RemoveAdminResponse{
			Code: 40004,
			Msg:  "不能移除自己的管理员权限",
		}, nil
	}

	// 检查目标用户是否是超级管理员
	admin, err := s.AdminRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}
	if admin == nil {
		return &adminapi.RemoveAdminResponse{
			Code: 40006,
			Msg:  "该用户不是管理员",
		}, nil
	}

	// 如果是超级管理员，检查是否是最后一个
	if admin.IsSuperAdmin() {
		count, err := s.AdminRepo.CountSuperAdmins(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to count super admins: %w", err)
		}
		if count <= 1 {
			return &adminapi.RemoveAdminResponse{
				Code: 40005,
				Msg:  "不能移除最后一个超级管理员",
			}, nil
		}
	}

	if err := s.AdminRepo.Delete(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to delete admin: %w", err)
	}

	return &adminapi.RemoveAdminResponse{
		Code: 0,
		Msg:  "移除成功",
	}, nil
}

// ListPublicModels 获取公共模型列表
func (s *AdminApplicationService) ListPublicModels(ctx context.Context, req *adminapi.ListPublicModelsRequest) (*adminapi.ListPublicModelsResponse, error) {
	models, err := s.ModelRepo.GetPublicModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list public models: %w", err)
	}

	items := make([]*adminapi.PublicModelItem, 0, len(models))
	for _, model := range models {
		// 获取图片URL
		iconURL := model.IconURL
		if iconURL == "" && model.IconURI != "" && s.StorageClient != nil {
			url, err := s.StorageClient.GetObjectUrl(ctx, model.IconURI)
			if err == nil {
				iconURL = url
			}
		}

		item := &adminapi.PublicModelItem{
			ID:       model.ID,
			Name:     model.Name,
			Protocol: model.Protocol,
			Status:   int32(model.Status),
		}

		if model.Description != "" {
			item.Description = &model.Description
		}
		if iconURL != "" {
			item.IconURL = &iconURL
		}
		if model.IconURI != "" {
			item.IconURI = &model.IconURI
		}
		if model.ContextLength > 0 {
			contextLen := model.ContextLength
			item.ContextLength = &contextLen
		}

		// TODO: 从 Scenario 或其他字段推断 model_type
		item.ModelType = "llm"

		items = append(items, item)
	}

	return &adminapi.ListPublicModelsResponse{
		Items: items,
		Code:  0,
		Msg:   "success",
	}, nil
}

// CreatePublicModel 创建公共模型
func (s *AdminApplicationService) CreatePublicModel(ctx context.Context, req *adminapi.CreatePublicModelRequest) (*adminapi.CreatePublicModelResponse, error) {
	// 构建能力配置
	capability := map[string]interface{}{}
	if req.InputTokens != nil {
		capability["input_tokens"] = *req.InputTokens
	}
	if req.OutputTokens != nil {
		capability["output_tokens"] = *req.OutputTokens
	}
	if req.FunctionCall != nil {
		capability["function_call"] = *req.FunctionCall
	}
	if req.JSONMode != nil {
		capability["json_mode"] = *req.JSONMode
	}
	if req.Reasoning != nil {
		capability["reasoning"] = *req.Reasoning
	}
	capabilityJSON, _ := json.Marshal(capability)

	// 构建连接配置
	connConfig := map[string]interface{}{
		"base_url": req.GetBaseURL(),
		"api_key":  req.GetAPIKey(),
		"model":    req.GetModel(),
	}
	if req.EnableThinking != nil {
		connConfig["enable_thinking"] = *req.EnableThinking
	}
	connConfigJSON, _ := json.Marshal(connConfig)

	// 创建 ModelMeta
	metaID := idgen.NextID()
	capStr := string(capabilityJSON)
	connStr := string(connConfigJSON)
	meta := &modelentity.ModelMeta{
		ID:         metaID,
		ModelName:  req.GetModel(),
		Protocol:   req.GetProtocol(),
		Capability: &capStr,
		ConnConfig: &connStr,
		Status:     1,
		IsPublic:   1,
	}
	// 设置图标：优先使用前端传递的图标，否则根据协议设置默认图标
	if req.IconURI != nil && *req.IconURI != "" {
		meta.IconURI = *req.IconURI
	} else {
		meta.IconURI = getDefaultIconByProtocol(req.GetProtocol())
	}

	if err := s.ModelRepo.CreateModelMeta(ctx, meta); err != nil {
		return nil, fmt.Errorf("failed to create model meta: %w", err)
	}

	// 构建描述
	descMap := map[string]string{"zh": "", "en": ""}
	if req.Description != nil {
		descMap["zh"] = *req.Description
	}
	descJSON, _ := json.Marshal(descMap)

	// 创建 ModelEntity
	modelID := idgen.NextID()
	descStr := string(descJSON)
	model := &modelentity.ModelEntity{
		ID:            modelID,
		MetaID:        metaID,
		Name:          req.GetName(),
		Description:   &descStr,
		DefaultParams: "[]",
		Scenario:      1, // LLM
		Status:        1,
		IsPublic:      1,
	}

	if err := s.ModelRepo.CreatePublicModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to create public model: %w", err)
	}

	return &adminapi.CreatePublicModelResponse{
		Data: &adminapi.PublicModelItem{
			ID:        fmt.Sprintf("%d", modelID),
			Name:      req.GetName(),
			ModelType: req.GetModelType(),
			Protocol:  req.GetProtocol(),
			Status:    1,
			CreatedAt: int64(time.Now().UnixMilli()),
		},
		Code: 0,
		Msg:  "创建成功",
	}, nil
}

// GetPublicModel 获取公共模型详情
func (s *AdminApplicationService) GetPublicModel(ctx context.Context, req *adminapi.GetPublicModelRequest) (*adminapi.GetPublicModelResponse, error) {
	modelID, err := strconv.ParseUint(req.GetModelID(), 10, 64)
	if err != nil {
		return &adminapi.GetPublicModelResponse{
			Code: 40001,
			Msg:  "无效的模型ID",
		}, nil
	}

	model, err := s.ModelRepo.GetPublicModelByID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public model: %w", err)
	}
	if model == nil {
		return &adminapi.GetPublicModelResponse{
			Code: 40404,
			Msg:  "模型不存在",
		}, nil
	}

	// 获取模型元数据
	meta, err := s.ModelRepo.GetModelMetaByID(ctx, model.MetaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model meta: %w", err)
	}

	// 解析连接配置
	var connConfig map[string]interface{}
	if meta != nil && meta.ConnConfig != nil {
		json.Unmarshal([]byte(*meta.ConnConfig), &connConfig)
	}

	// 解析能力配置
	var capability map[string]interface{}
	if meta != nil && meta.Capability != nil {
		json.Unmarshal([]byte(*meta.Capability), &capability)
	}

	// 构建响应 - 使用 PublicModelDetailOutput 类型
	detail := &adminapi.PublicModelDetailOutput{
		ID:        fmt.Sprintf("%d", model.ID),
		Name:      model.Name,
		Protocol:  meta.Protocol,
		Status:    int32(model.Status),
		ModelType: "llm",
		CreatedAt: int64(model.CreatedAt),
	}

	// 设置描述
	if model.Description != nil {
		var descMap map[string]string
		if err := json.Unmarshal([]byte(*model.Description), &descMap); err == nil {
			if zh, ok := descMap["zh"]; ok {
				detail.Description = &zh
			}
		} else {
			detail.Description = model.Description
		}
	}

	// 设置图标
	if meta != nil {
		if meta.IconURL != "" {
			detail.IconURL = &meta.IconURL
		}
		if meta.IconURI != "" {
			detail.IconURI = &meta.IconURI
		}
	}

	// 从连接配置中提取字段
	if baseURL, ok := connConfig["base_url"].(string); ok {
		detail.BaseURL = baseURL
	}
	if modelName, ok := connConfig["model"].(string); ok {
		detail.Model = modelName
	}
	// API Key 不返回（安全考虑）
	detail.APIKey = "***"

	// 从能力配置中提取字段
	if inputTokens, ok := capability["input_tokens"].(float64); ok {
		v := int32(inputTokens)
		detail.InputTokens = &v
	}
	if outputTokens, ok := capability["output_tokens"].(float64); ok {
		v := int32(outputTokens)
		detail.OutputTokens = &v
	}
	if fc, ok := capability["function_call"].(bool); ok {
		detail.FunctionCall = &fc
	}
	if jm, ok := capability["json_mode"].(bool); ok {
		detail.JSONMode = &jm
	}
	if r, ok := capability["reasoning"].(bool); ok {
		detail.Reasoning = &r
	}
	if et, ok := connConfig["enable_thinking"].(bool); ok {
		detail.EnableThinking = &et
	}

	return &adminapi.GetPublicModelResponse{
		Data: detail,
		Code: 0,
		Msg:  "success",
	}, nil
}

// UpdatePublicModel 更新公共模型
func (s *AdminApplicationService) UpdatePublicModel(ctx context.Context, req *adminapi.UpdatePublicModelRequest) (*adminapi.UpdatePublicModelResponse, error) {
	modelID, err := strconv.ParseUint(req.GetModelID(), 10, 64)
	if err != nil {
		return &adminapi.UpdatePublicModelResponse{
			Code: 40001,
			Msg:  "无效的模型ID",
		}, nil
	}

	// 获取现有模型
	model, err := s.ModelRepo.GetPublicModelByID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public model: %w", err)
	}
	if model == nil {
		return &adminapi.UpdatePublicModelResponse{
			Code: 40404,
			Msg:  "模型不存在",
		}, nil
	}

	// 获取现有元数据
	meta, err := s.ModelRepo.GetModelMetaByID(ctx, model.MetaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model meta: %w", err)
	}

	// 更新模型实体
	if req.Name != nil {
		model.Name = *req.Name
	}
	if req.Description != nil {
		descMap := map[string]string{"zh": *req.Description, "en": ""}
		descJSON, _ := json.Marshal(descMap)
		descStr := string(descJSON)
		model.Description = &descStr
	}

	// 更新能力配置
	capability := map[string]interface{}{}
	if meta.Capability != nil {
		json.Unmarshal([]byte(*meta.Capability), &capability)
	}
	if req.InputTokens != nil {
		capability["input_tokens"] = *req.InputTokens
	}
	if req.OutputTokens != nil {
		capability["output_tokens"] = *req.OutputTokens
	}
	if req.FunctionCall != nil {
		capability["function_call"] = *req.FunctionCall
	}
	if req.JSONMode != nil {
		capability["json_mode"] = *req.JSONMode
	}
	if req.Reasoning != nil {
		capability["reasoning"] = *req.Reasoning
	}
	capabilityJSON, _ := json.Marshal(capability)
	capStr := string(capabilityJSON)
	meta.Capability = &capStr

	// 更新连接配置
	connConfig := map[string]interface{}{}
	if meta.ConnConfig != nil {
		json.Unmarshal([]byte(*meta.ConnConfig), &connConfig)
	}
	if req.BaseURL != nil {
		connConfig["base_url"] = *req.BaseURL
	}
	if req.APIKey != nil && *req.APIKey != "" {
		connConfig["api_key"] = *req.APIKey
	}
	if req.Model != nil {
		connConfig["model"] = *req.Model
		meta.ModelName = *req.Model
	}
	if req.EnableThinking != nil {
		connConfig["enable_thinking"] = *req.EnableThinking
	}
	connConfigJSON, _ := json.Marshal(connConfig)
	connStr := string(connConfigJSON)
	meta.ConnConfig = &connStr

	// 保存更新
	if err := s.ModelRepo.UpdateModel(ctx, model); err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}
	if err := s.ModelRepo.UpdateModelMeta(ctx, meta); err != nil {
		return nil, fmt.Errorf("failed to update model meta: %w", err)
	}

	return &adminapi.UpdatePublicModelResponse{
		Code: 0,
		Msg:  "更新成功",
	}, nil
}

// DeletePublicModel 删除公共模型
func (s *AdminApplicationService) DeletePublicModel(ctx context.Context, req *adminapi.DeletePublicModelRequest) (*adminapi.DeletePublicModelResponse, error) {
	modelID, err := strconv.ParseUint(req.GetModelID(), 10, 64)
	if err != nil {
		return &adminapi.DeletePublicModelResponse{
			Code: 40001,
			Msg:  "无效的模型ID",
		}, nil
	}

	if err := s.ModelRepo.DeletePublicModel(ctx, modelID); err != nil {
		return nil, fmt.Errorf("failed to delete public model: %w", err)
	}

	return &adminapi.DeletePublicModelResponse{
		Code: 0,
		Msg:  "删除成功",
	}, nil
}

// EnablePublicModel 启用公共模型
func (s *AdminApplicationService) EnablePublicModel(ctx context.Context, req *adminapi.EnablePublicModelRequest) (*adminapi.EnablePublicModelResponse, error) {
	modelID, err := strconv.ParseUint(req.GetModelID(), 10, 64)
	if err != nil {
		return &adminapi.EnablePublicModelResponse{
			Code: 40001,
			Msg:  "无效的模型ID",
		}, nil
	}

	if err := s.ModelRepo.SetModelPublic(ctx, modelID, 1); err != nil {
		return nil, fmt.Errorf("failed to enable public model: %w", err)
	}

	return &adminapi.EnablePublicModelResponse{
		Code: 0,
		Msg:  "启用成功",
	}, nil
}

// DisablePublicModel 禁用公共模型
func (s *AdminApplicationService) DisablePublicModel(ctx context.Context, req *adminapi.DisablePublicModelRequest) (*adminapi.DisablePublicModelResponse, error) {
	modelID, err := strconv.ParseUint(req.GetModelID(), 10, 64)
	if err != nil {
		return &adminapi.DisablePublicModelResponse{
			Code: 40001,
			Msg:  "无效的模型ID",
		}, nil
	}

	// 这里是禁用模型的 status，不是 is_public
	// 需要更新 model_entity 的 status 字段
	if err := s.ModelRepo.UpdateModel(ctx, &modelentity.ModelEntity{
		ID:     modelID,
		Status: 2, // 禁用
	}); err != nil {
		return nil, fmt.Errorf("failed to disable public model: %w", err)
	}

	return &adminapi.DisablePublicModelResponse{
		Code: 0,
		Msg:  "禁用成功",
	}, nil
}

// IsAdmin 检查用户是否是管理员
func (s *AdminApplicationService) IsAdmin(ctx context.Context, userID uint64) (bool, error) {
	return s.AdminRepo.IsAdmin(ctx, userID)
}

// IsSuperAdmin 检查用户是否是超级管理员
func (s *AdminApplicationService) IsSuperAdmin(ctx context.Context, userID uint64) (bool, error) {
	admin, err := s.AdminRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if admin == nil {
		return false, nil
	}
	return admin.IsSuperAdmin(), nil
}

// getDefaultIconByProtocol 根据协议返回默认图标
func getDefaultIconByProtocol(protocol string) string {
	switch protocol {
	case "qwen":
		return "default_icon/qwen_v2.png"
	case "openai":
		return "default_icon/openai_v2.png"
	case "ark":
		return "default_icon/doubao_v2.png"
	case "deepseek":
		return "default_icon/deepseek_v2.png"
	case "ollama":
		return "default_icon/ollama.png"
	case "gemini":
		return "default_icon/gemini.png"
	case "claude":
		return "default_icon/claude.png"
	default:
		return "default_icon/model.png"
	}
}
