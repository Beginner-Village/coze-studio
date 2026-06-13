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

package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/internal/dal"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

type OperationLogRepository interface {
	BatchCreate(ctx context.Context, logs []*entity.OperationLog) error
	List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error)
	DeleteBefore(ctx context.Context, ts int64) (int64, error)
}

type operationLogRepo struct {
	dao *dal.OperationLogDAO
}

func NewOperationLogRepository(db *gorm.DB, idgen idgen.IDGenerator) OperationLogRepository {
	return &operationLogRepo{dao: dal.NewOperationLogDAO(db, idgen)}
}

func (r *operationLogRepo) BatchCreate(ctx context.Context, logs []*entity.OperationLog) error {
	return r.dao.BatchCreate(ctx, logs)
}

func (r *operationLogRepo) List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	return r.dao.List(ctx, f)
}

func (r *operationLogRepo) DeleteBefore(ctx context.Context, ts int64) (int64, error) {
	return r.dao.DeleteBefore(ctx, ts)
}
