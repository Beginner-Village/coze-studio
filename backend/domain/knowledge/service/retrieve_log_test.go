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
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestTopNScoreSummary(t *testing.T) {
	mkDoc := func(id string, score float64) *schema.Document {
		d := &schema.Document{ID: id}
		d.WithScore(score)
		return d
	}
	docs := []*schema.Document{
		mkDoc("1", 0.95),
		mkDoc("2", 0.82),
		mkDoc("3", 0.71),
		mkDoc("4", 0.50),
	}

	got := topNScoreSummary(docs, 3)
	want := "[{1:0.9500},{2:0.8200},{3:0.7100}]"
	if got != want {
		t.Errorf("top3 of 4: got %q, want %q", got, want)
	}

	got = topNScoreSummary(docs[:2], 3)
	want = "[{1:0.9500},{2:0.8200}]"
	if got != want {
		t.Errorf("top3 of 2: got %q, want %q", got, want)
	}

	if got := topNScoreSummary([]*schema.Document{}, 3); got != "[]" {
		t.Errorf("empty case: got %q, want %q", got, "[]")
	}

	if got := topNScoreSummary(nil, 3); got != "[]" {
		t.Errorf("nil case: got %q, want %q", got, "[]")
	}
}
