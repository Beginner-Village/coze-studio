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

package compose

import (
	"testing"

	"github.com/cloudwego/eino/compose"

	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
)

// Reproduces the "arrayDrillDown trying to drill down from an array of length 0"
// workflow failure: a knowledge-retrieval node finds nothing, so outputList is
// empty/null, and a downstream node references a field inside its elements.
// The drill-down should resolve to null instead of failing the whole workflow.
func TestDrillDownExtract_EmptyOrNullArray(t *testing.T) {
	nKey := vo.NodeKey("104489")
	fromPath := compose.FieldPath{"outputList", "content"}
	arraySegIndexes := []int{0}

	cases := []struct {
		name string
		in   any
	}{
		{"empty array", map[string]any{"outputList": []any{}}},
		{"null array field", map[string]any{"outputList": nil}},
		{"missing preceding value", map[string]any{"other": 1, "outputList": nil}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := drillDownExtract(nKey, fromPath, arraySegIndexes, c.in)
			if err != nil {
				t.Fatalf("expected no error for %s, got: %v", c.name, err)
			}
			if got != nil {
				t.Fatalf("expected nil drilled-down value for %s, got: %#v", c.name, got)
			}
		})
	}
}

// A non-empty array of objects must still drill into the first element.
func TestDrillDownExtract_NonEmptyArray(t *testing.T) {
	nKey := vo.NodeKey("104489")
	fromPath := compose.FieldPath{"outputList", "content"}
	arraySegIndexes := []int{0}

	in := map[string]any{"outputList": []any{
		map[string]any{"content": "hello"},
		map[string]any{"content": "world"},
	}}

	got, err := drillDownExtract(nKey, fromPath, arraySegIndexes, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected \"hello\", got: %#v", got)
	}
}

// A non-nil value of the wrong type (not an array where an array is expected)
// is a genuine type mismatch and must still error.
func TestDrillDownExtract_WrongTypeStillErrors(t *testing.T) {
	nKey := vo.NodeKey("104489")
	fromPath := compose.FieldPath{"outputList", "content"}
	arraySegIndexes := []int{0}

	in := map[string]any{"outputList": "not-an-array"}

	if _, err := drillDownExtract(nKey, fromPath, arraySegIndexes, in); err == nil {
		t.Fatalf("expected error for non-array value, got nil")
	}
}
