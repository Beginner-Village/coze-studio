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

package searchstore

import (
	"context"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
)

type SearchStore interface {
	indexer.Indexer

	retriever.Retriever

	Delete(ctx context.Context, ids []string) error

	// DeleteByQuery removes all documents in `index` matching the ES query DSL.
	// The query argument follows Elasticsearch _delete_by_query body shape, e.g.:
	//   map[string]any{"term": map[string]any{"space_id": 123}}
	//   map[string]any{"terms": map[string]any{"kb_id": []int64{1,2,3}}}
	// Returns the count of deleted documents.
	DeleteByQuery(ctx context.Context, index string, query map[string]any) (deletedCount int64, err error)

	// DeleteIndex removes the entire index. Safe to call on a non-existent index
	// (the call returns nil). Used by the per-space resync flow to drop a KB's
	// chunk index (openynet_<kb_id>) before re-embedding from MySQL.
	//
	// Non-ES stores (Milvus / OceanBase / VikingDB) treat this as a no-op since
	// vector wipes go through the searchstore Manager.Drop path instead.
	DeleteIndex(ctx context.Context, index string) error
}
