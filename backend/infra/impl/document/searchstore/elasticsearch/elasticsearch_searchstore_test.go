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

import "testing"

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
