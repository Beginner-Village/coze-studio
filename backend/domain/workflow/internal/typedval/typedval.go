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

// Package typedval provides type-safe value-access helpers over the
// workflow engine's ubiquitous map[string]any data flow.
//
// These helpers are pure functions with no side effects. They do NOT
// depend on eino, runtime state, or any workflow scheduling logic. They
// are intended as a foundation for incrementally migrating the engine
// away from ad-hoc runtime type assertions toward strongly-typed access.
//
// None of these functions are wired into existing nodes/compilers; they
// exist to be adopted gradually and safely.
package typedval

import (
	"strconv"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity/vo"
)

// GetString returns the string value at key.
// The second return value is false if the key is absent or the value is
// not a string.
func GetString(m map[string]any, key string) (string, bool) {
	v, ok := getKey(m, key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt64 returns the int64 value at key.
//
// Integral numeric values stored as int, int32, int64 or float64 (with no
// fractional part) are all coerced to int64, since JSON unmarshalling and
// upstream nodes may produce any of these representations for an integer.
func GetInt64(m map[string]any, key string) (int64, bool) {
	v, ok := getKey(m, key)
	if !ok {
		return 0, false
	}
	return toInt64(v)
}

// GetFloat64 returns the float64 value at key.
//
// int, int32, int64 and float32 values are coerced to float64.
func GetFloat64(m map[string]any, key string) (float64, bool) {
	v, ok := getKey(m, key)
	if !ok {
		return 0, false
	}
	return toFloat64(v)
}

// GetBool returns the bool value at key.
// The second return value is false if the key is absent or the value is
// not a bool.
func GetBool(m map[string]any, key string) (bool, bool) {
	v, ok := getKey(m, key)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// GetMap returns the map[string]any value at key.
// The second return value is false if the key is absent or the value is
// not a map[string]any.
func GetMap(m map[string]any, key string) (map[string]any, bool) {
	v, ok := getKey(m, key)
	if !ok {
		return nil, false
	}
	mm, ok := v.(map[string]any)
	return mm, ok
}

// GetSlice returns the []any value at key.
// The second return value is false if the key is absent or the value is
// not a []any.
func GetSlice(m map[string]any, key string) ([]any, bool) {
	v, ok := getKey(m, key)
	if !ok {
		return nil, false
	}
	s, ok := v.([]any)
	return s, ok
}

// GetByPath resolves a dotted path against a nested map, supporting array
// index subscripts. Supported syntaxes:
//
//	"a.b.c"      nested object access
//	"a[0].b"     array index then object access
//	"a.0.b"      array index expressed as a bare path segment
//
// The root container must be a map[string]any. It returns the resolved
// value and true on success, or (nil, false) if any segment is missing,
// an index is out of range, or a segment type does not match the access
// kind.
func GetByPath(m map[string]any, path string) (any, bool) {
	if m == nil {
		return nil, false
	}

	segs := parsePath(path)
	if len(segs) == 0 {
		return nil, false
	}

	var cur any = m
	for _, seg := range segs {
		if seg.isIndex {
			arr, ok := cur.([]any)
			if !ok {
				return nil, false
			}
			if seg.index < 0 || seg.index >= len(arr) {
				return nil, false
			}
			cur = arr[seg.index]
			continue
		}

		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := mm[seg.key]
		if !ok {
			return nil, false
		}
		cur = next
	}

	return cur, true
}

// Coerce applies a defensive, best-effort type normalization to value
// according to the supplied TypeInfo. It is intended to smooth over the
// representational differences (e.g. float64 vs int64 for integers) that
// arise from JSON round-tripping and heterogeneous upstream nodes.
//
// Only the basic scalar types are normalized: string, integer, number,
// boolean. For object/array/time/file and any value that cannot be
// normalized, the original value is returned unchanged together with a
// nil error.
//
// TODO: extend Coerce to recursively normalize object Properties and
// array ElemTypeInfo once the migration reaches composite types. Doing so
// now would risk diverging from the existing arrayDrillDown / mapping
// semantics, so it is intentionally deferred.
func Coerce(value any, t *vo.TypeInfo) (any, error) {
	if t == nil || value == nil {
		return value, nil
	}

	switch t.Type {
	case vo.DataTypeString:
		if s, ok := value.(string); ok {
			return s, nil
		}
		return value, nil
	case vo.DataTypeInteger:
		if i, ok := toInt64(value); ok {
			return i, nil
		}
		return value, nil
	case vo.DataTypeNumber:
		if f, ok := toFloat64(value); ok {
			return f, nil
		}
		return value, nil
	case vo.DataTypeBoolean:
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return value, nil
	default:
		// object / list / time / file: return as-is for now.
		return value, nil
	}
}

func getKey(m map[string]any, key string) (any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	return v, ok
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case float64:
		if n == float64(int64(n)) {
			return int64(n), true
		}
		return 0, false
	case float32:
		f := float64(n)
		if f == float64(int64(f)) {
			return int64(f), true
		}
		return 0, false
	default:
		return 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

type pathSeg struct {
	key     string
	index   int
	isIndex bool
}

// parsePath splits a path like "a.b[0].c" or "a.0.b" into ordered
// segments. A bare numeric segment is treated as an array index, matching
// the existing field-path convention where indices appear as plain path
// elements (see nodes.TakeMapValue / selector usage).
func parsePath(path string) []pathSeg {
	if path == "" {
		return nil
	}

	var segs []pathSeg
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		segs = append(segs, parseSegment(part)...)
	}
	return segs
}

// parseSegment parses a single dot-delimited part, peeling off any
// trailing bracketed indices, e.g. "a[0][1]" -> key "a", index 0, index 1.
func parseSegment(part string) []pathSeg {
	// Bare numeric segment -> array index.
	if idx, err := strconv.Atoi(part); err == nil {
		return []pathSeg{{index: idx, isIndex: true}}
	}

	open := strings.IndexByte(part, '[')
	if open < 0 {
		return []pathSeg{{key: part}}
	}

	var segs []pathSeg
	if open > 0 {
		segs = append(segs, pathSeg{key: part[:open]})
	}

	rest := part[open:]
	for len(rest) > 0 {
		if rest[0] != '[' {
			break
		}
		closeIdx := strings.IndexByte(rest, ']')
		if closeIdx < 0 {
			break
		}
		idxStr := rest[1:closeIdx]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			break
		}
		segs = append(segs, pathSeg{index: idx, isIndex: true})
		rest = rest[closeIdx+1:]
	}
	return segs
}
