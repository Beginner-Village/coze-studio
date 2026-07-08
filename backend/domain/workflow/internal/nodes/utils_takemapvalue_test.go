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

package nodes

import (
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/stretchr/testify/assert"
)

// TakeMapValue must not panic when an intermediate path segment is not a map
// (e.g. upstream produced a null or a scalar where a nested object was expected).
// It should report "not found" instead of crashing the node.
func TestTakeMapValue_NonMapIntermediate(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]any
		path compose.FieldPath
	}{
		{"intermediate is scalar", map[string]any{"a": "not-a-map"}, compose.FieldPath{"a", "b"}},
		{"intermediate is null", map[string]any{"a": nil}, compose.FieldPath{"a", "b"}},
		{"intermediate is slice", map[string]any{"a": []any{1, 2}}, compose.FieldPath{"a", "b"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, ok := TakeMapValue(c.m, c.path)
			assert.False(t, ok)
			assert.Nil(t, v)
		})
	}
}

// Regression guard: normal nested access and simple lookups still work.
func TestTakeMapValue_HappyPath(t *testing.T) {
	m := map[string]any{"a": map[string]any{"b": int64(7)}, "x": "y"}

	v, ok := TakeMapValue(m, compose.FieldPath{"a", "b"})
	assert.True(t, ok)
	assert.Equal(t, int64(7), v)

	v, ok = TakeMapValue(m, compose.FieldPath{"x"})
	assert.True(t, ok)
	assert.Equal(t, "y", v)

	_, ok = TakeMapValue(m, compose.FieldPath{"missing"})
	assert.False(t, ok)
}
