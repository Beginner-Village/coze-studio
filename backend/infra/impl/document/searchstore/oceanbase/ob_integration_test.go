//go:build integration

package oceanbase

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	einoEmb "github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
)

var _ = strconv.Itoa

// fakeEmbedder returns deterministic vectors for testing.
type fakeEmbedder struct {
	dims int64
}

func (e *fakeEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...einoEmb.Option) ([][]float64, error) {
	result := make([][]float64, len(texts))
	for i, text := range texts {
		vec := make([]float64, e.dims)
		for j := int64(0); j < e.dims && j < int64(len(text)); j++ {
			vec[j] = float64(text[j]) / 255.0
		}
		var norm float64
		for _, v := range vec {
			norm += v * v
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for j := range vec {
				vec[j] /= norm
			}
		}
		result[i] = vec
	}
	return result, nil
}

func (e *fakeEmbedder) EmbedStringsHybrid(ctx context.Context, texts []string, opts ...einoEmb.Option) ([][]float64, []map[int]float64, error) {
	dense, err := e.EmbedStrings(ctx, texts, opts...)
	return dense, nil, err
}

func (e *fakeEmbedder) Dimensions() int64 {
	return e.dims
}

func (e *fakeEmbedder) SupportStatus() embedding.SupportStatus {
	return embedding.SupportDense
}

var _ embedding.Embedder = (*fakeEmbedder)(nil)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("OCEANBASE_DSN")
	if dsn == "" {
		dsn = "coze@test:coze123@tcp(172.93.101.237:2881)/opencoze?charset=utf8mb4&parseTime=True"
	}
	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		t.Fatalf("connect OceanBase failed: %v", err)
	}
	return db
}

