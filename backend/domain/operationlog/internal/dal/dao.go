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

package dal

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

type OperationLogDAO struct {
	db    *gorm.DB
	idgen idgen.IDGenerator
}

func NewOperationLogDAO(db *gorm.DB, idgen idgen.IDGenerator) *OperationLogDAO {
	return &OperationLogDAO{db: db, idgen: idgen}
}

// BatchCreate 批量写入；为每条记录分配雪花 ID。
func (dao *OperationLogDAO) BatchCreate(ctx context.Context, logs []*entity.OperationLog) error {
	if len(logs) == 0 {
		return nil
	}
	ids, err := dao.idgen.GenMultiIDs(ctx, len(logs))
	if err != nil {
		return err
	}
	pos := make([]*model.OperationLog, 0, len(logs))
	for i, l := range logs {
		pos = append(pos, &model.OperationLog{
			ID:             ids[i],
			SpaceID:        l.SpaceID,
			OperatorID:     l.OperatorID,
			Module:         l.Module,
			ResourceType:   l.ResourceType,
			ResourceID:     l.ResourceID,
			ResourceName:   l.ResourceName,
			Action:         l.Action,
			Description:    l.Description,
			Method:         l.Method,
			Path:           l.Path,
			RequestSummary: l.RequestSummary,
			Status:         int32(l.Status),
			ErrorCode:      l.ErrorCode,
			ClientIP:       l.ClientIP,
			DurationMs:     l.DurationMs,
			LogID:          l.LogID,
			CreatedAt:      l.CreatedAt,
		})
	}
	return dao.db.WithContext(ctx).CreateInBatches(pos, 100).Error
}

// List 按过滤条件分页查询，按 created_at 倒序。
func (dao *OperationLogDAO) List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	q := dao.db.WithContext(ctx).Model(&model.OperationLog{}).Where("space_id = ?", f.SpaceID)
	if f.OperatorID != nil {
		q = q.Where("operator_id = ?", *f.OperatorID)
	}
	if f.ResourceType != nil {
		q = q.Where("resource_type = ?", *f.ResourceType)
	}
	if f.Action != nil {
		q = q.Where("action = ?", *f.Action)
	}
	if f.StartTime != nil {
		q = q.Where("created_at >= ?", *f.StartTime)
	}
	if f.EndTime != nil {
		q = q.Where("created_at <= ?", *f.EndTime)
	}
	if f.Keyword != nil && *f.Keyword != "" {
		kw := "%" + *f.Keyword + "%"
		q = q.Where("(description LIKE ? OR resource_name LIKE ?)", kw, kw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

	var pos []*model.OperationLog
	if err := q.Order("created_at DESC").
		Offset(int((page - 1) * size)).Limit(int(size)).
		Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	out := make([]*entity.OperationLog, 0, len(pos))
	for _, p := range pos {
		out = append(out, &entity.OperationLog{
			ID: p.ID, SpaceID: p.SpaceID, OperatorID: p.OperatorID,
			Module: p.Module, ResourceType: p.ResourceType, ResourceID: p.ResourceID,
			ResourceName: p.ResourceName, Action: p.Action, Description: p.Description,
			Method: p.Method, Path: p.Path, RequestSummary: p.RequestSummary,
			Status: entity.Status(p.Status), ErrorCode: p.ErrorCode, ClientIP: p.ClientIP,
			DurationMs: p.DurationMs, LogID: p.LogID, CreatedAt: p.CreatedAt,
		})
	}
	return out, total, nil
}

// DeleteBefore 删除 created_at 小于 ts 的记录，分批避免大事务。返回删除条数。
func (dao *OperationLogDAO) DeleteBefore(ctx context.Context, ts int64) (int64, error) {
	var totalDeleted int64
	for {
		res := dao.db.WithContext(ctx).
			Where("created_at < ?", ts).
			Limit(1000).
			Delete(&model.OperationLog{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}
		totalDeleted += res.RowsAffected
		if res.RowsAffected < 1000 {
			break
		}
	}
	return totalDeleted, nil
}
