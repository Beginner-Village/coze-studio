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

	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/internal/dal"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

type Repository interface {
	UpsertProduct(ctx context.Context, product *entity.Product) error
	GetProduct(ctx context.Context, productID int64) (*entity.Product, error)
	GetProductBySource(ctx context.Context, sourceType string, sourceID int64) (*entity.Product, error)
	ListProducts(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error)
	UpsertVersion(ctx context.Context, version *entity.ProductVersion) error
	ListVersions(ctx context.Context, productID int64) ([]*entity.ProductVersion, error)
	InstallProduct(ctx context.Context, installation *entity.ProductInstallation) error
	UpdateInstallation(ctx context.Context, installation *entity.ProductInstallation) error
	GetInstallation(ctx context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error)
	ListInstallations(ctx context.Context, req *entity.ListInstallationsRequest) ([]*entity.ProductInstallation, error)
	CreateAudit(ctx context.Context, audit *entity.AuditLog) error
	ListAudits(ctx context.Context, req *entity.ListAuditsRequest) ([]*entity.AuditLog, error)
	UpsertSessionRuntimeConfig(ctx context.Context, config *entity.SessionRuntimeConfig) error
	GetSessionRuntimeConfig(ctx context.Context, conversationID int64) (*entity.SessionRuntimeConfig, error)
	DeleteSessionRuntimeConfig(ctx context.Context, conversationID int64) error
}

func NewRepository(db *gorm.DB, idGen idgen.IDGenerator) Repository {
	return dal.NewProductDAO(db, idGen)
}
