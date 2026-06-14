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

package typedval

import (
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
)

func TestGetString(t *testing.T) {
	m := map[string]any{"a": "hello", "b": 42, "c": nil}

	if v, ok := GetString(m, "a"); !ok || v != "hello" {
		t.Fatalf("hit: got (%q,%v)", v, ok)
	}
	if _, ok := GetString(m, "missing"); ok {
		t.Fatalf("miss should be false")
	}
	if _, ok := GetString(m, "b"); ok {
		t.Fatalf("type mismatch (int) should be false")
	}
	if _, ok := GetString(m, "c"); ok {
		t.Fatalf("nil value should be false")
	}
	if _, ok := GetString(nil, "a"); ok {
		t.Fatalf("nil map should be false")
	}
}

func TestGetInt64(t *testing.T) {
	m := map[string]any{
		"i64":      int64(7),
		"i":        int(8),
		"i32":      int32(9),
		"floatInt": float64(10),
		"floatFr":  float64(10.5),
		"f32Int":   float32(11),
		"str":      "12",
	}

	cases := []struct {
		key  string
		want int64
		ok   bool
	}{
		{"i64", 7, true},
		{"i", 8, true},
		{"i32", 9, true},
		{"floatInt", 10, true},
		{"f32Int", 11, true},
		{"floatFr", 0, false},
		{"str", 0, false},
		{"missing", 0, false},
	}
	for _, c := range cases {
		v, ok := GetInt64(m, c.key)
		if ok != c.ok || (ok && v != c.want) {
			t.Fatalf("key %s: got (%d,%v) want (%d,%v)", c.key, v, ok, c.want, c.ok)
		}
	}
	if _, ok := GetInt64(nil, "i64"); ok {
		t.Fatalf("nil map should be false")
	}
}

func TestGetFloat64(t *testing.T) {
	m := map[string]any{
		"f":    float64(1.5),
		"f32":  float32(2.5),
		"i64":  int64(3),
		"i":    int(4),
		"i32":  int32(5),
		"str":  "x",
		"bool": true,
	}

	cases := []struct {
		key  string
		want float64
		ok   bool
	}{
		{"f", 1.5, true},
		{"f32", 2.5, true},
		{"i64", 3, true},
		{"i", 4, true},
		{"i32", 5, true},
		{"str", 0, false},
		{"bool", 0, false},
		{"missing", 0, false},
	}
	for _, c := range cases {
		v, ok := GetFloat64(m, c.key)
		if ok != c.ok || (ok && v != c.want) {
			t.Fatalf("key %s: got (%v,%v) want (%v,%v)", c.key, v, ok, c.want, c.ok)
		}
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{"t": true, "f": false, "s": "true", "n": 1}

	if v, ok := GetBool(m, "t"); !ok || v != true {
		t.Fatalf("t: got (%v,%v)", v, ok)
	}
	if v, ok := GetBool(m, "f"); !ok || v != false {
		t.Fatalf("f: got (%v,%v)", v, ok)
	}
	if _, ok := GetBool(m, "s"); ok {
		t.Fatalf("string should be false")
	}
	if _, ok := GetBool(m, "n"); ok {
		t.Fatalf("int should be false")
	}
	if _, ok := GetBool(m, "missing"); ok {
		t.Fatalf("missing should be false")
	}
}

func TestGetMap(t *testing.T) {
	inner := map[string]any{"k": "v"}
	m := map[string]any{"obj": inner, "notobj": []any{1, 2}}

	if v, ok := GetMap(m, "obj"); !ok || v["k"] != "v" {
		t.Fatalf("obj: got (%v,%v)", v, ok)
	}
	if _, ok := GetMap(m, "notobj"); ok {
		t.Fatalf("slice should be false")
	}
	if _, ok := GetMap(m, "missing"); ok {
		t.Fatalf("missing should be false")
	}
}

func TestGetSlice(t *testing.T) {
	m := map[string]any{"arr": []any{1, "two", 3.0}, "notarr": "x"}

	if v, ok := GetSlice(m, "arr"); !ok || len(v) != 3 {
		t.Fatalf("arr: got (%v,%v)", v, ok)
	}
	if _, ok := GetSlice(m, "notarr"); ok {
		t.Fatalf("string should be false")
	}
	if _, ok := GetSlice(m, "missing"); ok {
		t.Fatalf("missing should be false")
	}
}

func TestGetByPath(t *testing.T) {
	m := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep",
			},
			"list": []any{
				map[string]any{"x": int64(1)},
				map[string]any{"x": int64(2)},
			},
		},
		"nums": []any{10, 20, 30},
		"top":  "value",
	}

	cases := []struct {
		name string
		path string
		want any
		ok   bool
	}{
		{"single", "top", "value", true},
		{"nested", "a.b.c", "deep", true},
		{"array bracket then field", "a.list[1].x", int64(2), true},
		{"array bare index then field", "a.list.0.x", int64(1), true},
		{"bare index on root slice", "nums.2", 30, true},
		{"bracket index on root slice", "nums[0]", 10, true},
		{"missing key", "a.zzz", nil, false},
		{"missing nested", "a.b.zzz", nil, false},
		{"index out of range", "nums[9]", nil, false},
		{"negative-like out of range", "a.list[5].x", nil, false},
		{"drill into non-map", "top.x", nil, false},
		{"index into non-array", "a.b[0]", nil, false},
		{"empty path", "", nil, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := GetByPath(m, c.path)
			if ok != c.ok {
				t.Fatalf("path %q: ok=%v want %v (got=%v)", c.path, ok, c.ok, got)
			}
			if ok && got != c.want {
				t.Fatalf("path %q: got %v want %v", c.path, got, c.want)
			}
		})
	}

	if _, ok := GetByPath(nil, "a"); ok {
		t.Fatalf("nil map should be false")
	}
}

