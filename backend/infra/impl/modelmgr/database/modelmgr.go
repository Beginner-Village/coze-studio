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

package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/domain/model/entity"
	"github.com/coze-dev/coze-studio/backend/infra/contract/cache"
	"github.com/coze-dev/coze-studio/backend/infra/contract/chatmodel"
	"github.com/coze-dev/coze-studio/backend/infra/contract/modelmgr"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

 const (
	 // 空间级别的模型缓存前缀
	 spaceModelCachePrefix = "space:"
	 modelCacheTTL         = 5 * time.Minute
 )

 // ModelMgr 数据库模型管理器
 type ModelMgr struct {
	 db    *gorm.DB
	 redis cache.Cmdable
	 mu    sync.RWMutex
 }

 // NewModelMgr 创建数据库模型管理器实例
 func NewModelMgr(db *gorm.DB, redis cache.Cmdable) (modelmgr.Manager, error) {
	 if db == nil {
		 return nil, fmt.Errorf("database connection is required")
	 }

	 mgr := &ModelMgr{
		 db:    db,
		 redis: redis,
	 }

	 return mgr, nil
 }

 // ListModel 查询模型列表
 func (m *ModelMgr) ListModel(ctx context.Context, req *modelmgr.ListModelRequest) (*modelmgr.ListModelResponse, error) {
	 // Check if SpaceID is provided for filtering
	 if req.SpaceID != nil {
		 return m.listModelsBySpace(ctx, req)
	 }

	 // 先查询 model_entity
	 query := m.db.WithContext(ctx).Model(&entity.ModelEntity{}).
		 Where("deleted_at IS NULL")

	 // 处理模糊查询
	 if req.FuzzyModelName != nil && *req.FuzzyModelName != "" {
		 query = query.Where("name LIKE ?", "%"+*req.FuzzyModelName+"%")
	 }

	 // 处理游标
	 if req.Cursor != nil && *req.Cursor != "" {
		 cursorID, err := strconv.ParseInt(*req.Cursor, 10, 64)
		 if err != nil {
			 return nil, fmt.Errorf("invalid cursor: %w", err)
		 }
		 query = query.Where("id > ?", cursorID)
	 }

	 // 设置限制和排序
	 limit := req.Limit
	 if limit <= 0 {
		 limit = 20
	 }
	 query = query.Order("id ASC").Limit(limit + 1)

	 // 执行查询
	 var entities []entity.ModelEntity
	 if err := query.Find(&entities).Error; err != nil {
		 return nil, fmt.Errorf("failed to query model entities: %w", err)
	 }

	 if len(entities) == 0 {
		 return &modelmgr.ListModelResponse{
			 ModelList:  []*modelmgr.Model{},
			 HasMore:    false,
			 NextCursor: nil,
		 }, nil
	 }

	 // 收集 meta_id
	 metaIDs := make([]uint64, 0, len(entities))
	 for _, e := range entities {
		 metaIDs = append(metaIDs, e.MetaID)
	 }

	 // 查询 model_meta
	 var metas []entity.ModelMeta
	 metaQuery := m.db.WithContext(ctx).Model(&entity.ModelMeta{}).
		 Where("id IN ?", metaIDs).
		 Where("deleted_at IS NULL")

	 // 处理状态过滤
	 if len(req.Status) > 0 {
		 statusValues := make([]int, len(req.Status))
		 for i, s := range req.Status {
			 statusValues[i] = int(s)
		 }
		 metaQuery = metaQuery.Where("status IN ?", statusValues)
	 } else {
		 // 默认查询 default 和 in_use 状态
		 metaQuery = metaQuery.Where("status IN ?", []int{int(modelmgr.StatusDefault), int(modelmgr.StatusInUse)})
	 }

	 if err := metaQuery.Find(&metas).Error; err != nil {
		 return nil, fmt.Errorf("failed to query model metas: %w", err)
	 }

	 // 构建 meta map
	 metaMap := make(map[uint64]*entity.ModelMeta)
	 for i := range metas {
		 metaMap[metas[i].ID] = &metas[i]
	 }

	 // 转换结果
	 models := make([]*modelmgr.Model, 0, len(entities))
	 hasMore := false
	 var nextCursor *string

	 for i, entity := range entities {
		 if i >= limit {
			 hasMore = true
			 break
		 }

		 meta, ok := metaMap[entity.MetaID]
		 if !ok {
			 // 如果没有找到对应的 meta，跳过
			 continue
		 }

		 model, err := m.convertToModel(&entity, meta)
		 if err != nil {
			 logs.Warnf("failed to convert model, id=%d, err=%v", entity.ID, err)
			 continue
		 }

		 models = append(models, model)
		 // 使用最后一个模型的ID作为下一个游标
		 cursor := strconv.FormatInt(int64(entity.ID), 10)
		 nextCursor = &cursor
	 }

	 return &modelmgr.ListModelResponse{
		 ModelList:  models,
		 HasMore:    hasMore,
		 NextCursor: nextCursor,
	 }, nil
 }

 // listModelsBySpace 根据空间ID查询模型列表（包含空间私有模型和公共模型）
 func (m *ModelMgr) listModelsBySpace(ctx context.Context, req *modelmgr.ListModelRequest) (*modelmgr.ListModelResponse, error) {
	 logs.Infof("listModelsBySpace called with spaceID: %d", *req.SpaceID)

	 limit := req.Limit
	 if limit <= 0 {
		 limit = 20
	 }

	 models := make([]*modelmgr.Model, 0)

	 // ==================== 第一部分：查询空间私有模型 ====================
	 spaceModelQuery := m.db.WithContext(ctx).
		 Table("space_model sm").
		 Select("sm.model_entity_id, sm.id as space_model_id, sm.status").
		 Where("sm.space_id = ?", *req.SpaceID).
		 Where("sm.deleted_at IS NULL").
		 Order("sm.id ASC")

	 type spaceModelResult struct {
		 ModelEntityID uint64 `json:"model_entity_id"`
		 SpaceModelID  uint64 `json:"space_model_id"`
		 Status        int    `json:"status"`
	 }

	 var spaceModels []spaceModelResult
	 if err := spaceModelQuery.Scan(&spaceModels).Error; err != nil {
		 return nil, fmt.Errorf("failed to query space models: %w", err)
	 }

	 // 处理空间私有模型
	 if len(spaceModels) > 0 {
		 entityIDs := make([]uint64, 0, len(spaceModels))
		 spaceModelStatusMap := make(map[uint64]int)
		 for _, sm := range spaceModels {
			 entityIDs = append(entityIDs, sm.ModelEntityID)
			 spaceModelStatusMap[sm.ModelEntityID] = sm.Status
		 }

		 entityQuery := m.db.WithContext(ctx).Model(&entity.ModelEntity{}).
			 Where("id IN ?", entityIDs).
			 Where("deleted_at IS NULL")

		 if req.FuzzyModelName != nil && *req.FuzzyModelName != "" {
			 entityQuery = entityQuery.Where("name LIKE ?", "%"+*req.FuzzyModelName+"%")
		 }

		 var entities []entity.ModelEntity
		 if err := entityQuery.Find(&entities).Error; err != nil {
			 return nil, fmt.Errorf("failed to query model entities: %w", err)
		 }

		 metaIDs := make([]uint64, 0, len(entities))
		 for _, e := range entities {
			 metaIDs = append(metaIDs, e.MetaID)
		 }

		 if len(metaIDs) > 0 {
			 var metas []entity.ModelMeta
			 metaQuery := m.db.WithContext(ctx).Model(&entity.ModelMeta{}).
				 Where("id IN ?", metaIDs).
				 Where("deleted_at IS NULL")

			 if len(req.Status) > 0 {
				 statusValues := make([]int, len(req.Status))
				 for i, s := range req.Status {
					 statusValues[i] = int(s)
				 }
				 metaQuery = metaQuery.Where("status IN ?", statusValues)
			 } else {
				 metaQuery = metaQuery.Where("status IN ?", []int{int(modelmgr.StatusDefault), int(modelmgr.StatusInUse)})
			 }

			 if err := metaQuery.Find(&metas).Error; err != nil {
				 return nil, fmt.Errorf("failed to query model metas: %w", err)
			 }

			 metaMap := make(map[uint64]*entity.ModelMeta)
			 for i := range metas {
				 metaMap[metas[i].ID] = &metas[i]
			 }

			 for _, ent := range entities {
				 meta, ok := metaMap[ent.MetaID]
				 if !ok {
					 continue
				 }

				 model, err := m.convertToModel(&ent, meta)
				 if err != nil {
					 logs.Warnf("failed to convert model, id=%d, err=%v", ent.ID, err)
					 continue
				 }

				 if spaceStatus, ok := spaceModelStatusMap[ent.ID]; ok {
					 model.Meta.Status = modelmgr.ModelStatus(spaceStatus)
				 }

				 model.IsPublic = false // 标记为空间私有模型
				 models = append(models, model)

				 if m.redis != nil && req.SpaceID != nil {
					 _ = m.cacheSpaceModel(ctx, *req.SpaceID, model)
				 }
			 }
		 }
	 }

	 // ==================== 第二部分：查询公共模型 ====================
	 publicModelQuery := m.db.WithContext(ctx).Model(&entity.ModelEntity{}).
		 Where("is_public = 1").
		 Where("status = 1"). // 只查询启用状态的公共模型
		 Where("deleted_at IS NULL").
		 Order("created_at DESC")

	 if req.FuzzyModelName != nil && *req.FuzzyModelName != "" {
		 publicModelQuery = publicModelQuery.Where("name LIKE ?", "%"+*req.FuzzyModelName+"%")
	 }

	 var publicEntities []entity.ModelEntity
	 if err := publicModelQuery.Find(&publicEntities).Error; err != nil {
		 logs.Warnf("failed to query public models: %v", err)
		 // 不返回错误，继续返回已有的空间模型
	 } else if len(publicEntities) > 0 {
		 publicMetaIDs := make([]uint64, 0, len(publicEntities))
		 for _, e := range publicEntities {
			 publicMetaIDs = append(publicMetaIDs, e.MetaID)
		 }

		 var publicMetas []entity.ModelMeta
		 publicMetaQuery := m.db.WithContext(ctx).Model(&entity.ModelMeta{}).
			 Where("id IN ?", publicMetaIDs).
			 Where("deleted_at IS NULL")

		 if len(req.Status) > 0 {
			 statusValues := make([]int, len(req.Status))
			 for i, s := range req.Status {
				 statusValues[i] = int(s)
			 }
			 publicMetaQuery = publicMetaQuery.Where("status IN ?", statusValues)
		 } else {
			 publicMetaQuery = publicMetaQuery.Where("status IN ?", []int{int(modelmgr.StatusDefault), int(modelmgr.StatusInUse)})
		 }

		 if err := publicMetaQuery.Find(&publicMetas).Error; err != nil {
			 logs.Warnf("failed to query public model metas: %v", err)
		 } else {
			 publicMetaMap := make(map[uint64]*entity.ModelMeta)
			 for i := range publicMetas {
				 publicMetaMap[publicMetas[i].ID] = &publicMetas[i]
			 }

			 for _, ent := range publicEntities {
				 meta, ok := publicMetaMap[ent.MetaID]
				 if !ok {
					 continue
				 }

				 model, err := m.convertToModel(&ent, meta)
				 if err != nil {
					 logs.Warnf("failed to convert public model, id=%d, err=%v", ent.ID, err)
					 continue
				 }

				 model.IsPublic = true // 标记为公共模型
				 models = append(models, model)
			 }
		 }
	 }

	 // 处理分页
	 hasMore := len(models) > limit
	 if hasMore {
		 models = models[:limit]
	 }

	 var nextCursor *string
	 if len(models) > 0 {
		 cursor := strconv.FormatInt(models[len(models)-1].ID, 10)
		 nextCursor = &cursor
	 }

	 return &modelmgr.ListModelResponse{
		 ModelList:  models,
		 HasMore:    hasMore,
		 NextCursor: nextCursor,
	 }, nil
 }

 // ListInUseModel 查询使用中的模型
 func (m *ModelMgr) ListInUseModel(ctx context.Context, limit int, cursor *string) (*modelmgr.ListModelResponse, error) {
	 return m.ListModel(ctx, &modelmgr.ListModelRequest{
		 Status: []modelmgr.ModelStatus{modelmgr.StatusDefault, modelmgr.StatusInUse},
		 Limit:  limit,
		 Cursor: cursor,
	 })
 }

 // MGetModelByID 批量获取模型
 func (m *ModelMgr) MGetModelByID(ctx context.Context, req *modelmgr.MGetModelRequest) ([]*modelmgr.Model, error) {
	 if len(req.IDs) == 0 {
		 return []*modelmgr.Model{}, nil
	 }

	 models := make([]*modelmgr.Model, 0, len(req.IDs))
	 missedIDs := make([]int64, 0)

	 // 如果提供了SpaceID，尝试从空间级别的缓存获取
	 if req.SpaceID != nil && m.redis != nil {
		 for _, id := range req.IDs {
			 model, err := m.getSpaceModelFromCache(ctx, *req.SpaceID, id)
			 if err == nil && model != nil {
				 // 检查模型在空间中的状态
				 available, err := m.CheckModelAvailableInSpace(ctx, *req.SpaceID, id)
				 if err == nil && available {
					 models = append(models, model)
				 } else {
					 missedIDs = append(missedIDs, id)
				 }
			 } else {
				 missedIDs = append(missedIDs, id)
			 }
		 }
	 } else {
		 // 没有空间上下文，直接从数据库获取
		 missedIDs = req.IDs
	 }

	 // 从数据库获取缺失的模型
	 if len(missedIDs) > 0 {
		 logs.CtxInfof(ctx, "[MGetModelByID] Querying database for model IDs: %v", missedIDs)

		 // 查询 model_entity
		 var entities []entity.ModelEntity
		 err := m.db.WithContext(ctx).Model(&entity.ModelEntity{}).
			 Where("id IN ?", missedIDs).
			 Where("deleted_at IS NULL").
			 Find(&entities).Error

		 if err != nil {
			 return nil, fmt.Errorf("failed to get model entities by ids: %w", err)
		 }

		 logs.CtxInfof(ctx, "[MGetModelByID] Found %d entities", len(entities))

		 // 收集 meta_id
		 metaIDs := make([]uint64, 0, len(entities))
		 entityMap := make(map[uint64]*entity.ModelEntity)
		 for i := range entities {
			 metaIDs = append(metaIDs, entities[i].MetaID)
			 entityMap[entities[i].ID] = &entities[i]
			 logs.CtxInfof(ctx, "[MGetModelByID] Entity ID=%d, Name=%s, MetaID=%d, IsPublic=%d, Status=%d",
				 entities[i].ID, entities[i].Name, entities[i].MetaID, entities[i].IsPublic, entities[i].Status)
		 }

		 // 查询 model_meta
		 var metas []entity.ModelMeta
		 err = m.db.WithContext(ctx).Model(&entity.ModelMeta{}).
			 Where("id IN ?", metaIDs).
			 Where("deleted_at IS NULL").
			 Find(&metas).Error

		 if err != nil {
			 return nil, fmt.Errorf("failed to get model metas by ids: %w", err)
		 }

		 logs.CtxInfof(ctx, "[MGetModelByID] Found %d metas for metaIDs: %v", len(metas), metaIDs)
		 for i := range metas {
			 connConfigPreview := "nil"
			 if metas[i].ConnConfig != nil {
				 if len(*metas[i].ConnConfig) > 100 {
					 connConfigPreview = (*metas[i].ConnConfig)[:100] + "..."
				 } else {
					 connConfigPreview = *metas[i].ConnConfig
				 }
			 }
			 logs.CtxInfof(ctx, "[MGetModelByID] Meta ID=%d, Protocol=%s, Status=%d, ConnConfig=%s",
				 metas[i].ID, metas[i].Protocol, metas[i].Status, connConfigPreview)
		 }

		 // 构建 meta map
		 metaMap := make(map[uint64]*entity.ModelMeta)
		 for i := range metas {
			 metaMap[metas[i].ID] = &metas[i]
		 }

		 // 按照请求的 ID 顺序返回
		 for _, id := range missedIDs {
			 entity, ok := entityMap[uint64(id)]
			 if !ok {
				 logs.CtxWarnf(ctx, "[MGetModelByID] Entity not found for ID=%d", id)
				 continue
			 }

			 meta, ok := metaMap[entity.MetaID]
			 if !ok {
				 logs.CtxWarnf(ctx, "[MGetModelByID] Meta not found for Entity ID=%d, MetaID=%d", entity.ID, entity.MetaID)
				 continue
			 }
			 logs.CtxInfof(ctx, "[MGetModelByID] Processing entity ID=%d with meta ID=%d", entity.ID, meta.ID)

			 model, err := m.convertToModel(entity, meta)
			 if err != nil {
				 logs.Warnf("failed to convert model, id=%d, err=%v", entity.ID, err)
				 continue
			 }

			 // 如果提供了SpaceID，检查模型在空间中的可用性
			 if req.SpaceID != nil {
				 available, err := m.CheckModelAvailableInSpace(ctx, *req.SpaceID, model.ID)
				 if err != nil || !available {
					 // 模型在该空间不可用，跳过
					 continue
				 }
			 }

			 models = append(models, model)

			 // 如果有空间上下文和Redis，缓存到空间级别
			 if req.SpaceID != nil && m.redis != nil {
				 _ = m.cacheSpaceModel(ctx, *req.SpaceID, model)
			 }
		 }
	 }

	 // 按照请求的ID顺序返回
	 idToModel := make(map[int64]*modelmgr.Model)
	 for _, model := range models {
		 idToModel[model.ID] = model
	 }

	 result := make([]*modelmgr.Model, 0, len(req.IDs))
	 for _, id := range req.IDs {
		 if model, ok := idToModel[id]; ok {
			 result = append(result, model)
		 }
	 }

	 return result, nil
 }

 // convertToModel 将数据库实体转换为模型
 func (m *ModelMgr) convertToModel(entity *entity.ModelEntity, meta *entity.ModelMeta) (*modelmgr.Model, error) {
	 model := &modelmgr.Model{
		 ID:      int64(entity.ID),
		 Name:    entity.Name,
		 IconURI: meta.IconURI,
		 IconURL: meta.IconURL,
		 Meta: modelmgr.ModelMeta{
			 // Name field doesn't exist in ModelMeta, commented out temporarily
			 // Name:     meta.ModelName,
			 Protocol: chatmodel.Protocol(meta.Protocol),
			 Status:   modelmgr.ModelStatus(meta.Status),
		 },
	 }

	 // 解析描述
	 if entity.Description != nil && *entity.Description != "" {
		 var desc modelmgr.MultilingualText
		 if err := json.Unmarshal([]byte(*entity.Description), &desc); err != nil {
			 // 如果不是JSON格式，作为中文描述处理
			 desc.ZH = *entity.Description
		 }
		 model.Description = &desc
	 }

	 // 解析默认参数
	 if entity.DefaultParams != "" {
		 logs.Infof("Parsing DefaultParams for model %d: %s", entity.ID, entity.DefaultParams)
		 var params []*modelmgr.Parameter
		 if err := json.Unmarshal([]byte(entity.DefaultParams), &params); err != nil {
			 return nil, fmt.Errorf("failed to unmarshal default params: %w", err)
		 }
		 model.DefaultParameters = params
		 logs.Infof("Parsed %d parameters for model %d", len(params), entity.ID)
	 } else {
		 logs.Warnf("Model %d has empty DefaultParams", entity.ID)
		 model.DefaultParameters = []*modelmgr.Parameter{}
	 }

	 // 解析能力
	 if meta.Capability != nil && *meta.Capability != "" {
		 var cap modelmgr.Capability
		 if err := json.Unmarshal([]byte(*meta.Capability), &cap); err != nil {
			 return nil, fmt.Errorf("failed to unmarshal capability: %w", err)
		 }
		 model.Meta.Capability = &cap
	 }

	 // 解析连接配置
	 logs.Infof("[convertToModel] Model ID=%d, meta.ConnConfig is nil: %v", entity.ID, meta.ConnConfig == nil)
	 if meta.ConnConfig != nil {
		 logs.Infof("[convertToModel] Model ID=%d, meta.ConnConfig value: %s", entity.ID, *meta.ConnConfig)
	 }
	 if meta.ConnConfig != nil && *meta.ConnConfig != "" {
		 // 先尝试解析为 map 以处理特殊字段
		 var configMap map[string]interface{}
		 if err := json.Unmarshal([]byte(*meta.ConnConfig), &configMap); err != nil {
			 logs.Warnf("failed to unmarshal conn config as map, id=%d, err=%v", entity.ID, err)
		 } else {
			 // 处理 timeout 字段
			 if timeout, ok := configMap["timeout"]; ok {
				 switch v := timeout.(type) {
				 case string:
					 // 如果是字符串，尝试解析为 duration
					 if d, err := time.ParseDuration(v); err == nil {
						 // 如果超时小于30秒，设置为300秒
						 if d < 30*time.Second {
							 configMap["timeout"] = int64(300 * time.Second)
						 } else {
							 configMap["timeout"] = int64(d)
						 }
					 } else {
						 // 如果解析失败，删除该字段
						 delete(configMap, "timeout")
					 }
				 case float64:
					 // 如果是数字，假设是秒数，转换为纳秒
					 // 如果超时小于30秒，设置为300秒
					 if v < 30 {
						 configMap["timeout"] = int64(300 * time.Second)
					 } else {
						 configMap["timeout"] = int64(v * float64(time.Second))
					 }
				 }
			 }

			 // 重新序列化并解析
			 fixedJSON, err := json.Marshal(configMap)
			 if err != nil {
				 logs.Warnf("failed to marshal fixed config map, id=%d, err=%v", entity.ID, err)
			 } else {
				 var config chatmodel.Config
				 if err := json.Unmarshal(fixedJSON, &config); err != nil {
					 logs.Warnf("failed to unmarshal conn config, id=%d, err=%v", entity.ID, err)
				 } else {
					 model.Meta.ConnConfig = &config
					 logs.Infof("[convertToModel] Model ID=%d, ConnConfig successfully parsed, BaseURL=%s, Model=%s",
						 entity.ID, config.BaseURL, config.Model)
				 }
			 }
		 }
	 } else {
		 logs.Warnf("[convertToModel] Model ID=%d has nil or empty ConnConfig!", entity.ID)
	 }

	 logs.Infof("[convertToModel] Model ID=%d conversion done, ConnConfig is nil: %v", entity.ID, model.Meta.ConnConfig == nil)
	 return model, nil
 }

 // getModelFromCache 从缓存获取模型
 // getSpaceModelFromCache 从缓存获取空间级别的模型
 func (m *ModelMgr) getSpaceModelFromCache(ctx context.Context, spaceID uint64, modelID int64) (*modelmgr.Model, error) {
	 key := fmt.Sprintf("%s%d:model:%d", spaceModelCachePrefix, spaceID, modelID)
	 data, err := m.redis.Get(ctx, key).Bytes()
	 if err != nil {
		 return nil, err
	 }

	 var model modelmgr.Model
	 if err := json.Unmarshal(data, &model); err != nil {
		 return nil, err
	 }

	 return &model, nil
 }

 // cacheModel 缓存模型
 // cacheSpaceModel 缓存空间级别的模型
 func (m *ModelMgr) cacheSpaceModel(ctx context.Context, spaceID uint64, model *modelmgr.Model) error {
	 key := fmt.Sprintf("%s%d:model:%d", spaceModelCachePrefix, spaceID, model.ID)
	 data, err := json.Marshal(model)
	 if err != nil {
		 return err
	 }

	 return m.redis.Set(ctx, key, data, modelCacheTTL).Err()
 }

 // RefreshCache 刷新缓存（需要知道空间ID才能清理）
 // 由于模型可能在多个空间中使用，这里需要清理所有相关的空间缓存
 func (m *ModelMgr) RefreshCache(ctx context.Context, modelID int64) error {
	 if m.redis == nil {
		 return nil
	 }

	 // 查询使用此模型的所有空间
	 var spaceModels []struct {
		 SpaceID uint64 `gorm:"column:space_id"`
	 }
	 err := m.db.WithContext(ctx).
		 Table("space_model").
		 Select("DISTINCT space_id").
		 Where("model_entity_id = ? AND deleted_at IS NULL", modelID).
		 Scan(&spaceModels).Error

	 if err != nil {
		 logs.CtxWarnf(ctx, "Failed to query spaces for model %d: %v", modelID, err)
		 return err
	 }

	 // 清理所有相关空间的缓存
	 for _, sm := range spaceModels {
		 key := fmt.Sprintf("%s%d:model:%d", spaceModelCachePrefix, sm.SpaceID, modelID)
		 if err := m.redis.Del(ctx, key).Err(); err != nil {
			 logs.CtxWarnf(ctx, "Failed to delete cache for space %d model %d: %v", sm.SpaceID, modelID, err)
		 }
	 }

	 logs.CtxInfof(ctx, "Model cache refreshed for model %d in %d spaces", modelID, len(spaceModels))
	 return nil
 }

 // RefreshSpaceModelCache 刷新特定空间的模型缓存
 func (m *ModelMgr) RefreshSpaceModelCache(ctx context.Context, spaceID uint64, modelID int64) error {
	 if m.redis == nil {
		 return nil
	 }

	 key := fmt.Sprintf("%s%d:model:%d", spaceModelCachePrefix, spaceID, modelID)
	 return m.redis.Del(ctx, key).Err()
 }

 // CheckModelAvailableInSpace 检查模型在特定空间是否可用
 // 模型可用的条件：
 // 1. 模型在 space_model 表中且状态为启用
 // 2. 或者模型是公共模型（is_public=1）且状态为启用
 func (m *ModelMgr) CheckModelAvailableInSpace(ctx context.Context, spaceID uint64, modelID int64) (bool, error) {
	 // 首先检查是否为空间私有模型
	 var spaceModelCount int64
	 err := m.db.WithContext(ctx).
		 Table("space_model").
		 Where("space_id = ? AND model_entity_id = ? AND deleted_at IS NULL AND status = 1", spaceID, modelID).
		 Count(&spaceModelCount).Error

	 if err != nil {
		 return false, fmt.Errorf("failed to check model availability in space: %w", err)
	 }

	 if spaceModelCount > 0 {
		 return true, nil
	 }

	 // 如果不是空间私有模型，检查是否为公共模型
	 var publicModelCount int64
	 err = m.db.WithContext(ctx).
		 Model(&entity.ModelEntity{}).
		 Where("id = ? AND is_public = 1 AND status = 1 AND deleted_at IS NULL", modelID).
		 Count(&publicModelCount).Error

	 if err != nil {
		 return false, fmt.Errorf("failed to check public model availability: %w", err)
	 }

	 return publicModelCount > 0, nil
 }

 // RefreshAllCache 刷新所有缓存
 func (m *ModelMgr) RefreshAllCache(ctx context.Context) error {
	 if m.redis == nil {
		 return nil
	 }

	 // 由于cache.Cmdable接口没有Scan方法，我们暂时跳过批量删除
	 // 在实际使用中，可以通过维护一个键列表或使用其他方式来管理缓存
	 logs.CtxInfof(ctx, "RefreshAllCache: cache.Cmdable interface does not support Scan operation, skipping batch cache refresh")
	 return nil
 }
