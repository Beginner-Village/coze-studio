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

package builtin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamJSONArray(t *testing.T) {
	input := `[{"a":1},{"a":2},{"a":3}]`
	items, err := streamJSONArray(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, `{"a":1}`, items[0])
}

func TestStreamJSONArray_Empty(t *testing.T) {
	items, err := streamJSONArray(strings.NewReader(`[]`))
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestStreamJSONArray_NotArray(t *testing.T) {
	_, err := streamJSONArray(strings.NewReader(`{"x": 1}`))
	assert.Error(t, err)
}