func TestCoerce(t *testing.T) {
	cases := []struct {
		name  string
		value any
		ti    *vo.TypeInfo
		want  any
	}{
		{"nil typeinfo passthrough", "x", nil, "x"},
		{"nil value passthrough", nil, &vo.TypeInfo{Type: vo.DataTypeString}, nil},
		{"string ok", "hi", &vo.TypeInfo{Type: vo.DataTypeString}, "hi"},
		{"string mismatch passthrough", 5, &vo.TypeInfo{Type: vo.DataTypeString}, 5},
		{"int from float", float64(3), &vo.TypeInfo{Type: vo.DataTypeInteger}, int64(3)},
		{"int from int", int(4), &vo.TypeInfo{Type: vo.DataTypeInteger}, int64(4)},
		{"int from fractional float passthrough", float64(3.5), &vo.TypeInfo{Type: vo.DataTypeInteger}, float64(3.5)},
		{"number from int", int64(7), &vo.TypeInfo{Type: vo.DataTypeNumber}, float64(7)},
		{"number from float", float64(2.5), &vo.TypeInfo{Type: vo.DataTypeNumber}, float64(2.5)},
		{"bool ok", true, &vo.TypeInfo{Type: vo.DataTypeBoolean}, true},
		{"bool mismatch passthrough", "true", &vo.TypeInfo{Type: vo.DataTypeBoolean}, "true"},
		{"object passthrough", map[string]any{"k": 1}, &vo.TypeInfo{Type: vo.DataTypeObject}, map[string]any{"k": 1}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Coerce(c.value, c.ti)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Compare; maps compared by simple equality via type assertion.
			if mm, ok := c.want.(map[string]any); ok {
				gm, ok := got.(map[string]any)
				if !ok || len(gm) != len(mm) {
					t.Fatalf("map mismatch: got %v want %v", got, c.want)
				}
				for k, v := range mm {
					if gm[k] != v {
						t.Fatalf("map key %s: got %v want %v", k, gm[k], v)
					}
				}
				return
			}
			if got != c.want {
				t.Fatalf("got %v (%T) want %v (%T)", got, got, c.want, c.want)
			}
		})
	}
}
