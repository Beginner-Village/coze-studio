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

package service

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/repository"
)

type Components struct {
	ProductRepo repository.Repository
}

type Service interface {
	SyncProduct(ctx context.Context, product *entity.Product, version *entity.ProductVersion) (*entity.Product, error)
	ListMarketplace(ctx context.Context, req *entity.ListProductsRequest) (*entity.ListProductsResult, error)
	GetVisibleProduct(ctx context.Context, productID, spaceID, userID int64) (*entity.Product, error)
	Install(ctx context.Context, productID, spaceID, userID int64, version string) (*entity.ProductInstallation, error)
	Uninstall(ctx context.Context, productID, spaceID, userID int64) error
	Upgrade(ctx context.Context, productID, spaceID, userID int64) (*entity.ProductInstallation, error)
	ListInstalled(ctx context.Context, spaceID, userID int64, productType entity.AIProductType) ([]*entity.ProductInstallation, error)
	RecordAudit(ctx context.Context, audit *entity.AuditLog) error
}

func NewService(c *Components) Service {
	return &productServiceImpl{repo: c.ProductRepo}
}
