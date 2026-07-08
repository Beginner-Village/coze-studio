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

package loop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A ByArray loop whose incoming array is null / missing / empty (e.g. a
// knowledge-retrieval node returned no result) must gracefully degrade to zero
// iterations instead of panicking on reflect.TypeOf(nil) or the type assertion.
func TestGetMaxIter_ByArray_GracefulEmpty(t *testing.T) {
	l := &Loop{loopType: ByArray, inputArrays: []string{"a"}}

	cases := []struct {
		name string
		in   map[string]any
		want int
	}{
		{"null array", map[string]any{"a": nil}, 0},
		{"missing array", map[string]any{}, 0},
		{"empty slice", map[string]any{"a": []any{}}, 0},
		{"three items", map[string]any{"a": []any{1, 2, 3}}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, err := l.getMaxIter(c.in)
			assert.NoError(t, err)
			assert.Equal(t, c.want, n)
		})
	}
}

// ByIteration with a null / missing loop count degrades to zero iterations.
func TestGetMaxIter_ByIteration_GracefulNull(t *testing.T) {
	l := &Loop{loopType: ByIteration}

	n, err := l.getMaxIter(map[string]any{Count: nil})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	n, err = l.getMaxIter(map[string]any{Count: int64(5)})
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
}