func TestOceanBaseVectorIntegration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	emb := &fakeEmbedder{dims: 128}
	tableName := "test_ob_vector_integration"

	// Cleanup
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))

	// --- Test 1: Create collection ---
	t.Run("CreateCollection", func(t *testing.T) {
		mgr, err := NewManager(&ManagerConfig{
			DB:        db,
			Embedding: emb,
		})
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}

		err = mgr.Create(ctx, &searchstore.CreateRequest{
			CollectionName: tableName,
			Fields: []*searchstore.Field{
				{Name: searchstore.FieldID, Type: searchstore.FieldTypeInt64, IsPrimary: true},
				{Name: searchstore.FieldCreatorID, Type: searchstore.FieldTypeInt64},
				{Name: searchstore.FieldTextContent, Type: searchstore.FieldTypeText, Indexing: true},
			},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		// Verify table exists
		var count int64
		db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count)
		if count != 1 {
			t.Fatalf("expected table to exist, got count=%d", count)
		}
		t.Log("PASS: collection created successfully")
	})

	// --- Test 2: Store documents ---
	t.Run("StoreDocuments", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		ss, err := mgr.GetSearchStore(ctx, tableName)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		docs := []*schema.Document{
			{
				ID:      "1001",
				Content: "OceanBase is a distributed database",
				MetaData: map[string]any{
					document.MetaDataKeyCreatorID:       int64(100),
					document.MetaDataKeyExternalStorage: map[string]any{},
				},
			},
			{
				ID:      "1002",
				Content: "Milvus is a vector database",
				MetaData: map[string]any{
					document.MetaDataKeyCreatorID:       int64(100),
					document.MetaDataKeyExternalStorage: map[string]any{},
				},
			},
			{
				ID:      "1003",
				Content: "PostgreSQL supports pgvector extension",
				MetaData: map[string]any{
					document.MetaDataKeyCreatorID:       int64(200),
					document.MetaDataKeyExternalStorage: map[string]any{},
				},
			},
		}

		storeOpts := searchstore.WithIndexingFields([]string{searchstore.FieldTextContent})
		ids, err := ss.Store(ctx, docs, storeOpts)
		if err != nil {
			t.Fatalf("Store: %v", err)
		}
		if len(ids) != 3 {
			t.Fatalf("expected 3 ids, got %d", len(ids))
		}

		// Verify data in table
		var rowCount int64
		db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount)
		if rowCount != 3 {
			t.Fatalf("expected 3 rows, got %d", rowCount)
		}
		t.Logf("PASS: stored %d documents", len(ids))
	})

	// --- Test 3: Vector retrieval ---
	t.Run("VectorRetrieve", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		ss, err := mgr.GetSearchStore(ctx, tableName)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		docs, err := ss.Retrieve(ctx, "distributed database system",
			retriever.WithTopK(3),
		)
		if err != nil {
			t.Fatalf("Retrieve: %v", err)
		}
		if len(docs) == 0 {
			t.Fatal("expected results, got 0")
		}

		t.Logf("PASS: retrieved %d documents", len(docs))
		for i, doc := range docs {
			t.Logf("  [%d] id=%s score=%.4f content=%q", i, doc.ID, doc.Score(), doc.Content)
		}

		// Verify scores are in descending order
		for i := 1; i < len(docs); i++ {
			if docs[i].Score() > docs[i-1].Score() {
				t.Errorf("results not sorted by score: docs[%d].Score()=%.4f > docs[%d].Score()=%.4f",
					i, docs[i].Score(), i-1, docs[i-1].Score())
			}
		}
	})

	// --- Test 4: Retrieve with DSL filter ---
	t.Run("RetrieveWithDSLFilter", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		ss, err := mgr.GetSearchStore(ctx, tableName)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		// Filter by creator_id = 200 (only doc 1003)
		dslFilter := map[string]interface{}{
			"dsl": &searchstore.DSL{
				Op:    searchstore.OpEq,
				Field: searchstore.FieldCreatorID,
				Value: int64(200),
			},
		}

		docs, err := ss.Retrieve(ctx, "database",
			retriever.WithTopK(10),
			retriever.WithDSLInfo(dslFilter),
		)
		if err != nil {
			t.Fatalf("Retrieve with DSL: %v", err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected 1 document with creator_id=200, got %d", len(docs))
		}

		creatorID, _ := document.GetDocumentCreatorID(docs[0])
		if creatorID != 200 {
			t.Errorf("expected creator_id=200, got %d", creatorID)
		}
		t.Logf("PASS: DSL filter returned %d doc (id=%s, creator_id=%d)", len(docs), docs[0].ID, creatorID)
	})

	// --- Test 5: Store with partition ---
	t.Run("StoreAndRetrieveWithPartition", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		ss, err := mgr.GetSearchStore(ctx, tableName)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		partition := "space_42"
		docs := []*schema.Document{
			{
				ID:      "2001",
				Content: "Partitioned document for testing",
				MetaData: map[string]any{
					document.MetaDataKeyCreatorID:       int64(300),
					document.MetaDataKeyExternalStorage: map[string]any{},
				},
			},
		}

		storeOpts := searchstore.WithIndexingFields([]string{searchstore.FieldTextContent})
		partitionOpt := searchstore.WithPartition(partition)
		ids, err := ss.Store(ctx, docs, storeOpts, partitionOpt)
		if err != nil {
			t.Fatalf("Store with partition: %v", err)
		}
		if len(ids) != 1 {
			t.Fatalf("expected 1 id, got %d", len(ids))
		}

		// Retrieve with partition filter
		retrieveDocs, err := ss.Retrieve(ctx, "partitioned document",
			retriever.WithTopK(10),
			searchstore.WithPartitions([]string{partition}),
		)
		if err != nil {
			t.Fatalf("Retrieve with partition: %v", err)
		}

		found := false
		for _, d := range retrieveDocs {
			if d.ID == "2001" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected to find doc 2001 in partition results")
		}
		t.Logf("PASS: partition store/retrieve works, got %d docs", len(retrieveDocs))
	})

	// --- Test 6: Upsert (update existing document) ---
	t.Run("Upsert", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		ss, err := mgr.GetSearchStore(ctx, tableName)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		updatedDocs := []*schema.Document{
			{
				ID:      "1001",
				Content: "OceanBase is a distributed relational AND vector database",
				MetaData: map[string]any{
					document.MetaDataKeyCreatorID:       int64(100),
					document.MetaDataKeyExternalStorage: map[string]any{},
				},
			},
		}

		storeOpts := searchstore.WithIndexingFields([]string{searchstore.FieldTextContent})
		_, err = ss.Store(ctx, updatedDocs, storeOpts)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}

		// Verify content was updated
		var content string
		db.Raw(fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", searchstore.FieldTextContent, tableName, searchstore.FieldID), 1001).Scan(&content)
		expected := "OceanBase is a distributed relational AND vector database"
		if content != expected {
			t.Fatalf("upsert failed: got %q, expected %q", content, expected)
		}
		t.Log("PASS: upsert updated content correctly")
	})

	// --- Test 7: Delete documents ---
	t.Run("DeleteDocuments", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		ss, err := mgr.GetSearchStore(ctx, tableName)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		err = ss.Delete(ctx, []string{"1002"})
		if err != nil {
			t.Fatalf("Delete: %v", err)
		}

		var count int64
		db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", tableName, searchstore.FieldID), 1002).Scan(&count)
		if count != 0 {
			t.Fatalf("expected doc 1002 deleted, but found count=%d", count)
		}
		t.Log("PASS: document deleted successfully")
	})

	// --- Test 8: GetSearchStore for non-existent table ---
	t.Run("GetSearchStoreNonExistent", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		_, err := mgr.GetSearchStore(ctx, "non_existent_table_xyz")
		if err == nil {
			t.Fatal("expected error for non-existent table")
		}
		t.Logf("PASS: non-existent table returns error: %v", err)
	})

	// --- Test 9: Drop collection ---
	t.Run("DropCollection", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		err := mgr.Drop(ctx, &searchstore.DropRequest{CollectionName: tableName})
		if err != nil {
			t.Fatalf("Drop: %v", err)
		}

		var count int64
		db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count)
		if count != 0 {
			t.Fatalf("table should be dropped, but count=%d", count)
		}
		t.Log("PASS: collection dropped successfully")
	})

	// --- Test 10: Batch store performance ---
	t.Run("BatchStore", func(t *testing.T) {
		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb, BatchSize: 50})
		batchTable := "test_ob_batch_store"
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", batchTable))

		err := mgr.Create(ctx, &searchstore.CreateRequest{
			CollectionName: batchTable,
			Fields: []*searchstore.Field{
				{Name: searchstore.FieldID, Type: searchstore.FieldTypeInt64, IsPrimary: true},
				{Name: searchstore.FieldCreatorID, Type: searchstore.FieldTypeInt64},
				{Name: searchstore.FieldTextContent, Type: searchstore.FieldTypeText, Indexing: true},
			},
		})
		if err != nil {
			t.Fatalf("Create batch table: %v", err)
		}

		ss, err := mgr.GetSearchStore(ctx, batchTable)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		// Create 120 docs to test batch splitting (batch size = 50)
		docs := make([]*schema.Document, 120)
		for i := 0; i < 120; i++ {
			docs[i] = &schema.Document{
				ID:      strconv.Itoa(3000 + i),
				Content: fmt.Sprintf("Batch test document number %d with some content", i),
				MetaData: map[string]any{
					document.MetaDataKeyCreatorID:       int64(100),
					document.MetaDataKeyExternalStorage: map[string]any{},
				},
			}
		}

		storeOpts := searchstore.WithIndexingFields([]string{searchstore.FieldTextContent})
		ids, err := ss.Store(ctx, docs, storeOpts)
		if err != nil {
			t.Fatalf("BatchStore: %v", err)
		}
		if len(ids) != 120 {
			t.Fatalf("expected 120 ids, got %d", len(ids))
		}

		var count int64
		db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", batchTable)).Scan(&count)
		if count != 120 {
			t.Fatalf("expected 120 rows, got %d", count)
		}

		// Retrieve from batch
		results, err := ss.Retrieve(ctx, "batch test document", retriever.WithTopK(5))
		if err != nil {
			t.Fatalf("Retrieve from batch: %v", err)
		}
		if len(results) != 5 {
			t.Fatalf("expected 5 results, got %d", len(results))
		}

		// Cleanup
		mgr.Drop(ctx, &searchstore.DropRequest{CollectionName: batchTable})
		t.Logf("PASS: batch store 120 docs with batch_size=50 works correctly")
	})

	// --- Test 11: Retrieve with IN filter ---
	t.Run("RetrieveWithINFilter", func(t *testing.T) {
		inTable := "test_ob_in_filter"
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", inTable))

		mgr, _ := NewManager(&ManagerConfig{DB: db, Embedding: emb})
		err := mgr.Create(ctx, &searchstore.CreateRequest{
			CollectionName: inTable,
			Fields: []*searchstore.Field{
				{Name: searchstore.FieldID, Type: searchstore.FieldTypeInt64, IsPrimary: true},
				{Name: searchstore.FieldCreatorID, Type: searchstore.FieldTypeInt64},
				{Name: searchstore.FieldTextContent, Type: searchstore.FieldTypeText, Indexing: true},
			},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		ss, err := mgr.GetSearchStore(ctx, inTable)
		if err != nil {
			t.Fatalf("GetSearchStore: %v", err)
		}

		docs := []*schema.Document{
			{ID: "5001", Content: "Alpha content", MetaData: map[string]any{document.MetaDataKeyCreatorID: int64(10), document.MetaDataKeyExternalStorage: map[string]any{}}},
			{ID: "5002", Content: "Beta content", MetaData: map[string]any{document.MetaDataKeyCreatorID: int64(20), document.MetaDataKeyExternalStorage: map[string]any{}}},
			{ID: "5003", Content: "Gamma content", MetaData: map[string]any{document.MetaDataKeyCreatorID: int64(30), document.MetaDataKeyExternalStorage: map[string]any{}}},
		}
		storeOpts := searchstore.WithIndexingFields([]string{searchstore.FieldTextContent})
		_, err = ss.Store(ctx, docs, storeOpts)
		if err != nil {
			t.Fatalf("Store: %v", err)
		}

		// IN filter for creator_id in [10, 30]
		dslFilter := map[string]interface{}{
			"dsl": &searchstore.DSL{
				Op:    searchstore.OpIn,
				Field: searchstore.FieldCreatorID,
				Value: []int64{10, 30},
			},
		}

		results, err := ss.Retrieve(ctx, "content",
			retriever.WithTopK(10),
			retriever.WithDSLInfo(dslFilter),
		)
		if err != nil {
			t.Fatalf("Retrieve with IN: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results (creator 10,30), got %d", len(results))
		}

		idSet := map[string]bool{}
		for _, d := range results {
			idSet[d.ID] = true
		}
		if !idSet["5001"] || !idSet["5003"] {
			t.Fatalf("expected docs 5001 and 5003, got %v", idSet)
		}

		mgr.Drop(ctx, &searchstore.DropRequest{CollectionName: inTable})
		t.Logf("PASS: IN filter works correctly")
	})

	t.Log("\n=== ALL OCEANBASE VECTOR INTEGRATION TESTS PASSED ===")
}

// Ensure fakeEmbedder satisfies the interface
func TestFakeEmbedderInterface(t *testing.T) {
	var _ embedding.Embedder = (*fakeEmbedder)(nil)
}
