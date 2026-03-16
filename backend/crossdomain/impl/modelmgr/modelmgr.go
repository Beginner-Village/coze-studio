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

package modelmgr

import (
	"context"
	"fmt"

	eino "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"

	model "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/modelmgr"
	crossmodelmgr "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/modelmgr"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/service"
	"github.com/ynet-dev/ynet-studio/backend/domain/ynet_agent"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/modelmgr"
	chatmodel2 "github.com/ynet-dev/ynet-studio/backend/infra/impl/chatmodel"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/mysql"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

type modelManager struct {
	modelMgr modelmgr.Manager
	factory  chatmodel.Factory
}

func InitDomainService(m modelmgr.Manager, f chatmodel.Factory) crossmodelmgr.Manager {
	if f == nil {
		f = chatmodel2.NewDefaultFactory()
	}
	return &modelManager{
		modelMgr: m,
		factory:  f,
	}
}

func (m *modelManager) GetModel(ctx context.Context, params *model.LLMParams) (eino.BaseChatModel, *modelmgr.Model, error) {
	// 🆕 检查是否使用外部智能体或内部智能体
	if params.IsHiAgent {
		// 如果是 SingleAgent，使用特殊处理逻辑
		if params.ExternalAgentPlatform == "singleagent" {
			return m.getSingleAgentModel(ctx, params)
		}
		// 其他外部智能体（HiAgent/Dify）使用HTTP调用
		return m.getHiAgentModel(ctx, params)
	}

	// 原有的标准模型逻辑
	modelID := params.ModelType
	logs.CtxInfof(ctx, "[GetModel] Fetching model with ID: %d", modelID)
	models, err := m.modelMgr.MGetModelByID(ctx, &modelmgr.MGetModelRequest{
		IDs: []int64{modelID},
	})
	if err != nil {
		logs.CtxErrorf(ctx, "[GetModel] MGetModelByID failed: %v", err)
		return nil, nil, err
	}
	logs.CtxInfof(ctx, "[GetModel] MGetModelByID returned %d models", len(models))

	var config *chatmodel.Config
	var protocol chatmodel.Protocol
	var mdl *modelmgr.Model
	for i := range models {
		md := models[i]
		logs.CtxInfof(ctx, "[GetModel] Checking model ID=%d, Name=%s, Status=%v, ConnConfig is nil: %v",
			md.ID, md.Name, md.Meta.Status, md.Meta.ConnConfig == nil)
		if md.ID == modelID {
			// 检查模型状态，确保模型是启用状态
			if md.Meta.Status != modelmgr.StatusInUse {
				logs.CtxWarnf(ctx, "[GetModel] Model ID=%d is not in use, status=%v", modelID, md.Meta.Status)
				return nil, nil, fmt.Errorf("model is not available, modelID=%v, status=%v", modelID, md.Meta.Status)
			}
			protocol = md.Meta.Protocol
			config = md.Meta.ConnConfig
			mdl = md
			break
		}
	}

	if config == nil {
		logs.CtxErrorf(ctx, "[GetModel] Model ID=%d found but ConnConfig is nil!", modelID)
		return nil, nil, fmt.Errorf("model type %v ,not found config ", modelID)
	}

	if len(protocol) == 0 {
		return nil, nil, fmt.Errorf("model type %v ,not found protocol ", modelID)
	}

	if params.TopP != nil {
		config.TopP = ptr.Of(float32(ptr.From(params.TopP)))
	}

	if params.TopK != nil {
		config.TopK = params.TopK
	}

	if params.Temperature != nil {
		config.Temperature = ptr.Of(float32(ptr.From(params.Temperature)))
	}

	config.MaxTokens = ptr.Of(params.MaxTokens)

	// Whether you need to use a pointer
	config.FrequencyPenalty = ptr.Of(float32(params.FrequencyPenalty))
	config.PresencePenalty = ptr.Of(float32(params.PresencePenalty))

	cm, err := m.factory.CreateChatModel(ctx, protocol, config)
	if err != nil {
		return nil, nil, err
	}

	return cm, mdl, nil
}

// getHiAgentModel 获取外部智能体模型实例（支持 HiAgent 和 Dify）
func (m *modelManager) getHiAgentModel(ctx context.Context, params *model.LLMParams) (eino.BaseChatModel, *modelmgr.Model, error) {
	// 1. 验证必要参数
	if params.HiAgentID == "" {
		return nil, nil, fmt.Errorf("external_agent_id is required for external agent model")
	}
	if params.HiAgentSpaceID == 0 {
		return nil, nil, fmt.Errorf("space_id is required for external agent model")
	}

	// 2. 优先使用前端传递的 ExternalAgentPlatform 字段
	platform := params.ExternalAgentPlatform
	logs.CtxInfof(ctx, "🔍 Getting external agent model: agent_id=%s, space_id=%d, platform=%s",
		params.HiAgentID, params.HiAgentSpaceID, platform)

	// 3. 从 external_agent_config 表获取外部智能体配置
	// 注意：前端传递的 hiagentId 可能是数字 ID 或字符串 agent_id
	type ExternalAgentConfig struct {
		ID          int64   `gorm:"column:id;primaryKey"`
		SpaceID     int64   `gorm:"column:space_id"`
		Name        string  `gorm:"column:name"`
		Description *string `gorm:"column:description"`
		Platform    string  `gorm:"column:platform"`
		AgentURL    string  `gorm:"column:agent_url"`
		AgentKey    *string `gorm:"column:agent_key"`
		AgentID     *string `gorm:"column:agent_id"`
		Status      int32   `gorm:"column:status"`
	}

	// 获取数据库连接
	db, err := mysql.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var agentConfig ExternalAgentConfig

	// 尝试两种查询方式：
	// 1. 先尝试用 ID 查询（假设 hiagentId 是数字）
	// 2. 如果失败，尝试用 agent_id 字段查询（假设 hiagentId 是字符串）
	queryErr := db.WithContext(ctx).Table("external_agent_config").
		Where("id = ? AND space_id = ? AND deleted_at IS NULL", params.HiAgentID, params.HiAgentSpaceID).
		First(&agentConfig).Error

	if queryErr == gorm.ErrRecordNotFound {
		// 尝试用 agent_id 字段查询
		queryErr = db.WithContext(ctx).Table("external_agent_config").
			Where("agent_id = ? AND space_id = ? AND deleted_at IS NULL", params.HiAgentID, params.HiAgentSpaceID).
			First(&agentConfig).Error
	}

	if queryErr != nil {
		logs.CtxErrorf(ctx, "❌ Failed to get external agent from external_agent_config table: agent_id=%s, space_id=%d, error=%v",
			params.HiAgentID, params.HiAgentSpaceID, queryErr)
		return nil, nil, fmt.Errorf("failed to get external agent: %w", queryErr)
	}

	// 调试：打印从数据库读取的实际信息
	apiKeyPreview := "nil"
	if agentConfig.AgentKey != nil {
		if len(*agentConfig.AgentKey) > 10 {
			apiKeyPreview = (*agentConfig.AgentKey)[:10] + "..."
		} else {
			apiKeyPreview = *agentConfig.AgentKey
		}
	}
	logs.CtxInfof(ctx, "✅ External Agent loaded from DB - id=%d, agent_id=%v, name=%s, endpoint=%s, platform=%s, api_key_preview=%s",
		agentConfig.ID, agentConfig.AgentID, agentConfig.Name, agentConfig.AgentURL, agentConfig.Platform, apiKeyPreview)

	// 3. 检查外部智能体状态
	if agentConfig.Status != 1 {
		return nil, nil, fmt.Errorf("external agent is disabled: id=%d, status=%d", agentConfig.ID, agentConfig.Status)
	}

	// 4. 根据平台类型创建对应的模型实例
	var baseChatModel eino.BaseChatModel

	// 优先使用数据库中的 platform 字段，如果前端也传了就验证一致性
	platform = agentConfig.Platform
	if params.ExternalAgentPlatform != "" && params.ExternalAgentPlatform != platform {
		logs.CtxWarnf(ctx, "⚠️ Platform mismatch: frontend=%s, database=%s, using database value",
			params.ExternalAgentPlatform, platform)
	}

	logs.CtxInfof(ctx, "🎯 Using platform: %s for agent: id=%d", platform, agentConfig.ID)

	switch platform {
	case "dify":
		// 创建 Dify 模型
		if agentConfig.AgentKey == nil || *agentConfig.AgentKey == "" {
			return nil, nil, fmt.Errorf("agent_key is required for Dify agent")
		}

		// 构建 Dify agent 实体
		agentIDStr := ""
		if agentConfig.AgentID != nil {
			agentIDStr = *agentConfig.AgentID
		}

		difyAgent := &ynet_agent.DifyAgent{
			AgentID:     agentIDStr,
			Name:        agentConfig.Name,
			Description: agentConfig.Description,
			APIEndpoint: agentConfig.AgentURL,
			APIKey:      *agentConfig.AgentKey,
			SpaceID:     agentConfig.SpaceID,
		}
		difyModel, err := ynet_agent.NewDifyAgentChatModel(ctx, difyAgent)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create dify model: %w", err)
		}
		baseChatModel = difyModel
		logs.CtxInfof(ctx, "✅ Created Dify agent model: id=%d", agentConfig.ID)

	case "hiagent":
		// 创建 HiAgent 模型
		if agentConfig.AgentKey == nil || *agentConfig.AgentKey == "" {
			return nil, nil, fmt.Errorf("agent_key is required for HiAgent")
		}

		// 将 ExternalAgentConfig 转换为 HiAgent 实体
		agentIDStr := ""
		if agentConfig.AgentID != nil {
			agentIDStr = *agentConfig.AgentID
		}

		hiAgent := &ynet_agent.HiAgent{
			AgentID:     agentIDStr,
			SpaceID:     agentConfig.SpaceID,
			Name:        agentConfig.Name,
			Description: agentConfig.Description,
			Endpoint:    agentConfig.AgentURL,
			APIKey:      agentConfig.AgentKey,
			Status:      agentConfig.Status,
			Meta:        ynet_agent.MetaData{"platform": platform},
		}

		baseChatModel = ynet_agent.NewHiAgentChatModel(hiAgent)
		logs.CtxInfof(ctx, "✅ Created HiAgent model: id=%d", agentConfig.ID)

	default:
		return nil, nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	// 4.5 包装模型，注入ExecuteConfig到context
	chatModel := &hiAgentModelWrapper{
		base: baseChatModel,
	}

	// 5. 构造Model元信息（用于统计和展示）
	modelID := ""
	if agentConfig.AgentID != nil {
		modelID = *agentConfig.AgentID
	}

	modelInfo := &modelmgr.Model{
		ID:   0, // 外部智能体使用虚拟ID
		Name: agentConfig.Name,
		Meta: modelmgr.ModelMeta{
			Protocol: chatmodel.Protocol(platform),
			Status:   modelmgr.StatusInUse,
			ConnConfig: &chatmodel.Config{
				Model: modelID,
			},
		},
	}

	logs.CtxInfof(ctx, "✅ Created external agent model: id=%d, agent_id=%s, name=%s, platform=%s, space_id=%d",
		agentConfig.ID, modelID, agentConfig.Name, platform, params.HiAgentSpaceID)

	return chatModel, modelInfo, nil
}

// hiAgentModelWrapper 包装HiAgent模型，注入ExecuteConfig到context
type hiAgentModelWrapper struct {
	base eino.BaseChatModel
}

func (w *hiAgentModelWrapper) Generate(ctx context.Context, messages []*schema.Message, opts ...eino.Option) (*schema.Message, error) {
	// 从workflow execute context获取ExecuteConfig并注入
	ctx = w.injectExecuteConfig(ctx)
	return w.base.Generate(ctx, messages, opts...)
}

func (w *hiAgentModelWrapper) Stream(ctx context.Context, messages []*schema.Message, opts ...eino.Option) (*schema.StreamReader[*schema.Message], error) {
	// 从workflow execute context获取ExecuteConfig并注入
	ctx = w.injectExecuteConfig(ctx)
	return w.base.Stream(ctx, messages, opts...)
}

// injectExecuteConfig 从workflow execute context提取ExecuteConfig并注入到context
func (w *hiAgentModelWrapper) injectExecuteConfig(ctx context.Context) context.Context {
	// 使用service包的公共辅助函数获取ExecuteConfig
	exeCfg := service.ExtractExecuteConfig(ctx)
	if exeCfg != nil {
		ctx = ynet_agent.SetExecuteConfigToContext(ctx, exeCfg)
		logs.CtxInfof(ctx, "✅ injected ExecuteConfig to HiAgent context: conversation_id=%v, section_id=%v",
			exeCfg.ConversationID, exeCfg.SectionID)
		return ctx
	}

	logs.CtxWarnf(ctx, "❌ no workflow execute context found, HiAgent will create new conversation each time")
	return ctx
}

// getSingleAgentModel 获取内部智能体模型实例
func (m *modelManager) getSingleAgentModel(ctx context.Context, params *model.LLMParams) (eino.BaseChatModel, *modelmgr.Model, error) {
	// 1. 验证必要参数
	if params.SingleagentID == "" {
		return nil, nil, fmt.Errorf("singleagent_id is required for SingleAgent")
	}

	logs.CtxInfof(ctx, "🔍 Creating SingleAgent model: agent_id=%s, model_name=%s", params.SingleagentID, params.ModelName)

	// 2. 创建 SingleAgent 模型实例
	// SingleAgent 使用内部调用，不需要查询 external_agent_config 表
	singleAgentModel, err := ynet_agent.NewSingleAgentChatModel(ctx, params.SingleagentID, params.HiAgentSpaceID, params.ModelName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SingleAgent model: %w", err)
	}

	// 3. 包装模型，注入 ExecuteConfig 到 context
	chatModel := &hiAgentModelWrapper{
		base: singleAgentModel,
	}

	// 4. 构造 Model 元信息（用于统计和展示）
	modelInfo := &modelmgr.Model{
		ID:   0, // 内部智能体使用虚拟ID
		Name: params.ModelName,
		Meta: modelmgr.ModelMeta{
			Protocol: chatmodel.Protocol("singleagent"),
			Status:   modelmgr.StatusInUse,
			ConnConfig: &chatmodel.Config{
				Model: params.SingleagentID,
			},
		},
	}

	logs.CtxInfof(ctx, "✅ Created SingleAgent model: agent_id=%s, name=%s", params.SingleagentID, params.ModelName)
	return chatModel, modelInfo, nil
}
