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

package dal

import (
	"reflect"
	"testing"
)

// TestEncodeInt64SliceEmptyIsValidJSON 守护回归：mcp_product_ids / skill_product_ids
// 是 JSON 类型列，空切片必须编码成合法 JSON "[]" 而非 ""，否则 MySQL 写入报
// "The document is empty."（Error 3140），导致 runtime-config/update 500。
func TestEncodeInt64SliceEmptyIsValidJSON(t *testing.T) {
	for _, in := range [][]int64{nil, {}} {
		if got := encodeInt64Slice(in); got != "[]" {
			t.Fatalf("encodeInt64Slice(%v) = %q, want %q", in, got, "[]")
		}
	}
}

func TestEncodeDecodeInt64SliceRoundTrip(t *testing.T) {
	cases := [][]int64{nil, {}, {1}, {1, 2, 3}}
	for _, in := range cases {
		encoded := encodeInt64Slice(in)
		decoded := decodeInt64Slice(encoded)
		// nil 与空切片语义等价（len 0）。
		if len(in) == 0 {
			if len(decoded) != 0 {
				t.Fatalf("round-trip %v -> %q -> %v, want empty", in, encoded, decoded)
			}
			continue
		}
		if !reflect.DeepEqual(in, decoded) {
			t.Fatalf("round-trip %v -> %q -> %v", in, encoded, decoded)
		}
	}
}
