/*
 * Copyright 2025 coze-dev Authors
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

package oceanbase

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func sanitizeIdentifier(name string) (string, error) {
	if !validIdentifier.MatchString(name) {
		return "", fmt.Errorf("invalid identifier: %q", name)
	}
	return name, nil
}

func denseFieldName(name string) string {
	return "dense_" + name
}

func denseIndexName(name string) string {
	return "idx_dense_" + name
}

func vectorToString(vec []float64) string {
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func convertDSLMapToSQL(src map[string]interface{}) (string, []interface{}, error) {
	if src == nil {
		return "", nil, nil
	}

	dsl, err := searchstore.LoadDSL(src)
	if err != nil {
		return "", nil, err
	}
	if dsl == nil {
		return "", nil, nil
	}

	return convertDSLToSQL(dsl)
}

func convertDSLToSQL(dsl *searchstore.DSL) (string, []interface{}, error) {
	if dsl == nil {
		return "", nil, nil
	}

	switch dsl.Op {
	case searchstore.OpEq:
		if !validIdentifier.MatchString(dsl.Field) {
			return "", nil, fmt.Errorf("[convertDSLToSQL] invalid field name: %s", dsl.Field)
		}
		return fmt.Sprintf("%s = ?", dsl.Field), []interface{}{dsl.Value}, nil

	case searchstore.OpNe:
		if !validIdentifier.MatchString(dsl.Field) {
			return "", nil, fmt.Errorf("[convertDSLToSQL] invalid field name: %s", dsl.Field)
		}
		return fmt.Sprintf("%s != ?", dsl.Field), []interface{}{dsl.Value}, nil

	case searchstore.OpLike:
		if !validIdentifier.MatchString(dsl.Field) {
			return "", nil, fmt.Errorf("[convertDSLToSQL] invalid field name: %s", dsl.Field)
		}
		return fmt.Sprintf("%s LIKE ?", dsl.Field), []interface{}{dsl.Value}, nil

	case searchstore.OpIn:
		if !validIdentifier.MatchString(dsl.Field) {
			return "", nil, fmt.Errorf("[convertDSLToSQL] invalid field name: %s", dsl.Field)
		}
		rv := reflect.ValueOf(dsl.Value)
		if rv.Kind() != reflect.Slice {
			return "", nil, fmt.Errorf("[convertDSLToSQL] OpIn value must be a slice, got %T", dsl.Value)
		}
		placeholders := make([]string, rv.Len())
		args := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			placeholders[i] = "?"
			args[i] = rv.Index(i).Interface()
		}
		return fmt.Sprintf("%s IN (%s)", dsl.Field, strings.Join(placeholders, ",")), args, nil

	case searchstore.OpAnd, searchstore.OpOr:
		children, ok := dsl.Value.([]*searchstore.DSL)
		if !ok {
			return "", nil, fmt.Errorf("[convertDSLToSQL] %s value must be []*DSL, got %T", dsl.Op, dsl.Value)
		}
		clauses := make([]string, 0, len(children))
		var allArgs []interface{}
		for _, child := range children {
			clause, args, err := convertDSLToSQL(child)
			if err != nil {
				return "", nil, err
			}
			clauses = append(clauses, "("+clause+")")
			allArgs = append(allArgs, args...)
		}
		op := " AND "
		if dsl.Op == searchstore.OpOr {
			op = " OR "
		}
		return strings.Join(clauses, op), allArgs, nil

	default:
		return "", nil, fmt.Errorf("[convertDSLToSQL] unknown op: %v", dsl.Op)
	}
}
