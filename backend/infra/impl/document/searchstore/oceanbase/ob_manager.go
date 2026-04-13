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

package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// ManagerConfig holds the configuration for OceanBase vector store.
type ManagerConfig struct {
	DB        *gorm.DB           // required: OceanBase database connection (MySQL-compatible)
	Embedding embedding.Embedder // required: embedding provider

	DenseMetric        string // optional: "inner_product" (default), "l2", "cosine"
	HnswM              int    // optional: default 30
	HnswEfConstruction int    // optional: default 360
	BatchSize          int    // optional: default 100
}

func NewManager(config *ManagerConfig) (searchstore.Manager, error) {
	if config.DB == nil {
		return nil, fmt.Errorf("[NewManager] oceanbase DB not provided")
	}
	if config.Embedding == nil {
		return nil, fmt.Errorf("[NewManager] oceanbase embedder not provided")
	}

	if config.DenseMetric == "" {
		config.DenseMetric = "inner_product"
	}
	if config.HnswM == 0 {
		config.HnswM = 30
	}
	if config.HnswEfConstruction == 0 {
		config.HnswEfConstruction = 360
	}
	if config.BatchSize == 0 {
		config.BatchSize = defaultBatchSize
	}

	return &obManager{config: config}, nil
}

type obManager struct {
	config *ManagerConfig
}

func (m *obManager) Create(ctx context.Context, req *searchstore.CreateRequest) error {
	if req.CollectionName == "" || len(req.Fields) == 0 {
		return fmt.Errorf("[Create] invalid request params")
	}

	tableName, err := sanitizeIdentifier(req.CollectionName)
	if err != nil {
		return fmt.Errorf("[Create] %w", err)
	}

	exists, err := m.tableExists(ctx, tableName)
	if err != nil {
		return fmt.Errorf("[Create] check table existence failed, %w", err)
	}
	if exists {
		return nil
	}

	columns, indexingFields, err := m.buildColumns(req.Fields)
	if err != nil {
		return fmt.Errorf("[Create] build columns failed, %w", err)
	}

	// Add VECTOR INDEX definitions inline for vector columns
	for _, fieldName := range indexingFields {
		safeField, err := sanitizeIdentifier(fieldName)
		if err != nil {
			return fmt.Errorf("[Create] %w", err)
		}
		denseCol := denseFieldName(safeField)
		idxName := denseIndexName(safeField)

		distanceMetric := m.config.DenseMetric
		if distanceMetric == "inner_product" {
			distanceMetric = "l2" // OceanBase VECTOR INDEX uses l2/cosine/inner_product
		}

		columns = append(columns, fmt.Sprintf(
			"VECTOR INDEX %s (%s) WITH (distance=%s, type=hnsw, lib=vsag, m=%d, ef_construction=%d)",
			idxName, denseCol, distanceMetric,
			m.config.HnswM, m.config.HnswEfConstruction,
		))
	}

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", tableName, strings.Join(columns, ", "))
	if err := m.config.DB.WithContext(ctx).Exec(createSQL).Error; err != nil {
		return fmt.Errorf("[Create] CREATE TABLE failed, %w", err)
	}

	logs.CtxInfof(ctx, "[Create] table created with vector indexes, table=%s", tableName)

	return nil
}

func (m *obManager) Drop(ctx context.Context, req *searchstore.DropRequest) error {
	tableName, err := sanitizeIdentifier(req.CollectionName)
	if err != nil {
		return fmt.Errorf("[Drop] %w", err)
	}

	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	return m.config.DB.WithContext(ctx).Exec(dropSQL).Error
}

func (m *obManager) GetType() searchstore.SearchStoreType {
	return searchstore.TypeVectorStore
}

func (m *obManager) GetSearchStore(ctx context.Context, collectionName string) (searchstore.SearchStore, error) {
	tableName, err := sanitizeIdentifier(collectionName)
	if err != nil {
		return nil, fmt.Errorf("[GetSearchStore] %w", err)
	}

	exists, err := m.tableExists(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("[GetSearchStore] check table existence failed, %w", err)
	}
	if !exists {
		return nil, errorx.New(errno.ErrKnowledgeNonRetryableCode,
			errorx.KVf("reason", "[GetSearchStore] table=%v does not exist", tableName))
	}

	return &obSearchStore{
		config:    m.config,
		tableName: tableName,
	}, nil
}

func (m *obManager) GetEmbedding() embedding.Embedder {
	return m.config.Embedding
}

func (m *obManager) tableExists(ctx context.Context, tableName string) (bool, error) {
	var count int64
	err := m.config.DB.WithContext(ctx).Raw(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
		tableName,
	).Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *obManager) buildColumns(fields []*searchstore.Field) (columns []string, indexingFields []string, err error) {
	var foundID, foundCreatorID bool
	dims := m.config.Embedding.Dimensions()

	for _, f := range fields {
		safeName, err := sanitizeIdentifier(f.Name)
		if err != nil {
			return nil, nil, err
		}

		switch f.Name {
		case searchstore.FieldID:
			foundID = true
		case searchstore.FieldCreatorID:
			foundCreatorID = true
		}

		if f.IsPrimary {
			columns = append(columns, fmt.Sprintf("%s BIGINT PRIMARY KEY", safeName))
			continue
		}

		switch {
		case f.Type == searchstore.FieldTypeInt64:
			columns = append(columns, fmt.Sprintf("%s BIGINT", safeName))
		case f.Type == searchstore.FieldTypeText && f.Indexing:
			if f.Name == searchstore.FieldTextContent {
				columns = append(columns, fmt.Sprintf("%s TEXT", safeName))
			}
			columns = append(columns, fmt.Sprintf("%s VECTOR(%d)", denseFieldName(safeName), dims))
			indexingFields = append(indexingFields, f.Name)
		case f.Type == searchstore.FieldTypeText && !f.Indexing:
			columns = append(columns, fmt.Sprintf("%s TEXT", safeName))
		}
	}

	if !foundID {
		columns = append(columns, fmt.Sprintf("%s BIGINT PRIMARY KEY", searchstore.FieldID))
	}
	if !foundCreatorID {
		columns = append(columns, fmt.Sprintf("%s BIGINT NOT NULL", searchstore.FieldCreatorID))
	}

	columns = append(columns, "partition_key VARCHAR(255)")
	columns = append(columns, "INDEX idx_partition_key (partition_key)")

	return columns, indexingFields, nil
}
