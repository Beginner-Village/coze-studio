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
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/slices"
)

type obSearchStore struct {
	config    *ManagerConfig
	tableName string
}

func (s *obSearchStore) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) (ids []string, err error) {
	if len(docs) == 0 {
		return nil, nil
	}

	implSpecOptions := indexer.GetImplSpecificOptions(&searchstore.IndexerOptions{}, opts...)
	defer func() {
		if err != nil {
			if implSpecOptions.ProgressBar != nil {
				implSpecOptions.ProgressBar.ReportError(err)
			}
		}
	}()

	indexingFields := make(map[string]struct{})
	for _, field := range implSpecOptions.IndexingFields {
		indexingFields[field] = struct{}{}
	}

	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	for _, batch := range slices.Chunks(docs, batchSize) {
		batchIDs, err := s.storeBatch(ctx, batch, implSpecOptions.Partition, indexingFields)
		if err != nil {
			return nil, err
		}
		ids = append(ids, batchIDs...)

		if implSpecOptions.ProgressBar != nil {
			if err = implSpecOptions.ProgressBar.AddN(len(batch)); err != nil {
				return nil, err
			}
		}
	}

	return ids, nil
}

func (s *obSearchStore) storeBatch(ctx context.Context, docs []*schema.Document, partition *string, indexingFields map[string]struct{}) ([]string, error) {
	var (
		ids          []string
		idVals       []int64
		creatorIDs   []int64
		contents     []string
		extFieldData = make(map[string][]interface{})
	)

	for _, doc := range docs {
		if doc.MetaData == nil {
			return nil, fmt.Errorf("[Store] meta data is nil")
		}

		id, err := strconv.ParseInt(doc.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("[Store] parse id failed, %w", err)
		}
		idVals = append(idVals, id)
		ids = append(ids, doc.ID)
		contents = append(contents, doc.Content)

		creatorID, err := document.GetDocumentCreatorID(doc)
		if err != nil {
			return nil, fmt.Errorf("[Store] creator_id not found or type invalid, %w", err)
		}
		creatorIDs = append(creatorIDs, creatorID)

		ext, ok := doc.MetaData[document.MetaDataKeyExternalStorage].(map[string]any)
		if ok {
			for field, val := range ext {
				extFieldData[field] = append(extFieldData[field], val)
			}
		}
	}

	columns := []string{searchstore.FieldID, searchstore.FieldCreatorID, searchstore.FieldTextContent, "partition_key"}

	type rowData struct {
		values []interface{}
	}
	rows := make([]rowData, len(docs))
	for i := range rows {
		partitionVal := ""
		if partition != nil {
			partitionVal = *partition
		}
		rows[i].values = []interface{}{idVals[i], creatorIDs[i], contents[i], partitionVal}
	}

	for _, fieldName := range sortedKeys(indexingFields) {
		if fieldName == searchstore.FieldTextContent {
			dense, err := s.config.Embedding.EmbedStrings(ctx, contents)
			if err != nil {
				return nil, fmt.Errorf("[Store] EmbedStrings failed, %w", err)
			}
			denseColName := denseFieldName(fieldName)
			if !validIdentifier.MatchString(denseColName) {
				return nil, fmt.Errorf("[Store] invalid column name: %s", denseColName)
			}
			columns = append(columns, denseColName)
			for i := range rows {
				rows[i].values = append(rows[i].values, vectorToString(dense[i]))
			}
		}
	}

	for field := range extFieldData {
		if !validIdentifier.MatchString(field) {
			return nil, fmt.Errorf("[Store] invalid column name: %s", field)
		}
		columns = append(columns, field)
		for i := range rows {
			if i < len(extFieldData[field]) {
				rows[i].values = append(rows[i].values, extFieldData[field][i])
			} else {
				rows[i].values = append(rows[i].values, nil)
			}
		}
	}

	for _, col := range columns {
		if !validIdentifier.MatchString(col) {
			return nil, fmt.Errorf("[Store] invalid column name: %s", col)
		}
	}

	placeholdersPerRow := make([]string, len(columns))
	for i := range placeholdersPerRow {
		placeholdersPerRow[i] = "?"
	}
	rowPlaceholder := "(" + strings.Join(placeholdersPerRow, ",") + ")"

	var allPlaceholders []string
	var allArgs []interface{}
	for _, row := range rows {
		allPlaceholders = append(allPlaceholders, rowPlaceholder)
		allArgs = append(allArgs, row.values...)
	}

	// Use REPLACE INTO because OceanBase 4.3.x does not support
	// ON DUPLICATE KEY UPDATE on tables with VECTOR INDEX.
	sql := fmt.Sprintf("REPLACE INTO %s (%s) VALUES %s",
		s.tableName,
		strings.Join(columns, ","),
		strings.Join(allPlaceholders, ","),
	)

	if err := s.config.DB.WithContext(ctx).Exec(sql, allArgs...).Error; err != nil {
		return nil, fmt.Errorf("[Store] exec insert failed, %w", err)
	}

	return ids, nil
}

