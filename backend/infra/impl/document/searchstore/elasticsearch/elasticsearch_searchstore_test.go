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

package elasticsearch

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
)

// TestExtractDocsFromHitsPreservesRawBM25 guards against the regression where
// ES top1 score was always 1.0 (score/firstScore). RRF + downstream filtering
// rely on raw BM25 scores being preserved.
func TestExtractDocsFromHitsPreservesRawBM25(t *testing.T) {
	hits := []esHit{
		{ID: "1", Content: "a", Score: 12.5},
		{ID: "2", Content: "b", Score: 6.0},
		{ID: "3", Content: "c", Score: 3.0},
	}
	docs := extractDocsFromHits(hits)
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}
	expectedScores := []float64{12.5, 6.0, 3.0}
	expectedIDs := []string{"1", "2", "3"}
	for i, doc := range docs {
		if doc.Score() != expectedScores[i] {
			t.Errorf("docs[%d].Score() = %v, want %v", i, doc.Score(), expectedScores[i])
		}
		if doc.ID != expectedIDs[i] {
			t.Errorf("docs[%d].ID = %q, want %q", i, doc.ID, expectedIDs[i])
		}
		if doc.Content != hits[i].Content {
			t.Errorf("docs[%d].Content = %q, want %q", i, doc.Content, hits[i].Content)
		}
	}
}

func TestExtractDocsFromHitsEmpty(t *testing.T) {
	docs := extractDocsFromHits(nil)
	if len(docs) != 0 {
		t.Fatalf("expected 0 docs from nil input, got %d", len(docs))
	}
}

// fakeESClient is a minimal es.Client used to verify that esSearchStore
// forwards DeleteByQuery's index/query untouched and returns the underlying
// client's deleted count / error. Only DeleteByQuery is exercised — other
// methods panic if invoked, which surfaces accidental wiring changes.
type fakeESClient struct {
	gotIndex   string
	gotQuery   map[string]any
	retDeleted int64
	retErr     error
}

func (f *fakeESClient) DeleteByQuery(_ context.Context, index string, query map[string]any) (int64, error) {
	f.gotIndex = index
	f.gotQuery = query
	return f.retDeleted, f.retErr
}

func (f *fakeESClient) Create(context.Context, string, string, any) error { panic("not used") }
func (f *fakeESClient) Update(context.Context, string, string, any) error { panic("not used") }
func (f *fakeESClient) Delete(context.Context, string, string) error      { panic("not used") }
func (f *fakeESClient) Search(context.Context, string, *es.Request) (*es.Response, error) {
	panic("not used")
}
func (f *fakeESClient) Exists(context.Context, string) (bool, error) { panic("not used") }
func (f *fakeESClient) CreateIndex(context.Context, string, map[string]any) error {
	panic("not used")
}
func (f *fakeESClient) DeleteIndex(context.Context, string) error          { panic("not used") }
func (f *fakeESClient) Types() es.Types                                    { panic("not used") }
func (f *fakeESClient) NewBulkIndexer(string) (es.BulkIndexer, error)      { panic("not used") }

func TestESSearchStore_DeleteByQuery_DelegatesToClient(t *testing.T) {
	fake := &fakeESClient{retDeleted: 42}
	store := &esSearchStore{
		config:    &ManagerConfig{Client: fake},
		indexName: "ignored_bound_index",
	}

	idx := "kb_text_space_100"
	query := map[string]any{"term": map[string]any{"space_id": int64(100)}}

	deleted, err := store.DeleteByQuery(context.Background(), idx, query)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if deleted != 42 {
		t.Errorf("deleted = %d, want 42", deleted)
	}
	if fake.gotIndex != idx {
		t.Errorf("forwarded index = %q, want %q", fake.gotIndex, idx)
	}
	if !reflect.DeepEqual(fake.gotQuery, query) {
		t.Errorf("forwarded query = %#v, want %#v", fake.gotQuery, query)
	}
}

func TestESSearchStore_DeleteByQuery_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	fake := &fakeESClient{retErr: wantErr}
	store := &esSearchStore{
		config:    &ManagerConfig{Client: fake},
		indexName: "any",
	}

	_, err := store.DeleteByQuery(context.Background(), "x", map[string]any{"match_all": map[string]any{}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