func (s *obSearchStore) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	options := retriever.GetCommonOptions(&retriever.Options{TopK: ptr.Of(defaultTopK)}, opts...)
	implSpecOptions := retriever.GetImplSpecificOptions(&searchstore.RetrieverOptions{}, opts...)

	dense, err := s.config.Embedding.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("[Retrieve] EmbedStrings failed, %w", err)
	}
	queryVecStr := vectorToString(dense[0])

	selectCols := []string{
		searchstore.FieldID,
		searchstore.FieldCreatorID,
		searchstore.FieldTextContent,
	}

	var args []interface{}

	denseCol := denseFieldName(searchstore.FieldTextContent)
	distanceExpr := fmt.Sprintf("cosine_distance(%s, ?) AS distance", denseCol)
	args = append(args, queryVecStr)

	whereClauses := []string{"1=1"}

	if options.DSLInfo != nil {
		dslSQL, dslArgs, err := convertDSLMapToSQL(options.DSLInfo)
		if err != nil {
			return nil, fmt.Errorf("[Retrieve] convert DSL failed, %w", err)
		}
		if dslSQL != "" {
			whereClauses = append(whereClauses, dslSQL)
			args = append(args, dslArgs...)
		}
	}

	if len(implSpecOptions.Partitions) > 0 {
		placeholders := make([]string, len(implSpecOptions.Partitions))
		for i := range implSpecOptions.Partitions {
			placeholders[i] = "?"
			args = append(args, implSpecOptions.Partitions[i])
		}
		whereClauses = append(whereClauses, fmt.Sprintf("partition_key IN (%s)", strings.Join(placeholders, ",")))
	}

	orderExpr := fmt.Sprintf("cosine_distance(%s, ?)", denseCol)
	args = append(args, queryVecStr)

	topK := ptr.From(options.TopK)
	args = append(args, topK)

	sql := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s ORDER BY %s LIMIT ?",
		strings.Join(selectCols, ","),
		distanceExpr,
		s.tableName,
		strings.Join(whereClauses, " AND "),
		orderExpr,
	)

	rows, err := s.config.DB.WithContext(ctx).Raw(sql, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("[Retrieve] query failed, %w", err)
	}
	defer rows.Close()

	var docs []*schema.Document
	for rows.Next() {
		var (
			id          int64
			creatorID   int64
			textContent string
			distance    float64
		)

		if err := rows.Scan(&id, &creatorID, &textContent, &distance); err != nil {
			return nil, fmt.Errorf("[Retrieve] scan failed, %w", err)
		}

		doc := &schema.Document{
			ID:      strconv.FormatInt(id, 10),
			Content: textContent,
			MetaData: map[string]any{
				document.MetaDataKeyCreatorID:       creatorID,
				document.MetaDataKeyExternalStorage: map[string]any{},
			},
		}

		// cosine_distance returns [0, 2], convert to similarity [0, 1]
		score := 1.0 - distance/2.0
		doc.WithScore(score)
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("[Retrieve] rows iteration failed, %w", err)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Score() > docs[j].Score()
	})

	return docs, nil
}

func (s *obSearchStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	int64IDs := make([]int64, 0, len(ids))
	for _, sid := range ids {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			return fmt.Errorf("[Delete] parse id failed, %w", err)
		}
		int64IDs = append(int64IDs, id)
	}

	placeholders := make([]string, len(int64IDs))
	sqlArgs := make([]interface{}, len(int64IDs))
	for i, id := range int64IDs {
		placeholders[i] = "?"
		sqlArgs[i] = id
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)",
		s.tableName,
		searchstore.FieldID,
		strings.Join(placeholders, ","),
	)

	if err := s.config.DB.WithContext(ctx).Exec(sql, sqlArgs...).Error; err != nil {
		return fmt.Errorf("[Delete] exec delete failed, %w", err)
	}

	return nil
}

// DeleteByQuery is not supported by the OceanBase vector store. Callers that
// need space-wide wipes should use the SQL-backed Delete with an explicit ID
// list (the ES path is the only consumer of DeleteByQuery today).
func (s *obSearchStore) DeleteByQuery(ctx context.Context, index string, query map[string]any) (int64, error) {
	return 0, fmt.Errorf("[DeleteByQuery] not supported by oceanbase searchstore")
}

// DeleteIndex is a no-op for the OceanBase vector store. The ES path is the
// only owner of openynet_<kb_id> indices; OceanBase vector wipes happen via
// Manager.Drop / SQL delete, not via DeleteIndex. Returning nil lets the
// resync loop iterate all managers uniformly without per-type branching.
func (s *obSearchStore) DeleteIndex(ctx context.Context, index string) error {
	return nil
}
